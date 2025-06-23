package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/unbracketed/ccmgr-ultra/internal/analytics"
	"github.com/unbracketed/ccmgr-ultra/internal/config"
)

// RemoteManager handles remote repository operations
type RemoteManager struct {
	repo             *Repository
	config           *config.GitConfig
	clients          map[string]HostingClient
	gitCmd           GitInterface
	analyticsEmitter analytics.EventEmitter
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
	ID           int
	Number       int
	Title        string
	URL          string
	State        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Author       string
	SourceBranch string
	TargetBranch string
	Draft        bool
	Labels       []string
}

// GitHub API response structures
type GitHubPullRequestResponse struct {
	ID        int           `json:"id"`
	Number    int           `json:"number"`
	Title     string        `json:"title"`
	HTMLURL   string        `json:"html_url"`
	State     string        `json:"state"`
	Draft     bool          `json:"draft"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	User      GitHubUser    `json:"user"`
	Head      GitHubBranch  `json:"head"`
	Base      GitHubBranch  `json:"base"`
	Labels    []GitHubLabel `json:"labels"`
}

type GitHubUser struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
}

type GitHubBranch struct {
	Ref  string     `json:"ref"`
	SHA  string     `json:"sha"`
	Repo GitHubRepo `json:"repo"`
}

type GitHubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type GitHubLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GitHubClient implements HostingClient for GitHub
type GitHubClient struct {
	token  string
	apiURL string
}

// Removed GitLab and Bitbucket clients per Phase 5.3 spec - focusing on GitHub only

// GenericClient for repositories without PR/MR support
type GenericClient struct{}

// NewRemoteManager creates a new RemoteManager
func NewRemoteManager(repo *Repository, cfg *config.GitConfig, gitCmd GitInterface) *RemoteManager {
	if gitCmd == nil {
		gitCmd = NewGitCmd()
	}
	if cfg == nil {
		cfg = &config.GitConfig{}
	}

	rm := &RemoteManager{
		repo:             repo,
		config:           cfg,
		clients:          make(map[string]HostingClient),
		gitCmd:           gitCmd,
		analyticsEmitter: nil, // Will be set via SetAnalyticsEmitter if needed
	}

	// Initialize hosting clients
	rm.initializeClients()

	return rm
}

// SetAnalyticsEmitter sets the analytics emitter for tracking GitHub operations
func (rm *RemoteManager) SetAnalyticsEmitter(emitter analytics.EventEmitter) {
	rm.analyticsEmitter = emitter
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
	if req.Description == "" {
		// Use GitHub-specific template if available for GitHub service
		if service == "github" && rm.config.GitHubPRTemplate != "" {
			req.Description = rm.config.GitHubPRTemplate
		} else if rm.config.PRTemplate != "" {
			req.Description = rm.config.PRTemplate
		}
	}

	// Create the pull request
	pr, err := client.CreatePullRequest(req)

	// Emit analytics event for PR creation
	if rm.analyticsEmitter != nil && rm.analyticsEmitter.IsEnabled() {
		prEvent := analytics.AnalyticsEvent{
			Type:      analytics.EventTypeGitHubPRCreated,
			Timestamp: time.Now(),
			SessionID: rm.getCurrentSessionID(),
			Data: analytics.NewGitHubPREventData(
				getPRNumber(pr), getPRURL(pr), req.Title, req.SourceBranch, req.TargetBranch,
				worktree.Path, req.Draft, err == nil, getErrorMessage(err),
			),
		}
		rm.analyticsEmitter.EmitEventAsync(prEvent)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return pr, nil
}

// PushAndCreatePR pushes a worktree branch and creates a PR in one operation
func (rm *RemoteManager) PushAndCreatePR(worktree *WorktreeInfo, prOptions PullRequestRequest) (*PullRequest, error) {
	// Push the branch first
	pushErr := rm.ensureBranchPushed(worktree.Branch)

	// Emit analytics event for push operation
	if rm.analyticsEmitter != nil && rm.analyticsEmitter.IsEnabled() {
		pushEvent := analytics.AnalyticsEvent{
			Type:      analytics.EventTypeGitHubPush,
			Timestamp: time.Now(),
			SessionID: rm.getCurrentSessionID(),
			Data:      analytics.NewGitHubPushEventData(worktree.Branch, rm.config.DefaultRemote, worktree.Path, pushErr == nil, getErrorMessage(pushErr)),
		}
		rm.analyticsEmitter.EmitEventAsync(pushEvent)
	}

	if pushErr != nil {
		return nil, fmt.Errorf("failed to push branch: %w", pushErr)
	}

	// Create the pull request
	return rm.CreatePullRequest(worktree, prOptions)
}

// PushBranch pushes a branch to remote without creating a PR
func (rm *RemoteManager) PushBranch(branch string) error {
	err := rm.ensureBranchPushed(branch)

	// Emit analytics event for push operation
	if rm.analyticsEmitter != nil && rm.analyticsEmitter.IsEnabled() {
		pushEvent := analytics.AnalyticsEvent{
			Type:      analytics.EventTypeGitHubPush,
			Timestamp: time.Now(),
			SessionID: rm.getCurrentSessionID(),
			Data:      analytics.NewGitHubPushEventData(branch, rm.config.DefaultRemote, "", err == nil, getErrorMessage(err)),
		}
		rm.analyticsEmitter.EmitEventAsync(pushEvent)
	}

	return err
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

	// Get the appropriate token - currently only GitHub is supported
	var token string
	switch service {
	case "github":
		token = rm.config.GitHubToken
	default:
		return fmt.Errorf("authentication not supported for service: %s (only GitHub is currently supported)", service)
	}

	if token == "" {
		return fmt.Errorf("no authentication token configured for %s", service)
	}

	return client.AuthenticateToken(token)
}

// initializeClients sets up hosting service clients
func (rm *RemoteManager) initializeClients() {
	// GitHub client - primary focus for Phase 5.3
	if rm.config.GitHubToken != "" {
		rm.clients["github"] = &GitHubClient{
			token:  rm.config.GitHubToken,
			apiURL: "https://api.github.com",
		}
	}

	// Generic client (always available for non-GitHub repos)
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

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	headers := buildAuthHeaders("github", gc.token)
	resp, err := makeHTTPRequest("POST", apiURL, headers, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var githubPR GitHubPullRequestResponse
	if err := parseJSONResponse(resp, &githubPR); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our PR format
	pr := &PullRequest{
		ID:           githubPR.ID,
		Number:       githubPR.Number,
		Title:        githubPR.Title,
		URL:          githubPR.HTMLURL,
		State:        githubPR.State,
		CreatedAt:    githubPR.CreatedAt,
		UpdatedAt:    githubPR.UpdatedAt,
		Author:       githubPR.User.Login,
		SourceBranch: githubPR.Head.Ref,
		TargetBranch: githubPR.Base.Ref,
		Draft:        githubPR.Draft,
	}

	// Extract labels
	for _, label := range githubPR.Labels {
		pr.Labels = append(pr.Labels, label.Name)
	}

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

	// Call /user endpoint to validate token
	apiURL := fmt.Sprintf("%s/user", gc.apiURL)
	headers := buildAuthHeaders("github", token)

	resp, err := makeHTTPRequest("GET", apiURL, headers, nil)
	if err != nil {
		return fmt.Errorf("failed to authenticate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid GitHub token")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

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

// GitLab and Bitbucket clients removed per Phase 5.3 spec - focusing on GitHub only

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
func makeHTTPRequest(method, apiURL string, headers map[string]string, body []byte) (*http.Response, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// Create request
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set content type for POST/PUT requests
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Set User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "ccmgr-ultra/1.0")
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == 403 {
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, fmt.Errorf("GitHub API rate limit exceeded")
		}
	}

	return resp, nil
}

// parseJSONResponse is a helper function for parsing JSON responses
func parseJSONResponse(resp *http.Response, target interface{}) error {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return nil
}

// buildAuthHeaders creates authentication headers for GitHub
func buildAuthHeaders(service, token string) map[string]string {
	headers := make(map[string]string)

	switch service {
	case "github":
		headers["Authorization"] = fmt.Sprintf("token %s", token)
		headers["Accept"] = "application/vnd.github.v3+json"
	default:
		// Only GitHub is supported in Phase 5.3
		headers["Authorization"] = fmt.Sprintf("token %s", token)
		headers["Accept"] = "application/vnd.github.v3+json"
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

// Helper functions for analytics

// getCurrentSessionID gets the current session ID for analytics
func (rm *RemoteManager) getCurrentSessionID() string {
	// For now, return a simple session ID. In a full implementation,
	// this would integrate with the session management system
	return fmt.Sprintf("session-%d", time.Now().Unix())
}

// getErrorMessage safely extracts error message
func getErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// getPRNumber safely extracts PR number
func getPRNumber(pr *PullRequest) int {
	if pr == nil {
		return 0
	}
	return pr.Number
}

// getPRURL safely extracts PR URL
func getPRURL(pr *PullRequest) string {
	if pr == nil {
		return ""
	}
	return pr.URL
}

// GitLabClient - stub implementation for tests
type GitLabClient struct {
	token  string
	apiURL string
}

// NewGitLabClient creates a new GitLab client (stub implementation)
func NewGitLabClient(token string) *GitLabClient {
	return &GitLabClient{
		token:  token,
		apiURL: "https://gitlab.com/api/v4",
	}
}

// GetHostingService returns the service name
func (gc *GitLabClient) GetHostingService() string {
	return "gitlab"
}

// CreatePullRequest creates a GitLab merge request (stub implementation)
func (gc *GitLabClient) CreatePullRequest(req PullRequestRequest) (*PullRequest, error) {
	return nil, fmt.Errorf("GitLab client not fully implemented")
}

// GetPullRequests lists GitLab merge requests (stub implementation)
func (gc *GitLabClient) GetPullRequests(owner, repo string) ([]PullRequest, error) {
	return nil, fmt.Errorf("GitLab client not fully implemented")
}

// AuthenticateToken validates GitLab token (stub implementation)
func (gc *GitLabClient) AuthenticateToken(token string) error {
	return fmt.Errorf("GitLab client not fully implemented")
}

// ValidateRepository validates GitLab repository (stub implementation)
func (gc *GitLabClient) ValidateRepository(owner, repo string) error {
	return fmt.Errorf("GitLab client not fully implemented")
}

// BitbucketClient - stub implementation for tests
type BitbucketClient struct {
	token  string
	apiURL string
}

// NewBitbucketClient creates a new Bitbucket client (stub implementation)
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

// CreatePullRequest creates a Bitbucket pull request (stub implementation)
func (bc *BitbucketClient) CreatePullRequest(req PullRequestRequest) (*PullRequest, error) {
	return nil, fmt.Errorf("Bitbucket client not fully implemented")
}

// GetPullRequests lists Bitbucket pull requests (stub implementation)
func (bc *BitbucketClient) GetPullRequests(owner, repo string) ([]PullRequest, error) {
	return nil, fmt.Errorf("Bitbucket client not fully implemented")
}

// AuthenticateToken validates Bitbucket token (stub implementation)
func (bc *BitbucketClient) AuthenticateToken(token string) error {
	return fmt.Errorf("Bitbucket client not fully implemented")
}

// ValidateRepository validates Bitbucket repository (stub implementation)
func (bc *BitbucketClient) ValidateRepository(owner, repo string) error {
	return fmt.Errorf("Bitbucket client not fully implemented")
}
