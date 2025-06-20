# Phase 2.3 Git Worktree Management - Implementation Plan

## Overview

This document provides a comprehensive implementation plan for Phase 2.3: Git Worktree Management in the ccmgr-ultra project. This phase focuses on implementing robust git worktree operations that integrate seamlessly with the existing tmux session management and configuration systems.

## Objectives

Build a complete git worktree management system that provides:
- Full worktree lifecycle management (create, list, delete, merge)
- Integration with existing tmux session naming
- PR/MR creation capabilities for major git hosting services
- Robust error handling and validation
- Directory pattern management and auto-naming
- Branch synchronization and conflict resolution

## Architecture Overview

### Module Structure

```
internal/git/
├── worktree.go              # Core worktree operations and WorktreeManager
├── operations.go            # Git operations (push, merge, branch management)
├── repository.go            # Repository detection and validation
├── remote.go                # Remote repository operations and PR/MR creation
├── patterns.go              # Directory naming patterns and path management
├── validation.go            # Input validation and safety checks
├── worktree_test.go         # Unit tests for worktree management
├── operations_test.go       # Unit tests for git operations
├── repository_test.go       # Unit tests for repository operations
├── remote_test.go           # Unit tests for remote operations
├── patterns_test.go         # Unit tests for pattern management
├── validation_test.go       # Unit tests for validation logic
└── integration_test.go      # Integration tests with real git operations
```

## Implementation Tasks

### Task 1: Repository Detection and Validation (`internal/git/repository.go`)

**Priority: High**
**Estimated Time: 2-3 hours**

#### Requirements
- Detect if current directory is a git repository
- Validate repository state (clean working directory, no uncommitted changes)
- Identify repository metadata (origin, default branch, current branch)
- Handle various git repository states (bare, shallow, etc.)

#### Implementation Details

```go
// RepositoryManager handles git repository detection and validation
type RepositoryManager struct {
    repoPath string
    gitCmd   GitInterface
}

// Repository represents a git repository
type Repository struct {
    Path         string
    Origin       string
    DefaultBranch string
    CurrentBranch string
    IsClean      bool
    Remotes      []Remote
    Worktrees    []WorktreeInfo
}

// Remote represents a git remote
type Remote struct {
    Name     string
    URL      string
    Host     string    // github.com, gitlab.com, etc.
    Owner    string    // organization/user
    Repo     string    // repository name
    Protocol string    // https, ssh
}
```

#### Key Methods
- `DetectRepository(path string) (*Repository, error)`
- `ValidateRepositoryState(repo *Repository) error`
- `GetRepositoryInfo(path string) (*Repository, error)`
- `IsGitRepository(path string) bool`
- `GetRemoteInfo(remoteName string) (*Remote, error)`

#### Validation Rules
- Repository must exist and be valid
- Working directory should be clean for worktree operations
- Remote origin must be configured for PR/MR operations
- Current branch should not be the default branch for some operations

### Task 2: Directory Pattern Management (`internal/git/patterns.go`)

**Priority: High**
**Estimated Time: 2-3 hours**

#### Requirements
- Implement configurable directory naming patterns
- Support auto-directory creation based on patterns
- Handle path sanitization and validation
- Provide pattern variable substitution

#### Implementation Details

```go
// PatternManager handles directory naming patterns
type PatternManager struct {
    config *config.WorktreeConfig
}

// PatternContext provides variables for pattern substitution
type PatternContext struct {
    Project   string
    Branch    string
    Worktree  string
    Timestamp string
    UserName  string
}

// DirectoryPattern represents a naming pattern
type DirectoryPattern struct {
    Template    string
    AutoCreate  bool
    Sanitize    bool
    MaxLength   int
    Prefix      string
    Suffix      string
}
```

#### Key Methods
- `ApplyPattern(pattern string, context PatternContext) (string, error)`
- `ValidatePattern(pattern string) error`
- `SanitizePath(path string) string`
- `GenerateWorktreePath(branch, project string) (string, error)`
- `ResolvePatternVariables(template string, context PatternContext) string`

