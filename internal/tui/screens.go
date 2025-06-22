package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// Screen interface that all screens must implement
type Screen interface {
	tea.Model
	Title() string
	Help() []string
}

// DashboardModel represents the main dashboard screen
type DashboardModel struct {
	integration *Integration
	theme       Theme
	width       int
	height      int
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(integration *Integration, theme Theme) *DashboardModel {
	return &DashboardModel{
		integration: integration,
		theme:       theme,
	}
}

func (m *DashboardModel) Init() tea.Cmd {
	return nil
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			// New session - handled by app level, just pass through
			return m, nil
		case "w":
			// New worktree - handled by app level, just pass through
			return m, nil
		case "r":
			// Refresh data
			return m, func() tea.Msg {
				return RefreshDataMsg{}
			}
		case "c":
			// Configuration - handled by app level (key "4"), just pass through
			return m, nil
		}
	case RefreshDataMsg:
		// Refresh dashboard data
		return m, nil
	}
	return m, nil
}

func (m *DashboardModel) View() string {
	if m.width == 0 {
		return "Loading dashboard..."
	}

	// Get system status
	status := m.integration.GetSystemStatus()
	
	// Create dashboard sections
	header := m.theme.HeaderStyle.Render("🚀 CCMGR Ultra Dashboard")
	
	// System overview
	overview := m.renderSystemOverview(status)
	
	// Active sessions
	sessions := m.renderActiveSessions()
	
	// Recent worktrees
	worktrees := m.renderRecentWorktrees()
	
	// Quick actions
	actions := m.renderQuickActions()

	// Layout sections
	sections := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		overview,
		"",
		sessions,
		"",
		worktrees,
		"",
		actions,
	)

	return m.theme.ContentStyle.Render(sections)
}

func (m *DashboardModel) Title() string {
	return "Dashboard"
}

func (m *DashboardModel) Help() []string {
	return []string{
		"1-4: Switch screens",
		"?: Help",
		"q: Quit",
	}
}

func (m *DashboardModel) renderSystemOverview(status SystemStatus) string {
	title := m.theme.TitleStyle.Render("📊 System Overview")
	
	content := fmt.Sprintf(
		"Claude Processes: %d active\n"+
		"Tmux Sessions: %d running\n"+
		"Git Worktrees: %d tracked\n"+
		"Last Updated: %s",
		status.ActiveProcesses,
		status.ActiveSessions,
		status.TrackedWorktrees,
		status.LastUpdate.Format("15:04:05"),
	)
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

func (m *DashboardModel) renderActiveSessions() string {
	title := m.theme.TitleStyle.Render("🖥️  Active Sessions")
	
	sessions := m.integration.GetActiveSessions()
	if len(sessions) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			m.theme.ContentStyle.Render("No active sessions"),
		)
	}
	
	var sessionLines []string
	for _, session := range sessions {
		line := fmt.Sprintf("• %s (%s) - %s",
			session.Name,
			session.Project,
			session.Branch,
		)
		sessionLines = append(sessionLines, line)
	}
	
	content := strings.Join(sessionLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

func (m *DashboardModel) renderRecentWorktrees() string {
	title := m.theme.TitleStyle.Render("🌳 Recent Worktrees")
	
	worktrees := m.integration.GetRecentWorktrees()
	if len(worktrees) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			m.theme.ContentStyle.Render("No recent worktrees"),
		)
	}
	
	var worktreeLines []string
	for _, wt := range worktrees {
		line := fmt.Sprintf("• %s (%s) - %s",
			wt.Path,
			wt.Branch,
			wt.LastAccess.Format("Jan 2 15:04"),
		)
		worktreeLines = append(worktreeLines, line)
	}
	
	content := strings.Join(worktreeLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

func (m *DashboardModel) renderQuickActions() string {
	title := m.theme.TitleStyle.Render("⚡ Quick Actions")
	
	actions := []string{
		"n: New session",
		"w: New worktree",  
		"r: Refresh data",
		"c: Configuration",
	}
	
	content := strings.Join(actions, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.theme.ContentStyle.Render(content),
	)
}

// SessionsModel represents the sessions management screen
type SessionsModel struct {
	integration *Integration
	theme       Theme
	width       int
	height      int
	cursor      int
	sessions    []SessionInfo
}

func NewSessionsModel(integration *Integration, theme Theme) *SessionsModel {
	return &SessionsModel{
		integration: integration,
		theme:       theme,
	}
}

func (m *SessionsModel) Init() tea.Cmd {
	return nil
}

func (m *SessionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "enter":
			// Attach to selected session
			if m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				return m, m.integration.AttachSession(session.ID)
			}
		}
	case RefreshDataMsg:
		m.sessions = m.integration.GetAllSessions()
	}
	return m, nil
}

