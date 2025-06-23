# Session Commands Documentation

The `ccmgr-ultra session` command provides comprehensive tmux session management integrated with Claude Code processes. This documentation covers all available session subcommands and how to interact with sessions effectively.

## Overview

Sessions in ccmgr-ultra are tmux sessions specifically designed for Claude Code development. Each session:

- Is tied to a specific git worktree
- Has a unique identifier and name following configurable patterns
- Tracks Claude Code process status
- Maintains activity and health information
- Can be resumed, terminated, or cleaned up automatically

## Commands

### `session list`

List all active tmux sessions managed by ccmgr-ultra.

```bash
ccmgr-ultra session list [flags]
```

**Flags:**
- `-f, --format string`: Output format (table, json, yaml, compact) (default: "table")
- `-w, --worktree string`: Filter by worktree name
- `-p, --project string`: Filter by project name
- `-s, --status string`: Filter by status (active, idle, stale)
- `--with-processes`: Include Claude Code process details

**Examples:**

```bash
# List all sessions in table format
ccmgr-ultra session list

# List sessions for a specific worktree
ccmgr-ultra session list --worktree feature/new-api

# Show only active sessions with process info
ccmgr-ultra session list --status active --with-processes

# Export session data as JSON
ccmgr-ultra session list --format json > sessions.json
```

### `session new`

Create a new tmux session for a worktree.

```bash
ccmgr-ultra session new <worktree> [flags]
```

**Flags:**
- `--name string`: Custom session name suffix
- `--start-claude`: Automatically start Claude Code
- `-d, --detached`: Create session detached from terminal
- `--claude-config string`: Custom Claude Code config for session
- `--inherit-config`: Inherit config from parent directory

**Examples:**

```bash
# Create session for a worktree
ccmgr-ultra session new feature/auth-system

# Create session and start Claude Code
ccmgr-ultra session new feature/ui-redesign --start-claude

# Create detached session with custom name
ccmgr-ultra session new bugfix/memory-leak --name debug-session -d

# Create session with custom Claude config
ccmgr-ultra session new experiment/new-approach --claude-config ./custom-claude.md
```

### `session resume`

Resume an existing tmux session with health validation.

```bash
ccmgr-ultra session resume <session-id> [flags]
```

**Flags:**
- `-a, --attach`: Attach to session in current terminal
- `-d, --detached`: Resume session detached
- `--restart-claude`: Restart Claude Code if stopped
- `--force`: Force resume even if session appears unhealthy

**Examples:**

```bash
# Resume a session by ID
ccmgr-ultra session resume ccmgr-myproject-feature-auth

# Resume and attach to session
ccmgr-ultra session resume ccmgr-myproject-feature-auth --attach

# Resume with Claude restart
ccmgr-ultra session resume ccmgr-myproject-bugfix --restart-claude

# Force resume an unhealthy session
ccmgr-ultra session resume old-session --force
```

### `session kill`

Terminate a tmux session gracefully.

```bash
ccmgr-ultra session kill <session-id> [flags]
```

**Flags:**
- `-f, --force`: Skip confirmation prompts
- `--all-stale`: Kill all stale/orphaned sessions
- `--pattern string`: Kill sessions matching pattern
- `--cleanup`: Clean up related processes and state
- `--timeout int`: Timeout for graceful shutdown in seconds (default: 10)

**Examples:**

```bash
# Kill a specific session
ccmgr-ultra session kill ccmgr-myproject-old-feature

# Force kill without confirmation
ccmgr-ultra session kill abandoned-session -f

# Kill all stale sessions
ccmgr-ultra session kill any --all-stale --cleanup

# Kill sessions matching pattern
ccmgr-ultra session kill any --pattern "experiment/*" --force

# Kill with custom timeout
ccmgr-ultra session kill busy-session --timeout 30 --cleanup
```

### `session clean`

Clean up stale, orphaned, or invalid sessions.

