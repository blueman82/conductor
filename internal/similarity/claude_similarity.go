package similarity

import (
	"context"
	"fmt"
	"strings"
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

// batchSimilarityResponse holds the parsed batch comparison response
type batchSimilarityResponse struct {
	Scores []float64 `json:"scores"`
}

// Similarity interface defines the contract for semantic comparison implementations
type Similarity interface {
	Compare(ctx context.Context, desc1, desc2 string) (*SimilarityResult, error)
	CompareBatch(ctx context.Context, query string, candidates []string) ([]float64, error)
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

// CompareBatch computes semantic similarity between a query and multiple candidates in a single call.
// Returns scores in the same order as input candidates. Returns nil, nil for empty input.
func (cs *ClaudeSimilarity) CompareBatch(ctx context.Context, query string, candidates []string) ([]float64, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	prompt := cs.buildBatchPrompt(query, candidates)

	var result batchSimilarityResponse
	if err := cs.InvokeAndParse(ctx, prompt, BatchSimilaritySchema(), &result); err != nil {
		return nil, err
	}

	// Validate response length matches input
	if len(result.Scores) != len(candidates) {
		return nil, fmt.Errorf("score count mismatch: got %d, expected %d", len(result.Scores), len(candidates))
	}

	return result.Scores, nil
}

func (cs *ClaudeSimilarity) buildBatchPrompt(query string, candidates []string) string {
	var sb strings.Builder
	sb.WriteString("Compare the semantic similarity of a query against multiple candidates.\n\n")
	sb.WriteString("Query:\n")
	sb.WriteString(query)
	sb.WriteString("\n\nCandidates:\n")

	for i, c := range candidates {
		fmt.Fprintf(&sb, "[%d] %s\n", i, c)
	}

	sb.WriteString(`
Rate each candidate's similarity to the query from 0.0 (completely different) to 1.0 (essentially identical).

Consider for each:
- Shared concepts and terminology
- Similar intent or purpose
- Overlapping functionality or scope

Return a JSON object with a "scores" array containing one score per candidate, in the same order as listed above.
Example for 3 candidates: {"scores": [0.8, 0.3, 0.95]}`)

	return sb.String()
}

// Note: SimilaritySchema is defined in schema.go
