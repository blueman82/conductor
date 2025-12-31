package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// MockCommandRunnerForDocs is a mock command runner for documentation tests.
type MockCommandRunnerForDocs struct {
	Results map[string]string // command -> output
	Errors  map[string]error  // command -> error
}

func (m *MockCommandRunnerForDocs) Run(ctx context.Context, command string) (string, error) {
	if err, ok := m.Errors[command]; ok {
		return m.Results[command], err
	}
	return m.Results[command], nil
}

// Helper to create a doc task with documentation targets
func docTaskWithTarget(file, section string) models.Task {
	return models.Task{
		Number: "1",
		Name:   "Documentation task",
		Prompt: "Update docs",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: file, Section: section},
			},
		},
	}
}

func TestVerifyDocumentationTargets_Success(t *testing.T) {
	// Create temp dir with test doc
	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "docs", "CLI.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Write doc content with correct section
	docContent := `# CLI Documentation

## Overview
Some overview text.

## Run
The run command executes the plan.
It accepts various flags for configuration.

## Other Section
Other content here.
`
	if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "1",
		Name:   "Update CLI docs",
		Prompt: "Update the Run section",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: docPath, Section: "## Run"},
			},
		},
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected verification to pass, got failure: %s", results[0].Error)
	}

	// Criterion #1: LineNumber must be populated with file:line context
	if results[0].LineNumber == 0 {
		t.Errorf("Expected LineNumber to be set, got 0")
	}
	// "## Run" is on line 6 in the doc content
	if results[0].LineNumber != 6 {
		t.Errorf("Expected LineNumber to be 6, got %d", results[0].LineNumber)
	}
}

func TestVerifyDocumentationTargets_SectionMissing(t *testing.T) {
	// Create temp dir with test doc
	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "docs", "CLI.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Write doc content WITHOUT the expected section
	docContent := `# CLI Documentation

## Overview
Some overview text.

## Other Section
Other content here.
`
	if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "1",
		Name:   "Update CLI docs",
		Prompt: "Update the Run section",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: docPath, Section: "## Run"},
			},
		},
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	// Non-fatal for individual failures - returns results with error details
	if err != nil {
		t.Errorf("Expected no error (verification should complete), got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Errorf("Expected verification to fail for missing section")
	}

	if results[0].Error == nil {
		t.Errorf("Expected error to be populated for failed verification")
	}
}

func TestVerifyDocumentationTargets_FileMissing(t *testing.T) {
	task := models.Task{
		Number: "1",
		Name:   "Update nonexistent doc",
		Prompt: "Update docs",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: "/nonexistent/path/docs.md", Section: "## Run"},
			},
		},
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Errorf("Expected no error (verification should complete), got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Errorf("Expected verification to fail for missing file")
	}
}

func TestVerifyDocumentationTargets_MultipleTargets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first doc with correct section
	doc1Path := filepath.Join(tmpDir, "docs", "CLI.md")
	if err := os.MkdirAll(filepath.Dir(doc1Path), 0755); err != nil {
		t.Fatal(err)
	}
	doc1Content := `# CLI Documentation

## Run
Run section content.
`
	if err := os.WriteFile(doc1Path, []byte(doc1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second doc WITHOUT correct section
	doc2Path := filepath.Join(tmpDir, "docs", "API.md")
	doc2Content := `# API Documentation

## Endpoints
Endpoints content.
`
	if err := os.WriteFile(doc2Path, []byte(doc2Content), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "1",
		Name:   "Multi-doc update",
		Prompt: "Update multiple docs",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: doc1Path, Section: "## Run"},
				{Location: doc2Path, Section: "## Authentication"}, // Missing section
			},
		},
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// First should pass
	if !results[0].Passed {
		t.Errorf("Expected first target to pass")
	}

	// Second should fail
	if results[1].Passed {
		t.Errorf("Expected second target to fail (missing section)")
	}
}

func TestVerifyDocumentationTargets_NoTargets(t *testing.T) {
	// Task without documentation targets
	task := models.Task{
		Number: "1",
		Name:   "Regular task",
		Prompt: "Do something",
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for task without targets, got %d", len(results))
	}
}

func TestVerifyDocumentationTargets_EmptyRuntimeMetadata(t *testing.T) {
	task := models.Task{
		Number:          "1",
		Name:            "Task with empty metadata",
		Prompt:          "Do something",
		RuntimeMetadata: &models.TaskMetadataRuntime{},
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestVerifyDocumentationTargets_RegexSection(t *testing.T) {
	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "README.md")

	docContent := `# Project README

## Installation

### Prerequisites
Install Go 1.21 or later.

### Quick Start
Run make build.

## Usage
How to use the tool.
`
	if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		Number: "1",
		Name:   "Update prerequisites",
		Prompt: "Update prerequisites section",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: docPath, Section: "### Prerequisites"},
			},
		},
	}

	results, err := VerifyDocumentationTargets(context.Background(), task)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected verification to pass for nested section")
	}
}

