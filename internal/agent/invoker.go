package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
	"github.com/harrison/conductor/internal/models"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// ErrRateLimit is returned when Claude CLI output indicates a rate limit
type ErrRateLimit struct {
	RawMessage string
}

func (e *ErrRateLimit) Error() string {
	return fmt.Sprintf("rate limit: %s", e.RawMessage)
}

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
// Includes Claude 4 enhancements and XML-formatted response instructions for guaranteed valid JSON output
// Note: --json-schema is not enforced with --agents flag, so explicit format instruction is critical
func PrepareAgentPrompt(prompt string) string {
	// Add Claude 4-specific enhancements
	enhanced := EnhancePromptForClaude4(prompt)

	// XML-formatted response instructions
	responseFormat := XMLSection("response_format",
		`CRITICAL: Respond with ONLY valid JSON matching the provided schema.
No markdown, no code fences, no XML tags in output, no prose, no explanations.
Output raw JSON only.

Required JSON structure:
{"status":"success","summary":"...","output":"...","errors":[],"files_modified":[]}`)

	return enhanced + "\n\n" + responseFormat
}

// PrepareQCPrompt adds formatting instructions to QC review prompts
// Includes Claude 4 enhancements and XML-formatted response instructions for guaranteed valid QC response
// Note: --json-schema is not enforced with --agents flag, so explicit format instruction is critical
func PrepareQCPrompt(prompt string) string {
	// Add Claude 4-specific enhancements
	enhanced := EnhancePromptForClaude4(prompt)

	// Build XML-formatted instructions
	var sb strings.Builder

	sb.WriteString(XMLSection("response_instructions", `
<consistency_rule>
CRITICAL: Your feedback text MUST be consistent with criteria_results:
- If ANY criterion has "passed": false, feedback MUST mention which criterion failed and why
- NEVER say "successfully completed" or similar if any criterion failed
- If verdict is RED, feedback MUST describe what needs to be fixed
</consistency_rule>

<response_format>
Respond with ONLY valid JSON matching the QC schema. No prose, no markdown.
Required structure:
{"verdict":"GREEN|YELLOW|RED","feedback":"...","criteria_results":[{"index":0,"criterion":"...","passed":true,"evidence":"..."}],"should_retry":false}
</response_format>`))

	return enhanced + "\n\n" + sb.String()
}

