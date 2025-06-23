package ccmgr

import (
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the ccmgr library
func ExampleUsage() {
	// Create a new client with default configuration
	client, err := NewClient(nil)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get system status
	status := client.System().Status()
	fmt.Printf("System Status: %d active sessions, %d worktrees\n",
		status.ActiveSessions, status.TrackedWorktrees)

	// List all sessions
	sessions, err := client.Sessions().List()
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
		return
	}

	fmt.Printf("Found %d sessions:\n", len(sessions))
	for _, session := range sessions {
		fmt.Printf("  - %s (%s) in %s\n", session.Name, session.Status, session.Directory)
	}

	// List recent worktrees
	worktrees, err := client.Worktrees().Recent()
	if err != nil {
		log.Printf("Failed to list worktrees: %v", err)
		return
	}

	fmt.Printf("Recent worktrees (%d):\n", len(worktrees))
	for _, wt := range worktrees {
		fmt.Printf("  - %s (%s) - Claude: %s\n", wt.Path, wt.Branch, wt.ClaudeStatus.State)
	}

	// Create a new session (example)
	sessionID, err := client.Sessions().Create("my-project", "/path/to/project")
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return
	}

	fmt.Printf("Created session: %s\n", sessionID)
}
