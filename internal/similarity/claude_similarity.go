package similarity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/budget"
	"github.com/harrison/conductor/internal/claude"
)

// SimilarityResult contains Claude's semantic similarity assessment
type SimilarityResult struct {
	Score         float64 `json:"score"`          // 0.0 to 1.0 similarity score
	Reasoning     string  `json:"reasoning"`      // Explanation of the similarity assessment
	SemanticMatch bool    `json:"semantic_match"` // True if similarity meets threshold for semantic equivalence
}

// Similarity interface defines the contract for semantic comparison implementations
type Similarity interface {
	Compare(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error)
}

// ClaudeSimilarity computes semantic similarity using Claude CLI.
// Uses claude.Invoker for CLI invocation with rate limit handling.
type ClaudeSimilarity struct {
	inv    *claude.Invoker     // Invoker handles CLI invocation and rate limit retry
	Logger budget.WaiterLogger // For TTS + visual during rate limit wait (passed to Invoker)
}

// NewClaudeSimilarity creates a similarity instance with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewClaudeSimilarity(timeout time.Duration, logger budget.WaiterLogger) *ClaudeSimilarity {
	inv := claude.NewInvoker()
	inv.Timeout = timeout
	inv.Logger = logger
	return &ClaudeSimilarity{
		inv:    inv,
		Logger: logger,
	}
}

// Compare computes semantic similarity between two descriptions using Claude
func (cs *ClaudeSimilarity) Compare(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error) {
	prompt := cs.buildPrompt(desc1, desc2)

	req := claude.Request{
		Prompt: prompt,
		Schema: SimilaritySchema(),
	}

	// Invoke Claude CLI (rate limit handling is in Invoker)
	resp, err := cs.inv.Invoke(ctx, req)
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

	var result SimilarityResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
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

// Note: SimilaritySchema is defined in schema.go
