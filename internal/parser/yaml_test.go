package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

func TestParseYAMLPlan(t *testing.T) {
	yamlContent := `
plan:
  metadata:
    feature_name: "Test Plan"
    estimated_tasks: 2
  tasks:
    - task_number: 1
      name: "First Task"
      files: ["file1.go", "file2.go"]
      depends_on: []
      estimated_time: "30m"
      description: "Test task description"
      test_first:
        test_file: "test_file.go"
        example_skeleton: "func TestExample(t *testing.T) {}"
      implementation:
        approach: "Use TDD approach"
        code_structure: "package main"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
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
	if task.Files[0] != "file1.go" {
		t.Errorf("Expected first file 'file1.go', got '%s'", task.Files[0])
	}

	// Verify files appear in prompt (critical for agent compliance)
	// XML format: <target_files required="true">
	if !strings.Contains(task.Prompt, "<target_files required=\"true\">") {
		t.Error("Prompt should contain '<target_files required=\"true\">' section")
	}
	if !strings.Contains(task.Prompt, "<file>file1.go</file>") {
		t.Error("Prompt should contain file1.go as XML element")
	}
	if !strings.Contains(task.Prompt, "<file>file2.go</file>") {
		t.Error("Prompt should contain file2.go as XML element")
	}
	if !strings.Contains(task.Prompt, "MUST create/modify these exact files") {
		t.Error("Prompt should contain instruction about exact file paths")
	}
}

func TestParseYAMLWithDependencies(t *testing.T) {
	yamlContent := `
plan:
  metadata:
    feature_name: "Dependency Test"
  tasks:
    - task_number: 1
      name: "First Task"
      depends_on: []
      estimated_time: "30m"
      description: "First task"
    - task_number: 2
      name: "Second Task"
      depends_on: [1]
      estimated_time: "1h"
      description: "Second task"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}

	task2 := plan.Tasks[1]
	if len(task2.DependsOn) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(task2.DependsOn))
	}
	if task2.DependsOn[0] != "1" {
		t.Errorf("Expected dependency on task 1, got %s", task2.DependsOn[0])
	}
}

func TestParseYAMLEstimatedTime(t *testing.T) {
	tests := []struct {
		name        string
		timeStr     string
		expectedDur time.Duration
	}{
		{"30 minutes", "30m", 30 * time.Minute},
		{"1 hour", "1h", 1 * time.Hour},
		{"2 hours 30 minutes", "2h30m", 2*time.Hour + 30*time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Time Test"
      depends_on: []
      estimated_time: "` + tt.timeStr + `"
      description: "Test"
`

			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(yamlContent))
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}

			if plan.Tasks[0].EstimatedTime != tt.expectedDur {
				t.Errorf("Expected duration %v, got %v", tt.expectedDur, plan.Tasks[0].EstimatedTime)
			}
		})
	}
}

func TestParseYAMLWithConductorConfig(t *testing.T) {
	yamlContent := `
conductor:
  default_agent: "swiftdev"
  max_concurrency: 5
  quality_control:
    enabled: true
    review_agent: "quality-control"
    retry_on_red: 2

plan:
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if plan.DefaultAgent != "swiftdev" {
		t.Errorf("Expected default agent 'swiftdev', got '%s'", plan.DefaultAgent)
	}

	if !plan.QualityControl.Enabled {
		t.Error("Expected quality control to be enabled")
	}

	if plan.QualityControl.ReviewAgent != "quality-control" {
		t.Errorf("Expected review agent 'quality-control', got '%s'", plan.QualityControl.ReviewAgent)
	}

	if plan.QualityControl.RetryOnRed != 2 {
		t.Errorf("Expected retry count 2, got %d", plan.QualityControl.RetryOnRed)
	}
}

func TestParseYAMLPromptGeneration(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Main task description"
      test_first:
        test_file: "test.go"
        example_skeleton: "Example test code"
      implementation:
        approach: "TDD approach"
        code_structure: "Code structure example"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	prompt := plan.Tasks[0].Prompt
	if !strings.Contains(prompt, "Main task description") {
		t.Error("Prompt should contain description")
	}
	if !strings.Contains(prompt, "Example test code") {
		t.Error("Prompt should contain test example")
	}
	if !strings.Contains(prompt, "TDD approach") {
		t.Error("Prompt should contain implementation approach")
	}
	if !strings.Contains(prompt, "Code structure example") {
		t.Error("Prompt should contain code structure")
	}
}

func TestParseYAMLInvalidFormat(t *testing.T) {
	yamlContent := `
invalid yaml content {{{
not properly formatted
`

	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("Expected error for invalid YAML format")
	}
}

func TestParseYAMLEmptyTasks(t *testing.T) {
	yamlContent := `
plan:
  metadata:
    feature_name: "Empty Plan"
  tasks: []
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if len(plan.Tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(plan.Tasks))
	}
}

func TestParseYAMLWithAgent(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Swift Task"
      depends_on: []
      estimated_time: "30m"
      description: "Swift implementation"
      agent: "swiftdev"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if plan.Tasks[0].Agent != "swiftdev" {
		t.Errorf("Expected agent 'swiftdev', got '%s'", plan.Tasks[0].Agent)
	}
}

func TestParseYAMLWithStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		completedDate  string
		expectedStatus string
	}{
		{"pending task", "pending", "", "pending"},
		{"completed task", "completed", "2025-11-10", "completed"},
		{"in-progress task", "in_progress", "", "in_progress"},
		{"skipped task", "skipped", "", "skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      status: "` + tt.status + `"`

			if tt.completedDate != "" {
				yamlContent += `
      completed_date: "` + tt.completedDate + `"`
			}

			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(yamlContent))
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}

			if plan.Tasks[0].Status != tt.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, plan.Tasks[0].Status)
			}
		})
	}
}

