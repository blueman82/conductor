package pattern

import (
	"context"

	"github.com/harrison/conductor/internal/models"
)

// PatternIntelligence defines the interface for pattern-based task analysis.
// It implements the STOP protocol (Search/Think/Outline/Prove) from CLAWED methodology
// and duplicate detection to prevent redundant work.
type PatternIntelligence interface {
	// CheckTask analyzes a task for patterns and duplicates before execution.
	// Returns STOPResult for methodology analysis and DuplicateResult for similarity detection.
	CheckTask(ctx context.Context, task models.Task) (*STOPResult, *DuplicateResult, error)

	// Initialize performs lazy initialization of the pattern intelligence system.
	// Called automatically on first CheckTask if not already initialized.
	Initialize(ctx context.Context) error

	// IsInitialized returns whether the pattern intelligence system is ready.
	IsInitialized() bool
}

// STOPResult captures STOP protocol analysis results from CLAWED methodology.
// STOP = Search / Think / Outline / Prove
type STOPResult struct {
	// Search: Similar patterns found in codebase
	Search SearchResult `json:"search"`

	// Think: Analysis of task complexity and approach
	Think ThinkResult `json:"think"`

	// Outline: Suggested implementation outline
	Outline OutlineResult `json:"outline"`

	// Prove: Verification strategies for success
	Prove ProveResult `json:"prove"`

	// Confidence indicates how confident the analysis is (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// Recommendations for task execution based on analysis
	Recommendations []string `json:"recommendations"`
}

// SearchResult contains patterns found during the Search phase.
type SearchResult struct {
	// SimilarPatterns lists code patterns similar to the task requirements
	SimilarPatterns []PatternMatch `json:"similar_patterns"`

	// RelatedFiles lists files that may be relevant to this task
	RelatedFiles []string `json:"related_files"`

	// ExistingImplementations lists similar implementations found in codebase
	ExistingImplementations []ImplementationRef `json:"existing_implementations"`

	// SearchConfidence indicates confidence in search results (0.0-1.0)
	SearchConfidence float64 `json:"search_confidence"`
}

// PatternMatch represents a matched code pattern.
type PatternMatch struct {
	// Pattern name or description
	Name string `json:"name"`

	// FilePath where pattern was found
	FilePath string `json:"file_path"`

	// LineRange where pattern exists (e.g., "10-25")
	LineRange string `json:"line_range"`

	// Similarity score (0.0-1.0)
	Similarity float64 `json:"similarity"`

	// Description of why this pattern is relevant
	Description string `json:"description"`
}

// ImplementationRef references an existing implementation.
type ImplementationRef struct {
	// Name of the implementation (function, type, etc.)
	Name string `json:"name"`

	// FilePath where implementation exists
	FilePath string `json:"file_path"`

	// Type of implementation (function, struct, interface, etc.)
	Type string `json:"type"`

	// Relevance score (0.0-1.0)
	Relevance float64 `json:"relevance"`
}

// ThinkResult contains analysis from the Think phase.
type ThinkResult struct {
	// ComplexityScore estimates task complexity (1-10)
	ComplexityScore int `json:"complexity_score"`

	// RiskFactors lists potential risks identified
	RiskFactors []RiskFactor `json:"risk_factors"`

	// ApproachSuggestions lists recommended approaches
	ApproachSuggestions []string `json:"approach_suggestions"`

	// EstimatedEffort provides a rough effort estimate
	EstimatedEffort string `json:"estimated_effort"`

	// Dependencies lists inferred dependencies
	Dependencies []string `json:"dependencies"`
}

// RiskFactor represents an identified risk.
type RiskFactor struct {
	// Name of the risk
	Name string `json:"name"`

	// Severity (low, medium, high)
	Severity string `json:"severity"`

	// Mitigation suggestion
	Mitigation string `json:"mitigation"`
}

// OutlineResult contains the suggested implementation outline.
type OutlineResult struct {
	// Steps lists the suggested implementation steps
	Steps []OutlineStep `json:"steps"`

	// KeyDecisions lists important decisions to make
	KeyDecisions []string `json:"key_decisions"`

	// IntegrationPoints lists where integration is needed
	IntegrationPoints []string `json:"integration_points"`
}

