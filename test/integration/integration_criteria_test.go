package integration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/executor"
)

// TestIntegrationTask_DualCriteriaQC verifies that integration tasks with both
// success_criteria and integration_criteria are properly processed through QC.
func TestIntegrationTask_DualCriteriaQC(t *testing.T) {
	// Load dual-criteria-plan.yaml
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "dual-criteria-plan.yaml"))

	if len(plan.Tasks) < 2 {
		t.Fatal("dual-criteria-plan.yaml should have at least 2 tasks")
	}

	integrationTask := plan.Tasks[1]

	// Verify Task 2 has Type="integration"
	if integrationTask.Type != "integration" {
		t.Errorf("Task 2 type = %q, want %q", integrationTask.Type, "integration")
	}

	// Verify both SuccessCriteria and IntegrationCriteria are present
	if len(integrationTask.SuccessCriteria) == 0 {
		t.Error("Task 2 should have success_criteria")
	}
	if len(integrationTask.IntegrationCriteria) == 0 {
		t.Error("Task 2 should have integration_criteria")
	}

	// Create QC controller
	qc := executor.NewQualityController(&fakeInvoker{})
	qc.AgentConfig.Mode = ""

	// Build structured review prompt using BuildStructuredReviewPrompt
	prompt := qc.BuildStructuredReviewPrompt(context.Background(), integrationTask, "dummy output")

	// Verify prompt contains success criteria section
	if !strings.Contains(prompt, "## SUCCESS CRITERIA") {
		t.Error("prompt missing ## SUCCESS CRITERIA header")
	}

	// Verify prompt contains integration criteria section
	if !strings.Contains(prompt, "## INTEGRATION CRITERIA") {
		t.Error("prompt missing ## INTEGRATION CRITERIA header")
	}

	// Verify criterion numbering: success criteria (0-1), then integration criteria (2-4)
	// Success criteria: "Service file exists at internal/service.go" (0)
	//                   "Service imports base component" (1)
	// Integration criteria: "Properly integrated with base component from Task 1" (2)
	//                      "All dependency files were read and referenced" (3)
	//                      "No breaking changes to base interface" (4)

	// Check for numbered criteria in prompt
	if !strings.Contains(prompt, "0. [ ]") {
		t.Error("prompt missing criterion index 0")
	}
	if !strings.Contains(prompt, "1. [ ]") {
		t.Error("prompt missing criterion index 1")
	}
	if !strings.Contains(prompt, "2. [ ]") {
		t.Error("prompt missing criterion index 2 (first integration criterion)")
	}
	if !strings.Contains(prompt, "3. [ ]") {
		t.Error("prompt missing criterion index 3 (second integration criterion)")
	}
	if !strings.Contains(prompt, "4. [ ]") {
		t.Error("prompt missing criterion index 4 (third integration criterion)")
	}

	// Verify success criteria texts appear in prompt
	if !strings.Contains(prompt, "Service file exists at internal/service.go") {
		t.Error("prompt missing success criterion text")
	}
	if !strings.Contains(prompt, "Service imports base component") {
		t.Error("prompt missing success criterion text")
	}

	// Verify integration criteria texts appear in prompt
	if !strings.Contains(prompt, "Properly integrated with base component from Task 1") {
		t.Error("prompt missing integration criterion text")
	}
	if !strings.Contains(prompt, "All dependency files were read and referenced") {
		t.Error("prompt missing integration criterion text")
	}
	if !strings.Contains(prompt, "No breaking changes to base interface") {
		t.Error("prompt missing integration criterion text")
	}
}

