# Phase 5.1 Implementation Plan: Data Management

## Overview
Implement comprehensive data management capabilities for ccmgr-ultra including persistent session storage, configuration export/import, backup/restore, and search functionality.

## Architecture Decision
**Hybrid Storage Model** - Keep YAML for configuration, add SQLite for session history and search
- Maintains backwards compatibility
- Enables powerful querying capabilities
- Gradual migration path from existing JSON state

## Implementation Phases

### Phase 1: Data Persistence Foundation
**Objective**: Create SQLite infrastructure with migration support

1. **Storage Package Structure**
   ```
   internal/storage/
   ├── interfaces.go      # Repository interfaces
   ├── sqlite/
   │   ├── db.go         # Database connection management
   │   ├── migrations.go # Schema migration system
   │   └── session.go    # Session repository implementation
   └── migrations/
       └── 001_initial.sql
   ```

2. **Core Database Schema**
   ```sql
   CREATE TABLE sessions (
       id TEXT PRIMARY KEY,
       name TEXT NOT NULL,
       project TEXT,
       worktree TEXT,
       branch TEXT,
       directory TEXT,
       created_at TIMESTAMP,
       updated_at TIMESTAMP,
       last_access TIMESTAMP,
       metadata JSON
   );
   
   CREATE TABLE session_events (
       id INTEGER PRIMARY KEY,
       session_id TEXT,
       event_type TEXT,
       timestamp TIMESTAMP,
       data JSON,
       FOREIGN KEY (session_id) REFERENCES sessions(id)
   );
   ```

3. **Repository Pattern Implementation**
   - SessionRepository interface with CRUD operations
   - Transaction support for atomic operations
   - Connection pooling for performance
   - Migration system with version tracking

### Phase 2: Session History Management
**Objective**: Track and persist all session lifecycle events

1. **Event Tracking System**
   - Hook into existing tmux operations
   - Capture create, attach, detach, kill events
   - Background worker for async processing
   - Batch inserts for efficiency

2. **Enhanced Session Model**
   - Extend current Session struct with persistence fields
   - Metadata storage for extensibility
   - Activity tracking and statistics

3. **Migration from JSON State**
   - Import tool for existing state.json files
   - Parallel operation mode during transition
   - Feature flag for gradual rollout
   - Performance benchmarking

### Phase 3: Configuration Export/Import Enhancement
**Objective**: Version-aware configuration management with profiles

1. **Versioned Configuration**
   - Add version field to config schema
   - Migration support between versions
   - Backwards compatibility guarantees

2. **Advanced Features**
   - Configuration profiles (global, project, user)
   - Diff algorithm for change detection
   - Merge resolver for conflicts
   - Validation with detailed error messages

3. **Export/Import Commands**
   ```
   ccmgr-ultra config export [--profile <name>] [--format json|yaml]
   ccmgr-ultra config import <file> [--merge|--replace]
   ccmgr-ultra config diff <file1> <file2>
   ```

### Phase 4: Backup & Restore System
**Objective**: Comprehensive data protection with point-in-time recovery

1. **Backup Infrastructure**
   - Manifest-based backup format (tar.gz)
   - Incremental backup support
   - Compression and encryption options
   - Automatic rotation policies

2. **Restore Capabilities**
   - Full and selective restore
   - Validation before restore
   - Rollback on failure
   - Progress tracking

3. **CLI Integration**
   ```
   ccmgr-ultra backup create [--full|--incremental]
   ccmgr-ultra backup list
   ccmgr-ultra backup restore <backup-id> [--dry-run]
   ccmgr-ultra backup delete <backup-id>
   ```

### Phase 5: Search & Filtering Infrastructure
**Objective**: Fast, flexible search across all session data

1. **Full-Text Search**
   - SQLite FTS5 extension integration
   - Index on session name, project, worktree
   - Real-time index updates
   - Relevance scoring

2. **Query Language**
   - Simple keyword search
   - Advanced filters (project:, since:, status:)
   - Boolean operators (AND, OR, NOT)
   - Sort and pagination

3. **Search Commands**
   ```
   ccmgr-ultra session search "keyword"
   ccmgr-ultra session search "project:myapp status:active"
   ccmgr-ultra session history --since "1 week ago" --limit 50
   ```

## Implementation Dependencies

```
Phase 1: Foundation (Prerequisite for all)
    |
    ├── Phase 2: Session History (Blocks Phase 5)
    |       |
    |       └── Phase 5: Search & Filtering
    |
    ├── Phase 3: Config Export/Import (Independent)
    |
    └── Phase 4: Backup & Restore (Depends on 1 & 2)
```

## Key Design Decisions

