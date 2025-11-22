package integration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

func TestIntegrationTask_BuildsPromptWithDependencyContext(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "integration-plan.yaml"))

	if len(plan.Tasks) < 2 {
		t.Fatal("integration-plan.yaml should have at least 2 tasks")
	}

	integrationTask := plan.Tasks[1]
	if len(integrationTask.DependsOn) == 0 {
		t.Fatal("Task 2 should have dependencies")
	}

	inv := &fakeInvokerWithCapture{}
	te, err := executor.NewTaskExecutor(inv, nil, nil, executor.TaskExecutorConfig{})
	if err != nil {
		t.Fatalf("NewTaskExecutor() error = %v", err)
	}
	te.Plan = plan

	_, _ = te.Execute(context.Background(), integrationTask)

	if len(inv.capturedPrompts) == 0 {
		t.Fatal("no prompts captured")
	}

	prompt := inv.capturedPrompts[0]
	if !strings.Contains(prompt, "# INTEGRATION TASK CONTEXT") {
		t.Error("prompt missing INTEGRATION TASK CONTEXT header")
	}
	if !strings.Contains(prompt, "Before implementing, you MUST read these dependency files") {
		t.Error("prompt missing file read instruction")
	}
	if !strings.Contains(prompt, "## Dependency:") {
		t.Error("prompt missing dependency section")
	}
}

func TestIntegrationTask_QCValidatesIntegrationCriteria(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "integration-plan.yaml"))

	integrationTask := plan.Tasks[1]
	inv := &fakeInvoker{outputs: []string{
		`{"status":"success","output":"implemented","files_modified":["internal/service.go"]}`,
		`{"verdict":"GREEN","feedback":"Integration correct","criteria_results":[{"index":0,"passed":true}]}`,
	}}

	te, err := executor.NewTaskExecutor(inv, nil, nil, executor.TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{
			Enabled:  true,
			RetryOnRed: 2,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor() error = %v", err)
	}
	te.Plan = plan

	result, err := te.Execute(context.Background(), integrationTask)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != models.StatusGreen {
		t.Errorf("status = %s, want GREEN", result.Status)
	}
}

func TestIntegrationTask_FailsWhenDependencyFilesNotModified(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "integration-plan.yaml"))

	integrationTask := plan.Tasks[1]
	inv := &fakeInvoker{outputs: []string{
		`{"status":"success","output":"implemented","files_modified":["wrong/path.go"]}`,
		`{"verdict":"RED","feedback":"Missing expected file paths","criteria_results":[{"index":0,"passed":false}]}`,
	}}

	te, err := executor.NewTaskExecutor(inv, nil, nil, executor.TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{
			Enabled:  true,
			RetryOnRed: 0,
		},
	})
	if err != nil {
		t.Fatalf("NewTaskExecutor() error = %v", err)
	}
	te.Plan = plan

	result, _ := te.Execute(context.Background(), integrationTask)
	if result.Status != models.StatusRed {
		t.Errorf("status = %s, want RED", result.Status)
	}
}

type fakeInvokerWithCapture struct {
	capturedPrompts []string
}

func (f *fakeInvokerWithCapture) Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
	f.capturedPrompts = append(f.capturedPrompts, task.Prompt)
	return &agent.InvocationResult{
		Output:   `{"status":"success","output":"default"}`,
		ExitCode: 0,
	}, nil
}
