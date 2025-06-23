package storage_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
	"github.com/bcdekker/ccmgr-ultra/internal/storage/sqlite"
	"github.com/google/uuid"
)

func setupTestStorage(t *testing.T) (storage.Storage, func()) {
	tempDir, err := os.MkdirTemp("", "ccmgr-migrate-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sqlite.NewDB(dbPath)
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

func TestMigrator_ValidateJSONFile(t *testing.T) {
	storageInstance, cleanup := setupTestStorage(t)
	defer cleanup()

	migrator := storage.NewMigrator(storageInstance)

	t.Run("ValidJSON", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "valid-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		validState := storage.JSONState{
			Sessions: []storage.JSONSession{
				{
					ID:        uuid.New().String(),
					Name:      "session1",
					Project:   "project1",
					CreatedAt: time.Now(),
				},
				{
					ID:        uuid.New().String(),
					Name:      "session2",
					Project:   "project2",
					CreatedAt: time.Now(),
				},
			},
		}

		data, err := json.Marshal(validState)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if _, err := tempFile.Write(data); err != nil {
			t.Fatalf("Failed to write JSON file: %v", err)
		}
		tempFile.Close()

		if err := migrator.ValidateJSONFile(tempFile.Name()); err != nil {
			t.Errorf("Valid JSON file failed validation: %v", err)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "invalid-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		if _, err := tempFile.WriteString("invalid json"); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}
		tempFile.Close()

		if err := migrator.ValidateJSONFile(tempFile.Name()); err == nil {
			t.Error("Invalid JSON file passed validation")
		}
	})

	t.Run("EmptySession", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "empty-session-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		invalidState := storage.JSONState{
			Sessions: []storage.JSONSession{
				{
					ID:   uuid.New().String(),
					Name: "",
				},
			},
		}

		data, err := json.Marshal(invalidState)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if _, err := tempFile.Write(data); err != nil {
			t.Fatalf("Failed to write JSON file: %v", err)
		}
		tempFile.Close()

		if err := migrator.ValidateJSONFile(tempFile.Name()); err == nil {
			t.Error("JSON file with empty session name passed validation")
		}
	})

	t.Run("DuplicateSessionNames", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "duplicate-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		duplicateState := storage.JSONState{
			Sessions: []storage.JSONSession{
				{
					ID:   uuid.New().String(),
					Name: "duplicate",
				},
				{
					ID:   uuid.New().String(),
					Name: "duplicate",
				},
			},
		}

		data, err := json.Marshal(duplicateState)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if _, err := tempFile.Write(data); err != nil {
			t.Fatalf("Failed to write JSON file: %v", err)
		}
		tempFile.Close()

		if err := migrator.ValidateJSONFile(tempFile.Name()); err == nil {
			t.Error("JSON file with duplicate session names passed validation")
		}
	})
}

func TestMigrator_MigrateFromJSONFile(t *testing.T) {
	storageInstance, cleanup := setupTestStorage(t)
	defer cleanup()

	migrator := storage.NewMigrator(storageInstance)
	ctx := context.Background()

	t.Run("SuccessfulMigration", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "migrate-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() {
			// Clean up both original and backup files
			os.Remove(tempFile.Name())
			os.Remove(tempFile.Name() + ".migrated.*")
		}()

		now := time.Now()
		testState := storage.JSONState{
			Sessions: []storage.JSONSession{
				{
					ID:        uuid.New().String(),
					Name:      "migrate-session-1",
					Project:   "migrate-project",
					Worktree:  "main",
					Branch:    "feature/test",
					Directory: "/test/dir",
					CreatedAt: now,
				},
				{
					Name:      "migrate-session-2",
					Project:   "migrate-project-2",
					CreatedAt: now,
				},
			},
		}

		data, err := json.Marshal(testState)
		if err != nil {
			t.Fatalf("Failed to marshal test state: %v", err)
		}

		if _, err := tempFile.Write(data); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		tempFile.Close()

		if err := migrator.MigrateFromJSONFile(ctx, tempFile.Name()); err != nil {
			t.Fatalf("Migration failed: %v", err)
		}

		sessions, err := storageInstance.Sessions().List(ctx, storage.SessionFilter{Limit: 10})
		if err != nil {
			t.Fatalf("Failed to list sessions after migration: %v", err)
		}

		if len(sessions) != 2 {
			t.Errorf("Expected 2 migrated sessions, got %d", len(sessions))
		}

		for _, session := range sessions {
			if session.Name != "migrate-session-1" && session.Name != "migrate-session-2" {
				t.Errorf("Unexpected session name: %s", session.Name)
			}

			if session.ID == "" {
				t.Error("Session ID was not set")
			}

			if session.CreatedAt.IsZero() {
				t.Error("Session CreatedAt was not set")
			}
		}

		events, err := storageInstance.Events().GetByFilter(ctx, storage.EventFilter{
			EventTypes: []string{"migrated"},
			Limit:      10,
		})
		if err != nil {
			t.Fatalf("Failed to get migration events: %v", err)
		}

		if len(events) != 2 {
			t.Errorf("Expected 2 migration events, got %d", len(events))
		}

		if _, err := os.Stat(tempFile.Name()); !os.IsNotExist(err) {
			t.Error("Original JSON file was not moved to backup")
		}
	})

	t.Run("NonexistentFile", func(t *testing.T) {
		if err := migrator.MigrateFromJSONFile(ctx, "/nonexistent/file.json"); err == nil {
			t.Error("Migration of nonexistent file should fail")
		}
	})
}

func TestMigrator_FindJSONStateFiles(t *testing.T) {
	storageInstance, cleanup := setupTestStorage(t)
	defer cleanup()

	migrator := storage.NewMigrator(storageInstance)

	tempDir, err := os.MkdirTemp("", "find-json-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "state.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	files, err := migrator.FindJSONStateFiles()
	if err != nil {
		t.Fatalf("FindJSONStateFiles failed: %v", err)
	}

	found := false
	for _, file := range files {
		if filepath.Base(file) == "state.json" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find state.json in current directory")
	}
}