func TestFormatDocTargetResults_AllPassed(t *testing.T) {
	results := []DocTargetResult{
		{
			Location:   "docs/CLI.md",
			Section:    "## Run",
			LineNumber: 56, // Criterion #1: file:line context
			Passed:     true,
			Content:    "Some content found",
		},
		{
			Location:   "docs/API.md",
			Section:    "## Endpoints",
			LineNumber: 23,
			Passed:     true,
			Content:    "Endpoint docs",
		},
	}

	formatted := FormatDocTargetResults(results)

	if formatted == "" {
		t.Error("Expected non-empty formatted output")
	}

	// Should contain success indicators (XML format uses status="found")
	if !strings.Contains(formatted, `status="found"`) {
		t.Error("Expected status=\"found\" indicators in output")
	}

	// Should contain file attributes in XML format
	if !strings.Contains(formatted, `file="docs/CLI.md"`) {
		t.Errorf("Expected file attribute in output, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, `line="56"`) {
		t.Errorf("Expected line attribute in output, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, `file="docs/API.md"`) {
		t.Errorf("Expected file attribute for API.md in output, got:\n%s", formatted)
	}

	// Should indicate all passed
	if !strings.Contains(formatted, "All") {
		t.Error("Expected summary indicating all passed")
	}
}

func TestFormatDocTargetResults_SomeFailed(t *testing.T) {
	results := []DocTargetResult{
		{
			Location: "docs/CLI.md",
			Section:  "## Run",
			Passed:   true,
			Content:  "Content found",
		},
		{
			Location: "docs/API.md",
			Section:  "## Authentication",
			Passed:   false,
			Error:    ErrDocSectionNotFound,
		},
	}

	formatted := FormatDocTargetResults(results)

	if formatted == "" {
		t.Error("Expected non-empty formatted output")
	}

	// Should contain both found and missing statuses (XML format)
	if !strings.Contains(formatted, `status="found"`) || !strings.Contains(formatted, `status="missing"`) {
		t.Error("Expected both found and missing status indicators")
	}
}

func TestFormatDocTargetResults_Empty(t *testing.T) {
	results := []DocTargetResult{}

	formatted := FormatDocTargetResults(results)

	if formatted != "" {
		t.Error("Expected empty string for no results")
	}
}

func TestVerifyDocumentationTargets_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	task := models.Task{
		Number: "1",
		Name:   "Canceled task",
		Prompt: "This should be canceled",
		Type:   "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{
			DocumentationTargets: []models.DocumentationTarget{
				{Location: "/any/path.md", Section: "## Any"},
			},
		},
	}

	_, err := VerifyDocumentationTargets(ctx, task)
	if err == nil {
		t.Error("Expected error for canceled context")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestIsDocumentationTask validates doc task type detection
func TestIsDocumentationTask(t *testing.T) {
	tests := []struct {
		name     string
		taskType string
		expected bool
	}{
		{"documentation type", "documentation", true},
		{"component type", "component", false},
		{"integration type", "integration", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := models.Task{Type: tt.taskType}
			if got := IsDocumentationTask(task); got != tt.expected {
				t.Errorf("IsDocumentationTask() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDocTaskWithoutTargets_ValidationFailure tests criterion #3:
// Documentation tasks WITHOUT targets MUST fail validation (no graceful skip)
func TestDocTaskWithoutTargets_ValidationFailure(t *testing.T) {
	// Documentation task with nil RuntimeMetadata
	task1 := models.Task{
		Number: "1",
		Name:   "Doc task missing metadata",
		Type:   "documentation",
	}
	if !IsDocumentationTask(task1) {
		t.Error("Expected IsDocumentationTask to return true")
	}
	if HasDocumentationTargets(task1) {
		t.Error("Expected HasDocumentationTargets to return false for nil metadata")
	}

	// Documentation task with empty DocumentationTargets
	task2 := models.Task{
		Number:          "2",
		Name:            "Doc task empty targets",
		Type:            "documentation",
		RuntimeMetadata: &models.TaskMetadataRuntime{},
	}
	if !IsDocumentationTask(task2) {
		t.Error("Expected IsDocumentationTask to return true")
	}
	if HasDocumentationTargets(task2) {
		t.Error("Expected HasDocumentationTargets to return false for empty targets")
	}
}

// Helper function - use strings.Contains directly in tests
