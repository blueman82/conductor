package updater

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/filelock"
	"github.com/harrison/conductor/internal/parser"
)

func TestUpdateMarkdownTaskStatusCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")

	markdown := `# Test Plan
- [x] Task 11: Implement File Locking (status: completed)
- [ ] Task 12: Implement Plan Updater (status: pending)
`

	writeFile(t, planPath, markdown)

	completedAt := time.Date(2025, time.November, 8, 0, 0, 0, 0, time.UTC)

	if err := UpdateTaskStatus(planPath, "12", "completed", &completedAt); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	content := readFile(t, planPath)

	if !strings.Contains(content, "- [x] Task 12") {
		t.Fatalf("expected Task 12 checkbox to be checked, got:\n%s", content)
	}

	if !strings.Contains(content, "status: completed") {
		t.Fatalf("expected Task 12 status to be updated, got:\n%s", content)
	}
}

func TestUpdateMarkdownAddsStatusWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")

	markdown := `## Task List
- [ ] Task 12: Implement Plan Updater`
	writeFile(t, planPath, markdown)

	if err := UpdateTaskStatus(planPath, "12", "in-progress", nil); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	content := readFile(t, planPath)
	if !strings.Contains(content, "status: in-progress") {
		t.Fatalf("expected status annotation to be appended, got:\n%s", content)
	}
}

func TestUpdateMarkdownAlternateBullets(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")

	markdown := `# Plan
* [ ] Task 12: Implement Plan Updater`
	writeFile(t, planPath, markdown)

	if err := UpdateTaskStatus(planPath, "12", "completed", nil); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	content := readFile(t, planPath)
	if !strings.Contains(content, "* [x] Task 12") {
		t.Fatalf("expected checkbox to be toggled, got:\n%s", content)
	}
}

func TestUpdateMarkdownUnicode(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")

	markdown := `- [ ] Task 12: Handle emojis 🚀`
	writeFile(t, planPath, markdown)

	if err := UpdateTaskStatus(planPath, "12", "completed", nil); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	content := readFile(t, planPath)
	if !strings.Contains(content, "🚀") {
		t.Fatalf("expected unicode content to be preserved, got:\n%s", content)
	}
}

func TestUpdateMarkdownTaskNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")

	writeFile(t, planPath, "- [ ] Task 5: Something else")

	err := UpdateTaskStatus(planPath, "12", "completed", nil)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateYAMLTaskStatusCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")

	yamlContent := `plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 11
      name: "Task 11"
      status: "completed"
      completed_at: "2025-11-07"
    - task_number: 12
      name: "Task 12"
      status: "pending"
`
	writeFile(t, planPath, yamlContent)

	completedAt := time.Date(2025, time.November, 8, 0, 0, 0, 0, time.UTC)

	if err := UpdateTaskStatus(planPath, "12", "completed", &completedAt); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	content := readFile(t, planPath)

	if !strings.Contains(content, "status: \"completed\"") {
		t.Fatalf("expected YAML status to be updated, got:\n%s", content)
	}

	if !strings.Contains(content, "completed_date: \"2025-11-08\"") {
		t.Fatalf("expected completed_date to be set, got:\n%s", content)
	}
}

func TestUpdateYAMLTaskStatusReopenRemovesCompletedDate(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")

	yamlContent := `plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 12
      name: "Task 12"
      status: "completed"
      completed_date: "2025-11-08"
`
	writeFile(t, planPath, yamlContent)

	if err := UpdateTaskStatus(planPath, "12", "pending", nil); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	content := readFile(t, planPath)

	if !strings.Contains(content, "status: \"pending\"") {
		t.Fatalf("expected YAML status to be updated to pending, got:\n%s", content)
	}

	if strings.Contains(content, "completed_date") {
		t.Fatalf("expected completed_date to be removed, got:\n%s", content)
	}
}

