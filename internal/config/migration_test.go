package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDetectConfigVersion(t *testing.T) {
	t.Run("detects version from config", func(t *testing.T) {
		configData := `
version: "1.2.3"
status_hooks:
  enabled: true
`
		version, err := DetectConfigVersion([]byte(configData))
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("returns default version for config without version", func(t *testing.T) {
		configData := `
status_hooks:
  enabled: true
`
		version, err := DetectConfigVersion([]byte(configData))
		require.NoError(t, err)
		assert.Equal(t, "0.9.0", version)
	})

	t.Run("fails on invalid YAML", func(t *testing.T) {
		invalidYAML := `invalid: yaml: [`
		_, err := DetectConfigVersion([]byte(invalidYAML))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config")
	})
}

func TestNeedsMigration(t *testing.T) {
	tests := []struct {
		current string
		target  string
		needed  bool
	}{
		{"0.9.0", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+" to "+tt.target, func(t *testing.T) {
			result := NeedsMigration(tt.current, tt.target)
			assert.Equal(t, tt.needed, result)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0", "1.0.0", 0},
		{"1", "1.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMigrationRegistry(t *testing.T) {
	t.Run("creates registry with migrations", func(t *testing.T) {
		registry := NewMigrationRegistry()
		assert.NotNil(t, registry)
		assert.NotEmpty(t, registry.migrations)
	})

	t.Run("registers custom migration", func(t *testing.T) {
		registry := &MigrationRegistry{}

		migration := MigrationFunc{
			version: "1.1.0",
			migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				return config, nil
			},
		}

		registry.Register(migration)
		assert.Len(t, registry.migrations, 1)
		assert.Equal(t, "1.1.0", registry.migrations[0].Version())
	})

	t.Run("getMigrationsToApply returns correct migrations", func(t *testing.T) {
		registry := &MigrationRegistry{}

		// Add test migrations
		registry.Register(MigrationFunc{
			version: "1.0.0",
			migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				return config, nil
			},
		})
		registry.Register(MigrationFunc{
			version: "1.1.0",
			migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				return config, nil
			},
		})
		registry.Register(MigrationFunc{
			version: "1.2.0",
			migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				return config, nil
			},
		})

		// Test getting migrations from 0.9.0 to 1.1.0
		migrations := registry.getMigrationsToApply("0.9.0", "1.1.0")
		assert.Len(t, migrations, 2)
		assert.Equal(t, "1.0.0", migrations[0].Version())
		assert.Equal(t, "1.1.0", migrations[1].Version())
	})
}

func TestMigrateV090ToV100(t *testing.T) {
	t.Run("migrates hooks to status_hooks", func(t *testing.T) {
		oldConfig := map[string]interface{}{
			"version": "0.9.0",
			"hooks": map[string]interface{}{
				"enabled": true,
			},
		}

		newConfig, err := migrateV090ToV100(oldConfig)
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", newConfig["version"])
		assert.Contains(t, newConfig, "status_hooks")
		assert.NotContains(t, newConfig, "hooks")
	})

	t.Run("migrates worktree_config to worktree", func(t *testing.T) {
		oldConfig := map[string]interface{}{
			"version": "0.9.0",
			"worktree_config": map[string]interface{}{
				"auto_directory": true,
			},
		}

		newConfig, err := migrateV090ToV100(oldConfig)
		require.NoError(t, err)

		assert.Contains(t, newConfig, "worktree")
		assert.NotContains(t, newConfig, "worktree_config")
	})

	t.Run("migrates flat command structure", func(t *testing.T) {
		oldConfig := map[string]interface{}{
			"version":        "0.9.0",
			"claude_command": "old-claude",
			"git_command":    "old-git",
			"tmux_prefix":    "old-tmux",
		}

		newConfig, err := migrateV090ToV100(oldConfig)
		require.NoError(t, err)

		commands, ok := newConfig["commands"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "old-claude", commands["claude_command"])
		assert.Equal(t, "old-git", commands["git_command"])
		assert.Equal(t, "old-tmux", commands["tmux_prefix"])

		// Original flat keys should be removed
		assert.NotContains(t, newConfig, "claude_command")
		assert.NotContains(t, newConfig, "git_command")
		assert.NotContains(t, newConfig, "tmux_prefix")
	})
}

func TestMigrationRegistryIntegration(t *testing.T) {
	t.Run("applies migration chain", func(t *testing.T) {
		registry := NewMigrationRegistry()

		// Create old-style config
		oldConfigData := `
version: "0.9.0"
hooks:
  enabled: true
  idle:
    enabled: true
    script: "/path/to/idle.sh"
claude_command: "claude-old"
`

		migratedData, err := registry.Migrate([]byte(oldConfigData), "0.9.0", "1.0.0")
		require.NoError(t, err)

		// Parse migrated config
		var migratedConfig map[string]interface{}
		err = yaml.Unmarshal(migratedData, &migratedConfig)
		require.NoError(t, err)

		// Verify migration
		assert.Equal(t, "1.0.0", migratedConfig["version"])
		assert.Contains(t, migratedConfig, "status_hooks")
		assert.NotContains(t, migratedConfig, "hooks")
	})

	t.Run("no migration needed returns original", func(t *testing.T) {
		registry := NewMigrationRegistry()

		configData := `version: "1.0.0"`

		migratedData, err := registry.Migrate([]byte(configData), "1.0.0", "1.0.0")
		require.NoError(t, err)

		// Should return original data unchanged
		var config map[string]interface{}
		err = yaml.Unmarshal(migratedData, &config)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", config["version"])
	})
}

func TestMigrateConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Run("migrates config file in place", func(t *testing.T) {
		// Create old config file
		oldConfigData := `
version: "0.9.0"
hooks:
  enabled: true
claude_command: "claude-old"
`
		err := os.WriteFile(configPath, []byte(oldConfigData), 0600)
		require.NoError(t, err)

		// Migrate
		err = MigrateConfigFile(configPath, "1.0.0")
		require.NoError(t, err)

		// Verify migration
		migratedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var config map[string]interface{}
		err = yaml.Unmarshal(migratedData, &config)
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", config["version"])
		assert.Contains(t, config, "status_hooks")

		// Verify backup was created
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		var backupFound bool
		for _, file := range files {
			if filepath.Ext(file.Name()) == ".backup" ||
				(len(file.Name()) > len("config.yaml.backup.") &&
					file.Name()[:len("config.yaml.backup.")] == "config.yaml.backup.") {
				backupFound = true
				break
			}
		}
		assert.True(t, backupFound, "backup file should be created")
	})

	t.Run("skips migration when not needed", func(t *testing.T) {
		// Create current version config
		currentConfigData := `version: "1.0.0"`
		currentPath := filepath.Join(tmpDir, "current.yaml")
		err := os.WriteFile(currentPath, []byte(currentConfigData), 0600)
		require.NoError(t, err)

		// Try to migrate
		err = MigrateConfigFile(currentPath, "1.0.0")
		require.NoError(t, err)

		// Config should be unchanged
		data, err := os.ReadFile(currentPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), `version: "1.0.0"`)
	})

	t.Run("handles non-existent config file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "non-existent.yaml")
		err := MigrateConfigFile(nonExistentPath, "1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})
}

func TestExportImportForMigration(t *testing.T) {
	t.Run("export and import for migration", func(t *testing.T) {
		config := DefaultConfig()
		config.Version = "test-migration"

		// Export to migration format
		exported, err := ExportForMigration(config)
		require.NoError(t, err)
		assert.Equal(t, "test-migration", exported["version"])

		// Import from migration format
		imported, err := ImportFromMigration(exported)
		require.NoError(t, err)
		assert.Equal(t, "test-migration", imported.Version)
		assert.Equal(t, config.StatusHooks.Enabled, imported.StatusHooks.Enabled)
	})

	t.Run("import invalid migration data fails", func(t *testing.T) {
		invalidData := map[string]interface{}{
			"version": "", // Invalid: empty version
			"worktree": map[string]interface{}{
				"default_branch": "", // Invalid: empty default branch
			},
		}

		_, err := ImportFromMigration(invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config validation failed")
	})
}
