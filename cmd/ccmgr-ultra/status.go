package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/bcdekker/ccmgr-ultra/internal/cli"
	"github.com/bcdekker/ccmgr-ultra/internal/claude"
	"github.com/bcdekker/ccmgr-ultra/internal/git"
	"github.com/bcdekker/ccmgr-ultra/internal/hooks"
	"github.com/bcdekker/ccmgr-ultra/internal/tmux"
)

// StatusData represents the complete status information
type StatusData struct {
	System    SystemStatus    `json:"system" yaml:"system"`
	Worktrees []WorktreeStatus `json:"worktrees" yaml:"worktrees"`
	Sessions  []SessionStatus  `json:"sessions" yaml:"sessions"`
	Processes []ProcessStatus  `json:"processes" yaml:"processes"`
	Hooks     HookStatus      `json:"hooks" yaml:"hooks"`
	Timestamp time.Time       `json:"timestamp" yaml:"timestamp"`
}

// SystemStatus represents overall system health
type SystemStatus struct {
	Healthy            bool          `json:"healthy" yaml:"healthy"`
	TotalWorktrees     int           `json:"total_worktrees" yaml:"total_worktrees"`
	CleanWorktrees     int           `json:"clean_worktrees" yaml:"clean_worktrees"`
	DirtyWorktrees     int           `json:"dirty_worktrees" yaml:"dirty_worktrees"`
	ActiveSessions     int           `json:"active_sessions" yaml:"active_sessions"`
	TotalProcesses     int           `json:"total_processes" yaml:"total_processes"`
	HealthyProcesses   int           `json:"healthy_processes" yaml:"healthy_processes"`
	UnhealthyProcesses int           `json:"unhealthy_processes" yaml:"unhealthy_processes"`
	ProcessManagerRunning bool       `json:"process_manager_running" yaml:"process_manager_running"`
	HooksEnabled       bool          `json:"hooks_enabled" yaml:"hooks_enabled"`
	AverageUptime      time.Duration `json:"average_uptime" yaml:"average_uptime"`
}

// WorktreeStatus represents the status of a single worktree
type WorktreeStatus struct {
	Path         string    `json:"path" yaml:"path"`
	Branch       string    `json:"branch" yaml:"branch"`
	Head         string    `json:"head" yaml:"head"`
	IsClean      bool      `json:"is_clean" yaml:"is_clean"`
	HasUncommitted bool    `json:"has_uncommitted" yaml:"has_uncommitted"`
	TmuxSession  string    `json:"tmux_session" yaml:"tmux_session"`
	LastAccessed time.Time `json:"last_accessed" yaml:"last_accessed"`
	ProcessCount int       `json:"process_count" yaml:"process_count"`
}

// SessionStatus represents the status of a tmux session
type SessionStatus struct {
	ID         string    `json:"id" yaml:"id"`
	Name       string    `json:"name" yaml:"name"`
	Project    string    `json:"project" yaml:"project"`
	Worktree   string    `json:"worktree" yaml:"worktree"`
	Branch     string    `json:"branch" yaml:"branch"`
	Directory  string    `json:"directory" yaml:"directory"`
	Active     bool      `json:"active" yaml:"active"`
	Created    time.Time `json:"created" yaml:"created"`
	LastAccess time.Time `json:"last_access" yaml:"last_access"`
}

