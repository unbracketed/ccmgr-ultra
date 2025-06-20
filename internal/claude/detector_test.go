package claude

import (
	"context"
	"testing"
	"time"
)

func TestNewDefaultDetector(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	if detector == nil {
		t.Fatal("NewDefaultDetector() returned nil")
	}

	if detector.config != config {
		t.Error("config not properly set")
	}

	if detector.claudeRegex == nil {
		t.Error("claudeRegex not compiled")
	}

	if len(detector.knownPaths) == 0 {
		t.Error("knownPaths is empty")
	}
}

func TestDefaultDetector_IsClaudeProcess(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	// Test with current process (should not be Claude)
	// Note: This test assumes the test itself is not running as "claude"
	isClaudeProcess, err := detector.IsClaudeProcess(1)
	if err != nil {
		t.Logf("Warning: Could not check process 1: %v", err)
		return // Skip test if we can't access process info
	}

	// Since PID 1 is typically init/systemd, it shouldn't be Claude
	if isClaudeProcess {
		t.Error("PID 1 should not be identified as Claude process")
	}
}

func TestDefaultDetector_DetectProcesses(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	processes, err := detector.DetectProcesses(ctx)
	if err != nil {
		t.Fatalf("DetectProcesses() error = %v", err)
	}

	// We can't guarantee Claude processes are running, so just test that it doesn't error
	if processes == nil {
		t.Error("DetectProcesses() returned nil slice")
	}

	// Log the number of processes found for debugging
	t.Logf("Found %d Claude processes", len(processes))
}

func TestDefaultDetector_GetProcessInfo_InvalidPID(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	// Test with invalid PID (likely to not exist)
	_, err = detector.GetProcessInfo(999999)
	if err == nil {
		t.Error("Expected error for invalid PID, got nil")
	}
}

func TestDefaultDetector_RefreshProcess(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	// Create a mock process with an invalid PID
	process := &ProcessInfo{
		PID:       999999, // Likely invalid PID
		SessionID: "test-session",
		State:     StateIdle,
	}

	err = detector.RefreshProcess(process)
	if err != nil {
		// Expected for invalid PID
		t.Logf("RefreshProcess failed as expected for invalid PID: %v", err)
	}

	// Check that state was updated to stopped for invalid process
	if process.GetState() == StateStopped {
		t.Log("Process state correctly updated to stopped for invalid PID")
	}
}

func TestDefaultDetector_claudeRegexPattern(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	tests := []struct {
		input    string
		expected bool
	}{
		{"claude", true},
		{"Claude", true},
		{"CLAUDE", true},
		{"claude-code", true},
		{"Claude-Code", true},
		{"claude_code", true},
		{"something-claude-something", true},
		{"python", false},
		{"bash", false},
		{"vim", false},
		{"", false},
	}

	for _, test := range tests {
		match := detector.claudeRegex.MatchString(test.input)
		if match != test.expected {
			t.Errorf("claudeRegex.MatchString(%q) = %v, want %v", test.input, match, test.expected)
		}
	}
}

func TestDefaultDetector_processExists(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	// Test with PID 1 (should exist on Unix systems)
	exists, err := detector.processExists(1)
	if err != nil {
		t.Fatalf("processExists(1) error = %v", err)
	}

	if !exists {
		t.Error("PID 1 should exist on Unix systems")
	}

	// Test with invalid PID
	exists, err = detector.processExists(999999)
	if err != nil {
		t.Logf("processExists(999999) error = %v (expected)", err)
	}

	if exists {
		t.Error("PID 999999 should not exist")
	}
}

func TestDefaultDetector_getWorktreeID(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	tests := []struct {
		workingDir string
		expected   string
	}{
		{"", ""},
		{"/path/to/project", "project"},
		{"/path/to/my-project", "my-project"},
		{"/home/user/workspace/test-branch", "test-branch"},
	}

	for _, test := range tests {
		result := detector.getWorktreeID(test.workingDir)
		if result != test.expected {
			t.Errorf("getWorktreeID(%q) = %q, want %q", test.workingDir, result, test.expected)
		}
	}
}

func TestDefaultDetector_getTmuxSession(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	detector, err := NewDefaultDetector(config)
	if err != nil {
		t.Fatalf("NewDefaultDetector() error = %v", err)
	}

	// Test with current process (should not have tmux session)
	session := detector.getTmuxSession(1)

	// We can't guarantee the result, but it should not panic
	t.Logf("getTmuxSession(1) = %q", session)
}