func (m *SessionsModel) View() string {
	if m.width == 0 {
		return "Loading sessions..."
	}

	header := m.theme.HeaderStyle.Render("🖥️  Session Management")
	
	if len(m.sessions) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			m.theme.ContentStyle.Render("No sessions found"),
		)
	}

	var sessionLines []string
	for i, session := range m.sessions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		
		status := "●"
		statusColor := m.theme.Success
		if !session.Active {
			status = "○"
			statusColor = m.theme.Muted
		}
		
		line := fmt.Sprintf("%s %s %s (%s) - %s - %s",
			cursor,
			lipgloss.NewStyle().Foreground(statusColor).Render(status),
			session.Name,
			session.Project,
			session.Branch,
			session.Directory,
		)
		sessionLines = append(sessionLines, line)
	}
	
	content := strings.Join(sessionLines, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *SessionsModel) Title() string {
	return "Sessions"
}

func (m *SessionsModel) Help() []string {
	return []string{
		"↑/k: Move up",
		"↓/j: Move down",
		"Enter: Attach session",
		"n: New session",
	}
}

// WorktreesModel represents the worktrees management screen
type WorktreesModel struct {
	integration    *Integration
	theme          Theme
	width          int
	height         int
	cursor         int
	worktrees      []WorktreeInfo
	selectedItems  map[int]bool        // New: multi-selection state
	selectionMode  bool                // New: toggle selection mode
	filterText     string              // New: search filter
	sortMode       WorktreeSortMode    // New: sorting mode
	claudeStatuses map[string]ClaudeStatus // New: status tracking
	filteredIndices []int              // New: indices after filtering
	searchMode     bool                // New: search input mode
}

func NewWorktreesModel(integration *Integration, theme Theme) *WorktreesModel {
	return &WorktreesModel{
		integration:     integration,
		theme:           theme,
		selectedItems:   make(map[int]bool),
		selectionMode:   false,
		filterText:      "",
		sortMode:        SortByLastAccess,
		claudeStatuses:  make(map[string]ClaudeStatus),
		filteredIndices: []int{},
		searchMode:      false,
	}
}

func (m *WorktreesModel) Init() tea.Cmd {
	// Initialize with current worktrees and refresh data
	m.refreshWorktreeData()
	// Start real-time status updates
	return m.integration.StartRealtimeStatusUpdates()
}

// New methods for enhanced worktree functionality

// toggleSelectionMode toggles between single and multi-selection mode
func (m *WorktreesModel) toggleSelectionMode() {
	m.selectionMode = !m.selectionMode
	if !m.selectionMode {
		// Clear all selections when exiting selection mode
		m.selectedItems = make(map[int]bool)
	}
}

// toggleItemSelection toggles selection state of item at given index
func (m *WorktreesModel) toggleItemSelection(index int) {
	if !m.selectionMode {
		return
	}
	
	if len(m.filteredIndices) > 0 && index < len(m.filteredIndices) {
		realIndex := m.filteredIndices[index]
		m.selectedItems[realIndex] = !m.selectedItems[realIndex]
	} else if index < len(m.worktrees) {
		m.selectedItems[index] = !m.selectedItems[index]
	}
}

// toggleSelectAll selects or deselects all visible items
func (m *WorktreesModel) toggleSelectAll() {
	if !m.selectionMode {
		return
	}
	
	// Check if all visible items are selected
	allSelected := true
	indices := m.getVisibleIndices()
	
	for _, idx := range indices {
		if !m.selectedItems[idx] {
			allSelected = false
			break
		}
	}
	
	// Toggle: if all selected, deselect all; otherwise select all
	for _, idx := range indices {
		m.selectedItems[idx] = !allSelected
	}
}

