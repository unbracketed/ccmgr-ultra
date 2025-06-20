package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-username/ccmgr-ultra/internal/config"
)

func TestNewValidator(t *testing.T) {
	cfg := createTestConfig()
	validator := NewValidator(cfg)

	assert.NotNil(t, validator)
	assert.Equal(t, cfg, validator.config)
}

func TestNewValidator_NilConfig(t *testing.T) {
	validator := NewValidator(nil)

	assert.NotNil(t, validator)
	assert.NotNil(t, validator.config)
}

func TestValidateBranchName(t *testing.T) {
	validator := NewValidator(createTestConfig())

	testCases := []struct {
		name        string
		branchName  string
		valid       bool
		errorCount  int
		warnCount   int
	}{
		{
			name:       "Valid simple branch",
			branchName: "feature-branch",
			valid:      true,
		},
		{
			name:       "Valid branch with slashes",
			branchName: "feature/user-auth",
			valid:      true,
		},
		{
			name:       "Empty branch name",
			branchName: "",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch starting with dot",
			branchName: ".hidden-branch",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch starting with hyphen",
			branchName: "-invalid-branch",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch ending with dot",
			branchName: "branch.",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch ending with .lock",
			branchName: "branch.lock",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch with consecutive dots",
			branchName: "branch..name",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch with space",
			branchName: "branch name",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch with invalid characters",
			branchName: "branch~name",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch name as @",
			branchName: "@",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch with @{",
			branchName: "branch@{0}",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch starting with refs/",
			branchName: "refs/heads/branch",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Branch named HEAD",
			branchName: "HEAD",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Very long branch name",
			branchName: string(make([]byte, 150)),
			valid:      true,
			warnCount:  1,
		},
		{
			name:       "Reserved branch name",
			branchName: "master",
			valid:      true,
			warnCount:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.ValidateBranchName(tc.branchName)
			
			assert.Equal(t, tc.valid, result.Valid)
			if tc.errorCount > 0 {
				assert.Len(t, result.Errors, tc.errorCount)
			}
			if tc.warnCount > 0 {
				assert.Len(t, result.Warnings, tc.warnCount)
			}
		})
	}
}

func TestValidateWorktreePath(t *testing.T) {
	validator := NewValidator(createTestConfig())

	testCases := []struct {
		name       string
		path       string
		valid      bool
		errorCount int
	}{
		{
			name:  "Valid absolute path",
			path:  "/home/user/project",
			valid: true,
		},
		{
			name:       "Empty path",
			path:       "",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Relative path",
			path:       "relative/path",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Path with null byte",
			path:       "/home/user\x00/project",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Root directory",
			path:       "/",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Path with parent traversal",
			path:       "/home/../etc/passwd",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Windows reserved name",
			path:       "/home/user/CON",
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "Very long path",
			path:       "/" + string(make([]byte, 300)),
			valid:      false,
			errorCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.ValidateWorktreePath(tc.path)
			
			assert.Equal(t, tc.valid, result.Valid)
			if tc.errorCount > 0 {
				assert.Len(t, result.Errors, tc.errorCount)
			}
		})
	}
}

func TestValidateRepositoryState(t *testing.T) {
	validator := NewValidator(createTestConfig())

	// Test with nil repository
	result := validator.ValidateRepositoryState(nil)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "repository is nil")

	// Test with invalid repository
	repo := &Repository{
		RootPath: "",
		IsClean:  false,
	}
	result = validator.ValidateRepositoryState(repo)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1) // Empty root path

	// Test with valid repository
	tempDir := filepath.Join(os.TempDir(), "test-repo")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	repo = &Repository{
		RootPath: tempDir,
		IsClean:  true,
		Origin:   "git@github.com:user/repo.git",
	}
	result = validator.ValidateRepositoryState(repo)
	assert.True(t, result.Valid)
}

