package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
	// "github.com/unbracketed/ccmgr-ultra/internal/analytics" // Commented out to avoid import cycle
)

// ProcessManager provides a unified API for Claude Code process management
type ProcessManager struct {
	config   *ProcessConfig
	detector ProcessDetector
	monitor  StateMonitor
	tracker  ProcessTracker
	handlers []StateChangeHandler
	// eventChan   chan<- analytics.AnalyticsEvent // Commented out to avoid import cycle
	running bool
	mutex   sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewProcessManager creates a new process manager with default implementations
func NewProcessManager(config *ProcessConfig) (*ProcessManager, error) {
	if config == nil {
		config = &ProcessConfig{}
		config.SetDefaults()
	}

	// Create detector
	detector, err := NewDefaultDetector(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create detector: %w", err)
	}

	// Create monitor
	monitor := NewDefaultStateMonitor(config, detector)

	// Create tracker
	tracker := NewDefaultProcessTracker(config, detector, monitor)

	return &ProcessManager{
		config:   config,
		detector: detector,
		monitor:  monitor,
		tracker:  tracker,
		handlers: make([]StateChangeHandler, 0),
	}, nil
}

// Start initializes and starts the process management system
func (pm *ProcessManager) Start(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.running {
		return fmt.Errorf("process manager is already running")
	}

	pm.ctx, pm.cancel = context.WithCancel(ctx)
	pm.running = true

	// Start the tracker (which will start the monitor)
	if err := pm.tracker.Start(pm.ctx); err != nil {
		pm.running = false
		return fmt.Errorf("failed to start tracker: %w", err)
	}

	// Subscribe to state changes to forward to our handlers
	if err := pm.tracker.Subscribe(pm); err != nil {
		pm.running = false
		return fmt.Errorf("failed to subscribe to tracker: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the process management system
func (pm *ProcessManager) Stop() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.running {
		return nil
	}

	pm.running = false

	// Stop the tracker
	if err := pm.tracker.Stop(); err != nil {
		return fmt.Errorf("failed to stop tracker: %w", err)
	}

	if pm.cancel != nil {
		pm.cancel()
	}

	return nil
}

// IsRunning returns whether the process manager is currently running
func (pm *ProcessManager) IsRunning() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.running
}

// GetAllProcesses returns all currently tracked processes
func (pm *ProcessManager) GetAllProcesses() []*ProcessInfo {
	return pm.tracker.GetAllProcesses()
}

// GetProcess retrieves a specific process by ID
func (pm *ProcessManager) GetProcess(processID string) (*ProcessInfo, bool) {
	return pm.tracker.GetProcess(processID)
}

// GetProcessesByState returns processes in a specific state
func (pm *ProcessManager) GetProcessesByState(state ProcessState) []*ProcessInfo {
	return pm.tracker.GetProcessesByState(state)
}

// GetProcessesByWorktree returns processes associated with a worktree
func (pm *ProcessManager) GetProcessesByWorktree(worktreeID string) []*ProcessInfo {
	return pm.tracker.GetProcessesByWorktree(worktreeID)
}

// RefreshProcess manually refreshes a specific process
func (pm *ProcessManager) RefreshProcess(processID string) error {
	if tracker, ok := pm.tracker.(*DefaultProcessTracker); ok {
		return tracker.RefreshProcess(processID)
	}
	return fmt.Errorf("refresh not supported by current tracker implementation")
}

// RefreshAll manually refreshes all tracked processes
func (pm *ProcessManager) RefreshAll() error {
	processes := pm.GetAllProcesses()
	var errors []error

	for _, process := range processes {
		if err := pm.RefreshProcess(process.SessionID); err != nil {
			errors = append(errors, fmt.Errorf("failed to refresh process %s: %w", process.SessionID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to refresh %d processes: %v", len(errors), errors[0])
	}

	return nil
}

// AddStateChangeHandler adds a handler for state change events
func (pm *ProcessManager) AddStateChangeHandler(handler StateChangeHandler) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.handlers = append(pm.handlers, handler)
	return nil
}

// RemoveStateChangeHandler removes a state change handler
func (pm *ProcessManager) RemoveStateChangeHandler(handler StateChangeHandler) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for i, h := range pm.handlers {
		if h == handler {
			pm.handlers = append(pm.handlers[:i], pm.handlers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("handler not found")
}

// OnStateChange implements StateChangeHandler interface to forward events
func (pm *ProcessManager) OnStateChange(ctx context.Context, event StateChangeEvent) error {
	pm.mutex.RLock()
	handlers := make([]StateChangeHandler, len(pm.handlers))
	copy(handlers, pm.handlers)
	// eventChan := pm.eventChan // commented out due to import cycle
	pm.mutex.RUnlock()

	// Emit analytics event if channel is available
	// Commented out due to import cycle
	// _ = eventChan

	// Forward to all registered handlers
	for _, handler := range handlers {
		if err := handler.OnStateChange(ctx, event); err != nil {
			// Log error but continue with other handlers
			fmt.Printf("Error in state change handler: %v\n", err)
		}
	}

	return nil
}

// GetStats returns comprehensive statistics about the process management system
func (pm *ProcessManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Get tracker stats
	if tracker, ok := pm.tracker.(*DefaultProcessTracker); ok {
		trackerStats := tracker.GetTrackerStats()
		for k, v := range trackerStats {
			stats[k] = v
		}
	}

	// Get monitor stats
	if monitor, ok := pm.monitor.(*DefaultStateMonitor); ok {
		monitorStats := monitor.GetMonitorStats()
		for k, v := range monitorStats {
			stats["monitor_"+k] = v
		}
	}

	// Add manager-specific stats
	stats["manager_running"] = pm.IsRunning()
	stats["manager_handlers"] = len(pm.handlers)
	stats["config_poll_interval"] = pm.config.PollInterval.String()
	stats["config_max_processes"] = pm.config.MaxProcesses

	return stats
}

// GetConfig returns the current configuration
func (pm *ProcessManager) GetConfig() *ProcessConfig {
	return pm.config
}

// UpdateConfig updates the process manager configuration
func (pm *ProcessManager) UpdateConfig(config *ProcessConfig) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.running {
		return fmt.Errorf("cannot update configuration while process manager is running")
	}

	pm.config = config
	return nil
}

// SetEventChannel sets the analytics event channel for event emission
// func (pm *ProcessManager) SetEventChannel(eventChan chan<- analytics.AnalyticsEvent) {
//	pm.mutex.Lock()
//	defer pm.mutex.Unlock()
//	pm.eventChan = eventChan
// }

// EmitSessionEvent emits a session-related analytics event
func (pm *ProcessManager) EmitSessionEvent(eventType, sessionID, project, worktree, branch, directory string) {
	pm.mutex.RLock()
	// eventChan := pm.eventChan // commented out due to import cycle
	pm.mutex.RUnlock()

	// if eventChan == nil {
	//	return
	// }

	// analyticsEvent := analytics.AnalyticsEvent{
	// 	Type:      eventType,
	// 	Timestamp: time.Now(),
	// 	SessionID: sessionID,
	// Data: analytics.NewSessionEventData(
	//	eventType,
	//		project,
	//		worktree,
	//		branch,
	//		directory,
	// 		),
	// 	}

	// Non-blocking send - commented out due to import cycle
	// select {
	// case eventChan <- analyticsEvent:
	// default:
	//	// Channel is full, skip this event to avoid blocking
	// }
}

// DiscoverProcesses manually triggers process discovery
func (pm *ProcessManager) DiscoverProcesses() ([]*ProcessInfo, error) {
	return pm.detector.DetectProcesses(pm.ctx)
}

// WaitForState waits for a process to reach a specific state with timeout
func (pm *ProcessManager) WaitForState(processID string, targetState ProcessState, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(pm.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for process %s to reach state %s", processID, targetState.String())
		case <-ticker.C:
			if process, exists := pm.GetProcess(processID); exists {
				if process.GetState() == targetState {
					return nil
				}
			} else {
				return fmt.Errorf("process %s not found", processID)
			}
		}
	}
}

// WaitForAnyState waits for a process to reach any of the specified states
func (pm *ProcessManager) WaitForAnyState(processID string, targetStates []ProcessState, timeout time.Duration) (ProcessState, error) {
	ctx, cancel := context.WithTimeout(pm.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return StateUnknown, fmt.Errorf("timeout waiting for process %s to reach any target state", processID)
		case <-ticker.C:
			if process, exists := pm.GetProcess(processID); exists {
				currentState := process.GetState()
				for _, targetState := range targetStates {
					if currentState == targetState {
						return currentState, nil
					}
				}
			} else {
				return StateUnknown, fmt.Errorf("process %s not found", processID)
			}
		}
	}
}

// GetProcessHealth returns health information for a process
func (pm *ProcessManager) GetProcessHealth(processID string) (*ProcessHealth, error) {
	process, exists := pm.GetProcess(processID)
	if !exists {
		return nil, fmt.Errorf("process %s not found", processID)
	}

	uptime := time.Since(process.StartTime)
	lastUpdate := time.Since(process.LastUpdate)

	health := &ProcessHealth{
		ProcessID:        processID,
		State:            process.GetState(),
		Uptime:           uptime,
		LastUpdate:       lastUpdate,
		CPUPercent:       process.CPUPercent,
		MemoryMB:         process.MemoryMB,
		IsResponsive:     lastUpdate < pm.config.StateTimeout,
		IsHealthy:        process.GetState() != StateError && process.GetState() != StateStopped,
		WorkingDirectory: process.WorkingDir,
		TmuxSession:      process.TmuxSession,
		WorktreeID:       process.WorktreeID,
	}

	return health, nil
}

// GetSystemHealth returns overall system health
func (pm *ProcessManager) GetSystemHealth() *SystemHealth {
	processes := pm.GetAllProcesses()
	stats := pm.GetStats()

	health := &SystemHealth{
		TotalProcesses:     len(processes),
		HealthyProcesses:   0,
		UnhealthyProcesses: 0,
		StateDistribution:  make(map[ProcessState]int),
		IsManagerRunning:   pm.IsRunning(),
		LastUpdated:        time.Now(),
	}

	for _, process := range processes {
		state := process.GetState()
		health.StateDistribution[state]++

		if state != StateError && state != StateStopped {
			health.HealthyProcesses++
		} else {
			health.UnhealthyProcesses++
		}
	}

	// Extract average uptime from stats if available
	if avgUptime, ok := stats["average_uptime"].(string); ok {
		if duration, err := time.ParseDuration(avgUptime); err == nil {
			health.AverageUptime = duration
		}
	}

	return health
}

// ProcessHealth represents health information for a single process
type ProcessHealth struct {
	ProcessID        string        `json:"process_id"`
	State            ProcessState  `json:"state"`
	Uptime           time.Duration `json:"uptime"`
	LastUpdate       time.Duration `json:"last_update"`
	CPUPercent       float64       `json:"cpu_percent"`
	MemoryMB         int64         `json:"memory_mb"`
	IsResponsive     bool          `json:"is_responsive"`
	IsHealthy        bool          `json:"is_healthy"`
	WorkingDirectory string        `json:"working_directory"`
	TmuxSession      string        `json:"tmux_session,omitempty"`
	WorktreeID       string        `json:"worktree_id,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	TotalProcesses     int                  `json:"total_processes"`
	HealthyProcesses   int                  `json:"healthy_processes"`
	UnhealthyProcesses int                  `json:"unhealthy_processes"`
	StateDistribution  map[ProcessState]int `json:"state_distribution"`
	AverageUptime      time.Duration        `json:"average_uptime"`
	IsManagerRunning   bool                 `json:"is_manager_running"`
	LastUpdated        time.Time            `json:"last_updated"`
}
