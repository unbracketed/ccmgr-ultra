package ccmgr

import (
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/tui"
)

// SessionManager provides session management operations
type SessionManager interface {
	// List all sessions
	List() ([]SessionInfo, error)

	// Get active sessions only
	Active() ([]SessionInfo, error)

	// Create a new session
	Create(name, directory string) (string, error)

	// Attach to an existing session
	Attach(sessionID string) error

	// Resume a paused session
	Resume(sessionID string) error

	// Find sessions for a specific worktree
	FindForWorktree(worktreePath string) ([]SessionSummary, error)
}

// WorktreeManager provides worktree management operations
type WorktreeManager interface {
	// List all worktrees
	List() ([]WorktreeInfo, error)

	// Get recently accessed worktrees
	Recent() ([]WorktreeInfo, error)

	// Create a new worktree
	Create(path, branch string) error

	// Open a worktree directory
	Open(path string) error

	// Get Claude status for a worktree
	GetClaudeStatus(worktreePath string) ClaudeStatus

	// Update Claude status for a worktree
	UpdateClaudeStatus(worktreePath string, status ClaudeStatus)
}

// SystemManager provides system status and health monitoring
type SystemManager interface {
	// Get overall system status
	Status() SystemStatus

	// Refresh all data
	Refresh() error

	// Get system health information
	Health() HealthInfo
}

// SessionInfo represents session information
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

// WorktreeInfo represents worktree information
type WorktreeInfo struct {
	Path           string
	Branch         string
	Repository     string
	Active         bool
	LastAccess     time.Time
	HasChanges     bool
	Status         string
	ActiveSessions []SessionSummary
	ClaudeStatus   ClaudeStatus
	GitStatus      GitWorktreeStatus
}

// SessionSummary provides summary info about sessions
type SessionSummary struct {
	ID       string
	Name     string
	State    string
	LastUsed time.Time
}

// ClaudeStatus represents Claude Code process status
type ClaudeStatus struct {
	State      string
	ProcessID  int
	LastUpdate time.Time
	SessionID  string
}

// GitWorktreeStatus provides detailed git status information
type GitWorktreeStatus struct {
	IsClean      bool
	Ahead        int
	Behind       int
	Staged       int
	Modified     int
	Untracked    int
	Conflicted   int
	LastCommit   string
	LastCommitAt time.Time
}

// SystemStatus represents overall system status
type SystemStatus struct {
	ActiveProcesses  int
	ActiveSessions   int
	TrackedWorktrees int
	LastUpdate       time.Time
	IsHealthy        bool
	Errors           []string
	Memory           MemoryStats
	Performance      PerformanceStats
}

// MemoryStats holds memory usage information
type MemoryStats struct {
	UsedMB     int
	TotalMB    int
	Percentage float64
}

// PerformanceStats holds performance metrics
type PerformanceStats struct {
	CPUPercent   float64
	LoadAverage  float64
	ResponseTime time.Duration
	ErrorRate    float64
}

// HealthInfo provides system health details
type HealthInfo struct {
	Overall   string
	Services  map[string]string
	LastCheck time.Time
	Uptime    time.Duration
}

// convertSessionInfo converts internal session info to public API
func convertSessionInfo(internal tui.SessionInfo) SessionInfo {
	return SessionInfo{
		ID:         internal.ID,
		Name:       internal.Name,
		Project:    internal.Project,
		Branch:     internal.Branch,
		Directory:  internal.Directory,
		Active:     internal.Active,
		Created:    internal.Created,
		LastAccess: internal.LastAccess,
		PID:        internal.PID,
		Status:     internal.Status,
	}
}

// convertWorktreeInfo converts internal worktree info to public API
func convertWorktreeInfo(internal tui.WorktreeInfo) WorktreeInfo {
	return WorktreeInfo{
		Path:           internal.Path,
		Branch:         internal.Branch,
		Repository:     internal.Repository,
		Active:         internal.Active,
		LastAccess:     internal.LastAccess,
		HasChanges:     internal.HasChanges,
		Status:         internal.Status,
		ActiveSessions: convertSessionSummaries(internal.ActiveSessions),
		ClaudeStatus:   convertClaudeStatus(internal.ClaudeStatus),
		GitStatus:      convertGitStatus(internal.GitStatus),
	}
}

// convertSessionSummaries converts internal session summaries
func convertSessionSummaries(internal []tui.SessionSummary) []SessionSummary {
	result := make([]SessionSummary, len(internal))
	for i, s := range internal {
		result[i] = SessionSummary{
			ID:       s.ID,
			Name:     s.Name,
			State:    s.State,
			LastUsed: s.LastUsed,
		}
	}
	return result
}

// convertClaudeStatus converts internal Claude status
func convertClaudeStatus(internal tui.ClaudeStatus) ClaudeStatus {
	return ClaudeStatus{
		State:      internal.State,
		ProcessID:  internal.ProcessID,
		LastUpdate: internal.LastUpdate,
		SessionID:  internal.SessionID,
	}
}

// convertGitStatus converts internal git status
func convertGitStatus(internal tui.GitWorktreeStatus) GitWorktreeStatus {
	return GitWorktreeStatus{
		IsClean:      internal.IsClean,
		Ahead:        internal.Ahead,
		Behind:       internal.Behind,
		Staged:       internal.Staged,
		Modified:     internal.Modified,
		Untracked:    internal.Untracked,
		Conflicted:   internal.Conflicted,
		LastCommit:   internal.LastCommit,
		LastCommitAt: internal.LastCommitAt,
	}
}

// convertSystemStatus converts internal system status
func convertSystemStatus(internal tui.SystemStatus) SystemStatus {
	return SystemStatus{
		ActiveProcesses:  internal.ActiveProcesses,
		ActiveSessions:   internal.ActiveSessions,
		TrackedWorktrees: internal.TrackedWorktrees,
		LastUpdate:       internal.LastUpdate,
		IsHealthy:        internal.IsHealthy,
		Errors:           internal.Errors,
		Memory: MemoryStats{
			UsedMB:     internal.Memory.UsedMB,
			TotalMB:    internal.Memory.TotalMB,
			Percentage: internal.Memory.Percentage,
		},
		Performance: PerformanceStats{
			CPUPercent:   internal.Performance.CPUPercent,
			LoadAverage:  internal.Performance.LoadAverage,
			ResponseTime: internal.Performance.ResponseTime,
			ErrorRate:    internal.Performance.ErrorRate,
		},
	}
}
