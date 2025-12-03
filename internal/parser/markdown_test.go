package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

func TestParseMarkdownPlan(t *testing.T) {
	markdown := `# Implementation Plan: Test Plan

**Created**: 2025-11-07
**Estimated Tasks**: 2

## Task 1: First Task

**File(s)**: ` + "`file1.go`, `file2.go`" + `
**Depends on**: None
**Estimated time**: 30m
**Agent**: godev

### What you're building
Test task description

### Implementation
Implementation details here
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(plan.Tasks))
	}

	task := plan.Tasks[0]
	if task.Number != "1" {
		t.Errorf("Expected task number 1, got %s", task.Number)
	}
	if task.Name != "First Task" {
		t.Errorf("Expected task name 'First Task', got '%s'", task.Name)
	}
	if len(task.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(task.Files))
	}
	if task.EstimatedTime != 30*time.Minute {
		t.Errorf("Expected 30m, got %v", task.EstimatedTime)
	}
	if task.Agent != "godev" {
		t.Errorf("Expected agent 'godev', got '%s'", task.Agent)
	}
}

func TestExtractTasks(t *testing.T) {
	tests := []struct {
		name          string
		markdown      string
		expectedCount int
		expectedNames []string
	}{
		{
			name: "multiple tasks",
			markdown: `# Test Plan

## Task 1: First Task

**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m

Description here

## Task 2: Second Task

**File(s)**: ` + "`file2.go`" + `
**Depends on**: Task 1
**Estimated time**: 1h

Another description
`,
			expectedCount: 2,
			expectedNames: []string{"First Task", "Second Task"},
		},
		{
			name: "no tasks",
			markdown: `# Test Plan

Some content without task headings
`,
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name: "task with dependencies",
			markdown: `# Test Plan

## Task 3: Third Task

**File(s)**: ` + "`file3.go`" + `
**Depends on**: Task 1, Task 2
**Estimated time**: 45m

Description
`,
			expectedCount: 1,
			expectedNames: []string{"Third Task"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if len(plan.Tasks) != tt.expectedCount {
				t.Errorf("Expected %d tasks, got %d", tt.expectedCount, len(plan.Tasks))
			}

			for i, expectedName := range tt.expectedNames {
				if i >= len(plan.Tasks) {
					break
				}
				if plan.Tasks[i].Name != expectedName {
					t.Errorf("Task %d: expected name '%s', got '%s'", i, expectedName, plan.Tasks[i].Name)
				}
			}
		})
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name              string
		markdown          string
		expectFrontmatter bool
		expectedAgent     string
	}{
		{
			name: "with frontmatter",
			markdown: `---
conductor:
  default_agent: swiftdev
  quality_control:
    enabled: true
    review_agent: reviewer
---

# Test Plan

## Task 1: Test

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectFrontmatter: true,
			expectedAgent:     "swiftdev",
		},
		{
			name: "without frontmatter",
			markdown: `# Test Plan

## Task 1: Test

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectFrontmatter: false,
			expectedAgent:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if tt.expectFrontmatter {
				if plan.DefaultAgent != tt.expectedAgent {
					t.Errorf("Expected default agent '%s', got '%s'", tt.expectedAgent, plan.DefaultAgent)
				}
			}
		})
	}
}

func TestParseTaskMetadata(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedFiles []string
		expectedDeps  []string
		expectedTime  time.Duration
		expectedAgent string
	}{
		{
			name: "all metadata fields",
			content: `**File(s)**: ` + "`file1.go`, `file2.go`" + `
**Depends on**: Task 1, Task 2
**Estimated time**: 2h
**Agent**: godev`,
			expectedFiles: []string{"file1.go", "file2.go"},
			expectedDeps:  []string{"1", "2"},
			expectedTime:  2 * time.Hour,
			expectedAgent: "godev",
		},
		{
			name: "no dependencies",
			content: `**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m`,
			expectedFiles: []string{"file1.go"},
			expectedDeps:  []string{},
			expectedTime:  30 * time.Minute,
			expectedAgent: "",
		},
		{
			name: "dependencies with numbers only",
			content: `**File(s)**: ` + "`file1.go`" + `
**Depends on**: 3, 5
**Estimated time**: 1h`,
			expectedFiles: []string{"file1.go"},
			expectedDeps:  []string{"3", "5"},
			expectedTime:  1 * time.Hour,
			expectedAgent: "",
		},
		{
			name: "multiple files",
			content: `**File(s)**: ` + "`internal/parser/markdown.go`, `internal/parser/markdown_test.go`" + `
**Depends on**: Task 3
**Estimated time**: 2h30m
**Agent**: testdev`,
			expectedFiles: []string{"internal/parser/markdown.go", "internal/parser/markdown_test.go"},
			expectedDeps:  []string{"3"},
			expectedTime:  150 * time.Minute, // 2h30m
			expectedAgent: "testdev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{}
			parseTaskMetadata(task, tt.content)

			// Check files
			if len(task.Files) != len(tt.expectedFiles) {
				t.Errorf("Expected %d files, got %d", len(tt.expectedFiles), len(task.Files))
			}
			for i, expected := range tt.expectedFiles {
				if i >= len(task.Files) {
					break
				}
				// Remove backticks if present
				actual := strings.Trim(task.Files[i], "`")
				expected = strings.Trim(expected, "`")
				if actual != expected {
					t.Errorf("File %d: expected '%s', got '%s'", i, expected, actual)
				}
			}

			// Check dependencies
			if len(task.DependsOn) != len(tt.expectedDeps) {
				t.Errorf("Expected %d dependencies, got %d", len(tt.expectedDeps), len(task.DependsOn))
			}
			for i, expected := range tt.expectedDeps {
				if i >= len(task.DependsOn) {
					break
				}
				if task.DependsOn[i] != expected {
					t.Errorf("Dependency %d: expected %s, got %s", i, expected, task.DependsOn[i])
				}
			}

			// Check estimated time
			if task.EstimatedTime != tt.expectedTime {
				t.Errorf("Expected time %v, got %v", tt.expectedTime, task.EstimatedTime)
			}

			// Check agent
			if task.Agent != tt.expectedAgent {
				t.Errorf("Expected agent '%s', got '%s'", tt.expectedAgent, task.Agent)
			}
		})
	}
}

