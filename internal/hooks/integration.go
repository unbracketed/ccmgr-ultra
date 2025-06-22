package hooks

import (
	"context"
	"log"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// GlobalHookManager is a global instance for easy access across the application
var GlobalHookManager *Manager

// InitializeHooks initializes the global hook manager with the given configuration
func InitializeHooks(cfg *config.Config) *Manager {
	manager := NewManager(cfg)
	GlobalHookManager = manager
	return manager
}

// GetGlobalHookManager returns the global hook manager instance
func GetGlobalHookManager() *Manager {
	return GlobalHookManager
}

// Integration helpers for other components

// NotifyClaudeStateChange is a convenience function for notifying about Claude state changes
func NotifyClaudeStateChange(oldState, newState string, workingDir, branch, sessionID string) {
	if GlobalHookManager == nil {
		return
	}

	context := HookContext{
		WorktreePath:   workingDir,
		WorktreeBranch: branch,
		SessionID:      sessionID,
		OldState:       oldState,
		NewState:       newState,
	}

	GlobalHookManager.OnClaudeStateChange(oldState, newState, context)
}

// NotifyWorktreeCreated is a convenience function for notifying about worktree creation
func NotifyWorktreeCreated(worktreePath, branch, parentPath, projectName string) error {
	if GlobalHookManager == nil {
		return nil
	}

	return GlobalHookManager.OnWorktreeCreated(worktreePath, branch, parentPath, projectName)
}

// NotifySessionCreated is a convenience function for notifying about session creation
func NotifySessionCreated(workingDir, branch, sessionID, projectName string) error {
	if GlobalHookManager == nil {
		return nil
	}

	sessionInfo := SessionInfo{
		SessionID:   sessionID,
		WorkingDir:  workingDir,
		Branch:      branch,
		ProjectName: projectName,
	}

	return GlobalHookManager.OnSessionCreated(sessionInfo)
}

// NotifySessionContinued is a convenience function for notifying about session continuation
func NotifySessionContinued(workingDir, branch, sessionID, projectName string) error {
	if GlobalHookManager == nil {
		return nil
	}

	sessionInfo := SessionInfo{
		SessionID:   sessionID,
		WorkingDir:  workingDir,
		Branch:      branch,
		ProjectName: projectName,
	}

	return GlobalHookManager.OnSessionContinued(sessionInfo)
}

// NotifySessionResumed is a convenience function for notifying about session resumption
func NotifySessionResumed(workingDir, branch, sessionID, projectName, previousState string) error {
	if GlobalHookManager == nil {
		return nil
	}

	sessionInfo := SessionInfo{
		SessionID:   sessionID,
		WorkingDir:  workingDir,
		Branch:      branch,
		ProjectName: projectName,
	}

	return GlobalHookManager.OnSessionResumed(sessionInfo, previousState)
}

// ProcessStateChangeEvent represents a process state change event
type ProcessStateChangeEvent struct {
	ProcessID    string
	OldState     string
	NewState     string
	WorkingDir   string
	Branch       string
	SessionID    string
	ProjectName  string
	ProcessInfo  map[string]interface{}
}

// HandleProcessStateChange handles a process state change event
func HandleProcessStateChange(event ProcessStateChangeEvent) {
	if GlobalHookManager == nil {
		return
	}

	context := HookContext{
		WorktreePath:   event.WorkingDir,
		WorktreeBranch: event.Branch,
		ProjectName:    event.ProjectName,
		SessionID:      event.SessionID,
		OldState:       event.OldState,
		NewState:       event.NewState,
		CustomVars: map[string]string{
			"CCMGR_PROCESS_ID": event.ProcessID,
		},
	}

	// Add process info as custom variables
	for key, value := range event.ProcessInfo {
		if strValue, ok := value.(string); ok {
			context.CustomVars["CCMGR_PROCESS_"+key] = strValue
		}
	}

	GlobalHookManager.OnClaudeStateChange(event.OldState, event.NewState, context)
}

// WorktreeCreationEvent represents a worktree creation event
type WorktreeCreationEvent struct {
	WorktreePath string
	Branch       string
	ParentPath   string
	ProjectName  string
	Template     string
	Language     string
	Environment  map[string]string
}

// HandleWorktreeCreation handles a worktree creation event
func HandleWorktreeCreation(event WorktreeCreationEvent) error {
	if GlobalHookManager == nil {
		return nil
	}

	// Use the worktree integrator for more advanced bootstrap options
	integrator := GlobalHookManager.GetWorktreeIntegrator()
	
	if integrator != nil && integrator.IsEnabled() {
		ctx := context.Background()
		options := BootstrapOptions{
			ProjectName: event.ProjectName,
			ParentPath:  event.ParentPath,
			Template:    event.Template,
			Language:    event.Language,
			Environment: event.Environment,
		}

		return integrator.HandleWorktreeBootstrap(ctx, event.WorktreePath, event.Branch, options)
	}

	// Fallback to basic worktree creation notification
	return GlobalHookManager.OnWorktreeCreated(event.WorktreePath, event.Branch, event.ParentPath, event.ProjectName)
}

// SessionLifecycleEvent represents a session lifecycle event
type SessionLifecycleEvent struct {
	Type        string // "create", "continue", "resume", "pause", "stop"
	SessionID   string
	WorkingDir  string
	Branch      string
	ProjectName string
	State       string
	Metadata    map[string]string
}

// HandleSessionLifecycle handles a session lifecycle event
func HandleSessionLifecycle(event SessionLifecycleEvent) error {
	if GlobalHookManager == nil {
		return nil
	}

	sessionInfo := SessionInfo{
		SessionID:   event.SessionID,
		WorkingDir:  event.WorkingDir,
		Branch:      event.Branch,
		ProjectName: event.ProjectName,
	}

	switch event.Type {
	case "create":
		return GlobalHookManager.OnSessionCreated(sessionInfo)
	case "continue":
		return GlobalHookManager.OnSessionContinued(sessionInfo)
	case "resume":
		previousState := event.State
		if prev, exists := event.Metadata["previous_state"]; exists {
			previousState = prev
		}
		return GlobalHookManager.OnSessionResumed(sessionInfo, previousState)
	default:
		log.Printf("Unknown session lifecycle event type: %s", event.Type)
		return nil
	}
}

// EnableHooks enables hook execution globally
func EnableHooks() {
	if GlobalHookManager != nil {
		GlobalHookManager.Enable()
	}
}

// DisableHooks disables hook execution globally
func DisableHooks() {
	if GlobalHookManager != nil {
		GlobalHookManager.Disable()
	}
}

// UpdateHookConfig updates the hook manager configuration
func UpdateHookConfig(cfg *config.Config) {
	if GlobalHookManager != nil {
		GlobalHookManager.UpdateConfig(cfg)
	}
}

// StartHookServices starts background hook services
func StartHookServices(ctx context.Context) {
	if GlobalHookManager != nil {
		GlobalHookManager.Start(ctx)
	}
}

// IsHooksEnabled returns whether hooks are enabled globally
func IsHooksEnabled() bool {
	if GlobalHookManager == nil {
		return false
	}
	return GlobalHookManager.IsEnabled()
}