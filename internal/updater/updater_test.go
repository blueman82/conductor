package updater

import (
	"errors"
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
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
