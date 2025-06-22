package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
)

// QueryBuilder provides pre-built analytics queries for efficient data retrieval
type QueryBuilder struct {
	db *sql.DB
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(db *sql.DB) *QueryBuilder {
	return &QueryBuilder{db: db}
}

// SessionSummaryQuery represents a session summary query result
type SessionSummaryQuery struct {
	Project              string        `json:"project"`
	Worktree             string        `json:"worktree"`
	Branch               string        `json:"branch"`
	SessionCount         int           `json:"session_count"`
	TotalDurationMinutes float64       `json:"total_duration_minutes"`
	AvgDurationMinutes   float64       `json:"avg_duration_minutes"`
	SessionDate          time.Time     `json:"session_date"`
}

// DailyActivityQuery represents a daily activity query result
type DailyActivityQuery struct {
	ActivityDate   time.Time `json:"activity_date"`
	SessionID      string    `json:"session_id"`
	Project        string    `json:"project"`
	Worktree       string    `json:"worktree"`
	BusyCount      int       `json:"busy_count"`
	IdleCount      int       `json:"idle_count"`
	WaitingCount   int       `json:"waiting_count"`
	FirstActivity  time.Time `json:"first_activity"`
	LastActivity   time.Time `json:"last_activity"`
}

// ProjectActivityQuery represents project activity summary
type ProjectActivityQuery struct {
	Project              string    `json:"project"`
	TotalSessions        int       `json:"total_sessions"`
	WorktreeCount        int       `json:"worktree_count"`
	BranchCount          int       `json:"branch_count"`
	FirstSession         time.Time `json:"first_session"`
	LastSession          time.Time `json:"last_session"`
	AvgSessionMinutes    float64   `json:"avg_session_minutes"`
	TotalTimeMinutes     float64   `json:"total_time_minutes"`
}

// GetSessionSummary gets session summary for a date range
func (qb *QueryBuilder) GetSessionSummary(ctx context.Context, start, end time.Time, project, worktree string) ([]SessionSummaryQuery, error) {
	query := `
		SELECT 
			COALESCE(s.project, '') as project,
			COALESCE(s.worktree, '') as worktree,
			COALESCE(s.branch, '') as branch,
			COUNT(*) as session_count,
			SUM((julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60) as total_duration_minutes,
			AVG((julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60) as avg_duration_minutes,
			DATE(s.created_at) as session_date
		FROM sessions s
		WHERE s.created_at >= ? AND s.created_at <= ?
	`

	args := []interface{}{start, end}

	if project != "" {
		query += " AND s.project = ?"
		args = append(args, project)
	}

	if worktree != "" {
		query += " AND s.worktree = ?"
		args = append(args, worktree)
	}

	query += `
		GROUP BY DATE(s.created_at), s.project, s.worktree, s.branch
		ORDER BY session_date DESC, session_count DESC
	`

	rows, err := qb.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute session summary query: %w", err)
	}
	defer rows.Close()

	var results []SessionSummaryQuery
	for rows.Next() {
		var r SessionSummaryQuery
		var sessionDateStr string

		err := rows.Scan(
			&r.Project,
			&r.Worktree,
			&r.Branch,
			&r.SessionCount,
			&r.TotalDurationMinutes,
			&r.AvgDurationMinutes,
			&sessionDateStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session summary row: %w", err)
		}

		// Parse date
		if sessionDate, err := time.Parse("2006-01-02", sessionDateStr); err == nil {
			r.SessionDate = sessionDate
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// GetDailyActivity gets daily activity breakdown
func (qb *QueryBuilder) GetDailyActivity(ctx context.Context, start, end time.Time, project, worktree string) ([]DailyActivityQuery, error) {
	query := `
		SELECT 
			DATE(se.timestamp) as activity_date,
			se.session_id,
			COALESCE(s.project, '') as project,
			COALESCE(s.worktree, '') as worktree,
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
		WHERE se.timestamp >= ? AND se.timestamp <= ?
	`

	args := []interface{}{start, end}

	if project != "" {
		query += " AND s.project = ?"
		args = append(args, project)
	}

	if worktree != "" {
		query += " AND s.worktree = ?"
		args = append(args, worktree)
	}

	query += `
		GROUP BY DATE(se.timestamp), se.session_id, s.project, s.worktree
		ORDER BY activity_date DESC, first_activity ASC
	`

	rows, err := qb.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute daily activity query: %w", err)
	}
	defer rows.Close()

	var results []DailyActivityQuery
	for rows.Next() {
		var r DailyActivityQuery
		var activityDateStr string

		err := rows.Scan(
			&activityDateStr,
			&r.SessionID,
			&r.Project,
			&r.Worktree,
			&r.BusyCount,
			&r.IdleCount,
			&r.WaitingCount,
			&r.FirstActivity,
			&r.LastActivity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily activity row: %w", err)
		}

		// Parse date
		if activityDate, err := time.Parse("2006-01-02", activityDateStr); err == nil {
			r.ActivityDate = activityDate
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// GetProjectActivity gets project activity summary
func (qb *QueryBuilder) GetProjectActivity(ctx context.Context, start, end time.Time) ([]ProjectActivityQuery, error) {
	query := `
		SELECT 
			COALESCE(s.project, '') as project,
			COUNT(DISTINCT s.id) as total_sessions,
			COUNT(DISTINCT s.worktree) as worktree_count,
			COUNT(DISTINCT s.branch) as branch_count,
			MIN(s.created_at) as first_session,
			MAX(s.updated_at) as last_session,
			AVG((julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60) as avg_session_minutes,
			SUM((julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60) as total_time_minutes
		FROM sessions s
		WHERE s.created_at >= ? AND s.created_at <= ?
		  AND s.project IS NOT NULL AND s.project != ''
		GROUP BY s.project
		ORDER BY total_sessions DESC, total_time_minutes DESC
	`

	rows, err := qb.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to execute project activity query: %w", err)
	}
	defer rows.Close()

	var results []ProjectActivityQuery
	for rows.Next() {
		var r ProjectActivityQuery

		err := rows.Scan(
			&r.Project,
			&r.TotalSessions,
			&r.WorktreeCount,
			&r.BranchCount,
			&r.FirstSession,
			&r.LastSession,
			&r.AvgSessionMinutes,
			&r.TotalTimeMinutes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project activity row: %w", err)
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// GetStateTransitions gets state transition analysis
func (qb *QueryBuilder) GetStateTransitions(ctx context.Context, sessionID string, start, end time.Time) ([]StateTransition, error) {
	query := `
		SELECT 
			se.session_id,
			COALESCE(s.project, '') as project,
			COALESCE(s.worktree, '') as worktree,
			JSON_EXTRACT(se.data, '$.old_state') as old_state,
			JSON_EXTRACT(se.data, '$.new_state') as new_state,
			se.timestamp,
			LAG(se.timestamp) OVER (PARTITION BY se.session_id ORDER BY se.timestamp) as prev_timestamp
		FROM session_events se
		JOIN sessions s ON se.session_id = s.id
		WHERE se.event_type = 'claude_state_change'
		  AND se.timestamp >= ? AND se.timestamp <= ?
	`

	args := []interface{}{start, end}

	if sessionID != "" {
		query += " AND se.session_id = ?"
		args = append(args, sessionID)
	}

	query += " ORDER BY se.session_id, se.timestamp"

	rows, err := qb.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute state transitions query: %w", err)
	}
	defer rows.Close()

	var results []StateTransition
	for rows.Next() {
		var r StateTransition
		var prevTimestamp sql.NullTime

		err := rows.Scan(
			&r.SessionID,
			&r.Project,
			&r.Worktree,
			&r.OldState,
			&r.NewState,
			&r.Timestamp,
			&prevTimestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan state transition row: %w", err)
		}

		// Calculate time in previous state
		if prevTimestamp.Valid {
			r.TimeInPrevState = r.Timestamp.Sub(prevTimestamp.Time)
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// StateTransition represents a state transition record
type StateTransition struct {
	SessionID       string        `json:"session_id"`
	Project         string        `json:"project"`
	Worktree        string        `json:"worktree"`
	OldState        string        `json:"old_state"`
	NewState        string        `json:"new_state"`
	Timestamp       time.Time     `json:"timestamp"`
	TimeInPrevState time.Duration `json:"time_in_prev_state"`
}

// GetProductivityStats gets productivity statistics for a time range
func (qb *QueryBuilder) GetProductivityStats(ctx context.Context, start, end time.Time, project string) (*ProductivityStats, error) {
	query := `
		WITH state_durations AS (
			SELECT 
				se.session_id,
				JSON_EXTRACT(se.data, '$.new_state') as state,
				se.timestamp,
				LAG(se.timestamp) OVER (PARTITION BY se.session_id ORDER BY se.timestamp) as prev_timestamp,
				(julianday(se.timestamp) - julianday(LAG(se.timestamp) OVER (PARTITION BY se.session_id ORDER BY se.timestamp))) * 24 * 60 as duration_minutes
			FROM session_events se
			JOIN sessions s ON se.session_id = s.id
			WHERE se.event_type = 'claude_state_change'
			  AND se.timestamp >= ? AND se.timestamp <= ?
	`

	args := []interface{}{start, end}

	if project != "" {
		query += " AND s.project = ?"
		args = append(args, project)
	}

	query += `
		)
		SELECT 
			COUNT(DISTINCT session_id) as total_sessions,
			SUM(CASE WHEN state = 'busy' THEN duration_minutes ELSE 0 END) as active_minutes,
			SUM(CASE WHEN state IN ('idle', 'waiting') THEN duration_minutes ELSE 0 END) as idle_minutes,
			SUM(duration_minutes) as total_minutes,
			AVG(CASE WHEN state = 'busy' THEN duration_minutes END) as avg_focus_duration,
			COUNT(CASE WHEN state = 'busy' THEN 1 END) as focus_sessions
		FROM state_durations
		WHERE prev_timestamp IS NOT NULL
	`

	row := qb.db.QueryRowContext(ctx, query, args...)

	var stats ProductivityStats
	err := row.Scan(
		&stats.TotalSessions,
		&stats.ActiveMinutes,
		&stats.IdleMinutes,
		&stats.TotalMinutes,
		&stats.AvgFocusDuration,
		&stats.FocusSessions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan productivity stats: %w", err)
	}

	// Calculate derived metrics
	if stats.TotalMinutes > 0 {
		stats.ProductivityRatio = stats.ActiveMinutes / stats.TotalMinutes
	}

	return &stats, nil
}

// ProductivityStats represents productivity statistics
type ProductivityStats struct {
	TotalSessions      int     `json:"total_sessions"`
	ActiveMinutes      float64 `json:"active_minutes"`
	IdleMinutes        float64 `json:"idle_minutes"`
	TotalMinutes       float64 `json:"total_minutes"`
	ProductivityRatio  float64 `json:"productivity_ratio"`
	AvgFocusDuration   float64 `json:"avg_focus_duration"`
	FocusSessions      int     `json:"focus_sessions"`
}

// GetRecentSessions gets recent session data
func (qb *QueryBuilder) GetRecentSessions(ctx context.Context, limit int, project, worktree string) ([]RecentSession, error) {
	query := `
		SELECT 
			s.id,
			COALESCE(s.name, '') as name,
			COALESCE(s.project, '') as project,
			COALESCE(s.worktree, '') as worktree,
			COALESCE(s.branch, '') as branch,
			s.created_at,
			s.updated_at,
			s.last_access,
			(julianday(s.updated_at) - julianday(s.created_at)) * 24 * 60 as duration_minutes
		FROM sessions s
		WHERE 1=1
	`

	args := []interface{}{}

	if project != "" {
		query += " AND s.project = ?"
		args = append(args, project)
	}

	if worktree != "" {
		query += " AND s.worktree = ?"
		args = append(args, worktree)
	}

	query += ` ORDER BY s.last_access DESC LIMIT ?`
	args = append(args, limit)

	rows, err := qb.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute recent sessions query: %w", err)
	}
	defer rows.Close()

	var results []RecentSession
	for rows.Next() {
		var r RecentSession

		err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.Project,
			&r.Worktree,
			&r.Branch,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.LastAccess,
			&r.DurationMinutes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recent session row: %w", err)
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// RecentSession represents a recent session record
type RecentSession struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Project         string    `json:"project"`
	Worktree        string    `json:"worktree"`
	Branch          string    `json:"branch"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastAccess      time.Time `json:"last_access"`
	DurationMinutes float64   `json:"duration_minutes"`
}

// CleanupOldData removes old analytics data based on retention policy
func (qb *QueryBuilder) CleanupOldData(ctx context.Context, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Clean up old session events
	result, err := qb.db.ExecContext(ctx, 
		"DELETE FROM session_events WHERE timestamp < ?", 
		cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old session events: %w", err)
	}

	eventsDeleted, _ := result.RowsAffected()

	// Clean up old daily stats
	_, err = qb.db.ExecContext(ctx, 
		"DELETE FROM daily_session_stats WHERE created_at < ?", 
		cutoffDate)
	if err != nil {
		return eventsDeleted, fmt.Errorf("failed to cleanup old daily stats: %w", err)
	}

	// Clean up old productivity metrics
	_, err = qb.db.ExecContext(ctx, 
		"DELETE FROM productivity_metrics WHERE calculated_at < ?", 
		cutoffDate)
	if err != nil {
		return eventsDeleted, fmt.Errorf("failed to cleanup old productivity metrics: %w", err)
	}

	return eventsDeleted, nil
}