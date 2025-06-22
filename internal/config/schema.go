package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Config represents the main configuration structure
type Config struct {
	Version       string                 `yaml:"version" json:"version"`
	StatusHooks   StatusHooksConfig      `yaml:"status_hooks" json:"status_hooks"`
	WorktreeHooks WorktreeHooksConfig    `yaml:"worktree_hooks" json:"worktree_hooks"`
	Worktree      WorktreeConfig         `yaml:"worktree" json:"worktree"`
	Tmux          TmuxConfig             `yaml:"tmux" json:"tmux"`
	Git              GitConfig              `yaml:"git" json:"git"`
	Claude           ClaudeConfig           `yaml:"claude" json:"claude"`
	TUI              TUIConfig              `yaml:"tui" json:"tui"`
	Analytics        AnalyticsConfig        `yaml:"analytics" json:"analytics"`
	Shortcuts        map[string]string      `yaml:"shortcuts" json:"shortcuts"`
	Commands         CommandsConfig         `yaml:"commands" json:"commands"`
	LastModified     time.Time              `yaml:"last_modified" json:"last_modified"`
	
	// Additional common config fields
	ConfigFile       string                 `yaml:"-" json:"-"`
	LogLevel         string                 `yaml:"log_level" json:"log_level" default:"info"`
	RefreshInterval  int                    `yaml:"refresh_interval" json:"refresh_interval" default:"5"`
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

// WorktreeHooksConfig defines worktree lifecycle hooks
type WorktreeHooksConfig struct {
	Enabled        bool       `yaml:"enabled" json:"enabled"`
	CreationHook   HookConfig `yaml:"creation" json:"creation"`
	ActivationHook HookConfig `yaml:"activation" json:"activation"`
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

	// GitHub-specific configuration (Phase 5.3)
	GitHubPRTemplate       string `yaml:"github_pr_template" json:"github_pr_template"`
	DefaultPRTargetBranch  string `yaml:"default_pr_target_branch" json:"default_pr_target_branch" default:"main"`

	// Safety settings
	RequireCleanWorkdir bool `yaml:"require_clean_workdir" json:"require_clean_workdir" default:"true"`
	ConfirmDestructive  bool `yaml:"confirm_destructive" json:"confirm_destructive" default:"true"`
	BackupOnDelete      bool `yaml:"backup_on_delete" json:"backup_on_delete" default:"true"`
}

// TUIConfig defines TUI application configuration
type TUIConfig struct {
	// Display settings
	Theme           string `yaml:"theme" json:"theme" default:"default"`
	RefreshInterval int    `yaml:"refresh_interval" json:"refresh_interval" default:"5"` // seconds
	MouseSupport    bool   `yaml:"mouse_support" json:"mouse_support" default:"true"`
	
	// Screen settings
	DefaultScreen   string `yaml:"default_screen" json:"default_screen" default:"dashboard"`
	ShowStatusBar   bool   `yaml:"show_status_bar" json:"show_status_bar" default:"true"`
	ShowKeyHelp     bool   `yaml:"show_key_help" json:"show_key_help" default:"true"`
	
	// Behavior settings
	ConfirmQuit     bool   `yaml:"confirm_quit" json:"confirm_quit" default:"false"`
	AutoRefresh     bool   `yaml:"auto_refresh" json:"auto_refresh" default:"true"`
	DebugMode       bool   `yaml:"debug_mode" json:"debug_mode" default:"false"`
}

// AnalyticsConfig defines analytics configuration
type AnalyticsConfig struct {
	Enabled         bool                    `yaml:"enabled" json:"enabled" default:"true"`
	Collector       AnalyticsCollectorConfig `yaml:"collector" json:"collector"`
	Engine          AnalyticsEngineConfig    `yaml:"engine" json:"engine"`
	Hooks           AnalyticsHooksConfig     `yaml:"hooks" json:"hooks"`
	Retention       AnalyticsRetentionConfig `yaml:"retention" json:"retention"`
	Performance     AnalyticsPerformanceConfig `yaml:"performance" json:"performance"`
}

// AnalyticsCollectorConfig defines collector configuration
type AnalyticsCollectorConfig struct {
	PollInterval    time.Duration `yaml:"poll_interval" json:"poll_interval" default:"30s"`
	BufferSize      int           `yaml:"buffer_size" json:"buffer_size" default:"1000"`
	BatchSize       int           `yaml:"batch_size" json:"batch_size" default:"50"`
	EnableMetrics   bool          `yaml:"enable_metrics" json:"enable_metrics" default:"true"`
	RetentionDays   int           `yaml:"retention_days" json:"retention_days" default:"90"`
}

// AnalyticsEngineConfig defines engine configuration
type AnalyticsEngineConfig struct {
	CacheSize       int           `yaml:"cache_size" json:"cache_size" default:"1000"`
	CacheTTL        time.Duration `yaml:"cache_ttl" json:"cache_ttl" default:"5m"`
	BatchProcessing bool          `yaml:"batch_processing" json:"batch_processing" default:"true"`
	PrecomputeDaily bool          `yaml:"precompute_daily" json:"precompute_daily" default:"true"`
}

// AnalyticsHooksConfig defines hooks integration configuration
type AnalyticsHooksConfig struct {
	Enabled              bool `yaml:"enabled" json:"enabled" default:"true"`
	CaptureStateChanges  bool `yaml:"capture_state_changes" json:"capture_state_changes" default:"true"`
	CaptureWorktreeEvents bool `yaml:"capture_worktree_events" json:"capture_worktree_events" default:"true"`
	CaptureSessionEvents bool `yaml:"capture_session_events" json:"capture_session_events" default:"true"`
}

// AnalyticsRetentionConfig defines data retention configuration
type AnalyticsRetentionConfig struct {
	SessionEventsDays    int           `yaml:"session_events_days" json:"session_events_days" default:"90"`
	AggregatedDataDays   int           `yaml:"aggregated_data_days" json:"aggregated_data_days" default:"365"`
	CleanupInterval      time.Duration `yaml:"cleanup_interval" json:"cleanup_interval" default:"24h"`
	EnableAutoCleanup    bool          `yaml:"enable_auto_cleanup" json:"enable_auto_cleanup" default:"true"`
}

// AnalyticsPerformanceConfig defines performance configuration
type AnalyticsPerformanceConfig struct {
	MaxCPUUsage       float64       `yaml:"max_cpu_usage" json:"max_cpu_usage" default:"5.0"`
	MaxMemoryUsageMB  int64         `yaml:"max_memory_usage_mb" json:"max_memory_usage_mb" default:"100"`
	MaxQueryTime      time.Duration `yaml:"max_query_time" json:"max_query_time" default:"100ms"`
	EnableMonitoring  bool          `yaml:"enable_monitoring" json:"enable_monitoring" default:"true"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return errors.New("config version is required")
	}

	if err := c.StatusHooks.Validate(); err != nil {
		return fmt.Errorf("status hooks validation failed: %w", err)
	}

	if err := c.WorktreeHooks.Validate(); err != nil {
		return fmt.Errorf("worktree hooks validation failed: %w", err)
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

	if err := c.TUI.Validate(); err != nil {
		return fmt.Errorf("tui validation failed: %w", err)
	}

	if err := c.Analytics.Validate(); err != nil {
		return fmt.Errorf("analytics validation failed: %w", err)
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

// Validate validates worktree hooks configuration
func (w *WorktreeHooksConfig) Validate() error {
	if err := w.CreationHook.Validate(); err != nil {
		return fmt.Errorf("creation hook validation failed: %w", err)
	}

	if err := w.ActivationHook.Validate(); err != nil {
		return fmt.Errorf("activation hook validation failed: %w", err)
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
	c.WorktreeHooks.SetDefaults()

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

	// Set default TUI config
	c.TUI.SetDefaults()

	// Set default analytics config
	c.Analytics.SetDefaults()

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

// SetDefaults sets default values for worktree hooks
func (w *WorktreeHooksConfig) SetDefaults() {
	w.Enabled = true
	w.CreationHook.SetDefaults("creation")
	w.ActivationHook.SetDefaults("activation")
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
	
	// GitHub-specific defaults (Phase 5.3)
	if g.GitHubPRTemplate == "" {
		g.GitHubPRTemplate = `## Summary
Brief description of changes

## Test plan
- [ ] Manual testing completed
- [ ] Unit tests pass
- [ ] Integration tests pass

## Checklist
- [ ] Code follows project conventions
- [ ] Documentation updated if needed`
	}
	
	if g.DefaultPRTargetBranch == "" {
		g.DefaultPRTargetBranch = "main"
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

// SetDefaults sets default values for TUI configuration
func (t *TUIConfig) SetDefaults() {
	if t.Theme == "" {
		t.Theme = "default"
	}
	if t.RefreshInterval == 0 {
		t.RefreshInterval = 5
	}
	if t.DefaultScreen == "" {
		t.DefaultScreen = "dashboard"
	}
	// Boolean defaults are handled by Go's zero values
	t.MouseSupport = true
	t.ShowStatusBar = true
	t.ShowKeyHelp = true
	t.ConfirmQuit = false
	t.AutoRefresh = true
	t.DebugMode = false
}

// Validate validates TUI configuration
func (t *TUIConfig) Validate() error {
	if t.RefreshInterval < 1 {
		return errors.New("refresh interval must be at least 1 second")
	}
	
	validScreens := []string{"dashboard", "sessions", "worktrees", "config", "help"}
	validScreen := false
	for _, screen := range validScreens {
		if t.DefaultScreen == screen {
			validScreen = true
			break
		}
	}
	if !validScreen {
		return fmt.Errorf("invalid default screen: %s", t.DefaultScreen)
	}
	
	return nil
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

// Validate validates analytics configuration
func (a *AnalyticsConfig) Validate() error {
	if err := a.Collector.Validate(); err != nil {
		return fmt.Errorf("collector validation failed: %w", err)
	}
	
	if err := a.Engine.Validate(); err != nil {
		return fmt.Errorf("engine validation failed: %w", err)
	}
	
	if err := a.Hooks.Validate(); err != nil {
		return fmt.Errorf("hooks validation failed: %w", err)
	}
	
	if err := a.Retention.Validate(); err != nil {
		return fmt.Errorf("retention validation failed: %w", err)
	}
	
	if err := a.Performance.Validate(); err != nil {
		return fmt.Errorf("performance validation failed: %w", err)
	}
	
	return nil
}

// SetDefaults sets default values for analytics configuration
func (a *AnalyticsConfig) SetDefaults() {
	a.Enabled = true
	a.Collector.SetDefaults()
	a.Engine.SetDefaults()
	a.Hooks.SetDefaults()
	a.Retention.SetDefaults()
	a.Performance.SetDefaults()
}

// Validate validates collector configuration
func (c *AnalyticsCollectorConfig) Validate() error {
	if c.PollInterval < time.Second {
		return fmt.Errorf("poll interval must be at least 1 second")
	}
	if c.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if c.BatchSize > c.BufferSize {
		return fmt.Errorf("batch size cannot exceed buffer size")
	}
	if c.RetentionDays < 0 {
		return fmt.Errorf("retention days cannot be negative")
	}
	return nil
}

// SetDefaults sets default values for collector configuration
func (c *AnalyticsCollectorConfig) SetDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = 30 * time.Second
	}
	if c.BufferSize == 0 {
		c.BufferSize = 1000
	}
	if c.BatchSize == 0 {
		c.BatchSize = 50
	}
	if c.RetentionDays == 0 {
		c.RetentionDays = 90
	}
	c.EnableMetrics = true
}

// Validate validates engine configuration
func (e *AnalyticsEngineConfig) Validate() error {
	if e.CacheSize < 0 {
		return fmt.Errorf("cache size cannot be negative")
	}
	if e.CacheTTL < 0 {
		return fmt.Errorf("cache TTL cannot be negative")
	}
	return nil
}

// SetDefaults sets default values for engine configuration
func (e *AnalyticsEngineConfig) SetDefaults() {
	if e.CacheSize == 0 {
		e.CacheSize = 1000
	}
	if e.CacheTTL == 0 {
		e.CacheTTL = 5 * time.Minute
	}
	e.BatchProcessing = true
	e.PrecomputeDaily = true
}

// Validate validates hooks configuration
func (h *AnalyticsHooksConfig) Validate() error {
	// No specific validation needed
	return nil
}

// SetDefaults sets default values for hooks configuration
func (h *AnalyticsHooksConfig) SetDefaults() {
	h.Enabled = true
	h.CaptureStateChanges = true
	h.CaptureWorktreeEvents = true
	h.CaptureSessionEvents = true
}

// Validate validates retention configuration
func (r *AnalyticsRetentionConfig) Validate() error {
	if r.SessionEventsDays < 0 {
		return fmt.Errorf("session events retention days cannot be negative")
	}
	if r.AggregatedDataDays < 0 {
		return fmt.Errorf("aggregated data retention days cannot be negative")
	}
	if r.CleanupInterval < 0 {
		return fmt.Errorf("cleanup interval cannot be negative")
	}
	return nil
}

// SetDefaults sets default values for retention configuration
func (r *AnalyticsRetentionConfig) SetDefaults() {
	if r.SessionEventsDays == 0 {
		r.SessionEventsDays = 90
	}
	if r.AggregatedDataDays == 0 {
		r.AggregatedDataDays = 365
	}
	if r.CleanupInterval == 0 {
		r.CleanupInterval = 24 * time.Hour
	}
	r.EnableAutoCleanup = true
}

// Validate validates performance configuration
func (p *AnalyticsPerformanceConfig) Validate() error {
	if p.MaxCPUUsage < 0 {
		return fmt.Errorf("max CPU usage cannot be negative")
	}
	if p.MaxCPUUsage > 100 {
		return fmt.Errorf("max CPU usage cannot exceed 100%%")
	}
	if p.MaxMemoryUsageMB < 0 {
		return fmt.Errorf("max memory usage cannot be negative")
	}
	if p.MaxQueryTime < 0 {
		return fmt.Errorf("max query time cannot be negative")
	}
	return nil
}

// SetDefaults sets default values for performance configuration
func (p *AnalyticsPerformanceConfig) SetDefaults() {
	if p.MaxCPUUsage == 0 {
		p.MaxCPUUsage = 5.0
	}
	if p.MaxMemoryUsageMB == 0 {
		p.MaxMemoryUsageMB = 100
	}
	if p.MaxQueryTime == 0 {
		p.MaxQueryTime = 100 * time.Millisecond
	}
	p.EnableMonitoring = true
}