package workflows

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/your-username/ccmgr-ultra/internal/tui/modals"
)

// WorktreeCreationWizard implements a step-by-step worktree creation process
type WorktreeCreationWizard struct {
	integration WorktreeIntegration
	theme       modals.Theme
}

// WorktreeIntegration interface for worktree-specific backend operations
type WorktreeIntegration interface {
	GetAvailableRepositories() ([]RepositoryInfo, error)
	GetBranches(repoPath string) ([]BranchInfo, error)
	GetDefaultWorktreeDir(repoPath string) (string, error)
	ValidateBranchName(name string) error
	ValidateWorktreePath(path string) error
	CreateWorktree(config WorktreeConfig) error
	CheckRemoteExists(repoPath, branchName string) (bool, error)
}

// RepositoryInfo represents a Git repository
type RepositoryInfo struct {
	Name          string
	Path          string
	CurrentBranch string
	RemoteURL     string
	HasWorktrees  bool
	WorktreeCount int
}

// BranchInfo represents a Git branch
type BranchInfo struct {
	Name     string
	Remote   bool
	Current  bool
	LastCommit string
	Author   string
}

// WorktreeConfig represents the configuration for creating a new worktree
type WorktreeConfig struct {
	RepositoryPath string
	WorktreePath   string
	BranchName     string
	BaseBranch     string
	NewBranch      bool
	TrackRemote    bool
	CreateSession  bool
	SessionName    string
}

// NewWorktreeCreationWizard creates a new worktree creation wizard
func NewWorktreeCreationWizard(integration WorktreeIntegration, theme modals.Theme) *WorktreeCreationWizard {
	return &WorktreeCreationWizard{
		integration: integration,
		theme:       theme,
	}
}

// CreateWizard returns a multi-step modal for worktree creation
func (w *WorktreeCreationWizard) CreateWizard() *modals.MultiStepModal {
	steps := []modals.Step{
		&RepositorySelectionStep{wizard: w},
		&BranchSelectionStep{wizard: w},
		&WorktreePathStep{wizard: w},
		&WorktreeConfirmationStep{wizard: w},
	}
	
	return modals.NewMultiStepModal(modals.MultiStepModalConfig{
		Title:        "Create New Worktree",
		Steps:        steps,
		ShowProgress: true,
	})
}

// RepositorySelectionStep handles repository selection
type RepositorySelectionStep struct {
	wizard        *WorktreeCreationWizard
	repositories  []RepositoryInfo
	selectedIndex int
	loaded        bool
	error         error
}

func (s *RepositorySelectionStep) Title() string {
	return "Select Repository"
}

func (s *RepositorySelectionStep) Description() string {
	return "Choose the repository for your new worktree"
}

func (s *RepositorySelectionStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	if !s.loaded {
		s.loadRepositories()
	}
	
	if s.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		return errorStyle.Render("Error loading repositories: " + s.error.Error())
	}
	
	if len(s.repositories) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("No Git repositories found")
	}
	
	var elements []string
	
	// Repository list
	for i, repo := range s.repositories {
		cursor := " "
		if i == s.selectedIndex {
			cursor = ">"
		}
		
		// Status indicator
		status := lipgloss.NewStyle().Foreground(theme.Success).Render("●")
		
		// Main line
		line := fmt.Sprintf("%s %s %s", cursor, status, repo.Name)
		
		// Additional info
		info := []string{}
		if repo.CurrentBranch != "" {
			info = append(info, "on "+repo.CurrentBranch)
		}
		if repo.HasWorktrees {
			info = append(info, fmt.Sprintf("%d worktrees", repo.WorktreeCount))
		}
		
		if len(info) > 0 {
			infoStyle := lipgloss.NewStyle().Foreground(theme.Muted)
			line += " " + infoStyle.Render("("+strings.Join(info, ", ")+")")
		}
		
		elements = append(elements, line)
		
		// Path
		pathStyle := lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true)
		elements = append(elements, "  "+pathStyle.Render(repo.Path))
		
		if i < len(s.repositories)-1 {
			elements = append(elements, "")
		}
	}
	
	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Muted).
		Italic(true)
	help := helpStyle.Render("↑/↓: Navigate • Enter: Select")
	elements = append(elements, "", help)
	
	return strings.Join(elements, "\n")
}

func (s *RepositorySelectionStep) loadRepositories() {
	s.loaded = true
	repos, err := s.wizard.integration.GetAvailableRepositories()
	if err != nil {
		s.error = err
		return
	}
	s.repositories = repos
}

