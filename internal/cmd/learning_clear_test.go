package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/learning"
)

func TestClearCommand_SinglePlan(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "conductor-clear-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use temp database path
	testDBPath := filepath.Join(tmpDir, "learning.db")

	// Create and populate test database
	store, err := learning.NewStore(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	planFile := "test-plan.md"
	exec := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task",
		Success:      true,
		Output:       "output",
		DurationSecs: 1,
	}
	err = store.RecordExecution(context.Background(), exec)
	if err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}
	store.Close()

	// Test clear command with confirmation "y"
	cmd := newClearCommand()
	cmd.SetArgs([]string{planFile, "--db-path", testDBPath})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Simulate user input "y" for confirmation
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Clear command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Deleted") || !strings.Contains(output, "record") {
		t.Errorf("Expected deletion confirmation in output, got: %s", output)
	}

	// Verify data was cleared
	store, _ = learning.NewStore(testDBPath)
	defer store.Close()

	// Query to count remaining records for this plan
	query := `SELECT COUNT(*) FROM task_executions WHERE plan_file = ?`
	rows, _ := store.QueryRows(query, planFile)
	defer rows.Close()

	var count int
	if rows.Next() {
		rows.Scan(&count)
	}

	if count != 0 {
		t.Errorf("Expected 0 records after clear, got %d", count)
	}
}

func TestClearCommand_AllData(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "conductor-clear-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use temp database path
	testDBPath := filepath.Join(tmpDir, "learning.db")

	// Create and populate test database with multiple plans
	store, err := learning.NewStore(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	for i := 1; i <= 3; i++ {
		planFile := "plan" + string(rune('0'+i)) + ".md"
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    1,
			TaskNumber:   "1",
			TaskName:     "Test Task",
			Success:      true,
			Output:       "output",
			DurationSecs: 1,
		}
		err = store.RecordExecution(context.Background(), exec)
		if err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}
	store.Close()

	// Test clear command with --all flag
	cmd := newClearCommand()
	cmd.SetArgs([]string{"--all", "--db-path", testDBPath})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Simulate user input "y" for confirmation
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Clear command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Deleted") || !strings.Contains(output, "record") {
		t.Errorf("Expected deletion confirmation in output, got: %s", output)
	}

	// Verify all data was cleared
	store, _ = learning.NewStore(testDBPath)
	defer store.Close()

	// Query to count all remaining records
	query := `SELECT COUNT(*) FROM task_executions`
	rows, _ := store.QueryRows(query)
	defer rows.Close()

	var count int
	if rows.Next() {
		rows.Scan(&count)
	}

	if count != 0 {
		t.Errorf("Expected 0 records after clear --all, got %d", count)
	}
}

func TestClearCommand_Confirmation(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "conductor-clear-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use temp database path
	testDBPath := filepath.Join(tmpDir, "learning.db")

	// Create and populate test database
	store, err := learning.NewStore(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	planFile := "test-plan.md"
	exec := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task",
		Success:      true,
		Output:       "output",
		DurationSecs: 1,
	}
	err = store.RecordExecution(context.Background(), exec)
	if err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}
	store.Close()

	// Test clear command with confirmation "n" (cancel)
	cmd := newClearCommand()
	cmd.SetArgs([]string{planFile, "--db-path", testDBPath})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Simulate user input "n" for cancellation
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	go func() {
		w.Write([]byte("n\n"))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Clear command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cancelled") && !strings.Contains(output, "cancelled") {
		t.Errorf("Expected cancellation message in output, got: %s", output)
	}

	// Verify data was NOT cleared
	store, _ = learning.NewStore(testDBPath)
	defer store.Close()

	// Query to count remaining records for this plan
	query := `SELECT COUNT(*) FROM task_executions WHERE plan_file = ?`
	rows, _ := store.QueryRows(query, planFile)
	defer rows.Close()

	var count int
	if rows.Next() {
		rows.Scan(&count)
	}

	if count == 0 {
		t.Errorf("Expected data to remain after cancellation, got 0 records")
	}
}

func TestClearCommand_NoData(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "conductor-clear-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use temp database path
	testDBPath := filepath.Join(tmpDir, "learning.db")

	// Create empty test database
	store, err := learning.NewStore(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	store.Close()

	// Test clear command on empty database
	cmd := newClearCommand()
	cmd.SetArgs([]string{"test-plan.md", "--db-path", testDBPath})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Simulate user input "y" for confirmation
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Clear command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0 record") {
		t.Errorf("Expected 0 records message in output, got: %s", output)
	}
}
