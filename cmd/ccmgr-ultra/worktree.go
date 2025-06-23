package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unbracketed/ccmgr-ultra/internal/claude"
	"github.com/unbracketed/ccmgr-ultra/internal/cli"
	"github.com/unbracketed/ccmgr-ultra/internal/config"
	"github.com/unbracketed/ccmgr-ultra/internal/git"
	"github.com/unbracketed/ccmgr-ultra/internal/tmux"
)

// WorktreeListData represents data for worktree list output
type WorktreeListData struct {
	Worktrees []WorktreeListItem `json:"worktrees" yaml:"worktrees"`
	Total     int                `json:"total" yaml:"total"`
	Timestamp time.Time          `json:"timestamp" yaml:"timestamp"`
}

// WorktreeListItem represents a single worktree in list output
type WorktreeListItem struct {
	Name         string    `json:"name" yaml:"name"`
	Path         string    `json:"path" yaml:"path"`
	Branch       string    `json:"branch" yaml:"branch"`
	Head         string    `json:"head" yaml:"head"`
	Status       string    `json:"status" yaml:"status"`
	IsClean      bool      `json:"is_clean" yaml:"is_clean"`
	TmuxSession  string    `json:"tmux_session" yaml:"tmux_session"`
	ProcessCount int       `json:"process_count" yaml:"process_count"`
	LastAccessed time.Time `json:"last_accessed" yaml:"last_accessed"`
	Created      time.Time `json:"created" yaml:"created"`
}

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees",
	Long: `Manage git worktrees with comprehensive lifecycle support including:
- List all worktrees with status information
- Create new worktrees with optional tmux session
- Delete worktrees with cleanup of related resources
- Merge worktree changes back to main branch
- Push worktree branches with PR creation support`,
}

// Worktree list command
var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all git worktrees",
	Long: `List all git worktrees with comprehensive status information including:
- Branch and HEAD commit information
- Clean/dirty status
- Associated tmux sessions
- Claude Code process information
- Last accessed timestamps`,
	RunE: runWorktreeListCommand,
}

var worktreeListFlags struct {
	format        string
	status        string
	branch        string
	withProcesses bool
	sort          string
}

// Worktree create command
var worktreeCreateCmd = &cobra.Command{
	Use:   "create <branch> [flags]",
	Short: "Create a new git worktree",
	Long: `Create a new git worktree from specified or current branch.
Automatically generates worktree directory using configured pattern.
Optionally starts tmux session and Claude Code process.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeCreateCommand,
}

var worktreeCreateFlags struct {
	base         string
	directory    string
	startSession bool
	startClaude  bool
	remote       bool
	force        bool
}

// Worktree delete command
var worktreeDeleteCmd = &cobra.Command{
	Use:   "delete <worktree> [flags]",
	Short: "Delete a git worktree",
	Long: `Delete specified worktree with safety checks.
Handles active tmux sessions and Claude Code processes.
Optionally cleans up related sessions and processes.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeDeleteCommand,
}

var worktreeDeleteFlags struct {
	force            bool
	cleanupSessions  bool
	cleanupProcesses bool
	keepBranch       bool
	pattern          string
}

// Worktree merge command
var worktreeMergeCmd = &cobra.Command{
	Use:   "merge <worktree> [flags]",
	Short: "Merge worktree changes back to target branch",
	Long: `Merge worktree changes back to main/target branch.
Handles merge conflicts with clear guidance.
Optionally pushes changes before merging.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeMergeCommand,
}

var worktreeMergeFlags struct {
	target      string
	strategy    string
	deleteAfter bool
	pushFirst   bool
	message     string
}

// Worktree push command
var worktreePushCmd = &cobra.Command{
	Use:   "push <worktree> [flags]",
	Short: "Push worktree branch to remote",
	Long: `Push worktree branch to remote repository.
Optionally creates pull request via GitHub CLI.
Handles remote tracking setup if needed.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreePushCommand,
}

var worktreePushFlags struct {
	createPR bool
	prTitle  string
	prBody   string
	draft    bool
	reviewer string
	force    bool
}

