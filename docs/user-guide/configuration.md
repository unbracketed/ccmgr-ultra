# Configuration

ccmgr-ultra uses a flexible, hierarchical configuration system that allows you to customize behavior at both global and project levels.

## Configuration Files

### File Locations

ccmgr-ultra looks for configuration files in these locations:

1. **Global Configuration**: `~/.config/ccmgr-ultra/config.yaml`
   - Applies to all projects
   - Can also be at `$XDG_CONFIG_HOME/ccmgr-ultra/config.yaml`

2. **Project Configuration**: `<project-dir>/.ccmgr-ultra/config.yaml`
   - Optional, project-specific overrides
   - Takes precedence over global configuration

3. **Database**: `~/.config/ccmgr-ultra/data.db`
   - SQLite database for session and analytics data
   - Automatically created on first run

### Configuration Precedence

Settings are applied in this order (highest priority first):

1. **Command-line flags** - Override all other settings
2. **Environment variables** - Prefixed with `CCMGR_`
3. **Project configuration** - If present in working directory
4. **Global configuration** - User-wide settings
5. **Default values** - Built-in defaults

## Environment Variables

Any configuration option can be set via environment variables using the `CCMGR_` prefix and converting dots to underscores:

```bash
# Set worktree default branch
export CCMGR_WORKTREE_DEFAULT_BRANCH=develop

# Enable analytics
export CCMGR_ANALYTICS_ENABLED=true

# Set status hook timeout
export CCMGR_STATUS_HOOKS_IDLE_TIMEOUT=60
```

## Configuration Options

### Claude Process Monitoring

Configure how ccmgr-ultra monitors Claude Code processes:

```yaml
claude:
  enabled: true                    # Enable/disable Claude monitoring
  poll_interval: "3s"             # How often to check process status
  max_processes: 10               # Maximum processes to track
  cleanup_on_exit: true           # Clean up resources on exit
  log_paths:                      # Where to look for Claude logs
    - "~/.claude/logs"
    - "/tmp/claude-*"
  state_patterns:                 # Regex patterns for state detection
    busy: '(?i)(Processing|Executing|Running)'
    idle: '(?i)(Waiting for input|Ready)'
    waiting: '(?i)(Waiting|Pending)'
  integrate_with_tmux: true       # Link processes with tmux sessions
  integrate_with_worktrees: true  # Link processes with git worktrees
```

### Git Worktree Management

Control how worktrees are created and managed:

```yaml
worktree:
  base_directory: "../.worktrees/{{.Project}}"  # Where to create worktrees
  directory_pattern: "{{.Branch}}"              # How to name worktree directories
  auto_directory: true                          # Auto-create base directory
  default_branch: "main"                        # Default branch for new worktrees
  
git:
  default_branch: "main"                        # Default git branch
  protected_branches:                           # Branches that can't be deleted
    - "main"
    - "master" 
    - "develop"
  default_remote: "origin"                      # Default remote name
  auto_push: true                              # Auto-push new branches
  cleanup_on_merge: true                       # Delete worktree after merge
  force_push_allowed: false                    # Allow force push
```

!!! info "Template Variables"
    Directory patterns support these template variables:
    
    - `{{.Project}}` - Project name
    - `{{.Branch}}` - Branch name
    - `{{.Timestamp}}` - Current timestamp
    - `{{.Date}}` - Current date (YYYY-MM-DD)
    - `{{.User}}` - Current username

### Tmux Integration

Configure tmux session management:

```yaml
tmux:
  session_prefix: "ccmgr"                      # Prefix for session names
  naming_pattern: "{{.Prefix}}-{{.Project}}-{{.Branch}}"
  monitor_interval: "2s"                       # Status check interval
  auto_cleanup: true                          # Clean up dead sessions
  environment:                                # Environment variables for sessions
    EDITOR: "vim"
    TERM: "xterm-256color"
```

### Hook System

Execute scripts on state changes:

```yaml
status_hooks:
  enabled: true                               # Master switch for all hooks
  idle:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/idle.sh"
    timeout: 30                               # Seconds before timeout
    async: true                               # Run in background
  busy:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/busy.sh"
    timeout: 30
    async: false
  waiting:
    enabled: false
    script: "~/.config/ccmgr-ultra/hooks/waiting.sh"

worktree_hooks:
  on_create:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/worktree-create.sh"
  on_activate:
    enabled: true  
    script: "~/.config/ccmgr-ultra/hooks/worktree-activate.sh"
```

!!! tip "Hook Environment Variables"
    Hooks receive these environment variables:
    
    - `CCMGR_SESSION_ID` - Current session ID
    - `CCMGR_WORKTREE_ID` - Current worktree ID
    - `CCMGR_WORKING_DIR` - Working directory path
    - `CCMGR_NEW_STATE` - New state (for status hooks)
    - `CCMGR_OLD_STATE` - Previous state (for status hooks)

### Analytics

Control usage tracking and metrics:

