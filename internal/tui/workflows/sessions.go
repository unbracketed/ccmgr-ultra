package workflows

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/your-username/ccmgr-ultra/internal/tui/modals"
)

// SessionCreationWizard implements a step-by-step session creation process
type SessionCreationWizard struct {
	integration Integration // Backend integration interface
	theme       modals.Theme
}

// Integration interface for backend operations
type Integration interface {
	GetAvailableProjects() ([]ProjectInfo, error)
	GetAvailableWorktrees() ([]WorktreeInfo, error)
	GetDefaultClaudeConfig(projectPath string) (ClaudeConfig, error)
	CreateSession(config SessionConfig) error
	ValidateSessionName(name string) error
	ValidateProjectPath(path string) error
}

// ProjectInfo represents a project available for session creation
type ProjectInfo struct {
	Name        string
	Path        string
	Description string
	HasClaude   bool
	LastUsed    string
}

// WorktreeInfo represents a worktree available for session creation
type WorktreeInfo struct {
	Path         string
	Branch       string
	ProjectName  string
	LastAccess   string
	HasChanges   bool
}

// ClaudeConfig represents Claude Code configuration for a session
type ClaudeConfig struct {
	Enabled     bool
	MCPServers  []string
	Permissions []string
	ConfigPath  string
}

// SessionConfig represents the configuration for creating a new session
type SessionConfig struct {
	Name         string
	ProjectPath  string
	WorktreePath string
	Branch       string
	Description  string
	ClaudeConfig ClaudeConfig
	AutoStart    bool
}

// NewSessionCreationWizard creates a new session creation wizard
func NewSessionCreationWizard(integration Integration, theme modals.Theme) *SessionCreationWizard {
	return &SessionCreationWizard{
		integration: integration,
		theme:       theme,
	}
}

// CreateWizard returns a multi-step modal for session creation
func (w *SessionCreationWizard) CreateWizard() *modals.MultiStepModal {
	steps := []modals.Step{
		&ProjectSelectionStep{wizard: w},
		&SessionDetailsStep{wizard: w},
		&ClaudeConfigStep{wizard: w},
		&ConfirmationStep{wizard: w},
	}
	
	return modals.NewMultiStepModal(modals.MultiStepModalConfig{
		Title:        "Create New Session",
		Steps:        steps,
		ShowProgress: true,
	})
}

// ProjectSelectionStep handles project/worktree selection
type ProjectSelectionStep struct {
	wizard         *SessionCreationWizard
	selectedType   string // "project" or "worktree"
	selectedIndex  int
	projects       []ProjectInfo
	worktrees      []WorktreeInfo
	loaded         bool
	error          error
}

func (s *ProjectSelectionStep) Title() string {
	return "Select Project"
}

func (s *ProjectSelectionStep) Description() string {
	return "Choose a project or worktree for your new session"
}

func (s *ProjectSelectionStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	if !s.loaded {
		s.loadData()
	}
	
	if s.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		return errorStyle.Render("Error loading projects: " + s.error.Error())
	}
	
	var elements []string
	
	// Type selection
	typeStyle := lipgloss.NewStyle().Bold(true)
	elements = append(elements, typeStyle.Render("Select source type:"))
	
	projectButton := s.renderTypeButton("project", "Projects", theme, len(s.projects))
	worktreeButton := s.renderTypeButton("worktree", "Worktrees", theme, len(s.worktrees))
	
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, projectButton, "  ", worktreeButton)
	elements = append(elements, buttons)
	elements = append(elements, "")
	
	// List items based on selected type
	if s.selectedType == "project" {
		elements = append(elements, s.renderProjectList(theme, width))
	} else if s.selectedType == "worktree" {
		elements = append(elements, s.renderWorktreeList(theme, width))
	}
	
	return strings.Join(elements, "\n")
}

func (s *ProjectSelectionStep) renderTypeButton(buttonType, label string, theme modals.Theme, count int) string {
	style := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder())
	
	if s.selectedType == buttonType {
		style = style.
			Background(theme.Accent).
			BorderForeground(theme.Accent).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
	} else {
		style = style.
			BorderForeground(theme.Muted).
			Foreground(theme.Text)
	}
	
	text := fmt.Sprintf("%s (%d)", label, count)
	return style.Render(text)
}

