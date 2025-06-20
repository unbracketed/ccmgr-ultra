package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultProcessTracker implements ProcessTracker interface
type DefaultProcessTracker struct {
	registry    *ProcessRegistry
	detector    ProcessDetector
	monitor     StateMonitor
	config      *ProcessConfig
	subscribers []StateChangeHandler
	running     bool
	mutex       sync.RWMutex
	stopCh      chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewDefaultProcessTracker creates a new process tracker
func NewDefaultProcessTracker(config *ProcessConfig, detector ProcessDetector, monitor StateMonitor) *DefaultProcessTracker {
	return &DefaultProcessTracker{
		registry:    NewProcessRegistry(config),
		detector:    detector,
		monitor:     monitor,
		config:      config,
		subscribers: make([]StateChangeHandler, 0),
		stopCh:      make(chan struct{}),
	}
}

// Start begins tracking processes
func (t *DefaultProcessTracker) Start(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.running {
		return fmt.Errorf("tracker is already running")
	}

	t.ctx, t.cancel = context.WithCancel(ctx)
	t.running = true

	// Start the monitor
	if err := t.monitor.Start(t.ctx); err != nil {
		return fmt.Errorf("failed to start monitor: %w", err)
	}

	// Start tracking loop
	go t.trackingLoop()
	go t.cleanupLoop()

	return nil
}

// Stop stops the tracking process
func (t *DefaultProcessTracker) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		return nil
	}

	t.running = false
	if t.cancel != nil {
		t.cancel()
	}

	// Stop the monitor
	if err := t.monitor.Stop(); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to stop monitor: %v\n", err)
	}

	select {
	case <-t.stopCh:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for tracker to stop")
	}
}

// AddProcess adds a new process to track
func (t *DefaultProcessTracker) AddProcess(process *ProcessInfo) error {
	t.registry.mutex.Lock()
	defer t.registry.mutex.Unlock()

	if len(t.registry.processes) >= t.config.MaxProcesses {
		return fmt.Errorf("maximum number of processes (%d) already being tracked", t.config.MaxProcesses)
	}

	processID := process.SessionID
	if _, exists := t.registry.processes[processID]; exists {
		return fmt.Errorf("process %s is already being tracked", processID)
	}

	t.registry.processes[processID] = process
	
	// Notify subscribers of new process
	event := StateChangeEvent{
		ProcessID:   processID,
		PID:         process.PID,
		OldState:    StateUnknown,
		NewState:    process.GetState(),
		Timestamp:   time.Now(),
		SessionID:   process.SessionID,
		WorktreeID:  process.WorktreeID,
		TmuxSession: process.TmuxSession,
		WorkingDir:  process.WorkingDir,
	}

	go t.notifySubscribers(event)
	return nil
}

// RemoveProcess removes a process from tracking
func (t *DefaultProcessTracker) RemoveProcess(processID string) error {
	t.registry.mutex.Lock()
	defer t.registry.mutex.Unlock()

	process, exists := t.registry.processes[processID]
	if !exists {
		return fmt.Errorf("process %s is not being tracked", processID)
	}

	delete(t.registry.processes, processID)

	// Notify subscribers of process removal
	event := StateChangeEvent{
		ProcessID:   processID,
		PID:         process.PID,
		OldState:    process.GetState(),
		NewState:    StateStopped,
		Timestamp:   time.Now(),
		SessionID:   process.SessionID,
		WorktreeID:  process.WorktreeID,
		TmuxSession: process.TmuxSession,
		WorkingDir:  process.WorkingDir,
	}

	go t.notifySubscribers(event)
	return nil
}

// GetProcess retrieves a process by ID
func (t *DefaultProcessTracker) GetProcess(processID string) (*ProcessInfo, bool) {
	t.registry.mutex.RLock()
	defer t.registry.mutex.RUnlock()

	process, exists := t.registry.processes[processID]
	return process, exists
}

