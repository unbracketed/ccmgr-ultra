package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGitCmd implements GitInterface for testing
type MockGitCmd struct {
	mock.Mock
	commands map[string]string
	errors   map[string]error
}

func NewMockGitCmd() *MockGitCmd {
	return &MockGitCmd{
		commands: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *MockGitCmd) Execute(dir string, args ...string) (string, error) {
	key := strings.Join(args, " ")
	
	if err, exists := m.errors[key]; exists {
		return "", err
	}
	
	if output, exists := m.commands[key]; exists {
		return output, nil
	}
	
	return "", fmt.Errorf("mock command not found: %s", key)
}

func (m *MockGitCmd) ExecuteWithInput(dir, input string, args ...string) (string, error) {
	return m.Execute(dir, args...)
}

func (m *MockGitCmd) SetCommand(args string, output string) {
	m.commands[args] = output
}

func (m *MockGitCmd) SetError(args string, err error) {
	m.errors[args] = err
}

func TestNewRepositoryManager(t *testing.T) {
	// Test with nil git command
	rm := NewRepositoryManager(nil)
	assert.NotNil(t, rm)
	assert.IsType(t, &GitCmd{}, rm.gitCmd)

	// Test with mock git command
	mockGit := NewMockGitCmd()
	rm = NewRepositoryManager(mockGit)
	assert.NotNil(t, rm)
	assert.Equal(t, mockGit, rm.gitCmd)
}

func TestIsGitRepository(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Test valid repository
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	assert.True(t, rm.IsGitRepository("/path/to/repo"))

	// Test invalid repository
	mockGit.SetError("rev-parse --git-dir", fmt.Errorf("not a git repository"))
	assert.False(t, rm.IsGitRepository("/path/to/non-repo"))
}

func TestDetectRepository(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Setup mock responses for a valid repository
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("rev-parse --show-toplevel", "/home/user/repo")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/repo.git (fetch)")
	mockGit.SetCommand("worktree list --porcelain", "worktree /home/user/repo\nHEAD abc123\nbranch refs/heads/main")

	repo, err := rm.DetectRepository("/home/user/repo")
	
	require.NoError(t, err)
	assert.Equal(t, "/home/user/repo", repo.Path)
	assert.Equal(t, "/home/user/repo", repo.RootPath)
	assert.Equal(t, "main", repo.CurrentBranch)
	assert.Equal(t, "main", repo.DefaultBranch)
	assert.True(t, repo.IsClean)
	assert.Len(t, repo.Remotes, 1)
	assert.Equal(t, "origin", repo.Remotes[0].Name)
}

func TestDetectRepository_NotGitRepo(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Setup mock to return error for git directory check
	mockGit.SetError("rev-parse --git-dir", fmt.Errorf("not a git repository"))

	_, err := rm.DetectRepository("/path/to/non-repo")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestDetectRepository_EmptyPath(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Get current working directory
	cwd, _ := os.Getwd()

	// Setup mock responses for current directory
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("rev-parse --show-toplevel", cwd)
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "")
	mockGit.SetCommand("worktree list --porcelain", "")

	repo, err := rm.DetectRepository("")
	
	require.NoError(t, err)
	assert.Equal(t, cwd, repo.Path)
}

func TestParseRemoteURL(t *testing.T) {
	rm := NewRepositoryManager(nil)

	testCases := []struct {
		name     string
		url      string
		expected Remote
		hasError bool
	}{
		{
			name: "SSH URL",
			url:  "git@github.com:user/repo.git",
			expected: Remote{
				URL:      "git@github.com:user/repo.git",
				Protocol: "ssh",
				Host:     "github.com",
				Owner:    "user",
				Repo:     "repo",
			},
		},
		{
			name: "HTTPS URL",
			url:  "https://github.com/user/repo.git",
			expected: Remote{
				URL:      "https://github.com/user/repo.git",
				Protocol: "https",
				Host:     "github.com",
				Owner:    "user",
				Repo:     "repo",
			},
		},
		{
			name: "HTTPS URL without .git",
			url:  "https://gitlab.com/user/repo",
			expected: Remote{
				URL:      "https://gitlab.com/user/repo",
				Protocol: "https",
				Host:     "gitlab.com",
				Owner:    "user",
				Repo:     "repo",
			},
		},
		{
			name: "HTTP URL",
			url:  "http://example.com/user/repo.git",
			expected: Remote{
				URL:      "http://example.com/user/repo.git",
				Protocol: "http",
				Host:     "example.com",
				Owner:    "user",
				Repo:     "repo",
			},
		},
		{
			name:     "Invalid URL",
			url:      "invalid-url",
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			remote := &Remote{URL: tc.url}
			err := rm.parseRemoteURL(remote)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected.Protocol, remote.Protocol)
				assert.Equal(t, tc.expected.Host, remote.Host)
				assert.Equal(t, tc.expected.Owner, remote.Owner)
				assert.Equal(t, tc.expected.Repo, remote.Repo)
			}
		})
	}
}

