package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unbracketed/ccmgr-ultra/internal/storage"
)

// EngineConfig defines configuration for the analytics engine
type EngineConfig struct {
	CacheSize       int           `yaml:"cache_size" json:"cache_size" default:"1000"`
	CacheTTL        time.Duration `yaml:"cache_ttl" json:"cache_ttl" default:"5m"`
	BatchProcessing bool          `yaml:"batch_processing" json:"batch_processing" default:"true"`
	PrecomputeDaily bool          `yaml:"precompute_daily" json:"precompute_daily" default:"true"`
}

// SetDefaults sets default values for EngineConfig
func (e *EngineConfig) SetDefaults() {
	if e.CacheSize == 0 {
		e.CacheSize = 1000
	}
	if e.CacheTTL == 0 {
		e.CacheTTL = 5 * time.Minute
	}
	e.BatchProcessing = true
	e.PrecomputeDaily = true
}

// Validate validates the engine configuration
func (e *EngineConfig) Validate() error {
	if e.CacheSize < 0 {
		return fmt.Errorf("cache size cannot be negative")
	}
	if e.CacheTTL < 0 {
		return fmt.Errorf("cache TTL cannot be negative")
	}
	return nil
}

// Engine implements analytics processing and aggregation
type Engine struct {
	storage    storage.Storage
	cache      Cache
	config     *EngineConfig
	calculator MetricCalculator
	mutex      sync.RWMutex

	// Background processing
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
}

// NewEngine creates a new analytics engine
func NewEngine(storage storage.Storage, config *EngineConfig) *Engine {
	if config == nil {
		config = &EngineConfig{}
		config.SetDefaults()
	}

	cache := NewMemoryCache(config.CacheSize, config.CacheTTL)
	calculator := NewDefaultMetricCalculator(storage, cache)

	return &Engine{
		storage:    storage,
		cache:      cache,
		config:     config,
		calculator: calculator,
	}
}

// Start starts the background processing routines
func (e *Engine) Start(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.running {
		return fmt.Errorf("engine is already running")
	}

	if err := e.config.Validate(); err != nil {
		return fmt.Errorf("invalid engine configuration: %w", err)
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.running = true

	// Start daily precomputation if enabled
	if e.config.PrecomputeDaily {
		e.wg.Add(1)
		go e.dailyPrecomputeLoop()
	}

	// Start cache maintenance
	e.wg.Add(1)
	go e.cacheMaintenanceLoop()

	return nil
}

// Stop stops the background processing routines
func (e *Engine) Stop() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.running {
		return nil
	}

	e.running = false
	e.cancel()
	e.wg.Wait()

	return nil
}

// IsRunning returns whether the engine is currently running
func (e *Engine) IsRunning() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.running
}

// GetMetrics calculates metrics for the specified time range
func (e *Engine) GetMetrics(ctx context.Context, timeRange TimeRange) (*Metrics, error) {
	return e.calculator.Calculate(ctx, timeRange)
}

// GetMetricsForProject calculates metrics for a specific project
func (e *Engine) GetMetricsForProject(ctx context.Context, project string, timeRange TimeRange) (*Metrics, error) {
	return e.calculator.CalculateForProject(ctx, project, timeRange)
}

// GetMetricsForWorktree calculates metrics for a specific worktree
func (e *Engine) GetMetricsForWorktree(ctx context.Context, worktree string, timeRange TimeRange) (*Metrics, error) {
	return e.calculator.CalculateForWorktree(ctx, worktree, timeRange)
}

// GetDailyMetrics gets daily metrics for a specific date range
func (e *Engine) GetDailyMetrics(ctx context.Context, start, end time.Time) ([]DailyMetric, error) {
	cacheKey := fmt.Sprintf("daily_metrics_%s_%s", start.Format("2006-01-02"), end.Format("2006-01-02"))

	// Try cache first
	if cached, found := e.cache.Get(cacheKey); found {
		if metrics, ok := cached.([]DailyMetric); ok {
			return metrics, nil
		}
	}

	// Calculate if not cached
	metrics := []DailyMetric{}
	current := start

	for current.Before(end) || current.Equal(end) {
		dailyStart := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
		dailyEnd := dailyStart.Add(24 * time.Hour).Add(-time.Nanosecond)

		timeRange := TimeRange{Start: dailyStart, End: dailyEnd}
		dayMetrics, err := e.calculateDailyMetric(ctx, current, timeRange)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate metrics for %s: %w", current.Format("2006-01-02"), err)
		}

		metrics = append(metrics, *dayMetrics)
		current = current.AddDate(0, 0, 1)
	}

	// Cache the result
	e.cache.Set(cacheKey, metrics, e.config.CacheTTL)

	return metrics, nil
}

