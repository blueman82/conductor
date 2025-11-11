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
