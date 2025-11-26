package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
)

// NewObserveExportCmd creates the 'conductor observe export' subcommand
func NewObserveExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export behavioral data to file",
		Long: `Export behavioral metrics to JSON or Markdown format.

Supports filtering via global flags (--project, --session, --filter-type, etc.)
and exports filtered data to specified format.

Examples:
  conductor observe export --format json --output metrics.json
  conductor observe export --format markdown --output report.md
  conductor observe export --format json --project myapp --errors-only
  conductor observe export --format md  # Outputs to stdout`,
		RunE: HandleExportCommand,
	}

	cmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format: json, markdown (or md), csv")
	cmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (empty for stdout)")

	return cmd
}

// HandleExportCommand processes export requests with filtering
func HandleExportCommand(cmd *cobra.Command, args []string) error {
	// Parse filters from global flags
	criteria, err := ParseFilterFlags()
	if err != nil {
		return fmt.Errorf("invalid filter criteria: %w", err)
	}

	// Parse export format
	format, err := parseExportFormat(exportFormat)
	if err != nil {
		return err
	}

	// Load metrics from database (using global observeProject flag)
	metrics, err := loadMetricsFromDB(observeProject)
	if err != nil {
		return fmt.Errorf("load metrics: %w", err)
	}

	// Apply filters to detailed data
	if criteria.ErrorsOnly || criteria.EventType != "" || criteria.Search != "" {
		metrics.ToolExecutions = behavioral.ApplyFiltersToToolExecutions(metrics.ToolExecutions, criteria)
		metrics.BashCommands = behavioral.ApplyFiltersToBashCommands(metrics.BashCommands, criteria)
		metrics.FileOperations = behavioral.ApplyFiltersToFileOperations(metrics.FileOperations, criteria)
	}

	// Export based on output destination
	if exportOutput == "" {
		// Export to stdout
		content, err := behavioral.ExportToString(metrics, format)
		if err != nil {
			return fmt.Errorf("export failed: %w", err)
		}
		fmt.Println(content)
	} else {
		// Export to file
		outputPath, err := getExportPath(exportOutput)
		if err != nil {
			return err
		}

		if err := behavioral.ExportToFile(metrics, outputPath, format); err != nil {
			return fmt.Errorf("export to file failed: %w", err)
		}

		fmt.Printf("Exported metrics to: %s\n", outputPath)
	}

	return nil
}

// parseExportFormat validates and normalizes the export format
func parseExportFormat(format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))

	// Normalize aliases
	if format == "md" {
		format = "markdown"
	}

	// Validate format
	validFormats := map[string]bool{
		"json":     true,
		"markdown": true,
		"csv":      true,
	}

	if !validFormats[format] {
		return "", fmt.Errorf("invalid format '%s': must be one of: json, markdown (or md), csv", format)
	}

	return format, nil
}

// getExportPath resolves and validates the export output path
func getExportPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("output path cannot be empty")
	}

	// Expand home directory if present
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if parent directory exists (or can be created)
	parentDir := filepath.Dir(absPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		// Directory doesn't exist, will be created during export
		return absPath, nil
	}

	return absPath, nil
}

// loadMetricsFromDB loads behavioral metrics from the learning database
func loadMetricsFromDB(project string) (*behavioral.BehavioralMetrics, error) {
	// Get learning DB path (uses build-time injected root)
	dbPath, err := config.GetLearningDBPath()
	if err != nil {
		return nil, fmt.Errorf("get learning db path: %w", err)
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get summary stats
	summary, err := store.GetSummaryStats(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("get summary stats: %w", err)
	}

	// Get tool stats (up to 100)
	toolStats, err := store.GetToolStats(ctx, project, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("get tool stats: %w", err)
	}

	// Get bash stats (up to 100)
	bashStats, err := store.GetBashStats(ctx, project, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("get bash stats: %w", err)
	}

	// Get file stats (up to 100)
	fileStats, err := store.GetFileStats(ctx, project, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("get file stats: %w", err)
	}

	// Get error patterns
	errorPatterns, err := store.GetErrorPatterns(ctx, project, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("get error patterns: %w", err)
	}

	// Get agent stats
	agentStats, err := store.GetAgentTypeStats(ctx, project, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("get agent stats: %w", err)
	}

	// Convert to behavioral metrics
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions:   summary.TotalSessions,
		SuccessRate:     summary.SuccessRate,
		AverageDuration: time.Duration(summary.AvgDurationSeconds) * time.Second,
		TotalCost:       0, // Not tracked in current schema
		TokenUsage: behavioral.TokenUsage{
			InputTokens:  int64(summary.TotalInputTokens),
			OutputTokens: int64(summary.TotalOutputTokens),
		},
	}

	// Calculate error rate and total errors
	totalErrors := 0
	for _, ep := range errorPatterns {
		totalErrors += ep.Count
	}
	metrics.TotalErrors = totalErrors
	if summary.TotalSessions > 0 {
		metrics.ErrorRate = float64(totalErrors) / float64(summary.TotalSessions)
	}

	// Convert tool stats
	for _, ts := range toolStats {
		metrics.ToolExecutions = append(metrics.ToolExecutions, behavioral.ToolExecution{
			Name:         ts.ToolName,
			Count:        ts.CallCount,
			SuccessRate:  ts.SuccessRate,
			ErrorRate:    1.0 - ts.SuccessRate,
			AvgDuration:  time.Duration(int64(ts.AvgDurationMs)) * time.Millisecond,
			TotalSuccess: ts.SuccessCount,
			TotalErrors:  ts.FailureCount,
		})
	}

	// Convert bash stats
	for _, bs := range bashStats {
		metrics.BashCommands = append(metrics.BashCommands, behavioral.BashCommand{
			Command:  bs.Command,
			Success:  bs.SuccessCount > 0,
			Duration: time.Duration(int64(bs.AvgDurationMs)) * time.Millisecond,
			ExitCode: 0,
		})
	}

	// Convert file stats
	for _, fs := range fileStats {
		metrics.FileOperations = append(metrics.FileOperations, behavioral.FileOperation{
			Path:      fs.FilePath,
			Type:      fs.OperationType,
			Success:   fs.SuccessCount > 0,
			Duration:  int64(fs.AvgDurationMs),
			SizeBytes: fs.TotalBytes,
		})
	}

	// Convert agent stats to simple map
	agentPerf := make(map[string]int)
	for _, as := range agentStats {
		agentPerf[as.AgentType] = as.SuccessCount
	}
	metrics.AgentPerformance = agentPerf

	return metrics, nil
}