// GetWeeklyMetrics gets weekly metrics for a specific date range
func (e *Engine) GetWeeklyMetrics(ctx context.Context, start, end time.Time) ([]WeeklyMetric, error) {
	cacheKey := fmt.Sprintf("weekly_metrics_%s_%s", start.Format("2006-01-02"), end.Format("2006-01-02"))

	// Try cache first
	if cached, found := e.cache.Get(cacheKey); found {
		if metrics, ok := cached.([]WeeklyMetric); ok {
			return metrics, nil
		}
	}

	// Calculate if not cached
	metrics := []WeeklyMetric{}
	current := start

	// Align to start of week (Monday)
	for current.Weekday() != time.Monday {
		current = current.AddDate(0, 0, -1)
	}

	for current.Before(end) {
		weekEnd := current.AddDate(0, 0, 6)
		if weekEnd.After(end) {
			weekEnd = end
		}

		weekMetrics, err := e.calculateWeeklyMetric(ctx, current, weekEnd)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate weekly metrics for %s: %w", current.Format("2006-01-02"), err)
		}

		metrics = append(metrics, *weekMetrics)
		current = current.AddDate(0, 0, 7)
	}

	// Cache the result
	e.cache.Set(cacheKey, metrics, e.config.CacheTTL)

	return metrics, nil
}

// GetProductivityMetrics calculates productivity metrics
func (e *Engine) GetProductivityMetrics(ctx context.Context, timeRange TimeRange) ([]ProductivityMetric, error) {
	cacheKey := fmt.Sprintf("productivity_%s_%s", timeRange.Start.Format("2006-01-02"), timeRange.End.Format("2006-01-02"))

	// Try cache first
	if cached, found := e.cache.Get(cacheKey); found {
		if metrics, ok := cached.([]ProductivityMetric); ok {
			return metrics, nil
		}
	}

	// Get daily metrics for the range
	dailyMetrics, err := e.GetDailyMetrics(ctx, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily metrics: %w", err)
	}

	// Calculate productivity metrics from daily data
	productivityMetrics := make([]ProductivityMetric, len(dailyMetrics))
	for i, daily := range dailyMetrics {
		productivityMetrics[i] = e.calculateProductivityFromDaily(daily)
	}

	// Cache the result
	e.cache.Set(cacheKey, productivityMetrics, e.config.CacheTTL)

	return productivityMetrics, nil
}

// InvalidateCache invalidates cached data for a specific pattern
func (e *Engine) InvalidateCache(pattern string) {
	// This is a simplified implementation
	// In a real implementation, you might have pattern-based cache invalidation
	e.cache.Clear()
}

// GetStats returns engine statistics
func (e *Engine) GetStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	stats := map[string]interface{}{
		"running":          e.running,
		"cache_size":       e.config.CacheSize,
		"cache_ttl":        e.config.CacheTTL.String(),
		"batch_processing": e.config.BatchProcessing,
		"precompute_daily": e.config.PrecomputeDaily,
	}

	// Add cache statistics if available
	if cacheStats, ok := e.cache.(*MemoryCache); ok {
		stats["cache_items"] = cacheStats.Len()
		stats["cache_hits"] = cacheStats.GetHits()
		stats["cache_misses"] = cacheStats.GetMisses()
	}

	return stats
}

// dailyPrecomputeLoop runs daily precomputation in the background
func (e *Engine) dailyPrecomputeLoop() {
	defer e.wg.Done()

	// Run precomputation at 2 AM daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Calculate initial delay to 2 AM
	now := time.Now()
	next2AM := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if now.After(next2AM) {
		next2AM = next2AM.AddDate(0, 0, 1)
	}

	initialDelay := next2AM.Sub(now)
	initialTimer := time.NewTimer(initialDelay)
	defer initialTimer.Stop()

	select {
	case <-e.ctx.Done():
		return
	case <-initialTimer.C:
		e.performDailyPrecompute()
	}

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.performDailyPrecompute()
		}
	}
}

// cacheMaintenanceLoop performs periodic cache maintenance
func (e *Engine) cacheMaintenanceLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			// Clean expired entries
			if maintainable, ok := e.cache.(interface{ Cleanup() }); ok {
				maintainable.Cleanup()
			}
		}
	}
}

// performDailyPrecompute precomputes metrics for recent days
func (e *Engine) performDailyPrecompute() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Precompute metrics for the last 7 days
	end := time.Now()
	start := end.AddDate(0, 0, -7)

	// Precompute daily metrics
	_, err := e.GetDailyMetrics(ctx, start, end)
	if err != nil {
		fmt.Printf("Daily precompute failed: %v\n", err)
	}

	// Precompute weekly metrics
	weekStart := start.AddDate(0, 0, -7) // Go back one more week
	_, err = e.GetWeeklyMetrics(ctx, weekStart, end)
	if err != nil {
		fmt.Printf("Weekly precompute failed: %v\n", err)
	}
}

