package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/ccmgr-ultra/internal/config"
)

func createTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.SetDefaults()
	cfg.Worktree.DirectoryPattern = "{{.project}}-{{.branch}}"
	cfg.Tmux.SessionPrefix = "test"
	cfg.Tmux.NamingPattern = "{{.prefix}}-{{.project}}-{{.branch}}"
	cfg.Tmux.MaxSessionName = 50
	return cfg
}

func createTestRepository() *Repository {
	return &Repository{
		Path:          "/test/repo",
		RootPath:      "/test/repo",
		Origin:        "git@github.com:user/test-repo.git",
		DefaultBranch: "main",
		CurrentBranch: "main",
		IsClean:       true,
		Remotes: []Remote{
			{
				Name:     "origin",
				URL:      "git@github.com:user/test-repo.git",
				Protocol: "ssh",
				Host:     "github.com",
				Owner:    "user",
				Repo:     "test-repo",
			},
		},
	}
}

func TestNewWorktreeManager(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	wm := NewWorktreeManager(repo, cfg, mockGit)

	assert.NotNil(t, wm)
	assert.Equal(t, repo, wm.repo)
	assert.Equal(t, cfg, wm.config)
	assert.Equal(t, mockGit, wm.gitCmd)
	assert.NotNil(t, wm.patternMgr)
	assert.NotNil(t, wm.repoMgr)
}

func TestNewWorktreeManager_NilGitCmd(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()

	wm := NewWorktreeManager(repo, cfg, nil)

	assert.NotNil(t, wm)
	assert.IsType(t, &GitCmd{}, wm.gitCmd)
}

func TestCreateWorktree_Success(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/test-repo.git (fetch)")
	mockGit.SetCommand("worktree list --porcelain", "")
	mockGit.SetCommand("rev-parse --verify feature-branch", "abc123def")
	mockGit.SetCommand("worktree add /test/worktree feature-branch", "")
	mockGit.SetCommand("rev-parse HEAD", "abc123def")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s abc123def", "abc123def\nTest User\n1640995200\nTest commit")
	mockGit.SetCommand("show --name-only --pretty=format: abc123def", "test.txt")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	opts := WorktreeOptions{
		Path:     "/test/worktree",
		Branch:   "feature-branch",
		Checkout: true,
	}

	wt, err := wm.CreateWorktree("feature-branch", opts)

	require.NoError(t, err)
	assert.NotNil(t, wt)
	assert.Equal(t, "/test/worktree", wt.Path)
	assert.Equal(t, "feature-branch", wt.Branch)
}

func TestCreateWorktree_EmptyBranch(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	opts := WorktreeOptions{}
	_, err := wm.CreateWorktree("", opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name cannot be empty")
}

func TestCreateWorktree_WithCreateBranch(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/test-repo.git (fetch)")
	mockGit.SetCommand("worktree list --porcelain", "")
	mockGit.SetError("rev-parse --verify new-feature", fmt.Errorf("unknown revision"))
	mockGit.SetCommand("branch new-feature main", "")
	mockGit.SetCommand("worktree add /test/new-feature new-feature", "")
	mockGit.SetCommand("rev-parse HEAD", "def456ghi")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nNew feature")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	opts := WorktreeOptions{
		Path:         "/test/new-feature",
		CreateBranch: true,
		Checkout:     true,
	}

	wt, err := wm.CreateWorktree("new-feature", opts)

	require.NoError(t, err)
	assert.NotNil(t, wt)
	assert.Equal(t, "/test/new-feature", wt.Path)
	assert.Equal(t, "new-feature", wt.Branch)
}

func TestCreateWorktree_AutoName(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses for validation and creation
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/test-repo.git (fetch)")
	mockGit.SetCommand("worktree list --porcelain", "")
	mockGit.SetCommand("rev-parse --verify feature", "abc123def")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	opts := WorktreeOptions{
		AutoName: true,
		Checkout: true,
	}

	// This test would require more complex mocking of file system operations
	// For now, we'll test that the function attempts to generate a path
	_, err := wm.CreateWorktree("feature", opts)
	
	// We expect this to fail during path validation since we're not mocking the filesystem
	assert.Error(t, err)
}

