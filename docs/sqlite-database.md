# SQLite Database Architecture

This document provides comprehensive documentation for the ccmgr-ultra SQLite database implementation, covering architecture, schema design, analytics capabilities, and operational considerations.

## Overview

The ccmgr-ultra project uses SQLite as its primary data store with a sophisticated architecture that includes:

- **Core session and event tracking** for tmux session management
- **Advanced analytics layer** with real-time and aggregated data
- **Production-ready features** including WAL mode, connection pooling, and comprehensive indexing
- **Robust migration system** with embedded SQL files
- **Repository pattern** for clean separation of concerns

## Architecture

### Repository Pattern

The database implementation follows the repository pattern with clean interfaces:

```go
type Storage interface {
    Sessions() SessionRepository
    Events() SessionEventRepository
    Migrate() error
    Close() error
    BeginTx(ctx context.Context) (Transaction, error)
}
```

**Key Components:**
- `internal/storage/interfaces.go` - Core interfaces and data structures
- `internal/storage/sqlite/db.go` - Main database implementation
- `internal/storage/sqlite/session.go` - Session repository implementation  
- `internal/storage/sqlite/event.go` - Event repository implementation

### Database Configuration

The database is configured with production-ready settings:

```go
// Connection string with optimizations
dsn := fmt.Sprintf("%s?_journal=WAL&_timeout=5000&_foreign_keys=true", dbPath)

// Connection pooling
conn.SetMaxOpenConns(25)
conn.SetMaxIdleConns(5) 
conn.SetConnMaxLifetime(5 * time.Minute)
```

**Default Configuration:**
- **Database Path:** `~/.config/ccmgr-ultra/data.db`
- **WAL Mode:** Enabled for better concurrency
- **Foreign Keys:** Enforced for data integrity
- **Connection Pool:** 25 max connections, 5 idle connections
- **Connection Lifetime:** 5 minutes

## Schema Design

### Core Tables

#### sessions
Stores tmux session metadata and tracking information.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | Unique session identifier (UUID) |
| `name` | TEXT NOT NULL | Human-readable session name |
| `project` | TEXT | Associated project name |
| `worktree` | TEXT | Git worktree identifier |
| `branch` | TEXT | Git branch name |
| `directory` | TEXT | Working directory path |
| `created_at` | TIMESTAMP | Session creation time |
| `updated_at` | TIMESTAMP | Last modification time (auto-updated) |
| `last_access` | TIMESTAMP | Last access time |
| `metadata` | TEXT | JSON metadata (default: '{}') |

#### session_events
Tracks events and state changes for sessions.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Auto-incrementing event ID |
| `session_id` | TEXT NOT NULL | Foreign key to sessions.id |
| `event_type` | TEXT NOT NULL | Event type (e.g., 'claude_state_change') |
| `timestamp` | TIMESTAMP | Event timestamp |
| `data` | TEXT | JSON event data (default: '{}') |

**Foreign Key Constraint:**
```sql
FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
```

### Analytics Tables (Migration 002)

#### daily_session_stats
Aggregated daily statistics for efficient reporting.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Auto-incrementing ID |
| `date` | DATE NOT NULL | Aggregation date |
| `project` | TEXT | Project name |
| `worktree` | TEXT | Worktree identifier |
| `session_count` | INTEGER | Number of sessions |
| `total_duration_minutes` | INTEGER | Total session time |
| `avg_duration_minutes` | REAL | Average session duration |
| `active_time_minutes` | INTEGER | Active time (busy state) |
| `idle_time_minutes` | INTEGER | Idle time |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

**Unique Constraint:** `(date, project, worktree)`

#### productivity_metrics
Individual productivity metrics for detailed analysis.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Auto-incrementing ID |
| `session_id` | TEXT NOT NULL | Foreign key to sessions.id |
| `metric_type` | TEXT NOT NULL | Metric type identifier |
| `metric_value` | REAL NOT NULL | Numeric metric value |
| `calculated_at` | TIMESTAMP | Calculation timestamp |

## Analytics Views

The database includes 8 sophisticated analytical views for different reporting needs:

### session_analytics
Project-level session aggregation by date.

```sql
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
```

### daily_activity
Daily activity patterns with state transition counts.

### session_durations
Session duration analysis with categorization (short/medium/long).

### project_activity
High-level project usage summary.

### worktree_usage
Worktree-specific usage patterns and statistics.

### recent_activity
Activity data for the last 30 days.

### event_frequency
Event type frequency analysis for monitoring.

### state_transitions
Claude state change analysis with timing data.

## Performance Optimization

### Indexing Strategy

The database includes 30+ performance-optimized indexes:

