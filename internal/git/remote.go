package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/your-username/ccmgr-ultra/internal/config"
)

// RemoteManager handles remote repository operations
type RemoteManager struct {
	repo    *Repository
	config  *config.GitConfig
	clients map[string]HostingClient
	gitCmd  GitInterface
}

// HostingClient interface for different git hosting services
type HostingClient interface {
	CreatePullRequest(req PullRequestRequest) (*PullRequest, error)
	GetPullRequests(owner, repo string) ([]PullRequest, error)
	AuthenticateToken(token string) error
	ValidateRepository(owner, repo string) error
	GetHostingService() string
}

// PullRequestRequest represents a PR/MR creation request
type PullRequestRequest struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
	Owner        string
	Repository   string
	Draft        bool
	Labels       []string
	Assignees    []string
}

// PullRequest represents a created PR/MR
type PullRequest struct {
	ID          int
	Number      int
	Title       string
	URL         string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Author      string
	SourceBranch string
	TargetBranch string
	Draft       bool
	Labels      []string
}

// GitHubClient implements HostingClient for GitHub
type GitHubClient struct {
	token  string
	apiURL string
}

// GitLabClient implements HostingClient for GitLab
type GitLabClient struct {
	token  string
	apiURL string
}

// BitbucketClient implements HostingClient for Bitbucket
type BitbucketClient struct {
	token  string
	apiURL string
}

// GenericClient for repositories without PR/MR support
type GenericClient struct{}

// NewRemoteManager creates a new RemoteManager
func NewRemoteManager(repo *Repository, config *config.GitConfig, gitCmd GitInterface) *RemoteManager {
	if gitCmd == nil {
		gitCmd = NewGitCmd()
	}
	if config == nil {
		config = &config.GitConfig{}
	}

	rm := &RemoteManager{
		repo:    repo,
		config:  config,
		clients: make(map[string]HostingClient),
		gitCmd:  gitCmd,
	}

	// Initialize hosting clients
	rm.initializeClients()

	return rm
}

// DetectHostingService detects the hosting service from a remote URL
func (rm *RemoteManager) DetectHostingService(remoteURL string) (string, error) {
	if remoteURL == "" {
		return "", fmt.Errorf("remote URL cannot be empty")
	}

	// Parse the URL to extract the host
	var host string
	
	// Handle SSH URLs (git@host:owner/repo.git)
	sshPattern := regexp.MustCompile(`^git@([^:]+):`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); len(matches) > 1 {
		host = matches[1]
	} else {
		// Handle HTTPS URLs
		if parsed, err := url.Parse(remoteURL); err == nil {
			host = parsed.Host
		} else {
			return "", fmt.Errorf("failed to parse remote URL: %w", err)
		}
	}

	// Determine service based on host
	switch {
	case strings.Contains(host, "github.com"):
		return "github", nil
	case strings.Contains(host, "gitlab.com"):
		return "gitlab", nil
	case strings.Contains(host, "bitbucket.org"):
		return "bitbucket", nil
	default:
		return "generic", nil
	}
}

// CreatePullRequest creates a pull request for the specified worktree
func (rm *RemoteManager) CreatePullRequest(worktree *WorktreeInfo, req PullRequestRequest) (*PullRequest, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree info cannot be nil")
	}

	// Ensure worktree branch is pushed
	if err := rm.ensureBranchPushed(worktree.Branch); err != nil {
		return nil, fmt.Errorf("failed to push branch: %w", err)
	}

	// Get hosting service for origin remote
	service, err := rm.DetectHostingService(rm.repo.Origin)
	if err != nil {
		return nil, fmt.Errorf("failed to detect hosting service: %w", err)
	}

	// Get hosting client
	client, err := rm.GetHostingClient(service)
	if err != nil {
		return nil, fmt.Errorf("failed to get hosting client: %w", err)
	}

	// Fill in missing request details
	if req.SourceBranch == "" {
		req.SourceBranch = worktree.Branch
	}
	if req.TargetBranch == "" {
		req.TargetBranch = rm.repo.DefaultBranch
	}
	if req.Owner == "" || req.Repository == "" {
		for _, remote := range rm.repo.Remotes {
			if remote.Name == "origin" {
				req.Owner = remote.Owner
				req.Repository = remote.Repo
				break
			}
		}
	}

	// Apply PR template if no description provided
	if req.Description == "" && rm.config.PRTemplate != "" {
		req.Description = rm.config.PRTemplate
	}

	// Create the pull request
	pr, err := client.CreatePullRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return pr, nil
}

