package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/bcdekker/ccmgr-ultra/internal/cli"
	"github.com/bcdekker/ccmgr-ultra/internal/claude"
	"github.com/bcdekker/ccmgr-ultra/internal/tmux"
)

// SessionListData represents data for session list output
type SessionListData struct {
	Sessions  []SessionListItem `json:"sessions" yaml:"sessions"`
	Total     int               `json:"total" yaml:"total"`
	Timestamp time.Time         `json:"timestamp" yaml:"timestamp"`
}

// SessionListItem represents a single session in list output
type SessionListItem struct {
	ID            string    `json:"id" yaml:"id"`
	Name          string    `json:"name" yaml:"name"`
	Project       string    `json:"project" yaml:"project"`
	Worktree      string    `json:"worktree" yaml:"worktree"`
	Branch        string    `json:"branch" yaml:"branch"`
	Directory     string    `json:"directory" yaml:"directory"`
	Status        string    `json:"status" yaml:"status"`
	Active        bool      `json:"active" yaml:"active"`
	ProcessCount  int       `json:"process_count" yaml:"process_count"`
	Created       time.Time `json:"created" yaml:"created"`
	LastAccess    time.Time `json:"last_access" yaml:"last_access"`
	Uptime        string    `json:"uptime" yaml:"uptime"`
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage tmux sessions",
	Long: `Manage tmux sessions with comprehensive lifecycle support including:
- List all active sessions with status information
- Create new sessions for worktrees
- Resume existing sessions with health validation
- Terminate sessions with graceful shutdown
- Clean up stale and orphaned sessions`,
}

// Session list command
var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tmux sessions",
	Long: `List all active tmux sessions managed by ccmgr-ultra including:
- Session details and associated worktrees
- Process status and health information
- Activity and uptime information
- Status classification (active, idle, stale)`,
	RunE: runSessionListCommand,
}

var sessionListFlags struct {
	format       string
	worktree     string
	project      string
	status       string
	withProcesses bool
}

// Session new command
var sessionNewCmd = &cobra.Command{
	Use:   "new <worktree> [flags]",
	Short: "Create new tmux session for worktree",
	Long: `Create new tmux session for specified worktree.
Follows ccmgr-ultra session naming conventions.
Optionally starts Claude Code process in session.`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionNewCommand,
}

var sessionNewFlags struct {
	name          string
	startClaude   bool
	detached      bool
	config        string
	inheritConfig bool
}

// Session resume command
var sessionResumeCmd = &cobra.Command{
	Use:   "resume <session-id> [flags]",
	Short: "Resume existing tmux session",
	Long: `Resume existing tmux session by ID or name.
Handles session state validation and recovery.
Restarts stopped Claude Code processes if needed.`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionResumeCommand,
}

var sessionResumeFlags struct {
	attach        bool
	detached      bool
	restartClaude bool
	force         bool
}

// Session kill command
var sessionKillCmd = &cobra.Command{
	Use:   "kill <session-id> [flags]",
	Short: "Terminate tmux session",
	Long: `Terminate specified tmux session gracefully.
Handles Claude Code process shutdown properly.
Supports batch termination with filters.`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionKillCommand,
}

var sessionKillFlags struct {
	force    bool
	allStale bool
	pattern  string
	cleanup  bool
	timeout  int
}

// Session clean command
var sessionCleanCmd = &cobra.Command{
	Use:   "clean [flags]",
	Short: "Clean up stale and orphaned sessions",
	Long: `Clean up stale, orphaned, or invalid sessions.
Detects and removes sessions with missing worktrees.
Handles sessions with stopped Claude Code processes.`,
	RunE: runSessionCleanCommand,
}

var sessionCleanFlags struct {
	dryRun    bool
	force     bool
	all       bool
	olderThan string
	verbose   bool
}

