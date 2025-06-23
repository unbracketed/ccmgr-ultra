# Worktree Commands Documentation

The `ccmgr-ultra worktree` command provides comprehensive git worktree management integrated with tmux sessions and Claude Code processes. This documentation covers all available worktree subcommands and their usage.

## Overview

Git worktrees allow you to have multiple branches checked out simultaneously in different directories. ccmgr-ultra extends this functionality with:

- Automatic worktree directory naming based on configurable patterns
- Seamless tmux session integration
- Claude Code process tracking
- Safety checks for uncommitted changes
- GitHub pull request creation support

## Commands

### `worktree list`

List all git worktrees with comprehensive status information.

```bash
ccmgr-ultra worktree list [flags]
```

**Flags:**
- `-f, --format string`: Output format (table, json, yaml, compact) (default: "table")
- `-s, --status string`: Filter by status (clean, dirty, active, stale)
- `-b, --branch string`: Filter by branch name pattern
- `--with-processes`: Include Claude Code process information
- `--sort string`: Sort by (name, last-accessed, created, status) (default: "name")

**Examples:**

```bash
# List all worktrees in table format
ccmgr-ultra worktree list

# List only dirty worktrees
ccmgr-ultra worktree list --status dirty

# List worktrees matching a branch pattern
ccmgr-ultra worktree list --branch "feature/*"

# Show worktrees with process information in JSON format
ccmgr-ultra worktree list --with-processes --format json
```

### `worktree create`

Create a new git worktree with optional tmux session.

```bash
ccmgr-ultra worktree create <branch> [flags]
```

**Flags:**
- `-b, --base string`: Base branch for new worktree (default: current branch)
- `-d, --directory string`: Custom worktree directory path (auto-generated if not specified)
- `-s, --start-session`: Automatically start tmux session
- `--start-claude`: Automatically start Claude Code in new session
- `-r, --remote`: Track remote branch if exists
- `--force`: Overwrite existing worktree if present

**Examples:**

```bash
# Create a worktree for a new feature branch
ccmgr-ultra worktree create feature/new-auth

# Create worktree based on main branch with tmux session
ccmgr-ultra worktree create feature/api-v2 --base main --start-session

# Create worktree with custom directory
ccmgr-ultra worktree create bugfix/issue-123 -d ~/work/fixes/issue-123

# Create worktree and start Claude Code
ccmgr-ultra worktree create feature/ui-redesign -s --start-claude
```

### `worktree delete`

Delete a git worktree with safety checks and cleanup options.

```bash
ccmgr-ultra worktree delete <worktree> [flags]
```

**Flags:**
- `-f, --force`: Skip confirmation prompts
- `--cleanup-sessions`: Terminate related tmux sessions
- `--cleanup-processes`: Stop related Claude Code processes
- `--keep-branch`: Keep git branch after deleting worktree
- `--pattern string`: Delete multiple worktrees matching pattern

**Examples:**

```bash
# Delete a worktree with confirmation
ccmgr-ultra worktree delete feature/old-feature

# Force delete without confirmation
ccmgr-ultra worktree delete bugfix/resolved -f

# Delete worktree and cleanup all resources
ccmgr-ultra worktree delete feature/abandoned --cleanup-sessions --cleanup-processes

# Keep the branch when deleting worktree
ccmgr-ultra worktree delete experiment/test --keep-branch

# Delete multiple worktrees matching a pattern
ccmgr-ultra worktree delete --pattern "experiment/*" --force
```

### `worktree merge`

Merge worktree changes back to target branch.

```bash
ccmgr-ultra worktree merge <worktree> [flags]
```

**Flags:**
- `-t, --target string`: Target branch for merge (default: "main")
- `-s, --strategy string`: Merge strategy (merge, squash, rebase) (default: "merge")
- `--delete-after`: Delete worktree after successful merge
- `--push-first`: Push worktree branch before merging
- `-m, --message string`: Custom merge commit message

**Note:** This command is not yet fully implemented.

**Examples:**

```bash
# Merge feature branch to main
ccmgr-ultra worktree merge feature/completed

# Squash merge to develop branch
ccmgr-ultra worktree merge feature/experiment --target develop --strategy squash

# Merge and delete worktree after
ccmgr-ultra worktree merge feature/done --delete-after

# Push changes before merging
ccmgr-ultra worktree merge feature/reviewed --push-first
```

### `worktree push`

Push worktree branch to remote with optional PR creation.

```bash
ccmgr-ultra worktree push <worktree> [flags]
```

**Flags:**
- `--create-pr`: Create pull request after push
- `--pr-title string`: Pull request title
- `--pr-body string`: Pull request body
- `--draft`: Create draft pull request
- `--reviewer string`: Add reviewers to pull request
- `--force`: Force push (use with caution)

**Examples:**

