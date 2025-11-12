package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// Invoker manages execution of claude CLI commands
type Invoker struct {
	ClaudePath string
	Registry   *Registry
}

// InvocationResult captures the result of invoking the claude CLI
type InvocationResult struct {
	Output   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// ClaudeOutput represents the JSON output structure from claude CLI
type ClaudeOutput struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

// NewInvoker creates a new Invoker with default settings
func NewInvoker() *Invoker {
	return &Invoker{
		ClaudePath: "claude",
	}
}

// NewInvokerWithRegistry creates a new Invoker with a specified agent registry
func NewInvokerWithRegistry(registry *Registry) *Invoker {
	return &Invoker{
		ClaudePath: "claude",
		Registry:   registry,
	}
}

// BuildCommandArgs constructs the command-line arguments for invoking claude CLI
func (inv *Invoker) BuildCommandArgs(task models.Task) []string {
	args := []string{}

	// Build prompt with agent reference if specified
	prompt := task.Prompt
	if task.Agent != "" && inv.Registry != nil && inv.Registry.Exists(task.Agent) {
		// Reference agent in prompt
		prompt = fmt.Sprintf("use the %s subagent to: %s", task.Agent, task.Prompt)
	}

	// Add -p flag for non-interactive print mode (essential for automation)
	args = append(args, "-p", prompt)

	// Skip permissions for automation (allow file creation)
	args = append(args, "--dangerously-skip-permissions")

	// Disable hooks for automation
	args = append(args, "--settings", `{"disableAllHooks": true}`)

	// JSON output for easier parsing
	args = append(args, "--output-format", "json")

	return args
}

// Invoke executes the claude CLI command with the given context
func (inv *Invoker) Invoke(ctx context.Context, task models.Task) (*InvocationResult, error) {
	startTime := time.Now()

	// Build command args
	args := inv.BuildCommandArgs(task)

	// Create command with context (for timeout)
	cmd := exec.CommandContext(ctx, inv.ClaudePath, args...)

	// Capture output
	output, err := cmd.CombinedOutput()

	result := &InvocationResult{
		Output:   string(output),
		Duration: time.Since(startTime),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err
		}
	}

	return result, nil
}

// InvokeWithTimeout executes the claude CLI command with a specified timeout
func (inv *Invoker) InvokeWithTimeout(task models.Task, timeout time.Duration) (*InvocationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return inv.Invoke(ctx, task)
}

// ParseClaudeOutput parses the JSON output from claude CLI
// If the output is not valid JSON, it returns the raw output as content
func ParseClaudeOutput(output string) (*ClaudeOutput, error) {
	var co ClaudeOutput
	if err := json.Unmarshal([]byte(output), &co); err != nil {
		// If not JSON, return raw output as content
		return &ClaudeOutput{Content: output}, nil
	}
	return &co, nil
}