// ProcessStatus represents the status of a Claude Code process
type ProcessStatus struct {
	PID         int       `json:"pid" yaml:"pid"`
	SessionID   string    `json:"session_id" yaml:"session_id"`
	WorkingDir  string    `json:"working_dir" yaml:"working_dir"`
	State       string    `json:"state" yaml:"state"`
	StartTime   time.Time `json:"start_time" yaml:"start_time"`
	Uptime      string    `json:"uptime" yaml:"uptime"`
	TmuxSession string    `json:"tmux_session" yaml:"tmux_session"`
	WorktreeID  string    `json:"worktree_id" yaml:"worktree_id"`
	CPUPercent  float64   `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryMB    int64     `json:"memory_mb" yaml:"memory_mb"`
}

// HookStatus represents the status of the hook system
type HookStatus struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display system status information",
	Long: `Display comprehensive status information including:
- Git worktrees and their states
- Active tmux sessions
- Claude Code processes and health
- Hook system status
- Overall system health metrics`,
	RunE: runStatusCommand,
}

var statusFlags struct {
	worktree        string
	format          string
	watch           bool
	refreshInterval int
}

func init() {
	statusCmd.Flags().StringVarP(&statusFlags.worktree, "worktree", "w", "", "Show status for specific worktree")
	statusCmd.Flags().StringVarP(&statusFlags.format, "format", "f", "table", "Output format (table, json, yaml)")
	statusCmd.Flags().BoolVar(&statusFlags.watch, "watch", false, "Continuously monitor and update status")
	statusCmd.Flags().IntVar(&statusFlags.refreshInterval, "refresh-interval", 5, "Status refresh interval in seconds for watch mode")

	rootCmd.AddCommand(statusCmd)
}

func runStatusCommand(cmd *cobra.Command, args []string) error {
	if statusFlags.watch {
		return runWatchMode()
	}

	statusData, err := collectStatusData()
	if err != nil {
		return handleCLIError(err)
	}

	// Filter by worktree if specified
	if statusFlags.worktree != "" {
		statusData = filterByWorktree(statusData, statusFlags.worktree)
	}

	formatter, err := setupStatusOutputFormatter(statusFlags.format)
	if err != nil {
		return handleCLIError(err)
	}

	return formatter.Format(statusData)
}

func runWatchMode() error {
	if !shouldShowProgress() {
		return cli.NewError("watch mode requires interactive terminal")
	}

	ticker := time.NewTicker(time.Duration(statusFlags.refreshInterval) * time.Second)
	defer ticker.Stop()

	// Initial display
	if err := displayStatus(); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			// Clear screen and display updated status
			fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top
			if err := displayStatus(); err != nil {
				return err
			}
		}
	}
}

func displayStatus() error {
	statusData, err := collectStatusData()
	if err != nil {
		return handleCLIError(err)
	}

	if statusFlags.worktree != "" {
		statusData = filterByWorktree(statusData, statusFlags.worktree)
	}

	formatter, err := setupStatusOutputFormatter(statusFlags.format)
	if err != nil {
		return handleCLIError(err)
	}

	return formatter.Format(statusData)
}

func collectStatusData() (*StatusData, error) {
	
	// Load configuration
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return nil, err
	}

	status := &StatusData{
		Timestamp: time.Now(),
	}

	// Initialize managers
	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner("Collecting status information...")
		spinner.Start()
		defer spinner.Stop()
	}

	// Collect worktree information
	if spinner != nil {
		spinner.SetMessage("Collecting worktree information...")
	}
	
	// Create repository manager and detect repository
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		if isVerbose() {
			fmt.Printf("Warning: Failed to detect repository: %v\n", err)
		}
	} else {
		worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
		worktrees, err := worktreeManager.ListWorktrees()
		if err != nil && isVerbose() {
			fmt.Printf("Warning: Failed to list worktrees: %v\n", err)
		} else {
			status.Worktrees = convertWorktrees(worktrees)
		}

		// Get worktree stats
		stats, err := worktreeManager.GetWorktreeStats()
		if err == nil {
			status.System.TotalWorktrees = stats.Total
			status.System.CleanWorktrees = stats.Clean
			status.System.DirtyWorktrees = stats.Dirty
		}
	}

	// Collect tmux session information
	if spinner != nil {
		spinner.SetMessage("Collecting session information...")
	}
	sessionManager := tmux.NewSessionManager(cfg)
	sessions, err := sessionManager.ListSessions()
	if err != nil && isVerbose() {
		fmt.Printf("Warning: Failed to list sessions: %v\n", err)
	} else {
		status.Sessions = convertSessions(sessions)
		status.System.ActiveSessions = len(sessions)
	}

	// Collect process information
	if spinner != nil {
		spinner.SetMessage("Collecting process information...")
	}
	processManager, err := claude.NewProcessManager(nil) // Use default config
	if err != nil {
		if isVerbose() {
			fmt.Printf("Warning: Failed to create process manager: %v\n", err)
		}
	} else {
		processes := processManager.GetAllProcesses()
		status.Processes = convertProcesses(processes)
		
		// Get system health
		systemHealth := processManager.GetSystemHealth()
		if systemHealth != nil {
			status.System.TotalProcesses = systemHealth.TotalProcesses
			status.System.HealthyProcesses = systemHealth.HealthyProcesses
			status.System.UnhealthyProcesses = systemHealth.UnhealthyProcesses
			status.System.ProcessManagerRunning = systemHealth.IsManagerRunning
			status.System.AverageUptime = systemHealth.AverageUptime
		}
	}

	// Collect hook information
	if spinner != nil {
		spinner.SetMessage("Collecting hook information...")
	}
	// Create a basic hook executor (simplified for status)
	hookExecutor := hooks.NewDefaultExecutor(cfg)
	hookIntegrator := hooks.NewStatusHookIntegrator(hookExecutor)
	status.Hooks.Enabled = hookIntegrator.IsEnabled()
	status.System.HooksEnabled = status.Hooks.Enabled

	// Determine overall system health
	status.System.Healthy = determineSystemHealth(status)

	if spinner != nil {
		spinner.StopWithMessage("Status collection complete")
	}

	return status, nil
}

func convertWorktrees(worktrees []git.WorktreeInfo) []WorktreeStatus {
	result := make([]WorktreeStatus, len(worktrees))
	for i, wt := range worktrees {
		result[i] = WorktreeStatus{
			Path:           wt.Path,
			Branch:         wt.Branch,
			Head:           wt.Head,
			IsClean:        wt.IsClean,
			HasUncommitted: wt.HasUncommitted,
			TmuxSession:    wt.TmuxSession,
			LastAccessed:   wt.LastAccessed,
			ProcessCount:   0, // Will be populated later if needed
		}
	}
	return result
}

func convertSessions(sessions []*tmux.Session) []SessionStatus {
	result := make([]SessionStatus, len(sessions))
	for i, sess := range sessions {
		result[i] = SessionStatus{
			ID:         sess.ID,
			Name:       sess.Name,
			Project:    sess.Project,
			Worktree:   sess.Worktree,
			Branch:     sess.Branch,
			Directory:  sess.Directory,
			Active:     sess.Active,
			Created:    sess.Created,
			LastAccess: sess.LastAccess,
		}
	}
	return result
}

func convertProcesses(processes []*claude.ProcessInfo) []ProcessStatus {
	result := make([]ProcessStatus, len(processes))
	for i, proc := range processes {
		uptime := time.Since(proc.StartTime).Truncate(time.Second).String()
		
		result[i] = ProcessStatus{
			PID:         proc.PID,
			SessionID:   proc.SessionID,
			WorkingDir:  proc.WorkingDir,
			State:       proc.State.String(),
			StartTime:   proc.StartTime,
			Uptime:      uptime,
			TmuxSession: proc.TmuxSession,
			WorktreeID:  proc.WorktreeID,
			CPUPercent:  proc.CPUPercent,
			MemoryMB:    proc.MemoryMB,
		}
	}
	return result
}

func determineSystemHealth(status *StatusData) bool {
	// System is healthy if:
	// - Process manager is running
	// - No unhealthy processes (or all processes are healthy)
	// - At least some worktrees are clean (if any exist)
	
	if !status.System.ProcessManagerRunning {
		return false
	}
	
	if status.System.TotalProcesses > 0 && status.System.UnhealthyProcesses > 0 {
		return false
	}
	
	return true
}

func filterByWorktree(statusData *StatusData, worktreeName string) *StatusData {
	// Filter worktrees
	filteredWorktrees := make([]WorktreeStatus, 0)
	for _, wt := range statusData.Worktrees {
		if wt.Path == worktreeName || wt.Branch == worktreeName {
			filteredWorktrees = append(filteredWorktrees, wt)
		}
	}
	
	// Filter sessions related to the worktree
	filteredSessions := make([]SessionStatus, 0)
	for _, sess := range statusData.Sessions {
		if sess.Worktree == worktreeName {
			filteredSessions = append(filteredSessions, sess)
		}
	}
	
	// Filter processes related to the worktree
	filteredProcesses := make([]ProcessStatus, 0)
	for _, proc := range statusData.Processes {
		if proc.WorktreeID == worktreeName {
			filteredProcesses = append(filteredProcesses, proc)
		}
	}
	
	return &StatusData{
		System:    statusData.System, // Keep system status as is
		Worktrees: filteredWorktrees,
		Sessions:  filteredSessions,
		Processes: filteredProcesses,
		Hooks:     statusData.Hooks,
		Timestamp: statusData.Timestamp,
	}
}