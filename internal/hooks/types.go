package hooks

import (
	"context"
	"time"
)

// HookType represents the type of hook
type HookType int

const (
	HookTypeStatusIdle HookType = iota
	HookTypeStatusBusy
	HookTypeStatusWaiting
	HookTypeWorktreeCreation
	HookTypeWorktreeActivation
)

// String returns the string representation of the hook type
func (h HookType) String() string {
	switch h {
	case HookTypeStatusIdle:
		return "status_idle"
	case HookTypeStatusBusy:
		return "status_busy"
	case HookTypeStatusWaiting:
		return "status_waiting"
	case HookTypeWorktreeCreation:
		return "worktree_creation"
	case HookTypeWorktreeActivation:
		return "worktree_activation"
	default:
		return "unknown"
	}
}

// HookEvent represents a hook execution event
type HookEvent struct {
	Type      HookType
	Timestamp time.Time
	Context   HookContext
}

// HookContext provides context information for hook execution
type HookContext struct {
	WorktreePath   string
	WorktreeBranch string
	ProjectName    string
	SessionID      string
	SessionType    string // "new", "continue", "resume"
	OldState       string
	NewState       string
	CustomVars     map[string]string
}

// HookExecutor defines the interface for executing hooks
type HookExecutor interface {
	Execute(ctx context.Context, hookType HookType, hookCtx HookContext) error
	ExecuteAsync(hookType HookType, hookCtx HookContext) <-chan error
	ExecuteStatusHook(hookType HookType, hookCtx HookContext) error
	ExecuteWorktreeCreationHook(hookCtx HookContext) error
	ExecuteWorktreeActivationHook(hookCtx HookContext) error
}

// HookResult represents the result of a hook execution
type HookResult struct {
	HookType  HookType
	Success   bool
	Duration  time.Duration
	ExitCode  int
	Output    string
	Error     error
	Timestamp time.Time
}

// Environment represents the environment variables for hook execution
type Environment struct {
	// Common variables
	WorktreePath   string
	WorktreeBranch string
	ProjectName    string
	SessionID      string

	// Hook-specific variables
	Variables map[string]string
}

// ToMap converts the environment to a map of environment variables
func (e *Environment) ToMap() map[string]string {
	env := make(map[string]string)

	// Set standard variables
	if e.WorktreePath != "" {
		env["CCMGR_WORKTREE_PATH"] = e.WorktreePath
	}
	if e.WorktreeBranch != "" {
		env["CCMGR_WORKTREE_BRANCH"] = e.WorktreeBranch
	}
	if e.ProjectName != "" {
		env["CCMGR_PROJECT_NAME"] = e.ProjectName
	}
	if e.SessionID != "" {
		env["CCMGR_SESSION_ID"] = e.SessionID
	}

	// Add timestamp
	env["CCMGR_TIMESTAMP"] = time.Now().Format(time.RFC3339)

	// Add custom variables
	for key, value := range e.Variables {
		env[key] = value
	}

	return env
}

// Hook represents a configured hook
type Hook struct {
	Type        HookType
	Enabled     bool
	Script      string
	Timeout     time.Duration
	Async       bool
	Environment map[string]string
}
