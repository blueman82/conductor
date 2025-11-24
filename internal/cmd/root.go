package cmd

import (
	"github.com/harrison/conductor/internal/config"
	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags
var Version = "dev"

// ConductorRepoRoot is the path to the conductor repository root
// Injected at build time via -ldflags
var ConductorRepoRoot = ""

// GetConductorRepoRoot returns the conductor repository root path
// This is injected at build time and is guaranteed to be correct
func GetConductorRepoRoot() string {
	return ConductorRepoRoot
}

// NewRootCommand creates and returns the root cobra command for conductor
func NewRootCommand() *cobra.Command {
	// Initialize config with build-time injected repository root
	// This ensures database location is always correctly resolved
	config.SetBuildTimeRepoRoot(ConductorRepoRoot)

	cmd := &cobra.Command{
		Use:   "conductor",
		Short: "Autonomous multi-agent orchestration system",
		Long: `Conductor executes implementation plans by spawning and managing
multiple Claude Code CLI agents in coordinated waves.

It parses plan files (Markdown or YAML), calculates task dependencies,
and orchestrates parallel execution of tasks across multiple agents.`,
		Version: Version,
		// Silence usage on errors to avoid duplicate help text
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewValidateCommand())
	cmd.AddCommand(NewLearningCommand())
	cmd.AddCommand(NewObserveCommand())

	return cmd
}
