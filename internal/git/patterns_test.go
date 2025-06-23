package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bcdekker/ccmgr-ultra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "my-project")
	require.NoError(t, err)

	// Should be relative to sibling .worktrees directory
	cwd, _ := os.Getwd()
	expectedBase := filepath.Join(cwd, "../.worktrees/my-project")
	expectedPath := filepath.Join(expectedBase, "feature-auth")

	assert.Equal(t, filepath.Clean(expectedPath), path)
}

func TestGenerateWorktreePath_CreatesWorktreesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-worktrees-creation")
	defer os.RemoveAll(tempDir)

	// Create repo directory inside temp dir
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	// Verify .worktrees doesn't exist yet in parent
	worktreesDir := filepath.Join(tempDir, ".worktrees")
	_, err := os.Stat(worktreesDir)
	assert.True(t, os.IsNotExist(err))

	// Generate worktree path
	path, err := pm.GenerateWorktreePath("feature/test", "test-repo")
	require.NoError(t, err)

	// Verify .worktrees directory was created as sibling
	stat, err := os.Stat(worktreesDir)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Verify correct permissions
	assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())

	// Verify path structure - should be sibling to repo
	expectedPath := filepath.Join(tempDir, ".worktrees", "test-repo", "feature-test")
	// Resolve symlinks to handle /private/var vs /var on macOS
	expectedResolved, _ := filepath.EvalSymlinks(expectedPath)
	actualResolved, _ := filepath.EvalSymlinks(path)
	assert.Equal(t, expectedResolved, actualResolved)
	assert.Contains(t, path, ".worktrees")
}

func TestGenerateWorktreePath_UsesWorktreesBasePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-base-path")
	defer os.RemoveAll(tempDir)

	// Create repo directory inside temp dir
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "my-project")
	require.NoError(t, err)

	// Should use sibling .worktrees as base directory
	assert.Contains(t, path, ".worktrees")

	// Verify the path structure
	expectedPath := filepath.Join(tempDir, ".worktrees", "my-project", "feature-auth")
	// Resolve symlinks to handle /private/var vs /var on macOS
	expectedResolved, _ := filepath.EvalSymlinks(expectedPath)
	actualResolved, _ := filepath.EvalSymlinks(path)
	assert.Equal(t, expectedResolved, actualResolved)
}

func TestGenerateWorktreePath_DirectoryCreationError(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-creation-error")
	defer os.RemoveAll(tempDir)

	// Create repo directory
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	// Create a file where .worktrees directory should be (to cause error)
	worktreesPath := filepath.Join(tempDir, ".worktrees")
	file, err := os.Create(worktreesPath)
	require.NoError(t, err)
	file.Close()

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	// Should fail because .worktrees exists as a file, not directory
	_, err = pm.GenerateWorktreePath("feature/test", "test-repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create base directory")
}

func TestGenerateWorktreePath_PreservesPatternFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-pattern-functionality")
	defer os.RemoveAll(tempDir)

	// Create repo directory
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Prefix}}-{{.Branch}}-{{.Suffix}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "my-project")
	require.NoError(t, err)

	// Should preserve all pattern functionality
	assert.Contains(t, path, ".worktrees")
	assert.Contains(t, path, "main-feature-auth")

	// Verify the pattern variables are properly processed
	dirName := filepath.Base(path)
	assert.Equal(t, "main-feature-auth", dirName)
}

func TestGenerateWorktreePath_CorrectPermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-permissions")
	defer os.RemoveAll(tempDir)

	// Create repo directory
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	_, err := pm.GenerateWorktreePath("feature/test", "test-repo")
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

func TestDefaultConfigurationPatternsValid(t *testing.T) {
	cfg := &config.Config{}
	cfg.SetDefaults()

	pm := NewPatternManager(&cfg.Worktree)

	// Test WorktreeConfig default pattern
	err := pm.ValidatePattern(cfg.Worktree.DirectoryPattern)
	assert.NoError(t, err, "Default WorktreeConfig pattern should be valid")

	// Test GitConfig default pattern
	err = pm.ValidatePattern(cfg.Git.DirectoryPattern)
	assert.NoError(t, err, "Default GitConfig pattern should be valid")
}

// Tests for new sibling directory functionality

func TestGenerateWorktreePath_SiblingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-sibling-patterns")
	defer os.RemoveAll(tempDir)

	// Create repo directory inside temp dir
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory for testing
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	tests := []struct {
		name         string
		baseDir      string
		pattern      string
		branch       string
		project      string
		expectedPath string
		shouldError  bool
	}{
		{
			name:         "default sibling pattern",
			baseDir:      "../.worktrees/{{.Project}}",
			pattern:      "{{.Branch}}",
			branch:       "feature/auth",
			project:      "myapp",
			expectedPath: filepath.Join(tempDir, ".worktrees", "myapp", "feature-auth"),
		},
		{
			name:         "absolute base directory",
			baseDir:      filepath.Join(tempDir, "worktrees", "{{.Project}}"),
			pattern:      "{{.Branch}}",
			branch:       "main",
			project:      "testapp",
			expectedPath: filepath.Join(tempDir, "worktrees", "testapp", "main"),
		},
		{
			name:         "simple relative base",
			baseDir:      "../my-worktrees",
			pattern:      "{{.Project}}-{{.Branch}}",
			branch:       "feature/test",
			project:      "app",
			expectedPath: filepath.Join(tempDir, "my-worktrees", "app-feature-test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &config.WorktreeConfig{
				BaseDirectory:    tt.baseDir,
				DirectoryPattern: tt.pattern,
				DefaultBranch:    "main",
			}
			pm := NewPatternManager(config)

			path, err := pm.GenerateWorktreePath(tt.branch, tt.project)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Resolve symlinks to handle /var vs /private/var on macOS
				expectedResolved, _ := filepath.EvalSymlinks(tt.expectedPath)
				actualResolved, _ := filepath.EvalSymlinks(path)
				assert.Equal(t, expectedResolved, actualResolved)
			}
		})
	}
}

func TestValidateBaseDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-validation")
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "myproject")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory for testing
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	tests := []struct {
		name        string
		baseDir     string
		repoPath    string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "sibling directory should pass",
			baseDir:     "../.worktrees/myproject",
			repoPath:    repoDir,
			shouldError: false,
		},
		{
			name:        "absolute path outside repo should pass",
			baseDir:     filepath.Join(tempDir, "external-worktrees"),
			repoPath:    repoDir,
			shouldError: false,
		},
		{
			name:        "directory inside repo should fail",
			baseDir:     ".worktrees",
			repoPath:    repoDir,
			shouldError: true,
			errorMsg:    "cannot be inside repository",
		},
		{
			name:        "empty base directory should fail",
			baseDir:     "",
			repoPath:    repoDir,
			shouldError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "base directory with template outside repo should pass",
			baseDir:     "../.worktrees/{{.Project}}",
			repoPath:    repoDir,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PatternManager{
				config: &config.WorktreeConfig{}, // Initialize with empty config to avoid nil pointer
			}

			err := pm.ValidateBaseDirectory(tt.baseDir, tt.repoPath)

			if tt.shouldError {
				require.Error(t, err, "Expected error for baseDir=%s, repoPath=%s", tt.baseDir, tt.repoPath)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateWorktreePath_CreatesSiblingDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-sibling-creation")
	defer os.RemoveAll(tempDir)

	// Create project directory inside temp dir
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	// Verify worktrees directory doesn't exist yet
	worktreesDir := filepath.Join(tempDir, ".worktrees")
	_, err := os.Stat(worktreesDir)
	assert.True(t, os.IsNotExist(err))

	// Generate worktree path
	path, err := pm.GenerateWorktreePath("feature/test", "test-repo")
	require.NoError(t, err)

	// Verify .worktrees directory was created as sibling
	stat, err := os.Stat(worktreesDir)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Verify project subdirectory was created
	projectDir := filepath.Join(worktreesDir, "test-repo")
	stat, err = os.Stat(projectDir)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Verify path structure
	expectedPath := filepath.Join(tempDir, ".worktrees", "test-repo", "feature-test")
	// Resolve symlinks to handle /private/var vs /var on macOS
	expectedResolved, _ := filepath.EvalSymlinks(expectedPath)
	actualResolved, _ := filepath.EvalSymlinks(path)
	assert.Equal(t, expectedResolved, actualResolved)

	// Verify path is outside the repository
	absPath, _ := filepath.Abs(path)
	absRepoPath, _ := filepath.Abs(repoDir)
	assert.False(t, strings.HasPrefix(absPath, absRepoPath), "Worktree path should not be inside repository")
}

func TestGenerateWorktreePath_AbsoluteBasePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-absolute-base")
	defer os.RemoveAll(tempDir)

	absoluteBase := filepath.Join(tempDir, "worktrees")

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    absoluteBase,
		DirectoryPattern: "{{.Project}}-{{.Branch}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/auth", "myproject")
	require.NoError(t, err)

	// Verify absolute base directory was created
	stat, err := os.Stat(absoluteBase)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Verify correct path structure
	expectedPath := filepath.Join(absoluteBase, "myproject-feature-auth")
	assert.Equal(t, expectedPath, path)
}

func TestGenerateWorktreePath_TemplateInBaseDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-template-base")
	defer os.RemoveAll(tempDir)

	// Create repo directory inside temp dir
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.Project}}/{{.UserName}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	path, err := pm.GenerateWorktreePath("feature/test", "myproject")
	require.NoError(t, err)

	// Verify path contains resolved template variables
	assert.Contains(t, path, ".worktrees")
	assert.Contains(t, path, "myproject")
	assert.Contains(t, path, "feature-test")

	// Verify the path structure contains the expected components
	pathParts := strings.Split(path, string(filepath.Separator))
	assert.Contains(t, pathParts, ".worktrees")
	assert.Contains(t, pathParts, "myproject")
	assert.Contains(t, pathParts, "feature-test")
}

func TestValidateBaseDirectory_ResolveTemplates(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := filepath.Join(os.TempDir(), "ccmgr-test-template-validation")
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "myproject")
	os.MkdirAll(repoDir, 0755)

	// Change to repo directory for testing
	originalCwd, _ := os.Getwd()
	defer os.Chdir(originalCwd)
	os.Chdir(repoDir)

	pm := &PatternManager{
		config: &config.WorktreeConfig{}, // Initialize with empty config to avoid nil pointer
	}

	// Base directory with templates outside repo should pass validation
	// (templates themselves don't matter for validation, only the resolved path)
	err := pm.ValidateBaseDirectory("../worktrees/{{.Project}}", repoDir)
	assert.NoError(t, err)

	// Base directory that would resolve to inside repo should fail
	err = pm.ValidateBaseDirectory(".worktrees/{{.Project}}", repoDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be inside repository")
}

func TestGenerateWorktreePath_ErrorHandling(t *testing.T) {
	// Test invalid base directory template
	pm := NewPatternManager(&config.WorktreeConfig{
		BaseDirectory:    "../.worktrees/{{.InvalidVariable}}",
		DirectoryPattern: "{{.Branch}}",
		DefaultBranch:    "main",
	})

	_, err := pm.GenerateWorktreePath("feature/test", "myproject")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve base directory pattern")
}