func TestParseYAMLStatusWithCompletedDate(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Completed Task"
      depends_on: []
      estimated_time: "30m"
      description: "This task was completed"
      status: "completed"
      completed_date: "2025-11-10"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if task.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", task.Status)
	}

	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set, but it was nil")
	} else {
		// Verify it's parsed as a date (at least the date part matches)
		if task.CompletedAt.Format("2006-01-02") != "2025-11-10" {
			t.Errorf("Expected completion date '2025-11-10', got '%s'", task.CompletedAt.Format("2006-01-02"))
		}
	}
}

func TestParseYAMLStatusWithCompletedAt(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Completed Task"
      depends_on: []
      estimated_time: "30m"
      description: "This task was completed"
      status: "completed"
      completed_at: "2025-11-10T14:30:00Z"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if task.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", task.Status)
	}

	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set, but it was nil")
	} else {
		if task.CompletedAt.Format("2006-01-02") != "2025-11-10" {
			t.Errorf("Expected completion date '2025-11-10', got '%s'", task.CompletedAt.Format("2006-01-02"))
		}
	}
}

func TestParseYAMLStatusWithoutTimestamp(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Pending Task"
      depends_on: []
      estimated_time: "30m"
      description: "This task is pending"
      status: "pending"
`

	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if task.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", task.Status)
	}

	if task.CompletedAt != nil {
		t.Errorf("Expected CompletedAt to be nil for pending task, but got %v", task.CompletedAt)
	}
}

func TestParseYAMLWorktreeGroup(t *testing.T) {
	tests := []struct {
		name              string
		yamlContent       string
		expectedGroup     string
		expectSecondGroup string
	}{
		{
			name: "task with worktree_group",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Backend Task"
      depends_on: []
      estimated_time: "30m"
      description: "Backend implementation"
      worktree_group: "backend-core"
`,
			expectedGroup: "backend-core",
		},
		{
			name: "task without worktree_group",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Generic Task"
      depends_on: []
      estimated_time: "30m"
      description: "Generic task"
`,
			expectedGroup: "",
		},
		{
			name: "multiple tasks with different groups",
			yamlContent: `
plan:
  tasks:
    - task_number: 1
      name: "Backend Task"
      depends_on: []
      estimated_time: "30m"
      description: "Backend"
      worktree_group: "backend-api"
    - task_number: 2
      name: "Frontend Task"
      depends_on: []
      estimated_time: "45m"
      description: "Frontend"
      worktree_group: "frontend-ui"
`,
			expectedGroup:     "backend-api",
			expectSecondGroup: "frontend-ui",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(tt.yamlContent))
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}

			if len(plan.Tasks) < 1 {
				t.Fatal("Expected at least 1 task")
			}

			task := plan.Tasks[0]
			if task.WorktreeGroup != tt.expectedGroup {
				t.Errorf("Expected worktree group '%s', got '%s'", tt.expectedGroup, task.WorktreeGroup)
			}

			// For multi-task test, check second task
			if tt.expectSecondGroup != "" {
				if len(plan.Tasks) < 2 {
					t.Fatal("Expected at least 2 tasks")
				}
				task2 := plan.Tasks[1]
				if task2.WorktreeGroup != tt.expectSecondGroup {
					t.Errorf("Expected second task worktree group '%s', got '%s'", tt.expectSecondGroup, task2.WorktreeGroup)
				}
			}
		})
	}
}

func TestParseYAMLWithQCInvalidMode(t *testing.T) {
	yamlContent := `
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "INVALID"

plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`

	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("Expected error for invalid QC agents mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid QC agents mode") {
		t.Errorf("Expected error about invalid mode, got: %v", err)
	}
}

func TestParseYAMLWithQCExplicitModeNoList(t *testing.T) {
	yamlContent := `
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "explicit"
      explicit_list: []

plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`

	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("Expected error for explicit mode with empty explicit_list, got nil")
	}
	if !strings.Contains(err.Error(), "explicit mode requires non-empty explicit_list") {
		t.Errorf("Expected error about explicit_list requirement, got: %v", err)
	}
}

func TestParseYAMLWithQCModeCaseNormalization(t *testing.T) {
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
			yamlContent := `
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "` + tt.modeInput + `"`
			if tt.withList {
				yamlContent += `
      explicit_list:
        - agent1
        - agent2`
			}
			yamlContent += `

plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`

			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(yamlContent))

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

func TestYAMLParser_SuccessCriteria(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Test task"
      files: [test.go]
      depends_on: []
      estimated_time: 30m
      description: "Do something"
      success_criteria:
        - "Criterion 1"
        - "Criterion 2"
      test_commands:
        - "go test ./..."
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}
	task := plan.Tasks[0]
	if len(task.SuccessCriteria) != 2 {
		t.Errorf("expected 2 criteria, got %d", len(task.SuccessCriteria))
	}
	if len(task.SuccessCriteria) > 0 && task.SuccessCriteria[0] != "Criterion 1" {
		t.Errorf("first criterion mismatch: %s", task.SuccessCriteria[0])
	}
	if len(task.SuccessCriteria) > 1 && task.SuccessCriteria[1] != "Criterion 2" {
		t.Errorf("second criterion mismatch: %s", task.SuccessCriteria[1])
	}
	if len(task.TestCommands) != 1 {
		t.Errorf("expected 1 test command, got %d", len(task.TestCommands))
	}
	if len(task.TestCommands) > 0 && task.TestCommands[0] != "go test ./..." {
		t.Errorf("test command mismatch: %s", task.TestCommands[0])
	}
}

func TestYAMLParser_NoSuccessCriteria_BackwardCompatible(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Legacy task"
      files: [test.go]
      depends_on: []
      estimated_time: 30m
      description: "Legacy task without new fields"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}
	task := plan.Tasks[0]
	if task.SuccessCriteria != nil && len(task.SuccessCriteria) != 0 {
		t.Errorf("expected empty success_criteria, got %d items", len(task.SuccessCriteria))
	}
	if task.TestCommands != nil && len(task.TestCommands) != 0 {
		t.Errorf("expected empty test_commands, got %d items", len(task.TestCommands))
	}
}

func TestYAMLParser_MultipleTestCommands(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Task with multiple test commands"
      files: [test.go]
      depends_on: []
      estimated_time: 30m
      description: "Test"
      success_criteria:
        - "All tests pass"
        - "No linting errors"
        - "Code coverage above 80%"
      test_commands:
        - "go test ./..."
        - "golangci-lint run"
        - "go test -cover ./..."
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	task := plan.Tasks[0]
	if len(task.SuccessCriteria) != 3 {
		t.Errorf("expected 3 criteria, got %d", len(task.SuccessCriteria))
	}
	if len(task.TestCommands) != 3 {
		t.Errorf("expected 3 test commands, got %d", len(task.TestCommands))
	}
	if len(task.TestCommands) >= 2 && task.TestCommands[1] != "golangci-lint run" {
		t.Errorf("second test command mismatch: %s", task.TestCommands[1])
	}
}

func TestParseYAMLWithQCAgentConfig(t *testing.T) {
	tests := []struct {
		name               string
		yamlContent        string
		expectedMode       string
		expectedExplicit   []string
		expectedAdditional []string
		expectedBlocked    []string
	}{
		{
			name: "QC with auto mode",
			yamlContent: `
conductor:
  quality_control:
    enabled: true
    review_agent: "quality-control"
    retry_on_red: 2
    agents:
      mode: "auto"
      blocked: ["deprecated-agent"]
plan:
  metadata:
    feature_name: "QC Test"
  tasks:
    - task_number: 1
      name: "Task 1"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`,
			expectedMode:       "auto",
			expectedBlocked:    []string{"deprecated-agent"},
			expectedExplicit:   []string{},
			expectedAdditional: []string{},
		},
		{
			name: "QC with explicit mode",
			yamlContent: `
conductor:
  quality_control:
    enabled: true
    retry_on_red: 3
    agents:
      mode: "explicit"
      explicit_list: ["golang-pro", "code-reviewer"]
plan:
  metadata:
    feature_name: "QC Explicit Test"
  tasks:
    - task_number: 1
      name: "Task 1"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`,
			expectedMode:       "explicit",
			expectedExplicit:   []string{"golang-pro", "code-reviewer"},
			expectedAdditional: []string{},
			expectedBlocked:    []string{},
		},
		{
			name: "QC with mixed mode",
			yamlContent: `
conductor:
  quality_control:
    enabled: true
    agents:
      mode: "mixed"
      additional: ["security-auditor", "performance-reviewer"]
      blocked: ["old-agent"]
plan:
  metadata:
    feature_name: "QC Mixed Test"
  tasks:
    - task_number: 1
      name: "Task 1"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`,
			expectedMode:       "mixed",
			expectedExplicit:   []string{},
			expectedAdditional: []string{"security-auditor", "performance-reviewer"},
			expectedBlocked:    []string{"old-agent"},
		},
		{
			name: "QC without agents section (backward compatibility)",
			yamlContent: `
conductor:
  quality_control:
    enabled: true
    review_agent: "quality-control"
plan:
  metadata:
    feature_name: "QC Legacy Test"
  tasks:
    - task_number: 1
      name: "Task 1"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`,
			expectedMode:       "",
			expectedExplicit:   []string{},
			expectedAdditional: []string{},
			expectedBlocked:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(tt.yamlContent))
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
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

func TestParseYAMLWithCrossFileDependencies(t *testing.T) {
	tests := []struct {
		name              string
		yamlContent       string
		expectedTaskNum   string
		expectedDependsOn []string
		shouldError       bool
		errorContains     string
	}{
		{
			name: "simple cross-file dependency",
			yamlContent: `
plan:
  metadata:
    feature_name: "Cross-file Test"
  tasks:
    - task_number: 3
      name: "Task 3"
      depends_on:
        - file: "plan-01.yaml"
          task: 2
      estimated_time: "30m"
      description: "Task with cross-file dependency"
`,
			expectedTaskNum:   "3",
			expectedDependsOn: []string{"file:plan-01.yaml:task:2"},
			shouldError:       false,
		},
		{
			name: "mixed numeric and cross-file dependencies",
			yamlContent: `
plan:
  metadata:
    feature_name: "Mixed Dependencies"
  tasks:
    - task_number: 4
      name: "Task 4"
      depends_on:
        - 1
        - file: "plan-02.yaml"
          task: 3
        - 2
      estimated_time: "1h"
      description: "Task with mixed dependencies"
`,
			expectedTaskNum:   "4",
			expectedDependsOn: []string{"1", "file:plan-02.yaml:task:3", "2"},
			shouldError:       false,
		},
		{
			name: "multiple cross-file dependencies",
			yamlContent: `
plan:
  metadata:
    feature_name: "Multiple Cross-file"
  tasks:
    - task_number: 5
      name: "Task 5"
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 1
        - file: "plan-02-auth.yaml"
          task: 2
        - file: "plan-03-api.yaml"
          task: 4
      estimated_time: "2h"
      description: "Task with multiple cross-file dependencies"
`,
			expectedTaskNum:   "5",
			expectedDependsOn: []string{"file:plan-01-foundation.yaml:task:1", "file:plan-02-auth.yaml:task:2", "file:plan-03-api.yaml:task:4"},
			shouldError:       false,
		},
		{
			name: "cross-file with float task number",
			yamlContent: `
plan:
  metadata:
    feature_name: "Float Task Number"
  tasks:
    - task_number: 6
      name: "Task 6"
      depends_on:
        - file: "plan-01.yaml"
          task: 2.5
      estimated_time: "45m"
      description: "Task with float task ID in cross-file dependency"
`,
			expectedTaskNum:   "6",
			expectedDependsOn: []string{"file:plan-01.yaml:task:2.5"},
			shouldError:       false,
		},
		{
			name: "cross-file with string task number",
			yamlContent: `
plan:
  metadata:
    feature_name: "String Task Number"
  tasks:
    - task_number: 7
      name: "Task 7"
      depends_on:
        - file: "plan-setup.yaml"
          task: "integration-1"
      estimated_time: "1h"
      description: "Task with string task ID in cross-file dependency"
`,
			expectedTaskNum:   "7",
			expectedDependsOn: []string{"file:plan-setup.yaml:task:integration-1"},
			shouldError:       false,
		},
		{
			name: "cross-file dependency missing file field",
			yamlContent: `
plan:
  metadata:
    feature_name: "Missing File"
  tasks:
    - task_number: 8
      name: "Task 8"
      depends_on:
        - task: 2
      estimated_time: "30m"
      description: "Task with malformed cross-file dependency"
`,
			shouldError:   true,
			errorContains: "missing required 'file' field",
		},
		{
			name: "cross-file dependency missing task field",
			yamlContent: `
plan:
  metadata:
    feature_name: "Missing Task"
  tasks:
    - task_number: 9
      name: "Task 9"
      depends_on:
        - file: "plan-01.yaml"
      estimated_time: "30m"
      description: "Task with malformed cross-file dependency"
`,
			shouldError:   true,
			errorContains: "missing required 'task' field",
		},
		{
			name: "backward compatibility - numeric only dependencies",
			yamlContent: `
plan:
  metadata:
    feature_name: "Backward Compat"
  tasks:
    - task_number: 1
      name: "Task 1"
      depends_on: []
      estimated_time: "30m"
      description: "First task"
    - task_number: 2
      name: "Task 2"
      depends_on: [1]
      estimated_time: "45m"
      description: "Second task"
`,
			expectedTaskNum:   "2",
			expectedDependsOn: []string{"1"},
			shouldError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(tt.yamlContent))

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, but parsing succeeded")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			// Find the task we're testing
			var task *models.Task
			for i := range plan.Tasks {
				if plan.Tasks[i].Number == tt.expectedTaskNum {
					task = &plan.Tasks[i]
					break
				}
			}

			if task == nil {
				t.Fatalf("task %s not found in parsed plan", tt.expectedTaskNum)
			}

			// Verify dependencies
			if len(task.DependsOn) != len(tt.expectedDependsOn) {
				t.Errorf("expected %d dependencies, got %d", len(tt.expectedDependsOn), len(task.DependsOn))
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

// =============================================================================
// Runtime Enforcement Tests (v2.9+)
// =============================================================================

func TestPlannerComplianceSpec_Unmarshal(t *testing.T) {
	yamlContent := `
planner_compliance:
  planner_version: "1.2.0"
  strict_enforcement: true
  required_features:
    - dependency_checks
    - documentation_targets
    - success_criteria

plan:
  metadata:
    feature_name: "Compliance Test"
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      runtime_metadata:
        dependency_checks:
          - command: "go build ./..."
            description: "Build check"
        documentation_targets: []
        prompt_blocks: []
      success_criteria:
        - criterion: "Test passes"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if plan.PlannerCompliance == nil {
		t.Fatal("expected PlannerCompliance to be populated")
	}

	if plan.PlannerCompliance.PlannerVersion != "1.2.0" {
		t.Errorf("expected planner_version '1.2.0', got %q", plan.PlannerCompliance.PlannerVersion)
	}

	if !plan.PlannerCompliance.StrictEnforcement {
		t.Error("expected strict_enforcement to be true")
	}

	if len(plan.PlannerCompliance.RequiredFeatures) != 3 {
		t.Errorf("expected 3 required_features, got %d", len(plan.PlannerCompliance.RequiredFeatures))
	}
}

