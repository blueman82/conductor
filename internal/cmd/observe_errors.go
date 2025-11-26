package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayErrorAnalysis displays error pattern analysis from the learning database
func DisplayErrorAnalysis(project string, limit int) error {
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

	// Get error patterns
	patterns, err := store.GetErrorPatterns(ctx, project, limit, 0)
	if err != nil {
		return fmt.Errorf("get error patterns: %w", err)
	}

	// Display formatted output
	fmt.Println(formatErrorAnalysisTable(patterns, limit))

	return nil
}

// formatErrorAnalysisTable formats error patterns as a readable table
func formatErrorAnalysisTable(patterns []learning.ErrorPattern, limit int) string {
	var sb strings.Builder

	sb.WriteString("\n=== Error Pattern Analysis ===\n\n")

	if len(patterns) == 0 {
		sb.WriteString("No error data available\n")
		return sb.String()
	}

	// Header
	sb.WriteString(fmt.Sprintf("%-15s %-12s %-20s %s\n",
		"Type", "Count", "Last Occurred", "Component / Error"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	// Display rows
	displayLimit := len(patterns)
	if limit > 0 && limit < displayLimit {
		displayLimit = limit
	}

	for i := 0; i < displayLimit; i++ {
		pattern := patterns[i]

		// Determine component name based on error type
		component := ""
		switch pattern.ErrorType {
		case "tool":
			component = pattern.Tool
		case "bash":
			component = pattern.Command
		case "file":
			if pattern.OperationType != "" {
				component = pattern.OperationType + ": " + pattern.FilePath
			} else {
				component = pattern.FilePath
			}
		default:
			component = "unknown"
		}

		lastOccurred := ""
		if !pattern.LastOccurred.IsZero() {
			lastOccurred = pattern.LastOccurred.Format("2006-01-02 15:04:05")
		}

		sb.WriteString(fmt.Sprintf("%-15s %-12d %-20s %s\n",
			pattern.ErrorType,
			pattern.Count,
			lastOccurred,
			component))
		if pattern.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("%-15s %-12s %-20s   → %s\n", "", "", "", pattern.ErrorMessage))
		}
	}

	if limit > 0 && len(patterns) > displayLimit {
		sb.WriteString(fmt.Sprintf("\n(Showing %d of %d errors. Use --limit flag to see more)\n", displayLimit, len(patterns)))
	}

	// Summary statistics
	sb.WriteString("\n--- Error Summary ---\n")
	toolErrors := 0
	bashErrors := 0
	fileErrors := 0
	totalErrors := 0

	for _, pattern := range patterns {
		totalErrors += pattern.Count
		switch pattern.ErrorType {
		case "tool":
			toolErrors += pattern.Count
		case "bash":
			bashErrors += pattern.Count
		case "file":
			fileErrors += pattern.Count
		}
	}

	sb.WriteString(fmt.Sprintf("Total Errors: %d\n", totalErrors))
	if toolErrors > 0 {
		sb.WriteString(fmt.Sprintf("Tool Errors: %d (%.1f%%)\n", toolErrors, float64(toolErrors)*100/float64(totalErrors)))
	}
	if bashErrors > 0 {
		sb.WriteString(fmt.Sprintf("Bash Errors: %d (%.1f%%)\n", bashErrors, float64(bashErrors)*100/float64(totalErrors)))
	}
	if fileErrors > 0 {
		sb.WriteString(fmt.Sprintf("File Errors: %d (%.1f%%)\n", fileErrors, float64(fileErrors)*100/float64(totalErrors)))
	}

	sb.WriteString("\n")
	return sb.String()
}
