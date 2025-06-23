# Phase 4.1 Implementation Plan: CLI Interface Enhancement

## COMPREHENSIVE IMPLEMENTATION STRATEGY

### Executive Summary

Transform ccmgr-ultra from a TUI-first application to a comprehensive CLI tool supporting both interactive and automated workflows. This plan maintains 100% TUI compatibility while adding powerful command-line capabilities.

### Current State Analysis

```
CURRENT ARCHITECTURE:
cmd/ccmgr-ultra/main.go  →  TUI Application (Primary Interface)
                        →  Version Command (Basic CLI)

EXISTING ASSETS:
├── internal/config     →  Configuration management
├── internal/git        →  Git operations & worktree management  
├── internal/tmux       →  Session management
├── internal/claude     →  Process monitoring
├── internal/hooks      →  Status hooks system
└── internal/tui        →  Comprehensive TUI application
```

### Target Architecture

```
CLI COMMAND STRUCTURE:
ccmgr-ultra [global-flags] <command> [command-flags] [args]

GLOBAL FLAGS:
├── --non-interactive, -n    →  CLI-only mode
├── --config, -c            →  Custom config file
├── --verbose, -v           →  Detailed output  
├── --quiet, -q             →  Minimal output
└── --dry-run               →  Preview mode

COMMANDS:
├── init                    →  Project initialization
├── continue <worktree>     →  Session continuation
├── status                  →  System status
├── worktree <subcommand>   →  Worktree management
└── session <subcommand>    →  Session management
```

## IMPLEMENTATION PHASES

### Phase 1: Foundation Layer

**Priority:** Critical - All other phases depend on this foundation

#### Deliverables:

1. **Enhanced main.go Structure**
   ```go
   var rootCmd = &cobra.Command{
       Use:   "ccmgr-ultra",
       Run: func(cmd *cobra.Command, args []string) {
           if nonInteractive {
               // CLI-only mode logic
           } else {
               runTUI()
           }
       },
   }
   ```

2. **Global Flags Implementation**
   - `--non-interactive, -n`: Skip TUI, use CLI-only mode
   - `--config, -c`: Specify custom config file path
   - `--verbose, -v`: Enable detailed output
   - `--quiet, -q`: Suppress non-essential output  
   - `--dry-run`: Show what would be done without executing

3. **internal/cli Package Creation**
   ```
   internal/cli/
   ├── output.go        →  Multi-format output (table/JSON/YAML)
   ├── errors.go        →  Consistent CLI error handling
   ├── validation.go    →  Input validation helpers
   └── spinner.go       →  Progress indicators
   ```

4. **common.go Shared Utilities**
   ```go
   func loadConfigWithOverrides() (*config.Config, error)
   func handleCLIError(err error) error
   func setupOutputFormatter(format string) cli.OutputFormatter
   func validateWorktreeArg(name string) error
   ```

**Dependencies:** None (foundation layer)  
**Risk Level:** Low  
**Testing:** Unit tests for formatters and validators

### Phase 2: Core Commands Implementation

**Priority:** High - Essential CLI functionality

#### Day 1: `status` Command (Proof of Concept)

**File:** `cmd/ccmgr-ultra/status.go`

**Functionality:**
- Display current project status
- Show all worktrees and their states
- Display Claude Code process status
- Show active tmux sessions
- Report configuration status

**Flags:**
- `--worktree, -w`: Show status for specific worktree
- `--format, -f`: Output format (table, json, yaml)
- `--watch`: Continuously monitor and update status
- `--refresh-interval`: Status refresh interval for watch mode

**Integration Points:**
- internal/config: Configuration status
- internal/git: Worktree states
- internal/tmux: Active sessions
- internal/claude: Process monitoring

#### Day 2: `init` Command (Project Bootstrap)

**File:** `cmd/ccmgr-ultra/init.go`

**Functionality:**
- Detect if in empty/non-git directory
- Launch gum script for project initialization (if available)
- Accept flags for non-interactive initialization
- Create initial git repository structure
- Set up default ccmgr-ultra configuration
- Initialize first Claude Code session

**Flags:**
- `--repo-name, -r`: Repository name
- `--description, -d`: Project description
- `--template, -t`: Project template to use
- `--no-claude`: Skip Claude Code session initialization
- `--branch, -b`: Initial branch name (default: main)

#### Day 3: `continue` Command (Session Management)

**File:** `cmd/ccmgr-ultra/continue.go`

**Functionality:**
- Continue existing session for specified worktree
- Create new session if none exists
- Handle tmux session attachment/creation
- Support both interactive and non-interactive modes

**Flags:**
- `--new-session, -n`: Force new session creation
- `--session-id, -s`: Specific session ID to continue
- `--detached, -d`: Start session detached from terminal

### Phase 3: Command Groups Implementation

#### Day 1: `worktree` Command Group

**File:** `cmd/ccmgr-ultra/worktree.go`

**Subcommands:**
- `list`: List all worktrees with status
- `create <branch>`: Create new worktree
- `delete <worktree>`: Delete worktree
- `merge <worktree>`: Merge worktree changes
- `push <worktree>`: Push worktree and create PR

