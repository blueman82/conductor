package cmd

import (
	"github.com/spf13/cobra"
)

// NewLearningCommand creates the 'conductor learning' parent command
func NewLearningCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learning",
		Short: "Adaptive learning commands",
		Long: `Commands for viewing and managing adaptive learning data.

The learning subsystem tracks task execution history and provides
insights to improve plan execution over time.`,
	}

	// Add subcommands
	cmd.AddCommand(NewStatsCommand())
	cmd.AddCommand(NewShowCommand())
	cmd.AddCommand(newClearCommand())
	cmd.AddCommand(newExportCommand())

	return cmd
}