func TestSanitizeInput(t *testing.T) {
	validator := NewValidator(createTestConfig())

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "normal input",
			expected: "normal input",
		},
		{
			input:    "input\x00with\x00nulls",
			expected: "inputwithnulls",
		},
		{
			input:    "  extra   spaces  ",
			expected: "extra spaces",
		},
		{
			input:    "input\nwith\nnewlines",
			expected: "input\nwith\nnewlines",
		},
		{
			input:    "input\twith\ttabs",
			expected: "input\twith\ttabs",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := validator.SanitizeInput(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCheckPathSafety(t *testing.T) {
	validator := NewValidator(createTestConfig())

	// Set HOME environment variable for testing
	homeDir := "/home/testuser"
	os.Setenv("HOME", homeDir)
	defer os.Unsetenv("HOME")

	testCases := []struct {
		name string
		path string
		safe bool
	}{
		{
			name: "Empty path",
			path: "",
			safe: false,
		},
		{
			name: "Safe home path",
			path: "/home/testuser/project",
			safe: true,
		},
		{
			name: "Safe Users path",
			path: "/Users/testuser/project",
			safe: true,
		},
		{
			name: "Dangerous etc path",
			path: "/etc/passwd",
			safe: false,
		},
		{
			name: "Dangerous bin path",
			path: "/bin/bash",
			safe: false,
		},
		{
			name: "Path with parent traversal",
			path: "/home/testuser/../etc/passwd",
			safe: false,
		},
		{
			name: "Relative safe path",
			path: "relative/path",
			safe: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.CheckPathSafety(tc.path)
			assert.Equal(t, tc.safe, result)
		})
	}
}

func TestValidateOperationContext(t *testing.T) {
	validator := NewValidator(createTestConfig())
	
	tempDir := filepath.Join(os.TempDir(), "test-repo")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	repo := &Repository{
		RootPath: tempDir,
		IsClean:  true,
	}

	// Test create_worktree operation
	ctx := ValidationContext{
		Repository: repo,
		Operation:  "create_worktree",
		UserInput: map[string]interface{}{
			"branch": "feature-branch",
			"path":   "/home/user/new-worktree",
		},
	}

	result := validator.ValidateOperationContext(ctx)
	assert.True(t, result.Valid)

	// Test with invalid branch name
	ctx.UserInput["branch"] = ".invalid-branch"
	result = validator.ValidateOperationContext(ctx)
	assert.False(t, result.Valid)
}

func TestValidateConfiguration(t *testing.T) {
	cfg := createTestConfig()
	validator := NewValidator(cfg)

	result := validator.ValidateConfiguration()
	assert.True(t, result.Valid)

	// Test with invalid pattern
	cfg.Worktree.DirectoryPattern = "invalid-pattern"
	result = validator.ValidateConfiguration()
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "invalid directory pattern")
}

func TestContainsControlChars(t *testing.T) {
	validator := NewValidator(createTestConfig())

	assert.True(t, validator.containsControlChars("text\x00with\x01control"))
	assert.True(t, validator.containsControlChars("text\nwith\nnewline"))
	assert.False(t, validator.containsControlChars("normal text"))
}

func TestContainsInvalidChars(t *testing.T) {
	validator := NewValidator(createTestConfig())

	assert.True(t, validator.containsInvalidChars("text~with~invalid"))
	assert.True(t, validator.containsInvalidChars("text^with^invalid"))
	assert.True(t, validator.containsInvalidChars("text:with:invalid"))
	assert.True(t, validator.containsInvalidChars("text?with?invalid"))
	assert.True(t, validator.containsInvalidChars("text*with*invalid"))
	assert.True(t, validator.containsInvalidChars("text[with[invalid"))
	assert.True(t, validator.containsInvalidChars("text\\with\\invalid"))
	assert.False(t, validator.containsInvalidChars("normal-text"))
}

func TestIsReservedName(t *testing.T) {
	validator := NewValidator(createTestConfig())

	reservedNames := []string{
		"HEAD", "head", "Head",
		"master", "MASTER", "Master",
		"main", "MAIN", "Main",
		"develop", "DEVELOP", "Develop",
	}

	for _, name := range reservedNames {
		assert.True(t, validator.isReservedName(name))
	}

	assert.False(t, validator.isReservedName("feature-branch"))
	assert.False(t, validator.isReservedName("my-branch"))
}

func TestIsWindowsReservedPath(t *testing.T) {
	validator := NewValidator(createTestConfig())

	reservedPaths := []string{
		"/path/to/CON",
		"/path/to/PRN.txt",
		"/path/to/AUX",
		"/path/to/NUL.dat",
		"/path/to/COM1",
		"/path/to/LPT1.log",
	}

	for _, path := range reservedPaths {
		assert.True(t, validator.isWindowsReservedPath(path))
	}

	normalPaths := []string{
		"/path/to/normal",
		"/path/to/file.txt",
		"/path/to/CONSOLE", // Not exactly CON
	}

	for _, path := range normalPaths {
		assert.False(t, validator.isWindowsReservedPath(path))
	}
}

func TestValidatePathComponents(t *testing.T) {
	validator := NewValidator(createTestConfig())

	validPaths := []string{
		"/home/user/project",
		"/Users/john/workspace/app",
		"/opt/projects/my-app",
	}

	for _, path := range validPaths {
		err := validator.validatePathComponents(path)
		assert.NoError(t, err)
	}

	invalidPaths := []string{
		"/home/user/project<invalid>",
		"/home/user/project|pipe",
		"/home/user/project?question",
		"/home/user/" + string(make([]byte, 300)), // Component too long
	}

	for _, path := range invalidPaths {
		err := validator.validatePathComponents(path)
		assert.Error(t, err)
	}
}

func TestValidateWorktreeCreation(t *testing.T) {
	validator := NewValidator(createTestConfig())

	ctx := ValidationContext{
		Operation: "create_worktree",
		UserInput: map[string]interface{}{
			"branch": "valid-branch",
			"path":   "/home/user/worktree",
		},
	}

	result := &ValidationResult{Valid: true}
	result = validator.validateWorktreeCreation(ctx, result)
	assert.True(t, result.Valid)

	// Test with invalid branch
	ctx.UserInput["branch"] = ".invalid-branch"
	result = &ValidationResult{Valid: true}
	result = validator.validateWorktreeCreation(ctx, result)
	assert.False(t, result.Valid)
}

