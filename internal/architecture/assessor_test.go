package architecture

import (
	"testing"
	"time"
)

func TestNewAssessor(t *testing.T) {
	timeout := 90 * time.Second
	a := NewAssessor(timeout, nil)

	if a == nil {
		t.Fatal("expected non-nil assessor")
	}
	if a.Timeout != timeout {
		t.Errorf("expected %v timeout, got %v", timeout, a.Timeout)
	}
	if a.ClaudePath != "claude" {
		t.Errorf("expected claude path, got %s", a.ClaudePath)
	}
}

func TestNewAssessor_CustomTimeout(t *testing.T) {
	timeout := 60 * time.Second
	a := NewAssessor(timeout, nil)

	if a.Timeout != timeout {
		t.Errorf("expected %v timeout, got %v", timeout, a.Timeout)
	}
}

func TestAssessmentResult_HasArchitecturalImpact(t *testing.T) {
	tests := []struct {
		name     string
		result   AssessmentResult
		expected bool
	}{
		{
			name:     "all false",
			result:   AssessmentResult{},
			expected: false,
		},
		{
			name: "core infrastructure true",
			result: AssessmentResult{
				CoreInfrastructure: AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
		{
			name: "reuse concerns true",
			result: AssessmentResult{
				ReuseConcerns: AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
		{
			name: "new abstractions true",
			result: AssessmentResult{
				NewAbstractions: AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
		{
			name: "api contracts true",
			result: AssessmentResult{
				APIContracts: AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
		{
			name: "framework lifecycle true",
			result: AssessmentResult{
				FrameworkLifecycle: AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
		{
			name: "cross cutting concerns true",
			result: AssessmentResult{
				CrossCuttingConcerns: AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
		{
			name: "multiple true",
			result: AssessmentResult{
				CoreInfrastructure: AssessmentQuestion{Answer: true},
				APIContracts:       AssessmentQuestion{Answer: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.HasArchitecturalImpact()
			if got != tt.expected {
				t.Errorf("HasArchitecturalImpact() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAssessmentResult_FlaggedQuestions(t *testing.T) {
	tests := []struct {
		name     string
		result   AssessmentResult
		expected []string
	}{
		{
			name:     "none flagged",
			result:   AssessmentResult{},
			expected: nil,
		},
		{
			name: "one flagged",
			result: AssessmentResult{
				CoreInfrastructure: AssessmentQuestion{Answer: true},
			},
			expected: []string{"Core Infrastructure"},
		},
		{
			name: "multiple flagged",
			result: AssessmentResult{
				CoreInfrastructure:   AssessmentQuestion{Answer: true},
				NewAbstractions:      AssessmentQuestion{Answer: true},
				CrossCuttingConcerns: AssessmentQuestion{Answer: true},
			},
			expected: []string{"Core Infrastructure", "New Abstractions", "Cross-Cutting Concerns"},
		},
		{
			name: "all flagged",
			result: AssessmentResult{
				CoreInfrastructure:   AssessmentQuestion{Answer: true},
				ReuseConcerns:        AssessmentQuestion{Answer: true},
				NewAbstractions:      AssessmentQuestion{Answer: true},
				APIContracts:         AssessmentQuestion{Answer: true},
				FrameworkLifecycle:   AssessmentQuestion{Answer: true},
				CrossCuttingConcerns: AssessmentQuestion{Answer: true},
			},
			expected: []string{
				"Core Infrastructure",
				"Reuse Concerns",
				"New Abstractions",
				"API Contracts",
				"Framework Lifecycle",
				"Cross-Cutting Concerns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.FlaggedQuestions()
			if len(got) != len(tt.expected) {
				t.Errorf("FlaggedQuestions() len = %d, want %d", len(got), len(tt.expected))
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("FlaggedQuestions()[%d] = %s, want %s", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestAssessmentSchema(t *testing.T) {
	schema := AssessmentSchema()

	if schema == "" {
		t.Fatal("expected non-empty schema")
	}
	if !contains(schema, "core_infrastructure") {
		t.Error("schema missing core_infrastructure")
	}
	if !contains(schema, "requires_review") {
		t.Error("schema missing requires_review")
	}
	if !contains(schema, "overall_confidence") {
		t.Error("schema missing overall_confidence")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