func TestParseTaskPrompt(t *testing.T) {
	markdown := `# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

### What you're building
This is the description

### Implementation
This is the implementation section

### Test First
Write tests first
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(plan.Tasks))
	}

	task := plan.Tasks[0]
	if task.Prompt == "" {
		t.Error("Task prompt should not be empty")
	}

	// Verify prompt contains key sections
	if !strings.Contains(task.Prompt, "What you're building") {
		t.Error("Prompt should contain 'What you're building' section")
	}
	if !strings.Contains(task.Prompt, "Implementation") {
		t.Error("Prompt should contain 'Implementation' section")
	}
	if !strings.Contains(task.Prompt, "Test First") {
		t.Error("Prompt should contain 'Test First' section")
	}

	// Verify file injection is present
	if !strings.Contains(task.Prompt, "Target Files (REQUIRED)") {
		t.Error("Prompt should contain 'Target Files (REQUIRED)' section")
	}
	if !strings.Contains(task.Prompt, "test.go") {
		t.Error("Prompt should contain the file 'test.go'")
	}
	if !strings.Contains(task.Prompt, "MUST create/modify these exact files") {
		t.Error("Prompt should contain file injection instructions")
	}
}

func TestParseMarkdownIgnoresHeadingsInCodeBlocks(t *testing.T) {
	markdown := `# Test Plan

## Task 1: Real Task

**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m

### Implementation

Here's an example test:

` + "```go" + `
func TestExample(t *testing.T) {
    markdown := ` + "`# Plan\n\n## Task 2: Fake Task\n\n**File(s)**: file2.go`" + `
    // This is inside a code block
}
` + "```" + `

## Task 3: Another Real Task

**File(s)**: ` + "`file3.go`" + `
**Depends on**: Task 1
**Estimated time**: 1h
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Should only find Task 1 and Task 3, not Task 2 (which is in code block)
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks (ignoring code block), got %d", len(plan.Tasks))
		for i, task := range plan.Tasks {
			t.Logf("Task %d: Number=%s, Name=%s", i, task.Number, task.Name)
		}
	}

	// Verify tasks are 1 and 3, not 2
	if len(plan.Tasks) >= 1 && plan.Tasks[0].Number != "1" {
		t.Errorf("First task should be number 1, got %s", plan.Tasks[0].Number)
	}
	if len(plan.Tasks) >= 2 && plan.Tasks[1].Number != "3" {
		t.Errorf("Second task should be number 3, got %s", plan.Tasks[1].Number)
	}
}

func TestParseMarkdownExtractsHeadingsOutsideCodeBlocks(t *testing.T) {
	markdown := `# Test Plan

## Task 1: Before Code Block

**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m

Some content before code block.

` + "```go" + `
// Code example
func example() {
    // Not a real task
}
` + "```" + `

## Task 2: After Code Block

**File(s)**: ` + "`file2.go`" + `
**Depends on**: Task 1
**Estimated time**: 45m

Some content after code block.
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}

	if len(plan.Tasks) >= 1 && plan.Tasks[0].Name != "Before Code Block" {
		t.Errorf("First task name: expected 'Before Code Block', got '%s'", plan.Tasks[0].Name)
	}
	if len(plan.Tasks) >= 2 && plan.Tasks[1].Name != "After Code Block" {
		t.Errorf("Second task name: expected 'After Code Block', got '%s'", plan.Tasks[1].Name)
	}
}

func TestParseMetadataIgnoresCodeBlocks(t *testing.T) {
	// This test verifies that metadata extraction ignores patterns in code blocks
	// This is a regression test for the bug where variables like "agentRegex"
	// from code examples were being extracted as agent names
	content := `**Agent**: real-agent
**File(s)**: ` + "`real-file.go`" + `
**Depends on**: Task 1
**Estimated time**: 1h

### Implementation

Here's a code example that contains fake metadata:

` + "```go" + `
agentRegex := regexp.MustCompile(` + "`" + `\*\*Agent\*\*:\s*(\S+)` + "`" + `)
fileRegex := regexp.MustCompile(` + "`" + `\*\*File\(s\)\*\*:\s*(.+)` + "`" + `)
depRegex := regexp.MustCompile(` + "`" + `\*\*Depends on\*\*:\s*(.+)` + "`" + `)
// This code should not affect metadata extraction:
// **Agent**: fake-agent
// **File(s)**: fake-file.go
// **Depends on**: Task 99
// **Estimated time**: 5h
` + "```" + `

More content after code block.
`

	task := &models.Task{}
	parseTaskMetadata(task, content)

	// Verify only real metadata extracted, not fake metadata from code block
	if task.Agent != "real-agent" {
		t.Errorf("Expected agent 'real-agent', got '%s' (should not extract from code block)", task.Agent)
	}

	if len(task.Files) != 1 || task.Files[0] != "real-file.go" {
		t.Errorf("Expected files ['real-file.go'], got %v (should not extract from code block)", task.Files)
	}

	if len(task.DependsOn) != 1 || task.DependsOn[0] != "1" {
		t.Errorf("Expected dependencies [1], got %v (should not extract from code block)", task.DependsOn)
	}

	if task.EstimatedTime != 1*time.Hour {
		t.Errorf("Expected time 1h, got %v (should not extract from code block)", task.EstimatedTime)
	}
}

func TestRemoveCodeBlocks(t *testing.T) {
	// Test the helper function directly
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no code blocks",
			input:    "This is plain text\nNo code here",
			expected: "This is plain text\nNo code here\n",
		},
		{
			name: "single code block",
			input: `Text before
` + "```go" + `
code here
` + "```" + `
Text after`,
			expected: "Text before\nText after\n",
		},
		{
			name: "multiple code blocks",
			input: `Start
` + "```" + `
first block
` + "```" + `
Middle
` + "```" + `
second block
` + "```" + `
End`,
			expected: "Start\nMiddle\nEnd\n",
		},
		{
			name: "preserves metadata outside blocks",
			input: `**Agent**: real-agent
` + "```" + `
**Agent**: fake-agent
` + "```" + `
**File(s)**: real.go`,
			expected: "**Agent**: real-agent\n**File(s)**: real.go\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeCodeBlocks(tt.input)
			if result != tt.expected {
				t.Errorf("removeCodeBlocks() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseMarkdownHandlesMultipleCodeBlocks(t *testing.T) {
	markdown := `# Test Plan

## Task 1: Real Task One

**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m

First code block:

` + "```go" + `
## Task 99: Fake Task in First Block
` + "```" + `

## Task 2: Real Task Two

**File(s)**: ` + "`file2.go`" + `
**Depends on**: Task 1
**Estimated time**: 45m

Second code block:

` + "```markdown" + `
## Task 98: Fake Task in Second Block

**File(s)**: fake.go
` + "```" + `

## Task 3: Real Task Three

**File(s)**: ` + "`file3.go`" + `
**Depends on**: Task 2
**Estimated time**: 1h
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Should find 3 real tasks, ignoring 2 fake tasks in code blocks
	if len(plan.Tasks) != 3 {
		t.Errorf("Expected 3 tasks (ignoring code blocks), got %d", len(plan.Tasks))
		for i, task := range plan.Tasks {
			t.Logf("Task %d: Number=%s, Name=%s", i, task.Number, task.Name)
		}
	}

	expectedNumbers := []string{"1", "2", "3"}
	for i, expected := range expectedNumbers {
		if i >= len(plan.Tasks) {
			break
		}
		if plan.Tasks[i].Number != expected {
			t.Errorf("Task %d: expected number %s, got %s", i, expected, plan.Tasks[i].Number)
		}
	}
}

