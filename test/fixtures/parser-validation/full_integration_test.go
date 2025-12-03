package parser_validation

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/parser"
)

// TestFullIntegration validates that the markdown parser correctly parses all 4 new features
// in a single markdown document:
// - Integration Criteria: section
// - Runtime Metadata: with all subsections
// - Structured Criteria: with verification blocks
// - Key Points: with detail sub-bullets
//
// This is a comprehensive integration test proving feature parity across all new v2.9+ features.
func TestFullIntegration(t *testing.T) {
	markdown := `## Task 1: Complete Feature Integration Task

**File(s)**: ` + "`internal/integration/handler.go`, `internal/integration/service.go`" + `
**Depends on**: None
**Estimated time**: 2h
**Agent**: integration-specialist
**Type**: integration

**Success Criteria**:
- Handler correctly processes requests
- Service layer implements business logic
- Error handling works correctly

**Integration Criteria**:
- Handler invokes service methods in correct order
- Error propagates from service to handler to client
- Transaction boundaries are correctly maintained
- Authentication middleware executes before handlers

**Runtime Metadata**:
**Dependency Checks**:
- go version: Verify Go 1.21+ is installed
- go mod tidy: Ensure dependencies are resolved
- docker ps: Check Docker is running

**Documentation Targets**:
- docs/api.md (API Reference)
- README.md (Getting Started)
- docs/architecture.md

**Prompt Blocks**:
- context: This task implements the core integration layer
- instruction: Follow clean architecture principles
- constraint: Must maintain backward compatibility

**Structured Criteria**:
1. Build verification
   Verification:
   - Command: go build ./...
   - Expected: exit 0
   - Description: Code compiles without errors
2. Unit tests pass
   Verification:
   - Command: go test ./internal/integration/...
   - Expected: PASS
   - Description: All unit tests must pass
3. Integration tests pass
   Verification:
   - Command: go test -tags=integration ./...
   - Expected: ok
4. Documentation complete

**Key Points**:
1. Architecture decision for handler-service separation
   - Reference: docs/architecture.md
   - Impact: High
   - Note: Affects all downstream components
2. Error handling pattern
   - Reference: internal/errors/handler.go
   - Note: Follow existing error wrapping pattern
3. Transaction management approach
   - Reference: internal/db/tx.go
   - Impact: Critical
4. Simple implementation note

**Test Commands**:
- go test ./internal/integration/... -v
- go test -race ./internal/integration/...
- go test -tags=integration ./...

**Status:** pending
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

	// ===============================
	// Verify basic task metadata
	// ===============================
	if task.Number != "1" {
		t.Errorf("Expected task number '1', got %q", task.Number)
	}
	if task.Name != "Complete Feature Integration Task" {
		t.Errorf("Expected task name 'Complete Feature Integration Task', got %q", task.Name)
	}
	if task.Agent != "integration-specialist" {
		t.Errorf("Expected agent 'integration-specialist', got %q", task.Agent)
	}
	if task.Type != "integration" {
		t.Errorf("Expected type 'integration', got %q", task.Type)
	}
	if task.Status != "pending" {
		t.Errorf("Expected status 'pending', got %q", task.Status)
	}
	if len(task.Files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(task.Files), task.Files)
	} else {
		if task.Files[0] != "internal/integration/handler.go" {
			t.Errorf("Expected first file 'internal/integration/handler.go', got %q", task.Files[0])
		}
		if task.Files[1] != "internal/integration/service.go" {
			t.Errorf("Expected second file 'internal/integration/service.go', got %q", task.Files[1])
		}
	}

	// ===============================
	// 1. Verify Integration Criteria
	// ===============================
	expectedIntegrationCriteria := []string{
		"Handler invokes service methods in correct order",
		"Error propagates from service to handler to client",
		"Transaction boundaries are correctly maintained",
		"Authentication middleware executes before handlers",
	}
	if len(task.IntegrationCriteria) != len(expectedIntegrationCriteria) {
		t.Errorf("Expected %d integration criteria, got %d: %v",
			len(expectedIntegrationCriteria), len(task.IntegrationCriteria), task.IntegrationCriteria)
	} else {
		for i, expected := range expectedIntegrationCriteria {
			if task.IntegrationCriteria[i] != expected {
				t.Errorf("Integration criterion %d mismatch:\n  expected: %q\n  got:      %q",
					i, expected, task.IntegrationCriteria[i])
			}
		}
	}

	// ===============================
	// 2. Verify Runtime Metadata
	// ===============================
	if task.RuntimeMetadata == nil {
		t.Fatalf("Expected RuntimeMetadata to be non-nil")
	}

	rm := task.RuntimeMetadata

	// Verify Dependency Checks (3 items)
	if len(rm.DependencyChecks) != 3 {
		t.Errorf("Expected 3 dependency checks, got %d: %v", len(rm.DependencyChecks), rm.DependencyChecks)
	} else {
		expectedDependencyChecks := []struct {
			command     string
			description string
		}{
			{"go version", "Verify Go 1.21+ is installed"},
			{"go mod tidy", "Ensure dependencies are resolved"},
			{"docker ps", "Check Docker is running"},
		}
		for i, expected := range expectedDependencyChecks {
			if rm.DependencyChecks[i].Command != expected.command {
				t.Errorf("Dependency check %d command mismatch: expected %q, got %q",
					i, expected.command, rm.DependencyChecks[i].Command)
			}
			if rm.DependencyChecks[i].Description != expected.description {
				t.Errorf("Dependency check %d description mismatch: expected %q, got %q",
					i, expected.description, rm.DependencyChecks[i].Description)
			}
		}
	}

	// Verify Documentation Targets (3 items)
	if len(rm.DocumentationTargets) != 3 {
		t.Errorf("Expected 3 documentation targets, got %d: %v", len(rm.DocumentationTargets), rm.DocumentationTargets)
	} else {
		expectedDocTargets := []struct {
			location string
			section  string
		}{
			{"docs/api.md", "API Reference"},
			{"README.md", "Getting Started"},
			{"docs/architecture.md", ""}, // No section
		}
		for i, expected := range expectedDocTargets {
			if rm.DocumentationTargets[i].Location != expected.location {
				t.Errorf("Doc target %d location mismatch: expected %q, got %q",
					i, expected.location, rm.DocumentationTargets[i].Location)
			}
			if rm.DocumentationTargets[i].Section != expected.section {
				t.Errorf("Doc target %d section mismatch: expected %q, got %q",
					i, expected.section, rm.DocumentationTargets[i].Section)
			}
		}
	}

	// Verify Prompt Blocks (3 items)
	if len(rm.PromptBlocks) != 3 {
		t.Errorf("Expected 3 prompt blocks, got %d: %v", len(rm.PromptBlocks), rm.PromptBlocks)
	} else {
		expectedPromptBlocks := []struct {
			blockType string
			content   string
		}{
			{"context", "This task implements the core integration layer"},
			{"instruction", "Follow clean architecture principles"},
			{"constraint", "Must maintain backward compatibility"},
		}
		for i, expected := range expectedPromptBlocks {
			if rm.PromptBlocks[i].Type != expected.blockType {
				t.Errorf("Prompt block %d type mismatch: expected %q, got %q",
					i, expected.blockType, rm.PromptBlocks[i].Type)
			}
			if rm.PromptBlocks[i].Content != expected.content {
				t.Errorf("Prompt block %d content mismatch: expected %q, got %q",
					i, expected.content, rm.PromptBlocks[i].Content)
			}
		}
	}

	// ===============================
	// 3. Verify Structured Criteria
	// ===============================
	if len(task.StructuredCriteria) != 4 {
		t.Errorf("Expected 4 structured criteria, got %d: %+v", len(task.StructuredCriteria), task.StructuredCriteria)
	} else {
		// Criterion 1: Build verification with full verification block
		sc1 := task.StructuredCriteria[0]
		if sc1.Criterion != "Build verification" {
			t.Errorf("Structured criterion 1 text mismatch: got %q", sc1.Criterion)
		}
		if sc1.Verification == nil {
			t.Errorf("Structured criterion 1 should have verification block")
		} else {
			if sc1.Verification.Command != "go build ./..." {
				t.Errorf("Structured criterion 1 command mismatch: got %q", sc1.Verification.Command)
			}
			if sc1.Verification.Expected != "exit 0" {
				t.Errorf("Structured criterion 1 expected mismatch: got %q", sc1.Verification.Expected)
			}
			if sc1.Verification.Description != "Code compiles without errors" {
				t.Errorf("Structured criterion 1 description mismatch: got %q", sc1.Verification.Description)
			}
		}

		// Criterion 2: Unit tests pass with full verification block
		sc2 := task.StructuredCriteria[1]
		if sc2.Criterion != "Unit tests pass" {
			t.Errorf("Structured criterion 2 text mismatch: got %q", sc2.Criterion)
		}
		if sc2.Verification == nil {
			t.Errorf("Structured criterion 2 should have verification block")
		} else {
			if sc2.Verification.Command != "go test ./internal/integration/..." {
				t.Errorf("Structured criterion 2 command mismatch: got %q", sc2.Verification.Command)
			}
			if sc2.Verification.Expected != "PASS" {
				t.Errorf("Structured criterion 2 expected mismatch: got %q", sc2.Verification.Expected)
			}
			if sc2.Verification.Description != "All unit tests must pass" {
				t.Errorf("Structured criterion 2 description mismatch: got %q", sc2.Verification.Description)
			}
		}

		// Criterion 3: Integration tests pass (partial verification - no description)
		sc3 := task.StructuredCriteria[2]
		if sc3.Criterion != "Integration tests pass" {
			t.Errorf("Structured criterion 3 text mismatch: got %q", sc3.Criterion)
		}
		if sc3.Verification == nil {
			t.Errorf("Structured criterion 3 should have verification block")
		} else {
			if sc3.Verification.Command != "go test -tags=integration ./..." {
				t.Errorf("Structured criterion 3 command mismatch: got %q", sc3.Verification.Command)
			}
			if sc3.Verification.Expected != "ok" {
				t.Errorf("Structured criterion 3 expected mismatch: got %q", sc3.Verification.Expected)
			}
			if sc3.Verification.Description != "" {
				t.Errorf("Structured criterion 3 description should be empty, got %q", sc3.Verification.Description)
			}
		}

		// Criterion 4: Documentation complete (no verification)
		sc4 := task.StructuredCriteria[3]
		if sc4.Criterion != "Documentation complete" {
			t.Errorf("Structured criterion 4 text mismatch: got %q", sc4.Criterion)
		}
		if sc4.Verification != nil {
			t.Errorf("Structured criterion 4 should not have verification block")
		}
	}

	// ===============================
	// 4. Verify Key Points
	// ===============================
	if len(task.KeyPoints) != 4 {
		t.Errorf("Expected 4 key points, got %d: %+v", len(task.KeyPoints), task.KeyPoints)
	} else {
		// Key Point 1: All fields present
		kp1 := task.KeyPoints[0]
		if kp1.Point != "Architecture decision for handler-service separation" {
			t.Errorf("Key point 1 text mismatch: got %q", kp1.Point)
		}
		if kp1.Reference != "docs/architecture.md" {
			t.Errorf("Key point 1 reference mismatch: got %q", kp1.Reference)
		}
		if !strings.Contains(kp1.Details, "Impact: High") {
			t.Errorf("Key point 1 should contain 'Impact: High', got %q", kp1.Details)
		}
		if !strings.Contains(kp1.Details, "Note: Affects all downstream components") {
			t.Errorf("Key point 1 should contain 'Note: Affects all downstream components', got %q", kp1.Details)
		}

		// Key Point 2: Reference and Note (no Impact)
		kp2 := task.KeyPoints[1]
		if kp2.Point != "Error handling pattern" {
			t.Errorf("Key point 2 text mismatch: got %q", kp2.Point)
		}
		if kp2.Reference != "internal/errors/handler.go" {
			t.Errorf("Key point 2 reference mismatch: got %q", kp2.Reference)
		}
		if !strings.Contains(kp2.Details, "Note: Follow existing error wrapping pattern") {
			t.Errorf("Key point 2 should contain 'Note: Follow existing error wrapping pattern', got %q", kp2.Details)
		}
		if strings.Contains(kp2.Details, "Impact") {
			t.Errorf("Key point 2 should not contain Impact, got %q", kp2.Details)
		}

		// Key Point 3: Reference and Impact (no Note)
		kp3 := task.KeyPoints[2]
		if kp3.Point != "Transaction management approach" {
			t.Errorf("Key point 3 text mismatch: got %q", kp3.Point)
		}
		if kp3.Reference != "internal/db/tx.go" {
			t.Errorf("Key point 3 reference mismatch: got %q", kp3.Reference)
		}
		if !strings.Contains(kp3.Details, "Impact: Critical") {
			t.Errorf("Key point 3 should contain 'Impact: Critical', got %q", kp3.Details)
		}
		if strings.Contains(kp3.Details, "Note") {
			t.Errorf("Key point 3 should not contain Note, got %q", kp3.Details)
		}

		// Key Point 4: No sub-bullets
		kp4 := task.KeyPoints[3]
		if kp4.Point != "Simple implementation note" {
			t.Errorf("Key point 4 text mismatch: got %q", kp4.Point)
		}
		if kp4.Reference != "" {
			t.Errorf("Key point 4 should have no reference, got %q", kp4.Reference)
		}
		if kp4.Details != "" {
			t.Errorf("Key point 4 should have no details, got %q", kp4.Details)
		}
	}

	// ===============================
	// Verify Success Criteria (standard section)
	// ===============================
	expectedSuccessCriteria := []string{
		"Handler correctly processes requests",
		"Service layer implements business logic",
		"Error handling works correctly",
	}
	if len(task.SuccessCriteria) != len(expectedSuccessCriteria) {
		t.Errorf("Expected %d success criteria, got %d: %v",
			len(expectedSuccessCriteria), len(task.SuccessCriteria), task.SuccessCriteria)
	} else {
		for i, expected := range expectedSuccessCriteria {
			if task.SuccessCriteria[i] != expected {
				t.Errorf("Success criterion %d mismatch: expected %q, got %q",
					i, expected, task.SuccessCriteria[i])
			}
		}
	}

	// ===============================
	// Verify Test Commands (standard section)
	// ===============================
	expectedTestCommands := []string{
		"go test ./internal/integration/... -v",
		"go test -race ./internal/integration/...",
		"go test -tags=integration ./...",
	}
	if len(task.TestCommands) != len(expectedTestCommands) {
		t.Errorf("Expected %d test commands, got %d: %v",
			len(expectedTestCommands), len(task.TestCommands), task.TestCommands)
	} else {
		for i, expected := range expectedTestCommands {
			if task.TestCommands[i] != expected {
				t.Errorf("Test command %d mismatch: expected %q, got %q",
					i, expected, task.TestCommands[i])
			}
		}
	}
}

// TestFullIntegrationMultipleTasks verifies that all 4 new features work correctly
// across multiple tasks in the same markdown document
func TestFullIntegrationMultipleTasks(t *testing.T) {
	markdown := `## Task 1: Component Task

**File(s)**: ` + "`component.go`" + `
**Depends on**: None
**Estimated time**: 30m
**Type**: component

**Success Criteria**:
- Component implements interface

**Structured Criteria**:
1. Interface compliance
   Verification:
   - Command: go vet ./...
   - Expected: exit 0

**Key Points**:
1. Follow interface segregation
   - Reference: docs/interfaces.md

## Task 2: Integration Task

**File(s)**: ` + "`integration.go`" + `
**Depends on**: Task 1
**Estimated time**: 1h
**Type**: integration

**Success Criteria**:
- Integration layer works

**Integration Criteria**:
- Components wire correctly
- Data flows end-to-end

**Runtime Metadata**:
**Dependency Checks**:
- go build: Verify build works

**Documentation Targets**:
- docs/integration.md (Setup)

**Prompt Blocks**:
- context: Wire components together

**Structured Criteria**:
1. Integration tests pass
   Verification:
   - Command: go test -tags=integration ./...
   - Expected: PASS

**Key Points**:
1. Verify component compatibility
   - Impact: High
   - Note: Check interface contracts
`

	p := parser.NewMarkdownParser()
	plan, err := p.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(plan.Tasks))
	}

	// ===============================
	// Verify Task 1 (Component Task)
	// ===============================
	task1 := plan.Tasks[0]
	if task1.Number != "1" {
		t.Errorf("Task 1: expected number '1', got %q", task1.Number)
	}
	if task1.Type != "component" {
		t.Errorf("Task 1: expected type 'component', got %q", task1.Type)
	}
	if len(task1.IntegrationCriteria) != 0 {
		t.Errorf("Task 1: expected no integration criteria, got %d", len(task1.IntegrationCriteria))
	}
	if task1.RuntimeMetadata != nil {
		t.Errorf("Task 1: expected no runtime metadata, got %+v", task1.RuntimeMetadata)
	}
	if len(task1.StructuredCriteria) != 1 {
		t.Errorf("Task 1: expected 1 structured criteria, got %d", len(task1.StructuredCriteria))
	}
	if len(task1.KeyPoints) != 1 {
		t.Errorf("Task 1: expected 1 key point, got %d", len(task1.KeyPoints))
	}

	// ===============================
	// Verify Task 2 (Integration Task)
	// ===============================
	task2 := plan.Tasks[1]
	if task2.Number != "2" {
		t.Errorf("Task 2: expected number '2', got %q", task2.Number)
	}
	if task2.Type != "integration" {
		t.Errorf("Task 2: expected type 'integration', got %q", task2.Type)
	}

	// Integration Criteria
	if len(task2.IntegrationCriteria) != 2 {
		t.Errorf("Task 2: expected 2 integration criteria, got %d: %v",
			len(task2.IntegrationCriteria), task2.IntegrationCriteria)
	} else {
		if task2.IntegrationCriteria[0] != "Components wire correctly" {
			t.Errorf("Task 2: integration criterion 0 mismatch: got %q", task2.IntegrationCriteria[0])
		}
		if task2.IntegrationCriteria[1] != "Data flows end-to-end" {
			t.Errorf("Task 2: integration criterion 1 mismatch: got %q", task2.IntegrationCriteria[1])
		}
	}

	// Runtime Metadata
	if task2.RuntimeMetadata == nil {
		t.Errorf("Task 2: expected runtime metadata to be non-nil")
	} else {
		if len(task2.RuntimeMetadata.DependencyChecks) != 1 {
			t.Errorf("Task 2: expected 1 dependency check, got %d", len(task2.RuntimeMetadata.DependencyChecks))
		}
		if len(task2.RuntimeMetadata.DocumentationTargets) != 1 {
			t.Errorf("Task 2: expected 1 documentation target, got %d", len(task2.RuntimeMetadata.DocumentationTargets))
		}
		if len(task2.RuntimeMetadata.PromptBlocks) != 1 {
			t.Errorf("Task 2: expected 1 prompt block, got %d", len(task2.RuntimeMetadata.PromptBlocks))
		}
	}

	// Structured Criteria
	if len(task2.StructuredCriteria) != 1 {
		t.Errorf("Task 2: expected 1 structured criteria, got %d", len(task2.StructuredCriteria))
	} else {
		sc := task2.StructuredCriteria[0]
		if sc.Criterion != "Integration tests pass" {
			t.Errorf("Task 2: structured criterion text mismatch: got %q", sc.Criterion)
		}
		if sc.Verification == nil {
			t.Errorf("Task 2: structured criterion should have verification block")
		} else {
			if sc.Verification.Command != "go test -tags=integration ./..." {
				t.Errorf("Task 2: structured criterion command mismatch: got %q", sc.Verification.Command)
			}
		}
	}

	// Key Points
	if len(task2.KeyPoints) != 1 {
		t.Errorf("Task 2: expected 1 key point, got %d", len(task2.KeyPoints))
	} else {
		kp := task2.KeyPoints[0]
		if kp.Point != "Verify component compatibility" {
			t.Errorf("Task 2: key point text mismatch: got %q", kp.Point)
		}
		if !strings.Contains(kp.Details, "Impact: High") {
			t.Errorf("Task 2: key point should contain 'Impact: High', got %q", kp.Details)
		}
		if !strings.Contains(kp.Details, "Note: Check interface contracts") {
			t.Errorf("Task 2: key point should contain 'Note: Check interface contracts', got %q", kp.Details)
		}
	}
}

// TestFullIntegrationEmptySections verifies parser handles empty or missing sections gracefully
func TestFullIntegrationEmptySections(t *testing.T) {
	markdown := `## Task 1: Minimal Task

**File(s)**: ` + "`minimal.go`" + `
**Depends on**: None
**Estimated time**: 15m

This is a minimal task with no new v2.9+ features.

**Success Criteria**:
- Task completes
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

	// All new features should be empty/nil
	if len(task.IntegrationCriteria) != 0 {
		t.Errorf("Expected no integration criteria, got %d", len(task.IntegrationCriteria))
	}
	if task.RuntimeMetadata != nil {
		t.Errorf("Expected RuntimeMetadata to be nil, got %+v", task.RuntimeMetadata)
	}
	if len(task.StructuredCriteria) != 0 {
		t.Errorf("Expected no structured criteria, got %d", len(task.StructuredCriteria))
	}
	if len(task.KeyPoints) != 0 {
		t.Errorf("Expected no key points, got %d", len(task.KeyPoints))
	}

	// Standard fields should still work
	if len(task.SuccessCriteria) != 1 {
		t.Errorf("Expected 1 success criteria, got %d", len(task.SuccessCriteria))
	}
}