#### Pattern Variables
- `{{.project}}` - Project name
- `{{.branch}}` - Branch name (sanitized)
- `{{.worktree}}` - Worktree identifier
- `{{.timestamp}}` - Current timestamp
- `{{.user}}` - Git user name
- `{{.prefix}}` - Configured prefix
- `{{.suffix}}` - Configured suffix

### Task 3: Core Worktree Operations (`internal/git/worktree.go`)

**Priority: High**
**Estimated Time: 4-5 hours**

#### Requirements
- Create new worktrees with branch creation/checkout
- List existing worktrees with status information
- Delete worktrees with cleanup
- Manage worktree metadata and state

#### Implementation Details

```go
// WorktreeManager handles git worktree operations
type WorktreeManager struct {
    repo        *Repository
    patternMgr  *PatternManager
    gitCmd      GitInterface
    config      *config.GitConfig
}

// WorktreeInfo represents a git worktree
type WorktreeInfo struct {
    Path        string
    Branch      string
    Head        string
    IsClean     bool
    HasUncommitted bool
    LastCommit  CommitInfo
    TmuxSession string    // Associated tmux session
    Created     time.Time
    LastAccessed time.Time
}

// WorktreeOptions for worktree creation
type WorktreeOptions struct {
    Path       string
    Branch     string
    CreateBranch bool
    Force      bool
    Checkout   bool
    Remote     string
    TrackRemote bool
}
```

#### Key Methods
- `CreateWorktree(branch string, opts WorktreeOptions) (*WorktreeInfo, error)`
- `ListWorktrees() ([]WorktreeInfo, error)`
- `DeleteWorktree(path string, force bool) error`
- `GetWorktreeInfo(path string) (*WorktreeInfo, error)`
- `RefreshWorktreeStatus(path string) (*WorktreeInfo, error)`
- `PruneWorktrees() error` // Remove stale worktrees

#### Worktree Creation Logic
1. Validate branch name and target path
2. Apply directory naming pattern if needed
3. Create branch if it doesn't exist
4. Create worktree directory
5. Checkout branch in worktree
6. Update worktree metadata
7. Create associated tmux session if configured

### Task 4: Git Operations (`internal/git/operations.go`)

**Priority: High**
**Estimated Time: 3-4 hours**

#### Requirements
- Branch management (create, delete, merge)
- Push operations with conflict detection
- Merge operations with conflict resolution
- Commit and status operations
- Stash management for worktrees

#### Implementation Details

```go
// GitOperations handles low-level git operations
type GitOperations struct {
    repo   *Repository
    gitCmd GitInterface
}

// CommitInfo represents a git commit
type CommitInfo struct {
    Hash    string
    Author  string
    Date    time.Time
    Message string
    Files   []string
}

// BranchInfo represents a git branch
type BranchInfo struct {
    Name      string
    Remote    string
    Upstream  string
    Current   bool
    Head      string
    Behind    int
    Ahead     int
}

// MergeResult represents the result of a merge operation
type MergeResult struct {
    Success      bool
    Conflicts    []string
    FilesChanged int
    CommitHash   string
    Message      string
}
```

#### Key Methods
- `CreateBranch(name, source string) error`
- `DeleteBranch(name string, force bool) error`
- `MergeBranch(source, target string) (*MergeResult, error)`
- `PushBranch(branch, remote string, force bool) error`
- `GetBranchInfo(branch string) (*BranchInfo, error)`
- `GetCommitHistory(branch string, limit int) ([]CommitInfo, error)`
- `StashChanges(message string) error`
- `PopStash() error`

#### Safety Checks
- Prevent deletion of current branch
- Validate branch exists before operations
- Check for uncommitted changes before destructive operations
- Verify remote exists before push operations

### Task 5: Remote Operations and PR/MR Creation (`internal/git/remote.go`)

**Priority: Medium**
**Estimated Time: 4-5 hours**

#### Requirements
- Detect git hosting service (GitHub, GitLab, Bitbucket)
- Create pull requests / merge requests via API
- Handle authentication (tokens, SSH keys)
- Parse remote URLs and extract metadata

