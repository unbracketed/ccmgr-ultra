package tmux

import (
	"os"
	"testing"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

func TestNewSessionManager(t *testing.T) {
	cfg := &config.Config{}
	sm := NewSessionManager(cfg)

	if sm == nil {
		t.Error("NewSessionManager returned nil")
	}

	if sm.config != cfg {
		t.Error("SessionManager config not set correctly")
	}

	if sm.tmux == nil {
		t.Error("SessionManager tmux not initialized")
	}
}

func TestNewTmuxCmd(t *testing.T) {
	tmux := NewTmuxCmd()

	if tmux == nil {
		t.Error("NewTmuxCmd returned nil")
	}

	if tmux.executable != "tmux" {
		t.Errorf("Expected executable to be 'tmux', got %s", tmux.executable)
	}
}

func TestCheckTmuxAvailable(t *testing.T) {
	err := CheckTmuxAvailable()

	if err != nil {
		t.Skipf("tmux not available for testing: %v", err)
	}
}

func TestSessionToPersistedSession(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:         "test-session",
		Name:       "test-session",
		Project:    "test-project",
		Worktree:   "test-worktree",
		Branch:     "test-branch",
		Directory:  "/test/dir",
		Created:    now,
		LastAccess: now,
		Active:     true,
	}

	persisted := session.toPersistedSession()

	if persisted.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, persisted.ID)
	}

	if persisted.Name != session.Name {
		t.Errorf("Expected Name %s, got %s", session.Name, persisted.Name)
	}

	if persisted.Project != session.Project {
		t.Errorf("Expected Project %s, got %s", session.Project, persisted.Project)
	}

	if persisted.Worktree != session.Worktree {
		t.Errorf("Expected Worktree %s, got %s", session.Worktree, persisted.Worktree)
	}

	if persisted.Branch != session.Branch {
		t.Errorf("Expected Branch %s, got %s", session.Branch, persisted.Branch)
	}

	if persisted.Directory != session.Directory {
		t.Errorf("Expected Directory %s, got %s", session.Directory, persisted.Directory)
	}

	if persisted.Created != session.Created {
		t.Errorf("Expected Created %v, got %v", session.Created, persisted.Created)
	}

	if persisted.LastAccess != session.LastAccess {
		t.Errorf("Expected LastAccess %v, got %v", session.LastAccess, persisted.LastAccess)
	}

	if persisted.LastState != StateUnknown {
		t.Errorf("Expected LastState %v, got %v", StateUnknown, persisted.LastState)
	}

	if persisted.Environment == nil {
		t.Error("Expected Environment to be initialized")
	}

	if persisted.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
}

func TestTmuxCmdMethods(t *testing.T) {
	tmux := NewTmuxCmd()

	t.Run("ListSessions with no server", func(t *testing.T) {
		sessions, err := tmux.ListSessions()
		if err != nil {
			t.Skipf("tmux server not running: %v", err)
		}

		if sessions == nil {
			t.Error("Expected sessions slice, got nil")
		}
	})

	t.Run("HasSession non-existent", func(t *testing.T) {
		exists, err := tmux.HasSession("non-existent-session-12345")
		if err != nil {
			t.Skipf("tmux not available: %v", err)
		}

		if exists {
			t.Error("Expected non-existent session to return false")
		}
	})
}

func TestSessionManagerWithoutTmux(t *testing.T) {
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", originalPath)

	cfg := &config.Config{}
	sm := NewSessionManager(cfg)

	_, err := sm.CreateSession("test", "main", "master", "/tmp")
	if err == nil {
		t.Error("Expected error when tmux not available")
	}

	_, err = sm.ListSessions()
	if err == nil {
		t.Error("Expected error when tmux not available")
	}

	_, err = sm.GetSession("test")
	if err == nil {
		t.Error("Expected error when tmux not available")
	}

	err = sm.AttachSession("test")
	if err == nil {
		t.Error("Expected error when tmux not available")
	}

	err = sm.DetachSession("test")
	if err == nil {
		t.Error("Expected error when tmux not available")
	}

	err = sm.KillSession("test")
	if err == nil {
		t.Error("Expected error when tmux not available")
	}

	_, err = sm.IsSessionActive("test")
	if err == nil {
		t.Error("Expected error when tmux not available")
	}
}
