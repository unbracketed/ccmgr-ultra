package hooks

import (
	"context"
	"path/filepath"
	"time"
)

// WorktreeHookManager manages worktree lifecycle hook execution
type WorktreeHookManager struct {
	executor HookExecutor
	enabled  bool
}

// NewWorktreeHookManager creates a new worktree hook manager
func NewWorktreeHookManager(executor HookExecutor) *WorktreeHookManager {
	return &WorktreeHookManager{
		executor: executor,
		enabled:  true,
	}
}

// SetEnabled enables or disables worktree hook execution
func (whm *WorktreeHookManager) SetEnabled(enabled bool) {
	whm.enabled = enabled
}

// IsEnabled returns whether worktree hooks are enabled
func (whm *WorktreeHookManager) IsEnabled() bool {
	return whm.enabled
}

// OnWorktreeCreated triggers the worktree creation hook
func (whm *WorktreeHookManager) OnWorktreeCreated(worktreePath, branch, parentPath, projectName string) error {
	if !whm.IsEnabled() {
		return nil
	}

	context := HookContext{
		WorktreePath:   worktreePath,
		WorktreeBranch: branch,
		ProjectName:    projectName,
		SessionType:    "new",
		CustomVars: map[string]string{
			"CCMGR_PARENT_PATH":   parentPath,
			"CCMGR_WORKTREE_TYPE": "new",
		},
	}

	return whm.executor.ExecuteWorktreeCreationHook(context)
}

// OnWorktreeActivated triggers the worktree activation hook
func (whm *WorktreeHookManager) OnWorktreeActivated(worktreePath, branch, sessionID, sessionType, projectName string) error {
	if !whm.IsEnabled() {
		return nil
	}

	context := HookContext{
		WorktreePath:   worktreePath,
		WorktreeBranch: branch,
		ProjectName:    projectName,
		SessionID:      sessionID,
		SessionType:    sessionType,
		CustomVars:     make(map[string]string),
	}

	return whm.executor.ExecuteWorktreeActivationHook(context)
}

// OnSessionCreated handles session creation (triggers activation hook)
func (whm *WorktreeHookManager) OnSessionCreated(sessionInfo SessionInfo) error {
	return whm.OnWorktreeActivated(
		sessionInfo.WorkingDir,
		sessionInfo.Branch,
		sessionInfo.SessionID,
		"new",
		sessionInfo.ProjectName,
	)
}

// OnSessionContinued handles session continuation (triggers activation hook)
func (whm *WorktreeHookManager) OnSessionContinued(sessionInfo SessionInfo) error {
	return whm.OnWorktreeActivated(
		sessionInfo.WorkingDir,
		sessionInfo.Branch,
		sessionInfo.SessionID,
		"continue",
		sessionInfo.ProjectName,
	)
}

// OnSessionResumed handles session resumption (triggers activation hook)
func (whm *WorktreeHookManager) OnSessionResumed(sessionInfo SessionInfo, previousState string) error {
	context := HookContext{
		WorktreePath:   sessionInfo.WorkingDir,
		WorktreeBranch: sessionInfo.Branch,
		ProjectName:    sessionInfo.ProjectName,
		SessionID:      sessionInfo.SessionID,
		SessionType:    "resume",
		CustomVars: map[string]string{
			"CCMGR_PREVIOUS_STATE": previousState,
		},
	}

	return whm.executor.ExecuteWorktreeActivationHook(context)
}

// SessionInfo contains information about a session
type SessionInfo struct {
	SessionID   string
	WorkingDir  string
	Branch      string
	ProjectName string
	CreatedAt   time.Time
	LastActive  time.Time
}

// WorktreeHookIntegrator provides integration points for the worktree hook system
type WorktreeHookIntegrator struct {
	hookManager *WorktreeHookManager
	enabled     bool
}

// NewWorktreeHookIntegrator creates a new worktree hook integrator
func NewWorktreeHookIntegrator(executor HookExecutor) *WorktreeHookIntegrator {
	return &WorktreeHookIntegrator{
		hookManager: NewWorktreeHookManager(executor),
		enabled:     true,
	}
}

