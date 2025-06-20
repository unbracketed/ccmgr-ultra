package tmux

import (
	"strings"
	"testing"
)

func TestGenerateSessionName(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		worktree string
		branch   string
		expected string
	}{
		{
			name:     "basic names",
			project:  "myproject",
			worktree: "main",
			branch:   "feature",
			expected: "ccmgr-myproject-main-feature",
		},
		{
			name:     "names with special chars",
			project:  "my-project@v1",
			worktree: "main/dev",
			branch:   "feature/auth",
			expected: "ccmgr-my_project_v1-main_dev-feature_auth",
		},
		{
			name:     "empty values",
			project:  "",
			worktree: "",
			branch:   "",
			expected: "ccmgr-unnamed-unnamed-unnamed",
		},
		{
			name:     "long names",
			project:  "verylongprojectname",
			worktree: "verylongworktreename",
			branch:   "verylongbranchname",
			expected: "ccmgr-verylongprojectname-verylongworktreename~-verylongbranchname~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSessionName(tt.project, tt.worktree, tt.branch)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
			
			if len(result) > maxNameLength {
				t.Errorf("Generated name too long: %d > %d", len(result), maxNameLength)
			}
			
			if !ValidateSessionName(result) {
				t.Errorf("Generated name is not valid: %s", result)
			}
		})
	}
}

func TestParseSessionName(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		expectErr   bool
		project     string
		worktree    string
		branch      string
	}{
		{
			name:        "valid basic name",
			sessionName: "ccmgr-myproject-main-feature",
			expectErr:   false,
			project:     "myproject",
			worktree:    "main",
			branch:      "feature",
		},
		{
			name:        "valid name with sanitized chars",
			sessionName: "ccmgr-my_project-main_dev-feature_auth",
			expectErr:   false,
			project:     "my_project",
			worktree:    "main_dev",
			branch:      "feature_auth",
		},
		{
			name:        "invalid prefix",
			sessionName: "wrong-myproject-main-feature",
			expectErr:   true,
		},
		{
			name:        "missing components",
			sessionName: "ccmgr-myproject-main",
			expectErr:   true,
		},
		{
			name:        "empty string",
			sessionName: "",
			expectErr:   true,
		},
		{
			name:        "complex branch name",
			sessionName: "ccmgr-proj-wt-feature_long_branch_name",
			expectErr:   false,
			project:     "proj",
			worktree:    "wt",
			branch:      "feature_long_branch_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, worktree, branch, err := ParseSessionName(tt.sessionName)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if project != tt.project {
				t.Errorf("Expected project %s, got %s", tt.project, project)
			}
			
			if worktree != tt.worktree {
				t.Errorf("Expected worktree %s, got %s", tt.worktree, worktree)
			}
			
			if branch != tt.branch {
				t.Errorf("Expected branch %s, got %s", tt.branch, branch)
			}
		})
	}
}

func TestValidateSessionName(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		expected    bool
	}{
		{
			name:        "valid name",
			sessionName: "ccmgr-project-worktree-branch",
			expected:    true,
		},
		{
			name:        "wrong prefix",
			sessionName: "wrong-project-worktree-branch",
			expected:    false,
		},
		{
			name:        "too long",
			sessionName: "ccmgr-" + strings.Repeat("a", maxNameLength),
			expected:    false,
		},
		{
			name:        "missing components",
			sessionName: "ccmgr-project-worktree",
			expected:    false,
		},
		{
			name:        "empty",
			sessionName: "",
			expected:    false,
		},
		{
			name:        "just prefix",
			sessionName: "ccmgr-",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSessionName(tt.sessionName)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t for name: %s", tt.expected, result, tt.sessionName)
			}
		})
	}
}

func TestSanitizeNameComponent(t *testing.T) {
	tests := []struct {
		name      string
		component string
		expected  string
	}{
		{
			name:      "basic name",
			component: "project",
			expected:  "project",
		},
		{
			name:      "with special chars",
			component: "my-project@v1.0",
			expected:  "my_project_v1_0",
		},
		{
			name:      "empty string",
			component: "",
			expected:  "unnamed",
		},
		{
			name:      "only special chars",
			component: "@#$%",
			expected:  "unnamed",
		},
		{
			name:      "leading/trailing underscores",
			component: "_project_",
			expected:  "project",
		},
		{
			name:      "too long",
			component: "verylongcomponentnamethatexceedsmaxlength",
			expected:  "verylongcomponentnam",
		},
		{
			name:      "spaces",
			component: "my project name",
			expected:  "my_project_name",
		},
		{
			name:      "mixed case",
			component: "MyProject",
			expected:  "MyProject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeNameComponent(tt.component)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
			
			if len(result) > 20 {
				t.Errorf("Sanitized component too long: %d > 20", len(result))
			}
		})
	}
}

func TestTruncateSessionName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "ccmgr-short-name-branch",
			maxLen:   50,
			expected: "ccmgr-short-name-branch",
		},
		{
			name:     "truncation needed",
			input:    "ccmgr-verylongproject-verylongworktree-verylongbranch",
			maxLen:   30,
			expected: "ccmgr-verylon~-verylon~-veryl~",
		},
		{
			name:     "extreme truncation",
			input:    "ccmgr-project-worktree-branch",
			maxLen:   15,
			expected: "ccmgr-pro~-w~-b~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateSessionName(tt.input, tt.maxLen)
			if len(result) > tt.maxLen {
				t.Errorf("Truncated name too long: %d > %d", len(result), tt.maxLen)
			}
			
			if !strings.HasPrefix(result, "ccmgr-") {
				t.Errorf("Truncated name should still have prefix: %s", result)
			}
		})
	}
}

func TestRoundTripParsing(t *testing.T) {
	tests := []struct {
		project  string
		worktree string
		branch   string
	}{
		{"myproject", "main", "feature"},
		{"test-proj", "dev-branch", "fix/bug-123"},
		{"name_with_underscores", "wt", "long_branch_name"},
	}

	for _, tt := range tests {
		t.Run("round trip", func(t *testing.T) {
			sessionName := GenerateSessionName(tt.project, tt.worktree, tt.branch)
			
			parsedProject, parsedWorktree, parsedBranch, err := ParseSessionName(sessionName)
			if err != nil {
				t.Errorf("Failed to parse generated session name: %v", err)
				return
			}
			
			if SanitizeNameComponent(tt.project) != parsedProject {
				t.Errorf("Project mismatch: expected %s, got %s", SanitizeNameComponent(tt.project), parsedProject)
			}
			
			if SanitizeNameComponent(tt.worktree) != parsedWorktree {
				t.Errorf("Worktree mismatch: expected %s, got %s", SanitizeNameComponent(tt.worktree), parsedWorktree)
			}
			
			if SanitizeNameComponent(tt.branch) != parsedBranch {
				t.Errorf("Branch mismatch: expected %s, got %s", SanitizeNameComponent(tt.branch), parsedBranch)
			}
		})
	}
}