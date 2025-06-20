package git

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/ccmgr-ultra/internal/config"
)

func createTestGitConfig() *config.GitConfig {
	return &config.GitConfig{
		DefaultRemote:  "origin",
		DefaultBranch:  "main",
		GitHubToken:    "github_token_123",
		GitLabToken:    "gitlab_token_456",
		BitbucketToken: "bitbucket_token_789",
		PRTemplate: `## Summary
Test PR description

## Testing
Tested manually`,
	}
}

func TestNewRemoteManager(t *testing.T) {
	repo := createTestRepository()
	gitConfig := createTestGitConfig()
	mockGit := NewMockGitCmd()

	rm := NewRemoteManager(repo, gitConfig, mockGit)

	assert.NotNil(t, rm)
	assert.Equal(t, repo, rm.repo)
	assert.Equal(t, gitConfig, rm.config)
	assert.Equal(t, mockGit, rm.gitCmd)
	assert.NotNil(t, rm.clients)

	// Check that clients are initialized
	assert.Contains(t, rm.clients, "github")
	assert.Contains(t, rm.clients, "gitlab")
	assert.Contains(t, rm.clients, "bitbucket")
	assert.Contains(t, rm.clients, "generic")
}

func TestNewRemoteManager_NilParams(t *testing.T) {
	repo := createTestRepository()

	rm := NewRemoteManager(repo, nil, nil)

	assert.NotNil(t, rm)
	assert.NotNil(t, rm.config)
	assert.IsType(t, &GitCmd{}, rm.gitCmd)
}

func TestDetectHostingService(t *testing.T) {
	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), NewMockGitCmd())

	testCases := []struct {
		name        string
		remoteURL   string
		expected    string
		expectError bool
	}{
		{
			name:      "GitHub SSH URL",
			remoteURL: "git@github.com:user/repo.git",
			expected:  "github",
		},
		{
			name:      "GitHub HTTPS URL",
			remoteURL: "https://github.com/user/repo.git",
			expected:  "github",
		},
		{
			name:      "GitLab SSH URL",
			remoteURL: "git@gitlab.com:user/repo.git",
			expected:  "gitlab",
		},
		{
			name:      "GitLab HTTPS URL",
			remoteURL: "https://gitlab.com/user/repo.git",
			expected:  "gitlab",
		},
		{
			name:      "Bitbucket SSH URL",
			remoteURL: "git@bitbucket.org:user/repo.git",
			expected:  "bitbucket",
		},
		{
			name:      "Bitbucket HTTPS URL",
			remoteURL: "https://bitbucket.org/user/repo.git",
			expected:  "bitbucket",
		},
		{
			name:      "Generic Git URL",
			remoteURL: "https://custom-git.example.com/user/repo.git",
			expected:  "generic",
		},
		{
			name:        "Empty URL",
			remoteURL:   "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := rm.DetectHostingService(tc.remoteURL)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, service)
			}
		})
	}
}

func TestCreatePullRequest_Success(t *testing.T) {
	repo := createTestRepository()
	gitConfig := createTestGitConfig()
	mockGit := NewMockGitCmd()

	// Mock git commands for pushing branch
	mockGit.SetCommand("rev-parse --verify feature-branch", "abc123def")
	mockGit.SetCommand("push -u origin feature-branch", "")

	rm := NewRemoteManager(repo, gitConfig, mockGit)

	worktree := &WorktreeInfo{
		Branch: "feature-branch",
		Path:   "/test/feature",
	}

	req := PullRequestRequest{
		Title:       "Add new feature",
		Description: "This adds a new feature",
	}

	pr, err := rm.CreatePullRequest(worktree, req)

	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, "Add new feature", pr.Title)
	assert.Equal(t, "feature-branch", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
}

func TestCreatePullRequest_NilWorktree(t *testing.T) {
	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), NewMockGitCmd())

	req := PullRequestRequest{
		Title: "Test PR",
	}

	_, err := rm.CreatePullRequest(nil, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree info cannot be nil")
}