#### Core Table Indexes
```sql
-- Session table indexes
CREATE INDEX idx_sessions_name ON sessions(name);
CREATE INDEX idx_sessions_project ON sessions(project);
CREATE INDEX idx_sessions_worktree ON sessions(worktree);
CREATE INDEX idx_sessions_branch ON sessions(branch);
CREATE INDEX idx_sessions_created_at ON sessions(created_at);
CREATE INDEX idx_sessions_updated_at ON sessions(updated_at);
CREATE INDEX idx_sessions_last_access ON sessions(last_access);

-- Event table indexes  
CREATE INDEX idx_session_events_session_id ON session_events(session_id);
CREATE INDEX idx_session_events_type ON session_events(event_type);
CREATE INDEX idx_session_events_timestamp ON session_events(timestamp);
```

#### Analytics Indexes
```sql
-- Composite indexes for analytics queries
CREATE INDEX idx_sessions_project_worktree_date 
ON sessions(project, worktree, DATE(created_at));

-- JSON data extraction indexes
CREATE INDEX idx_events_data_state_change 
ON session_events(JSON_EXTRACT(data, '$.new_state')) 
WHERE event_type = 'claude_state_change';

-- Time-based indexes for efficient date ranges
CREATE INDEX idx_sessions_recent_week 
ON sessions(created_at, project, worktree) 
WHERE created_at >= DATE('now', '-7 days');
```

#### Partial Indexes
Space-efficient indexes for non-null values:

```sql
CREATE INDEX idx_sessions_project_not_null 
ON sessions(project, created_at) WHERE project IS NOT NULL;
```

### Query Optimization Features

- **Covering Indexes:** Include all columns needed for common queries
- **Functional Indexes:** Index computed values like JSON extractions
- **Partial Indexes:** Only index relevant rows to save space
- **Time-based Indexes:** Optimize recent data queries

## Migration System

### Architecture

The migration system uses Go's `embed.FS` to embed SQL files directly in the binary:

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS
```

### Migration Files

Migrations follow the naming convention: `XXX_description.sql`

**Available Migrations:**
- `001_initial.sql` - Core schema with sessions and events tables
- `002_analytics_views.sql` - Analytics layer with views and aggregation tables

### Migration Format

Each migration file must include UP and DOWN sections:

```sql
-- UP
CREATE TABLE example (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- DOWN  
DROP TABLE IF EXISTS example;
```

### Migration Execution

Migrations are tracked in the `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMP NOT NULL
);
```

**Process:**
1. Check current schema version
2. Apply migrations in order
3. Record successful migrations
4. Use transactions for atomicity

## Data Integrity Features

### Triggers

#### Auto-update Timestamp
```sql
CREATE TRIGGER update_sessions_updated_at 
    AFTER UPDATE ON sessions
    FOR EACH ROW
    BEGIN
        UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;
```

#### Daily Stats Maintenance
Automatically updates `daily_session_stats` when sessions are modified.

### Constraints

- **Foreign Key Constraints:** Enforce referential integrity
- **Unique Constraints:** Prevent duplicate daily stats
- **Check Constraints:** Validate data ranges (via application logic)

## Transaction Management

### Repository Pattern Support

Each repository can work with either a direct database connection or a transaction:

```go
type sessionRepository struct {
    db *DB
    tx *sql.Tx
}

