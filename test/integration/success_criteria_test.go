package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

func TestQualityController_Review_UsesPlanSuccessCriteria(t *testing.T) {
	plan := loadPlanFromPath(t, filepath.Join("fixtures", "e2e-criteria-plan.yaml"))
	if len(plan.Tasks) == 0 {
		t.Fatal("plan should contain tasks")
	}

	tests := []struct {
		name     string
		response string
		wantFlag string
	}{
		{
			name:     "criteria all pass returns GREEN",
			response: `{"verdict":"GREEN","feedback":"All good","criteria_results":[{"index":0,"passed":true,"evidence":"tests"},{"index":1,"passed":true,"evidence":"race free"}]}`,
			wantFlag: "GREEN",
		},
		{
			name:     "criteria failure overrides verdict",
			response: `{"verdict":"GREEN","feedback":"Looks ok","criteria_results":[{"index":0,"passed":true},{"index":1,"passed":false,"fail_reason":"race detected"}]}`,
			wantFlag: "RED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := &fakeInvoker{outputs: []string{tt.response}}
			qc := executor.NewQualityController(inv)
			qc.AgentConfig.Mode = ""

			result, err := qc.Review(context.Background(), plan.Tasks[0], "agent output")
			if err != nil {
				t.Fatalf("Review() error = %v", err)
			}

			if result.Flag != tt.wantFlag {
				t.Fatalf("Review() flag = %s, want %s", result.Flag, tt.wantFlag)
			}
		})
	}
}

type fakeInvoker struct {
	outputs []string
	idx     int
}

func (f *fakeInvoker) Invoke(ctx context.Context, task models.Task) (*agent.InvocationResult, error) {
	if f.idx >= len(f.outputs) {
		return &agent.InvocationResult{Output: `{"verdict":"GREEN","feedback":"default"}`}, nil
	}
	output := f.outputs[f.idx]
	f.idx++
	return &agent.InvocationResult{
		Output:   output,
		ExitCode: 0,
	}, nil
}
