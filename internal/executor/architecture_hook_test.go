package executor

import (
	"testing"

	"github.com/harrison/conductor/internal/architecture"
	"github.com/harrison/conductor/internal/config"
)

func TestNewArchitectureCheckpointHook_Disabled(t *testing.T) {
	cfg := &config.ArchitectureConfig{Enabled: false}
	hook := NewArchitectureCheckpointHook(nil, cfg, nil)
	if hook != nil {
		t.Error("expected nil hook when disabled")
	}
}

func TestNewArchitectureCheckpointHook_NilConfig(t *testing.T) {
	hook := NewArchitectureCheckpointHook(nil, nil, nil)
	if hook != nil {
		t.Error("expected nil hook when config is nil")
	}
}

func TestNewArchitectureCheckpointHook_NilAssessor(t *testing.T) {
	cfg := &config.ArchitectureConfig{Enabled: true}
	hook := NewArchitectureCheckpointHook(nil, cfg, nil)
	if hook != nil {
		t.Error("expected nil hook when assessor is nil")
	}
}

func TestArchitectureCheckpointHook_RequireJustification(t *testing.T) {
	tests := []struct {
		name     string
		hook     *ArchitectureCheckpointHook
		expected bool
	}{
		{
			name:     "nil hook",
			hook:     nil,
			expected: false,
		},
		{
			name: "disabled",
			hook: &ArchitectureCheckpointHook{
				config: &config.ArchitectureConfig{RequireJustification: false},
			},
			expected: false,
		},
		{
			name: "enabled",
			hook: &ArchitectureCheckpointHook{
				config: &config.ArchitectureConfig{RequireJustification: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hook.RequireJustification()
			if got != tt.expected {
				t.Errorf("RequireJustification() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestArchitectureCheckpointHook_BuildPromptInjection(t *testing.T) {
	hook := &ArchitectureCheckpointHook{
		config: &config.ArchitectureConfig{
			Enabled:              true,
			RequireJustification: true,
		},
	}

	tests := []struct {
		name       string
		assessment *architecture.AssessmentResult
		wantEmpty  bool
		contains   []string
	}{
		{
			name:       "nil assessment",
			assessment: nil,
			wantEmpty:  true,
		},
		{
			name: "no review required",
			assessment: &architecture.AssessmentResult{
				RequiresReview: false,
			},
			wantEmpty: true,
		},
		{
			name: "review required with flagged questions",
			assessment: &architecture.AssessmentResult{
				RequiresReview: true,
				Summary:        "This task introduces new patterns",
				CoreInfrastructure: architecture.AssessmentQuestion{
					Answer:    true,
					Reasoning: "Touches shared auth service",
				},
				NewAbstractions: architecture.AssessmentQuestion{
					Answer:    true,
					Reasoning: "Introduces new error handling pattern",
				},
			},
			wantEmpty: false,
			contains: []string{
				"ARCHITECTURE CHECKPOINT CONTEXT",
				"This task introduces new patterns",
				"Core Infrastructure",
				"New Abstractions",
				"Touches shared auth service",
				"Introduces new error handling pattern",
				"justify any architectural decisions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hook.buildPromptInjection(tt.assessment)

			if tt.wantEmpty && got != "" {
				t.Errorf("buildPromptInjection() = %q, want empty", got)
				return
			}

			if !tt.wantEmpty && got == "" {
				t.Error("buildPromptInjection() returned empty, want non-empty")
				return
			}

			for _, substr := range tt.contains {
				if !containsSubstring(got, substr) {
					t.Errorf("buildPromptInjection() missing %q", substr)
				}
			}
		})
	}
}

// containsSubstring and containsSubstringHelper are defined in package_guard_test.go
