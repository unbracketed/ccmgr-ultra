package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
)

// MetricCalculator defines the interface for metric calculations
type MetricCalculator interface {
	Calculate(ctx context.Context, timeRange TimeRange) (*Metrics, error)
	CalculateForProject(ctx context.Context, project string, timeRange TimeRange) (*Metrics, error)
	CalculateForWorktree(ctx context.Context, worktree string, timeRange TimeRange) (*Metrics, error)
}

// DefaultMetricCalculator implements MetricCalculator using storage and cache
type DefaultMetricCalculator struct {
	storage storage.Storage
	cache   Cache
}

// NewDefaultMetricCalculator creates a new metric calculator
func NewDefaultMetricCalculator(storage storage.Storage, cache Cache) *DefaultMetricCalculator {
	return &DefaultMetricCalculator{
		storage: storage,
		cache:   cache,
	}
}

// Calculate calculates comprehensive metrics for the specified time range
func (c *DefaultMetricCalculator) Calculate(ctx context.Context, timeRange TimeRange) (*Metrics, error) {
	cacheKey := fmt.Sprintf("metrics_%s_%s", timeRange.Start.Format("2006-01-02"), timeRange.End.Format("2006-01-02"))
	
	// Try cache first
	if cached, found := c.cache.Get(cacheKey); found {
		if metrics, ok := cached.(*Metrics); ok {
			return metrics, nil
		}
	}

	// Calculate fresh metrics
	metrics, err := c.calculateMetrics(ctx, timeRange, "", "")
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache.Set(cacheKey, metrics, 5*time.Minute)
	
	return metrics, nil
}

// CalculateForProject calculates metrics for a specific project
func (c *DefaultMetricCalculator) CalculateForProject(ctx context.Context, project string, timeRange TimeRange) (*Metrics, error) {
	cacheKey := fmt.Sprintf("metrics_project_%s_%s_%s", project, timeRange.Start.Format("2006-01-02"), timeRange.End.Format("2006-01-02"))
	
	// Try cache first
	if cached, found := c.cache.Get(cacheKey); found {
		if metrics, ok := cached.(*Metrics); ok {
			return metrics, nil
		}
	}

	// Calculate fresh metrics
	metrics, err := c.calculateMetrics(ctx, timeRange, project, "")
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache.Set(cacheKey, metrics, 5*time.Minute)
	
	return metrics, nil
}

// CalculateForWorktree calculates metrics for a specific worktree
func (c *DefaultMetricCalculator) CalculateForWorktree(ctx context.Context, worktree string, timeRange TimeRange) (*Metrics, error) {
	cacheKey := fmt.Sprintf("metrics_worktree_%s_%s_%s", worktree, timeRange.Start.Format("2006-01-02"), timeRange.End.Format("2006-01-02"))
	
	// Try cache first
	if cached, found := c.cache.Get(cacheKey); found {
		if metrics, ok := cached.(*Metrics); ok {
			return metrics, nil
		}
	}

	// Calculate fresh metrics
	metrics, err := c.calculateMetrics(ctx, timeRange, "", worktree)
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache.Set(cacheKey, metrics, 5*time.Minute)
	
	return metrics, nil
}

// calculateMetrics performs the actual metric calculation
func (c *DefaultMetricCalculator) calculateMetrics(ctx context.Context, timeRange TimeRange, project, worktree string) (*Metrics, error) {
	// Build session filter
	filter := storage.SessionFilter{
		Since:     timeRange.Start,
		Until:     timeRange.End,
		Project:   project,
		Worktree:  worktree,
		SortBy:    "created_at",
		SortOrder: "asc",
	}

	// Get sessions
	sessions, err := c.storage.Sessions().List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	// Calculate basic session metrics
	sessionMetrics := c.calculateSessionMetrics(sessions, timeRange)

	// Calculate activity metrics
	activityMetrics, err := c.calculateActivityMetrics(ctx, timeRange, project, worktree)
	if err != nil {
		// Don't fail completely, log and continue
		fmt.Printf("Failed to calculate activity metrics: %v\n", err)
		activityMetrics = &ActivityMetrics{}
	}

	// Calculate project and worktree distribution
	projectDist, worktreeDist := c.calculateDistribution(sessions)

	// Calculate trends
	dailyTrends, err := c.calculateDailyTrends(ctx, timeRange, project, worktree)
	if err != nil {
		// Don't fail completely, log and continue
		fmt.Printf("Failed to calculate daily trends: %v\n", err)
		dailyTrends = []DailyMetric{}
	}

	weeklyTrends, err := c.calculateWeeklyTrends(ctx, timeRange, project, worktree)
	if err != nil {
		// Don't fail completely, log and continue
		fmt.Printf("Failed to calculate weekly trends: %v\n", err)
		weeklyTrends = []WeeklyMetric{}
	}

	// Combine all metrics
	metrics := &Metrics{
		// Session metrics
		TotalSessions:      sessionMetrics.TotalSessions,
		AverageSessionTime: sessionMetrics.AverageSessionTime,
		SessionFrequency:   sessionMetrics.SessionFrequency,

		// Activity metrics
		ActiveTime:        activityMetrics.ActiveTime,
		IdleTime:          activityMetrics.IdleTime,
		ProductivityRatio: activityMetrics.ProductivityRatio,

		// Distribution
		ProjectDistribution: projectDist,
		WorktreeUsage:       worktreeDist,

		// Trends
		DailyTrends:  dailyTrends,
		WeeklyTrends: weeklyTrends,

		// Metadata
		ComputedAt: time.Now(),
		TimeRange:  timeRange,
	}

	return metrics, nil
}

