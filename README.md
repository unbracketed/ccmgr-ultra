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
go install github.com/your-username/ccmgr-ultra/cmd/ccmgr-ultra@latest
```

## Usage

Launch the TUI:
```bash
ccmgr-ultra
```

Check version:
```bash
ccmgr-ultra version
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