func TestPlannerComplianceSpec_MissingVersion(t *testing.T) {
	yamlContent := `
planner_compliance:
  strict_enforcement: true

plan:
  metadata:
    feature_name: "Missing Version Test"
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
`
	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("expected error for missing planner_version, got nil")
	}
	if !strings.Contains(err.Error(), "planner_version") {
		t.Errorf("expected error about planner_version, got: %v", err)
	}
}

func TestTaskMetadataRuntime_RequiredFields(t *testing.T) {
	yamlContent := `
planner_compliance:
  planner_version: "1.0.0"
  strict_enforcement: true

plan:
  metadata:
    feature_name: "Runtime Metadata Test"
  tasks:
    - task_number: 1
      name: "Task with runtime metadata"
      depends_on: []
      estimated_time: "30m"
      description: "Test task"
      runtime_metadata:
        dependency_checks:
          - command: "go build ./..."
            description: "Build verification"
          - command: "go vet ./..."
            description: "Static analysis"
        documentation_targets:
          - location: "docs/readme.md"
            section: "Usage"
        prompt_blocks:
          - type: "context"
            content: "Important context here"
      success_criteria:
        - criterion: "All tests pass"
          verification:
            command: "go test ./..."
            expected: "PASS"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}

	task := plan.Tasks[0]
	if task.RuntimeMetadata == nil {
		t.Fatal("expected RuntimeMetadata to be populated")
	}

	// Check dependency checks
	if len(task.RuntimeMetadata.DependencyChecks) != 2 {
		t.Errorf("expected 2 dependency_checks, got %d", len(task.RuntimeMetadata.DependencyChecks))
	}
	if task.RuntimeMetadata.DependencyChecks[0].Command != "go build ./..." {
		t.Errorf("expected first check command 'go build ./...', got %q", task.RuntimeMetadata.DependencyChecks[0].Command)
	}

	// Check documentation targets
	if len(task.RuntimeMetadata.DocumentationTargets) != 1 {
		t.Errorf("expected 1 documentation_target, got %d", len(task.RuntimeMetadata.DocumentationTargets))
	}
	if task.RuntimeMetadata.DocumentationTargets[0].Location != "docs/readme.md" {
		t.Errorf("expected location 'docs/readme.md', got %q", task.RuntimeMetadata.DocumentationTargets[0].Location)
	}

	// Check prompt blocks
	if len(task.RuntimeMetadata.PromptBlocks) != 1 {
		t.Errorf("expected 1 prompt_block, got %d", len(task.RuntimeMetadata.PromptBlocks))
	}
	if task.RuntimeMetadata.PromptBlocks[0].Type != "context" {
		t.Errorf("expected prompt block type 'context', got %q", task.RuntimeMetadata.PromptBlocks[0].Type)
	}
}

func TestCriterionVerification_Optional(t *testing.T) {
	yamlContent := `
planner_compliance:
  planner_version: "1.0.0"
  strict_enforcement: false

plan:
  metadata:
    feature_name: "Verification Test"
  tasks:
    - task_number: 1
      name: "Task with mixed criteria"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      runtime_metadata:
        dependency_checks: []
        documentation_targets: []
        prompt_blocks: []
      success_criteria:
        - criterion: "Has verification block"
          verification:
            command: "go test -run TestFoo"
            expected: "PASS"
            description: "Run specific test"
        - criterion: "No verification block"
        - criterion: "Another with verification"
          verification:
            command: "make lint"
            expected: "0 issues"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if len(task.StructuredCriteria) != 3 {
		t.Fatalf("expected 3 structured criteria, got %d", len(task.StructuredCriteria))
	}

	// First criterion has verification
	if task.StructuredCriteria[0].Verification == nil {
		t.Error("expected first criterion to have verification block")
	} else {
		if task.StructuredCriteria[0].Verification.Command != "go test -run TestFoo" {
			t.Errorf("wrong verification command: %q", task.StructuredCriteria[0].Verification.Command)
		}
		if task.StructuredCriteria[0].Verification.Expected != "PASS" {
			t.Errorf("wrong verification expected: %q", task.StructuredCriteria[0].Verification.Expected)
		}
		if task.StructuredCriteria[0].Verification.Description != "Run specific test" {
			t.Errorf("wrong verification description: %q", task.StructuredCriteria[0].Verification.Description)
		}
	}

	// Second criterion has no verification
	if task.StructuredCriteria[1].Verification != nil {
		t.Error("expected second criterion to have no verification block")
	}

	// Third criterion has verification
	if task.StructuredCriteria[2].Verification == nil {
		t.Error("expected third criterion to have verification block")
	}
}

