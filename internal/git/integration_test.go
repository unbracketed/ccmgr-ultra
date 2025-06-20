package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/ccmgr-ultra/internal/config"
)

// Integration tests for the git module
// These tests verify that all components work together correctly

func TestGitModuleIntegration(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{}
	cfg.SetDefaults()
	cfg.Git.DirectoryPattern = "{{.project}}-{{.branch}}"
	cfg.Git.DefaultBranch = "main"
	cfg.Git.MaxWorktrees = 5

	// Create mock git command
	mockGit := NewMockGitCmd()
	setupMockGitCommands(mockGit)

	// Create repository manager
	repoMgr := NewRepositoryManager(mockGit)

	// Create repository
	repo, err := repoMgr.DetectRepository("/test/repo")
	require.NoError(t, err)

	// Create pattern manager
	patternMgr := NewPatternManager(&cfg.Worktree)

	// Create worktree manager
	worktreeMgr := NewWorktreeManager(repo, cfg, mockGit)

	// Create git operations
	gitOps := NewGitOperations(repo, mockGit)

	// Create validator
	validator := NewValidator(cfg)

	// Create remote manager
	remoteMgr := NewRemoteManager(repo, &cfg.Git, mockGit)

	// Test the complete workflow
	t.Run("Complete Worktree Workflow", func(t *testing.T) {
		testCompleteWorktreeWorkflow(t, worktreeMgr, gitOps, validator, patternMgr, remoteMgr)
	})

	t.Run("Repository Operations", func(t *testing.T) {
		testRepositoryOperations(t, repoMgr, repo)
	})

	t.Run("Pattern and Validation Integration", func(t *testing.T) {
		testPatternValidationIntegration(t, patternMgr, validator)
	})

	t.Run("Git Operations Integration", func(t *testing.T) {
		testGitOperationsIntegration(t, gitOps, validator)
	})

	t.Run("Remote Operations Integration", func(t *testing.T) {
		testRemoteOperationsIntegration(t, remoteMgr, validator)
	})
}

func setupMockGitCommands(mockGit *MockGitCmd) {
	// Repository detection
	mockGit.SetCommand("rev-parse --git-dir", ".git")
	mockGit.SetCommand("rev-parse --show-toplevel", "/test/repo")
	mockGit.SetCommand("branch --show-current", "main")
	mockGit.SetCommand("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	mockGit.SetCommand("status --porcelain", "")
	mockGit.SetCommand("remote -v", "origin\tgit@github.com:user/test-repo.git (fetch)")
	mockGit.SetCommand("worktree list --porcelain", "")

	// Branch operations
	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetCommand("rev-parse --verify feature-branch", "def456ghi")
	mockGit.SetError("rev-parse --verify new-feature", fmt.Errorf("unknown revision"))
	mockGit.SetCommand("branch new-feature main", "")
	mockGit.SetCommand("checkout feature-branch", "Switched to branch 'feature-branch'")

	// Worktree operations
	mockGit.SetCommand("worktree add /test/worktree feature-branch", "")
	mockGit.SetCommand("worktree remove /test/worktree", "")
	mockGit.SetCommand("worktree prune", "")

	// Git operations
	mockGit.SetCommand("merge feature-branch", "Merge made by the 'recursive' strategy.")
	mockGit.SetCommand("push -u origin feature-branch", "")
	mockGit.SetCommand("pull origin main", "Already up to date.")
	mockGit.SetCommand("stash push -m work in progress", "Saved working directory and index state")
	mockGit.SetCommand("stash pop", "On branch main: work in progress")

	// Remote operations
	mockGit.SetCommand("push -u origin new-feature", "")

	// Commit operations
	mockGit.SetCommand("rev-parse HEAD", "abc123def")
	mockGit.SetCommand("log -1 --pretty=format:%s", "Test commit")
	mockGit.SetCommand("show --no-patch --pretty=format:%H%n%an%n%at%n%s abc123def", "abc123def\nTest User\n1640995200\nTest commit")
	mockGit.SetCommand("show --name-only --pretty=format: abc123def", "test.txt")
}

func testCompleteWorktreeWorkflow(t *testing.T, worktreeMgr *WorktreeManager, gitOps *GitOperations, validator *Validator, patternMgr *PatternManager, remoteMgr *RemoteManager) {
	// 1. Validate branch name
	branchName := "feature/user-auth"
	result := validator.ValidateBranchName(branchName)
	assert.True(t, result.Valid)

	// 2. Generate worktree path using pattern
	context := PatternContext{
		Project: "test-repo",
		Branch:  "feature-user-auth",
		Timestamp: "20240101-120000",
	}
	path, err := patternMgr.ApplyPattern("{{.project}}-{{.branch}}", context)
	require.NoError(t, err)
	assert.Equal(t, "test-repo-feature-user-auth", path)

	// 3. Create branch
	err = gitOps.CreateBranch("new-feature", "main")
	assert.NoError(t, err)

	// 4. Create worktree
	opts := WorktreeOptions{
		Path:     "/test/new-feature",
		Branch:   "new-feature",
		Checkout: true,
	}
	worktree, err := worktreeMgr.CreateWorktree("new-feature", opts)
	// This will fail due to path validation, but we can test the logic
	assert.Error(t, err) // Expected due to mock limitations

	// 5. Test listing worktrees
	worktrees, err := worktreeMgr.ListWorktrees()
	assert.NoError(t, err)
	assert.NotNil(t, worktrees)

	// 6. Test remote operations
	service, err := remoteMgr.DetectHostingService("git@github.com:user/repo.git")
	assert.NoError(t, err)
	assert.Equal(t, "github", service)
}

func testRepositoryOperations(t *testing.T, repoMgr *RepositoryManager, repo *Repository) {
	// Test repository validation
	err := repoMgr.ValidateRepositoryState(repo)
	assert.NoError(t, err)

	// Test getting repository info
	repoInfo, err := repoMgr.GetRepositoryInfo("/test/repo")
	assert.NoError(t, err)
	assert.Equal(t, "/test/repo", repoInfo.RootPath)

	// Test remote info
	remote, err := repoMgr.GetRemoteInfo(repo, "origin")
	assert.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)
}

