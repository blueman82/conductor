package pattern

import (
	"testing"
)

func TestNewEmptySTOPResult(t *testing.T) {
	result := NewEmptySTOPResult()

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify Search is initialized
	if result.Search.SimilarPatterns == nil {
		t.Error("expected SimilarPatterns to be initialized")
	}
	if result.Search.RelatedFiles == nil {
		t.Error("expected RelatedFiles to be initialized")
	}
	if result.Search.ExistingImplementations == nil {
		t.Error("expected ExistingImplementations to be initialized")
	}

	// Verify Think is initialized
	if result.Think.RiskFactors == nil {
		t.Error("expected RiskFactors to be initialized")
	}
	if result.Think.ApproachSuggestions == nil {
		t.Error("expected ApproachSuggestions to be initialized")
	}
	if result.Think.Dependencies == nil {
		t.Error("expected Dependencies to be initialized")
	}

	// Verify Outline is initialized
	if result.Outline.Steps == nil {
		t.Error("expected Steps to be initialized")
	}
	if result.Outline.KeyDecisions == nil {
		t.Error("expected KeyDecisions to be initialized")
	}
	if result.Outline.IntegrationPoints == nil {
		t.Error("expected IntegrationPoints to be initialized")
	}

	// Verify Prove is initialized
	if result.Prove.VerificationSteps == nil {
		t.Error("expected VerificationSteps to be initialized")
	}
	if result.Prove.TestCommands == nil {
		t.Error("expected TestCommands to be initialized")
	}
	if result.Prove.AcceptanceCriteria == nil {
		t.Error("expected AcceptanceCriteria to be initialized")
	}
	if result.Prove.RegressionRisks == nil {
		t.Error("expected RegressionRisks to be initialized")
	}

	// Verify Recommendations is initialized
	if result.Recommendations == nil {
		t.Error("expected Recommendations to be initialized")
	}

	// Verify default values
	if result.Confidence != 0 {
		t.Errorf("expected Confidence 0, got %f", result.Confidence)
	}
}

func TestNewEmptyDuplicateResult(t *testing.T) {
	result := NewEmptyDuplicateResult()

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify slices are initialized
	if result.DuplicateOf == nil {
		t.Error("expected DuplicateOf to be initialized")
	}
	if result.OverlapAreas == nil {
		t.Error("expected OverlapAreas to be initialized")
	}

	// Verify default values
	if result.IsDuplicate {
		t.Error("expected IsDuplicate to be false")
	}
	if result.SimilarityScore != 0 {
		t.Errorf("expected SimilarityScore 0, got %f", result.SimilarityScore)
	}
	if result.ShouldSkip {
		t.Error("expected ShouldSkip to be false")
	}
	if result.SkipReason != "" {
		t.Errorf("expected empty SkipReason, got %q", result.SkipReason)
	}
	if result.Confidence != 0 {
		t.Errorf("expected Confidence 0, got %f", result.Confidence)
	}
}

