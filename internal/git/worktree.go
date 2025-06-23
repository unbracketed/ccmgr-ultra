package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// WorktreeManager handles git worktree operations
type WorktreeManager struct {
	repo       *Repository
	patternMgr *PatternManager
	gitCmd     GitInterface
	config     *config.Config
	repoMgr    *RepositoryManager
}

// WorktreeOptions for worktree creation
type WorktreeOptions struct {
	Path         string
	Branch       string
	CreateBranch bool
	Force        bool
	Checkout     bool
	Remote       string
	TrackRemote  bool
	AutoName     bool // Use pattern manager for naming
}

// NewWorktreeManager creates a new WorktreeManager
func NewWorktreeManager(repo *Repository, config *config.Config, gitCmd GitInterface) *WorktreeManager {
	if gitCmd == nil {
		gitCmd = NewGitCmd()
	}

	repoMgr := NewRepositoryManager(gitCmd)

	// Create worktree config that uses Git config values for compatibility
	worktreeConfig := config.Worktree
	if config.Git.DirectoryPattern != "" {
		worktreeConfig.DirectoryPattern = config.Git.DirectoryPattern
	}
	if config.Git.AutoDirectory != worktreeConfig.AutoDirectory {
		worktreeConfig.AutoDirectory = config.Git.AutoDirectory
	}

	patternMgr := NewPatternManager(&worktreeConfig)

	return &WorktreeManager{
		repo:       repo,
		patternMgr: patternMgr,
		gitCmd:     gitCmd,
		config:     config,
		repoMgr:    repoMgr,
	}
}

// CreateWorktree creates a new git worktree
func (wm *WorktreeManager) CreateWorktree(branch string, opts WorktreeOptions) (*WorktreeInfo, error) {
	if branch == "" {
		return nil, fmt.Errorf("branch name cannot be empty")
	}

	// Validate repository state
	if err := wm.repoMgr.ValidateRepositoryState(wm.repo); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Validate base directory configuration
	if err := wm.patternMgr.ValidateBaseDirectory(wm.patternMgr.config.BaseDirectory, wm.repo.RootPath); err != nil {
		return nil, fmt.Errorf("invalid base directory configuration: %w", err)
	}

	// Determine target path
	targetPath := opts.Path
	if targetPath == "" || opts.AutoName {
		projectName := wm.getProjectName()
		generatedPath, err := wm.patternMgr.GenerateWorktreePath(branch, projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate worktree path: %w", err)
		}
		if targetPath == "" {
			targetPath = generatedPath
		}
	}

	// Validate target path
	if err := wm.validateWorktreePath(targetPath); err != nil {
		return nil, fmt.Errorf("invalid worktree path: %w", err)
	}

	// Check if path is available
	if err := wm.patternMgr.CheckPathAvailable(targetPath); err != nil && !opts.Force {
		return nil, fmt.Errorf("path not available: %w", err)
	}

	// Check if branch already has a worktree
	if err := wm.checkBranchWorktreeConflict(branch); err != nil && !opts.Force {
		return nil, fmt.Errorf("branch conflict: %w", err)
	}

	// Create branch if needed
	if opts.CreateBranch {
		if err := wm.createBranchForWorktree(branch, opts); err != nil {
			return nil, fmt.Errorf("failed to create branch: %w", err)
		}
	}

	// Create the worktree
	if err := wm.executeWorktreeCreate(targetPath, branch, opts); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Get worktree information
	worktreeInfo, err := wm.GetWorktreeInfo(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree info: %w", err)
	}

	// Create tmux session if configured
	if wm.config.Tmux.SessionPrefix != "" {
		if err := wm.createTmuxSession(worktreeInfo); err != nil {
			// Log warning but don't fail worktree creation
			fmt.Printf("Warning: failed to create tmux session: %v\n", err)
		}
	}

	return worktreeInfo, nil
}

