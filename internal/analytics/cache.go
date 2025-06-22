package analytics

import (
	"fmt"
	"sync"
	"time"
)

// Cache defines the interface for caching analytics data
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// cacheItem represents a single cache entry
type cacheItem struct {
	value   interface{}
	expires time.Time
}

// isExpired checks if the cache item has expired
func (item *cacheItem) isExpired() bool {
	return time.Now().After(item.expires)
}

// MemoryCache implements an in-memory cache with TTL support
type MemoryCache struct {
	data     map[string]*cacheItem
	mutex    sync.RWMutex
	maxSize  int
	cleanup  time.Duration
	
	// Statistics
	hits   int64
	misses int64
	
	// Background cleanup
	stopCleanup chan struct{}
	wg          sync.WaitGroup
}

// NewMemoryCache creates a new memory cache with the specified max size and default TTL
func NewMemoryCache(maxSize int, defaultTTL time.Duration) *MemoryCache {
	cache := &MemoryCache{
		data:        make(map[string]*cacheItem),
		maxSize:     maxSize,
		cleanup:     15 * time.Minute, // Run cleanup every 15 minutes
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	cache.wg.Add(1)
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.data[key]
	if !exists {
		c.misses++
		return nil, false
	}

	if item.isExpired() {
		c.misses++
		// Don't delete here to avoid write lock upgrade
		// Let cleanup handle expired items
		return nil, false
	}

	c.hits++
	return item.value, true
}

// Set stores a value in the cache with the specified TTL
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict items to make space
	if len(c.data) >= c.maxSize {
		c.evictOldest()
	}

	c.data[key] = &cacheItem{
		value:   value,
		expires: time.Now().Add(ttl),
	}
}

// Delete removes a key from the cache
func (c *MemoryCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
}

// Clear removes all items from the cache
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]*cacheItem)
}

// Len returns the number of items in the cache
func (c *MemoryCache) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.data)
}

// GetHits returns the number of cache hits
func (c *MemoryCache) GetHits() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.hits
}

// GetMisses returns the number of cache misses
func (c *MemoryCache) GetMisses() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.misses
}

// GetHitRatio returns the cache hit ratio
func (c *MemoryCache) GetHitRatio() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	total := c.hits + c.misses
	if total == 0 {
		return 0
	}
	return float64(c.hits) / float64(total)
}

// Cleanup removes expired items from the cache
func (c *MemoryCache) Cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if now.After(item.expires) {
			delete(c.data, key)
		}
	}
}

// Stop stops the background cleanup goroutine
func (c *MemoryCache) Stop() {
	close(c.stopCleanup)
	c.wg.Wait()
}

// cleanupLoop runs periodic cleanup in the background
func (c *MemoryCache) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCleanup:
			return
		case <-ticker.C:
			c.Cleanup()
		}
	}
}

// evictOldest removes the oldest item from the cache (simple LRU)
// Note: This is called with write lock already held
func (c *MemoryCache) evictOldest() {
	if len(c.data) == 0 {
		return
	}

	// Find the item with the earliest expiration time
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range c.data {
		if first || item.expires.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.expires
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

// GetStats returns cache statistics
func (c *MemoryCache) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"size":       len(c.data),
		"max_size":   c.maxSize,
		"hits":       c.hits,
		"misses":     c.misses,
		"hit_ratio":  c.GetHitRatio(),
	}
}

// WarmupCache pre-loads frequently accessed data into the cache
type WarmupCache struct {
	cache   Cache
	storage interface{} // Could be storage interface
}

// NewWarmupCache creates a cache with warmup capabilities
func NewWarmupCache(cache Cache) *WarmupCache {
	return &WarmupCache{
		cache: cache,
	}
}

// WarmupDaily pre-loads daily metrics for recent days
func (w *WarmupCache) WarmupDaily(days int) error {
	// Implementation would pre-load common daily metrics
	// This is a placeholder for the warmup logic
	return nil
}

// WarmupWeekly pre-loads weekly metrics for recent weeks
func (w *WarmupCache) WarmupWeekly(weeks int) error {
	// Implementation would pre-load common weekly metrics
	// This is a placeholder for the warmup logic
	return nil
}

// MultiLevelCache implements a multi-level caching strategy
type MultiLevelCache struct {
	l1Cache Cache // Fast, small cache (e.g., in-memory)
	l2Cache Cache // Slower, larger cache (e.g., file-based)
	mutex   sync.RWMutex
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(l1, l2 Cache) *MultiLevelCache {
	return &MultiLevelCache{
		l1Cache: l1,
		l2Cache: l2,
	}
}

// Get retrieves a value from the multi-level cache
func (m *MultiLevelCache) Get(key string) (interface{}, bool) {
	// Try L1 cache first
	if value, found := m.l1Cache.Get(key); found {
		return value, true
	}

	// Try L2 cache
	if value, found := m.l2Cache.Get(key); found {
		// Promote to L1 cache
		m.l1Cache.Set(key, value, 5*time.Minute)
		return value, true
	}

	return nil, false
}

// Set stores a value in the multi-level cache
func (m *MultiLevelCache) Set(key string, value interface{}, ttl time.Duration) {
	// Store in both levels
	m.l1Cache.Set(key, value, ttl)
	m.l2Cache.Set(key, value, ttl*2) // Keep in L2 longer
}

// Delete removes a key from both cache levels
func (m *MultiLevelCache) Delete(key string) {
	m.l1Cache.Delete(key)
	m.l2Cache.Delete(key)
}

// Clear removes all items from both cache levels
func (m *MultiLevelCache) Clear() {
	m.l1Cache.Clear()
	m.l2Cache.Clear()
}

// CacheConfig defines configuration for cache behavior
type CacheConfig struct {
	MaxSize        int           `yaml:"max_size" json:"max_size" default:"1000"`
	DefaultTTL     time.Duration `yaml:"default_ttl" json:"default_ttl" default:"5m"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval" default:"15m"`
	EnableWarmup   bool          `yaml:"enable_warmup" json:"enable_warmup" default:"true"`
	WarmupDays     int           `yaml:"warmup_days" json:"warmup_days" default:"7"`
}

// SetDefaults sets default values for cache configuration
func (c *CacheConfig) SetDefaults() {
	if c.MaxSize == 0 {
		c.MaxSize = 1000
	}
	if c.DefaultTTL == 0 {
		c.DefaultTTL = 5 * time.Minute
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = 15 * time.Minute
	}
	if c.WarmupDays == 0 {
		c.WarmupDays = 7
	}
	c.EnableWarmup = true
}

// Validate validates the cache configuration
func (c *CacheConfig) Validate() error {
	if c.MaxSize < 0 {
		return fmt.Errorf("max size cannot be negative")
	}
	if c.DefaultTTL < 0 {
		return fmt.Errorf("default TTL cannot be negative")
	}
	if c.CleanupInterval < 0 {
		return fmt.Errorf("cleanup interval cannot be negative")
	}
	if c.WarmupDays < 0 {
		return fmt.Errorf("warmup days cannot be negative")
	}
	return nil
}

// NewCacheFromConfig creates a cache instance from configuration
func NewCacheFromConfig(config *CacheConfig) Cache {
	if config == nil {
		config = &CacheConfig{}
		config.SetDefaults()
	}

	return NewMemoryCache(config.MaxSize, config.DefaultTTL)
}