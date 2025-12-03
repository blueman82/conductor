package agent

import (
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// TestValidateTaskAgents_MissingAgent tests detection of missing task agents
func TestValidateTaskAgents_MissingAgent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry with some agents
	registry := NewRegistry(tmpDir)
	registry.agents["python-pro"] = &Agent{Name: "python-pro", Description: "Python agent"}
	registry.agents["golang-pro"] = &Agent{Name: "golang-pro", Description: "Go agent"}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Agent: "python-pro", Prompt: "Do task"},      // OK
		{Number: "2", Name: "Task 2", Agent: "nonexistent", Prompt: "Do task"},     // FAIL
		{Number: "3", Name: "Task 3", Agent: "", Prompt: "Do task"},                // OK (no agent)
		{Number: "4", Name: "Task 4", Agent: "another-missing", Prompt: "Do task"}, // FAIL
	}

	errors := ValidateTaskAgents(tasks, registry)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Check first error
	if errors[0].AgentName != "nonexistent" {
		t.Errorf("Expected agent name 'nonexistent', got '%s'", errors[0].AgentName)
	}
	if errors[0].UsageType != "task" {
		t.Errorf("Expected usage type 'task', got '%s'", errors[0].UsageType)
	}
	if errors[0].TaskNumber != 2 {
		t.Errorf("Expected task number 2, got %d", errors[0].TaskNumber)
	}

	// Check second error
	if errors[1].AgentName != "another-missing" {
		t.Errorf("Expected agent name 'another-missing', got '%s'", errors[1].AgentName)
	}
	if errors[1].TaskNumber != 4 {
		t.Errorf("Expected task number 4, got %d", errors[1].TaskNumber)
	}

	// Check error message includes task number
	errMsg := errors[0].Error()
	if !strings.Contains(errMsg, "task 2") {
		t.Errorf("Error message should contain 'task 2', got: %s", errMsg)
	}

	// Check error message includes available agents
	if !strings.Contains(errMsg, "golang-pro") || !strings.Contains(errMsg, "python-pro") {
		t.Errorf("Error message should list available agents, got: %s", errMsg)
	}
}

// TestValidateTaskAgents_AllValid tests when all task agents exist
func TestValidateTaskAgents_AllValid(t *testing.T) {
	tmpDir := t.TempDir()

	registry := NewRegistry(tmpDir)
	registry.agents["python-pro"] = &Agent{Name: "python-pro", Description: "Python agent"}
	registry.agents["golang-pro"] = &Agent{Name: "golang-pro", Description: "Go agent"}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Agent: "python-pro", Prompt: "Do task"},
		{Number: "2", Name: "Task 2", Agent: "golang-pro", Prompt: "Do task"},
		{Number: "3", Name: "Task 3", Agent: "", Prompt: "Do task"}, // No agent OK
	}

	errors := ValidateTaskAgents(tasks, registry)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

// TestValidateTaskAgents_NoAgents tests when no task agents are specified
func TestValidateTaskAgents_NoAgents(t *testing.T) {
	tmpDir := t.TempDir()

	registry := NewRegistry(tmpDir)
	registry.agents["python-pro"] = &Agent{Name: "python-pro", Description: "Python agent"}

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Agent: "", Prompt: "Do task"},
		{Number: "2", Name: "Task 2", Agent: "", Prompt: "Do task"},
	}

	errors := ValidateTaskAgents(tasks, registry)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors when no agents specified, got %d", len(errors))
	}
}

// TestValidateTaskAgents_EmptyRegistry tests with empty agent registry
func TestValidateTaskAgents_EmptyRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)

	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Agent: "python-pro", Prompt: "Do task"},
		{Number: "2", Name: "Task 2", Agent: "golang-pro", Prompt: "Do task"},
	}

	errors := ValidateTaskAgents(tasks, registry)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors for empty registry, got %d", len(errors))
	}

	// Check error message indicates no agents available
	errMsg := errors[0].Error()
	if !strings.Contains(errMsg, "No agents found in registry") {
		t.Errorf("Error should mention no agents available, got: %s", errMsg)
	}
}

