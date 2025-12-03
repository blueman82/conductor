package parser_validation

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/parser"
)

// TestIntegrationCriteria validates that parseIntegrationCriteria works correctly
// by parsing markdown content with **Integration Criteria**: section and verifying
// the task.IntegrationCriteria contains expected values.
func TestIntegrationCriteria(t *testing.T) {
	tests := []struct {
		name             string
		markdown         string
		expectedCriteria []string
	}{
		{
			name: "basic_integration_criteria",
			markdown: `## Task 1: Test Task

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Integration Criteria**:
- Component A calls Component B correctly
- Error propagates end-to-end

Some other content here
`,
			expectedCriteria: []string{
				"Component A calls Component B correctly",
				"Error propagates end-to-end",
			},
		},
		{
			name: "multiple_integration_criteria",
			markdown: `## Task 1: Integration Test Task

**File(s):** ` + "`integration.go`" + `
**Depends on:** Task 2, Task 3
**Estimated time:** 1h
**Type:** integration

**Integration Criteria**:
- Auth middleware executes before handlers
- Database transaction commits atomically
- Error propagates end-to-end
- Cache invalidation triggers correctly

**Success Criteria**:
- All tests pass
`,
			expectedCriteria: []string{
				"Auth middleware executes before handlers",
				"Database transaction commits atomically",
				"Error propagates end-to-end",
				"Cache invalidation triggers correctly",
			},
		},
		{
			name: "no_integration_criteria",
			markdown: `## Task 1: Regular Task

**File(s):** ` + "`regular.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Success Criteria**:
- Task completes successfully

Just a regular task without integration criteria.
`,
			expectedCriteria: []string{},
		},
		{
			name: "integration_criteria_with_continuation_lines",
			markdown: `## Task 1: Complex Integration Task

**File(s):** ` + "`complex.go`" + `
**Depends on:** None
**Estimated time:** 45m

**Integration Criteria**:
- First criterion with
  continuation on next line
- Second criterion standalone

**Success Criteria**:
- Tests pass
`,
			expectedCriteria: []string{
				"First criterion with continuation on next line",
				"Second criterion standalone",
			},
		},
		{
			name: "integration_criteria_before_next_section",
			markdown: `## Task 1: Ordered Sections Task

**File(s):** ` + "`ordered.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Integration Criteria**:
- Cross-component validation works
- Service calls are properly chained

**Agent:** integration-specialist

More content below
`,
			expectedCriteria: []string{
				"Cross-component validation works",
				"Service calls are properly chained",
			},
		},
		{
			name: "empty_integration_criteria_section",
			markdown: `## Task 1: Empty Integration Task

**File(s):** ` + "`empty.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Integration Criteria**:

**Success Criteria**:
- Task completes
`,
			expectedCriteria: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewMarkdownParser()
			plan, err := p.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if len(plan.Tasks) != 1 {
				t.Fatalf("Expected 1 task, got %d", len(plan.Tasks))
			}

			task := plan.Tasks[0]

			// Verify IntegrationCriteria length
			if len(task.IntegrationCriteria) != len(tt.expectedCriteria) {
				t.Errorf("Expected %d integration criteria, got %d",
					len(tt.expectedCriteria), len(task.IntegrationCriteria))
				t.Errorf("Got criteria: %v", task.IntegrationCriteria)
				return
			}

			// Verify each criterion matches
			for i, expected := range tt.expectedCriteria {
				if i >= len(task.IntegrationCriteria) {
					t.Errorf("Missing criterion at index %d: expected %q", i, expected)
					continue
				}
				if task.IntegrationCriteria[i] != expected {
					t.Errorf("Criterion %d mismatch:\n  expected: %q\n  got:      %q",
						i, expected, task.IntegrationCriteria[i])
				}
			}
		})
	}
}

// TestIntegrationCriteriaWithSuccessCriteria verifies both criteria types
// can be parsed from the same task
func TestIntegrationCriteriaWithSuccessCriteria(t *testing.T) {
	markdown := `## Task 1: Dual Criteria Task

**File(s):** ` + "`dual.go`" + `
**Depends on:** None
**Estimated time:** 1h
**Type:** integration

**Success Criteria**:
- Router accepts requests
- Auth function correct signature

**Integration Criteria**:
- Auth middleware executes before handlers
- Database transaction commits atomically
- Error propagates end-to-end
`

	p := parser.NewMarkdownParser()
	plan, err := p.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(plan.Tasks))
	}

	task := plan.Tasks[0]

	// Verify Success Criteria
	expectedSuccessCriteria := []string{
		"Router accepts requests",
		"Auth function correct signature",
	}
	if len(task.SuccessCriteria) != len(expectedSuccessCriteria) {
		t.Errorf("Expected %d success criteria, got %d: %v",
			len(expectedSuccessCriteria), len(task.SuccessCriteria), task.SuccessCriteria)
	}
	for i, expected := range expectedSuccessCriteria {
		if i >= len(task.SuccessCriteria) {
			break
		}
		if task.SuccessCriteria[i] != expected {
			t.Errorf("Success criterion %d mismatch: expected %q, got %q",
				i, expected, task.SuccessCriteria[i])
		}
	}

	// Verify Integration Criteria
	expectedIntegrationCriteria := []string{
		"Auth middleware executes before handlers",
		"Database transaction commits atomically",
		"Error propagates end-to-end",
	}
	if len(task.IntegrationCriteria) != len(expectedIntegrationCriteria) {
		t.Errorf("Expected %d integration criteria, got %d: %v",
			len(expectedIntegrationCriteria), len(task.IntegrationCriteria), task.IntegrationCriteria)
	}
	for i, expected := range expectedIntegrationCriteria {
		if i >= len(task.IntegrationCriteria) {
			break
		}
		if task.IntegrationCriteria[i] != expected {
			t.Errorf("Integration criterion %d mismatch: expected %q, got %q",
				i, expected, task.IntegrationCriteria[i])
		}
	}
}