// getSelectedWorktrees returns currently selected worktrees
func (m *WorktreesModel) getSelectedWorktrees() []WorktreeInfo {
	var selected []WorktreeInfo
	for idx, isSelected := range m.selectedItems {
		if isSelected && idx < len(m.worktrees) {
			selected = append(selected, m.worktrees[idx])
		}
	}
	return selected
}

// getCurrentWorktree returns the worktree at cursor position
func (m *WorktreesModel) getCurrentWorktree() *WorktreeInfo {
	indices := m.getVisibleIndices()
	if m.cursor < len(indices) {
		realIndex := indices[m.cursor]
		if realIndex < len(m.worktrees) {
			return &m.worktrees[realIndex]
		}
	}
	return nil
}

// getVisibleIndices returns indices of currently visible worktrees
func (m *WorktreesModel) getVisibleIndices() []int {
	if len(m.filteredIndices) > 0 {
		return m.filteredIndices
	}
	
	indices := make([]int, len(m.worktrees))
	for i := range indices {
		indices[i] = i
	}
	return indices
}

// applyFilter filters worktrees based on current filter text
func (m *WorktreesModel) applyFilter() {
	m.filteredIndices = []int{}
	
	if m.filterText == "" {
		// No filter, show all
		return
	}
	
	filterLower := strings.ToLower(m.filterText)
	for i, wt := range m.worktrees {
		// Search in path, branch name, and repository
		if strings.Contains(strings.ToLower(wt.Path), filterLower) ||
		   strings.Contains(strings.ToLower(wt.Branch), filterLower) ||
		   strings.Contains(strings.ToLower(wt.Repository), filterLower) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}
	
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.filteredIndices) {
		m.cursor = len(m.filteredIndices) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// sortWorktrees sorts the worktree list according to current sort mode
func (m *WorktreesModel) sortWorktrees() {
	switch m.sortMode {
	case SortByName:
		// Sort by path (name)
		for i := 0; i < len(m.worktrees)-1; i++ {
			for j := i + 1; j < len(m.worktrees); j++ {
				if strings.ToLower(m.worktrees[i].Path) > strings.ToLower(m.worktrees[j].Path) {
					m.worktrees[i], m.worktrees[j] = m.worktrees[j], m.worktrees[i]
				}
			}
		}
	case SortByLastAccess:
		// Sort by last access time (most recent first)
		for i := 0; i < len(m.worktrees)-1; i++ {
			for j := i + 1; j < len(m.worktrees); j++ {
				if m.worktrees[i].LastAccess.Before(m.worktrees[j].LastAccess) {
					m.worktrees[i], m.worktrees[j] = m.worktrees[j], m.worktrees[i]
				}
			}
		}
	case SortByBranch:
		// Sort by branch name
		for i := 0; i < len(m.worktrees)-1; i++ {
			for j := i + 1; j < len(m.worktrees); j++ {
				if strings.ToLower(m.worktrees[i].Branch) > strings.ToLower(m.worktrees[j].Branch) {
					m.worktrees[i], m.worktrees[j] = m.worktrees[j], m.worktrees[i]
				}
			}
		}
	case SortByStatus:
		// Sort by Claude status and git status
		for i := 0; i < len(m.worktrees)-1; i++ {
			for j := i + 1; j < len(m.worktrees); j++ {
				// Primary sort: Claude status (busy > idle > error)
				statusPriority := func(status string) int {
					switch status {
					case "busy": return 3
					case "idle": return 2
					case "waiting": return 1
					default: return 0
					}
				}
				
				iPriority := statusPriority(m.worktrees[i].ClaudeStatus.State)
				jPriority := statusPriority(m.worktrees[j].ClaudeStatus.State)
				
				if iPriority < jPriority {
					m.worktrees[i], m.worktrees[j] = m.worktrees[j], m.worktrees[i]
				}
			}
		}
	}
}

// cycleSortMode cycles through available sort modes
func (m *WorktreesModel) cycleSortMode() {
	m.sortMode = (m.sortMode + 1) % 4
	m.sortWorktrees()
	m.applyFilter() // Reapply filter after sorting
}

// refreshWorktreeData refreshes worktree data and applies current sorting/filtering
func (m *WorktreesModel) refreshWorktreeData() {
	m.worktrees = m.integration.GetAllWorktrees()
	m.refreshClaudeStatuses()
	m.sortWorktrees()
	m.applyFilter()
}

// refreshClaudeStatuses updates Claude status information for all worktrees
func (m *WorktreesModel) refreshClaudeStatuses() {
	// This would query the actual Claude processes for each worktree
	// For now, we'll update the statuses in the worktree info directly
	for i := range m.worktrees {
		// In a real implementation, this would check for Claude processes
		// running in each worktree directory
		if m.worktrees[i].ClaudeStatus.State == "" {
			m.worktrees[i].ClaudeStatus = ClaudeStatus{
				State:      "idle",
				ProcessID:  0,
				LastUpdate: time.Now(),
				SessionID:  "",
			}
		}
	}
}

// enterSearchMode enables search input mode
func (m *WorktreesModel) enterSearchMode() {
	m.searchMode = true
}

// exitSearchMode disables search input mode
func (m *WorktreesModel) exitSearchMode() {
	m.searchMode = false
}

// handleSearchInput processes search input characters
func (m *WorktreesModel) handleSearchInput(char string) {
	m.filterText += char
	m.applyFilter()
}

// clearSearch clears the current search filter
func (m *WorktreesModel) clearSearch() {
	m.filterText = ""
	m.filteredIndices = []int{}
	m.cursor = 0
}

func (m *WorktreesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		// Handle search mode input
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.exitSearchMode()
			case "enter":
				m.exitSearchMode()
			case "backspace":
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.applyFilter()
				}
			case "ctrl+c":
				m.clearSearch()
				m.exitSearchMode()
			default:
				// Add character to search
				if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
					m.handleSearchInput(msg.String())
				}
			}
			return m, nil
		}
		
		// Normal mode keyboard handling
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			indices := m.getVisibleIndices()
			if m.cursor < len(indices)-1 {
				m.cursor++
			}
		case "enter":
			// Open selected worktree
			if wt := m.getCurrentWorktree(); wt != nil {
				return m, m.integration.OpenWorktree(wt.Path)
			}
		case "n":
			// New session for current/selected worktrees
			return m, m.createNewSessionForSelection()
		case "c":
			// Continue session for current/selected worktrees  
			return m, m.continueSessionForSelection()
		case "r":
			// Resume session for current/selected worktrees
			return m, m.resumeSessionForSelection()
		case " ":
			// Toggle selection of current item (space bar)
			m.toggleItemSelection(m.cursor)
		case "a":
			// Select all / deselect all
			m.toggleSelectAll()
		case "/":
			// Enter search/filter mode
			m.enterSearchMode()
		case "s":
			// Cycle through sort modes
			m.cycleSortMode()
		case "tab":
			// Toggle selection mode
			m.toggleSelectionMode()
		case "esc":
			// Clear search filter or exit selection mode
			if m.filterText != "" {
				m.clearSearch()
			} else if m.selectionMode {
				m.toggleSelectionMode()
			}
		}
	case RefreshDataMsg:
		m.refreshWorktreeData()
	case RealtimeStatusUpdateMsg:
		// Process real-time status update
		return m, m.integration.ProcessRealtimeStatusUpdate()
	case StatusUpdatedMsg:
		// Refresh worktree data when status is updated
		m.refreshWorktreeData()
		// Schedule next update
		return m, m.integration.StartRealtimeStatusUpdates()
	}
	return m, nil
}

