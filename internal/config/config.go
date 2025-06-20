package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config paths
const (
	ConfigFileName = "config.yaml"
	ConfigDirName  = "ccmgr-ultra"
)

// Load loads configuration from the specified path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults for missing values
	config.SetDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// Save saves configuration to the specified path
func Save(config *Config, path string) error {
	// Update last modified time
	config.LastModified = time.Now()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML with proper formatting
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temporary file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Rename to actual path (atomic on most systems)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // cleanup on error
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// LoadOrCreate loads configuration or creates default if not exists
func LoadOrCreate(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		config := DefaultConfig()
		if err := Save(config, path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	return Load(path)
}

// GetConfigPath returns the user config directory path
func GetConfigPath() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, ConfigDirName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return filepath.Join(".", ".config", ConfigDirName)
	}

	return filepath.Join(home, ".config", ConfigDirName)
}

// GetProjectConfigPath returns project-specific config path
func GetProjectConfigPath(projectPath string) string {
	return filepath.Join(projectPath, ".ccmgr-ultra", ConfigFileName)
}

// GetGlobalConfigPath returns global config file path
func GetGlobalConfigPath() string {
	return filepath.Join(GetConfigPath(), ConfigFileName)
}

// MergeConfigs merges project config over global config
func MergeConfigs(global, project *Config) *Config {
	if global == nil {
		return project
	}
	if project == nil {
		return global
	}

	// Create a copy of global config
	merged := *global

	// Override with project values
	if project.Version != "" {
		merged.Version = project.Version
	}

	// Merge status hooks
	if project.StatusHooks.Enabled {
		merged.StatusHooks = project.StatusHooks
	}

	// Merge worktree config
	if project.Worktree.DefaultBranch != "" {
		merged.Worktree = project.Worktree
	}

	// Merge shortcuts (project additions/overrides)
	if merged.Shortcuts == nil {
		merged.Shortcuts = make(map[string]string)
	}
	for key, value := range project.Shortcuts {
		merged.Shortcuts[key] = value
	}

	// Merge commands
	if project.Commands.ClaudeCommand != "" {
		merged.Commands.ClaudeCommand = project.Commands.ClaudeCommand
	}
	if project.Commands.GitCommand != "" {
		merged.Commands.GitCommand = project.Commands.GitCommand
	}
	if project.Commands.TmuxPrefix != "" {
		merged.Commands.TmuxPrefix = project.Commands.TmuxPrefix
	}

	// Merge environment variables
	if merged.Commands.Environment == nil {
		merged.Commands.Environment = make(map[string]string)
	}
	for key, value := range project.Commands.Environment {
		merged.Commands.Environment[key] = value
	}

	// Use the most recent modification time
	if project.LastModified.After(merged.LastModified) {
		merged.LastModified = project.LastModified
	}

	return &merged
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	config := &Config{
		Version:      "1.0.0",
		LastModified: time.Now(),
	}

	// Set all defaults using the SetDefaults method
	config.SetDefaults()

	return config
}

// LoadWithViper loads configuration using Viper
func LoadWithViper(configPath string) (*Config, error) {
	v := viper.New()
	
	// Set config file
	v.SetConfigFile(configPath)
	
	// Read config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default
			config := DefaultConfig()
			if err := Save(config, configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
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

// Watch watches configuration file for changes
func Watch(configPath string, onChange func(*Config)) error {
	v := viper.New()
	v.SetConfigFile(configPath)

	// Initial read
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Watch for changes
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		var config Config
		if err := v.Unmarshal(&config); err == nil {
			config.SetDefaults()
			if err := config.Validate(); err == nil {
				onChange(&config)
			}
		}
	})

	return nil
}

// CopyDefaultTemplate copies the default configuration template to destination
func CopyDefaultTemplate(destPath string) error {
	config := DefaultConfig()
	
	// Read template from embedded file or generate from default config
	templateData, err := generateTemplate(config)
	if err != nil {
		return fmt.Errorf("failed to generate template: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write template file
	if err := os.WriteFile(destPath, templateData, 0600); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// generateTemplate generates a YAML template with comments
func generateTemplate(config *Config) ([]byte, error) {
	// For now, just marshal the config
	// In a real implementation, this would include helpful comments
	return yaml.Marshal(config)
}

// ExpandPath expands ~ and environment variables in paths
func ExpandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand ~ to home directory
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Expand environment variables
	return os.ExpandEnv(path)
}

// BackupConfig creates a backup of the configuration file
func BackupConfig(configPath string) error {
	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // nothing to backup
	}

	// Read original file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Create backup filename with timestamp
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, time.Now().Format("20060102-150405"))

	// Write backup
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// ValidateConfigFile validates a configuration file without loading it
func ValidateConfigFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// ExportConfig exports configuration to a writer
func ExportConfig(config *Config, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return encoder.Close()
}

// ImportConfig imports configuration from a reader
func ImportConfig(r io.Reader) (*Config, error) {
	decoder := yaml.NewDecoder(r)
	
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Set defaults and validate
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}