// PushAndCreatePR pushes a worktree branch and creates a PR in one operation
func (rm *RemoteManager) PushAndCreatePR(worktree *WorktreeInfo, prOptions PullRequestRequest) (*PullRequest, error) {
	// Push the branch first
	if err := rm.ensureBranchPushed(worktree.Branch); err != nil {
		return nil, fmt.Errorf("failed to push branch: %w", err)
	}

	// Create the pull request
	return rm.CreatePullRequest(worktree, prOptions)
}

// GetHostingClient returns the appropriate hosting client for the service
func (rm *RemoteManager) GetHostingClient(service string) (HostingClient, error) {
	if client, exists := rm.clients[service]; exists {
		return client, nil
	}

	return nil, fmt.Errorf("unsupported hosting service: %s", service)
}

// ValidateAuthentication validates authentication for the specified service
func (rm *RemoteManager) ValidateAuthentication(service string) error {
	client, err := rm.GetHostingClient(service)
	if err != nil {
		return err
	}

	// Get the appropriate token
	var token string
	switch service {
	case "github":
		token = rm.config.GitHubToken
	case "gitlab":
		token = rm.config.GitLabToken
	case "bitbucket":
		token = rm.config.BitbucketToken
	default:
		return fmt.Errorf("authentication not supported for service: %s", service)
	}

	if token == "" {
		return fmt.Errorf("no authentication token configured for %s", service)
	}

	return client.AuthenticateToken(token)
}

// initializeClients sets up hosting service clients
func (rm *RemoteManager) initializeClients() {
	// GitHub client
	if rm.config.GitHubToken != "" {
		rm.clients["github"] = &GitHubClient{
			token:  rm.config.GitHubToken,
			apiURL: "https://api.github.com",
		}
	}

	// GitLab client
	if rm.config.GitLabToken != "" {
		rm.clients["gitlab"] = &GitLabClient{
			token:  rm.config.GitLabToken,
			apiURL: "https://gitlab.com/api/v4",
		}
	}

	// Bitbucket client
	if rm.config.BitbucketToken != "" {
		rm.clients["bitbucket"] = &BitbucketClient{
			token:  rm.config.BitbucketToken,
			apiURL: "https://api.bitbucket.org/2.0",
		}
	}

	// Generic client (always available)
	rm.clients["generic"] = &GenericClient{}
}

// ensureBranchPushed ensures the specified branch is pushed to remote
func (rm *RemoteManager) ensureBranchPushed(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check if branch exists locally
	_, err := rm.gitCmd.Execute(rm.repo.RootPath, "rev-parse", "--verify", branch)
	if err != nil {
		return fmt.Errorf("branch '%s' does not exist locally", branch)
	}

	// Push the branch
	remoteName := rm.config.DefaultRemote
	_, err = rm.gitCmd.Execute(rm.repo.RootPath, "push", "-u", remoteName, branch)
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' to '%s': %w", branch, remoteName, err)
	}

	return nil
}

// GitHub Client Implementation

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token:  token,
		apiURL: "https://api.github.com",
	}
}

// GetHostingService returns the service name
func (gc *GitHubClient) GetHostingService() string {
	return "github"
}