func (s *RepositorySelectionStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "up", "k":
		if s.selectedIndex > 0 {
			s.selectedIndex--
		}
		
	case "down", "j":
		if s.selectedIndex < len(s.repositories)-1 {
			s.selectedIndex++
		}
		
	case "enter", " ":
		if s.selectedIndex < len(s.repositories) {
			repo := s.repositories[s.selectedIndex]
			data["repository_path"] = repo.Path
			data["repository_name"] = repo.Name
			data["current_branch"] = repo.CurrentBranch
		}
	}
	
	return data, nil, nil
}

func (s *RepositorySelectionStep) Validate(data map[string]interface{}) error {
	if _, ok := data["repository_path"].(string); !ok {
		return fmt.Errorf("please select a repository")
	}
	return nil
}

func (s *RepositorySelectionStep) IsComplete(data map[string]interface{}) bool {
	return s.Validate(data) == nil
}

// BranchSelectionStep handles branch selection and creation
type BranchSelectionStep struct {
	wizard         *WorktreeCreationWizard
	branches       []BranchInfo
	selectedIndex  int
	mode           string // "existing" or "new"
	newBranchName  string
	baseBranch     string
	loaded         bool
	error          error
}

func (s *BranchSelectionStep) Title() string {
	return "Select Branch"
}

func (s *BranchSelectionStep) Description() string {
	return "Choose an existing branch or create a new one"
}

func (s *BranchSelectionStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	if !s.loaded {
		s.loadBranches(data)
	}
	
	if s.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		return errorStyle.Render("Error loading branches: " + s.error.Error())
	}
	
	var elements []string
	
	// Mode selection
	modeStyle := lipgloss.NewStyle().Bold(true)
	elements = append(elements, modeStyle.Render("Branch mode:"))
	
	existingButton := s.renderModeButton("existing", "Existing Branch", theme)
	newButton := s.renderModeButton("new", "New Branch", theme)
	
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, existingButton, "  ", newButton)
	elements = append(elements, buttons)
	elements = append(elements, "")
	
	if s.mode == "existing" {
		elements = append(elements, s.renderBranchList(theme))
	} else if s.mode == "new" {
		elements = append(elements, s.renderNewBranchForm(theme, width))
	}
	
	return strings.Join(elements, "\n")
}

func (s *BranchSelectionStep) renderModeButton(buttonMode, label string, theme modals.Theme) string {
	style := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder())
	
	if s.mode == buttonMode {
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
	
	return style.Render(label)
}

func (s *BranchSelectionStep) renderBranchList(theme modals.Theme) string {
	if len(s.branches) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("No branches found")
	}
	
	var items []string
	for i, branch := range s.branches {
		cursor := " "
		if i == s.selectedIndex {
			cursor = ">"
		}
		
		// Branch type indicator
		indicator := ""
		if branch.Current {
			indicator = lipgloss.NewStyle().Foreground(theme.Success).Render("● ")
		} else if branch.Remote {
			indicator = lipgloss.NewStyle().Foreground(theme.Accent).Render("○ ")
		} else {
			indicator = "  "
		}
		
		line := fmt.Sprintf("%s %s%s", cursor, indicator, branch.Name)
		
		// Additional info
		if branch.LastCommit != "" {
			infoStyle := lipgloss.NewStyle().Foreground(theme.Muted)
			line += " " + infoStyle.Render("- "+branch.LastCommit)
		}
		
		items = append(items, line)
	}
	
	content := strings.Join(items, "\n")
	
	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Muted).
		Italic(true)
	help := helpStyle.Render("↑/↓: Navigate • Enter: Select • Tab: Switch mode")
	
	return lipgloss.JoinVertical(lipgloss.Left, content, "", help)
}

func (s *BranchSelectionStep) renderNewBranchForm(theme modals.Theme, width int) string {
	var elements []string
	
	// New branch name
	nameLabel := lipgloss.NewStyle().Bold(true).Render("New branch name:")
	elements = append(elements, nameLabel)
	
	nameStyle := lipgloss.NewStyle().
		Width(width - 12).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(0, 1)
	
	nameField := nameStyle.Render(s.newBranchName + "│")
	elements = append(elements, nameField)
	elements = append(elements, "")
	
	// Base branch selection
	baseLabel := lipgloss.NewStyle().Bold(true).Render("Base branch:")
	elements = append(elements, baseLabel)
	
	if s.baseBranch == "" && len(s.branches) > 0 {
		s.baseBranch = s.branches[0].Name
	}
	
	baseStyle := lipgloss.NewStyle().
		Width(width - 12).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Muted).
		Padding(0, 1)
	
	baseField := baseStyle.Render(s.baseBranch)
	elements = append(elements, baseField)
	
	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Muted).
		Italic(true)
	help := helpStyle.Render("Type: Enter branch name • Tab: Switch mode")
	elements = append(elements, "", help)
	
	return strings.Join(elements, "\n")
}