func TestParseStatusCheckbox(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedStatus string
	}{
		{
			name:           "completed task with [x]",
			content:        `[x] Task completed`,
			expectedStatus: "completed",
		},
		{
			name:           "pending task with [ ]",
			content:        `[ ] Task pending`,
			expectedStatus: "pending",
		},
		{
			name:           "no checkbox defaults to pending",
			content:        `Some content without checkbox`,
			expectedStatus: "pending",
		},
		{
			name:           "completed with uppercase [X]",
			content:        `[X] Task completed`,
			expectedStatus: "completed",
		},
		{
			name:           "checkbox anywhere in content",
			content:        `[x] Task is done`,
			expectedStatus: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{}
			parseTaskMetadata(task, tt.content)

			if task.Status != tt.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, task.Status)
			}
		})
	}
}

func TestParseStatusInlineAnnotation(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedStatus string
	}{
		{
			name:           "status: completed",
			content:        `**Status**: completed`,
			expectedStatus: "completed",
		},
		{
			name:           "status: pending",
			content:        `**Status**: pending`,
			expectedStatus: "pending",
		},
		{
			name:           "status: in_progress",
			content:        `**Status**: in_progress`,
			expectedStatus: "in_progress",
		},
		{
			name:           "status: skipped",
			content:        `**Status**: skipped`,
			expectedStatus: "skipped",
		},
		{
			name:           "status with extra whitespace",
			content:        `**Status**:   completed  `,
			expectedStatus: "completed",
		},
		{
			name:           "no status field defaults to pending",
			content:        `**File(s)**: test.go`,
			expectedStatus: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{}
			parseTaskMetadata(task, tt.content)

			if task.Status != tt.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, task.Status)
			}
		})
	}
}

func TestParseStatusPrecedence(t *testing.T) {
	// Inline annotation should take precedence over checkbox
	content := `[x] Some checkbox text
**Status**: pending
**File(s)**: test.go`

	task := &models.Task{}
	parseTaskMetadata(task, content)

	// Inline annotation takes precedence
	if task.Status != "pending" {
		t.Errorf("Expected inline annotation to take precedence, got status '%s'", task.Status)
	}
}

func TestParseCompleteTaskMetadata(t *testing.T) {
	// Test that status extraction works alongside other metadata
	content := `[x] Task is complete
**File(s)**: ` + "`file1.go`, `file2.go`" + `
**Depends on**: Task 1, Task 2
**Estimated time**: 2h
**Agent**: godev`

	task := &models.Task{}
	parseTaskMetadata(task, content)

	if task.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", task.Status)
	}
	if len(task.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(task.Files))
	}
	if len(task.DependsOn) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task.DependsOn))
	}
	if task.Agent != "godev" {
		t.Errorf("Expected agent 'godev', got '%s'", task.Agent)
	}
}

