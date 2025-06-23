package tmux

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/unbracketed/ccmgr-ultra/internal/config"
)

type MockTmux struct {
	sessions map[string]bool
	outputs  map[string]string
	panes    map[string][]string
	pids     map[string]int
	failOps  map[string]bool
}

func NewMockTmux() *MockTmux {
	return &MockTmux{
		sessions: make(map[string]bool),
		outputs:  make(map[string]string),
		panes:    make(map[string][]string),
		pids:     make(map[string]int),
		failOps:  make(map[string]bool),
	}
}

func (m *MockTmux) NewSession(name, dir string) error {
	if m.failOps["NewSession"] {
		return fmt.Errorf("mock error: new session failed")
	}

	if m.sessions[name] {
		return fmt.Errorf("session already exists")
	}

	m.sessions[name] = true
	m.panes[name] = []string{"0"}
	m.pids[name+":0"] = 1234
	m.outputs[name+":0"] = "claude> ready"

	return nil
}

func (m *MockTmux) ListSessions() ([]string, error) {
	if m.failOps["ListSessions"] {
		return nil, fmt.Errorf("mock error: list sessions failed")
	}

	var sessions []string
	for name := range m.sessions {
		sessions = append(sessions, name)
	}

	return sessions, nil
}

func (m *MockTmux) HasSession(name string) (bool, error) {
	if m.failOps["HasSession"] {
		return false, fmt.Errorf("mock error: has session failed")
	}

	return m.sessions[name], nil
}

