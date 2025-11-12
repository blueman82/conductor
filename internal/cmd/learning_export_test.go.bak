package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/learning"
)

func TestExportCommand_JSON(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "learning.db")

	// Initialize store and add test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add test executions
	exec1 := &learning.TaskExecution{
		PlanFile:     "test-plan.md",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task 1",
		Agent:        "test-agent",
		Prompt:       "Test prompt 1",
		Success:      true,
		Output:       "Test output 1",
		ErrorMessage: "",
		DurationSecs: 30,
	}
	exec2 := &learning.TaskExecution{
		PlanFile:     "test-plan.md",
		RunNumber:    1,
		TaskNumber:   "2",
		TaskName:     "Test Task 2",
		Agent:        "test-agent",
		Prompt:       "Test prompt 2",
		Success:      false,
		Output:       "Test output 2",
		ErrorMessage: "test error",
		DurationSecs: 45,
	}

	if err := store.RecordExecution(context.Background(), exec1); err != nil {
		t.Fatalf("Failed to record execution 1: %v", err)
	}
	if err := store.RecordExecution(context.Background(), exec2); err != nil {
		t.Fatalf("Failed to record execution 2: %v", err)
	}

	// Create temp output file
	outputPath := filepath.Join(tmpDir, "export.json")

	// Create command
	cmd := newExportCommand()
	cmd.SetArgs([]string{"test-plan.md", "--format", "json", "--output", outputPath, "--db-path", dbPath})

	// Execute command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Read and verify output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Parse JSON
	var executions []*learning.TaskExecution
	if err := json.Unmarshal(data, &executions); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify data
	if len(executions) != 2 {
		t.Errorf("Expected 2 executions, got %d", len(executions))
	}

	// Verify executions (note: ordered by ID DESC, so exec2 comes first)
	if executions[0].TaskNumber != "2" {
		t.Errorf("Expected task number '2', got '%s'", executions[0].TaskNumber)
	}
	if executions[0].Success {
		t.Errorf("Expected success=false, got true")
	}
	if executions[0].Agent != "test-agent" {
		t.Errorf("Expected agent 'test-agent', got '%s'", executions[0].Agent)
	}
	if executions[0].ErrorMessage != "test error" {
		t.Errorf("Expected error msg 'test error', got '%s'", executions[0].ErrorMessage)
	}

	// Verify second execution
	if executions[1].TaskNumber != "1" {
		t.Errorf("Expected task number '1', got '%s'", executions[1].TaskNumber)
	}
	if !executions[1].Success {
		t.Errorf("Expected success=true, got false")
	}
}

func TestExportCommand_CSV(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "learning.db")

	// Initialize store and add test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add test executions
	exec1 := &learning.TaskExecution{
		PlanFile:     "test-plan.md",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task 1",
		Agent:        "test-agent",
		Prompt:       "Test prompt 1",
		Success:      true,
		Output:       "Test output 1",
		ErrorMessage: "",
		DurationSecs: 30,
	}
	exec2 := &learning.TaskExecution{
		PlanFile:     "test-plan.md",
		RunNumber:    1,
		TaskNumber:   "2",
		TaskName:     "Test Task 2",
		Agent:        "test-agent",
		Prompt:       "Test prompt 2",
		Success:      false,
		Output:       "Test output 2",
		ErrorMessage: "test error",
		DurationSecs: 45,
	}

	if err := store.RecordExecution(context.Background(), exec1); err != nil {
		t.Fatalf("Failed to record execution 1: %v", err)
	}
	if err := store.RecordExecution(context.Background(), exec2); err != nil {
		t.Fatalf("Failed to record execution 2: %v", err)
	}

	// Create temp output file
	outputPath := filepath.Join(tmpDir, "export.csv")

	// Create command
	cmd := newExportCommand()
	cmd.SetArgs([]string{"test-plan.md", "--format", "csv", "--output", outputPath, "--db-path", dbPath})

	// Execute command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Read and verify output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify CSV structure
	if len(records) != 3 { // Header + 2 data rows
		t.Fatalf("Expected 3 rows (header + 2 data), got %d", len(records))
	}

	// Verify header
	expectedHeaders := []string{"id", "plan_file", "run_number", "task_number", "task_name", "agent", "success", "error_message", "duration_seconds", "timestamp"}
	for i, expected := range expectedHeaders {
		if records[0][i] != expected {
			t.Errorf("Expected header '%s' at position %d, got '%s'", expected, i, records[0][i])
		}
	}

	// Verify data rows (ordered by ID DESC, so exec2 comes first)
	if records[1][1] != "test-plan.md" {
		t.Errorf("Expected plan_file 'test-plan.md', got '%s'", records[1][1])
	}
	if records[1][3] != "2" {
		t.Errorf("Expected task_number '2', got '%s'", records[1][3])
	}
	if records[1][6] != "false" {
		t.Errorf("Expected success 'false', got '%s'", records[1][6])
	}
	if records[1][7] != "test error" {
		t.Errorf("Expected error_message 'test error', got '%s'", records[1][7])
	}

	// Verify second data row
	if records[2][3] != "1" {
		t.Errorf("Expected task_number '1', got '%s'", records[2][3])
	}
	if records[2][6] != "true" {
		t.Errorf("Expected success 'true', got '%s'", records[2][6])
	}
}

func TestExportCommand_InvalidFormat(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "learning.db")

	// Initialize store (even though we won't use it)
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create command with invalid format
	cmd := newExportCommand()
	cmd.SetArgs([]string{"test-plan.md", "--format", "xml", "--db-path", dbPath})

	// Execute command - should fail
	err = cmd.Execute()
	if err == nil {
		t.Fatal("Expected error for invalid format, got nil")
	}

	// Verify error message mentions invalid format
	if !strings.Contains(err.Error(), "invalid format") && !strings.Contains(err.Error(), "format must be") {
		t.Errorf("Expected error about invalid format, got: %v", err)
	}
}

func TestExportCommand_EmptyDatabase(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "learning.db")

	// Initialize empty store
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create temp output file
	outputPath := filepath.Join(tmpDir, "export.json")

	// Create command
	cmd := newExportCommand()
	cmd.SetArgs([]string{"test-plan.md", "--format", "json", "--output", outputPath, "--db-path", dbPath})

	// Execute command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Read and verify output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Parse JSON
	var executions []*learning.TaskExecution
	if err := json.Unmarshal(data, &executions); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify empty array
	if len(executions) != 0 {
		t.Errorf("Expected 0 executions, got %d", len(executions))
	}
}

func TestExportCommand_StdoutOutput(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "learning.db")

	// Initialize store and add test data
	store, err := learning.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add test execution
	exec1 := &learning.TaskExecution{
		PlanFile:     "test-plan.md",
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task 1",
		Agent:        "test-agent",
		Prompt:       "Test prompt 1",
		Success:      true,
		Output:       "Test output 1",
		ErrorMessage: "",
		DurationSecs: 30,
	}

	if err := store.RecordExecution(context.Background(), exec1); err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Create command without output flag (should use stdout)
	cmd := newExportCommand()
	cmd.SetArgs([]string{"test-plan.md", "--format", "json", "--db-path", dbPath})

	// Execute command (we can't easily capture stdout in this test,
	// but we can verify it doesn't error)
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
}
