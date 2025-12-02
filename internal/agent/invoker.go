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
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
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
	SessionID     string
}

// ClaudeOutput represents the JSON output structure from claude CLI
type ClaudeOutput struct {
	Content   string `json:"content"`
	Error     string `json:"error"`
	SessionID string `json:"session_id,omitempty"`
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
// Includes minimal JSON instruction - structure is enforced via --json-schema flag
func PrepareAgentPrompt(prompt string) string {
	const instructionSuffix = `

After completing all work, respond with JSON containing: status ("success" or "failed"), summary, output, errors (array), files_modified (array).`
	return prompt + instructionSuffix
}

// parseAgentJSON parses JSON response from agent
// With --json-schema enforcement, responses are guaranteed to be valid JSON
// Returns error if response is invalid
func parseAgentJSON(output string) (*models.AgentResponse, error) {
	var resp models.AgentResponse

	// Parse as JSON (schema enforcement should guarantee valid JSON)
	err := json.Unmarshal([]byte(output), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent response as JSON: %w", err)
	}

	// Validate parsed JSON structure
	if err := resp.Validate(); err != nil {
		return nil, fmt.Errorf("agent response failed validation: %w", err)
	}

	return &resp, nil
}

// BuildCommandArgs constructs the command-line arguments for invoking claude CLI
// Enforces JSON-structured responses using --json-schema flag
//
// Argument order:
//  1. --agents (if agent specified and found in registry)
//  2. --json-schema (enforces response structure via JSON schema - can be custom or default AgentResponseSchema)
//  3. -p (prompt without JSON instructions)
//  4. --permission-mode bypassPermissions
//  5. --settings (disableAllHooks)
//  6. --output-format json
//
// Behavior:
//   - If task.Agent is specified AND exists in registry: adds --agents flag with JSON definition
//   - If task.Agent is specified but not found: falls back to plain prompt (no agent)
//   - If task.Agent is empty: plain prompt (no agent flags)
//   - If task.JSONSchema is set: uses custom schema; otherwise uses AgentResponseSchema
//   - --json-schema enforces the response structure, eliminating need for prompt-based JSON instructions
func (inv *Invoker) BuildCommandArgs(task models.Task) []string {
	args := []string{}

	// If agent is specified and exists in registry, add --agents flag with JSON definition
	if task.Agent != "" && inv.Registry != nil {
		if agent, exists := inv.Registry.Get(task.Agent); exists {
			// Serialize agent to JSON
			agentJSON, err := serializeAgentToJSON(agent)
			if err == nil {
				// Add --agents flag BEFORE other flags (must come first)
				args = append(args, "--agents", agentJSON)
			}
		}
	}

	// Add JSON schema for structured responses
	// If task specifies custom JSONSchema, use it; otherwise use default AgentResponseSchema
	schemaJSON := task.JSONSchema
	if schemaJSON == "" {
		schemaJSON = models.AgentResponseSchema()
	}
	args = append(args, "--json-schema", schemaJSON)

	// Build prompt with formatting instructions
	prompt := task.Prompt
	prompt = PrepareAgentPrompt(prompt)

	// Add -p flag for non-interactive print mode (essential for automation)
	args = append(args, "-p", prompt)

	// Skip permissions for automation (allow file creation)
	args = append(args, "--permission-mode", "bypassPermissions")

	// Disable hooks for automation
	args = append(args, "--settings", `{"disableAllHooks": true}`)

	// JSON output for easier parsing (wrapper format, not content format)
	args = append(args, "--output-format", "json")

	return args
}

// getBoxWidth returns the terminal width for box drawing, with sensible bounds.
func getBoxWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		return 80 // fallback
	}
	if width > 120 {
		return 120 // cap for readability
	}
	return width
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

	// Get dynamic box width
	boxWidth := getBoxWidth()
	innerWidth := boxWidth - 4 // Account for "│ " and " │"

	// Box drawing - all borders in cyan
	hLine := strings.Repeat("─", boxWidth-2)
	fmt.Fprintf(os.Stderr, "\n%s┌%s┐%s\n", cyan, hLine, reset)

	// Header line - use runewidth for proper emoji width calculation
	headerText := "🤖 Agent Invocation"
	headerVisibleLen := runewidth.StringWidth(headerText)
	headerPad := innerWidth - headerVisibleLen
	if headerPad < 0 {
		headerPad = 0
	}
	fmt.Fprintf(os.Stderr, "%s│%s %s%s%s%s %s│%s\n",
		cyan, reset, bold, headerText, reset, strings.Repeat(" ", headerPad), cyan, reset)
	fmt.Fprintf(os.Stderr, "%s├%s┤%s\n", cyan, hLine, reset)

	// Helper to print a labeled line with proper alignment
	printLine := func(label, value, valueColor string) {
		// Use runewidth for proper width calculation (handles emojis, CJK chars)
		labelWidth := runewidth.StringWidth(label)
		valueWidth := runewidth.StringWidth(value)

		// Truncate value if too long
		maxValueWidth := innerWidth - labelWidth - 2 // -2 for ": "
		if valueWidth > maxValueWidth && maxValueWidth > 3 {
			// Truncate with runewidth awareness
			value = runewidth.Truncate(value, maxValueWidth-3, "...")
			valueWidth = runewidth.StringWidth(value)
		}

		// Calculate padding: innerWidth - label - ": " - value
		padding := innerWidth - labelWidth - 2 - valueWidth
		if padding < 0 {
			padding = 0
		}

		// Print: cyan│ yellow_label: value_color_value padding cyan│
		fmt.Fprintf(os.Stderr, "%s│%s %s%s:%s %s%s%s%s %s│%s\n",
			cyan, reset,
			yellow, label, reset,
			valueColor, value, reset,
			strings.Repeat(" ", padding),
			cyan, reset)
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

	// Schema enforcement indicator
	printLine("Schema", "Enforced", green)

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

	fmt.Fprintf(os.Stderr, "%s└%s┘%s\n\n", cyan, hLine, reset)
}