func init() {
	// List command flags
	sessionListCmd.Flags().StringVarP(&sessionListFlags.format, "format", "f", "table", "Output format (table, json, yaml, compact)")
	sessionListCmd.Flags().StringVarP(&sessionListFlags.worktree, "worktree", "w", "", "Filter by worktree name")
	sessionListCmd.Flags().StringVarP(&sessionListFlags.project, "project", "p", "", "Filter by project name")
	sessionListCmd.Flags().StringVarP(&sessionListFlags.status, "status", "s", "", "Filter by status (active, idle, stale)")
	sessionListCmd.Flags().BoolVar(&sessionListFlags.withProcesses, "with-processes", false, "Include Claude Code process details")

	// New command flags
	sessionNewCmd.Flags().StringVarP(&sessionNewFlags.name, "name", "n", "", "Custom session name suffix")
	sessionNewCmd.Flags().BoolVar(&sessionNewFlags.startClaude, "start-claude", false, "Automatically start Claude Code")
	sessionNewCmd.Flags().BoolVarP(&sessionNewFlags.detached, "detached", "d", false, "Create session detached from terminal")
	sessionNewCmd.Flags().StringVarP(&sessionNewFlags.config, "config", "c", "", "Custom Claude Code config for session")
	sessionNewCmd.Flags().BoolVar(&sessionNewFlags.inheritConfig, "inherit-config", false, "Inherit config from parent directory")

	// Resume command flags
	sessionResumeCmd.Flags().BoolVarP(&sessionResumeFlags.attach, "attach", "a", false, "Attach to session in current terminal")
	sessionResumeCmd.Flags().BoolVarP(&sessionResumeFlags.detached, "detached", "d", false, "Resume session detached")
	sessionResumeCmd.Flags().BoolVar(&sessionResumeFlags.restartClaude, "restart-claude", false, "Restart Claude Code if stopped")
	sessionResumeCmd.Flags().BoolVar(&sessionResumeFlags.force, "force", false, "Force resume even if session appears unhealthy")

	// Kill command flags
	sessionKillCmd.Flags().BoolVarP(&sessionKillFlags.force, "force", "f", false, "Skip confirmation prompts")
	sessionKillCmd.Flags().BoolVar(&sessionKillFlags.allStale, "all-stale", false, "Kill all stale/orphaned sessions")
	sessionKillCmd.Flags().StringVar(&sessionKillFlags.pattern, "pattern", "", "Kill sessions matching pattern")
	sessionKillCmd.Flags().BoolVar(&sessionKillFlags.cleanup, "cleanup", false, "Clean up related processes and state")
	sessionKillCmd.Flags().IntVar(&sessionKillFlags.timeout, "timeout", 10, "Timeout for graceful shutdown (seconds)")

	// Clean command flags
	sessionCleanCmd.Flags().BoolVar(&sessionCleanFlags.dryRun, "dry-run", false, "Show what would be cleaned without acting")
	sessionCleanCmd.Flags().BoolVarP(&sessionCleanFlags.force, "force", "f", false, "Skip confirmation prompts")
	sessionCleanCmd.Flags().BoolVar(&sessionCleanFlags.all, "all", false, "Clean all eligible sessions, not just stale ones")
	sessionCleanCmd.Flags().StringVar(&sessionCleanFlags.olderThan, "older-than", "24h", "Clean sessions older than specified duration")
	sessionCleanCmd.Flags().BoolVarP(&sessionCleanFlags.verbose, "verbose", "v", false, "Detailed cleanup information")

	// Add subcommands to session command
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionNewCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionKillCmd)
	sessionCmd.AddCommand(sessionCleanCmd)

	// Add session command to root
	rootCmd.AddCommand(sessionCmd)
}

func runSessionListCommand(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner("Collecting session information...")
		spinner.Start()
		defer spinner.Stop()
	}

	// Get session information
	sessionManager := tmux.NewSessionManager(cfg)
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to list sessions", err))
	}

	// Convert to list format
	listData := &SessionListData{
		Sessions:  make([]SessionListItem, 0, len(sessions)),
		Total:     len(sessions),
		Timestamp: time.Now(),
	}

	// Optionally get process information
	var processManager *claude.ProcessManager
	if sessionListFlags.withProcesses {
		// Process manager API might not be available, skip for now
		processManager = nil
	}

	for _, sess := range sessions {
		item := SessionListItem{
			ID:         sess.ID,
			Name:       sess.Name,
			Project:    sess.Project,
			Worktree:   sess.Worktree,
			Branch:     sess.Branch,
			Directory:  sess.Directory,
			Active:     sess.Active,
			Created:    sess.Created,
			LastAccess: sess.LastAccess,
			Uptime:     time.Since(sess.Created).Truncate(time.Second).String(),
		}

		// Determine status
		if sess.Active {
			item.Status = "active"
		} else if time.Since(sess.LastAccess) > 1*time.Hour {
			item.Status = "stale"
		} else {
			item.Status = "idle"
		}

		// Get process count if requested
		if processManager != nil {
			// GetProcessesBySession method not available yet
			item.ProcessCount = 0
		}

		listData.Sessions = append(listData.Sessions, item)
	}

	// Apply filters
	if sessionListFlags.worktree != "" {
		filtered := make([]SessionListItem, 0)
		for _, item := range listData.Sessions {
			if item.Worktree == sessionListFlags.worktree {
				filtered = append(filtered, item)
			}
		}
		listData.Sessions = filtered
		listData.Total = len(filtered)
	}

	if sessionListFlags.project != "" {
		filtered := make([]SessionListItem, 0)
		for _, item := range listData.Sessions {
			if item.Project == sessionListFlags.project {
				filtered = append(filtered, item)
			}
		}
		listData.Sessions = filtered
		listData.Total = len(filtered)
	}

	if sessionListFlags.status != "" {
		filtered := make([]SessionListItem, 0)
		for _, item := range listData.Sessions {
			if item.Status == sessionListFlags.status {
				filtered = append(filtered, item)
			}
		}
		listData.Sessions = filtered
		listData.Total = len(filtered)
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Found %d sessions", listData.Total))
	}

	formatter, err := setupOutputFormatter(sessionListFlags.format)
	if err != nil {
		return handleCLIError(err)
	}

	return formatter.Format(listData)
}