func TestParseWorktreeGroup(t *testing.T) {
	tests := []struct {
		name                  string
		content               string
		expectedWorktreeGroup string
	}{
		{
			name: "worktree group chain-1",
			content: `**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m
**Agent**: godev
**WorktreeGroup**: chain-1`,
			expectedWorktreeGroup: "chain-1",
		},
		{
			name: "worktree group independent-3",
			content: `**File(s)**: ` + "`file2.go`" + `
**Depends on**: Task 1
**Estimated time**: 1h
**Agent**: godev
**WorktreeGroup**: independent-3`,
			expectedWorktreeGroup: "independent-3",
		},
		{
			name: "no worktree group field",
			content: `**File(s)**: ` + "`file3.go`" + `
**Depends on**: None
**Estimated time**: 45m
**Agent**: godev`,
			expectedWorktreeGroup: "",
		},
		{
			name: "worktree group with extra whitespace",
			content: `**File(s)**: ` + "`file4.go`" + `
**WorktreeGroup**:   backend-service  `,
			expectedWorktreeGroup: "backend-service",
		},
		{
			name: "worktree group at beginning",
			content: `**WorktreeGroup**: auth-service
**File(s)**: ` + "`auth.go`" + `
**Depends on**: None`,
			expectedWorktreeGroup: "auth-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{}
			parseTaskMetadata(task, tt.content)

			if task.WorktreeGroup != tt.expectedWorktreeGroup {
				t.Errorf("Expected WorktreeGroup '%s', got '%s'", tt.expectedWorktreeGroup, task.WorktreeGroup)
			}
		})
	}
}

func TestParseWorktreeGroupIgnoresCodeBlocks(t *testing.T) {
	// Test that WorktreeGroup extraction ignores patterns in code blocks
	content := `**WorktreeGroup**: real-group
**File(s)**: ` + "`real-file.go`" + `
**Agent**: godev

### Implementation

Here's a code example:

` + "```go" + `
// Example code that should not affect metadata extraction:
// **WorktreeGroup**: fake-group
worktreeGroupRegex := regexp.MustCompile(` + "`" + `\*\*WorktreeGroup\*\*:\s*(\S+)` + "`" + `)
` + "```" + `

More content after code block.
`

	task := &models.Task{}
	parseTaskMetadata(task, content)

	// Verify only real metadata extracted, not fake metadata from code block
	if task.WorktreeGroup != "real-group" {
		t.Errorf("Expected WorktreeGroup 'real-group', got '%s' (should not extract from code block)", task.WorktreeGroup)
	}
}

