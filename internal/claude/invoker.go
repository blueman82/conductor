// Package claude provides utilities for invoking Claude CLI.
package claude

import (
	"context"
	"fmt"
	"os/exec"
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

// Invoke executes a Claude CLI command with rate limit retry.
// It creates a context with timeout from inv.Timeout (if set), calls the internal invoke() helper,
// and on rate limit error waits and retries once.
//
// The method follows the pattern from ClaudeSimilarity.Compare:
//  1. Call internal invoke() helper
//  2. On error, check for rate limit via budget.ParseRateLimitFromError
//  3. If rate limit, wait using budget.NewRateLimitWaiter and retry once
//  4. Return Response with raw output
func (inv *Invoker) Invoke(ctx context.Context, req Request) (*Response, error) {
	// Create context with timeout if Invoker has Timeout set
	ctxToUse := ctx
	var cancel context.CancelFunc
	if inv.Timeout > 0 {
		ctxToUse, cancel = context.WithTimeout(ctx, inv.Timeout)
		defer cancel()
	}

	result, err := inv.invoke(ctxToUse, req)

	// Handle rate limit with retry (follows ClaudeSimilarity pattern)
	if err != nil {
		if info := budget.ParseRateLimitFromError(err.Error()); info != nil {
			// Use 24h as max - waiter uses actual reset time from info
			waiter := budget.NewRateLimitWaiter(24*time.Hour, 15*time.Second, 30*time.Second, inv.Logger)
			if waiter.ShouldWait(info) {
				if waitErr := waiter.WaitForReset(ctxToUse, info); waitErr != nil {
					return nil, waitErr
				}
				// Retry once after wait
				return inv.invoke(ctxToUse, req)
			}
		}
		return nil, err
	}

	return result, nil
}

// invoke performs the actual Claude CLI call.
// Always includes: --system-prompt, -p, --json-schema (if set), --output-format json, --settings
// Optional flags based on Request fields: AgentJSON -> --agents, ResumeID -> --resume, BypassPerms -> --permission-mode
func (inv *Invoker) invoke(ctx context.Context, req Request) (*Response, error) {
	// Build command arguments
	args := []string{}

	// Resume from previous session if available (for rate limit recovery)
	if req.ResumeID != "" {
		args = append(args, "--resume", req.ResumeID)
	}

	// Add agent definition if provided
	if req.AgentJSON != "" {
		args = append(args, "--agents", req.AgentJSON)
	}

	// System prompt is always included (use default if not set on Invoker)
	systemPrompt := inv.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = DefaultSystemPrompt
	}
	args = append(args, "--system-prompt", systemPrompt)

	// Prompt is required
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	args = append(args, "-p", req.Prompt)

	// JSON schema for structured responses (optional)
	if req.Schema != "" {
		args = append(args, "--json-schema", req.Schema)
	}

	// JSON output format
	args = append(args, "--output-format", "json")

	// Bypass permissions if requested
	if req.BypassPerms {
		args = append(args, "--permission-mode", "bypassPermissions")
	}

	// Disable hooks for automation
	args = append(args, "--settings", `{"disableAllHooks": true}`)

	// Determine claude binary path
	claudePath := inv.ClaudePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, claudePath, args...)
	SetCleanEnv(cmd)

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude invocation failed: %w (output: %s)", err, string(output))
	}

	return &Response{
		RawOutput: output,
	}, nil
}
