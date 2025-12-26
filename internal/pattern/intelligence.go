package pattern

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/harrison/conductor/internal/config"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
)

// Recommendation represents the recommended action based on pattern analysis.
type Recommendation string

const (
	// RecommendProceed means the task should execute normally.
	RecommendProceed Recommendation = "proceed"

	// RecommendWarn means the task should proceed with warnings logged.
	RecommendWarn Recommendation = "warn"

	// RecommendBlock means the task should be blocked from execution.
	RecommendBlock Recommendation = "block"

	// RecommendSuggestAgent means a specific agent is recommended based on patterns.
	RecommendSuggestAgent Recommendation = "suggest_agent"
)

// PatternIntelligenceImpl implements the PatternIntelligence interface.
// It orchestrates the hasher, searcher, and library to provide full STOP protocol analysis.
type PatternIntelligenceImpl struct {
	config   *config.PatternConfig
	hasher   *TaskHasher
	searcher *STOPSearcher
	library  *PatternLibrary
	store    *learning.Store
	enhancer *ClaudeEnhancer // nil if LLM enhancement disabled

	mu          sync.RWMutex
	initialized bool
}

// NewPatternIntelligence creates a new PatternIntelligence orchestrator.
// Returns nil if Pattern Intelligence is disabled or store is nil (graceful degradation).
// Components are lazily initialized on first CheckTask call.
func NewPatternIntelligence(cfg *config.PatternConfig, store *learning.Store) PatternIntelligence {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	// Store can be nil - graceful degradation will be handled in Initialize
	return &PatternIntelligenceImpl{
		config:      cfg,
		store:       store,
		initialized: false,
	}
}

// Initialize performs lazy initialization of the pattern intelligence system.
// Called automatically on first CheckTask if not already initialized.
func (pi *PatternIntelligenceImpl) Initialize(ctx context.Context) error {
	if pi == nil {
		return nil
	}

	pi.mu.Lock()
	defer pi.mu.Unlock()

	if pi.initialized {
		return nil
	}

	// Create hasher (always available, no external dependencies)
	pi.hasher = NewTaskHasher()

	// Create searcher (uses store for history search, nil store is handled gracefully)
	pi.searcher = NewSTOPSearcher(pi.store)

	// Create library (uses store and config)
	pi.library = NewPatternLibrary(pi.store, pi.config)

	pi.initialized = true
	return nil
}

// IsInitialized returns whether the pattern intelligence system is ready.
func (pi *PatternIntelligenceImpl) IsInitialized() bool {
	if pi == nil {
		return false
	}

	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.initialized
}

// CheckTask analyzes a task for patterns and duplicates before execution.
// Returns STOPResult for methodology analysis and DuplicateResult for similarity detection.
// Implements graceful degradation: returns nil results on error, never blocks execution.
func (pi *PatternIntelligenceImpl) CheckTask(ctx context.Context, task models.Task) (*STOPResult, *DuplicateResult, error) {
	// Graceful degradation on nil receiver (follows GuardProtocol pattern)
	if pi == nil {
		return nil, nil, nil
	}

	// Lazy initialization
	if !pi.IsInitialized() {
		if err := pi.Initialize(ctx); err != nil {
			// Graceful degradation: log error but don't fail execution
			return nil, nil, nil
		}
	}

	// If still not initialized after attempt, return empty
	if pi.hasher == nil {
		return nil, nil, nil
	}

	// Build task description for analysis
	description := buildTaskDescription(task)
	files := task.Files

	// 1. Hash the incoming task
	hashResult := pi.hasher.Hash(description, files)

	// 2. Check for duplicates in pattern library
	duplicateResult := pi.checkDuplicates(ctx, description, files, hashResult)

	// 3. Run STOP protocol search (if enabled)
	var stopResult *STOPResult
	if pi.config.EnableSTOP {
		stopResult = pi.runSTOPAnalysis(ctx, description, files, hashResult)
	}

	return stopResult, duplicateResult, nil
}

