# CCMgr Library

This package provides a clean Go library API for ccmgr-ultra functionality, allowing easy integration into other Go applications.

## Overview

The `ccmgr` package exposes three main manager interfaces:
- **SessionManager** - Manage tmux sessions
- **WorktreeManager** - Manage git worktrees
- **SystemManager** - Monitor system status and health

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/unbracketed/ccmgr-ultra/pkg/ccmgr"
)

func main() {
    // Create a new client
    client, err := ccmgr.NewClient(nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Get system status
    status := client.System().Status()
    fmt.Printf("Active sessions: %d\n", status.ActiveSessions)
    
    // List sessions
    sessions, err := client.Sessions().List()
    if err != nil {
        log.Fatal(err)
    }
    
    for _, session := range sessions {
        fmt.Printf("Session: %s (%s)\n", session.Name, session.Status)
    }
}
```

## API Reference

### Client Creation

```go
// Create with default config
client, err := ccmgr.NewClient(nil)

// Create with custom config
cfg, _ := config.Load()
client, err := ccmgr.NewClient(cfg)

// Create with context
ctx := context.Background()
client, err := ccmgr.NewClientWithContext(ctx, cfg)
```

### Session Management

```go
sm := client.Sessions()

// List all sessions
sessions, err := sm.List()

// Get only active sessions
active, err := sm.Active()

// Create new session
sessionID, err := sm.Create("project-name", "/path/to/project")

// Attach to session
err := sm.Attach(sessionID)

// Resume paused session
err := sm.Resume(sessionID)

// Find sessions for worktree
sessions, err := sm.FindForWorktree("/path/to/worktree")
```

### Worktree Management

```go
wm := client.Worktrees()

// List all worktrees
worktrees, err := wm.List()

// Get recent worktrees
recent, err := wm.Recent()

// Create new worktree
err := wm.Create("/path/to/worktree", "feature-branch")

// Open worktree
err := wm.Open("/path/to/worktree")

// Get Claude status
status := wm.GetClaudeStatus("/path/to/worktree")

// Update Claude status
wm.UpdateClaudeStatus("/path/to/worktree", newStatus)
```

### System Monitoring

```go
sys := client.System()

// Get system status
status := sys.Status()

// Refresh all data
err := sys.Refresh()

// Get health information
health := sys.Health()
```

## Data Types

### SessionInfo
- `ID` - Unique session identifier
- `Name` - Human-readable session name
- `Project` - Associated project name
- `Branch` - Git branch
- `Directory` - Working directory
- `Active` - Whether session is currently active
- `Created` - Creation timestamp
- `LastAccess` - Last access timestamp
- `PID` - Process ID
- `Status` - Current status

### WorktreeInfo
- `Path` - Worktree filesystem path
- `Branch` - Git branch name
- `Repository` - Repository name
- `Active` - Whether worktree is active
- `LastAccess` - Last access timestamp
- `HasChanges` - Whether there are uncommitted changes
- `Status` - Git status (clean, modified, conflicts, etc.)
- `ActiveSessions` - Associated sessions
- `ClaudeStatus` - Claude Code process status
- `GitStatus` - Detailed git status information

### SystemStatus
- `ActiveProcesses` - Number of active processes
- `ActiveSessions` - Number of active sessions
- `TrackedWorktrees` - Number of tracked worktrees
- `LastUpdate` - Last update timestamp
- `IsHealthy` - Overall health status
- `Errors` - List of system errors
- `Memory` - Memory usage statistics
- `Performance` - Performance metrics

## Architecture

This library is built on top of the existing ccmgr-ultra internal modules:
- Wraps the `internal/tui/integration.go` layer
- Provides clean, stable public interfaces
- Maintains backward compatibility with internal implementations
- Enables easy testing and mocking

## Example CLI Usage

See `cmd/ccmgr-ultra/library_demo.go` for examples of how to use this library in CLI applications:

```bash
# Show system status via library
ccmgr-ultra library-demo status

# List sessions via library
ccmgr-ultra library-demo sessions

# List worktrees via library
ccmgr-ultra library-demo worktrees
```

## Migration Guide

### Before (Direct Internal Imports)
```go
import (
    "github.com/unbracketed/ccmgr-ultra/internal/tmux"
    "github.com/unbracketed/ccmgr-ultra/internal/claude"
    "github.com/unbracketed/ccmgr-ultra/internal/git"
)

// Direct usage of internal APIs
tmuxMgr := tmux.NewSessionManager(config)
sessions, err := tmuxMgr.ListSessions()
```

### After (Library API)
```go
import "github.com/unbracketed/ccmgr-ultra/pkg/ccmgr"

// Clean library interface
client, err := ccmgr.NewClient(nil)
sessions, err := client.Sessions().List()
```

## Benefits

- **Clean API**: Well-defined interfaces separate from internal implementation
- **Stable**: Public API remains stable while internals can evolve
- **Testable**: Easy to mock and test individual components
- **Reusable**: Can be imported by other Go applications
- **Type Safe**: Strong typing with clear data structures
- **Documentation**: Self-documenting interfaces and examples