func TestPushAndCreatePR(t *testing.T) {
	repo := createTestRepository()
	gitConfig := createTestGitConfig()
	mockGit := NewMockGitCmd()

	// Mock git commands
	mockGit.SetCommand("rev-parse --verify feature-branch", "abc123def")
	mockGit.SetCommand("push -u origin feature-branch", "")

	rm := NewRemoteManager(repo, gitConfig, mockGit)

	worktree := &WorktreeInfo{
		Branch: "feature-branch",
		Path:   "/test/feature",
	}

	req := PullRequestRequest{
		Title: "Test PR",
	}

	pr, err := rm.PushAndCreatePR(worktree, req)

	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, "Test PR", pr.Title)
}

func TestGetHostingClient_Success(t *testing.T) {
	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), NewMockGitCmd())

	testCases := []string{"github", "gitlab", "bitbucket", "generic"}

	for _, service := range testCases {
		t.Run(service, func(t *testing.T) {
			client, err := rm.GetHostingClient(service)
			
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, service, client.GetHostingService())
		})
	}
}

func TestGetHostingClient_Unsupported(t *testing.T) {
	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), NewMockGitCmd())

	_, err := rm.GetHostingClient("unsupported")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported hosting service")
}

func TestValidateAuthentication(t *testing.T) {
	gitConfig := createTestGitConfig()
	rm := NewRemoteManager(createTestRepository(), gitConfig, NewMockGitCmd())

	// Test valid authentication
	err := rm.ValidateAuthentication("github")
	assert.NoError(t, err)

	err = rm.ValidateAuthentication("gitlab")
	assert.NoError(t, err)

	err = rm.ValidateAuthentication("bitbucket")
	assert.NoError(t, err)

	// Test unsupported service
	err = rm.ValidateAuthentication("unsupported")
	assert.Error(t, err)
}

func TestValidateAuthentication_NoToken(t *testing.T) {
	gitConfig := &config.GitConfig{} // No tokens configured
	rm := NewRemoteManager(createTestRepository(), gitConfig, NewMockGitCmd())

	err := rm.ValidateAuthentication("github")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no authentication token configured")
}

func TestEnsureBranchPushed_Success(t *testing.T) {
	mockGit := NewMockGitCmd()
	mockGit.SetCommand("rev-parse --verify feature-branch", "abc123def")
	mockGit.SetCommand("push -u origin feature-branch", "")

	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), mockGit)

	err := rm.ensureBranchPushed("feature-branch")

	assert.NoError(t, err)
}

func TestEnsureBranchPushed_EmptyBranch(t *testing.T) {
	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), NewMockGitCmd())

	err := rm.ensureBranchPushed("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name cannot be empty")
}

func TestEnsureBranchPushed_BranchNotExists(t *testing.T) {
	mockGit := NewMockGitCmd()
	mockGit.SetError("rev-parse --verify nonexistent", fmt.Errorf("unknown revision"))

	rm := NewRemoteManager(createTestRepository(), createTestGitConfig(), mockGit)

	err := rm.ensureBranchPushed("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist locally")
}

func TestGetRemoteInfo_Success(t *testing.T) {
	repo := createTestRepository()
	rm := NewRemoteManager(repo, createTestGitConfig(), NewMockGitCmd())

	remote, err := rm.GetRemoteInfo("origin")

	require.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)
	assert.Equal(t, "git@github.com:user/test-repo.git", remote.URL)
}

func TestGetRemoteInfo_DefaultRemote(t *testing.T) {
	repo := createTestRepository()
	rm := NewRemoteManager(repo, createTestGitConfig(), NewMockGitCmd())

	remote, err := rm.GetRemoteInfo("")

	require.NoError(t, err)
	assert.Equal(t, "origin", remote.Name)
}

func TestRemoteManager_GetRemoteInfo_NotFound(t *testing.T) {
	repo := createTestRepository()
	rm := NewRemoteManager(repo, createTestGitConfig(), NewMockGitCmd())

	_, err := rm.GetRemoteInfo("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote 'nonexistent' not found")
}

