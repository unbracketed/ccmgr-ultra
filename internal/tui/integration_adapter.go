package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unbracketed/ccmgr-ultra/internal/config"
	"github.com/unbracketed/ccmgr-ultra/internal/tui/workflows"
)

// IntegrationAdapter adapts the TUI Integration to workflow interfaces
type IntegrationAdapter struct {
	integration *Integration
	config      *config.Config
}

// NewIntegrationAdapter creates a new integration adapter
func NewIntegrationAdapter(integration *Integration, config *config.Config) *IntegrationAdapter {
	return &IntegrationAdapter{
		integration: integration,
		config:      config,
	}
}

// GetAvailableProjects derives project list from worktrees
func (a *IntegrationAdapter) GetAvailableProjects() ([]workflows.ProjectInfo, error) {
	worktrees := a.integration.GetAllWorktrees()
	projectMap := make(map[string]workflows.ProjectInfo)

	for _, wt := range worktrees {
		if _, exists := projectMap[wt.Repository]; !exists {
			// Count worktrees for this repository
			worktreeCount := 0
			hasClaudeCount := 0
			for _, w := range worktrees {
				if w.Repository == wt.Repository {
					worktreeCount++
					if len(w.ActiveSessions) > 0 {
						hasClaudeCount++
					}
				}
			}

			projectMap[wt.Repository] = workflows.ProjectInfo{
				Name:        wt.Repository,
				Path:        filepath.Dir(wt.Path),
				Description: fmt.Sprintf("Repository with %d worktrees", worktreeCount),
				HasClaude:   hasClaudeCount > 0,
				LastUsed:    wt.LastAccess.Format("2006-01-02 15:04"),
			}
		}
	}

	var projects []workflows.ProjectInfo
	for _, project := range projectMap {
		projects = append(projects, project)
	}

	return projects, nil
}

// GetAvailableWorktrees wraps existing method with interface conversion
func (a *IntegrationAdapter) GetAvailableWorktrees() ([]workflows.WorktreeInfo, error) {
	worktrees := a.integration.GetAllWorktrees()
	var result []workflows.WorktreeInfo

	for _, wt := range worktrees {
		result = append(result, workflows.WorktreeInfo{
			Path:        wt.Path,
			Branch:      wt.Branch,
			ProjectName: wt.Repository,
			LastAccess:  wt.LastAccess.Format("2006-01-02 15:04"),
			HasChanges:  wt.HasChanges,
		})
	}

	return result, nil
}

// CreateSession creates a new session using the integration layer
func (a *IntegrationAdapter) CreateSession(config workflows.SessionConfig) error {
	// Use the integration layer to create the session
	cmd := a.integration.CreateSession(config.Name, config.WorktreePath)

	// Execute the command (this is a simplified approach)
	// In a real implementation, we would need to handle the async nature properly
	if cmd != nil {
		// For now, we assume success
		// TODO: Implement proper async handling
		return nil
	}

	return fmt.Errorf("failed to create session")
}

// ValidateSessionName validates a session name
func (a *IntegrationAdapter) ValidateSessionName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	if len(name) > 50 {
		return fmt.Errorf("session name cannot exceed 50 characters")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("session name cannot contain whitespace")
	}

	// Check if session name already exists
	sessions := a.integration.GetAllSessions()
	for _, session := range sessions {
		if session.Name == name {
			return fmt.Errorf("session name '%s' already exists", name)
		}
	}

	return nil
}

// ValidateProjectPath validates a project path
func (a *IntegrationAdapter) ValidateProjectPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("project path cannot be empty")
	}

	// Check if path exists and is accessible
	// TODO: Add actual filesystem validation

	return nil
}

// ValidateWorktreePath validates a worktree path
func (a *IntegrationAdapter) ValidateWorktreePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	// Check if the worktree exists in our tracked worktrees
	worktrees := a.integration.GetAllWorktrees()
	for _, wt := range worktrees {
		if wt.Path == path {
			return nil // Found it
		}
	}

	return fmt.Errorf("worktree path '%s' not found in tracked worktrees", path)
}

// GetDefaultClaudeConfig returns default Claude configuration for a project
func (a *IntegrationAdapter) GetDefaultClaudeConfig(projectPath string) (workflows.ClaudeConfig, error) {
	// Check if project already has Claude configuration
	worktrees := a.integration.GetAllWorktrees()

	for _, wt := range worktrees {
		if strings.HasPrefix(wt.Path, projectPath) && len(wt.ActiveSessions) > 0 {
			// Project has existing Claude sessions
			return workflows.ClaudeConfig{
				Enabled:     true,
				MCPServers:  []string{"memory", "filesystem", "web"},
				Permissions: []string{"read", "write", "execute"},
				ConfigPath:  filepath.Join(projectPath, ".claude/config.json"),
			}, nil
		}
	}

	// Return default configuration
	return workflows.ClaudeConfig{
		Enabled:     false,
		MCPServers:  []string{"memory", "filesystem"},
		Permissions: []string{"read", "write"},
		ConfigPath:  filepath.Join(projectPath, ".claude/config.json"),
	}, nil
}

// FindSessionsForWorktree finds existing sessions for a worktree
func (a *IntegrationAdapter) FindSessionsForWorktree(worktreePath string) ([]workflows.SessionInfo, error) {
	sessions := a.integration.GetAllSessions()
	var result []workflows.SessionInfo

	for _, session := range sessions {
		if session.Directory == worktreePath {
			result = append(result, workflows.SessionInfo{
				ID:         session.ID,
				Name:       session.Name,
				Path:       session.Directory,
				Branch:     session.Branch,
				Active:     session.Active,
				Created:    session.Created.Format("2006-01-02 15:04:05"),
				LastAccess: session.LastAccess.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return result, nil
}

// AttachToSession attaches to an existing session
func (a *IntegrationAdapter) AttachToSession(sessionID string) error {
	cmd := a.integration.AttachSession(sessionID)
	if cmd != nil {
		// For now, we assume success
		// TODO: Implement proper async handling
		return nil
	}

	return fmt.Errorf("failed to attach to session")
}

// GetDefaultWorktreeDir returns the default directory for creating worktrees
func (a *IntegrationAdapter) GetDefaultWorktreeDir(repoPath string) (string, error) {
	// Return a sensible default based on the repository path
	return filepath.Join(filepath.Dir(repoPath), "worktrees"), nil
}

// GetAvailableProjects and GetAvailableWorktrees are already implemented above

// Additional helper methods for enhanced functionality

// GetWorktreeByPath finds a specific worktree by path
func (a *IntegrationAdapter) GetWorktreeByPath(path string) (*workflows.WorktreeInfo, error) {
	worktrees, err := a.GetAvailableWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Path == path {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("worktree not found: %s", path)
}

// ValidateSessionConfig validates a complete session configuration
func (a *IntegrationAdapter) ValidateSessionConfig(config workflows.SessionConfig) error {
	if err := a.ValidateSessionName(config.Name); err != nil {
		return err
	}

	if config.WorktreePath != "" {
		if err := a.ValidateWorktreePath(config.WorktreePath); err != nil {
			return err
		}
	} else if config.ProjectPath != "" {
		if err := a.ValidateProjectPath(config.ProjectPath); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("either worktree path or project path must be specified")
	}

	return nil
}
