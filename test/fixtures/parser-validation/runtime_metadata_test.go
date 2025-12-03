package parser_validation

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/parser"
)

// TestRuntimeMetadata validates that parseRuntimeMetadataMarkdown works correctly
// by parsing markdown content with **Runtime Metadata**: section and verifying
// the task.RuntimeMetadata contains expected values for all subsections.
func TestRuntimeMetadata(t *testing.T) {
	tests := []struct {
		name                      string
		markdown                  string
		expectNil                 bool
		expectedDependencyChecks  int
		expectedDocTargets        int
		expectedPromptBlocks      int
		verifyDependencyChecks    func(t *testing.T, checks []struct{ Command, Description string })
		verifyDocTargets          func(t *testing.T, targets []struct{ Location, Section string })
		verifyPromptBlocks        func(t *testing.T, blocks []struct{ Type, Content string })
	}{
		{
			name: "complete_runtime_metadata",
			markdown: `## Task 1: Test Task

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Runtime Metadata**:
**Dependency Checks**:
- go version: Verify Go is installed
- go mod tidy: Ensure dependencies are resolved

**Documentation Targets**:
- docs/api.md (Overview)
- README.md (Installation)

**Prompt Blocks**:
- context: This task implements the core API
- constraint: Must maintain backward compatibility

Some other content here
`,
			expectNil:                false,
			expectedDependencyChecks: 2,
			expectedDocTargets:       2,
			expectedPromptBlocks:     2,
			verifyDependencyChecks: func(t *testing.T, checks []struct{ Command, Description string }) {
				if len(checks) < 2 {
					return
				}
				if checks[0].Command != "go version" || checks[0].Description != "Verify Go is installed" {
					t.Errorf("First dependency check mismatch: got command=%q, desc=%q", checks[0].Command, checks[0].Description)
				}
				if checks[1].Command != "go mod tidy" || checks[1].Description != "Ensure dependencies are resolved" {
					t.Errorf("Second dependency check mismatch: got command=%q, desc=%q", checks[1].Command, checks[1].Description)
				}
			},
			verifyDocTargets: func(t *testing.T, targets []struct{ Location, Section string }) {
				if len(targets) < 2 {
					return
				}
				if targets[0].Location != "docs/api.md" || targets[0].Section != "Overview" {
					t.Errorf("First doc target mismatch: got location=%q, section=%q", targets[0].Location, targets[0].Section)
				}
				if targets[1].Location != "README.md" || targets[1].Section != "Installation" {
					t.Errorf("Second doc target mismatch: got location=%q, section=%q", targets[1].Location, targets[1].Section)
				}
			},
			verifyPromptBlocks: func(t *testing.T, blocks []struct{ Type, Content string }) {
				if len(blocks) < 2 {
					return
				}
				if blocks[0].Type != "context" || blocks[0].Content != "This task implements the core API" {
					t.Errorf("First prompt block mismatch: got type=%q, content=%q", blocks[0].Type, blocks[0].Content)
				}
				if blocks[1].Type != "constraint" || blocks[1].Content != "Must maintain backward compatibility" {
					t.Errorf("Second prompt block mismatch: got type=%q, content=%q", blocks[1].Type, blocks[1].Content)
				}
			},
		},
		{
			name: "dependency_checks_only",
			markdown: `## Task 1: Dependency Check Task

**File(s):** ` + "`check.go`" + `
**Depends on:** None
**Estimated time:** 15m

**Runtime Metadata**:
**Dependency Checks**:
- npm install: Install dependencies
- npm run build: Verify build works

**Success Criteria**:
- Task completes successfully
`,
			expectNil:                false,
			expectedDependencyChecks: 2,
			expectedDocTargets:       0,
			expectedPromptBlocks:     0,
			verifyDependencyChecks: func(t *testing.T, checks []struct{ Command, Description string }) {
				if len(checks) < 2 {
					return
				}
				if checks[0].Command != "npm install" || checks[0].Description != "Install dependencies" {
					t.Errorf("First dependency check mismatch: got command=%q, desc=%q", checks[0].Command, checks[0].Description)
				}
				if checks[1].Command != "npm run build" || checks[1].Description != "Verify build works" {
					t.Errorf("Second dependency check mismatch: got command=%q, desc=%q", checks[1].Command, checks[1].Description)
				}
			},
		},
		{
			name: "documentation_targets_only",
			markdown: `## Task 1: Documentation Task

**File(s):** ` + "`docs.go`" + `
**Depends on:** None
**Estimated time:** 20m

**Runtime Metadata**:
**Documentation Targets**:
- CHANGELOG.md (Added)
- docs/architecture.md

**Success Criteria**:
- Documentation updated
`,
			expectNil:          false,
			expectedDocTargets: 2,
			verifyDocTargets: func(t *testing.T, targets []struct{ Location, Section string }) {
				if len(targets) < 2 {
					return
				}
				if targets[0].Location != "CHANGELOG.md" || targets[0].Section != "Added" {
					t.Errorf("First doc target mismatch: got location=%q, section=%q", targets[0].Location, targets[0].Section)
				}
				if targets[1].Location != "docs/architecture.md" || targets[1].Section != "" {
					t.Errorf("Second doc target mismatch: got location=%q, section=%q (expected empty section)", targets[1].Location, targets[1].Section)
				}
			},
		},
		{
			name: "prompt_blocks_only",
			markdown: `## Task 1: Prompt Block Task

**File(s):** ` + "`prompt.go`" + `
**Depends on:** None
**Estimated time:** 25m

**Runtime Metadata**:
**Prompt Blocks**:
- instruction: Follow TDD principles
- context: This is a critical path component
- constraint: No breaking changes allowed

**Success Criteria**:
- Task completes
`,
			expectNil:            false,
			expectedPromptBlocks: 3,
			verifyPromptBlocks: func(t *testing.T, blocks []struct{ Type, Content string }) {
				if len(blocks) < 3 {
					return
				}
				if blocks[0].Type != "instruction" || blocks[0].Content != "Follow TDD principles" {
					t.Errorf("First prompt block mismatch: got type=%q, content=%q", blocks[0].Type, blocks[0].Content)
				}
				if blocks[1].Type != "context" || blocks[1].Content != "This is a critical path component" {
					t.Errorf("Second prompt block mismatch: got type=%q, content=%q", blocks[1].Type, blocks[1].Content)
				}
				if blocks[2].Type != "constraint" || blocks[2].Content != "No breaking changes allowed" {
					t.Errorf("Third prompt block mismatch: got type=%q, content=%q", blocks[2].Type, blocks[2].Content)
				}
			},
		},
		{
			name: "no_runtime_metadata",
			markdown: `## Task 1: Regular Task

**File(s):** ` + "`regular.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Success Criteria**:
- Task completes successfully

Just a regular task without runtime metadata.
`,
			expectNil: true,
		},
		{
			name: "empty_runtime_metadata",
			markdown: `## Task 1: Empty Metadata Task

**File(s):** ` + "`empty.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Runtime Metadata**:

**Success Criteria**:
- Task completes
`,
			expectNil: true,
		},
		{
			name: "dependency_check_without_description",
			markdown: `## Task 1: Simple Check Task

**File(s):** ` + "`simple.go`" + `
**Depends on:** None
**Estimated time:** 15m

**Runtime Metadata**:
**Dependency Checks**:
- go test ./...

**Success Criteria**:
- Tests pass
`,
			expectNil:                false,
			expectedDependencyChecks: 1,
			verifyDependencyChecks: func(t *testing.T, checks []struct{ Command, Description string }) {
				if len(checks) < 1 {
					return
				}
				if checks[0].Command != "go test ./..." {
					t.Errorf("Dependency check command mismatch: got %q", checks[0].Command)
				}
				// Description can be empty for simple commands
			},
		},
		{
			name: "documentation_target_without_section",
			markdown: `## Task 1: Doc Target Task

**File(s):** ` + "`doc.go`" + `
**Depends on:** None
**Estimated time:** 15m

**Runtime Metadata**:
**Documentation Targets**:
- docs/README.md

**Success Criteria**:
- Docs updated
`,
			expectNil:          false,
			expectedDocTargets: 1,
			verifyDocTargets: func(t *testing.T, targets []struct{ Location, Section string }) {
				if len(targets) < 1 {
					return
				}
				if targets[0].Location != "docs/README.md" {
					t.Errorf("Doc target location mismatch: got %q", targets[0].Location)
				}
				if targets[0].Section != "" {
					t.Errorf("Doc target section should be empty, got %q", targets[0].Section)
				}
			},
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

			// Verify RuntimeMetadata presence/absence
			if tt.expectNil {
				if task.RuntimeMetadata != nil {
					t.Errorf("Expected RuntimeMetadata to be nil, but got: %+v", task.RuntimeMetadata)
				}
				return
			}

			if task.RuntimeMetadata == nil {
				t.Fatalf("Expected RuntimeMetadata to be non-nil")
			}

			rm := task.RuntimeMetadata

			// Verify DependencyChecks count
			if tt.expectedDependencyChecks > 0 {
				if len(rm.DependencyChecks) != tt.expectedDependencyChecks {
					t.Errorf("Expected %d dependency checks, got %d: %v",
						tt.expectedDependencyChecks, len(rm.DependencyChecks), rm.DependencyChecks)
				}
				if tt.verifyDependencyChecks != nil {
					// Convert to simple struct for verification
					checks := make([]struct{ Command, Description string }, len(rm.DependencyChecks))
					for i, c := range rm.DependencyChecks {
						checks[i].Command = c.Command
						checks[i].Description = c.Description
					}
					tt.verifyDependencyChecks(t, checks)
				}
			}

			// Verify DocumentationTargets count
			if tt.expectedDocTargets > 0 {
				if len(rm.DocumentationTargets) != tt.expectedDocTargets {
					t.Errorf("Expected %d documentation targets, got %d: %v",
						tt.expectedDocTargets, len(rm.DocumentationTargets), rm.DocumentationTargets)
				}
				if tt.verifyDocTargets != nil {
					// Convert to simple struct for verification
					targets := make([]struct{ Location, Section string }, len(rm.DocumentationTargets))
					for i, t := range rm.DocumentationTargets {
						targets[i].Location = t.Location
						targets[i].Section = t.Section
					}
					tt.verifyDocTargets(t, targets)
				}
			}

			// Verify PromptBlocks count
			if tt.expectedPromptBlocks > 0 {
				if len(rm.PromptBlocks) != tt.expectedPromptBlocks {
					t.Errorf("Expected %d prompt blocks, got %d: %v",
						tt.expectedPromptBlocks, len(rm.PromptBlocks), rm.PromptBlocks)
				}
				if tt.verifyPromptBlocks != nil {
					// Convert to simple struct for verification
					blocks := make([]struct{ Type, Content string }, len(rm.PromptBlocks))
					for i, b := range rm.PromptBlocks {
						blocks[i].Type = b.Type
						blocks[i].Content = b.Content
					}
					tt.verifyPromptBlocks(t, blocks)
				}
			}
		})
	}
}

// TestRuntimeMetadataDependencyChecksParsing verifies that Dependency Checks subsection
// correctly parses command and description pairs
func TestRuntimeMetadataDependencyChecksParsing(t *testing.T) {
	markdown := `## Task 1: Dependency Checks Test

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Runtime Metadata**:
**Dependency Checks**:
- go version: Verify Go 1.21+ is installed
- docker ps: Check Docker is running
- kubectl version: Verify kubectl is configured
- make lint: Run linter to check code style

**Success Criteria**:
- All checks pass
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
	if task.RuntimeMetadata == nil {
		t.Fatalf("Expected RuntimeMetadata to be non-nil")
	}

	checks := task.RuntimeMetadata.DependencyChecks
	if len(checks) != 4 {
		t.Fatalf("Expected 4 dependency checks, got %d", len(checks))
	}

	// Verify each check
	expectedChecks := []struct {
		command     string
		description string
	}{
		{"go version", "Verify Go 1.21+ is installed"},
		{"docker ps", "Check Docker is running"},
		{"kubectl version", "Verify kubectl is configured"},
		{"make lint", "Run linter to check code style"},
	}

	for i, expected := range expectedChecks {
		if checks[i].Command != expected.command {
			t.Errorf("Check %d command mismatch: expected %q, got %q", i, expected.command, checks[i].Command)
		}
		if checks[i].Description != expected.description {
			t.Errorf("Check %d description mismatch: expected %q, got %q", i, expected.description, checks[i].Description)
		}
	}
}

// TestRuntimeMetadataDocumentationTargetsParsing verifies that Documentation Targets subsection
// correctly parses location and section pairs
func TestRuntimeMetadataDocumentationTargetsParsing(t *testing.T) {
	markdown := `## Task 1: Documentation Targets Test

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Runtime Metadata**:
**Documentation Targets**:
- docs/api.md (API Reference)
- README.md (Getting Started)
- CHANGELOG.md (v2.9.0)
- docs/architecture.md

**Success Criteria**:
- Documentation updated
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
	if task.RuntimeMetadata == nil {
		t.Fatalf("Expected RuntimeMetadata to be non-nil")
	}

	targets := task.RuntimeMetadata.DocumentationTargets
	if len(targets) != 4 {
		t.Fatalf("Expected 4 documentation targets, got %d", len(targets))
	}

	// Verify each target
	expectedTargets := []struct {
		location string
		section  string
	}{
		{"docs/api.md", "API Reference"},
		{"README.md", "Getting Started"},
		{"CHANGELOG.md", "v2.9.0"},
		{"docs/architecture.md", ""}, // No section specified
	}

	for i, expected := range expectedTargets {
		if targets[i].Location != expected.location {
			t.Errorf("Target %d location mismatch: expected %q, got %q", i, expected.location, targets[i].Location)
		}
		if targets[i].Section != expected.section {
			t.Errorf("Target %d section mismatch: expected %q, got %q", i, expected.section, targets[i].Section)
		}
	}
}

// TestRuntimeMetadataPromptBlocksParsing verifies that Prompt Blocks subsection
// correctly parses type and content pairs
func TestRuntimeMetadataPromptBlocksParsing(t *testing.T) {
	markdown := `## Task 1: Prompt Blocks Test

**File(s):** ` + "`test.go`" + `
**Depends on:** None
**Estimated time:** 30m

**Runtime Metadata**:
**Prompt Blocks**:
- context: This implements the authentication layer
- instruction: Follow security best practices
- constraint: Must not break existing API contracts
- warning: Handle sensitive data carefully

**Success Criteria**:
- Implementation complete
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
	if task.RuntimeMetadata == nil {
		t.Fatalf("Expected RuntimeMetadata to be non-nil")
	}

	blocks := task.RuntimeMetadata.PromptBlocks
	if len(blocks) != 4 {
		t.Fatalf("Expected 4 prompt blocks, got %d", len(blocks))
	}

	// Verify each block
	expectedBlocks := []struct {
		blockType string
		content   string
	}{
		{"context", "This implements the authentication layer"},
		{"instruction", "Follow security best practices"},
		{"constraint", "Must not break existing API contracts"},
		{"warning", "Handle sensitive data carefully"},
	}

	for i, expected := range expectedBlocks {
		if blocks[i].Type != expected.blockType {
			t.Errorf("Block %d type mismatch: expected %q, got %q", i, expected.blockType, blocks[i].Type)
		}
		if blocks[i].Content != expected.content {
			t.Errorf("Block %d content mismatch: expected %q, got %q", i, expected.content, blocks[i].Content)
		}
	}
}

// TestRuntimeMetadataWithOtherSections verifies that Runtime Metadata parsing
// works correctly alongside other task sections
func TestRuntimeMetadataWithOtherSections(t *testing.T) {
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

**Runtime Metadata**:
**Dependency Checks**:
- go build ./...: Verify code compiles
- go test ./...: Run unit tests

**Documentation Targets**:
- docs/integration.md (Architecture)

**Prompt Blocks**:
- context: Integration task for API and database layers

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

	// Verify other metadata is parsed correctly alongside RuntimeMetadata
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

	// Verify RuntimeMetadata
	if task.RuntimeMetadata == nil {
		t.Fatalf("Expected RuntimeMetadata to be non-nil")
	}

	rm := task.RuntimeMetadata
	if len(rm.DependencyChecks) != 2 {
		t.Errorf("Expected 2 dependency checks, got %d", len(rm.DependencyChecks))
	}
	if len(rm.DocumentationTargets) != 1 {
		t.Errorf("Expected 1 documentation target, got %d", len(rm.DocumentationTargets))
	}
	if len(rm.PromptBlocks) != 1 {
		t.Errorf("Expected 1 prompt block, got %d", len(rm.PromptBlocks))
	}

	// Verify specific values
	if rm.DependencyChecks[0].Command != "go build ./..." {
		t.Errorf("Expected first dependency check command 'go build ./...', got %q", rm.DependencyChecks[0].Command)
	}
	if rm.DocumentationTargets[0].Location != "docs/integration.md" {
		t.Errorf("Expected doc target 'docs/integration.md', got %q", rm.DocumentationTargets[0].Location)
	}
	if rm.DocumentationTargets[0].Section != "Architecture" {
		t.Errorf("Expected doc target section 'Architecture', got %q", rm.DocumentationTargets[0].Section)
	}
	if rm.PromptBlocks[0].Type != "context" {
		t.Errorf("Expected prompt block type 'context', got %q", rm.PromptBlocks[0].Type)
	}
}