func TestValidateWorktreeDeletion(t *testing.T) {
	validator := NewValidator(createTestConfig())

	tempDir := filepath.Join(os.TempDir(), "test-repo")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	repo := &Repository{RootPath: tempDir}

	ctx := ValidationContext{
		Repository: repo,
		Operation:  "delete_worktree",
		UserInput: map[string]interface{}{
			"path": "/home/user/worktree",
		},
	}

	result := &ValidationResult{Valid: true}
	result = validator.validateWorktreeeDeletion(ctx, result)
	assert.True(t, result.Valid)

	// Test deleting main repository
	ctx.UserInput["path"] = tempDir
	result = &ValidationResult{Valid: true}
	result = validator.validateWorktreeeDeletion(ctx, result)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "cannot delete main repository")
}

func TestValidateBranchMerge(t *testing.T) {
	validator := NewValidator(createTestConfig())

	ctx := ValidationContext{
		Operation: "merge_branch",
		UserInput: map[string]interface{}{
			"source": "feature-branch",
			"target": "main",
		},
	}

	result := &ValidationResult{Valid: true}
	result = validator.validateBranchMerge(ctx, result)
	assert.True(t, result.Valid)

	// Test merging branch into itself
	ctx.UserInput["target"] = "feature-branch"
	result = &ValidationResult{Valid: true}
	result = validator.validateBranchMerge(ctx, result)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "cannot merge branch into itself")
}

func TestValidateBranchPush(t *testing.T) {
	validator := NewValidator(createTestConfig())

	repo := createTestRepository()
	ctx := ValidationContext{
		Repository: repo,
		Operation:  "push_branch",
		UserInput: map[string]interface{}{
			"branch": "feature-branch",
			"remote": "origin",
		},
	}

	result := &ValidationResult{Valid: true}
	result = validator.validateBranchPush(ctx, result)
	assert.True(t, result.Valid)

	// Test with non-existent remote
	ctx.UserInput["remote"] = "nonexistent"
	result = &ValidationResult{Valid: true}
	result = validator.validateBranchPush(ctx, result)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "remote 'nonexistent' not found")
}

func TestValidateCommitMessage(t *testing.T) {
	validator := NewValidator(createTestConfig())

	testCases := []struct {
		name       string
		message    string
		valid      bool
		hasWarning bool
	}{
		{
			name:    "Valid short message",
			message: "Fix bug in user authentication",
			valid:   true,
		},
		{
			name: "Valid message with body",
			message: "Fix bug in user authentication\n\nThis commit fixes the issue where users couldn't log in.",
			valid: true,
		},
		{
			name:    "Empty message",
			message: "",
			valid:   false,
		},
		{
			name:       "Long subject line",
			message:    string(make([]byte, 80)),
			valid:      true,
			hasWarning: true,
		},
		{
			name:    "Very long subject line",
			message: string(make([]byte, 150)),
			valid:   false,
		},
		{
			name:       "Missing blank line after subject",
			message:    "Subject\nBody without blank line",
			valid:      true,
			hasWarning: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.ValidateCommitMessage(tc.message)
			assert.Equal(t, tc.valid, result.Valid)
			if tc.hasWarning {
				assert.NotEmpty(t, result.Warnings)
			}
		})
	}
}

func TestValidateTagName(t *testing.T) {
	validator := NewValidator(createTestConfig())

	testCases := []struct {
		name    string
		tagName string
		valid   bool
	}{
		{
			name:    "Valid semver tag",
			tagName: "v1.0.0",
			valid:   true,
		},
		{
			name:    "Valid simple tag",
			tagName: "release-1",
			valid:   true,
		},
		{
			name:    "Empty tag name",
			tagName: "",
			valid:   false,
		},
		{
			name:    "Tag starting with dot",
			tagName: ".hidden-tag",
			valid:   false,
		},
		{
			name:    "Tag ending with dot",
			tagName: "tag.",
			valid:   false,
		},
		{
			name:    "Tag with consecutive dots",
			tagName: "v1..0",
			valid:   false,
		},
		{
			name:    "Tag with space",
			tagName: "tag name",
			valid:   false,
		},
		{
			name:    "Tag with invalid characters",
			tagName: "tag~name",
			valid:   false,
		},
		{
			name:    "Tag with @{",
			tagName: "tag@{0}",
			valid:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.ValidateTagName(tc.tagName)
			assert.Equal(t, tc.valid, result.Valid)
		})
	}
}

func TestPathExists(t *testing.T) {
	validator := NewValidator(createTestConfig())

	// Test with existing path (temp directory)
	tempDir := os.TempDir()
	assert.True(t, validator.pathExists(tempDir))

	// Test with non-existing path
	assert.False(t, validator.pathExists("/this/path/does/not/exist"))
}