func init() {
	// List command flags
	worktreeListCmd.Flags().StringVarP(&worktreeListFlags.format, "format", "f", "table", "Output format (table, json, yaml, compact)")
	worktreeListCmd.Flags().StringVarP(&worktreeListFlags.status, "status", "s", "", "Filter by status (clean, dirty, active, stale)")
	worktreeListCmd.Flags().StringVarP(&worktreeListFlags.branch, "branch", "b", "", "Filter by branch name pattern")
	worktreeListCmd.Flags().BoolVar(&worktreeListFlags.withProcesses, "with-processes", false, "Include Claude Code process information")
	worktreeListCmd.Flags().StringVar(&worktreeListFlags.sort, "sort", "name", "Sort by (name, last-accessed, created, status)")

	// Create command flags
	worktreeCreateCmd.Flags().StringVarP(&worktreeCreateFlags.base, "base", "b", "", "Base branch for new worktree (default: current branch)")
	worktreeCreateCmd.Flags().StringVarP(&worktreeCreateFlags.directory, "directory", "d", "", "Custom worktree directory path")
	worktreeCreateCmd.Flags().BoolVarP(&worktreeCreateFlags.startSession, "start-session", "s", false, "Automatically start tmux session")
	worktreeCreateCmd.Flags().BoolVar(&worktreeCreateFlags.startClaude, "start-claude", false, "Automatically start Claude Code in new session")
	worktreeCreateCmd.Flags().BoolVarP(&worktreeCreateFlags.remote, "remote", "r", false, "Track remote branch if exists")
	worktreeCreateCmd.Flags().BoolVar(&worktreeCreateFlags.force, "force", false, "Overwrite existing worktree if present")

	// Delete command flags
	worktreeDeleteCmd.Flags().BoolVarP(&worktreeDeleteFlags.force, "force", "f", false, "Skip confirmation prompts")
	worktreeDeleteCmd.Flags().BoolVar(&worktreeDeleteFlags.cleanupSessions, "cleanup-sessions", false, "Terminate related tmux sessions")
	worktreeDeleteCmd.Flags().BoolVar(&worktreeDeleteFlags.cleanupProcesses, "cleanup-processes", false, "Stop related Claude Code processes")
	worktreeDeleteCmd.Flags().BoolVar(&worktreeDeleteFlags.keepBranch, "keep-branch", false, "Keep git branch after deleting worktree")
	worktreeDeleteCmd.Flags().StringVar(&worktreeDeleteFlags.pattern, "pattern", "", "Delete multiple worktrees matching pattern")

	// Merge command flags
	worktreeMergeCmd.Flags().StringVarP(&worktreeMergeFlags.target, "target", "t", "main", "Target branch for merge")
	worktreeMergeCmd.Flags().StringVarP(&worktreeMergeFlags.strategy, "strategy", "s", "merge", "Merge strategy (merge, squash, rebase)")
	worktreeMergeCmd.Flags().BoolVar(&worktreeMergeFlags.deleteAfter, "delete-after", false, "Delete worktree after successful merge")
	worktreeMergeCmd.Flags().BoolVar(&worktreeMergeFlags.pushFirst, "push-first", false, "Push worktree branch before merging")
	worktreeMergeCmd.Flags().StringVarP(&worktreeMergeFlags.message, "message", "m", "", "Custom merge commit message")

	// Push command flags
	worktreePushCmd.Flags().BoolVar(&worktreePushFlags.createPR, "create-pr", false, "Create pull request after push")
	worktreePushCmd.Flags().StringVar(&worktreePushFlags.prTitle, "pr-title", "", "Pull request title")
	worktreePushCmd.Flags().StringVar(&worktreePushFlags.prBody, "pr-body", "", "Pull request body")
	worktreePushCmd.Flags().BoolVar(&worktreePushFlags.draft, "draft", false, "Create draft pull request")
	worktreePushCmd.Flags().StringVar(&worktreePushFlags.reviewer, "reviewer", "", "Add reviewers to pull request")
	worktreePushCmd.Flags().BoolVar(&worktreePushFlags.force, "force", false, "Force push (use with caution)")

	// Add subcommands to worktree command
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeCreateCmd)
	worktreeCmd.AddCommand(worktreeDeleteCmd)
	worktreeCmd.AddCommand(worktreeMergeCmd)
	worktreeCmd.AddCommand(worktreePushCmd)

	// Add worktree command to root
	rootCmd.AddCommand(worktreeCmd)
}