func TestUpdateTaskStatusUnknownTask(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")

	yamlContent := `plan:
  metadata:
    feature_name: "Test Plan"
  tasks:
    - task_number: 5
      name: "Task 5"
      status: "pending"
`
	writeFile(t, planPath, yamlContent)

	err := UpdateTaskStatus(planPath, "12", "completed", nil)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateTaskStatusUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.txt")
	writeFile(t, planPath, "irrelevant")

	err := UpdateTaskStatus(planPath, "12", "completed", nil)
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestUpdateTaskStatusInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	writeFile(t, planPath, "not: [valid")

	err := UpdateTaskStatus(planPath, "12", "completed", nil)
	if !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestUpdateTaskStatusPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 000 semantics differ on Windows")
	}

	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	writeFile(t, planPath, "- [ ] Task 12: Pending")

	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatalf("failed to chmod directory: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	err := UpdateTaskStatus(planPath, "12", "completed", nil)
	if err == nil {
		t.Fatal("expected permission error, got nil")
	}
	if !errors.Is(err, fs.ErrPermission) && !os.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}
}

func TestUpdateTaskStatusMonitorReceivesMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	writeFile(t, planPath, `plan:
  tasks:
    - task_number: 12
      name: "Task 12"
      status: "pending"
`)

	metricsCh := make(chan UpdateMetrics, 1)
	monitor := func(metrics UpdateMetrics) {
		metricsCh <- metrics
	}

	if err := UpdateTaskStatus(planPath, "12", "completed", nil, WithMonitor(monitor)); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	select {
	case metrics := <-metricsCh:
		if metrics.Format != parser.FormatYAML {
			t.Fatalf("expected YAML format, got %v", metrics.Format)
		}
		if metrics.OldStatus != "pending" || metrics.NewStatus != "completed" {
			t.Fatalf("unexpected statuses: %+v", metrics)
		}
		if metrics.BytesRead == 0 || metrics.BytesWritten == 0 {
			t.Fatalf("expected bytes metrics to be populated: %+v", metrics)
		}
		if metrics.Err != nil {
			t.Fatalf("expected nil error in metrics, got %v", metrics.Err)
		}
	case <-time.After(time.Second):
		t.Fatal("monitor did not receive metrics")
	}
}

func TestUpdateTaskStatusMonitorReceivesErrorMetrics(t *testing.T) {
	metricsCh := make(chan UpdateMetrics, 1)
	monitor := func(metrics UpdateMetrics) {
		metricsCh <- metrics
	}

	err := UpdateTaskStatus("plan.txt", "12", "completed", nil, WithMonitor(monitor))
	if err == nil {
		t.Fatal("expected error")
	}

	select {
	case metrics := <-metricsCh:
		if metrics.Err == nil {
			t.Fatal("expected error in metrics")
		}
		if !errors.Is(metrics.Err, ErrUnsupportedFormat) {
			t.Fatalf("expected ErrUnsupportedFormat, got %v", metrics.Err)
		}
	case <-time.After(time.Second):
		t.Fatal("monitor did not receive error metrics")
	}
}

func TestUpdateTaskStatusTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	writeFile(t, planPath, "- [ ] Task 12: Pending")

	lock := filelock.NewFileLock(planPath + ".lock")
	if err := lock.Lock(); err != nil {
		t.Fatalf("failed to pre-lock: %v", err)
	}
	defer lock.Unlock()

	err := UpdateTaskStatus(planPath, "12", "completed", nil, WithTimeout(100*time.Millisecond))
	if !errors.Is(err, filelock.ErrLockTimeout) {
		t.Fatalf("expected lock timeout error, got %v", err)
	}
}

func TestConcurrentMarkdownUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	writeFile(t, planPath, `# Plan
- [ ] Task 12: Implement Plan Updater (status: pending)
`)

	statuses := []string{"in-progress", "completed", "blocked", "completed"}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		status := statuses[i%len(statuses)]
		go func() {
			defer wg.Done()
			if err := UpdateTaskStatus(planPath, "12", status, nil); err != nil {
				t.Errorf("UpdateTaskStatus failed: %v", err)
			}
		}()
	}

	wg.Wait()
	content := readFile(t, planPath)
	found := false
	for _, status := range statuses {
		if strings.Contains(content, "status: "+status) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected one of %v statuses, got:\n%s", statuses, content)
	}
}

func TestConcurrentYAMLUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	writeFile(t, planPath, `plan:
  tasks:
    - task_number: 12
      name: "Task 12"
      status: "pending"
`)

	statuses := []string{"pending", "completed", "review", "completed"}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		status := statuses[i%len(statuses)]
		delay := time.Duration(i*5) * time.Millisecond
		go func() {
			defer wg.Done()
			time.Sleep(delay)
			if err := UpdateTaskStatus(planPath, "12", status, nil); err != nil {
				t.Errorf("UpdateTaskStatus failed: %v", err)
			}
		}()
	}

	wg.Wait()
	content := readFile(t, planPath)
	found := false
	for _, status := range statuses {
		if strings.Contains(content, "status: \""+status+"\"") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected one of %v statuses, got:\n%s", statuses, content)
	}
}

func TestMonitorReceivesTimeoutMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	writeFile(t, planPath, "- [ ] Task 12: Pending")

	lock := filelock.NewFileLock(planPath + ".lock")
	if err := lock.Lock(); err != nil {
		t.Fatalf("failed to pre-lock: %v", err)
	}
	defer lock.Unlock()

	var count int32
	monitor := func(metrics UpdateMetrics) {
		atomic.AddInt32(&count, 1)
		if metrics.Err == nil {
			t.Errorf("expected error metrics in timeout case")
		}
	}

	UpdateTaskStatus(planPath, "12", "completed", nil, WithTimeout(50*time.Millisecond), WithMonitor(monitor))

	if atomic.LoadInt32(&count) == 0 {
		t.Fatal("expected monitor invocation")
	}
}

func TestUpdateTaskStatus_DeletesLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	lockPath := planPath + ".lock"

	markdown := `# Test Plan
- [ ] Task 12: Test lock cleanup`

	writeFile(t, planPath, markdown)

	completedAt := time.Date(2025, time.November, 10, 0, 0, 0, 0, time.UTC)
	err := UpdateTaskStatus(planPath, "12", "completed", &completedAt)
	if err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// Verify the plan was updated
	content := readFile(t, planPath)
	if !strings.Contains(content, "- [x] Task 12") {
		t.Fatalf("expected Task 12 checkbox to be checked, got:\n%s", content)
	}

	// Verify lock file was deleted
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("Lock file %s was not deleted", lockPath)
	}
}

func TestUpdateTaskStatus_DeletesLockFileYAML(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	lockPath := planPath + ".lock"

	yaml := `plan:
  tasks:
    - task_number: "12"
      name: "Test lock cleanup"
      status: "pending"`

	writeFile(t, planPath, yaml)

	completedAt := time.Date(2025, time.November, 10, 0, 0, 0, 0, time.UTC)
	err := UpdateTaskStatus(planPath, "12", "completed", &completedAt)
	if err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// Verify the plan was updated
	content := readFile(t, planPath)
	if !strings.Contains(content, `status: "completed"`) {
		t.Fatalf("expected Task 12 status to be completed, got:\n%s", content)
	}

	// Verify lock file was deleted
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("Lock file %s was not deleted", lockPath)
	}
}

