package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// CommandsConfigModel represents the commands configuration screen
type CommandsConfigModel struct {
	config       *config.CommandsConfig
	original     *config.CommandsConfig
	theme        Theme
	width        int
	height       int
	focusedIndex int
	components   []interface{}
}

func NewCommandsConfigModel(cfg *config.CommandsConfig, theme Theme) *CommandsConfigModel {
	original := &config.CommandsConfig{
		ClaudeCommand: cfg.ClaudeCommand,
		GitCommand:    cfg.GitCommand,
		TmuxPrefix:    cfg.TmuxPrefix,
		Environment:   make(map[string]string),
	}
	for k, v := range cfg.Environment {
		original.Environment[k] = v
	}

	m := &CommandsConfigModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}
	m.initComponents()
	return m
}

func (m *CommandsConfigModel) initComponents() {
	envList := make([]string, 0, len(m.config.Environment))
	for k, v := range m.config.Environment {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	m.components = []interface{}{
		NewConfigSection("Command Configuration", m.theme),
		NewConfigTextInput("Claude command path", m.config.ClaudeCommand, "claude", m.theme),
		NewConfigTextInput("Git command path", m.config.GitCommand, "git", m.theme),
		NewConfigTextInput("Tmux prefix", m.config.TmuxPrefix, "ccmgr", m.theme),
		NewConfigSection("Environment Variables", m.theme),
		NewConfigListInput("Environment variables", envList, m.theme),
	}
}

func (m *CommandsConfigModel) Init() tea.Cmd                 { return nil }
func (m *CommandsConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *CommandsConfigModel) View() string                 { return "Commands Configuration (placeholder)" }
func (m *CommandsConfigModel) Title() string                { return "Commands" }
func (m *CommandsConfigModel) Help() []string               { return []string{"Esc: Back"} }
func (m *CommandsConfigModel) HasUnsavedChanges() bool      { return false }
func (m *CommandsConfigModel) Save() error                  { return nil }
func (m *CommandsConfigModel) Cancel()                      {}
func (m *CommandsConfigModel) Reset()                       {}
func (m *CommandsConfigModel) GetConfig() interface{}       { return m.config }

// TUISettingsModel represents the TUI settings configuration screen
type TUISettingsModel struct {
	config       *config.TUIConfig
	original     *config.TUIConfig
	theme        Theme
	width        int
	height       int
	focusedIndex int
	components   []interface{}
}

func NewTUISettingsModel(cfg *config.TUIConfig, theme Theme) *TUISettingsModel {
	original := &config.TUIConfig{
		Theme:           cfg.Theme,
		RefreshInterval: cfg.RefreshInterval,
		MouseSupport:    cfg.MouseSupport,
		DefaultScreen:   cfg.DefaultScreen,
		ShowStatusBar:   cfg.ShowStatusBar,
		ShowKeyHelp:     cfg.ShowKeyHelp,
		ConfirmQuit:     cfg.ConfirmQuit,
		AutoRefresh:     cfg.AutoRefresh,
		DebugMode:       cfg.DebugMode,
	}

	m := &TUISettingsModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}
	m.initComponents()
	return m
}

func (m *TUISettingsModel) initComponents() {
	m.components = []interface{}{
		NewConfigSection("TUI Settings", m.theme),
		NewConfigTextInput("Theme", m.config.Theme, "default", m.theme),
		NewConfigNumberInput("Refresh interval (seconds)", m.config.RefreshInterval, 1, 60, 1, m.theme),
		NewConfigToggle("Mouse support", m.config.MouseSupport, m.theme),
		NewConfigTextInput("Default screen", m.config.DefaultScreen, "dashboard", m.theme),
		NewConfigToggle("Show status bar", m.config.ShowStatusBar, m.theme),
		NewConfigToggle("Show key help", m.config.ShowKeyHelp, m.theme),
		NewConfigToggle("Confirm quit", m.config.ConfirmQuit, m.theme),
		NewConfigToggle("Auto refresh", m.config.AutoRefresh, m.theme),
		NewConfigToggle("Debug mode", m.config.DebugMode, m.theme),
	}
}

