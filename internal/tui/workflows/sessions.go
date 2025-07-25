package workflows

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/unbracketed/ccmgr-ultra/internal/tui/modals"
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
	Path        string
	Branch      string
	ProjectName string
	LastAccess  string
	HasChanges  bool
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

// SessionInfo represents session information for workflows
type SessionInfo struct {
	ID         string
	Name       string
	Path       string
	Branch     string
	Active     bool
	Created    string
	LastAccess string
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
	wizard        *SessionCreationWizard
	selectedType  string // "project" or "worktree"
	selectedIndex int
	projects      []ProjectInfo
	worktrees     []WorktreeInfo
	loaded        bool
	error         error
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
		Width(width-8).
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
		Width(width-8).
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
	wizard        *SessionCreationWizard
	enableClaude  bool
	configLoaded  bool
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

// Enhanced Session-Worktree Integration Workflows
// These methods support the n/c/r keyboard shortcuts from WorktreesModel

// WorktreeSessionIntegration provides session operations for specific worktrees
type WorktreeSessionIntegration struct {
	integration Integration
	theme       modals.Theme
}

// NewWorktreeSessionIntegration creates a new worktree session integration
func NewWorktreeSessionIntegration(integration Integration, theme modals.Theme) *WorktreeSessionIntegration {
	return &WorktreeSessionIntegration{
		integration: integration,
		theme:       theme,
	}
}

// CreateNewSessionForWorktree creates a new session for a specific worktree
func (w *WorktreeSessionIntegration) CreateNewSessionForWorktree(worktree WorktreeInfo) *modals.MultiStepModal {
	steps := []modals.Step{
		&WorktreeSessionDetailsStep{
			integration: w.integration,
			worktree:    worktree,
		},
		&WorktreeClaudeConfigStep{
			integration: w.integration,
			worktree:    worktree,
		},
		&WorktreeSessionConfirmationStep{
			integration: w.integration,
			worktree:    worktree,
		},
	}

	return modals.NewMultiStepModal(modals.MultiStepModalConfig{
		Title:        fmt.Sprintf("New Session for %s", worktree.Branch),
		Steps:        steps,
		ShowProgress: true,
	})
}

// CreateBulkSessionsForWorktrees creates sessions for multiple worktrees
func (w *WorktreeSessionIntegration) CreateBulkSessionsForWorktrees(worktrees []WorktreeInfo) *modals.MultiStepModal {
	steps := []modals.Step{
		&BulkSessionConfigStep{
			integration: w.integration,
			worktrees:   worktrees,
		},
		&BulkSessionConfirmationStep{
			integration: w.integration,
			worktrees:   worktrees,
		},
	}

	return modals.NewMultiStepModal(modals.MultiStepModalConfig{
		Title:        fmt.Sprintf("Create Sessions for %d Worktrees", len(worktrees)),
		Steps:        steps,
		ShowProgress: true,
	})
}

// ContinueSessionInWorktree finds and attaches to existing session for a worktree
func (w *WorktreeSessionIntegration) ContinueSessionInWorktree(worktree WorktreeInfo) tea.Cmd {
	return func() tea.Msg {
		// Find existing sessions for the worktree
		sessions, err := w.integration.GetAvailableWorktrees()
		if err != nil {
			return SessionContinueMsg{
				WorktreePath: worktree.Path,
				Success:      false,
				Message:      fmt.Sprintf("Error finding sessions: %v", err),
			}
		}

		// Look for sessions in this worktree
		for _, wt := range sessions {
			if wt.Path == worktree.Path {
				// Check if worktree has active sessions (simplified check)
				if wt.HasChanges { // Using HasChanges as a proxy for activity
					return SessionContinueMsg{
						WorktreePath: worktree.Path,
						Success:      true,
						Message:      fmt.Sprintf("Continuing session in %s", worktree.Branch),
						SessionID:    "session-" + worktree.Branch,
					}
				}
			}
		}

		return SessionContinueMsg{
			WorktreePath: worktree.Path,
			Success:      false,
			Message:      "No existing sessions found for this worktree",
		}
	}
}

// ResumeSessionInWorktree restores a paused session for a worktree
func (w *WorktreeSessionIntegration) ResumeSessionInWorktree(worktree WorktreeInfo) tea.Cmd {
	return func() tea.Msg {
		// This would restore a paused session
		// For demonstration, we'll simulate finding and resuming a session
		sessionID := "session-" + worktree.Branch + "-paused"

		return SessionResumeMsg{
			WorktreePath: worktree.Path,
			Success:      true,
			Message:      fmt.Sprintf("Resuming session in %s", worktree.Branch),
			SessionID:    sessionID,
		}
	}
}

// WorktreeSessionDetailsStep handles session configuration for a specific worktree
type WorktreeSessionDetailsStep struct {
	integration Integration
	worktree    WorktreeInfo
	cursor      int
	nameInput   string
	descInput   string
}

func (s *WorktreeSessionDetailsStep) Title() string {
	return "Session Details"
}

func (s *WorktreeSessionDetailsStep) Description() string {
	return fmt.Sprintf("Configure session for worktree: %s (%s)", s.worktree.Path, s.worktree.Branch)
}

func (s *WorktreeSessionDetailsStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string

	// Worktree info
	worktreeStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	elements = append(elements, worktreeStyle.Render(fmt.Sprintf("Worktree: %s", s.worktree.Path)))
	elements = append(elements, fmt.Sprintf("Branch: %s", s.worktree.Branch))
	elements = append(elements, "")

	// Session name input
	nameLabel := "Session Name:"
	if s.cursor == 0 {
		nameLabel = "> " + nameLabel
	} else {
		nameLabel = "  " + nameLabel
	}
	elements = append(elements, nameLabel)

	nameValue := s.nameInput
	if nameValue == "" {
		nameValue = fmt.Sprintf("session-%s", strings.ReplaceAll(s.worktree.Branch, "/", "-"))
	}
	elements = append(elements, fmt.Sprintf("  %s", nameValue))
	elements = append(elements, "")

	// Description input
	descLabel := "Description (optional):"
	if s.cursor == 1 {
		descLabel = "> " + descLabel
	} else {
		descLabel = "  " + descLabel
	}
	elements = append(elements, descLabel)
	elements = append(elements, fmt.Sprintf("  %s", s.descInput))

	return strings.Join(elements, "\n")
}

func (s *WorktreeSessionDetailsStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < 1 {
			s.cursor++
		}
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
	sessionName := s.nameInput
	if sessionName == "" {
		sessionName = fmt.Sprintf("session-%s", strings.ReplaceAll(s.worktree.Branch, "/", "-"))
	}

	data["session_name"] = sessionName
	data["session_description"] = s.descInput
	data["worktree_path"] = s.worktree.Path
	data["project_path"] = s.worktree.Path
	data["branch"] = s.worktree.Branch

	return data, nil, nil
}

func (s *WorktreeSessionDetailsStep) Validate(data map[string]interface{}) error {
	name, _ := data["session_name"].(string)
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("session name is required")
	}

	return s.integration.ValidateSessionName(name)
}

