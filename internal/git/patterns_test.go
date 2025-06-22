package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

func TestNewPatternManager(t *testing.T) {
	// Test with nil config
	pm := NewPatternManager(nil)
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.config)

	// Test with provided config
	cfg := &config.WorktreeConfig{
		DirectoryPattern: "{{.project}}-{{.branch}}",
	}
	pm = NewPatternManager(cfg)
	assert.NotNil(t, pm)
	assert.Equal(t, cfg, pm.config)
}

func TestValidatePattern(t *testing.T) {
	pm := NewPatternManager(nil)

	testCases := []struct {
		name    string
		pattern string
		valid   bool
	}{
		{
			name:    "Valid simple pattern",
			pattern: "{{.Project}}-{{.Branch}}",
			valid:   true,
		},
		{
			name:    "Valid complex pattern",
			pattern: "{{.Prefix}}-{{.Project}}-{{.Branch}}-{{.Timestamp}}",
			valid:   true,
		},
		{
			name:    "Empty pattern",
			pattern: "",
			valid:   false,
		},
		{
			name:    "No template variables",
			pattern: "static-name",
			valid:   false,
		},
		{
			name:    "Invalid template syntax",
			pattern: "{{.Project}-{{.Branch}}",
			valid:   false,
		},
		{
			name:    "Parent directory traversal",
			pattern: "../{{.Project}}-{{.Branch}}",
			valid:   false,
		},
		{
			name:    "Home directory",
			pattern: "~/{{.Project}}-{{.Branch}}",
			valid:   false,
		},
		{
			name:    "Absolute path",
			pattern: "/tmp/{{.Project}}-{{.Branch}}",
			valid:   false,
		},
		{
			name:    "Unknown variable",
			pattern: "{{.Project}}-{{.unknown}}",
			valid:   false,
		},
		{
			name:    "Valid with functions",
			pattern: "{{.Project | lower}}-{{.Branch | sanitize}}",
			valid:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pm.ValidatePattern(tc.pattern)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	pm := NewPatternManager(nil)

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "normal-path",
			expected: "normal-path",
		},
		{
			input:    "path with spaces",
			expected: "path-with-spaces",
		},
		{
			input:    "path/with/slashes",
			expected: "path-with-slashes",
		},
		{
			input:    "path<>:\"|?*with-unsafe-chars",
			expected: "path-with-unsafe-chars",
		},
		{
			input:    "path---with---multiple---dashes",
			expected: "path-with-multiple-dashes",
		},
		{
			input:    "---leading-and-trailing---",
			expected: "leading-and-trailing",
		},
		{
			input:    "",
			expected: "worktree",
		},
		{
			input:    "___",
			expected: "worktree",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := pm.SanitizePath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeComponent(t *testing.T) {
	pm := NewPatternManager(nil)

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "feature/user-auth",
			expected: "feature-user-auth",
		},
		{
			input:    "bugfix\\memory-leak",
			expected: "bugfix-memory-leak",
		},
		{
			input:    "Feature Branch",
			expected: "feature-branch",
		},
		{
			input:    "UPPER_CASE_BRANCH",
			expected: "upper-case-branch",
		},
		{
			input:    "branch@#$%^&*()",
			expected: "branch",
		},
		{
			input:    "branch-with-numbers-123",
			expected: "branch-with-numbers-123",
		},
		{
			input:    "branch.with.dots",
			expected: "branch.with.dots",
		},
		{
			input:    "",
			expected: "unnamed",
		},
		{
			input:    "@#$%",
			expected: "unnamed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := pm.sanitizeComponent(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestApplyPattern(t *testing.T) {
	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
	})

	context := PatternContext{
		Project:   "my-project",
		Branch:    "feature-auth",
		Worktree:  "feature-auth-0102-1430",
		Timestamp: "20240102-143045",
		UserName:  "john-doe",
		Prefix:    "main",
		Suffix:    "dev",
	}

	testCases := []struct {
		name     string
		pattern  string
		expected string
		hasError bool
	}{
		{
			name:     "Simple pattern",
			pattern:  "{{.Project}}-{{.Branch}}",
			expected: "my-project-feature-auth",
			hasError: false,
		},
		{
			name:     "Complex pattern",
			pattern:  "{{.Prefix}}-{{.Project}}-{{.Branch}}-{{.Timestamp}}",
			expected: "main-my-project-feature-auth-20240102-143045",
			hasError: false,
		},
		{
			name:     "Pattern with functions",
			pattern:  "{{.Project | upper}}-{{.Branch | lower}}",
			expected: "MY-PROJECT-feature-auth",
			hasError: false,
		},
		{
			name:     "Invalid pattern",
			pattern:  "{{.unknown}}",
			expected: "",
			hasError: true,
		},
		{
			name:     "Empty pattern uses default",
			pattern:  "",
			expected: "my-project-feature-auth",
			hasError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pm.ApplyPattern(tc.pattern, context)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestResolvePatternVariables(t *testing.T) {
	pm := NewPatternManager(nil)

	context := PatternContext{
		Project:   "test-project",
		Branch:    "main",
		Worktree:  "main-0102-1430",
		Timestamp: "20240102-143045",
		UserName:  "test-user",
		Prefix:    "prefix",
		Suffix:    "suffix",
	}

	testCases := []struct {
		name     string
		template string
		expected string
		hasError bool
	}{
		{
			name:     "All variables",
			template: "{{.Project}}-{{.Branch}}-{{.Worktree}}-{{.Timestamp}}-{{.UserName}}-{{.Prefix}}-{{.Suffix}}",
			expected: "test-project-main-main-0102-1430-20240102-143045-test-user-prefix-suffix",
			hasError: false,
		},
		{
			name:     "With functions",
			template: "{{.Project | upper}}-{{.Branch | lower}}",
			expected: "TEST-PROJECT-main",
			hasError: false,
		},
		{
			name:     "With replace function",
			template: "{{.Branch | replace \"/\" \"-\"}}",
			expected: "main",
			hasError: false,
		},
		{
			name:     "Invalid syntax",
			template: "{{.Project}-{{.Branch}}",
			expected: "",
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pm.ResolvePatternVariables(tc.template, context)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "my-project")
	require.NoError(t, err)

	// Should be relative to .worktrees directory
	cwd, _ := os.Getwd()
	expectedBase := filepath.Join(cwd, ".worktrees")
	expectedPath := filepath.Join(expectedBase, "my-project-feature-auth")

	assert.Equal(t, filepath.Clean(expectedPath), path)
}

func TestGenerateWorktreePath_CreatesWorktreesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-worktrees-creation")
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	
	os.MkdirAll(tempDir, 0755)
	os.Chdir(tempDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
		DefaultBranch:    "main",
	})

	// Verify .worktrees doesn't exist yet
	worktreesDir := filepath.Join(tempDir, ".worktrees")
	_, err := os.Stat(worktreesDir)
	assert.True(t, os.IsNotExist(err))

	// Generate worktree path
	path, err := pm.GenerateWorktreePath("feature/test", "test-project")
	require.NoError(t, err)

	// Verify .worktrees directory was created
	stat, err := os.Stat(worktreesDir)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Verify correct permissions
	assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())

	// Verify path is within .worktrees
	assert.Contains(t, path, ".worktrees")
	assert.Contains(t, path, "test-project-feature-test")
}

func TestGenerateWorktreePath_UsesWorktreesBasePath(t *testing.T) {
	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "my-project")
	require.NoError(t, err)

	// Should use .worktrees as base directory, not parent directory
	assert.Contains(t, path, ".worktrees")
	assert.NotContains(t, path, "..")
	
	cwd, _ := os.Getwd()
	expectedPrefix := filepath.Join(cwd, ".worktrees")
	assert.True(t, strings.HasPrefix(path, expectedPrefix))
}

func TestGenerateWorktreePath_DirectoryCreationError(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-creation-error")
	defer os.RemoveAll(tempDir)
	
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	
	os.MkdirAll(tempDir, 0755)
	os.Chdir(tempDir)

	// Create a file where .worktrees directory should be (to cause error)
	worktreesPath := filepath.Join(tempDir, ".worktrees")
	file, err := os.Create(worktreesPath)
	require.NoError(t, err)
	file.Close()

	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
		DefaultBranch:    "main",
	})

	// Should fail because .worktrees exists as a file, not directory
	_, err = pm.GenerateWorktreePath("feature/test", "test-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create .worktrees directory")
}