func TestRuntimeMetadata_StrictEnforcementValidation(t *testing.T) {
	// When strict_enforcement is true and planner_compliance is present,
	// tasks MUST have runtime_metadata
	yamlContent := `
planner_compliance:
  planner_version: "1.0.0"
  strict_enforcement: true

plan:
  metadata:
    feature_name: "Strict Test"
  tasks:
    - task_number: 1
      name: "Task without runtime metadata"
      depends_on: []
      estimated_time: "30m"
      description: "This task is missing required runtime_metadata"
`
	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("expected error for missing runtime_metadata under strict enforcement")
	}
	if !strings.Contains(err.Error(), "runtime_metadata") {
		t.Errorf("expected error about runtime_metadata, got: %v", err)
	}
}

func TestRuntimeMetadata_NonStrictAllowsMissing(t *testing.T) {
	// When strict_enforcement is false, runtime_metadata is optional
	yamlContent := `
planner_compliance:
  planner_version: "1.0.0"
  strict_enforcement: false

plan:
  metadata:
    feature_name: "Non-Strict Test"
  tasks:
    - task_number: 1
      name: "Task without runtime metadata"
      depends_on: []
      estimated_time: "30m"
      description: "This is allowed when not strict"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Tasks[0].RuntimeMetadata != nil {
		t.Error("expected nil RuntimeMetadata for task without it")
	}
}

func TestPlannerComplianceSpec_UnknownFields(t *testing.T) {
	yamlContent := `
planner_compliance:
  planner_version: "1.0.0"
  strict_enforcement: true
  unknown_field: "should fail"

plan:
  metadata:
    feature_name: "Unknown Fields Test"
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      runtime_metadata:
        dependency_checks: []
        documentation_targets: []
        prompt_blocks: []
`
	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("expected error for unknown field in planner_compliance")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected error about unknown field, got: %v", err)
	}
}

func TestVerificationBlock_InvalidCommand(t *testing.T) {
	yamlContent := `
planner_compliance:
  planner_version: "1.0.0"
  strict_enforcement: false

plan:
  metadata:
    feature_name: "Invalid Verification Test"
  tasks:
    - task_number: 1
      name: "Task with invalid verification"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      runtime_metadata:
        dependency_checks: []
        documentation_targets: []
        prompt_blocks: []
      success_criteria:
        - criterion: "Test"
          verification:
            command: ""
            expected: "PASS"
`
	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("expected error for empty verification command")
	}
	if !strings.Contains(err.Error(), "command") {
		t.Errorf("expected error about command, got: %v", err)
	}
}

func TestRuntimeEnforcement_FromFixture(t *testing.T) {
	// Test using the actual fixture file
	plan := mustParseYAMLFile(t, "testdata/runtime_enforcement.yaml")

	// Verify PlannerCompliance
	if plan.PlannerCompliance == nil {
		t.Fatal("expected PlannerCompliance from fixture")
	}
	if plan.PlannerCompliance.PlannerVersion != "2.9.0" {
		t.Errorf("expected version '2.9.0', got %q", plan.PlannerCompliance.PlannerVersion)
	}
	if !plan.PlannerCompliance.StrictEnforcement {
		t.Error("expected strict_enforcement true")
	}

	// Verify first task has full runtime metadata
	if len(plan.Tasks) < 1 {
		t.Fatal("expected at least 1 task")
	}
	task1 := plan.Tasks[0]
	if task1.RuntimeMetadata == nil {
		t.Fatal("expected RuntimeMetadata on first task")
	}
	if len(task1.RuntimeMetadata.DependencyChecks) != 3 {
		t.Errorf("expected 3 dependency_checks, got %d", len(task1.RuntimeMetadata.DependencyChecks))
	}
	if len(task1.RuntimeMetadata.DocumentationTargets) != 2 {
		t.Errorf("expected 2 documentation_targets, got %d", len(task1.RuntimeMetadata.DocumentationTargets))
	}
	if len(task1.RuntimeMetadata.PromptBlocks) != 3 {
		t.Errorf("expected 3 prompt_blocks, got %d", len(task1.RuntimeMetadata.PromptBlocks))
	}

	// Check structured criteria with verification blocks
	if len(task1.StructuredCriteria) != 4 {
		t.Errorf("expected 4 structured criteria, got %d", len(task1.StructuredCriteria))
	}

	// First criterion has verification
	if task1.StructuredCriteria[0].Verification != nil {
		if task1.StructuredCriteria[0].Verification.Command != "go test -run TestYAMLSerialization ./internal/parser" {
			t.Errorf("wrong first verification command: %q", task1.StructuredCriteria[0].Verification.Command)
		}
	}

	// Second task has minimal but valid metadata
	if len(plan.Tasks) >= 2 {
		task2 := plan.Tasks[1]
		if task2.RuntimeMetadata == nil {
			t.Fatal("expected RuntimeMetadata on second task")
		}
		if len(task2.RuntimeMetadata.DependencyChecks) != 1 {
			t.Errorf("expected 1 dependency_check on task 2, got %d", len(task2.RuntimeMetadata.DependencyChecks))
		}
		// Empty arrays should be valid
		if task2.RuntimeMetadata.DocumentationTargets == nil {
			t.Error("expected empty DocumentationTargets array, not nil")
		}
	}
}

func TestLegacyPlan_NoPlannerCompliance(t *testing.T) {
	// Legacy plans without planner_compliance should still work
	yamlContent := `
conductor:
  default_agent: "golang-pro"

plan:
  metadata:
    feature_name: "Legacy Plan"
  tasks:
    - task_number: 1
      name: "Legacy Task"
      depends_on: []
      estimated_time: "30m"
      description: "This is a legacy task without runtime metadata"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("legacy plan should parse without error: %v", err)
	}

	if plan.PlannerCompliance != nil {
		t.Error("expected nil PlannerCompliance for legacy plan")
	}
	if plan.Tasks[0].RuntimeMetadata != nil {
		t.Error("expected nil RuntimeMetadata for legacy task")
	}
}

