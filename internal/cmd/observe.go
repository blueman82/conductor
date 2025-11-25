package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/behavioral"
	"github.com/spf13/cobra"
)

// Global flags for observe commands
var (
	observeProject      string
	observeSession      string
	observeFilterType   string
	observeErrorsOnly   bool
	observeTimeRange    string
	observePollInterval string
	observeLimit        int
	observeWithIngest   bool
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
	cmd.PersistentFlags().StringVar(&observePollInterval, "poll-interval", "2s", "Poll interval for streaming (default: 2s)")
	cmd.PersistentFlags().IntVarP(&observeLimit, "limit", "n", 20, "Limit number of results (0 for all)")

	// Add subcommands
	cmd.AddCommand(NewObserveImportCmd())
	cmd.AddCommand(NewObserveIngestCmd())
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

	// Run interactive session selection
	return runInteractiveSessionSelection(observeProject)
}

// runInteractiveSessionSelection displays project summary and session selection menu
func runInteractiveSessionSelection(project string) error {
	return runInteractiveSessionSelectionWithReader(project, &DefaultMenuReader{
		reader: bufio.NewReader(os.Stdin),
	})
}

// runInteractiveSessionSelectionWithReader allows injection of reader for testing
func runInteractiveSessionSelectionWithReader(project string, reader MenuReader) error {
	aggregator := behavioral.NewAggregator(50)
	cyan := color.New(color.FgCyan)
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	for {
		// Get sessions for project
		sessions, err := aggregator.ListSessions(project)
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		// Show quick project summary
		bold.Printf("\n=== Quick Summary ===\n")
		successCount := 0
		var lastActivity time.Time
		for _, s := range sessions {
			// Approximate success by checking file size (real success detection would require parsing)
			if s.FileSize > 1000 {
				successCount++
			}
			if s.CreatedAt.After(lastActivity) {
				lastActivity = s.CreatedAt
			}
		}
		successRate := 0.0
		if len(sessions) > 0 {
			successRate = float64(successCount) / float64(len(sessions)) * 100
		}
		lastActivityStr := "N/A"
		if !lastActivity.IsZero() {
			lastActivityStr = formatTimeAgo(lastActivity)
		}
		fmt.Printf("Sessions: %d | Success: %.0f%% | Last: %s\n", len(sessions), successRate, lastActivityStr)

		if len(sessions) == 0 {
			fmt.Println("\nNo sessions found for this project.")
			return nil
		}

		// Show recent sessions (max 10)
		bold.Printf("\n=== Recent Sessions ===\n")
		fmt.Println(strings.Repeat("-", 60))
		displayCount := 10
		if len(sessions) < displayCount {
			displayCount = len(sessions)
		}

		for i := 0; i < displayCount; i++ {
			session := sessions[i]
			// Estimate duration from file size (rough heuristic)
			durationStr := estimateDuration(session.FileSize)
			// Determine status based on file size (rough heuristic)
			status := "success"
			statusColor := green
			if session.FileSize < 1000 {
				status = "failed"
				statusColor = red
			}
			timeStr := session.CreatedAt.Format("15:04")
			shortID := session.SessionID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			fmt.Printf("  %s %-10s %8s  %s  %s\n",
				yellow.Sprintf("[%d]", i+1),
				shortID,
				durationStr,
				statusColor.Sprint(status),
				timeStr,
			)
		}

		if len(sessions) > displayCount {
			fmt.Printf("  (showing %d of %d sessions)\n", displayCount, len(sessions))
		}

		fmt.Println(strings.Repeat("-", 60))
		cyan.Print("\nEnter session number (or 'q' to quit): ")

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "q" {
			fmt.Println("Exiting interactive mode.")
			return nil
		}

		// Parse selection
		var selection int
		_, err = fmt.Sscanf(input, "%d", &selection)
		if err != nil || selection < 1 || selection > displayCount {
			red.Printf("Invalid selection. Please enter 1-%d or 'q' to quit.\n", displayCount)
			continue
		}

		// Display selected session
		selectedSession := sessions[selection-1]
		fmt.Println()
		if err := DisplaySessionAnalysis(selectedSession.SessionID, project); err != nil {
			red.Printf("Error displaying session: %v\n", err)
		}

		// Prompt to continue
		cyan.Print("\nPress Enter to return to session list (or 'q' to quit): ")
		input, err = reader.ReadString('\n')
		if err != nil {
			return nil
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "q" {
			fmt.Println("Exiting interactive mode.")
			return nil
		}
	}
}

// formatTimeAgo formats a time as "Xh ago", "Xm ago", etc.
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	default:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

// estimateDuration estimates session duration from file size (rough heuristic)
func estimateDuration(fileSize int64) string {
	// Rough estimate: ~1KB per second of activity
	seconds := fileSize / 1024
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
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
			// Resolve project name: args[0] > --project flag
			project := observeProject
			if len(args) > 0 {
				project = args[0]
			}
			return DisplayProjectAnalysis(project, observeLimit)
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
			// Session ID is required
			if len(args) == 0 {
				return fmt.Errorf("session ID is required")
			}
			sessionID := args[0]

			// Project is optional (from --project flag)
			return DisplaySessionAnalysis(sessionID, observeProject)
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
			return DisplayToolAnalysis(observeProject, observeLimit)
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
			return DisplayBashAnalysis(observeProject, observeLimit)
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
			return DisplayFileAnalysis(observeProject, observeLimit)
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
			return DisplayErrorAnalysis(observeProject, observeLimit)
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
- Agent performance breakdown grouped by agent type`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return DisplayStats(observeProject, observeLimit)
		},
	}
	return cmd
}

// NewObserveStreamCmd creates the 'conductor observe stream' subcommand
func NewObserveStreamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Stream real-time activity",
		Long: `Watch and display agent activity in real-time, like 'tail -f'.
Polls the behavioral database for new sessions and displays them as they occur.
Only shows activity that occurs after the command starts.

With --with-ingest flag, spawns an ingestion daemon for real-time JSONL processing,
enabling live data to appear in the stream as events occur.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Parse poll interval
			pollInterval, err := parsePollInterval(observePollInterval)
			if err != nil {
				return fmt.Errorf("invalid poll-interval: %w", err)
			}

			return StreamActivity(ctx, observeProject, pollInterval, observeWithIngest)
		},
	}

	cmd.Flags().BoolVar(&observeWithIngest, "with-ingest", false,
		"Spawn ingestion daemon for real-time JSONL processing")

	return cmd
}

// parsePollInterval parses a duration string like "2s", "500ms", "1m"
func parsePollInterval(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
