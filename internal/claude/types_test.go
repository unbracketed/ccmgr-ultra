package claude

import (
	"testing"
	"time"
)

func TestProcessState_String(t *testing.T) {
	tests := []struct {
		state    ProcessState
		expected string
	}{
		{StateUnknown, "unknown"},
		{StateStarting, "starting"},
		{StateIdle, "idle"},
		{StateBusy, "busy"},
		{StateWaiting, "waiting"},
		{StateError, "error"},
		{StateStopped, "stopped"},
	}

	for _, test := range tests {
		if got := test.state.String(); got != test.expected {
			t.Errorf("ProcessState.String() = %v, want %v", got, test.expected)
		}
	}
}

func TestProcessInfo_GetState(t *testing.T) {
	process := &ProcessInfo{
		PID:   1234,
		State: StateBusy,
	}

	if got := process.GetState(); got != StateBusy {
		t.Errorf("GetState() = %v, want %v", got, StateBusy)
	}
}

func TestProcessInfo_SetState(t *testing.T) {
	process := &ProcessInfo{
		PID:   1234,
		State: StateIdle,
	}

	before := time.Now()
	process.SetState(StateBusy)
	after := time.Now()

	if got := process.GetState(); got != StateBusy {
		t.Errorf("After SetState(), GetState() = %v, want %v", got, StateBusy)
	}

	if process.LastUpdate.Before(before) || process.LastUpdate.After(after) {
		t.Errorf("LastUpdate not properly set: %v", process.LastUpdate)
	}
}

func TestProcessInfo_UpdateStats(t *testing.T) {
	process := &ProcessInfo{
		PID:        1234,
		CPUPercent: 0.0,
		MemoryMB:   0,
	}

	before := time.Now()
	process.UpdateStats(25.5, 128)
	after := time.Now()

	if process.CPUPercent != 25.5 {
		t.Errorf("CPUPercent = %v, want %v", process.CPUPercent, 25.5)
	}

	if process.MemoryMB != 128 {
		t.Errorf("MemoryMB = %v, want %v", process.MemoryMB, 128)
	}

	if process.LastUpdate.Before(before) || process.LastUpdate.After(after) {
		t.Errorf("LastUpdate not properly set: %v", process.LastUpdate)
	}
}

func TestProcessConfig_SetDefaults(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	if config.PollInterval != 3*time.Second {
		t.Errorf("PollInterval = %v, want %v", config.PollInterval, 3*time.Second)
	}

	if config.MaxProcesses != 10 {
		t.Errorf("MaxProcesses = %v, want %v", config.MaxProcesses, 10)
	}

	if config.CleanupInterval != 5*time.Minute {
		t.Errorf("CleanupInterval = %v, want %v", config.CleanupInterval, 5*time.Minute)
	}

	if len(config.LogPaths) != 2 {
		t.Errorf("LogPaths length = %v, want %v", len(config.LogPaths), 2)
	}

	if len(config.StatePatterns) != 4 {
		t.Errorf("StatePatterns length = %v, want %v", len(config.StatePatterns), 4)
	}

	if !config.EnableLogParsing {
		t.Error("EnableLogParsing should be true by default")
	}

	if !config.EnableResourceMonitoring {
		t.Error("EnableResourceMonitoring should be true by default")
	}
}

func TestProcessConfig_CompilePatterns(t *testing.T) {
	config := &ProcessConfig{
		StatePatterns: map[string]string{
			"busy":    `(?i)(Processing|Executing)`,
			"idle":    `(?i)(Waiting|Ready)`,
			"waiting": `(?i)(Confirm|Y/n)`,
			"error":   `(?i)(Error|Failed)`,
		},
	}

	err := config.CompilePatterns()
	if err != nil {
		t.Fatalf("CompilePatterns() error = %v", err)
	}

	// Test that patterns were compiled
	if config.GetCompiledPattern(StateBusy) == nil {
		t.Error("Busy pattern not compiled")
	}

	if config.GetCompiledPattern(StateIdle) == nil {
		t.Error("Idle pattern not compiled")
	}

	if config.GetCompiledPattern(StateWaiting) == nil {
		t.Error("Waiting pattern not compiled")
	}

	if config.GetCompiledPattern(StateError) == nil {
		t.Error("Error pattern not compiled")
	}
}

func TestProcessConfig_CompilePatterns_InvalidRegex(t *testing.T) {
	config := &ProcessConfig{
		StatePatterns: map[string]string{
			"busy": `[invalid regex`,
		},
	}

	err := config.CompilePatterns()
	if err == nil {
		t.Error("Expected error for invalid regex, got nil")
	}
}

func TestLogMonitor_GetSetLastOffset(t *testing.T) {
	monitor := &LogMonitor{
		LogPath:   "/test/path",
		ProcessID: "test-process",
	}

	// Test initial state
	if offset := monitor.GetLastOffset(); offset != 0 {
		t.Errorf("Initial offset = %v, want 0", offset)
	}

	// Test setting offset
	before := time.Now()
	monitor.SetLastOffset(1024)
	after := time.Now()

	if offset := monitor.GetLastOffset(); offset != 1024 {
		t.Errorf("After SetLastOffset(1024), GetLastOffset() = %v, want 1024", offset)
	}

	if monitor.LastCheck.Before(before) || monitor.LastCheck.After(after) {
		t.Errorf("LastCheck not properly set: %v", monitor.LastCheck)
	}
}

func TestNewProcessRegistry(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	registry := NewProcessRegistry(config)

	if registry == nil {
		t.Fatal("NewProcessRegistry() returned nil")
	}

	if registry.processes == nil {
		t.Error("processes map is nil")
	}

	if registry.subscribers == nil {
		t.Error("subscribers slice is nil")
	}

	if registry.config != config {
		t.Error("config not properly set")
	}

	if registry.stopCh == nil {
		t.Error("stopCh is nil")
	}
}

func TestProcessRegistry_GetStats(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	registry := NewProcessRegistry(config)

	// Test empty registry
	stats := registry.GetStats()
	if stats.TotalProcesses != 0 {
		t.Errorf("TotalProcesses = %v, want 0", stats.TotalProcesses)
	}

	if len(stats.StateDistribution) != 0 {
		t.Errorf("StateDistribution length = %v, want 0", len(stats.StateDistribution))
	}

	// Add a process
	process := &ProcessInfo{
		PID:       1234,
		SessionID: "test-session",
		State:     StateBusy,
		StartTime: time.Now().Add(-time.Hour),
	}

	registry.processes["test-session"] = process

	// Test with one process
	stats = registry.GetStats()
	if stats.TotalProcesses != 1 {
		t.Errorf("TotalProcesses = %v, want 1", stats.TotalProcesses)
	}

	if stats.StateDistribution[StateBusy] != 1 {
		t.Errorf("StateDistribution[StateBusy] = %v, want 1", stats.StateDistribution[StateBusy])
	}

	if stats.AverageUptime <= 0 {
		t.Errorf("AverageUptime = %v, should be > 0", stats.AverageUptime)
	}
}