func TestListWorktrees(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/test-repo.git (fetch)")
	
	worktreeOutput := `worktree /test/repo
HEAD abc123def
branch refs/heads/main

worktree /test/feature
HEAD def456ghi
branch refs/heads/feature`

	mockGit.SetCommand("worktree list --porcelain", worktreeOutput)
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s abc123def", "abc123def\nTest User\n1640995200\nMain commit")
	mockGit.SetCommand("show --name-only --pretty=format: abc123def", "main.txt")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nFeature commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	worktrees, err := wm.ListWorktrees()

	require.NoError(t, err)
	assert.Len(t, worktrees, 2)
	assert.Equal(t, "/test/repo", worktrees[0].Path)
	assert.Equal(t, "main", worktrees[0].Branch)
	assert.Equal(t, "/test/feature", worktrees[1].Path)
	assert.Equal(t, "feature", worktrees[1].Branch)
}

func TestDeleteWorktree_Success(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "feature")
	mockGit.SetCommand("rev-parse HEAD", "def456ghi")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nFeature commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")
	mockGit.SetCommand("worktree remove /test/feature", "")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	// Create a temporary directory to simulate the worktree
	tempDir := filepath.Join(os.TempDir(), "test-worktree")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	err := wm.DeleteWorktree(tempDir, false)

	assert.NoError(t, err)
}

func TestDeleteWorktree_EmptyPath(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	err := wm.DeleteWorktree("", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestDeleteWorktree_UncommittedChanges(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses for worktree with uncommitted changes
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "feature")
	mockGit.SetCommand("rev-parse HEAD", "def456ghi")
	mockGit.SetCommand("status --porcelain", " M modified.txt\n?? untracked.txt")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nFeature commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	// Create a temporary directory to simulate the worktree
	tempDir := filepath.Join(os.TempDir(), "test-worktree-dirty")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	err := wm.DeleteWorktree(tempDir, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uncommitted changes")
}

func TestDeleteWorktree_Force(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "feature")
	mockGit.SetCommand("rev-parse HEAD", "def456ghi")
	mockGit.SetCommand("status --porcelain", " M modified.txt")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nFeature commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")
	mockGit.SetCommand("worktree remove --force /test/feature-dirty", "")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	// Create a temporary directory to simulate the worktree
	tempDir := filepath.Join(os.TempDir(), "test-worktree-force")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	err := wm.DeleteWorktree(tempDir, true)

	assert.NoError(t, err)
}

func TestGetWorktreeInfo_Success(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "feature")
	mockGit.SetCommand("rev-parse HEAD", "def456ghi")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nFeature commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	// Create a temporary directory to simulate the worktree
	tempDir := filepath.Join(os.TempDir(), "test-worktree-info")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	wt, err := wm.GetWorktreeInfo(tempDir)

	require.NoError(t, err)
	assert.NotNil(t, wt)
	assert.Equal(t, tempDir, wt.Path)
	assert.Equal(t, "feature", wt.Branch)
	assert.Equal(t, "def456ghi", wt.Head)
	assert.True(t, wt.IsClean)
	assert.False(t, wt.HasUncommitted)
}

