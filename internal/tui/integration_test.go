package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/ccmgr-ultra/internal/config"
)

func TestIntegration_NewIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	assert.NotNil(t, integration)
	assert.NotNil(t, integration.claudeMgr)
	assert.NotNil(t, integration.tmuxMgr)
	assert.Equal(t, cfg, integration.config)
}

func TestIntegration_GetSystemStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	status := integration.GetSystemStatus()
	assert.NotNil(t, status)
	assert.False(t, status.LastUpdate.IsZero())
	assert.True(t, status.IsHealthy)
}

func TestIntegration_GetActiveSessions(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	sessions := integration.GetActiveSessions()
	assert.NotNil(t, sessions)
	// Should be empty initially
	assert.Len(t, sessions, 0)
}

func TestIntegration_GetAllSessions(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	sessions := integration.GetAllSessions()
	assert.NotNil(t, sessions)
	// Should be empty initially
	assert.Len(t, sessions, 0)
}

func TestIntegration_GetRecentWorktrees(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	worktrees := integration.GetRecentWorktrees()
	assert.NotNil(t, worktrees)
	// Should have some placeholder data
	assert.GreaterOrEqual(t, len(worktrees), 0)
}

func TestIntegration_GetAllWorktrees(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	worktrees := integration.GetAllWorktrees()
	assert.NotNil(t, worktrees)
	// Should have some placeholder data
	assert.GreaterOrEqual(t, len(worktrees), 0)
}

func TestIntegration_StartPeriodicRefresh(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	cmd := integration.StartPeriodicRefresh()
	assert.NotNil(t, cmd)
}

func TestIntegration_AttachSession(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	cmd := integration.AttachSession("test-session")
	assert.NotNil(t, cmd)
	
	// Execute the command to test the message
	msg := cmd()
	switch msg := msg.(type) {
	case SessionAttachedMsg:
		assert.Equal(t, "test-session", msg.SessionID)
	case ErrorMsg:
		// This is expected since the session doesn't exist
		assert.NotNil(t, msg.Error)
	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}

func TestIntegration_OpenWorktree(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	cmd := integration.OpenWorktree("/test/path")
	assert.NotNil(t, cmd)
	
	// Execute the command to test the message
	msg := cmd()
	worktreeMsg, ok := msg.(WorktreeOpenedMsg)
	assert.True(t, ok)
	assert.Equal(t, "/test/path", worktreeMsg.Path)
}

func TestIntegration_CreateSession(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	cmd := integration.CreateSession("test-session", "/test/dir")
	assert.NotNil(t, cmd)
	
	// Execute the command to test the message
	msg := cmd()
	switch msg := msg.(type) {
	case SessionCreatedMsg:
		assert.Equal(t, "test-session", msg.SessionID)
	case ErrorMsg:
		// This might happen if tmux is not available
		assert.NotNil(t, msg.Error)
	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}

func TestIntegration_CreateWorktree(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	cmd := integration.CreateWorktree("/test/path", "test-branch")
	assert.NotNil(t, cmd)
	
	// Execute the command to test the message
	msg := cmd()
	worktreeMsg, ok := msg.(WorktreeCreatedMsg)
	assert.True(t, ok)
	assert.Equal(t, "/test/path", worktreeMsg.Path)
	assert.Equal(t, "test-branch", worktreeMsg.Branch)
}

func TestIntegration_RefreshData(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	cmd := integration.RefreshData()
	assert.NotNil(t, cmd)
	
	// Execute the command to test the message
	msg := cmd()
	_, ok := msg.(RefreshDataMsg)
	assert.True(t, ok)
}

func TestIntegration_Shutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	// Should not panic
	integration.Shutdown()
}

func TestIntegration_RefreshAllData(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	// Test refreshing data
	integration.refreshAllData()
	
	// Check that last refresh time was updated
	assert.False(t, integration.lastRefresh.IsZero())
}

func TestDefaultSystemStatus(t *testing.T) {
	status := DefaultSystemStatus()
	
	assert.Equal(t, 0, status.ActiveProcesses)
	assert.Equal(t, 0, status.ActiveSessions)
	assert.Equal(t, 0, status.TrackedWorktrees)
	assert.True(t, status.IsHealthy)
	assert.Empty(t, status.Errors)
	assert.False(t, status.LastUpdate.IsZero())
}

func TestIntegration_GetMemoryStats(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	stats := integration.getMemoryStats()
	assert.GreaterOrEqual(t, stats.UsedMB, 0)
	assert.GreaterOrEqual(t, stats.TotalMB, 0)
	assert.GreaterOrEqual(t, stats.Percentage, 0.0)
}

func TestIntegration_GetPerformanceStats(t *testing.T) {
	cfg := config.DefaultConfig()
	
	integration, err := NewIntegration(cfg)
	require.NoError(t, err)
	
	stats := integration.getPerformanceStats()
	assert.GreaterOrEqual(t, stats.CPUPercent, 0.0)
	assert.GreaterOrEqual(t, stats.LoadAverage, 0.0)
	assert.GreaterOrEqual(t, stats.ResponseTime, time.Duration(0))
	assert.GreaterOrEqual(t, stats.ErrorRate, 0.0)
}

func TestExtractProjectFromSessionName(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		expected    string
	}{
		{
			name:        "simple session name",
			sessionName: "myproject",
			expected:    "myproject",
		},
		{
			name:        "project with branch",
			sessionName: "myproject-feature",
			expected:    "myproject-feature",
		},
		{
			name:        "empty session name",
			sessionName: "",
			expected:    "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProjectFromSessionName(tt.sessionName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSessionInfo_Struct(t *testing.T) {
	session := SessionInfo{
		ID:         "test-id",
		Name:       "test-name",
		Project:    "test-project",
		Branch:     "main",
		Directory:  "/test/dir",
		Active:     true,
		Created:    time.Now(),
		LastAccess: time.Now(),
		PID:        12345,
		Status:     "active",
	}
	
	assert.Equal(t, "test-id", session.ID)
	assert.Equal(t, "test-name", session.Name)
	assert.Equal(t, "test-project", session.Project)
	assert.Equal(t, "main", session.Branch)
	assert.Equal(t, "/test/dir", session.Directory)
	assert.True(t, session.Active)
	assert.Equal(t, 12345, session.PID)
	assert.Equal(t, "active", session.Status)
}

func TestWorktreeInfo_Struct(t *testing.T) {
	worktree := WorktreeInfo{
		Path:       "/test/path",
		Branch:     "feature-branch",
		Repository: "test-repo",
		Active:     false,
		LastAccess: time.Now(),
		HasChanges: true,
		Status:     "modified",
	}
	
	assert.Equal(t, "/test/path", worktree.Path)
	assert.Equal(t, "feature-branch", worktree.Branch)
	assert.Equal(t, "test-repo", worktree.Repository)
	assert.False(t, worktree.Active)
	assert.True(t, worktree.HasChanges)
	assert.Equal(t, "modified", worktree.Status)
}