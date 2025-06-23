package git

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GitOperations handles low-level git operations
type GitOperations struct {
	repo   *Repository
	gitCmd GitInterface
}

// BranchInfo represents a git branch
type BranchInfo struct {
	Name     string
	Remote   string
	Upstream string
	Current  bool
	Head     string
	Behind   int
	Ahead    int
}

// MergeResult represents the result of a merge operation
type MergeResult struct {
	Success      bool
	Conflicts    []string
	FilesChanged int
	CommitHash   string
	Message      string
}

// StashInfo represents a git stash entry
type StashInfo struct {
	Index   int
	Message string
	Branch  string
	Hash    string
	Date    time.Time
}

// TagInfo represents a git tag
type TagInfo struct {
	Name    string
	Hash    string
	Message string
	Date    time.Time
	Tagger  string
}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations(repo *Repository, gitCmd GitInterface) *GitOperations {
	if gitCmd == nil {
		gitCmd = NewGitCmd()
	}
	return &GitOperations{
		repo:   repo,
		gitCmd: gitCmd,
	}
}

// Branch Management Operations

// CreateBranch creates a new branch from the specified source
func (ops *GitOperations) CreateBranch(name, source string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	if source == "" {
		source = ops.repo.DefaultBranch
	}

	// Check if branch already exists
	if ops.BranchExists(name) {
		return fmt.Errorf("branch '%s' already exists", name)
	}

	// Create the branch
	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "branch", name, source)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s' from '%s': %w", name, source, err)
	}

	return nil
}

// DeleteBranch deletes the specified branch
func (ops *GitOperations) DeleteBranch(name string, force bool) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check if it's the current branch
	if name == ops.repo.CurrentBranch {
		return fmt.Errorf("cannot delete current branch '%s'", name)
	}

	// Check if branch exists
	if !ops.BranchExists(name) {
		return fmt.Errorf("branch '%s' does not exist", name)
	}

	// Delete the branch
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, name)

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, args...)
	if err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", name, err)
	}

	return nil
}

// BranchExists checks if a branch exists
func (ops *GitOperations) BranchExists(name string) bool {
	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "rev-parse", "--verify", name)
	return err == nil
}

// GetBranchInfo gets detailed information about a branch
func (ops *GitOperations) GetBranchInfo(branch string) (*BranchInfo, error) {
	if branch == "" {
		branch = ops.repo.CurrentBranch
	}

	if !ops.BranchExists(branch) {
		return nil, fmt.Errorf("branch '%s' does not exist", branch)
	}

	info := &BranchInfo{
		Name:    branch,
		Current: branch == ops.repo.CurrentBranch,
	}

	// Get HEAD commit
	head, err := ops.gitCmd.Execute(ops.repo.RootPath, "rev-parse", branch)
	if err == nil {
		info.Head = head
	}

	// Get upstream information
	upstream, err := ops.gitCmd.Execute(ops.repo.RootPath, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err == nil {
		info.Upstream = upstream
		parts := strings.Split(upstream, "/")
		if len(parts) >= 2 {
			info.Remote = parts[0]
		}

		// Get ahead/behind counts
		if ahead, behind, err := ops.getAheadBehindCounts(branch, upstream); err == nil {
			info.Ahead = ahead
			info.Behind = behind
		}
	}

	return info, nil
}

// ListBranches lists all branches in the repository
func (ops *GitOperations) ListBranches(includeRemote bool) ([]BranchInfo, error) {
	args := []string{"branch"}
	if includeRemote {
		args = append(args, "-a")
	}

	output, err := ops.gitCmd.Execute(ops.repo.RootPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []BranchInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse branch line
		isCurrent := strings.HasPrefix(line, "*")
		if isCurrent {
			line = strings.TrimSpace(line[1:])
		}

		// Skip special entries
		if strings.Contains(line, "->") || strings.HasPrefix(line, "(") {
			continue
		}

		branchName := line
		if strings.HasPrefix(line, "remotes/") {
			branchName = strings.TrimPrefix(line, "remotes/")
		}

		branch := BranchInfo{
			Name:    branchName,
			Current: isCurrent,
		}

		// Get additional info for each branch
		if info, err := ops.GetBranchInfo(branchName); err == nil {
			branch.Head = info.Head
			branch.Upstream = info.Upstream
			branch.Remote = info.Remote
			branch.Ahead = info.Ahead
			branch.Behind = info.Behind
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// Merge Operations

// MergeBranch merges the source branch into the target branch
func (ops *GitOperations) MergeBranch(source, target string) (*MergeResult, error) {
	if source == "" || target == "" {
		return nil, fmt.Errorf("source and target branches must be specified")
	}

	// Check if branches exist
	if !ops.BranchExists(source) {
		return nil, fmt.Errorf("source branch '%s' does not exist", source)
	}
	if !ops.BranchExists(target) {
		return nil, fmt.Errorf("target branch '%s' does not exist", target)
	}

	// Ensure we're on the target branch
	if ops.repo.CurrentBranch != target {
		if err := ops.CheckoutBranch(target); err != nil {
			return nil, fmt.Errorf("failed to checkout target branch '%s': %w", target, err)
		}
	}

	// Perform the merge
	output, err := ops.gitCmd.Execute(ops.repo.RootPath, "merge", source)

	result := &MergeResult{
		Success: err == nil,
	}

	// Parse merge output
	if err != nil {
		// Check if it's a merge conflict
		if strings.Contains(output, "CONFLICT") {
			result.Conflicts = ops.parseConflicts(output)
		}
		return result, fmt.Errorf("merge failed: %w", err)
	}

	// Parse successful merge output
	result.FilesChanged = ops.parseFilesChanged(output)

	// Get the new commit hash
	if hash, err := ops.gitCmd.Execute(ops.repo.RootPath, "rev-parse", "HEAD"); err == nil {
		result.CommitHash = hash
	}

	// Extract merge message
	if msg, err := ops.gitCmd.Execute(ops.repo.RootPath, "log", "-1", "--pretty=format:%s"); err == nil {
		result.Message = msg
	}

	return result, nil
}

// CheckoutBranch switches to the specified branch
func (ops *GitOperations) CheckoutBranch(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	if !ops.BranchExists(branch) {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "checkout", branch)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", branch, err)
	}

	// Update current branch in repo
	ops.repo.CurrentBranch = branch

	return nil
}

// Push Operations

// PushBranch pushes the specified branch to remote
func (ops *GitOperations) PushBranch(branch, remote string, force bool) error {
	if branch == "" {
		branch = ops.repo.CurrentBranch
	}
	if remote == "" {
		remote = "origin"
	}

	if !ops.BranchExists(branch) {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, remote, branch)

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, args...)
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' to '%s': %w", branch, remote, err)
	}

	return nil
}

// PushBranchWithUpstream pushes branch and sets upstream
func (ops *GitOperations) PushBranchWithUpstream(branch, remote string) error {
	if branch == "" {
		branch = ops.repo.CurrentBranch
	}
	if remote == "" {
		remote = "origin"
	}

	if !ops.BranchExists(branch) {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "push", "-u", remote, branch)
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' with upstream to '%s': %w", branch, remote, err)
	}

	return nil
}