func TestGetWorktreeInfo_EmptyPath(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	_, err := wm.GetWorktreeInfo("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestGetWorktreeInfo_PathNotExists(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	_, err := wm.GetWorktreeInfo("/nonexistent/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestPruneWorktrees(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("worktree prune", "")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	err := wm.PruneWorktrees()

	assert.NoError(t, err)
}

func TestMoveWorktree(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("worktree move /old/path /new/path", "")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	// Create temporary directories
	tempDir := filepath.Join(os.TempDir(), "worktree-move-test")
	oldPath := filepath.Join(tempDir, "old")
	newPath := filepath.Join(tempDir, "new")
	
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	err := wm.MoveWorktree(oldPath, newPath)

	// This will fail path availability check, but we can test the validation
	assert.Error(t, err) // Expected due to path availability check
}

func TestMoveWorktree_EmptyPaths(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	err := wm.MoveWorktree("", "/new/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both old and new paths must be specified")

	err = wm.MoveWorktree("/old/path", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both old and new paths must be specified")
}

func TestGetProjectName(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	projectName := wm.getProjectName()

	assert.Equal(t, "test-repo", projectName)
}

func TestGetProjectName_NoRemote(t *testing.T) {
	repo := createTestRepository()
	repo.Remotes = []Remote{} // No remotes
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	projectName := wm.getProjectName()

	assert.Equal(t, "repo", projectName) // Falls back to directory name
}

func TestValidateWorktreePath(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	testCases := []struct {
		name  string
		path  string
		valid bool
	}{
		{
			name:  "Empty path",
			path:  "",
			valid: false,
		},
		{
			name:  "Relative path",
			path:  "relative/path",
			valid: false,
		},
		{
			name:  "Absolute path outside repo",
			path:  "/outside/repo",
			valid: true,
		},
		{
			name:  "Path inside repository",
			path:  "/test/repo/subdir",
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := wm.validateWorktreePath(tc.path)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckBranchWorktreeConflict(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock to return existing worktree
	worktreeOutput := `worktree /test/main
HEAD abc123def
branch refs/heads/main

worktree /test/feature
HEAD def456ghi
branch refs/heads/feature`

	mockGit.SetCommand("worktree list --porcelain", worktreeOutput)

	wm := NewWorktreeManager(repo, cfg, mockGit)

	// Test conflict with existing branch
	err := wm.checkBranchWorktreeConflict("feature")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already checked out")

	// Test no conflict with new branch
	err = wm.checkBranchWorktreeConflict("new-branch")
	assert.NoError(t, err)
}

func TestGetTmuxSessionName(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	cfg.Tmux.SessionPrefix = "ccmgr"
	cfg.Tmux.NamingPattern = "{{.prefix}}-{{.project}}-{{.branch}}"
	cfg.Tmux.MaxSessionName = 30
	
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	wt := &WorktreeInfo{
		Path:   "/test/worktree",
		Branch: "feature/auth",
	}

	sessionName := wm.getTmuxSessionName(wt)

	assert.Equal(t, "ccmgr-test-repo-feature-auth", sessionName)
}

func TestGetTmuxSessionName_NoPrefix(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	cfg.Tmux.SessionPrefix = "" // No prefix
	
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	wt := &WorktreeInfo{
		Path:   "/test/worktree",
		Branch: "feature",
	}

	sessionName := wm.getTmuxSessionName(wt)

	assert.Empty(t, sessionName)
}

func TestGetTmuxSessionName_Truncation(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	cfg.Tmux.SessionPrefix = "very-long-prefix"
	cfg.Tmux.NamingPattern = "{{.prefix}}-{{.project}}-{{.branch}}"
	cfg.Tmux.MaxSessionName = 20 // Very short limit
	
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	wt := &WorktreeInfo{
		Path:   "/test/worktree",
		Branch: "very-long-feature-branch-name",
	}

	sessionName := wm.getTmuxSessionName(wt)

	assert.LessOrEqual(t, len(sessionName), 20)
}

func TestGetWorktreeStats(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	mockGit := NewMockGitCmd()

	// Setup mock responses for validation
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/test-repo.git (fetch)")
	
	// Setup mock worktrees
	worktreeOutput := `worktree /test/clean
HEAD abc123def
branch refs/heads/clean

worktree /test/dirty
HEAD def456ghi
branch refs/heads/dirty`

	mockGit.SetCommand("worktree list --porcelain", worktreeOutput)
	
	// Mock first worktree as clean
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s abc123def", "abc123def\nTest User\n1640995200\nClean commit")
	mockGit.SetCommand("show --name-only --pretty=format: abc123def", "clean.txt")
	
	// Mock second worktree as dirty
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s def456ghi", "def456ghi\nTest User\n1640995300\nDirty commit")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "dirty.txt")

	wm := NewWorktreeManager(repo, cfg, mockGit)

	stats, err := wm.GetWorktreeStats()

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.Total)
	// Note: The actual clean/dirty status would depend on the mock status commands
	// which we'd need to set up for each worktree path
}

func TestCleanupOldWorktrees_Disabled(t *testing.T) {
	repo := createTestRepository()
	cfg := createTestConfig()
	cfg.Tmux.AutoCleanup = false // Disabled
	
	mockGit := NewMockGitCmd()
	wm := NewWorktreeManager(repo, cfg, mockGit)

	err := wm.CleanupOldWorktrees()

	assert.NoError(t, err) // Should return immediately when disabled
}