package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unbracketed/ccmgr-ultra/internal/config"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	// Create a test script
	testScript := createTestScript(t, `#!/bin/bash
echo "Hook executed successfully"
exit 0`)

	// Update config to use test script
	cfg.StatusHooks.IdleHook.Script = testScript
	cfg.StatusHooks.IdleHook.Enabled = true
	cfg.StatusHooks.IdleHook.Timeout = 10

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hookCtx := HookContext{
		WorktreePath:   "/tmp/test-worktree",
		WorktreeBranch: "test-branch",
		ProjectName:    "test-project",
		SessionID:      "test-session",
	}

	err := executor.Execute(ctx, HookTypeStatusIdle, hookCtx)
	assert.NoError(t, err)
}

func TestDefaultExecutor_ExecuteTimeout(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	// Create a script that sleeps longer than timeout
	testScript := createTestScript(t, `#!/bin/bash
sleep 5
echo "This should not execute"`)

	cfg.StatusHooks.IdleHook.Script = testScript
	cfg.StatusHooks.IdleHook.Enabled = true
	cfg.StatusHooks.IdleHook.Timeout = 1 // 1 second timeout

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hookCtx := HookContext{
		WorktreePath: "/tmp/test-worktree",
	}

	err := executor.Execute(ctx, HookTypeStatusIdle, hookCtx)
	assert.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
}

func TestDefaultExecutor_ExecuteScriptNotFound(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	cfg.StatusHooks.IdleHook.Script = "/nonexistent/script.sh"
	cfg.StatusHooks.IdleHook.Enabled = true

	ctx := context.Background()
	hookCtx := HookContext{}

	err := executor.Execute(ctx, HookTypeStatusIdle, hookCtx)
	assert.Error(t, err)

	var notFoundErr *ScriptNotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestDefaultExecutor_ExecuteScriptError(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	// Create a script that exits with error
	testScript := createTestScript(t, `#!/bin/bash
echo "Error occurred" >&2
exit 1`)

	cfg.StatusHooks.IdleHook.Script = testScript
	cfg.StatusHooks.IdleHook.Enabled = true

	ctx := context.Background()
	hookCtx := HookContext{}

	err := executor.Execute(ctx, HookTypeStatusIdle, hookCtx)
	assert.Error(t, err)

	var execErr *ScriptExecutionError
	assert.ErrorAs(t, err, &execErr)
	assert.Equal(t, 1, execErr.ExitCode)
}

func TestDefaultExecutor_ExecuteAsync(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	testScript := createTestScript(t, `#!/bin/bash
echo "Async hook executed"
exit 0`)

	cfg.StatusHooks.BusyHook.Script = testScript
	cfg.StatusHooks.BusyHook.Enabled = true

	hookCtx := HookContext{
		WorktreePath: "/tmp/test-worktree",
	}

	errChan := executor.ExecuteAsync(HookTypeStatusBusy, hookCtx)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Async execution timed out")
	}
}

func TestDefaultExecutor_ExecuteStatusHook(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	testScript := createTestScript(t, `#!/bin/bash
if [ "$CCMGR_NEW_STATE" = "idle" ]; then
    echo "Status changed to idle"
    exit 0
else
    echo "Unexpected state: $CCMGR_NEW_STATE"
    exit 1
fi`)

	cfg.StatusHooks.IdleHook.Script = testScript
	cfg.StatusHooks.IdleHook.Enabled = true
	cfg.StatusHooks.Enabled = true

	hookCtx := HookContext{
		NewState: "idle",
		OldState: "busy",
	}

	err := executor.ExecuteStatusHook(HookTypeStatusIdle, hookCtx)
	assert.NoError(t, err)
}

func TestDefaultExecutor_ExecuteWorktreeCreationHook(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	testScript := createTestScript(t, `#!/bin/bash
if [ "$CCMGR_WORKTREE_TYPE" = "new" ]; then
    echo "Worktree creation hook executed"
    exit 0
else
    echo "Expected CCMGR_WORKTREE_TYPE=new, got: $CCMGR_WORKTREE_TYPE"
    exit 1
fi`)

	cfg.WorktreeHooks.CreationHook.Script = testScript
	cfg.WorktreeHooks.CreationHook.Enabled = true
	cfg.WorktreeHooks.Enabled = true

	hookCtx := HookContext{
		WorktreePath:   "/tmp/test-worktree",
		WorktreeBranch: "feature-branch",
		ProjectName:    "test-project",
		CustomVars: map[string]string{
			"CCMGR_PARENT_PATH": "/tmp/parent",
		},
	}

	err := executor.ExecuteWorktreeCreationHook(hookCtx)
	assert.NoError(t, err)
}