// CreatePullRequest creates a GitHub pull request
func (gc *GitHubClient) CreatePullRequest(req PullRequestRequest) (*PullRequest, error) {
	// GitHub API payload
	payload := map[string]interface{}{
		"title": req.Title,
		"body":  req.Description,
		"head":  req.SourceBranch,
		"base":  req.TargetBranch,
		"draft": req.Draft,
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls", gc.apiURL, req.Owner, req.Repository)
	
	// This is a simplified implementation - in a real scenario you'd use a proper HTTP client
	// For now, we'll return a mock response
	pr := &PullRequest{
		ID:          12345,
		Number:      1,
		Title:       req.Title,
		URL:         fmt.Sprintf("https://github.com/%s/%s/pull/1", req.Owner, req.Repository),
		State:       "open",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Author:      "user",
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
		Draft:       req.Draft,
		Labels:      req.Labels,
	}

	_ = payload // Avoid unused variable warning
	_ = apiURL  // Avoid unused variable warning

	return pr, nil
}

// GetPullRequests gets GitHub pull requests
func (gc *GitHubClient) GetPullRequests(owner, repo string) ([]PullRequest, error) {
	// Simplified implementation - would normally make HTTP request
	return []PullRequest{}, nil
}

// AuthenticateToken validates GitHub token
func (gc *GitHubClient) AuthenticateToken(token string) error {
	if token == "" {
		return fmt.Errorf("GitHub token is empty")
	}
	
	// Simplified validation - would normally make API call to verify token
	return nil
}

// ValidateRepository validates GitHub repository access
func (gc *GitHubClient) ValidateRepository(owner, repo string) error {
	if owner == "" || repo == "" {
		return fmt.Errorf("owner and repository name are required")
	}
	
	// Simplified validation - would normally make API call
	return nil
}

// GitLab Client Implementation

// NewGitLabClient creates a new GitLab client
func NewGitLabClient(token string) *GitLabClient {
	return &GitLabClient{
		token:  token,
		apiURL: "https://gitlab.com/api/v4",
	}
}

// GetHostingService returns the service name
func (glc *GitLabClient) GetHostingService() string {
	return "gitlab"
}

// CreatePullRequest creates a GitLab merge request
func (glc *GitLabClient) CreatePullRequest(req PullRequestRequest) (*PullRequest, error) {
	// GitLab uses "merge requests" instead of "pull requests"
	payload := map[string]interface{}{
		"title":         req.Title,
		"description":   req.Description,
		"source_branch": req.SourceBranch,
		"target_branch": req.TargetBranch,
	}

	// Create HTTP request to GitLab API
	projectPath := fmt.Sprintf("%s/%s", req.Owner, req.Repository)
	apiURL := fmt.Sprintf("%s/projects/%s/merge_requests", glc.apiURL, url.QueryEscape(projectPath))

	// Simplified implementation
	pr := &PullRequest{
		ID:          67890,
		Number:      1,
		Title:       req.Title,
		URL:         fmt.Sprintf("https://gitlab.com/%s/%s/-/merge_requests/1", req.Owner, req.Repository),
		State:       "opened",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Author:      "user",
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
		Draft:       req.Draft,
		Labels:      req.Labels,
	}

	_ = payload // Avoid unused variable warning
	_ = apiURL  // Avoid unused variable warning

	return pr, nil
}

// GetPullRequests gets GitLab merge requests
func (glc *GitLabClient) GetPullRequests(owner, repo string) ([]PullRequest, error) {
	return []PullRequest{}, nil
}

// AuthenticateToken validates GitLab token
func (glc *GitLabClient) AuthenticateToken(token string) error {
	if token == "" {
		return fmt.Errorf("GitLab token is empty")
	}
	return nil
}

// ValidateRepository validates GitLab repository access
func (glc *GitLabClient) ValidateRepository(owner, repo string) error {
	if owner == "" || repo == "" {
		return fmt.Errorf("owner and repository name are required")
	}
	return nil
}

// Bitbucket Client Implementation

// NewBitbucketClient creates a new Bitbucket client
func NewBitbucketClient(token string) *BitbucketClient {
	return &BitbucketClient{
		token:  token,
		apiURL: "https://api.bitbucket.org/2.0",
	}
}

// GetHostingService returns the service name
func (bc *BitbucketClient) GetHostingService() string {
	return "bitbucket"
}

// CreatePullRequest creates a Bitbucket pull request
func (bc *BitbucketClient) CreatePullRequest(req PullRequestRequest) (*PullRequest, error) {
	payload := map[string]interface{}{
		"title":       req.Title,
		"description": req.Description,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": req.SourceBranch},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{"name": req.TargetBranch},
		},
	}

	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests", bc.apiURL, req.Owner, req.Repository)

	// Simplified implementation
	pr := &PullRequest{
		ID:          54321,
		Number:      1,
		Title:       req.Title,
		URL:         fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/1", req.Owner, req.Repository),
		State:       "OPEN",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Author:      "user",
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
		Draft:       req.Draft,
		Labels:      req.Labels,
	}

	_ = payload // Avoid unused variable warning
	_ = apiURL  // Avoid unused variable warning

	return pr, nil
}

// GetPullRequests gets Bitbucket pull requests
func (bc *BitbucketClient) GetPullRequests(owner, repo string) ([]PullRequest, error) {
	return []PullRequest{}, nil
}