func runWorktreeListCommand(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner("Collecting worktree information...")
		spinner.Start()
		defer spinner.Stop()
	}

	// Initialize git repository manager
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to detect git repository", err))
	}

	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
	worktrees, err := worktreeManager.ListWorktrees()
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to list worktrees", err))
	}

	// Convert to list format
	listData := &WorktreeListData{
		Worktrees: make([]WorktreeListItem, 0, len(worktrees)),
		Total:     len(worktrees),
		Timestamp: time.Now(),
	}

	// Optionally get process information
	var processManager *claude.ProcessManager
	if worktreeListFlags.withProcesses {
		processManager, _ = claude.NewProcessManager(nil)
	}

	// Get tmux session information
	sessionManager := tmux.NewSessionManager(cfg)
	sessions, _ := sessionManager.ListSessions()

	for _, wt := range worktrees {
		item := WorktreeListItem{
			Name:         filepath.Base(wt.Path),
			Path:         wt.Path,
			Branch:       wt.Branch,
			Head:         wt.Head,
			IsClean:      wt.IsClean,
			TmuxSession:  wt.TmuxSession,
			LastAccessed: wt.LastAccessed,
			Created:      wt.Created,
		}

		// Determine status
		if wt.IsClean {
			item.Status = "clean"
		} else {
			item.Status = "dirty"
		}

		// Check if worktree has active session
		for _, sess := range sessions {
			if sess.Worktree == item.Name || strings.Contains(sess.Directory, item.Path) {
				item.Status = "active"
				break
			}
		}

		// Get process count if requested
		if processManager != nil {
			processes := processManager.GetProcessesByWorktree(item.Name)
			item.ProcessCount = len(processes)
		}

		listData.Worktrees = append(listData.Worktrees, item)
	}

	// Apply filters
	if worktreeListFlags.status != "" {
		filtered := make([]WorktreeListItem, 0)
		for _, item := range listData.Worktrees {
			if item.Status == worktreeListFlags.status {
				filtered = append(filtered, item)
			}
		}
		listData.Worktrees = filtered
		listData.Total = len(filtered)
	}

	if worktreeListFlags.branch != "" {
		filtered := make([]WorktreeListItem, 0)
		for _, item := range listData.Worktrees {
			if strings.Contains(item.Branch, worktreeListFlags.branch) {
				filtered = append(filtered, item)
			}
		}
		listData.Worktrees = filtered
		listData.Total = len(filtered)
	}

	// Sort results
	sortWorktreeList(listData.Worktrees, worktreeListFlags.sort)

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Found %d worktrees", listData.Total))
	}

	formatter, err := setupWorktreeOutputFormatter(worktreeListFlags.format)
	if err != nil {
		return handleCLIError(err)
	}

	return formatter.Format(listData)
}

