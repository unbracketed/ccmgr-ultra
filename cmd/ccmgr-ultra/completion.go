package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bcdekker/ccmgr-ultra/internal/cli"
	"github.com/bcdekker/ccmgr-ultra/internal/git"
	"github.com/bcdekker/ccmgr-ultra/internal/tmux"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(ccmgr-ultra completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ ccmgr-ultra completion bash > /etc/bash_completion.d/ccmgr-ultra
  # macOS:
  $ ccmgr-ultra completion bash > /usr/local/etc/bash_completion.d/ccmgr-ultra

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ ccmgr-ultra completion zsh > "${fpath[1]}/_ccmgr-ultra"

  # You will need to start a new shell for this setup to take effect.

fish:
  $ ccmgr-ultra completion fish | source

  # To load completions for each session, execute once:
  $ ccmgr-ultra completion fish > ~/.config/fish/completions/ccmgr-ultra.fish

PowerShell:
  PS> ccmgr-ultra completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> ccmgr-ultra completion powershell > ccmgr-ultra.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// Add completion functions for dynamic completion
	registerCompletionFunctions()
}

// registerCompletionFunctions registers custom completion functions for various arguments
func registerCompletionFunctions() {
	// Worktree name completion
	worktreeListCmd.RegisterFlagCompletionFunc("branch", completeWorktreeBranches)
	worktreeListCmd.RegisterFlagCompletionFunc("status", completeWorktreeStatuses)

	worktreeCreateCmd.RegisterFlagCompletionFunc("base", completeBranches)
	worktreeCreateCmd.RegisterFlagCompletionFunc("directory", completeDirectories)

	worktreeDeleteCmd.ValidArgsFunction = completeWorktreeNames
	worktreeMergeCmd.ValidArgsFunction = completeWorktreeNames
	worktreeMergeCmd.RegisterFlagCompletionFunc("target", completeBranches)
	worktreeMergeCmd.RegisterFlagCompletionFunc("strategy", completeMergeStrategies)

	worktreePushCmd.ValidArgsFunction = completeWorktreeNames

	// Session name completion
	sessionListCmd.RegisterFlagCompletionFunc("worktree", completeWorktreeNames)
	sessionListCmd.RegisterFlagCompletionFunc("project", completeProjectNames)
	sessionListCmd.RegisterFlagCompletionFunc("status", completeSessionStatuses)

	sessionNewCmd.ValidArgsFunction = completeWorktreeNames
	sessionNewCmd.RegisterFlagCompletionFunc("config", completeConfigFiles)

	sessionResumeCmd.ValidArgsFunction = completeSessionIDs
	sessionKillCmd.ValidArgsFunction = completeSessionIDs

	// Status command completion
	statusCmd.RegisterFlagCompletionFunc("worktree", completeWorktreeNames)
	statusCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}