// Session workflow commands (to be implemented in step 3)

func (m *WorktreesModel) createNewSessionForSelection() tea.Cmd {
	return func() tea.Msg {
		// Get selected worktrees or current worktree
		worktrees := m.getSelectedWorktrees()
		if len(worktrees) == 0 {
			if wt := m.getCurrentWorktree(); wt != nil {
				worktrees = []WorktreeInfo{*wt}
			}
		}
		
		// For now, just return a placeholder message
		// This will be properly implemented in step 3
		return NewSessionRequestedMsg{Worktrees: worktrees}
	}
}

func (m *WorktreesModel) continueSessionForSelection() tea.Cmd {
	return func() tea.Msg {
		// Get selected worktrees or current worktree
		worktrees := m.getSelectedWorktrees()
		if len(worktrees) == 0 {
			if wt := m.getCurrentWorktree(); wt != nil {
				worktrees = []WorktreeInfo{*wt}
			}
		}
		
		// For now, just return a placeholder message
		return ContinueSessionRequestedMsg{Worktrees: worktrees}
	}
}

func (m *WorktreesModel) resumeSessionForSelection() tea.Cmd {
	return func() tea.Msg {
		// Get selected worktrees or current worktree
		worktrees := m.getSelectedWorktrees()
		if len(worktrees) == 0 {
			if wt := m.getCurrentWorktree(); wt != nil {
				worktrees = []WorktreeInfo{*wt}
			}
		}
		
		// For now, just return a placeholder message
		return ResumeSessionRequestedMsg{Worktrees: worktrees}
	}
}