func TestParseFullTaskWithWorktreeGroup(t *testing.T) {
	// Test full task parsing including WorktreeGroup
	markdown := `# Test Plan

## Task 1: Backend API Task

**File(s)**: ` + "`internal/api/handler.go`" + `
**Depends on**: None
**Estimated time**: 2h
**Agent**: godev
**WorktreeGroup**: backend-api

### Implementation
Create API handler for user endpoints.
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(plan.Tasks))
	}

	task := plan.Tasks[0]
	if task.Number != "1" {
		t.Errorf("Expected task number '1', got '%s'", task.Number)
	}
	if task.Name != "Backend API Task" {
		t.Errorf("Expected task name 'Backend API Task', got '%s'", task.Name)
	}
	if task.Agent != "godev" {
		t.Errorf("Expected agent 'godev', got '%s'", task.Agent)
	}
	if task.WorktreeGroup != "backend-api" {
		t.Errorf("Expected WorktreeGroup 'backend-api', got '%s'", task.WorktreeGroup)
	}
	if task.EstimatedTime != 2*time.Hour {
		t.Errorf("Expected time 2h, got %v", task.EstimatedTime)
	}
}

func TestParseMarkdownWithQCInvalidMode(t *testing.T) {
	markdown := `---
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "INVALID"
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`

	parser := NewMarkdownParser()
	_, err := parser.Parse(strings.NewReader(markdown))
	if err == nil {
		t.Error("Expected error for invalid QC agents mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid QC agents mode") {
		t.Errorf("Expected error about invalid mode, got: %v", err)
	}
}

func TestParseMarkdownWithQCExplicitModeNoList(t *testing.T) {
	markdown := `---
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "explicit"
      explicit_list: []
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`

	parser := NewMarkdownParser()
	_, err := parser.Parse(strings.NewReader(markdown))
	if err == nil {
		t.Error("Expected error for explicit mode with empty explicit_list, got nil")
	}
	if !strings.Contains(err.Error(), "explicit mode requires non-empty explicit_list") {
		t.Errorf("Expected error about explicit_list requirement, got: %v", err)
	}
}

func TestParseMarkdownWithQCModeCaseNormalization(t *testing.T) {
	tests := []struct {
		name       string
		modeInput  string
		expectPass bool
		withList   bool
	}{
		{"auto lowercase", "auto", true, false},
		{"auto uppercase", "AUTO", true, false},
		{"auto mixed case", "Auto", true, false},
		{"explicit lowercase", "explicit", true, true},
		{"explicit uppercase", "EXPLICIT", true, true},
		{"mixed lowercase", "mixed", true, false},
		{"invalid mode", "INVALID", false, false},
		{"spaces trimmed", "  auto  ", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := `---
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "` + tt.modeInput + `"`
			if tt.withList {
				markdown += `
      explicit_list:
        - agent1
        - agent2`
			}
			markdown += `
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`

			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(markdown))

			if tt.expectPass {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				// Mode should be normalized to lowercase
				expectedMode := strings.ToLower(strings.TrimSpace(tt.modeInput))
				if plan.QualityControl.Agents.Mode != expectedMode {
					t.Errorf("Expected normalized mode %q, got %q", expectedMode, plan.QualityControl.Agents.Mode)
				}
			} else {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			}
		})
	}
}

func TestParseMarkdownWithQCAgentConfig(t *testing.T) {
	tests := []struct {
		name               string
		markdown           string
		expectedMode       string
		expectedExplicit   []string
		expectedAdditional []string
		expectedBlocked    []string
	}{
		{
			name: "markdown with QC auto mode",
			markdown: `---
conductor:
  quality_control:
    enabled: true
    review_agent: "quality-control"
    retry_on_red: 2
    agents:
      mode: "auto"
      blocked: ["deprecated-agent"]
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectedMode:       "auto",
			expectedBlocked:    []string{"deprecated-agent"},
			expectedExplicit:   []string{},
			expectedAdditional: []string{},
		},
		{
			name: "markdown with QC explicit mode",
			markdown: `---
conductor:
  quality_control:
    enabled: true
    retry_on_red: 3
    agents:
      mode: "explicit"
      explicit_list:
        - golang-pro
        - code-reviewer
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectedMode:       "explicit",
			expectedExplicit:   []string{"golang-pro", "code-reviewer"},
			expectedAdditional: []string{},
			expectedBlocked:    []string{},
		},
		{
			name: "markdown with QC mixed mode",
			markdown: `---
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "mixed"
      additional:
        - security-auditor
        - performance-reviewer
      blocked:
        - old-agent
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectedMode:       "mixed",
			expectedExplicit:   []string{},
			expectedAdditional: []string{"security-auditor", "performance-reviewer"},
			expectedBlocked:    []string{"old-agent"},
		},
		{
			name: "markdown without agents section (backward compatibility)",
			markdown: `---
conductor:
  quality_control:
    enabled: true
    review_agent: "quality-control"
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectedMode:       "",
			expectedExplicit:   []string{},
			expectedAdditional: []string{},
			expectedBlocked:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			// Verify QC configuration
			if plan.QualityControl.Agents.Mode != tt.expectedMode {
				t.Errorf("Expected mode %q, got %q", tt.expectedMode, plan.QualityControl.Agents.Mode)
			}

			// Check explicit list
			if len(plan.QualityControl.Agents.ExplicitList) != len(tt.expectedExplicit) {
				t.Errorf("Expected %d explicit agents, got %d", len(tt.expectedExplicit), len(plan.QualityControl.Agents.ExplicitList))
			}
			for i, agent := range tt.expectedExplicit {
				if i >= len(plan.QualityControl.Agents.ExplicitList) {
					break
				}
				if plan.QualityControl.Agents.ExplicitList[i] != agent {
					t.Errorf("Expected explicit agent %d to be %q, got %q", i, agent, plan.QualityControl.Agents.ExplicitList[i])
				}
			}

			// Check additional agents
			if len(plan.QualityControl.Agents.AdditionalAgents) != len(tt.expectedAdditional) {
				t.Errorf("Expected %d additional agents, got %d", len(tt.expectedAdditional), len(plan.QualityControl.Agents.AdditionalAgents))
			}
			for i, agent := range tt.expectedAdditional {
				if i >= len(plan.QualityControl.Agents.AdditionalAgents) {
					break
				}
				if plan.QualityControl.Agents.AdditionalAgents[i] != agent {
					t.Errorf("Expected additional agent %d to be %q, got %q", i, agent, plan.QualityControl.Agents.AdditionalAgents[i])
				}
			}

			// Check blocked agents
			if len(plan.QualityControl.Agents.BlockedAgents) != len(tt.expectedBlocked) {
				t.Errorf("Expected %d blocked agents, got %d", len(tt.expectedBlocked), len(plan.QualityControl.Agents.BlockedAgents))
			}
			for i, agent := range tt.expectedBlocked {
				if i >= len(plan.QualityControl.Agents.BlockedAgents) {
					break
				}
				if plan.QualityControl.Agents.BlockedAgents[i] != agent {
					t.Errorf("Expected blocked agent %d to be %q, got %q", i, agent, plan.QualityControl.Agents.BlockedAgents[i])
				}
			}
		})
	}
}

func TestParseMarkdownCrossFileDependencies(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		expectedDependsOn []string
	}{
		{
			name: "simple cross-file dependency with slash notation",
			content: `**File(s)**: ` + "`internal/feature/feature.go`" + `
**Depends on**: file:plan-01-setup.md/task:2
**Estimated time**: 1h`,
			expectedDependsOn: []string{"file:plan-01-setup.md:task:2"},
		},
		{
			name: "simple cross-file dependency with colon notation",
			content: `**File(s)**: ` + "`internal/feature/feature.go`" + `
**Depends on**: file:plan-01-setup.yaml:task:2
**Estimated time**: 1h`,
			expectedDependsOn: []string{"file:plan-01-setup.yaml:task:2"},
		},
		{
			name: "mixed numeric and cross-file dependencies",
			content: `**File(s)**: ` + "`cmd/api/router.go`" + `
**Depends on**: Task 1, file:plan-01-foundation.yaml/task:2, 3
**Estimated time**: 2h`,
			expectedDependsOn: []string{"1", "file:plan-01-foundation.yaml:task:2", "3"},
		},
		{
			name: "multiple cross-file dependencies",
			content: `**File(s)**: ` + "`internal/integration/integration.go`" + `
**Depends on**: file:plan-01-foundation.yaml:task:1, file:plan-02-auth.yaml:task:2, file:plan-03-api.yaml:task:4
**Estimated time**: 3h`,
			expectedDependsOn: []string{"file:plan-01-foundation.yaml:task:1", "file:plan-02-auth.yaml:task:2", "file:plan-03-api.yaml:task:4"},
		},
		{
			name: "cross-file with alphanumeric task numbers",
			content: `**File(s)**: ` + "`internal/feature/feature.go`" + `
**Depends on**: file:plan-integration.yaml:task:integration-1
**Estimated time**: 1h30m`,
			expectedDependsOn: []string{"file:plan-integration.yaml:task:integration-1"},
		},
		{
			name: "task notation with cross-file",
			content: `**File(s)**: ` + "`internal/api/api.go`" + `
**Depends on**: Task 1, file:plan-02.md/task:3, Task 4
**Estimated time**: 2h`,
			expectedDependsOn: []string{"1", "file:plan-02.md:task:3", "4"},
		},
		{
			name: "backward compatibility - numeric only",
			content: `**File(s)**: ` + "`file.go`" + `
**Depends on**: 1, 2, 3
**Estimated time**: 1h`,
			expectedDependsOn: []string{"1", "2", "3"},
		},
		{
			name: "backward compatibility - task notation",
			content: `**File(s)**: ` + "`file.go`" + `
**Depends on**: Task 1, Task 2
**Estimated time**: 1h`,
			expectedDependsOn: []string{"1", "2"},
		},
		{
			name: "cross-file with whitespace",
			content: `**File(s)**: ` + "`file.go`" + `
**Depends on**: file:plan-01.yaml / task:2, 3
**Estimated time**: 1h`,
			expectedDependsOn: []string{"file:plan-01.yaml:task:2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{}
			parseTaskMetadata(task, tt.content)

			// Verify dependencies
			if len(task.DependsOn) != len(tt.expectedDependsOn) {
				t.Errorf("expected %d dependencies, got %d", len(tt.expectedDependsOn), len(task.DependsOn))
				t.Logf("Expected: %v", tt.expectedDependsOn)
				t.Logf("Got: %v", task.DependsOn)
			}

			for i, expected := range tt.expectedDependsOn {
				if i >= len(task.DependsOn) {
					break
				}
				if task.DependsOn[i] != expected {
					t.Errorf("dependency %d: expected %q, got %q", i, expected, task.DependsOn[i])
				}
			}
		})
	}
}

func TestInjectFilesIntoPrompt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		files   []string
		wantHas []string
		wantNot []string
	}{
		{
			name:    "no files - no injection",
			content: "Original content",
			files:   nil,
			wantHas: []string{"Original content"},
			wantNot: []string{"Target Files", "MUST create"},
		},
		{
			name:    "empty files - no injection",
			content: "Original content",
			files:   []string{},
			wantHas: []string{"Original content"},
			wantNot: []string{"Target Files", "MUST create"},
		},
		{
			name:    "single file - injection",
			content: "Original content",
			files:   []string{"src/main.go"},
			wantHas: []string{
				"Target Files (REQUIRED)",
				"MUST create/modify these exact files",
				"`src/main.go`",
				"Do NOT create files with different names",
				"Original content",
			},
			wantNot: nil,
		},
		{
			name:    "multiple files - injection",
			content: "Task description here",
			files:   []string{"file1.go", "file2.go", "pkg/util.go"},
			wantHas: []string{
				"Target Files (REQUIRED)",
				"`file1.go`",
				"`file2.go`",
				"`pkg/util.go`",
				"Task description here",
			},
			wantNot: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectFilesIntoPrompt(tt.content, tt.files)

			for _, want := range tt.wantHas {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNot {
				if strings.Contains(result, notWant) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

// =============================================================================
// Runtime Enforcement Tests - Markdown Parity (v2.9+)
// =============================================================================

func TestMarkdown_PlannerComplianceSpec(t *testing.T) {
	markdown := `---
planner_compliance:
  planner_version: "1.2.0"
  strict_enforcement: false
  required_features:
    - dependency_checks
    - documentation_targets
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if plan.PlannerCompliance == nil {
		t.Fatal("expected PlannerCompliance to be populated")
	}

	if plan.PlannerCompliance.PlannerVersion != "1.2.0" {
		t.Errorf("expected planner_version '1.2.0', got %q", plan.PlannerCompliance.PlannerVersion)
	}

	if plan.PlannerCompliance.StrictEnforcement {
		t.Error("expected strict_enforcement to be false")
	}

	if len(plan.PlannerCompliance.RequiredFeatures) != 2 {
		t.Errorf("expected 2 required_features, got %d", len(plan.PlannerCompliance.RequiredFeatures))
	}
}

