# Phase 5.2 Implementation Plan: Session History & Analytics

## Overview

Phase 5.2 builds upon the solid data persistence foundation from Phase 5.1 to provide comprehensive session tracking, analysis, and insights. This phase transforms the SQLite storage infrastructure into a valuable analytics platform for development productivity.

**Goal:** Enable automatic session activity tracking, provide meaningful development analytics, and deliver actionable insights through CLI and TUI interfaces.

## Architecture & Strategy

### Data Flow Architecture
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ ProcessManager  │───▶│ Event Collector  │───▶│ SQLite Storage  │
│ (existing)      │    │ (background)     │    │ (Phase 5.1)     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Hooks System    │───▶│ Analytics Engine │◀───│ Query Interface │
│ (existing)      │    │ (new)            │    │ (new)           │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
┌─────────────────┐    ┌──────────────────┐
│ CLI Commands    │───▶│ TUI Dashboard    │
│ (new)           │    │ (enhanced)       │
└─────────────────┘    └──────────────────┘
```

### Key Design Principles

1. **Event-Driven Architecture**: Use Go channels for non-blocking event collection
2. **Performance First**: Minimal overhead on development workflow (<5% CPU)
3. **Incremental Value**: Provide insights from day one of usage
4. **Seamless Integration**: Work transparently with existing session management
5. **Data Quality**: Ensure accurate and reliable analytics data

## Implementation Phases

### Phase 1: Event Collection Foundation

**Duration:** Week 1  
**Goal:** Establish automatic background monitoring of session activities

#### 1.1 Extend ProcessManager Integration

**File:** `internal/claude/process.go`

Add event emission capabilities to existing ProcessManager:

```go
// Add to ProcessManager struct
type ProcessManager struct {
    // ... existing fields
    eventChan chan<- AnalyticsEvent
}