// GetAllProcesses returns all tracked processes
func (t *DefaultProcessTracker) GetAllProcesses() []*ProcessInfo {
	t.registry.mutex.RLock()
	defer t.registry.mutex.RUnlock()

	processes := make([]*ProcessInfo, 0, len(t.registry.processes))
	for _, process := range t.registry.processes {
		processes = append(processes, process)
	}

	return processes
}

// GetProcessesByState returns processes in a specific state
func (t *DefaultProcessTracker) GetProcessesByState(state ProcessState) []*ProcessInfo {
	t.registry.mutex.RLock()
	defer t.registry.mutex.RUnlock()

	var processes []*ProcessInfo
	for _, process := range t.registry.processes {
		if process.GetState() == state {
			processes = append(processes, process)
		}
	}

	return processes
}

// GetProcessesByWorktree returns processes associated with a worktree
func (t *DefaultProcessTracker) GetProcessesByWorktree(worktreeID string) []*ProcessInfo {
	t.registry.mutex.RLock()
	defer t.registry.mutex.RUnlock()

	var processes []*ProcessInfo
	for _, process := range t.registry.processes {
		if process.WorktreeID == worktreeID {
			processes = append(processes, process)
		}
	}

	return processes
}

// Subscribe adds a state change handler
func (t *DefaultProcessTracker) Subscribe(handler StateChangeHandler) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.subscribers = append(t.subscribers, handler)
	return nil
}

// Unsubscribe removes a state change handler
func (t *DefaultProcessTracker) Unsubscribe(handler StateChangeHandler) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for i, subscriber := range t.subscribers {
		// Note: This is a simple pointer comparison. In a real implementation,
		// you might want a more sophisticated subscription management system.
		if subscriber == handler {
			t.subscribers = append(t.subscribers[:i], t.subscribers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("handler not found in subscribers")
}

// trackingLoop is the main tracking loop
func (t *DefaultProcessTracker) trackingLoop() {
	defer func() {
		t.stopCh <- struct{}{}
	}()

	discoveryTicker := time.NewTicker(t.config.PollInterval * 2) // Discover less frequently
	monitorTicker := time.NewTicker(t.config.PollInterval)
	defer discoveryTicker.Stop()
	defer monitorTicker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-discoveryTicker.C:
			t.discoverNewProcesses()
		case <-monitorTicker.C:
			t.monitorExistingProcesses()
		}
	}
}

// cleanupLoop handles periodic cleanup tasks
func (t *DefaultProcessTracker) cleanupLoop() {
	ticker := time.NewTicker(t.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.cleanupStaleProcesses()
		}
	}
}

// discoverNewProcesses finds and adds new Claude Code processes
func (t *DefaultProcessTracker) discoverNewProcesses() {
	processes, err := t.detector.DetectProcesses(t.ctx)
	if err != nil {
		fmt.Printf("Error detecting processes: %v\n", err)
		return
	}

	for _, process := range processes {
		// Check if we're already tracking this process
		if _, exists := t.GetProcess(process.SessionID); !exists {
			if err := t.AddProcess(process); err != nil {
				fmt.Printf("Error adding process %s: %v\n", process.SessionID, err)
			}
		}
	}
}

// monitorExistingProcesses checks state of existing processes
func (t *DefaultProcessTracker) monitorExistingProcesses() {
	processes := t.GetAllProcesses()

	for _, process := range processes {
		// Check if process still exists and update its state
		currentState, err := t.monitor.MonitorState(t.ctx, process)
		if err != nil {
			fmt.Printf("Error monitoring process %s: %v\n", process.SessionID, err)
			continue
		}

		oldState := process.GetState()
		if currentState != oldState {
			process.SetState(currentState)

			// Notify subscribers of state change
			event := StateChangeEvent{
				ProcessID:   process.SessionID,
				PID:         process.PID,
				OldState:    oldState,
				NewState:    currentState,
				Timestamp:   time.Now(),
				SessionID:   process.SessionID,
				WorktreeID:  process.WorktreeID,
				TmuxSession: process.TmuxSession,
				WorkingDir:  process.WorkingDir,
			}

			go t.notifySubscribers(event)
		}

		// If process is stopped, remove it from tracking
		if currentState == StateStopped {
			if err := t.RemoveProcess(process.SessionID); err != nil {
				fmt.Printf("Error removing stopped process %s: %v\n", process.SessionID, err)
			}
		}
	}
}

