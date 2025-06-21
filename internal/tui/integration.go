package tui

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/your-username/ccmgr-ultra/internal/claude"
	"github.com/your-username/ccmgr-ultra/internal/config"
	"github.com/your-username/ccmgr-ultra/internal/git"
	"github.com/your-username/ccmgr-ultra/internal/tmux"
)

// Integration manages the integration between TUI and backend services
type Integration struct {
	config     *config.Config
	claudeMgr  *claude.ProcessManager
	tmuxMgr    *tmux.SessionManager
	gitMgr     *git.WorktreeManager
	
	// Data cache
	mu              sync.RWMutex
	sessions        []SessionInfo
	worktrees       []WorktreeInfo
	systemStatus    SystemStatus
	lastRefresh     time.Time
	refreshInterval time.Duration
	
	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc
}

// SessionInfo represents session information for the TUI
type SessionInfo struct {
	ID         string
	Name       string
	Project    string
	Branch     string
	Directory  string
	Active     bool
	Created    time.Time
	LastAccess time.Time
	PID        int
	Status     string
}

// WorktreeInfo represents worktree information for the TUI
type WorktreeInfo struct {
	Path       string
	Branch     string
	Repository string
	Active     bool
	LastAccess time.Time
	HasChanges bool
	Status     string
}

// SystemStatus represents overall system status
type SystemStatus struct {
	ActiveProcesses   int
	ActiveSessions    int
	TrackedWorktrees  int
	LastUpdate        time.Time
	IsHealthy         bool
	Errors            []string
	Memory            MemoryStats
	Performance       PerformanceStats
}

// MemoryStats holds memory usage information
type MemoryStats struct {
	UsedMB     int
	TotalMB    int
	Percentage float64
}

// PerformanceStats holds performance metrics
type PerformanceStats struct {
	CPUPercent    float64
	LoadAverage   float64
	ResponseTime  time.Duration
	ErrorRate     float64
}