func (s *BranchSelectionStep) loadBranches(data map[string]interface{}) {
	s.loaded = true
	
	repoPath, ok := data["repository_path"].(string)
	if !ok {
		s.error = fmt.Errorf("repository not selected")
		return
	}
	
	branches, err := s.wizard.integration.GetBranches(repoPath)
	if err != nil {
		s.error = err
		return
	}
	
	s.branches = branches
	
	// Set default mode
	if len(branches) > 0 {
		s.mode = "existing"
		
		// Set default base branch for new branches
		for _, branch := range branches {
			if branch.Current {
				s.baseBranch = branch.Name
				break
			}
		}
		if s.baseBranch == "" {
			s.baseBranch = branches[0].Name
		}
	} else {
		s.mode = "new"
	}
}

func (s *BranchSelectionStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case "tab":
		if s.mode == "existing" {
			s.mode = "new"
		} else {
			s.mode = "existing"
		}
		
	case "up", "k":
		if s.mode == "existing" && s.selectedIndex > 0 {
			s.selectedIndex--
		}
		
	case "down", "j":
		if s.mode == "existing" && s.selectedIndex < len(s.branches)-1 {
			s.selectedIndex++
		}
		
	case "backspace":
		if s.mode == "new" && len(s.newBranchName) > 0 {
			s.newBranchName = s.newBranchName[:len(s.newBranchName)-1]
		}
		
	case "enter", " ":
		if s.mode == "existing" && s.selectedIndex < len(s.branches) {
			branch := s.branches[s.selectedIndex]
			data["branch_name"] = branch.Name
			data["new_branch"] = false
			data["track_remote"] = branch.Remote
		} else if s.mode == "new" && s.newBranchName != "" {
			data["branch_name"] = s.newBranchName
			data["base_branch"] = s.baseBranch
			data["new_branch"] = true
			data["track_remote"] = false
		}
		
	default:
		if s.mode == "new" && len(msg.Runes) > 0 && len(s.newBranchName) < 50 {
			char := string(msg.Runes[0])
			// Basic branch name validation
			if (char >= "a" && char <= "z") || (char >= "A" && char <= "Z") || 
			   (char >= "0" && char <= "9") || char == "-" || char == "_" || char == "/" {
				s.newBranchName += char
			}
		}
	}
	
	return data, nil, nil
}

func (s *BranchSelectionStep) Validate(data map[string]interface{}) error {
	branchName, ok := data["branch_name"].(string)
	if !ok || branchName == "" {
		return fmt.Errorf("please select or enter a branch name")
	}
	
	if newBranch, ok := data["new_branch"].(bool); ok && newBranch {
		if err := s.wizard.integration.ValidateBranchName(branchName); err != nil {
			return err
		}
	}
	
	return nil
}

func (s *BranchSelectionStep) IsComplete(data map[string]interface{}) bool {
	return s.Validate(data) == nil
}

// WorktreePathStep handles worktree directory selection
type WorktreePathStep struct {
	wizard       *WorktreeCreationWizard
	path         string
	defaultPath  string
	pathLoaded   bool
	createSession bool
	sessionName  string
}

func (s *WorktreePathStep) Title() string {
	return "Worktree Path"
}

func (s *WorktreePathStep) Description() string {
	return "Choose the directory for your new worktree"
}

func (s *WorktreePathStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	if !s.pathLoaded {
		s.loadDefaultPath(data)
	}
	
	var elements []string
	
	// Path input
	pathLabel := lipgloss.NewStyle().Bold(true).Render("Worktree directory:")
	elements = append(elements, pathLabel)
	
	pathStyle := lipgloss.NewStyle().
		Width(width - 8).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(0, 1)
	
	displayPath := s.path
	if displayPath == "" {
		displayPath = s.defaultPath
	}
	
	pathField := pathStyle.Render(displayPath + "│")
	elements = append(elements, pathField)
	elements = append(elements, "")
	
	// Session creation option
	sessionLabel := lipgloss.NewStyle().Bold(true).Render("Session options:")
	elements = append(elements, sessionLabel)
	
	checkBox := "☐"
	if s.createSession {
		checkBox = "☑"
	}
	
	checkStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true)
	
	checkbox := lipgloss.JoinHorizontal(lipgloss.Left,
		checkStyle.Render(checkBox), " Create session for this worktree")
	elements = append(elements, checkbox)
	
	if s.createSession {
		elements = append(elements, "")
		
		sessionNameLabel := lipgloss.NewStyle().Bold(true).Render("Session name:")
		elements = append(elements, sessionNameLabel)
		
		sessionStyle := lipgloss.NewStyle().
			Width(width - 8).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Muted).
			Padding(0, 1)
		
		sessionField := sessionStyle.Render(s.sessionName + "│")
		elements = append(elements, sessionField)
	}
	
	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Muted).
		Italic(true)
	help := helpStyle.Render("Type: Edit path • Space: Toggle session creation")
	elements = append(elements, "", help)
	
	return strings.Join(elements, "\n")
}