func TestGetRemotes(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Test with multiple remotes
	remoteOutput := `origin	git@github.com:user/repo.git (fetch)
origin	git@github.com:user/repo.git (push)
upstream	https://github.com/upstream/repo.git (fetch)
upstream	https://github.com/upstream/repo.git (push)`

	mockGit.SetCommand("remote -v", remoteOutput)

	repo := &Repository{RootPath: "/test/repo"}
	remotes, err := rm.getRemotes(repo)

	require.NoError(t, err)
	assert.Len(t, remotes, 2)

	// Find origin remote
	var origin *Remote
	for _, remote := range remotes {
		if remote.Name == "origin" {
			origin = &remote
			break
		}
	}
	require.NotNil(t, origin)
	assert.Equal(t, "ssh", origin.Protocol)
	assert.Equal(t, "github.com", origin.Host)
	assert.Equal(t, "user", origin.Owner)
	assert.Equal(t, "repo", origin.Repo)
}

func TestGetRemotes_Empty(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	mockGit.SetCommand("remote -v", "")

	repo := &Repository{RootPath: "/test/repo"}
	remotes, err := rm.getRemotes(repo)

	require.NoError(t, err)
	assert.Len(t, remotes, 0)
}

func TestGetWorktrees(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Test with multiple worktrees
	worktreeOutput := `worktree /home/user/repo
HEAD abc123def
branch refs/heads/main

worktree /home/user/repo-feature
HEAD def456ghi
branch refs/heads/feature-branch`

	mockGit.SetCommand("worktree list --porcelain", worktreeOutput)
	
	// Mock status and commit info for worktrees
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s abc123def", "abc123def\nJohn Doe\n1640995200\nInitial commit")
	mockGit.SetCommand("show --name-only --pretty=format: abc123def", "file1.txt\nfile2.txt")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nJane Doe\n1640995300\nFeature commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")

	repo := &Repository{RootPath: "/home/user/repo"}
	worktrees, err := rm.getWorktrees(repo)

	require.NoError(t, err)
	assert.Len(t, worktrees, 2)

	// Check first worktree
	assert.Equal(t, "/home/user/repo", worktrees[0].Path)
	assert.Equal(t, "main", worktrees[0].Branch)
	assert.Equal(t, "abc123def", worktrees[0].Head)
	assert.True(t, worktrees[0].IsClean)

	// Check second worktree
	assert.Equal(t, "/home/user/repo-feature", worktrees[1].Path)
	assert.Equal(t, "feature-branch", worktrees[1].Branch)
	assert.Equal(t, "def456ghi", worktrees[1].Head)
}