// Invoke executes the claude CLI command with the given context
func (inv *Invoker) Invoke(ctx context.Context, task models.Task) (*InvocationResult, error) {
	startTime := time.Now()

	// Build command args
	args := inv.BuildCommandArgs(task)

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
		// Extract session_id from Claude CLI output
		result.SessionID = parsedOutput.SessionID

		// Log session_id if present
		if result.SessionID != "" {
			fmt.Fprintf(os.Stderr, "Session ID: %s\n", result.SessionID)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: No session_id in Claude CLI output\n")
		}

		// Parse the content as AgentResponse JSON
		agentResp, parseErr := parseAgentJSON(parsedOutput.Content)
		if parseErr != nil {
			result.Error = fmt.Errorf("failed to parse agent response: %w", parseErr)
		} else if agentResp != nil {
			// Set session_id in AgentResponse as well
			agentResp.SessionID = result.SessionID
			result.AgentResponse = agentResp
		}
	} else {
		// If we couldn't extract content from Claude output, try parsing raw as fallback
		agentResp, parseErr := parseAgentJSON(string(output))
		if parseErr != nil {
			result.Error = fmt.Errorf("failed to parse agent response: %w", parseErr)
		} else {
			result.AgentResponse = agentResp
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
// Handles "structured_output" (from --json-schema), "content", and "result" fields
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

	// Check for "session_id" field (available in Claude CLI output)
	if sessionIDField, ok := jsonMap["session_id"]; ok {
		if sessionIDStr, ok := sessionIDField.(string); ok {
			result.SessionID = sessionIDStr
		}
	}

	// Check for "structured_output" field (used when --json-schema is specified)
	// This takes highest precedence as it's the schema-validated output
	if structuredOutput, ok := jsonMap["structured_output"]; ok {
		// Re-serialize the structured output as JSON string for downstream parsing
		if outputBytes, err := json.Marshal(structuredOutput); err == nil {
			result.Content = string(outputBytes)
			return result, nil
		}
	}

	// Check for "result" field (used by some agents like conductor-qc)
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
