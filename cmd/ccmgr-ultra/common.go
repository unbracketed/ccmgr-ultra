package main

import (
	"github.com/bcdekker/ccmgr-ultra/internal/cli"
	"github.com/bcdekker/ccmgr-ultra/internal/config"
)

// loadConfigWithOverrides loads configuration with command-line overrides
func loadConfigWithOverrides() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if configPath != "" {
		// Load from custom path
		cfg, err = config.LoadFromPath(configPath)
		if err != nil {
			return nil, cli.NewErrorWithCause("failed to load custom config", err)
		}
	} else {
		// Load from default locations
		cfg, err = config.Load()
		if err != nil {
			return nil, cli.NewErrorWithCause("failed to load configuration", err)
		}
	}

	// Apply any global flag overrides here if needed
	// For now, just return the loaded config
	return cfg, nil
}

// handleCLIError processes errors in a consistent way for CLI commands
func handleCLIError(err error) error {
	if err == nil {
		return nil
	}
	
	return cli.HandleCLIError(err)
}

// setupOutputFormatter creates an output formatter based on the format string
func setupOutputFormatter(format string) (cli.OutputFormatter, error) {
	outputFormat, err := cli.ValidateFormat(format)
	if err != nil {
		return nil, err
	}
	
	return cli.NewFormatter(outputFormat, nil), nil
}

// validateWorktreeArg validates a worktree name argument
func validateWorktreeArg(name string) error {
	return cli.ValidateWorktreeName(name)
}

// validateSessionArg validates a session name argument
func validateSessionArg(name string) error {
	return cli.ValidateSessionName(name)
}

// validateBranchArg validates a branch name argument
func validateBranchArg(name string) error {
	return cli.ValidateBranchName(name)
}

// shouldShowProgress determines if progress indicators should be displayed
func shouldShowProgress() bool {
	return !quiet && cli.ShouldShowProgress()
}

// isVerbose returns true if verbose output is enabled
func isVerbose() bool {
	return verbose
}

// isQuiet returns true if quiet mode is enabled
func isQuiet() bool {
	return quiet
}

// isDryRun returns true if dry-run mode is enabled
func isDryRun() bool {
	return dryRun
}