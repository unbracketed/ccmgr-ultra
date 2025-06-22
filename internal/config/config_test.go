package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	t.Run("valid config passes validation", func(t *testing.T) {
		config := DefaultConfig()
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing version fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.Version = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("invalid hook timeout fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.StatusHooks.IdleHook.Timeout = -1
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout cannot be negative")
	})

	t.Run("hook timeout exceeding max fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.StatusHooks.IdleHook.Timeout = 400
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout cannot exceed 300 seconds")
	})

	t.Run("empty default branch fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.Worktree.DefaultBranch = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default branch is required")
	})

	t.Run("invalid directory pattern fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.Worktree.DirectoryPattern = "invalid-pattern"
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must contain template variables")
	})

	t.Run("empty shortcut key fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.Shortcuts[""] = "some_action"
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shortcut key cannot be empty")
	})

	t.Run("empty shortcut action fails validation", func(t *testing.T) {
		config := DefaultConfig()
		config.Shortcuts["x"] = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shortcut action for key 'x' cannot be empty")
	})
}

func TestHookConfigValidation(t *testing.T) {
	t.Run("enabled hook without script fails validation", func(t *testing.T) {
		hook := HookConfig{
			Enabled: true,
			Script:  "",
		}
		err := hook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "script path is required when hook is enabled")
	})

	t.Run("disabled hook without script passes validation", func(t *testing.T) {
		hook := HookConfig{
			Enabled: false,
			Script:  "",
		}
		err := hook.Validate()
		assert.NoError(t, err)
	})

	t.Run("zero timeout gets default value", func(t *testing.T) {
		hook := HookConfig{
			Enabled: true,
			Script:  "/path/to/script",
			Timeout: 0,
		}
		err := hook.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 30, hook.Timeout)
	})
}