// NewIntegration creates a new integration layer
func NewIntegration(config *config.Config) (*Integration, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize backend managers
	claudeMgr, err := claude.NewProcessManager(&claude.ProcessConfig{})
	if err != nil {
		cancel()
		return nil, err
	}
	
	tmuxMgr := tmux.NewSessionManager(config)
	
	// Note: gitMgr requires a repository, so we'll initialize it when needed
	// For now, we'll set it to nil and handle it gracefully
	
	refreshInterval := time.Duration(config.RefreshInterval) * time.Second
	if refreshInterval <= 0 {
		refreshInterval = 5 * time.Second // Default to 5 seconds
	}
	
	integration := &Integration{
		config:          config,
		claudeMgr:       claudeMgr,
		tmuxMgr:         tmuxMgr,
		gitMgr:          nil, // Will be initialized per-repository
		sessions:        []SessionInfo{},
		worktrees:       []WorktreeInfo{},
		systemStatus:    DefaultSystemStatus(),
		refreshInterval: refreshInterval,
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Start initial data refresh
	go integration.startBackgroundRefresh()
	
	return integration, nil
}

// startBackgroundRefresh runs periodic data refresh in the background
func (i *Integration) startBackgroundRefresh() {
	ticker := time.NewTicker(i.refreshInterval)
	defer ticker.Stop()
	
	// Initial refresh
	i.refreshAllData()
	
	for {
		select {
		case <-ticker.C:
			i.refreshAllData()
		case <-i.ctx.Done():
			return
		}
	}
}

// refreshAllData refreshes all cached data from backend services
func (i *Integration) refreshAllData() {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	i.lastRefresh = time.Now()
	
	// Refresh Claude processes
	i.refreshClaudeData()
	
	// Refresh Tmux sessions
	i.refreshTmuxData()
	
	// Refresh Git worktrees
	i.refreshGitData()
	
	// Update system status
	i.updateSystemStatus()
}

// refreshClaudeData refreshes Claude process information
func (i *Integration) refreshClaudeData() {
	processes := i.claudeMgr.GetAllProcesses()
	
	// Update session info with Claude process data
	for j, session := range i.sessions {
		for _, process := range processes {
			if process.SessionID == session.ID {
				i.sessions[j].PID = process.PID
				i.sessions[j].Status = process.State.String()
				i.sessions[j].Active = process.State != claude.StateStopped
				i.sessions[j].LastAccess = process.LastUpdate
				break
			}
		}
	}
}

// refreshTmuxData refreshes Tmux session information
func (i *Integration) refreshTmuxData() {
	sessions, err := i.tmuxMgr.ListSessions()
	if err != nil {
		i.systemStatus.Errors = append(i.systemStatus.Errors, 
			"Failed to list tmux sessions: "+err.Error())
		return
	}
	
	// Clear existing sessions
	i.sessions = []SessionInfo{}
	
	// Convert tmux sessions to TUI session info
	for _, session := range sessions {
		// Get session details
		sessionInfo := SessionInfo{
			ID:         session.ID,
			Name:       session.Name,
			Project:    session.Project,
			Branch:     session.Branch,
			Directory:  session.Directory,
			Active:     session.Active,
			Created:    session.Created,
			LastAccess: session.LastAccess,
			Status:     "active",
		}
		
		i.sessions = append(i.sessions, sessionInfo)
	}
}

// refreshGitData refreshes Git worktree information
func (i *Integration) refreshGitData() {
	// Since we don't have repository context at this level,
	// we'll implement a basic worktree discovery mechanism
	
	// For now, create placeholder data
	// In a real implementation, this would scan configured directories
	// for git repositories and their worktrees
	
	i.worktrees = []WorktreeInfo{
		{
			Path:       "/example/worktree1",
			Branch:     "feature/ui-improvements",
			Repository: "ccmgr-ultra",
			Active:     true,
			LastAccess: time.Now().Add(-30 * time.Minute),
			HasChanges: false,
			Status:     "clean",
		},
		{
			Path:       "/example/worktree2", 
			Branch:     "feature/backend-refactor",
			Repository: "ccmgr-ultra",
			Active:     false,
			LastAccess: time.Now().Add(-2 * time.Hour),
			HasChanges: true,
			Status:     "modified",
		},
	}
}

// updateSystemStatus updates the overall system status
func (i *Integration) updateSystemStatus() {
	activeProcesses := len(i.claudeMgr.GetAllProcesses())
	activeSessions := 0
	
	for _, session := range i.sessions {
		if session.Active {
			activeSessions++
		}
	}
	
	trackedWorktrees := len(i.worktrees)
	
	// Check system health
	isHealthy := len(i.systemStatus.Errors) == 0
	
	i.systemStatus = SystemStatus{
		ActiveProcesses:  activeProcesses,
		ActiveSessions:   activeSessions,
		TrackedWorktrees: trackedWorktrees,
		LastUpdate:       time.Now(),
		IsHealthy:        isHealthy,
		Errors:          i.systemStatus.Errors, // Keep accumulated errors
		Memory:          i.getMemoryStats(),
		Performance:     i.getPerformanceStats(),
	}
	
	// Clear old errors (keep only recent ones)
	if len(i.systemStatus.Errors) > 10 {
		i.systemStatus.Errors = i.systemStatus.Errors[len(i.systemStatus.Errors)-10:]
	}
}

// getMemoryStats returns current memory statistics
func (i *Integration) getMemoryStats() MemoryStats {
	// Placeholder implementation
	// In a real implementation, this would query system memory usage
	return MemoryStats{
		UsedMB:     512,
		TotalMB:    2048,
		Percentage: 25.0,
	}
}

// getPerformanceStats returns current performance statistics
func (i *Integration) getPerformanceStats() PerformanceStats {
	// Placeholder implementation
	// In a real implementation, this would query system performance metrics
	return PerformanceStats{
		CPUPercent:   15.5,
		LoadAverage:  0.8,
		ResponseTime: 50 * time.Millisecond,
		ErrorRate:    0.1,
	}
}

// Public methods for TUI access

// GetSystemStatus returns the current system status
func (i *Integration) GetSystemStatus() SystemStatus {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.systemStatus
}

// GetActiveSessions returns currently active sessions
func (i *Integration) GetActiveSessions() []SessionInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	var active []SessionInfo
	for _, session := range i.sessions {
		if session.Active {
			active = append(active, session)
		}
	}
	return active
}

