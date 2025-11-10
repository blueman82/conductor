package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/parser"
)

func TestParseAndValidateSimpleMarkdown(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.md")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}

	if plan.Tasks[0].Number != "1" || plan.Tasks[0].Name != "First Task" {
		t.Errorf("First task not parsed correctly")
	}

	if plan.Tasks[1].DependsOn[0] != "1" {
		t.Error("Task 2 dependency incorrect")
	}
}

func TestParseAndValidateSimpleYAML(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.yaml")

	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse YAML plan: %v", err)
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}
}

func TestCalculateWavesSimplePlan(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.md")
	plan, _ := parser.ParseFile(planPath)

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	if len(waves) != 2 {
		t.Errorf("Expected 2 waves, got %d", len(waves))
	}
}

func TestCalculateWavesMultiWave(t *testing.T) {
	planPath := filepath.Join("fixtures", "multi-wave.md")
	plan, _ := parser.ParseFile(planPath)

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	if len(waves) != 4 {
		t.Errorf("Expected 4 waves, got %d", len(waves))
	}
}

func TestCyclicDependencyDetection(t *testing.T) {
	planPath := filepath.Join("fixtures", "cyclic.md")
	plan, _ := parser.ParseFile(planPath)

	_, err := executor.CalculateWaves(plan.Tasks)
	if err == nil {
		t.Error("Expected error for cyclic dependencies")
	}
}

func TestInvalidDependencies(t *testing.T) {
	planPath := filepath.Join("fixtures", "invalid.md")
	plan, _ := parser.ParseFile(planPath)

	_, err := executor.CalculateWaves(plan.Tasks)
	if err == nil {
		t.Error("Expected error for invalid dependency")
	}
}

func TestDryRunValidation(t *testing.T) {
	planPath := filepath.Join("fixtures", "simple-plan.md")
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	waves, err := executor.CalculateWaves(plan.Tasks)
	if err != nil {
		t.Fatalf("Failed to calculate waves: %v", err)
	}

	if len(waves) == 0 {
		t.Error("Expected waves")
	}
}

func TestTimeoutContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected timeout")
	case <-ctx.Done():
		// OK
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-ctx.Done():
		// OK
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestMultipleFormatParsing(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		taskCount int
	}{
		{"Markdown", filepath.Join("fixtures", "simple-plan.md"), 2},
		{"YAML", filepath.Join("fixtures", "simple-plan.yaml"), 2},
		{"MultiWave", filepath.Join("fixtures", "multi-wave.md"), 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := parser.ParseFile(tt.fixture)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			if len(plan.Tasks) != tt.taskCount {
				t.Errorf("Expected %d tasks, got %d", tt.taskCount, len(plan.Tasks))
			}
		})
	}
}

func TestWaveExecutionOrder(t *testing.T) {
	planPath := filepath.Join("fixtures", "multi-wave.md")
	plan, _ := parser.ParseFile(planPath)
	waves, _ := executor.CalculateWaves(plan.Tasks)

	expectedWaveStructure := []int{1, 2, 1, 1}
	for i, wave := range waves {
		if i >= len(expectedWaveStructure) {
			break
		}
		if len(wave.TaskNumbers) != expectedWaveStructure[i] {
			t.Errorf("Wave %d: expected %d tasks, got %d", i+1, expectedWaveStructure[i], len(wave.TaskNumbers))
		}
	}
}

func TestFileExists(t *testing.T) {
	fixtures := []string{
		"fixtures/simple-plan.md",
		"fixtures/simple-plan.yaml",
		"fixtures/multi-wave.md",
		"fixtures/cyclic.md",
		"fixtures/invalid.md",
	}

	for _, fixture := range fixtures {
		if _, err := os.Stat(fixture); os.IsNotExist(err) {
			t.Errorf("Fixture file missing: %s", fixture)
		}
	}
}