// GetManager returns the worktree hook manager
func (whi *WorktreeHookIntegrator) GetManager() *WorktreeHookManager {
	return whi.hookManager
}

// Enable enables worktree hook integration
func (whi *WorktreeHookIntegrator) Enable() {
	whi.enabled = true
	whi.hookManager.SetEnabled(true)
}

// Disable disables worktree hook integration
func (whi *WorktreeHookIntegrator) Disable() {
	whi.enabled = false
	whi.hookManager.SetEnabled(false)
}

// IsEnabled returns whether worktree hook integration is enabled
func (whi *WorktreeHookIntegrator) IsEnabled() bool {
	return whi.enabled && whi.hookManager.IsEnabled()
}

// HandleWorktreeCreate handles worktree creation
func (whi *WorktreeHookIntegrator) HandleWorktreeCreate(worktreePath, branch, parentPath string) error {
	if !whi.IsEnabled() {
		return nil
	}

	projectName := extractProjectName(parentPath)
	return whi.hookManager.OnWorktreeCreated(worktreePath, branch, parentPath, projectName)
}

// HandleSessionCreate handles session creation
func (whi *WorktreeHookIntegrator) HandleSessionCreate(workingDir, branch, sessionID string) error {
	if !whi.IsEnabled() {
		return nil
	}

	projectName := extractProjectName(workingDir)
	sessionInfo := SessionInfo{
		SessionID:   sessionID,
		WorkingDir:  workingDir,
		Branch:      branch,
		ProjectName: projectName,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	return whi.hookManager.OnSessionCreated(sessionInfo)
}

// HandleSessionContinue handles session continuation
func (whi *WorktreeHookIntegrator) HandleSessionContinue(sessionInfo SessionInfo) error {
	if !whi.IsEnabled() {
		return nil
	}

	return whi.hookManager.OnSessionContinued(sessionInfo)
}

// HandleSessionResume handles session resumption
func (whi *WorktreeHookIntegrator) HandleSessionResume(sessionInfo SessionInfo, previousState string) error {
	if !whi.IsEnabled() {
		return nil
	}

	return whi.hookManager.OnSessionResumed(sessionInfo, previousState)
}

// HandleWorktreeBootstrap handles worktree bootstrapping with custom context
func (whi *WorktreeHookIntegrator) HandleWorktreeBootstrap(ctx context.Context, worktreePath, branch string, options BootstrapOptions) error {
	if !whi.IsEnabled() {
		return nil
	}

	hookContext := HookContext{
		WorktreePath:   worktreePath,
		WorktreeBranch: branch,
		ProjectName:    options.ProjectName,
		SessionType:    "new",
		CustomVars:     make(map[string]string),
	}

	// Add bootstrap-specific variables
	if options.ParentPath != "" {
		hookContext.CustomVars["CCMGR_PARENT_PATH"] = options.ParentPath
	}
	if options.Template != "" {
		hookContext.CustomVars["CCMGR_TEMPLATE"] = options.Template
	}
	if options.Language != "" {
		hookContext.CustomVars["CCMGR_LANGUAGE"] = options.Language
	}

	// Merge additional environment variables
	for key, value := range options.Environment {
		hookContext.CustomVars[key] = value
	}

	return whi.hookManager.executor.ExecuteWorktreeCreationHook(hookContext)
}

// BootstrapOptions contains options for worktree bootstrapping
type BootstrapOptions struct {
	ProjectName string
	ParentPath  string
	Template    string
	Language    string
	Environment map[string]string
}

// extractProjectName extracts the project name from a path
func extractProjectName(path string) string {
	if path == "" {
		return ""
	}

	// Get the last component of the path
	projectName := filepath.Base(path)

	// Remove common worktree suffixes
	if idx := findWorktreeSuffix(projectName); idx > 0 {
		projectName = projectName[:idx]
	}

	return projectName
}

// findWorktreeSuffix finds the index of common worktree suffixes
func findWorktreeSuffix(name string) int {
	suffixes := []string{"-worktree", "_worktree", "-wt", "_wt"}

	for _, suffix := range suffixes {
		if idx := len(name) - len(suffix); idx > 0 && name[idx:] == suffix {
			return idx
		}
	}

	return -1
}