func (r *sessionRepository) exec() interface{...} {
    if r.tx != nil {
        return r.tx
    }
    return r.db.conn
}
```

### Transaction Examples

```go
// Begin transaction
tx, err := storage.BeginTx(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

// Use transaction repositories
err = tx.Sessions().Create(ctx, session)
if err != nil {
    return err
}

err = tx.Events().Create(ctx, event)
if err != nil {
    return err
}

// Commit transaction
return tx.Commit()
```

## Analytics Capabilities

### Real-time Event Tracking

Events are captured in real-time as they occur:

```go
event := &storage.SessionEvent{
    SessionID: sessionID,
    EventType: "claude_state_change", 
    Timestamp: time.Now(),
    Data: map[string]interface{}{
        "old_state": "idle",
        "new_state": "busy",
        "duration_ms": 1500,
    },
}
```

### Aggregated Analytics

Daily stats are maintained automatically via triggers and can be queried efficiently:

```sql
-- Project productivity over time
SELECT 
    date,
    project,
    session_count,
    total_duration_minutes,
    avg_duration_minutes
FROM daily_session_stats 
WHERE date >= DATE('now', '-30 days')
ORDER BY date DESC;
```

### State Transition Analysis

Track Claude Code state changes over time:

```sql
-- Average time in each state
SELECT 
    JSON_EXTRACT(se.data, '$.old_state') as state,
    AVG(time_in_prev_state_minutes) as avg_minutes
FROM state_transitions st
WHERE st.time_in_prev_state_minutes IS NOT NULL
GROUP BY JSON_EXTRACT(se.data, '$.old_state');
```

## Testing Infrastructure

### Test Database Setup

The test infrastructure provides isolated test databases:

```go
func setupTestDB(t *testing.T) (*DB, func()) {
    tempDir, err := os.MkdirTemp("", "ccmgr-test-*")
    // ... setup test database
    
    cleanup := func() {
        db.Close()
        os.RemoveAll(tempDir)
    }
    
    return db, cleanup
}
```

### Test Coverage

Tests cover:
- **CRUD Operations:** Create, read, update, delete for all entities
- **Transaction Handling:** Rollback and commit scenarios  
- **Migration Execution:** Up and down migrations
- **Filtering and Search:** Complex query scenarios
- **Concurrent Access:** Multi-connection scenarios

## Operational Considerations

### Database Location

**Default Path:** `~/.config/ccmgr-ultra/data.db`

**Configuration Override:**
```yaml
database_path: "/custom/path/to/database.db"
enable_wal_mode: true
max_connections: 25
```

### Backup Strategy

*Note: Backup functionality is configured but not yet implemented.*

**Planned Features:**
- Automatic daily backups
- Configurable retention policies
- Backup verification
- Point-in-time recovery

### Maintenance Requirements

#### Data Retention

*Note: Retention cleanup is configured but not yet implemented.*

**Planned Cleanup:**
- Remove events older than 90 days (configurable)
- Archive old session data
- Maintain aggregated statistics longer than raw events

#### Database Maintenance

**Recommended Periodic Tasks:**
```sql
-- Analyze query planner statistics
ANALYZE;

-- Reclaim space from deleted records
VACUUM;

-- Check database integrity
PRAGMA integrity_check;
```

### Performance Monitoring

#### Key Metrics to Monitor

- **Query Performance:** Slow query identification
- **Index Usage:** `EXPLAIN QUERY PLAN` analysis
- **Database Size:** Monitor growth patterns
- **Connection Pool:** Monitor connection utilization

#### Analytics Queries

Monitor the most expensive operations:
- Daily aggregation queries
- State transition analysis  
- Large date range queries
- JSON data extraction operations

## Security Considerations

### SQL Injection Prevention

All queries use parameterized statements:

```go
query := `SELECT id, name FROM sessions WHERE project = ?`
rows, err := db.QueryContext(ctx, query, projectName)
```

### Data Sensitivity

- **No Sensitive Data:** Database contains only metadata and operational data
- **Local Storage:** Database is stored locally, not transmitted
- **Access Control:** File system permissions control access

### Foreign Key Enforcement

```sql
PRAGMA foreign_keys = ON;
```

Ensures referential integrity and prevents orphaned records.

## Future Enhancements

### Planned Improvements

1. **Backup Implementation**
   - Automated backup scheduling
   - Backup verification and restore testing
   - Cloud backup integration options

2. **Data Retention Automation**
   - Configurable retention policies
   - Automated cleanup jobs
   - Archive functionality for historical data

3. **Performance Enhancements**
   - Query optimization monitoring
   - Adaptive indexing based on usage patterns
   - Connection pool tuning

4. **Analytics Expansion**
   - Machine learning insights
   - Predictive analytics for session patterns
   - Advanced visualization data

### Migration Path Considerations

The repository pattern enables future database migrations:

- **PostgreSQL Migration:** For larger deployments
- **Distributed Databases:** For multi-machine scenarios
- **Cloud Databases:** For cloud-native deployments

The interface abstractions ensure application code remains unchanged during migrations.

## Troubleshooting

### Common Issues

#### Database Locked Errors
```
database is locked
```
**Solutions:**
- Check for long-running transactions
- Verify WAL mode is enabled
- Monitor connection pool usage

#### Migration Failures
```
migration X failed to apply
```
**Solutions:**
- Check database permissions
- Verify disk space availability
- Review migration SQL syntax
- Check for schema conflicts

#### Performance Issues
**Symptoms:** Slow queries, high CPU usage
**Solutions:**
- Run `ANALYZE` to update query planner statistics
- Review query execution plans
- Check index usage with `EXPLAIN QUERY PLAN`
- Consider adding specialized indexes

### Diagnostic Queries

#### Check Database Status
```sql
PRAGMA database_list;
PRAGMA journal_mode;
PRAGMA foreign_keys;
```

#### Monitor Table Sizes
```sql
SELECT 
    name,
    COUNT(*) as row_count
FROM sqlite_master 
WHERE type = 'table'
GROUP BY name;
```

#### Index Usage Analysis
```sql
SELECT 
    name,
    tbl_name,
    sql
FROM sqlite_master 
WHERE type = 'index'
ORDER BY tbl_name, name;
```

---

## References

- **Source Code:** `internal/storage/` directory
- **Migration Files:** `internal/storage/sqlite/migrations/`
- **Test Files:** `internal/storage/sqlite/*_test.go`
- **Configuration:** `internal/storage/config.go`

This documentation reflects the current implementation as of the latest codebase analysis. For the most up-to-date information, consult the source code and test files.