package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentBuilder_WithContext(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{
		WorktreePath:   "/tmp/test-worktree",
		WorktreeBranch: "feature-branch",
		ProjectName:    "test-project",
		SessionID:      "session-123",
		SessionType:    "new",
		OldState:       "busy",
		NewState:       "idle",
		CustomVars: map[string]string{
			"CUSTOM_VAR": "custom-value",
		},
	}

	env := builder.WithContext(ctx).BuildMap()

	assert.Equal(t, "/tmp/test-worktree", env["CCMGR_WORKTREE_PATH"])
	assert.Equal(t, "feature-branch", env["CCMGR_WORKTREE_BRANCH"])
	assert.Equal(t, "test-project", env["CCMGR_PROJECT_NAME"])
	assert.Equal(t, "session-123", env["CCMGR_SESSION_ID"])
	assert.Equal(t, "new", env["CCMGR_SESSION_TYPE"])
	assert.Equal(t, "busy", env["CCMGR_OLD_STATE"])
	assert.Equal(t, "idle", env["CCMGR_NEW_STATE"])
	assert.Equal(t, "custom-value", env["CUSTOM_VAR"])
}

func TestEnvironmentBuilder_WithStatusHookVars(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{
		WorktreePath:   "/tmp/test-worktree",
		WorktreeBranch: "main",
		NewState:       "idle",
		SessionID:      "session-456",
	}

	env := builder.WithStatusHookVars(HookTypeStatusIdle, ctx).BuildMap()

	// Check new variables
	assert.Equal(t, "/tmp/test-worktree", env["CCMGR_WORKTREE_PATH"])
	assert.Equal(t, "main", env["CCMGR_WORKTREE_BRANCH"])
	assert.Equal(t, "idle", env["CCMGR_NEW_STATE"])
	assert.Equal(t, "session-456", env["CCMGR_SESSION_ID"])

	// Check legacy variables for backward compatibility
	assert.Equal(t, "/tmp/test-worktree", env["CCMANAGER_WORKTREE"])
	assert.Equal(t, "main", env["CCMANAGER_WORKTREE_BRANCH"])
	assert.Equal(t, "idle", env["CCMANAGER_NEW_STATE"])
	assert.Equal(t, "session-456", env["CCMANAGER_SESSION_ID"])

	// Check timestamp is set
	assert.NotEmpty(t, env["CCMANAGER_TIMESTAMP"])
}

func TestEnvironmentBuilder_WithWorktreeCreationVars(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{
		WorktreePath:   "/tmp/new-worktree",
		WorktreeBranch: "feature-123",
		ProjectName:    "my-project",
		CustomVars: map[string]string{
			"CCMGR_PARENT_PATH": "/tmp/parent-repo",
		},
	}

	env := builder.WithWorktreeCreationVars(ctx).BuildMap()

	assert.Equal(t, "/tmp/new-worktree", env["CCMGR_WORKTREE_PATH"])
	assert.Equal(t, "feature-123", env["CCMGR_WORKTREE_BRANCH"])
	assert.Equal(t, "my-project", env["CCMGR_PROJECT_NAME"])
	assert.Equal(t, "new", env["CCMGR_WORKTREE_TYPE"])
	assert.Equal(t, "/tmp/parent-repo", env["CCMGR_PARENT_PATH"])
}

func TestEnvironmentBuilder_WithWorktreeActivationVars(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{
		WorktreePath:   "/tmp/existing-worktree",
		WorktreeBranch: "develop",
		ProjectName:    "web-app",
		SessionID:      "session-789",
		SessionType:    "resume",
		CustomVars: map[string]string{
			"CCMGR_PREVIOUS_STATE": "paused",
		},
	}

	env := builder.WithWorktreeActivationVars(ctx).BuildMap()

	assert.Equal(t, "/tmp/existing-worktree", env["CCMGR_WORKTREE_PATH"])
	assert.Equal(t, "develop", env["CCMGR_WORKTREE_BRANCH"])
	assert.Equal(t, "web-app", env["CCMGR_PROJECT_NAME"])
	assert.Equal(t, "session-789", env["CCMGR_SESSION_ID"])
	assert.Equal(t, "resume", env["CCMGR_SESSION_TYPE"])
	assert.Equal(t, "paused", env["CCMGR_PREVIOUS_STATE"])
}

func TestEnvironmentBuilder_WithCustomVar(t *testing.T) {
	builder := NewEnvironmentBuilder()

	env := builder.
		WithCustomVar("TEST_VAR1", "value1").
		WithCustomVar("TEST_VAR2", "value2").
		BuildMap()

	assert.Equal(t, "value1", env["TEST_VAR1"])
	assert.Equal(t, "value2", env["TEST_VAR2"])
}

func TestEnvironmentBuilder_Build(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{
		WorktreePath: "/tmp/test",
		ProjectName:  "test-proj",
	}

	envSlice := builder.WithContext(ctx).Build()

	// Should include system environment plus our variables
	assert.Greater(t, len(envSlice), 2)

	// Check that our variables are present
	found := false
	for _, envVar := range envSlice {
		if envVar == "CCMGR_WORKTREE_PATH=/tmp/test" {
			found = true
			break
		}
	}
	assert.True(t, found, "Custom environment variable not found in slice")
}

func TestEnvironmentBuilder_EmptyContext(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{}
	env := builder.WithContext(ctx).BuildMap()

	// Should have timestamp but no other custom variables
	assert.NotEmpty(t, env["CCMGR_TIMESTAMP"])
	assert.Empty(t, env["CCMGR_WORKTREE_PATH"])
	assert.Empty(t, env["CCMGR_PROJECT_NAME"])
}

func TestEnvironmentBuilder_DefaultSessionType(t *testing.T) {
	builder := NewEnvironmentBuilder()

	ctx := HookContext{
		WorktreePath: "/tmp/test",
		// SessionType is empty
	}

	env := builder.WithWorktreeActivationVars(ctx).BuildMap()

	// Should default to "new" when session type is not specified
	assert.Equal(t, "new", env["CCMGR_SESSION_TYPE"])
}

func TestValidateEnvironmentKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{"valid key", "VALID_KEY_123", false},
		{"empty key", "", true},
		{"key with equals", "KEY=VALUE", true},
		{"key with space", "KEY WITH SPACE", true},
		{"key with special chars", "KEY@#$", true},
		{"underscore key", "_UNDERSCORE_KEY_", false},
		{"numeric start", "123KEY", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvironmentKey(tt.key)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeEnvironmentValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal value", "normal value"},
		{"value with null\x00byte", "value with nullbyte"},
		{"value with\nnewline", "value with\\nnewline"},
		{"value\x00with\nmultiple\x00issues", "valuewith\\nmultipleissues"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeEnvironmentValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeEnvironmentMaps(t *testing.T) {
	map1 := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	map2 := map[string]string{
		"KEY2": "new_value2", // Override
		"KEY3": "value3",     // New
	}

	map3 := map[string]string{
		"KEY4": "value4",
	}

	result := mergeEnvironmentMaps(map1, map2, map3)

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "new_value2", // Should be overridden
		"KEY3": "value3",
		"KEY4": "value4",
	}

	assert.Equal(t, expected, result)
}
