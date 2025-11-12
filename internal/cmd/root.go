package cmd

import (
	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags
var Version = "dev"

// NewRootCommand creates and returns the root cobra command for conductor
func NewRootCommand() *cobra.Command {
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

	return cmd
}
