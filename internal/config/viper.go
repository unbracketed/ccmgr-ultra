package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ViperManager manages Viper configuration instances
type ViperManager struct {
	global  *viper.Viper
	project *viper.Viper
	merged  *Config
}

// NewViperManager creates a new Viper manager
func NewViperManager() *ViperManager {
	return &ViperManager{
		global:  viper.New(),
		project: viper.New(),
	}
}

// InitGlobalViper initializes the global Viper instance
func (vm *ViperManager) InitGlobalViper() error {
	vm.global.SetConfigName("config")
	vm.global.SetConfigType("yaml")
	
	// Add config paths
	configPath := GetConfigPath()
	vm.global.AddConfigPath(configPath)
	
	// Set defaults
	vm.setDefaults(vm.global)
	
	// Bind environment variables
	vm.bindEnvironment(vm.global)
	
	// Try to read config
	if err := vm.global.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Create default config
			config := DefaultConfig()
			configFile := filepath.Join(configPath, ConfigFileName)
			if err := Save(config, configFile); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
			// Try reading again
			if err := vm.global.ReadInConfig(); err != nil {
				return fmt.Errorf("failed to read created config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read global config: %w", err)
		}
	}
	
	return nil
}

// InitProjectViper initializes the project-specific Viper instance
func (vm *ViperManager) InitProjectViper(projectPath string) error {
	vm.project.SetConfigName("config")
	vm.project.SetConfigType("yaml")
	
	// Add project config path
	projectConfigPath := filepath.Join(projectPath, ".ccmgr-ultra")
	vm.project.AddConfigPath(projectConfigPath)
	
	// Set defaults (not needed for project config as we merge with global)
	// vm.setDefaults(vm.project)
	
	// Bind environment variables with higher precedence prefix
	vm.bindProjectEnvironment(vm.project)
	
	// Try to read config (it's ok if it doesn't exist)
	vm.project.ReadInConfig()
	
	return nil
}

// GetMergedConfig returns the merged configuration
func (vm *ViperManager) GetMergedConfig() (*Config, error) {
	// Get global config
	var globalConfig Config
	if err := vm.global.Unmarshal(&globalConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global config: %w", err)
	}
	
	// Set defaults for global config
	globalConfig.SetDefaults()
	
	// Get project config if available
	var projectConfig *Config
	if vm.project.ConfigFileUsed() != "" {
		var pc Config
		if err := vm.project.Unmarshal(&pc); err == nil {
			projectConfig = &pc
		}
	}
	
	// Merge configs
	merged := MergeConfigs(&globalConfig, projectConfig)
	
	// Apply environment variable overrides
	vm.applyEnvironmentOverrides(merged)
	
	// Validate merged config
	if err := merged.Validate(); err != nil {
		return nil, fmt.Errorf("merged config validation failed: %w", err)
	}
	
	vm.merged = merged
	return merged, nil
}

// WatchConfigs watches both global and project configs for changes
func (vm *ViperManager) WatchConfigs(onChange func(*Config)) {
	// Watch global config
	vm.global.WatchConfig()
	vm.global.OnConfigChange(func(e fsnotify.Event) {
		if config, err := vm.GetMergedConfig(); err == nil {
			onChange(config)
		}
	})
	
	// Watch project config if available
	if vm.project.ConfigFileUsed() != "" {
		vm.project.WatchConfig()
		vm.project.OnConfigChange(func(e fsnotify.Event) {
			if config, err := vm.GetMergedConfig(); err == nil {
				onChange(config)
			}
		})
	}
}

// setDefaults sets default values in Viper
func (vm *ViperManager) setDefaults(v *viper.Viper) {
	// Version
	v.SetDefault("version", "1.0.0")
	
	// Status hooks
	v.SetDefault("status_hooks.enabled", true)
	v.SetDefault("status_hooks.idle.enabled", true)
	v.SetDefault("status_hooks.idle.script", "~/.config/ccmgr-ultra/hooks/idle.sh")
	v.SetDefault("status_hooks.idle.timeout", 30)
	v.SetDefault("status_hooks.idle.async", true)
	v.SetDefault("status_hooks.busy.enabled", true)
	v.SetDefault("status_hooks.busy.script", "~/.config/ccmgr-ultra/hooks/busy.sh")
	v.SetDefault("status_hooks.busy.timeout", 30)
	v.SetDefault("status_hooks.busy.async", true)
	v.SetDefault("status_hooks.waiting.enabled", true)
	v.SetDefault("status_hooks.waiting.script", "~/.config/ccmgr-ultra/hooks/waiting.sh")
	v.SetDefault("status_hooks.waiting.timeout", 30)
	v.SetDefault("status_hooks.waiting.async", true)
	
	// Worktree
	v.SetDefault("worktree.auto_directory", true)
	v.SetDefault("worktree.directory_pattern", "{{.Project}}-{{.Branch}}")
	v.SetDefault("worktree.default_branch", "main")
	v.SetDefault("worktree.cleanup_on_merge", false)
	
	// Commands
	v.SetDefault("commands.claude_command", "claude")
	v.SetDefault("commands.git_command", "git")
	v.SetDefault("commands.tmux_prefix", "ccmgr")
	
	// Shortcuts
	shortcuts := DefaultShortcuts()
	for key, action := range shortcuts {
		v.SetDefault(fmt.Sprintf("shortcuts.%s", key), action)
	}
}