func (s *ProjectSelectionStep) renderProjectList(theme modals.Theme, width int) string {
	if len(s.projects) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("No projects found")
	}
	
	var items []string
	for i, project := range s.projects {
		cursor := " "
		if i == s.selectedIndex {
			cursor = ">"
		}
		
		status := ""
		if project.HasClaude {
			status = lipgloss.NewStyle().Foreground(theme.Success).Render("●")
		} else {
			status = lipgloss.NewStyle().Foreground(theme.Muted).Render("○")
		}
		
		line := fmt.Sprintf("%s %s %s", cursor, status, project.Name)
		if project.Description != "" {
			line += lipgloss.NewStyle().Foreground(theme.Muted).Render(" - " + project.Description)
		}
		
		items = append(items, line)
	}
	
	return strings.Join(items, "\n")
}

func (s *ProjectSelectionStep) renderWorktreeList(theme modals.Theme, width int) string {
	if len(s.worktrees) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("No worktrees found")
	}
	
	var items []string
	for i, worktree := range s.worktrees {
		cursor := " "
		if i == s.selectedIndex {
			cursor = ">"
		}
		
		status := ""
		if worktree.HasChanges {
			status = lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
		} else {
			status = lipgloss.NewStyle().Foreground(theme.Success).Render("●")
		}
		
		line := fmt.Sprintf("%s %s %s (%s)", cursor, status, filepath.Base(worktree.Path), worktree.Branch)
		
		items = append(items, line)
	}
	
	return strings.Join(items, "\n")
}

func (s *ProjectSelectionStep) loadData() {
	s.loaded = true
	
	projects, err := s.wizard.integration.GetAvailableProjects()
	if err != nil {
		s.error = err
		return
	}
	s.projects = projects
	
	worktrees, err := s.wizard.integration.GetAvailableWorktrees()
	if err != nil {
		s.error = err
		return
	}
	s.worktrees = worktrees
	
	// Set default selection
	if len(s.projects) > 0 {
		s.selectedType = "project"
	} else if len(s.worktrees) > 0 {
		s.selectedType = "worktree"
	}
}

func (s *ProjectSelectionStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "tab":
		if s.selectedType == "project" && len(s.worktrees) > 0 {
			s.selectedType = "worktree"
			s.selectedIndex = 0
		} else if s.selectedType == "worktree" && len(s.projects) > 0 {
			s.selectedType = "project"
			s.selectedIndex = 0
		}
		
	case "up", "k":
		if s.selectedIndex > 0 {
			s.selectedIndex--
		}
		
	case "down", "j":
		maxIndex := 0
		if s.selectedType == "project" {
			maxIndex = len(s.projects) - 1
		} else if s.selectedType == "worktree" {
			maxIndex = len(s.worktrees) - 1
		}
		
		if s.selectedIndex < maxIndex {
			s.selectedIndex++
		}
		
	case "enter", " ":
		// Store selection in data
		if s.selectedType == "project" && s.selectedIndex < len(s.projects) {
			project := s.projects[s.selectedIndex]
			data["project_path"] = project.Path
			data["project_name"] = project.Name
			data["has_claude"] = project.HasClaude
		} else if s.selectedType == "worktree" && s.selectedIndex < len(s.worktrees) {
			worktree := s.worktrees[s.selectedIndex]
			data["worktree_path"] = worktree.Path
			data["project_path"] = worktree.Path
			data["project_name"] = worktree.ProjectName
			data["branch"] = worktree.Branch
		}
		data["source_type"] = s.selectedType
	}
	
	return data, nil, nil
}

func (s *ProjectSelectionStep) Validate(data map[string]interface{}) error {
	sourceType, ok := data["source_type"].(string)
	if !ok || sourceType == "" {
		return fmt.Errorf("please select a project or worktree")
	}
	
	if sourceType == "project" {
		projectPath, ok := data["project_path"].(string)
		if !ok || projectPath == "" {
			return fmt.Errorf("please select a project")
		}
	} else if sourceType == "worktree" {
		worktreePath, ok := data["worktree_path"].(string)
		if !ok || worktreePath == "" {
			return fmt.Errorf("please select a worktree")
		}
	}
	
	return nil
}

func (s *ProjectSelectionStep) IsComplete(data map[string]interface{}) bool {
	return s.Validate(data) == nil
}

