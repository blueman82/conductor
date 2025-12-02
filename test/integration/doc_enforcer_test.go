package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

// TestDocEnforcerIntegration_CorrectSection verifies enforcement passes when
// documentation exists with the correct section.
func TestDocEnforcerIntegration_CorrectSection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a realistic documentation structure
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create CLI documentation with proper sections
	cliDocPath := filepath.Join(docsDir, "CLI.md")
	cliContent := `# CLI Documentation

## Overview
Conductor is a multi-agent orchestration CLI.

## Installation
Install using go install:
` + "```bash\ngo install github.com/harrison/conductor/cmd/conductor@latest\n```\n\n" + `
## Run Command
The run command executes implementation plans.

### Usage
` + "```bash\nconductor run plan.yaml\n```\n\n" + `
### Flags
- --dry-run: Validate without execution
- --max-concurrency: Limit parallel tasks

## Validate Command
The validate command checks plan files for errors.
`
	if err := os.WriteFile(cliDocPath, []byte(cliContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "1",
		Name:   "Update CLI Run Command docs",
		Prompt: "Update the Run Command section with new flags",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: cliDocPath, Section: "## Run Command"},
			},
		},
	}

	results, err := executor.VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected verification to pass for existing section, got error: %v", results[0].Error)
	}

	// Verify content was captured
	if results[0].Content == "" {
		t.Error("Expected content to be captured")
	}

	if !strings.Contains(results[0].Content, "Run Command") {
		t.Error("Expected content to contain section heading")
	}

	// Criterion #1: LineNumber must be set for file:line context in QC prompt
	if results[0].LineNumber == 0 {
		t.Error("Expected LineNumber to be set for file:line context")
	}
	// "## Run Command" appears on line 13 in the content
	if results[0].LineNumber != 13 {
		t.Errorf("Expected LineNumber to be 13, got %d", results[0].LineNumber)
	}
}

// TestDocEnforcerIntegration_MissingSection verifies enforcement fails when
// the expected section is not present in the documentation.
func TestDocEnforcerIntegration_MissingSection(t *testing.T) {
	tmpDir := t.TempDir()

	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create doc WITHOUT the expected section
	apiDocPath := filepath.Join(docsDir, "API.md")
	apiContent := `# API Documentation

## Endpoints
List of API endpoints.

## Authentication
How to authenticate.
`
	if err := os.WriteFile(apiDocPath, []byte(apiContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "2",
		Name:   "Add Rate Limiting section",
		Prompt: "Add rate limiting documentation",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: apiDocPath, Section: "## Rate Limiting"},
			},
		},
	}

	results, err := executor.VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Error("Expected verification to FAIL for missing section")
	}

	if results[0].Error == nil {
		t.Error("Expected error to be populated")
	}

	// Verify error message is actionable
	errStr := results[0].Error.Error()
	if !strings.Contains(errStr, "Rate Limiting") {
		t.Error("Expected error to mention the missing section")
	}

	if !strings.Contains(errStr, "not found") {
		t.Error("Expected error to indicate section was not found")
	}
}

// TestDocEnforcerIntegration_MultipleTargets tests verification of multiple
// documentation targets in a single task.
func TestDocEnforcerIntegration_MultipleTargets(t *testing.T) {
	tmpDir := t.TempDir()

	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create first doc with correct section
	readmePath := filepath.Join(tmpDir, "README.md")
	readmeContent := `# Project README

## Quick Start
Get started quickly with these steps.

## Contributing
How to contribute to the project.
`
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second doc WITHOUT expected section
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")
	changelogContent := `# Changelog

## [2.0.0] - 2024-01-01
- Major release

## [1.0.0] - 2023-06-01
- Initial release
`
	if err := os.WriteFile(changelogPath, []byte(changelogContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "3",
		Name:   "Update project documentation",
		Prompt: "Update README and add v2.1.0 to changelog",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: readmePath, Section: "## Quick Start"},
				{Location: changelogPath, Section: "## [2.1.0]"}, // Missing
			},
		},
	}

	results, err := executor.VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// First target should pass
	if !results[0].Passed {
		t.Errorf("Expected first target (Quick Start) to pass, got error: %v", results[0].Error)
	}

	// Second target should fail
	if results[1].Passed {
		t.Error("Expected second target (v2.1.0) to fail")
	}
}

// TestDocEnforcerIntegration_FormatOutput tests that formatted output is QC-friendly.
func TestDocEnforcerIntegration_FormatOutput(t *testing.T) {
	results := []executor.DocTargetResult{
		{
			Location: "docs/CLI.md",
			Section:  "## Run",
			Passed:   true,
			Content:  "## Run\nThe run command...",
		},
		{
			Location: "docs/API.md",
			Section:  "## Rate Limiting",
			Passed:   false,
			Error:    executor.ErrDocSectionNotFound,
		},
	}

	formatted := executor.FormatDocTargetResults(results)

	// Should be non-empty
	if formatted == "" {
		t.Fatal("Expected non-empty formatted output")
	}

	// Should contain section header
	if !strings.Contains(formatted, "DOCUMENTATION TARGET VERIFICATION") {
		t.Error("Expected formatted output to contain section header")
	}

	// Should show file paths
	if !strings.Contains(formatted, "docs/CLI.md") {
		t.Error("Expected formatted output to show file paths")
	}

	// Should show PASS/FAIL status
	if !strings.Contains(formatted, "PASS") || !strings.Contains(formatted, "FAIL") {
		t.Error("Expected formatted output to show PASS/FAIL status")
	}

	// Should show summary
	if !strings.Contains(formatted, "1/2") {
		t.Error("Expected formatted output to show pass count (1/2)")
	}
}

