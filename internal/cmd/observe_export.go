package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harrison/conductor/internal/behavioral"
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

	cmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format: json, markdown (or md)")
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

	// TODO: Load metrics from behavioral database
	// For now, return placeholder metrics
	metrics := &behavioral.BehavioralMetrics{
		TotalSessions: 0,
		SuccessRate:   0.0,
		ErrorRate:     0.0,
	}

	// Apply filters to metrics
	// TODO: Implement actual filtering when database integration is complete
	_ = criteria

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
	}

	if !validFormats[format] {
		return "", fmt.Errorf("invalid format '%s': must be one of: json, markdown (or md)", format)
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