// SessionDetailsStep handles session name and description
type SessionDetailsStep struct {
	wizard    *SessionCreationWizard
	nameInput string
	descInput string
	cursor    int // 0 = name, 1 = description
}

func (s *SessionDetailsStep) Title() string {
	return "Session Details"
}

func (s *SessionDetailsStep) Description() string {
	return "Enter a name and description for your session"
}

func (s *SessionDetailsStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string
	
	// Session name
	nameLabel := lipgloss.NewStyle().Bold(true).Render("Session Name:")
	elements = append(elements, nameLabel)
	
	nameStyle := lipgloss.NewStyle().
		Width(width - 8).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	
	if s.cursor == 0 {
		nameStyle = nameStyle.BorderForeground(theme.Accent)
	} else {
		nameStyle = nameStyle.BorderForeground(theme.Muted)
	}
	
	nameDisplay := s.nameInput
	if s.cursor == 0 {
		nameDisplay += "│"
	}
	
	nameField := nameStyle.Render(nameDisplay)
	elements = append(elements, nameField)
	elements = append(elements, "")
	
	// Description
	descLabel := lipgloss.NewStyle().Bold(true).Render("Description (optional):")
	elements = append(elements, descLabel)
	
	descStyle := lipgloss.NewStyle().
		Width(width - 8).
		Height(3).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	
	if s.cursor == 1 {
		descStyle = descStyle.BorderForeground(theme.Accent)
	} else {
		descStyle = descStyle.BorderForeground(theme.Muted)
	}
	
	descDisplay := s.descInput
	if s.cursor == 1 {
		descDisplay += "│"
	}
	
	descField := descStyle.Render(descDisplay)
	elements = append(elements, descField)
	
	// Help
	helpStyle := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	help := helpStyle.Render("Tab: Switch fields • Type to enter text")
	elements = append(elements, "", help)
	
	return strings.Join(elements, "\n")
}

func (s *SessionDetailsStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "tab":
		s.cursor = (s.cursor + 1) % 2
		
	case "backspace":
		if s.cursor == 0 && len(s.nameInput) > 0 {
			s.nameInput = s.nameInput[:len(s.nameInput)-1]
		} else if s.cursor == 1 && len(s.descInput) > 0 {
			s.descInput = s.descInput[:len(s.descInput)-1]
		}
		
	default:
		if len(msg.Runes) > 0 {
			char := string(msg.Runes[0])
			if s.cursor == 0 && len(s.nameInput) < 50 {
				s.nameInput += char
			} else if s.cursor == 1 && len(s.descInput) < 200 {
				s.descInput += char
			}
		}
	}
	
	// Store in data
	data["session_name"] = s.nameInput
	data["session_description"] = s.descInput
	
	return data, nil, nil
}

func (s *SessionDetailsStep) Validate(data map[string]interface{}) error {
	name, _ := data["session_name"].(string)
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("session name is required")
	}
	
	// Validate with backend
	if err := s.wizard.integration.ValidateSessionName(name); err != nil {
		return err
	}
	
	return nil
}

func (s *SessionDetailsStep) IsComplete(data map[string]interface{}) bool {
	return s.Validate(data) == nil
}

// ClaudeConfigStep handles Claude Code configuration
type ClaudeConfigStep struct {
	wizard      *SessionCreationWizard
	enableClaude bool
	configLoaded bool
	defaultConfig ClaudeConfig
}

func (s *ClaudeConfigStep) Title() string {
	return "Claude Configuration"
}

func (s *ClaudeConfigStep) Description() string {
	return "Configure Claude Code integration for this session"
}