func TestUpdateMarkdownFeedback(t *testing.T) {
	tests := []struct {
		name         string
		existingPlan string
		attempt      *ExecutionAttempt
		wantContains []string
	}{
		{
			name: "first attempt",
			existingPlan: `# Test Plan
- [ ] Task 12: Implement Feature

## Description
Task description here.`,
			attempt: &ExecutionAttempt{
				AttemptNumber: 1,
				Agent:         "python-pro",
				Verdict:       "RED",
				AgentOutput:   "Error: undefined variable",
				QCFeedback:    "Missing import statement",
				Timestamp:     time.Date(2025, 11, 15, 10, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"### Execution History",
				"#### Attempt 1",
				"Agent: python-pro",
				"Verdict: RED",
				"Error: undefined variable",
				"Missing import statement",
				"2025-11-15 10:00:00",
			},
		},
		{
			name: "second attempt",
			existingPlan: `# Test Plan
- [ ] Task 12: Implement Feature

## Description
Task description here.

### Execution History

#### Attempt 1 (2025-11-15 10:00:00)
Agent: python-pro
Verdict: RED

Agent Output:
Error: undefined variable

QC Feedback:
Missing import statement`,
			attempt: &ExecutionAttempt{
				AttemptNumber: 2,
				Agent:         "python-pro",
				Verdict:       "GREEN",
				AgentOutput:   "Successfully implemented",
				QCFeedback:    "Looks good",
				Timestamp:     time.Date(2025, 11, 15, 11, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"#### Attempt 1",
				"#### Attempt 2",
				"Verdict: GREEN",
				"Successfully implemented",
				"Looks good",
				"2025-11-15 11:00:00",
			},
		},
		{
			name: "yellow verdict",
			existingPlan: `# Test Plan
- [ ] Task 5: Database Setup`,
			attempt: &ExecutionAttempt{
				AttemptNumber: 1,
				Agent:         "database-admin",
				Verdict:       "YELLOW",
				AgentOutput:   "Partial success",
				QCFeedback:    "Review migration scripts",
				Timestamp:     time.Date(2025, 11, 15, 12, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"### Execution History",
				"Verdict: YELLOW",
				"Partial success",
				"Review migration scripts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			planPath := filepath.Join(tmpDir, "plan.md")
			writeFile(t, planPath, tt.existingPlan)

			err := UpdateTaskFeedback(planPath, "12", tt.attempt)
			if tt.name == "yellow verdict" {
				err = UpdateTaskFeedback(planPath, "5", tt.attempt)
			}
			if err != nil {
				t.Fatalf("UpdateTaskFeedback failed: %v", err)
			}

			content := readFile(t, planPath)
			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("expected content to contain %q, got:\n%s", want, content)
				}
			}
		})
	}
}

func TestUpdateMarkdownFeedbackPreservesStructure(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")

	existingPlan := `# Test Plan

- [x] Task 11: Previous Task (status: completed)
- [ ] Task 12: Current Task (status: pending)

## Description
This is the task description.

## Implementation
Implementation details.

## Verification
Test steps.`

	writeFile(t, planPath, existingPlan)

	attempt := &ExecutionAttempt{
		AttemptNumber: 1,
		Agent:         "golang-pro",
		Verdict:       "RED",
		AgentOutput:   "Compilation error",
		QCFeedback:    "Fix syntax",
		Timestamp:     time.Now(),
	}

	err := UpdateTaskFeedback(planPath, "12", attempt)
	if err != nil {
		t.Fatalf("UpdateTaskFeedback failed: %v", err)
	}

	content := readFile(t, planPath)

	// Verify original sections preserved
	if !strings.Contains(content, "## Description") {
		t.Error("expected Description section to be preserved")
	}
	if !strings.Contains(content, "## Implementation") {
		t.Error("expected Implementation section to be preserved")
	}
	if !strings.Contains(content, "## Verification") {
		t.Error("expected Verification section to be preserved")
	}

	// Verify history added
	if !strings.Contains(content, "### Execution History") {
		t.Error("expected Execution History section to be added")
	}
}

func TestUpdateTaskFeedbackUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.txt")
	writeFile(t, planPath, "invalid")

	attempt := &ExecutionAttempt{
		AttemptNumber: 1,
		Agent:         "test",
		Verdict:       "GREEN",
		Timestamp:     time.Now(),
	}

	err := UpdateTaskFeedback(planPath, "12", attempt)
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestUpdateYAMLFeedback(t *testing.T) {
	tests := []struct {
		name         string
		existingPlan string
		taskNumber   string
		attempt      *ExecutionAttempt
		wantContains []string
		wantErr      error
	}{
		{
			name: "first attempt creates history array",
			existingPlan: `plan:
  tasks:
    - task_number: 1
      name: "Test Task"
      status: "pending"`,
			taskNumber: "1",
			attempt: &ExecutionAttempt{
				AttemptNumber: 1,
				Agent:         "python-pro",
				Verdict:       "RED",
				AgentOutput:   "Error: undefined variable",
				QCFeedback:    "Missing import statement",
				Timestamp:     time.Date(2025, 11, 15, 10, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"execution_history:",
				"agent: python-pro",
				"verdict: RED",
				"agent_output:",
				"Error: undefined variable",
				"qc_feedback:",
				"Missing import statement",
				"timestamp: \"2025-11-15T10:00:00Z\"",
			},
		},
		{
			name: "second attempt appends to history",
			existingPlan: `plan:
  tasks:
    - task_number: 2
      name: "Test Task"
      status: "pending"
      execution_history:
        - attempt_number: 1
          agent: python-pro
          verdict: RED
          agent_output: "Error"
          qc_feedback: "Fix it"
          timestamp: "2025-11-15T10:00:00Z"`,
			taskNumber: "2",
			attempt: &ExecutionAttempt{
				AttemptNumber: 2,
				Agent:         "python-pro",
				Verdict:       "GREEN",
				AgentOutput:   "Success",
				QCFeedback:    "Looks good",
				Timestamp:     time.Date(2025, 11, 15, 11, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"attempt_number: 1",
				"verdict: GREEN",
				"agent_output:",
				"Success",
				"qc_feedback:",
				"Looks good",
				"timestamp: \"2025-11-15T11:00:00Z\"",
			},
		},
		{
			name: "task not found",
			existingPlan: `plan:
  tasks:
    - task_number: 5
      name: "Different Task"`,
			taskNumber: "999",
			attempt: &ExecutionAttempt{
				AttemptNumber: 1,
				Agent:         "test",
				Verdict:       "GREEN",
				Timestamp:     time.Now(),
			},
			wantErr: ErrTaskNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			planPath := filepath.Join(tmpDir, "plan.yaml")
			writeFile(t, planPath, tt.existingPlan)

			err := UpdateTaskFeedback(planPath, tt.taskNumber, tt.attempt)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("UpdateTaskFeedback failed: %v", err)
			}

			content := readFile(t, planPath)
			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("expected content to contain %q, got:\n%s", want, content)
				}
			}
		})
	}
}

