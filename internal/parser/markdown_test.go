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
	if task.Number != 1 {
		t.Errorf("Expected task number 1, got %d", task.Number)
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
		expectedDeps  []int
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
			expectedDeps:  []int{1, 2},
			expectedTime:  2 * time.Hour,
			expectedAgent: "godev",
		},
		{
			name: "no dependencies",
			content: `**File(s)**: ` + "`file1.go`" + `
**Depends on**: None
**Estimated time**: 30m`,
			expectedFiles: []string{"file1.go"},
			expectedDeps:  []int{},
			expectedTime:  30 * time.Minute,
			expectedAgent: "",
		},
		{
			name: "dependencies with numbers only",
			content: `**File(s)**: ` + "`file1.go`" + `
**Depends on**: 3, 5
**Estimated time**: 1h`,
			expectedFiles: []string{"file1.go"},
			expectedDeps:  []int{3, 5},
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
			expectedDeps:  []int{3},
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
					t.Errorf("Dependency %d: expected %d, got %d", i, expected, task.DependsOn[i])
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