func TestGenerateWorktreePath_PreservesPatternFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-pattern-functionality")
	defer os.RemoveAll(tempDir)
	
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	
	os.MkdirAll(tempDir, 0755)
	os.Chdir(tempDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Prefix}}-{{.Project}}-{{.Branch}}-{{.Suffix}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "my-project")
	require.NoError(t, err)

	// Should preserve all pattern functionality
	assert.Contains(t, path, ".worktrees")
	assert.Contains(t, path, "main-my-project-feature-auth")
	
	// Verify the pattern variables are properly processed
	dirName := filepath.Base(path)
	assert.Equal(t, "main-my-project-feature-auth", dirName)
}

func TestGenerateWorktreePath_CorrectPermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-permissions")
	defer os.RemoveAll(tempDir)
	
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	
	os.MkdirAll(tempDir, 0755)
	os.Chdir(tempDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
		DefaultBranch:    "main",
	})

	_, err := pm.GenerateWorktreePath("feature/test", "test-project")
	require.NoError(t, err)

	// Verify .worktrees directory has correct permissions
	worktreesDir := filepath.Join(tempDir, ".worktrees")
	stat, err := os.Stat(worktreesDir)
	require.NoError(t, err)
	
	assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())
}

func TestTruncatePath(t *testing.T) {
	pm := NewPatternManager(nil)

	testCases := []struct {
		name      string
		path      string
		maxLength int
		expected  string
	}{
		{
			name:      "No truncation needed",
			path:      "short-path",
			maxLength: 20,
			expected:  "short-path",
		},
		{
			name:      "Truncate at word boundary",
			path:      "very-long-path-with-many-parts",
			maxLength: 15,
			expected:  "very-long-path",
		},
		{
			name:      "Simple truncation",
			path:      "verylongpathwithnoparts",
			maxLength: 10,
			expected:  "verylong...",
		},
		{
			name:      "Very short limit",
			path:      "path",
			maxLength: 2,
			expected:  "pa",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pm.truncatePath(tc.path, tc.maxLength)
			assert.Equal(t, tc.expected, result)
			assert.LessOrEqual(t, len(result), tc.maxLength)
		})
	}
}

