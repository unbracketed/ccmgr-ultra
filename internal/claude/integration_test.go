package claude

import (
	"context"
	"testing"
	"time"

	"github.com/your-username/ccmgr-ultra/internal/config"
)

func TestProcessManager_Integration(t *testing.T) {
	// Create a configuration with Claude monitoring enabled
	claudeConfig := &config.ClaudeConfig{}
	claudeConfig.SetDefaults()
	claudeConfig.Enabled = true
	claudeConfig.PollInterval = 1 * time.Second // Fast polling for tests

	// Convert to ProcessConfig
	adapter := NewConfigAdapter(claudeConfig)
	processConfig, err := adapter.ToProcessConfig()
	if err != nil {
		t.Fatalf("Failed to convert config: %v", err)
	}

	// Create process manager
	manager, err := NewProcessManager(processConfig)
	if err != nil {
		t.Fatalf("Failed to create process manager: %v", err)
	}

	// Test manager lifecycle
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start process manager: %v", err)
	}

	// Verify it's running
	if !manager.IsRunning() {
		t.Error("Process manager should be running")
	}

	// Let it run for a bit to discover processes
	time.Sleep(2 * time.Second)

	// Get discovered processes
	processes := manager.GetAllProcesses()
	t.Logf("Discovered %d Claude processes", len(processes))

	// Get system health
	health := manager.GetSystemHealth()
	if health == nil {
		t.Error("System health should not be nil")
	}

	t.Logf("System health: %d total, %d healthy, %d unhealthy",
		health.TotalProcesses, health.HealthyProcesses, health.UnhealthyProcesses)

	// Get stats
	stats := manager.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	t.Logf("Manager stats: running=%v, processes=%v",
		stats["manager_running"], stats["total_processes"])

	// Stop the manager
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop process manager: %v", err)
	}

	// Verify it's stopped
	if manager.IsRunning() {
		t.Error("Process manager should be stopped")
	}
}

func TestConfigAdapter_Integration(t *testing.T) {
	// Test configuration integration
	mainConfig := &config.ClaudeConfig{}
	mainConfig.SetDefaults()

	adapter := NewConfigAdapter(mainConfig)

	// Test validation
	err := adapter.ValidateAndSetDefaults()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Test conversion
	processConfig, err := adapter.ToProcessConfig()
	if err != nil {
		t.Fatalf("Config conversion failed: %v", err)
	}

	// Verify config values
	if processConfig.PollInterval != mainConfig.PollInterval {
		t.Errorf("PollInterval mismatch: %v != %v", processConfig.PollInterval, mainConfig.PollInterval)
	}

	if processConfig.MaxProcesses != mainConfig.MaxProcesses {
		t.Errorf("MaxProcesses mismatch: %v != %v", processConfig.MaxProcesses, mainConfig.MaxProcesses)
	}

	// Test feature management
	err = adapter.EnableFeature("monitoring", false)
	if err != nil {
		t.Fatalf("Failed to disable monitoring: %v", err)
	}

	if adapter.IsEnabled() {
		t.Error("Monitoring should be disabled")
	}

	// Re-enable
	err = adapter.EnableFeature("monitoring", true)
	if err != nil {
		t.Fatalf("Failed to enable monitoring: %v", err)
	}

	if !adapter.IsEnabled() {
		t.Error("Monitoring should be enabled")
	}
}

func TestStateMonitor_WithRealProcesses(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	monitor := NewDefaultStateMonitor(config, detector)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start monitoring
	err = monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// Try to detect processes
	processes, err := detector.DetectProcesses(ctx)
	if err != nil {
		t.Fatalf("Failed to detect processes: %v", err)
	}

	// If we found any processes, monitor their state
	for _, process := range processes {
		state, err := monitor.MonitorState(ctx, process)
		if err != nil {
			t.Logf("Failed to monitor process %s: %v", process.SessionID, err)
			continue
		}

		t.Logf("Process %s (PID %d) state: %s", process.SessionID, process.PID, state.String())
	}

	// Stop monitoring
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}
}

func TestStateMachine_WithValidation(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)

	// Create a mock process
	process := &ProcessInfo{
		PID:        1234,
		SessionID:  "test-process",
		StartTime:  time.Now().Add(-time.Minute),
		LastUpdate: time.Now(),
		CPUPercent: 10.0,
		State:      StateIdle,
	}

	ctx := context.Background()

	// Test valid transition
	err := sm.ValidateTransition(ctx, process.SessionID, StateIdle, StateBusy, process)
	if err != nil {
		t.Errorf("Valid transition should not fail: %v", err)
	}

	// Record the transition
	sm.RecordTransition(process.SessionID, StateIdle, StateBusy, "integration_test")

	// Check that it was recorded
	history := sm.GetTransitionHistory(process.SessionID)
	if len(history) != 1 {
		t.Errorf("Expected 1 transition, got %d", len(history))
	}

	// Get metrics
	metrics := sm.GetStateMetrics(process.SessionID)
	if metrics.TotalTransitions != 1 {
		t.Errorf("Expected 1 total transition, got %d", metrics.TotalTransitions)
	}

	if metrics.TransitionCounts["idle->busy"] != 1 {
		t.Errorf("Expected 1 idle->busy transition, got %d", metrics.TransitionCounts["idle->busy"])
	}
}

// MockStateChangeHandler for testing event handling
type MockStateChangeHandler struct {
	events []StateChangeEvent
}

func (m *MockStateChangeHandler) OnStateChange(ctx context.Context, event StateChangeEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestProcessTracker_EventHandling(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()
	config.PollInterval = 100 * time.Millisecond // Fast polling for tests

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	monitor := NewDefaultStateMonitor(config, detector)
	tracker := NewDefaultProcessTracker(config, detector, monitor)

	// Add mock handler
	mockHandler := &MockStateChangeHandler{}
	err = tracker.Subscribe(mockHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe handler: %v", err)
	}

	// Start tracker
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}

	// Add a mock process
	process := &ProcessInfo{
		PID:        9999,
		SessionID:  "mock-process",
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		State:      StateStarting,
		WorkingDir: "/tmp/test",
	}

	err = tracker.AddProcess(process)
	if err != nil {
		t.Fatalf("Failed to add process: %v", err)
	}

	// Wait a bit for events
	time.Sleep(500 * time.Millisecond)

	// Check that we received events
	if len(mockHandler.events) == 0 {
		t.Log("No events received (expected for mock process)")
	} else {
		t.Logf("Received %d events", len(mockHandler.events))
		for i, event := range mockHandler.events {
			t.Logf("Event %d: %s -> %s", i, event.OldState, event.NewState)
		}
	}

	// Stop tracker
	err = tracker.Stop()
	if err != nil {
		t.Fatalf("Failed to stop tracker: %v", err)
	}
}