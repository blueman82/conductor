package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

// NewShowCommand creates the 'conductor learning show' command
func NewShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <plan-file> <task-number>",
		Short: "Show detailed history for a task",
		Long: `Display detailed execution history for a specific task including:
  - All execution attempts
  - Agents used for each attempt
  - Success/failure verdicts
  - Timestamps and durations
  - Output and error messages`,
		Args: cobra.ExactArgs(2),
		RunE: runShow,
	}

	return cmd
}

// runShow executes the show command
func runShow(cmd *cobra.Command, args []string) error {
	planFile := args[0]
	taskNumber := args[1]
	output := cmd.OutOrStdout()

	// Resolve plan file path
	absPath, err := filepath.Abs(planFile)
	if err != nil {
		return fmt.Errorf("resolve plan file path: %w", err)
	}

	// Check if plan file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("plan file not found: %s", absPath)
	}

	// Use centralized conductor home database location
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("failed to get learning database path: %w", err)
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintf(output, "No execution data found for plan: %s\n", filepath.Base(absPath))
		return nil
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	// Get execution history for the task
	ctx := context.Background()
	executions, err := store.GetExecutionHistory(ctx, filepath.Base(absPath), taskNumber)
	if err != nil {
		return fmt.Errorf("get execution history: %w", err)
	}

	// Check if we have any executions
	if len(executions) == 0 {
		fmt.Fprintf(output, "No execution history found for Task %s\n", taskNumber)
		return nil
	}

	// Print execution history
	printExecutionHistory(output, taskNumber, executions)

	return nil
}

// printExecutionHistory formats and prints the execution history for a task
func printExecutionHistory(w io.Writer, taskNumber string, executions []*learning.TaskExecution) {
	// Colors
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	gray := color.New(color.FgHiBlack)

	// Get task name from first execution
	taskName := "Unknown"
	if len(executions) > 0 {
		taskName = executions[0].TaskName
	}

	// Header
	cyan.Fprintf(w, "\n=== Execution History for Task %s: %s ===\n\n", taskNumber, taskName)
	fmt.Fprintf(w, "Total attempts: %d\n\n", len(executions))

	// Print executions in chronological order (most recent first)
	for i, exec := range executions {
		// Attempt number (reverse order since we show most recent first)
		attemptNum := len(executions) - i
		cyan.Fprintf(w, "Attempt #%d\n", attemptNum)

		// Timestamp
		fmt.Fprintf(w, "  Time: %s ", formatTimestamp(exec.Timestamp))
		gray.Fprintf(w, "(%s ago)\n", formatDuration(time.Since(exec.Timestamp)))

		// Agent
		fmt.Fprintf(w, "  Agent: ")
		if exec.Agent != "" {
			yellow.Fprintf(w, "%s\n", exec.Agent)
		} else {
			fmt.Fprintf(w, "none\n")
		}

		// Verdict
		fmt.Fprintf(w, "  Verdict: ")
		if exec.Success {
			green.Fprintf(w, "GREEN (Success)\n")
		} else {
			red.Fprintf(w, "RED (Failed)\n")
		}

		// Duration
		fmt.Fprintf(w, "  Duration: %d seconds\n", exec.DurationSecs)

		// Output (truncated if too long)
		fmt.Fprintf(w, "  Output: ")
		output := strings.TrimSpace(exec.Output)
		if output == "" {
			gray.Fprintf(w, "(no output)\n")
		} else {
			// Truncate long output
			const maxOutputLen = 200
			if len(output) > maxOutputLen {
				output = output[:maxOutputLen] + "..."
			}
			// Replace newlines with spaces for compact display
			output = strings.ReplaceAll(output, "\n", " ")
			fmt.Fprintf(w, "%s\n", output)
		}

		// Error message (if any)
		if exec.ErrorMessage != "" {
			fmt.Fprintf(w, "  Error: ")
			red.Fprintf(w, "%s\n", strings.TrimSpace(exec.ErrorMessage))
		}

		// Separator between attempts
		if i < len(executions)-1 {
			fmt.Fprintln(w)
		}
	}

	fmt.Fprintln(w)

	// Summary statistics
	successCount := 0
	totalDuration := int64(0)
	agentUsage := make(map[string]int)

	for _, exec := range executions {
		if exec.Success {
			successCount++
		}
		totalDuration += exec.DurationSecs
		if exec.Agent != "" {
			agentUsage[exec.Agent]++
		}
	}

	cyan.Fprintf(w, "Summary:\n")
	fmt.Fprintf(w, "  Success rate: ")
	successRate := float64(successCount) / float64(len(executions)) * 100
	if successRate >= 70 {
		green.Fprintf(w, "%.1f%%", successRate)
	} else if successRate >= 40 {
		yellow.Fprintf(w, "%.1f%%", successRate)
	} else {
		red.Fprintf(w, "%.1f%%", successRate)
	}
	fmt.Fprintf(w, " (%d/%d)\n", successCount, len(executions))

	avgDuration := float64(totalDuration) / float64(len(executions))
	fmt.Fprintf(w, "  Average duration: %.1f seconds\n", avgDuration)

	if len(agentUsage) > 0 {
		fmt.Fprintf(w, "  Agents used: ")
		agentNames := make([]string, 0, len(agentUsage))
		for agent := range agentUsage {
			agentNames = append(agentNames, agent)
		}
		fmt.Fprintf(w, "%s\n", strings.Join(agentNames, ", "))
	}

	fmt.Fprintln(w)
}

// formatTimestamp formats a timestamp for display
func formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// formatDuration formats a duration for human-readable display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}
