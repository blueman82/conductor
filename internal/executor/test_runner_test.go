package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// === Test Runner Tests ===

func TestRunTestCommands_AllPass(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("go test ./...", "ok")
	runner.SetOutput("go vet ./...", "ok")

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: []string{"go test ./...", "go vet ./..."},
	}

	results, err := RunTestCommands(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Verify all commands were executed in order
	cmds := runner.Commands()
	if len(cmds) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0] != "go test ./..." {
		t.Errorf("expected first command 'go test ./...', got %q", cmds[0])
	}
	if cmds[1] != "go vet ./..." {
		t.Errorf("expected second command 'go vet ./...', got %q", cmds[1])
	}

	// Verify results
	for i, r := range results {
		if r.Error != nil {
			t.Errorf("result[%d] expected no error, got %v", i, r.Error)
		}
		if !r.Passed {
			t.Errorf("result[%d] expected passed=true", i)
		}
	}
}

func TestRunTestCommands_FailureStopsExecution(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("go test ./...", "FAIL")
	runner.SetError("go test ./...", errors.New("exit status 1"))
	runner.SetOutput("go vet ./...", "should not run")

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: []string{"go test ./...", "go vet ./..."},
	}

	results, err := RunTestCommands(context.Background(), runner, task)
	if err == nil {
		t.Error("expected error, got nil")
	}

	if !errors.Is(err, ErrTestCommandFailed) {
		t.Errorf("expected ErrTestCommandFailed, got %v", err)
	}

	// Verify only first command was executed (stopped on failure)
	cmds := runner.Commands()
	if len(cmds) != 1 {
		t.Errorf("expected 1 command (stopped on failure), got %d: %v", len(cmds), cmds)
	}

	// Verify results contain failure
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected first result to have passed=false")
	}
}

func TestRunTestCommands_SecondCommandFails(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("go test ./...", "ok")
	runner.SetOutput("go lint ./...", "lint errors found")
	runner.SetError("go lint ./...", errors.New("lint fail"))

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: []string{"go test ./...", "go lint ./..."},
	}

	results, err := RunTestCommands(context.Background(), runner, task)
	if err == nil {
		t.Error("expected error from lint failure")
	}

	if !errors.Is(err, ErrTestCommandFailed) {
		t.Errorf("expected ErrTestCommandFailed, got %v", err)
	}

	// Both commands should have been executed
	cmds := runner.Commands()
	if len(cmds) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cmds))
	}

	// First result should pass, second should fail
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if !results[0].Passed {
		t.Error("expected first result to pass")
	}
	if results[1].Passed {
		t.Error("expected second result to fail")
	}
}

func TestRunTestCommands_EmptyList(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: []string{},
	}

	results, err := RunTestCommands(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error for empty list, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	if len(runner.Commands()) != 0 {
		t.Errorf("expected 0 commands, got %d", len(runner.Commands()))
	}
}

func TestRunTestCommands_NilList(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: nil,
	}

	results, err := RunTestCommands(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error for nil list, got %v", err)
	}

	if results != nil && len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRunTestCommands_ContextCancellation(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("go test ./...", "ok")

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: []string{"go test ./..."},
	}

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunTestCommands(ctx, runner, task)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestRunTestCommands_Timeout(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetError("slow command", context.DeadlineExceeded)

	task := models.Task{
		Number:       "1",
		Name:         "Test Task",
		TestCommands: []string{"slow command"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := RunTestCommands(ctx, runner, task)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestTestCommandResult_Fields(t *testing.T) {
	result := &TestCommandResult{
		Command:  "go test ./...",
		Output:   "PASS",
		Error:    nil,
		Passed:   true,
		Duration: 100 * time.Millisecond,
	}

	if result.Command != "go test ./..." {
		t.Errorf("expected command 'go test ./...', got %q", result.Command)
	}
	if result.Output != "PASS" {
		t.Errorf("expected output 'PASS', got %q", result.Output)
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %v", result.Duration)
	}
}

func TestFormatTestResults_AllPass(t *testing.T) {
	results := []TestCommandResult{
		{Command: "go test ./...", Passed: true, Output: "PASS"},
		{Command: "go vet ./...", Passed: true, Output: "ok"},
	}

	formatted := FormatTestResults(results)

	if formatted == "" {
		t.Error("expected non-empty formatted string")
	}

	// Should contain success markers
	if !containsString(formatted, "✅") && !containsString(formatted, "PASS") {
		t.Error("expected success indicators in output")
	}
}

func TestFormatTestResults_WithFailure(t *testing.T) {
	results := []TestCommandResult{
		{Command: "go test ./...", Passed: true, Output: "PASS"},
		{Command: "go lint", Passed: false, Output: "errors found", Error: errors.New("lint failed")},
	}

	formatted := FormatTestResults(results)

	if formatted == "" {
		t.Error("expected non-empty formatted string")
	}

	// Should contain failure markers
	if !containsString(formatted, "❌") && !containsString(formatted, "FAIL") {
		t.Error("expected failure indicators in output")
	}
}

func TestFormatTestResults_Empty(t *testing.T) {
	results := []TestCommandResult{}
	formatted := FormatTestResults(results)
	if formatted != "" {
		t.Errorf("expected empty string for empty results, got %q", formatted)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
