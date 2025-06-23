package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_NewManager(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	assert.NotNil(t, manager)
	assert.True(t, manager.IsEnabled())
	assert.NotNil(t, manager.GetStatusIntegrator())
	assert.NotNil(t, manager.GetWorktreeIntegrator())
	assert.NotNil(t, manager.GetExecutor())
}

func TestManager_EnableDisable(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Initially enabled
	assert.True(t, manager.IsEnabled())
	assert.True(t, manager.GetStatusIntegrator().IsEnabled())
	assert.True(t, manager.GetWorktreeIntegrator().IsEnabled())

	// Disable
	manager.Disable()
	assert.False(t, manager.IsEnabled())
	assert.False(t, manager.GetStatusIntegrator().IsEnabled())
	assert.False(t, manager.GetWorktreeIntegrator().IsEnabled())

	// Re-enable
	manager.Enable()
	assert.True(t, manager.IsEnabled())
	assert.True(t, manager.GetStatusIntegrator().IsEnabled())
	assert.True(t, manager.GetWorktreeIntegrator().IsEnabled())
}

func TestManager_UpdateConfig(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Update config
	newCfg := createTestConfig()
	newCfg.StatusHooks.Enabled = false
	newCfg.WorktreeHooks.Enabled = false

	manager.UpdateConfig(newCfg)

	// Verify config was updated
	assert.Equal(t, newCfg, manager.GetConfig())
}

func TestManager_OnClaudeStateChange(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Create test script
	testScript := createTestScript(t, `#!/bin/bash
echo "State changed from $CCMGR_OLD_STATE to $CCMGR_NEW_STATE"
exit 0`)

	cfg.StatusHooks.IdleHook.Script = testScript
	cfg.StatusHooks.IdleHook.Enabled = true
	cfg.StatusHooks.Enabled = true

	manager.UpdateConfig(cfg)

	ctx := HookContext{
		WorktreePath:   "/tmp/test",
		WorktreeBranch: "main",
		SessionID:      "session-123",
	}

	// This should not panic or return error
	manager.OnClaudeStateChange("busy", "idle", ctx)
}

func TestManager_OnWorktreeCreated(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Create test script
	testScript := createTestScript(t, `#!/bin/bash
echo "Worktree created at $CCMGR_WORKTREE_PATH"
exit 0`)

	cfg.WorktreeHooks.CreationHook.Script = testScript
	cfg.WorktreeHooks.CreationHook.Enabled = true
	cfg.WorktreeHooks.Enabled = true

	manager.UpdateConfig(cfg)

	err := manager.OnWorktreeCreated("/tmp/new-worktree", "feature-branch", "/tmp/parent", "test-project")
	assert.NoError(t, err)
}

func TestManager_OnSessionCreated(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Create test script
	testScript := createTestScript(t, `#!/bin/bash
echo "Session created for $CCMGR_PROJECT_NAME"
exit 0`)

	cfg.WorktreeHooks.ActivationHook.Script = testScript
	cfg.WorktreeHooks.ActivationHook.Enabled = true
	cfg.WorktreeHooks.Enabled = true

	manager.UpdateConfig(cfg)

	sessionInfo := SessionInfo{
		SessionID:   "session-456",
		WorkingDir:  "/tmp/worktree",
		Branch:      "main",
		ProjectName: "test-project",
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	err := manager.OnSessionCreated(sessionInfo)
	assert.NoError(t, err)
}

func TestManager_OnSessionContinued(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	testScript := createTestScript(t, `#!/bin/bash
if [ "$CCMGR_SESSION_TYPE" = "continue" ]; then
    echo "Session continued"
    exit 0
else
    echo "Unexpected session type: $CCMGR_SESSION_TYPE"
    exit 1
fi`)

	cfg.WorktreeHooks.ActivationHook.Script = testScript
	cfg.WorktreeHooks.ActivationHook.Enabled = true
	cfg.WorktreeHooks.Enabled = true

	manager.UpdateConfig(cfg)

	sessionInfo := SessionInfo{
		SessionID:   "session-789",
		WorkingDir:  "/tmp/worktree",
		Branch:      "develop",
		ProjectName: "web-app",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		LastActive:  time.Now().Add(-30 * time.Minute),
	}

	err := manager.OnSessionContinued(sessionInfo)
	assert.NoError(t, err)
}

func TestManager_OnSessionResumed(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	testScript := createTestScript(t, `#!/bin/bash
if [ "$CCMGR_SESSION_TYPE" = "resume" ] && [ "$CCMGR_PREVIOUS_STATE" = "paused" ]; then
    echo "Session resumed from paused state"
    exit 0
else
    echo "Unexpected session type or previous state"
    exit 1
fi`)

	cfg.WorktreeHooks.ActivationHook.Script = testScript
	cfg.WorktreeHooks.ActivationHook.Enabled = true
	cfg.WorktreeHooks.Enabled = true

	manager.UpdateConfig(cfg)

	sessionInfo := SessionInfo{
		SessionID:   "session-101",
		WorkingDir:  "/tmp/worktree",
		Branch:      "hotfix",
		ProjectName: "urgent-fix",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		LastActive:  time.Now().Add(-1 * time.Hour),
	}

	err := manager.OnSessionResumed(sessionInfo, "paused")
	assert.NoError(t, err)
}

func TestManager_ExecuteHookDisabled(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Disable manager
	manager.Disable()

	ctx := context.Background()
	hookCtx := HookContext{}

	// Should return nil without executing anything
	err := manager.ExecuteHook(ctx, HookTypeStatusIdle, hookCtx)
	assert.NoError(t, err)
}

func TestManager_ExecuteHookAsync(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	testScript := createTestScript(t, `#!/bin/bash
sleep 0.1
echo "Async hook completed"
exit 0`)

	cfg.StatusHooks.BusyHook.Script = testScript
	cfg.StatusHooks.BusyHook.Enabled = true
	cfg.StatusHooks.Enabled = true

	manager.UpdateConfig(cfg)

	hookCtx := HookContext{
		WorktreePath: "/tmp/test",
	}

	errChan := manager.ExecuteHookAsync(HookTypeStatusBusy, hookCtx)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Async execution timed out")
	}
}

func TestManager_ExecuteHookAsyncDisabled(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Disable manager
	manager.Disable()

	hookCtx := HookContext{}
	errChan := manager.ExecuteHookAsync(HookTypeStatusIdle, hookCtx)

	// Should immediately return closed channel
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Channel should be closed immediately when disabled")
	}
}

func TestManager_Start(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should not panic or block
	manager.Start(ctx)

	// Wait for context to be done
	<-ctx.Done()
}

func TestManager_GetStats(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	stats := manager.GetStats()

	// Should return empty stats for now
	assert.Equal(t, int64(0), stats.TotalExecutions)
	assert.Equal(t, int64(0), stats.SuccessCount)
	assert.Equal(t, int64(0), stats.FailureCount)
	assert.Nil(t, stats.LastExecution)
}
