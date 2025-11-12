package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/harrison/conductor/internal/learning"
	"github.com/spf13/cobra"
)

// newClearCommand creates the 'conductor learning clear' command
func newClearCommand() *cobra.Command {
	var clearAll bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "clear [plan-file]",
		Short: "Clear learning data",
		Long: `Clear learning data for a specific plan file or the entire database.

Examples:
  # Clear data for a specific plan (requires confirmation)
  conductor learning clear plan.md

  # Clear all learning data (requires confirmation)
  conductor learning clear --all`,
		Args: func(cmd *cobra.Command, args []string) error {
			clearAll, _ := cmd.Flags().GetBool("all")
			if clearAll && len(args) > 0 {
				return fmt.Errorf("cannot specify plan file when using --all flag")
			}
			if !clearAll && len(args) != 1 {
				return fmt.Errorf("requires plan file argument or --all flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClear(cmd, args, clearAll, dbPath)
		},
	}

	cmd.Flags().BoolVar(&clearAll, "all", false, "Clear entire database")
	cmd.Flags().StringVar(&dbPath, "db-path", "", "Path to learning database (for testing)")

	return cmd
}

// runClear executes the clear command
func runClear(cmd *cobra.Command, args []string, clearAll bool, dbPathOverride string) error {
	output := cmd.OutOrStdout()

	var planFile string
	var dbPath string

	if clearAll {
		// Confirm clearing all data
		fmt.Fprintf(output, "WARNING: This will delete ALL learning data from the database.\n")
		if !confirmAction(output) {
			fmt.Fprintf(output, "Operation cancelled.\n")
			return nil
		}

		// Use override path if provided (for testing), otherwise use default
		if dbPathOverride != "" {
			dbPath = dbPathOverride
		} else {
			// For --all, we need to determine the database path
			// Use current directory as reference
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			dbPath = getLearningDBPath(cwd)
		}
	} else {
		planFile = args[0]

		// Use override path if provided (for testing), otherwise use default
		if dbPathOverride != "" {
			dbPath = dbPathOverride
		} else {
			// Resolve plan file path and get database path
			absPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			dbPath = getLearningDBPath(absPath)
		}

		// Confirm clearing specific plan data
		fmt.Fprintf(output, "This will delete all learning data for plan: %s\n", planFile)
		if !confirmAction(output) {
			fmt.Fprintf(output, "Operation cancelled.\n")
			return nil
		}
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintf(output, "No learning database found at: %s\n", dbPath)
		return nil
	}

	// Open learning store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open learning store: %w", err)
	}
	defer store.Close()

	// Execute delete query
	var deletedCount int64
	if clearAll {
		// Delete all records
		result, err := store.Exec("DELETE FROM task_executions")
		if err != nil {
			return fmt.Errorf("delete records: %w", err)
		}
		deletedCount, _ = result.RowsAffected()
	} else {
		// Delete records for specific plan
		result, err := store.Exec("DELETE FROM task_executions WHERE plan_file = ?", planFile)
		if err != nil {
			return fmt.Errorf("delete records: %w", err)
		}
		deletedCount, _ = result.RowsAffected()
	}

	// Report results
	recordText := "record"
	if deletedCount != 1 {
		recordText = "records"
	}
	fmt.Fprintf(output, "Deleted %d %s.\n", deletedCount, recordText)

	return nil
}

// confirmAction prompts the user for confirmation
func confirmAction(output interface{}) bool {
	// Create scanner for stdin
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Fprintf(output.(interface{ Write(p []byte) (n int, err error) }), "Continue? [y/N]: ")

	if !scanner.Scan() {
		return false
	}

	response := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return response == "y" || response == "yes"
}