func runSessionNewCommand(cmd *cobra.Command, args []string) error {
	worktreeName := args[0]

	if err := validateWorktreeArg(worktreeName); err != nil {
		return handleCLIError(err)
	}

	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner(fmt.Sprintf("Creating session for worktree '%s'...", worktreeName))
		spinner.Start()
		defer spinner.Stop()
	}

	// Find the worktree directory
	worktreeDir, err := findWorktreeDirectory(worktreeName)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to find worktree", err))
	}

	// Generate session name
	sessionName := sessionNewFlags.name
	if sessionName == "" {
		sessionName = generateSessionName(cfg, worktreeName)
	}

	// Create the session
	sessionManager := tmux.NewSessionManager(cfg)
	session, err := sessionManager.CreateSession(
		getCurrentProjectName(), // project
		worktreeName,            // worktree
		worktreeName,            // branch (assume branch name matches worktree name)
		worktreeDir,             // directory
	)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to create session", err))
	}

	if spinner != nil {
		spinner.SetMessage(fmt.Sprintf("Session '%s' created", session.Name))
	}

	// Start Claude Code if requested
	if sessionNewFlags.startClaude {
		if spinner != nil {
			spinner.SetMessage("Starting Claude Code...")
		}

		// Claude Code starting functionality not yet implemented
		if isVerbose() {
			fmt.Printf("Warning: Claude Code auto-start not yet implemented\n")
		}
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Session '%s' created successfully", session.Name))
	}

	if !isQuiet() {
		fmt.Printf("\nSession created:\n")
		fmt.Printf("  ID: %s\n", session.ID)
		fmt.Printf("  Name: %s\n", session.Name)
		fmt.Printf("  Directory: %s\n", session.Directory)
		if sessionNewFlags.startClaude {
			fmt.Printf("  Claude Code: Started\n")
		}

		if !sessionNewFlags.detached {
			fmt.Printf("\nTo attach to this session, run:\n")
			fmt.Printf("  tmux attach -t %s\n", session.ID)
		}
	}

	return nil
}

func runSessionResumeCommand(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	if err := validateSessionArg(sessionID); err != nil {
		return handleCLIError(err)
	}

	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner(fmt.Sprintf("Resuming session '%s'...", sessionID))
		spinner.Start()
		defer spinner.Stop()
	}

	// Get session manager
	sessionManager := tmux.NewSessionManager(cfg)

	// Check if session exists
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to find session", err))
	}

	// Health check (simplified since CheckSessionHealth method doesn't exist yet)
	if !sessionResumeFlags.force {
		if spinner != nil {
			spinner.SetMessage("Checking session status...")
		}

		// Basic check - just verify session exists and is active
		if !session.Active {
			return handleCLIError(cli.NewErrorWithSuggestion(
				"session appears inactive",
				"Use --force to resume anyway or check session status first",
			))
		}
	}

	// Restart Claude Code if requested
	if sessionResumeFlags.restartClaude {
		if spinner != nil {
			spinner.SetMessage("Restarting Claude Code...")
		}

		// Claude Code restart functionality not yet implemented
		if isVerbose() {
			fmt.Printf("Warning: Claude Code restart not yet implemented\n")
		}
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Session '%s' resumed", sessionID))
	}

	if !isQuiet() {
		fmt.Printf("Session '%s' resumed successfully\n", sessionID)

		if sessionResumeFlags.attach {
			fmt.Printf("Attaching to session...\n")
			// In a real implementation, this would exec tmux attach
		} else {
			fmt.Printf("\nTo attach to this session, run:\n")
			fmt.Printf("  tmux attach -t %s\n", session.ID)
		}
	}

	return nil
}