// OutlineStep represents a single implementation step.
type OutlineStep struct {
	// Order of the step (1-based)
	Order int `json:"order"`

	// Description of what to do
	Description string `json:"description"`

	// Files to modify
	Files []string `json:"files"`

	// TestStrategy for this step
	TestStrategy string `json:"test_strategy"`
}

// ProveResult contains verification strategies from the Prove phase.
type ProveResult struct {
	// VerificationSteps lists how to verify success
	VerificationSteps []string `json:"verification_steps"`

	// TestCommands suggests test commands to run
	TestCommands []string `json:"test_commands"`

	// AcceptanceCriteria derived from task analysis
	AcceptanceCriteria []string `json:"acceptance_criteria"`

	// RegressionRisks lists potential regression areas
	RegressionRisks []string `json:"regression_risks"`
}

// DuplicateResult captures duplicate detection analysis.
type DuplicateResult struct {
	// IsDuplicate indicates if this task duplicates existing work
	IsDuplicate bool `json:"is_duplicate"`

	// SimilarityScore indicates how similar to existing work (0.0-1.0)
	SimilarityScore float64 `json:"similarity_score"`

	// DuplicateOf lists tasks this may be a duplicate of
	DuplicateOf []DuplicateRef `json:"duplicate_of"`

	// OverlapAreas lists specific areas of overlap
	OverlapAreas []string `json:"overlap_areas"`

	// ShouldSkip recommends whether to skip this task
	ShouldSkip bool `json:"should_skip"`

	// SkipReason explains why task should be skipped (if applicable)
	SkipReason string `json:"skip_reason"`

	// Confidence in duplicate detection (0.0-1.0)
	Confidence float64 `json:"confidence"`
}

// DuplicateRef references a potentially duplicate task.
type DuplicateRef struct {
	// TaskNumber of the potential duplicate
	TaskNumber string `json:"task_number"`

	// TaskName of the potential duplicate
	TaskName string `json:"task_name"`

	// SourceFile where the duplicate task is defined
	SourceFile string `json:"source_file"`

	// SimilarityScore (0.0-1.0)
	SimilarityScore float64 `json:"similarity_score"`

	// OverlapReason explains the overlap
	OverlapReason string `json:"overlap_reason"`
}

// CheckResult combines STOPResult and DuplicateResult for convenience.
type CheckResult struct {
	// STOP contains STOP protocol analysis
	STOP *STOPResult `json:"stop"`

	// Duplicate contains duplicate detection results
	Duplicate *DuplicateResult `json:"duplicate"`

	// ShouldBlock indicates if task execution should be blocked
	ShouldBlock bool `json:"should_block"`

	// BlockReason explains why task is blocked (if applicable)
	BlockReason string `json:"block_reason"`

	// Suggestions for the executing agent
	Suggestions []string `json:"suggestions"`
}

// NewEmptySTOPResult creates an empty STOPResult with zero values.
func NewEmptySTOPResult() *STOPResult {
	return &STOPResult{
		Search: SearchResult{
			SimilarPatterns:         []PatternMatch{},
			RelatedFiles:            []string{},
			ExistingImplementations: []ImplementationRef{},
		},
		Think: ThinkResult{
			RiskFactors:         []RiskFactor{},
			ApproachSuggestions: []string{},
			Dependencies:         []string{},
		},
		Outline: OutlineResult{
			Steps:             []OutlineStep{},
			KeyDecisions:      []string{},
			IntegrationPoints: []string{},
		},
		Prove: ProveResult{
			VerificationSteps:  []string{},
			TestCommands:       []string{},
			AcceptanceCriteria: []string{},
			RegressionRisks:    []string{},
		},
		Recommendations: []string{},
	}
}

// NewEmptyDuplicateResult creates an empty DuplicateResult with zero values.
func NewEmptyDuplicateResult() *DuplicateResult {
	return &DuplicateResult{
		DuplicateOf:  []DuplicateRef{},
		OverlapAreas: []string{},
	}
}