// =============================================================================
// CommitSpec Parsing Tests (v2.30+)
// =============================================================================

func TestYAMLParser_CommitSpec_Full(t *testing.T) {
	yamlContent := `
plan:
  metadata:
    feature_name: "Commit Spec Test"
  tasks:
    - task_number: 1
      name: "Task with full commit spec"
      depends_on: []
      estimated_time: "30m"
      description: "Test task"
      commit:
        type: "feat"
        message: "add new feature"
        body: "This adds a comprehensive new feature."
        files:
          - "internal/parser/yaml.go"
          - "internal/parser/yaml_test.go"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}

	task := plan.Tasks[0]
	if task.CommitSpec == nil {
		t.Fatal("expected CommitSpec to be populated")
	}

	if task.CommitSpec.Type != "feat" {
		t.Errorf("expected type 'feat', got %q", task.CommitSpec.Type)
	}
	if task.CommitSpec.Message != "add new feature" {
		t.Errorf("expected message 'add new feature', got %q", task.CommitSpec.Message)
	}
	if task.CommitSpec.Body != "This adds a comprehensive new feature." {
		t.Errorf("expected body 'This adds a comprehensive new feature.', got %q", task.CommitSpec.Body)
	}
	if len(task.CommitSpec.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(task.CommitSpec.Files))
	}
	if len(task.CommitSpec.Files) > 0 && task.CommitSpec.Files[0] != "internal/parser/yaml.go" {
		t.Errorf("expected first file 'internal/parser/yaml.go', got %q", task.CommitSpec.Files[0])
	}
}

func TestYAMLParser_CommitSpec_MinimalValid(t *testing.T) {
	yamlContent := `
plan:
  metadata:
    feature_name: "Minimal Commit Test"
  tasks:
    - task_number: 1
      name: "Task with minimal commit spec"
      depends_on: []
      estimated_time: "30m"
      description: "Test task"
      commit:
        message: "fix bug"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if task.CommitSpec == nil {
		t.Fatal("expected CommitSpec to be populated")
	}

	if task.CommitSpec.Type != "" {
		t.Errorf("expected empty type, got %q", task.CommitSpec.Type)
	}
	if task.CommitSpec.Message != "fix bug" {
		t.Errorf("expected message 'fix bug', got %q", task.CommitSpec.Message)
	}
}

