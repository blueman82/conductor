// Package claude provides utilities for invoking Claude CLI.
package claude

import (
	"time"

	"github.com/harrison/conductor/internal/budget"
)

// DefaultSystemPrompt is the standard system prompt enforcing JSON-only output.
// This prevents agents from outputting prose, markdown, XML tags, or other content
// that breaks JSON parsing.
const DefaultSystemPrompt = "You are a developer assistant. Your ONLY output must be valid JSON matching the provided schema. No markdown, no code fences, no XML tags, no prose, no explanations. Output raw JSON only."

// Invoker is a reusable client for invoking Claude CLI commands.
// It follows the http.Client pattern: create once, use many times.
// Thread-safe for concurrent use.
type Invoker struct {
	// ClaudePath is the path to the claude CLI binary.
	// Defaults to "claude" (found in PATH).
	ClaudePath string

	// Timeout is the default timeout for invocations.
	// Can be overridden per-request via context.
	Timeout time.Duration

	// Logger receives rate limit countdown notifications.
	// Can be nil for silent operation.
	Logger budget.WaiterLogger

	// SystemPrompt is the system prompt sent with all invocations.
	// Defaults to DefaultSystemPrompt if empty when using NewInvoker.
	SystemPrompt string
}

// Request holds per-invocation configuration for a Claude CLI call.
// Create a new Request for each invocation.
type Request struct {
	// Prompt is the user prompt to send to Claude (required).
	Prompt string

	// Schema is the JSON schema for structured output (optional).
	// When set, enforces response structure via --json-schema flag.
	Schema string

	// AgentJSON is the serialized agent definition for --agents flag (optional).
	// Format: {"agent-name": {"description": "...", "prompt": "...", "tools": [...]}}
	AgentJSON string

	// ResumeID is a session ID to resume from (optional).
	// Used for rate limit recovery.
	ResumeID string

	// BypassPerms enables --permission-mode bypassPermissions (optional).
	// Allows file creation without permission prompts.
	BypassPerms bool
}

// Response holds the raw output from a Claude CLI invocation.
// The caller is responsible for unmarshaling RawOutput into the appropriate type.
type Response struct {
	// RawOutput contains the raw bytes from Claude CLI stdout.
	// The caller should unmarshal this into the expected response type.
	RawOutput []byte

	// SessionID is the Claude CLI session identifier.
	// Used for resuming sessions (e.g., after rate limit recovery).
	SessionID string
}

// NewInvoker creates a new Invoker with default settings.
// ClaudePath defaults to "claude", SystemPrompt defaults to DefaultSystemPrompt.
func NewInvoker() *Invoker {
	return &Invoker{
		ClaudePath:   "claude",
		SystemPrompt: DefaultSystemPrompt,
	}
}
