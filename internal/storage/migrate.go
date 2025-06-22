package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type JSONSession struct {
	ID        string    `json:"id,omitempty"`
	Name      string    `json:"name"`
	Project   string    `json:"project,omitempty"`
	Worktree  string    `json:"worktree,omitempty"`
	Branch    string    `json:"branch,omitempty"`
	Directory string    `json:"directory,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type JSONState struct {
	Sessions []JSONSession `json:"sessions"`
}

type Migrator struct {
	storage Storage
}

func NewMigrator(storage Storage) *Migrator {
	return &Migrator{
		storage: storage,
	}
}

func (m *Migrator) MigrateFromJSONFile(ctx context.Context, jsonPath string) error {
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return fmt.Errorf("JSON state file does not exist: %s", jsonPath)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var state JSONState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse JSON state: %w", err)
	}

	tx, err := m.storage.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, jsonSession := range state.Sessions {
		session := &Session{
			ID:         jsonSession.ID,
			Name:       jsonSession.Name,
			Project:    jsonSession.Project,
			Worktree:   jsonSession.Worktree,
			Branch:     jsonSession.Branch,
			Directory:  jsonSession.Directory,
			CreatedAt:  jsonSession.CreatedAt,
			UpdatedAt:  time.Now(),
			LastAccess: time.Now(),
			Metadata:   make(map[string]interface{}),
		}

		if session.ID == "" {
			session.ID = uuid.New().String()
		}

		if session.CreatedAt.IsZero() {
			session.CreatedAt = time.Now()
		}

		if err := tx.Sessions().Create(ctx, session); err != nil {
			return fmt.Errorf("failed to create session %s: %w", session.Name, err)
		}

		event := &SessionEvent{
			SessionID: session.ID,
			EventType: "migrated",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"source": "json",
				"file":   jsonPath,
			},
		}

		if err := tx.Events().Create(ctx, event); err != nil {
			return fmt.Errorf("failed to create migration event for session %s: %w", session.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	backupPath := jsonPath + ".migrated." + time.Now().Format("20060102-150405")
	if err := os.Rename(jsonPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup original JSON file: %w", err)
	}

	return nil
}

func (m *Migrator) FindJSONStateFiles() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	var files []string
	searchPaths := []string{
		filepath.Join(homeDir, ".config", "ccmgr-ultra", "state.json"),
		filepath.Join(homeDir, ".ccmgr-ultra", "state.json"),
		"state.json",
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}

	return files, nil
}

func (m *Migrator) ValidateJSONFile(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var state JSONState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	if len(state.Sessions) == 0 {
		return fmt.Errorf("no sessions found in JSON file")
	}

	sessionNames := make(map[string]bool)
	for i, session := range state.Sessions {
		if session.Name == "" {
			return fmt.Errorf("session %d has empty name", i)
		}

		if sessionNames[session.Name] {
			return fmt.Errorf("duplicate session name: %s", session.Name)
		}
		sessionNames[session.Name] = true
	}

	return nil
}