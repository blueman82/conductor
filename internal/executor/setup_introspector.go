package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
)

// SetupCommand represents a single setup command to run before wave execution
type SetupCommand struct {
	Command  string `json:"command"`  // The command to execute (e.g., "npm install", "go mod tidy")
	Purpose  string `json:"purpose"`  // Human-readable description of what this command does
	Required bool   `json:"required"` // True if task execution should fail if this command fails
}

// SetupResult contains the result of Claude's project introspection
type SetupResult struct {
	Commands  []SetupCommand `json:"commands"`  // List of setup commands to run
	Reasoning string         `json:"reasoning"` // Explanation of why these commands are needed
}

// SetupIntrospector uses Claude CLI to analyze a project and determine setup commands.
// Follows the ClaudeSimilarity pattern from internal/similarity/claude_similarity.go
type SetupIntrospector struct {
	Timeout    time.Duration
	ClaudePath string
	Logger     budget.WaiterLogger // For TTS + visual during rate limit wait
}

// NewSetupIntrospector creates a setup introspector with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewSetupIntrospector(timeout time.Duration, logger budget.WaiterLogger) *SetupIntrospector {
	return &SetupIntrospector{
		Timeout:    timeout,
		ClaudePath: "claude",
		Logger:     logger,
	}
}

// SetupSchema returns the JSON schema for Claude CLI enforcement.
// This schema ensures Claude returns the expected SetupResult structure.
func SetupSchema() string {
	return `{
		"type": "object",
		"properties": {
			"commands": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"command": {
							"type": "string",
							"description": "The shell command to execute"
						},
						"purpose": {
							"type": "string",
							"description": "Human-readable description of what this command does"
						},
						"required": {
							"type": "boolean",
							"description": "True if task execution should fail if this command fails"
						}
					},
					"required": ["command", "purpose", "required"],
					"additionalProperties": false
				},
				"description": "List of setup commands to run before wave execution"
			},
			"reasoning": {
				"type": "string",
				"description": "Explanation of why these commands are needed based on project analysis"
			}
		},
		"required": ["commands", "reasoning"],
		"additionalProperties": false
	}`
}

// Introspect calls Claude CLI to analyze the project and determine setup commands
func (si *SetupIntrospector) Introspect(ctx context.Context) (*SetupResult, error) {
	result, err := si.invoke(ctx)

	// Handle rate limit with retry (TTS + visual countdown)
	// Wait for actual reset time from Claude output - no arbitrary caps
	if err != nil {
		if info := budget.ParseRateLimitFromError(err.Error()); info != nil {
			// Use 24h as max - waiter uses actual reset time from info
			waiter := budget.NewRateLimitWaiter(24*time.Hour, 15*time.Second, 30*time.Second, si.Logger)
			if waiter.ShouldWait(info) {
				if waitErr := waiter.WaitForReset(ctx, info); waitErr != nil {
					return nil, waitErr
				}
				// Retry once after wait
				return si.invoke(ctx)
			}
		}
		return nil, err
	}

	return result, nil
}

// invoke performs the actual Claude CLI call (follows ClaudeSimilarity.invoke() pattern)
func (si *SetupIntrospector) invoke(ctx context.Context) (*SetupResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, si.Timeout)
	defer cancel()

	prompt, err := si.buildPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	args := []string{
		"-p", prompt,
		"--json-schema", SetupSchema(),
		"--output-format", "json",
		"--settings", `{"disableAllHooks": true}`,
	}

	cmd := exec.CommandContext(ctxWithTimeout, si.ClaudePath, args...)
	claude.SetCleanEnv(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude invocation failed: %w (output: %s)", err, string(output))
	}

	parsed, err := agent.ParseClaudeOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}

	if parsed.Content == "" {
		return nil, fmt.Errorf("empty response from claude")
	}

	var result SetupResult
	if err := json.Unmarshal([]byte(parsed.Content), &result); err != nil {
		// Try extracting JSON from mixed output (fallback)
		start := strings.Index(parsed.Content, "{")
		end := strings.LastIndex(parsed.Content, "}")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(parsed.Content[start:end+1]), &result); err != nil {
				return nil, fmt.Errorf("failed to extract JSON: %w", err)
			}
			return &result, nil
		}
		return nil, fmt.Errorf("failed to parse setup result: %w", err)
	}

	return &result, nil
}

// buildPrompt generates the introspection prompt with project file listing
func (si *SetupIntrospector) buildPrompt() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Collect project files (limit depth and count for reasonable prompt size)
	var files []string
	maxFiles := 100
	maxDepth := 3

	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Calculate depth
		relPath, _ := filepath.Rel(cwd, path)
		depth := strings.Count(relPath, string(os.PathSeparator))
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories and common non-essential directories
		name := info.Name()
		if info.IsDir() && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__") {
			return filepath.SkipDir
		}

		// Skip hidden files and binary/build outputs
		if !info.IsDir() && !strings.HasPrefix(name, ".") {
			if len(files) < maxFiles {
				files = append(files, relPath)
			}
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to list project files: %w", err)
	}

	fileList := strings.Join(files, "\n")

	return fmt.Sprintf(`Analyze this project and determine what setup commands need to run before task execution.

Project files:
%s

Based on the project structure, identify:
1. Package manager commands (npm install, go mod tidy, pip install, cargo build, etc.)
2. Build prerequisites (compile assets, generate code, etc.)
3. Environment setup (database migrations, config generation, etc.)

For each command:
- Set "required" to true for essential commands (missing dependencies = task failure)
- Set "required" to false for optional/nice-to-have commands

Return ONLY commands that are actually needed based on project indicators (package.json, go.mod, requirements.txt, Cargo.toml, etc.).
If no setup is needed, return an empty commands array.

Respond with JSON only.`, fileList), nil
}

// RunSetupCommands executes the setup commands returned by Introspect
func (si *SetupIntrospector) RunSetupCommands(ctx context.Context, result *SetupResult) error {
	if result == nil || len(result.Commands) == 0 {
		return nil
	}

	for i, cmd := range result.Commands {
		// Execute command
		execCmd := exec.CommandContext(ctx, "sh", "-c", cmd.Command)
		output, err := execCmd.CombinedOutput()

		if err != nil {
			if cmd.Required {
				return fmt.Errorf("required setup command %d/%d failed: %s\nCommand: %s\nOutput: %s\nError: %w",
					i+1, len(result.Commands), cmd.Purpose, cmd.Command, string(output), err)
			}
			// Optional command failed - continue execution
			fmt.Printf("setup: optional command failed (continuing): %s - %v\n", cmd.Purpose, err)
			continue
		}

		// Command completed successfully
		fmt.Printf("setup: command %d/%d completed: %s\n", i+1, len(result.Commands), cmd.Purpose)
	}

	return nil
}
