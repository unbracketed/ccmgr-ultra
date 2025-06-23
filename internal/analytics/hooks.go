package analytics

import (
	"context"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/claude"
	"github.com/bcdekker/ccmgr-ultra/internal/hooks"
)

// HooksConfig defines configuration for hook-based analytics collection
type HooksConfig struct {
	Enabled               bool `yaml:"enabled" json:"enabled" default:"true"`
	CaptureStateChanges   bool `yaml:"capture_state_changes" json:"capture_state_changes" default:"true"`
	CaptureWorktreeEvents bool `yaml:"capture_worktree_events" json:"capture_worktree_events" default:"true"`
	CaptureSessionEvents  bool `yaml:"capture_session_events" json:"capture_session_events" default:"true"`
}

// SetDefaults sets default values for HooksConfig
func (h *HooksConfig) SetDefaults() {
	h.Enabled = true
	h.CaptureStateChanges = true
	h.CaptureWorktreeEvents = true
	h.CaptureSessionEvents = true
}

// Validate validates the hooks configuration
func (h *HooksConfig) Validate() error {
	// No specific validation needed for now
	return nil
}

// HooksCollector integrates with the existing hooks system to collect analytics events
type HooksCollector struct {
	collector EventCollector
	config    *HooksConfig
}

// NewHooksCollector creates a new hooks-based analytics collector
func NewHooksCollector(collector EventCollector, config *HooksConfig) *HooksCollector {
	if config == nil {
		config = &HooksConfig{}
		config.SetDefaults()
	}

	return &HooksCollector{
		collector: collector,
		config:    config,
	}
}

// OnStateChange implements StateChangeHandler interface for collecting analytics from state changes
func (hc *HooksCollector) OnStateChange(ctx context.Context, event claude.StateChangeEvent) error {
	if !hc.config.Enabled || !hc.config.CaptureStateChanges {
		return nil
	}

	analyticsEvent := AnalyticsEvent{
		Type:      EventTypeStateChange,
		Timestamp: event.Timestamp,
		SessionID: event.SessionID,
		Data: NewStateChangeEventData(
			event.OldState.String(),
			event.NewState.String(),
			event.WorktreeID,
			"", // Branch information not available in StateChangeEvent
		),
	}

	// Add additional context data
	analyticsEvent.Data["pid"] = event.PID
	analyticsEvent.Data["working_dir"] = event.WorkingDir
	if event.TmuxSession != "" {
		analyticsEvent.Data["tmux_session"] = event.TmuxSession
	}

	return hc.collector.CollectEvent(analyticsEvent)
}

// OnHookExecution captures analytics events from hook executions
func (hc *HooksCollector) OnHookExecution(hookType hooks.HookType, hookCtx hooks.HookContext, result *hooks.HookResult) error {
	if !hc.config.Enabled {
		return nil
	}

	// Determine event type based on hook type
	var eventType string
	var captureEnabled bool

	switch hookType {
	case hooks.HookTypeStatusIdle, hooks.HookTypeStatusBusy, hooks.HookTypeStatusWaiting:
		if !hc.config.CaptureStateChanges {
			return nil
		}
		eventType = EventTypeStateChange
		captureEnabled = true
	case hooks.HookTypeWorktreeCreation, hooks.HookTypeWorktreeActivation:
		if !hc.config.CaptureWorktreeEvents {
			return nil
		}
		eventType = EventTypeWorktreeSwitch
		captureEnabled = true
	default:
		captureEnabled = false
	}

	if !captureEnabled {
		return nil
	}

	// Create analytics event
	analyticsEvent := AnalyticsEvent{
		Type:      eventType,
		Timestamp: result.Timestamp,
		SessionID: hookCtx.SessionID,
		Data: map[string]interface{}{
			"hook_type":       hookType.String(),
			"hook_success":    result.Success,
			"hook_duration":   result.Duration.Milliseconds(),
			"hook_exit_code":  result.ExitCode,
			"project_name":    hookCtx.ProjectName,
			"worktree_path":   hookCtx.WorktreePath,
			"worktree_branch": hookCtx.WorktreeBranch,
			"session_type":    hookCtx.SessionType,
		},
	}

	// Add state change specific data
	if eventType == EventTypeStateChange {
		analyticsEvent.Data["old_state"] = hookCtx.OldState
		analyticsEvent.Data["new_state"] = hookCtx.NewState
	}

	// Add hook output if available (truncated to avoid large data)
	if result.Output != "" && len(result.Output) < 1000 {
		analyticsEvent.Data["hook_output"] = result.Output
	}

	// Add any custom variables
	if len(hookCtx.CustomVars) > 0 {
		analyticsEvent.Data["custom_vars"] = hookCtx.CustomVars
	}

	return hc.collector.CollectEvent(analyticsEvent)
}