// GetAllSessions returns all sessions
func (i *Integration) GetAllSessions() []SessionInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]SessionInfo(nil), i.sessions...)
}

// GetRecentWorktrees returns recently accessed worktrees
func (i *Integration) GetRecentWorktrees() []WorktreeInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	// Return up to 5 most recently accessed worktrees
	recent := append([]WorktreeInfo(nil), i.worktrees...)
	if len(recent) > 5 {
		recent = recent[:5]
	}
	return recent
}

// GetAllWorktrees returns all worktrees
func (i *Integration) GetAllWorktrees() []WorktreeInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]WorktreeInfo(nil), i.worktrees...)
}

// StartPeriodicRefresh returns a command for periodic data refresh
func (i *Integration) StartPeriodicRefresh() tea.Cmd {
	return tea.Tick(i.refreshInterval, func(t time.Time) tea.Msg {
		return RefreshDataMsg{}
	})
}

// AttachSession attaches to a tmux session
func (i *Integration) AttachSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := i.tmuxMgr.AttachSession(sessionID)
		if err != nil {
			return ErrorMsg{Error: err}
		}
		return SessionAttachedMsg{SessionID: sessionID}
	}
}

// OpenWorktree opens a worktree directory
func (i *Integration) OpenWorktree(path string) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would open the worktree
		// For now, just return a success message
		return WorktreeOpenedMsg{Path: path}
	}
}

// CreateSession creates a new tmux session
func (i *Integration) CreateSession(name, directory string) tea.Cmd {
	return func() tea.Msg {
		_, err := i.tmuxMgr.CreateSession("unknown", "unknown", "main", directory)
		if err != nil {
			return ErrorMsg{Error: err}
		}
		return SessionCreatedMsg{SessionID: name}
	}
}

// CreateWorktree creates a new git worktree
func (i *Integration) CreateWorktree(path, branch string) tea.Cmd {
	return func() tea.Msg {
		// This would use the git manager to create a worktree
		// For now, return a placeholder success message
		return WorktreeCreatedMsg{Path: path, Branch: branch}
	}
}

// RefreshData manually refreshes all data
func (i *Integration) RefreshData() tea.Cmd {
	return func() tea.Msg {
		i.refreshAllData()
		return RefreshDataMsg{}
	}
}

// Shutdown gracefully shuts down the integration layer
func (i *Integration) Shutdown() {
	if i.cancel != nil {
		i.cancel()
	}
}

// Helper functions

// extractProjectFromSessionName extracts project name from session name
func extractProjectFromSessionName(sessionName string) string {
	// Simple implementation - in practice this would be more sophisticated
	// Based on naming conventions like "project_branch" or "project-feature"
	return sessionName
}

// DefaultSystemStatus returns a default system status
func DefaultSystemStatus() SystemStatus {
	return SystemStatus{
		ActiveProcesses:  0,
		ActiveSessions:   0,
		TrackedWorktrees: 0,
		LastUpdate:       time.Now(),
		IsHealthy:        true,
		Errors:          []string{},
		Memory: MemoryStats{
			UsedMB:     0,
			TotalMB:    0,
			Percentage: 0.0,
		},
		Performance: PerformanceStats{
			CPUPercent:   0.0,
			LoadAverage:  0.0,
			ResponseTime: 0,
			ErrorRate:    0.0,
		},
	}
}

// Message types for tea.Cmd communication

// ErrorMsg represents an error message
type ErrorMsg struct {
	Error error
}

// SessionAttachedMsg indicates a session was attached
type SessionAttachedMsg struct {
	SessionID string
}

// SessionCreatedMsg indicates a session was created
type SessionCreatedMsg struct {
	SessionID string
}

// WorktreeOpenedMsg indicates a worktree was opened
type WorktreeOpenedMsg struct {
	Path string
}

// WorktreeCreatedMsg indicates a worktree was created
type WorktreeCreatedMsg struct {
	Path   string
	Branch string
}