package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWorktreeTableFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected []string // strings that should be present in output
		wantErr  bool
	}{
		{
			name: "valid worktree data",
			data: struct {
				Worktrees []struct {
					Name         string
					Branch       string
					Head         string
					IsClean      bool
					TmuxSession  string
					LastAccessed time.Time
				}
				Total int
			}{
				Worktrees: []struct {
					Name         string
					Branch       string
					Head         string
					IsClean      bool
					TmuxSession  string
					LastAccessed time.Time
				}{
					{
						Name:         "test-worktree",
						Branch:       "feature/test",
						Head:         "abc1234567890",
						IsClean:      true,
						TmuxSession:  "ccmgr-test",
						LastAccessed: time.Now().Add(-2 * time.Hour),
					},
				},
				Total: 1,
			},
			expected: []string{
				"Worktrees",
				"test-worktree",
				"feature/test",
				"abc12345", // head should be truncated
				"✓ Clean",
				"ccmgr-test",
				"Total worktrees: 1",
			},
			wantErr: false,
		},
		{
			name: "dirty worktree",
			data: struct {
				Worktrees []struct {
					Name         string
					Branch       string
					Head         string
					IsClean      bool
					TmuxSession  string
					LastAccessed time.Time
				}
				Total int
			}{
				Worktrees: []struct {
					Name         string
					Branch       string
					Head         string
					IsClean      bool
					TmuxSession  string
					LastAccessed time.Time
				}{
					{
						Name:         "dirty-worktree",
						Branch:       "feature/dirty",
						Head:         "def4567890123",
						IsClean:      false,
						TmuxSession:  "",
						LastAccessed: time.Now().Add(-4 * time.Hour),
					},
				},
				Total: 1,
			},
			expected: []string{
				"dirty-worktree",
				"feature/dirty",
				"def45678",
				"⚠ Dirty",
			},
			wantErr: false,
		},
		{
			name: "empty worktrees",
			data: struct {
				Worktrees []struct {
					Name         string
					Branch       string
					Head         string
					IsClean      bool
					TmuxSession  string
					LastAccessed time.Time
				}
				Total int
			}{
				Worktrees: []struct {
					Name         string
					Branch       string
					Head         string
					IsClean      bool
					TmuxSession  string
					LastAccessed time.Time
				}{},
				Total: 0,
			},
			expected: []string{
				"No worktrees found",
			},
			wantErr: false,
		},
		{
			name:     "nil data",
			data:     nil,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid data type",
			data:     "invalid",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewWorktreeTableFormatter(&buf)

			err := formatter.Format(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("WorktreeTableFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("WorktreeTableFormatter.Format() output does not contain expected string %q\nOutput:\n%s", expected, output)
				}
			}
		})
	}
}

func TestWorktreeTableFormatter_FormatWorktreeStatusFromFields(t *testing.T) {
	tests := []struct {
		name     string
		isClean  bool
		expected string
	}{
		{
			name:     "clean worktree",
			isClean:  true,
			expected: "✓ Clean",
		},
		{
			name:     "dirty worktree",
			isClean:  false,
			expected: "⚠ Dirty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWorktreeStatusFromFields(tt.isClean)
			if result != tt.expected {
				t.Errorf("formatWorktreeStatusFromFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}