// Helper methods for calculating specific metrics

func (e *Engine) calculateDailyMetric(ctx context.Context, date time.Time, timeRange TimeRange) (*DailyMetric, error) {
	// Get sessions for the day
	filter := storage.SessionFilter{
		Since:     timeRange.Start,
		Until:     timeRange.End,
		SortBy:    "created_at",
		SortOrder: "asc",
	}

	sessions, err := e.storage.Sessions().List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	metric := &DailyMetric{
		Date:          date,
		TotalSessions: len(sessions),
		Projects:      make(map[string]int),
		Worktrees:     make(map[string]int),
	}

	var totalDuration time.Duration
	for _, session := range sessions {
		sessionDuration := session.UpdatedAt.Sub(session.CreatedAt)
		totalDuration += sessionDuration

		// Count projects and worktrees
		if session.Project != "" {
			metric.Projects[session.Project]++
		}
		if session.Worktree != "" {
			metric.Worktrees[session.Worktree]++
		}
	}

	metric.TotalDuration = totalDuration
	if len(sessions) > 0 {
		metric.AverageDuration = totalDuration / time.Duration(len(sessions))
	}

	// Calculate active/idle time from events
	activeTime, idleTime, err := e.calculateActiveIdleTime(ctx, timeRange)
	if err != nil {
		// Don't fail completely, just log and continue
		fmt.Printf("Failed to calculate active/idle time for %s: %v\n", date.Format("2006-01-02"), err)
	} else {
		metric.ActiveTime = activeTime
		metric.IdleTime = idleTime
	}

	return metric, nil
}

func (e *Engine) calculateWeeklyMetric(ctx context.Context, weekStart, weekEnd time.Time) (*WeeklyMetric, error) {
	// Get daily metrics for each day of the week
	dailyMetrics, err := e.GetDailyMetrics(ctx, weekStart, weekEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily metrics: %w", err)
	}

	metric := &WeeklyMetric{
		WeekStart:      weekStart,
		WeekEnd:        weekEnd,
		DailyBreakdown: dailyMetrics,
		TopProjects:    make(map[string]int),
		TopWorktrees:   make(map[string]int),
	}

	// Aggregate from daily metrics
	for _, daily := range dailyMetrics {
		metric.TotalSessions += daily.TotalSessions
		metric.TotalDuration += daily.TotalDuration
		metric.ActiveTime += daily.ActiveTime
		metric.IdleTime += daily.IdleTime

		// Aggregate projects and worktrees
		for project, count := range daily.Projects {
			metric.TopProjects[project] += count
		}
		for worktree, count := range daily.Worktrees {
			metric.TopWorktrees[worktree] += count
		}
	}

	if metric.TotalSessions > 0 {
		metric.AverageDuration = metric.TotalDuration / time.Duration(metric.TotalSessions)
	}

	return metric, nil
}

func (e *Engine) calculateProductivityFromDaily(daily DailyMetric) ProductivityMetric {
	var productivityRatio float64
	totalTime := daily.ActiveTime + daily.IdleTime
	if totalTime > 0 {
		productivityRatio = float64(daily.ActiveTime) / float64(totalTime)
	}

	return ProductivityMetric{
		Date:              daily.Date,
		ProductivityRatio: productivityRatio,
		FocusTime:         daily.ActiveTime,
		DistractionTime:   daily.IdleTime,
		SessionsPerDay:    float64(daily.TotalSessions),
		// Additional metrics would be calculated here
	}
}

func (e *Engine) calculateActiveIdleTime(ctx context.Context, timeRange TimeRange) (activeTime, idleTime time.Duration, err error) {
	// Get events for the time range
	filter := storage.EventFilter{
		EventTypes: []string{EventTypeStateChange, EventTypeActivity, EventTypeIdle},
		Since:      timeRange.Start,
		Until:      timeRange.End,
	}

	events, err := e.storage.Events().GetByFilter(ctx, filter)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get events: %w", err)
	}

	// Simple calculation - count time in active vs idle states
	// This is a simplified implementation
	for _, event := range events {
		if event.EventType == EventTypeActivity {
			if durationMs, ok := event.Data["duration_ms"].(float64); ok {
				activeTime += time.Duration(durationMs) * time.Millisecond
			}
		} else if event.EventType == EventTypeIdle {
			if durationMs, ok := event.Data["duration_ms"].(float64); ok {
				idleTime += time.Duration(durationMs) * time.Millisecond
			}
		}
	}

	return activeTime, idleTime, nil
}
