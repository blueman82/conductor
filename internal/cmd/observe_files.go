package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
)

// DisplayFileAnalysis displays file operation analysis from the learning database
func DisplayFileAnalysis(project string, limit int) error {
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

	// Get file stats
	fileStats, err := store.GetFileStats(ctx, project, limit, 0)
	if err != nil {
		return fmt.Errorf("get file stats: %w", err)
	}

	// Display formatted output
	fmt.Println(formatFileAnalysisTable(fileStats, limit))

	return nil
}

// formatFileAnalysisTable formats file operation statistics as a readable table
func formatFileAnalysisTable(files []learning.FileStats, limit int) string {
	var sb strings.Builder

	sb.WriteString("\n=== File Operation Analysis ===\n\n")

	if len(files) == 0 {
		sb.WriteString("No file operation data available\n")
		return sb.String()
	}

	// Header
	sb.WriteString(fmt.Sprintf("%-50s %-12s %-10s %-12s %-12s %-15s %-12s\n",
		"File", "Op Type", "Calls", "Success", "Failures", "Avg Duration", "Total Bytes"))
	sb.WriteString(strings.Repeat("-", 125) + "\n")

	// Display rows
	displayLimit := len(files)
	if limit > 0 && limit < displayLimit {
		displayLimit = limit
	}

	for i := 0; i < displayLimit; i++ {
		file := files[i]
		durationStr := "0ms"
		if file.AvgDurationMs > 0 {
			durationStr = formatMs(file.AvgDurationMs)
		}

		sb.WriteString(fmt.Sprintf("%-50s %-12s %-10d %-12d %-12d %-15s %-12d\n",
			truncateString(file.FilePath, 49),
			file.OperationType,
			file.OpCount,
			file.SuccessCount,
			file.FailureCount,
			durationStr,
			file.TotalBytes))
	}

	if limit > 0 && len(files) > displayLimit {
		sb.WriteString(fmt.Sprintf("\n(Showing %d of %d file operations. Use --limit flag to see more)\n", displayLimit, len(files)))
	}

	sb.WriteString("\n")
	return sb.String()
}