// New event types
type AnalyticsEvent struct {
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    SessionID string                 `json:"session_id"`
    Data      map[string]interface{} `json:"data"`
}
```

**Tasks:**
- Add event emission to state change handlers
- Create structured event types for all Claude Code states
- Implement non-blocking event channels with buffering
- Add configuration option to enable/disable event emission

#### 1.2 Create Analytics Collector Service

**File:** `internal/analytics/collector.go`

```go
type Collector struct {
    storage    storage.Storage
    eventChan  <-chan AnalyticsEvent
    config     *CollectorConfig
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

type CollectorConfig struct {
    PollInterval    time.Duration `yaml:"poll_interval" default:"30s"`
    BufferSize      int           `yaml:"buffer_size" default:"1000"`
    BatchSize       int           `yaml:"batch_size" default:"50"`
    EnableMetrics   bool          `yaml:"enable_metrics" default:"true"`
    RetentionDays   int           `yaml:"retention_days" default:"90"`
}
```

**Tasks:**
- Implement background goroutine with graceful shutdown
- Add event batching for database efficiency
- Create automatic session lifecycle tracking
- Implement retry logic for failed database writes
- Add health checking and recovery mechanisms

#### 1.3 Hooks System Integration

**File:** `internal/analytics/hooks.go`

Integrate with existing hooks system in `internal/hooks/`:

```go
type HooksCollector struct {
    collector *Collector
    config    *HooksConfig
}

func (hc *HooksCollector) OnStateChange(event hooks.StateChangeEvent) error {
    analyticsEvent := AnalyticsEvent{
        Type:      "claude_state_change",
        Timestamp: time.Now(),
        SessionID: event.SessionID,
        Data: map[string]interface{}{
            "old_state": event.OldState,
            "new_state": event.NewState,
            "worktree":  event.Worktree,
            "branch":    event.Branch,
        },
    }
    return hc.collector.CollectEvent(analyticsEvent)
}
```

**Tasks:**
- Hook into existing state change events
- Capture worktree switches and branch changes
- Track session start/stop/resume events
- Record activity patterns and idle detection

#### 1.4 Configuration Integration

**File:** `internal/config/config.go`

Add analytics configuration to existing config system:

```go
type Config struct {
    // ... existing fields
    Analytics AnalyticsConfig `yaml:"analytics"`
}

type AnalyticsConfig struct {
    Enabled         bool            `yaml:"enabled" default:"true"`
    Collector       CollectorConfig `yaml:"collector"`
    Retention       RetentionConfig `yaml:"retention"`
    Performance     PerformanceConfig `yaml:"performance"`
}
```

**Tasks:**
- Add analytics section to configuration schema
- Implement configuration validation
- Create migration for existing config files
- Add CLI flags for analytics options

### Phase 2: Data Processing & Analytics Engine

**Duration:** Week 2  
**Goal:** Transform raw events into actionable insights

#### 2.1 Analytics Engine Core

**File:** `internal/analytics/engine.go`

```go
type Engine struct {
    storage storage.Storage
    cache   Cache
    config  *EngineConfig
}

type EngineConfig struct {
    CacheSize       int           `yaml:"cache_size" default:"1000"`
    CacheTTL        time.Duration `yaml:"cache_ttl" default:"5m"`
    BatchProcessing bool          `yaml:"batch_processing" default:"true"`
    PrecomputeDaily bool          `yaml:"precompute_daily" default:"true"`
}
```

**Tasks:**
- Implement data aggregation pipelines
- Create metric calculation algorithms
- Add caching layer for performance optimization
- Implement batch processing for heavy computations

#### 2.2 Core Metrics Implementation

**File:** `internal/analytics/metrics.go`

Define comprehensive metrics system:

```go
type Metrics struct {
    // Session Metrics
    TotalSessions       int64         `json:"total_sessions"`
    AverageSessionTime  time.Duration `json:"average_session_time"`
    SessionFrequency    float64       `json:"sessions_per_day"`
    
    // Productivity Metrics
    ActiveTime          time.Duration `json:"active_time"`
    IdleTime            time.Duration `json:"idle_time"`
    ProductivityRatio   float64       `json:"productivity_ratio"`
    
    // Project Metrics
    ProjectDistribution map[string]int `json:"project_distribution"`
    WorktreeUsage       map[string]int `json:"worktree_usage"`
    
    // Trends
    DailyTrends         []DailyMetric  `json:"daily_trends"`
    WeeklyTrends        []WeeklyMetric `json:"weekly_trends"`
}

type MetricCalculator interface {
    Calculate(ctx context.Context, timeRange TimeRange) (*Metrics, error)
    CalculateForProject(ctx context.Context, project string, timeRange TimeRange) (*Metrics, error)
    CalculateForWorktree(ctx context.Context, worktree string, timeRange TimeRange) (*Metrics, error)
}
```

**Tasks:**
- Implement session duration and frequency calculations
- Create development productivity indicators
- Add Claude Code usage statistics
- Build project and worktree activity analysis
- Implement trend analysis and pattern detection

#### 2.3 Database Optimization

**File:** `internal/storage/sqlite/analytics_views.sql`

Create optimized SQL views for common analytics queries:

```sql
-- Session summary view
CREATE VIEW session_analytics AS
SELECT 
    s.project,
    s.worktree,
    s.branch,
    COUNT(*) as session_count,
    AVG(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as avg_duration_minutes,
    SUM(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as total_duration_minutes,
    DATE(s.created_at) as session_date
FROM sessions s
GROUP BY s.project, s.worktree, s.branch, DATE(s.created_at);

-- Daily activity view
CREATE VIEW daily_activity AS
SELECT 
    DATE(se.timestamp) as activity_date,
    se.session_id,
    s.project,
    s.worktree,
    COUNT(CASE WHEN se.event_type = 'state_change' AND 
               JSON_EXTRACT(se.data, '$.new_state') = 'busy' THEN 1 END) as busy_count,
    COUNT(CASE WHEN se.event_type = 'state_change' AND 
               JSON_EXTRACT(se.data, '$.new_state') = 'idle' THEN 1 END) as idle_count
FROM session_events se
JOIN sessions s ON se.session_id = s.id
GROUP BY DATE(se.timestamp), se.session_id, s.project, s.worktree;
```

**File:** `internal/storage/sqlite/analytics_indexes.sql`

Add performance indexes:

```sql
-- Indexes for fast analytics queries
CREATE INDEX idx_sessions_project_date ON sessions(project, DATE(created_at));
CREATE INDEX idx_sessions_worktree_date ON sessions(worktree, DATE(created_at));
CREATE INDEX idx_events_session_type_time ON session_events(session_id, event_type, timestamp);
CREATE INDEX idx_events_type_time ON session_events(event_type, timestamp);
```

**Tasks:**
- Create SQL views for common analytics patterns
- Add performance indexes for fast lookups
- Implement data retention and cleanup procedures
- Optimize queries for large datasets (>10k sessions)

#### 2.4 Performance Features

**File:** `internal/analytics/cache.go`

Implement caching layer:

```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
}

type MemoryCache struct {
    data    map[string]cacheItem
    mutex   sync.RWMutex
    cleanup time.Duration
}

type cacheItem struct {
    value   interface{}
    expires time.Time
}
```

**Tasks:**
- Implement in-memory caching with TTL
- Add cache warmup for frequently accessed data
- Create cache invalidation strategies
- Implement graceful degradation under memory pressure

### Phase 3: CLI Interface & Reporting

**Duration:** Week 3  
**Goal:** Provide accessible analytics through command-line interface

#### 3.1 Analytics Command Structure

**File:** `cmd/ccmgr-ultra/analytics.go`

```go
var analyticsCmd = &cobra.Command{
    Use:   "analytics",
    Short: "View and analyze session data",
    Long:  `Access comprehensive analytics about your development sessions, productivity patterns, and project usage.`,
}

// Subcommands
var (
    analyticsDashboardCmd    = &cobra.Command{...}  // Interactive TUI dashboard
    analyticsSummaryCmd      = &cobra.Command{...}  // Quick overview statistics
    analyticsSessionsCmd     = &cobra.Command{...}  // Detailed session analysis
    analyticsProductivityCmd = &cobra.Command{...}  // Development patterns and insights
    analyticsExportCmd       = &cobra.Command{...}  // Data export functionality
)
```

Command structure:
```
ccmgr-ultra analytics
├── dashboard              # Interactive TUI dashboard
├── summary [options]      # Quick overview statistics
├── sessions [filter]      # Detailed session analysis
├── productivity [period]  # Development patterns and insights
└── export [format]        # Data export (JSON, CSV, etc.)
```

**Tasks:**
- Create analytics command hierarchy
- Implement flag parsing for time ranges and filters
- Add project and worktree-specific analytics
- Create help text and usage examples

#### 3.2 Dashboard Implementation

**File:** `cmd/ccmgr-ultra/analytics_dashboard.go`

Interactive TUI dashboard:

```go
type DashboardModel struct {
    engine      *analytics.Engine
    metrics     *analytics.Metrics
    timeRange   analytics.TimeRange
    activeView  string
    viewport    viewport.Model
    spinner     spinner.Model
    loading     bool
}

type DashboardView string

const (
    OverviewView      DashboardView = "overview"
    SessionsView      DashboardView = "sessions"
    ProductivityView  DashboardView = "productivity"
    ProjectsView      DashboardView = "projects"
)
```

**Tasks:**
- Build real-time session monitoring interface
- Create historical trend visualization using ASCII charts
- Add interactive filtering and drill-down capabilities
- Implement keyboard navigation and controls

#### 3.3 Report Generation

**File:** `cmd/ccmgr-ultra/analytics_reports.go`

```go
type ReportGenerator struct {
    engine *analytics.Engine
    config *ReportConfig
}

type ReportConfig struct {
    Format     string        `json:"format"`     // table, json, csv
    TimeRange  string        `json:"time_range"` // today, week, month, custom
    GroupBy    string        `json:"group_by"`   // project, worktree, day
    Filters    ReportFilters `json:"filters"`
}

type ReportFilters struct {
    Projects  []string `json:"projects,omitempty"`
    Worktrees []string `json:"worktrees,omitempty"`
    Since     string   `json:"since,omitempty"`
    Until     string   `json:"until,omitempty"`
}
```

**Tasks:**
- Implement time-range filtering (day, week, month, custom)
- Create project and worktree-specific reports
- Add export capabilities (JSON, CSV, table formats)
- Build customizable output formatting

#### 3.4 TUI Integration

**File:** `internal/tui/analytics.go`

Enhance existing TUI with analytics:

```go
type AnalyticsScreen struct {
    app       *tui.App
    engine    *analytics.Engine
    model     AnalyticsModel
    viewport  viewport.Model
    table     table.Model
}

type AnalyticsModel struct {
    CurrentMetrics  *analytics.Metrics
    HistoricalData  []analytics.DailyMetric
    SelectedPeriod  string
    SelectedProject string
    Loading         bool
    Error           string
}
```

**Tasks:**
- Add analytics screen to existing TUI navigation
- Create live dashboard for active sessions
- Implement historical data browsing
- Add visual charts using ASCII art and Unicode characters

### Phase 4: Polish & Integration

**Duration:** Week 4  
**Goal:** Optimize performance and integrate seamlessly with existing workflows

#### 4.1 Performance Optimization

**File:** `internal/analytics/performance.go`

```go
type PerformanceMonitor struct {
    collector   *Collector
    engine      *Engine
    metrics     *PerformanceMetrics
    thresholds  *PerformanceThresholds
}

type PerformanceMetrics struct {
    CPUUsage        float64       `json:"cpu_usage"`
    MemoryUsage     int64         `json:"memory_usage_mb"`
    QueryTimes      []time.Duration `json:"query_times"`
    EventsPerSecond float64       `json:"events_per_second"`
    CacheHitRatio   float64       `json:"cache_hit_ratio"`
}

type PerformanceThresholds struct {
    MaxCPUUsage     float64       `json:"max_cpu_usage" default:"5.0"`
    MaxMemoryUsage  int64         `json:"max_memory_usage_mb" default:"100"`
    MaxQueryTime    time.Duration `json:"max_query_time" default:"100ms"`
}
```

**Tasks:**
- Monitor and reduce background CPU usage to <5%
- Optimize database queries for sub-100ms response times
- Implement efficient data structures and algorithms
- Add memory usage monitoring and optimization
- Create performance alerts and auto-tuning

#### 4.2 Testing & Validation

**File:** `internal/analytics/testing.go`

Comprehensive testing framework:

```go
type TestSuite struct {
    storage    storage.Storage
    collector  *Collector
    engine     *Engine
    testData   *TestDataGenerator
}

type TestDataGenerator struct {
    sessionCount  int
    eventCount    int
    timeSpan      time.Duration
    projects      []string
    worktrees     []string
}
```

**Tasks:**
- Create unit tests for all analytics components
- Implement integration tests for background processes
- Add performance benchmarks and load testing
- Create test data generators for various scenarios
- Implement regression testing for analytics accuracy

#### 4.3 Documentation & Examples

**Files:**
- `docs/analytics-guide.md`
- `docs/analytics-configuration.md`
- `docs/analytics-api.md`

**Tasks:**
- Create comprehensive usage guides
- Document configuration options and tuning
- Provide example workflows and use cases
- Create troubleshooting and FAQ sections
- Add API documentation for extensibility

#### 4.4 Workflow Integration

**File:** `internal/analytics/integration.go`

```go
type WorkflowIntegrator struct {
    sessionManager *tmux.SessionManager
    collector      *Collector
    engine         *analytics.Engine
    hooks          *hooks.Manager
}

func (wi *WorkflowIntegrator) OnSessionStart(session *tmux.Session) error {
    // Automatically start analytics collection
    return wi.collector.StartTracking(session.ID)
}

func (wi *WorkflowIntegrator) OnSessionStop(session *tmux.Session) error {
    // Finalize analytics data
    return wi.collector.StopTracking(session.ID)
}
```

**Tasks:**
- Ensure seamless operation with existing session management
- Integrate with current hook scripts and environment variables
- Maintain backward compatibility with existing configurations
- Create migration tools for upgrading from previous versions
- Add analytics insights to existing status displays

## New Package Structure

```
internal/analytics/
├── collector.go           # Background event collection service
├── engine.go              # Analytics processing and aggregation
├── metrics.go             # Metric definitions and calculations
├── queries.go             # Pre-built analytics queries
├── cache.go               # Caching layer implementation
├── hooks.go               # Integration with hooks system
├── performance.go         # Performance monitoring and optimization
├── integration.go         # Workflow integration utilities
└── testing.go             # Testing utilities and data generators

cmd/ccmgr-ultra/
├── analytics.go           # Main analytics command
├── analytics_dashboard.go # TUI dashboard implementation
├── analytics_reports.go   # Report generation
└── analytics_export.go    # Data export functionality

internal/storage/sqlite/
├── analytics_views.sql    # SQL views for fast analytics
├── analytics_indexes.sql  # Performance indexes
└── analytics_migrations/  # Database schema updates
    ├── 002_analytics_views.sql
    └── 003_analytics_indexes.sql

internal/tui/
├── analytics.go           # Analytics TUI screens
└── analytics_charts.go    # ASCII chart rendering

docs/
├── analytics-guide.md     # User guide and examples
├── analytics-configuration.md # Configuration reference
└── analytics-api.md       # API documentation
```

## Configuration Schema

**File:** `internal/config/analytics.yaml`

```yaml
analytics:
  enabled: true
  
  collector:
    poll_interval: "30s"
    buffer_size: 1000
    batch_size: 50
    enable_metrics: true
    retention_days: 90
  
  engine:
    cache_size: 1000
    cache_ttl: "5m"
    batch_processing: true
    precompute_daily: true
  
  performance:
    max_cpu_usage: 5.0
    max_memory_usage_mb: 100
    max_query_time: "100ms"
    enable_monitoring: true
  
  retention:
    session_events_days: 90
    aggregated_data_days: 365
    cleanup_interval: "24h"
    enable_auto_cleanup: true
```

## Database Schema Extensions

**File:** `internal/storage/sqlite/migrations/002_analytics_views.sql`

```sql
-- Add analytics-specific columns to existing tables
ALTER TABLE sessions ADD COLUMN analytics_data TEXT DEFAULT '{}';
ALTER TABLE session_events ADD COLUMN processed_at TIMESTAMP;

-- Create analytics aggregation tables
CREATE TABLE daily_session_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL,
    project TEXT,
    worktree TEXT,
    session_count INTEGER DEFAULT 0,
    total_duration_minutes INTEGER DEFAULT 0,
    avg_duration_minutes REAL DEFAULT 0,
    active_time_minutes INTEGER DEFAULT 0,
    idle_time_minutes INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, project, worktree)
);