// TestDocEnforcerIntegration_NestedSections tests detection of nested headings.
func TestDocEnforcerIntegration_NestedSections(t *testing.T) {
	tmpDir := t.TempDir()

	docPath := filepath.Join(tmpDir, "ARCHITECTURE.md")
	content := `# Architecture

## Overview
High-level architecture overview.

## Components

### Parser
The parser component handles plan file parsing.

#### Markdown Parser
Handles .md files.

#### YAML Parser
Handles .yaml files.

### Executor
The executor runs tasks.
`
	if err := os.WriteFile(docPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "4",
		Name:   "Update Parser docs",
		Prompt: "Document the new parser features",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: docPath, Section: "### Parser"},
			},
		},
	}

	results, err := executor.VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected verification to pass for nested section, got error: %v", results[0].Error)
	}
}

// TestDocEnforcerIntegration_NonExistentFile tests behavior when doc file doesn't exist.
func TestDocEnforcerIntegration_NonExistentFile(t *testing.T) {
	task := models.Task{
		Number: "5",
		Name:   "Update non-existent docs",
		Prompt: "This should fail",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: "/this/path/does/not/exist/docs.md", Section: "## Any"},
			},
		},
	}

	results, err := executor.VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Error("Expected verification to fail for non-existent file")
	}

	// Error should indicate file not found
	if results[0].Error == nil {
		t.Fatal("Expected error to be populated")
	}

	errStr := results[0].Error.Error()
	if !strings.Contains(errStr, "not found") && !strings.Contains(errStr, "no such file") {
		t.Errorf("Expected error about file not found, got: %s", errStr)
	}
}

// TestDocEnforcerIntegration_GracefulSkip tests that non-doc tasks skip verification.
func TestDocEnforcerIntegration_GracefulSkip(t *testing.T) {
	// Task without RuntimeMetadata - should skip gracefully
	task := models.Task{
		Number: "6",
		Name:   "Regular implementation task",
		Prompt: "Implement the feature",
	}

	results, err := executor.VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Fatalf("Expected no error for non-doc task, got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for task without doc targets, got %d", len(results))
	}
}

// TestDocEnforcerIntegration_FormatWithLineNumbers tests file:line format in QC output.
func TestDocEnforcerIntegration_FormatWithLineNumbers(t *testing.T) {
	results := []executor.DocTargetResult{
		{
			Location:   "docs/CLI.md",
			Section:    "## Run",
			LineNumber: 42, // Criterion #1: file:line context
			Passed:     true,
			Content:    "## Run\nThe run command...",
		},
		{
			Location:   "docs/API.md",
			Section:    "## Auth",
			LineNumber: 0, // Not found - no line number
			Passed:     false,
			Error:      executor.ErrDocSectionNotFound,
		},
	}

	formatted := executor.FormatDocTargetResults(results)

	// Should contain file:line format for passing result
	if !strings.Contains(formatted, "docs/CLI.md:42") {
		t.Errorf("Expected file:line format (docs/CLI.md:42) in output, got:\n%s", formatted)
	}

	// Should NOT contain :0 for failed result
	if strings.Contains(formatted, ":0") {
		t.Errorf("Should not include :0 for failed result, got:\n%s", formatted)
	}
}

// TestDocEnforcerIntegration_DocTaskMissingTargets tests criterion #3:
// Documentation tasks without targets MUST fail validation (no graceful skip).
func TestDocEnforcerIntegration_DocTaskMissingTargets(t *testing.T) {
	// Documentation task with nil RuntimeMetadata
	task1 := models.Task{
		Number: "1",
		Name:   "Doc task missing metadata",
		Type:   "documentation",
	}
	if !executor.IsDocumentationTask(task1) {
		t.Error("Expected IsDocumentationTask to return true")
	}
	if executor.HasDocumentationTargets(task1) {
		t.Error("Expected HasDocumentationTargets to return false")
	}

	// Documentation task with empty DocumentationTargets
	task2 := models.Task{
		Number:          "2",
		Name:            "Doc task empty targets",
		Type:            "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{},
	}
	if !executor.IsDocumentationTask(task2) {
		t.Error("Expected IsDocumentationTask to return true")
	}
	if executor.HasDocumentationTargets(task2) {
		t.Error("Expected HasDocumentationTargets to return false")
	}

	// The validation failure happens in task executor, not VerifyDocumentationTargets
	// This test validates the helper functions work correctly for the executor check
}

// TestDocEnforcerIntegration_HasDocumentationTargets tests the helper function.
func TestDocEnforcerIntegration_HasDocumentationTargets(t *testing.T) {
	tests := []struct {
		name     string
		task     models.Task
		expected bool
	}{
		{
			name:     "nil metadata",
			task:     models.Task{Number: "1"},
			expected: false,
		},
		{
			name: "empty metadata",
			task: models.Task{
				Number:          "2",
				RuntimeMetadata: &models.TaskMetadataRuntime{},
			},
			expected: false,
		},
		{
			name: "with targets",
			task: models.Task{
				Number: "3",
				RuntimeMetadata: &models.TaskMetadataRuntime{
					DocumentationTargets: []models.DocumentationTarget{
						{Location: "docs/CLI.md", Section: "## Run"},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.HasDocumentationTargets(tt.task)
			if result != tt.expected {
				t.Errorf("HasDocumentationTargets() = %v, want %v", result, tt.expected)
			}
		})
	}
}
