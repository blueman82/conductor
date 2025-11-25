package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayToolAnalysis displays tool usage analysis from the learning database
func DisplayToolAnalysis(project string, limit int) error {
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

	// Get tool stats
	toolStats, err := store.GetToolStats(ctx, project, limit, 0)
	if err != nil {
		return fmt.Errorf("get tool stats: %w", err)
	}

	// Display formatted output
	fmt.Println(formatToolAnalysisTable(toolStats, limit))

	return nil
}

// formatToolAnalysisTable formats tool statistics as a readable table
func formatToolAnalysisTable(tools []learning.ToolStats, limit int) string {
	var sb strings.Builder

	sb.WriteString("\n=== Tool Usage Analysis ===\n\n")

	if len(tools) == 0 {
		sb.WriteString("No tool data available\n")
		return sb.String()
	}

	// Header
	sb.WriteString(fmt.Sprintf("%-35s %-10s %-12s %-12s %-15s %-10s\n",
		"Tool", "Calls", "Success", "Failures", "Avg Duration", "Success%"))
	sb.WriteString(strings.Repeat("-", 95) + "\n")

	// Display rows
	displayLimit := len(tools)
	if limit > 0 && limit < displayLimit {
		displayLimit = limit
	}

	for i := 0; i < displayLimit; i++ {
		tool := tools[i]
		durationStr := "0ms"
		if tool.AvgDurationMs > 0 {
			durationStr = formatMs(tool.AvgDurationMs)
		}
		successPct := 0.0
		if tool.CallCount > 0 {
			successPct = tool.SuccessRate * 100
		}

		sb.WriteString(fmt.Sprintf("%-35s %-10d %-12d %-12d %-15s %-10.1f%%\n",
			truncateString(tool.ToolName, 34),
			tool.CallCount,
			tool.SuccessCount,
			tool.FailureCount,
			durationStr,
			successPct))
	}

	if limit > 0 && len(tools) > displayLimit {
		sb.WriteString(fmt.Sprintf("\n(Showing %d of %d tools. Use --limit flag to see more)\n", displayLimit, len(tools)))
	}

	sb.WriteString("\n")
	return sb.String()
}

// formatMs formats milliseconds as a readable duration
func formatMs(ms float64) string {
	d := time.Duration(int64(ms)) * time.Millisecond
	return formatDuration(d)
}
