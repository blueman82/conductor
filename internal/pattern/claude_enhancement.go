package pattern

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
)

// EnhancementResult contains Claude's confidence assessment
type EnhancementResult struct {
	AdjustedConfidence float64  `json:"adjusted_confidence"`
	Reasoning          string   `json:"reasoning"`
	RiskFactors        []string `json:"risk_factors"`
}

// ClaudeEnhancer enhances pattern confidence using Claude CLI
type ClaudeEnhancer struct {
	Timeout    time.Duration
	ClaudePath string
	Logger     budget.WaiterLogger // For TTS + visual during rate limit wait
}

// NewClaudeEnhancer creates an enhancer with defaults
func NewClaudeEnhancer(logger budget.WaiterLogger) *ClaudeEnhancer {
	return &ClaudeEnhancer{
		Timeout:    30 * time.Second,
		ClaudePath: "claude",
		Logger:     logger,
	}
}

// NewClaudeEnhancerWithConfig creates an enhancer from config values
func NewClaudeEnhancerWithConfig(timeout time.Duration, logger budget.WaiterLogger) *ClaudeEnhancer {
	return &ClaudeEnhancer{
		Timeout:    timeout,
		ClaudePath: "claude",
		Logger:     logger,
	}
}

// ShouldEnhance returns true if confidence is in uncertain range
func (ce *ClaudeEnhancer) ShouldEnhance(confidence float64, minConf, maxConf float64) bool {
	return confidence >= minConf && confidence <= maxConf
}

// Enhance calls Claude for confidence refinement with rate limit retry
func (ce *ClaudeEnhancer) Enhance(ctx context.Context, taskDesc string, patterns string, baseConfidence float64) (*EnhancementResult, error) {
	result, err := ce.invoke(ctx, taskDesc, patterns, baseConfidence)

	// Handle rate limit with retry (TTS + visual countdown)
	// Wait for actual reset time from Claude output - no arbitrary caps
	if err != nil {
		if info := budget.ParseRateLimitFromError(err.Error()); info != nil {
			// Use 24h as max - waiter uses actual reset time from info
			waiter := budget.NewRateLimitWaiter(24*time.Hour, 15*time.Second, 30*time.Second, ce.Logger)
			if waiter.ShouldWait(info) {
				if waitErr := waiter.WaitForReset(ctx, info); waitErr != nil {
					return nil, waitErr
				}
				// Retry once after wait
				return ce.invoke(ctx, taskDesc, patterns, baseConfidence)
			}
		}
		return nil, err
	}

	return result, nil
}

// invoke performs the actual Claude CLI call (follows qc_intelligent.go pattern)
func (ce *ClaudeEnhancer) invoke(ctx context.Context, taskDesc, patterns string, baseConfidence float64) (*EnhancementResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ce.Timeout)
	defer cancel()

	prompt := ce.buildPrompt(taskDesc, patterns, baseConfidence)

	args := []string{
		"-p", prompt,
		"--json-schema", EnhancementSchema(),
		"--output-format", "json",
		"--settings", `{"disableAllHooks": true}`,
	}

	cmd := exec.CommandContext(ctxWithTimeout, ce.ClaudePath, args...)
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

	var result EnhancementResult
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