func (m *TUISettingsModel) Init() tea.Cmd                 { return nil }
func (m *TUISettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *TUISettingsModel) View() string                 { return "TUI Settings (placeholder)" }
func (m *TUISettingsModel) Title() string                { return "TUI Settings" }
func (m *TUISettingsModel) Help() []string               { return []string{"Esc: Back"} }
func (m *TUISettingsModel) HasUnsavedChanges() bool      { return false }
func (m *TUISettingsModel) Save() error                  { return nil }
func (m *TUISettingsModel) Cancel()                      {}
func (m *TUISettingsModel) Reset()                       {}
func (m *TUISettingsModel) GetConfig() interface{}       { return m.config }

// GitSettingsModel represents the git settings configuration screen
type GitSettingsModel struct {
	config       *config.GitConfig
	original     *config.GitConfig
	theme        Theme
	width        int
	height       int
	focusedIndex int
	components   []interface{}
}

func NewGitSettingsModel(cfg *config.GitConfig, theme Theme) *GitSettingsModel {
	original := &config.GitConfig{
		AutoDirectory:       cfg.AutoDirectory,
		DirectoryPattern:    cfg.DirectoryPattern,
		MaxWorktrees:        cfg.MaxWorktrees,
		CleanupAge:          cfg.CleanupAge,
		DefaultBranch:       cfg.DefaultBranch,
		ProtectedBranches:   append([]string(nil), cfg.ProtectedBranches...),
		AllowForceDelete:    cfg.AllowForceDelete,
		DefaultRemote:       cfg.DefaultRemote,
		AutoPush:            cfg.AutoPush,
		CreatePR:            cfg.CreatePR,
		PRTemplate:          cfg.PRTemplate,
		GitHubToken:         cfg.GitHubToken,
		GitLabToken:         cfg.GitLabToken,
		BitbucketToken:      cfg.BitbucketToken,
		RequireCleanWorkdir: cfg.RequireCleanWorkdir,
		ConfirmDestructive:  cfg.ConfirmDestructive,
		BackupOnDelete:      cfg.BackupOnDelete,
	}

	m := &GitSettingsModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}
	m.initComponents()
	return m
}

func (m *GitSettingsModel) initComponents() {
	m.components = []interface{}{
		NewConfigSection("Git Configuration", m.theme),
		NewConfigToggle("Auto-create directories", m.config.AutoDirectory, m.theme),
		NewConfigTextInput("Directory pattern", m.config.DirectoryPattern, "{{.project}}-{{.branch}}", m.theme),
		NewConfigNumberInput("Max worktrees", m.config.MaxWorktrees, 1, 50, 1, m.theme),
		NewConfigTextInput("Default branch", m.config.DefaultBranch, "main", m.theme),
		NewConfigListInput("Protected branches", m.config.ProtectedBranches, m.theme),
		NewConfigToggle("Allow force delete", m.config.AllowForceDelete, m.theme),
		NewConfigTextInput("Default remote", m.config.DefaultRemote, "origin", m.theme),
		NewConfigToggle("Auto push", m.config.AutoPush, m.theme),
		NewConfigToggle("Create PR", m.config.CreatePR, m.theme),
		NewConfigToggle("Require clean workdir", m.config.RequireCleanWorkdir, m.theme),
		NewConfigToggle("Confirm destructive", m.config.ConfirmDestructive, m.theme),
		NewConfigToggle("Backup on delete", m.config.BackupOnDelete, m.theme),
	}
}

func (m *GitSettingsModel) Init() tea.Cmd                 { return nil }
func (m *GitSettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *GitSettingsModel) View() string                 { return "Git Settings (placeholder)" }
func (m *GitSettingsModel) Title() string                { return "Git Settings" }
func (m *GitSettingsModel) Help() []string               { return []string{"Esc: Back"} }
func (m *GitSettingsModel) HasUnsavedChanges() bool      { return false }
func (m *GitSettingsModel) Save() error                  { return nil }
func (m *GitSettingsModel) Cancel()                      {}
func (m *GitSettingsModel) Reset()                       {}
func (m *GitSettingsModel) GetConfig() interface{}       { return m.config }

// TmuxSettingsModel represents the tmux settings configuration screen
type TmuxSettingsModel struct {
	config       *config.TmuxConfig
	original     *config.TmuxConfig
	theme        Theme
	width        int
	height       int
	focusedIndex int
	components   []interface{}
}

