package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

func TestNewAppModel(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, ScreenDashboard, app.currentScreen)
	assert.NotNil(t, app.integration)
	assert.NotNil(t, app.keyHandler)
	assert.Len(t, app.screens, 5) // dashboard, sessions, worktrees, config, help
}

func TestAppModel_Init(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	cmd := app.Init()
	assert.NotNil(t, cmd)
}

func TestAppModel_Update_WindowSize(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Test window size update
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedApp, cmd := app.Update(msg)
	
	assert.NotNil(t, updatedApp)
	assert.NotNil(t, cmd)
	
	appModel := updatedApp.(*AppModel)
	assert.Equal(t, 100, appModel.width)
	assert.Equal(t, 50, appModel.height)
	assert.True(t, appModel.ready)
}

func TestAppModel_Update_KeyPress_Quit(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Test quit key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedApp, cmd := app.Update(msg)
	
	assert.NotNil(t, updatedApp)
	assert.NotNil(t, cmd)
	
	appModel := updatedApp.(*AppModel)
	assert.True(t, appModel.quitting)
}

func TestAppModel_Update_KeyPress_Navigation(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Test navigation to sessions screen
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	updatedApp, cmd := app.Update(msg)
	
	assert.NotNil(t, updatedApp)
	assert.NotNil(t, cmd)
	
	appModel := updatedApp.(*AppModel)
	assert.Equal(t, ScreenSessions, appModel.currentScreen)
}

func TestAppModel_SwitchScreen(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Test switching to different screens
	screens := []AppScreen{
		ScreenDashboard,
		ScreenSessions,
		ScreenWorktrees,
		ScreenConfig,
		ScreenHelp,
	}
	
	for _, screen := range screens {
		updatedApp, cmd := app.switchScreen(screen)
		assert.NotNil(t, updatedApp)
		assert.NotNil(t, cmd)
		
		appModel := updatedApp.(*AppModel)
		assert.Equal(t, screen, appModel.currentScreen)
	}
}

func TestAppModel_View_NotReady(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Test view when not ready
	view := app.View()
	assert.Contains(t, view, "Initializing CCMGR Ultra")
}

func TestAppModel_View_Quitting(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Set ready and quitting
	app.ready = true
	app.quitting = true
	
	view := app.View()
	assert.Contains(t, view, "Thanks for using CCMGR Ultra")
}

func TestAppModel_View_Ready(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Set ready state and dimensions
	app.ready = true
	app.width = 100
	app.height = 50
	
	view := app.View()
	assert.NotEmpty(t, view)
	assert.NotContains(t, view, "Initializing CCMGR Ultra")
}

func TestAppModel_GetCurrentScreen(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	assert.Equal(t, ScreenDashboard, app.GetCurrentScreen())
	
	app.currentScreen = ScreenSessions
	assert.Equal(t, ScreenSessions, app.GetCurrentScreen())
}

func TestAppModel_GetIntegration(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	integration := app.GetIntegration()
	assert.NotNil(t, integration)
	assert.Same(t, app.integration, integration)
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	
	assert.NotEmpty(t, theme.Primary)
	assert.NotEmpty(t, theme.Secondary)
	assert.NotEmpty(t, theme.Background)
	assert.NotEmpty(t, theme.Text)
	assert.NotNil(t, theme.BorderStyle)
	assert.NotNil(t, theme.TitleStyle)
	assert.NotNil(t, theme.HeaderStyle)
	assert.NotNil(t, theme.ContentStyle)
	assert.NotNil(t, theme.FooterStyle)
}

func TestAppModel_Update_RefreshData(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Initialize window size first
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := app.Update(windowMsg)
	app = updatedModel.(*AppModel)
	
	// Test refresh data message
	msg := RefreshDataMsg{}
	updatedApp, cmd := app.Update(msg)
	
	assert.NotNil(t, updatedApp)
	assert.NotNil(t, cmd)
}

func TestAppModel_Update_TickMsg(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Initialize window size first  
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := app.Update(windowMsg)
	app = updatedModel.(*AppModel)
	
	// Test tick message
	msg := TickMsg(time.Now())
	updatedApp, cmd := app.Update(msg)
	
	assert.NotNil(t, updatedApp)
	assert.NotNil(t, cmd)
}

func TestAppModel_InitializeScreens(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	
	app, err := NewAppModel(ctx, cfg)
	require.NoError(t, err)
	
	// Check that all screens are initialized
	assert.Contains(t, app.screens, ScreenDashboard)
	assert.Contains(t, app.screens, ScreenSessions)
	assert.Contains(t, app.screens, ScreenWorktrees)
	assert.Contains(t, app.screens, ScreenConfig)
	assert.Contains(t, app.screens, ScreenHelp)
	
	// Check that all screens are not nil
	for screen, model := range app.screens {
		assert.NotNil(t, model, "Screen %v should not be nil", screen)
	}
}