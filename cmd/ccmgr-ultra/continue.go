package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/your-username/ccmgr-ultra/internal/cli"
	"github.com/your-username/ccmgr-ultra/internal/config"
	"github.com/your-username/ccmgr-ultra/internal/git"
	"github.com/your-username/ccmgr-ultra/internal/tmux"
)

var continueCmd = &cobra.Command{
	Use:   "continue [worktree]",
	Short: "Continue or create a session for the specified worktree",
	Long: `Continue an existing session or create a new one for the specified worktree.
If no worktree is specified, uses the current directory's worktree.

This command will:
- Detect the current or specified worktree
- Resume an existing tmux session if available
- Create a new tmux session if none exists
- Attach to the session for interactive use`,
	Args: cobra.MaximumNArgs(1),
	RunE: runContinueCommand,
}

var continueFlags struct {
	newSession bool
	sessionID  string
	detached   bool
}

func init() {
	continueCmd.Flags().BoolVar(&continueFlags.newSession, "new-session", false, "Force new session creation")
	continueCmd.Flags().StringVarP(&continueFlags.sessionID, "session-id", "s", "", "Specific session ID to continue")
	continueCmd.Flags().BoolVarP(&continueFlags.detached, "detached", "d", false, "Start session detached from terminal")

	rootCmd.AddCommand(continueCmd)
}

func runContinueCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return handleCLIError(err)
	}

	// Determine target worktree
	var worktreePath string
	if len(args) > 0 {
		// Worktree specified as argument
		worktreePath = args[0]
		if err := validateWorktreeArg(worktreePath); err != nil {
			return handleCLIError(err)
		}
	} else {
		// Use current directory
		cwd, err := os.Getwd()
		if err != nil {
			return handleCLIError(cli.NewErrorWithCause("failed to get current directory", err))
		}
		worktreePath = cwd
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner("Setting up session...")
		spinner.Start()
		defer spinner.Stop()
	}

	// Detect worktree information
	if spinner != nil {
		spinner.SetMessage("Detecting worktree information...")
	}
	worktreeInfo, err := detectWorktreeInfo(worktreePath, cfg)
	if err != nil {
		return handleCLIError(err)
	}

	// Initialize session manager
	sessionManager := tmux.NewSessionManager(cfg)

	// Check for existing session if not forcing new
	var session *tmux.Session
	if !continueFlags.newSession && continueFlags.sessionID == "" {
		if spinner != nil {
			spinner.SetMessage("Checking for existing sessions...")
		}
		session, err = findExistingSession(sessionManager, worktreeInfo)
		if err != nil && isVerbose() {
			fmt.Printf("Warning: Failed to find existing session: %v\n", err)
		}
	} else if continueFlags.sessionID != "" {
		if spinner != nil {
			spinner.SetMessage("Looking up specified session...")
		}
		session, err = sessionManager.GetSession(continueFlags.sessionID)
		if err != nil {
			return handleCLIError(cli.NewErrorWithSuggestion(
				fmt.Sprintf("session '%s' not found", continueFlags.sessionID),
				"Use 'ccmgr-ultra session list' to see available sessions",
			))
		}
	}

	// Create new session if needed
	if session == nil {
		if spinner != nil {
			spinner.SetMessage("Creating new session...")
		}
		session, err = createNewSession(sessionManager, worktreeInfo, cfg)
		if err != nil {
			return handleCLIError(err)
		}
		
		if isVerbose() {
			fmt.Printf("Created new session: %s\n", session.ID)
		}
	} else {
		if isVerbose() {
			fmt.Printf("Using existing session: %s\n", session.ID)
		}
	}

	// Attach to session
	if spinner != nil {
		spinner.SetMessage("Attaching to session...")
	}

	if isDryRun() {
		fmt.Printf("Would attach to session: %s\n", session.ID)
		if continueFlags.detached {
			fmt.Println("Session would run in detached mode")
		}
		return nil
	}

	if continueFlags.detached {
		// Just verify session is running, don't attach
		isActive, err := sessionManager.IsSessionActive(session.ID)
		if err != nil {
			return handleCLIError(cli.NewErrorWithCause("failed to check session status", err))
		}
		if !isActive {
			return handleCLIError(cli.NewError("session is not active"))
		}
		
		if spinner != nil {
			spinner.StopWithMessage("Session started in detached mode")
		}
		
		if !isQuiet() {
			fmt.Printf("Session '%s' is running in detached mode\n", session.ID)
			fmt.Printf("Use 'tmux attach-session -t %s' to attach later\n", session.Name)
		}
	} else {
		// Attach interactively
		if spinner != nil {
			spinner.StopWithMessage("Attaching to session...")
		}
		
		if err := sessionManager.AttachSession(session.ID); err != nil {
			return handleCLIError(cli.NewErrorWithCause("failed to attach to session", err))
		}
	}

	return nil
}