// Pull Operations

// PullBranch pulls changes from remote for the specified branch
func (ops *GitOperations) PullBranch(branch, remote string) error {
	if branch == "" {
		branch = ops.repo.CurrentBranch
	}
	if remote == "" {
		remote = "origin"
	}

	// Ensure we're on the correct branch
	if ops.repo.CurrentBranch != branch {
		if err := ops.CheckoutBranch(branch); err != nil {
			return fmt.Errorf("failed to checkout branch '%s': %w", branch, err)
		}
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "pull", remote, branch)
	if err != nil {
		return fmt.Errorf("failed to pull branch '%s' from '%s': %w", branch, remote, err)
	}

	return nil
}

// FetchAll fetches all remotes
func (ops *GitOperations) FetchAll() error {
	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "fetch", "--all")
	if err != nil {
		return fmt.Errorf("failed to fetch all remotes: %w", err)
	}

	return nil
}

// Stash Operations

// StashChanges creates a stash with the given message
func (ops *GitOperations) StashChanges(message string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, args...)
	if err != nil {
		return fmt.Errorf("failed to stash changes: %w", err)
	}

	return nil
}

// PopStash applies and removes the most recent stash
func (ops *GitOperations) PopStash() error {
	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "stash", "pop")
	if err != nil {
		return fmt.Errorf("failed to pop stash: %w", err)
	}

	return nil
}

// ApplyStash applies the specified stash without removing it
func (ops *GitOperations) ApplyStash(stashRef string) error {
	if stashRef == "" {
		stashRef = "stash@{0}"
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "stash", "apply", stashRef)
	if err != nil {
		return fmt.Errorf("failed to apply stash '%s': %w", stashRef, err)
	}

	return nil
}

// ListStashes lists all stashes
func (ops *GitOperations) ListStashes() ([]StashInfo, error) {
	output, err := ops.gitCmd.Execute(ops.repo.RootPath, "stash", "list", "--pretty=format:%gd|%gs|%gD|%at")
	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []StashInfo{}, nil
	}

	var stashes []StashInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			index := ops.parseStashIndex(parts[0])
			message := parts[1]
			hash := parts[2]

			timestamp, _ := strconv.ParseInt(parts[3], 10, 64)
			date := time.Unix(timestamp, 0)

			stash := StashInfo{
				Index:   index,
				Message: message,
				Hash:    hash,
				Date:    date,
			}

			stashes = append(stashes, stash)
		}
	}

	return stashes, nil
}

// DropStash removes the specified stash
func (ops *GitOperations) DropStash(stashRef string) error {
	if stashRef == "" {
		stashRef = "stash@{0}"
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "stash", "drop", stashRef)
	if err != nil {
		return fmt.Errorf("failed to drop stash '%s': %w", stashRef, err)
	}

	return nil
}

// Commit Operations