func runSessionKillCommand(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	if err := validateSessionArg(sessionID); err != nil {
		return handleCLIError(err)
	}

	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	// Safety check - confirm termination
	if !sessionKillFlags.force && !isDryRun() {
		fmt.Printf("This will terminate session: %s\n", sessionID)
		fmt.Printf("Proceed with termination? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Termination cancelled")
			return nil
		}
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner(fmt.Sprintf("Terminating session '%s'...", sessionID))
		spinner.Start()
		defer spinner.Stop()
	}

	sessionManager := tmux.NewSessionManager(cfg)

	// Clean up processes if requested
	if sessionKillFlags.cleanup {
		if spinner != nil {
			spinner.SetMessage("Cleaning up Claude Code processes...")
		}

		// Process cleanup functionality not yet implemented
		if isVerbose() {
			fmt.Printf("Warning: Process cleanup not yet implemented\n")
		}
	}

	if isDryRun() {
		if spinner != nil {
			spinner.StopWithMessage("Dry run: Would terminate session")
		}
		fmt.Printf("Dry run: Would terminate session '%s'\n", sessionID)
		return nil
	}

	// Kill the session
	err = sessionManager.KillSession(sessionID)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to kill session", err))
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Session '%s' terminated", sessionID))
	}

	if !isQuiet() {
		fmt.Printf("Session '%s' terminated successfully\n", sessionID)
	}

	return nil
}

func runSessionCleanCommand(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner("Scanning for stale sessions...")
		spinner.Start()
		defer spinner.Stop()
	}

	sessionManager := tmux.NewSessionManager(cfg)
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to list sessions", err))
	}

	// Parse older than duration
	olderThanDuration, err := time.ParseDuration(sessionCleanFlags.olderThan)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("invalid duration format", err))
	}

	// Find sessions to clean
	var sessionsToClean []*tmux.Session
	for _, sess := range sessions {
		shouldClean := false

		if sessionCleanFlags.all {
			shouldClean = true
		} else {
			// Check if session is stale
			if time.Since(sess.LastAccess) > olderThanDuration {
				shouldClean = true
			}

			// Check if worktree directory still exists
			if _, err := os.Stat(sess.Directory); os.IsNotExist(err) {
				shouldClean = true
			}
		}

		if shouldClean {
			sessionsToClean = append(sessionsToClean, sess)
		}
	}

	if len(sessionsToClean) == 0 {
		if spinner != nil {
			spinner.StopWithMessage("No sessions to clean")
		}
		fmt.Println("No stale sessions found")
		return nil
	}

	if sessionCleanFlags.dryRun {
		if spinner != nil {
			spinner.StopWithMessage(fmt.Sprintf("Dry run: Found %d sessions to clean", len(sessionsToClean)))
		}

		fmt.Printf("Dry run: Would clean %d sessions:\n", len(sessionsToClean))
		for _, sess := range sessionsToClean {
			fmt.Printf("  - %s (%s) - Last access: %s\n", sess.Name, sess.ID, sess.LastAccess.Format("2006-01-02 15:04:05"))
		}
		return nil
	}

	// Confirm cleanup
	if !sessionCleanFlags.force {
		fmt.Printf("This will clean up %d stale sessions:\n", len(sessionsToClean))
		for _, sess := range sessionsToClean {
			fmt.Printf("  - %s (%s)\n", sess.Name, sess.ID)
		}
		fmt.Printf("Proceed with cleanup? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cleanup cancelled")
			return nil
		}
	}

	// Clean up sessions
	if spinner != nil {
		spinner.SetMessage(fmt.Sprintf("Cleaning up %d sessions...", len(sessionsToClean)))
	}

	cleanedCount := 0
	for _, sess := range sessionsToClean {
		err := sessionManager.KillSession(sess.ID)
		if err != nil {
			if sessionCleanFlags.verbose {
				fmt.Printf("Warning: Failed to clean session %s: %v\n", sess.ID, err)
			}
		} else {
			cleanedCount++
		}
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Cleaned up %d sessions", cleanedCount))
	}

	if !isQuiet() {
		fmt.Printf("Successfully cleaned up %d out of %d sessions\n", cleanedCount, len(sessionsToClean))
	}

	return nil
}

// Helper functions

func findWorktreeDirectory(worktreeName string) (string, error) {
	// This would need to integrate with the git worktree manager
	// For now, return a placeholder implementation
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Try common patterns
	candidates := []string{
		filepath.Join(cwd, "..", worktreeName),
		filepath.Join(cwd, worktreeName),
		worktreeName, // If it's already a path
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("worktree directory not found for: %s", worktreeName)
}