// TestIntegrationTask_DualCriteriaVerification verifies QC correctly validates both
// success and integration criteria for integration tasks.
func TestIntegrationTask_DualCriteriaVerification(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "dual-criteria-plan.yaml"))
	integrationTask := plan.Tasks[1]

	tests := []struct {
		name         string
		response     string
		wantFlag     string
		wantCriteria int
	}{
		{
			name: "all criteria pass",
			response: `{"verdict":"GREEN","feedback":"All criteria met","criteria_results":[
				{"index":0,"passed":true,"evidence":"file exists"},
				{"index":1,"passed":true,"evidence":"imports present"},
				{"index":2,"passed":true,"evidence":"integration correct"},
				{"index":3,"passed":true,"evidence":"dependencies read"},
				{"index":4,"passed":true,"evidence":"interface intact"}
			]}`,
			wantFlag:     "GREEN",
			wantCriteria: 5,
		},
		{
			name: "success criteria fail",
			response: `{"verdict":"GREEN","feedback":"Integration ok","criteria_results":[
				{"index":0,"passed":false,"evidence":"file missing"},
				{"index":1,"passed":true,"evidence":"imports present"},
				{"index":2,"passed":true,"evidence":"integration correct"},
				{"index":3,"passed":true,"evidence":"dependencies read"},
				{"index":4,"passed":true,"evidence":"interface intact"}
			]}`,
			wantFlag:     "RED",
			wantCriteria: 5,
		},
		{
			name: "integration criterion fails",
			response: `{"verdict":"GREEN","feedback":"Success criteria ok","criteria_results":[
				{"index":0,"passed":true,"evidence":"file exists"},
				{"index":1,"passed":true,"evidence":"imports present"},
				{"index":2,"passed":false,"evidence":"integration broken"},
				{"index":3,"passed":true,"evidence":"dependencies read"},
				{"index":4,"passed":true,"evidence":"interface intact"}
			]}`,
			wantFlag:     "RED",
			wantCriteria: 5,
		},
		{
			name: "multiple integration criteria fail",
			response: `{"verdict":"GREEN","feedback":"Some criteria failed","criteria_results":[
				{"index":0,"passed":true,"evidence":"file exists"},
				{"index":1,"passed":true,"evidence":"imports present"},
				{"index":2,"passed":false,"evidence":"integration broken"},
				{"index":3,"passed":false,"evidence":"dependencies not read"},
				{"index":4,"passed":true,"evidence":"interface intact"}
			]}`,
			wantFlag:     "RED",
			wantCriteria: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := &fakeInvoker{outputs: []string{tt.response}}
			qc := executor.NewQualityController(inv)
			qc.AgentConfig.Mode = ""

			result, err := qc.Review(context.Background(), integrationTask, "agent output")
			if err != nil {
				t.Fatalf("Review() error = %v", err)
			}

			if result.Flag != tt.wantFlag {
				t.Errorf("Review() flag = %s, want %s", result.Flag, tt.wantFlag)
			}

			if len(result.CriteriaResults) != tt.wantCriteria {
				t.Errorf("Review() criteria count = %d, want %d", len(result.CriteriaResults), tt.wantCriteria)
			}
		})
	}
}

// TestIntegrationTask_TaskTypeAndCriteria verifies integration task type is
// correctly parsed and criteria are combined in proper order.
func TestIntegrationTask_TaskTypeAndCriteria(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "dual-criteria-plan.yaml"))

	if len(plan.Tasks) < 2 {
		t.Fatal("expected at least 2 tasks")
	}

	baseTask := plan.Tasks[0]
	integrationTask := plan.Tasks[1]

	// Base task should NOT be integration type
	if baseTask.Type == "integration" {
		t.Error("Task 1 should not have type=integration")
	}

	// Integration task should have type="integration"
	if integrationTask.Type != "integration" {
		t.Errorf("Task 2 type = %q, want %q", integrationTask.Type, "integration")
	}

	// Base task should only have success criteria, no integration criteria
	if len(baseTask.SuccessCriteria) == 0 {
		t.Error("Task 1 should have success_criteria")
	}
	if len(baseTask.IntegrationCriteria) > 0 {
		t.Error("Task 1 should not have integration_criteria")
	}

	// Integration task should have both success and integration criteria
	if len(integrationTask.SuccessCriteria) == 0 {
		t.Error("Task 2 should have success_criteria")
	}
	if len(integrationTask.IntegrationCriteria) == 0 {
		t.Error("Task 2 should have integration_criteria")
	}

	// Verify success criteria count
	if len(integrationTask.SuccessCriteria) != 2 {
		t.Errorf("Task 2 success_criteria count = %d, want 2", len(integrationTask.SuccessCriteria))
	}

	// Verify integration criteria count
	if len(integrationTask.IntegrationCriteria) != 3 {
		t.Errorf("Task 2 integration_criteria count = %d, want 3", len(integrationTask.IntegrationCriteria))
	}
}

// TestIntegrationTask_DependencyContext verifies that integration tasks with
// dependencies include dependency context in the task prompt.
func TestIntegrationTask_DependencyContext(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "dual-criteria-plan.yaml"))

	if len(plan.Tasks) < 2 {
		t.Fatal("expected at least 2 tasks")
	}

	integrationTask := plan.Tasks[1]

	// Verify task has dependencies
	if len(integrationTask.DependsOn) == 0 {
		t.Fatal("Task 2 should have dependencies")
	}

	if integrationTask.DependsOn[0] != "1" {
		t.Errorf("Task 2 depends_on = %v, want [\"1\"]", integrationTask.DependsOn)
	}

	// Create a fake invoker that captures prompts
	captureInv := &fakeInvokerWithCapture{}

	te, err := executor.NewTaskExecutor(captureInv, nil, nil, executor.TaskExecutorConfig{})
	if err != nil {
		t.Fatalf("NewTaskExecutor() error = %v", err)
	}
	te.Plan = plan

	// Execute the integration task
	_, _ = te.Execute(context.Background(), integrationTask)

	if len(captureInv.capturedPrompts) == 0 {
		t.Fatal("no prompts captured during execution")
	}

	prompt := captureInv.capturedPrompts[0]

	// Verify integration task context appears in prompt
	if !strings.Contains(prompt, "# INTEGRATION TASK CONTEXT") {
		t.Error("prompt missing INTEGRATION TASK CONTEXT header")
	}

	if !strings.Contains(prompt, "Before implementing") {
		t.Error("prompt missing instruction about reading dependency files")
	}

	if !strings.Contains(prompt, "## Dependency:") {
		t.Error("prompt missing dependency section")
	}

	// Verify Task 1 file is referenced in dependency context
	if !strings.Contains(prompt, "internal/base.go") {
		t.Error("prompt missing Task 1 files in dependency context")
	}
}

