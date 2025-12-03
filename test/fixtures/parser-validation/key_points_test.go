package parser_validation

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/parser"
)

// TestKeyPoints validates that parseKeyPoints works correctly
// by parsing markdown content with **Key Points**: section and verifying
// the task.KeyPoints contains expected values including optional Reference, Impact, Note fields.
func TestKeyPoints(t *testing.T) {
	tests := []struct {
		name               string
		markdown           string
		expectedPointCount int
		verifyPoints       func(t *testing.T, points []struct {
			Point     string
			Reference string
			Details   string
		})
	}{
		{
			name: "key_points_with_all_detail_fields",
			markdown: `## Task 1: Test Task

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:
1. Important design decision
   - Reference: docs/architecture.md
   - Impact: High
   - Note: Affects all downstream components

**Status**: pending
`,
			expectedPointCount: 1,
			verifyPoints: func(t *testing.T, points []struct {
				Point     string
				Reference string
				Details   string
			}) {
				if len(points) < 1 {
					return
				}
				// First key point with all detail fields
				if points[0].Point != "Important design decision" {
					t.Errorf("First point text mismatch: got %q", points[0].Point)
				}
				if points[0].Reference != "docs/architecture.md" {
					t.Errorf("First point reference mismatch: got %q", points[0].Reference)
				}
				// Details should contain combined Impact and Note
				if !strings.Contains(points[0].Details, "Impact: High") {
					t.Errorf("First point details should contain Impact: High, got %q", points[0].Details)
				}
				if !strings.Contains(points[0].Details, "Note: Affects all downstream components") {
					t.Errorf("First point details should contain Note, got %q", points[0].Details)
				}
			},
		},
		{
			name: "multiple_key_points_with_varying_fields",
			markdown: `## Task 1: Multiple Points Task

**File(s):** ` + "`multiple.go`" + `
**Depends on:** None
**Estimated time:** 1h

**Key Points**:
1. First design consideration
   - Reference: docs/design.md
   - Impact: Critical
   - Note: Must review before implementation
2. Second implementation note
   - Reference: internal/core/handler.go
   - Note: Existing pattern to follow
3. Third point without details

**Success Criteria**:
- All tests pass
`,
			expectedPointCount: 3,
			verifyPoints: func(t *testing.T, points []struct {
				Point     string
				Reference string
				Details   string
			}) {
				if len(points) < 3 {
					return
				}
				// First point with all fields
				if points[0].Point != "First design consideration" {
					t.Errorf("First point text mismatch: got %q", points[0].Point)
				}
				if points[0].Reference != "docs/design.md" {
					t.Errorf("First point reference mismatch: got %q", points[0].Reference)
				}
				if !strings.Contains(points[0].Details, "Impact: Critical") {
					t.Errorf("First point details should contain Impact: Critical, got %q", points[0].Details)
				}
				if !strings.Contains(points[0].Details, "Note: Must review before implementation") {
					t.Errorf("First point details should contain Note, got %q", points[0].Details)
				}

				// Second point with reference and note only
				if points[1].Point != "Second implementation note" {
					t.Errorf("Second point text mismatch: got %q", points[1].Point)
				}
				if points[1].Reference != "internal/core/handler.go" {
					t.Errorf("Second point reference mismatch: got %q", points[1].Reference)
				}
				if !strings.Contains(points[1].Details, "Note: Existing pattern to follow") {
					t.Errorf("Second point details should contain Note, got %q", points[1].Details)
				}

				// Third point without any details
				if points[2].Point != "Third point without details" {
					t.Errorf("Third point text mismatch: got %q", points[2].Point)
				}
				if points[2].Reference != "" {
					t.Errorf("Third point should have no reference, got %q", points[2].Reference)
				}
				if points[2].Details != "" {
					t.Errorf("Third point should have no details, got %q", points[2].Details)
				}
			},
		},
		{
			name: "no_key_points_section",
			markdown: `## Task 1: Regular Task

**File(s):** ` + "`regular.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Success Criteria**:
- Task completes successfully

Just a regular task without key points.
`,
			expectedPointCount: 0,
		},
		{
			name: "key_points_reference_only",
			markdown: `## Task 1: Reference Only Task

**File(s):** ` + "`refs.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:
1. Follow existing patterns
   - Reference: internal/parser/markdown.go
2. Use standard library
   - Reference: https://pkg.go.dev/strings

**Success Criteria**:
- All tests pass
`,
			expectedPointCount: 2,
			verifyPoints: func(t *testing.T, points []struct {
				Point     string
				Reference string
				Details   string
			}) {
				if len(points) < 2 {
					return
				}
				// First point with reference only
				if points[0].Point != "Follow existing patterns" {
					t.Errorf("First point text mismatch: got %q", points[0].Point)
				}
				if points[0].Reference != "internal/parser/markdown.go" {
					t.Errorf("First point reference mismatch: got %q", points[0].Reference)
				}
				if points[0].Details != "" {
					t.Errorf("First point should have no details, got %q", points[0].Details)
				}

				// Second point with URL reference
				if points[1].Point != "Use standard library" {
					t.Errorf("Second point text mismatch: got %q", points[1].Point)
				}
				if points[1].Reference != "https://pkg.go.dev/strings" {
					t.Errorf("Second point reference mismatch: got %q", points[1].Reference)
				}
			},
		},
		{
			name: "key_points_impact_only",
			markdown: `## Task 1: Impact Only Task

**File(s):** ` + "`impact.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:
1. Critical performance consideration
   - Impact: High
2. Minor styling preference
   - Impact: Low

**Success Criteria**:
- All tests pass
`,
			expectedPointCount: 2,
			verifyPoints: func(t *testing.T, points []struct {
				Point     string
				Reference string
				Details   string
			}) {
				if len(points) < 2 {
					return
				}
				// First point with high impact
				if points[0].Point != "Critical performance consideration" {
					t.Errorf("First point text mismatch: got %q", points[0].Point)
				}
				if points[0].Reference != "" {
					t.Errorf("First point should have no reference, got %q", points[0].Reference)
				}
				if !strings.Contains(points[0].Details, "Impact: High") {
					t.Errorf("First point details should contain Impact: High, got %q", points[0].Details)
				}

				// Second point with low impact
				if points[1].Point != "Minor styling preference" {
					t.Errorf("Second point text mismatch: got %q", points[1].Point)
				}
				if !strings.Contains(points[1].Details, "Impact: Low") {
					t.Errorf("Second point details should contain Impact: Low, got %q", points[1].Details)
				}
			},
		},
		{
			name: "key_points_note_only",
			markdown: `## Task 1: Note Only Task

**File(s):** ` + "`notes.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:
1. Important note for developers
   - Note: This is a crucial implementation detail
2. Another note
   - Note: Remember to test edge cases

**Success Criteria**:
- All tests pass
`,
			expectedPointCount: 2,
			verifyPoints: func(t *testing.T, points []struct {
				Point     string
				Reference string
				Details   string
			}) {
				if len(points) < 2 {
					return
				}
				// First point with note only
				if points[0].Point != "Important note for developers" {
					t.Errorf("First point text mismatch: got %q", points[0].Point)
				}
				if points[0].Reference != "" {
					t.Errorf("First point should have no reference, got %q", points[0].Reference)
				}
				if !strings.Contains(points[0].Details, "Note: This is a crucial implementation detail") {
					t.Errorf("First point details should contain Note, got %q", points[0].Details)
				}

				// Second point with note only
				if points[1].Point != "Another note" {
					t.Errorf("Second point text mismatch: got %q", points[1].Point)
				}
				if !strings.Contains(points[1].Details, "Note: Remember to test edge cases") {
					t.Errorf("Second point details should contain Note, got %q", points[1].Details)
				}
			},
		},
		{
			name: "empty_key_points_section",
			markdown: `## Task 1: Empty Key Points Task

**File(s):** ` + "`empty.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:

**Success Criteria**:
- Task completes
`,
			expectedPointCount: 0,
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

			// Verify KeyPoints length
			if len(task.KeyPoints) != tt.expectedPointCount {
				t.Errorf("Expected %d key points, got %d",
					tt.expectedPointCount, len(task.KeyPoints))
				t.Errorf("Got key points: %+v", task.KeyPoints)
				return
			}

			// Skip verification if no callback or empty points expected
			if tt.verifyPoints == nil || tt.expectedPointCount == 0 {
				return
			}

			// Convert to simple struct for verification
			points := make([]struct {
				Point     string
				Reference string
				Details   string
			}, len(task.KeyPoints))

			for i, kp := range task.KeyPoints {
				points[i].Point = kp.Point
				points[i].Reference = kp.Reference
				points[i].Details = kp.Details
			}
			tt.verifyPoints(t, points)
		})
	}
}

// TestKeyPointsNumberedListParsing verifies that numbered list parsing works correctly
// for key points items (1., 2., 3., etc.)
func TestKeyPointsNumberedListParsing(t *testing.T) {
	markdown := `## Task 1: Numbered List Task

**File(s):** ` + "`numbered.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:
1. First key point
2. Second key point
3. Third key point
4. Fourth key point
5. Fifth key point

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
	if len(task.KeyPoints) != 5 {
		t.Fatalf("Expected 5 key points, got %d: %+v", len(task.KeyPoints), task.KeyPoints)
	}

	expectedPoints := []string{
		"First key point",
		"Second key point",
		"Third key point",
		"Fourth key point",
		"Fifth key point",
	}

	for i, expected := range expectedPoints {
		if task.KeyPoints[i].Point != expected {
			t.Errorf("Key point %d mismatch: expected %q, got %q", i, expected, task.KeyPoints[i].Point)
		}
		if task.KeyPoints[i].Reference != "" {
			t.Errorf("Key point %d should not have reference", i)
		}
		if task.KeyPoints[i].Details != "" {
			t.Errorf("Key point %d should not have details", i)
		}
	}
}

// TestKeyPointsDetailSubBulletsParsing verifies that detail sub-bullets
// correctly parse Reference, Impact, and Note fields
func TestKeyPointsDetailSubBulletsParsing(t *testing.T) {
	markdown := `## Task 1: Detail Sub-Bullets Task

**File(s):** ` + "`details.go`" + `
**Depends on:** None
**Estimated time:** 45m

**Key Points**:
1. First key point with all fields
   - Reference: docs/architecture.md
   - Impact: High
   - Note: Critical for system stability
2. Second key point with reference and impact
   - Reference: internal/core/handler.go
   - Impact: Medium
3. Third key point with reference only
   - Reference: pkg/utils/helpers.go

**Success Criteria**:
- All details parsed correctly
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
	if len(task.KeyPoints) != 3 {
		t.Fatalf("Expected 3 key points, got %d: %+v", len(task.KeyPoints), task.KeyPoints)
	}

	// Verify each key point's detail sub-bullets
	expectedDetails := []struct {
		point      string
		reference  string
		hasImpact  bool
		impact     string
		hasNote    bool
		noteSubstr string
	}{
		{
			point:      "First key point with all fields",
			reference:  "docs/architecture.md",
			hasImpact:  true,
			impact:     "High",
			hasNote:    true,
			noteSubstr: "Critical for system stability",
		},
		{
			point:     "Second key point with reference and impact",
			reference: "internal/core/handler.go",
			hasImpact: true,
			impact:    "Medium",
			hasNote:   false,
		},
		{
			point:     "Third key point with reference only",
			reference: "pkg/utils/helpers.go",
			hasImpact: false,
			hasNote:   false,
		},
	}

	for i, expected := range expectedDetails {
		kp := task.KeyPoints[i]

		if kp.Point != expected.point {
			t.Errorf("Key point %d text mismatch: expected %q, got %q", i, expected.point, kp.Point)
		}

		if kp.Reference != expected.reference {
			t.Errorf("Key point %d reference mismatch: expected %q, got %q", i, expected.reference, kp.Reference)
		}

		if expected.hasImpact {
			if !strings.Contains(kp.Details, "Impact: "+expected.impact) {
				t.Errorf("Key point %d should have Impact: %s in details, got %q", i, expected.impact, kp.Details)
			}
		}

		if expected.hasNote {
			if !strings.Contains(kp.Details, "Note: "+expected.noteSubstr) {
				t.Errorf("Key point %d should have Note containing %q in details, got %q", i, expected.noteSubstr, kp.Details)
			}
		}
	}
}

// TestKeyPointsWithOtherSections verifies that Key Points parsing
// works correctly alongside other task sections
func TestKeyPointsWithOtherSections(t *testing.T) {
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

**Key Points**:
1. Architecture decision
   - Reference: docs/architecture.md
   - Impact: High
   - Note: Affects all downstream components
2. Implementation pattern
   - Reference: internal/patterns/template.go

**Structured Criteria**:
1. Build verification
   Verification:
   - Command: go build ./...
   - Expected: exit 0
   - Description: Code compiles without errors

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

	// Verify other metadata is parsed correctly alongside KeyPoints
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
	if len(task.StructuredCriteria) != 1 {
		t.Errorf("Expected 1 structured criteria, got %d", len(task.StructuredCriteria))
	}

	// Verify KeyPoints
	if len(task.KeyPoints) != 2 {
		t.Fatalf("Expected 2 key points, got %d: %+v", len(task.KeyPoints), task.KeyPoints)
	}

	// Verify first key point
	if task.KeyPoints[0].Point != "Architecture decision" {
		t.Errorf("Expected first key point 'Architecture decision', got %q", task.KeyPoints[0].Point)
	}
	if task.KeyPoints[0].Reference != "docs/architecture.md" {
		t.Errorf("Expected first key point reference 'docs/architecture.md', got %q", task.KeyPoints[0].Reference)
	}
	if !strings.Contains(task.KeyPoints[0].Details, "Impact: High") {
		t.Errorf("Expected first key point to have Impact: High, got %q", task.KeyPoints[0].Details)
	}
	if !strings.Contains(task.KeyPoints[0].Details, "Note: Affects all downstream components") {
		t.Errorf("Expected first key point to have Note, got %q", task.KeyPoints[0].Details)
	}

	// Verify second key point
	if task.KeyPoints[1].Point != "Implementation pattern" {
		t.Errorf("Expected second key point 'Implementation pattern', got %q", task.KeyPoints[1].Point)
	}
	if task.KeyPoints[1].Reference != "internal/patterns/template.go" {
		t.Errorf("Expected second key point reference 'internal/patterns/template.go', got %q", task.KeyPoints[1].Reference)
	}
	if task.KeyPoints[1].Details != "" {
		t.Errorf("Expected second key point to have no details, got %q", task.KeyPoints[1].Details)
	}
}

// TestKeyPointsOptionalDetailFields verifies that optional fields
// in key point detail sub-bullets can be omitted
func TestKeyPointsOptionalDetailFields(t *testing.T) {
	markdown := `## Task 1: Optional Fields Task

**File(s):** ` + "`optional.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Key Points**:
1. Reference only point
   - Reference: docs/guide.md
2. Impact only point
   - Impact: Medium
3. Note only point
   - Note: Important consideration
4. Reference and impact
   - Reference: internal/handler.go
   - Impact: High
5. Reference and note
   - Reference: pkg/utils.go
   - Note: Follow existing pattern
6. Impact and note
   - Impact: Low
   - Note: Optional enhancement
7. All fields present
   - Reference: docs/full.md
   - Impact: Critical
   - Note: Must complete before release

**Success Criteria**:
- All fields parsed correctly
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
	if len(task.KeyPoints) != 7 {
		t.Fatalf("Expected 7 key points, got %d: %+v", len(task.KeyPoints), task.KeyPoints)
	}

	// Point 1: Reference only
	kp1 := task.KeyPoints[0]
	if kp1.Reference != "docs/guide.md" {
		t.Errorf("Key point 1 reference mismatch: got %q", kp1.Reference)
	}
	if kp1.Details != "" {
		t.Errorf("Key point 1 should have no details, got %q", kp1.Details)
	}

	// Point 2: Impact only
	kp2 := task.KeyPoints[1]
	if kp2.Reference != "" {
		t.Errorf("Key point 2 should have no reference, got %q", kp2.Reference)
	}
	if !strings.Contains(kp2.Details, "Impact: Medium") {
		t.Errorf("Key point 2 should have Impact: Medium, got %q", kp2.Details)
	}

	// Point 3: Note only
	kp3 := task.KeyPoints[2]
	if kp3.Reference != "" {
		t.Errorf("Key point 3 should have no reference, got %q", kp3.Reference)
	}
	if !strings.Contains(kp3.Details, "Note: Important consideration") {
		t.Errorf("Key point 3 should have Note, got %q", kp3.Details)
	}

	// Point 4: Reference and impact
	kp4 := task.KeyPoints[3]
	if kp4.Reference != "internal/handler.go" {
		t.Errorf("Key point 4 reference mismatch: got %q", kp4.Reference)
	}
	if !strings.Contains(kp4.Details, "Impact: High") {
		t.Errorf("Key point 4 should have Impact: High, got %q", kp4.Details)
	}

	// Point 5: Reference and note
	kp5 := task.KeyPoints[4]
	if kp5.Reference != "pkg/utils.go" {
		t.Errorf("Key point 5 reference mismatch: got %q", kp5.Reference)
	}
	if !strings.Contains(kp5.Details, "Note: Follow existing pattern") {
		t.Errorf("Key point 5 should have Note, got %q", kp5.Details)
	}

	// Point 6: Impact and note
	kp6 := task.KeyPoints[5]
	if kp6.Reference != "" {
		t.Errorf("Key point 6 should have no reference, got %q", kp6.Reference)
	}
	if !strings.Contains(kp6.Details, "Impact: Low") {
		t.Errorf("Key point 6 should have Impact: Low, got %q", kp6.Details)
	}
	if !strings.Contains(kp6.Details, "Note: Optional enhancement") {
		t.Errorf("Key point 6 should have Note, got %q", kp6.Details)
	}

	// Point 7: All fields present
	kp7 := task.KeyPoints[6]
	if kp7.Reference != "docs/full.md" {
		t.Errorf("Key point 7 reference mismatch: got %q", kp7.Reference)
	}
	if !strings.Contains(kp7.Details, "Impact: Critical") {
		t.Errorf("Key point 7 should have Impact: Critical, got %q", kp7.Details)
	}
	if !strings.Contains(kp7.Details, "Note: Must complete before release") {
		t.Errorf("Key point 7 should have Note, got %q", kp7.Details)
	}
}
