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

// serializeAgentToJSON serializes an agent to JSON format for --agents flag (Method 1)
// Returns JSON string in the format: {"agent-name": {"name": "...", "description": "...", "tools": [...]}}
//
// Example output:
//
//	{"golang-pro": {"name": "golang-pro", "description": "Go expert", "tools": ["Read", "Write", "Edit"]}}
//
// This allows claude CLI to use the agent definition without requiring discovery.
func serializeAgentToJSON(agent *Agent) (string, error) {
	// Create a map with the agent name as key and agent struct as value
	agentMap := map[string]*Agent{
		agent.Name: agent,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(agentMap)
	if err != nil {
		return "", fmt.Errorf("failed to serialize agent to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// PrepareAgentPrompt adds formatting instructions to agent prompts for consistent output
func PrepareAgentPrompt(prompt string) string {
	const instructionPrefix = "Do not use markdown formatting or emojis in your response. "
	return instructionPrefix + prompt
}

// BuildCommandArgs constructs the command-line arguments for invoking claude CLI
// Uses Method 1: --agents JSON flag to pass agent definition explicitly for better automation reliability
//
// Method 1 (current): Pass agent via --agents JSON flag
//   - More reliable for automation (explicit definition)
//   - Example: claude --agents '{"golang-pro":{...}}' -p "use the golang-pro subagent to: ..."
//
// Behavior:
//   - If task.Agent is specified AND exists in registry: adds --agents flag with JSON definition
//   - If task.Agent is specified but not found: falls back to plain prompt (no agent)
//   - If task.Agent is empty: plain prompt (no agent flags)
//
// Arguments order:
//  1. --agents (if agent specified and found)
//  2. -p (prompt with "use the X subagent to:" prefix if agent present)
//  3. --dangerously-skip-permissions
//  4. --settings (disableAllHooks)
//  5. --output-format json
func (inv *Invoker) BuildCommandArgs(task models.Task) []string {
	args := []string{}

	// If agent is specified and exists in registry, add --agents flag with JSON definition
	if task.Agent != "" && inv.Registry != nil {
		if agent, exists := inv.Registry.Get(task.Agent); exists {
			// Serialize agent to JSON
			agentJSON, err := serializeAgentToJSON(agent)
			if err == nil {
				// Add --agents flag BEFORE -p flag (must come first)
				args = append(args, "--agents", agentJSON)
			}
		}
	}

	// Build prompt with agent reference if specified
	prompt := task.Prompt
	if task.Agent != "" && inv.Registry != nil && inv.Registry.Exists(task.Agent) {
		// Reference agent in prompt (still needed with Method 1)
		prompt = fmt.Sprintf("use the %s subagent to: %s", task.Agent, task.Prompt)
	}

	// Add formatting instructions to the prompt
	prompt = PrepareAgentPrompt(prompt)

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
// Handles both "content" and "result" fields for flexible agent response formats
// If the output is not valid JSON, it returns the raw output as content
func ParseClaudeOutput(output string) (*ClaudeOutput, error) {
	// First try to parse as JSON
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonMap); err != nil {
		// If not JSON, return raw output as content
		return &ClaudeOutput{Content: output}, nil
	}

	// Build result with both content and error fields
	result := &ClaudeOutput{}

	// Check for "result" field (used by some agents like conductor-qc)
	// This takes precedence as it's a custom field used by our agents
	if resultField, ok := jsonMap["result"]; ok {
		if resultStr, ok := resultField.(string); ok {
			result.Content = resultStr
			return result, nil
		}
	}

	// Check for "content" field (standard claude CLI JSON format)
	if content, ok := jsonMap["content"]; ok {
		if contentStr, ok := content.(string); ok {
			result.Content = contentStr
		}
	}

	// Check for "error" field (only if content is empty)
	if result.Content == "" {
		if errField, ok := jsonMap["error"]; ok {
			if errStr, ok := errField.(string); ok {
				result.Error = errStr
			}
		}
	}

	// If we have content or error, return it
	if result.Content != "" || result.Error != "" {
		return result, nil
	}

	// If no recognized fields, return the JSON as-is as content
	return &ClaudeOutput{Content: output}, nil
}
