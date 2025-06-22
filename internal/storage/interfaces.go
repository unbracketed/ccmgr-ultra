package storage

import (
	"context"
	"time"
)

type Session struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Project    string                 `json:"project,omitempty"`
	Worktree   string                 `json:"worktree,omitempty"`
	Branch     string                 `json:"branch,omitempty"`
	Directory  string                 `json:"directory,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	LastAccess time.Time              `json:"last_access"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type SessionEvent struct {
	ID        int64                  `json:"id"`
	SessionID string                 `json:"session_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type SessionFilter struct {
	Project   string
	Worktree  string
	Branch    string
	Since     time.Time
	Until     time.Time
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Session, error)
	GetByName(ctx context.Context, name string) (*Session, error)
	List(ctx context.Context, filter SessionFilter) ([]*Session, error)
	Search(ctx context.Context, query string, filter SessionFilter) ([]*Session, error)
	Count(ctx context.Context, filter SessionFilter) (int64, error)
}

type SessionEventRepository interface {
	Create(ctx context.Context, event *SessionEvent) error
	CreateBatch(ctx context.Context, events []*SessionEvent) error
	GetBySessionID(ctx context.Context, sessionID string, limit int) ([]*SessionEvent, error)
	GetByFilter(ctx context.Context, filter EventFilter) ([]*SessionEvent, error)
}

type EventFilter struct {
	SessionID  string
	EventTypes []string
	Since      time.Time
	Until      time.Time
	Limit      int
	Offset     int
}

type Storage interface {
	Sessions() SessionRepository
	Events() SessionEventRepository
	Migrate() error
	Close() error
	BeginTx(ctx context.Context) (Transaction, error)
}

type Transaction interface {
	Sessions() SessionRepository
	Events() SessionEventRepository
	Commit() error
	Rollback() error
}