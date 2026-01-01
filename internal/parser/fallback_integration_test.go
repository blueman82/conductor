package parser

import (
	"strings"
	"testing"
)

// TestRetryOnRedFallback_EndToEnd tests the complete fallback chain with parsed plans
func TestRetryOnRedFallback_EndToEnd(t *testing.T) {
	t.Run("YAML plan with explicit retry_on_red", func(t *testing.T) {
		yamlContent := `
conductor:
  quality_control:
    enabled: true
    agents:
      mode: explicit
      explicit_list: ["quality-control"]
    retry_on_red: 5

plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 1
      name: "Test Task"
      estimated_time: "30m"
      description: "Test content"
`
		parser := NewYAMLParser()
		plan, err := parser.Parse(strings.NewReader(yamlContent))
		if err != nil {
			t.Fatalf("Failed to parse YAML: %v", err)
		}

		// Before fallback application
		if plan.QualityControl.RetryOnRed != 5 {
			t.Errorf("Before fallback: Expected RetryOnRed=5, got %d", plan.QualityControl.RetryOnRed)
		}

		// Apply fallback with config value 10
		ApplyRetryOnRedFallback(plan, 10)

		// After fallback: should still be 5 (explicit plan value takes precedence)
		if plan.QualityControl.RetryOnRed != 5 {
			t.Errorf("After fallback: Expected RetryOnRed=5 (plan value), got %d", plan.QualityControl.RetryOnRed)
		}
	})

	t.Run("YAML plan without retry_on_red uses config fallback", func(t *testing.T) {
		yamlContent := `
conductor:
  quality_control:
    enabled: true
    agents:
      mode: explicit
      explicit_list: ["quality-control"]

plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 1
      name: "Test Task"
      estimated_time: "30m"
      description: "Test content"
`
		parser := NewYAMLParser()
		plan, err := parser.Parse(strings.NewReader(yamlContent))
		if err != nil {
			t.Fatalf("Failed to parse YAML: %v", err)
		}

		// Before fallback application
		if plan.QualityControl.RetryOnRed != 0 {
			t.Errorf("Before fallback: Expected RetryOnRed=0, got %d", plan.QualityControl.RetryOnRed)
		}

		// Apply fallback with config value 7
		ApplyRetryOnRedFallback(plan, 7)

		// After fallback: should be 7 (config value)
		if plan.QualityControl.RetryOnRed != 7 {
			t.Errorf("After fallback: Expected RetryOnRed=7 (config value), got %d", plan.QualityControl.RetryOnRed)
		}
	})

	t.Run("YAML plan without retry_on_red and no config uses default", func(t *testing.T) {
		yamlContent := `
conductor:
  quality_control:
    enabled: true
    review_agent: "quality-control"

plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 1
      name: "Test Task"
      estimated_time: "30m"
      description: "Test content"
`
		parser := NewYAMLParser()
		plan, err := parser.Parse(strings.NewReader(yamlContent))
		if err != nil {
			t.Fatalf("Failed to parse YAML: %v", err)
		}

		// Before fallback application
		if plan.QualityControl.RetryOnRed != 0 {
			t.Errorf("Before fallback: Expected RetryOnRed=0, got %d", plan.QualityControl.RetryOnRed)
		}

		// Apply fallback with no config value (0)
		ApplyRetryOnRedFallback(plan, 0)

		// After fallback: should be 2 (default)
		if plan.QualityControl.RetryOnRed != 2 {
			t.Errorf("After fallback: Expected RetryOnRed=2 (default), got %d", plan.QualityControl.RetryOnRed)
		}
	})

	t.Run("Markdown plan with frontmatter retry_on_red", func(t *testing.T) {
		markdownContent := `---
conductor:
  quality_control:
    enabled: true
    review_agent: quality-control
    retry_on_red: 3
---

# Test Plan

## Task 1: Test Task

**File(s)**: test.go
**Depends on**: None
**Estimated time**: 30m

Test content
`
		parser := NewMarkdownParser()
		plan, err := parser.Parse(strings.NewReader(markdownContent))
		if err != nil {
			t.Fatalf("Failed to parse Markdown: %v", err)
		}

		// Before fallback application
		if plan.QualityControl.RetryOnRed != 3 {
			t.Errorf("Before fallback: Expected RetryOnRed=3, got %d", plan.QualityControl.RetryOnRed)
		}

		// Apply fallback with config value 8
		ApplyRetryOnRedFallback(plan, 8)

		// After fallback: should still be 3 (explicit plan value)
		if plan.QualityControl.RetryOnRed != 3 {
			t.Errorf("After fallback: Expected RetryOnRed=3 (plan value), got %d", plan.QualityControl.RetryOnRed)
		}
	})

	t.Run("Markdown plan without retry_on_red uses fallback", func(t *testing.T) {
		markdownContent := `---
conductor:
  quality_control:
    enabled: true
    review_agent: quality-control
---

# Test Plan

## Task 1: Test Task

**File(s)**: test.go
**Depends on**: None
**Estimated time**: 30m

Test content
`
		parser := NewMarkdownParser()
		plan, err := parser.Parse(strings.NewReader(markdownContent))
		if err != nil {
			t.Fatalf("Failed to parse Markdown: %v", err)
		}

		// Before fallback application
		if plan.QualityControl.RetryOnRed != 0 {
			t.Errorf("Before fallback: Expected RetryOnRed=0, got %d", plan.QualityControl.RetryOnRed)
		}

		// Apply fallback with config value 6
		ApplyRetryOnRedFallback(plan, 6)

		// After fallback: should be 6 (config value)
		if plan.QualityControl.RetryOnRed != 6 {
			t.Errorf("After fallback: Expected RetryOnRed=6 (config value), got %d", plan.QualityControl.RetryOnRed)
		}
	})
}