// TestValidateTaskAgents_NilRegistry tests with nil registry
func TestValidateTaskAgents_NilRegistry(t *testing.T) {
	tasks := []models.Task{
		{Number: "1", Name: "Task 1", Agent: "python-pro", Prompt: "Do task"},
	}

	errors := ValidateTaskAgents(tasks, nil)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors with nil registry, got %d", len(errors))
	}
}

// TestValidateTaskAgents_StringNumberParsing tests numeric string parsing in error message
func TestValidateTaskAgents_StringNumberParsing(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)

	tests := []struct {
		name     string
		number   string
		expected int
	}{
		{"numeric string", "42", 42},
		{"zero", "0", 0},
		{"non-numeric", "abc", 0}, // Non-numeric strings parse to 0
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []models.Task{
				{Number: tt.number, Name: "Task", Agent: "missing", Prompt: "Do task"},
			}

			errors := ValidateTaskAgents(tasks, registry)

			if len(errors) != 1 {
				t.Fatalf("Expected 1 error, got %d", len(errors))
			}

			if errors[0].TaskNumber != tt.expected {
				t.Errorf("Expected task number %d, got %d", tt.expected, errors[0].TaskNumber)
			}
		})
	}
}

// TestValidateQCAgents_AllMissing tests when all QC agents are missing
func TestValidateQCAgents_AllMissing(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)
	// Empty registry - no agents

	qcAgents := []string{"code-reviewer", "test-automator"}

	errors := ValidateQCAgents(qcAgents, registry)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Check first error
	if errors[0].AgentName != "code-reviewer" {
		t.Errorf("Expected agent name 'code-reviewer', got '%s'", errors[0].AgentName)
	}
	if errors[0].UsageType != "quality_control" {
		t.Errorf("Expected usage type 'quality_control', got '%s'", errors[0].UsageType)
	}

	// Check error message
	errMsg := errors[0].Error()
	if !strings.Contains(errMsg, "quality_control.agents config") {
		t.Errorf("Error message should mention QC config, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "No agents found in registry") {
		t.Errorf("Error message should mention no agents available, got: %s", errMsg)
	}
}

// TestValidateQCAgents_SomeMissing tests when some QC agents exist and some are missing
func TestValidateQCAgents_SomeMissing(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)
	registry.agents["code-reviewer"] = &Agent{Name: "code-reviewer", Description: "Code reviewer"}

	qcAgents := []string{"code-reviewer", "test-automator", "doc-generator"}

	errors := ValidateQCAgents(qcAgents, registry)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors for missing agents, got %d", len(errors))
	}

	// Check error names
	missingNames := make(map[string]bool)
	for _, err := range errors {
		missingNames[err.AgentName] = true
	}

	if !missingNames["test-automator"] || !missingNames["doc-generator"] {
		t.Errorf("Expected test-automator and doc-generator to be missing")
	}

	// Check available agents listed
	errMsg := errors[0].Error()
	if !strings.Contains(errMsg, "code-reviewer") {
		t.Errorf("Available agents should include code-reviewer, got: %s", errMsg)
	}
}

// TestValidateQCAgents_AllValid tests when all QC agents exist
func TestValidateQCAgents_AllValid(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)
	registry.agents["code-reviewer"] = &Agent{Name: "code-reviewer", Description: "Code reviewer"}
	registry.agents["test-automator"] = &Agent{Name: "test-automator", Description: "Test automator"}

	qcAgents := []string{"code-reviewer", "test-automator"}

	errors := ValidateQCAgents(qcAgents, registry)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors when all agents exist, got %d", len(errors))
	}
}

