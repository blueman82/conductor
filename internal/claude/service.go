// Package claude provides utilities for invoking Claude CLI.
package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/budget"
)

// Service is a base type for components that invoke Claude CLI.
// It encapsulates the common invocation pattern used across multiple packages:
// pattern/ClaudeEnhancer, similarity/ClaudeSimilarity, architecture/Assessor,
// executor/IntelligentSelector, executor/TaskAgentSelector, executor/SetupIntrospector.
//
// Usage: Embed Service in your struct and use InvokeAndParse for Claude calls.
//
//	type MyComponent struct {
//	    claude.Service
//	    // ... other fields
//	}
//
//	func (c *MyComponent) DoSomething(ctx context.Context) (*Result, error) {
//	    var result Result
//	    if err := c.InvokeAndParse(ctx, prompt, schema, &result); err != nil {
//	        return nil, err
//	    }
//	    return &result, nil
//	}
type Service struct {
	inv    *Invoker
	Logger budget.WaiterLogger
}

// NewService creates a new Service with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewService(timeout time.Duration, logger budget.WaiterLogger) *Service {
	inv := NewInvoker()
	inv.Timeout = timeout
	inv.Logger = logger
	return &Service{
		inv:    inv,
		Logger: logger,
	}
}

// NewServiceWithInvoker creates a Service using an external Invoker.
// This allows sharing a single Invoker across multiple components for consistent
// configuration and rate limit handling. The invoker should already have Timeout
// and Logger configured.
func NewServiceWithInvoker(inv *Invoker) *Service {
	var logger budget.WaiterLogger
	if inv != nil {
		logger = inv.Logger
	}
	return &Service{
		inv:    inv,
		Logger: logger,
	}
}

// Invoker returns the underlying Invoker for advanced use cases.
// Most callers should use InvokeAndParse instead.
func (s *Service) Invoker() *Invoker {
	return s.inv
}

// InvokeAndParse invokes Claude CLI with the given prompt and schema,
// then parses the response into the provided result pointer.
//
// This consolidates the common invocation flow:
//  1. Create Request with prompt and schema
//  2. Invoke Claude CLI (with automatic rate limit retry)
//  3. Parse the response to extract JSON content
//  4. Check for empty response
//  5. Unmarshal JSON into the result
//
// The result parameter must be a pointer to a struct that can be unmarshaled
// from the JSON response.
//
// Returns an error if:
//   - Claude invocation fails
//   - Response parsing fails
//   - Response is empty
//   - JSON unmarshaling fails
func (s *Service) InvokeAndParse(ctx context.Context, prompt, schema string, result interface{}) error {
	req := Request{
		Prompt: prompt,
		Schema: schema,
	}

	resp, err := s.inv.Invoke(ctx, req)
	if err != nil {
		return err
	}

	content, _, err := ParseResponse(resp.RawOutput)
	if err != nil {
		return fmt.Errorf("failed to parse claude output: %w", err)
	}

	if content == "" {
		return fmt.Errorf("empty response from claude")
	}

	if err := json.Unmarshal([]byte(content), result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w (content: %s)", err, truncate(content, 200))
	}

	return nil
}

// InvokeAndParseWithFallback is like InvokeAndParse but attempts JSON extraction
// from mixed content if initial unmarshaling fails. This handles edge cases where
// Claude outputs prose before/after JSON.
//
// Use this when the response might contain non-JSON content mixed with the JSON payload.
func (s *Service) InvokeAndParseWithFallback(ctx context.Context, prompt, schema string, result interface{}) error {
	req := Request{
		Prompt: prompt,
		Schema: schema,
	}

	resp, err := s.inv.Invoke(ctx, req)
	if err != nil {
		return err
	}

	content, _, err := ParseResponse(resp.RawOutput)
	if err != nil {
		return fmt.Errorf("failed to parse claude output: %w", err)
	}

	if content == "" {
		return fmt.Errorf("empty response from claude")
	}

	if err := json.Unmarshal([]byte(content), result); err != nil {
		// Fallback: try to extract JSON from content
		if extracted := ExtractJSON(content); extracted != "" {
			if err := json.Unmarshal([]byte(extracted), result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w (content: %s)", err, truncate(content, 200))
			}
			return nil
		}
		return fmt.Errorf("failed to unmarshal response: %w (content: %s)", err, truncate(content, 200))
	}

	return nil
}

// ExtractJSON attempts to extract a JSON object from mixed content.
// It finds the first '{' and last '}' to extract the JSON substring.
// Returns empty string if no valid JSON boundaries found.
func ExtractJSON(content string) string {
	start := -1
	end := -1

	for i, c := range content {
		if c == '{' {
			start = i
			break
		}
	}

	for i := len(content) - 1; i >= 0; i-- {
		if content[i] == '}' {
			end = i
			break
		}
	}

	if start >= 0 && end > start {
		return content[start : end+1]
	}
	return ""
}

// truncate returns s truncated to maxLen characters with "..." suffix if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
