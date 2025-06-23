package tui

import (
	"fmt"

	"github.com/bcdekker/ccmgr-ultra/internal/tui/modals"
	"github.com/bcdekker/ccmgr-ultra/internal/tui/workflows"
)

// WorkflowFactory creates and manages workflow wizards
type WorkflowFactory struct {
	integration workflows.Integration
	theme       modals.Theme
}

// NewWorkflowFactory creates a new workflow factory
func NewWorkflowFactory(integration workflows.Integration, theme modals.Theme) *WorkflowFactory {
	return &WorkflowFactory{
		integration: integration,
		theme:       theme,
	}
}

// CreateSessionWizard creates a general session creation wizard
func (f *WorkflowFactory) CreateSessionWizard() *workflows.SessionCreationWizard {
	return workflows.NewSessionCreationWizard(f.integration, f.theme)
}

// CreateWorktreeWizard creates a worktree creation wizard
func (f *WorkflowFactory) CreateWorktreeWizard() *workflows.WorktreeCreationWizard {
	// Note: This would need to be implemented in workflows/worktrees.go
	// For now, we'll return nil and handle this case
	return nil
}

// CreateSingleWorktreeSessionWizard creates session wizard for specific worktree
func (f *WorkflowFactory) CreateSingleWorktreeSessionWizard(worktree WorktreeInfo) *modals.MultiStepModal {
	integration := workflows.NewWorktreeSessionIntegration(f.integration, f.theme)
	return integration.CreateNewSessionForWorktree(workflows.WorktreeInfo{
		Path:        worktree.Path,
		Branch:      worktree.Branch,
		ProjectName: worktree.Repository,
		LastAccess:  worktree.LastAccess.Format("2006-01-02 15:04"),
		HasChanges:  worktree.HasChanges,
	})
}

// CreateBulkWorktreeSessionWizard creates bulk session wizard
func (f *WorkflowFactory) CreateBulkWorktreeSessionWizard(worktrees []WorktreeInfo) *modals.MultiStepModal {
	integration := workflows.NewWorktreeSessionIntegration(f.integration, f.theme)

	var workflowWorktrees []workflows.WorktreeInfo
	for _, wt := range worktrees {
		workflowWorktrees = append(workflowWorktrees, workflows.WorktreeInfo{
			Path:        wt.Path,
			Branch:      wt.Branch,
			ProjectName: wt.Repository,
			LastAccess:  wt.LastAccess.Format("2006-01-02 15:04"),
			HasChanges:  wt.HasChanges,
		})
	}

	return integration.CreateBulkSessionsForWorktrees(workflowWorktrees)
}

// CreateGeneralSessionWizard creates a general session wizard
func (f *WorkflowFactory) CreateGeneralSessionWizard() *modals.MultiStepModal {
	wizard := f.CreateSessionWizard()
	return wizard.CreateWizard()
}

// HandleContinueOperation handles continue session operations
func (f *WorkflowFactory) HandleContinueOperation(worktrees []WorktreeInfo) error {
	integration := workflows.NewWorktreeSessionIntegration(f.integration, f.theme)

	// Find and continue sessions for each worktree
	for _, wt := range worktrees {
		workflowWorktree := workflows.WorktreeInfo{
			Path:        wt.Path,
			Branch:      wt.Branch,
			ProjectName: wt.Repository,
			LastAccess:  wt.LastAccess.Format("2006-01-02 15:04"),
			HasChanges:  wt.HasChanges,
		}

		// Execute continue operation
		cmd := integration.ContinueSessionInWorktree(workflowWorktree)
		if cmd != nil {
			// TODO: Execute the command properly
			// For now, we'll assume success
		}
	}

	return nil
}

// HandleResumeOperation handles resume session operations
func (f *WorkflowFactory) HandleResumeOperation(worktrees []WorktreeInfo) error {
	integration := workflows.NewWorktreeSessionIntegration(f.integration, f.theme)

	// Find and resume sessions for each worktree
	for _, wt := range worktrees {
		workflowWorktree := workflows.WorktreeInfo{
			Path:        wt.Path,
			Branch:      wt.Branch,
			ProjectName: wt.Repository,
			LastAccess:  wt.LastAccess.Format("2006-01-02 15:04"),
			HasChanges:  wt.HasChanges,
		}

		// Execute resume operation
		cmd := integration.ResumeSessionInWorktree(workflowWorktree)
		if cmd != nil {
			// TODO: Execute the command properly
			// For now, we'll assume success
		}
	}

	return nil
}

// ValidateWorktreeForOperation validates that a worktree can be used for an operation
func (f *WorkflowFactory) ValidateWorktreeForOperation(worktree WorktreeInfo, operation string) error {
	switch operation {
	case "new":
		// Any worktree can have a new session
		return nil

	case "continue":
		// Check if worktree has existing sessions
		if len(worktree.ActiveSessions) == 0 {
			return fmt.Errorf("no existing sessions found for worktree %s", worktree.Path)
		}
		return nil

	case "resume":
		// Check if worktree has paused sessions
		hasPausedSession := false
		for _, session := range worktree.ActiveSessions {
			if session.State == "paused" {
				hasPausedSession = true
				break
			}
		}
		if !hasPausedSession {
			return fmt.Errorf("no paused sessions found for worktree %s", worktree.Path)
		}
		return nil

	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

// GetOperationSummary returns a summary of what an operation will do
func (f *WorkflowFactory) GetOperationSummary(worktrees []WorktreeInfo, operation string) string {
	switch operation {
	case "new":
		if len(worktrees) == 1 {
			return fmt.Sprintf("Create new session for %s (%s)", worktrees[0].Path, worktrees[0].Branch)
		}
		return fmt.Sprintf("Create new sessions for %d worktrees", len(worktrees))

	case "continue":
		if len(worktrees) == 1 {
			sessionCount := len(worktrees[0].ActiveSessions)
			return fmt.Sprintf("Continue %d existing session(s) in %s", sessionCount, worktrees[0].Path)
		}
		totalSessions := 0
		for _, wt := range worktrees {
			totalSessions += len(wt.ActiveSessions)
		}
		return fmt.Sprintf("Continue %d existing sessions across %d worktrees", totalSessions, len(worktrees))

	case "resume":
		if len(worktrees) == 1 {
			pausedCount := 0
			for _, session := range worktrees[0].ActiveSessions {
				if session.State == "paused" {
					pausedCount++
				}
			}
			return fmt.Sprintf("Resume %d paused session(s) in %s", pausedCount, worktrees[0].Path)
		}
		totalPaused := 0
		for _, wt := range worktrees {
			for _, session := range wt.ActiveSessions {
				if session.State == "paused" {
					totalPaused++
				}
			}
		}
		return fmt.Sprintf("Resume %d paused sessions across %d worktrees", totalPaused, len(worktrees))

	default:
		return fmt.Sprintf("Unknown operation: %s", operation)
	}
}
