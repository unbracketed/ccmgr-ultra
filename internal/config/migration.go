package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Migration represents a configuration migration
type Migration interface {
	Version() string
	Migrate(oldConfig map[string]interface{}) (map[string]interface{}, error)
}

// MigrationFunc is a function-based migration
type MigrationFunc struct {
	version string
	migrate func(map[string]interface{}) (map[string]interface{}, error)
}

// Version returns the migration version
func (m MigrationFunc) Version() string {
	return m.version
}

// Migrate applies the migration
func (m MigrationFunc) Migrate(oldConfig map[string]interface{}) (map[string]interface{}, error) {
	return m.migrate(oldConfig)
}

// MigrationRegistry manages configuration migrations
type MigrationRegistry struct {
	migrations []Migration
}

// NewMigrationRegistry creates a new migration registry
func NewMigrationRegistry() *MigrationRegistry {
	registry := &MigrationRegistry{
		migrations: make([]Migration, 0),
	}
	
	// Register all migrations
	registry.registerMigrations()
	
	return registry
}

// Register adds a migration to the registry
func (r *MigrationRegistry) Register(migration Migration) {
	r.migrations = append(r.migrations, migration)
}

// registerMigrations registers all known migrations
func (r *MigrationRegistry) registerMigrations() {
	// Migration from 0.9.0 to 1.0.0
	r.Register(MigrationFunc{
		version: "1.0.0",
		migrate: migrateV090ToV100,
	})
}

// Migrate applies all necessary migrations to reach target version
func (r *MigrationRegistry) Migrate(configData []byte, currentVersion, targetVersion string) ([]byte, error) {
	// Parse config to map for migrations
	var config map[string]interface{}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config for migration: %w", err)
	}

	// Get migrations to apply
	migrations := r.getMigrationsToApply(currentVersion, targetVersion)
	
	// Apply migrations in order
	for _, migration := range migrations {
		newConfig, err := migration.Migrate(config)
		if err != nil {
			return nil, fmt.Errorf("migration to version %s failed: %w", migration.Version(), err)
		}
		config = newConfig
		
		// Update version in config
		config["version"] = migration.Version()
	}

	// Marshal back to YAML
	migratedData, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	return migratedData, nil
}

// getMigrationsToApply returns migrations needed to reach target version
func (r *MigrationRegistry) getMigrationsToApply(currentVersion, targetVersion string) []Migration {
	var toApply []Migration
	
	// Sort migrations by version
	sorted := make([]Migration, len(r.migrations))
	copy(sorted, r.migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return compareVersions(sorted[i].Version(), sorted[j].Version()) < 0
	})

	// Find migrations to apply
	for _, migration := range sorted {
		if compareVersions(migration.Version(), currentVersion) > 0 &&
			compareVersions(migration.Version(), targetVersion) <= 0 {
			toApply = append(toApply, migration)
		}
	}

	return toApply
}

// DetectConfigVersion detects the version of a configuration file
func DetectConfigVersion(data []byte) (string, error) {
	var config struct {
		Version string `yaml:"version"`
	}
	
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	// If no version field, assume it's a pre-1.0.0 version
	if config.Version == "" {
		return "0.9.0", nil
	}

	return config.Version, nil
}

// NeedsMigration checks if migration is needed
func NeedsMigration(current, target string) bool {
	return compareVersions(current, target) < 0
}

// compareVersions compares two semantic versions
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Pad with zeros if needed
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	// Compare each part
	for i := 0; i < 3; i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])
		
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	return 0
}

// migrateV090ToV100 migrates from version 0.9.0 to 1.0.0
func migrateV090ToV100(oldConfig map[string]interface{}) (map[string]interface{}, error) {
	newConfig := make(map[string]interface{})
	
	// Copy existing values
	for k, v := range oldConfig {
		newConfig[k] = v
	}

	// Migrate old structure to new structure
	// Example: rename fields, restructure nested configs, etc.
	
	// If old config had 'hooks' instead of 'status_hooks'
	if hooks, ok := oldConfig["hooks"]; ok {
		newConfig["status_hooks"] = hooks
		delete(newConfig, "hooks")
	}

	// If old config had 'worktree_config' instead of 'worktree'
	if worktreeConfig, ok := oldConfig["worktree_config"]; ok {
		newConfig["worktree"] = worktreeConfig
		delete(newConfig, "worktree_config")
	}

	// If old config had flat structure for commands
	if _, ok := oldConfig["commands"]; !ok {
		commands := make(map[string]interface{})
		
		if claude, ok := oldConfig["claude_command"]; ok {
			commands["claude_command"] = claude
			delete(newConfig, "claude_command")
		}
		
		if git, ok := oldConfig["git_command"]; ok {
			commands["git_command"] = git
			delete(newConfig, "git_command")
		}
		
		if tmux, ok := oldConfig["tmux_prefix"]; ok {
			commands["tmux_prefix"] = tmux
			delete(newConfig, "tmux_prefix")
		}
		
		if len(commands) > 0 {
			newConfig["commands"] = commands
		}
	}

	// Ensure version is set
	newConfig["version"] = "1.0.0"

	return newConfig, nil
}

// MigrateConfigFile migrates a config file in place
func MigrateConfigFile(path string, targetVersion string) error {
	// Backup original file
	if err := BackupConfig(path); err != nil {
		return fmt.Errorf("failed to backup config: %w", err)
	}

	// Read current config
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Detect current version
	currentVersion, err := DetectConfigVersion(data)
	if err != nil {
		return fmt.Errorf("failed to detect config version: %w", err)
	}

	// Check if migration needed
	if !NeedsMigration(currentVersion, targetVersion) {
		return nil // already at target version
	}

	// Apply migrations
	registry := NewMigrationRegistry()
	migratedData, err := registry.Migrate(data, currentVersion, targetVersion)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Write migrated config
	if err := os.WriteFile(path, migratedData, 0600); err != nil {
		return fmt.Errorf("failed to write migrated config: %w", err)
	}

	return nil
}

// ExportForMigration exports config to a migration-friendly format
func ExportForMigration(config *Config) (map[string]interface{}, error) {
	// Marshal to JSON first to get a clean map
	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

// ImportFromMigration imports config from a migration-friendly format
func ImportFromMigration(data map[string]interface{}) (*Config, error) {
	// Marshal map to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %w", err)
	}

	// Unmarshal to Config struct
	var config Config
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to config: %w", err)
	}

	// Check for explicitly invalid values before setting defaults
	if version, ok := data["version"]; ok && version == "" {
		return nil, fmt.Errorf("config validation failed: version is required")
	}

	// Set defaults and validate
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}