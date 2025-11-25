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
	// Load config to get DB path
	cfg, err := config.LoadConfigFromDir(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(cfg.Learning.DBPath)
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

	// Header
	sb.WriteString(fmt.Sprintf("%-50s %-10s %-12s %-12s %-15s %-10s\n",
		"Command", "Calls", "Success", "Failures", "Avg Duration", "Success%"))
	sb.WriteString(strings.Repeat("-", 110) + "\n")

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

		sb.WriteString(fmt.Sprintf("%-50s %-10d %-12d %-12d %-15s %-10.1f%%\n",
			truncateString(cmd.Command, 49),
			cmd.CallCount,
			cmd.SuccessCount,
			cmd.FailureCount,
			durationStr,
			successPct))
	}

	if limit > 0 && len(commands) > displayLimit {
		sb.WriteString(fmt.Sprintf("\n(Showing %d of %d commands. Use --limit flag to see more)\n", displayLimit, len(commands)))
	}

	sb.WriteString("\n")
	return sb.String()
}
