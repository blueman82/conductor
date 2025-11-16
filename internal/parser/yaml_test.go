package parser

import (
	"strings"
	"testing"
	"time"
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

func TestParseYAMLWithQCAgentConfig(t *testing.T) {
	tests := []struct {
		name              string
		yamlContent       string
		expectedMode      string
		expectedExplicit  []string
		expectedAdditional []string
		expectedBlocked   []string
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
			expectedMode:      "auto",
			expectedBlocked:   []string{"deprecated-agent"},
			expectedExplicit:  []string{},
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
			expectedMode:      "explicit",
			expectedExplicit:  []string{"golang-pro", "code-reviewer"},
			expectedAdditional: []string{},
			expectedBlocked:   []string{},
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
			expectedMode:      "mixed",
			expectedExplicit:  []string{},
			expectedAdditional: []string{"security-auditor", "performance-reviewer"},
			expectedBlocked:   []string{"old-agent"},
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
			expectedMode:      "",
			expectedExplicit:  []string{},
			expectedAdditional: []string{},
			expectedBlocked:   []string{},
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
