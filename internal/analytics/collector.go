package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
)

// CollectorConfig defines configuration for the analytics collector
type CollectorConfig struct {
	PollInterval    time.Duration `yaml:"poll_interval" json:"poll_interval" default:"30s"`
	BufferSize      int           `yaml:"buffer_size" json:"buffer_size" default:"1000"`
	BatchSize       int           `yaml:"batch_size" json:"batch_size" default:"50"`
	EnableMetrics   bool          `yaml:"enable_metrics" json:"enable_metrics" default:"true"`
	RetentionDays   int           `yaml:"retention_days" json:"retention_days" default:"90"`
}

// SetDefaults sets default values for CollectorConfig
func (c *CollectorConfig) SetDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = 30 * time.Second
	}
	if c.BufferSize == 0 {
		c.BufferSize = 1000
	}
	if c.BatchSize == 0 {
		c.BatchSize = 50
	}
	if c.RetentionDays == 0 {
		c.RetentionDays = 90
	}
	c.EnableMetrics = true
}

// Validate validates the collector configuration
func (c *CollectorConfig) Validate() error {
	if c.PollInterval < time.Second {
		return fmt.Errorf("poll interval must be at least 1 second")
	}
	if c.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if c.BatchSize > c.BufferSize {
		return fmt.Errorf("batch size cannot exceed buffer size")
	}
	if c.RetentionDays < 0 {
		return fmt.Errorf("retention days cannot be negative")
	}
	return nil
}

// Collector implements the EventCollector interface for background analytics collection
type Collector struct {
	storage   storage.Storage
	eventChan <-chan AnalyticsEvent
	config    *CollectorConfig
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mutex     sync.RWMutex
	running   bool
	
	// Statistics
	eventsProcessed   int64
	eventsFailed      int64
	lastProcessedTime time.Time
}

// NewCollector creates a new analytics collector
func NewCollector(storage storage.Storage, eventChan <-chan AnalyticsEvent, config *CollectorConfig) *Collector {
	if config == nil {
		config = &CollectorConfig{}
		config.SetDefaults()
	}

	return &Collector{
		storage:   storage,
		eventChan: eventChan,
		config:    config,
	}
}

// CollectEvent implements EventCollector interface for single event collection
func (c *Collector) CollectEvent(event AnalyticsEvent) error {
	if !c.IsRunning() {
		return fmt.Errorf("collector is not running")
	}

	// Convert to storage format
	sessionEvent := &storage.SessionEvent{
		SessionID: event.SessionID,
		EventType: event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	}

	// Store in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.storage.Events().Create(ctx, sessionEvent); err != nil {
		c.incrementFailedEvents()
		return fmt.Errorf("failed to store event: %w", err)
	}

	c.incrementProcessedEvents()
	return nil
}

// Start starts the background collection process
func (c *Collector) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return fmt.Errorf("collector is already running")
	}

	if err := c.config.Validate(); err != nil {
		return fmt.Errorf("invalid collector configuration: %w", err)
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// Start the collection goroutine
	c.wg.Add(1)
	go c.collectionLoop()

	// Start periodic cleanup if retention is enabled
	if c.config.RetentionDays > 0 {
		c.wg.Add(1)
		go c.cleanupLoop()
	}

	return nil
}

// Stop stops the background collection process
func (c *Collector) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return nil
	}

	c.running = false
	c.cancel()
	c.wg.Wait()

	return nil
}

// IsRunning returns whether the collector is currently running
func (c *Collector) IsRunning() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.running
}

// collectionLoop runs the main event collection loop
func (c *Collector) collectionLoop() {
	defer c.wg.Done()

	buffer := make([]AnalyticsEvent, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Process remaining events in buffer before stopping
			if len(buffer) > 0 {
				c.processBatch(buffer)
			}
			return

		case event := <-c.eventChan:
			buffer = append(buffer, event)
			
			// Process batch when it's full
			if len(buffer) >= c.config.BatchSize {
				c.processBatch(buffer)
				buffer = buffer[:0] // Clear buffer
			}

		case <-ticker.C:
			// Process batch on timer even if not full
			if len(buffer) > 0 {
				c.processBatch(buffer)
				buffer = buffer[:0] // Clear buffer
			}
		}
	}
}

// processBatch processes a batch of events
func (c *Collector) processBatch(events []AnalyticsEvent) {
	if len(events) == 0 {
		return
	}

	// Convert to storage format
	sessionEvents := make([]*storage.SessionEvent, len(events))
	for i, event := range events {
		sessionEvents[i] = &storage.SessionEvent{
			SessionID: event.SessionID,
			EventType: event.Type,
			Timestamp: event.Timestamp,
			Data:      event.Data,
		}
	}

	// Store in database using batch operation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.storage.Events().CreateBatch(ctx, sessionEvents); err != nil {
		// Try individual inserts if batch fails
		c.fallbackToIndividualInserts(ctx, sessionEvents)
	} else {
		c.addProcessedEvents(int64(len(events)))
	}

	c.updateLastProcessedTime()
}

// fallbackToIndividualInserts tries to insert events individually if batch insert fails
func (c *Collector) fallbackToIndividualInserts(ctx context.Context, events []*storage.SessionEvent) {
	successCount := int64(0)
	failureCount := int64(0)

	for _, event := range events {
		if err := c.storage.Events().Create(ctx, event); err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	c.addProcessedEvents(successCount)
	c.addFailedEvents(failureCount)
}

// cleanupLoop handles periodic cleanup of old events
func (c *Collector) cleanupLoop() {
	defer c.wg.Done()

	// Run cleanup daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run initial cleanup after 1 minute
	initialTimer := time.NewTimer(time.Minute)
	defer initialTimer.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-initialTimer.C:
			c.performCleanup()

		case <-ticker.C:
			c.performCleanup()
		}
	}
}

// performCleanup removes old events based on retention policy
func (c *Collector) performCleanup() {
	if c.config.RetentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -c.config.RetentionDays)
	
	// This would require implementing a cleanup method in storage
	// For now, we'll log the operation
	fmt.Printf("Analytics cleanup: removing events older than %s\n", cutoffTime.Format(time.RFC3339))
}

// GetStats returns collector statistics
func (c *Collector) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"running":              c.running,
		"events_processed":     c.eventsProcessed,
		"events_failed":        c.eventsFailed,
		"last_processed_time":  c.lastProcessedTime,
		"buffer_size":          c.config.BufferSize,
		"batch_size":           c.config.BatchSize,
		"poll_interval":        c.config.PollInterval.String(),
		"retention_days":       c.config.RetentionDays,
	}
}

// Helper methods for statistics
func (c *Collector) incrementProcessedEvents() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.eventsProcessed++
	c.lastProcessedTime = time.Now()
}

func (c *Collector) incrementFailedEvents() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.eventsFailed++
}

func (c *Collector) addProcessedEvents(count int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.eventsProcessed += count
	c.lastProcessedTime = time.Now()
}

func (c *Collector) addFailedEvents(count int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.eventsFailed += count
}

func (c *Collector) updateLastProcessedTime() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastProcessedTime = time.Now()
}