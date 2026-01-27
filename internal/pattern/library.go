package pattern

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/similarity"
)

// PatternLibrary provides methods for storing and retrieving successful task execution patterns.
// It wraps learning.Store with pattern-specific logic and uses TaskHasher for similarity matching.
type PatternLibrary struct {
	store  *learning.Store
	hasher *TaskHasher
	config *config.PatternConfig
}

// StoredPattern represents a pattern stored in the library with similarity information.
type StoredPattern struct {
	// TaskHash is the hash of the task description
	TaskHash string `json:"task_hash"`

	// Description is the task description
	Description string `json:"description"`

	// LastAgent is the agent that most recently completed this task successfully
	LastAgent string `json:"last_agent"`

	// SuccessCount is the number of times this pattern has succeeded
	SuccessCount int `json:"success_count"`

	// LastUsed is when this pattern was last used
	LastUsed time.Time `json:"last_used"`

	// CreatedAt is when this pattern was first stored
	CreatedAt time.Time `json:"created_at"`

	// Similarity is the similarity score to a query (only set during retrieval)
	Similarity float64 `json:"similarity,omitempty"`

	// Metadata contains additional pattern context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AgentRecommendation represents a recommended agent based on pattern history.
type AgentRecommendation struct {
	// Agent is the recommended agent name
	Agent string `json:"agent"`

	// SuccessCount is total successes for this agent on similar patterns
	SuccessCount int `json:"success_count"`

	// Confidence is the confidence in this recommendation (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// MatchingPatterns are the patterns that led to this recommendation
	MatchingPatterns []StoredPattern `json:"matching_patterns"`
}

// NewPatternLibrary creates a new PatternLibrary with the given store and configuration.
// If store is nil, operations will return empty results gracefully.
func NewPatternLibrary(store *learning.Store, cfg *config.PatternConfig) *PatternLibrary {
	if cfg == nil {
		defaultCfg := config.DefaultPatternConfig()
		cfg = &defaultCfg
	}

	return &PatternLibrary{
		store:  store,
		hasher: NewTaskHasher(),
		config: cfg,
	}
}

// Store saves a successful pattern to the library.
// It calculates the hash from the description, then stores the pattern with the given agent.
func (l *PatternLibrary) Store(ctx context.Context, description string, files []string, agent string) error {
	if l.store == nil {
		return nil // Graceful no-op when no store is available
	}

	if description == "" {
		return fmt.Errorf("pattern description cannot be empty")
	}

	// Calculate hash
	hashResult := l.hasher.Hash(description, files)

	// Prepare metadata
	metadata := map[string]interface{}{
		"files": files,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	// Create pattern for storage
	pattern := &learning.SuccessfulPattern{
		TaskHash:           hashResult.FullHash,
		PatternDescription: description,
		LastAgent:          agent,
		Metadata:           string(metadataJSON),
	}

	// Store pattern (AddPattern handles upsert and success count increment)
	if err := l.store.AddPattern(ctx, pattern); err != nil {
		return fmt.Errorf("add pattern: %w", err)
	}

	return nil
}

// Retrieve finds patterns similar to the given description.
// Uses hash prefix matching to find candidate patterns.
// Returns patterns sorted by hash match quality (highest first).
// For semantic similarity scoring, use RetrieveWithSimilarity instead.
func (l *PatternLibrary) Retrieve(ctx context.Context, description string, files []string, limit int) ([]StoredPattern, error) {
	if l.store == nil {
		return []StoredPattern{}, nil // Graceful fallback when no store
	}

	if description == "" {
		return []StoredPattern{}, nil
	}

	if limit <= 0 {
		limit = l.config.MaxPatternsPerTask
		if limit <= 0 {
			limit = 10
		}
	}

	// Calculate hash for the query
	hashResult := l.hasher.Hash(description, files)

	// Use normalized hash prefix for initial matching (8 characters)
	hashPrefix := hashResult.NormalizedHash
	if len(hashPrefix) > 8 {
		hashPrefix = hashPrefix[:8]
	}

	// Query similar patterns from store using hash prefix
	dbPatterns, err := l.store.GetSimilarPatterns(ctx, hashPrefix, limit*3)
	if err != nil {
		return nil, fmt.Errorf("query similar patterns: %w", err)
	}

	// Also get top patterns to ensure we find matches even when hash prefix differs
	// This provides broader coverage for semantic similarity
	topPatterns, err := l.store.GetTopPatterns(ctx, limit*3)
	if err != nil {
		return nil, fmt.Errorf("query top patterns: %w", err)
	}

	// Merge pattern lists, avoiding duplicates
	seen := make(map[string]bool)
	var allPatterns []*learning.SuccessfulPattern
	for _, p := range dbPatterns {
		if !seen[p.TaskHash] {
			seen[p.TaskHash] = true
			allPatterns = append(allPatterns, p)
		}
	}
	for _, p := range topPatterns {
		if !seen[p.TaskHash] {
			seen[p.TaskHash] = true
			allPatterns = append(allPatterns, p)
		}
	}

	// Convert to StoredPattern (no similarity score without ClaudeSimilarity)
	results := make([]StoredPattern, 0, len(allPatterns))
	for _, p := range allPatterns {
		// Parse metadata
		var metadata map[string]interface{}
		if p.Metadata != "" {
			json.Unmarshal([]byte(p.Metadata), &metadata)
		}

		results = append(results, StoredPattern{
			TaskHash:     p.TaskHash,
			Description:  p.PatternDescription,
			LastAgent:    p.LastAgent,
			SuccessCount: p.SuccessCount,
			LastUsed:     p.LastUsed,
			CreatedAt:    p.CreatedAt,
			Similarity:   0.0, // Cannot compute without ClaudeSimilarity
			Metadata:     metadata,
		})
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// RetrieveWithSimilarity finds patterns similar to the given description using Claude semantic matching.
// Uses hash prefix matching to find candidate patterns, then scores each with ClaudeSimilarity.
// Returns patterns sorted by similarity (highest first).
func (l *PatternLibrary) RetrieveWithSimilarity(ctx context.Context, description string, files []string, limit int, sim *similarity.ClaudeSimilarity) ([]StoredPattern, error) {
	if l.store == nil {
		return []StoredPattern{}, nil // Graceful fallback when no store
	}

	if description == "" {
		return []StoredPattern{}, nil
	}

	if limit <= 0 {
		limit = l.config.MaxPatternsPerTask
		if limit <= 0 {
			limit = 10
		}
	}

	// Calculate hash for the query
	hashResult := l.hasher.Hash(description, files)

	// Use normalized hash prefix for initial matching (8 characters)
	hashPrefix := hashResult.NormalizedHash
	if len(hashPrefix) > 8 {
		hashPrefix = hashPrefix[:8]
	}

	// Query similar patterns from store using hash prefix
	dbPatterns, err := l.store.GetSimilarPatterns(ctx, hashPrefix, limit*3)
	if err != nil {
		return nil, fmt.Errorf("query similar patterns: %w", err)
	}

	// Also get top patterns to ensure we find matches even when hash prefix differs
	// This provides broader coverage for semantic similarity
	topPatterns, err := l.store.GetTopPatterns(ctx, limit*3)
	if err != nil {
		return nil, fmt.Errorf("query top patterns: %w", err)
	}

	// Merge pattern lists, avoiding duplicates
	seen := make(map[string]bool)
	var allPatterns []*learning.SuccessfulPattern
	for _, p := range dbPatterns {
		if !seen[p.TaskHash] {
			seen[p.TaskHash] = true
			allPatterns = append(allPatterns, p)
		}
	}
	for _, p := range topPatterns {
		if !seen[p.TaskHash] {
			seen[p.TaskHash] = true
			allPatterns = append(allPatterns, p)
		}
	}

	// Calculate semantic similarity for all patterns using batched ClaudeSimilarity
	results := make([]StoredPattern, 0, len(allPatterns))
	threshold := l.config.SimilarityThreshold
	if threshold <= 0 {
		threshold = 0.3 // Default threshold
	}

	// Batch similarity scoring - single Claude call for all candidates
	var scores []float64
	if sim != nil && len(allPatterns) > 0 {
		candidates := make([]string, len(allPatterns))
		for i, p := range allPatterns {
			candidates[i] = p.PatternDescription
		}
		var err error
		scores, err = sim.CompareBatch(ctx, description, candidates)
		if err != nil {
			// Graceful degradation: continue without similarity scores
			scores = make([]float64, len(allPatterns))
		}
	} else {
		scores = make([]float64, len(allPatterns))
	}

	// Filter patterns above threshold and fetch reasoning for matches
	for i, p := range allPatterns {
		simScore := scores[i]

		if simScore >= threshold {
			var reasoning string
			if sim != nil {
				if result, err := sim.Compare(ctx, description, p.PatternDescription); err == nil && result != nil {
					reasoning = result.Reasoning
				}
			}

			fmt.Fprintf(os.Stderr, "[PATTERN MATCH] Score: %.2f (threshold: %.2f)\n  Pattern: %q\n  Reasoning: %s\n",
				simScore, threshold, truncateStr(p.PatternDescription, 80), reasoning)

			var metadata map[string]interface{}
			if p.Metadata != "" {
				json.Unmarshal([]byte(p.Metadata), &metadata)
			}

			results = append(results, StoredPattern{
				TaskHash:     p.TaskHash,
				Description:  p.PatternDescription,
				LastAgent:    p.LastAgent,
				SuccessCount: p.SuccessCount,
				LastUsed:     p.LastUsed,
				CreatedAt:    p.CreatedAt,
				Similarity:   simScore,
				Metadata:     metadata,
			})
		}
	}

	// Sort by similarity descending
	sortPatternsBySimilarity(results)

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// IncrementSuccess updates the success count for a pattern when a task succeeds again.
// Uses the exact hash to find and update the pattern.
func (l *PatternLibrary) IncrementSuccess(ctx context.Context, description string, files []string, agent string) error {
	if l.store == nil {
		return nil // Graceful no-op when no store
	}

	// Calculate hash
	hashResult := l.hasher.Hash(description, files)

	// Check if pattern exists
	existing, err := l.store.GetPattern(ctx, hashResult.FullHash)
	if err != nil {
		return fmt.Errorf("get pattern: %w", err)
	}

	if existing == nil {
		// Pattern doesn't exist, store it as new
		return l.Store(ctx, description, files, agent)
	}

	// Pattern exists - AddPattern handles increment
	// Update with current agent
	metadata := map[string]interface{}{
		"files": files,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	pattern := &learning.SuccessfulPattern{
		TaskHash:           hashResult.FullHash,
		PatternDescription: description,
		LastAgent:          agent,
		Metadata:           string(metadataJSON),
	}

	if err := l.store.AddPattern(ctx, pattern); err != nil {
		return fmt.Errorf("increment pattern: %w", err)
	}

	return nil
}

// GetTopPatterns returns the most successful patterns ordered by success count.
// These can be used for agent recommendations.
func (l *PatternLibrary) GetTopPatterns(ctx context.Context, limit int) ([]StoredPattern, error) {
	if l.store == nil {
		return []StoredPattern{}, nil // Graceful fallback
	}

	if limit <= 0 {
		limit = 10
	}

	dbPatterns, err := l.store.GetTopPatterns(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get top patterns: %w", err)
	}

	results := make([]StoredPattern, 0, len(dbPatterns))
	for _, p := range dbPatterns {
		var metadata map[string]interface{}
		if p.Metadata != "" {
			json.Unmarshal([]byte(p.Metadata), &metadata)
		}

		results = append(results, StoredPattern{
			TaskHash:     p.TaskHash,
			Description:  p.PatternDescription,
			LastAgent:    p.LastAgent,
			SuccessCount: p.SuccessCount,
			LastUsed:     p.LastUsed,
			CreatedAt:    p.CreatedAt,
			Metadata:     metadata,
		})
	}

	return results, nil
}

// RecommendAgent analyzes similar patterns and recommends the best agent.
// Returns the agent that has the highest success count on similar patterns.
func (l *PatternLibrary) RecommendAgent(ctx context.Context, description string, files []string) (*AgentRecommendation, error) {
	if l.store == nil {
		return nil, nil // No recommendation without store
	}

	// Retrieve similar patterns
	patterns, err := l.Retrieve(ctx, description, files, 10)
	if err != nil {
		return nil, fmt.Errorf("retrieve patterns: %w", err)
	}

	if len(patterns) == 0 {
		return nil, nil // No patterns to base recommendation on
	}

	// Aggregate by agent
	agentStats := make(map[string]*AgentRecommendation)
	for _, p := range patterns {
		if p.LastAgent == "" {
			continue
		}

		if _, exists := agentStats[p.LastAgent]; !exists {
			agentStats[p.LastAgent] = &AgentRecommendation{
				Agent:            p.LastAgent,
				MatchingPatterns: []StoredPattern{},
			}
		}

		stats := agentStats[p.LastAgent]
		stats.SuccessCount += p.SuccessCount
		stats.MatchingPatterns = append(stats.MatchingPatterns, p)
	}

	if len(agentStats) == 0 {
		return nil, nil // No agents found
	}

	// Find best agent
	var best *AgentRecommendation
	for _, stats := range agentStats {
		if best == nil || stats.SuccessCount > best.SuccessCount {
			best = stats
		}
	}

	// Calculate confidence based on number of matching patterns and success counts
	if best != nil {
		// Confidence factors:
		// 1. Number of matching patterns (more patterns = higher confidence)
		// 2. Average similarity of patterns
		// 3. Total success count

		patternCount := float64(len(best.MatchingPatterns))
		avgSimilarity := 0.0
		for _, p := range best.MatchingPatterns {
			avgSimilarity += p.Similarity
		}
		if patternCount > 0 {
			avgSimilarity /= patternCount
		}

		// Normalize confidence to 0-1 range
		// More patterns = higher confidence (cap at 5 patterns for max)
		patternFactor := patternCount / 5.0
		if patternFactor > 1.0 {
			patternFactor = 1.0
		}

		// Combine factors
		best.Confidence = (patternFactor*0.4 + avgSimilarity*0.6)
	}

	return best, nil
}

// GetExactMatch retrieves a pattern by exact task hash.
func (l *PatternLibrary) GetExactMatch(ctx context.Context, description string, files []string) (*StoredPattern, error) {
	if l.store == nil {
		return nil, nil // No match without store
	}

	hashResult := l.hasher.Hash(description, files)
	dbPattern, err := l.store.GetPattern(ctx, hashResult.FullHash)
	if err != nil {
		return nil, fmt.Errorf("get pattern: %w", err)
	}

	if dbPattern == nil {
		return nil, nil // No match
	}

	var metadata map[string]interface{}
	if dbPattern.Metadata != "" {
		json.Unmarshal([]byte(dbPattern.Metadata), &metadata)
	}

	return &StoredPattern{
		TaskHash:     dbPattern.TaskHash,
		Description:  dbPattern.PatternDescription,
		LastAgent:    dbPattern.LastAgent,
		SuccessCount: dbPattern.SuccessCount,
		LastUsed:     dbPattern.LastUsed,
		CreatedAt:    dbPattern.CreatedAt,
		Similarity:   1.0, // Exact match
		Metadata:     metadata,
	}, nil
}

// sortPatternsBySimilarity sorts patterns by similarity in descending order.
func sortPatternsBySimilarity(patterns []StoredPattern) {
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Similarity > patterns[i].Similarity {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
