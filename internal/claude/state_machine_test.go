package claude

import (
	"context"
	"testing"
	"time"
)

func TestNewStateMachine(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)

	if sm == nil {
		t.Fatal("NewStateMachine() returned nil")
	}

	if len(sm.rules) == 0 {
		t.Error("StateMachine should have default rules")
	}

	if sm.transitions == nil {
		t.Error("transitions map should be initialized")
	}

	if len(sm.validators) == 0 {
		t.Error("StateMachine should have default validators")
	}

	if sm.config != config {
		t.Error("config not properly set")
	}
}

func TestStateMachine_ValidateTransition(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	process := &ProcessInfo{
		PID:        1234,
		SessionID:  "test-process",
		StartTime:  time.Now().Add(-time.Minute),
		LastUpdate: time.Now(),
		CPUPercent: 5.0,
	}

	ctx := context.Background()

	tests := []struct {
		name        string
		from        ProcessState
		to          ProcessState
		shouldError bool
	}{
		{"idle to busy", StateIdle, StateBusy, false},
		{"busy to idle", StateBusy, StateIdle, false},
		{"idle to waiting", StateIdle, StateWaiting, false},
		{"starting to idle", StateStarting, StateIdle, false},
		{"any to error", StateIdle, StateError, false},
		{"any to stopped", StateIdle, StateStopped, false},
		{"stopped to busy", StateStopped, StateBusy, true}, // Should be invalid
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := sm.ValidateTransition(ctx, process.SessionID, test.from, test.to, process)

			if test.shouldError && err == nil {
				t.Errorf("Expected error for transition %s -> %s, got nil", test.from, test.to)
			}

			if !test.shouldError && err != nil {
				t.Errorf("Unexpected error for transition %s -> %s: %v", test.from, test.to, err)
			}
		})
	}
}

func TestStateMachine_RecordTransition(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	processID := "test-process"

	// Record a transition
	sm.RecordTransition(processID, StateIdle, StateBusy, "cpu_increase")

	// Check that it was recorded
	history := sm.GetTransitionHistory(processID)
	if len(history) != 1 {
		t.Errorf("Expected 1 transition, got %d", len(history))
	}

	transition := history[0]
	if transition.From != StateIdle {
		t.Errorf("From = %v, want %v", transition.From, StateIdle)
	}

	if transition.To != StateBusy {
		t.Errorf("To = %v, want %v", transition.To, StateBusy)
	}

	if transition.Trigger != "cpu_increase" {
		t.Errorf("Trigger = %v, want %v", transition.Trigger, "cpu_increase")
	}

	if transition.ProcessID != processID {
		t.Errorf("ProcessID = %v, want %v", transition.ProcessID, processID)
	}
}

func TestStateMachine_GetTransitionHistory(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	processID := "test-process"

	// Test empty history
	history := sm.GetTransitionHistory(processID)
	if history != nil {
		t.Errorf("Expected nil for non-existent process, got %v", history)
	}

	// Record multiple transitions
	sm.RecordTransition(processID, StateStarting, StateIdle, "startup_complete")
	sm.RecordTransition(processID, StateIdle, StateBusy, "work_started")
	sm.RecordTransition(processID, StateBusy, StateIdle, "work_complete")

	history = sm.GetTransitionHistory(processID)
	if len(history) != 3 {
		t.Errorf("Expected 3 transitions, got %d", len(history))
	}

	// Check order (should be chronological)
	if history[0].From != StateStarting || history[0].To != StateIdle {
		t.Error("First transition should be Starting -> Idle")
	}

	if history[1].From != StateIdle || history[1].To != StateBusy {
		t.Error("Second transition should be Idle -> Busy")
	}

	if history[2].From != StateBusy || history[2].To != StateIdle {
		t.Error("Third transition should be Busy -> Idle")
	}
}

func TestStateMachine_GetLastTransition(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	processID := "test-process"

	// Test no transitions
	last := sm.GetLastTransition(processID)
	if last != nil {
		t.Errorf("Expected nil for no transitions, got %v", last)
	}

	// Record transitions
	sm.RecordTransition(processID, StateIdle, StateBusy, "first")
	sm.RecordTransition(processID, StateBusy, StateIdle, "second")

	last = sm.GetLastTransition(processID)
	if last == nil {
		t.Fatal("Expected transition, got nil")
	}

	if last.Trigger != "second" {
		t.Errorf("Last transition trigger = %v, want %v", last.Trigger, "second")
	}
}