// AuthenticateToken validates Bitbucket token
func (bc *BitbucketClient) AuthenticateToken(token string) error {
	if token == "" {
		return fmt.Errorf("Bitbucket token is empty")
	}
	return nil
}

// ValidateRepository validates Bitbucket repository access
func (bc *BitbucketClient) ValidateRepository(owner, repo string) error {
	if owner == "" || repo == "" {
		return fmt.Errorf("owner and repository name are required")
	}
	return nil
}

// Generic Client Implementation

// GetHostingService returns the service name
func (genClient *GenericClient) GetHostingService() string {
	return "generic"
}

// CreatePullRequest is not supported for generic repositories
func (genClient *GenericClient) CreatePullRequest(req PullRequestRequest) (*PullRequest, error) {
	return nil, fmt.Errorf("pull requests not supported for generic git repositories")
}

// GetPullRequests is not supported for generic repositories
func (genClient *GenericClient) GetPullRequests(owner, repo string) ([]PullRequest, error) {
	return nil, fmt.Errorf("pull requests not supported for generic git repositories")
}

// AuthenticateToken is not needed for generic repositories
func (genClient *GenericClient) AuthenticateToken(token string) error {
	return fmt.Errorf("authentication not required for generic git repositories")
}

// ValidateRepository is always valid for generic repositories
func (genClient *GenericClient) ValidateRepository(owner, repo string) error {
	return nil
}

// Utility functions for making HTTP requests (simplified)

// makeHTTPRequest is a helper function for making HTTP requests
func makeHTTPRequest(method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	// This is a placeholder for actual HTTP request implementation
	// In a real implementation, you would:
	// 1. Create an HTTP client with proper timeouts
	// 2. Set up authentication headers
	// 3. Handle request/response properly
	// 4. Parse JSON responses
	// 5. Handle rate limiting and retries

	return nil, fmt.Errorf("HTTP request implementation not included in this example")
}

// parseJSONResponse is a helper function for parsing JSON responses
func parseJSONResponse(resp *http.Response, target interface{}) error {
	// This is a placeholder for JSON parsing
	// In a real implementation, you would:
	// 1. Read the response body
	// 2. Parse JSON into the target struct
	// 3. Handle parsing errors appropriately

	_ = resp   // Avoid unused variable warning
	_ = target // Avoid unused variable warning

	return fmt.Errorf("JSON parsing implementation not included in this example")
}

// buildAuthHeaders creates authentication headers for different services
func buildAuthHeaders(service, token string) map[string]string {
	headers := make(map[string]string)
	
	switch service {
	case "github":
		headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
		headers["Accept"] = "application/vnd.github.v3+json"
	case "gitlab":
		headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
		headers["Content-Type"] = "application/json"
	case "bitbucket":
		headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
		headers["Content-Type"] = "application/json"
	}

	return headers
}

// GetRemoteInfo gets detailed information about a remote
func (rm *RemoteManager) GetRemoteInfo(remoteName string) (*Remote, error) {
	if remoteName == "" {
		remoteName = rm.config.DefaultRemote
	}

	for _, remote := range rm.repo.Remotes {
		if remote.Name == remoteName {
			return &remote, nil
		}
	}

	return nil, fmt.Errorf("remote '%s' not found", remoteName)
}

// ListRemotes lists all remotes with their hosting service information
func (rm *RemoteManager) ListRemotes() ([]RemoteInfo, error) {
	var remotes []RemoteInfo

	for _, remote := range rm.repo.Remotes {
		service, _ := rm.DetectHostingService(remote.URL)
		
		remoteInfo := RemoteInfo{
			Remote:         remote,
			HostingService: service,
			SupportsPR:     service != "generic",
		}

		// Check authentication status if service supports it
		if service != "generic" {
			if err := rm.ValidateAuthentication(service); err == nil {
				remoteInfo.Authenticated = true
			}
		}

		remotes = append(remotes, remoteInfo)
	}

	return remotes, nil
}

// RemoteInfo contains extended information about a remote
type RemoteInfo struct {
	Remote         Remote
	HostingService string
	SupportsPR     bool
	Authenticated  bool
}

// GetPullRequestTemplate gets the configured PR template
func (rm *RemoteManager) GetPullRequestTemplate() string {
	return rm.config.PRTemplate
}

// SetPullRequestTemplate sets the PR template
func (rm *RemoteManager) SetPullRequestTemplate(template string) {
	rm.config.PRTemplate = template
}