#### Implementation Details

```go
// RemoteManager handles remote repository operations
type RemoteManager struct {
    repo     *Repository
    config   *config.GitConfig
    clients  map[string]HostingClient
}

// HostingClient interface for different git hosting services
type HostingClient interface {
    CreatePullRequest(req PullRequestRequest) (*PullRequest, error)
    GetPullRequests(owner, repo string) ([]PullRequest, error)
    AuthenticateToken(token string) error
    ValidateRepository(owner, repo string) error
}

// PullRequestRequest represents a PR/MR creation request
type PullRequestRequest struct {
    Title       string
    Description string
    SourceBranch string
    TargetBranch string
    Owner       string
    Repository  string
    Draft       bool
    Labels      []string
}

// PullRequest represents a created PR/MR
type PullRequest struct {
    ID          int
    Number      int
    Title       string
    URL         string
    State       string
    CreatedAt   time.Time
    Author      string
}
```

#### Key Methods
- `DetectHostingService(remoteURL string) (string, error)`
- `CreatePullRequest(worktree *WorktreeInfo, req PullRequestRequest) (*PullRequest, error)`
- `PushAndCreatePR(worktree *WorktreeInfo, prOptions PullRequestRequest) (*PullRequest, error)`
- `GetHostingClient(service string) (HostingClient, error)`
- `ValidateAuthentication(service string) error`

#### Supported Services
- **GitHub**: GitHub API v4 (GraphQL) and v3 (REST)
- **GitLab**: GitLab API v4
- **Bitbucket**: Bitbucket API 2.0
- **Generic**: Basic git operations without PR/MR support

### Task 6: Input Validation and Safety (`internal/git/validation.go`)

**Priority: High**
**Estimated Time: 2-3 hours**

#### Requirements
- Validate branch names according to git rules
- Sanitize user inputs to prevent injection
- Validate file paths and directory names
- Check for potentially dangerous operations

#### Implementation Details

```go
// Validator handles input validation and safety checks
type Validator struct {
    config *config.GitConfig
}

// ValidationResult represents validation outcome
type ValidationResult struct {
    Valid   bool
    Errors  []string
    Warnings []string
}

// SafetyCheck represents a safety validation
type SafetyCheck struct {
    Name        string
    Description string
    Check       func(input interface{}) bool
    Required    bool
}
```

#### Key Methods
- `ValidateBranchName(name string) *ValidationResult`
- `ValidateWorktreePath(path string) *ValidationResult`
- `ValidateRepositoryState(repo *Repository) *ValidationResult`
- `SanitizeInput(input string) string`
- `CheckPathSafety(path string) bool`

#### Validation Rules
- Branch names must follow git naming conventions
- Paths must be within allowed directories
- No shell injection characters in inputs
- Prevent operations on protected branches
- Check disk space before creating worktrees

### Task 7: Configuration Integration

**Priority: Medium**
**Estimated Time: 2 hours**

#### Requirements
- Extend configuration schema for git settings
- Add validation for git configuration
- Provide sensible defaults
- Support configuration inheritance

#### Configuration Schema Extension

```go
// Add to internal/config/schema.go
type GitConfig struct {
    // Worktree settings
    AutoDirectory   bool              `yaml:"auto_directory" default:"true"`
    DirectoryPattern string           `yaml:"directory_pattern" default:"{{.project}}-{{.branch}}"`
    MaxWorktrees    int               `yaml:"max_worktrees" default:"10"`
    CleanupAge      time.Duration     `yaml:"cleanup_age" default:"7d"`
    
    // Branch settings
    DefaultBranch   string            `yaml:"default_branch" default:"main"`
    ProtectedBranches []string        `yaml:"protected_branches"`
    AllowForceDelete bool             `yaml:"allow_force_delete" default:"false"`
    
    // Remote settings
    DefaultRemote   string            `yaml:"default_remote" default:"origin"`
    AutoPush        bool              `yaml:"auto_push" default:"true"`
    CreatePR        bool              `yaml:"create_pr" default:"false"`
    PRTemplate      string            `yaml:"pr_template"`
    
    // Authentication
    GitHubToken     string            `yaml:"github_token" env:"GITHUB_TOKEN"`
    GitLabToken     string            `yaml:"gitlab_token" env:"GITLAB_TOKEN"`
    BitbucketToken  string            `yaml:"bitbucket_token" env:"BITBUCKET_TOKEN"`
    
    // Safety settings
    RequireCleanWorkdir bool          `yaml:"require_clean_workdir" default:"true"`
    ConfirmDestructive  bool          `yaml:"confirm_destructive" default:"true"`
    BackupOnDelete      bool          `yaml:"backup_on_delete" default:"true"`
}
```

