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

	// Header - Tool at end for full display
	sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %-12s %-8s %s\n",
		"Calls", "Success", "Failures", "Duration", "Rate", "Tool"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

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

		sb.WriteString(fmt.Sprintf("%-10d %-10d %-10d %-12s %6.1f%%  %s\n",
			tool.CallCount,
			tool.SuccessCount,
			tool.FailureCount,
			durationStr,
			successPct,
			tool.ToolName))
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