// ListWorktrees lists all worktrees in the repository
func (wm *WorktreeManager) ListWorktrees() ([]WorktreeInfo, error) {
	// Refresh repository information
	if err := wm.repoMgr.ValidateRepositoryState(wm.repo); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Get worktrees from repository
	worktrees, err := wm.repoMgr.getWorktrees(wm.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get worktrees: %w", err)
	}

	// Enhance worktree information
	for i := range worktrees {
		if err := wm.enhanceWorktreeInfo(&worktrees[i]); err != nil {
			// Log warning but continue with other worktrees
			fmt.Printf("Warning: failed to enhance worktree info for %s: %v\n", worktrees[i].Path, err)
		}
	}

	return worktrees, nil
}

// DeleteWorktree deletes a git worktree
func (wm *WorktreeManager) DeleteWorktree(path string, force bool) error {
	if path == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	// Get worktree info before deletion
	worktreeInfo, err := wm.GetWorktreeInfo(path)
	if err != nil {
		if !force {
			return fmt.Errorf("failed to get worktree info: %w", err)
		}
		// Continue with deletion even if we can't get info
	}

	// Check if worktree has uncommitted changes
	if worktreeInfo != nil && worktreeInfo.HasUncommitted && !force {
		return fmt.Errorf("worktree has uncommitted changes, use force to delete anyway")
	}

	// Backup worktree if configured
	if wm.config.Worktree.CleanupOnMerge {
		if err := wm.backupWorktree(path); err != nil {
			fmt.Printf("Warning: failed to backup worktree: %v\n", err)
		}
	}

	// Remove tmux session if it exists
	if worktreeInfo != nil && worktreeInfo.TmuxSession != "" {
		if err := wm.removeTmuxSession(worktreeInfo.TmuxSession); err != nil {
			fmt.Printf("Warning: failed to remove tmux session: %v\n", err)
		}
	}

	// Execute worktree removal
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	if _, err := wm.gitCmd.Execute(wm.repo.RootPath, args...); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Clean up any remaining directory if it exists
	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("Warning: failed to remove directory %s: %v\n", path, err)
		}
	}

	return nil
}

// GetWorktreeInfo gets detailed information about a specific worktree
func (wm *WorktreeManager) GetWorktreeInfo(path string) (*WorktreeInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("worktree path cannot be empty")
	}

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("worktree path does not exist: %s", path)
	}

	// Check if it's a git worktree
	if !wm.repoMgr.IsGitRepository(path) {
		return nil, fmt.Errorf("path is not a git repository: %s", path)
	}

	// Get basic worktree information
	worktreeInfo := &WorktreeInfo{
		Path: path,
	}

	// Get current branch
	branch, err := wm.gitCmd.Execute(path, "branch", "--show-current")
	if err != nil {
		// Try alternative method for detached HEAD
		head, err2 := wm.gitCmd.Execute(path, "rev-parse", "--abbrev-ref", "HEAD")
		if err2 != nil {
			return nil, fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = head
	}
	worktreeInfo.Branch = branch

	// Get HEAD commit
	head, err := wm.gitCmd.Execute(path, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}
	worktreeInfo.Head = head

	// Enhance with additional information
	if err := wm.enhanceWorktreeInfo(worktreeInfo); err != nil {
		return nil, fmt.Errorf("failed to enhance worktree info: %w", err)
	}

	return worktreeInfo, nil
}

// RefreshWorktreeStatus refreshes the status information for a worktree
func (wm *WorktreeManager) RefreshWorktreeStatus(path string) (*WorktreeInfo, error) {
	return wm.GetWorktreeInfo(path)
}

// PruneWorktrees removes stale worktree references
func (wm *WorktreeManager) PruneWorktrees() error {
	_, err := wm.gitCmd.Execute(wm.repo.RootPath, "worktree", "prune")
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}

	return nil
}