```bash
# Simple push to remote
ccmgr-ultra worktree push feature/ready

# Push and create pull request
ccmgr-ultra worktree push feature/complete --create-pr

# Create draft PR with custom title
ccmgr-ultra worktree push feature/wip --create-pr --draft --pr-title "WIP: New authentication system"

# Push with PR and reviewer
ccmgr-ultra worktree push feature/reviewed --create-pr --reviewer "teammate"

# Force push (careful!)
ccmgr-ultra worktree push feature/rebased --force
```

## Configuration

Worktree behavior can be configured in `~/.config/ccmgr-ultra/config.yaml`:

```yaml
git:
  worktree:
    directory_pattern: ".git/{{.Branch}}"  # Template for auto-generated paths
    auto_cleanup: true                     # Clean up abandoned worktrees
    cleanup_days: 30                       # Days before considering stale
  
  # Pull request settings
  default_pr_target_branch: "main"
  github_pr_template: |
    ## Description
    Brief description of changes
    
    ## Type of Change
    - [ ] Bug fix
    - [ ] New feature
    - [ ] Breaking change
    
    ## Testing
    - [ ] Tests pass locally
    - [ ] Added new tests

tmux:
  naming_pattern: "ccmgr-{project}-{branch}"  # Session naming pattern
```

## Worktree Directory Patterns

The `directory_pattern` configuration supports Go template syntax with these variables:

- `{{.Branch}}`: Branch name (with `/` replaced by `-`)
- `{{.Project}}`: Current project name
- `{{.Date}}`: Current date (YYYY-MM-DD)
- `{{.Timestamp}}`: Unix timestamp

**Examples:**
- `.git/{{.Branch}}` → `.git/feature-auth`
- `worktrees/{{.Project}}-{{.Branch}}` → `worktrees/myapp-feature-auth`
- `work/{{.Date}}/{{.Branch}}` → `work/2024-01-15/feature-auth`

## Integration with Tmux Sessions

When creating worktrees with the `--start-session` flag, ccmgr-ultra:

1. Creates a new tmux session named according to the configured pattern
2. Sets the working directory to the worktree path
3. Optionally starts Claude Code if `--start-claude` is specified
4. Tracks the session in the database for easy management

## Safety Features

- **Uncommitted Changes Warning**: Prevents accidental deletion of worktrees with uncommitted changes
- **Active Session Detection**: Warns when deleting worktrees with active tmux sessions
- **Branch Protection**: Option to keep branches when deleting worktrees
- **Confirmation Prompts**: Requires confirmation for destructive operations (unless `--force`)

## Common Workflows

### Feature Development

```bash
# Create a new feature worktree with session
ccmgr-ultra worktree create feature/new-feature --base develop -s

# Work on the feature...

# Push and create PR when ready
ccmgr-ultra worktree push feature/new-feature --create-pr

# Clean up after merge
ccmgr-ultra worktree delete feature/new-feature --cleanup-sessions
```

### Quick Bug Fix

```bash
# Create worktree from main
ccmgr-ultra worktree create bugfix/critical --base main -s --start-claude

# Fix the bug...

# Push with PR
ccmgr-ultra worktree push bugfix/critical --create-pr --pr-title "Fix: Critical bug in auth"

# Merge and cleanup
ccmgr-ultra worktree merge bugfix/critical --delete-after
```

### Experimentation

```bash
# Create experimental worktree
ccmgr-ultra worktree create experiment/new-approach

# Try things out...

# If unsuccessful, just delete
ccmgr-ultra worktree delete experiment/new-approach -f

# If successful, push for review
ccmgr-ultra worktree push experiment/new-approach --create-pr --draft
```

## Tips and Best Practices

1. **Naming Conventions**: Use descriptive branch names that follow your team's conventions (e.g., `feature/`, `bugfix/`, `hotfix/`)

2. **Regular Cleanup**: Periodically run `ccmgr-ultra worktree list --status stale` to identify abandoned worktrees

3. **Session Integration**: Use `--start-session` to automatically create tmux sessions for better workflow integration

4. **PR Templates**: Configure `github_pr_template` in your config for consistent PR descriptions

5. **Safety First**: Always check worktree status before deletion, especially for worktrees with uncommitted changes

## Troubleshooting

### "Worktree has uncommitted changes"
Check the worktree status and either commit or stash changes before deletion:
```bash
cd /path/to/worktree
git status
git stash  # or git commit
```

### "Template pattern error"
Check your `directory_pattern` configuration. It should use Go template syntax:
```yaml
directory_pattern: "{{.Project}}/{{.Branch}}"  # Good
directory_pattern: "{project}/{branch}"        # Bad
```

### "GitHub authentication failed"
Set up GitHub authentication:
```bash
export GITHUB_TOKEN=your_token
# or configure in config.yaml
```

### Finding Worktree Paths
If you're unsure of the exact worktree name:
```bash
# List all worktrees with paths
ccmgr-ultra worktree list

# Search by branch pattern
ccmgr-ultra worktree list --branch "feature/*"
```