func TestMarkdown_PlannerComplianceMissingVersion(t *testing.T) {
	markdown := `---
planner_compliance:
  strict_enforcement: true
---

# Test Plan

## Task 1: Test Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`

	parser := NewMarkdownParser()
	_, err := parser.Parse(strings.NewReader(markdown))
	if err == nil {
		t.Error("expected error for missing planner_version")
	}
	if !strings.Contains(err.Error(), "planner_version") {
		t.Errorf("expected error about planner_version, got: %v", err)
	}
}

func TestMarkdown_LegacyPlanNoPlannerCompliance(t *testing.T) {
	markdown := `---
conductor:
  default_agent: "golang-pro"
---

# Test Plan

## Task 1: Legacy Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.PlannerCompliance != nil {
		t.Error("expected nil PlannerCompliance for legacy plan")
	}
}

func TestMarkdown_SuccessCriteria(t *testing.T) {
	tests := []struct {
		name              string
		markdown          string
		expectedCriteria  []string
	}{
		{
			name: "task with success criteria",
			markdown: `# Test Plan

## Task 1: Task with Criteria

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Success Criteria**:
- Function returns correct value
- Error handling works for edge cases
- Performance meets requirements
`,
			expectedCriteria: []string{
				"Function returns correct value",
				"Error handling works for edge cases",
				"Performance meets requirements",
			},
		},
		{
			name: "task with multi-line success criteria",
			markdown: `# Test Plan

## Task 1: Task with Multi-line Criteria

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Success Criteria**:
- Long criterion that spans
  multiple lines with proper indentation
- Another criterion
`,
			expectedCriteria: []string{
				"Long criterion that spans multiple lines with proper indentation",
				"Another criterion",
			},
		},
		{
			name: "task without success criteria",
			markdown: `# Test Plan

## Task 1: Task without Criteria

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectedCriteria: []string{},
		},
		{
			name: "task with empty success criteria section",
			markdown: `# Test Plan

## Task 1: Task with Empty Criteria

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Success Criteria**:

**Status**: pending
`,
			expectedCriteria: []string{},
		},
		{
			name: "task with criteria followed by other sections",
			markdown: `# Test Plan

## Task 1: Task with Multiple Sections

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Success Criteria**:
- Criteria 1
- Criteria 2

**Test Commands**:
` + "```bash" + `
go test ./...
` + "```" + `
`,
			expectedCriteria: []string{
				"Criteria 1",
				"Criteria 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if len(plan.Tasks) == 0 {
				t.Fatal("expected at least 1 task")
			}

			task := plan.Tasks[0]

			// Check criteria count
			if len(task.SuccessCriteria) != len(tt.expectedCriteria) {
				t.Errorf("expected %d criteria, got %d", len(tt.expectedCriteria), len(task.SuccessCriteria))
				t.Logf("Expected: %v", tt.expectedCriteria)
				t.Logf("Got: %v", task.SuccessCriteria)
			}

			// Check each criterion
			for i, expected := range tt.expectedCriteria {
				if i >= len(task.SuccessCriteria) {
					break
				}
				if task.SuccessCriteria[i] != expected {
					t.Errorf("criterion %d: expected %q, got %q", i, expected, task.SuccessCriteria[i])
				}
			}
		})
	}
}