func TestCommandsConfigValidation(t *testing.T) {
	t.Run("empty claude command fails validation", func(t *testing.T) {
		config := CommandsConfig{
			ClaudeCommand: "",
			GitCommand:    "git",
			TmuxPrefix:    "ccmgr",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "claude command is required")
	})

	t.Run("environment variable with equals in key fails validation", func(t *testing.T) {
		config := CommandsConfig{
			ClaudeCommand: "claude",
			GitCommand:    "git",
			TmuxPrefix:    "ccmgr",
			Environment: map[string]string{
				"KEY=WITH=EQUALS": "value",
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot contain '='")
	})

	t.Run("empty environment variable value fails validation", func(t *testing.T) {
		config := CommandsConfig{
			ClaudeCommand: "claude",
			GitCommand:    "git",
			TmuxPrefix:    "ccmgr",
			Environment: map[string]string{
				"KEY": "",
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot have empty value")
	})
}

func TestConfigFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Run("save and load config", func(t *testing.T) {
		original := DefaultConfig()
		original.Version = "test-version"

		// Save configuration
		err := Save(original, configPath)
		require.NoError(t, err)

		// Load configuration
		loaded, err := LoadFromPath(configPath)
		require.NoError(t, err)

		assert.Equal(t, original.Version, loaded.Version)
		assert.Equal(t, original.StatusHooks.Enabled, loaded.StatusHooks.Enabled)
		assert.Equal(t, original.Worktree.DefaultBranch, loaded.Worktree.DefaultBranch)
		assert.True(t, loaded.LastModified.After(time.Time{}))
	})

	t.Run("load non-existent config fails", func(t *testing.T) {
		_, err := LoadFromPath("/non/existent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("load invalid YAML fails", func(t *testing.T) {
		invalidYAML := "invalid: yaml: content: ["
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte(invalidYAML), 0600)
		require.NoError(t, err)

		_, err = LoadFromPath(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("LoadOrCreate creates default when missing", func(t *testing.T) {
		newPath := filepath.Join(tmpDir, "new-config.yaml")

		config, err := LoadOrCreate(newPath)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", config.Version)

		// Verify file was created
		_, err = os.Stat(newPath)
		assert.NoError(t, err)
	})

	t.Run("LoadOrCreate loads existing config", func(t *testing.T) {
		existingConfig := DefaultConfig()
		existingConfig.Version = "existing-version"

		existingPath := filepath.Join(tmpDir, "existing-config.yaml")
		err := Save(existingConfig, existingPath)
		require.NoError(t, err)

		loaded, err := LoadOrCreate(existingPath)
		require.NoError(t, err)
		assert.Equal(t, "existing-version", loaded.Version)
	})
}

func TestConfigPaths(t *testing.T) {
	t.Run("GetConfigPath uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		oldXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

		os.Setenv("XDG_CONFIG_HOME", "/custom/config")
		path := GetConfigPath()
		assert.Equal(t, "/custom/config/ccmgr-ultra", path)
	})

	t.Run("GetProjectConfigPath returns correct path", func(t *testing.T) {
		path := GetProjectConfigPath("/path/to/project")
		expected := "/path/to/project/.ccmgr-ultra/config.yaml"
		assert.Equal(t, expected, path)
	})

	t.Run("GetGlobalConfigPath returns correct path", func(t *testing.T) {
		globalPath := GetGlobalConfigPath()
		assert.True(t, strings.HasSuffix(globalPath, "config.yaml"))
		assert.Contains(t, globalPath, "ccmgr-ultra")
	})
}

func TestConfigMerging(t *testing.T) {
	t.Run("merge configs with project overrides", func(t *testing.T) {
		global := DefaultConfig()
		global.Version = "global-version"
		global.Commands.ClaudeCommand = "global-claude"

		project := &Config{
			Version: "project-version",
			Commands: CommandsConfig{
				ClaudeCommand: "project-claude",
				GitCommand:    "project-git",
			},
			Shortcuts: map[string]string{
				"x": "custom_action",
			},
		}

		merged := MergeConfigs(global, project)

		assert.Equal(t, "project-version", merged.Version)
		assert.Equal(t, "project-claude", merged.Commands.ClaudeCommand)
		assert.Equal(t, "project-git", merged.Commands.GitCommand)
		assert.Equal(t, "custom_action", merged.Shortcuts["x"])
		// Global values should be preserved when not overridden
		assert.Equal(t, global.Worktree.DefaultBranch, merged.Worktree.DefaultBranch)
	})

	t.Run("merge with nil global returns project", func(t *testing.T) {
		project := DefaultConfig()
		merged := MergeConfigs(nil, project)
		assert.Equal(t, project, merged)
	})

	t.Run("merge with nil project returns global", func(t *testing.T) {
		global := DefaultConfig()
		merged := MergeConfigs(global, nil)
		assert.Equal(t, global, merged)
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Run("DefaultConfig returns valid configuration", func(t *testing.T) {
		config := DefaultConfig()
		err := config.Validate()
		assert.NoError(t, err)

		assert.Equal(t, "1.0.0", config.Version)
		assert.True(t, config.StatusHooks.Enabled)
		assert.Equal(t, "main", config.Worktree.DefaultBranch)
		assert.Equal(t, "claude", config.Commands.ClaudeCommand)
		assert.NotEmpty(t, config.Shortcuts)
	})

	t.Run("SetDefaults fills missing values", func(t *testing.T) {
		config := &Config{}
		config.SetDefaults()

		assert.Equal(t, "1.0.0", config.Version)
		assert.NotEmpty(t, config.Shortcuts)
		assert.Equal(t, "claude", config.Commands.ClaudeCommand)
	})

	t.Run("DefaultShortcuts returns expected shortcuts", func(t *testing.T) {
		shortcuts := DefaultShortcuts()
		assert.Equal(t, "new_worktree", shortcuts["n"])
		assert.Equal(t, "quit", shortcuts["q"])
		assert.Len(t, shortcuts, 7)
	})
}

func TestExpandPath(t *testing.T) {
	t.Run("expands tilde to home directory", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		expanded := ExpandPath("~/test/path")
		expected := filepath.Join(home, "test/path")
		assert.Equal(t, expected, expanded)
	})

	t.Run("expands environment variables", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		expanded := ExpandPath("$TEST_VAR/path")
		assert.Equal(t, "test_value/path", expanded)
	})

	t.Run("returns empty string unchanged", func(t *testing.T) {
		expanded := ExpandPath("")
		assert.Equal(t, "", expanded)
	})
}

func TestBackupConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Run("backup creates backup file", func(t *testing.T) {
		// Create original config
		config := DefaultConfig()
		err := Save(config, configPath)
		require.NoError(t, err)

		// Create backup
		err = BackupConfig(configPath)
		require.NoError(t, err)

		// Check backup exists
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		var backupFound bool
		for _, file := range files {
			if strings.Contains(file.Name(), "config.yaml.backup.") {
				backupFound = true
				break
			}
		}
		assert.True(t, backupFound, "backup file should be created")
	})

	t.Run("backup non-existent file does nothing", func(t *testing.T) {
		err := BackupConfig("/non/existent/path")
		assert.NoError(t, err)
	})
}

func TestValidateConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("validates correct config file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid.yaml")
		config := DefaultConfig()
		err := Save(config, configPath)
		require.NoError(t, err)

		err = ValidateConfigFile(configPath)
		assert.NoError(t, err)
	})

	t.Run("fails on invalid YAML", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte("invalid: yaml: ["), 0600)
		require.NoError(t, err)

		err = ValidateConfigFile(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid YAML syntax")
	})

	t.Run("fails on validation errors", func(t *testing.T) {
		invalidConfigPath := filepath.Join(tmpDir, "invalid-config.yaml")
		invalidConfig := `
version: ""
status_hooks:
  enabled: true
`
		err := os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0600)
		require.NoError(t, err)

		err = ValidateConfigFile(invalidConfigPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

func TestImportExportConfig(t *testing.T) {
	t.Run("export and import roundtrip", func(t *testing.T) {
		original := DefaultConfig()
		original.Version = "test-export"

		var buf bytes.Buffer
		err := ExportConfig(original, &buf)
		require.NoError(t, err)

		imported, err := ImportConfig(&buf)
		require.NoError(t, err)

		assert.Equal(t, original.Version, imported.Version)
		assert.Equal(t, original.StatusHooks.Enabled, imported.StatusHooks.Enabled)
	})

	t.Run("import invalid config fails", func(t *testing.T) {
		invalidYAML := strings.NewReader("invalid: yaml: [")
		_, err := ImportConfig(invalidYAML)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode config")
	})
}