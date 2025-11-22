package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	Output        string
	ExitCode      int
	Duration      time.Duration
	Error         error
	AgentResponse *models.AgentResponse
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
// Returns JSON string in the format: {"agent-name": {"description": "...", "prompt": "...", "model": "...", "tools": [...]}}
//
// Example output:
//
//	{"golang-pro": {"description": "Go expert", "prompt": "You are...", "model": "haiku", "tools": ["Read", "Write", "Edit"]}}
//
// This allows claude CLI to use the agent definition without requiring discovery.
// Note: "name" field is omitted since it's redundant with the map key.
func serializeAgentToJSON(agent *Agent) (string, error) {
	// Create agent config without redundant name field
	agentConfig := map[string]interface{}{
		"description": agent.Description,
		"prompt":      agent.Prompt,
		"tools":       agent.Tools,
	}

	// Add model if specified
	if agent.Model != "" {
		agentConfig["model"] = agent.Model
	}

	// Create map with agent name as key
	agentMap := map[string]interface{}{
		agent.Name: agentConfig,
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

// buildJSONPrompt appends JSON instruction to agent prompts
func (inv *Invoker) buildJSONPrompt(originalPrompt string) string {
	jsonInstruction := `

IMPORTANT: Respond ONLY with valid JSON in this format:
{
  "status": "success|failed",
  "summary": "Brief description",
  "output": "Full execution output",
  "errors": ["error1"],
  "files_modified": ["file1.go"],
  "metadata": {}
}`
	return originalPrompt + jsonInstruction
}

// parseAgentJSON parses JSON response from agent, with fallback to plain text
func parseAgentJSON(output string) (*models.AgentResponse, error) {
	var resp models.AgentResponse

	// Try parsing as JSON
	err := json.Unmarshal([]byte(output), &resp)
	if err != nil {
		// Fallback: wrap plain text
		resp = models.AgentResponse{
			Status:   "success",
			Summary:  "Plain text response",
			Output:   output,
			Errors:   []string{},
			Files:    []string{},
			Metadata: map[string]interface{}{"parse_fallback": true},
		}
		return &resp, nil
	}

	// Validate parsed JSON
	if err := resp.Validate(); err != nil {
		// Invalid JSON structure, fallback to plain text
		resp = models.AgentResponse{
			Status:   "success",
			Summary:  "Plain text response",
			Output:   output,
			Errors:   []string{},
			Files:    []string{},
			Metadata: map[string]interface{}{"parse_fallback": true},
		}
		return &resp, nil
	}

	return &resp, nil
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
//  2. -p (prompt with "use the X subagent to:" prefix if agent present + JSON instruction)
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

	// Build prompt - NO agent prefix needed when using --agents with prompt field
	// The agent definition already includes its system prompt via the "prompt" field
	prompt := task.Prompt

	// Add formatting instructions to the prompt
	prompt = PrepareAgentPrompt(prompt)

	// Add JSON instruction to prompt
	prompt = inv.buildJSONPrompt(prompt)

	// Add -p flag for non-interactive print mode (essential for automation)
	args = append(args, "-p", prompt)

	// Skip permissions for automation (allow file creation)
	args = append(args, "--permission-mode", "bypassPermissions")

	// Disable hooks for automation
	args = append(args, "--settings", `{"disableAllHooks": true}`)

	// JSON output for easier parsing
	args = append(args, "--output-format", "json")

	return args
}

// logInvocation prints a pretty summary of the agent invocation with colors
func (inv *Invoker) logInvocation(task models.Task, args []string) {
	// ANSI color codes
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	magenta := "\033[35m"
	blue := "\033[34m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Fprintf(os.Stderr, "\n%s┌────────────────────────────────────────────────────────────────┐%s\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "%s│ %s🤖 Agent Invocation%s                                            │%s\n", cyan, bold, reset, cyan+reset)
	fmt.Fprintf(os.Stderr, "%s├────────────────────────────────────────────────────────────────┤%s\n", cyan, reset)

	// Helper to pad and colorize a line
	// Box width: 64 chars total
	// Format: "│ Label:  value                                      │"
	// Available for "Label:  value" = 62 chars (64 - 2 for borders)
	printLine := func(label, value, valueColor string) {
		// Truncate value if too long
		labelLen := len(label) + 3 // "Label:  " = label + colon + 2 spaces
		maxValueLen := 62 - labelLen
		if len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}

		// Calculate padding needed (62 - labelLen - valueLen)
		padding := 62 - labelLen - len(value)
		spacer := strings.Repeat(" ", padding)

		fmt.Fprintf(os.Stderr, "%s│%s %s%s:%s  %s%s%s%s %s│%s\n",
			cyan, reset, yellow, label, reset, valueColor, value, reset, spacer, cyan, reset)
	}

	// Agent info
	if task.Agent != "" {
		printLine("Agent", task.Agent, green+bold)

		// Get agent details if available
		if inv.Registry != nil {
			if agent, exists := inv.Registry.Get(task.Agent); exists {
				// Show tools
				toolsStr := strings.Join(agent.Tools, ", ")
				printLine("Tools", toolsStr, blue)
			}
		}
	} else {
		printLine("Agent", "(base model - no tools)", magenta)
	}

	// Task info
	printLine("Task", task.Name, bold)

	// Files being modified
	if len(task.Files) > 0 {
		filesStr := fmt.Sprintf("%d files", len(task.Files))
		if len(task.Files) <= 3 {
			filesStr = strings.Join(task.Files, ", ")
		}
		printLine("Files", filesStr, green)
	}

	fmt.Fprintf(os.Stderr, "%s└────────────────────────────────────────────────────────────────┘%s\n\n", cyan, reset)
}

// Invoke executes the claude CLI command with the given context
func (inv *Invoker) Invoke(ctx context.Context, task models.Task) (*InvocationResult, error) {
	startTime := time.Now()

	// Build command args
	args := inv.BuildCommandArgs(task)

	// DEBUG: Log the actual --agents JSON being sent
	for i, arg := range args {
		if arg == "--agents" && i+1 < len(args) {
			fmt.Fprintf(os.Stderr, "\n[DEBUG] --agents JSON:\n%s\n\n", args[i+1][:500])
			break
		}
	}

	// Pretty log the agent invocation
	inv.logInvocation(task, args)

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

	// Parse agent response from output
	parsedOutput, parseErr := ParseClaudeOutput(string(output))
	if parseErr == nil && parsedOutput.Content != "" {
		// Parse the content as AgentResponse JSON
		agentResp, _ := parseAgentJSON(parsedOutput.Content)
		result.AgentResponse = agentResp
	} else {
		// Fallback: parse raw output as AgentResponse
		agentResp, _ := parseAgentJSON(string(output))
		result.AgentResponse = agentResp
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

	// If both content and error are empty strings, return empty result
	// This handles the case where JSON is valid but both fields are ""
	if result.Content == "" && result.Error == "" {
		return result, nil
	}

	// If we have content or error, return it
	if result.Content != "" || result.Error != "" {
		return result, nil
	}

	// If no recognized fields, return the JSON as-is as content
	return &ClaudeOutput{Content: output}, nil
}
