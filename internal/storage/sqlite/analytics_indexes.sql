-- Additional Analytics Performance Indexes
-- These indexes are optimized for common analytics query patterns

-- Composite indexes for session queries
CREATE INDEX IF NOT EXISTS idx_sessions_project_worktree_date 
ON sessions(project, worktree, DATE(created_at));

CREATE INDEX IF NOT EXISTS idx_sessions_created_updated 
ON sessions(created_at, updated_at);

CREATE INDEX IF NOT EXISTS idx_sessions_last_access_project 
ON sessions(last_access, project);

CREATE INDEX IF NOT EXISTS idx_sessions_branch_project 
ON sessions(branch, project) WHERE branch IS NOT NULL;

-- Event-specific indexes for analytics
CREATE INDEX IF NOT EXISTS idx_events_timestamp_session 
ON session_events(timestamp, session_id);

CREATE INDEX IF NOT EXISTS idx_events_data_state_change 
ON session_events(JSON_EXTRACT(data, '$.new_state')) 
WHERE event_type = 'claude_state_change';

CREATE INDEX IF NOT EXISTS idx_events_data_old_state 
ON session_events(JSON_EXTRACT(data, '$.old_state')) 
WHERE event_type = 'claude_state_change';

CREATE INDEX IF NOT EXISTS idx_events_activity_duration 
ON session_events(JSON_EXTRACT(data, '$.duration_ms')) 
WHERE event_type IN ('activity', 'idle_detection');

-- Time-based indexes for efficient date range queries
CREATE INDEX IF NOT EXISTS idx_sessions_date_only 
ON sessions(DATE(created_at));

CREATE INDEX IF NOT EXISTS idx_sessions_hour 
ON sessions(strftime('%H', created_at));

CREATE INDEX IF NOT EXISTS idx_sessions_weekday 
ON sessions(strftime('%w', created_at));

CREATE INDEX IF NOT EXISTS idx_sessions_month_year 
ON sessions(strftime('%Y-%m', created_at));

-- Event time-based indexes
CREATE INDEX IF NOT EXISTS idx_events_date_only 
ON session_events(DATE(timestamp));

CREATE INDEX IF NOT EXISTS idx_events_hour 
ON session_events(strftime('%H', timestamp));

-- Productivity metrics indexes
CREATE INDEX IF NOT EXISTS idx_productivity_calculated_date 
ON productivity_metrics(DATE(calculated_at));

CREATE INDEX IF NOT EXISTS idx_productivity_value_type 
ON productivity_metrics(metric_value, metric_type);

-- Daily stats indexes for fast aggregation
CREATE INDEX IF NOT EXISTS idx_daily_stats_date_range 
ON daily_session_stats(date) 
WHERE date >= DATE('now', '-90 days');

CREATE INDEX IF NOT EXISTS idx_daily_stats_project_recent 
ON daily_session_stats(project, date) 
WHERE date >= DATE('now', '-30 days');

-- Covering indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_sessions_analytics_covering 
ON sessions(project, worktree, created_at, updated_at, id);

CREATE INDEX IF NOT EXISTS idx_events_analytics_covering 
ON session_events(session_id, event_type, timestamp, data);

-- Partial indexes for non-null values to save space
CREATE INDEX IF NOT EXISTS idx_sessions_project_not_null 
ON sessions(project, created_at) WHERE project IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_worktree_not_null 
ON sessions(worktree, created_at) WHERE worktree IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_branch_not_null 
ON sessions(branch, created_at) WHERE branch IS NOT NULL;

-- Functional indexes for JSON data analysis
CREATE INDEX IF NOT EXISTS idx_events_state_change_transitions 
ON session_events(
    session_id, 
    timestamp, 
    JSON_EXTRACT(data, '$.old_state'), 
    JSON_EXTRACT(data, '$.new_state')
) WHERE event_type = 'claude_state_change';

-- Indexes for specific analytics queries
CREATE INDEX IF NOT EXISTS idx_sessions_duration_calc 
ON sessions(
    project, 
    (julianday(updated_at) - julianday(created_at))
) WHERE updated_at > created_at;

-- Indexes for recent data (last 7 days) - these will be very fast
CREATE INDEX IF NOT EXISTS idx_sessions_recent_week 
ON sessions(created_at, project, worktree) 
WHERE created_at >= DATE('now', '-7 days');

CREATE INDEX IF NOT EXISTS idx_events_recent_week 
ON session_events(timestamp, session_id, event_type) 
WHERE timestamp >= DATE('now', '-7 days');

-- Indexes optimized for dashboard queries
CREATE INDEX IF NOT EXISTS idx_sessions_dashboard_today 
ON sessions(project, worktree, created_at) 
WHERE DATE(created_at) = DATE('now');

CREATE INDEX IF NOT EXISTS idx_sessions_dashboard_yesterday 
ON sessions(project, worktree, created_at) 
WHERE DATE(created_at) = DATE('now', '-1 day');

-- Cleanup indexes for maintenance operations
CREATE INDEX IF NOT EXISTS idx_events_old_data 
ON session_events(timestamp) 
WHERE timestamp < DATE('now', '-90 days');

CREATE INDEX IF NOT EXISTS idx_sessions_old_data 
ON sessions(created_at) 
WHERE created_at < DATE('now', '-365 days');

-- Index for analytics data cleanup
CREATE INDEX IF NOT EXISTS idx_daily_stats_cleanup 
ON daily_session_stats(created_at) 
WHERE created_at < DATE('now', '-365 days');