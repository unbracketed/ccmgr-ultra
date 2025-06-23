package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unicode"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// Validator handles input validation and safety checks
type Validator struct {
	config *config.Config
}

// ValidationResult represents validation outcome
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// SafetyCheck represents a safety validation
type SafetyCheck struct {
	Name        string
	Description string
	Check       func(input interface{}) bool
	Required    bool
}

// ValidationContext provides context for validation
type ValidationContext struct {
	Repository *Repository
	Operation  string
	UserInput  map[string]interface{}
}

// NewValidator creates a new Validator instance
func NewValidator(cfg *config.Config) *Validator {
	if cfg == nil {
		cfg = &config.Config{}
		cfg.SetDefaults()
	}
	return &Validator{config: cfg}
}

// ValidateBranchName validates a git branch name according to git rules
func (v *Validator) ValidateBranchName(name string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "branch name cannot be empty")
		return result
	}

	// Git branch naming rules
	checks := []struct {
		condition bool
		error     string
	}{
		// Cannot start with a dot
		{strings.HasPrefix(name, "."), "branch name cannot start with a dot"},

		// Cannot start with a hyphen
		{strings.HasPrefix(name, "-"), "branch name cannot start with a hyphen"},

		// Cannot end with a dot
		{strings.HasSuffix(name, "."), "branch name cannot end with a dot"},

		// Cannot end with .lock
		{strings.HasSuffix(name, ".lock"), "branch name cannot end with .lock"},

		// Cannot contain consecutive dots
		{strings.Contains(name, ".."), "branch name cannot contain consecutive dots"},

		// Cannot contain space
		{strings.Contains(name, " "), "branch name cannot contain spaces"},

		// Cannot contain control characters
		{v.containsControlChars(name), "branch name cannot contain control characters"},

		// Cannot contain certain special characters
		{v.containsInvalidChars(name), "branch name cannot contain invalid characters (~, ^, :, ?, *, [, \\)"},

		// Cannot be just @ or contain @{
		{name == "@" || strings.Contains(name, "@{"), "branch name cannot be '@' or contain '@{'"},

		// Cannot start with refs/
		{strings.HasPrefix(name, "refs/"), "branch name cannot start with 'refs/'"},

		// Cannot be HEAD
		{strings.ToUpper(name) == "HEAD", "branch name cannot be 'HEAD'"},
	}

	for _, check := range checks {
		if check.condition {
			result.Valid = false
			result.Errors = append(result.Errors, check.error)
		}
	}

	// Warnings for potentially problematic names
	warnings := []struct {
		condition bool
		warning   string
	}{
		{len(name) > 100, "branch name is very long (>100 characters)"},
		{strings.Contains(name, "//"), "branch name contains consecutive slashes"},
		{v.isReservedName(name), "branch name might conflict with reserved names"},
	}

	for _, warning := range warnings {
		if warning.condition {
			result.Warnings = append(result.Warnings, warning.warning)
		}
	}

	return result
}

// ValidateWorktreePath validates a worktree path
func (v *Validator) ValidateWorktreePath(path string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if path == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "worktree path cannot be empty")
		return result
	}

	// Path safety checks
	checks := []struct {
		condition bool
		error     string
	}{
		// Must be absolute path
		{!filepath.IsAbs(path), "worktree path must be absolute"},

		// Cannot contain null bytes
		{strings.Contains(path, "\x00"), "path cannot contain null bytes"},

		// Cannot be root directory
		{filepath.Clean(path) == "/", "cannot create worktree in root directory"},

		// Cannot contain parent directory traversal after cleaning
		{strings.Contains(filepath.Clean(path), ".."), "path cannot contain parent directory traversal"},

		// Cannot be a reserved name on Windows
		{v.isWindowsReservedPath(path), "path uses Windows reserved name"},

		// Path length check
		{len(path) > 260, "path is too long (>260 characters)"},
	}

	for _, check := range checks {
		if check.condition {
			result.Valid = false
			result.Errors = append(result.Errors, check.error)
		}
	}

	// Additional path validation
	if err := v.validatePathComponents(path); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Check disk space if path parent exists
	if parentDir := filepath.Dir(path); parentDir != "." {
		if v.checkDiskSpace(parentDir) {
			result.Warnings = append(result.Warnings, "low disk space in target directory")
		}
	}

	return result
}

