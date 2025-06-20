package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/your-username/ccmgr-ultra/internal/config"
)

type TmuxInterface interface {
	NewSession(name, startDir string) error
	ListSessions() ([]string, error)
	HasSession(name string) (bool, error)
	AttachSession(name string) error
	DetachSession(name string) error
	KillSession(name string) error
	SendKeys(session, keys string) error
	GetSessionPanes(session string) ([]string, error)
	CapturePane(session, pane string) (string, error)
	GetPanePID(session, pane string) (int, error)
}

type SessionManager struct {
	config *config.Config
	state  *SessionState
	tmux   TmuxInterface
}

type Session struct {
	ID         string
	Name       string
	Project    string
	Worktree   string
	Branch     string
	Directory  string
	Created    time.Time
	LastAccess time.Time
	Active     bool
}

type TmuxCmd struct {
	executable string
}

func NewSessionManager(config *config.Config) *SessionManager {
	return &SessionManager{
		config: config,
		tmux:   NewTmuxCmd(),
	}
}

func NewTmuxCmd() *TmuxCmd {
	return &TmuxCmd{
		executable: "tmux",
	}
}

func (sm *SessionManager) CreateSession(project, worktree, branch, directory string) (*Session, error) {
	if err := CheckTmuxAvailable(); err != nil {
		return nil, fmt.Errorf("tmux not available: %w", err)
	}

	sessionName := GenerateSessionName(project, worktree, branch)
	
	exists, err := sm.tmux.HasSession(sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if session exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", sessionName)
	}

	if err := sm.tmux.NewSession(sessionName, directory); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	session := &Session{
		ID:         sessionName,
		Name:       sessionName,
		Project:    project,
		Worktree:   worktree,
		Branch:     branch,
		Directory:  directory,
		Created:    time.Now(),
		LastAccess: time.Now(),
		Active:     true,
	}

	if sm.state != nil {
		if err := sm.state.AddSession(session.toPersistedSession()); err != nil {
			return nil, fmt.Errorf("failed to persist session: %w", err)
		}
	}

	return session, nil
}

func (sm *SessionManager) ListSessions() ([]*Session, error) {
	if err := CheckTmuxAvailable(); err != nil {
		return nil, fmt.Errorf("tmux not available: %w", err)
	}

	tmuxSessions, err := sm.tmux.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	var sessions []*Session
	for _, sessionName := range tmuxSessions {
		if !strings.HasPrefix(sessionName, "ccmgr-") {
			continue
		}

		project, worktree, branch, err := ParseSessionName(sessionName)
		if err != nil {
			continue
		}

		session := &Session{
			ID:         sessionName,
			Name:       sessionName,
			Project:    project,
			Worktree:   worktree,
			Branch:     branch,
			Active:     true,
			LastAccess: time.Now(),
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	if err := CheckTmuxAvailable(); err != nil {
		return nil, fmt.Errorf("tmux not available: %w", err)
	}

	exists, err := sm.tmux.HasSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	project, worktree, branch, err := ParseSessionName(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse session name: %w", err)
	}

	return &Session{
		ID:         sessionID,
		Name:       sessionID,
		Project:    project,
		Worktree:   worktree,
		Branch:     branch,
		Active:     true,
		LastAccess: time.Now(),
	}, nil
}

func (sm *SessionManager) AttachSession(sessionID string) error {
	if err := CheckTmuxAvailable(); err != nil {
		return fmt.Errorf("tmux not available: %w", err)
	}

	exists, err := sm.tmux.HasSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to check session: %w", err)
	}
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return sm.tmux.AttachSession(sessionID)
}

func (sm *SessionManager) DetachSession(sessionID string) error {
	if err := CheckTmuxAvailable(); err != nil {
		return fmt.Errorf("tmux not available: %w", err)
	}

	return sm.tmux.DetachSession(sessionID)
}

func (sm *SessionManager) KillSession(sessionID string) error {
	if err := CheckTmuxAvailable(); err != nil {
		return fmt.Errorf("tmux not available: %w", err)
	}

	if err := sm.tmux.KillSession(sessionID); err != nil {
		return fmt.Errorf("failed to kill session: %w", err)
	}

	if sm.state != nil {
		if err := sm.state.RemoveSession(sessionID); err != nil {
			return fmt.Errorf("failed to remove session from state: %w", err)
		}
	}

	return nil
}

func (sm *SessionManager) IsSessionActive(sessionID string) (bool, error) {
	if err := CheckTmuxAvailable(); err != nil {
		return false, fmt.Errorf("tmux not available: %w", err)
	}

	return sm.tmux.HasSession(sessionID)
}

func (t *TmuxCmd) NewSession(name, startDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "new-session", "-d", "-s", name, "-c", startDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}
	return nil
}

func (t *TmuxCmd) ListSessions() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}

	return lines, nil
}

func (t *TmuxCmd) HasSession(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "has-session", "-t", name)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check tmux session: %w", err)
	}
	return true, nil
}

func (t *TmuxCmd) AttachSession(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "attach-session", "-t", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to attach to tmux session: %w", err)
	}
	return nil
}

func (t *TmuxCmd) DetachSession(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "detach-session", "-t", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to detach from tmux session: %w", err)
	}
	return nil
}

func (t *TmuxCmd) KillSession(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "kill-session", "-t", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}
	return nil
}

func (t *TmuxCmd) SendKeys(session, keys string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "send-keys", "-t", session, keys, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send keys to tmux session: %w", err)
	}
	return nil
}

func (t *TmuxCmd) GetSessionPanes(session string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "list-panes", "-t", session, "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}

	return lines, nil
}

func (t *TmuxCmd) CapturePane(session, pane string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.executable, "capture-pane", "-t", session+":"+pane, "-p")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}

	return string(output), nil
}

func (t *TmuxCmd) GetPanePID(session, pane string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	target := session
	if pane != "" {
		target = session + ":" + pane
	}

	cmd := exec.CommandContext(ctx, t.executable, "display-message", "-t", target, "-p", "#{pane_pid}")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get pane PID: %w", err)
	}

	pidStr := strings.TrimSpace(string(output))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID format: %s", pidStr)
	}

	return pid, nil
}

func CheckTmuxAvailable() error {
	if _, err := exec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}
	return nil
}

func (s *Session) toPersistedSession() *PersistedSession {
	return &PersistedSession{
		ID:         s.ID,
		Name:       s.Name,
		Project:    s.Project,
		Worktree:   s.Worktree,
		Branch:     s.Branch,
		Directory:  s.Directory,
		Created:    s.Created,
		LastAccess: s.LastAccess,
		LastState:  StateUnknown,
		Environment: make(map[string]string),
		Metadata:   make(map[string]interface{}),
	}
}