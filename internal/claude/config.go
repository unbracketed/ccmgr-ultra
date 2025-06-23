package claude

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// ConfigAdapter adapts the main config.ClaudeConfig to our internal ProcessConfig
type ConfigAdapter struct {
	mainConfig *config.ClaudeConfig
}

// NewConfigAdapter creates a new config adapter
func NewConfigAdapter(mainConfig *config.ClaudeConfig) *ConfigAdapter {
	return &ConfigAdapter{
		mainConfig: mainConfig,
	}
}

// ToProcessConfig converts ClaudeConfig to ProcessConfig
func (a *ConfigAdapter) ToProcessConfig() (*ProcessConfig, error) {
	if a.mainConfig == nil {
		return nil, fmt.Errorf("main config is nil")
	}

	processConfig := &ProcessConfig{
		PollInterval:             a.mainConfig.PollInterval,
		LogPaths:                 make([]string, len(a.mainConfig.LogPaths)),
		StatePatterns:            make(map[string]string),
		MaxProcesses:             a.mainConfig.MaxProcesses,
		CleanupInterval:          a.mainConfig.CleanupInterval,
		EnableLogParsing:         a.mainConfig.EnableLogParsing,
		EnableResourceMonitoring: a.mainConfig.EnableResourceMonitoring,
		StateTimeout:             a.mainConfig.StateTimeout,
		StartupTimeout:           a.mainConfig.StartupTimeout,
	}

	// Copy log paths
	copy(processConfig.LogPaths, a.mainConfig.LogPaths)

	// Copy state patterns
	for key, value := range a.mainConfig.StatePatterns {
		processConfig.StatePatterns[key] = value
	}

	// Compile patterns
	if err := processConfig.CompilePatterns(); err != nil {
		return nil, fmt.Errorf("failed to compile patterns: %w", err)
	}

	return processConfig, nil
}

// FromProcessConfig updates ClaudeConfig from ProcessConfig
func (a *ConfigAdapter) FromProcessConfig(processConfig *ProcessConfig) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if processConfig == nil {
		return fmt.Errorf("process config is nil")
	}

	a.mainConfig.PollInterval = processConfig.PollInterval
	a.mainConfig.MaxProcesses = processConfig.MaxProcesses
	a.mainConfig.CleanupInterval = processConfig.CleanupInterval
	a.mainConfig.EnableLogParsing = processConfig.EnableLogParsing
	a.mainConfig.EnableResourceMonitoring = processConfig.EnableResourceMonitoring
	a.mainConfig.StateTimeout = processConfig.StateTimeout
	a.mainConfig.StartupTimeout = processConfig.StartupTimeout

	// Copy log paths
	a.mainConfig.LogPaths = make([]string, len(processConfig.LogPaths))
	copy(a.mainConfig.LogPaths, processConfig.LogPaths)

	// Copy state patterns
	a.mainConfig.StatePatterns = make(map[string]string)
	for key, value := range processConfig.StatePatterns {
		a.mainConfig.StatePatterns[key] = value
	}

	return nil
}

// ValidateAndSetDefaults validates and sets defaults for the configuration
func (a *ConfigAdapter) ValidateAndSetDefaults() error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	// Set defaults first
	a.mainConfig.SetDefaults()

	// Then validate
	return a.mainConfig.Validate()
}

// GetMainConfig returns the underlying main config
func (a *ConfigAdapter) GetMainConfig() *config.ClaudeConfig {
	return a.mainConfig
}

// IsEnabled returns whether Claude monitoring is enabled
func (a *ConfigAdapter) IsEnabled() bool {
	return a.mainConfig != nil && a.mainConfig.Enabled
}

// ShouldIntegrateTmux returns whether tmux integration is enabled
func (a *ConfigAdapter) ShouldIntegrateTmux() bool {
	return a.mainConfig != nil && a.mainConfig.IntegrateTmux
}

// ShouldIntegrateWorktrees returns whether worktree integration is enabled
func (a *ConfigAdapter) ShouldIntegrateWorktrees() bool {
	return a.mainConfig != nil && a.mainConfig.IntegrateWorktrees
}

// UpdatePattern updates a specific state pattern
func (a *ConfigAdapter) UpdatePattern(state string, pattern string) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if state == "" {
		return fmt.Errorf("state cannot be empty")
	}

	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Validate the pattern by trying to compile it
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	if a.mainConfig.StatePatterns == nil {
		a.mainConfig.StatePatterns = make(map[string]string)
	}

	a.mainConfig.StatePatterns[state] = pattern
	return nil
}

// RemovePattern removes a state pattern
func (a *ConfigAdapter) RemovePattern(state string) {
	if a.mainConfig != nil && a.mainConfig.StatePatterns != nil {
		delete(a.mainConfig.StatePatterns, state)
	}
}

// AddLogPath adds a new log path to monitor
func (a *ConfigAdapter) AddLogPath(path string) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check if path already exists
	for _, existing := range a.mainConfig.LogPaths {
		if existing == path {
			return fmt.Errorf("path %s already exists", path)
		}
	}

	a.mainConfig.LogPaths = append(a.mainConfig.LogPaths, path)
	return nil
}