// ValidateRepositoryState validates the current state of a repository
func (v *Validator) ValidateRepositoryState(repo *Repository) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if repo == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "repository is nil")
		return result
	}

	// Repository state checks
	checks := []struct {
		condition bool
		error     string
		required  bool
	}{
		{repo.RootPath == "", "repository root path is empty", true},
		{!v.pathExists(repo.RootPath), "repository path does not exist", true},
		{!repo.IsClean, "repository has uncommitted changes", false}, // Warning for some operations
		{repo.Origin == "", "repository has no origin remote", false},
	}

	for _, check := range checks {
		if check.condition {
			if check.required {
				result.Valid = false
				result.Errors = append(result.Errors, check.error)
			} else {
				result.Warnings = append(result.Warnings, check.error)
			}
		}
	}

	return result
}

// SanitizeInput sanitizes user input to prevent injection attacks
func (v *Validator) SanitizeInput(input string) string {
	if input == "" {
		return input
	}

	// Remove null bytes
	sanitized := strings.ReplaceAll(input, "\x00", "")

	// Remove other control characters except newline and tab
	var result strings.Builder
	for _, r := range sanitized {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}

	sanitized = result.String()

	// Trim excessive whitespace
	sanitized = strings.TrimSpace(sanitized)

	// Replace multiple consecutive spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	sanitized = spaceRegex.ReplaceAllString(sanitized, " ")

	return sanitized
}

// CheckPathSafety performs comprehensive path safety checks
func (v *Validator) CheckPathSafety(path string) bool {
	if path == "" {
		return false
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"..", "/etc", "/bin", "/usr/bin", "/sbin", "/usr/sbin",
		"/var", "/tmp", "/dev", "/proc", "/sys", "/root",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cleanPath, pattern) {
			return false
		}
	}

	// Check if path is absolute and starts with allowed prefixes
	if filepath.IsAbs(cleanPath) {
		allowedPrefixes := []string{
			os.Getenv("HOME"),
			"/home",
			"/Users",
			"/opt",
			"/workspace",
		}

		allowed := false
		for _, prefix := range allowedPrefixes {
			if prefix != "" && strings.HasPrefix(cleanPath, prefix) {
				allowed = true
				break
			}
		}

		if !allowed {
			return false
		}
	}

	return true
}

// ValidateOperationContext validates the context for a git operation
func (v *Validator) ValidateOperationContext(ctx ValidationContext) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate repository if required
	if ctx.Repository != nil {
		repoResult := v.ValidateRepositoryState(ctx.Repository)
		if !repoResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, repoResult.Errors...)
		}
		result.Warnings = append(result.Warnings, repoResult.Warnings...)
	}

	// Operation-specific validation
	switch ctx.Operation {
	case "create_worktree":
		result = v.validateWorktreeCreation(ctx, result)
	case "delete_worktree":
		result = v.validateWorktreeeDeletion(ctx, result)
	case "merge_branch":
		result = v.validateBranchMerge(ctx, result)
	case "push_branch":
		result = v.validateBranchPush(ctx, result)
	}

	return result
}

// ValidateConfiguration validates git-related configuration
func (v *Validator) ValidateConfiguration() *ValidationResult {
	result := &ValidationResult{Valid: true}

	if v.config == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "configuration is nil")
		return result
	}

	// Validate worktree configuration
	if v.config.Worktree.DirectoryPattern != "" {
		patternMgr := NewPatternManager(&v.config.Worktree)
		if err := patternMgr.ValidatePattern(v.config.Worktree.DirectoryPattern); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("invalid directory pattern: %v", err))
		}
	}

	// Validate tmux configuration
	if v.config.Tmux.MaxSessionName < 1 {
		result.Warnings = append(result.Warnings, "tmux max session name length is very small")
	}

	if v.config.Tmux.MonitorInterval < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "tmux monitor interval cannot be negative")
	}

	return result
}

