package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// FakeCommandRunner implements CommandRunner for testing
type FakeCommandRunner struct {
	outputs  map[string]string
	errors   map[string]error
	commands []string
}

// NewFakeCommandRunner creates a new FakeCommandRunner
func NewFakeCommandRunner() *FakeCommandRunner {
	return &FakeCommandRunner{
		outputs:  make(map[string]string),
		errors:   make(map[string]error),
		commands: []string{},
	}
}

// SetOutput sets the output for a given command
func (f *FakeCommandRunner) SetOutput(cmd, output string) {
	f.outputs[cmd] = output
}

// SetError sets the error for a given command
func (f *FakeCommandRunner) SetError(cmd string, err error) {
	f.errors[cmd] = err
}

// Run executes the command and returns output/error based on configuration
func (f *FakeCommandRunner) Run(ctx context.Context, command string) (string, error) {
	f.commands = append(f.commands, command)

	// Check for context cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if err, ok := f.errors[command]; ok {
		return f.outputs[command], err
	}

	return f.outputs[command], nil
}

// Commands returns all executed commands
func (f *FakeCommandRunner) Commands() []string {
	return f.commands
}

// === Tests ===

func TestRunDependencyChecks_Success(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("go version", "go version go1.21.0 darwin/arm64")
	runner.SetOutput("echo hello", "hello")

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{Command: "go version", Description: "Check Go version"},
				{Command: "echo hello", Description: "Echo test"},
			},
		},
	}

	err := RunDependencyChecks(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify all commands were executed
	cmds := runner.Commands()
	if len(cmds) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cmds))
	}
}

func TestRunDependencyChecks_Failure(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetError("failing command", errors.New("exit status 1: command not found"))
	runner.SetOutput("failing command", "some stderr output")

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{Command: "failing command", Description: "This should fail"},
			},
		},
	}

	err := RunDependencyChecks(context.Background(), runner, task)
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify error message contains command
	if !errors.Is(err, ErrDependencyCheckFailed) {
		t.Errorf("expected ErrDependencyCheckFailed, got %v", err)
	}
}

func TestRunDependencyChecks_ContextCancellation(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{Command: "long running command", Description: "Should be cancelled"},
			},
		},
	}

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunDependencyChecks(ctx, runner, task)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestRunDependencyChecks_Timeout(t *testing.T) {
	runner := NewFakeCommandRunner()
	// Simulate a command that will timeout
	runner.SetError("timeout command", context.DeadlineExceeded)

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{Command: "timeout command", Description: "Should timeout"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := RunDependencyChecks(ctx, runner, task)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestRunDependencyChecks_EmptyDependencyList(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{},
		},
	}

	err := RunDependencyChecks(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error for empty dependency list, got %v", err)
	}

	// Verify no commands were executed
	if len(runner.Commands()) != 0 {
		t.Errorf("expected 0 commands, got %d", len(runner.Commands()))
	}
}

func TestRunDependencyChecks_NilRuntimeMetadata(t *testing.T) {
	runner := NewFakeCommandRunner()

	task := models.Task{
		Number:          "1",
		Name:            "Test Task",
		RuntimeMetadata: nil,
	}

	err := RunDependencyChecks(context.Background(), runner, task)
	if err != nil {
		t.Errorf("expected no error for nil RuntimeMetadata, got %v", err)
	}
}

func TestRunDependencyChecks_StopsOnFirstFailure(t *testing.T) {
	runner := NewFakeCommandRunner()
	runner.SetOutput("cmd1", "success")
	runner.SetError("cmd2", errors.New("exit status 1"))
	runner.SetOutput("cmd3", "should not run")

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DependencyChecks: []models.DependencyCheck{
				{Command: "cmd1", Description: "First command"},
				{Command: "cmd2", Description: "Failing command"},
				{Command: "cmd3", Description: "Third command"},
			},
		},
	}

	err := RunDependencyChecks(context.Background(), runner, task)
	if err == nil {
		t.Error("expected error from failing command")
	}

	// Verify only first two commands were executed
	cmds := runner.Commands()
	if len(cmds) != 2 {
		t.Errorf("expected 2 commands (stopped on failure), got %d: %v", len(cmds), cmds)
	}
}

func TestDependencyCheckResult_Fields(t *testing.T) {
	result := &DependencyCheckResult{
		Command:     "test command",
		Description: "Test description",
		Output:      "test output",
		Error:       errors.New("test error"),
		Duration:    100 * time.Millisecond,
	}

	if result.Command != "test command" {
		t.Errorf("expected command 'test command', got %q", result.Command)
	}
	if result.Description != "Test description" {
		t.Errorf("expected description 'Test description', got %q", result.Description)
	}
	if result.Output != "test output" {
		t.Errorf("expected output 'test output', got %q", result.Output)
	}
	if result.Error == nil || result.Error.Error() != "test error" {
		t.Errorf("expected error 'test error', got %v", result.Error)
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %v", result.Duration)
	}
}