func (s *WorktreePathStep) loadDefaultPath(data map[string]interface{}) {
	s.pathLoaded = true
	
	repoPath, _ := data["repository_path"].(string)
	branchName, _ := data["branch_name"].(string)
	
	if repoPath != "" {
		if defaultDir, err := s.wizard.integration.GetDefaultWorktreeDir(repoPath); err == nil {
			if branchName != "" {
				s.defaultPath = filepath.Join(defaultDir, branchName)
			} else {
				s.defaultPath = defaultDir
			}
		}
	}
	
	// Set default session name
	if branchName != "" {
		s.sessionName = branchName
	}
}

func (s *WorktreePathStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	switch msg.String() {
	case " ":
		s.createSession = !s.createSession
		
	case "backspace":
		if len(s.path) > 0 {
			s.path = s.path[:len(s.path)-1]
		}
		
	default:
		if len(msg.Runes) > 0 && len(s.path) < 200 {
			s.path += string(msg.Runes[0])
		}
	}
	
	// Store in data
	finalPath := s.path
	if finalPath == "" {
		finalPath = s.defaultPath
	}
	
	data["worktree_path"] = finalPath
	data["create_session"] = s.createSession
	data["session_name"] = s.sessionName
	
	return data, nil, nil
}

func (s *WorktreePathStep) Validate(data map[string]interface{}) error {
	path, ok := data["worktree_path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("worktree path is required")
	}
	
	if err := s.wizard.integration.ValidateWorktreePath(path); err != nil {
		return err
	}
	
	return nil
}

func (s *WorktreePathStep) IsComplete(data map[string]interface{}) bool {
	return s.Validate(data) == nil
}

// WorktreeConfirmationStep shows summary and confirms creation
type WorktreeConfirmationStep struct {
	wizard *WorktreeCreationWizard
}

func (s *WorktreeConfirmationStep) Title() string {
	return "Confirmation"
}

func (s *WorktreeConfirmationStep) Description() string {
	return "Review and confirm your worktree configuration"
}

func (s *WorktreeConfirmationStep) Render(theme modals.Theme, width int, data map[string]interface{}) string {
	var elements []string
	
	summaryStyle := lipgloss.NewStyle().Bold(true)
	elements = append(elements, summaryStyle.Render("Worktree Summary:"))
	elements = append(elements, "")
	
	// Repository
	if repoName, ok := data["repository_name"].(string); ok {
		elements = append(elements, fmt.Sprintf("Repository: %s", repoName))
	}
	
	// Branch
	if branchName, ok := data["branch_name"].(string); ok {
		branchLine := fmt.Sprintf("Branch: %s", branchName)
		if newBranch, ok := data["new_branch"].(bool); ok && newBranch {
			branchLine += " (new)"
			if baseBranch, ok := data["base_branch"].(string); ok {
				branchLine += fmt.Sprintf(" from %s", baseBranch)
			}
		}
		elements = append(elements, branchLine)
	}
	
	// Path
	if path, ok := data["worktree_path"].(string); ok {
		elements = append(elements, fmt.Sprintf("Path: %s", path))
	}
	
	// Session
	if createSession, ok := data["create_session"].(bool); ok && createSession {
		sessionLine := "Session: Will be created"
		if sessionName, ok := data["session_name"].(string); ok && sessionName != "" {
			sessionLine = fmt.Sprintf("Session: %s (will be created)", sessionName)
		}
		elements = append(elements, sessionLine)
	}
	
	elements = append(elements, "")
	
	// Confirmation message
	confirmStyle := lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true)
	elements = append(elements, confirmStyle.Render("Press Ctrl+Enter to create worktree"))
	
	return strings.Join(elements, "\n")
}

func (s *WorktreeConfirmationStep) HandleKey(msg tea.KeyMsg, data map[string]interface{}) (map[string]interface{}, tea.Cmd, error) {
	return data, nil, nil
}

func (s *WorktreeConfirmationStep) Validate(data map[string]interface{}) error {
	return nil
}

func (s *WorktreeConfirmationStep) IsComplete(data map[string]interface{}) bool {
	return true
}