func TestCheckPathAvailable(t *testing.T) {
	pm := NewPatternManager(nil)

	// Test empty path
	err := pm.CheckPathAvailable("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Test existing path
	existingPath := filepath.Join(tempDir, "existing")
	os.MkdirAll(existingPath, 0755)
	err = pm.CheckPathAvailable(existingPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test available path
	availablePath := filepath.Join(tempDir, "available")
	err = pm.CheckPathAvailable(availablePath)
	assert.NoError(t, err)

	// Test path with non-existent parent
	invalidPath := filepath.Join(tempDir, "nonexistent", "child")
	err = pm.CheckPathAvailable(invalidPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestCreateDirectory(t *testing.T) {
	pm := NewPatternManager(nil)

	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-create")
	defer os.RemoveAll(tempDir)

	// Test creating new directory
	newDir := filepath.Join(tempDir, "new-directory")
	err := pm.CreateDirectory(newDir)
	assert.NoError(t, err)

	// Verify directory was created
	stat, err := os.Stat(newDir)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Test creating directory that already exists
	err = pm.CreateDirectory(newDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestValidatePatternResult(t *testing.T) {
	pm := NewPatternManager(nil)

	testCases := []struct {
		name   string
		result string
		valid  bool
	}{
		{
			name:   "Valid relative path",
			result: "project-branch",
			valid:  true,
		},
		{
			name:   "Valid with subdirectory",
			result: "prefix/project-branch",
			valid:  true,
		},
		{
			name:   "Empty result",
			result: "",
			valid:  false,
		},
		{
			name:   "Absolute path",
			result: "/absolute/path",
			valid:  false,
		},
		{
			name:   "Parent directory traversal",
			result: "../parent",
			valid:  false,
		},
		{
			name:   "Reserved name (Windows)",
			result: "CON",
			valid:  false,
		},
		{
			name:   "Reserved name with extension",
			result: "PRN.txt",
			valid:  false,
		},
		{
			name:   "Very long path",
			result: strings.Repeat("a", 300),
			valid:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pm.ValidatePatternResult(tc.result)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestGenerateExamplePaths(t *testing.T) {
	pm := NewPatternManager(nil)

	pattern := "{{.Project}}-{{.Branch}}"
	examples, err := pm.GenerateExamplePaths(pattern)

	require.NoError(t, err)
	assert.Len(t, examples, 3) // Should have 3 examples

	// Check that all examples are different
	uniqueExamples := make(map[string]bool)
	for _, example := range examples {
		assert.NotEmpty(t, example)
		uniqueExamples[example] = true
	}
	assert.Len(t, uniqueExamples, 3)
}

func TestGenerateExamplePaths_InvalidPattern(t *testing.T) {
	pm := NewPatternManager(nil)

	pattern := "{{.invalid}}"
	_, err := pm.GenerateExamplePaths(pattern)

	assert.Error(t, err)
}

func TestGetPatternVariables(t *testing.T) {
	pm := NewPatternManager(nil)

	variables := pm.GetPatternVariables()
	
	assert.NotEmpty(t, variables)
	assert.Contains(t, variables, "{{.Project}}")
	assert.Contains(t, variables, "{{.Branch}}")
	assert.Contains(t, variables, "{{.Worktree}}")
	assert.Contains(t, variables, "{{.Timestamp}}")
	assert.Contains(t, variables, "{{.UserName}}")
	assert.Contains(t, variables, "{{.Prefix}}")
	assert.Contains(t, variables, "{{.Suffix}}")
}

func TestGetPatternFunctions(t *testing.T) {
	pm := NewPatternManager(nil)

	functions := pm.GetPatternFunctions()
	
	assert.NotEmpty(t, functions)
	assert.Contains(t, functions, "lower")
	assert.Contains(t, functions, "upper")
	assert.Contains(t, functions, "title")
	assert.Contains(t, functions, "replace")
	assert.Contains(t, functions, "trim")
	assert.Contains(t, functions, "sanitize")
	assert.Contains(t, functions, "truncate")
}

func TestTemplateFunctions(t *testing.T) {
	// Test sanitizeForFilesystem function
	result := sanitizeForFilesystem("path/with\\unsafe<chars>")
	assert.Equal(t, "path-with-unsafe-chars", result)

	// Test truncateString function
	result = truncateString("verylongstring", 8)
	assert.Equal(t, "veryl...", result)

	result = truncateString("short", 10)
	assert.Equal(t, "short", result)

	result = truncateString("abc", 2)
	assert.Equal(t, "ab", result)
}

func TestGenerateWorktreeID(t *testing.T) {
	pm := NewPatternManager(nil)

	id := pm.generateWorktreeID("feature/auth")
	
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "feature-auth")
	assert.Contains(t, id, "-") // Should contain timestamp separator
}

func TestGetUserName(t *testing.T) {
	pm := NewPatternManager(nil)

	name := pm.getUserName()
	
	assert.NotEmpty(t, name)
	// Should be sanitized (no special characters)
	assert.NotContains(t, name, "/")
	assert.NotContains(t, name, "\\")
	assert.NotContains(t, name, " ")
}