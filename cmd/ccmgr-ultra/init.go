package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/unbracketed/ccmgr-ultra/internal/cli"
	"github.com/unbracketed/ccmgr-ultra/internal/config"
	"github.com/unbracketed/ccmgr-ultra/internal/git"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new project with ccmgr-ultra configuration",
	Long: `Initialize a new project by setting up:
- Git repository (if not already present)
- ccmgr-ultra configuration
- Default project structure
- Optional Claude Code session initialization

This command can be run in an empty directory to create a new project,
or in an existing git repository to add ccmgr-ultra support.`,
	RunE: runInitCommand,
}

var initFlags struct {
	repoName    string
	description string
	template    string
	noClaude    bool
	branch      string
	force       bool
}

func init() {
	initCmd.Flags().StringVarP(&initFlags.repoName, "repo-name", "r", "", "Repository name (defaults to directory name)")
	initCmd.Flags().StringVarP(&initFlags.description, "description", "d", "", "Project description")
	initCmd.Flags().StringVarP(&initFlags.template, "template", "t", "", "Project template to use")
	initCmd.Flags().BoolVar(&initFlags.noClaude, "no-claude", false, "Skip Claude Code session initialization")
	initCmd.Flags().StringVarP(&initFlags.branch, "branch", "b", "main", "Initial branch name")
	initCmd.Flags().BoolVar(&initFlags.force, "force", false, "Force initialization even if configuration exists")

	rootCmd.AddCommand(initCmd)
}

func runInitCommand(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return handleCLIError(cli.NewErrorWithCause("failed to get current directory", err))
	}

	// Validate project name if provided
	projectName := initFlags.repoName
	if projectName == "" {
		projectName = filepath.Base(cwd)
	}

	if err := cli.ValidateProjectName(projectName); err != nil {
		return handleCLIError(err)
	}

	// Check if already initialized
	if !initFlags.force {
		if err := checkExistingConfiguration(cwd); err != nil {
			return handleCLIError(err)
		}
	}

	var spinner *cli.Spinner
	if shouldShowProgress() {
		spinner = cli.NewSpinner("Initializing project...")
		spinner.Start()
		defer spinner.Stop()
	}

	// Initialize git repository if needed
	if spinner != nil {
		spinner.SetMessage("Setting up git repository...")
	}
	if err := initializeGitRepository(cwd); err != nil {
		return handleCLIError(err)
	}

	// Create ccmgr-ultra configuration
	if spinner != nil {
		spinner.SetMessage("Creating ccmgr-ultra configuration...")
	}
	if err := createConfiguration(cwd, projectName); err != nil {
		return handleCLIError(err)
	}

	// Initialize Claude Code session if requested
	if !initFlags.noClaude {
		if spinner != nil {
			spinner.SetMessage("Setting up Claude Code session...")
		}
		if err := initializeClaudeSession(cwd); err != nil {
			// Don't fail if Claude setup fails, just warn
			if isVerbose() {
				fmt.Printf("Warning: Failed to initialize Claude Code session: %v\n", err)
			}
		}
	}

	if spinner != nil {
		spinner.StopWithMessage("Project initialization complete!")
	}

	if !isQuiet() {
		fmt.Printf("\nProject '%s' has been successfully initialized!\n", projectName)
		fmt.Println("\nNext steps:")
		fmt.Println("  - Review the configuration in .ccmgr-ultra/config.yaml")
		fmt.Println("  - Run 'ccmgr-ultra status' to check your setup")
		if !initFlags.noClaude {
			fmt.Println("  - Run 'ccmgr-ultra continue' to start a Claude Code session")
		}
		fmt.Println("  - Use 'ccmgr-ultra --help' to explore available commands")
	}

	return nil
}

func checkExistingConfiguration(projectDir string) error {
	configDir := filepath.Join(projectDir, ".ccmgr-ultra")
	configFile := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(configFile); err == nil {
		return cli.NewErrorWithSuggestion(
			"ccmgr-ultra configuration already exists",
			"Use --force to reinitialize or run 'ccmgr-ultra status' to check current setup",
		)
	}

	return nil
}

func initializeGitRepository(projectDir string) error {
	// Check if we're already in a git repository
	gitCmd := git.NewGitCmd()
	repoManager := git.NewRepositoryManager(gitCmd)

	_, err := repoManager.DetectRepository(projectDir)
	if err == nil {
		// Already in a git repository
		if isVerbose() {
			fmt.Println("Using existing git repository")
		}
		return nil
	}

	// Initialize new git repository
	if isDryRun() {
		fmt.Printf("Would initialize git repository in: %s\n", projectDir)
		return nil
	}

	if _, err := gitCmd.Execute(projectDir, "init"); err != nil {
		return cli.NewErrorWithCause("failed to initialize git repository", err)
	}

	// Create initial branch if specified
	if initFlags.branch != "master" && initFlags.branch != "" {
		if _, err := gitCmd.Execute(projectDir, "checkout", "-b", initFlags.branch); err != nil {
			// Don't fail if branch creation fails
			if isVerbose() {
				fmt.Printf("Warning: Failed to create initial branch '%s': %v\n", initFlags.branch, err)
			}
		}
	}

	if isVerbose() {
		fmt.Printf("Initialized git repository with branch: %s\n", initFlags.branch)
	}

	return nil
}

func createConfiguration(projectDir, projectName string) error {
	configDir := filepath.Join(projectDir, ".ccmgr-ultra")
	configFile := filepath.Join(configDir, "config.yaml")

	if isDryRun() {
		fmt.Printf("Would create configuration directory: %s\n", configDir)
		fmt.Printf("Would create configuration file: %s\n", configFile)
		return nil
	}

	// Create configuration directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return cli.NewErrorWithCause("failed to create configuration directory", err)
	}

	// Create default configuration
	cfg := createDefaultConfig(projectName)

	// Write configuration to file
	if err := config.Save(cfg, configFile); err != nil {
		return cli.NewErrorWithCause("failed to write configuration file", err)
	}

	if isVerbose() {
		fmt.Printf("Created configuration file: %s\n", configFile)
	}

	return nil
}

func createDefaultConfig(projectName string) *config.Config {
	cfg := config.DefaultConfig()

	// Override with user-provided values if there's a project section
	// For now, just set basic fields that exist in the config

	// Set git configuration
	if initFlags.branch != "" {
		cfg.Git.DefaultBranch = initFlags.branch
	}

	return cfg
}

func initializeClaudeSession(projectDir string) error {
	if isDryRun() {
		fmt.Println("Would initialize Claude Code session")
		return nil
	}

	// Load the newly created configuration
	cfg, err := loadConfigWithOverrides()
	if err != nil {
		return err
	}

	// Check if Claude Code is available
	// This is a simplified check - in practice, you might want to verify Claude Code installation

	if isVerbose() {
		fmt.Println("Claude Code session initialization would happen here")
		fmt.Println("Note: This is a placeholder for future Claude Code integration")
	}

	// For now, just ensure the configuration supports Claude
	if cfg.Claude.Enabled {
		if isVerbose() {
			fmt.Println("Claude Code integration is enabled in configuration")
		}
	}

	return nil
}

// Utility function to check if we can run gum scripts
func isGumAvailable() bool {
	// Check if gum is available in PATH
	// This is used for interactive project setup scripts
	return false // Placeholder - implement actual gum detection
}

func runGumInitScript() error {
	// This would run the gum-based interactive initialization script
	// Placeholder for future gum integration
	return nil
}