1. **Database Location**: `~/.config/ccmgr-ultra/data.db` (global)
   - Project-specific data referenced by project ID
   - Enables cross-project session search

2. **Migration Strategy**: Feature flags with parallel operation
   - Run JSON and SQLite side-by-side initially
   - Gradual migration with fallback option
   - Keep JSON backups for 30 days

3. **Performance Targets**:
   - Sub-100ms search response for 10k sessions
   - <5 second backup/restore cycle
   - Minimal overhead on session operations

## Risk Mitigation

1. **Data Loss Prevention**
   - Automatic pre-migration backups
   - Atomic operations with rollback
   - Extensive test coverage

2. **Performance Impact**
   - Benchmarking suite for regression detection
   - Query optimization and indexing
   - Connection pooling and caching

3. **Compatibility**
   - Feature flags for gradual rollout
   - Version detection and auto-migration
   - Clear upgrade documentation

## Success Criteria

- Zero data loss during migration
- All existing functionality preserved
- Search performance meets targets
- 90%+ test coverage on new code
- Smooth upgrade path for users

## First Steps

1. Create internal/storage package structure
2. Implement basic SQLite connection and migration system
3. Define repository interfaces
4. Write initial schema migration
5. Create unit tests for repository pattern

## Implementation Timeline

### Week 1: Foundation & Migration
- **Day 1-2**: Storage Package Setup
  - Create `internal/storage` package structure
  - Implement database connection management with connection pooling
  - Write migration system with up/down support
  - Create initial schema migration (001_initial.sql)

- **Day 3-4**: Repository Implementation
  - Implement SessionRepository with SQLite backend
  - Create Session model with JSON serialization for metadata
  - Add transaction support for atomic operations
  - Write comprehensive unit tests with test database

- **Day 5**: Migration from JSON State
  - Create migration tool to import existing JSON state
  - Implement parallel operation mode (JSON + SQLite)
  - Add feature flag for gradual rollout
  - Performance benchmarks comparing JSON vs SQLite

### Week 2: Core Features
- **Day 1-2**: Session Event System
  - Implement event tracking (create, attach, detach, kill)
  - Add event hooks to existing tmux operations
  - Create event repository with efficient batch inserts
  - Background worker for event processing

- **Day 3-4**: Enhanced Config Management
  - Add version field to config schema
  - Implement config diff algorithm
  - Create merge resolver for conflicts
  - Profile management (global, project, user)

- **Day 5**: Backup Infrastructure
  - Design backup manifest format
  - Implement incremental backup logic
  - Create restore with validation
  - Add CLI commands for backup operations

### Week 3: Advanced Features
- **Day 1-2**: Search Implementation
  - Add SQLite FTS5 extension support
  - Create search index on session data
  - Implement query parser for advanced filters
  - Add relevance scoring algorithm

- **Day 3-4**: CLI Integration
  - Add new commands for session history, search, backup, and config
  - Update existing commands to use new storage
  - Add progress indicators for long operations

- **Day 5**: Testing & Documentation
  - Integration tests for full workflows
  - Performance testing with 10k+ sessions
  - Update documentation with examples
  - Migration guide for existing users

## Code Examples

### Database Connection Management
```go
// internal/storage/sqlite/db.go
type DB struct {
    conn *sql.DB
    cfg  *config.StorageConfig
}

func NewDB(cfg *config.StorageConfig) (*DB, error) {
    dsn := fmt.Sprintf("%s?_journal=WAL&_timeout=5000", cfg.DatabasePath)
    conn, err := sql.Open("sqlite3", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool
    conn.SetMaxOpenConns(25)
    conn.SetMaxIdleConns(5)
    conn.SetConnMaxLifetime(5 * time.Minute)
    
    return &DB{conn: conn, cfg: cfg}, nil
}
```

### Session Repository Pattern
```go
// internal/storage/interfaces.go
type SessionFilter struct {
    Project   string
    Worktree  string
    Since     time.Time
    Limit     int
    Offset    int
}

type SessionRepository interface {
    Create(ctx context.Context, session *Session) error
    Update(ctx context.Context, id string, updates map[string]interface{}) error
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context, filter SessionFilter) ([]*Session, error)
    Search(ctx context.Context, query string, filter SessionFilter) ([]*Session, error)
}
```

### Migration Strategy
```go
// internal/storage/sqlite/migrations.go
type Migration struct {
    Version     int
    Description string
    Up          string
    Down        string
}

var migrations = []Migration{
    {
        Version:     1,
        Description: "Initial schema",
        Up:          sqlInitialSchema,
        Down:        "DROP TABLE IF EXISTS sessions; DROP TABLE IF EXISTS session_events;",
    },
}
```