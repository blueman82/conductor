package pattern

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
)

// EnhancementResult contains Claude's confidence assessment
type EnhancementResult struct {
	AdjustedConfidence float64  `json:"adjusted_confidence"`
	Reasoning          string   `json:"reasoning"`
	RiskFactors        []string `json:"risk_factors"`
}

// ClaudeEnhancer enhances pattern confidence using Claude CLI.
// Uses claude.Invoker for CLI invocation with rate limit handling.
type ClaudeEnhancer struct {
	inv    *claude.Invoker     // Invoker handles CLI invocation and rate limit retry
	Logger budget.WaiterLogger // For TTS + visual during rate limit wait (passed to Invoker)
}

// NewClaudeEnhancer creates an enhancer with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewClaudeEnhancer(timeout time.Duration, logger budget.WaiterLogger) *ClaudeEnhancer {
	inv := claude.NewInvoker()
	inv.Timeout = timeout
	inv.Logger = logger
	return &ClaudeEnhancer{
		inv:    inv,
		Logger: logger,
	}
}

// Enhance calls Claude for confidence refinement with rate limit retry
func (ce *ClaudeEnhancer) Enhance(ctx context.Context, taskDesc string, patterns string, baseConfidence float64) (*EnhancementResult, error) {
	prompt := ce.buildPrompt(taskDesc, patterns, baseConfidence)

	req := claude.Request{
		Prompt: prompt,
		Schema: EnhancementSchema(),
	}

	// Invoke Claude CLI (rate limit handling is in Invoker)
	resp, err := ce.inv.Invoke(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse the response
	content, _, err := claude.ParseResponse(resp.RawOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w", err)
	}

	if content == "" {
		return nil, fmt.Errorf("empty response from claude")
	}

	var result EnhancementResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse enhancement result: %w", err)
	}

	return &result, nil
}

func (ce *ClaudeEnhancer) buildPrompt(taskDesc, patterns string, baseConfidence float64) string {
	return fmt.Sprintf(`Analyze this task and refine the confidence assessment.

Task: %s

Pattern Analysis Results:
%s

Current Confidence: %.2f (uncertain range 0.3-0.7)

Assess whether this task:
1. Has sufficient prior art to proceed confidently
2. Has risk factors that lower confidence
3. Should have adjusted confidence based on patterns found

Consider:
- If similar patterns succeeded before, confidence should increase
- If patterns show failures or complexity, confidence should decrease
- If no relevant patterns found, maintain base confidence

Respond with JSON only.`, taskDesc, patterns, baseConfidence)
}

// EnhancementSchema returns the JSON schema for enforcement
func EnhancementSchema() string {
	return `{"type":"object","properties":{"adjusted_confidence":{"type":"number","minimum":0,"maximum":1},"reasoning":{"type":"string"},"risk_factors":{"type":"array","items":{"type":"string"}}},"required":["adjusted_confidence","reasoning"]}`
}