func (m *MockTmux) AttachSession(name string) error {
	if m.failOps["AttachSession"] {
		return fmt.Errorf("mock error: attach failed")
	}

	if !m.sessions[name] {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (m *MockTmux) DetachSession(name string) error {
	if m.failOps["DetachSession"] {
		return fmt.Errorf("mock error: detach failed")
	}

	return nil
}

func (m *MockTmux) KillSession(name string) error {
	if m.failOps["KillSession"] {
		return fmt.Errorf("mock error: kill session failed")
	}

	if !m.sessions[name] {
		return fmt.Errorf("session not found")
	}

	delete(m.sessions, name)
	delete(m.panes, name)

	for key := range m.pids {
		if strings.HasPrefix(key, name+":") {
			delete(m.pids, key)
		}
	}

	for key := range m.outputs {
		if strings.HasPrefix(key, name+":") {
			delete(m.outputs, key)
		}
	}

	return nil
}

func (m *MockTmux) SendKeys(session, keys string) error {
	if m.failOps["SendKeys"] {
		return fmt.Errorf("mock error: send keys failed")
	}

	if !m.sessions[session] {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (m *MockTmux) GetSessionPanes(session string) ([]string, error) {
	if m.failOps["GetSessionPanes"] {
		return nil, fmt.Errorf("mock error: get panes failed")
	}

	if !m.sessions[session] {
		return nil, fmt.Errorf("session not found")
	}

	panes, exists := m.panes[session]
	if !exists {
		return []string{}, nil
	}

	return panes, nil
}

func (m *MockTmux) CapturePane(session, pane string) (string, error) {
	if m.failOps["CapturePane"] {
		return "", fmt.Errorf("mock error: capture pane failed")
	}

	key := session + ":" + pane
	output, exists := m.outputs[key]
	if !exists {
		return "no output", nil
	}

	return output, nil
}

func (m *MockTmux) GetPanePID(session, pane string) (int, error) {
	if m.failOps["GetPanePID"] {
		return 0, fmt.Errorf("mock error: get pane PID failed")
	}

	key := session + ":" + pane
	pid, exists := m.pids[key]
	if !exists {
		return 0, fmt.Errorf("pane not found")
	}

	return pid, nil
}

func (m *MockTmux) SetOutput(session, pane, output string) {
	key := session + ":" + pane
	m.outputs[key] = output
}

func (m *MockTmux) SetFailure(operation string, fail bool) {
	m.failOps[operation] = fail
}

func TestFullSessionLifecycle(t *testing.T) {
	mockTmux := NewMockTmux()

	cfg := &config.Config{}
	sm := NewSessionManager(cfg)
	sm.tmux = mockTmux

	t.Run("create session", func(t *testing.T) {
		session, err := sm.CreateSession("testproject", "main", "feature", "/tmp")
		if err != nil {
			t.Errorf("Failed to create session: %v", err)
		}

		if session == nil {
			t.Error("Expected session to be created")
		}

		if session.Project != "testproject" {
			t.Errorf("Expected project testproject, got %s", session.Project)
		}

		if session.Worktree != "main" {
			t.Errorf("Expected worktree main, got %s", session.Worktree)
		}

		if session.Branch != "feature" {
			t.Errorf("Expected branch feature, got %s", session.Branch)
		}
	})

	t.Run("list sessions", func(t *testing.T) {
		sessions, err := sm.ListSessions()
		if err != nil {
			t.Errorf("Failed to list sessions: %v", err)
		}

		if len(sessions) != 1 {
			t.Errorf("Expected 1 session, got %d", len(sessions))
		}
	})

	t.Run("get session", func(t *testing.T) {
		sessionName := GenerateSessionName("testproject", "main", "feature")
		session, err := sm.GetSession(sessionName)
		if err != nil {
			t.Errorf("Failed to get session: %v", err)
		}

		if session == nil {
			t.Error("Expected session to be returned")
		}

		if session.ID != sessionName {
			t.Errorf("Expected session ID %s, got %s", sessionName, session.ID)
		}
	})

	t.Run("check session active", func(t *testing.T) {
		sessionName := GenerateSessionName("testproject", "main", "feature")
		active, err := sm.IsSessionActive(sessionName)
		if err != nil {
			t.Errorf("Failed to check if session is active: %v", err)
		}

		if !active {
			t.Error("Expected session to be active")
		}
	})

	t.Run("attach session", func(t *testing.T) {
		sessionName := GenerateSessionName("testproject", "main", "feature")
		err := sm.AttachSession(sessionName)
		if err != nil {
			t.Errorf("Failed to attach session: %v", err)
		}
	})

	t.Run("detach session", func(t *testing.T) {
		sessionName := GenerateSessionName("testproject", "main", "feature")
		err := sm.DetachSession(sessionName)
		if err != nil {
			t.Errorf("Failed to detach session: %v", err)
		}
	})

	t.Run("kill session", func(t *testing.T) {
		sessionName := GenerateSessionName("testproject", "main", "feature")
		err := sm.KillSession(sessionName)
		if err != nil {
			t.Errorf("Failed to kill session: %v", err)
		}

		active, err := sm.IsSessionActive(sessionName)
		if err != nil {
			t.Errorf("Failed to check if session is active: %v", err)
		}

		if active {
			t.Error("Expected session to be inactive after kill")
		}
	})
}

func TestMultipleSessionsManagement(t *testing.T) {
	mockTmux := NewMockTmux()

	cfg := &config.Config{}
	sm := NewSessionManager(cfg)
	sm.tmux = mockTmux

	sessions := []struct {
		project  string
		worktree string
		branch   string
	}{
		{"proj1", "main", "feature1"},
		{"proj1", "main", "feature2"},
		{"proj2", "dev", "bugfix"},
		{"proj3", "staging", "release"},
	}

	t.Run("create multiple sessions", func(t *testing.T) {
		for _, s := range sessions {
			session, err := sm.CreateSession(s.project, s.worktree, s.branch, "/tmp")
			if err != nil {
				t.Errorf("Failed to create session for %s-%s-%s: %v", s.project, s.worktree, s.branch, err)
			}

			if session == nil {
				t.Errorf("Expected session to be created for %s-%s-%s", s.project, s.worktree, s.branch)
			}
		}
	})

	t.Run("list all sessions", func(t *testing.T) {
		allSessions, err := sm.ListSessions()
		if err != nil {
			t.Errorf("Failed to list sessions: %v", err)
		}

		if len(allSessions) != len(sessions) {
			t.Errorf("Expected %d sessions, got %d", len(sessions), len(allSessions))
		}
	})

	t.Run("verify each session exists", func(t *testing.T) {
		for _, s := range sessions {
			sessionName := GenerateSessionName(s.project, s.worktree, s.branch)
			active, err := sm.IsSessionActive(sessionName)
			if err != nil {
				t.Errorf("Failed to check session %s: %v", sessionName, err)
			}

			if !active {
				t.Errorf("Expected session %s to be active", sessionName)
			}
		}
	})

	t.Run("kill all sessions", func(t *testing.T) {
		for _, s := range sessions {
			sessionName := GenerateSessionName(s.project, s.worktree, s.branch)
			err := sm.KillSession(sessionName)
			if err != nil {
				t.Errorf("Failed to kill session %s: %v", sessionName, err)
			}
		}

		allSessions, err := sm.ListSessions()
		if err != nil {
			t.Errorf("Failed to list sessions: %v", err)
		}

		if len(allSessions) != 0 {
			t.Errorf("Expected 0 sessions after killing all, got %d", len(allSessions))
		}
	})
}

func TestProcessMonitoringIntegration(t *testing.T) {
	mockTmux := NewMockTmux()

	cfg := &config.Config{
		Tmux: config.TmuxConfig{
			MonitorInterval: 100 * time.Millisecond,
		},
	}

	pm := NewProcessMonitor(cfg)
	pm.tmux = mockTmux
	defer pm.Shutdown()

	sessionName := "ccmgr-test-main-feature"
	mockTmux.NewSession(sessionName, "/tmp")
	mockTmux.SetOutput(sessionName, "0", "claude> ready for input")

	hook := &mockStateHook{}
	pm.RegisterStateHook(hook)

	t.Run("start monitoring", func(t *testing.T) {
		err := pm.StartMonitoring(sessionName)
		if err != nil {
			t.Errorf("Failed to start monitoring: %v", err)
		}
	})

	t.Run("get process state", func(t *testing.T) {
		time.Sleep(200 * time.Millisecond)

		state, err := pm.GetProcessState(sessionName)
		if err != nil {
			t.Errorf("Failed to get process state: %v", err)
		}

		if state == StateUnknown {
			t.Error("Expected state to be detected")
		}
	})

	t.Run("detect state change", func(t *testing.T) {
		mockTmux.SetOutput(sessionName, "0", "Processing... please wait")

		time.Sleep(200 * time.Millisecond)

		changed, newState, err := pm.DetectStateChange(sessionName)
		if err != nil {
			t.Errorf("Failed to detect state change: %v", err)
		}

		if changed && newState == StateBusy {
			t.Log("State change detected successfully")
		}
	})

	t.Run("stop monitoring", func(t *testing.T) {
		err := pm.StopMonitoring(sessionName)
		if err != nil {
			t.Errorf("Failed to stop monitoring: %v", err)
		}
	})
}

func TestSessionRecoveryAfterRestart(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := tempDir + "/sessions.json"

	mockTmux := NewMockTmux()

	t.Run("create sessions and save state", func(t *testing.T) {
		cfg := &config.Config{}
		sm := NewSessionManager(cfg)
		sm.tmux = mockTmux

		state, err := LoadState(stateFile)
		if err != nil {
			t.Errorf("Failed to load state: %v", err)
		}
		sm.state = state

		session1, err := sm.CreateSession("proj1", "main", "feature", "/tmp")
		if err != nil {
			t.Errorf("Failed to create session1: %v", err)
		}

		session2, err := sm.CreateSession("proj2", "dev", "bugfix", "/tmp")
		if err != nil {
			t.Errorf("Failed to create session2: %v", err)
		}

		if session1 == nil || session2 == nil {
			t.Error("Expected sessions to be created")
		}
	})

	t.Run("simulate restart and recover state", func(t *testing.T) {
		state, err := LoadState(stateFile)
		if err != nil {
			t.Errorf("Failed to load state after restart: %v", err)
		}

		sessions := state.ListSessions()
		if len(sessions) != 2 {
			t.Errorf("Expected 2 sessions after restart, got %d", len(sessions))
		}

		found1 := false
		found2 := false

		for _, session := range sessions {
			if session.Project == "proj1" && session.Branch == "feature" {
				found1 = true
			}
			if session.Project == "proj2" && session.Branch == "bugfix" {
				found2 = true
			}
		}

		if !found1 || !found2 {
			t.Error("Expected both sessions to be recovered")
		}
	})

	t.Run("verify sessions still exist in tmux", func(t *testing.T) {
		sessions := []struct {
			project  string
			worktree string
			branch   string
		}{
			{"proj1", "main", "feature"},
			{"proj2", "dev", "bugfix"},
		}

		for _, s := range sessions {
			sessionName := GenerateSessionName(s.project, s.worktree, s.branch)
			exists, err := mockTmux.HasSession(sessionName)
			if err != nil {
				t.Errorf("Failed to check session %s: %v", sessionName, err)
			}

			if !exists {
				t.Errorf("Expected session %s to exist in tmux", sessionName)
			}
		}
	})
}

func TestErrorHandling(t *testing.T) {
	mockTmux := NewMockTmux()

	cfg := &config.Config{}
	sm := NewSessionManager(cfg)
	sm.tmux = mockTmux

	t.Run("create duplicate session", func(t *testing.T) {
		_, err := sm.CreateSession("test", "main", "feature", "/tmp")
		if err != nil {
			t.Errorf("First session creation should succeed: %v", err)
		}

		_, err = sm.CreateSession("test", "main", "feature", "/tmp")
		if err == nil {
			t.Error("Expected error when creating duplicate session")
		}
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, err := sm.GetSession("non-existent-session")
		if err == nil {
			t.Error("Expected error when getting non-existent session")
		}
	})

	t.Run("attach to non-existent session", func(t *testing.T) {
		err := sm.AttachSession("non-existent-session")
		if err == nil {
			t.Error("Expected error when attaching to non-existent session")
		}
	})

	t.Run("kill non-existent session", func(t *testing.T) {
		err := sm.KillSession("non-existent-session")
		if err == nil {
			t.Error("Expected error when killing non-existent session")
		}
	})

	t.Run("tmux operation failures", func(t *testing.T) {
		mockTmux.SetFailure("NewSession", true)

		_, err := sm.CreateSession("failtest", "main", "feature", "/tmp")
		if err == nil {
			t.Error("Expected error when tmux new session fails")
		}

		mockTmux.SetFailure("NewSession", false)
		mockTmux.SetFailure("ListSessions", true)

		_, err = sm.ListSessions()
		if err == nil {
			t.Error("Expected error when tmux list sessions fails")
		}
	})
}
