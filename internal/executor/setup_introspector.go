package executor

import (
	"time"

	"github.com/harrison/conductor/internal/budget"
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