func TestSTOPResult_Components(t *testing.T) {
	result := &STOPResult{
		Search: SearchResult{
			SimilarPatterns: []PatternMatch{
				{
					Name:       "TestPattern",
					FilePath:   "/test/file.go",
					LineRange:  "10-20",
					Similarity: 0.85,
				},
			},
			RelatedFiles: []string{"/test/related.go"},
			ExistingImplementations: []ImplementationRef{
				{
					Name:      "ExistingFunc",
					FilePath:  "/test/existing.go",
					Type:      "function",
					Relevance: 0.9,
				},
			},
			SearchConfidence: 0.8,
		},
		Think: ThinkResult{
			ComplexityScore: 7,
			RiskFactors: []RiskFactor{
				{
					Name:       "Complexity",
					Severity:   "medium",
					Mitigation: "Break into smaller tasks",
				},
			},
			ApproachSuggestions: []string{"Use existing pattern"},
			EstimatedEffort:     "2 hours",
			Dependencies:        []string{"dep1", "dep2"},
		},
		Outline: OutlineResult{
			Steps: []OutlineStep{
				{
					Order:        1,
					Description:  "Step 1",
					Files:        []string{"/test/file.go"},
					TestStrategy: "unit test",
				},
			},
			KeyDecisions:      []string{"Decision 1"},
			IntegrationPoints: []string{"API endpoint"},
		},
		Prove: ProveResult{
			VerificationSteps:  []string{"Run tests"},
			TestCommands:       []string{"go test ./..."},
			AcceptanceCriteria: []string{"All tests pass"},
			RegressionRisks:    []string{"May affect module X"},
		},
		Confidence:      0.85,
		Recommendations: []string{"Follow existing patterns"},
	}

	// Verify Search
	if len(result.Search.SimilarPatterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(result.Search.SimilarPatterns))
	}
	if result.Search.SimilarPatterns[0].Name != "TestPattern" {
		t.Errorf("expected pattern name 'TestPattern', got %q", result.Search.SimilarPatterns[0].Name)
	}

	// Verify Think
	if result.Think.ComplexityScore != 7 {
		t.Errorf("expected complexity 7, got %d", result.Think.ComplexityScore)
	}
	if len(result.Think.RiskFactors) != 1 {
		t.Errorf("expected 1 risk factor, got %d", len(result.Think.RiskFactors))
	}

	// Verify Outline
	if len(result.Outline.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(result.Outline.Steps))
	}
	if result.Outline.Steps[0].Order != 1 {
		t.Errorf("expected step order 1, got %d", result.Outline.Steps[0].Order)
	}

	// Verify Prove
	if len(result.Prove.TestCommands) != 1 {
		t.Errorf("expected 1 test command, got %d", len(result.Prove.TestCommands))
	}

	// Verify top-level
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", result.Confidence)
	}
}

func TestDuplicateResult_WithDuplicate(t *testing.T) {
	result := &DuplicateResult{
		IsDuplicate:     true,
		SimilarityScore: 0.95,
		DuplicateOf: []DuplicateRef{
			{
				TaskNumber:      "5",
				TaskName:        "Similar Task",
				SourceFile:      "plan.yaml",
				SimilarityScore: 0.95,
				OverlapReason:   "Same files modified",
			},
		},
		OverlapAreas: []string{"file1.go", "file2.go"},
		ShouldSkip:   true,
		SkipReason:   "Already implemented in Task 5",
		Confidence:   0.9,
	}

	if !result.IsDuplicate {
		t.Error("expected IsDuplicate to be true")
	}
	if result.SimilarityScore != 0.95 {
		t.Errorf("expected SimilarityScore 0.95, got %f", result.SimilarityScore)
	}
	if len(result.DuplicateOf) != 1 {
		t.Errorf("expected 1 duplicate ref, got %d", len(result.DuplicateOf))
	}
	if result.DuplicateOf[0].TaskNumber != "5" {
		t.Errorf("expected task number '5', got %q", result.DuplicateOf[0].TaskNumber)
	}
	if !result.ShouldSkip {
		t.Error("expected ShouldSkip to be true")
	}
}

func TestCheckResult_Combined(t *testing.T) {
	checkResult := &CheckResult{
		STOP:        NewEmptySTOPResult(),
		Duplicate:   NewEmptyDuplicateResult(),
		ShouldBlock: false,
		BlockReason: "",
		Suggestions: []string{"Suggestion 1", "Suggestion 2"},
	}

	if checkResult.STOP == nil {
		t.Error("expected STOP to be non-nil")
	}
	if checkResult.Duplicate == nil {
		t.Error("expected Duplicate to be non-nil")
	}
	if checkResult.ShouldBlock {
		t.Error("expected ShouldBlock to be false")
	}
	if len(checkResult.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(checkResult.Suggestions))
	}
}