func TestParseSuccessCriteria_ExtractsBulletListItems(t *testing.T) {
	// Success criteria #0: parseSuccessCriteria() extracts bullet list items
	content := `**Success Criteria**:
- First criterion
- Second criterion
- Third criterion

**Status**: pending`

	criteria := parseSuccessCriteriaMarkdown(content)

	if len(criteria) != 3 {
		t.Errorf("expected 3 criteria, got %d", len(criteria))
	}

	expectedCriteria := []string{
		"First criterion",
		"Second criterion",
		"Third criterion",
	}

	for i, expected := range expectedCriteria {
		if i >= len(criteria) {
			break
		}
		if criteria[i] != expected {
			t.Errorf("criterion %d: expected %q, got %q", i, expected, criteria[i])
		}
	}
}

func TestParseSuccessCriteria_HandlesMultilineCriteria(t *testing.T) {
	content := `**Success Criteria**:
- First criterion spanning
  multiple lines with indentation
- Second criterion
  continued on next line
  and another line
- Single line criterion

**Test Commands**:`

	criteria := parseSuccessCriteriaMarkdown(content)

	if len(criteria) != 3 {
		t.Errorf("expected 3 criteria, got %d", len(criteria))
		t.Logf("Got: %v", criteria)
	}

	if len(criteria) > 0 && !strings.Contains(criteria[0], "First criterion spanning") {
		t.Errorf("first criterion should contain 'First criterion spanning', got %q", criteria[0])
	}
}

func TestParseSuccessCriteria_EmptySection(t *testing.T) {
	content := `**Success Criteria**:

**Status**: pending`

	criteria := parseSuccessCriteriaMarkdown(content)

	if len(criteria) != 0 {
		t.Errorf("expected empty criteria, got %v", criteria)
	}
}

func TestParseSuccessCriteria_NotFound(t *testing.T) {
	content := `**File(s)**: test.go
**Depends on**: None
**Estimated time**: 30m

### Implementation
No success criteria section here`

	criteria := parseSuccessCriteriaMarkdown(content)

	if len(criteria) != 0 {
		t.Errorf("expected empty criteria when section not found, got %v", criteria)
	}
}

// =============================================================================
// Test Commands Parsing Tests
// =============================================================================

func TestParseTestCommands_CodeBlockFormat(t *testing.T) {
	tests := []struct {
		name             string
		markdown         string
		expectedCommands []string
	}{
		{
			name: "code block with single command",
			markdown: `# Test Plan

## Task 1: Task with Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
` + "```bash" + `
go test ./...
` + "```" + `
`,
			expectedCommands: []string{"go test ./..."},
		},
		{
			name: "code block with multiple commands",
			markdown: `# Test Plan

## Task 1: Task with Multiple Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
` + "```bash" + `
npm test
npm run build
npm run lint
` + "```" + `
`,
			expectedCommands: []string{"npm test", "npm run build", "npm run lint"},
		},
		{
			name: "code block with empty lines",
			markdown: `# Test Plan

## Task 1: Task with Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
` + "```bash" + `
go test ./...

go test -race ./...
` + "```" + `
`,
			expectedCommands: []string{"go test ./...", "go test -race ./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if len(plan.Tasks) == 0 {
				t.Fatal("expected at least 1 task")
			}

			task := plan.Tasks[0]

			// Check command count
			if len(task.TestCommands) != len(tt.expectedCommands) {
				t.Errorf("expected %d commands, got %d", len(tt.expectedCommands), len(task.TestCommands))
				t.Logf("Expected: %v", tt.expectedCommands)
				t.Logf("Got: %v", task.TestCommands)
			}

			// Check each command
			for i, expected := range tt.expectedCommands {
				if i >= len(task.TestCommands) {
					break
				}
				if task.TestCommands[i] != expected {
					t.Errorf("command %d: expected %q, got %q", i, expected, task.TestCommands[i])
				}
			}
		})
	}
}