func TestListRemotes(t *testing.T) {
	repo := createTestRepository()
	rm := NewRemoteManager(repo, createTestGitConfig(), NewMockGitCmd())

	remotes, err := rm.ListRemotes()

	require.NoError(t, err)
	assert.Len(t, remotes, 1)
	assert.Equal(t, "origin", remotes[0].Remote.Name)
	assert.Equal(t, "github", remotes[0].HostingService)
	assert.True(t, remotes[0].SupportsPR)
	assert.True(t, remotes[0].Authenticated)
}

func TestGetPullRequestTemplate(t *testing.T) {
	gitConfig := createTestGitConfig()
	rm := NewRemoteManager(createTestRepository(), gitConfig, NewMockGitCmd())

	template := rm.GetPullRequestTemplate()

	assert.NotEmpty(t, template)
	assert.Contains(t, template, "## Summary")
}

func TestSetPullRequestTemplate(t *testing.T) {
	gitConfig := createTestGitConfig()
	rm := NewRemoteManager(createTestRepository(), gitConfig, NewMockGitCmd())

	newTemplate := "# New PR Template\nDescription here"
	rm.SetPullRequestTemplate(newTemplate)

	assert.Equal(t, newTemplate, rm.GetPullRequestTemplate())
}

// Test GitHub Client

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient("test_token")

	assert.NotNil(t, client)
	assert.Equal(t, "test_token", client.token)
	assert.Equal(t, "https://api.github.com", client.apiURL)
	assert.Equal(t, "github", client.GetHostingService())
}

func TestGitHubClient_CreatePullRequest(t *testing.T) {
	client := NewGitHubClient("test_token")

	req := PullRequestRequest{
		Title:        "Test PR",
		Description:  "Test description",
		SourceBranch: "feature",
		TargetBranch: "main",
		Owner:        "user",
		Repository:   "repo",
	}

	pr, err := client.CreatePullRequest(req)

	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "feature", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
	assert.Contains(t, pr.URL, "github.com")
}

func TestGitHubClient_AuthenticateToken(t *testing.T) {
	client := NewGitHubClient("test_token")

	err := client.AuthenticateToken("valid_token")
	assert.NoError(t, err)

	err = client.AuthenticateToken("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is empty")
}

func TestGitHubClient_ValidateRepository(t *testing.T) {
	client := NewGitHubClient("test_token")

	err := client.ValidateRepository("user", "repo")
	assert.NoError(t, err)

	err = client.ValidateRepository("", "repo")
	assert.Error(t, err)

	err = client.ValidateRepository("user", "")
	assert.Error(t, err)
}

// Test GitLab Client

func TestNewGitLabClient(t *testing.T) {
	client := NewGitLabClient("test_token")

	assert.NotNil(t, client)
	assert.Equal(t, "test_token", client.token)
	assert.Equal(t, "https://gitlab.com/api/v4", client.apiURL)
	assert.Equal(t, "gitlab", client.GetHostingService())
}

func TestGitLabClient_CreatePullRequest(t *testing.T) {
	client := NewGitLabClient("test_token")

	req := PullRequestRequest{
		Title:        "Test MR",
		Description:  "Test description",
		SourceBranch: "feature",
		TargetBranch: "main",
		Owner:        "user",
		Repository:   "repo",
	}

	pr, err := client.CreatePullRequest(req)

	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, "Test MR", pr.Title)
	assert.Equal(t, "feature", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
	assert.Contains(t, pr.URL, "gitlab.com")
}

// Test Bitbucket Client

func TestNewBitbucketClient(t *testing.T) {
	client := NewBitbucketClient("test_token")

	assert.NotNil(t, client)
	assert.Equal(t, "test_token", client.token)
	assert.Equal(t, "https://api.bitbucket.org/2.0", client.apiURL)
	assert.Equal(t, "bitbucket", client.GetHostingService())
}

func TestBitbucketClient_CreatePullRequest(t *testing.T) {
	client := NewBitbucketClient("test_token")

	req := PullRequestRequest{
		Title:        "Test PR",
		Description:  "Test description",
		SourceBranch: "feature",
		TargetBranch: "main",
		Owner:        "user",
		Repository:   "repo",
	}

	pr, err := client.CreatePullRequest(req)

	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "feature", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
	assert.Contains(t, pr.URL, "bitbucket.org")
}