func TestYAMLParser_CommitSpec_MissingMessage(t *testing.T) {
	yamlContent := `
plan:
  metadata:
    feature_name: "Invalid Commit Test"
  tasks:
    - task_number: 1
      name: "Task with invalid commit spec"
      depends_on: []
      estimated_time: "30m"
      description: "Test task"
      commit:
        type: "feat"
        files:
          - "some/file.go"
`
	parser := NewYAMLParser()
	_, err := parser.Parse(strings.NewReader(yamlContent))
	if err == nil {
		t.Error("expected error for commit spec without message")
	}
	if !strings.Contains(err.Error(), "commit message is required") {
		t.Errorf("expected error about missing message, got: %v", err)
	}
}

func TestYAMLParser_CommitSpec_BackwardCompatible(t *testing.T) {
	// Tasks without commit section should work and have nil CommitSpec
	yamlContent := `
plan:
  metadata:
    feature_name: "No Commit Test"
  tasks:
    - task_number: 1
      name: "Task without commit"
      depends_on: []
      estimated_time: "30m"
      description: "Legacy task"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if task.CommitSpec != nil {
		t.Errorf("expected nil CommitSpec for task without commit section, got %+v", task.CommitSpec)
	}
}

func TestYAMLParser_CommitSpec_EmptySection(t *testing.T) {
	// Empty commit section should result in nil CommitSpec (backward compat)
	yamlContent := `
plan:
  metadata:
    feature_name: "Empty Commit Test"
  tasks:
    - task_number: 1
      name: "Task with empty commit"
      depends_on: []
      estimated_time: "30m"
      description: "Task"
      commit:
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	if task.CommitSpec != nil {
		t.Errorf("expected nil CommitSpec for empty commit section, got %+v", task.CommitSpec)
	}
}