func (s *WorktreeSessionDetailsStep) IsComplete(data map[string]interface{}) bool {
	return s.Validate(data) == nil
}

// WorktreeClaudeConfigStep handles Claude configuration for worktree sessions
type WorktreeClaudeConfigStep struct {
	integration   Integration
	worktree      WorktreeInfo
	enableClaude  bool
	configLoaded  bool
	defaultConfig ClaudeConfig
}

func (s *WorktreeClaudeConfigStep) Title() string {
	return "Claude Configuration"
}

func (s *WorktreeClaudeConfigStep) Description() string {
	return "Configure Claude Code integration for this session"
}

func (s *WorktreeClaudeConfigStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	if !s.configLoaded {
		// Load default config for worktree
		config, err := s.integration.GetDefaultClaudeConfig(s.worktree.Path)
		if err == nil {
			s.defaultConfig = config
			s.enableClaude = config.Enabled
		}
		s.configLoaded = true
	}

	var elements []string

	// Enable/disable toggle
	enableText := "Disable Claude Code"
	if !s.enableClaude {
		enableText = "Enable Claude Code"
	}
	elements = append(elements, fmt.Sprintf("> %s", enableText))
	elements = append(elements, "")

	if s.enableClaude {
		elements = append(elements, "Claude Code Configuration:")
		elements = append(elements, fmt.Sprintf("  Config Path: %s", s.defaultConfig.ConfigPath))
		elements = append(elements, fmt.Sprintf("  MCP Servers: %d configured", len(s.defaultConfig.MCPServers)))
		elements = append(elements, fmt.Sprintf("  Permissions: %d configured", len(s.defaultConfig.Permissions)))
	} else {
		elements = append(elements, "Claude Code will not be enabled for this session.")
	}

	return strings.Join(elements, "\n")
}

func (s *WorktreeClaudeConfigStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "enter", " ":
		s.enableClaude = !s.enableClaude
	}

	// Store configuration
	if s.enableClaude {
		data["claude_config"] = s.defaultConfig
	} else {
		data["claude_config"] = ClaudeConfig{Enabled: false}
	}

	return data, nil, nil
}

func (s *WorktreeClaudeConfigStep) Validate(data map[string]interface{}) error {
	return nil
}

func (s *WorktreeClaudeConfigStep) IsComplete(data map[string]interface{}) bool {
	return true
}

// WorktreeSessionConfirmationStep shows confirmation for worktree session creation
type WorktreeSessionConfirmationStep struct {
	integration Integration
	worktree    WorktreeInfo
}

func (s *WorktreeSessionConfirmationStep) Title() string {
	return "Confirm Session Creation"
}

func (s *WorktreeSessionConfirmationStep) Description() string {
	return "Review session configuration before creation"
}

func (s *WorktreeSessionConfirmationStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	elements = append(elements, headerStyle.Render("Session Configuration Summary"))
	elements = append(elements, "")

	// Session details
	if sessionName, ok := data["session_name"].(string); ok {
		elements = append(elements, fmt.Sprintf("Session Name: %s", sessionName))
	}

	if description, ok := data["session_description"].(string); ok && description != "" {
		elements = append(elements, fmt.Sprintf("Description: %s", description))
	}

	elements = append(elements, fmt.Sprintf("Worktree: %s", s.worktree.Path))
	elements = append(elements, fmt.Sprintf("Branch: %s", s.worktree.Branch))

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
	confirmStyle := lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
	elements = append(elements, confirmStyle.Render("Press Ctrl+Enter to create session"))

	return strings.Join(elements, "\n")
}

func (s *WorktreeSessionConfirmationStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	return data, nil, nil
}

func (s *WorktreeSessionConfirmationStep) Validate(data map[string]interface{}) error {
	return nil
}

func (s *WorktreeSessionConfirmationStep) IsComplete(data map[string]interface{}) bool {
	return true
}

// Bulk session creation steps

// BulkSessionConfigStep handles configuration for bulk session creation
type BulkSessionConfigStep struct {
	integration   Integration
	worktrees     []WorktreeInfo
	cursor        int
	namingPattern string
	enableClaude  bool
	autoStart     bool
}

func (s *BulkSessionConfigStep) Title() string {
	return "Bulk Session Configuration"
}

func (s *BulkSessionConfigStep) Description() string {
	return fmt.Sprintf("Configure sessions for %d worktrees", len(s.worktrees))
}