func TestStateMachine_GetRecentTransitions(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	processID := "test-process"

	now := time.Now()

	// Record transitions with specific timestamps (simulate by adding to history directly)
	sm.transitions[processID] = []StateTransition{
		{From: StateIdle, To: StateBusy, Timestamp: now.Add(-time.Hour), Trigger: "old"},
		{From: StateBusy, To: StateIdle, Timestamp: now.Add(-time.Minute), Trigger: "recent"},
	}

	// Get recent transitions (within 30 minutes)
	recent := sm.GetRecentTransitions(processID, now.Add(-30*time.Minute))

	if len(recent) != 1 {
		t.Errorf("Expected 1 recent transition, got %d", len(recent))
	}

	if recent[0].Trigger != "recent" {
		t.Errorf("Recent transition trigger = %v, want %v", recent[0].Trigger, "recent")
	}
}

func TestStateMachine_GetStateMetrics(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	processID := "test-process"

	// Test empty metrics
	metrics := sm.GetStateMetrics(processID)
	if metrics.TotalTransitions != 0 {
		t.Errorf("TotalTransitions = %v, want 0", metrics.TotalTransitions)
	}

	// Add some transitions
	now := time.Now()
	sm.transitions[processID] = []StateTransition{
		{From: StateStarting, To: StateIdle, Timestamp: now.Add(-time.Hour), Trigger: "init"},
		{From: StateIdle, To: StateBusy, Timestamp: now.Add(-30 * time.Minute), Trigger: "work"},
		{From: StateBusy, To: StateIdle, Timestamp: now, Trigger: "done"},
	}

	metrics = sm.GetStateMetrics(processID)

	if metrics.TotalTransitions != 3 {
		t.Errorf("TotalTransitions = %v, want 3", metrics.TotalTransitions)
	}

	if metrics.StateDistribution[StateIdle] != 2 {
		t.Errorf("StateDistribution[StateIdle] = %v, want 2", metrics.StateDistribution[StateIdle])
	}

	if metrics.StateDistribution[StateBusy] != 1 {
		t.Errorf("StateDistribution[StateBusy] = %v, want 1", metrics.StateDistribution[StateBusy])
	}

	if metrics.TransitionCounts["starting->idle"] != 1 {
		t.Errorf("TransitionCounts[starting->idle] = %v, want 1", metrics.TransitionCounts["starting->idle"])
	}
}

func TestStateMachine_CleanupOldTransitions(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)
	processID := "test-process"

	now := time.Now()

	// Add old and new transitions
	sm.transitions[processID] = []StateTransition{
		{From: StateIdle, To: StateBusy, Timestamp: now.Add(-2 * time.Hour), Trigger: "old"},
		{From: StateBusy, To: StateIdle, Timestamp: now.Add(-time.Minute), Trigger: "recent"},
	}

	// Cleanup transitions older than 1 hour
	sm.CleanupOldTransitions(time.Hour)

	history := sm.GetTransitionHistory(processID)
	if len(history) != 1 {
		t.Errorf("Expected 1 transition after cleanup, got %d", len(history))
	}

	if history[0].Trigger != "recent" {
		t.Errorf("Remaining transition trigger = %v, want %v", history[0].Trigger, "recent")
	}
}

func TestDefaultStateValidator_ValidateTransition(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	validator := &DefaultStateValidator{config: config}
	ctx := context.Background()

	tests := []struct {
		name        string
		from        ProcessState
		to          ProcessState
		process     *ProcessInfo
		shouldError bool
	}{
		{
			name: "valid busy transition",
			from: StateIdle,
			to:   StateBusy,
			process: &ProcessInfo{
				StartTime:  time.Now().Add(-time.Minute),
				LastUpdate: time.Now(),
				CPUPercent: 10.0,
			},
			shouldError: false,
		},
		{
			name: "invalid busy transition - low CPU",
			from: StateIdle,
			to:   StateBusy,
			process: &ProcessInfo{
				StartTime:  time.Now().Add(-time.Minute),
				LastUpdate: time.Now(),
				CPUPercent: 0.5,
			},
			shouldError: true,
		},
		{
			name: "stale process",
			from: StateIdle,
			to:   StateBusy,
			process: &ProcessInfo{
				StartTime:  time.Now().Add(-time.Hour),
				LastUpdate: time.Now().Add(-time.Hour),
				CPUPercent: 10.0,
			},
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateTransition(ctx, test.from, test.to, test.process)

			if test.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !test.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestStateMachine_isTransitionAllowed(t *testing.T) {
	config := &ProcessConfig{}
	config.SetDefaults()

	sm := NewStateMachine(config)

	tests := []struct {
		from     ProcessState
		to       ProcessState
		expected bool
	}{
		{StateIdle, StateBusy, true},
		{StateBusy, StateIdle, true},
		{StateIdle, StateWaiting, true},
		{StateError, StateIdle, true},
		{StateStopped, StateBusy, false}, // Not allowed
		{StateIdle, StateIdle, true},     // Same state always allowed
	}

	for _, test := range tests {
		result := sm.isTransitionAllowed(test.from, test.to)
		if result != test.expected {
			t.Errorf("isTransitionAllowed(%s, %s) = %v, want %v",
				test.from, test.to, result, test.expected)
		}
	}
}