### Task 8: Testing Implementation

**Priority: High**
**Estimated Time: 6-8 hours**

#### Test Categories

**Unit Tests (80% coverage minimum)**
- Repository detection and validation
- Pattern management and substitution
- Worktree operations with mocked git
- Git operations with mocked commands
- Input validation and sanitization
- Configuration validation

**Integration Tests**
- End-to-end worktree workflows
- Git operations with real repositories
- Pattern application in various scenarios
- Error handling with corrupted repositories
- Authentication with mock services

**Mock Implementation**
```go
// MockGitCmd for testing git operations
type MockGitCmd struct {
    commands []MockCommand
    results  map[string]MockResult
}

// MockHostingClient for testing remote operations
type MockHostingClient struct {
    prs     []PullRequest
    authOK  bool
    repoOK  bool
}
```

#### Test Scenarios
- Create worktree in various repository states
- Handle network failures during remote operations
- Recover from interrupted operations
- Validate input sanitization prevents injection
- Test pattern substitution with edge cases

## Integration Points

### Integration with Tmux Module
- Automatic tmux session creation for new worktrees
- Session naming consistency with worktree patterns
- Session cleanup when worktrees are deleted
- State synchronization between worktree and session

### Integration with Configuration System
- Git configuration validation and defaults
- Pattern configuration and testing
- Authentication token management
- Safety setting enforcement

### Integration with Hook System
- Pre/post worktree creation hooks
- Branch creation and deletion hooks
- Push and PR creation hooks
- Error and validation hooks

## Error Handling Strategy

### Error Categories
1. **User Errors**: Invalid input, missing requirements
2. **System Errors**: Git command failures, filesystem issues
3. **Network Errors**: Remote operation failures, API errors
4. **Configuration Errors**: Invalid settings, missing tokens

### Error Recovery
- Automatic retry for transient failures
- Rollback mechanisms for failed operations
- Graceful degradation when services unavailable
- Clear error messages with suggested actions

### Error Examples
```go
var (
    ErrRepositoryNotFound   = errors.New("git repository not found")
    ErrWorktreeExists      = errors.New("worktree already exists")
    ErrBranchInUse         = errors.New("branch is used by another worktree")
    ErrRemoteNotConfigured = errors.New("remote origin not configured")
    ErrAuthenticationFailed = errors.New("authentication failed for hosting service")
    ErrInvalidBranchName   = errors.New("invalid branch name")
    ErrWorkdirNotClean     = errors.New("working directory has uncommitted changes")
)
```

## Performance Considerations

### Optimization Strategies
- Cache repository information to avoid repeated git calls
- Lazy loading of worktree status information
- Efficient pattern matching and substitution
- Batch operations where possible
- Background refresh of worktree status

### Performance Targets
- Worktree creation: < 2 seconds
- Worktree listing: < 500ms
- Pattern application: < 100ms
- Repository detection: < 200ms
- Remote operations: < 5 seconds

## Security Considerations

### Security Measures
- Input sanitization to prevent command injection
- Path validation to prevent directory traversal
- Token encryption for stored credentials
- Secure API communication (HTTPS only)
- Audit logging for sensitive operations

### Security Validations
- Validate all user inputs before git operations
- Check file permissions before operations
- Verify SSL certificates for remote operations
- Prevent operations outside repository boundaries
- Log security-relevant events

## Deployment Strategy