func NewTmuxSettingsModel(cfg *config.TmuxConfig, theme Theme) *TmuxSettingsModel {
	original := &config.TmuxConfig{
		SessionPrefix:   cfg.SessionPrefix,
		NamingPattern:   cfg.NamingPattern,
		MaxSessionName:  cfg.MaxSessionName,
		MonitorInterval: cfg.MonitorInterval,
		StateFile:       cfg.StateFile,
		DefaultEnv:      make(map[string]string),
		AutoCleanup:     cfg.AutoCleanup,
		CleanupAge:      cfg.CleanupAge,
	}
	for k, v := range cfg.DefaultEnv {
		original.DefaultEnv[k] = v
	}

	m := &TmuxSettingsModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}
	m.initComponents()
	return m
}

func (m *TmuxSettingsModel) initComponents() {
	envList := make([]string, 0, len(m.config.DefaultEnv))
	for k, v := range m.config.DefaultEnv {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	m.components = []interface{}{
		NewConfigSection("Tmux Configuration", m.theme),
		NewConfigTextInput("Session prefix", m.config.SessionPrefix, "ccmgr", m.theme),
		NewConfigTextInput("Naming pattern", m.config.NamingPattern, "{{.prefix}}-{{.project}}-{{.branch}}", m.theme),
		NewConfigNumberInput("Max session name length", m.config.MaxSessionName, 10, 100, 5, m.theme),
		NewConfigTextInput("State file", m.config.StateFile, "~/.config/ccmgr-ultra/tmux-sessions.json", m.theme),
		NewConfigListInput("Default environment", envList, m.theme),
		NewConfigToggle("Auto cleanup", m.config.AutoCleanup, m.theme),
	}
}

func (m *TmuxSettingsModel) Init() tea.Cmd                 { return nil }
func (m *TmuxSettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *TmuxSettingsModel) View() string                 { return "Tmux Settings (placeholder)" }
func (m *TmuxSettingsModel) Title() string                { return "Tmux Settings" }
func (m *TmuxSettingsModel) Help() []string               { return []string{"Esc: Back"} }
func (m *TmuxSettingsModel) HasUnsavedChanges() bool      { return false }
func (m *TmuxSettingsModel) Save() error                  { return nil }
func (m *TmuxSettingsModel) Cancel()                      {}
func (m *TmuxSettingsModel) Reset()                       {}
func (m *TmuxSettingsModel) GetConfig() interface{}       { return m.config }

// ClaudeSettingsModel represents the Claude settings configuration screen
type ClaudeSettingsModel struct {
	config       *config.ClaudeConfig
	original     *config.ClaudeConfig
	theme        Theme
	width        int
	height       int
	focusedIndex int
	components   []interface{}
}

func NewClaudeSettingsModel(cfg *config.ClaudeConfig, theme Theme) *ClaudeSettingsModel {
	original := &config.ClaudeConfig{
		Enabled:                      cfg.Enabled,
		PollInterval:                 cfg.PollInterval,
		MaxProcesses:                 cfg.MaxProcesses,
		CleanupInterval:              cfg.CleanupInterval,
		StateTimeout:                 cfg.StateTimeout,
		StartupTimeout:               cfg.StartupTimeout,
		LogPaths:                     append([]string(nil), cfg.LogPaths...),
		StatePatterns:                make(map[string]string),
		EnableLogParsing:             cfg.EnableLogParsing,
		EnableResourceMonitoring:     cfg.EnableResourceMonitoring,
		IntegrateTmux:                cfg.IntegrateTmux,
		IntegrateWorktrees:           cfg.IntegrateWorktrees,
	}
	for k, v := range cfg.StatePatterns {
		original.StatePatterns[k] = v
	}

	m := &ClaudeSettingsModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}
	m.initComponents()
	return m
}

