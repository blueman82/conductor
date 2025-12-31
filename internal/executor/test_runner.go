package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/models"
)

// ErrTestCommandFailed indicates a test command exited with non-zero status.
var ErrTestCommandFailed = errors.New("test command failed")

// RunTestCommands executes all test commands for a task sequentially.
// Returns error immediately on first failure (task fails before QC review).
// Returns nil if all commands pass or if there are no commands to run.
func RunTestCommands(ctx context.Context, runner CommandRunner, task models.Task) ([]TestCommandResult, error) {
	// No test commands - nothing to do
	if len(task.TestCommands) == 0 {
		return nil, nil
	}

	results := make([]TestCommandResult, 0, len(task.TestCommands))

	for _, cmd := range task.TestCommands {
		// Check context before running
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		start := time.Now()
		output, err := runner.Run(ctx, cmd)
		duration := time.Since(start)

		result := TestCommandResult{
			Command:  cmd,
			Output:   output,
			Error:    err,
			Passed:   err == nil,
			Duration: duration,
		}
		results = append(results, result)

		if err != nil {
			// Build detailed error message
			errMsg := fmt.Sprintf(
				"test command failed for task %s: %q failed after %v: %v",
				task.Number,
				cmd,
				duration.Round(time.Millisecond),
				err,
			)
			if output != "" {
				errMsg += fmt.Sprintf("\nOutput:\n%s", strings.TrimSpace(output))
			}

			return results, fmt.Errorf("%w: %s", ErrTestCommandFailed, errMsg)
		}
	}

	return results, nil
}

// FormatTestResults formats test command results for injection into QC prompt.
// Returns empty string if no results.
func FormatTestResults(results []TestCommandResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<test_command_results>\n")

	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
		}

		status := "passed"
		if !r.Passed {
			status = "failed"
		}

		sb.WriteString(fmt.Sprintf("<test_result status=\"%s\" command=\"%s\" duration=\"%v\">\n",
			status, r.Command, r.Duration.Round(time.Millisecond)))

		if r.Output != "" {
			sb.WriteString(agent.XMLSection("output", strings.TrimSpace(r.Output)))
			sb.WriteString("\n")
		}

		if r.Error != nil {
			sb.WriteString(agent.XMLTag("error", fmt.Sprintf("%v", r.Error)))
			sb.WriteString("\n")
		}
		sb.WriteString("</test_result>\n")
	}

	if allPassed {
		sb.WriteString("<summary>All test commands passed</summary>\n")
	} else {
		sb.WriteString("<summary>One or more test commands failed</summary>\n")
	}

	sb.WriteString("</test_command_results>\n")
	return sb.String()
}
