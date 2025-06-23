# ccmgr-ultra

**Comprehensive CLI tool for managing Claude Code sessions across multiple projects and git worktrees**

---

## What is ccmgr-ultra?

ccmgr-ultra is a powerful command-line tool that streamlines your development workflow by combining seamless tmux session management with git worktree support and Claude Code integration. It's designed for developers who work across multiple projects and feature branches simultaneously.

## Key Features

### 🚀 **Tmux Session Management**
Create and manage tmux sessions specifically designed for Claude Code development with automatic naming, health monitoring, and easy resumption.

### 🌿 **Git Worktree Support**
Effortlessly create, manage, and merge git worktrees with intelligent directory naming and safety checks for uncommitted changes.

### 📊 **Real-time Status Monitoring**
Monitor Claude Code process states (idle, busy, waiting) with customizable hooks for automation and notifications.

### 🎨 **Beautiful Terminal UI**
Intuitive terminal user interface built with Charm's Bubble Tea for seamless navigation and management.

### 🔧 **Extensible Hook System**
Execute custom scripts on state changes to automate your workflow and integrate with other tools.

## Quick Start

### Installation

```bash
go install github.com/unbracketed/ccmgr-ultra/cmd/ccmgr-ultra@latest
```

### Basic Usage

```bash
# Launch the interactive TUI
ccmgr-ultra

# Initialize a new project
ccmgr-ultra init

# Create a new worktree with session
ccmgr-ultra worktree create feature/new-feature --start-session

# List all sessions
ccmgr-ultra session list

# Check system status
ccmgr-ultra status
```

## Core Workflows

### Feature Development Workflow

1. **Create a worktree** for your feature branch
2. **Start a tmux session** with Claude Code
3. **Develop** with real-time process monitoring
4. **Push and create PR** when ready
5. **Clean up** worktree and session after merge

### Multi-Project Management

- Manage multiple projects from a single interface
- Switch between different Claude Code sessions
- Monitor all active processes across projects
- Automated cleanup of stale sessions and worktrees

## Getting Started

<div class="grid cards" markdown>

-   :material-rocket-launch: **[Project Initialization](user-guide/init.md)**

    ---

    Initialize new projects and set up your development environment

-   :material-play-circle: **[Session Commands](session-commands.md)**

    ---

    Learn how to manage tmux sessions with Claude Code

-   :material-cog: **[Configuration](user-guide/configuration.md)**

    ---

    Customize ccmgr-ultra to fit your workflow

-   :material-help-circle: **[Worktree Commands](worktree-commands.md)**

    ---

    Master git worktree management for feature development

</div>

## Why ccmgr-ultra?

**Traditional Development Challenges:**
- Juggling multiple feature branches across different directories
- Managing tmux sessions manually for each project
- Losing track of Claude Code processes and their states
- Repetitive setup for new features and experiments

**ccmgr-ultra Solutions:**
- ✅ Automated worktree creation with intelligent naming
- ✅ Integrated tmux sessions with Claude Code monitoring
- ✅ Real-time status tracking with customizable hooks
- ✅ Streamlined cleanup and project lifecycle management
- ✅ Beautiful TUI for intuitive workflow management

## Architecture Overview

ccmgr-ultra is built with modern Go practices and proven technologies:

- **Language**: Go 1.24.4 for performance and reliability
- **UI Framework**: Charm's Bubble Tea for beautiful terminal interfaces
- **Database**: SQLite for session and analytics storage
- **Configuration**: Viper + YAML for flexible configuration management
- **CLI Framework**: Cobra for robust command-line interfaces

## Community and Support

- **Documentation**: Comprehensive guides and examples
- **GitHub Issues**: Bug reports and feature requests
- **Contributing**: Open source with contribution guidelines

---

Ready to streamline your Claude Code development workflow? [Initialize your first project](user-guide/init.md) or explore the [configuration guide](user-guide/configuration.md) to learn more.