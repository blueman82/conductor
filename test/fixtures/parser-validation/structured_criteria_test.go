package parser_validation

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/parser"
)

// TestStructuredCriteria validates that parseStructuredCriteria works correctly
// by parsing markdown content with **Structured Criteria**: section and verifying
// the task.StructuredCriteria contains expected values including optional verification blocks.
func TestStructuredCriteria(t *testing.T) {
	tests := []struct {
		name                  string
		markdown              string
		expectedCriteriaCount int
		verifyCriteria        func(t *testing.T, criteria []struct {
			Criterion       string
			HasVerification bool
			Command         string
			Expected        string
			Description     string
		})
	}{
		{
			name: "structured_criteria_with_verification",
			markdown: `## Task 1: Test Task

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Structured Criteria**:
1. First criterion
   Verification:
   - Command: go test ./...
   - Expected: PASS
   - Description: All tests pass
2. Second criterion without verification

**Status**: pending
`,
			expectedCriteriaCount: 2,
			verifyCriteria: func(t *testing.T, criteria []struct {
				Criterion       string
				HasVerification bool
				Command         string
				Expected        string
				Description     string
			}) {
				if len(criteria) < 2 {
					return
				}
				// First criterion with verification
				if criteria[0].Criterion != "First criterion" {
					t.Errorf("First criterion text mismatch: got %q", criteria[0].Criterion)
				}
				if !criteria[0].HasVerification {
					t.Errorf("First criterion should have verification block")
				}
				if criteria[0].Command != "go test ./..." {
					t.Errorf("First criterion command mismatch: got %q", criteria[0].Command)
				}
				if criteria[0].Expected != "PASS" {
					t.Errorf("First criterion expected mismatch: got %q", criteria[0].Expected)
				}
				if criteria[0].Description != "All tests pass" {
					t.Errorf("First criterion description mismatch: got %q", criteria[0].Description)
				}

				// Second criterion without verification
				if criteria[1].Criterion != "Second criterion without verification" {
					t.Errorf("Second criterion text mismatch: got %q", criteria[1].Criterion)
				}
				if criteria[1].HasVerification {
					t.Errorf("Second criterion should not have verification block")
				}
			},
		},
		{
			name: "multiple_criteria_with_mixed_verification",
			markdown: `## Task 1: Mixed Verification Task

**File(s):** ` + "`mixed.go`" + `
**Depends on:** None
**Estimated time:** 1h

**Structured Criteria**:
1. Build succeeds
   Verification:
   - Command: go build ./...
   - Expected: exit 0
   - Description: Code compiles without errors
2. Tests pass
   Verification:
   - Command: go test -v ./...
   - Expected: PASS
   - Description: All unit tests pass
3. Documentation complete

**Success Criteria**:
- All tests pass
`,
			expectedCriteriaCount: 3,
			verifyCriteria: func(t *testing.T, criteria []struct {
				Criterion       string
				HasVerification bool
				Command         string
				Expected        string
				Description     string
			}) {
				if len(criteria) < 3 {
					return
				}
				// First criterion with verification
				if criteria[0].Criterion != "Build succeeds" {
					t.Errorf("First criterion text mismatch: got %q", criteria[0].Criterion)
				}
				if !criteria[0].HasVerification {
					t.Errorf("First criterion should have verification block")
				}
				if criteria[0].Command != "go build ./..." {
					t.Errorf("First criterion command mismatch: got %q", criteria[0].Command)
				}

				// Second criterion with verification
				if criteria[1].Criterion != "Tests pass" {
					t.Errorf("Second criterion text mismatch: got %q", criteria[1].Criterion)
				}
				if !criteria[1].HasVerification {
					t.Errorf("Second criterion should have verification block")
				}
				if criteria[1].Command != "go test -v ./..." {
					t.Errorf("Second criterion command mismatch: got %q", criteria[1].Command)
				}

				// Third criterion without verification
				if criteria[2].Criterion != "Documentation complete" {
					t.Errorf("Third criterion text mismatch: got %q", criteria[2].Criterion)
				}
				if criteria[2].HasVerification {
					t.Errorf("Third criterion should not have verification block")
				}
			},
		},
		{
			name: "no_structured_criteria",
			markdown: `## Task 1: Regular Task

**File(s):** ` + "`regular.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Success Criteria**:
- Task completes successfully

Just a regular task without structured criteria.
`,
			expectedCriteriaCount: 0,
		},
		{
			name: "structured_criteria_without_verification",
			markdown: `## Task 1: Simple Structured Task

**File(s):** ` + "`simple.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Structured Criteria**:
1. First simple criterion
2. Second simple criterion
3. Third simple criterion

**Success Criteria**:
- All tests pass
`,
			expectedCriteriaCount: 3,
			verifyCriteria: func(t *testing.T, criteria []struct {
				Criterion       string
				HasVerification bool
				Command         string
				Expected        string
				Description     string
			}) {
				if len(criteria) < 3 {
					return
				}
				for i, expected := range []string{"First simple criterion", "Second simple criterion", "Third simple criterion"} {
					if criteria[i].Criterion != expected {
						t.Errorf("Criterion %d text mismatch: expected %q, got %q", i, expected, criteria[i].Criterion)
					}
					if criteria[i].HasVerification {
						t.Errorf("Criterion %d should not have verification block", i)
					}
				}
			},
		},
		{
			name: "structured_criteria_partial_verification",
			markdown: `## Task 1: Partial Verification Task

**File(s):** ` + "`partial.go`" + `
**Depends on:** None
**Estimated time:** 45m

**Structured Criteria**:
1. Code compiles
   Verification:
   - Command: make build
2. Lint passes
   Verification:
   - Command: golangci-lint run
   - Description: No linting errors

**Success Criteria**:
- All checks pass
`,
			expectedCriteriaCount: 2,
			verifyCriteria: func(t *testing.T, criteria []struct {
				Criterion       string
				HasVerification bool
				Command         string
				Expected        string
				Description     string
			}) {
				if len(criteria) < 2 {
					return
				}
				// First criterion - command only
				if !criteria[0].HasVerification {
					t.Errorf("First criterion should have verification block")
				}
				if criteria[0].Command != "make build" {
					t.Errorf("First criterion command mismatch: got %q", criteria[0].Command)
				}
				if criteria[0].Expected != "" {
					t.Errorf("First criterion expected should be empty, got %q", criteria[0].Expected)
				}
				if criteria[0].Description != "" {
					t.Errorf("First criterion description should be empty, got %q", criteria[0].Description)
				}

				// Second criterion - command and description
				if !criteria[1].HasVerification {
					t.Errorf("Second criterion should have verification block")
				}
				if criteria[1].Command != "golangci-lint run" {
					t.Errorf("Second criterion command mismatch: got %q", criteria[1].Command)
				}
				if criteria[1].Expected != "" {
					t.Errorf("Second criterion expected should be empty, got %q", criteria[1].Expected)
				}
				if criteria[1].Description != "No linting errors" {
					t.Errorf("Second criterion description mismatch: got %q", criteria[1].Description)
				}
			},
		},
		{
			name: "empty_structured_criteria_section",
			markdown: `## Task 1: Empty Structured Task

**File(s):** ` + "`empty.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Structured Criteria**:

**Success Criteria**:
- Task completes
`,
			expectedCriteriaCount: 0,
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

			// Verify StructuredCriteria length
			if len(task.StructuredCriteria) != tt.expectedCriteriaCount {
				t.Errorf("Expected %d structured criteria, got %d",
					tt.expectedCriteriaCount, len(task.StructuredCriteria))
				t.Errorf("Got criteria: %+v", task.StructuredCriteria)
				return
			}

			// Skip verification if no callback or empty criteria expected
			if tt.verifyCriteria == nil || tt.expectedCriteriaCount == 0 {
				return
			}

			// Convert to simple struct for verification
			criteria := make([]struct {
				Criterion       string
				HasVerification bool
				Command         string
				Expected        string
				Description     string
			}, len(task.StructuredCriteria))

			for i, c := range task.StructuredCriteria {
				criteria[i].Criterion = c.Criterion
				criteria[i].HasVerification = c.Verification != nil
				if c.Verification != nil {
					criteria[i].Command = c.Verification.Command
					criteria[i].Expected = c.Verification.Expected
					criteria[i].Description = c.Verification.Description
				}
			}
			tt.verifyCriteria(t, criteria)
		})
	}
}

// TestStructuredCriteriaNumberedListParsing verifies that numbered list parsing works correctly
// for structured criteria items (1., 2., 3., etc.)
func TestStructuredCriteriaNumberedListParsing(t *testing.T) {
	markdown := `## Task 1: Numbered List Task

**File(s):** ` + "`numbered.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Structured Criteria**:
1. First numbered criterion
2. Second numbered criterion
3. Third numbered criterion
4. Fourth numbered criterion
5. Fifth numbered criterion

**Success Criteria**:
- All criteria verified
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
	if len(task.StructuredCriteria) != 5 {
		t.Fatalf("Expected 5 structured criteria, got %d: %+v", len(task.StructuredCriteria), task.StructuredCriteria)
	}

	expectedCriteria := []string{
		"First numbered criterion",
		"Second numbered criterion",
		"Third numbered criterion",
		"Fourth numbered criterion",
		"Fifth numbered criterion",
	}

	for i, expected := range expectedCriteria {
		if task.StructuredCriteria[i].Criterion != expected {
			t.Errorf("Criterion %d mismatch: expected %q, got %q", i, expected, task.StructuredCriteria[i].Criterion)
		}
		if task.StructuredCriteria[i].Verification != nil {
			t.Errorf("Criterion %d should not have verification", i)
		}
	}
}

// TestStructuredCriteriaVerificationBlockParsing verifies that verification blocks
// correctly parse Command, Expected, and Description fields
func TestStructuredCriteriaVerificationBlockParsing(t *testing.T) {
	markdown := `## Task 1: Verification Block Task

**File(s):** ` + "`verify.go`" + `
**Depends on:** None
**Estimated time:** 45m

**Structured Criteria**:
1. Unit tests pass
   Verification:
   - Command: go test ./... -v
   - Expected: PASS
   - Description: All unit tests must pass with verbose output
2. Integration tests pass
   Verification:
   - Command: go test -tags=integration ./...
   - Expected: ok
   - Description: Integration tests validate end-to-end behavior
3. Race detection clean
   Verification:
   - Command: go test -race ./...
   - Expected: PASS
   - Description: No race conditions detected

**Success Criteria**:
- All verification commands succeed
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
	if len(task.StructuredCriteria) != 3 {
		t.Fatalf("Expected 3 structured criteria, got %d: %+v", len(task.StructuredCriteria), task.StructuredCriteria)
	}

	// Verify each criterion's verification block
	expectedVerifications := []struct {
		criterion   string
		command     string
		expected    string
		description string
	}{
		{
			criterion:   "Unit tests pass",
			command:     "go test ./... -v",
			expected:    "PASS",
			description: "All unit tests must pass with verbose output",
		},
		{
			criterion:   "Integration tests pass",
			command:     "go test -tags=integration ./...",
			expected:    "ok",
			description: "Integration tests validate end-to-end behavior",
		},
		{
			criterion:   "Race detection clean",
			command:     "go test -race ./...",
			expected:    "PASS",
			description: "No race conditions detected",
		},
	}

	for i, expected := range expectedVerifications {
		sc := task.StructuredCriteria[i]

		if sc.Criterion != expected.criterion {
			t.Errorf("Criterion %d text mismatch: expected %q, got %q", i, expected.criterion, sc.Criterion)
		}

		if sc.Verification == nil {
			t.Errorf("Criterion %d should have verification block", i)
			continue
		}

		if sc.Verification.Command != expected.command {
			t.Errorf("Criterion %d command mismatch: expected %q, got %q", i, expected.command, sc.Verification.Command)
		}
		if sc.Verification.Expected != expected.expected {
			t.Errorf("Criterion %d expected mismatch: expected %q, got %q", i, expected.expected, sc.Verification.Expected)
		}
		if sc.Verification.Description != expected.description {
			t.Errorf("Criterion %d description mismatch: expected %q, got %q", i, expected.description, sc.Verification.Description)
		}
	}
}

// TestStructuredCriteriaWithOtherSections verifies that Structured Criteria parsing
// works correctly alongside other task sections
func TestStructuredCriteriaWithOtherSections(t *testing.T) {
	markdown := `## Task 1: Complete Task

**File(s):** ` + "`complete.go`" + `
**Depends on:** Task 2
**Estimated time:** 1h
**Agent**: golang-pro
**Type**: integration

**Success Criteria**:
- All tests pass
- Code compiles without errors

**Integration Criteria**:
- Components integrate correctly
- Error handling propagates properly

**Structured Criteria**:
1. Build verification
   Verification:
   - Command: go build ./...
   - Expected: exit 0
   - Description: Code compiles without errors
2. Test verification
   Verification:
   - Command: go test ./...
   - Expected: PASS

**Test Commands**:
- go test ./... -v
- go test -race ./...
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

	// Verify other metadata is parsed correctly alongside StructuredCriteria
	if task.Type != "integration" {
		t.Errorf("Expected Type 'integration', got %q", task.Type)
	}
	if task.Agent != "golang-pro" {
		t.Errorf("Expected Agent 'golang-pro', got %q", task.Agent)
	}
	if len(task.SuccessCriteria) != 2 {
		t.Errorf("Expected 2 success criteria, got %d", len(task.SuccessCriteria))
	}
	if len(task.IntegrationCriteria) != 2 {
		t.Errorf("Expected 2 integration criteria, got %d", len(task.IntegrationCriteria))
	}
	if len(task.TestCommands) != 2 {
		t.Errorf("Expected 2 test commands, got %d", len(task.TestCommands))
	}

	// Verify StructuredCriteria
	if len(task.StructuredCriteria) != 2 {
		t.Fatalf("Expected 2 structured criteria, got %d: %+v", len(task.StructuredCriteria), task.StructuredCriteria)
	}

	// Verify first structured criterion
	if task.StructuredCriteria[0].Criterion != "Build verification" {
		t.Errorf("Expected first criterion 'Build verification', got %q", task.StructuredCriteria[0].Criterion)
	}
	if task.StructuredCriteria[0].Verification == nil {
		t.Errorf("Expected first criterion to have verification block")
	} else {
		if task.StructuredCriteria[0].Verification.Command != "go build ./..." {
			t.Errorf("Expected first criterion command 'go build ./...', got %q", task.StructuredCriteria[0].Verification.Command)
		}
		if task.StructuredCriteria[0].Verification.Expected != "exit 0" {
			t.Errorf("Expected first criterion expected 'exit 0', got %q", task.StructuredCriteria[0].Verification.Expected)
		}
		if task.StructuredCriteria[0].Verification.Description != "Code compiles without errors" {
			t.Errorf("Expected first criterion description 'Code compiles without errors', got %q", task.StructuredCriteria[0].Verification.Description)
		}
	}

	// Verify second structured criterion
	if task.StructuredCriteria[1].Criterion != "Test verification" {
		t.Errorf("Expected second criterion 'Test verification', got %q", task.StructuredCriteria[1].Criterion)
	}
	if task.StructuredCriteria[1].Verification == nil {
		t.Errorf("Expected second criterion to have verification block")
	} else {
		if task.StructuredCriteria[1].Verification.Command != "go test ./..." {
			t.Errorf("Expected second criterion command 'go test ./...', got %q", task.StructuredCriteria[1].Verification.Command)
		}
		if task.StructuredCriteria[1].Verification.Expected != "PASS" {
			t.Errorf("Expected second criterion expected 'PASS', got %q", task.StructuredCriteria[1].Verification.Expected)
		}
	}
}

// TestStructuredCriteriaOptionalVerificationFields verifies that optional fields
// in verification blocks can be omitted
func TestStructuredCriteriaOptionalVerificationFields(t *testing.T) {
	markdown := `## Task 1: Optional Fields Task

**File(s):** ` + "`optional.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Structured Criteria**:
1. Command only
   Verification:
   - Command: echo hello
2. Command and expected
   Verification:
   - Command: go version
   - Expected: go version
3. Command and description
   Verification:
   - Command: make clean
   - Description: Clean build artifacts
4. All fields present
   Verification:
   - Command: go test ./...
   - Expected: PASS
   - Description: All tests pass

**Success Criteria**:
- All commands executed
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
	if len(task.StructuredCriteria) != 4 {
		t.Fatalf("Expected 4 structured criteria, got %d: %+v", len(task.StructuredCriteria), task.StructuredCriteria)
	}

	// Criterion 1: Command only
	sc1 := task.StructuredCriteria[0]
	if sc1.Verification == nil {
		t.Errorf("Criterion 1 should have verification block")
	} else {
		if sc1.Verification.Command != "echo hello" {
			t.Errorf("Criterion 1 command mismatch: got %q", sc1.Verification.Command)
		}
		if sc1.Verification.Expected != "" {
			t.Errorf("Criterion 1 expected should be empty, got %q", sc1.Verification.Expected)
		}
		if sc1.Verification.Description != "" {
			t.Errorf("Criterion 1 description should be empty, got %q", sc1.Verification.Description)
		}
	}

	// Criterion 2: Command and expected
	sc2 := task.StructuredCriteria[1]
	if sc2.Verification == nil {
		t.Errorf("Criterion 2 should have verification block")
	} else {
		if sc2.Verification.Command != "go version" {
			t.Errorf("Criterion 2 command mismatch: got %q", sc2.Verification.Command)
		}
		if sc2.Verification.Expected != "go version" {
			t.Errorf("Criterion 2 expected mismatch: got %q", sc2.Verification.Expected)
		}
		if sc2.Verification.Description != "" {
			t.Errorf("Criterion 2 description should be empty, got %q", sc2.Verification.Description)
		}
	}

	// Criterion 3: Command and description
	sc3 := task.StructuredCriteria[2]
	if sc3.Verification == nil {
		t.Errorf("Criterion 3 should have verification block")
	} else {
		if sc3.Verification.Command != "make clean" {
			t.Errorf("Criterion 3 command mismatch: got %q", sc3.Verification.Command)
		}
		if sc3.Verification.Expected != "" {
			t.Errorf("Criterion 3 expected should be empty, got %q", sc3.Verification.Expected)
		}
		if sc3.Verification.Description != "Clean build artifacts" {
			t.Errorf("Criterion 3 description mismatch: got %q", sc3.Verification.Description)
		}
	}

	// Criterion 4: All fields present
	sc4 := task.StructuredCriteria[3]
	if sc4.Verification == nil {
		t.Errorf("Criterion 4 should have verification block")
	} else {
		if sc4.Verification.Command != "go test ./..." {
			t.Errorf("Criterion 4 command mismatch: got %q", sc4.Verification.Command)
		}
		if sc4.Verification.Expected != "PASS" {
			t.Errorf("Criterion 4 expected mismatch: got %q", sc4.Verification.Expected)
		}
		if sc4.Verification.Description != "All tests pass" {
			t.Errorf("Criterion 4 description mismatch: got %q", sc4.Verification.Description)
		}
	}
}