func TestParseTestCommands_BulletListFormat(t *testing.T) {
	tests := []struct {
		name             string
		markdown         string
		expectedCommands []string
	}{
		{
			name: "bullet list with single command",
			markdown: `# Test Plan

## Task 1: Task with Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
- go test ./...
`,
			expectedCommands: []string{"go test ./..."},
		},
		{
			name: "bullet list with multiple commands",
			markdown: `# Test Plan

## Task 1: Task with Multiple Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
- npm test
- npm run build
- npm run lint
`,
			expectedCommands: []string{"npm test", "npm run build", "npm run lint"},
		},
		{
			name: "bullet list followed by other section",
			markdown: `# Test Plan

## Task 1: Task with Commands and More

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
- go test ./...
- go test -race ./...

**Success Criteria**:
- Test passes
`,
			expectedCommands: []string{"go test ./...", "go test -race ./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if len(plan.Tasks) == 0 {
				t.Fatal("expected at least 1 task")
			}

			task := plan.Tasks[0]

			// Check command count
			if len(task.TestCommands) != len(tt.expectedCommands) {
				t.Errorf("expected %d commands, got %d", len(tt.expectedCommands), len(task.TestCommands))
				t.Logf("Expected: %v", tt.expectedCommands)
				t.Logf("Got: %v", task.TestCommands)
			}

			// Check each command
			for i, expected := range tt.expectedCommands {
				if i >= len(task.TestCommands) {
					break
				}
				if task.TestCommands[i] != expected {
					t.Errorf("command %d: expected %q, got %q", i, expected, task.TestCommands[i])
				}
			}
		})
	}
}

func TestParseTestCommands_EmptySection(t *testing.T) {
	markdown := `# Test Plan

## Task 1: Task with Empty Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:

**Status**: pending
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) == 0 {
		t.Fatal("expected at least 1 task")
	}

	task := plan.Tasks[0]

	if len(task.TestCommands) != 0 {
		t.Errorf("expected empty test commands, got %v", task.TestCommands)
	}
}

func TestParseTestCommands_NotFound(t *testing.T) {
	markdown := `# Test Plan

## Task 1: Task without Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

### Implementation
No test commands section here
`

	parser := NewMarkdownParser()
	plan, err := parser.Parse(strings.NewReader(markdown))
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if len(plan.Tasks) == 0 {
		t.Fatal("expected at least 1 task")
	}

	task := plan.Tasks[0]

	if len(task.TestCommands) != 0 {
		t.Errorf("expected empty test commands when section not found, got %v", task.TestCommands)
	}
}

func TestParseTestCommands_DirectFunction(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedCommands []string
	}{
		{
			name: "code block format",
			content: `**Test Commands**:
` + "```bash" + `
go test ./...
go test -race ./...
` + "```" + ``,
			expectedCommands: []string{"go test ./...", "go test -race ./..."},
		},
		{
			name: "bullet list format",
			content: `**Test Commands**:
- npm test
- npm run build`,
			expectedCommands: []string{"npm test", "npm run build"},
		},
		{
			name: "empty content",
			content: `**Test Commands**:

**Status**: pending`,
			expectedCommands: []string{},
		},
		{
			name: "section not found",
			content: `**File(s)**: test.go
**Depends on**: None`,
			expectedCommands: []string{},
		},
		{
			name: "code block with mixed language identifier",
			content: `**Test Commands**:
` + "```go" + `
go test ./...
` + "```" + ``,
			expectedCommands: []string{"go test ./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parseTestCommands(tt.content)

			if len(commands) != len(tt.expectedCommands) {
				t.Errorf("expected %d commands, got %d", len(tt.expectedCommands), len(commands))
				t.Logf("Expected: %v", tt.expectedCommands)
				t.Logf("Got: %v", commands)
			}

			for i, expected := range tt.expectedCommands {
				if i >= len(commands) {
					break
				}
				if commands[i] != expected {
					t.Errorf("command %d: expected %q, got %q", i, expected, commands[i])
				}
			}
		})
	}
}

func TestMarkdown_TestCommands(t *testing.T) {
	tests := []struct {
		name             string
		markdown         string
		expectedCommands []string
	}{
		{
			name: "task with test commands in code block",
			markdown: `# Test Plan

## Task 1: Task with Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
` + "```bash" + `
go test ./...
go test ./... -race
` + "```" + `
`,
			expectedCommands: []string{"go test ./...", "go test ./... -race"},
		},
		{
			name: "task with test commands in bullet list",
			markdown: `# Test Plan

## Task 1: Task with Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
- npm test
- npm run build
- npm run lint
`,
			expectedCommands: []string{"npm test", "npm run build", "npm run lint"},
		},
		{
			name: "task without test commands",
			markdown: `# Test Plan

## Task 1: Task without Test Commands

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m
`,
			expectedCommands: []string{},
		},
		{
			name: "task with test commands followed by success criteria",
			markdown: `# Test Plan

## Task 1: Complete Task

**File(s)**: ` + "`test.go`" + `
**Depends on**: None
**Estimated time**: 30m

**Test Commands**:
- go test ./...

**Success Criteria**:
- Tests pass
- Code builds
`,
			expectedCommands: []string{"go test ./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			plan, err := parser.Parse(strings.NewReader(tt.markdown))
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}

			if len(plan.Tasks) == 0 {
				t.Fatal("expected at least 1 task")
			}

			task := plan.Tasks[0]

			if len(task.TestCommands) != len(tt.expectedCommands) {
				t.Errorf("expected %d commands, got %d", len(tt.expectedCommands), len(task.TestCommands))
				t.Logf("Expected: %v", tt.expectedCommands)
				t.Logf("Got: %v", task.TestCommands)
			}

			for i, expected := range tt.expectedCommands {
				if i >= len(task.TestCommands) {
					break
				}
				if task.TestCommands[i] != expected {
					t.Errorf("command %d: expected %q, got %q", i, expected, task.TestCommands[i])
				}
			}
		})
	}
}
