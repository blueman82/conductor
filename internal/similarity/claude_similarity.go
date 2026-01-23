package similarity

import (
	"context"
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
// Embeds claude.Service for CLI invocation with rate limit handling.
type ClaudeSimilarity struct {
	claude.Service
}

// NewClaudeSimilarity creates a similarity instance with the specified timeout.
// The timeout parameter controls how long to wait for Claude CLI responses.
// Use config.DefaultTimeoutsConfig().LLM for the standard timeout value.
func NewClaudeSimilarity(timeout time.Duration, logger budget.WaiterLogger) *ClaudeSimilarity {
	return &ClaudeSimilarity{
		Service: *claude.NewService(timeout, logger),
	}
}

// NewClaudeSimilarityWithInvoker creates a similarity instance using an external Invoker.
// This allows sharing a single Invoker across multiple components for consistent
// configuration and rate limit handling. The invoker should already have Timeout
// and Logger configured.
func NewClaudeSimilarityWithInvoker(inv *claude.Invoker) *ClaudeSimilarity {
	return &ClaudeSimilarity{
		Service: *claude.NewServiceWithInvoker(inv),
	}
}

// Compare computes semantic similarity between two descriptions using Claude
func (cs *ClaudeSimilarity) Compare(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error) {
	prompt := cs.buildPrompt(desc1, desc2)

	var result SimilarityResult
	if err := cs.InvokeAndParse(ctx, prompt, SimilaritySchema(), &result); err != nil {
		return nil, err
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