```bash
ccmgr-ultra session clean [flags]
```

**Flags:**
- `--dry-run`: Show what would be cleaned without acting
- `-f, --force`: Skip confirmation prompts
- `--all`: Clean all eligible sessions, not just stale ones
- `--older-than string`: Clean sessions older than specified duration (default: "24h")
- `--verbose`: Detailed cleanup information

**Examples:**

```bash
# Preview cleanup operation
ccmgr-ultra session clean --dry-run

# Clean sessions older than 48 hours
ccmgr-ultra session clean --older-than 48h

# Force clean all eligible sessions
ccmgr-ultra session clean --all --force

# Clean with verbose output
ccmgr-ultra session clean --verbose --older-than 7d
```

## Session Interaction Methods

### Direct Tmux Commands

You can interact with ccmgr-ultra sessions using standard tmux commands:

```bash
# List all tmux sessions
tmux ls

# Attach to a session
tmux attach -t ccmgr-myproject-feature

# Detach from current session
# Press: Ctrl+b, then d

# Switch between sessions
# Press: Ctrl+b, then s (select from list)

# Kill a session
tmux kill-session -t ccmgr-myproject-old
```

### Session Navigation

Inside a tmux session, use these key combinations:

- **Ctrl+b, d**: Detach from session
- **Ctrl+b, s**: List and switch sessions
- **Ctrl+b, $**: Rename current session
- **Ctrl+b, (**: Switch to previous session
- **Ctrl+b, )**: Switch to next session

### Window Management

Each session can have multiple windows:

- **Ctrl+b, c**: Create new window
- **Ctrl+b, n**: Next window
- **Ctrl+b, p**: Previous window
- **Ctrl+b, 0-9**: Switch to window by number
- **Ctrl+b, w**: List all windows
- **Ctrl+b, ,**: Rename current window
- **Ctrl+b, &**: Close current window

### Pane Management

Split windows into panes for multiple views:

- **Ctrl+b, %**: Split vertically
- **Ctrl+b, "**: Split horizontally
- **Ctrl+b, arrow**: Navigate between panes
- **Ctrl+b, z**: Zoom/unzoom current pane
- **Ctrl+b, space**: Cycle through pane layouts
- **Ctrl+b, x**: Close current pane

## Configuration

Session behavior can be configured in `~/.config/ccmgr-ultra/config.yaml`:

```yaml
tmux:
  naming_pattern: "ccmgr-{project}-{branch}"  # Session naming template
  auto_resume: true                           # Auto-resume on attach
  clean_on_exit: false                        # Clean up on session exit
  
  # Session defaults
  default_window_name: "claude"
  start_directory: "."                        # Relative to worktree
  
  # Health check settings
  health_check_interval: 60                   # Seconds between health checks
  stale_threshold: 3600                       # Seconds before marking stale

# Claude integration
claude:
  auto_start: false                           # Start Claude automatically
  restart_on_failure: true                    # Restart if Claude crashes
  config_inheritance: true                    # Inherit parent configs
```

## Session Naming Patterns

The `naming_pattern` configuration supports these variables:

- `{project}`: Current project name
- `{branch}`: Git branch name
- `{worktree}`: Worktree directory name
- `{date}`: Current date (YYYY-MM-DD)
- `{time}`: Current time (HH-MM)

**Examples:**
- `ccmgr-{project}-{branch}` → `ccmgr-myapp-feature-auth`
- `{project}/{worktree}` → `myapp/feature-auth`
- `dev-{date}-{branch}` → `dev-2024-01-15-bugfix`

## Common Workflows

### Development Session

```bash
# Create worktree with session
ccmgr-ultra worktree create feature/new-feature -s --start-claude

# Later, resume the session
ccmgr-ultra session resume ccmgr-myproject-feature-new-feature -a

# When done for the day, detach
# Press Ctrl+b, d

# Clean up when feature is complete
ccmgr-ultra session kill ccmgr-myproject-feature-new-feature --cleanup
ccmgr-ultra worktree delete feature/new-feature
```