func TestUpdateYAMLFeedbackPreservesStructure(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")

	existingPlan := `plan:
  metadata:
    feature_name: "Test Feature"
  tasks:
    - task_number: 1
      name: "Task One"
      status: "completed"
      files:
        - main.go
    - task_number: 2
      name: "Task Two"
      status: "pending"
      depends_on:
        - 1`

	writeFile(t, planPath, existingPlan)

	attempt := &ExecutionAttempt{
		AttemptNumber: 1,
		Agent:         "golang-pro",
		Verdict:       "RED",
		AgentOutput:   "Compilation error",
		QCFeedback:    "Fix syntax",
		Timestamp:     time.Date(2025, 11, 15, 12, 0, 0, 0, time.UTC),
	}

	err := UpdateTaskFeedback(planPath, "2", attempt)
	if err != nil {
		t.Fatalf("UpdateTaskFeedback failed: %v", err)
	}

	content := readFile(t, planPath)

	// Verify structure preserved
	if !strings.Contains(content, "metadata:") {
		t.Error("expected metadata section to be preserved")
	}
	if !strings.Contains(content, "feature_name: \"Test Feature\"") {
		t.Error("expected feature_name to be preserved")
	}
	if !strings.Contains(content, "task_number: 1") {
		t.Error("expected task 1 to be preserved")
	}
	if !strings.Contains(content, "depends_on:") {
		t.Error("expected depends_on to be preserved")
	}

	// Verify history added to task 2
	if !strings.Contains(content, "execution_history:") {
		t.Error("expected execution_history to be added")
	}
	if !strings.Contains(content, "verdict: RED") {
		t.Error("expected attempt to be recorded")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func TestConcurrentMarkdownFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	writeFile(t, planPath, `# Plan
- [ ] Task 12: Implement Feature (status: pending)

## Description
Task description.`)

	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		attemptNum := i
		go func() {
			defer wg.Done()
			attempt := &ExecutionAttempt{
				AttemptNumber: attemptNum,
				Agent:         "python-pro",
				Verdict:       "RED",
				AgentOutput:   fmt.Sprintf("Output %d", attemptNum),
				QCFeedback:    fmt.Sprintf("Feedback %d", attemptNum),
				Timestamp:     time.Now(),
			}
			if err := UpdateTaskFeedback(planPath, "12", attempt); err != nil {
				t.Errorf("UpdateTaskFeedback failed: %v", err)
			}
		}()
	}

	wg.Wait()
	content := readFile(t, planPath)

	// Verify all attempts recorded
	for i := 1; i <= 5; i++ {
		if !strings.Contains(content, fmt.Sprintf("Attempt %d", i)) {
			t.Errorf("expected Attempt %d to be recorded", i)
		}
		if !strings.Contains(content, fmt.Sprintf("Output %d", i)) {
			t.Errorf("expected Output %d to be recorded", i)
		}
	}
}

func TestConcurrentYAMLFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	writeFile(t, planPath, `plan:
  tasks:
    - task_number: 12
      name: "Test Task"
      status: "pending"`)

	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		attemptNum := i
		go func() {
			defer wg.Done()
			attempt := &ExecutionAttempt{
				AttemptNumber: attemptNum,
				Agent:         "golang-pro",
				Verdict:       "GREEN",
				AgentOutput:   fmt.Sprintf("Output %d", attemptNum),
				QCFeedback:    fmt.Sprintf("Feedback %d", attemptNum),
				Timestamp:     time.Date(2025, 11, 15, 10, attemptNum, 0, 0, time.UTC),
			}
			if err := UpdateTaskFeedback(planPath, "12", attempt); err != nil {
				t.Errorf("UpdateTaskFeedback failed: %v", err)
			}
		}()
	}

	wg.Wait()
	content := readFile(t, planPath)

	// Verify all attempts recorded
	for i := 1; i <= 5; i++ {
		if !strings.Contains(content, fmt.Sprintf("attempt_number: \"%d\"", i)) {
			t.Errorf("expected attempt %d to be recorded", i)
		}
		if !strings.Contains(content, fmt.Sprintf("Output %d", i)) {
			t.Errorf("expected Output %d to be recorded", i)
		}
	}
}

func TestMarkdownFeedbackMultipleAttempts(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	writeFile(t, planPath, `# Plan
- [ ] Task 5: Feature`)

	attempts := []struct {
		num     int
		verdict string
		output  string
	}{
		{1, "RED", "Error 1"},
		{2, "RED", "Error 2"},
		{3, "GREEN", "Success"},
	}

	for _, att := range attempts {
		attempt := &ExecutionAttempt{
			AttemptNumber: att.num,
			Agent:         "test-agent",
			Verdict:       att.verdict,
			AgentOutput:   att.output,
			QCFeedback:    "Feedback",
			Timestamp:     time.Now(),
		}
		if err := UpdateTaskFeedback(planPath, "5", attempt); err != nil {
			t.Fatalf("UpdateTaskFeedback failed: %v", err)
		}
	}

	content := readFile(t, planPath)
	for _, att := range attempts {
		if !strings.Contains(content, fmt.Sprintf("Attempt %d", att.num)) {
			t.Errorf("expected Attempt %d", att.num)
		}
		if !strings.Contains(content, att.output) {
			t.Errorf("expected %s", att.output)
		}
	}
}

func TestYAMLFeedbackMultipleAttempts(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	writeFile(t, planPath, `plan:
  tasks:
    - task_number: 3
      name: "Test"
      status: "pending"`)

	attempts := []struct {
		num     int
		verdict string
		output  string
	}{
		{1, "RED", "Fail 1"},
		{2, "YELLOW", "Warn"},
		{3, "GREEN", "Pass"},
	}

	for _, att := range attempts {
		attempt := &ExecutionAttempt{
			AttemptNumber: att.num,
			Agent:         "test-agent",
			Verdict:       att.verdict,
			AgentOutput:   att.output,
			QCFeedback:    "Feedback",
			Timestamp:     time.Now(),
		}
		if err := UpdateTaskFeedback(planPath, "3", attempt); err != nil {
			t.Fatalf("UpdateTaskFeedback failed: %v", err)
		}
	}

	content := readFile(t, planPath)
	for _, att := range attempts {
		if !strings.Contains(content, fmt.Sprintf("attempt_number: \"%d\"", att.num)) {
			t.Errorf("expected attempt %d", att.num)
		}
		if !strings.Contains(content, att.output) {
			t.Errorf("expected %s", att.output)
		}
		if !strings.Contains(content, fmt.Sprintf("verdict: %s", att.verdict)) {
			t.Errorf("expected verdict %s", att.verdict)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(data)
}