// OnSessionEvent captures session-related events
func (hc *HooksCollector) OnSessionEvent(eventType, sessionID, project, worktree, branch, directory string) error {
	if !hc.config.Enabled || !hc.config.CaptureSessionEvents {
		return nil
	}

	analyticsEvent := AnalyticsEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Data: NewSessionEventData(
			eventType,
			project,
			worktree,
			branch,
			directory,
		),
	}

	return hc.collector.CollectEvent(analyticsEvent)
}

// OnWorktreeSwitch captures worktree switch events
func (hc *HooksCollector) OnWorktreeSwitch(sessionID, oldWorktree, newWorktree, oldBranch, newBranch, project string) error {
	if !hc.config.Enabled || !hc.config.CaptureWorktreeEvents {
		return nil
	}

	analyticsEvent := AnalyticsEvent{
		Type:      EventTypeWorktreeSwitch,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Data: map[string]interface{}{
			"action":       "switch",
			"old_worktree": oldWorktree,
			"new_worktree": newWorktree,
			"old_branch":   oldBranch,
			"new_branch":   newBranch,
			"project":      project,
		},
	}

	return hc.collector.CollectEvent(analyticsEvent)
}

// OnBranchChange captures branch change events
func (hc *HooksCollector) OnBranchChange(sessionID, worktree, oldBranch, newBranch, project string) error {
	if !hc.config.Enabled || !hc.config.CaptureStateChanges {
		return nil
	}

	analyticsEvent := AnalyticsEvent{
		Type:      EventTypeBranchChange,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Data: map[string]interface{}{
			"action":     "change",
			"worktree":   worktree,
			"old_branch": oldBranch,
			"new_branch": newBranch,
			"project":    project,
		},
	}

	return hc.collector.CollectEvent(analyticsEvent)
}

// OnActivityDetection captures activity and idle detection events
func (hc *HooksCollector) OnActivityDetection(sessionID, activityType string, duration time.Duration, metadata map[string]interface{}) error {
	if !hc.config.Enabled {
		return nil
	}

	var eventType string
	switch activityType {
	case "idle":
		eventType = EventTypeIdle
	case "active":
		eventType = EventTypeActivity
	default:
		eventType = EventTypeActivity
	}

	analyticsEvent := AnalyticsEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Data:      NewActivityEventData(activityType, duration),
	}

	// Add any additional metadata
	if metadata != nil {
		for key, value := range metadata {
			analyticsEvent.Data[key] = value
		}
	}

	return hc.collector.CollectEvent(analyticsEvent)
}

// GetConfig returns the current hooks configuration
func (hc *HooksCollector) GetConfig() *HooksConfig {
	return hc.config
}

// UpdateConfig updates the hooks configuration
func (hc *HooksCollector) UpdateConfig(config *HooksConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	hc.config = config
	return nil
}

// IsEnabled returns whether the hooks collector is enabled
func (hc *HooksCollector) IsEnabled() bool {
	return hc.config.Enabled
}

// GetStats returns statistics about hook-based event collection
func (hc *HooksCollector) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":                 hc.config.Enabled,
		"capture_state_changes":   hc.config.CaptureStateChanges,
		"capture_worktree_events": hc.config.CaptureWorktreeEvents,
		"capture_session_events":  hc.config.CaptureSessionEvents,
		"collector_running":       hc.collector.IsRunning(),
	}
}