func (m *WorktreesModel) View() string {
	if m.width == 0 {
		return "Loading worktrees..."
	}

	// Build header with mode indicators
	headerText := "🌳 Worktree Selection"
	if m.selectionMode {
		selectedCount := len(m.getSelectedWorktrees())
		headerText += fmt.Sprintf(" [MULTI-SELECT: %d selected]", selectedCount)
	}
	if m.filterText != "" {
		headerText += fmt.Sprintf(" [FILTER: %s]", m.filterText)
	}
	
	// Add sort mode indicator
	sortNames := []string{"Name", "Last Access", "Branch", "Status"}
	headerText += fmt.Sprintf(" [SORT: %s]", sortNames[m.sortMode])
	
	header := m.theme.HeaderStyle.Render(headerText)
	
	// Get visible worktrees
	indices := m.getVisibleIndices()
	if len(indices) == 0 {
		noResults := "No worktrees found"
		if m.filterText != "" {
			noResults = fmt.Sprintf("No worktrees match filter: %s", m.filterText)
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			m.theme.ContentStyle.Render(noResults),
		)
	}

	var worktreeLines []string
	for i, idx := range indices {
		wt := m.worktrees[idx]
		
		// Cursor indicator
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		
		// Selection indicator (checkbox style)
		selection := " "
		if m.selectionMode {
			if m.selectedItems[idx] {
				selection = "✓"
			} else {
				selection = "☐"
			}
		}
		
		// Claude status indicator
		statusIcon := "○"
		statusColor := m.theme.Muted
		switch wt.ClaudeStatus.State {
		case "busy":
			statusIcon = "●"
			statusColor = m.theme.Warning
		case "idle":
			statusIcon = "●"
			statusColor = m.theme.Success
		case "waiting":
			statusIcon = "◐"
			statusColor = m.theme.Info
		case "error":
			statusIcon = "✗"
			statusColor = m.theme.Error
		}
		
		// Session count indicator
		sessionCount := len(wt.ActiveSessions)
		sessionIndicator := ""
		if sessionCount > 0 {
			sessionIndicator = fmt.Sprintf(" [%d]", sessionCount)
		}
		
		// Git status indicator
		gitIndicator := ""
		if !wt.GitStatus.IsClean {
			changes := wt.GitStatus.Modified + wt.GitStatus.Staged + wt.GitStatus.Untracked
			if changes > 0 {
				gitIndicator = fmt.Sprintf(" +%d", changes)
			}
		}
		if wt.GitStatus.Ahead > 0 || wt.GitStatus.Behind > 0 {
			gitIndicator += fmt.Sprintf(" ↑%d↓%d", wt.GitStatus.Ahead, wt.GitStatus.Behind)
		}
		
		// Format the line
		line := fmt.Sprintf("%s%s %s %s (%s)%s%s - %s",
			cursor,
			selection,
			lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon),
			wt.Path,
			wt.Branch,
			sessionIndicator,
			gitIndicator,
			wt.LastAccess.Format("Jan 2 15:04"),
		)
		
		// Apply highlighting for current item
		if i == m.cursor {
			line = m.theme.SelectedStyle.Render(line)
		}
		
		worktreeLines = append(worktreeLines, line)
	}
	
	content := strings.Join(worktreeLines, "\n")
	
	// Build status/help bar
	var statusParts []string
	
	if m.searchMode {
		statusParts = append(statusParts, fmt.Sprintf("Search: %s|", m.filterText))
		statusParts = append(statusParts, "Enter/Esc: Exit search")
	} else {
		// Show current mode and available actions
		if m.selectionMode {
			statusParts = append(statusParts, "Multi-select mode")
		}
		
		// Key shortcuts
		shortcuts := []string{
			"n:New", "c:Continue", "r:Resume",
		}
		if !m.selectionMode {
			shortcuts = append(shortcuts, "Space:Select", "Tab:Multi-mode")
		}
		shortcuts = append(shortcuts, "/:Search", "s:Sort")
		
		statusParts = append(statusParts, strings.Join(shortcuts, " "))
	}
	
	statusBar := m.theme.StatusStyle.Render(strings.Join(statusParts, " | "))
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
		"",
		statusBar,
	)
}

