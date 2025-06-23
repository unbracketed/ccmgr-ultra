package ccmgr

import (
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/tui"
)

// systemManager implements the SystemManager interface
type systemManager struct {
	integration *tui.Integration
}

// Status returns the current system status
func (sm *systemManager) Status() SystemStatus {
	internal := sm.integration.GetSystemStatus()
	return convertSystemStatus(internal)
}

// Refresh refreshes all system data
func (sm *systemManager) Refresh() error {
	// Use the integration layer's RefreshData method
	_ = sm.integration.RefreshData()
	return nil
}

// Health returns system health information
func (sm *systemManager) Health() HealthInfo {
	status := sm.integration.GetSystemStatus()

	overall := "healthy"
	if !status.IsHealthy {
		overall = "unhealthy"
	}

	services := map[string]string{
		"tmux":   "running",
		"claude": "running",
		"git":    "running",
	}

	// Check for errors in specific services
	for _, err := range status.Errors {
		if contains(err, "tmux") {
			services["tmux"] = "error"
		}
		if contains(err, "claude") {
			services["claude"] = "error"
		}
		if contains(err, "git") {
			services["git"] = "error"
		}
	}

	return HealthInfo{
		Overall:   overall,
		Services:  services,
		LastCheck: status.LastUpdate,
		Uptime:    time.Since(status.LastUpdate),
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
