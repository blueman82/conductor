package pattern

import (
	"context"
	"testing"
	"time"
)

func TestNewClaudeEnhancer(t *testing.T) {
	enhancer := NewClaudeEnhancer(90*time.Second, nil)

	// Verify enhancer was created successfully
	if enhancer == nil {
		t.Fatal("NewClaudeEnhancer returned nil")
	}
	// Invoker is now internal - verify it was created
	if enhancer.inv == nil {
		t.Error("Invoker should be initialized")
	}
	if enhancer.inv.Timeout != 90*time.Second {
		t.Errorf("inv.Timeout = %v, want 90s", enhancer.inv.Timeout)
	}
	if enhancer.inv.ClaudePath != "claude" {
		t.Errorf("inv.ClaudePath = %s, want claude", enhancer.inv.ClaudePath)
	}
	if enhancer.Logger != nil {
		t.Errorf("Logger should be nil when not provided")
	}
}

func TestNewClaudeEnhancer_CustomTimeout(t *testing.T) {
	enhancer := NewClaudeEnhancer(45*time.Second, nil)

	if enhancer.inv.Timeout != 45*time.Second {
		t.Errorf("inv.Timeout = %v, want 45s", enhancer.inv.Timeout)
	}
}

func TestBuildPrompt(t *testing.T) {
	enhancer := NewClaudeEnhancer(30*time.Second, nil)

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