### Multiple Active Sessions

```bash
# Create multiple sessions
ccmgr-ultra session new feature/api
ccmgr-ultra session new feature/ui
ccmgr-ultra session new bugfix/memory

# List all sessions
ccmgr-ultra session list

# Switch between them
tmux attach -t ccmgr-myproject-feature-api
# Press Ctrl+b, s to see session list
# Select different session

# Clean up stale sessions periodically
ccmgr-ultra session clean --older-than 48h
```

### Session Recovery

```bash
# Check session health
ccmgr-ultra session list --with-processes

# Resume unhealthy session
ccmgr-ultra session resume broken-session --force --restart-claude

# If that fails, kill and recreate
ccmgr-ultra session kill broken-session -f
ccmgr-ultra session new feature/continued
```

## Advanced Session Management

### Custom Session Scripts

Create a custom session with specific setup:

```bash
#!/bin/bash
# custom-session.sh

WORKTREE=$1
SESSION_NAME="ccmgr-custom-$WORKTREE"

# Create session
tmux new-session -d -s "$SESSION_NAME" -c "$WORKTREE"

# Setup windows
tmux rename-window -t "$SESSION_NAME:0" "claude"
tmux new-window -t "$SESSION_NAME" -n "tests"
tmux new-window -t "$SESSION_NAME" -n "logs"

# Start processes
tmux send-keys -t "$SESSION_NAME:0" "claude" Enter
tmux send-keys -t "$SESSION_NAME:1" "npm test --watch" Enter
tmux send-keys -t "$SESSION_NAME:2" "tail -f logs/*.log" Enter

# Attach
tmux attach -t "$SESSION_NAME"
```

### Session Hooks

Use tmux hooks for automation:

```bash
# ~/.tmux.conf
# Run command when session is created
set-hook -g session-created 'run-shell "ccmgr-ultra hook session-start"'

# Run command when session closes
set-hook -g session-closed 'run-shell "ccmgr-ultra hook session-end"'
```

### Persistent Sessions

Keep sessions alive across system restarts using tmux-resurrect:

```bash
# Install tmux-resurrect
git clone https://github.com/tmux-plugins/tmux-resurrect ~/tmux-resurrect

# Add to ~/.tmux.conf
run-shell ~/tmux-resurrect/resurrect.tmux

# Save session state
# Press Ctrl+b, Ctrl+s

# Restore session state
# Press Ctrl+b, Ctrl+r
```

## Tips and Best Practices

1. **Session Organization**: Use consistent naming patterns to easily identify sessions

2. **Regular Cleanup**: Run `session clean` weekly to remove abandoned sessions

3. **Health Monitoring**: Use `--with-processes` flag to monitor Claude Code status

4. **Detached Creation**: Use `-d` flag to create sessions without attaching immediately

5. **Session Groups**: Organize related sessions using similar naming prefixes

## Troubleshooting

### "Session not found"
```bash
# List all sessions to verify name
ccmgr-ultra session list
tmux ls

# Use exact session ID from list
ccmgr-ultra session resume [exact-id]
```

### "Session appears unhealthy"
```bash
# Force resume
ccmgr-ultra session resume session-id --force

# Or kill and recreate
ccmgr-ultra session kill session-id -f
ccmgr-ultra session new worktree-name
```

### "Claude Code not starting"
```bash
# Check Claude is in PATH
which claude

# Resume with restart flag
ccmgr-ultra session resume session-id --restart-claude

# Or manually start in session
tmux attach -t session-id
claude  # Run manually
```

### Session Permissions Issues
```bash
# Check tmux server
tmux info

# Kill tmux server if needed
tmux kill-server

# Restart with ccmgr-ultra
ccmgr-ultra session new worktree-name
```