func (s *BulkSessionConfigStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string

	// Worktree list
	elements = append(elements, "Selected Worktrees:")
	for _, wt := range s.worktrees {
		elements = append(elements, fmt.Sprintf("  • %s (%s)", wt.Path, wt.Branch))
	}
	elements = append(elements, "")

	// Configuration options
	options := []string{
		fmt.Sprintf("Naming Pattern: %s", s.getNameExample()),
		fmt.Sprintf("Enable Claude: %t", s.enableClaude),
		fmt.Sprintf("Auto Start: %t", s.autoStart),
	}

	for i, option := range options {
		if i == s.cursor {
			elements = append(elements, "> "+option)
		} else {
			elements = append(elements, "  "+option)
		}
	}

	return strings.Join(elements, "\n")
}

func (s *BulkSessionConfigStep) getNameExample() string {
	if s.namingPattern == "" {
		s.namingPattern = "session-{branch}"
	}
	return s.namingPattern
}

func (s *BulkSessionConfigStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < 2 {
			s.cursor++
		}
	case "enter", " ":
		switch s.cursor {
		case 1: // Enable Claude
			s.enableClaude = !s.enableClaude
		case 2: // Auto Start
			s.autoStart = !s.autoStart
		}
	}

	// Store configuration
	data["naming_pattern"] = s.namingPattern
	data["enable_claude"] = s.enableClaude
	data["auto_start"] = s.autoStart
	data["worktrees"] = s.worktrees

	return data, nil, nil
}

func (s *BulkSessionConfigStep) Validate(data map[string]interface{}) error {
	return nil
}

func (s *BulkSessionConfigStep) IsComplete(data map[string]interface{}) bool {
	return true
}

// BulkSessionConfirmationStep shows confirmation for bulk session creation
type BulkSessionConfirmationStep struct {
	integration Integration
	worktrees   []WorktreeInfo
}

func (s *BulkSessionConfirmationStep) Title() string {
	return "Confirm Bulk Session Creation"
}

func (s *BulkSessionConfirmationStep) Description() string {
	return "Review bulk session configuration"
}

func (s *BulkSessionConfirmationStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string

	headerStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	elements = append(elements, headerStyle.Render("Bulk Session Creation Summary"))
	elements = append(elements, "")

	elements = append(elements, fmt.Sprintf("Sessions to create: %d", len(s.worktrees)))

	if pattern, ok := data["naming_pattern"].(string); ok {
		elements = append(elements, fmt.Sprintf("Naming pattern: %s", pattern))
	}

	if enableClaude, ok := data["enable_claude"].(bool); ok {
		elements = append(elements, fmt.Sprintf("Claude enabled: %t", enableClaude))
	}

	if autoStart, ok := data["auto_start"].(bool); ok {
		elements = append(elements, fmt.Sprintf("Auto start: %t", autoStart))
	}

	elements = append(elements, "")
	elements = append(elements, "Sessions will be created for:")
	for _, wt := range s.worktrees {
		sessionName := strings.ReplaceAll("session-{branch}", "{branch}", wt.Branch)
		elements = append(elements, fmt.Sprintf("  • %s → %s", wt.Path, sessionName))
	}

	elements = append(elements, "")
	confirmStyle := lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
	elements = append(elements, confirmStyle.Render("Press Ctrl+Enter to create all sessions"))

	return strings.Join(elements, "\n")
}

func (s *BulkSessionConfirmationStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	return data, nil, nil
}

func (s *BulkSessionConfirmationStep) Validate(data map[string]interface{}) error {
	return nil
}

func (s *BulkSessionConfirmationStep) IsComplete(data map[string]interface{}) bool {
	return true
}

// Message types for session-worktree integration
type SessionContinueMsg struct {
	WorktreePath string
	Success      bool
	Message      string
	SessionID    string
}

type SessionResumeMsg struct {
	WorktreePath string
	Success      bool
	Message      string
	SessionID    string
}

type SessionCreatedMsg struct {
	WorktreePath string
	SessionID    string
	Success      bool
	Message      string
}