// completeWorktreeNames provides completion for worktree names
func completeWorktreeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	worktrees, err := getWorktreeNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return filterCompletions(worktrees, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeWorktreeBranches provides completion for worktree branch names
func completeWorktreeBranches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	branches, err := getWorktreeBranches()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return filterCompletions(branches, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeWorktreeStatuses provides completion for worktree status values
func completeWorktreeStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	statuses := []string{"clean", "dirty", "active", "stale"}
	return filterCompletions(statuses, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeBranches provides completion for git branch names
func completeBranches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	branches, err := getBranches()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return filterCompletions(branches, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeDirectories provides completion for directory paths
func completeDirectories(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Use default directory completion
	return nil, cobra.ShellCompDirectiveFilterDirs
}

// completeMergeStrategies provides completion for merge strategies
func completeMergeStrategies(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	strategies := []string{"merge", "squash", "rebase"}
	return filterCompletions(strategies, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeSessionIDs provides completion for session IDs
func completeSessionIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	sessionIDs, err := getSessionIDs()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return filterCompletions(sessionIDs, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeSessionStatuses provides completion for session status values
func completeSessionStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	statuses := []string{"active", "idle", "stale"}
	return filterCompletions(statuses, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeProjectNames provides completion for project names
func completeProjectNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projects, err := getProjectNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return filterCompletions(projects, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeConfigFiles provides completion for config file paths
func completeConfigFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Look for common config file patterns
	patterns := []string{"*.toml", "*.yaml", "*.yml", "*.json"}
	var files []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			files = append(files, matches...)
		}
	}

	return filterCompletions(files, toComplete), cobra.ShellCompDirectiveDefault
}

// completeOutputFormats provides completion for output format values
func completeOutputFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	formats := []string{"table", "json", "yaml", "compact"}
	return filterCompletions(formats, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// Helper functions for gathering completion data

// getWorktreeNames returns a list of available worktree names
func getWorktreeNames() ([]string, error) {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return nil, err
	}

	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return nil, err
	}

	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
	worktrees, err := worktreeManager.ListWorktrees()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(worktrees))
	for _, wt := range worktrees {
		names = append(names, filepath.Base(wt.Path))
	}

	return names, nil
}

// getWorktreeBranches returns a list of branches from worktrees
func getWorktreeBranches() ([]string, error) {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return nil, err
	}

	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return nil, err
	}

	worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
	worktrees, err := worktreeManager.ListWorktrees()
	if err != nil {
		return nil, err
	}

	branches := make([]string, 0, len(worktrees))
	for _, wt := range worktrees {
		if wt.Branch != "" {
			branches = append(branches, wt.Branch)
		}
	}

	return uniqueStrings(branches), nil
}

// getBranches returns a list of all git branches
func getBranches() ([]string, error) {
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)
	repo, err := repoManager.DetectRepository(".")
	if err != nil {
		return nil, err
	}

	gitOps := git.NewGitOperations(repo, gitCmd)
	branchInfos, err := gitOps.ListBranches(false) // false for local only
	if err != nil {
		return nil, err
	}

	branches := make([]string, 0, len(branchInfos))
	for _, branchInfo := range branchInfos {
		branches = append(branches, branchInfo.Name)
	}

	// Clean up branch names (remove prefixes like "origin/")
	cleaned := make([]string, 0, len(branches))
	for _, branch := range branches {
		// Remove "origin/" prefix if present
		if strings.HasPrefix(branch, "origin/") {
			branch = strings.TrimPrefix(branch, "origin/")
		}
		// Remove "* " prefix for current branch
		if strings.HasPrefix(branch, "* ") {
			branch = strings.TrimPrefix(branch, "* ")
		}
		branch = strings.TrimSpace(branch)
		if branch != "" {
			cleaned = append(cleaned, branch)
		}
	}

	return uniqueStrings(cleaned), nil
}

// getSessionIDs returns a list of active session IDs
func getSessionIDs() ([]string, error) {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return nil, err
	}

	sessionManager := tmux.NewSessionManager(cfg)
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(sessions))
	for _, sess := range sessions {
		ids = append(ids, sess.ID)
		// Also add session names as alternatives
		if sess.Name != sess.ID {
			ids = append(ids, sess.Name)
		}
	}

	return uniqueStrings(ids), nil
}

// getProjectNames returns a list of project names
func getProjectNames() ([]string, error) {
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return nil, err
	}

	sessionManager := tmux.NewSessionManager(cfg)
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return nil, err
	}

	projects := make([]string, 0)
	for _, sess := range sessions {
		if sess.Project != "" {
			projects = append(projects, sess.Project)
		}
	}

	return uniqueStrings(projects), nil
}

// filterCompletions filters completion options based on the current input
func filterCompletions(options []string, toComplete string) []string {
	if toComplete == "" {
		return options
	}

	filtered := make([]string, 0)
	for _, option := range options {
		if strings.HasPrefix(option, toComplete) {
			filtered = append(filtered, option)
		}
	}

	return filtered
}

// uniqueStrings removes duplicates from a slice of strings
func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	unique := make([]string, 0, len(slice))

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			unique = append(unique, item)
		}
	}

	return unique
}

// Advanced completion functions for specific scenarios

