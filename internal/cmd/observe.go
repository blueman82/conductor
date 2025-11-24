package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Global flags for observe commands
var (
	observeProject    string
	observeSession    string
	observeFilterType string
	observeErrorsOnly bool
	observeTimeRange  string
)

// NewObserveCommand creates the 'conductor observe' parent command
func NewObserveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observe",
		Short: "Agent Watch observability commands",
		Long: `Commands for observing and analyzing Claude Code agent behavior.

The observe subsystem provides insights into agent sessions, tool usage,
file operations, errors, and behavioral patterns extracted from agent
execution history.`,
		RunE: observeInteractive,
	}

	// Global flags available to all subcommands
	cmd.PersistentFlags().StringVarP(&observeProject, "project", "p", "", "Filter by project name")
	cmd.PersistentFlags().StringVarP(&observeSession, "session", "s", "", "Filter by session ID")
	cmd.PersistentFlags().StringVar(&observeFilterType, "filter-type", "", "Filter by type (tool, bash, file)")
	cmd.PersistentFlags().BoolVar(&observeErrorsOnly, "errors-only", false, "Show only errors")
	cmd.PersistentFlags().StringVar(&observeTimeRange, "time-range", "", "Time range (e.g., '24h', '7d', '30d')")

	// Add subcommands
	cmd.AddCommand(NewObserveProjectCmd())
	cmd.AddCommand(NewObserveSessionCmd())
	cmd.AddCommand(NewObserveToolsCmd())
	cmd.AddCommand(NewObserveBashCmd())
	cmd.AddCommand(NewObserveFilesCmd())
	cmd.AddCommand(NewObserveErrorsCmd())
	cmd.AddCommand(NewObserveStatsCmd())
	cmd.AddCommand(NewObserveStreamCmd())
	cmd.AddCommand(NewObserveExportCmd())

	return cmd
}

// observeInteractive is the default interactive mode when no subcommand is specified
func observeInteractive(cmd *cobra.Command, args []string) error {
	fmt.Println("Agent Watch Interactive Mode")
	fmt.Println("=============================")

	// If project flag not set, show interactive menu
	if observeProject == "" {
		selectedProject, err := DisplayProjectMenu()
		if err != nil {
			return fmt.Errorf("project selection failed: %w", err)
		}
		observeProject = selectedProject
		fmt.Printf("\nSelected project: %s\n", observeProject)
	}

	// TODO: After project selection, show session selection or project summary
	fmt.Println()
	fmt.Println("Available subcommands:")
	fmt.Println("  project   - View project-level metrics")
	fmt.Println("  session   - Analyze specific session")
	fmt.Println("  tools     - Tool usage analysis")
	fmt.Println("  bash      - Bash command analysis")
	fmt.Println("  files     - File operation analysis")
	fmt.Println("  errors    - Error analysis and patterns")
	fmt.Println("  stats     - Display summary statistics")
	fmt.Println("  stream    - Stream real-time activity")
	fmt.Println()
	fmt.Println("Use 'conductor observe <subcommand> --help' for more information")

	return nil
}

// NewObserveProjectCmd creates the 'conductor observe project' subcommand
func NewObserveProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project [project-name]",
		Short: "View project-level behavioral metrics",
		Long: `Display aggregated metrics for a specific project including:
- Session count and success rate
- Tool usage patterns
- File operation statistics
- Error rates and common issues
- Agent performance comparison`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement project-level analysis
			fmt.Println("Project-level analysis not yet implemented")
			return nil
		},
	}
	return cmd
}

// NewObserveSessionCmd creates the 'conductor observe session' subcommand
func NewObserveSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session [session-id]",
		Short: "Analyze a specific agent session",
		Long: `Display detailed metrics for a specific session including:
- Session metadata (duration, status, agent used)
- Tool execution timeline
- Bash command history
- File operations
- Token usage and cost
- Errors and issues`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement session analysis
			fmt.Println("Session analysis not yet implemented")
			return nil
		},
	}
	return cmd
}

// NewObserveToolsCmd creates the 'conductor observe tools' subcommand
func NewObserveToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Analyze tool usage patterns",
		Long: `Display tool usage statistics including:
- Tool execution counts
- Success and error rates per tool
- Average execution duration
- Most/least used tools
- Tool usage trends over time`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement tool usage analysis
			fmt.Println("Tool usage analysis not yet implemented")
			return nil
		},
	}
	return cmd
}

// NewObserveBashCmd creates the 'conductor observe bash' subcommand
func NewObserveBashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Analyze bash command patterns",
		Long: `Display bash command statistics including:
- Command frequency
- Exit codes and success rates
- Most common commands
- Failed commands
- Output patterns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement bash command analysis
			fmt.Println("Bash command analysis not yet implemented")
			return nil
		},
	}
	return cmd
}

// NewObserveFilesCmd creates the 'conductor observe files' subcommand
func NewObserveFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Analyze file operation patterns",
		Long: `Display file operation statistics including:
- Read/Write/Edit/Delete counts
- Most accessed files
- File size distributions
- Operation success rates
- File modification patterns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement file operation analysis
			fmt.Println("File operation analysis not yet implemented")
			return nil
		},
	}
	return cmd
}

// NewObserveErrorsCmd creates the 'conductor observe errors' subcommand
func NewObserveErrorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Analyze error patterns and issues",
		Long: `Display error analysis including:
- Error frequency by type
- Common error patterns
- Error rates over time
- Tool/bash errors breakdown
- Failed operations summary`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement error analysis
			fmt.Println("Error analysis not yet implemented")
			return nil
		},
	}
	return cmd
}

// NewObserveStatsCmd creates the 'conductor observe stats' subcommand
func NewObserveStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Display summary statistics",
		Long: `Display aggregate statistics including:
- Total sessions and agents
- Success and error rates
- Average duration
- Token usage and cost
- Top tools by usage
- Agent performance breakdown`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return DisplayStats(observeProject)
		},
	}
	return cmd
}

// NewObserveStreamCmd creates the 'conductor observe stream' subcommand
func NewObserveStreamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Stream real-time activity",
		Long: `Watch and display agent activity in real-time.
Polls the behavioral database for new sessions and displays them as they occur.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return StreamActivity(ctx, observeProject)
		},
	}
	return cmd
}