// TestValidateQCAgents_EmptyAgentList tests with empty QC agent list
func TestValidateQCAgents_EmptyAgentList(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)

	qcAgents := []string{}

	errors := ValidateQCAgents(qcAgents, registry)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors for empty agent list, got %d", len(errors))
	}
}

// TestValidateQCAgents_SkipsEmptyAgentNames tests that empty agent names are skipped
func TestValidateQCAgents_SkipsEmptyAgentNames(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)

	qcAgents := []string{"", "code-reviewer", "", "missing-agent"}

	errors := ValidateQCAgents(qcAgents, registry)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors (empty names skipped), got %d", len(errors))
	}

	// Verify empty strings weren't reported as errors
	for _, err := range errors {
		if err.AgentName == "" {
			t.Error("Empty agent names should be skipped")
		}
	}
}

// TestValidateQCAgents_NilRegistry tests with nil registry
func TestValidateQCAgents_NilRegistry(t *testing.T) {
	qcAgents := []string{"code-reviewer"}

	errors := ValidateQCAgents(qcAgents, nil)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors with nil registry, got %d", len(errors))
	}
}

// TestValidationError_ErrorMessage tests error message formatting for task agents
func TestValidationError_ErrorMessage(t *testing.T) {
	err := &ValidationError{
		AgentName:  "python-pro",
		UsageType:  "task",
		TaskNumber: 5,
		Available:  []string{"golang-pro", "rust-pro"},
	}

	msg := err.Error()

	// Check message contains agent name
	if !strings.Contains(msg, "python-pro") {
		t.Errorf("Error message should contain agent name, got: %s", msg)
	}

	// Check message mentions task reference
	if !strings.Contains(msg, "task 5") {
		t.Errorf("Error message should mention task 5, got: %s", msg)
	}

	// Check message lists available agents
	if !strings.Contains(msg, "golang-pro") || !strings.Contains(msg, "rust-pro") {
		t.Errorf("Error message should list available agents, got: %s", msg)
	}
}

// TestValidationError_ErrorMessageQC tests error message formatting for QC agents
func TestValidationError_ErrorMessageQC(t *testing.T) {
	err := &ValidationError{
		AgentName: "code-reviewer",
		UsageType: "quality_control",
		Available: []string{"reviewer-1", "reviewer-2"},
	}

	msg := err.Error()

	// Check message contains agent name
	if !strings.Contains(msg, "code-reviewer") {
		t.Errorf("Error message should contain agent name, got: %s", msg)
	}

	// Check message mentions QC config
	if !strings.Contains(msg, "quality_control.agents config") {
		t.Errorf("Error message should mention QC config, got: %s", msg)
	}

	// Check message lists available agents
	if !strings.Contains(msg, "reviewer-1") || !strings.Contains(msg, "reviewer-2") {
		t.Errorf("Error message should list available agents, got: %s", msg)
	}
}

// TestValidationError_NoAvailableAgents tests error message when no agents are available
func TestValidationError_NoAvailableAgents(t *testing.T) {
	err := &ValidationError{
		AgentName: "python-pro",
		UsageType: "task",
		TaskNumber: 1,
		Available: []string{},
	}

	msg := err.Error()

	if !strings.Contains(msg, "No agents found in registry") {
		t.Errorf("Error message should indicate no agents, got: %s", msg)
	}

	if !strings.Contains(msg, "~/.claude/agents/") {
		t.Errorf("Error message should suggest agents directory, got: %s", msg)
	}
}

// TestValidationError_TaskNumberZero tests task number 0 handling
func TestValidationError_TaskNumberZero(t *testing.T) {
	err := &ValidationError{
		AgentName:  "agent",
		UsageType:  "task",
		TaskNumber: 0, // Zero task number
		Available:  []string{},
	}

	msg := err.Error()

	// Should still mention task 0
	if !strings.Contains(msg, "task 0") {
		t.Errorf("Error message should mention task 0, got: %s", msg)
	}
}

