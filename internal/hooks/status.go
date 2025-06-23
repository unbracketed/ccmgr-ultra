package hooks

import (
	"context"
	"log"
	"sync"
	"time"
)

// StatusHookManager manages status hook execution
type StatusHookManager struct {
	executor         HookExecutor
	enabled          bool
	debounceInterval time.Duration
	lastStateChange  map[string]time.Time
	mu               sync.RWMutex
}

// NewStatusHookManager creates a new status hook manager
func NewStatusHookManager(executor HookExecutor) *StatusHookManager {
	return &StatusHookManager{
		executor:         executor,
		enabled:          true,
		debounceInterval: 1 * time.Second, // Debounce rapid state changes
		lastStateChange:  make(map[string]time.Time),
	}
}

// SetEnabled enables or disables status hook execution
func (shm *StatusHookManager) SetEnabled(enabled bool) {
	shm.mu.Lock()
	defer shm.mu.Unlock()
	shm.enabled = enabled
}

// IsEnabled returns whether status hooks are enabled
func (shm *StatusHookManager) IsEnabled() bool {
	shm.mu.RLock()
	defer shm.mu.RUnlock()
	return shm.enabled
}

// OnStateChange handles a state change event and triggers appropriate hooks
func (shm *StatusHookManager) OnStateChange(oldState, newState string, context HookContext) {
	if !shm.IsEnabled() {
		return
	}

	// Debounce rapid state changes for the same process/session
	key := context.SessionID
	if key == "" {
		key = context.WorktreePath
	}

	shm.mu.Lock()
	now := time.Now()
	if lastChange, exists := shm.lastStateChange[key]; exists {
		if now.Sub(lastChange) < shm.debounceInterval {
			shm.mu.Unlock()
			return // Skip this state change due to debouncing
		}
	}
	shm.lastStateChange[key] = now
	shm.mu.Unlock()

	// Map state to hook type
	hookType, ok := mapStateToHookType(newState)
	if !ok {
		return // Unknown state, skip
	}

	// Update context with state information
	context.OldState = oldState
	context.NewState = newState

	// Execute the appropriate status hook
	if err := shm.executor.ExecuteStatusHook(hookType, context); err != nil {
		log.Printf("Status hook execution failed for state %s: %v", newState, err)
	}
}

// OnIdleState triggers the idle hook
func (shm *StatusHookManager) OnIdleState(context HookContext) {
	shm.OnStateChange("", "idle", context)
}

// OnBusyState triggers the busy hook
func (shm *StatusHookManager) OnBusyState(context HookContext) {
	shm.OnStateChange("", "busy", context)
}

// OnWaitingState triggers the waiting hook
func (shm *StatusHookManager) OnWaitingState(context HookContext) {
	shm.OnStateChange("", "waiting", context)
}

// CleanupDebounceMap cleans up old entries from the debounce map
func (shm *StatusHookManager) CleanupDebounceMap() {
	shm.mu.Lock()
	defer shm.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute) // Clean entries older than 5 minutes
	for key, timestamp := range shm.lastStateChange {
		if timestamp.Before(cutoff) {
			delete(shm.lastStateChange, key)
		}
	}
}

// StartCleanupRoutine starts a background routine to clean up the debounce map
func (shm *StatusHookManager) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			shm.CleanupDebounceMap()
		case <-ctx.Done():
			return
		}
	}
}

// mapStateToHookType maps a state string to a hook type
func mapStateToHookType(state string) (HookType, bool) {
	switch state {
	case "idle":
		return HookTypeStatusIdle, true
	case "busy", "processing", "executing", "running", "working", "analyzing", "generating":
		return HookTypeStatusBusy, true
	case "waiting", "waiting_for_input", "waiting_for_confirmation", "confirm", "prompt":
		return HookTypeStatusWaiting, true
	default:
		return HookTypeStatusIdle, false // Unknown state
	}
}

// StatusHookIntegrator provides integration points for the status hook system
type StatusHookIntegrator struct {
	hookManager *StatusHookManager
	enabled     bool
}

// NewStatusHookIntegrator creates a new status hook integrator
func NewStatusHookIntegrator(executor HookExecutor) *StatusHookIntegrator {
	return &StatusHookIntegrator{
		hookManager: NewStatusHookManager(executor),
		enabled:     true,
	}
}

// GetManager returns the status hook manager
func (shi *StatusHookIntegrator) GetManager() *StatusHookManager {
	return shi.hookManager
}

// Enable enables status hook integration
func (shi *StatusHookIntegrator) Enable() {
	shi.enabled = true
	shi.hookManager.SetEnabled(true)
}

// Disable disables status hook integration
func (shi *StatusHookIntegrator) Disable() {
	shi.enabled = false
	shi.hookManager.SetEnabled(false)
}

// IsEnabled returns whether status hook integration is enabled
func (shi *StatusHookIntegrator) IsEnabled() bool {
	return shi.enabled && shi.hookManager.IsEnabled()
}

// HandleProcessStateChange handles a process state change event
func (shi *StatusHookIntegrator) HandleProcessStateChange(processID string, oldState, newState string, workingDir, branch, sessionID string) {
	if !shi.IsEnabled() {
		return
	}

	context := HookContext{
		WorktreePath:   workingDir,
		WorktreeBranch: branch,
		SessionID:      sessionID,
		OldState:       oldState,
		NewState:       newState,
		CustomVars: map[string]string{
			"CCMGR_PROCESS_ID": processID,
		},
	}

	shi.hookManager.OnStateChange(oldState, newState, context)
}

// HandleClaudeProcessEvent handles a Claude Code process event
func (shi *StatusHookIntegrator) HandleClaudeProcessEvent(event string, processInfo map[string]interface{}) {
	if !shi.IsEnabled() {
		return
	}

	// Extract information from process event
	var context HookContext

	if workingDir, ok := processInfo["working_dir"].(string); ok {
		context.WorktreePath = workingDir
	}

	if branch, ok := processInfo["branch"].(string); ok {
		context.WorktreeBranch = branch
	}

	if sessionID, ok := processInfo["session_id"].(string); ok {
		context.SessionID = sessionID
	}

	if projectName, ok := processInfo["project_name"].(string); ok {
		context.ProjectName = projectName
	}

	// Add custom variables from process info
	context.CustomVars = make(map[string]string)
	for key, value := range processInfo {
		if strValue, ok := value.(string); ok {
			context.CustomVars["CCMGR_"+key] = strValue
		}
	}

	// Trigger appropriate hook based on event
	switch event {
	case "process_idle", "idle":
		shi.hookManager.OnIdleState(context)
	case "process_busy", "busy", "processing":
		shi.hookManager.OnBusyState(context)
	case "process_waiting", "waiting", "waiting_for_input":
		shi.hookManager.OnWaitingState(context)
	default:
		// Try to map the event to a state
		if hookType, ok := mapStateToHookType(event); ok {
			context.NewState = event
			if err := shi.hookManager.executor.ExecuteStatusHook(hookType, context); err != nil {
				log.Printf("Status hook execution failed for event %s: %v", event, err)
			}
		}
	}
}