// SessionMetrics holds basic session metrics
type SessionMetrics struct {
	TotalSessions      int64
	AverageSessionTime time.Duration
	SessionFrequency   float64
	TotalDuration      time.Duration
}

// ActivityMetrics holds activity-related metrics
type ActivityMetrics struct {
	ActiveTime        time.Duration
	IdleTime          time.Duration
	ProductivityRatio float64
}

// calculateSessionMetrics calculates basic session statistics
func (c *DefaultMetricCalculator) calculateSessionMetrics(sessions []*storage.Session, timeRange TimeRange) *SessionMetrics {
	metrics := &SessionMetrics{
		TotalSessions: int64(len(sessions)),
	}

	if len(sessions) == 0 {
		return metrics
	}

	// Calculate total duration and average
	var totalDuration time.Duration
	for _, session := range sessions {
		sessionDuration := session.UpdatedAt.Sub(session.CreatedAt)
		totalDuration += sessionDuration
	}

	metrics.TotalDuration = totalDuration
	metrics.AverageSessionTime = totalDuration / time.Duration(len(sessions))

	// Calculate session frequency (sessions per day)
	daysDiff := timeRange.Duration().Hours() / 24
	if daysDiff > 0 {
		metrics.SessionFrequency = float64(len(sessions)) / daysDiff
	}

	return metrics
}

// calculateActivityMetrics calculates activity and productivity metrics
func (c *DefaultMetricCalculator) calculateActivityMetrics(ctx context.Context, timeRange TimeRange, project, worktree string) (*ActivityMetrics, error) {
	// Get relevant events
	eventTypes := []string{EventTypeStateChange, EventTypeActivity, EventTypeIdle}
	filter := storage.EventFilter{
		EventTypes: eventTypes,
		Since:      timeRange.Start,
		Until:      timeRange.End,
	}

	events, err := c.storage.Events().GetByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	metrics := &ActivityMetrics{}

	// Group events by session and calculate active/idle time
	sessionEvents := make(map[string][]*storage.SessionEvent)
	for _, event := range events {
		sessionEvents[event.SessionID] = append(sessionEvents[event.SessionID], event)
	}

	for sessionID, sessionEventList := range sessionEvents {
		// Filter by project/worktree if specified
		if project != "" || worktree != "" {
			if !c.sessionMatchesFilter(ctx, sessionID, project, worktree) {
				continue
			}
		}

		// Sort events by timestamp
		sort.Slice(sessionEventList, func(i, j int) bool {
			return sessionEventList[i].Timestamp.Before(sessionEventList[j].Timestamp)
		})

		// Calculate active/idle periods for this session
		sessionActive, sessionIdle := c.calculateSessionActivity(sessionEventList)
		metrics.ActiveTime += sessionActive
		metrics.IdleTime += sessionIdle
	}

	// Calculate productivity ratio
	totalTime := metrics.ActiveTime + metrics.IdleTime
	if totalTime > 0 {
		metrics.ProductivityRatio = float64(metrics.ActiveTime) / float64(totalTime)
	}

	return metrics, nil
}

// calculateSessionActivity calculates active and idle time for a single session
func (c *DefaultMetricCalculator) calculateSessionActivity(events []*storage.SessionEvent) (activeTime, idleTime time.Duration) {
	if len(events) < 2 {
		return 0, 0
	}

	// Track state transitions and calculate time in each state
	currentState := "unknown"
	lastTransition := events[0].Timestamp

	for i, event := range events {
		if event.EventType == EventTypeStateChange {
			if newStateData, ok := event.Data["new_state"].(string); ok {
				// Calculate time spent in previous state
				timeDiff := event.Timestamp.Sub(lastTransition)
				
				switch currentState {
				case "busy":
					activeTime += timeDiff
				case "idle", "waiting":
					idleTime += timeDiff
				}

				// Update current state
				currentState = newStateData
				lastTransition = event.Timestamp
			}
		} else if event.EventType == EventTypeActivity {
			if durationMs, ok := event.Data["duration_ms"].(float64); ok {
				activeTime += time.Duration(durationMs) * time.Millisecond
			}
		} else if event.EventType == EventTypeIdle {
			if durationMs, ok := event.Data["duration_ms"].(float64); ok {
				idleTime += time.Duration(durationMs) * time.Millisecond
			}
		}

		// Handle last event - assume current state continues for some time
		if i == len(events)-1 {
			// Assume the session continued for at least 5 minutes after last event
			remainingTime := 5 * time.Minute
			switch currentState {
			case "busy":
				activeTime += remainingTime
			case "idle", "waiting":
				idleTime += remainingTime
			}
		}
	}

	return activeTime, idleTime
}

