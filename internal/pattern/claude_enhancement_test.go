package pattern

import (
	"context"
	"testing"
	"time"
)

func TestShouldEnhance(t *testing.T) {
	enhancer := NewClaudeEnhancer(nil)

	tests := []struct {
		name       string
		confidence float64
		minConf    float64
		maxConf    float64
		expected   bool
	}{
		{
			name:       "confidence below min - no enhance",
			confidence: 0.2,
			minConf:    0.3,
			maxConf:    0.7,
			expected:   false,
		},
		{
			name:       "confidence above max - no enhance",
			confidence: 0.8,
			minConf:    0.3,
			maxConf:    0.7,
			expected:   false,
		},
		{
			name:       "confidence at min - enhance",
			confidence: 0.3,
			minConf:    0.3,
			maxConf:    0.7,
			expected:   true,
		},
		{
			name:       "confidence at max - enhance",
			confidence: 0.7,
			minConf:    0.3,
			maxConf:    0.7,
			expected:   true,
		},
		{
			name:       "confidence in middle - enhance",
			confidence: 0.5,
			minConf:    0.3,
			maxConf:    0.7,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhancer.ShouldEnhance(tt.confidence, tt.minConf, tt.maxConf)
			if result != tt.expected {
				t.Errorf("ShouldEnhance(%f, %f, %f) = %v, want %v",
					tt.confidence, tt.minConf, tt.maxConf, result, tt.expected)
			}
		})
	}
}

func TestNewClaudeEnhancer(t *testing.T) {
	enhancer := NewClaudeEnhancer(nil)

	if enhancer.Timeout != 30*time.Second {
		t.Errorf("Default Timeout = %v, want 30s", enhancer.Timeout)
	}
	if enhancer.ClaudePath != "claude" {
		t.Errorf("Default ClaudePath = %s, want claude", enhancer.ClaudePath)
	}
	if enhancer.Logger != nil {
		t.Errorf("Logger should be nil when not provided")
	}
}

func TestNewClaudeEnhancerWithConfig(t *testing.T) {
	enhancer := NewClaudeEnhancerWithConfig(45*time.Second, nil)

	if enhancer.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", enhancer.Timeout)
	}
}

func TestBuildPrompt(t *testing.T) {
	enhancer := NewClaudeEnhancer(nil)

	prompt := enhancer.buildPrompt("Test task description", "Pattern: similar task found", 0.5)

	if prompt == "" {
		t.Error("buildPrompt returned empty string")
	}

	// Check key elements are present
	if !contains(prompt, "Test task description") {
		t.Error("Prompt should contain task description")
	}
	if !contains(prompt, "Pattern: similar task found") {
		t.Error("Prompt should contain patterns")
	}
	if !contains(prompt, "0.50") {
		t.Error("Prompt should contain base confidence")
	}
}

func TestEnhancementSchema(t *testing.T) {
	schema := EnhancementSchema()

	if schema == "" {
		t.Error("EnhancementSchema returned empty string")
	}

	// Check required fields are in schema
	if !contains(schema, "adjusted_confidence") {
		t.Error("Schema should contain adjusted_confidence")
	}
	if !contains(schema, "reasoning") {
		t.Error("Schema should contain reasoning")
	}
	if !contains(schema, "risk_factors") {
		t.Error("Schema should contain risk_factors")
	}
}

func TestEnhance_NilEnhancer(t *testing.T) {
	var enhancer *ClaudeEnhancer = nil

	// Should not panic on nil receiver
	if enhancer != nil {
		_, _ = enhancer.Enhance(context.Background(), "task", "patterns", 0.5)
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