func (s *ClaudeConfigStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	if !s.configLoaded {
		s.loadDefaultConfig(data)
	}
	
	var elements []string
	
	// Claude toggle
	toggleStyle := lipgloss.NewStyle().Bold(true)
	elements = append(elements, toggleStyle.Render("Enable Claude Code Integration:"))
	
	checkBox := "☐"
	if s.enableClaude {
		checkBox = "☑"
	}
	
	checkStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true)
	
	checkbox := lipgloss.JoinHorizontal(lipgloss.Left,
		checkStyle.Render(checkBox), " Enable Claude Code")
	elements = append(elements, checkbox)
	
	if s.enableClaude {
		elements = append(elements, "")
		
		// Configuration details
		if s.defaultConfig.ConfigPath != "" {
			configStyle := lipgloss.NewStyle().Foreground(theme.Muted)
			elements = append(elements, configStyle.Render("Config file: "+s.defaultConfig.ConfigPath))
		}
		
		if len(s.defaultConfig.MCPServers) > 0 {
			mcpStyle := lipgloss.NewStyle().Foreground(theme.Muted)
			elements = append(elements, mcpStyle.Render("MCP Servers: "+strings.Join(s.defaultConfig.MCPServers, ", ")))
		}
		
		if len(s.defaultConfig.Permissions) > 0 {
			permStyle := lipgloss.NewStyle().Foreground(theme.Muted)
			elements = append(elements, permStyle.Render("Permissions: "+strings.Join(s.defaultConfig.Permissions, ", ")))
		}
	}
	
	// Help
	helpStyle := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	help := helpStyle.Render("Space: Toggle Claude integration")
	elements = append(elements, "", help)
	
	return strings.Join(elements, "\n")
}

func (s *ClaudeConfigStep) loadDefaultConfig(data map[string]interface{}) {
	s.configLoaded = true
	
	projectPath, _ := data["project_path"].(string)
	if projectPath != "" {
		config, err := s.wizard.integration.GetDefaultClaudeConfig(projectPath)
		if err == nil {
			s.defaultConfig = config
			s.enableClaude = config.Enabled
		}
	}
	
	// Check if project already has Claude
	if hasClaude, ok := data["has_claude"].(bool); ok {
		s.enableClaude = hasClaude
	}
}

func (s *ClaudeConfigStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case " ":
		s.enableClaude = !s.enableClaude
	}
	
	// Store in data
	claudeConfig := s.defaultConfig
	claudeConfig.Enabled = s.enableClaude
	data["claude_config"] = claudeConfig
	
	return data, nil, nil
}

func (s *ClaudeConfigStep) Validate(data map[string]interface{}) error {
	// No validation needed for this step
	return nil
}

func (s *ClaudeConfigStep) IsComplete(data map[string]interface{}) bool {
	return true
}

// ConfirmationStep shows a summary and confirms session creation
type ConfirmationStep struct {
	wizard *SessionCreationWizard
}

func (s *ConfirmationStep) Title() string {
	return "Confirmation"
}

func (s *ConfirmationStep) Description() string {
	return "Review and confirm your session configuration"
}

func (s *ConfirmationStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string
	
	summaryStyle := lipgloss.NewStyle().Bold(true)
	elements = append(elements, summaryStyle.Render("Session Summary:"))
	elements = append(elements, "")
	
	// Session details
	if name, ok := data["session_name"].(string); ok {
		elements = append(elements, fmt.Sprintf("Name: %s", name))
	}
	
	if desc, ok := data["session_description"].(string); ok && desc != "" {
		elements = append(elements, fmt.Sprintf("Description: %s", desc))
	}
	
	if projectName, ok := data["project_name"].(string); ok {
		elements = append(elements, fmt.Sprintf("Project: %s", projectName))
	}
	
	if projectPath, ok := data["project_path"].(string); ok {
		elements = append(elements, fmt.Sprintf("Path: %s", projectPath))
	}
	
	if branch, ok := data["branch"].(string); ok && branch != "" {
		elements = append(elements, fmt.Sprintf("Branch: %s", branch))
	}
	
	// Claude configuration
	if claudeConfig, ok := data["claude_config"].(ClaudeConfig); ok {
		claudeStatus := "Disabled"
		if claudeConfig.Enabled {
			claudeStatus = "Enabled"
		}
		elements = append(elements, fmt.Sprintf("Claude Code: %s", claudeStatus))
	}
	
	elements = append(elements, "")
	
	// Confirmation message
	confirmStyle := lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true)
	elements = append(elements, confirmStyle.Render("Press Ctrl+Enter to create session"))
	
	return strings.Join(elements, "\n")
}

func (s *ConfirmationStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	// No key handling needed - completion handled by wizard
	return data, nil, nil
}

func (s *ConfirmationStep) Validate(data map[string]interface{}) error {
	return nil
}

func (s *ConfirmationStep) IsComplete(data map[string]interface{}) bool {
	return true
}