func (m *WorktreesModel) Title() string {
	return "Worktrees"
}

func (m *WorktreesModel) Help() []string {
	if m.searchMode {
		return []string{
			"Type to search",
			"Enter/Esc: Exit search",
			"Backspace: Delete character",
			"Ctrl+C: Clear and exit",
		}
	}
	
	helpItems := []string{
		"↑/k, ↓/j: Navigate",
		"Enter: Open worktree",
		"n: New session",
		"c: Continue session",
		"r: Resume session",
	}
	
	if m.selectionMode {
		helpItems = append(helpItems, []string{
			"Space: Toggle selection",
			"a: Select/deselect all",
			"Tab: Exit multi-select",
		}...)
	} else {
		helpItems = append(helpItems, []string{
			"Tab: Multi-select mode",
			"Space: Quick select",
		}...)
	}
	
	helpItems = append(helpItems, []string{
		"/: Search/filter",
		"s: Cycle sort mode",
		"Esc: Clear filter/exit mode",
	}...)
	
	return helpItems
}

// ConfigModel represents the configuration screen
type ConfigModel struct {
	config *config.Config
	theme  Theme
	width  int
	height int
}

func NewConfigModel(config *config.Config, theme Theme) *ConfigModel {
	return &ConfigModel{
		config: config,
		theme:  theme,
	}
}

func (m *ConfigModel) Init() tea.Cmd {
	return nil
}

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *ConfigModel) View() string {
	if m.width == 0 {
		return "Loading configuration..."
	}

	header := m.theme.HeaderStyle.Render("⚙️  Configuration")
	
	content := fmt.Sprintf(
		"Config File: %s\n"+
		"Log Level: %s\n"+
		"Claude Enabled: %t\n"+
		"TUI Theme: %s\n"+
		"Auto Refresh: %ds",
		m.config.ConfigFile,
		m.config.LogLevel,
		m.config.Claude.Enabled,
		m.config.TUI.Theme,
		m.config.RefreshInterval,
	)
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *ConfigModel) Title() string {
	return "Configuration"
}

func (m *ConfigModel) Help() []string {
	return []string{
		"e: Edit config",
		"r: Reload config",
	}
}

// HelpModel represents the help screen
type HelpModel struct {
	theme  Theme
	width  int
	height int
}

func NewHelpModel(theme Theme) *HelpModel {
	return &HelpModel{
		theme: theme,
	}
}

func (m *HelpModel) Init() tea.Cmd {
	return nil
}

func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *HelpModel) View() string {
	if m.width == 0 {
		return "Loading help..."
	}

	header := m.theme.HeaderStyle.Render("❓ Help & Keyboard Shortcuts")
	
	sections := []string{
		m.theme.TitleStyle.Render("Global Navigation:"),
		"1: Dashboard",
		"2: Sessions",
		"3: Worktrees", 
		"4: Configuration",
		"?: Help (this screen)",
		"q, Ctrl+C: Quit",
		"",
		m.theme.TitleStyle.Render("Dashboard:"),
		"r: Refresh data",
		"n: New session",
		"w: New worktree",
		"",
		m.theme.TitleStyle.Render("Sessions:"),
		"↑/k, ↓/j: Navigate",
		"Enter: Attach to session",
		"n: Create new session",
		"d: Delete session", 
		"",
		m.theme.TitleStyle.Render("Worktrees:"),
		"↑/k, ↓/j: Navigate",
		"Enter: Open worktree",
		"n: Create new worktree",
		"d: Delete worktree",
	}
	
	content := strings.Join(sections, "\n")
	
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.theme.ContentStyle.Render(content),
	)
}

func (m *HelpModel) Title() string {
	return "Help"
}

func (m *HelpModel) Help() []string {
	return []string{
		"Navigate with numbers 1-4",
		"q: Return to previous screen",
	}
}