// TestFullIntegrationSectionOrdering verifies parser handles different section orderings
func TestFullIntegrationSectionOrdering(t *testing.T) {
	// Test with sections in non-standard order
	markdown := `## Task 1: Reversed Order Task

**File(s)**: ` + "`reversed.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Key Points**:
1. First key point
   - Reference: ref1.md

**Structured Criteria**:
1. First structured
   Verification:
   - Command: echo test

**Runtime Metadata**:
**Prompt Blocks**:
- context: Test context

**Integration Criteria**:
- Integration criterion 1

**Success Criteria**:
- Task completes

**Type**: integration
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

	// All features should parse correctly regardless of order
	if len(task.IntegrationCriteria) != 1 {
		t.Errorf("Expected 1 integration criteria, got %d", len(task.IntegrationCriteria))
	}
	if task.RuntimeMetadata == nil {
		t.Errorf("Expected RuntimeMetadata to be non-nil")
	} else if len(task.RuntimeMetadata.PromptBlocks) != 1 {
		t.Errorf("Expected 1 prompt block, got %d", len(task.RuntimeMetadata.PromptBlocks))
	}
	if len(task.StructuredCriteria) != 1 {
		t.Errorf("Expected 1 structured criteria, got %d", len(task.StructuredCriteria))
	}
	if len(task.KeyPoints) != 1 {
		t.Errorf("Expected 1 key point, got %d", len(task.KeyPoints))
	}
	if task.Type != "integration" {
		t.Errorf("Expected type 'integration', got %q", task.Type)
	}
}