// bindEnvironment binds environment variables
func (vm *ViperManager) bindEnvironment(v *viper.Viper) {
	v.SetEnvPrefix("CCMGR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Specific bindings for nested configs
	v.BindEnv("status_hooks.enabled")
	v.BindEnv("status_hooks.idle.enabled")
	v.BindEnv("status_hooks.idle.script")
	v.BindEnv("status_hooks.idle.timeout")
	v.BindEnv("status_hooks.idle.async")
	v.BindEnv("status_hooks.busy.enabled")
	v.BindEnv("status_hooks.busy.script")
	v.BindEnv("status_hooks.busy.timeout")
	v.BindEnv("status_hooks.busy.async")
	v.BindEnv("status_hooks.waiting.enabled")
	v.BindEnv("status_hooks.waiting.script")
	v.BindEnv("status_hooks.waiting.timeout")
	v.BindEnv("status_hooks.waiting.async")
	
	v.BindEnv("worktree.auto_directory")
	v.BindEnv("worktree.directory_pattern")
	v.BindEnv("worktree.default_branch")
	v.BindEnv("worktree.cleanup_on_merge")
	
	v.BindEnv("commands.claude_command")
	v.BindEnv("commands.git_command")
	v.BindEnv("commands.tmux_prefix")
}

// bindProjectEnvironment binds project-specific environment variables
func (vm *ViperManager) bindProjectEnvironment(v *viper.Viper) {
	v.SetEnvPrefix("CCMGR_PROJECT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
}

// applyEnvironmentOverrides applies environment variable overrides to config
func (vm *ViperManager) applyEnvironmentOverrides(config *Config) {
	// Check for specific environment variable overrides
	// This ensures environment variables take precedence over all config files
	
	// Status hooks
	if val := os.Getenv("CCMGR_STATUS_HOOKS_ENABLED"); val != "" {
		config.StatusHooks.Enabled = val == "true"
	}
	
	// Worktree
	if val := os.Getenv("CCMGR_WORKTREE_AUTO_DIRECTORY"); val != "" {
		config.Worktree.AutoDirectory = val == "true"
	}
	if val := os.Getenv("CCMGR_WORKTREE_DIRECTORY_PATTERN"); val != "" {
		config.Worktree.DirectoryPattern = val
	}
	if val := os.Getenv("CCMGR_WORKTREE_DEFAULT_BRANCH"); val != "" {
		config.Worktree.DefaultBranch = val
	}
	if val := os.Getenv("CCMGR_WORKTREE_CLEANUP_ON_MERGE"); val != "" {
		config.Worktree.CleanupOnMerge = val == "true"
	}
	
	// Commands
	if val := os.Getenv("CCMGR_COMMANDS_CLAUDE_COMMAND"); val != "" {
		config.Commands.ClaudeCommand = val
	}
	if val := os.Getenv("CCMGR_COMMANDS_GIT_COMMAND"); val != "" {
		config.Commands.GitCommand = val
	}
	if val := os.Getenv("CCMGR_COMMANDS_TMUX_PREFIX"); val != "" {
		config.Commands.TmuxPrefix = val
	}
}

// GetValue gets a specific configuration value
func (vm *ViperManager) GetValue(key string) interface{} {
	// Check project config first
	if vm.project.IsSet(key) {
		return vm.project.Get(key)
	}
	// Fall back to global config
	return vm.global.Get(key)
}

// SetValue sets a configuration value
func (vm *ViperManager) SetValue(key string, value interface{}, isProjectSpecific bool) {
	if isProjectSpecific {
		vm.project.Set(key, value)
	} else {
		vm.global.Set(key, value)
	}
}

// SaveGlobalConfig saves the global configuration
func (vm *ViperManager) SaveGlobalConfig() error {
	return vm.global.WriteConfig()
}

// SaveProjectConfig saves the project configuration
func (vm *ViperManager) SaveProjectConfig() error {
	if vm.project.ConfigFileUsed() == "" {
		// Need to set config file path
		projectPath, _ := os.Getwd()
		configPath := GetProjectConfigPath(projectPath)
		
		// Ensure directory exists
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create project config directory: %w", err)
		}
		
		vm.project.SetConfigFile(configPath)
	}
	return vm.project.WriteConfig()
}

// InitViper initializes a standalone Viper instance with defaults
func InitViper() *viper.Viper {
	v := viper.New()
	
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("$HOME/.config/ccmgr-ultra")
	v.AddConfigPath(".")
	
	// Set all defaults
	manager := &ViperManager{}
	manager.setDefaults(v)
	
	// Bind environment variables
	manager.bindEnvironment(v)
	
	return v
}

// LoadConfigWithViper loads configuration using a Viper instance
func LoadConfigWithViper(v *viper.Viper) (*Config, error) {
	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Check if it's a file not found error from os package
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
		}
		// File not found is ok, continue with defaults
	}
	
	// Unmarshal to config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Set defaults and validate
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}