```yaml
analytics:
  enabled: true                               # Enable analytics collection
  collector:
    poll_interval: "30s"                      # Collection frequency
    retention_days: 90                        # How long to keep data
    batch_size: 100                           # Events per batch
  performance:
    track_cpu: true                           # Monitor CPU usage
    track_memory: true                        # Monitor memory usage
    track_disk: false                         # Monitor disk usage
```

### TUI Settings

Customize the terminal user interface:

```yaml
tui:
  theme: "default"                            # Color theme
  refresh_interval: 5                         # Seconds between refreshes
  mouse_support: true                         # Enable mouse interactions
  show_key_help: true                         # Show keyboard shortcuts
  default_screen: "home"                      # Starting screen
  max_items_per_page: 20                      # Pagination limit
```

### Commands

Configure external command paths:

```yaml
commands:
  claude: "claude"                            # Claude CLI command
  git: "git"                                  # Git command
  tmux_prefix: "tmux"                         # Tmux command prefix
  environment:                                # Additional env vars
    GIT_MERGE_AUTOEDIT: "no"
```

### Shortcuts

Customize keyboard shortcuts:

```yaml
shortcuts:
  quit: "q"
  help: "?"
  refresh: "r"
  navigate_up: "k"
  navigate_down: "j"
  select: "enter"
  back: "esc"
  create_worktree: "w"
  create_session: "s"
  delete: "d"
  merge: "m"
```

## Example Configuration

Here's a complete example configuration file:

```yaml
version: "2.0.0"

# Claude process monitoring
claude:
  enabled: true
  poll_interval: "3s"
  state_patterns:
    busy: '(?i)(Processing|Executing|Running|Thinking)'
    idle: '(?i)(Waiting for input|Ready|└─►)'

# Worktree settings
worktree:
  base_directory: "../.worktrees/{{.Project}}"
  directory_pattern: "{{.Branch}}"
  auto_directory: true

# Git configuration  
git:
  default_branch: "main"
  protected_branches: ["main", "master"]
  auto_push: true
  cleanup_on_merge: true

# Tmux integration
tmux:
  session_prefix: "ccmgr"
  naming_pattern: "{{.Prefix}}-{{.Project}}-{{.Branch}}"
  environment:
    EDITOR: "nvim"

# Status change hooks
status_hooks:
  enabled: true
  idle:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/on-idle.sh"
    timeout: 30
    async: true
  busy:
    enabled: true
    script: "~/.config/ccmgr-ultra/hooks/on-busy.sh"
    async: true

# Analytics
analytics:
  enabled: true
  collector:
    retention_days: 30

# TUI preferences
tui:
  theme: "default"
  mouse_support: true
  refresh_interval: 5
```

## Creating a Configuration File

To create your initial configuration:

```bash
# Create config directory
mkdir -p ~/.config/ccmgr-ultra

# Copy example configuration
cp example-claude-config.yaml ~/.config/ccmgr-ultra/config.yaml

# Edit with your preferred editor
$EDITOR ~/.config/ccmgr-ultra/config.yaml
```

## Validating Configuration

ccmgr-ultra validates configuration on startup. To check your configuration:

```bash
# Validate configuration
ccmgr-ultra config validate

# Show current configuration
ccmgr-ultra config show

# Show configuration with sources (where each value comes from)
ccmgr-ultra config show --sources
```

## Project-Specific Configuration

To override settings for a specific project:

```bash
# Create project config directory
mkdir .ccmgr-ultra

# Create project-specific config
cat > .ccmgr-ultra/config.yaml << EOF
# Project overrides
worktree:
  default_branch: "develop"
  
git:
  protected_branches: ["develop", "staging", "production"]
  
tmux:
  naming_pattern: "myproject-{{.Branch}}"
EOF
```

## Best Practices

1. **Start Simple**: Begin with minimal configuration and add options as needed
2. **Use Templates**: Leverage template variables for flexible naming
3. **Project Overrides**: Use project configs for team-specific settings
4. **Environment Variables**: Use env vars for temporary overrides
5. **Version Control**: Commit project configs to share with team
6. **Backup**: Keep backups of your global configuration

## Troubleshooting

### Configuration Not Loading

1. Check file permissions: `ls -la ~/.config/ccmgr-ultra/config.yaml`
2. Validate YAML syntax: `yamllint ~/.config/ccmgr-ultra/config.yaml`
3. Run with verbose logging: `ccmgr-ultra -v status`

### Environment Variables Not Working

Ensure proper naming format:
- Replace dots with underscores
- Prefix with `CCMGR_`
- Use uppercase

Example: `claude.poll_interval` → `CCMGR_CLAUDE_POLL_INTERVAL`

### Project Config Not Applied

1. Ensure file is at `.ccmgr-ultra/config.yaml` in project root
2. Check you're in the correct directory
3. Verify file permissions
4. Check for YAML syntax errors

## Next Steps

- Learn about [Session Commands](../session-commands.md) for session management
- Explore [Worktree Management](../worktree-commands.md) for git worktree workflows
- Read about [Project Initialization](init.md) for setting up new projects