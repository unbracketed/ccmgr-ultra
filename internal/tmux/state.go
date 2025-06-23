package tmux

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SessionState struct {
	FilePath string
	Sessions map[string]*PersistedSession
	mutex    sync.RWMutex
}

type PersistedSession struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Project     string                 `json:"project"`
	Worktree    string                 `json:"worktree"`
	Branch      string                 `json:"branch"`
	Directory   string                 `json:"directory"`
	Created     time.Time              `json:"created"`
	LastAccess  time.Time              `json:"last_access"`
	LastState   ProcessState           `json:"last_state"`
	Environment map[string]string      `json:"environment"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func LoadState(filePath string) (*SessionState, error) {
	state := &SessionState{
		FilePath: filePath,
		Sessions: make(map[string]*PersistedSession),
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := state.SaveState(); err != nil {
			return nil, fmt.Errorf("failed to create initial state file: %w", err)
		}
		return state, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	if len(data) == 0 {
		return state, nil
	}

	var sessions map[string]*PersistedSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		backupPath := filePath + ".backup." + time.Now().Format("20060102-150405")
		if backupErr := os.WriteFile(backupPath, data, 0644); backupErr == nil {
			fmt.Printf("Warning: Corrupted state file backed up to %s\n", backupPath)
		}
		return state, nil
	}

	state.Sessions = sessions
	return state, nil
}

func (ss *SessionState) SaveState() error {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if err := os.MkdirAll(filepath.Dir(ss.FilePath), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(ss.Sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	tempFile := ss.FilePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tempFile, ss.FilePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to atomic write state file: %w", err)
	}

	return nil
}

func (ss *SessionState) AddSession(session *PersistedSession) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if session.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if session.Environment == nil {
		session.Environment = make(map[string]string)
	}
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}

	ss.Sessions[session.ID] = session
	return ss.saveStateUnsafe()
}

func (ss *SessionState) RemoveSession(sessionID string) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if _, exists := ss.Sessions[sessionID]; !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	delete(ss.Sessions, sessionID)
	return ss.saveStateUnsafe()
}

func (ss *SessionState) UpdateSession(sessionID string, updates map[string]interface{}) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	session, exists := ss.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	for key, value := range updates {
		switch key {
		case "last_access":
			if t, ok := value.(time.Time); ok {
				session.LastAccess = t
			}
		case "last_state":
			if state, ok := value.(ProcessState); ok {
				session.LastState = state
			}
		case "directory":
			if dir, ok := value.(string); ok {
				session.Directory = dir
			}
		case "branch":
			if branch, ok := value.(string); ok {
				session.Branch = branch
			}
		default:
			if session.Metadata == nil {
				session.Metadata = make(map[string]interface{})
			}
			session.Metadata[key] = value
		}
	}

	return ss.saveStateUnsafe()
}

func (ss *SessionState) GetSession(sessionID string) (*PersistedSession, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	session, exists := ss.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	sessionCopy := *session

	if sessionCopy.Environment == nil {
		sessionCopy.Environment = make(map[string]string)
	}
	if sessionCopy.Metadata == nil {
		sessionCopy.Metadata = make(map[string]interface{})
	}

	envCopy := make(map[string]string)
	for k, v := range session.Environment {
		envCopy[k] = v
	}
	sessionCopy.Environment = envCopy

	metaCopy := make(map[string]interface{})
	for k, v := range session.Metadata {
		metaCopy[k] = v
	}
	sessionCopy.Metadata = metaCopy

	return &sessionCopy, nil
}

func (ss *SessionState) ListSessions() []*PersistedSession {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	sessions := make([]*PersistedSession, 0, len(ss.Sessions))
	for _, session := range ss.Sessions {
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}

	return sessions
}

func (ss *SessionState) CleanupStaleEntries(maxAge time.Duration) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	staleSessions := make([]string, 0)

	for id, session := range ss.Sessions {
		if session.LastAccess.Before(cutoff) {
			exists, err := checkSessionExists(id)
			if err != nil || !exists {
				staleSessions = append(staleSessions, id)
			}
		}
	}

	for _, id := range staleSessions {
		delete(ss.Sessions, id)
	}

	if len(staleSessions) > 0 {
		return ss.saveStateUnsafe()
	}

	return nil
}

func (ss *SessionState) saveStateUnsafe() error {
	if err := os.MkdirAll(filepath.Dir(ss.FilePath), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(ss.Sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	tempFile := ss.FilePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tempFile, ss.FilePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to atomic write state file: %w", err)
	}

	return nil
}

func checkSessionExists(sessionID string) (bool, error) {
	tmux := NewTmuxCmd()
	return tmux.HasSession(sessionID)
}

func (ss *SessionState) GetSessionCount() int {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	return len(ss.Sessions)
}

func (ss *SessionState) GetSessionsByProject(project string) []*PersistedSession {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	sessions := make([]*PersistedSession, 0)
	for _, session := range ss.Sessions {
		if session.Project == project {
			sessionCopy := *session
			sessions = append(sessions, &sessionCopy)
		}
	}

	return sessions
}

func (ss *SessionState) GetSessionsByWorktree(worktree string) []*PersistedSession {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	sessions := make([]*PersistedSession, 0)
	for _, session := range ss.Sessions {
		if session.Worktree == worktree {
			sessionCopy := *session
			sessions = append(sessions, &sessionCopy)
		}
	}

	return sessions
}