func TestYAMLParser_CommitSpec_InPrompt(t *testing.T) {
	// Verify commit section appears in generated prompt
	yamlContent := `
plan:
  metadata:
    feature_name: "Commit Prompt Test"
  tasks:
    - task_number: 1
      name: "Task with commit in prompt"
      depends_on: []
      estimated_time: "30m"
      description: "Test task"
      commit:
        type: "fix"
        message: "resolve memory leak"
        files:
          - "internal/cache.go"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	// XML format: <commit> section
	if !strings.Contains(task.Prompt, "<commit>") {
		t.Error("expected prompt to contain '<commit>' section")
	}
	if !strings.Contains(task.Prompt, "<type>fix</type>") {
		t.Error("expected prompt to contain commit type 'fix' as XML element")
	}
	if !strings.Contains(task.Prompt, "resolve memory leak") {
		t.Error("expected prompt to contain commit message")
	}
	if !strings.Contains(task.Prompt, "<file>internal/cache.go</file>") {
		t.Error("expected prompt to contain commit file as XML element")
	}
}

func TestYAMLParser_CommitSpec_MandatoryCommitSection(t *testing.T) {
	// Verify MANDATORY COMMIT section with imperative instructions
	yamlContent := `
plan:
  metadata:
    feature_name: "Mandatory Commit Test"
  tasks:
    - task_number: 1
      name: "Task with mandatory commit"
      depends_on: []
      estimated_time: "30m"
      description: "Test task"
      commit:
        type: "feat"
        message: "add CommitExecutor interface"
        files:
          - "internal/executor/commit.go"
          - "internal/executor/commit_test.go"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	task := plan.Tasks[0]
	prompt := task.Prompt

	// Verify MANDATORY COMMIT section exists (XML format)
	if !strings.Contains(prompt, "<mandatory_commit required=\"true\">") {
		t.Error("expected prompt to contain '<mandatory_commit required=\"true\">' section")
	}

	// Verify imperative instruction
	if !strings.Contains(prompt, "you MUST commit them to git") {
		t.Error("expected prompt to contain imperative commit instruction")
	}

	// Verify exact commit message format (feat: add CommitExecutor interface)
	if !strings.Contains(prompt, "feat: add CommitExecutor interface") {
		t.Error("expected prompt to contain formatted commit message 'feat: add CommitExecutor interface'")
	}

	// Verify files are listed for staging (XML format)
	if !strings.Contains(prompt, "<file>internal/executor/commit.go</file>") {
		t.Error("expected prompt to contain first commit file as XML element")
	}
	if !strings.Contains(prompt, "<file>internal/executor/commit_test.go</file>") {
		t.Error("expected prompt to contain second commit file as XML element")
	}

	// Verify git add command example (in XML code block)
	if !strings.Contains(prompt, "git add") {
		t.Error("expected prompt to contain 'git add' command")
	}

	// Verify git commit command example
	if !strings.Contains(prompt, "git commit -m") {
		t.Error("expected prompt to contain 'git commit -m' command")
	}

	// Verify warning about task completion (XML warning element)
	if !strings.Contains(prompt, "<warning>Your task is NOT complete until changes are committed.</warning>") {
		t.Error("expected prompt to contain task completion warning as XML element")
	}
}

func TestYAMLParser_CommitSpec_MandatoryCommitWithoutType(t *testing.T) {
	// Verify MANDATORY COMMIT section works without type prefix
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Task without type"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      commit:
        message: "quick fix for edge case"
        files:
          - "internal/parser/yaml.go"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	prompt := plan.Tasks[0].Prompt

	// Should contain message without type prefix
	if !strings.Contains(prompt, "quick fix for edge case") {
		t.Error("expected prompt to contain raw commit message")
	}
	if !strings.Contains(prompt, "git commit -m \"quick fix for edge case\"") {
		t.Error("expected git commit command with raw message (no type prefix)")
	}
}

func TestYAMLParser_CommitSpec_MandatoryCommitNoFiles(t *testing.T) {
	// Verify MANDATORY COMMIT section works without explicit file list
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Task without files"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      commit:
        type: "docs"
        message: "update documentation"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	prompt := plan.Tasks[0].Prompt

	// Should have MANDATORY COMMIT section (XML format)
	if !strings.Contains(prompt, "<mandatory_commit required=\"true\">") {
		t.Error("expected '<mandatory_commit required=\"true\">' section even without files")
	}

	// Should have generic file staging instruction
	if !strings.Contains(prompt, "Stage your modified files") {
		t.Error("expected generic file staging instruction when no files specified")
	}

	// Should have proper commit message
	if !strings.Contains(prompt, "docs: update documentation") {
		t.Error("expected formatted commit message")
	}
}

func TestYAMLParser_CommitSpec_NoMandatoryCommitWithoutSpec(t *testing.T) {
	// Verify MANDATORY COMMIT section is NOT present when no commit spec
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Task without commit"
      depends_on: []
      estimated_time: "30m"
      description: "Legacy task"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	prompt := plan.Tasks[0].Prompt

	// Should NOT have MANDATORY COMMIT section
	if strings.Contains(prompt, "MANDATORY COMMIT") {
		t.Error("expected NO MANDATORY COMMIT section when commit spec is absent")
	}
}

