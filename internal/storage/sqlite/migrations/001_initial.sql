-- UP
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    project TEXT,
    worktree TEXT,
    branch TEXT,
    directory TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_access TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT DEFAULT '{}'
);

CREATE TABLE session_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data TEXT DEFAULT '{}',
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_name ON sessions(name);
CREATE INDEX idx_sessions_project ON sessions(project);
CREATE INDEX idx_sessions_worktree ON sessions(worktree);
CREATE INDEX idx_sessions_branch ON sessions(branch);
CREATE INDEX idx_sessions_created_at ON sessions(created_at);
CREATE INDEX idx_sessions_updated_at ON sessions(updated_at);
CREATE INDEX idx_sessions_last_access ON sessions(last_access);

CREATE INDEX idx_session_events_session_id ON session_events(session_id);
CREATE INDEX idx_session_events_type ON session_events(event_type);
CREATE INDEX idx_session_events_timestamp ON session_events(timestamp);

CREATE TRIGGER update_sessions_updated_at 
    AFTER UPDATE ON sessions
    FOR EACH ROW
    BEGIN
        UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

-- DOWN
DROP TRIGGER IF EXISTS update_sessions_updated_at;
DROP INDEX IF EXISTS idx_session_events_timestamp;
DROP INDEX IF EXISTS idx_session_events_type;
DROP INDEX IF EXISTS idx_session_events_session_id;
DROP INDEX IF EXISTS idx_sessions_last_access;
DROP INDEX IF EXISTS idx_sessions_updated_at;
DROP INDEX IF EXISTS idx_sessions_created_at;
DROP INDEX IF EXISTS idx_sessions_branch;
DROP INDEX IF EXISTS idx_sessions_worktree;
DROP INDEX IF EXISTS idx_sessions_project;
DROP INDEX IF EXISTS idx_sessions_name;
DROP TABLE IF EXISTS session_events;
DROP TABLE IF EXISTS sessions;