func (m *ClaudeSettingsModel) initComponents() {
	pollSeconds := int(m.config.PollInterval / time.Second)
	cleanupMinutes := int(m.config.CleanupInterval / time.Minute)

	m.components = []interface{}{
		NewConfigSection("Claude Process Monitoring", m.theme),
		NewConfigToggle("Enable monitoring", m.config.Enabled, m.theme),
		NewConfigNumberInput("Poll interval (seconds)", pollSeconds, 1, 60, 1, m.theme),
		NewConfigNumberInput("Max processes", m.config.MaxProcesses, 1, 100, 1, m.theme),
		NewConfigNumberInput("Cleanup interval (minutes)", cleanupMinutes, 1, 60, 1, m.theme),
		NewConfigListInput("Log paths", m.config.LogPaths, m.theme),
		NewConfigToggle("Enable log parsing", m.config.EnableLogParsing, m.theme),
		NewConfigToggle("Enable resource monitoring", m.config.EnableResourceMonitoring, m.theme),
		NewConfigToggle("Integrate with Tmux", m.config.IntegrateTmux, m.theme),
		NewConfigToggle("Integrate with worktrees", m.config.IntegrateWorktrees, m.theme),
	}
}

func (m *ClaudeSettingsModel) Init() tea.Cmd                 { return nil }
func (m *ClaudeSettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *ClaudeSettingsModel) View() string                 { return "Claude Settings (placeholder)" }
func (m *ClaudeSettingsModel) Title() string                { return "Claude Settings" }
func (m *ClaudeSettingsModel) Help() []string               { return []string{"Esc: Back"} }
func (m *ClaudeSettingsModel) HasUnsavedChanges() bool      { return false }
func (m *ClaudeSettingsModel) Save() error                  { return nil }
func (m *ClaudeSettingsModel) Cancel()                      {}
func (m *ClaudeSettingsModel) Reset()                       {}
func (m *ClaudeSettingsModel) GetConfig() interface{}       { return m.config }

// WorktreeHooksConfigModel represents the worktree hooks configuration screen
type WorktreeHooksConfigModel struct {
	config       *config.WorktreeHooksConfig
	original     *config.WorktreeHooksConfig
	theme        Theme
	width        int
	height       int
	focusedIndex int
	components   []interface{}
}

func NewWorktreeHooksConfigModel(cfg *config.WorktreeHooksConfig, theme Theme) *WorktreeHooksConfigModel {
	original := &config.WorktreeHooksConfig{
		Enabled:        cfg.Enabled,
		CreationHook:   cfg.CreationHook,
		ActivationHook: cfg.ActivationHook,
	}

	m := &WorktreeHooksConfigModel{
		config:   cfg,
		original: original,
		theme:    theme,
	}
	m.initComponents()
	return m
}

func (m *WorktreeHooksConfigModel) initComponents() {
	m.components = []interface{}{
		NewConfigSection("Worktree Hooks Configuration", m.theme),
		NewConfigToggle("Enable worktree hooks", m.config.Enabled, m.theme),
		NewConfigSection("Creation Hook", m.theme),
		NewConfigToggle("Enable creation hook", m.config.CreationHook.Enabled, m.theme),
		NewConfigTextInput("Creation script", m.config.CreationHook.Script, "~/hooks/worktree-create.sh", m.theme),
		NewConfigNumberInput("Creation timeout", m.config.CreationHook.Timeout, 1, 300, 5, m.theme),
		NewConfigToggle("Run creation hook async", m.config.CreationHook.Async, m.theme),
		NewConfigSection("Activation Hook", m.theme),
		NewConfigToggle("Enable activation hook", m.config.ActivationHook.Enabled, m.theme),
		NewConfigTextInput("Activation script", m.config.ActivationHook.Script, "~/hooks/worktree-activate.sh", m.theme),
		NewConfigNumberInput("Activation timeout", m.config.ActivationHook.Timeout, 1, 300, 5, m.theme),
		NewConfigToggle("Run activation hook async", m.config.ActivationHook.Async, m.theme),
	}
}

func (m *WorktreeHooksConfigModel) Init() tea.Cmd                 { return nil }
func (m *WorktreeHooksConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *WorktreeHooksConfigModel) View() string                 { return "Worktree Hooks (placeholder)" }
func (m *WorktreeHooksConfigModel) Title() string                { return "Worktree Hooks" }
func (m *WorktreeHooksConfigModel) Help() []string               { return []string{"Esc: Back"} }
func (m *WorktreeHooksConfigModel) HasUnsavedChanges() bool      { return false }
func (m *WorktreeHooksConfigModel) Save() error                  { return nil }
func (m *WorktreeHooksConfigModel) Cancel()                      {}
func (m *WorktreeHooksConfigModel) Reset()                       {}
func (m *WorktreeHooksConfigModel) GetConfig() interface{}       { return m.config }