// sessionMatchesFilter checks if a session matches project/worktree filter
func (c *DefaultMetricCalculator) sessionMatchesFilter(ctx context.Context, sessionID, project, worktree string) bool {
	session, err := c.storage.Sessions().GetByID(ctx, sessionID)
	if err != nil {
		return false
	}

	if project != "" && session.Project != project {
		return false
	}

	if worktree != "" && session.Worktree != worktree {
		return false
	}

	return true
}

// calculateDistribution calculates project and worktree usage distribution
func (c *DefaultMetricCalculator) calculateDistribution(sessions []*storage.Session) (map[string]int, map[string]int) {
	projectDist := make(map[string]int)
	worktreeDist := make(map[string]int)

	for _, session := range sessions {
		if session.Project != "" {
			projectDist[session.Project]++
		}
		if session.Worktree != "" {
			worktreeDist[session.Worktree]++
		}
	}

	return projectDist, worktreeDist
}

// calculateDailyTrends calculates daily metrics trends
func (c *DefaultMetricCalculator) calculateDailyTrends(ctx context.Context, timeRange TimeRange, project, worktree string) ([]DailyMetric, error) {
	trends := []DailyMetric{}
	current := timeRange.Start

	for current.Before(timeRange.End) || current.Equal(timeRange.End) {
		dayStart := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
		dayEnd := dayStart.Add(24 * time.Hour).Add(-time.Nanosecond)

		dayRange := TimeRange{Start: dayStart, End: dayEnd}
		dayMetric, err := c.calculateDailyMetric(ctx, current, dayRange, project, worktree)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate daily metric for %s: %w", current.Format("2006-01-02"), err)
		}

		trends = append(trends, *dayMetric)
		current = current.AddDate(0, 0, 1)
	}

	return trends, nil
}

// calculateWeeklyTrends calculates weekly metrics trends
func (c *DefaultMetricCalculator) calculateWeeklyTrends(ctx context.Context, timeRange TimeRange, project, worktree string) ([]WeeklyMetric, error) {
	trends := []WeeklyMetric{}
	current := timeRange.Start

	// Align to start of week (Monday)
	for current.Weekday() != time.Monday {
		current = current.AddDate(0, 0, -1)
	}

	for current.Before(timeRange.End) {
		weekEnd := current.AddDate(0, 0, 6)
		if weekEnd.After(timeRange.End) {
			weekEnd = timeRange.End
		}

		weekMetric, err := c.calculateWeeklyMetric(ctx, current, weekEnd, project, worktree)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate weekly metric for %s: %w", current.Format("2006-01-02"), err)
		}

		trends = append(trends, *weekMetric)
		current = current.AddDate(0, 0, 7)
	}

	return trends, nil
}

// calculateDailyMetric calculates metrics for a single day
func (c *DefaultMetricCalculator) calculateDailyMetric(ctx context.Context, date time.Time, timeRange TimeRange, project, worktree string) (*DailyMetric, error) {
	// Get sessions for the day
	filter := storage.SessionFilter{
		Since:     timeRange.Start,
		Until:     timeRange.End,
		Project:   project,
		Worktree:  worktree,
		SortBy:    "created_at",
		SortOrder: "asc",
	}

	sessions, err := c.storage.Sessions().List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	metric := &DailyMetric{
		Date:          date,
		TotalSessions: len(sessions),
		Projects:      make(map[string]int),
		Worktrees:     make(map[string]int),
	}

	// Calculate session metrics
	sessionMetrics := c.calculateSessionMetrics(sessions, timeRange)
	metric.TotalDuration = sessionMetrics.TotalDuration
	metric.AverageDuration = sessionMetrics.AverageSessionTime

	// Count projects and worktrees
	for _, session := range sessions {
		if session.Project != "" {
			metric.Projects[session.Project]++
		}
		if session.Worktree != "" {
			metric.Worktrees[session.Worktree]++
		}
	}

	// Calculate activity time
	activityMetrics, err := c.calculateActivityMetrics(ctx, timeRange, project, worktree)
	if err != nil {
		// Don't fail, just log
		fmt.Printf("Failed to calculate activity for %s: %v\n", date.Format("2006-01-02"), err)
	} else {
		metric.ActiveTime = activityMetrics.ActiveTime
		metric.IdleTime = activityMetrics.IdleTime
	}

	return metric, nil
}

// calculateWeeklyMetric calculates metrics for a single week
func (c *DefaultMetricCalculator) calculateWeeklyMetric(ctx context.Context, weekStart, weekEnd time.Time, project, worktree string) (*WeeklyMetric, error) {
	// Get daily metrics for the week
	weekRange := TimeRange{Start: weekStart, End: weekEnd}
	dailyMetrics, err := c.calculateDailyTrends(ctx, weekRange, project, worktree)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily metrics: %w", err)
	}

	metric := &WeeklyMetric{
		WeekStart:       weekStart,
		WeekEnd:         weekEnd,
		DailyBreakdown:  dailyMetrics,
		TopProjects:     make(map[string]int),
		TopWorktrees:    make(map[string]int),
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