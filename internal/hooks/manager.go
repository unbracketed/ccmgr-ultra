package hooks

import (
	"context"
	"sync"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// Manager is the main hook management system
type Manager struct {
	config              *config.Config
	executor            HookExecutor
	statusIntegrator    *StatusHookIntegrator
	worktreeIntegrator  *WorktreeHookIntegrator
	enabled             bool
	mu                  sync.RWMutex
}

// NewManager creates a new hook manager
func NewManager(cfg *config.Config) *Manager {
	executor := NewDefaultExecutor(cfg)
	
	return &Manager{
		config:             cfg,
		executor:           executor,
		statusIntegrator:   NewStatusHookIntegrator(executor),
		worktreeIntegrator: NewWorktreeHookIntegrator(executor),
		enabled:            true,
	}
}

// Enable enables all hook execution
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.enabled = true
	m.statusIntegrator.Enable()
	m.worktreeIntegrator.Enable()
}

// Disable disables all hook execution
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.enabled = false
	m.statusIntegrator.Disable()
	m.worktreeIntegrator.Disable()
}

// IsEnabled returns whether hooks are globally enabled
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// GetStatusIntegrator returns the status hook integrator
func (m *Manager) GetStatusIntegrator() *StatusHookIntegrator {
	return m.statusIntegrator
}

// GetWorktreeIntegrator returns the worktree hook integrator
func (m *Manager) GetWorktreeIntegrator() *WorktreeHookIntegrator {
	return m.worktreeIntegrator
}

// GetExecutor returns the hook executor
func (m *Manager) GetExecutor() HookExecutor {
	return m.executor
}

// UpdateConfig updates the configuration and reinitializes components
func (m *Manager) UpdateConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.config = cfg
	
	// Create new executor with updated config
	m.executor = NewDefaultExecutor(cfg)
	
	// Update integrators
	m.statusIntegrator = NewStatusHookIntegrator(m.executor)
	m.worktreeIntegrator = NewWorktreeHookIntegrator(m.executor)
	
	// Restore enabled state
	if m.enabled {
		m.statusIntegrator.Enable()
		m.worktreeIntegrator.Enable()
	} else {
		m.statusIntegrator.Disable()
		m.worktreeIntegrator.Disable()
	}
}

// Start starts background hook management routines
func (m *Manager) Start(ctx context.Context) {
	// Start cleanup routine for status hooks
	go m.statusIntegrator.GetManager().StartCleanupRoutine(ctx)
}

// ExecuteHook executes a hook directly
func (m *Manager) ExecuteHook(ctx context.Context, hookType HookType, hookCtx HookContext) error {
	if !m.IsEnabled() {
		return nil
	}
	
	return m.executor.Execute(ctx, hookType, hookCtx)
}

// ExecuteHookAsync executes a hook asynchronously
func (m *Manager) ExecuteHookAsync(hookType HookType, hookCtx HookContext) <-chan error {
	if !m.IsEnabled() {
		errChan := make(chan error, 1)
		close(errChan)
		return errChan
	}
	
	return m.executor.ExecuteAsync(hookType, hookCtx)
}

// Convenience methods for common hook operations

// OnClaudeStateChange handles Claude Code state changes
func (m *Manager) OnClaudeStateChange(oldState, newState string, context HookContext) {
	if m.IsEnabled() && m.statusIntegrator.IsEnabled() {
		m.statusIntegrator.GetManager().OnStateChange(oldState, newState, context)
	}
}

// OnWorktreeCreated handles worktree creation
func (m *Manager) OnWorktreeCreated(worktreePath, branch, parentPath, projectName string) error {
	if !m.IsEnabled() || !m.worktreeIntegrator.IsEnabled() {
		return nil
	}
	
	return m.worktreeIntegrator.GetManager().OnWorktreeCreated(worktreePath, branch, parentPath, projectName)
}

// OnSessionCreated handles session creation
func (m *Manager) OnSessionCreated(sessionInfo SessionInfo) error {
	if !m.IsEnabled() || !m.worktreeIntegrator.IsEnabled() {
		return nil
	}
	
	return m.worktreeIntegrator.GetManager().OnSessionCreated(sessionInfo)
}

// OnSessionContinued handles session continuation
func (m *Manager) OnSessionContinued(sessionInfo SessionInfo) error {
	if !m.IsEnabled() || !m.worktreeIntegrator.IsEnabled() {
		return nil
	}
	
	return m.worktreeIntegrator.GetManager().OnSessionContinued(sessionInfo)
}

// OnSessionResumed handles session resumption
func (m *Manager) OnSessionResumed(sessionInfo SessionInfo, previousState string) error {
	if !m.IsEnabled() || !m.worktreeIntegrator.IsEnabled() {
		return nil
	}
	
	return m.worktreeIntegrator.GetManager().OnSessionResumed(sessionInfo, previousState)
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *config.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetStats returns hook execution statistics
func (m *Manager) GetStats() HookStats {
	// This would be implemented with actual metrics collection
	return HookStats{
		TotalExecutions: 0,
		SuccessCount:    0,
		FailureCount:    0,
		LastExecution:   nil,
	}
}

// HookStats contains hook execution statistics
type HookStats struct {
	TotalExecutions int64
	SuccessCount    int64
	FailureCount    int64
	LastExecution   *HookResult
}