# Phase 2.1: Configuration Management Implementation Plan

## Overview
This document provides a detailed implementation and validation plan for the Configuration Management component of ccmgr-ultra, located in `internal/config/`.

## Implementation Steps

### Step 1: Define Configuration Schema (internal/config/schema.go)

#### 1.1 Create Data Structures
```go
// Define the main configuration structure
type Config struct {
    Version      string                 `yaml:"version"`
    StatusHooks  StatusHooksConfig      `yaml:"status_hooks"`
    Worktree     WorktreeConfig         `yaml:"worktree"`
    Shortcuts    map[string]string      `yaml:"shortcuts"`
    Commands     CommandsConfig         `yaml:"commands"`
    LastModified time.Time              `yaml:"last_modified"`
}

// Status hooks configuration
type StatusHooksConfig struct {
    Enabled      bool              `yaml:"enabled"`
    IdleHook     HookConfig        `yaml:"idle"`
    BusyHook     HookConfig        `yaml:"busy"`
    WaitingHook  HookConfig        `yaml:"waiting"`
}

type HookConfig struct {
    Enabled  bool   `yaml:"enabled"`
    Script   string `yaml:"script"`
    Timeout  int    `yaml:"timeout"` // seconds
    Async    bool   `yaml:"async"`
}

// Worktree configuration
type WorktreeConfig struct {
    AutoDirectory    bool   `yaml:"auto_directory"`
    DirectoryPattern string `yaml:"directory_pattern"` // e.g., "{{.project}}-{{.branch}}"
    DefaultBranch    string `yaml:"default_branch"`
    CleanupOnMerge   bool   `yaml:"cleanup_on_merge"`
}

// Commands configuration
type CommandsConfig struct {
    ClaudeCommand   string            `yaml:"claude_command"`
    GitCommand      string            `yaml:"git_command"`
    TmuxPrefix      string            `yaml:"tmux_prefix"`
    Environment     map[string]string `yaml:"environment"`
}
```

#### 1.2 Add Validation Methods
```go
// Implement validation for each config section
func (c *Config) Validate() error
func (h *HookConfig) Validate() error
func (w *WorktreeConfig) Validate() error
```

### Step 2: Implement Config File Operations (internal/config/config.go)

#### 2.1 Core Functions
```go
// Loading and saving
func Load(path string) (*Config, error)
func Save(config *Config, path string) error
func LoadOrCreate(path string) (*Config, error)

// Config file locations
func GetConfigPath() string
func GetProjectConfigPath(projectPath string) string
func GetGlobalConfigPath() string

// Merging configurations (project overrides global)
func MergeConfigs(global, project *Config) *Config
```

#### 2.2 Default Configuration
```go
func DefaultConfig() *Config {
    return &Config{
        Version: "1.0.0",
        StatusHooks: StatusHooksConfig{
            Enabled: true,
            IdleHook: HookConfig{
                Enabled: true,
                Script:  "~/.config/ccmgr-ultra/hooks/idle.sh",
                Timeout: 30,
                Async:   true,
            },
            BusyHook: HookConfig{
                Enabled: true,
                Script:  "~/.config/ccmgr-ultra/hooks/busy.sh",
                Timeout: 30,
                Async:   true,
            },
            WaitingHook: HookConfig{
                Enabled: true,
                Script:  "~/.config/ccmgr-ultra/hooks/waiting.sh",
                Timeout: 30,
                Async:   true,
            },
        },
        Worktree: WorktreeConfig{
            AutoDirectory:    true,
            DirectoryPattern: "{{.project}}-{{.branch}}",
            DefaultBranch:    "main",
            CleanupOnMerge:   false,
        },
        Shortcuts: map[string]string{
            "n": "new_worktree",
            "m": "merge_worktree",
            "d": "delete_worktree",
            "p": "push_worktree",
            "c": "continue_session",
            "r": "resume_session",
            "q": "quit",
        },
        Commands: CommandsConfig{
            ClaudeCommand: "claude",
            GitCommand:    "git",
            TmuxPrefix:    "ccmgr",
            Environment:   map[string]string{},
        },
    }
}
```

### Step 3: Configuration Migration

#### 3.1 Migration System
```go
// Migration interface
type Migration interface {
    Version() string
    Migrate(oldConfig map[string]interface{}) (map[string]interface{}, error)
}

// Migration registry
type MigrationRegistry struct {
    migrations []Migration
}

// Apply migrations
func (m *MigrationRegistry) Migrate(config *Config) error
```

#### 3.2 Version Detection
```go
func DetectConfigVersion(data []byte) (string, error)
func NeedsMigration(current, target string) bool
```

### Step 4: Viper Integration

#### 4.1 Setup Viper
```go
func InitViper() {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("$HOME/.config/ccmgr-ultra")
    viper.AddConfigPath(".")
    
    // Set defaults
    viper.SetDefault("version", "1.0.0")
    viper.SetDefault("status_hooks.enabled", true)
    // ... other defaults
}
```

