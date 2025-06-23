# ccmgr-ultra

Claude Multi-Project Multi-Session Manager - A comprehensive CLI tool for managing Claude Code sessions across multiple projects and git worktrees.

## Overview

ccmgr-ultra combines the best features of CCManager and Claude Squad to provide:

- **Tmux Session Management**: Seamless creation and management of tmux sessions for Claude Code
- **Git Worktree Support**: Easy creation, merging, and management of git worktrees
- **Status Monitoring**: Real-time monitoring of Claude Code process states (idle, busy, waiting)
- **Hook System**: Execute custom scripts on state changes for automation
- **Beautiful TUI**: Intuitive terminal user interface built with Charm's Bubble Tea
- **Multi-Project Support**: Manage multiple projects and sessions from a single interface

## Features

- Create and manage git worktrees with auto-naming conventions
- Launch Claude Code in isolated tmux sessions per worktree
- Monitor Claude Code status and execute hooks on state changes
- Push worktrees to remote with automatic PR creation
- Configure shortcuts and custom commands
- Support for new project initialization with gum scripts

## Installation

```bash
go install github.com/unbracketed/ccmgr-ultra/cmd/ccmgr-ultra@latest
```

## Usage

### Quick Start

Launch the interactive TUI:
```bash
ccmgr-ultra
```

For CLI-only mode:
```bash
ccmgr-ultra --non-interactive
```

### Project Lifecycle Management

#### 1. Initialize New Project
```bash
# Initialize new project with default settings
ccmgr-ultra init

# Initialize with custom settings
ccmgr-ultra init --repo-name my-project --description "My awesome project" --branch develop
```

#### 2. Create Worktrees for Feature Development
```bash
# Create new worktree from current branch
ccmgr-ultra worktree create feature/new-feature

# Create worktree with automatic session start
ccmgr-ultra worktree create feature/auth --start-session --start-claude

# Create from specific base branch
ccmgr-ultra worktree create hotfix/bug-123 --base main
```

#### 3. Switch Between Sessions
```bash
# Continue existing session or create new one
ccmgr-ultra continue feature/new-feature

# Resume specific session
ccmgr-ultra session resume ccmgr-feature-new-feature

# List all active sessions
ccmgr-ultra session list
```

#### 4. Monitor Project Status
```bash
# View comprehensive status
ccmgr-ultra status

# Watch status with auto-refresh
ccmgr-ultra status --watch

# Status for specific worktree
ccmgr-ultra status --worktree feature/new-feature
```

#### 5. Push and Create Pull Requests
```bash
# Push worktree and create PR
ccmgr-ultra worktree push feature/new-feature --create-pr --pr-title "Add new feature"

# Push with custom PR details
ccmgr-ultra worktree push feature/auth --create-pr --pr-title "Implement authentication" --pr-body "Adds user auth with JWT tokens"
```

#### 6. Merge and Cleanup
```bash
# Merge worktree back to main
ccmgr-ultra worktree merge feature/new-feature --target main --delete-after

# Clean up stale sessions
ccmgr-ultra session clean --older-than 48h
```

### Common Workflows

**New Feature Development:**
```bash
ccmgr-ultra worktree create feature/my-feature --start-session --start-claude
ccmgr-ultra continue feature/my-feature
# ... develop feature ...
ccmgr-ultra worktree push feature/my-feature --create-pr
```

**Quick Bug Fix:**
```bash
ccmgr-ultra worktree create hotfix/fix-123 --base main --start-session
# ... fix bug ...
ccmgr-ultra worktree merge hotfix/fix-123 --delete-after
```

**Project Exploration:**
```bash
ccmgr-ultra status --watch
ccmgr-ultra session list --with-processes
ccmgr-ultra worktree list --sort last-accessed
```

### Additional Commands

Check version:
```bash
ccmgr-ultra version
```

Enable shell completion:
```bash
ccmgr-ultra completion bash > /etc/bash_completion.d/ccmgr-ultra
# or
ccmgr-ultra completion install-completion
```

## Project Status

This project is currently in early development. See [steps-to-implement.md](steps-to-implement.md) for the implementation roadmap.

## Requirements

- Go 1.18+
- tmux
- git
- gum (for interactive scripts)

## License

MIT