// cleanupStaleProcesses removes processes that haven't been updated recently
func (t *DefaultProcessTracker) cleanupStaleProcesses() {
	t.registry.mutex.Lock()
	defer t.registry.mutex.Unlock()

	cutoff := time.Now().Add(-t.config.StateTimeout * 2)
	var staleProcesses []string

	for processID, process := range t.registry.processes {
		if process.LastUpdate.Before(cutoff) {
			staleProcesses = append(staleProcesses, processID)
		}
	}

	// Remove stale processes
	for _, processID := range staleProcesses {
		if process, exists := t.registry.processes[processID]; exists {
			delete(t.registry.processes, processID)

			// Notify subscribers
			event := StateChangeEvent{
				ProcessID:   processID,
				PID:         process.PID,
				OldState:    process.GetState(),
				NewState:    StateStopped,
				Timestamp:   time.Now(),
				SessionID:   process.SessionID,
				WorktreeID:  process.WorktreeID,
				TmuxSession: process.TmuxSession,
				WorkingDir:  process.WorkingDir,
			}

			go t.notifySubscribers(event)
		}
	}
}

// notifySubscribers sends state change events to all subscribers
func (t *DefaultProcessTracker) notifySubscribers(event StateChangeEvent) {
	t.mutex.RLock()
	subscribers := make([]StateChangeHandler, len(t.subscribers))
	copy(subscribers, t.subscribers)
	t.mutex.RUnlock()

	for _, handler := range subscribers {
		func() {
			// Use a timeout to prevent blocking
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := handler.OnStateChange(ctx, event); err != nil {
				fmt.Printf("Error in state change handler: %v\n", err)
			}
		}()
	}
}

// GetTrackerStats returns statistics about the tracker
func (t *DefaultProcessTracker) GetTrackerStats() map[string]interface{} {
	stats := t.registry.GetStats()
	
	return map[string]interface{}{
		"running":             t.running,
		"total_processes":     stats.TotalProcesses,
		"state_distribution":  stats.StateDistribution,
		"average_uptime":      stats.AverageUptime.String(),
		"subscribers":         len(t.subscribers),
		"max_processes":       t.config.MaxProcesses,
		"poll_interval":       t.config.PollInterval.String(),
		"cleanup_interval":    t.config.CleanupInterval.String(),
		"last_updated":        stats.LastUpdated,
	}
}

// RefreshProcess manually refreshes a specific process
func (t *DefaultProcessTracker) RefreshProcess(processID string) error {
	process, exists := t.GetProcess(processID)
	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	// Update process information using detector
	if detector, ok := t.detector.(*DefaultDetector); ok {
		if err := detector.RefreshProcess(process); err != nil {
			return fmt.Errorf("failed to refresh process: %w", err)
		}
	}

	// Check state
	currentState, err := t.monitor.MonitorState(t.ctx, process)
	if err != nil {
		return fmt.Errorf("failed to monitor process state: %w", err)
	}

	oldState := process.GetState()
	if currentState != oldState {
		process.SetState(currentState)

		// Notify subscribers
		event := StateChangeEvent{
			ProcessID:   processID,
			PID:         process.PID,
			OldState:    oldState,
			NewState:    currentState,
			Timestamp:   time.Now(),
			SessionID:   process.SessionID,
			WorktreeID:  process.WorktreeID,
			TmuxSession: process.TmuxSession,
			WorkingDir:  process.WorkingDir,
		}

		go t.notifySubscribers(event)
	}

	return nil
}

// IsRunning returns whether the tracker is currently running
func (t *DefaultProcessTracker) IsRunning() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.running
}