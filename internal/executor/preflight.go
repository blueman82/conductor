package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// ErrDependencyCheckFailed indicates a dependency check command failed.
var ErrDependencyCheckFailed = errors.New("dependency check failed")

// CommandRunner abstracts shell command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, command string) (output string, err error)
}

// ShellCommandRunner executes commands via the system shell.
type ShellCommandRunner struct {
	WorkDir string // Working directory for commands (empty = current dir)
}

// NewShellCommandRunner creates a CommandRunner that executes real shell commands.
func NewShellCommandRunner(workDir string) *ShellCommandRunner {
	return &ShellCommandRunner{WorkDir: workDir}
}

// Run executes a command via sh -c and returns combined stdout/stderr.
func (r *ShellCommandRunner) Run(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DependencyCheckResult holds the result of a single dependency check.
type DependencyCheckResult struct {
	Command     string
	Description string
	Output      string
	Error       error
	Duration    time.Duration
}

// RunDependencyChecks executes all dependency checks for a task.
// Returns nil if all checks pass or if there are no checks to run.
// Returns ErrDependencyCheckFailed (wrapped) if any check fails.
// Stops on first failure.
func RunDependencyChecks(ctx context.Context, runner CommandRunner, task models.Task) error {
	// No metadata or no checks - nothing to do
	if task.RuntimeMetadata == nil {
		return nil
	}

	checks := task.RuntimeMetadata.DependencyChecks
	if len(checks) == 0 {
		return nil
	}

	for _, check := range checks {
		// Check context before running
		if ctx.Err() != nil {
			return ctx.Err()
		}

		start := time.Now()
		output, err := runner.Run(ctx, check.Command)
		duration := time.Since(start)

		if err != nil {
			// Build detailed error message
			errMsg := fmt.Sprintf(
				"dependency check failed for task %s: command %q (%s) failed after %v: %v",
				task.Number,
				check.Command,
				check.Description,
				duration.Round(time.Millisecond),
				err,
			)
			if output != "" {
				errMsg += fmt.Sprintf("\nOutput:\n%s", strings.TrimSpace(output))
			}

			return fmt.Errorf("%w: %s", ErrDependencyCheckFailed, errMsg)
		}
	}

	return nil
}

// RunDependencyChecksWithResults executes all checks and returns detailed results.
// Unlike RunDependencyChecks, this continues on failure and returns all results.
func RunDependencyChecksWithResults(ctx context.Context, runner CommandRunner, task models.Task) []DependencyCheckResult {
	if task.RuntimeMetadata == nil {
		return nil
	}

	checks := task.RuntimeMetadata.DependencyChecks
	if len(checks) == 0 {
		return nil
	}

	results := make([]DependencyCheckResult, 0, len(checks))

	for _, check := range checks {
		if ctx.Err() != nil {
			results = append(results, DependencyCheckResult{
				Command:     check.Command,
				Description: check.Description,
				Error:       ctx.Err(),
			})
			break
		}

		start := time.Now()
		output, err := runner.Run(ctx, check.Command)
		duration := time.Since(start)

		results = append(results, DependencyCheckResult{
			Command:     check.Command,
			Description: check.Description,
			Output:      output,
			Error:       err,
			Duration:    duration,
		})
	}

	return results
}
