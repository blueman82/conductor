package cmd

import (
	"github.com/spf13/cobra"
)

const Version = "1.0.0"

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

	// Future subcommands (run, validate) will be added here
	// in subsequent tasks

	return cmd
}
