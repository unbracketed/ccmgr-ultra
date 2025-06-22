package analytics

import (
	"context"
	"time"
)

// AnalyticsEvent represents an event for analytics collection
type AnalyticsEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data"`
}

// EventTypes for common analytics events
const (
	EventTypeStateChange    = "claude_state_change"
	EventTypeSessionStart   = "session_start"
	EventTypeSessionStop    = "session_stop"
	EventTypeSessionResume  = "session_resume"
	EventTypeWorktreeSwitch = "worktree_switch"
	EventTypeBranchChange   = "branch_change"
	EventTypeActivity       = "activity"
	EventTypeIdle           = "idle_detection"
)

// EventEmitter defines the interface for emitting analytics events
type EventEmitter interface {
	EmitEvent(event AnalyticsEvent) error
	EmitEventAsync(event AnalyticsEvent)
	IsEnabled() bool
}

// EventCollector defines the interface for collecting analytics events
type EventCollector interface {
	CollectEvent(event AnalyticsEvent) error
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewTimeRange creates a new time range
func NewTimeRange(start, end time.Time) TimeRange {
	return TimeRange{Start: start, End: end}
}

// NewTimeRangeFromDuration creates a time range from now going back by duration
func NewTimeRangeFromDuration(duration time.Duration) TimeRange {
	end := time.Now()
	start := end.Add(-duration)
	return TimeRange{Start: start, End: end}
}

// Contains checks if a time is within the range
func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && !t.After(tr.End)
}

// Duration returns the duration of the time range
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// SessionMetric represents metrics for a session
type SessionMetric struct {
	SessionID    string        `json:"session_id"`
	Duration     time.Duration `json:"duration"`
	ActiveTime   time.Duration `json:"active_time"`
	IdleTime     time.Duration `json:"idle_time"`
	ProjectName  string        `json:"project_name"`
	WorktreeName string        `json:"worktree_name"`
	BranchName   string        `json:"branch_name"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
}

// DailyMetric represents daily aggregated metrics
type DailyMetric struct {
	Date            time.Time         `json:"date"`
	TotalSessions   int               `json:"total_sessions"`
	TotalDuration   time.Duration     `json:"total_duration"`
	AverageDuration time.Duration     `json:"average_duration"`
	ActiveTime      time.Duration     `json:"active_time"`
	IdleTime        time.Duration     `json:"idle_time"`
	Projects        map[string]int    `json:"projects"`
	Worktrees       map[string]int    `json:"worktrees"`
}

// WeeklyMetric represents weekly aggregated metrics
type WeeklyMetric struct {
	WeekStart       time.Time         `json:"week_start"`
	WeekEnd         time.Time         `json:"week_end"`
	TotalSessions   int               `json:"total_sessions"`
	TotalDuration   time.Duration     `json:"total_duration"`
	AverageDuration time.Duration     `json:"average_duration"`
	ActiveTime      time.Duration     `json:"active_time"`
	IdleTime        time.Duration     `json:"idle_time"`
	DailyBreakdown  []DailyMetric     `json:"daily_breakdown"`
	TopProjects     map[string]int    `json:"top_projects"`
	TopWorktrees    map[string]int    `json:"top_worktrees"`
}

// ProductivityMetric represents productivity analysis
type ProductivityMetric struct {
	Date                time.Time     `json:"date"`
	ProductivityRatio   float64       `json:"productivity_ratio"`
	FocusTime           time.Duration `json:"focus_time"`
	DistractionTime     time.Duration `json:"distraction_time"`
	SessionsPerDay      float64       `json:"sessions_per_day"`
	AverageSessionGap   time.Duration `json:"average_session_gap"`
	PeakProductiveHour  int           `json:"peak_productive_hour"`
}

// Metrics represents comprehensive analytics metrics
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
	
	// Computed at
	ComputedAt          time.Time      `json:"computed_at"`
	TimeRange           TimeRange      `json:"time_range"`
}

// EventData convenience functions for creating common event data
func NewStateChangeEventData(oldState, newState, worktree, branch string) map[string]interface{} {
	return map[string]interface{}{
		"old_state": oldState,
		"new_state": newState,
		"worktree":  worktree,
		"branch":    branch,
	}
}

func NewSessionEventData(action, project, worktree, branch, directory string) map[string]interface{} {
	return map[string]interface{}{
		"action":    action,
		"project":   project,
		"worktree":  worktree,
		"branch":    branch,
		"directory": directory,
	}
}

func NewActivityEventData(activityType string, duration time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"activity_type": activityType,
		"duration_ms":   duration.Milliseconds(),
	}
}