package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayBashAnalysis displays bash command analysis from the learning database
func DisplayBashAnalysis(project string, limit int) error {
	// Get learning DB path (uses build-time injected root)
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return fmt.Errorf("get learning db path: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get bash stats
	bashStats, err := store.GetBashStats(ctx, project, limit, 0)
	if err != nil {
		return fmt.Errorf("get bash stats: %w", err)
	}

	// Display formatted output
	fmt.Println(formatBashAnalysisTable(bashStats, limit))

	return nil
}

// formatBashAnalysisTable formats bash command statistics as a readable table
func formatBashAnalysisTable(commands []learning.BashStats, limit int) string {
	var sb strings.Builder

	sb.WriteString("\n=== Bash Command Analysis ===\n\n")

	if len(commands) == 0 {
		sb.WriteString("No bash command data available\n")
		return sb.String()
	}

	// Header - Command at end for full display
	sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %-12s %-8s %s\n",
		"Calls", "Success", "Failures", "Duration", "Rate", "Command"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	// Display rows
	displayLimit := len(commands)
	if limit > 0 && limit < displayLimit {
		displayLimit = limit
	}

	for i := 0; i < displayLimit; i++ {
		cmd := commands[i]
		durationStr := "0ms"
		if cmd.AvgDurationMs > 0 {
			durationStr = formatMs(cmd.AvgDurationMs)
		}
		successPct := 0.0
		if cmd.CallCount > 0 {
			successPct = cmd.SuccessRate * 100
		}

		sb.WriteString(fmt.Sprintf("%-10d %-10d %-10d %-12s %6.1f%%  %s\n",
			cmd.CallCount,
			cmd.SuccessCount,
			cmd.FailureCount,
			durationStr,
			successPct,
			cmd.Command))
	}

	if limit > 0 && len(commands) > displayLimit {
		sb.WriteString(fmt.Sprintf("\n(Showing %d of %d commands. Use --limit flag to see more)\n", displayLimit, len(commands)))
	}

	sb.WriteString("\n")
	return sb.String()
}