CREATE TABLE productivity_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    metric_value REAL NOT NULL,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id),
    INDEX idx_productivity_session_type (session_id, metric_type)
);
```

## Success Metrics

### Performance Targets
- **Background monitoring CPU usage:** <5%
- **Analytics query response time:** <100ms for most queries, <500ms for complex aggregations
- **Data collection uptime:** >99% availability
- **Memory usage growth:** <10MB over 24-hour period
- **Database query efficiency:** <1ms for cached queries, <50ms for indexed queries

### User Experience Goals
- **Zero-configuration setup:** Analytics work automatically without user intervention
- **Immediate value:** Useful insights available within first day of usage
- **Intuitive interface:** CLI commands follow existing patterns and conventions
- **Seamless integration:** No disruption to existing development workflows
- **Fast feedback:** Dashboard updates within 5 seconds of activity

### Data Quality Standards
- **Event capture rate:** 100% capture of session lifecycle events
- **Data consistency:** <0.1% data loss during system restarts or failures
- **Time accuracy:** Millisecond precision for all timestamps
- **Data integrity:** Full referential integrity between sessions and events
- **Analytics accuracy:** <1% variance in calculated metrics vs manual verification

## Risk Mitigation Strategies

### Performance Risks
- **Background overhead:** Configurable monitoring intervals with automatic adjustment
- **Database performance:** Query optimization, indexing, and connection pooling
- **Memory leaks:** Regular garbage collection monitoring and cache size limits
- **Storage growth:** Automatic data retention policies and cleanup procedures

### Data Integrity Risks
- **Event loss:** Buffering, retry logic, and persistent queues for reliability
- **Database corruption:** Regular backups, transaction logging, and recovery procedures
- **Schema evolution:** Versioned migrations with rollback capabilities
- **Clock synchronization:** NTP verification and timestamp validation

### Integration Risks
- **Compatibility:** Comprehensive testing with existing components and configurations
- **Migration issues:** Gradual rollout with fallback to previous versions
- **Dependency failures:** Graceful degradation when analytics services are unavailable
- **Configuration conflicts:** Validation and automatic conflict resolution

## Implementation Timeline

### Week 1: Foundation (Event Collection)
- **Day 1-2:** ProcessManager integration and event emission
- **Day 3-4:** Collector service implementation and configuration
- **Day 5:** Hooks system integration and testing

### Week 2: Analytics Engine
- **Day 1-2:** Core engine and metrics calculation
- **Day 3-4:** Database optimization and caching
- **Day 5:** Performance tuning and validation

### Week 3: User Interface
- **Day 1-2:** CLI command structure and basic reports
- **Day 3-4:** TUI dashboard and interactive features
- **Day 5:** Export functionality and formatting

### Week 4: Integration & Polish
- **Day 1-2:** Performance optimization and monitoring
- **Day 3-4:** Testing, documentation, and examples
- **Day 5:** Final integration and deployment preparation

## Post-Implementation Considerations

### Future Enhancements
- **Machine learning insights:** Pattern recognition and productivity recommendations
- **Integration with external tools:** Export to time tracking and project management systems
- **Advanced visualizations:** Web-based dashboard for team analytics
- **Predictive analytics:** Forecasting and capacity planning features

### Maintenance Requirements
- **Regular performance monitoring:** Weekly performance reviews and optimization
- **Data retention management:** Monthly cleanup and archival procedures
- **Security updates:** Quarterly security review and dependency updates
- **Feature evolution:** Continuous user feedback collection and improvement planning

This implementation plan provides a comprehensive roadmap for building robust session history and analytics capabilities that will significantly enhance the value of ccmgr-ultra's data persistence foundation.