// Helper functions

// containsControlChars checks if string contains control characters
func (v *Validator) containsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// containsInvalidChars checks for git-invalid characters
func (v *Validator) containsInvalidChars(s string) bool {
	invalidChars := "~^:?*[\\"
	return strings.ContainsAny(s, invalidChars)
}

// isReservedName checks if name conflicts with common reserved names
func (v *Validator) isReservedName(name string) bool {
	reserved := []string{
		"HEAD", "FETCH_HEAD", "ORIG_HEAD", "MERGE_HEAD",
		"master", "main", "develop", "development",
		"staging", "production", "prod", "test",
	}

	lowerName := strings.ToLower(name)
	for _, res := range reserved {
		if lowerName == strings.ToLower(res) {
			return true
		}
	}

	return false
}

// isWindowsReservedPath checks for Windows reserved paths
func (v *Validator) isWindowsReservedPath(path string) bool {
	base := strings.ToUpper(filepath.Base(path))

	// Remove extension
	if idx := strings.LastIndex(base, "."); idx != -1 {
		base = base[:idx]
	}

	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, res := range reserved {
		if base == res {
			return true
		}
	}

	return false
}

// validatePathComponents validates individual path components
func (v *Validator) validatePathComponents(path string) error {
	components := strings.Split(path, string(filepath.Separator))

	for _, component := range components {
		if component == "" {
			continue
		}

		// Check component length
		if len(component) > 255 {
			return fmt.Errorf("path component '%s' is too long (>255 characters)", component)
		}

		// Check for invalid characters in filename
		invalidChars := "<>:\"|?*"
		if strings.ContainsAny(component, invalidChars) {
			return fmt.Errorf("path component '%s' contains invalid characters", component)
		}

		// Check for leading/trailing dots or spaces
		if strings.HasPrefix(component, ".") && len(component) > 1 {
			// Allow single dot, but warn about hidden files
			continue
		}

		if strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return fmt.Errorf("path component '%s' cannot end with dot or space", component)
		}
	}

	return nil
}

// pathExists checks if a path exists
func (v *Validator) pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// checkDiskSpace checks if disk space is low (basic implementation)
func (v *Validator) checkDiskSpace(path string) bool {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return false
	}

	// Calculate available space in bytes
	availableSpace := stat.Bavail * uint64(stat.Bsize)

	// Consider low if less than 1GB available
	return availableSpace < 1024*1024*1024
}

// validateWorktreeCreation validates worktree creation context
func (v *Validator) validateWorktreeCreation(ctx ValidationContext, result *ValidationResult) *ValidationResult {
	// Check if branch name is provided and valid
	if branchName, ok := ctx.UserInput["branch"].(string); ok {
		branchResult := v.ValidateBranchName(branchName)
		if !branchResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, branchResult.Errors...)
		}
		result.Warnings = append(result.Warnings, branchResult.Warnings...)
	}

	// Check if worktree path is provided and valid
	if path, ok := ctx.UserInput["path"].(string); ok {
		pathResult := v.ValidateWorktreePath(path)
		if !pathResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, pathResult.Errors...)
		}
		result.Warnings = append(result.Warnings, pathResult.Warnings...)
	}

	return result
}

// validateWorktreeeDeletion validates worktree deletion context
func (v *Validator) validateWorktreeeDeletion(ctx ValidationContext, result *ValidationResult) *ValidationResult {
	// Check if path is provided
	if path, ok := ctx.UserInput["path"].(string); ok && path != "" {
		// Ensure we're not deleting the main repository
		if ctx.Repository != nil && filepath.Clean(path) == filepath.Clean(ctx.Repository.RootPath) {
			result.Valid = false
			result.Errors = append(result.Errors, "cannot delete main repository worktree")
		}

		// Check path safety
		if !v.CheckPathSafety(path) {
			result.Valid = false
			result.Errors = append(result.Errors, "worktree path is not safe for deletion")
		}
	} else {
		result.Valid = false
		result.Errors = append(result.Errors, "worktree path must be provided for deletion")
	}

	return result
}

