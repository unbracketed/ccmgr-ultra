-- Analytics Views and Enhancements Migration
-- This migration adds analytics-specific columns and views for efficient analytics queries

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
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);

-- Session summary view for analytics
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

-- Daily activity view for productivity analysis
CREATE VIEW daily_activity AS
SELECT 
    DATE(se.timestamp) as activity_date,
    se.session_id,
    s.project,
    s.worktree,
    COUNT(CASE WHEN se.event_type = 'claude_state_change' AND 
               JSON_EXTRACT(se.data, '$.new_state') = 'busy' THEN 1 END) as busy_count,
    COUNT(CASE WHEN se.event_type = 'claude_state_change' AND 
               JSON_EXTRACT(se.data, '$.new_state') = 'idle' THEN 1 END) as idle_count,
    COUNT(CASE WHEN se.event_type = 'claude_state_change' AND 
               JSON_EXTRACT(se.data, '$.new_state') = 'waiting' THEN 1 END) as waiting_count,
    MIN(se.timestamp) as first_activity,
    MAX(se.timestamp) as last_activity
FROM session_events se
JOIN sessions s ON se.session_id = s.id
GROUP BY DATE(se.timestamp), se.session_id, s.project, s.worktree;

-- Session duration view for quick session analysis
CREATE VIEW session_durations AS
SELECT 
    s.id,
    s.name,
    s.project,
    s.worktree,
    s.branch,
    s.created_at,
    s.updated_at,
    s.last_access,
    julianday(s.updated_at) - julianday(s.created_at) as duration_days,
    (julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as duration_minutes,
    CASE 
        WHEN julianday(s.updated_at) - julianday(s.created_at) > 1 THEN 'long'
        WHEN julianday(s.updated_at) - julianday(s.created_at) > 0.5 THEN 'medium'
        ELSE 'short'
    END as duration_category
FROM sessions s;

-- Project activity summary view
CREATE VIEW project_activity AS
SELECT 
    s.project,
    COUNT(DISTINCT s.id) as total_sessions,
    COUNT(DISTINCT s.worktree) as worktree_count,
    COUNT(DISTINCT s.branch) as branch_count,
    MIN(s.created_at) as first_session,
    MAX(s.updated_at) as last_session,
    AVG(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as avg_session_minutes,
    SUM(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as total_time_minutes
FROM sessions s
WHERE s.project IS NOT NULL AND s.project != ''
GROUP BY s.project;

-- Worktree usage view
CREATE VIEW worktree_usage AS
SELECT 
    s.worktree,
    s.project,
    COUNT(*) as session_count,
    MIN(s.created_at) as first_used,
    MAX(s.updated_at) as last_used,
    AVG(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as avg_session_minutes,
    COUNT(DISTINCT DATE(s.created_at)) as days_used
FROM sessions s
WHERE s.worktree IS NOT NULL AND s.worktree != ''
GROUP BY s.worktree, s.project;

-- Recent activity view (last 30 days)
CREATE VIEW recent_activity AS
SELECT 
    DATE(s.created_at) as activity_date,
    s.project,
    s.worktree,
    COUNT(*) as session_count,
    SUM(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as total_minutes,
    AVG(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as avg_minutes
FROM sessions s
WHERE s.created_at >= DATE('now', '-30 days')
GROUP BY DATE(s.created_at), s.project, s.worktree
ORDER BY activity_date DESC;

-- Event frequency view for debugging and monitoring
CREATE VIEW event_frequency AS
SELECT 
    se.event_type,
    COUNT(*) as event_count,
    COUNT(DISTINCT se.session_id) as unique_sessions,
    MIN(se.timestamp) as first_event,
    MAX(se.timestamp) as last_event,
    AVG(julianday(se.timestamp)) as avg_timestamp_julian
FROM session_events se
GROUP BY se.event_type;

-- State transition analysis view
CREATE VIEW state_transitions AS
SELECT 
    se.session_id,
    s.project,
    s.worktree,
    JSON_EXTRACT(se.data, '$.old_state') as old_state,
    JSON_EXTRACT(se.data, '$.new_state') as new_state,
    se.timestamp,
    LAG(se.timestamp) OVER (PARTITION BY se.session_id ORDER BY se.timestamp) as prev_timestamp,
    (julianday(se.timestamp) - julianday(LAG(se.timestamp) OVER (PARTITION BY se.session_id ORDER BY se.timestamp))) * 24 * 60 as time_in_prev_state_minutes
FROM session_events se
JOIN sessions s ON se.session_id = s.id
WHERE se.event_type = 'claude_state_change'
ORDER BY se.session_id, se.timestamp;

-- Add indexes for performance optimization (referenced in separate file but included here for migration)
CREATE INDEX idx_sessions_project_date ON sessions(project, DATE(created_at));
CREATE INDEX idx_sessions_worktree_date ON sessions(worktree, DATE(created_at));
CREATE INDEX idx_sessions_analytics_data ON sessions(analytics_data) WHERE analytics_data != '{}';
CREATE INDEX idx_events_session_type_time ON session_events(session_id, event_type, timestamp);
CREATE INDEX idx_events_type_time ON session_events(event_type, timestamp);
CREATE INDEX idx_events_processed_at ON session_events(processed_at) WHERE processed_at IS NOT NULL;
CREATE INDEX idx_daily_stats_date_project ON daily_session_stats(date, project);
CREATE INDEX idx_daily_stats_project_date ON daily_session_stats(project, date);
CREATE INDEX idx_productivity_session_type ON productivity_metrics(session_id, metric_type);
CREATE INDEX idx_productivity_type_calculated ON productivity_metrics(metric_type, calculated_at);

-- Create trigger to update daily_session_stats automatically
CREATE TRIGGER update_daily_stats_on_session_update
AFTER UPDATE ON sessions
FOR EACH ROW
WHEN OLD.updated_at != NEW.updated_at
BEGIN
    INSERT OR REPLACE INTO daily_session_stats (
        date, 
        project, 
        worktree, 
        session_count, 
        total_duration_minutes, 
        avg_duration_minutes,
        updated_at
    )
    SELECT 
        DATE(NEW.created_at) as date,
        NEW.project,
        NEW.worktree,
        COUNT(*) as session_count,
        SUM((julianday(updated_at) - julianday(created_at)) * 24 * 60) as total_duration_minutes,
        AVG((julianday(updated_at) - julianday(created_at)) * 24 * 60) as avg_duration_minutes,
        CURRENT_TIMESTAMP
    FROM sessions 
    WHERE DATE(created_at) = DATE(NEW.created_at)
      AND project = NEW.project 
      AND worktree = NEW.worktree;
END;