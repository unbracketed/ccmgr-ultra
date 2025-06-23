package tmux

import (
	"testing"
	"time"

	"github.com/unbracketed/ccmgr-ultra/internal/config"
)

func TestProcessStateString(t *testing.T) {
	tests := []struct {
		state    ProcessState
		expected string
	}{
		{StateUnknown, "unknown"},
		{StateIdle, "idle"},
		{StateBusy, "busy"},
		{StateWaiting, "waiting"},
		{StateError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNewProcessMonitor(t *testing.T) {
	cfg := &config.Config{
		Tmux: config.TmuxConfig{
			MonitorInterval: 5 * time.Second,
		},
	}

	pm := NewProcessMonitor(cfg)

	if pm == nil {
		t.Error("NewProcessMonitor returned nil")
	}

	if pm.sessions == nil {
		t.Error("ProcessMonitor sessions not initialized")
	}

	if pm.stateHooks == nil {
		t.Error("ProcessMonitor stateHooks not initialized")
	}

	if pm.pollInterval != 5*time.Second {
		t.Errorf("Expected poll interval 5s, got %v", pm.pollInterval)
	}

	if pm.tmux == nil {
		t.Error("ProcessMonitor tmux not initialized")
	}

	if pm.ctx == nil {
		t.Error("ProcessMonitor context not initialized")
	}

	pm.Shutdown()
}

func TestNewProcessMonitorWithNilConfig(t *testing.T) {
	pm := NewProcessMonitor(nil)

	if pm == nil {
		t.Error("NewProcessMonitor returned nil")
	}

	if pm.pollInterval != 2*time.Second {
		t.Errorf("Expected default poll interval 2s, got %v", pm.pollInterval)
	}

	pm.Shutdown()
}

func TestMonitoredSessionOperations(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	sessionID := "test-session"

	t.Run("GetProcessState before monitoring", func(t *testing.T) {
		_, err := pm.GetProcessState(sessionID)
		if err == nil {
			t.Error("Expected error when getting state of non-monitored session")
		}
	})

	t.Run("GetProcessPID before monitoring", func(t *testing.T) {
		_, err := pm.GetProcessPID(sessionID)
		if err == nil {
			t.Error("Expected error when getting PID of non-monitored session")
		}
	})

	t.Run("StopMonitoring before starting", func(t *testing.T) {
		err := pm.StopMonitoring(sessionID)
		if err == nil {
			t.Error("Expected error when stopping monitoring of non-monitored session")
		}
	})
}

type mockStateHook struct {
	calls []StateChangeCall
}

type StateChangeCall struct {
	sessionID string
	from      ProcessState
	to        ProcessState
}

func (m *mockStateHook) OnStateChange(sessionID string, from, to ProcessState) error {
	m.calls = append(m.calls, StateChangeCall{
		sessionID: sessionID,
		from:      from,
		to:        to,
	})
	return nil
}

func TestRegisterStateHook(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	hook := &mockStateHook{}
	pm.RegisterStateHook(hook)

	if len(pm.stateHooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(pm.stateHooks))
	}
}

func TestUpdateSessionState(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	sessionID := "test-session"

	session := &MonitoredSession{
		SessionID:       sessionID,
		ProcessPID:      1234,
		CurrentState:    StateUnknown,
		LastStateChange: time.Now(),
		StateHistory:    make([]StateChange, 0),
	}

	pm.sessions[sessionID] = session

	pm.updateSessionState(sessionID, StateIdle, "test")

	if session.CurrentState != StateIdle {
		t.Errorf("Expected state to be StateIdle, got %v", session.CurrentState)
	}

	if len(session.StateHistory) != 1 {
		t.Errorf("Expected 1 state change in history, got %d", len(session.StateHistory))
	}

	change := session.StateHistory[0]
	if change.From != StateUnknown {
		t.Errorf("Expected From to be StateUnknown, got %v", change.From)
	}

	if change.To != StateIdle {
		t.Errorf("Expected To to be StateIdle, got %v", change.To)
	}

	if change.Trigger != "test" {
		t.Errorf("Expected trigger to be 'test', got %s", change.Trigger)
	}
}

func TestAnalyzeOutput(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	tests := []struct {
		name     string
		output   string
		expected ProcessState
	}{
		{
			name:     "idle state",
			output:   "claude> ready for input",
			expected: StateIdle,
		},
		{
			name:     "busy state",
			output:   "Processing... please wait",
			expected: StateBusy,
		},
		{
			name:     "waiting state",
			output:   "Waiting for input from user",
			expected: StateWaiting,
		},
		{
			name:     "error state",
			output:   "Error: failed to process request",
			expected: StateError,
		},
		{
			name:     "exception state",
			output:   "Exception: something went wrong",
			expected: StateError,
		},
		{
			name:     "unknown state",
			output:   "some random output",
			expected: StateUnknown,
		},
		{
			name:     "multiple patterns - highest confidence wins",
			output:   "Processing... Error: failed",
			expected: StateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.analyzeOutput(tt.output)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDetectStateByProcess(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	t.Run("invalid PID", func(t *testing.T) {
		_, err := pm.detectStateByProcess(0)
		if err == nil {
			t.Error("Expected error for invalid PID")
		}

		_, err = pm.detectStateByProcess(-1)
		if err == nil {
			t.Error("Expected error for negative PID")
		}
	})

	t.Run("non-existent PID", func(t *testing.T) {
		_, err := pm.detectStateByProcess(999999)
		if err == nil {
			t.Error("Expected error for non-existent PID")
		}
	})
}

func TestStateChangeHistory(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	sessionID := "test-session"
	session := &MonitoredSession{
		SessionID:       sessionID,
		ProcessPID:      1234,
		CurrentState:    StateUnknown,
		LastStateChange: time.Now(),
		StateHistory:    make([]StateChange, 0),
	}

	pm.sessions[sessionID] = session

	for i := 0; i < 105; i++ {
		pm.updateSessionState(sessionID, StateIdle, "test")
		pm.updateSessionState(sessionID, StateBusy, "test")
	}

	if len(session.StateHistory) > 100 {
		t.Errorf("Expected history to be limited to 100 entries, got %d", len(session.StateHistory))
	}
}

func TestExecuteHooks(t *testing.T) {
	pm := NewProcessMonitor(nil)
	defer pm.Shutdown()

	hook1 := &mockStateHook{}
	hook2 := &mockStateHook{}

	pm.RegisterStateHook(hook1)
	pm.RegisterStateHook(hook2)

	sessionID := "test-session"
	pm.executeHooks(sessionID, StateIdle, StateBusy)

	if len(hook1.calls) != 1 {
		t.Errorf("Expected 1 call to hook1, got %d", len(hook1.calls))
	}

	if len(hook2.calls) != 1 {
		t.Errorf("Expected 1 call to hook2, got %d", len(hook2.calls))
	}

	call := hook1.calls[0]
	if call.sessionID != sessionID {
		t.Errorf("Expected sessionID %s, got %s", sessionID, call.sessionID)
	}

	if call.from != StateIdle {
		t.Errorf("Expected from StateIdle, got %v", call.from)
	}

	if call.to != StateBusy {
		t.Errorf("Expected to StateBusy, got %v", call.to)
	}
}