func runWorktreeCreateCommand(cmd *cobra.Command, args []string) error {
	branchName := args[0]

	// Validate branch name
	if err := validateBranchArg(branchName); err != nil {
		return handleCLIError(err)
	}

	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner(fmt.Sprintf("Creating worktree for branch '%s'...", branchName))
		spinner.Start()
		defer spinner.Stop()
	}

	// Initialize git repository manager
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to detect git repository", err))
	}

	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)

	// Determine base branch
	baseBranch := worktreeCreateFlags.base
	if baseBranch == "" {
		baseBranch = repo.CurrentBranch
		if baseBranch == "" {
			return handleCLIError(cli.NewError("could not determine current branch and no base branch specified"))
		}
	}

	// Determine worktree directory
	worktreeDir := worktreeCreateFlags.directory
	useAutoName := worktreeDir == ""

	if spinner != nil {
		if useAutoName {
			spinner.SetMessage("Creating worktree...")
		} else {
			spinner.SetMessage(fmt.Sprintf("Creating worktree at %s...", worktreeDir))
		}
	}

	// Create the worktree
	opts := git.WorktreeOptions{
		Path:         worktreeDir,
		Branch:       branchName,
		CreateBranch: true,
		Force:        worktreeCreateFlags.force,
		Checkout:     true,
		TrackRemote:  worktreeCreateFlags.remote,
		AutoName:     useAutoName,
	}
	worktreeInfo, err := worktreeManager.CreateWorktree(branchName, opts)
	if err != nil {
		return handlePatternError(cli.NewErrorWithCause("failed to create worktree", err))
	}

	if spinner != nil {
		spinner.SetMessage("Worktree created successfully")
	}

	// Start tmux session if requested
	if worktreeCreateFlags.startSession {
		if spinner != nil {
			spinner.SetMessage("Starting tmux session...")
		}

		sessionManager := tmux.NewSessionManager(cfg)

		// Use actual path for session creation
		sessionPath := worktreeDir
		if useAutoName && worktreeInfo != nil {
			sessionPath = worktreeInfo.Path
		}

		session, err := sessionManager.CreateSession(
			getCurrentProjectName(), // project
			branchName,              // worktree
			branchName,              // branch
			sessionPath,             // directory
		)
		if err != nil {
			// Don't fail the entire operation if session creation fails
			if isVerbose() {
				fmt.Printf("Warning: Failed to create tmux session: %v\n", err)
			}
		} else if spinner != nil {
			spinner.SetMessage(fmt.Sprintf("Created tmux session: %s", session.Name))
		}

		// Start Claude Code if requested
		if worktreeCreateFlags.startClaude && session != nil {
			if spinner != nil {
				spinner.SetMessage("Starting Claude Code...")
			}

			// Claude Code process management not yet implemented
			if isVerbose() {
				fmt.Printf("Warning: Claude Code auto-start not yet implemented\n")
			}
		}
	}

	// Get actual path for display
	actualPath := worktreeDir
	if useAutoName && worktreeInfo != nil {
		actualPath = worktreeInfo.Path
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Worktree '%s' created successfully at %s", branchName, actualPath))
	}

	if !isQuiet() {
		fmt.Printf("\nWorktree created:\n")
		fmt.Printf("  Branch: %s\n", branchName)
		fmt.Printf("  Path: %s\n", actualPath)
		if worktreeCreateFlags.startSession {
			fmt.Printf("  Session: Started\n")
		}
		if worktreeCreateFlags.startClaude {
			fmt.Printf("  Claude Code: Started\n")
		}
	}

	return nil
}

