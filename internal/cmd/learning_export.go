package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	var format string
	var output string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "export [plan-file]",
		Short: "Export learning data to JSON or CSV format",
		Long: `Export learning data to JSON or CSV format for external analysis or backup.

The command exports all execution records for the specified plan file.
If no output file is specified, data is written to stdout.

Examples:
  # Export to JSON file
  conductor learning export plan.md --format json --output export.json

  # Export to CSV file
  conductor learning export plan.md --format csv --output export.csv

  # Export to stdout
  conductor learning export plan.md --format json

Supported formats:
  - json: JSON array of execution records
  - csv: CSV with headers`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			planFile := args[0]
			return runExport(planFile, format, output, dbPath)
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "Export format (json|csv)")
	cmd.Flags().StringVar(&output, "output", "", "Output file path (stdout if not specified)")
	cmd.Flags().StringVar(&dbPath, "db-path", "", "Path to learning database (default: ~/.conductor/learning.db)")

	return cmd
}

func runExport(planFile, format, output, dbPathOverride string) error {
	// Validate format
	if format != "json" && format != "csv" {
		return fmt.Errorf("invalid format '%s': format must be 'json' or 'csv'", format)
	}

	// Determine database path: use override if provided (for testing), otherwise use centralized location
	var dbPath string
	if dbPathOverride != "" {
		dbPath = dbPathOverride
	} else {
		// Use centralized conductor home database location
		var err error
		dbPath, err = config.GetLearningDBPath()
		if err != nil {
			return fmt.Errorf("failed to get learning database path: %w", err)
		}
	}

	// Initialize store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize learning store: %w", err)
	}
	defer store.Close()

	// Get all executions for plan file
	executions, err := store.GetExecutions(planFile)
	if err != nil {
		return fmt.Errorf("failed to retrieve executions: %w", err)
	}

	// Initialize empty slice if nil to ensure JSON output is [] not null
	if executions == nil {
		executions = make([]*learning.TaskExecution, 0)
	}

	// Determine output destination
	var writer io.Writer
	if output == "" {
		writer = os.Stdout
	} else {
		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// Export based on format
	switch format {
	case "json":
		return exportJSON(writer, executions)
	case "csv":
		return exportCSV(writer, executions)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportJSON(writer io.Writer, executions []*learning.TaskExecution) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(executions); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func exportCSV(writer io.Writer, executions []*learning.TaskExecution) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"id",
		"plan_file",
		"run_number",
		"task_number",
		"task_name",
		"agent",
		"success",
		"error_message",
		"duration_seconds",
		"timestamp",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, exec := range executions {
		row := []string{
			strconv.FormatInt(exec.ID, 10),
			exec.PlanFile,
			strconv.Itoa(exec.RunNumber),
			exec.TaskNumber,
			exec.TaskName,
			exec.Agent,
			strconv.FormatBool(exec.Success),
			exec.ErrorMessage,
			strconv.FormatInt(exec.DurationSecs, 10),
			exec.Timestamp.Format("2006-01-02 15:04:05"),
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}
