package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Config represents the main configuration structure
type Config struct {
	Version      string                 `yaml:"version" json:"version"`
	StatusHooks  StatusHooksConfig      `yaml:"status_hooks" json:"status_hooks"`
	Worktree     WorktreeConfig         `yaml:"worktree" json:"worktree"`
	Tmux         TmuxConfig             `yaml:"tmux" json:"tmux"`
	Git          GitConfig              `yaml:"git" json:"git"`
	Claude       ClaudeConfig           `yaml:"claude" json:"claude"`
	Shortcuts    map[string]string      `yaml:"shortcuts" json:"shortcuts"`
	Commands     CommandsConfig         `yaml:"commands" json:"commands"`
	LastModified time.Time              `yaml:"last_modified" json:"last_modified"`
}

// StatusHooksConfig defines status hook configuration
type StatusHooksConfig struct {
	Enabled      bool              `yaml:"enabled" json:"enabled"`
	IdleHook     HookConfig        `yaml:"idle" json:"idle"`
	BusyHook     HookConfig        `yaml:"busy" json:"busy"`
	WaitingHook  HookConfig        `yaml:"waiting" json:"waiting"`
}

// HookConfig defines individual hook configuration
type HookConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Script   string `yaml:"script" json:"script"`
	Timeout  int    `yaml:"timeout" json:"timeout"` // seconds
	Async    bool   `yaml:"async" json:"async"`
}

// WorktreeConfig defines worktree configuration
type WorktreeConfig struct {
	AutoDirectory    bool   `yaml:"auto_directory" json:"auto_directory"`
	DirectoryPattern string `yaml:"directory_pattern" json:"directory_pattern"` // e.g., "{{.project}}-{{.branch}}"
	DefaultBranch    string `yaml:"default_branch" json:"default_branch"`
	CleanupOnMerge   bool   `yaml:"cleanup_on_merge" json:"cleanup_on_merge"`
}

// CommandsConfig defines command configuration
type CommandsConfig struct {
	ClaudeCommand   string            `yaml:"claude_command" json:"claude_command"`
	GitCommand      string            `yaml:"git_command" json:"git_command"`
	TmuxPrefix      string            `yaml:"tmux_prefix" json:"tmux_prefix"`
	Environment     map[string]string `yaml:"environment" json:"environment"`
}

// TmuxConfig defines tmux integration configuration
type TmuxConfig struct {
	SessionPrefix    string            `yaml:"session_prefix" json:"session_prefix"`
	NamingPattern    string            `yaml:"naming_pattern" json:"naming_pattern"`
	MaxSessionName   int               `yaml:"max_session_name" json:"max_session_name"`
	MonitorInterval  time.Duration     `yaml:"monitor_interval" json:"monitor_interval"`
	StateFile        string            `yaml:"state_file" json:"state_file"`
	DefaultEnv       map[string]string `yaml:"default_env" json:"default_env"`
	AutoCleanup      bool              `yaml:"auto_cleanup" json:"auto_cleanup"`
	CleanupAge       time.Duration     `yaml:"cleanup_age" json:"cleanup_age"`
}

// ClaudeConfig defines Claude Code process monitoring configuration
type ClaudeConfig struct {
	// Monitoring settings
	Enabled              bool              `yaml:"enabled" json:"enabled" default:"true"`
	PollInterval         time.Duration     `yaml:"poll_interval" json:"poll_interval" default:"3s"`
	MaxProcesses         int               `yaml:"max_processes" json:"max_processes" default:"10"`
	CleanupInterval      time.Duration     `yaml:"cleanup_interval" json:"cleanup_interval" default:"5m"`
	StateTimeout         time.Duration     `yaml:"state_timeout" json:"state_timeout" default:"30s"`
	StartupTimeout       time.Duration     `yaml:"startup_timeout" json:"startup_timeout" default:"10s"`
	
	// Detection settings
	LogPaths             []string          `yaml:"log_paths" json:"log_paths"`
	StatePatterns        map[string]string `yaml:"state_patterns" json:"state_patterns"`
	EnableLogParsing     bool              `yaml:"enable_log_parsing" json:"enable_log_parsing" default:"true"`
	EnableResourceMonitoring bool          `yaml:"enable_resource_monitoring" json:"enable_resource_monitoring" default:"true"`
	
	// Integration settings
	IntegrateTmux        bool              `yaml:"integrate_tmux" json:"integrate_tmux" default:"true"`
	IntegrateWorktrees   bool              `yaml:"integrate_worktrees" json:"integrate_worktrees" default:"true"`
}

