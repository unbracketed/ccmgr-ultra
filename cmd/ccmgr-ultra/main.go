package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/your-username/ccmgr-ultra/internal/config"
	"github.com/your-username/ccmgr-ultra/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "ccmgr-ultra",
	Short: "Claude Multi-Project Multi-Session Manager",
	Long: `ccmgr-ultra is a comprehensive CLI tool for managing Claude Code sessions
across multiple projects and git worktrees. It combines the best features of
CCManager and Claude Squad to provide seamless tmux session management,
status monitoring, and workflow automation.`,
	Run: func(cmd *cobra.Command, args []string) {
		runTUI()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ccmgr-ultra",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ccmgr-ultra %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// runTUI initializes and runs the TUI application
func runTUI() {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create TUI application
	app, err := tui.NewAppModel(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TUI application: %v\n", err)
		os.Exit(1)
	}

	// Configure program options
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}