// checkDuplicates checks for task duplicates using the pattern library.
func (pi *PatternIntelligenceImpl) checkDuplicates(ctx context.Context, description string, files []string, hashResult HashResult) *DuplicateResult {
	if !pi.config.EnableDuplicateDetection {
		return NewEmptyDuplicateResult()
	}

	if pi.library == nil {
		return NewEmptyDuplicateResult()
	}

	result := NewEmptyDuplicateResult()

	// Check for exact match first
	exactMatch, err := pi.library.GetExactMatch(ctx, description, files)
	if err == nil && exactMatch != nil {
		result.IsDuplicate = true
		result.SimilarityScore = 1.0
		result.MatchedTaskID = exactMatch.TaskHash
		result.Confidence = 1.0
		result.DuplicateOf = append(result.DuplicateOf, DuplicateRef{
			TaskNumber:      exactMatch.TaskHash,
			TaskName:        exactMatch.Description,
			SimilarityScore: 1.0,
			OverlapReason:   "Exact hash match",
		})

		// Determine recommendation based on mode
		result.Recommendation = pi.evaluateDuplicateRecommendation(result.SimilarityScore)
		result.ShouldSkip = result.Recommendation == "skip"
		if result.ShouldSkip {
			result.SkipReason = fmt.Sprintf("Exact duplicate of existing pattern (hash: %s)", exactMatch.TaskHash[:8])
		}

		return result
	}

	// Check for similar patterns
	similarPatterns, err := pi.library.Retrieve(ctx, description, files, 5)
	if err != nil || len(similarPatterns) == 0 {
		return result
	}

	// Find highest similarity match
	var highestSimilarity float64
	var bestMatch *StoredPattern
	for i, p := range similarPatterns {
		if p.Similarity > highestSimilarity {
			highestSimilarity = p.Similarity
			bestMatch = &similarPatterns[i]
		}

		// Add to duplicate references if above threshold
		if p.Similarity >= pi.config.SimilarityThreshold {
			result.DuplicateOf = append(result.DuplicateOf, DuplicateRef{
				TaskNumber:      p.TaskHash,
				TaskName:        p.Description,
				SimilarityScore: p.Similarity,
				OverlapReason:   "High keyword similarity",
			})
		}
	}

	// Set duplicate status based on threshold
	if highestSimilarity >= pi.config.DuplicateThreshold {
		result.IsDuplicate = true
		result.SimilarityScore = highestSimilarity
		if bestMatch != nil {
			result.MatchedTaskID = bestMatch.TaskHash
		}
		result.Confidence = highestSimilarity // Use similarity as confidence proxy
		result.Recommendation = pi.evaluateDuplicateRecommendation(highestSimilarity)
		result.ShouldSkip = result.Recommendation == "skip"
		if result.ShouldSkip {
			result.SkipReason = fmt.Sprintf("High similarity (%.0f%%) to existing pattern", highestSimilarity*100)
		}
	} else if highestSimilarity >= pi.config.SimilarityThreshold {
		// Similar but not duplicate - add to overlap areas
		result.SimilarityScore = highestSimilarity
		result.OverlapAreas = append(result.OverlapAreas, "Partial keyword overlap with existing patterns")
		result.Recommendation = "proceed"
	}

	return result
}

// evaluateDuplicateRecommendation determines the recommendation based on similarity and mode.
func (pi *PatternIntelligenceImpl) evaluateDuplicateRecommendation(similarity float64) string {
	switch config.PatternMode(pi.config.Mode) {
	case config.PatternModeBlock:
		if similarity >= pi.config.DuplicateThreshold {
			return "skip"
		}
		return "proceed"
	case config.PatternModeWarn:
		if similarity >= pi.config.DuplicateThreshold {
			return "review"
		}
		return "proceed"
	case config.PatternModeSuggest:
		return "proceed"
	default:
		return "proceed"
	}
}

// runSTOPAnalysis performs the STOP protocol analysis (Search/Think/Outline/Prove).
func (pi *PatternIntelligenceImpl) runSTOPAnalysis(ctx context.Context, description string, files []string, hashResult HashResult) *STOPResult {
	result := NewEmptySTOPResult()

	if pi.searcher == nil {
		return result
	}

	// Run parallel searches across git, issues, docs, and history
	searchResults := pi.searcher.Search(ctx, description, files)

	// Convert search results to STOPResult.Search
	result.Search = searchResults.ToSearchResult()

	// Populate Think phase based on search results
	result.Think = pi.generateThinkResult(searchResults, hashResult)

	// Populate Outline phase with suggested steps
	result.Outline = pi.generateOutlineResult(description, files, searchResults)

	// Populate Prove phase with verification strategies
	result.Prove = pi.generateProveResult(description, files)

	// Calculate overall confidence
	result.Confidence = pi.calculateConfidence(searchResults)

	// Generate recommendations
	result.Recommendations = pi.generateRecommendations(searchResults, hashResult)

	return result
}