// CreateCommit creates a commit with the given message
func (ops *GitOperations) CreateCommit(message string, files []string) error {
	if message == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	// Add files if specified
	if len(files) > 0 {
		args := append([]string{"add"}, files...)
		if _, err := ops.gitCmd.Execute(ops.repo.RootPath, args...); err != nil {
			return fmt.Errorf("failed to add files: %w", err)
		}
	}

	// Create commit
	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

// GetCommitHistory gets commit history for the specified branch
func (ops *GitOperations) GetCommitHistory(branch string, limit int) ([]CommitInfo, error) {
	if branch == "" {
		branch = ops.repo.CurrentBranch
	}
	if limit <= 0 {
		limit = 10
	}

	args := []string{"log", "--pretty=format:%H|%an|%at|%s", "-n", strconv.Itoa(limit)}
	if branch != "" {
		args = append(args, branch)
	}

	output, err := ops.gitCmd.Execute(ops.repo.RootPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []CommitInfo{}, nil
	}

	var commits []CommitInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
			date := time.Unix(timestamp, 0)

			commit := CommitInfo{
				Hash:    parts[0],
				Author:  parts[1],
				Date:    date,
				Message: parts[3],
			}

			// Get files for this commit
			if files, err := ops.getCommitFiles(commit.Hash); err == nil {
				commit.Files = files
			}

			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// Tag Operations

// CreateTag creates a new tag
func (ops *GitOperations) CreateTag(name, message string, commit string) error {
	if name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	args := []string{"tag"}
	if message != "" {
		args = append(args, "-a", name, "-m", message)
	} else {
		args = append(args, name)
	}

	if commit != "" {
		args = append(args, commit)
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, args...)
	if err != nil {
		return fmt.Errorf("failed to create tag '%s': %w", name, err)
	}

	return nil
}

// DeleteTag deletes the specified tag
func (ops *GitOperations) DeleteTag(name string) error {
	if name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	_, err := ops.gitCmd.Execute(ops.repo.RootPath, "tag", "-d", name)
	if err != nil {
		return fmt.Errorf("failed to delete tag '%s': %w", name, err)
	}

	return nil
}

// ListTags lists all tags
func (ops *GitOperations) ListTags() ([]TagInfo, error) {
	output, err := ops.gitCmd.Execute(ops.repo.RootPath, "tag", "-l", "--format=%(refname:short)|%(objectname)|%(contents)|%(taggerdate:unix)|%(taggername)")
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []TagInfo{}, nil
	}

	var tags []TagInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			name := parts[0]
			hash := parts[1]
			message := parts[2]

			tag := TagInfo{
				Name:    name,
				Hash:    hash,
				Message: message,
			}

			if len(parts) >= 4 {
				timestamp, _ := strconv.ParseInt(parts[3], 10, 64)
				tag.Date = time.Unix(timestamp, 0)
			}

			if len(parts) >= 5 {
				tag.Tagger = parts[4]
			}

			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// Status Operations

// GetStatus gets the current repository status
func (ops *GitOperations) GetStatus() (map[string]string, error) {
	output, err := ops.gitCmd.Execute(ops.repo.RootPath, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	status := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) >= 3 {
			statusCode := line[:2]
			filename := strings.TrimSpace(line[3:])
			status[filename] = statusCode
		}
	}

	return status, nil
}

// IsClean checks if the working directory is clean
func (ops *GitOperations) IsClean() (bool, error) {
	status, err := ops.GetStatus()
	if err != nil {
		return false, err
	}

	return len(status) == 0, nil
}

// Helper functions

// getAheadBehindCounts gets the ahead/behind counts between two branches
func (ops *GitOperations) getAheadBehindCounts(local, remote string) (ahead, behind int, err error) {
	output, err := ops.gitCmd.Execute(ops.repo.RootPath, "rev-list", "--left-right", "--count", local+"..."+remote)
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(output)
	if len(parts) == 2 {
		ahead, _ = strconv.Atoi(parts[0])
		behind, _ = strconv.Atoi(parts[1])
	}

	return ahead, behind, nil
}

// parseConflicts parses merge conflicts from output
func (ops *GitOperations) parseConflicts(output string) []string {
	var conflicts []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "CONFLICT") {
			// Extract filename from conflict line
			re := regexp.MustCompile(`CONFLICT.*?in (.+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				conflicts = append(conflicts, matches[1])
			}
		}
	}

	return conflicts
}

// parseFilesChanged parses the number of files changed from merge output
func (ops *GitOperations) parseFilesChanged(output string) int {
	re := regexp.MustCompile(`(\d+) files? changed`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count
		}
	}
	return 0
}

// parseStashIndex parses stash index from stash reference
func (ops *GitOperations) parseStashIndex(stashRef string) int {
	re := regexp.MustCompile(`stash@\{(\d+)\}`)
	matches := re.FindStringSubmatch(stashRef)
	if len(matches) > 1 {
		if index, err := strconv.Atoi(matches[1]); err == nil {
			return index
		}
	}
	return 0
}

// getCommitFiles gets the files changed in a specific commit
func (ops *GitOperations) getCommitFiles(commitHash string) ([]string, error) {
	output, err := ops.gitCmd.Execute(ops.repo.RootPath, "show", "--name-only", "--pretty=format:", commitHash)
	if err != nil {
		return nil, err
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}