#### 4.2 Environment Variable Support
```go
func BindEnvironmentVariables() {
    viper.SetEnvPrefix("CCMGR")
    viper.AutomaticEnv()
    
    // Specific bindings
    viper.BindEnv("claude_command", "CCMGR_CLAUDE_COMMAND")
    viper.BindEnv("worktree.auto_directory", "CCMGR_AUTO_DIRECTORY")
}
```

### Step 5: Template Creation

#### 5.1 Default Config Template File
Create `internal/config/template.yaml`:
```yaml
# ccmgr-ultra configuration file
version: "1.0.0"

# Status hooks - scripts executed on Claude Code state changes
status_hooks:
  enabled: true
  idle:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/idle.sh"
    timeout: 30
    async: true
  busy:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/busy.sh"
    timeout: 30
    async: true
  waiting:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/waiting.sh"
    timeout: 30
    async: true

# Worktree settings
worktree:
  auto_directory: true
  directory_pattern: "{{.project}}-{{.branch}}"
  default_branch: "main"
  cleanup_on_merge: false

# Keyboard shortcuts
shortcuts:
  n: "new_worktree"
  m: "merge_worktree"
  d: "delete_worktree"
  p: "push_worktree"
  c: "continue_session"
  r: "resume_session"
  q: "quit"

# Command settings
commands:
  claude_command: "claude"
  git_command: "git"
  tmux_prefix: "ccmgr"
  environment: {}
```

#### 5.2 Example Hook Scripts
Create example scripts in `scripts/hooks/`:
- `idle.sh.example`
- `busy.sh.example`
- `waiting.sh.example`

## Validation Plan

### Unit Tests (internal/config/config_test.go)

#### Test Cases:
1. **Schema Validation**
   - Valid configuration passes validation
   - Invalid hook timeout rejected
   - Invalid directory pattern rejected
   - Missing required fields detected

2. **File Operations**
   - Load existing config file
   - Create new config with defaults
   - Save config preserves formatting
   - Handle missing config file gracefully
   - Handle corrupted config file

3. **Configuration Merging**
   - Project config overrides global
   - Partial overrides work correctly
   - Shortcuts merge properly
   - Environment variables merge

4. **Migration System**
   - Detect old version correctly
   - Migrate v0.9.0 to v1.0.0
   - Preserve custom settings during migration
   - Backup created before migration

5. **Template System**
   - Generate default config from template
   - Variable substitution works
   - Comments preserved

### Integration Tests

1. **Viper Integration**
   - Environment variables override config
   - Config file changes detected
   - Multiple config paths work

2. **File System Integration**
   - Correct paths on different OS
   - Handle permissions issues
   - XDG Base Directory compliance

### Manual Testing Checklist

1. **Initial Setup**
   - [ ] Fresh install creates default config
   - [ ] Config location follows OS conventions
   - [ ] Example hooks copied to user directory

2. **Configuration Editing**
   - [ ] Manual edits preserved on reload
   - [ ] Invalid YAML shows helpful error
   - [ ] Comments in config preserved

3. **Hook Configuration**
   - [ ] Enable/disable hooks works
   - [ ] Hook script paths validated
   - [ ] Missing hook scripts handled gracefully

4. **Worktree Settings**
   - [ ] Directory pattern variables work
   - [ ] Auto-directory toggle functions
   - [ ] Invalid patterns rejected

5. **Migration Testing**
   - [ ] Old config format detected
   - [ ] Migration preserves customizations
   - [ ] Backup created successfully
   - [ ] Can rollback if needed

## Performance Considerations

1. **Config Loading**
   - Cache parsed config in memory
   - Lazy load project configs
   - Watch for file changes efficiently

2. **Validation**
   - Validate on load, not every access
   - Cache validation results
   - Fast-fail on critical errors

## Security Considerations

1. **File Permissions**
   - Config files created with 0600
   - Warn if config world-readable
   - Validate hook script permissions

2. **Path Validation**
   - Prevent directory traversal
   - Validate script paths exist
   - Sanitize template variables

## Error Handling

1. **Graceful Degradation**
   - Use defaults if config missing
   - Continue with partial config
   - Clear error messages

2. **User Feedback**
   - Show which config file has error
   - Indicate line number for YAML errors
   - Suggest fixes for common issues

## Documentation Requirements

1. **Config Reference**
   - Document all settings
   - Provide examples
   - Explain precedence rules

2. **Migration Guide**
   - Document breaking changes
   - Provide migration scripts
   - Show before/after examples

## Success Metrics

1. **Functionality**
   - All config options accessible
   - Migration handles all versions
   - Validation catches common errors

2. **Performance**
   - Config loads in <50ms
   - Validation in <10ms
   - No memory leaks

3. **Usability**
   - Clear error messages
   - Intuitive defaults
   - Easy customization

## Timeline

- **Day 1-2**: Implement schema and basic structures
- **Day 3-4**: File operations and Viper integration
- **Day 5-6**: Migration system and templates
- **Day 7-8**: Comprehensive testing
- **Day 9-10**: Documentation and polish

## Dependencies

- github.com/spf13/viper (config management)
- gopkg.in/yaml.v3 (YAML parsing)
- github.com/mitchellh/mapstructure (config decoding)
- github.com/stretchr/testify (testing)