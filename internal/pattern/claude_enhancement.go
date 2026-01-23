package pattern

import (
	"context"
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
// Embeds claude.Service for CLI invocation with rate limit handling.
type ClaudeEnhancer struct {
	claude.Service
}

// NewClaudeEnhancer creates an enhancer with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewClaudeEnhancer(timeout time.Duration, logger budget.WaiterLogger) *ClaudeEnhancer {
	return &ClaudeEnhancer{
		Service: *claude.NewService(timeout, logger),
	}
}

// NewClaudeEnhancerWithInvoker creates an enhancer using an external Invoker.
// This allows sharing a single Invoker across multiple components for consistent
// configuration and rate limit handling. The invoker should already have Timeout
// and Logger configured.
func NewClaudeEnhancerWithInvoker(inv *claude.Invoker) *ClaudeEnhancer {
	return &ClaudeEnhancer{
		Service: *claude.NewServiceWithInvoker(inv),
	}
}

// Enhance calls Claude for confidence refinement with rate limit retry
func (ce *ClaudeEnhancer) Enhance(ctx context.Context, taskDesc string, patterns string, baseConfidence float64) (*EnhancementResult, error) {
	prompt := ce.buildPrompt(taskDesc, patterns, baseConfidence)

	var result EnhancementResult
	if err := ce.InvokeAndParse(ctx, prompt, EnhancementSchema(), &result); err != nil {
		return nil, err
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