// Test Generic Client

func TestGenericClient_GetHostingService(t *testing.T) {
	client := &GenericClient{}
	assert.Equal(t, "generic", client.GetHostingService())
}

func TestGenericClient_CreatePullRequest(t *testing.T) {
	client := &GenericClient{}

	req := PullRequestRequest{
		Title: "Test PR",
	}

	_, err := client.CreatePullRequest(req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pull requests not supported")
}

func TestGenericClient_AuthenticateToken(t *testing.T) {
	client := &GenericClient{}

	err := client.AuthenticateToken("token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication not required")
}

func TestGenericClient_ValidateRepository(t *testing.T) {
	client := &GenericClient{}

	err := client.ValidateRepository("user", "repo")

	assert.NoError(t, err)
}

// Test Utility Functions

func TestBuildAuthHeaders(t *testing.T) {
	testCases := []struct {
		service string
		token   string
		expects map[string]string
	}{
		{
			service: "github",
			token:   "github_token",
			expects: map[string]string{
				"Authorization": "Bearer github_token",
				"Accept":        "application/vnd.github.v3+json",
			},
		},
		{
			service: "gitlab",
			token:   "gitlab_token",
			expects: map[string]string{
				"Authorization": "Bearer gitlab_token",
				"Content-Type":  "application/json",
			},
		},
		{
			service: "bitbucket",
			token:   "bitbucket_token",
			expects: map[string]string{
				"Authorization": "Bearer bitbucket_token",
				"Content-Type":  "application/json",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.service, func(t *testing.T) {
			headers := buildAuthHeaders(tc.service, tc.token)
			
			for key, expectedValue := range tc.expects {
				assert.Equal(t, expectedValue, headers[key])
			}
		})
	}
}

func TestRemoteInfo_Fields(t *testing.T) {
	remote := Remote{
		Name: "origin",
		URL:  "https://github.com/user/repo.git",
	}

	remoteInfo := RemoteInfo{
		Remote:         remote,
		HostingService: "github",
		SupportsPR:     true,
		Authenticated:  true,
	}

	assert.Equal(t, "origin", remoteInfo.Remote.Name)
	assert.Equal(t, "github", remoteInfo.HostingService)
	assert.True(t, remoteInfo.SupportsPR)
	assert.True(t, remoteInfo.Authenticated)
}

func TestPullRequest_Fields(t *testing.T) {
	now := time.Now()
	
	pr := PullRequest{
		ID:          123,
		Number:      1,
		Title:       "Test PR",
		URL:         "https://github.com/user/repo/pull/1",
		State:       "open",
		CreatedAt:   now,
		UpdatedAt:   now,
		Author:      "testuser",
		SourceBranch: "feature",
		TargetBranch: "main",
		Draft:       false,
		Labels:      []string{"enhancement", "feature"},
	}

	assert.Equal(t, 123, pr.ID)
	assert.Equal(t, 1, pr.Number)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "open", pr.State)
	assert.Equal(t, "feature", pr.SourceBranch)
	assert.Equal(t, "main", pr.TargetBranch)
	assert.False(t, pr.Draft)
	assert.Len(t, pr.Labels, 2)
}

func TestPullRequestRequest_Fields(t *testing.T) {
	req := PullRequestRequest{
		Title:        "Test PR",
		Description:  "Test description",
		SourceBranch: "feature",
		TargetBranch: "main",
		Owner:        "user",
		Repository:   "repo",
		Draft:        true,
		Labels:       []string{"enhancement"},
		Assignees:    []string{"reviewer1"},
	}

	assert.Equal(t, "Test PR", req.Title)
	assert.Equal(t, "Test description", req.Description)
	assert.Equal(t, "feature", req.SourceBranch)
	assert.Equal(t, "main", req.TargetBranch)
	assert.True(t, req.Draft)
	assert.Len(t, req.Labels, 1)
	assert.Len(t, req.Assignees, 1)
}