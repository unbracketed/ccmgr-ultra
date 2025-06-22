package ccmgr

import (
	"github.com/bcdekker/ccmgr-ultra/internal/tui"
)

// worktreeManager implements the WorktreeManager interface
type worktreeManager struct {
	integration *tui.Integration
}

// List returns all worktrees
func (wm *worktreeManager) List() ([]WorktreeInfo, error) {
	internal := wm.integration.GetAllWorktrees()
	result := make([]WorktreeInfo, len(internal))
	for i, worktree := range internal {
		result[i] = convertWorktreeInfo(worktree)
	}
	return result, nil
}

// Recent returns recently accessed worktrees
func (wm *worktreeManager) Recent() ([]WorktreeInfo, error) {
	internal := wm.integration.GetRecentWorktrees()
	result := make([]WorktreeInfo, len(internal))
	for i, worktree := range internal {
		result[i] = convertWorktreeInfo(worktree)
	}
	return result, nil
}

// Create creates a new worktree
func (wm *worktreeManager) Create(path, branch string) error {
	// Use the integration layer's CreateWorktree method
	_ = wm.integration.CreateWorktree(path, branch)
	return nil
}

// Open opens a worktree directory
func (wm *worktreeManager) Open(path string) error {
	// Use the integration layer's OpenWorktree method
	_ = wm.integration.OpenWorktree(path)
	return nil
}

// GetClaudeStatus returns Claude status for a worktree
func (wm *worktreeManager) GetClaudeStatus(worktreePath string) ClaudeStatus {
	internal := wm.integration.GetClaudeStatusForWorktree(worktreePath)
	return convertClaudeStatus(internal)
}

// UpdateClaudeStatus updates Claude status for a worktree
func (wm *worktreeManager) UpdateClaudeStatus(worktreePath string, status ClaudeStatus) {
	internal := tui.ClaudeStatus{
		State:      status.State,
		ProcessID:  status.ProcessID,
		LastUpdate: status.LastUpdate,
		SessionID:  status.SessionID,
	}
	wm.integration.UpdateClaudeStatusForWorktree(worktreePath, internal)
}