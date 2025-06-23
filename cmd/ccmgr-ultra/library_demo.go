package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unbracketed/ccmgr-ultra/pkg/ccmgr"
)

// libraryDemoCmd demonstrates using the ccmgr library instead of direct internal imports
var libraryDemoCmd = &cobra.Command{
	Use:   "library-demo",
	Short: "Demonstrate ccmgr library usage (refactored approach)",
	Long: `This command demonstrates the new library-based approach where CLI commands
use the pkg/ccmgr library instead of directly importing internal modules.`,
	Run: runLibraryDemo,
}

// statusLibraryCmd shows system status using the library
var statusLibraryCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status using library API",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ccmgr.NewClient(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		status := client.System().Status()
		health := client.System().Health()

		fmt.Printf("System Status (via Library API)\n")
		fmt.Printf("================================\n")
		fmt.Printf("Overall Health: %s\n", health.Overall)
		fmt.Printf("Active Sessions: %d\n", status.ActiveSessions)
		fmt.Printf("Active Processes: %d\n", status.ActiveProcesses)
		fmt.Printf("Tracked Worktrees: %d\n", status.TrackedWorktrees)
		fmt.Printf("Last Update: %s\n", status.LastUpdate.Format("2006-01-02 15:04:05"))

		if len(status.Errors) > 0 {
			fmt.Printf("\nErrors:\n")
			for _, err := range status.Errors {
				fmt.Printf("  - %s\n", err)
			}
		}

		fmt.Printf("\nServices:\n")
		for service, serviceStatus := range health.Services {
			fmt.Printf("  - %s: %s\n", service, serviceStatus)
		}
	},
}

// sessionsLibraryCmd lists sessions using the library
var sessionsLibraryCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List sessions using library API",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ccmgr.NewClient(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		sessions, err := client.Sessions().List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list sessions: %v\n", err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found")
			return
		}

		fmt.Printf("Sessions (via Library API)\n")
		fmt.Printf("===========================\n")
		for _, session := range sessions {
			status := "inactive"
			if session.Active {
				status = "active"
			}
			fmt.Printf("ID: %s\n", session.ID)
			fmt.Printf("  Name: %s\n", session.Name)
			fmt.Printf("  Project: %s\n", session.Project)
			fmt.Printf("  Directory: %s\n", session.Directory)
			fmt.Printf("  Status: %s\n", status)
			fmt.Printf("  Created: %s\n", session.Created.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Last Access: %s\n", session.LastAccess.Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
	},
}

// worktreesLibraryCmd lists worktrees using the library
var worktreesLibraryCmd = &cobra.Command{
	Use:   "worktrees",
	Short: "List worktrees using library API",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ccmgr.NewClient(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		worktrees, err := client.Worktrees().Recent()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
			os.Exit(1)
		}

		if len(worktrees) == 0 {
			fmt.Println("No worktrees found")
			return
		}

		fmt.Printf("Recent Worktrees (via Library API)\n")
		fmt.Printf("===================================\n")
		for _, wt := range worktrees {
			fmt.Printf("Path: %s\n", wt.Path)
			fmt.Printf("  Branch: %s\n", wt.Branch)
			fmt.Printf("  Repository: %s\n", wt.Repository)
			fmt.Printf("  Status: %s\n", wt.Status)
			fmt.Printf("  Claude Status: %s\n", wt.ClaudeStatus.State)
			fmt.Printf("  Has Changes: %t\n", wt.HasChanges)
			fmt.Printf("  Last Access: %s\n", wt.LastAccess.Format("2006-01-02 15:04:05"))

			if len(wt.ActiveSessions) > 0 {
				fmt.Printf("  Active Sessions:\n")
				for _, session := range wt.ActiveSessions {
					fmt.Printf("    - %s (%s)\n", session.Name, session.State)
				}
			}
			fmt.Println()
		}
	},
}

func runLibraryDemo(cmd *cobra.Command, args []string) {
	fmt.Println("CCMgr Library Demo")
	fmt.Println("==================")
	fmt.Println("This demonstrates the refactored library-based approach.")
	fmt.Println("Available subcommands:")
	fmt.Println("  status     - Show system status")
	fmt.Println("  sessions   - List sessions")
	fmt.Println("  worktrees  - List worktrees")
	fmt.Println()
	fmt.Println("Usage: ccmgr-ultra library-demo <subcommand>")

	// Demonstrate JSON output capability
	client, err := ccmgr.NewClient(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	status := client.System().Status()
	jsonData, _ := json.MarshalIndent(status, "", "  ")
	fmt.Printf("\nExample JSON Output:\n%s\n", jsonData)
}

func init() {
	// Add the library demo command with subcommands
	libraryDemoCmd.AddCommand(statusLibraryCmd)
	libraryDemoCmd.AddCommand(sessionsLibraryCmd)
	libraryDemoCmd.AddCommand(worktreesLibraryCmd)

	// Add to root command
	rootCmd.AddCommand(libraryDemoCmd)
}