// WorktreeInfo represents information about a worktree for session management
type WorktreeInfo struct {
	Path    string
	Branch  string
	Project string
	Name    string
}

func detectWorktreeInfo(path string, cfg *config.Config) (*WorktreeInfo, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, cli.NewErrorWithCause("failed to resolve path", err)
	}

	// Create git repository manager
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	
	// Detect repository
	repo, err := repoManager.DetectRepository(absPath)
	if err != nil {
		return nil, cli.NewErrorWithSuggestion(
			"not in a git repository",
			"Run this command from within a git repository or use 'ccmgr-ultra init' to create one",
		)
	}

	// Get current branch
	branch, err := gitCmd.Execute(absPath, "branch", "--show-current")
	if err != nil {
		// Fallback to HEAD if branch detection fails
		branch = "HEAD"
		if isVerbose() {
			fmt.Printf("Warning: Failed to detect current branch: %v\n", err)
		}
	}

	// Determine project name (from repository name or directory)
	projectName := filepath.Base(repo.RootPath)
	
	// Create worktree name from path relative to repository root
	relPath, err := filepath.Rel(repo.RootPath, absPath)
	if err != nil {
		relPath = filepath.Base(absPath)
	}
	
	worktreeName := relPath
	if worktreeName == "." {
		worktreeName = "main"
	}

	return &WorktreeInfo{
		Path:    absPath,
		Branch:  branch,
		Project: projectName,
		Name:    worktreeName,
	}, nil
}

func findExistingSession(sessionManager *tmux.SessionManager, worktreeInfo *WorktreeInfo) (*tmux.Session, error) {
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return nil, err
	}

	// Look for sessions matching this worktree
	for _, session := range sessions {
		if session.Worktree == worktreeInfo.Name || 
		   session.Directory == worktreeInfo.Path ||
		   session.Branch == worktreeInfo.Branch {
			
			// Check if session is still active
			isActive, err := sessionManager.IsSessionActive(session.ID)
			if err != nil {
				continue // Skip sessions we can't verify
			}
			
			if isActive {
				return session, nil
			}
		}
	}

	return nil, nil // No matching active session found
}

func createNewSession(sessionManager *tmux.SessionManager, worktreeInfo *WorktreeInfo, cfg *config.Config) (*tmux.Session, error) {
	// Create the session using the correct method signature
	session, err := sessionManager.CreateSession(
		worktreeInfo.Project,
		worktreeInfo.Name,
		worktreeInfo.Branch,
		worktreeInfo.Path,
	)
	if err != nil {
		return nil, cli.NewErrorWithCause("failed to create tmux session", err)
	}

	return session, nil
}

// Utility functions for session management
func isSessionRunning(sessionManager *tmux.SessionManager, sessionID string) bool {
	active, err := sessionManager.IsSessionActive(sessionID)
	return err == nil && active
}

func getSessionInfo(sessionManager *tmux.SessionManager, sessionID string) (*tmux.Session, error) {
	return sessionManager.GetSession(sessionID)
}