// GitConfig defines git worktree and operations configuration
type GitConfig struct {
	// Worktree settings
	AutoDirectory    bool          `yaml:"auto_directory" json:"auto_directory" default:"true"`
	DirectoryPattern string        `yaml:"directory_pattern" json:"directory_pattern" default:"{{.project}}-{{.branch}}"`
	MaxWorktrees     int           `yaml:"max_worktrees" json:"max_worktrees" default:"10"`
	CleanupAge       time.Duration `yaml:"cleanup_age" json:"cleanup_age" default:"168h"`

	// Branch settings
	DefaultBranch     string   `yaml:"default_branch" json:"default_branch" default:"main"`
	ProtectedBranches []string `yaml:"protected_branches" json:"protected_branches"`
	AllowForceDelete  bool     `yaml:"allow_force_delete" json:"allow_force_delete" default:"false"`

	// Remote settings
	DefaultRemote string `yaml:"default_remote" json:"default_remote" default:"origin"`
	AutoPush      bool   `yaml:"auto_push" json:"auto_push" default:"true"`
	CreatePR      bool   `yaml:"create_pr" json:"create_pr" default:"false"`
	PRTemplate    string `yaml:"pr_template" json:"pr_template"`

	// Authentication
	GitHubToken    string `yaml:"github_token" json:"github_token" env:"GITHUB_TOKEN"`
	GitLabToken    string `yaml:"gitlab_token" json:"gitlab_token" env:"GITLAB_TOKEN"`
	BitbucketToken string `yaml:"bitbucket_token" json:"bitbucket_token" env:"BITBUCKET_TOKEN"`

	// Safety settings
	RequireCleanWorkdir bool `yaml:"require_clean_workdir" json:"require_clean_workdir" default:"true"`
	ConfirmDestructive  bool `yaml:"confirm_destructive" json:"confirm_destructive" default:"true"`
	BackupOnDelete      bool `yaml:"backup_on_delete" json:"backup_on_delete" default:"true"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return errors.New("config version is required")
	}

	if err := c.StatusHooks.Validate(); err != nil {
		return fmt.Errorf("status hooks validation failed: %w", err)
	}

	if err := c.Worktree.Validate(); err != nil {
		return fmt.Errorf("worktree validation failed: %w", err)
	}

	if err := c.Commands.Validate(); err != nil {
		return fmt.Errorf("commands validation failed: %w", err)
	}

	if err := c.Tmux.Validate(); err != nil {
		return fmt.Errorf("tmux validation failed: %w", err)
	}

	if err := c.Git.Validate(); err != nil {
		return fmt.Errorf("git validation failed: %w", err)
	}

	if err := c.Claude.Validate(); err != nil {
		return fmt.Errorf("claude validation failed: %w", err)
	}

	// Validate shortcuts
	for key, action := range c.Shortcuts {
		if key == "" {
			return errors.New("shortcut key cannot be empty")
		}
		if action == "" {
			return fmt.Errorf("shortcut action for key '%s' cannot be empty", key)
		}
	}

	return nil
}

// Validate validates status hooks configuration
func (s *StatusHooksConfig) Validate() error {
	if err := s.IdleHook.Validate(); err != nil {
		return fmt.Errorf("idle hook validation failed: %w", err)
	}

	if err := s.BusyHook.Validate(); err != nil {
		return fmt.Errorf("busy hook validation failed: %w", err)
	}

	if err := s.WaitingHook.Validate(); err != nil {
		return fmt.Errorf("waiting hook validation failed: %w", err)
	}

	return nil
}

// Validate validates individual hook configuration
func (h *HookConfig) Validate() error {
	if h.Enabled && h.Script == "" {
		return errors.New("hook script path is required when hook is enabled")
	}

	if h.Timeout < 0 {
		return errors.New("hook timeout cannot be negative")
	}

	if h.Timeout == 0 && h.Enabled {
		h.Timeout = 30 // default timeout
	}

	if h.Timeout > 300 { // 5 minutes max
		return errors.New("hook timeout cannot exceed 300 seconds")
	}

	return nil
}

// Validate validates worktree configuration
func (w *WorktreeConfig) Validate() error {
	if w.DefaultBranch == "" {
		return errors.New("default branch is required")
	}

	if w.AutoDirectory && w.DirectoryPattern == "" {
		return errors.New("directory pattern is required when auto directory is enabled")
	}

	// Validate directory pattern contains valid placeholders
	if w.DirectoryPattern != "" {
		if !strings.Contains(w.DirectoryPattern, "{{") || !strings.Contains(w.DirectoryPattern, "}}") {
			return errors.New("directory pattern must contain template variables like {{.project}} or {{.branch}}")
		}
	}

	return nil
}

// Validate validates commands configuration
func (c *CommandsConfig) Validate() error {
	if c.ClaudeCommand == "" {
		return errors.New("claude command is required")
	}

	if c.GitCommand == "" {
		return errors.New("git command is required")
	}

	if c.TmuxPrefix == "" {
		return errors.New("tmux prefix is required")
	}

	// Validate environment variables
	for key, value := range c.Environment {
		if key == "" {
			return errors.New("environment variable key cannot be empty")
		}
		if strings.Contains(key, "=") {
			return fmt.Errorf("environment variable key '%s' cannot contain '='", key)
		}
		if value == "" {
			return fmt.Errorf("environment variable '%s' cannot have empty value", key)
		}
	}

	return nil
}

// Validate validates git configuration
func (g *GitConfig) Validate() error {
	if g.DefaultBranch == "" {
		return errors.New("default branch is required")
	}

	if g.AutoDirectory && g.DirectoryPattern == "" {
		return errors.New("directory pattern is required when auto directory is enabled")
	}

	// Validate directory pattern contains valid placeholders
	if g.DirectoryPattern != "" {
		if !strings.Contains(g.DirectoryPattern, "{{") || !strings.Contains(g.DirectoryPattern, "}}") {
			return errors.New("directory pattern must contain template variables like {{.project}} or {{.branch}}")
		}
	}

	if g.MaxWorktrees < 0 {
		return errors.New("max worktrees cannot be negative")
	}

	if g.CleanupAge < 0 {
		return errors.New("cleanup age cannot be negative")
	}

	// Validate protected branches
	for _, branch := range g.ProtectedBranches {
		if branch == "" {
			return errors.New("protected branch name cannot be empty")
		}
	}

	if g.DefaultRemote == "" {
		return errors.New("default remote is required")
	}

	return nil
}

// Validate validates tmux configuration
func (t *TmuxConfig) Validate() error {
	if t.SessionPrefix == "" {
		return errors.New("tmux session prefix is required")
	}

	if t.MaxSessionName < 0 {
		return errors.New("max session name length cannot be negative")
	}

	if t.MonitorInterval < 0 {
		return errors.New("monitor interval cannot be negative")
	}

	if t.CleanupAge < 0 {
		return errors.New("cleanup age cannot be negative")
	}

	return nil
}

// Validate validates Claude configuration
func (c *ClaudeConfig) Validate() error {
	if c.PollInterval < 0 {
		return errors.New("poll interval cannot be negative")
	}
	
	if c.PollInterval < time.Second {
		return errors.New("poll interval must be at least 1 second")
	}
	
	if c.MaxProcesses < 0 {
		return errors.New("max processes cannot be negative")
	}
	
	if c.MaxProcesses > 100 {
		return errors.New("max processes cannot exceed 100")
	}
	
	if c.CleanupInterval < 0 {
		return errors.New("cleanup interval cannot be negative")
	}
	
	if c.StateTimeout < 0 {
		return errors.New("state timeout cannot be negative")
	}
	
	if c.StartupTimeout < 0 {
		return errors.New("startup timeout cannot be negative")
	}
	
	// Validate log paths
	for _, path := range c.LogPaths {
		if path == "" {
			return errors.New("log path cannot be empty")
		}
	}
	
	// Validate state patterns
	for key, pattern := range c.StatePatterns {
		if key == "" {
			return errors.New("state pattern key cannot be empty")
		}
		if pattern == "" {
			return fmt.Errorf("state pattern for '%s' cannot be empty", key)
		}
	}
	
	return nil
}

// SetDefaults sets default values for missing configuration
func (c *Config) SetDefaults() {
	if c.Version == "" {
		c.Version = "1.0.0"
	}

	// Set default hooks
	c.StatusHooks.SetDefaults()

	// Set default worktree config
	c.Worktree.SetDefaults()

	// Set default commands
	c.Commands.SetDefaults()

	// Set default tmux config
	c.Tmux.SetDefaults()

	// Set default git config
	c.Git.SetDefaults()

	// Set default claude config
	c.Claude.SetDefaults()

	// Set default shortcuts if none provided
	if len(c.Shortcuts) == 0 {
		c.Shortcuts = DefaultShortcuts()
	}
}

// SetDefaults sets default values for status hooks
func (s *StatusHooksConfig) SetDefaults() {
	s.Enabled = true
	s.IdleHook.SetDefaults("idle")
	s.BusyHook.SetDefaults("busy")
	s.WaitingHook.SetDefaults("waiting")
}

// SetDefaults sets default values for individual hook
func (h *HookConfig) SetDefaults(hookType string) {
	h.Enabled = true // Enable hooks by default
	if h.Script == "" {
		h.Script = fmt.Sprintf("~/.config/ccmgr-ultra/hooks/%s.sh", hookType)
	}
	if h.Timeout == 0 {
		h.Timeout = 30
	}
	h.Async = true // Set async by default
}

// SetDefaults sets default values for worktree config
func (w *WorktreeConfig) SetDefaults() {
	if w.DirectoryPattern == "" {
		w.DirectoryPattern = "{{.project}}-{{.branch}}"
	}
	if w.DefaultBranch == "" {
		w.DefaultBranch = "main"
	}
}

// SetDefaults sets default values for commands config
func (c *CommandsConfig) SetDefaults() {
	if c.ClaudeCommand == "" {
		c.ClaudeCommand = "claude"
	}
	if c.GitCommand == "" {
		c.GitCommand = "git"
	}
	if c.TmuxPrefix == "" {
		c.TmuxPrefix = "ccmgr"
	}
	if c.Environment == nil {
		c.Environment = make(map[string]string)
	}
}

// SetDefaults sets default values for tmux config
func (t *TmuxConfig) SetDefaults() {
	if t.SessionPrefix == "" {
		t.SessionPrefix = "ccmgr"
	}
	if t.NamingPattern == "" {
		t.NamingPattern = "{{.prefix}}-{{.project}}-{{.worktree}}-{{.branch}}"
	}
	if t.MaxSessionName == 0 {
		t.MaxSessionName = 50
	}
	if t.MonitorInterval == 0 {
		t.MonitorInterval = 2 * time.Second
	}
	if t.StateFile == "" {
		t.StateFile = "~/.config/ccmgr-ultra/tmux-sessions.json"
	}
	if t.DefaultEnv == nil {
		t.DefaultEnv = make(map[string]string)
	}
	if t.CleanupAge == 0 {
		t.CleanupAge = 24 * time.Hour
	}
}

// SetDefaults sets default values for git config
func (g *GitConfig) SetDefaults() {
	if g.DirectoryPattern == "" {
		g.DirectoryPattern = "{{.project}}-{{.branch}}"
	}
	if g.DefaultBranch == "" {
		g.DefaultBranch = "main"
	}
	if g.MaxWorktrees == 0 {
		g.MaxWorktrees = 10
	}
	if g.CleanupAge == 0 {
		g.CleanupAge = 168 * time.Hour // 7 days
	}
	if g.DefaultRemote == "" {
		g.DefaultRemote = "origin"
	}
	if g.ProtectedBranches == nil {
		g.ProtectedBranches = []string{"main", "master", "develop"}
	}
	if g.PRTemplate == "" {
		g.PRTemplate = `## Summary
Brief description of changes

## Testing
How the changes were tested`
	}
	// Boolean defaults are handled by Go's zero values and struct tags
}

// SetDefaults sets default values for claude config
func (c *ClaudeConfig) SetDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = 3 * time.Second
	}
	if c.MaxProcesses == 0 {
		c.MaxProcesses = 10
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = 5 * time.Minute
	}
	if c.StateTimeout == 0 {
		c.StateTimeout = 30 * time.Second
	}
	if c.StartupTimeout == 0 {
		c.StartupTimeout = 10 * time.Second
	}
	if len(c.LogPaths) == 0 {
		c.LogPaths = []string{
			"~/.claude/logs",
			"/tmp/claude-*",
		}
	}
	if len(c.StatePatterns) == 0 {
		c.StatePatterns = map[string]string{
			"busy":    `(?i)(Processing|Executing|Running|Working on|Analyzing|Generating)`,
			"idle":    `(?i)(Waiting for input|Ready|Idle|Available)`,
			"waiting": `(?i)(Waiting for confirmation|Press any key|Continue\?|Y/n)`,
			"error":   `(?i)(Error|Failed|Exception|Panic|Fatal)`,
		}
	}
	// Boolean defaults are handled by Go's zero values and struct tags
	c.Enabled = true
	c.EnableLogParsing = true
	c.EnableResourceMonitoring = true
	c.IntegrateTmux = true
	c.IntegrateWorktrees = true
}

// DefaultShortcuts returns the default keyboard shortcuts
func DefaultShortcuts() map[string]string {
	return map[string]string{
		"n": "new_worktree",
		"m": "merge_worktree",
		"d": "delete_worktree",
		"p": "push_worktree",
		"c": "continue_session",
		"r": "resume_session",
		"q": "quit",
	}
}