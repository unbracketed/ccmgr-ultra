package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
		// TODO: Launch TUI
		fmt.Println("ccmgr-ultra - Claude Multi-Project Multi-Session Manager")
		fmt.Println("TUI implementation coming soon...")
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}