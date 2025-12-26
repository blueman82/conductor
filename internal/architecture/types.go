package architecture

// AssessmentQuestion represents one of the 6 architecture assessment questions
type AssessmentQuestion struct {
	Answer     bool    `json:"answer"`     // YES = true, NO = false
	Confidence float64 `json:"confidence"` // 0.0-1.0
	Reasoning  string  `json:"reasoning"`
	Examples   string  `json:"examples,omitempty"` // Specific examples from task
}

// AssessmentResult contains the full 6-question assessment from Claude
type AssessmentResult struct {
	// The 6 architecture assessment questions
	CoreInfrastructure   AssessmentQuestion `json:"core_infrastructure"`
	ReuseConcerns        AssessmentQuestion `json:"reuse_concerns"`
	NewAbstractions      AssessmentQuestion `json:"new_abstractions"`
	APIContracts         AssessmentQuestion `json:"api_contracts"`
	FrameworkLifecycle   AssessmentQuestion `json:"framework_lifecycle"`
	CrossCuttingConcerns AssessmentQuestion `json:"cross_cutting_concerns"`

	// Aggregated decision
	RequiresReview    bool    `json:"requires_review"`    // ANY yes = true
	OverallConfidence float64 `json:"overall_confidence"` // Average confidence
	Summary           string  `json:"summary"`            // Brief summary
	SkipJustification string  `json:"skip_justification"` // If all NO, why skip is safe
}

// CheckpointResult is returned by the pre-task architecture hook
type CheckpointResult struct {
	ShouldBlock     bool
	ShouldWarn      bool
	ShouldEscalate  bool
	BlockReason     string
	Assessment      *AssessmentResult
	PromptInjection string // Context to inject into agent prompt
}

// HasArchitecturalImpact returns true if any question answered YES
func (r *AssessmentResult) HasArchitecturalImpact() bool {
	return r.CoreInfrastructure.Answer ||
		r.ReuseConcerns.Answer ||
		r.NewAbstractions.Answer ||
		r.APIContracts.Answer ||
		r.FrameworkLifecycle.Answer ||
		r.CrossCuttingConcerns.Answer
}

// FlaggedQuestions returns the questions that were answered YES
func (r *AssessmentResult) FlaggedQuestions() []string {
	var flagged []string
	if r.CoreInfrastructure.Answer {
		flagged = append(flagged, "Core Infrastructure")
	}
	if r.ReuseConcerns.Answer {
		flagged = append(flagged, "Reuse Concerns")
	}
	if r.NewAbstractions.Answer {
		flagged = append(flagged, "New Abstractions")
	}
	if r.APIContracts.Answer {
		flagged = append(flagged, "API Contracts")
	}
	if r.FrameworkLifecycle.Answer {
		flagged = append(flagged, "Framework Lifecycle")
	}
	if r.CrossCuttingConcerns.Answer {
		flagged = append(flagged, "Cross-Cutting Concerns")
	}
	return flagged
}
