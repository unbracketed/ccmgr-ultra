package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tempDir, err := os.MkdirTemp("", "ccmgr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

func TestDB_SessionCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	session := &storage.Session{
		ID:         uuid.New().String(),
		Name:       "test-session",
		Project:    "test-project",
		Worktree:   "main",
		Branch:     "feature/test",
		Directory:  "/test/dir",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastAccess: now,
		Metadata: map[string]interface{}{
			"test": "value",
		},
	}

	t.Run("Create", func(t *testing.T) {
		err := db.Sessions().Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		retrieved, err := db.Sessions().GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get session by ID: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Session not found")
		}

		if retrieved.Name != session.Name {
			t.Errorf("Expected name %s, got %s", session.Name, retrieved.Name)
		}

		if retrieved.Project != session.Project {
			t.Errorf("Expected project %s, got %s", session.Project, retrieved.Project)
		}

		if len(retrieved.Metadata) == 0 {
			t.Error("Metadata was not preserved")
		}
	})

	t.Run("GetByName", func(t *testing.T) {
		retrieved, err := db.Sessions().GetByName(ctx, session.Name)
		if err != nil {
			t.Fatalf("Failed to get session by name: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Session not found")
		}

		if retrieved.ID != session.ID {
			t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		updates := map[string]interface{}{
			"branch": "updated-branch",
			"metadata": map[string]interface{}{
				"updated": true,
			},
		}

		err := db.Sessions().Update(ctx, session.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		retrieved, err := db.Sessions().GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get updated session: %v", err)
		}

		if retrieved.Branch != "updated-branch" {
			t.Errorf("Expected branch updated-branch, got %s", retrieved.Branch)
		}

		if updated, ok := retrieved.Metadata["updated"].(bool); !ok || !updated {
			t.Error("Metadata was not updated correctly")
		}
	})

	t.Run("List", func(t *testing.T) {
		filter := storage.SessionFilter{
			Project: "test-project",
			Limit:   10,
		}

		sessions, err := db.Sessions().List(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(sessions) != 1 {
			t.Errorf("Expected 1 session, got %d", len(sessions))
		}

		if sessions[0].ID != session.ID {
			t.Errorf("Expected session ID %s, got %s", session.ID, sessions[0].ID)
		}
	})

	t.Run("Search", func(t *testing.T) {
		filter := storage.SessionFilter{
			Limit: 10,
		}

		sessions, err := db.Sessions().Search(ctx, "test", filter)
		if err != nil {
			t.Fatalf("Failed to search sessions: %v", err)
		}

		if len(sessions) != 1 {
			t.Errorf("Expected 1 session, got %d", len(sessions))
		}
	})

	t.Run("Count", func(t *testing.T) {
		filter := storage.SessionFilter{
			Project: "test-project",
		}

		count, err := db.Sessions().Count(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to count sessions: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.Sessions().Delete(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		retrieved, err := db.Sessions().GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to check deleted session: %v", err)
		}

		if retrieved != nil {
			t.Error("Session was not deleted")
		}
	})
}

func TestDB_EventCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	sessionID := uuid.New().String()

	session := &storage.Session{
		ID:         sessionID,
		Name:       "test-session",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		LastAccess: time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	if err := db.Sessions().Create(ctx, session); err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	event := &storage.SessionEvent{
		SessionID: sessionID,
		EventType: "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"test": "data",
		},
	}

	t.Run("Create", func(t *testing.T) {
		err := db.Events().Create(ctx, event)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}

		if event.ID == 0 {
			t.Error("Event ID was not set")
		}
	})

	t.Run("GetBySessionID", func(t *testing.T) {
		events, err := db.Events().GetBySessionID(ctx, sessionID, 10)
		if err != nil {
			t.Fatalf("Failed to get events by session ID: %v", err)
		}

		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}

		if events[0].EventType != "test-event" {
			t.Errorf("Expected event type test-event, got %s", events[0].EventType)
		}
	})

	t.Run("CreateBatch", func(t *testing.T) {
		batchEvents := []*storage.SessionEvent{
			{
				SessionID: sessionID,
				EventType: "batch-1",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"batch": 1},
			},
			{
				SessionID: sessionID,
				EventType: "batch-2",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"batch": 2},
			},
		}

		err := db.Events().CreateBatch(ctx, batchEvents)
		if err != nil {
			t.Fatalf("Failed to create batch events: %v", err)
		}

		events, err := db.Events().GetBySessionID(ctx, sessionID, 10)
		if err != nil {
			t.Fatalf("Failed to get events: %v", err)
		}

		if len(events) != 3 {
			t.Errorf("Expected 3 events, got %d", len(events))
		}
	})

	t.Run("GetByFilter", func(t *testing.T) {
		filter := storage.EventFilter{
			SessionID:  sessionID,
			EventTypes: []string{"batch-1", "batch-2"},
			Limit:      10,
		}

		events, err := db.Events().GetByFilter(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to get events by filter: %v", err)
		}

		if len(events) != 2 {
			t.Errorf("Expected 2 events, got %d", len(events))
		}
	})
}

func TestDB_Transaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("CommitTransaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		session := &storage.Session{
			ID:         uuid.New().String(),
			Name:       "tx-test",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			LastAccess: time.Now(),
			Metadata:   make(map[string]interface{}),
		}

		if err := tx.Sessions().Create(ctx, session); err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create session in transaction: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		retrieved, err := db.Sessions().GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get session after commit: %v", err)
		}

		if retrieved == nil {
			t.Error("Session was not committed")
		}
	})

	t.Run("RollbackTransaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		session := &storage.Session{
			ID:         uuid.New().String(),
			Name:       "rollback-test",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			LastAccess: time.Now(),
			Metadata:   make(map[string]interface{}),
		}

		if err := tx.Sessions().Create(ctx, session); err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create session in transaction: %v", err)
		}

		if err := tx.Rollback(); err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		retrieved, err := db.Sessions().GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to check session after rollback: %v", err)
		}

		if retrieved != nil {
			t.Error("Session was not rolled back")
		}
	})
}