// generateThinkResult creates the Think phase analysis.
func (pi *PatternIntelligenceImpl) generateThinkResult(searchResults SearchResults, hashResult HashResult) ThinkResult {
	result := ThinkResult{
		RiskFactors:         []RiskFactor{},
		ApproachSuggestions: []string{},
		Dependencies:        []string{},
	}

	// Estimate complexity based on search results
	complexity := 3 // Base complexity
	if len(searchResults.HistoryMatches) > 0 {
		// Previous patterns found - can inform approach
		complexity = 4
		result.ApproachSuggestions = append(result.ApproachSuggestions,
			"Similar patterns found in execution history - review previous approaches")
	}
	if len(searchResults.GitMatches) > 0 {
		complexity++
		result.ApproachSuggestions = append(result.ApproachSuggestions,
			"Related commits found - check for relevant implementation patterns")
	}
	if len(searchResults.DocMatches) > 0 {
		result.ApproachSuggestions = append(result.ApproachSuggestions,
			"Documentation found - review for requirements and constraints")
	}

	result.ComplexityScore = min(complexity, 10)

	// Identify risks based on search results
	if len(searchResults.Errors) > 0 {
		result.RiskFactors = append(result.RiskFactors, RiskFactor{
			Name:       "Search Incomplete",
			Severity:   "low",
			Mitigation: "Some search sources unavailable - proceed with available information",
		})
	}

	// Extract dependencies from history matches
	for _, match := range searchResults.HistoryMatches {
		if match.LastAgent != "" {
			result.Dependencies = append(result.Dependencies,
				fmt.Sprintf("Previous: %s agent used successfully", match.LastAgent))
		}
	}

	// Estimate effort
	switch {
	case complexity <= 3:
		result.EstimatedEffort = "Low"
	case complexity <= 6:
		result.EstimatedEffort = "Medium"
	default:
		result.EstimatedEffort = "High"
	}

	return result
}

// generateOutlineResult creates the Outline phase with suggested implementation steps.
func (pi *PatternIntelligenceImpl) generateOutlineResult(description string, files []string, searchResults SearchResults) OutlineResult {
	result := OutlineResult{
		Steps:             []OutlineStep{},
		KeyDecisions:      []string{},
		IntegrationPoints: []string{},
	}

	stepNum := 1

	// Add step to review existing implementations if found
	if len(searchResults.HistoryMatches) > 0 || len(searchResults.GitMatches) > 0 {
		result.Steps = append(result.Steps, OutlineStep{
			Order:        stepNum,
			Description:  "Review existing implementations and patterns",
			Files:        []string{},
			TestStrategy: "Verify understanding of existing patterns",
		})
		stepNum++
	}

	// Add step for implementation
	result.Steps = append(result.Steps, OutlineStep{
		Order:        stepNum,
		Description:  "Implement the task requirements",
		Files:        files,
		TestStrategy: "Run test commands to verify implementation",
	})
	stepNum++

	// Add integration step if multiple files
	if len(files) > 1 {
		result.Steps = append(result.Steps, OutlineStep{
			Order:        stepNum,
			Description:  "Verify integration between modified files",
			Files:        files,
			TestStrategy: "Run integration tests",
		})
		result.IntegrationPoints = append(result.IntegrationPoints,
			fmt.Sprintf("Cross-file integration (%d files)", len(files)))
	}

	return result
}

// generateProveResult creates the Prove phase with verification strategies.
func (pi *PatternIntelligenceImpl) generateProveResult(description string, files []string) ProveResult {
	result := ProveResult{
		VerificationSteps:  []string{},
		TestCommands:       []string{},
		AcceptanceCriteria: []string{},
		RegressionRisks:    []string{},
	}

	// Standard verification steps
	result.VerificationSteps = append(result.VerificationSteps,
		"Verify all modified files compile without errors",
		"Run relevant test suite",
		"Review changes against task requirements",
	)

	// Suggest test commands based on files
	for _, file := range files {
		if strings.Contains(file, "_test.go") || strings.Contains(file, "_test.") {
			result.TestCommands = append(result.TestCommands,
				fmt.Sprintf("go test -v -run ... %s", file))
		}
	}

	if len(result.TestCommands) == 0 {
		result.TestCommands = append(result.TestCommands, "go test ./...")
	}

	// Standard regression risks
	result.RegressionRisks = append(result.RegressionRisks,
		"Changes may affect existing functionality - review test coverage")

	return result
}