// TestValidateTaskAgents_TableDriven comprehensive table-driven test
func TestValidateTaskAgents_TableDriven(t *testing.T) {
	type args struct {
		taskAgent     string
		registryAgents []string
	}

	type want struct {
		shouldError bool
		errorAgent  string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "task agent exists",
			args: args{taskAgent: "python-pro", registryAgents: []string{"python-pro", "golang-pro"}},
			want: want{shouldError: false},
		},
		{
			name: "task agent missing",
			args: args{taskAgent: "missing", registryAgents: []string{"python-pro", "golang-pro"}},
			want: want{shouldError: true, errorAgent: "missing"},
		},
		{
			name: "task agent empty",
			args: args{taskAgent: "", registryAgents: []string{"python-pro"}},
			want: want{shouldError: false},
		},
		{
			name: "empty registry with agent",
			args: args{taskAgent: "python-pro", registryAgents: []string{}},
			want: want{shouldError: true, errorAgent: "python-pro"},
		},
		{
			name: "case sensitive agent names",
			args: args{taskAgent: "Python-Pro", registryAgents: []string{"python-pro"}},
			want: want{shouldError: true, errorAgent: "Python-Pro"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry(t.TempDir())
			for _, agentName := range tt.args.registryAgents {
				registry.agents[agentName] = &Agent{Name: agentName}
			}

			tasks := []models.Task{
				{Number: "1", Name: "Test", Agent: tt.args.taskAgent, Prompt: "Do task"},
			}

			errors := ValidateTaskAgents(tasks, registry)

			if tt.want.shouldError && len(errors) == 0 {
				t.Error("Expected validation error, got none")
			}
			if !tt.want.shouldError && len(errors) != 0 {
				t.Errorf("Expected no error, got %d", len(errors))
			}
			if tt.want.shouldError && len(errors) > 0 && errors[0].AgentName != tt.want.errorAgent {
				t.Errorf("Expected error for '%s', got '%s'", tt.want.errorAgent, errors[0].AgentName)
			}
		})
	}
}

// TestValidateQCAgents_TableDriven comprehensive table-driven test
func TestValidateQCAgents_TableDriven(t *testing.T) {
	type args struct {
		qcAgents       []string
		registryAgents []string
	}

	type want struct {
		errorCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "all agents exist",
			args: args{qcAgents: []string{"reviewer", "automator"}, registryAgents: []string{"reviewer", "automator"}},
			want: want{errorCount: 0},
		},
		{
			name: "some agents missing",
			args: args{qcAgents: []string{"reviewer", "missing"}, registryAgents: []string{"reviewer"}},
			want: want{errorCount: 1},
		},
		{
			name: "all agents missing",
			args: args{qcAgents: []string{"reviewer", "automator"}, registryAgents: []string{}},
			want: want{errorCount: 2},
		},
		{
			name: "empty agent list",
			args: args{qcAgents: []string{}, registryAgents: []string{"reviewer"}},
			want: want{errorCount: 0},
		},
		{
			name: "mixed with empty strings",
			args: args{qcAgents: []string{"", "reviewer", ""}, registryAgents: []string{}},
			want: want{errorCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry(t.TempDir())
			for _, agentName := range tt.args.registryAgents {
				registry.agents[agentName] = &Agent{Name: agentName}
			}

			errors := ValidateQCAgents(tt.args.qcAgents, registry)

			if len(errors) != tt.want.errorCount {
				t.Errorf("Expected %d errors, got %d", tt.want.errorCount, len(errors))
			}
		})
	}
}

// TestValidationError_Implements_error tests that ValidationError implements error interface
func TestValidationError_Implements_error(t *testing.T) {
	err := &ValidationError{
		AgentName: "test",
		UsageType: "task",
	}

	// Should be able to use as error
	var e error = err
	if e.Error() == "" {
		t.Error("Error() should not return empty string")
	}
}