// validateBranchMerge validates branch merge context
func (v *Validator) validateBranchMerge(ctx ValidationContext, result *ValidationResult) *ValidationResult {
	// Check if source and target branches are provided
	sourceBranch, hasSource := ctx.UserInput["source"].(string)
	targetBranch, hasTarget := ctx.UserInput["target"].(string)

	if !hasSource || sourceBranch == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "source branch must be provided for merge")
	}

	if !hasTarget || targetBranch == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "target branch must be provided for merge")
	}

	// Validate branch names
	if hasSource {
		branchResult := v.ValidateBranchName(sourceBranch)
		if !branchResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, branchResult.Errors...)
		}
	}

	if hasTarget {
		branchResult := v.ValidateBranchName(targetBranch)
		if !branchResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, branchResult.Errors...)
		}
	}

	// Check if trying to merge branch into itself
	if hasSource && hasTarget && sourceBranch == targetBranch {
		result.Valid = false
		result.Errors = append(result.Errors, "cannot merge branch into itself")
	}

	return result
}

// validateBranchPush validates branch push context
func (v *Validator) validateBranchPush(ctx ValidationContext, result *ValidationResult) *ValidationResult {
	// Check if repository has remotes
	if ctx.Repository != nil && len(ctx.Repository.Remotes) == 0 {
		result.Warnings = append(result.Warnings, "repository has no remotes configured")
	}

	// Check if branch is provided and valid
	if branchName, ok := ctx.UserInput["branch"].(string); ok && branchName != "" {
		branchResult := v.ValidateBranchName(branchName)
		if !branchResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, branchResult.Errors...)
		}
	}

	// Check if remote is valid
	if remoteName, ok := ctx.UserInput["remote"].(string); ok && remoteName != "" {
		if ctx.Repository != nil {
			found := false
			for _, remote := range ctx.Repository.Remotes {
				if remote.Name == remoteName {
					found = true
					break
				}
			}
			if !found {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("remote '%s' not found", remoteName))
			}
		}
	}

	return result
}

// ValidateCommitMessage validates a git commit message
func (v *Validator) ValidateCommitMessage(message string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if message == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "commit message cannot be empty")
		return result
	}

	// Basic commit message validation
	lines := strings.Split(message, "\n")

	// Check first line (subject)
	subject := strings.TrimSpace(lines[0])
	if len(subject) > 72 {
		result.Warnings = append(result.Warnings, "commit subject line is longer than 72 characters")
	}

	if len(subject) > 100 {
		result.Valid = false
		result.Errors = append(result.Errors, "commit subject line is too long (>100 characters)")
	}

	// Check for blank line after subject if there's a body
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		result.Warnings = append(result.Warnings, "commit message should have blank line after subject")
	}

	// Check body lines
	for i, line := range lines[2:] {
		if len(line) > 100 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d is longer than 100 characters", i+3))
		}
	}

	return result
}

// ValidateTagName validates a git tag name
func (v *Validator) ValidateTagName(name string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "tag name cannot be empty")
		return result
	}

	// Similar to branch name validation but with different rules
	checks := []struct {
		condition bool
		error     string
	}{
		{strings.HasPrefix(name, "."), "tag name cannot start with a dot"},
		{strings.HasSuffix(name, "."), "tag name cannot end with a dot"},
		{strings.Contains(name, ".."), "tag name cannot contain consecutive dots"},
		{strings.Contains(name, " "), "tag name cannot contain spaces"},
		{v.containsControlChars(name), "tag name cannot contain control characters"},
		{v.containsInvalidChars(name), "tag name cannot contain invalid characters"},
		{strings.Contains(name, "@{"), "tag name cannot contain '@{'"},
	}

	for _, check := range checks {
		if check.condition {
			result.Valid = false
			result.Errors = append(result.Errors, check.error)
		}
	}

	return result
}