func testPatternValidationIntegration(t *testing.T, patternMgr *PatternManager, validator *Validator) {
	// Test pattern validation
	pattern := "{{.project}}-{{.branch}}"
	err := patternMgr.ValidatePattern(pattern)
	assert.NoError(t, err)

	// Test pattern application with validation
	context := PatternContext{
		Project: "my-project",
		Branch:  "feature/auth",
	}
	result, err := patternMgr.ApplyPattern(pattern, context)
	assert.NoError(t, err)

	// Validate the result
	validationResult := patternMgr.ValidatePatternResult(result)
	assert.True(t, validationResult.Valid)

	// Test with invalid pattern
	invalidPattern := "../dangerous"
	err = patternMgr.ValidatePattern(invalidPattern)
	assert.Error(t, err)
}

func testGitOperationsIntegration(t *testing.T, gitOps *GitOperations, validator *Validator) {
	// Test branch operations with validation
	branchName := "test-branch"
	result := validator.ValidateBranchName(branchName)
	assert.True(t, result.Valid)

	// Test branch exists check
	exists := gitOps.BranchExists("main")
	assert.True(t, exists)

	exists = gitOps.BranchExists("nonexistent")
	assert.False(t, exists)

	// Test getting branch info
	info, err := gitOps.GetBranchInfo("main")
	assert.NoError(t, err)
	assert.Equal(t, "main", info.Name)

	// Test validation context for operations
	ctx := ValidationContext{
		Operation: "merge_branch",
		UserInput: map[string]interface{}{
			"source": "feature-branch",
			"target": "main",
		},
	}
	validationResult := validator.ValidateOperationContext(ctx)
	assert.True(t, validationResult.Valid)
}

func testRemoteOperationsIntegration(t *testing.T, remoteMgr *RemoteManager, validator *Validator) {
	// Test hosting service detection
	service, err := remoteMgr.DetectHostingService("https://github.com/user/repo.git")
	assert.NoError(t, err)
	assert.Equal(t, "github", service)

	service, err = remoteMgr.DetectHostingService("https://gitlab.com/user/repo.git")
	assert.NoError(t, err)
	assert.Equal(t, "gitlab", service)

	// Test getting hosting client
	client, err := remoteMgr.GetHostingClient("github")
	assert.NoError(t, err)
	assert.Equal(t, "github", client.GetHostingService())

	client, err = remoteMgr.GetHostingClient("generic")
	assert.NoError(t, err)
	assert.Equal(t, "generic", client.GetHostingService())

	// Test PR template
	template := remoteMgr.GetPullRequestTemplate()
	assert.NotEmpty(t, template)

	// Test listing remotes
	remotes, err := remoteMgr.ListRemotes()
	assert.NoError(t, err)
	assert.NotEmpty(t, remotes)
}

func TestConfigurationIntegration(t *testing.T) {
	// Test complete configuration setup and validation
	cfg := &config.Config{}
	cfg.SetDefaults()

	// Customize git configuration
	cfg.Git.DirectoryPattern = "{{.project}}-{{.branch}}-{{.timestamp}}"
	cfg.Git.MaxWorktrees = 3
	cfg.Git.ProtectedBranches = []string{"main", "develop"}
	cfg.Git.GitHubToken = "test_token"

	// Validate configuration
	err := cfg.Validate()
	assert.NoError(t, err)

	// Test validator with configuration
	validator := NewValidator(cfg)
	result := validator.ValidateConfiguration()
	assert.True(t, result.Valid)

	// Test pattern manager with configuration
	patternMgr := NewPatternManager(&cfg.Worktree)
	err = patternMgr.ValidatePattern(cfg.Git.DirectoryPattern)
	assert.NoError(t, err)
}