// calculateConfidence calculates overall analysis confidence.
func (pi *PatternIntelligenceImpl) calculateConfidence(searchResults SearchResults) float64 {
	if !searchResults.HasRelevantResults() {
		return 0.3 // Low confidence without supporting data
	}

	confidence := 0.5 // Base confidence

	// Increase confidence with more results
	if len(searchResults.GitMatches) > 0 {
		confidence += 0.1
	}
	if len(searchResults.DocMatches) > 0 {
		confidence += 0.1
	}
	if len(searchResults.HistoryMatches) > 0 {
		confidence += 0.2 // History provides strongest signal
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateRecommendations generates actionable recommendations.
func (pi *PatternIntelligenceImpl) generateRecommendations(searchResults SearchResults, hashResult HashResult) []string {
	recommendations := []string{}

	// Recommend reviewing history matches
	for _, match := range searchResults.HistoryMatches {
		if match.Similarity >= 0.7 {
			recommendations = append(recommendations,
				fmt.Sprintf("Review pattern '%s' (%.0f%% similar, %d successes with %s)",
					truncate(match.PatternDescription, 50),
					match.Similarity*100,
					match.SuccessCount,
					match.LastAgent))
		}
	}

	// Recommend checking related commits
	if len(searchResults.GitMatches) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Check %d related commits for implementation patterns", len(searchResults.GitMatches)))
	}

	// Recommend checking documentation
	if len(searchResults.DocMatches) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review %d documentation matches for requirements", len(searchResults.DocMatches)))
	}

	// Recommend agent if pattern history suggests one
	if len(searchResults.HistoryMatches) > 0 {
		agentCounts := make(map[string]int)
		for _, match := range searchResults.HistoryMatches {
			if match.LastAgent != "" {
				agentCounts[match.LastAgent] += match.SuccessCount
			}
		}
		var bestAgent string
		var bestCount int
		for agent, count := range agentCounts {
			if count > bestCount {
				bestAgent = agent
				bestCount = count
			}
		}
		if bestAgent != "" && bestCount >= 2 {
			recommendations = append(recommendations,
				fmt.Sprintf("Consider using %s agent (%d successful executions on similar tasks)", bestAgent, bestCount))
		}
	}

	return recommendations
}

// buildTaskDescription creates a searchable description from task metadata.
func buildTaskDescription(task models.Task) string {
	parts := []string{task.Name}

	// Add success criteria for more context
	for _, criteria := range task.SuccessCriteria {
		parts = append(parts, criteria)
	}

	return strings.Join(parts, " ")
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetCheckResult combines STOP and Duplicate results into a CheckResult with recommendations.
func GetCheckResult(stop *STOPResult, duplicate *DuplicateResult, cfg *config.PatternConfig) *CheckResult {
	if stop == nil && duplicate == nil {
		return nil
	}

	result := &CheckResult{
		STOP:        stop,
		Duplicate:   duplicate,
		Suggestions: []string{},
	}

	// Determine if task should be blocked
	if duplicate != nil && duplicate.IsDuplicate {
		if cfg != nil && cfg.Mode == config.PatternModeBlock {
			if duplicate.SimilarityScore >= cfg.DuplicateThreshold {
				result.ShouldBlock = true
				result.BlockReason = fmt.Sprintf("Duplicate detected (%.0f%% similarity exceeds %.0f%% threshold)",
					duplicate.SimilarityScore*100, cfg.DuplicateThreshold*100)
			}
		}
	}

	// Add suggestions from STOP analysis
	if stop != nil {
		result.Suggestions = append(result.Suggestions, stop.Recommendations...)
	}

	// Add suggestions from duplicate analysis
	if duplicate != nil && len(duplicate.DuplicateOf) > 0 {
		for _, dup := range duplicate.DuplicateOf {
			result.Suggestions = append(result.Suggestions,
				fmt.Sprintf("Similar to task '%s' (%.0f%% match)", truncate(dup.TaskName, 40), dup.SimilarityScore*100))
		}
	}

	return result
}

// RecordSuccess records a successful task execution in the pattern library.
// This allows future tasks to benefit from learned patterns.
func (pi *PatternIntelligenceImpl) RecordSuccess(ctx context.Context, task models.Task, agent string) error {
	if pi == nil || pi.library == nil {
		return nil // Graceful degradation
	}

	description := buildTaskDescription(task)
	return pi.library.Store(ctx, description, task.Files, agent)
}