**Examples:**
```bash
ccmgr-ultra worktree list --format=table
ccmgr-ultra worktree create feature-auth --base=main
ccmgr-ultra worktree delete feature-old --force
ccmgr-ultra worktree merge feature-complete --delete-after
ccmgr-ultra worktree push feature-ready --create-pr
```

#### Day 2: `session` Command Group

**File:** `cmd/ccmgr-ultra/session.go`

**Subcommands:**
- `list`: List all active sessions
- `new <worktree>`: Create new session
- `resume <session-id>`: Resume existing session
- `kill <session-id>`: Terminate session
- `clean`: Clean up stale sessions

**Examples:**
```bash
ccmgr-ultra session list --worktree=feature-branch
ccmgr-ultra session new feature-branch --name=dev-session
ccmgr-ultra session resume ccmgr-main-feature-abc123
ccmgr-ultra session kill --all-stale
```

### Phase 4: Polish & Validation

#### User Experience Enhancement
- Progress indicators and spinners for long operations
- Enhanced error messages with actionable suggestions
- Help text and usage examples for all commands
- Shell completion support (bash/zsh)
- Verbose and quiet modes functionality
- Dry-run mode implementation

#### Comprehensive Testing
- Unit test coverage for all new commands
- Integration tests with existing TUI functionality
- Cross-platform testing (macOS/Linux)
- Performance validation (status commands <1s)
- Error scenario testing
- Command help documentation
- README updates with CLI examples

## TECHNICAL IMPLEMENTATION PATTERNS

### Command Registration Pattern

```go
var (
    nonInteractive bool
    configPath     string
    verbose        bool
    quiet          bool
    dryRun         bool
)

func init() {
    rootCmd.PersistentFlags().BoolVarP(&nonInteractive, "non-interactive", "n", false, "Skip TUI, use CLI-only mode")
    rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Custom config file path")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
    rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
    rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without executing")
    
    rootCmd.AddCommand(initCmd, continueCmd, statusCmd, worktreeCmd, sessionCmd)
}
```

### Output Formatting Strategy

```go
// internal/cli/output.go
type OutputFormatter interface {
    Format(data interface{}) error
}

type TableFormatter struct{ /* implementation */ }
type JSONFormatter struct{ /* implementation */ }
type YAMLFormatter struct{ /* implementation */ }
```

### Error Handling Pattern

```go
// internal/cli/errors.go
func HandleCLIError(err error) error {
    // Wrap errors with helpful context
    // Provide actionable suggestions
    // Maintain consistent error codes
}
```

## RISK MITIGATION STRATEGIES

### High Risk: TUI Integration Breakage
**Mitigation:** 
- Shared configuration loading between CLI and TUI
- Extensive regression testing after each phase
- Gradual rollout with fallback mechanisms

**Validation:** Run full TUI test suite after each phase

### Medium Risk: Performance Issues
**Mitigation:**
- Lazy loading for resource-intensive operations
- Caching for frequently accessed status information
- Async operations where appropriate

**Validation:** Benchmark critical paths, profile memory usage

### Low Risk: Cross-platform Compatibility
**Mitigation:**
- Use existing internal packages (already tested cross-platform)
- Platform-agnostic implementation patterns
- Consistent path handling

**Validation:** CI testing on both macOS and Linux

## IMMEDIATE NEXT STEPS

### Ready-to-Implement Actions

1. **Create internal/cli package structure:**
   ```bash
   mkdir -p internal/cli
   touch internal/cli/{output.go,errors.go,validation.go,spinner.go}
   ```

2. **Implement basic output formatters:**
   - Start with internal/cli/output.go
   - Create OutputFormatter interface
   - Implement TableFormatter and JSONFormatter
   - Add basic tests

3. **Enhance main.go with global flags:**
   - Add persistent flags for all global options
   - Modify TUI launch logic to respect --non-interactive flag
   - Add configuration override logic

4. **Create common.go utilities:**
   - Configuration loading with overrides
   - Standard error handling patterns
   - Flag parsing helpers

5. **Implement status command as proof of concept:**
   - Create cmd/ccmgr-ultra/status.go
   - Integrate with all internal packages
   - Test multi-format output
   - Validate performance (<1s execution)

## SUCCESS CRITERIA

### Functional Requirements
- [ ] 15+ CLI commands fully functional
- [ ] Both interactive and non-interactive modes working
- [ ] Multi-format output (table/JSON/YAML) implemented
- [ ] Proper integration with all internal packages
- [ ] Seamless compatibility with existing TUI functionality

### Performance Requirements
- [ ] Status/list commands execute in <1 second
- [ ] No regression in TUI startup time
- [ ] Memory usage remains reasonable for automation

### User Experience Requirements
- [ ] Clear, helpful error messages with actionable suggestions
- [ ] Intuitive command structure following Unix conventions
- [ ] Comprehensive help text and usage examples
- [ ] Consistent output formatting across all commands

### Technical Requirements
- [ ] Comprehensive test coverage (>80%)
- [ ] Cross-platform compatibility verified (macOS/Linux)
- [ ] Clean, maintainable code architecture
- [ ] Zero breaking changes to existing functionality

---

**PLAN STATUS: Complete and Ready for Implementation**

This comprehensive plan provides clear phases, concrete deliverables, technical implementation patterns, risk mitigation strategies, and immediate actionable next steps for successfully implementing Phase 4.1: CLI Interface Enhancement.