func TestDefaultExecutor_ExecuteWorktreeActivationHook(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	testScript := createTestScript(t, `#!/bin/bash
if [ "$CCMGR_SESSION_TYPE" = "new" ]; then
    echo "Worktree activation hook executed"
    exit 0
else
    echo "Expected CCMGR_SESSION_TYPE=new, got: $CCMGR_SESSION_TYPE"
    exit 1
fi`)

	cfg.WorktreeHooks.ActivationHook.Script = testScript
	cfg.WorktreeHooks.ActivationHook.Enabled = true
	cfg.WorktreeHooks.Enabled = true

	hookCtx := HookContext{
		WorktreePath:   "/tmp/test-worktree",
		WorktreeBranch: "feature-branch",
		ProjectName:    "test-project",
		SessionID:      "session-123",
		SessionType:    "new",
	}

	err := executor.ExecuteWorktreeActivationHook(hookCtx)
	assert.NoError(t, err)
}

func TestDefaultExecutor_DisabledHooks(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	// Disable all hooks
	cfg.StatusHooks.Enabled = false
	cfg.WorktreeHooks.Enabled = false

	hookCtx := HookContext{}

	// All hook executions should return nil without error
	err := executor.ExecuteStatusHook(HookTypeStatusIdle, hookCtx)
	assert.NoError(t, err)

	err = executor.ExecuteWorktreeCreationHook(hookCtx)
	assert.NoError(t, err)

	err = executor.ExecuteWorktreeActivationHook(hookCtx)
	assert.NoError(t, err)
}

func TestDefaultExecutor_EnvironmentVariables(t *testing.T) {
	cfg := createTestConfig()
	executor := NewDefaultExecutor(cfg)

	testScript := createTestScript(t, `#!/bin/bash
echo "CCMGR_WORKTREE_PATH: $CCMGR_WORKTREE_PATH"
echo "CCMGR_WORKTREE_BRANCH: $CCMGR_WORKTREE_BRANCH"
echo "CCMGR_PROJECT_NAME: $CCMGR_PROJECT_NAME"
echo "CCMGR_SESSION_ID: $CCMGR_SESSION_ID"
echo "CCMGR_TIMESTAMP: $CCMGR_TIMESTAMP"

# Check that all required variables are set
if [ -z "$CCMGR_WORKTREE_PATH" ] || [ -z "$CCMGR_WORKTREE_BRANCH" ]; then
    echo "Missing required environment variables"
    exit 1
fi

exit 0`)

	cfg.StatusHooks.IdleHook.Script = testScript
	cfg.StatusHooks.IdleHook.Enabled = true
	cfg.StatusHooks.Enabled = true

	ctx := context.Background()
	hookCtx := HookContext{
		WorktreePath:   "/tmp/test-worktree",
		WorktreeBranch: "test-branch",
		ProjectName:    "test-project",
		SessionID:      "session-123",
	}

	err := executor.Execute(ctx, HookTypeStatusIdle, hookCtx)
	assert.NoError(t, err)
}

// Helper functions

func createTestConfig() *config.Config {
	cfg := &config.Config{
		Version: "1.0.0",
		StatusHooks: config.StatusHooksConfig{
			Enabled: true,
			IdleHook: config.HookConfig{
				Enabled: true,
				Timeout: 30,
				Async:   false,
			},
			BusyHook: config.HookConfig{
				Enabled: true,
				Timeout: 30,
				Async:   false,
			},
			WaitingHook: config.HookConfig{
				Enabled: true,
				Timeout: 30,
				Async:   false,
			},
		},
		WorktreeHooks: config.WorktreeHooksConfig{
			Enabled: true,
			CreationHook: config.HookConfig{
				Enabled: true,
				Timeout: 300,
				Async:   false,
			},
			ActivationHook: config.HookConfig{
				Enabled: true,
				Timeout: 60,
				Async:   false,
			},
		},
	}
	return cfg
}

func createTestScript(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-hook.sh")

	err := os.WriteFile(scriptPath, []byte(content), 0755)
	require.NoError(t, err)

	return scriptPath
}