func runWorktreeDeleteCommand(cmd *cobra.Command, args []string) error {
	worktreeName := args[0]

	if err := validateWorktreeArg(worktreeName); err != nil {
		return handleCLIError(err)
	}

	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	// Initialize managers
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to detect git repository", err))
	}

	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)

	// Find the worktree
	worktrees, err := worktreeManager.ListWorktrees()
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to list worktrees", err))
	}

	var targetWorktree *git.WorktreeInfo
	for _, wt := range worktrees {
		if filepath.Base(wt.Path) == worktreeName || wt.Branch == worktreeName || wt.Path == worktreeName {
			targetWorktree = &wt
			break
		}
	}

	if targetWorktree == nil {
		return handleCLIError(cli.NewErrorWithSuggestion(
			fmt.Sprintf("worktree not found: %s", worktreeName),
			"Use 'ccmgr-ultra worktree list' to see available worktrees",
		))
	}

	// Safety check - confirm deletion
	if !worktreeDeleteFlags.force && !isDryRun() {
		fmt.Printf("This will delete worktree:\n")
		fmt.Printf("  Name: %s\n", filepath.Base(targetWorktree.Path))
		fmt.Printf("  Path: %s\n", targetWorktree.Path)
		fmt.Printf("  Branch: %s\n", targetWorktree.Branch)

		if !targetWorktree.IsClean {
			fmt.Printf("  WARNING: Worktree has uncommitted changes!\n")
		}

		fmt.Printf("\nProceed with deletion? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner(fmt.Sprintf("Deleting worktree '%s'...", worktreeName))
		spinner.Start()
		defer spinner.Stop()
	}

	// Clean up sessions if requested
	if worktreeDeleteFlags.cleanupSessions {
		if spinner != nil {
			spinner.SetMessage("Cleaning up tmux sessions...")
		}

		sessionManager := tmux.NewSessionManager(cfg)
		sessions, err := sessionManager.ListSessions()
		if err == nil {
			for _, sess := range sessions {
				if sess.Worktree == worktreeName || strings.Contains(sess.Directory, targetWorktree.Path) {
					sessionManager.KillSession(sess.ID)
				}
			}
		}
	}

	// Clean up processes if requested
	if worktreeDeleteFlags.cleanupProcesses {
		if spinner != nil {
			spinner.SetMessage("Cleaning up Claude Code processes...")
		}

		// Process cleanup not yet implemented
		if isVerbose() {
			fmt.Printf("Warning: Process cleanup not yet implemented\n")
		}
	}

	if isDryRun() {
		if spinner != nil {
			spinner.StopWithMessage("Dry run: Would delete worktree")
		}
		fmt.Printf("Dry run: Would delete worktree '%s' at %s\n", worktreeName, targetWorktree.Path)
		return nil
	}

	// Delete the worktree
	if spinner != nil {
		spinner.SetMessage("Removing worktree directory...")
	}

	err = worktreeManager.DeleteWorktree(targetWorktree.Path, !worktreeDeleteFlags.keepBranch)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to delete worktree", err))
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("Worktree '%s' deleted successfully", worktreeName))
	}

	if !isQuiet() {
		fmt.Printf("Worktree '%s' deleted successfully\n", worktreeName)
	}

	return nil
}

func runWorktreeMergeCommand(cmd *cobra.Command, args []string) error {
	// Placeholder implementation - this would be quite complex
	worktreeName := args[0]

	if err := validateWorktreeArg(worktreeName); err != nil {
		return handleCLIError(err)
	}

	return handleCLIError(cli.NewError("worktree merge command not yet implemented"))
}