// TestIntegrationTask_PromptBuilding verifies BuildStructuredReviewPrompt
// produces correct output for integration tasks.
func TestIntegrationTask_PromptBuilding(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "dual-criteria-plan.yaml"))
	integrationTask := plan.Tasks[1]

	qc := executor.NewQualityController(&fakeInvoker{})

	prompt := qc.BuildStructuredReviewPrompt(context.Background(), integrationTask, "test output")

	// Verify basic structure
	if !strings.Contains(prompt, "# Quality Control Review:") {
		t.Error("prompt missing title")
	}

	if !strings.Contains(prompt, "## Task Requirements") {
		t.Error("prompt missing task requirements section")
	}

	// Verify criteria sections are separated
	successIdx := strings.Index(prompt, "## SUCCESS CRITERIA")
	integrationIdx := strings.Index(prompt, "## INTEGRATION CRITERIA")

	if successIdx == -1 {
		t.Error("prompt missing SUCCESS CRITERIA section")
	}
	if integrationIdx == -1 {
		t.Error("prompt missing INTEGRATION CRITERIA section")
	}

	// SUCCESS CRITERIA should come before INTEGRATION CRITERIA
	if successIdx > integrationIdx {
		t.Error("SUCCESS CRITERIA section should come before INTEGRATION CRITERIA")
	}

	// Verify each criterion is numbered
	allLines := strings.Split(prompt, "\n")
	criteriaStarted := false
	for i, line := range allLines {
		if strings.Contains(line, "## SUCCESS CRITERIA") {
			criteriaStarted = true
		}
		if strings.Contains(line, "## INTEGRATION CRITERIA") {
			// After this point, verify integration criteria are numbered
			for j := i + 1; j < len(allLines) && strings.TrimSpace(allLines[j]) != ""; j++ {
				if strings.HasPrefix(strings.TrimSpace(allLines[j]), "[") || strings.HasPrefix(strings.TrimSpace(allLines[j]), "-") {
					break
				}
			}
		}
	}

	if !criteriaStarted {
		t.Error("criteria section was not started")
	}
}

// TestIntegrationTask_CriteriaOrdering verifies that criteria are properly
// ordered with success criteria first, then integration criteria.
func TestIntegrationTask_CriteriaOrdering(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "dual-criteria-plan.yaml"))
	integrationTask := plan.Tasks[1]

	qc := executor.NewQualityController(&fakeInvoker{})
	prompt := qc.BuildStructuredReviewPrompt(context.Background(), integrationTask, "output")

	// Extract lines with criteria indices by looking for numbered criteria format
	lines := strings.Split(prompt, "\n")

	var criteriaLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for lines with numbered criteria: "0. [ ]", "1. [ ]", etc.
		if len(trimmed) > 0 && (trimmed[0] >= '0' && trimmed[0] <= '9') && strings.Contains(trimmed, "[ ]") {
			criteriaLines = append(criteriaLines, trimmed)
		}
	}

	// Verify we have criteria
	if len(criteriaLines) < 5 {
		t.Errorf("expected at least 5 criteria lines, got %d (criteria lines: %v)", len(criteriaLines), criteriaLines)
		return
	}

	// Verify first two are success criteria (indices 0-1)
	if !strings.Contains(criteriaLines[0], "0. [ ]") {
		t.Errorf("first criterion should be index 0, got %s", criteriaLines[0])
	}
	if !strings.Contains(criteriaLines[1], "1. [ ]") {
		t.Errorf("second criterion should be index 1, got %s", criteriaLines[1])
	}

	// Verify next three are integration criteria (indices 2-4)
	if !strings.Contains(criteriaLines[2], "2. [ ]") {
		t.Errorf("third criterion should be index 2, got %s", criteriaLines[2])
	}
	if !strings.Contains(criteriaLines[3], "3. [ ]") {
		t.Errorf("fourth criterion should be index 3, got %s", criteriaLines[3])
	}
	if !strings.Contains(criteriaLines[4], "4. [ ]") {
		t.Errorf("fifth criterion should be index 4, got %s", criteriaLines[4])
	}
}
