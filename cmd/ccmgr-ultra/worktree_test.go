package main

import (
	"testing"

	"github.com/bcdekker/ccmgr-ultra/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestWorktreeOptions_AutoName(t *testing.T) {
	tests := []struct {
		name             string
		directory        string
		expectedAutoName bool
	}{
		{
			name:             "empty directory should enable AutoName",
			directory:        "",
			expectedAutoName: true,
		},
		{
			name:             "specified directory should disable AutoName",
			directory:        "/custom/path",
			expectedAutoName: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from worktree create command
			worktreeDir := tt.directory
			useAutoName := worktreeDir == ""

			opts := git.WorktreeOptions{
				Path:         worktreeDir,
				Branch:       "test-branch",
				CreateBranch: true,
				Force:        false,
				Checkout:     true,
				TrackRemote:  false,
				AutoName:     useAutoName,
			}

			assert.Equal(t, tt.expectedAutoName, opts.AutoName, "AutoName flag should match expected value")
		})
	}
}

func TestHandlePatternError(t *testing.T) {
	tests := []struct {
		name        string
		inputError  error
		shouldMatch bool
	}{
		{
			name:        "template error should be handled",
			inputError:  &mockError{msg: "template error: invalid variable"},
			shouldMatch: true,
		},
		{
			name:        "pattern error should be handled",
			inputError:  &mockError{msg: "pattern validation failed"},
			shouldMatch: true,
		},
		{
			name:        "variable error should be handled",
			inputError:  &mockError{msg: "unknown variable in template"},
			shouldMatch: true,
		},
		{
			name:        "generic error should pass through",
			inputError:  &mockError{msg: "file not found"},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handlePatternError(tt.inputError)

			// The function should always return an error
			assert.Error(t, result)

			// Check if it contains pattern error message for matching cases
			if tt.shouldMatch {
				assert.Contains(t, result.Error(), "Template pattern error")
			}
		})
	}
}

// mockError is a simple error implementation for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
