package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// GitInterface defines the interface for git command execution
type GitInterface interface {
	Execute(dir string, args ...string) (string, error)
	ExecuteWithInput(dir, input string, args ...string) (string, error)
}

// GitCmd implements GitInterface using the git binary
type GitCmd struct {
	gitPath string
}

// NewGitCmd creates a new GitCmd instance
func NewGitCmd() *GitCmd {
	gitPath := "git"
	if path, err := exec.LookPath("git"); err == nil {
		gitPath = path
	}
	return &GitCmd{gitPath: gitPath}
}

// Execute runs a git command in the specified directory
func (g *GitCmd) Execute(dir string, args ...string) (string, error) {
	cmd := exec.Command(g.gitPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s: %w", string(output), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ExecuteWithInput runs a git command with stdin input
func (g *GitCmd) ExecuteWithInput(dir, input string, args ...string) (string, error) {
	cmd := exec.Command(g.gitPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s: %w", string(output), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Repository represents a git repository
type Repository struct {
	Path          string
	Origin        string
	DefaultBranch string
	CurrentBranch string
	IsClean       bool
	Remotes       []Remote
	Worktrees     []WorktreeInfo
	RootPath      string
}

// Remote represents a git remote
type Remote struct {
	Name     string
	URL      string
	Host     string
	Owner    string
	Repo     string
	Protocol string
}

// WorktreeInfo represents a git worktree
type WorktreeInfo struct {
	Path           string
	Branch         string
	Head           string
	IsClean        bool
	HasUncommitted bool
	LastCommit     CommitInfo
	TmuxSession    string
	Created        time.Time
	LastAccessed   time.Time
}

// CommitInfo represents a git commit
type CommitInfo struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
	Files   []string
}

// RepositoryManager handles git repository detection and validation
type RepositoryManager struct {
	gitCmd GitInterface
}

// NewRepositoryManager creates a new RepositoryManager
func NewRepositoryManager(gitCmd GitInterface) *RepositoryManager {
	if gitCmd == nil {
		gitCmd = NewGitCmd()
	}
	return &RepositoryManager{gitCmd: gitCmd}
}

// DetectRepository detects if the given path is a git repository
func (rm *RepositoryManager) DetectRepository(path string) (*Repository, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check if path is within a git repository
	if !rm.IsGitRepository(path) {
		return nil, fmt.Errorf("not a git repository: %s", path)
	}

	// Get repository root
	rootPath, err := rm.gitCmd.Execute(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	repo := &Repository{
		Path:     path,
		RootPath: rootPath,
	}

	// Get repository information
	if err := rm.populateRepositoryInfo(repo); err != nil {
		return nil, fmt.Errorf("failed to populate repository info: %w", err)
	}

	return repo, nil
}

// IsGitRepository checks if the given path is within a git repository
func (rm *RepositoryManager) IsGitRepository(path string) bool {
	_, err := rm.gitCmd.Execute(path, "rev-parse", "--git-dir")
	return err == nil
}

// GetRepositoryInfo gets comprehensive information about a repository
func (rm *RepositoryManager) GetRepositoryInfo(path string) (*Repository, error) {
	repo, err := rm.DetectRepository(path)
	if err != nil {
		return nil, err
	}

	// Validate repository state
	if err := rm.ValidateRepositoryState(repo); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	return repo, nil
}

// ValidateRepositoryState validates the current state of the repository
func (rm *RepositoryManager) ValidateRepositoryState(repo *Repository) error {
	if repo == nil {
		return fmt.Errorf("repository is nil")
	}

	// Check if repository exists
	if _, err := os.Stat(repo.RootPath); os.IsNotExist(err) {
		return fmt.Errorf("repository path does not exist: %s", repo.RootPath)
	}

	// Check if it's still a git repository
	if !rm.IsGitRepository(repo.RootPath) {
		return fmt.Errorf("path is no longer a git repository: %s", repo.RootPath)
	}

	// Refresh repository information
	if err := rm.populateRepositoryInfo(repo); err != nil {
		return fmt.Errorf("failed to refresh repository info: %w", err)
	}

	return nil
}

// GetRemoteInfo gets information about a specific remote
func (rm *RepositoryManager) GetRemoteInfo(repo *Repository, remoteName string) (*Remote, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	// Get remote URL
	url, err := rm.gitCmd.Execute(repo.RootPath, "remote", "get-url", remoteName)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote URL for %s: %w", remoteName, err)
	}

	remote := &Remote{
		Name: remoteName,
		URL:  url,
	}

	// Parse remote URL to extract components
	if err := rm.parseRemoteURL(remote); err != nil {
		return nil, fmt.Errorf("failed to parse remote URL: %w", err)
	}

	return remote, nil
}

// populateRepositoryInfo fills in repository information
func (rm *RepositoryManager) populateRepositoryInfo(repo *Repository) error {
	// Get current branch
	currentBranch, err := rm.gitCmd.Execute(repo.RootPath, "branch", "--show-current")
	if err != nil {
		// Try alternative method for older git versions or detached HEAD
		head, err2 := rm.gitCmd.Execute(repo.RootPath, "rev-parse", "--abbrev-ref", "HEAD")
		if err2 != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		currentBranch = head
	}
	repo.CurrentBranch = currentBranch

	// Get default branch
	defaultBranch, err := rm.getDefaultBranch(repo)
	if err != nil {
		// Fall back to common defaults
		for _, branch := range []string{"main", "master", "develop"} {
			if rm.branchExists(repo, branch) {
				defaultBranch = branch
				break
			}
		}
		if defaultBranch == "" {
			defaultBranch = "main" // ultimate fallback
		}
	}
	repo.DefaultBranch = defaultBranch

	// Check if working directory is clean
	status, err := rm.gitCmd.Execute(repo.RootPath, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to get repository status: %w", err)
	}
	repo.IsClean = strings.TrimSpace(status) == ""

	// Get remotes
	remotes, err := rm.getRemotes(repo)
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}
	repo.Remotes = remotes

	// Set origin if available
	for _, remote := range remotes {
		if remote.Name == "origin" {
			repo.Origin = remote.URL
			break
		}
	}

	// Get worktrees
	worktrees, err := rm.getWorktrees(repo)
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %w", err)
	}
	repo.Worktrees = worktrees

	return nil
}

// getDefaultBranch attempts to determine the default branch
func (rm *RepositoryManager) getDefaultBranch(repo *Repository) (string, error) {
	// Try to get default branch from remote HEAD
	output, err := rm.gitCmd.Execute(repo.RootPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		parts := strings.Split(output, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Try to get from git config
	output, err = rm.gitCmd.Execute(repo.RootPath, "config", "--get", "init.defaultBranch")
	if err == nil && output != "" {
		return output, nil
	}

	return "", fmt.Errorf("could not determine default branch")
}

// branchExists checks if a branch exists
func (rm *RepositoryManager) branchExists(repo *Repository, branch string) bool {
	_, err := rm.gitCmd.Execute(repo.RootPath, "rev-parse", "--verify", branch)
	return err == nil
}

// getRemotes gets all remotes for the repository
func (rm *RepositoryManager) getRemotes(repo *Repository) ([]Remote, error) {
	output, err := rm.gitCmd.Execute(repo.RootPath, "remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []Remote{}, nil
	}

	remoteMap := make(map[string]*Remote)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse format: "remotename url (fetch|push)"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			url := parts[1]

			if _, exists := remoteMap[name]; !exists {
				remote := &Remote{
					Name: name,
					URL:  url,
				}
				if err := rm.parseRemoteURL(remote); err == nil {
					remoteMap[name] = remote
				}
			}
		}
	}

	remotes := make([]Remote, 0, len(remoteMap))
	for _, remote := range remoteMap {
		remotes = append(remotes, *remote)
	}

	return remotes, nil
}

// parseRemoteURL parses a remote URL to extract host, owner, and repo
func (rm *RepositoryManager) parseRemoteURL(remote *Remote) error {
	url := remote.URL

	// Handle SSH URLs (git@host:owner/repo.git)
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 4 {
		remote.Protocol = "ssh"
		remote.Host = matches[1]
		remote.Owner = matches[2]
		remote.Repo = matches[3]
		return nil
	}

	// Handle HTTPS URLs (https://host/owner/repo.git)
	httpsPattern := regexp.MustCompile(`^https://([^/]+)/([^/]+)/(.+?)(?:\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 4 {
		remote.Protocol = "https"
		remote.Host = matches[1]
		remote.Owner = matches[2]
		remote.Repo = matches[3]
		return nil
	}

	// Handle HTTP URLs (http://host/owner/repo.git)
	httpPattern := regexp.MustCompile(`^http://([^/]+)/([^/]+)/(.+?)(?:\.git)?$`)
	if matches := httpPattern.FindStringSubmatch(url); len(matches) == 4 {
		remote.Protocol = "http"
		remote.Host = matches[1]
		remote.Owner = matches[2]
		remote.Repo = matches[3]
		return nil
	}

	return fmt.Errorf("unsupported remote URL format: %s", url)
}

// getWorktrees gets all worktrees for the repository
func (rm *RepositoryManager) getWorktrees(repo *Repository) ([]WorktreeInfo, error) {
	output, err := rm.gitCmd.Execute(repo.RootPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []WorktreeInfo{}, nil
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	// Add the last worktree if any
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	// Populate additional information for each worktree
	for i := range worktrees {
		if err := rm.populateWorktreeInfo(&worktrees[i]); err != nil {
			// Log error but continue with other worktrees
			continue
		}
	}

	return worktrees, nil
}

// populateWorktreeInfo fills in additional information for a worktree
func (rm *RepositoryManager) populateWorktreeInfo(wt *WorktreeInfo) error {
	// Check if working directory is clean
	status, err := rm.gitCmd.Execute(wt.Path, "status", "--porcelain")
	if err == nil {
		wt.IsClean = strings.TrimSpace(status) == ""
		wt.HasUncommitted = !wt.IsClean
	}

	// Get last commit information
	if wt.Head != "" {
		commit, err := rm.getCommitInfo(wt.Path, wt.Head)
		if err == nil {
			wt.LastCommit = *commit
		}
	}

	// Set timestamps (would be enhanced with actual file system info)
	if stat, err := os.Stat(wt.Path); err == nil {
		wt.LastAccessed = stat.ModTime()
	}

	return nil
}

// getCommitInfo gets information about a specific commit
func (rm *RepositoryManager) getCommitInfo(repoPath, commitHash string) (*CommitInfo, error) {
	if commitHash == "" {
		return nil, fmt.Errorf("commit hash is empty")
	}

	// Get commit information
	format := "--pretty=format:%H%n%an%n%at%n%s"
	output, err := rm.gitCmd.Execute(repoPath, "show", "--no-patch", format, commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 4 {
		return nil, fmt.Errorf("unexpected git show output format")
	}

	// Parse timestamp
	timestamp, err := time.Parse("1136239445", lines[2])
	if err != nil {
		timestamp = time.Now() // fallback
	}

	commit := &CommitInfo{
		Hash:    lines[0],
		Author:  lines[1],
		Date:    timestamp,
		Message: lines[3],
	}

	// Get files changed in commit
	filesOutput, err := rm.gitCmd.Execute(repoPath, "show", "--name-only", "--pretty=format:", commitHash)
	if err == nil {
		files := strings.Split(strings.TrimSpace(filesOutput), "\n")
		var cleanFiles []string
		for _, file := range files {
			if strings.TrimSpace(file) != "" {
				cleanFiles = append(cleanFiles, strings.TrimSpace(file))
			}
		}
		commit.Files = cleanFiles
	}

	return commit, nil
}
