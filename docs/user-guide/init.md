# Project Initialization

The `init` command helps you quickly set up a new project or add ccmgr-ultra support to an existing repository.

## Overview

The `init` command performs the following tasks:

1. **Git Repository Setup** - Initializes a git repository if one doesn't exist
2. **Configuration Creation** - Creates `.ccmgr-ultra/config.yaml` with sensible defaults
3. **Project Structure** - Sets up the basic project structure
4. **Claude Integration** - Optionally prepares the project for Claude Code sessions

## Basic Usage

### Initialize a New Project

```bash
# In an empty directory
ccmgr-ultra init
```

### Initialize an Existing Repository

```bash
# In an existing git repository
ccmgr-ultra init
```

### Custom Project Name

```bash
# Specify a custom project name
ccmgr-ultra init --repo-name my-awesome-project
```

## Command Options

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--repo-name` | `-r` | Repository/project name | Directory name |
| `--description` | `-d` | Project description | None |
| `--template` | `-t` | Project template to use | None |
| `--branch` | `-b` | Initial git branch name | `main` |
| `--no-claude` | | Skip Claude Code setup | false |
| `--force` | | Force initialization (overwrite existing) | false |

## Examples

### Basic Initialization

```bash
# Initialize with defaults
ccmgr-ultra init
```

Output:
```
✓ Project initialization complete!

Project 'my-project' has been successfully initialized!

Next steps:
  - Review the configuration in .ccmgr-ultra/config.yaml
  - Run 'ccmgr-ultra status' to check your setup
  - Run 'ccmgr-ultra continue' to start a Claude Code session
  - Use 'ccmgr-ultra --help' to explore available commands
```

### Custom Branch Name

```bash
# Initialize with 'develop' as the default branch
ccmgr-ultra init --branch develop
```

### Skip Claude Setup

```bash
# Initialize without Claude Code integration
ccmgr-ultra init --no-claude
```

### Force Reinitialization

```bash
# Overwrite existing configuration
ccmgr-ultra init --force
```

## What Gets Created

### Directory Structure

```
your-project/
├── .ccmgr-ultra/
│   └── config.yaml     # Project-specific configuration
└── .git/               # Git repository (if not already present)
```

### Default Configuration

The `init` command creates a basic configuration file with:

```yaml
# .ccmgr-ultra/config.yaml
version: "2.0.0"

# Project-specific overrides can be added here
git:
  default_branch: "main"  # Or your specified branch

claude:
  enabled: true           # Unless --no-claude is used

worktree:
  # Inherits from global config
  
tmux:
  # Inherits from global config
```

## Initialization Process

### 1. Validation Phase

- Checks if current directory is writable
- Validates project name (if provided)
- Checks for existing configuration

### 2. Git Setup

- Detects existing git repository
- If none exists, runs `git init`
- Creates initial branch if specified

### 3. Configuration Creation

- Creates `.ccmgr-ultra` directory
- Generates default configuration
- Merges with any template settings

### 4. Claude Integration (Optional)

- Verifies Claude Code availability
- Prepares environment for Claude sessions
- Sets up integration preferences

## Working with Templates

!!! note "Template Support"
    Template functionality is planned for future releases. This will allow you to initialize projects with predefined configurations and structures.

Future template usage:
```bash
# Initialize with a template (coming soon)
ccmgr-ultra init --template python-fastapi
ccmgr-ultra init --template node-typescript
```

## Best Practices

### 1. Project Naming

Choose descriptive project names that:
- Are URL-safe (avoid special characters)
- Follow your team's naming conventions
- Are unique within your workspace

### 2. Branch Strategy

Consider your workflow when choosing the initial branch:
```bash
# For GitHub Flow
ccmgr-ultra init --branch main

# For Git Flow
ccmgr-ultra init --branch develop

# For custom workflows
ccmgr-ultra init --branch trunk
```

### 3. Configuration Review

Always review the generated configuration:
```bash
# After initialization
cat .ccmgr-ultra/config.yaml

# Edit if needed
$EDITOR .ccmgr-ultra/config.yaml
```

## Troubleshooting

### "Configuration already exists" Error

If you see this error:
```
Error: ccmgr-ultra configuration already exists
Suggestion: Use --force to reinitialize or run 'ccmgr-ultra status' to check current setup
```

Options:
1. Check existing setup: `ccmgr-ultra status`
2. Force reinitialize: `ccmgr-ultra init --force`
3. Manually edit: `$EDITOR .ccmgr-ultra/config.yaml`

### Git Repository Issues

If git initialization fails:
```bash
# Check git installation
git --version

# Initialize manually if needed
git init
git checkout -b main

# Then run ccmgr-ultra init
ccmgr-ultra init
```

### Permission Errors

Ensure you have write permissions:
```bash
# Check permissions
ls -la

# Fix if needed
chmod u+w .

# Retry initialization
ccmgr-ultra init
```

## Integration with Existing Projects

### Adding to Existing Git Repository

ccmgr-ultra seamlessly integrates with existing repositories:

```bash
# In your existing project
cd my-existing-project

# Add ccmgr-ultra support
ccmgr-ultra init

# The existing git setup is preserved
git status  # Shows only new .ccmgr-ultra directory
```

### Monorepo Support

For monorepos, initialize in the root:
```bash
# In monorepo root
ccmgr-ultra init --repo-name my-monorepo

# Each sub-project can have its own worktrees
ccmgr-ultra worktree create --path packages/frontend feature/new-ui
ccmgr-ultra worktree create --path packages/backend feature/new-api
```

### CI/CD Integration

Add `.ccmgr-ultra` to version control:
```bash
# After initialization
git add .ccmgr-ultra
git commit -m "Add ccmgr-ultra configuration"
```

This allows team members to:
- Share project-specific settings
- Maintain consistent configurations
- Track configuration changes

## Next Steps

After initialization:

1. **Check Status**: Run `ccmgr-ultra status` to verify setup
2. **Create Worktree**: Use `ccmgr-ultra worktree create` for feature branches
3. **Start Session**: Run `ccmgr-ultra continue` to begin coding
4. **Configure**: Edit `.ccmgr-ultra/config.yaml` for project-specific settings

## Related Commands

- [`worktree create`](../worktree-commands.md) - Create feature branch worktrees
- [`session`](../session-commands.md) - Start or resume Claude Code sessions
- [`config`](configuration.md) - Manage configuration settings