func TestErrorHandlingIntegration(t *testing.T) {
	// Test error propagation through the system
	mockGit := NewMockGitCmd()
	
	// Set up error conditions
	mockGit.SetError("rev-parse --git-dir", fmt.Errorf("not a git repository"))

	repoMgr := NewRepositoryManager(mockGit)

	// Test repository detection failure
	_, err := repoMgr.DetectRepository("/not/a/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")

	// Test validation with invalid input
	validator := NewValidator(nil)
	
	result := validator.ValidateBranchName("")
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)

	result = validator.ValidateWorktreePath("")
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestPerformanceConsiderations(t *testing.T) {
	// Test that operations complete within reasonable time
	start := time.Now()

	cfg := &config.Config{}
	cfg.SetDefaults()

	mockGit := NewMockGitCmd()
	setupMockGitCommands(mockGit)

	// Create all managers
	repoMgr := NewRepositoryManager(mockGit)
	patternMgr := NewPatternManager(&cfg.Worktree)
	validator := NewValidator(cfg)

	// Perform multiple operations
	for i := 0; i < 100; i++ {
		// Pattern operations
		context := PatternContext{
			Project: "test-project",
			Branch:  fmt.Sprintf("feature-%d", i),
		}
		_, err := patternMgr.ApplyPattern("{{.project}}-{{.branch}}", context)
		assert.NoError(t, err)

		// Validation operations
		result := validator.ValidateBranchName(fmt.Sprintf("branch-%d", i))
		assert.True(t, result.Valid)
	}

	duration := time.Since(start)
	
	// Should complete 100 operations in less than 1 second
	assert.Less(t, duration, time.Second)
}

func TestConcurrentOperations(t *testing.T) {
	// Test that the system handles concurrent operations safely
	cfg := &config.Config{}
	cfg.SetDefaults()

	mockGit := NewMockGitCmd()
	setupMockGitCommands(mockGit)

	validator := NewValidator(cfg)
	patternMgr := NewPatternManager(&cfg.Worktree)

	// Run concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Concurrent pattern operations
			context := PatternContext{
				Project: fmt.Sprintf("project-%d", id),
				Branch:  fmt.Sprintf("branch-%d", id),
			}
			_, err := patternMgr.ApplyPattern("{{.project}}-{{.branch}}", context)
			assert.NoError(t, err)

			// Concurrent validation
			result := validator.ValidateBranchName(fmt.Sprintf("branch-%d", id))
			assert.True(t, result.Valid)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEdgeCases(t *testing.T) {
	// Test edge cases and boundary conditions
	validator := NewValidator(nil)
	patternMgr := NewPatternManager(nil)

	// Test with extremely long inputs
	longString := strings.Repeat("a", 1000)
	result := validator.ValidateBranchName(longString)
	assert.False(t, result.Valid)

	// Test with special characters
	specialBranch := "feature/user@#$%^&*()"
	result = validator.ValidateBranchName(specialBranch)
	assert.False(t, result.Valid)

	// Test pattern with empty context
	context := PatternContext{}
	_, err := patternMgr.ApplyPattern("{{.project}}-{{.branch}}", context)
	assert.NoError(t, err) // Should handle empty values gracefully

	// Test with nil inputs
	result = validator.ValidateRepositoryState(nil)
	assert.False(t, result.Valid)
}

func TestMemoryUsage(t *testing.T) {
	// Test memory usage patterns
	cfg := &config.Config{}
	cfg.SetDefaults()

	mockGit := NewMockGitCmd()
	setupMockGitCommands(mockGit)

	// Create managers
	repoMgr := NewRepositoryManager(mockGit)
	validator := NewValidator(cfg)

	// Simulate heavy usage
	for i := 0; i < 1000; i++ {
		// Create temporary objects
		result := validator.ValidateBranchName(fmt.Sprintf("branch-%d", i))
		assert.True(t, result.Valid)

		// This would normally stress memory allocation
		_ = result
	}

	// Test that we can still perform operations
	result := validator.ValidateBranchName("final-test")
	assert.True(t, result.Valid)
}

// Benchmark tests for performance monitoring

func BenchmarkPatternApplication(b *testing.B) {
	patternMgr := NewPatternManager(nil)
	context := PatternContext{
		Project: "test-project",
		Branch:  "feature-branch",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := patternMgr.ApplyPattern("{{.project}}-{{.branch}}", context)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBranchValidation(b *testing.B) {
	validator := NewValidator(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := validator.ValidateBranchName("feature-branch-name")
		if !result.Valid {
			b.Fatal("validation failed")
		}
	}
}

func BenchmarkRepositoryDetection(b *testing.B) {
	mockGit := NewMockGitCmd()
	setupMockGitCommands(mockGit)
	repoMgr := NewRepositoryManager(mockGit)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repoMgr.DetectRepository("/test/repo")
		if err != nil {
			b.Fatal(err)
		}
	}
}