// completeWorktreeNamesWithStatus provides worktree completion with status filtering
func completeWorktreeNamesWithStatus(status string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := loadConfigWithOverrides()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		gitCmd := git.NewGitCmd()
		repoManager := git.NewRepositoryManager(gitCmd)
		repo, err := repoManager.DetectRepository(".")
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		worktreeManager := git.NewWorktreeManager(repo, cfg, gitCmd)
		worktrees, err := worktreeManager.ListWorktrees()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var filtered []string
		for _, wt := range worktrees {
			// Filter by status if specified
			if status != "" {
				wtStatus := "clean"
				if !wt.IsClean {
					wtStatus = "dirty"
				}
				if wtStatus != status {
					continue
				}
			}

			filtered = append(filtered, filepath.Base(wt.Path))
		}

		return filterCompletions(filtered, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// completeSessionIDsWithStatus provides session completion with status filtering
func completeSessionIDsWithStatus(status string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := loadConfigWithOverrides()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		sessionManager := tmux.NewSessionManager(cfg)
		sessions, err := sessionManager.ListSessions()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var filtered []string
		for _, sess := range sessions {
			// Filter by status if specified
			if status != "" {
				sessStatus := "idle"
				if sess.Active {
					sessStatus = "active"
				}
				// Add stale check logic here if needed
				if sessStatus != status {
					continue
				}
			}

			filtered = append(filtered, sess.ID)
			if sess.Name != sess.ID {
				filtered = append(filtered, sess.Name)
			}
		}

		return filterCompletions(uniqueStrings(filtered), toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// installCompletionCommand provides a helper command to install completions
var installCompletionCmd = &cobra.Command{
	Use:   "install-completion [shell]",
	Short: "Install shell completion for ccmgr-ultra",
	Long: `Install shell completion for ccmgr-ultra to the appropriate location
for your shell. Supports bash, zsh, and fish.`,
	ValidArgs: []string{"bash", "zsh", "fish"},
	Args:      cobra.MaximumNArgs(1),
	RunE:      runInstallCompletion,
}

func init() {
	completionCmd.AddCommand(installCompletionCmd)
}

func runInstallCompletion(cmd *cobra.Command, args []string) error {
	shell := detectShell()
	if len(args) > 0 {
		shell = args[0]
	}

	switch shell {
	case "bash":
		return installBashCompletion()
	case "zsh":
		return installZshCompletion()
	case "fish":
		return installFishCompletion()
	default:
		return cli.NewErrorWithSuggestion(
			fmt.Sprintf("unsupported shell: %s", shell),
			"Specify one of: bash, zsh, fish",
		)
	}
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash" // Default fallback
	}

	shell = filepath.Base(shell)
	switch shell {
	case "bash", "zsh", "fish":
		return shell
	default:
		return "bash"
	}
}

func installBashCompletion() error {
	// Try system locations
	locations := []string{
		"/usr/local/etc/bash_completion.d/ccmgr-ultra",
		"/etc/bash_completion.d/ccmgr-ultra",
		filepath.Join(os.Getenv("HOME"), ".bash_completion.d", "ccmgr-ultra"),
	}

	for _, location := range locations {
		dir := filepath.Dir(location)
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}

		file, err := os.Create(location)
		if err != nil {
			continue
		}
		defer file.Close()

		if err := rootCmd.GenBashCompletion(file); err != nil {
			continue
		}

		fmt.Printf("Bash completion installed to: %s\n", location)
		fmt.Println("Restart your shell or run: source ~/.bashrc")
		return nil
	}

	return cli.NewError("failed to install bash completion - no writable location found")
}

func installZshCompletion() error {
	// Get zsh fpath
	fpath := os.Getenv("fpath")
	if fpath == "" {
		fpath = "/usr/local/share/zsh/site-functions"
	}

	// Use first directory in fpath
	fpathDirs := strings.Split(fpath, ":")
	location := filepath.Join(fpathDirs[0], "_ccmgr-ultra")

	dir := filepath.Dir(location)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return cli.NewErrorWithCause("failed to create completion directory", err)
	}

	file, err := os.Create(location)
	if err != nil {
		return cli.NewErrorWithCause("failed to create completion file", err)
	}
	defer file.Close()

	if err := rootCmd.GenZshCompletion(file); err != nil {
		return cli.NewErrorWithCause("failed to generate completion", err)
	}

	fmt.Printf("Zsh completion installed to: %s\n", location)
	fmt.Println("Restart your shell for changes to take effect")
	return nil
}

func installFishCompletion() error {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}

	location := filepath.Join(configDir, "fish", "completions", "ccmgr-ultra.fish")
	dir := filepath.Dir(location)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return cli.NewErrorWithCause("failed to create completion directory", err)
	}

	file, err := os.Create(location)
	if err != nil {
		return cli.NewErrorWithCause("failed to create completion file", err)
	}
	defer file.Close()

	if err := rootCmd.GenFishCompletion(file, true); err != nil {
		return cli.NewErrorWithCause("failed to generate completion", err)
	}

	fmt.Printf("Fish completion installed to: %s\n", location)
	fmt.Println("Restart your shell for changes to take effect")
	return nil
}
