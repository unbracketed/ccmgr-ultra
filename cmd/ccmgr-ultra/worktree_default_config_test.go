package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unbracketed/ccmgr-ultra/internal/config"
	"github.com/unbracketed/ccmgr-ultra/internal/git"
)

func TestWorktreeCreateWithDefaultConfig(t *testing.T) {
	// Setup test repository
	testDir := setupTestRepo(t)
	defer os.RemoveAll(testDir)

	// Create config with defaults
	cfg := &config.Config{}
	cfg.SetDefaults()

	// Test worktree creation
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(testDir)
	require.NoError(t, err)

	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)

	// This should not fail with template variable error
	worktreeInfo, err := worktreeManager.CreateWorktree("test-branch", git.WorktreeOptions{
		CreateBranch: true,
		AutoName:     true,
	})

	assert.NoError(t, err)
	assert.NotNil(t, worktreeInfo)
	assert.Contains(t, worktreeInfo.Path, "test-branch")
}

func setupTestRepo(t *testing.T) string {
	testDir, err := os.MkdirTemp("", "ccmgr-test-*")
	require.NoError(t, err)

	// Initialize git repo
	gitCmd := git.NewGitCmd()
	_, err = gitCmd.Execute(testDir, "init")
	require.NoError(t, err)

	// Create initial commit
	_, err = gitCmd.Execute(testDir, "config", "user.email", "test@example.com")
	require.NoError(t, err)
	_, err = gitCmd.Execute(testDir, "config", "user.name", "Test User")
	require.NoError(t, err)

	readmeFile := filepath.Join(testDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repo"), 0644)
	require.NoError(t, err)

	_, err = gitCmd.Execute(testDir, "add", ".")
	require.NoError(t, err)
	_, err = gitCmd.Execute(testDir, "commit", "-m", "Initial commit")
	require.NoError(t, err)

	return testDir
}