func TestYAMLParser_CommitSpec_AllConventionalTypes(t *testing.T) {
	types := []string{"feat", "fix", "docs", "style", "refactor", "test", "chore", "perf", "ci", "build", "revert"}

	for _, commitType := range types {
		t.Run(commitType, func(t *testing.T) {
			yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Test Task"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      commit:
        type: "` + commitType + `"
        message: "test message"
`
			parser := NewYAMLParser()
			plan, err := parser.Parse(strings.NewReader(yamlContent))
			if err != nil {
				t.Fatalf("Failed to parse with type %q: %v", commitType, err)
			}

			if plan.Tasks[0].CommitSpec == nil {
				t.Fatalf("expected CommitSpec for type %q", commitType)
			}
			if plan.Tasks[0].CommitSpec.Type != commitType {
				t.Errorf("expected type %q, got %q", commitType, plan.Tasks[0].CommitSpec.Type)
			}
		})
	}
}

func TestYAMLParser_CommitSpec_WithGlobPatterns(t *testing.T) {
	yamlContent := `
plan:
  tasks:
    - task_number: 1
      name: "Glob Pattern Test"
      depends_on: []
      estimated_time: "30m"
      description: "Test"
      commit:
        type: "refactor"
        message: "restructure parser"
        files:
          - "internal/parser/*.go"
          - "internal/parser/**/*_test.go"
`
	parser := NewYAMLParser()
	plan, err := parser.Parse(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if plan.Tasks[0].CommitSpec == nil {
		t.Fatal("expected CommitSpec")
	}
	if len(plan.Tasks[0].CommitSpec.Files) != 2 {
		t.Errorf("expected 2 file patterns, got %d", len(plan.Tasks[0].CommitSpec.Files))
	}
	if plan.Tasks[0].CommitSpec.Files[0] != "internal/parser/*.go" {
		t.Errorf("expected glob pattern preserved, got %q", plan.Tasks[0].CommitSpec.Files[0])
	}
}

// mustParseYAMLFile is a test helper that parses a YAML file and fails on error
func mustParseYAMLFile(t *testing.T, path string) *models.Plan {
	t.Helper()
	parser := NewYAMLParser()
	plan, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	return plan
}

// TestRuntimeEnforcementPlanFixture verifies the runtime enforcement test fixture parses correctly
// and contains all expected enforcement metadata blocks.
func TestRuntimeEnforcementPlanFixture(t *testing.T) {
	plan := mustParseYAMLFile(t, "testdata/runtime_enforcement.yaml")

	// Verify basic plan structure
	if len(plan.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(plan.Tasks))
	}

	// Verify PlannerComplianceSpec is parsed
	if plan.PlannerCompliance == nil {
		t.Fatal("expected PlannerComplianceSpec to be parsed")
	}
	if plan.PlannerCompliance.PlannerVersion != "2.9.0" {
		t.Errorf("expected planner version 2.9.0, got %s", plan.PlannerCompliance.PlannerVersion)
	}
	if !plan.PlannerCompliance.StrictEnforcement {
		t.Error("expected strict_enforcement to be true")
	}
	expectedFeatures := []string{"dependency_checks", "test_commands", "documentation_targets", "success_criteria", "package_guard"}
	for _, feature := range expectedFeatures {
		found := false
		for _, f := range plan.PlannerCompliance.RequiredFeatures {
			if f == feature {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected required feature %s not found", feature)
		}
	}

	// Verify DataFlowRegistry is parsed
	if plan.DataFlowRegistry == nil {
		t.Fatal("expected DataFlowRegistry to be parsed")
	}
	if len(plan.DataFlowRegistry.Producers) == 0 {
		t.Error("expected producers in DataFlowRegistry")
	}
	if len(plan.DataFlowRegistry.Consumers) == 0 {
		t.Error("expected consumers in DataFlowRegistry")
	}
	if len(plan.DataFlowRegistry.DocumentationTargets) == 0 {
		t.Error("expected documentation_targets in DataFlowRegistry")
	}

	// Verify Task 1: Full runtime metadata
	task1 := plan.Tasks[0]
	if task1.Number != "1" {
		t.Errorf("expected task 1, got %s", task1.Number)
	}
	if task1.RuntimeMetadata == nil {
		t.Fatal("task 1 should have RuntimeMetadata")
	}
	if len(task1.RuntimeMetadata.DependencyChecks) != 3 {
		t.Errorf("task 1: expected 3 dependency checks, got %d", len(task1.RuntimeMetadata.DependencyChecks))
	}
	if len(task1.RuntimeMetadata.DocumentationTargets) != 2 {
		t.Errorf("task 1: expected 2 doc targets, got %d", len(task1.RuntimeMetadata.DocumentationTargets))
	}
	if len(task1.RuntimeMetadata.PromptBlocks) != 3 {
		t.Errorf("task 1: expected 3 prompt blocks, got %d", len(task1.RuntimeMetadata.PromptBlocks))
	}
	if len(task1.TestCommands) != 2 {
		t.Errorf("task 1: expected 2 test commands, got %d", len(task1.TestCommands))
	}

	// Verify Task 2: Minimal runtime metadata (empty arrays)
	task2 := plan.Tasks[1]
	if task2.Number != "2" {
		t.Errorf("expected task 2, got %s", task2.Number)
	}
	if task2.RuntimeMetadata == nil {
		t.Fatal("task 2 should have RuntimeMetadata")
	}
	if len(task2.RuntimeMetadata.DependencyChecks) != 1 {
		t.Errorf("task 2: expected 1 dependency check, got %d", len(task2.RuntimeMetadata.DependencyChecks))
	}
	if len(task2.RuntimeMetadata.DocumentationTargets) != 0 {
		t.Errorf("task 2: expected 0 doc targets (empty array), got %d", len(task2.RuntimeMetadata.DocumentationTargets))
	}
	if len(task2.RuntimeMetadata.PromptBlocks) != 0 {
		t.Errorf("task 2: expected 0 prompt blocks (empty array), got %d", len(task2.RuntimeMetadata.PromptBlocks))
	}

	// Verify Task 3: Integration task with dual criteria
	task3 := plan.Tasks[2]
	if task3.Number != "3" {
		t.Errorf("expected task 3, got %s", task3.Number)
	}
	if task3.Type != "integration" {
		t.Errorf("task 3: expected type 'integration', got '%s'", task3.Type)
	}
	if len(task3.SuccessCriteria) < 1 {
		t.Error("task 3: expected success criteria")
	}
	if len(task3.IntegrationCriteria) != 5 {
		t.Errorf("task 3: expected 5 integration criteria, got %d", len(task3.IntegrationCriteria))
	}

	// Verify structured criteria with verification blocks in Task 1
	if len(task1.StructuredCriteria) > 0 {
		var foundVerification bool
		for _, sc := range task1.StructuredCriteria {
			if sc.Verification != nil && sc.Verification.Command != "" {
				foundVerification = true
				break
			}
		}
		if foundVerification {
			t.Log("Found criterion with verification block in task 1")
		}
	}
}
