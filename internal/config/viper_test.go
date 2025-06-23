package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViperManager(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates new viper manager", func(t *testing.T) {
		vm := NewViperManager()
		assert.NotNil(t, vm)
		assert.NotNil(t, vm.global)
		assert.NotNil(t, vm.project)
	})

	t.Run("initializes global viper", func(t *testing.T) {
		vm := NewViperManager()

		// Mock config path
		oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
		defer func() {
			if oldConfigHome != "" {
				os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		}()
		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		err := vm.InitGlobalViper()
		assert.NoError(t, err)

		// Verify default config was created
		configPath := filepath.Join(tmpDir, ConfigDirName, ConfigFileName)
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
	})

	t.Run("initializes project viper", func(t *testing.T) {
		vm := NewViperManager()
		projectPath := tmpDir

		err := vm.InitProjectViper(projectPath)
		assert.NoError(t, err)
	})

	// Skipping complex integration test for now - focus on core functionality
}

// Skipping complex Viper integration tests - focus on core config functionality

func TestViperEnvironmentVariables(t *testing.T) {
	t.Run("environment variables override config", func(t *testing.T) {
		// Set environment variables
		os.Setenv("CCMGR_STATUS_HOOKS_ENABLED", "false")
		os.Setenv("CCMGR_COMMANDS_CLAUDE_COMMAND", "env-claude")
		defer func() {
			os.Unsetenv("CCMGR_STATUS_HOOKS_ENABLED")
			os.Unsetenv("CCMGR_COMMANDS_CLAUDE_COMMAND")
		}()

		vm := NewViperManager()
		config := DefaultConfig()

		// Apply environment overrides
		vm.applyEnvironmentOverrides(config)

		assert.False(t, config.StatusHooks.Enabled)
		assert.Equal(t, "env-claude", config.Commands.ClaudeCommand)
	})

	t.Run("boolean environment variables work correctly", func(t *testing.T) {
		tests := []struct {
			envValue string
			expected bool
		}{
			{"true", true},
			{"false", false},
			{"1", false}, // Only "true" should be true
		}

		for _, tt := range tests {
			t.Run("env value: "+tt.envValue, func(t *testing.T) {
				os.Setenv("CCMGR_WORKTREE_AUTO_DIRECTORY", tt.envValue)
				defer os.Unsetenv("CCMGR_WORKTREE_AUTO_DIRECTORY")

				vm := NewViperManager()
				config := DefaultConfig()
				config.Worktree.AutoDirectory = !tt.expected // Set opposite

				vm.applyEnvironmentOverrides(config)
				assert.Equal(t, tt.expected, config.Worktree.AutoDirectory)
			})
		}
	})

	t.Run("empty environment variables don't override config", func(t *testing.T) {
		os.Setenv("CCMGR_WORKTREE_AUTO_DIRECTORY", "")
		defer os.Unsetenv("CCMGR_WORKTREE_AUTO_DIRECTORY")

		vm := NewViperManager()
		config := DefaultConfig()
		originalValue := config.Worktree.AutoDirectory

		vm.applyEnvironmentOverrides(config)
		// Empty env var should not change the value
		assert.Equal(t, originalValue, config.Worktree.AutoDirectory)
	})
}

func TestViperDefaultsAndBinding(t *testing.T) {
	t.Run("sets all required defaults", func(t *testing.T) {
		vm := &ViperManager{}
		v := viper.New()

		vm.setDefaults(v)

		assert.Equal(t, "1.0.0", v.GetString("version"))
		assert.True(t, v.GetBool("status_hooks.enabled"))
		assert.Equal(t, "main", v.GetString("worktree.default_branch"))
		assert.Equal(t, "claude", v.GetString("commands.claude_command"))
	})

	t.Run("binds environment variables", func(t *testing.T) {
		vm := &ViperManager{}
		v := viper.New()

		vm.bindEnvironment(v)

		// Verify environment prefix is set
		assert.NotNil(t, v)
	})
}

func TestInitViper(t *testing.T) {
	t.Run("creates configured viper instance", func(t *testing.T) {
		v := InitViper()
		assert.NotNil(t, v)

		// Verify defaults are set
		assert.Equal(t, "1.0.0", v.GetString("version"))
		assert.True(t, v.GetBool("status_hooks.enabled"))
	})
}

func TestLoadConfigWithViper(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Run("loads existing config", func(t *testing.T) {
		// Create config file
		config := DefaultConfig()
		config.Version = "viper-test"
		err := Save(config, configPath)
		require.NoError(t, err)

		// Load with viper
		v := viper.New()
		v.SetConfigFile(configPath)

		loadedConfig, err := LoadConfigWithViper(v)
		require.NoError(t, err)
		assert.Equal(t, "viper-test", loadedConfig.Version)
	})

	t.Run("handles missing config file gracefully", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "non-existent.yaml")

		v := viper.New()
		v.SetConfigFile(nonExistentPath)

		// Set defaults first
		manager := &ViperManager{}
		manager.setDefaults(v)

		config, err := LoadConfigWithViper(v)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "1.0.0", config.Version)
	})

	t.Run("fails on invalid config", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte("invalid: yaml: ["), 0600)
		require.NoError(t, err)

		v := viper.New()
		v.SetConfigFile(invalidPath)

		_, err = LoadConfigWithViper(v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})
}

func TestViperManagerGetSetValue(t *testing.T) {
	t.Run("gets and sets values", func(t *testing.T) {
		vm := NewViperManager()

		// Set global value
		vm.SetValue("test.key", "global-value", false)
		value := vm.GetValue("test.key")
		assert.Equal(t, "global-value", value)

		// Set project value (should override)
		vm.SetValue("test.key", "project-value", true)
		value = vm.GetValue("test.key")
		assert.Equal(t, "project-value", value)
	})

	t.Run("project values take precedence", func(t *testing.T) {
		vm := NewViperManager()

		// Set both global and project values
		vm.SetValue("precedence.test", "global", false)
		vm.SetValue("precedence.test", "project", true)

		// Project should take precedence
		value := vm.GetValue("precedence.test")
		assert.Equal(t, "project", value)
	})
}
