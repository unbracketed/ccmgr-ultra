# Purpose: Claude Multi-Project Mult-Session Manager

I'd like to combine what I feel are the best features of "CCManager" and "Claude Squad" into my own custom tool for managing multiple aspects of multiple projects and driving coding agents.

https://github.com/kbwo/ccmanager
https://github.com/smtg-ai/claude-squad


## Top-level Requirements

A CLI tool that can be run from anywhere in the system for initiating and/or managing the state of a project. A project will utilize Git and worktrees to conduct multiple ongoing lines of work. 

- Use tmux for managing Claude Code sessions. If possible, standardize the naming of tmux sessions to align with the project and worktree / branch
- Option to continue or resume for a project
- Monitor Claude Code state changes (busy, idle, waiting) and run a configurable hook script when state changes. It's OK to reuse the status hook env var names from CCManager because I already have a hook script based on `skate` for tracking changes. 

### New Projects

If ccmgr-ultra is started in a directory with no Git repo, or an empty directory

- Initialize a Git repo if none exists; use a `gum` script to collect data about the repo name, project name, and description
- Display menu with options:
  - New Claude Code session (in current branch)
  - New Worktree session - creates a Git worktree and starts a Claude Code session

### Existing Projects

Example from CCManager:

```
CCManager - Claude Code Worktree Manager

Select a worktree to start or resume a Claude Code session:

❯ master (main)
  ─────────────
  ⊕ New Worktree
  ⇄ Merge Worktree
  ✕ Delete Worktree
  ⌨ Configuration
  ⏻ Exit

Status: ● Busy ◐ Waiting ○ Idle
Controls: ↑↓ Navigate Enter Select
```

I'd also like to have an option to "Push Worktree" to push the branch to the remote origin and open a pull request
I'd also like to have the option to be able to continue or resume in a branch or worktree interactively. So for example, maybe highlighting the branch/worktree would bring up key options "n" for new session, "c" for continue session, "r" for resume.  

### Configuration

Follow the system that CCManager is using:

**Main config menu**

```
Configuration

Select a configuration option:

  ⌨  Configure Shortcuts
  🔧  Configure Status Hooks
  📁  Configure Worktree Settings
  🚀  Configure Command
❯ ← Back to Main Menu
```


**Configure Status Hooks**

```
Set commands to run when Claude Code session status changes:

❯ Idle: ✓ ~/code/ccmgr-hooks/hook.sh
  Busy: ✗ ~/code/ccmgr-hooks/hook.sh
  Waiting for Input: ✓ ~/code/ccmgr-hooks/hook.sh
  ─────────────
  💾 Save and Return
  ← Cancel
```

**Configure Worktree Settings**

```
Configure Worktree Settings

Configure automatic worktree directory generation

Example: branch "feature/my-feature" → directory ".git/feature-my-feature"

❯ Auto Directory: ✅ Enabled
  Pattern: .git/{branch}
  💾 Save Changes
  ← Cancel

Press Esc to cancel without saving
```



### Implementation

Use the toolkit from charm.sh for implementing a TUI:
https://github.com/charmbracelet/gum
https://github.com/charmbracelet/bubbletea
https://github.com/charmbracelet/bubbles

Use tmux for running Claude Code processes in the background and being able to attach / detach easily from any terminal window

Here is my current status hook script that I'm using with CCManager:

```
#!/bin/bash

# Set defaults if environment variables are not defined
if [ -z "$CCMANAGER_WORKTREE" ]; then
    CCMANAGER_WORKTREE=$(pwd)
fi

if [ -z "$CCMANAGER_WORKTREE_BRANCH" ]; then
    # Check if we're in a git repository and get the branch name
    if git rev-parse --git-dir > /dev/null 2>&1; then
        CCMANAGER_WORKTREE_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
    else
        CCMANAGER_WORKTREE_BRANCH="NO-REPO"
    fi
fi

# current status for worktree
skate set "${CCMANAGER_WORKTREE_BRANCH}@ccmgr-status" "${CCMANAGER_NEW_STATE}"

# track active work
# Strip /Users/brian/code or ~/code prefix from the worktree path
STRIPPED_WORKTREE="${CCMANAGER_WORKTREE#/Users/brian/code/}"
STRIPPED_WORKTREE="${STRIPPED_WORKTREE#~/code/}"
skate set "${CCMANAGER_WORKTREE_BRANCH}@ccmgr-projects" "${STRIPPED_WORKTREE}"

# current sessions
# Strip /Users/brian/code or ~/code prefix from the key
STRIPPED_KEY="${CCMANAGER_WORKTREE#/Users/brian/code/}"
STRIPPED_KEY="${STRIPPED_KEY#~/code/}"
skate set "${STRIPPED_KEY}@ccmgr-sessions" "${CCMANAGER_SESSION_ID}"
```


### Future Roadmap

- A problem to solve is being able to share or reuse parts of the Claude Code project configurations when new worktrees are created. When we make a worktree and start a new session, Claude Code will have a brand new config, but we will often want to inherit settings for permissions and MCPs from the top-level project dir. For example, if I'm in /somedir/code and I have my Claude Code configured the way I like, with MCPs, etc., and then I make a new worktree in /somedir/.git/newtree, starting Claude Code there won't have any of the settings from the "parent" location /somedir/code.  So the tool might have an option / config for syncing settings for new worktrees
 