### Development Phases
1. **Phase 1** (Days 1-2): Repository and pattern management
2. **Phase 2** (Days 3-4): Core worktree operations
3. **Phase 3** (Days 5-6): Git operations and safety checks
4. **Phase 4** (Days 7-8): Remote operations and PR/MR creation
5. **Phase 5** (Days 9-10): Testing and integration
6. **Phase 6** (Days 11-12): Documentation and polish

### Testing Milestones
- Unit tests after each component
- Integration tests after core functionality
- End-to-end tests before remote operations
- Performance testing before finalization
- Security testing throughout development

## Success Criteria

### Functional Requirements
- ✅ Create and manage git worktrees reliably
- ✅ Apply directory naming patterns correctly
- ✅ Integrate with existing tmux session management
- ✅ Support major git hosting services for PR/MR creation
- ✅ Provide robust error handling and validation
- ✅ Maintain repository safety and integrity

### Performance Requirements
- ✅ Worktree operations complete within performance targets
- ✅ Support concurrent worktree management
- ✅ Efficient resource usage and cleanup
- ✅ Responsive user experience

### Quality Requirements
- ✅ Comprehensive test coverage (>80%)
- ✅ Clear error messages and documentation
- ✅ Security validation and input sanitization
- ✅ Configuration validation and defaults
- ✅ Integration with existing codebase

### Integration Requirements
- ✅ Seamless integration with tmux module
- ✅ Configuration system compatibility
- ✅ Hook system integration
- ✅ Consistent with project architecture

## Risk Mitigation

### Identified Risks
1. **Git Repository Corruption**: Atomic operations and validation
2. **Network Failures**: Retry mechanisms and offline modes
3. **Authentication Issues**: Clear error messages and fallback options
4. **Performance Issues**: Caching and optimization strategies
5. **Security Vulnerabilities**: Input validation and secure practices

### Mitigation Strategies
- Comprehensive testing with various repository states
- Mock implementations for testing edge cases
- Fallback modes when services are unavailable
- Performance monitoring and optimization
- Security audits and validation

## Documentation Requirements

### Code Documentation
- Comprehensive godoc comments for all public APIs
- Usage examples in documentation
- Error handling documentation
- Configuration reference

### User Documentation
- Worktree management guide
- Pattern configuration examples
- PR/MR integration setup
- Troubleshooting guide

## Dependencies

### External Dependencies
- **git binary**: Required for all git operations
- **GitHub API**: Optional for GitHub integration
- **GitLab API**: Optional for GitLab integration
- **Bitbucket API**: Optional for Bitbucket integration

### Internal Dependencies
- **internal/config**: Configuration management
- **internal/tmux**: Session integration
- **internal/hooks**: Event hooks

### New Dependencies
- **github.com/go-git/go-git/v5**: Git operations library
- **github.com/google/go-github/v58**: GitHub API client
- **github.com/xanzy/go-gitlab**: GitLab API client

## Configuration Example

```yaml
git:
  # Worktree settings
  auto_directory: true
  directory_pattern: "{{.project}}-{{.branch}}"
  max_worktrees: 10
  cleanup_age: 168h  # 7 days
  
  # Branch settings
  default_branch: "main"
  protected_branches: ["main", "master", "develop"]
  allow_force_delete: false
  
  # Remote settings
  default_remote: "origin"
  auto_push: true
  create_pr: false
  pr_template: |
    ## Description
    Brief description of changes
    
    ## Testing
    How the changes were tested
  
  # Safety settings
  require_clean_workdir: true
  confirm_destructive: true
  backup_on_delete: true
```

## Conclusion

This implementation plan provides a comprehensive roadmap for implementing Phase 2.3: Git Worktree Management. The plan emphasizes robustness, security, and integration with existing components while maintaining high performance and usability standards.

The modular design ensures maintainability and extensibility, while comprehensive testing and validation provide confidence in the implementation's reliability. The integration points with existing modules ensure seamless user experience across the entire ccmgr-ultra application.

**Estimated Implementation Time: 10-12 days**
**Estimated Testing Time: 4-6 days**
**Total Estimated Time: 14-18 days**