// MoveWorktree moves a worktree to a new location
func (wm *WorktreeManager) MoveWorktree(oldPath, newPath string) error {
	if oldPath == "" || newPath == "" {
		return fmt.Errorf("both old and new paths must be specified")
	}

	// Check if new path is available
	if err := wm.patternMgr.CheckPathAvailable(newPath); err != nil {
		return fmt.Errorf("new path not available: %w", err)
	}

	// Execute worktree move
	_, err := wm.gitCmd.Execute(wm.repo.RootPath, "worktree", "move", oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to move worktree: %w", err)
	}

	return nil
}

// Internal helper methods

// getProjectName extracts the project name from the repository
func (wm *WorktreeManager) getProjectName() string {
	if wm.repo.Origin != "" {
		// Extract from remote URL
		for _, remote := range wm.repo.Remotes {
			if remote.Name == "origin" && remote.Repo != "" {
				return remote.Repo
			}
		}
	}

	// Fall back to directory name
	return filepath.Base(wm.repo.RootPath)
}

// validateWorktreePath validates a worktree path
func (wm *WorktreeManager) validateWorktreePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Check if path is within repository (prevent creating worktree inside repository)
	repoPath, err := filepath.Abs(wm.repo.RootPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute repository path: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if strings.HasPrefix(absPath, repoPath) {
		return fmt.Errorf("worktree path cannot be inside repository: %s", path)
	}

	return nil
}

// checkBranchWorktreeConflict checks if a branch is already used by another worktree
func (wm *WorktreeManager) checkBranchWorktreeConflict(branch string) error {
	worktrees, err := wm.repoMgr.getWorktrees(wm.repo)
	if err != nil {
		return fmt.Errorf("failed to get existing worktrees: %w", err)
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return fmt.Errorf("branch '%s' is already checked out in worktree: %s", branch, wt.Path)
		}
	}

	return nil
}

// createBranchForWorktree creates a new branch for the worktree
func (wm *WorktreeManager) createBranchForWorktree(branch string, opts WorktreeOptions) error {
	// Check if branch already exists
	_, err := wm.gitCmd.Execute(wm.repo.RootPath, "rev-parse", "--verify", branch)
	if err == nil {
		// Branch exists, don't create
		return nil
	}

	// Determine source branch
	sourceBranch := wm.repo.DefaultBranch
	if opts.Remote != "" && opts.TrackRemote {
		sourceBranch = fmt.Sprintf("%s/%s", opts.Remote, branch)
	}

	// Create branch
	_, err = wm.gitCmd.Execute(wm.repo.RootPath, "branch", branch, sourceBranch)
	if err != nil {
		return fmt.Errorf("failed to create branch %s from %s: %w", branch, sourceBranch, err)
	}

	return nil
}

// executeWorktreeCreate executes the git worktree add command
func (wm *WorktreeManager) executeWorktreeCreate(path, branch string, opts WorktreeOptions) error {
	args := []string{"worktree", "add"}

	if opts.Force {
		args = append(args, "--force")
	}

	if !opts.Checkout {
		args = append(args, "--no-checkout")
	}

	args = append(args, path, branch)

	_, err := wm.gitCmd.Execute(wm.repo.RootPath, args...)
	if err != nil {
		return fmt.Errorf("git worktree add failed: %w", err)
	}

	return nil
}

// enhanceWorktreeInfo adds additional information to worktree info
func (wm *WorktreeManager) enhanceWorktreeInfo(wt *WorktreeInfo) error {
	// Check if working directory is clean
	status, err := wm.gitCmd.Execute(wt.Path, "status", "--porcelain")
	if err == nil {
		wt.IsClean = strings.TrimSpace(status) == ""
		wt.HasUncommitted = !wt.IsClean
	}

	// Get last commit information
	if wt.Head != "" {
		commit, err := wm.repoMgr.getCommitInfo(wt.Path, wt.Head)
		if err == nil {
			wt.LastCommit = *commit
		}
	}

	// Set timestamps
	if stat, err := os.Stat(wt.Path); err == nil {
		wt.LastAccessed = stat.ModTime()
		// For created time, we'll use the commit time or file time
		if !wt.LastCommit.Date.IsZero() {
			wt.Created = wt.LastCommit.Date
		} else {
			wt.Created = stat.ModTime()
		}
	}

	// Check for associated tmux session
	wt.TmuxSession = wm.getTmuxSessionName(wt)

	return nil
}