func TestGetCommitInfo(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	commitHash := "abc123def"
	commitOutput := "abc123def\nJohn Doe\n1640995200\nInitial commit"
	filesOutput := "file1.txt\nfile2.txt"

	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s "+commitHash, commitOutput)
	mockGit.SetCommand("show --name-only --pretty=format: "+commitHash, filesOutput)

	commit, err := rm.getCommitInfo("/test/repo", commitHash)

	require.NoError(t, err)
	assert.Equal(t, "abc123def", commit.Hash)
	assert.Equal(t, "John Doe", commit.Author)
	assert.Equal(t, "Initial commit", commit.Message)
	assert.Len(t, commit.Files, 2)
	assert.Contains(t, commit.Files, "file1.txt")
	assert.Contains(t, commit.Files, "file2.txt")
}

func TestGetCommitInfo_EmptyHash(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	_, err := rm.getCommitInfo("/test/repo", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit hash is empty")
}

func TestRepositoryManager_ValidateRepositoryState(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	// Test with nil repository
	err := rm.ValidateRepositoryState(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository is nil")

	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "test-repo")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	repo := &Repository{RootPath: tempDir}

	// Test with non-git directory
	mockGit.SetError("rev-parse --git-dir", fmt.Errorf("not a git repository"))
	err = rm.ValidateRepositoryState(repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no longer a git repository")

	// Test with valid repository
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "")
	mockGit.SetCommand("worktree list --porcelain", "")

	err = rm.ValidateRepositoryState(repo)
	assert.NoError(t, err)
}

func TestGetRemoteInfo(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	repo := &Repository{RootPath: "/test/repo"}

	// Test getting origin remote
	mockGit.SetCommand("remote get-url origin", "git@github.com:user/repo.git")

	remote, err := rm.GetRemoteInfo(repo, "origin")

	require.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)
	assert.Equal(t, "git@github.com:user/repo.git", remote.URL)
	assert.Equal(t, "ssh", remote.Protocol)
	assert.Equal(t, "github.com", remote.Host)
	assert.Equal(t, "user", remote.Owner)
	assert.Equal(t, "repo", remote.Repo)
}

func TestGetRemoteInfo_Default(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	repo := &Repository{RootPath: "/test/repo"}

	// Test with empty remote name (should default to origin)
	mockGit.SetCommand("remote get-url origin", "https://github.com/user/repo.git")

	remote, err := rm.GetRemoteInfo(repo, "")

	require.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)
	assert.Equal(t, "https", remote.Protocol)
}

func TestRepositoryManager_GetRemoteInfo_NotFound(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	repo := &Repository{RootPath: "/test/repo"}

	// Test with non-existent remote
	mockGit.SetError("remote get-url nonexistent", fmt.Errorf("No such remote 'nonexistent'"))

	_, err := rm.GetRemoteInfo(repo, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get remote URL")
}

func TestRepositoryManager_BranchExists(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	repo := &Repository{RootPath: "/test/repo"}

	// Test existing branch
	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	assert.True(t, rm.branchExists(repo, "main"))

	// Test non-existing branch
	mockGit.SetError("rev-parse --verify nonexistent", fmt.Errorf("unknown revision"))
	assert.False(t, rm.branchExists(repo, "nonexistent"))
}

func TestGetDefaultBranch(t *testing.T) {
	mockGit := NewMockGitCmd()
	rm := NewRepositoryManager(mockGit)

	repo := &Repository{RootPath: "/test/repo"}

	// Test getting default branch from remote HEAD
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	
	branch, err := rm.getDefaultBranch(repo)
	require.NoError(t, err)
	assert.Equal(t, "main", branch)

	// Test fallback to git config
	mockGit.SetError("symbolic-ref refs/remotes/origin/HEAD", fmt.Errorf("not found"))
	mockGit.SetCommand("config --get init.defaultBranch", "develop")
	
	branch, err = rm.getDefaultBranch(repo)
	require.NoError(t, err)
	assert.Equal(t, "develop", branch)

	// Test complete failure
	mockGit.SetError("config --get init.defaultBranch", fmt.Errorf("not found"))
	
	_, err = rm.getDefaultBranch(repo)
	assert.Error(t, err)
}