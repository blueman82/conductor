package similarity

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

// SimilarityResult contains Claude's semantic similarity assessment
type SimilarityResult struct {
	Score     float64 `json:"score"`      // 0.0 to 1.0 similarity score
	Reasoning string  `json:"reasoning"`  // Explanation of the similarity assessment
}

// Similarity interface defines the contract for semantic comparison implementations
type Similarity interface {
	Compare(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error)
}

// ClaudeSimilarity computes semantic similarity using Claude CLI
// Follows the ClaudeEnhancer pattern from internal/pattern/claude_enhancement.go
type ClaudeSimilarity struct {
	Timeout    time.Duration
	ClaudePath string
	Logger     budget.WaiterLogger // For TTS + visual during rate limit wait
}

// NewClaudeSimilarity creates a similarity instance with defaults
func NewClaudeSimilarity(logger budget.WaiterLogger) *ClaudeSimilarity {
	return &ClaudeSimilarity{
		Timeout:    30 * time.Second,
		ClaudePath: "claude",
		Logger:     logger,
	}
}

// NewClaudeSimilarityWithConfig creates a similarity instance with custom timeout
func NewClaudeSimilarityWithConfig(timeout time.Duration, logger budget.WaiterLogger) *ClaudeSimilarity {
	return &ClaudeSimilarity{
		Timeout:    timeout,
		ClaudePath: "claude",
		Logger:     logger,
	}
}

// Compare computes semantic similarity between two descriptions using Claude
func (cs *ClaudeSimilarity) Compare(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error) {
	result, err := cs.invoke(ctx, desc1, desc2)

	// Handle rate limit with retry (TTS + visual countdown)
	// Wait for actual reset time from Claude output - no arbitrary caps
	if err != nil {
		if info := budget.ParseRateLimitFromError(err.Error()); info != nil {
			// Use 24h as max - waiter uses actual reset time from info
			waiter := budget.NewRateLimitWaiter(24*time.Hour, 15*time.Second, 30*time.Second, cs.Logger)
			if waiter.ShouldWait(info) {
				if waitErr := waiter.WaitForReset(ctx, info); waitErr != nil {
					return nil, waitErr
				}
				// Retry once after wait
				return cs.invoke(ctx, desc1, desc2)
			}
		}
		return nil, err
	}

	return result, nil
}

// invoke performs the actual Claude CLI call (follows qc_intelligent.go pattern)
func (cs *ClaudeSimilarity) invoke(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cs.Timeout)
	defer cancel()

	prompt := cs.buildPrompt(desc1, desc2)

	args := []string{
		"-p", prompt,
		"--json-schema", SimilaritySchema(),
		"--output-format", "json",
		"--settings", `{"disableAllHooks": true}`,
	}

	cmd := exec.CommandContext(ctxWithTimeout, cs.ClaudePath, args...)
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

	var result SimilarityResult
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
		return nil, fmt.Errorf("failed to parse similarity result: %w", err)
	}

	return &result, nil
}

func (cs *ClaudeSimilarity) buildPrompt(desc1, desc2 string) string {
	return fmt.Sprintf(`Compare the semantic similarity of these two descriptions.

Description 1:
%s

Description 2:
%s

Analyze the semantic meaning, intent, and key concepts of both descriptions.
Rate their similarity from 0.0 (completely different) to 1.0 (essentially identical).

Consider:
- Shared concepts and terminology
- Similar intent or purpose
- Overlapping functionality or scope
- Structural similarities

Respond with JSON only.`, desc1, desc2)
}

// SimilaritySchema returns the JSON schema for enforcement
func SimilaritySchema() string {
	return `{"type":"object","properties":{"score":{"type":"number","minimum":0,"maximum":1},"reasoning":{"type":"string"}},"required":["score","reasoning"]}`
}