// getTmuxSessionName generates or finds the tmux session name for a worktree
func (wm *WorktreeManager) getTmuxSessionName(wt *WorktreeInfo) string {
	if wm.config.Tmux.SessionPrefix == "" {
		return ""
	}

	// Generate session name based on tmux naming pattern
	context := map[string]string{
		"prefix":   wm.config.Tmux.SessionPrefix,
		"project":  wm.getProjectName(),
		"worktree": filepath.Base(wt.Path),
		"branch":   wt.Branch,
	}

	// Simple pattern substitution for tmux session names
	sessionName := wm.config.Tmux.NamingPattern
	for key, value := range context {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		sessionName = strings.ReplaceAll(sessionName, placeholder, value)
	}

	// Sanitize session name
	sessionName = strings.ReplaceAll(sessionName, " ", "-")
	sessionName = strings.ReplaceAll(sessionName, "/", "-")

	// Truncate if too long
	if len(sessionName) > wm.config.Tmux.MaxSessionName {
		sessionName = sessionName[:wm.config.Tmux.MaxSessionName]
	}

	return sessionName
}

// createTmuxSession creates a tmux session for the worktree
func (wm *WorktreeManager) createTmuxSession(wt *WorktreeInfo) error {
	if wt.TmuxSession == "" {
		return fmt.Errorf("no tmux session name generated")
	}

	// Create tmux session
	// This would integrate with the tmux module - for now just a placeholder
	fmt.Printf("Creating tmux session: %s for worktree: %s\n", wt.TmuxSession, wt.Path)

	return nil
}

// removeTmuxSession removes a tmux session
func (wm *WorktreeManager) removeTmuxSession(sessionName string) error {
	// This would integrate with the tmux module - for now just a placeholder
	fmt.Printf("Removing tmux session: %s\n", sessionName)

	return nil
}

// backupWorktree creates a backup of the worktree before deletion
func (wm *WorktreeManager) backupWorktree(path string) error {
	// Create backup directory
	backupDir := filepath.Join(os.TempDir(), "ccmgr-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create timestamped backup name
	timestamp := time.Now().Format("20060102-150405")
	worktreeName := filepath.Base(path)
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s-%s.tar.gz", worktreeName, timestamp))

	// For now, just log the backup action
	fmt.Printf("Backing up worktree %s to %s\n", path, backupPath)

	return nil
}

// GetWorktreeStats returns statistics about worktrees
func (wm *WorktreeManager) GetWorktreeStats() (*WorktreeStats, error) {
	worktrees, err := wm.ListWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	stats := &WorktreeStats{
		Total: len(worktrees),
	}

	for _, wt := range worktrees {
		if wt.IsClean {
			stats.Clean++
		} else {
			stats.Dirty++
		}

		if wt.TmuxSession != "" {
			stats.WithTmuxSession++
		}
	}

	return stats, nil
}

// WorktreeStats contains statistics about worktrees
type WorktreeStats struct {
	Total           int
	Clean           int
	Dirty           int
	WithTmuxSession int
}

// CleanupOldWorktrees removes old unused worktrees based on configuration
func (wm *WorktreeManager) CleanupOldWorktrees() error {
	if !wm.config.Tmux.AutoCleanup {
		return nil
	}

	worktrees, err := wm.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	cutoffTime := time.Now().Add(-wm.config.Tmux.CleanupAge)

	for _, wt := range worktrees {
		if wt.LastAccessed.Before(cutoffTime) && wt.IsClean {
			fmt.Printf("Cleaning up old worktree: %s (last accessed: %s)\n",
				wt.Path, wt.LastAccessed.Format("2006-01-02 15:04:05"))

			if err := wm.DeleteWorktree(wt.Path, false); err != nil {
				fmt.Printf("Warning: failed to cleanup worktree %s: %v\n", wt.Path, err)
			}
		}
	}

	return nil
}
