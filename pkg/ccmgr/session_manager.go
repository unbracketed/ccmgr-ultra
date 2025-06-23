package ccmgr

import (
	"github.com/bcdekker/ccmgr-ultra/internal/tui"
)

// sessionManager implements the SessionManager interface
type sessionManager struct {
	integration *tui.Integration
}

// List returns all sessions
func (sm *sessionManager) List() ([]SessionInfo, error) {
	internal := sm.integration.GetAllSessions()
	result := make([]SessionInfo, len(internal))
	for i, session := range internal {
		result[i] = convertSessionInfo(session)
	}
	return result, nil
}

// Active returns only active sessions
func (sm *sessionManager) Active() ([]SessionInfo, error) {
	internal := sm.integration.GetActiveSessions()
	result := make([]SessionInfo, len(internal))
	for i, session := range internal {
		result[i] = convertSessionInfo(session)
	}
	return result, nil
}

// Create creates a new session
func (sm *sessionManager) Create(name, directory string) (string, error) {
	// Use the integration layer's CreateSession method
	// Note: This is simplified - in a real implementation, we'd handle the tea.Cmd properly
	_ = sm.integration.CreateSession(name, directory)
	return name, nil
}

// Attach attaches to an existing session
func (sm *sessionManager) Attach(sessionID string) error {
	// Use the integration layer's AttachSession method
	_ = sm.integration.AttachSession(sessionID)
	return nil
}

// Resume resumes a paused session
func (sm *sessionManager) Resume(sessionID string) error {
	// Use the integration layer's ResumeSession method
	_ = sm.integration.ResumeSession(sessionID)
	return nil
}

// FindForWorktree finds sessions for a specific worktree
func (sm *sessionManager) FindForWorktree(worktreePath string) ([]SessionSummary, error) {
	internal, err := sm.integration.FindSessionsForWorktree(worktreePath)
	if err != nil {
		return nil, err
	}
	return convertSessionSummaries(internal), nil
}