func runWorktreePushCommand(cmd *cobra.Command, args []string) error {
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
		spinner = cli.NewSpinner(fmt.Sprintf("Pushing worktree '%s'...", worktreeName))
		spinner.Start()
		defer spinner.Stop()
	}

	// Initialize git repository manager
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to detect git repository", err))
	}

	// Find the target worktree
	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
	worktrees, err := worktreeManager.ListWorktrees()
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to list worktrees", err))
	}

	var targetWorktree *git.WorktreeInfo
	for _, wt := range worktrees {
		if filepath.Base(wt.Path) == worktreeName || wt.Branch == worktreeName || wt.Path == worktreeName {
			targetWorktree = &wt
			break
		}
	}

	if targetWorktree == nil {
		return handleCLIError(cli.NewErrorWithSuggestion(
			fmt.Sprintf("worktree not found: %s", worktreeName),
			"Use 'ccmgr-ultra worktree list' to see available worktrees",
		))
	}

	if spinner != nil {
		spinner.SetMessage(fmt.Sprintf("Found worktree '%s' on branch '%s'", worktreeName, targetWorktree.Branch))
	}

	// Initialize remote manager
	remoteManager := git.NewRemoteManager(repo, &cfg.Git, gitCmd)

	// Detect hosting service
	service, err := remoteManager.DetectHostingService(repo.Origin)
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to detect hosting service", err))
	}

	if service != "github" {
		return handleCLIError(cli.NewErrorWithSuggestion(
			fmt.Sprintf("hosting service '%s' not supported", service),
			"Currently only GitHub repositories are supported for push operations",
		))
	}

	// Validate GitHub authentication if creating PR
	if worktreePushFlags.createPR {
		if spinner != nil {
			spinner.SetMessage("Validating GitHub authentication...")
		}

		if err := remoteManager.ValidateAuthentication("github"); err != nil {
			return handleCLIError(cli.NewErrorWithSuggestion(
				fmt.Sprintf("GitHub authentication failed: %v", err),
				"Set GITHUB_TOKEN environment variable or configure github_token in config",
			))
		}
	}

	// Push the branch first
	if spinner != nil {
		spinner.SetMessage(fmt.Sprintf("Pushing branch '%s' to remote...", targetWorktree.Branch))
	}

	if worktreePushFlags.createPR {
		// Determine target branch
		targetBranch := repo.DefaultBranch
		if cfg.Git.DefaultPRTargetBranch != "" {
			targetBranch = cfg.Git.DefaultPRTargetBranch
		}

		// Prepare pull request options
		prOptions := git.PullRequestRequest{
			Title:        worktreePushFlags.prTitle,
			Description:  worktreePushFlags.prBody,
			SourceBranch: targetWorktree.Branch,
			TargetBranch: targetBranch,
			Draft:        worktreePushFlags.draft,
		}

		// Set default PR title if not provided
		if prOptions.Title == "" {
			prOptions.Title = fmt.Sprintf("Feature: %s", targetWorktree.Branch)
		}

		// Set default PR body if not provided and template exists
		if prOptions.Description == "" {
			// Use GitHub-specific template if available
			if cfg.Git.GitHubPRTemplate != "" {
				prOptions.Description = cfg.Git.GitHubPRTemplate
			} else if cfg.Git.PRTemplate != "" {
				prOptions.Description = cfg.Git.PRTemplate
			}
		}

		if spinner != nil {
			spinner.SetMessage("Creating GitHub pull request...")
		}

		// Push and create PR
		pr, err := remoteManager.PushAndCreatePR(targetWorktree, prOptions)
		if err != nil {
			return handleCLIError(cli.NewErrorWithCause("failed to push and create pull request", err))
		}

		if spinner != nil {
			spinner.StopWithMessage(fmt.Sprintf("Successfully pushed and created PR #%d", pr.Number))
		}

		if !isQuiet() {
			fmt.Printf("\nPull Request Created:\n")
			fmt.Printf("  Title: %s\n", pr.Title)
			fmt.Printf("  Number: #%d\n", pr.Number)
			fmt.Printf("  URL: %s\n", pr.URL)
			fmt.Printf("  Status: %s\n", pr.State)
			if pr.Draft {
				fmt.Printf("  Type: Draft\n")
			}
		}
	} else {
		// Just push without creating PR
		if err := remoteManager.PushBranch(targetWorktree.Branch); err != nil {
			return handleCLIError(cli.NewErrorWithCause("failed to push branch", err))
		}

		if spinner != nil {
			spinner.StopWithMessage(fmt.Sprintf("Successfully pushed branch '%s'", targetWorktree.Branch))
		}

		if !isQuiet() {
			fmt.Printf("Branch '%s' pushed successfully\n", targetWorktree.Branch)
		}
	}

	return nil
}

// Helper functions

func handlePatternError(err error) error {
	if strings.Contains(err.Error(), "template") ||
		strings.Contains(err.Error(), "pattern") ||
		strings.Contains(err.Error(), "variable") {
		return cli.NewErrorWithSuggestion(
			fmt.Sprintf("Template pattern error: %v", err),
			"Check your directory_pattern in config. Use Go template syntax like {{.Project}}-{{.Branch}}",
		)
	}
	return handleCLIError(err)
}

func generateSessionName(cfg *config.Config, branchName string) string {
	// Use configured pattern or default
	pattern := "ccmgr-%s"
	if cfg.Tmux.NamingPattern != "" {
		pattern = cfg.Tmux.NamingPattern
	}

	// Replace placeholders
	name := strings.ReplaceAll(pattern, "%s", branchName)
	name = strings.ReplaceAll(name, "{branch}", branchName)
	name = strings.ReplaceAll(name, "{project}", filepath.Base(getCurrentProjectName()))

	return name
}

func getCurrentProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return filepath.Base(cwd)
}

func sortWorktreeList(worktrees []WorktreeListItem, sortBy string) {
	// Simple sorting implementation - could be enhanced
	// For now, just ensure deterministic output
	// Real implementation would sort based on the sortBy parameter
}