// parseAgentJSON parses JSON response from agent
// Extracts JSON object from output (skips any prose before opening brace)
// Returns error if response is invalid
func parseAgentJSON(output string) (*models.AgentResponse, error) {
	var resp models.AgentResponse

	// Extract JSON portion - find opening and closing braces
	// This handles agents outputting prose before JSON, or wrapping in markdown code blocks
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")
	if jsonStart < 0 || jsonEnd < 0 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("failed to parse agent response as JSON: no complete JSON object found in output")
	}
	jsonStr := output[jsonStart : jsonEnd+1]

	// Parse as JSON
	err := json.Unmarshal([]byte(jsonStr), &resp)
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
// Enforces JSON-structured responses using explicit format instructions (since --json-schema doesn't work with --agents)
//
// Argument order:
//  1. --agents (if agent specified and found in registry)
//  2. --json-schema (enforces response structure via JSON schema - can be custom or default)
//  3. --append-system-prompt (for QC tasks, enforces JSON-only output at system level)
//  4. -p (prompt with explicit JSON format instructions appended)
//  5. --permission-mode bypassPermissions
//  6. --settings (disableAllHooks)
//  7. --output-format json
//
// Behavior:
//   - If task.Agent is specified AND exists in registry: adds --agents flag with JSON definition
//   - If task.Agent is specified but not found: falls back to plain prompt (no agent)
//   - If task.Agent is empty: plain prompt (no agent flags)
//   - If task.JSONSchema is set: uses custom schema; otherwise uses AgentResponseSchema
//   - Regular agent tasks: PrepareAgentPrompt adds explicit JSON format at end of prompt
//   - QC review tasks: PrepareQCPrompt already applied by BuildReviewPrompt/BuildStructuredReviewPrompt
//   - QC review tasks: --append-system-prompt enforces JSON-only at system level
func (inv *Invoker) BuildCommandArgs(task models.Task) []string {
	args := []string{}

	// Resume from previous session if available (rate limit recovery)
	if task.ResumeSessionID != "" {
		args = append(args, "--resume", task.ResumeSessionID)
	}

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

	// Override system prompt to enforce JSON-only output for all tasks
	// Using --system-prompt (not --append) to fully replace default prompt
	// This prevents agents from outputting prose, markdown, XML tags, or other content that breaks JSON parsing
	args = append(args, "--system-prompt", "You are a developer assistant. Your ONLY output must be valid JSON matching the provided schema. No markdown, no code fences, no XML tags, no prose, no explanations. Output raw JSON only.")

	// Build prompt with formatting instructions
	// QC review tasks come pre-formatted from BuildReviewPrompt/BuildStructuredReviewPrompt
	// Regular agent tasks need PrepareAgentPrompt for guaranteed JSON output
	prompt := task.Prompt
	if !strings.Contains(task.Name, "QC Review:") {
		// Regular agent task - add agent-specific JSON format instructions
		prompt = PrepareAgentPrompt(prompt)
	}

	// Add -p flag for non-interactive print mode (essential for automation)
	// NOTE: -p is a boolean flag, NOT a flag that takes an argument
	// The prompt must be passed as a positional argument at the END
	args = append(args, "-p")

	// Skip permissions for automation (allow file creation)
	args = append(args, "--permission-mode", "bypassPermissions")

	// Disable hooks for automation
	args = append(args, "--settings", `{"disableAllHooks": true}`)

	// JSON output for easier parsing (wrapper format, not content format)
	args = append(args, "--output-format", "json")

	// Prompt as positional argument at END (per claude --help: "Usage: claude [options] [command] [prompt]")
	args = append(args, prompt)

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
// Uses a buffer to write atomically and prevent interleaving with parallel agents
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

	// Buffer output to write atomically (prevents interleaving with parallel agents)
	var buf strings.Builder

	// Box drawing - all borders in cyan
	hLine := strings.Repeat("─", boxWidth-2)
	fmt.Fprintf(&buf, "\n%s┌%s┐%s\n", cyan, hLine, reset)

	// Header line - use runewidth for proper emoji width calculation
	headerText := "🤖 Agent Invocation"
	headerVisibleLen := runewidth.StringWidth(headerText)
	headerPad := innerWidth - headerVisibleLen
	if headerPad < 0 {
		headerPad = 0
	}
	fmt.Fprintf(&buf, "%s│%s %s%s%s%s %s│%s\n",
		cyan, reset, bold, headerText, reset, strings.Repeat(" ", headerPad), cyan, reset)
	fmt.Fprintf(&buf, "%s├%s┤%s\n", cyan, hLine, reset)

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
		fmt.Fprintf(&buf, "%s│%s %s%s:%s %s%s%s%s %s│%s\n",
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

	fmt.Fprintf(&buf, "%s└%s┘%s\n\n", cyan, hLine, reset)

	// Write entire box atomically to prevent interleaving
	os.Stderr.WriteString(buf.String())
}

// Invoke executes the claude CLI command with the given context
func (inv *Invoker) Invoke(ctx context.Context, task models.Task) (*InvocationResult, error) {
	startTime := time.Now()

	// Build command args
	args := inv.BuildCommandArgs(task)

	// DEBUG: Log raw command syntax for debugging invocation issues
	fmt.Fprintf(os.Stderr, "\n\033[33m[DEBUG] Raw Claude CLI command:\033[0m\n")
	fmt.Fprintf(os.Stderr, "\033[36m%s", inv.ClaudePath)
	for _, arg := range args {
		// Quote args containing spaces or special chars
		if strings.ContainsAny(arg, " \t\n\"'{}[]") {
			fmt.Fprintf(os.Stderr, " '%s'", strings.ReplaceAll(arg, "'", "'\"'\"'"))
		} else {
			fmt.Fprintf(os.Stderr, " %s", arg)
		}
	}
	fmt.Fprintf(os.Stderr, "\033[0m\n\n")

	// Pretty log the agent invocation
	inv.logInvocation(task, args)

	// Create command with context (for timeout)
	cmd := exec.CommandContext(ctx, inv.ClaudePath, args...)
	claude.SetCleanEnv(cmd)

	// Capture stdout and stderr separately
	// This prevents Claude CLI stderr noise (e.g., file watcher errors) from
	// polluting the agent output stored in execution history
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Log stderr warnings to terminal (but don't store in output)
	// Filter out known Claude CLI noise (file watcher errors on socket files)
	if stderr.Len() > 0 {
		stderrStr := strings.TrimSpace(stderr.String())
		// Skip known noise: EOPNOTSUPP errors from watching socket files
		if stderrStr != "" && !strings.Contains(stderrStr, "EOPNOTSUPP") {
			fmt.Fprintf(os.Stderr, "\n[Claude CLI stderr]\n%s\n", stderrStr)
		}
	}

	result := &InvocationResult{
		Output:   stdout.String(),
		Duration: time.Since(startTime),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err
		}
	}

	// Check for rate limit in raw output BEFORE JSON parsing (v2.20.1+)
	// This ensures rate limit messages like "You're out of extra usage · resets 1am"
	// are properly detected even when they fail JSON parsing
	// NOTE: Only check if command failed (err != nil) to avoid false positives
	// when agent output contains text about rate limiting (v2.28+)
	// Uses budget.ParseRateLimitFromOutput for single source of truth
	rawOutput := stdout.String()
	if err != nil && budget.ParseRateLimitFromOutput(rawOutput) != nil {
		rateLimitErr := &ErrRateLimit{RawMessage: rawOutput}
		result.Error = rateLimitErr
		return result, rateLimitErr // Return error so executeWithRateLimitRecovery can catch it
	}

	// Parse agent response from stdout
	parsedOutput, parseErr := ParseClaudeOutput(stdout.String())
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
		agentResp, parseErr := parseAgentJSON(stdout.String())
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
// If the output is not valid JSON, it attempts to extract JSON from mixed output
// (e.g., when Claude CLI prints errors to stderr before the JSON response)
func ParseClaudeOutput(output string) (*ClaudeOutput, error) {
	// First try to parse as JSON directly
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonMap); err != nil {
		// Direct parse failed - try to extract JSON from mixed output
		// Claude CLI sometimes outputs errors/warnings before the JSON response
		jsonStart := strings.Index(output, "{")
		jsonEnd := strings.LastIndex(output, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			jsonStr := output[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
				// Still can't parse - return raw output as content
				return &ClaudeOutput{Content: output}, nil
			}
			// Successfully extracted JSON from mixed output
		} else {
			// No JSON found - return raw output as content
			return &ClaudeOutput{Content: output}, nil
		}
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
	// NOTE: When using --agents flag, --json-schema may not be enforced by Claude CLI,
	// resulting in structured_output being null or empty. We must check for valid content
	// before using it, otherwise fall through to content/result fields.
	if structuredOutput, ok := jsonMap["structured_output"]; ok && structuredOutput != nil {
		// Check if it's a non-empty map (valid structured response)
		if structMap, isMap := structuredOutput.(map[string]interface{}); isMap && len(structMap) > 0 {
			// Re-serialize the structured output as JSON string for downstream parsing
			if outputBytes, err := json.Marshal(structuredOutput); err == nil {
				result.Content = string(outputBytes)
				return result, nil
			}
		}
		// If structured_output is null, empty, or not a map, fall through to other fields
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
