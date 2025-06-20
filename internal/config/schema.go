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

	// Set default shortcuts if none provided
	if len(c.Shortcuts) == 0 {
		c.Shortcuts = DefaultShortcuts()
	}
}

// SetDefaults sets default values for status hooks
func (s *StatusHooksConfig) SetDefaults() {
	s.IdleHook.SetDefaults("idle")
	s.BusyHook.SetDefaults("busy")
	s.WaitingHook.SetDefaults("waiting")
}

// SetDefaults sets default values for individual hook
func (h *HookConfig) SetDefaults(hookType string) {
	if h.Script == "" && h.Enabled {
		h.Script = fmt.Sprintf("~/.config/ccmgr-ultra/hooks/%s.sh", hookType)
	}
	if h.Timeout == 0 {
		h.Timeout = 30
	}
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