// RemoveLogPath removes a log path
func (a *ConfigAdapter) RemoveLogPath(path string) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	for i, existing := range a.mainConfig.LogPaths {
		if existing == path {
			a.mainConfig.LogPaths = append(a.mainConfig.LogPaths[:i], a.mainConfig.LogPaths[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("path %s not found", path)
}

// SetPollInterval updates the polling interval
func (a *ConfigAdapter) SetPollInterval(interval time.Duration) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if interval < time.Second {
		return fmt.Errorf("poll interval must be at least 1 second")
	}

	if interval > 60*time.Second {
		return fmt.Errorf("poll interval cannot exceed 60 seconds")
	}

	a.mainConfig.PollInterval = interval
	return nil
}

// SetMaxProcesses updates the maximum number of processes to track
func (a *ConfigAdapter) SetMaxProcesses(max int) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if max < 1 {
		return fmt.Errorf("max processes must be at least 1")
	}

	if max > 100 {
		return fmt.Errorf("max processes cannot exceed 100")
	}

	a.mainConfig.MaxProcesses = max
	return nil
}

// SetCleanupInterval updates the cleanup interval
func (a *ConfigAdapter) SetCleanupInterval(interval time.Duration) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if interval < time.Minute {
		return fmt.Errorf("cleanup interval must be at least 1 minute")
	}

	a.mainConfig.CleanupInterval = interval
	return nil
}

// SetStateTimeout updates the state timeout
func (a *ConfigAdapter) SetStateTimeout(timeout time.Duration) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	if timeout < 5*time.Second {
		return fmt.Errorf("state timeout must be at least 5 seconds")
	}

	a.mainConfig.StateTimeout = timeout
	return nil
}

// EnableFeature enables or disables a specific feature
func (a *ConfigAdapter) EnableFeature(feature string, enabled bool) error {
	if a.mainConfig == nil {
		return fmt.Errorf("main config is nil")
	}

	switch feature {
	case "monitoring":
		a.mainConfig.Enabled = enabled
	case "log_parsing":
		a.mainConfig.EnableLogParsing = enabled
	case "resource_monitoring":
		a.mainConfig.EnableResourceMonitoring = enabled
	case "tmux_integration":
		a.mainConfig.IntegrateTmux = enabled
	case "worktree_integration":
		a.mainConfig.IntegrateWorktrees = enabled
	default:
		return fmt.Errorf("unknown feature: %s", feature)
	}

	return nil
}

// GetFeatureStatus returns the status of a specific feature
func (a *ConfigAdapter) GetFeatureStatus(feature string) (bool, error) {
	if a.mainConfig == nil {
		return false, fmt.Errorf("main config is nil")
	}

	switch feature {
	case "monitoring":
		return a.mainConfig.Enabled, nil
	case "log_parsing":
		return a.mainConfig.EnableLogParsing, nil
	case "resource_monitoring":
		return a.mainConfig.EnableResourceMonitoring, nil
	case "tmux_integration":
		return a.mainConfig.IntegrateTmux, nil
	case "worktree_integration":
		return a.mainConfig.IntegrateWorktrees, nil
	default:
		return false, fmt.Errorf("unknown feature: %s", feature)
	}
}

// Clone creates a deep copy of the configuration
func (a *ConfigAdapter) Clone() *ConfigAdapter {
	if a.mainConfig == nil {
		return &ConfigAdapter{}
	}

	cloned := &config.ClaudeConfig{
		Enabled:                  a.mainConfig.Enabled,
		PollInterval:             a.mainConfig.PollInterval,
		MaxProcesses:             a.mainConfig.MaxProcesses,
		CleanupInterval:          a.mainConfig.CleanupInterval,
		StateTimeout:             a.mainConfig.StateTimeout,
		StartupTimeout:           a.mainConfig.StartupTimeout,
		EnableLogParsing:         a.mainConfig.EnableLogParsing,
		EnableResourceMonitoring: a.mainConfig.EnableResourceMonitoring,
		IntegrateTmux:            a.mainConfig.IntegrateTmux,
		IntegrateWorktrees:       a.mainConfig.IntegrateWorktrees,
	}

	// Clone log paths
	if a.mainConfig.LogPaths != nil {
		cloned.LogPaths = make([]string, len(a.mainConfig.LogPaths))
		copy(cloned.LogPaths, a.mainConfig.LogPaths)
	}

	// Clone state patterns
	if a.mainConfig.StatePatterns != nil {
		cloned.StatePatterns = make(map[string]string)
		for key, value := range a.mainConfig.StatePatterns {
			cloned.StatePatterns[key] = value
		}
	}

	return &ConfigAdapter{mainConfig: cloned}
}

// String returns a string representation of the configuration
func (a *ConfigAdapter) String() string {
	if a.mainConfig == nil {
		return "ClaudeConfig{nil}"
	}

	return fmt.Sprintf("ClaudeConfig{Enabled: %t, PollInterval: %s, MaxProcesses: %d, LogPaths: %d, StatePatterns: %d}",
		a.mainConfig.Enabled,
		a.mainConfig.PollInterval.String(),
		a.mainConfig.MaxProcesses,
		len(a.mainConfig.LogPaths),
		len(a.mainConfig.StatePatterns),
	)
}
