package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
	"github.com/harrison/conductor/internal/learning"
	"github.com/harrison/conductor/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLearningPipeline_SingleTask tests full pipeline execution with learning store
func TestLearningPipeline_SingleTask(t *testing.T) {
	// Setup: Create plan, learning store
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	planFile := filepath.Join(t.TempDir(), "plan.md")
	plan := &models.Plan{
		FilePath: planFile,
		Name:     "Test Plan",
		Tasks: []models.Task{
			{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
				Agent:  "backend-developer",
			},
		},
		Waves: []models.Wave{
			{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
		},
	}

	// Create mock invoker that simulates successful execution
	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"Task completed successfully"}`,
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	})

	// Create executor with learning store
	updater := &recordingUpdater{}
	taskExec, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: planFile,
		QualityControl: models.QualityControlConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)

	waveExec := NewWaveExecutor(taskExec, nil)
	orch := NewOrchestrator(waveExec, nil)

	// Execute
	result, err := orch.ExecutePlan(context.Background(), plan)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify execution completed successfully
	assert.Equal(t, 1, result.Completed)
	assert.Equal(t, 0, result.Failed)

	// Verify learning store can record execution
	exec := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   "1",
		TaskName:     "Test Task",
		Agent:        "backend-developer",
		Prompt:       "Do something",
		Success:      true,
		Output:       "Task completed successfully",
		DurationSecs: 0,
	}
	err = store.RecordExecution(exec)
	require.NoError(t, err)

	// Verify execution was recorded
	history, err := store.GetExecutionHistory("1")
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "Test Task", history[0].TaskName)
	assert.Equal(t, "backend-developer", history[0].Agent)
	assert.True(t, history[0].Success)
}

// TestLearningPipeline_FailureAdaptation tests agent switching after repeated failures
func TestLearningPipeline_FailureAdaptation(t *testing.T) {
	// Setup: Create plan, learning store
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	planFile := "plan.md"
	taskNumber := "1"

	// Simulate 2 failures with backend-developer
	for i := 1; i <= 2; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    i,
			TaskNumber:   taskNumber,
			TaskName:     "Problematic Task",
			Agent:        "backend-developer",
			Prompt:       "Fix the bug",
			Success:      false,
			Output:       "compilation_error: undefined variable",
			ErrorMessage: "Task failed",
			DurationSecs: 30,
		}
		err = store.RecordExecution(exec)
		require.NoError(t, err)
	}

	// Record successful executions with general-purpose agent to establish pattern
	for i := 1; i <= 6; i++ {
		exec := &learning.TaskExecution{
			PlanFile:     "other-plan.md",
			RunNumber:    i,
			TaskNumber:   fmt.Sprintf("task-%d", i),
			TaskName:     fmt.Sprintf("Other Task %d", i),
			Agent:        "general-purpose",
			Prompt:       "Do something else",
			Success:      true,
			Output:       "Completed",
			DurationSecs: 10,
		}
		err = store.RecordExecution(exec)
		require.NoError(t, err)
	}

	// Analyze failures
	analysis, err := store.AnalyzeFailures(context.Background(), planFile, taskNumber)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Verify failure analysis
	assert.Equal(t, 2, analysis.TotalAttempts)
	assert.Equal(t, 2, analysis.FailedAttempts)
	assert.Contains(t, analysis.TriedAgents, "backend-developer")
	assert.True(t, analysis.ShouldTryDifferentAgent)
	assert.NotEmpty(t, analysis.SuggestedAgent)
	assert.NotEqual(t, "backend-developer", analysis.SuggestedAgent)

	// Verify common patterns detected
	assert.Contains(t, analysis.CommonPatterns, "compilation_error")
}

// TestLearningPipeline_PatternDetection tests failure pattern extraction
func TestLearningPipeline_PatternDetection(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	planFile := "plan.md"
	taskNumber := "1"

	// Test cases with different failure patterns
	testCases := []struct {
		name            string
		output          string
		expectedPattern string
	}{
		{
			name:            "compilation error",
			output:          "Error: compilation_error on line 42",
			expectedPattern: "compilation_error",
		},
		{
			name:            "test failure",
			output:          "Test suite failed: test_failure in TestFoo",
			expectedPattern: "test_failure",
		},
		{
			name:            "missing dependency",
			output:          "Cannot import module: dependency_missing",
			expectedPattern: "dependency_missing",
		},
		{
			name:            "timeout",
			output:          "Operation timeout after 30 seconds",
			expectedPattern: "timeout",
		},
		{
			name:            "syntax error",
			output:          "SyntaxError: syntax_error at line 10",
			expectedPattern: "syntax_error",
		},
		{
			name:            "type error",
			output:          "TypeError: type_error - expected string, got int",
			expectedPattern: "type_error",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exec := &learning.TaskExecution{
				PlanFile:     planFile,
				RunNumber:    i + 1,
				TaskNumber:   taskNumber,
				TaskName:     "Pattern Test Task",
				Agent:        "test-agent",
				Prompt:       "Do something",
				Success:      false,
				Output:       tc.output,
				ErrorMessage: "Task failed",
				DurationSecs: 10,
			}
			err = store.RecordExecution(exec)
			require.NoError(t, err)
		})
	}

	// Analyze failures to extract patterns
	analysis, err := store.AnalyzeFailures(context.Background(), planFile, taskNumber)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Verify patterns were detected
	assert.Equal(t, 6, analysis.TotalAttempts)
	assert.Equal(t, 6, analysis.FailedAttempts)
	assert.True(t, len(analysis.CommonPatterns) >= 6)

	// Verify all expected patterns are present
	for _, tc := range testCases {
		assert.Contains(t, analysis.CommonPatterns, tc.expectedPattern,
			"Expected pattern %s to be detected", tc.expectedPattern)
	}
}

// TestLearningPipeline_RunNumberTracking tests run number increment across executions
func TestLearningPipeline_RunNumberTracking(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	planFile := "plan.md"

	// Test initial run count (should be 0)
	runCount := store.GetRunCount(planFile)
	assert.Equal(t, 0, runCount, "Initial run count should be 0")

	// Record executions for runs 1-5
	for runNum := 1; runNum <= 5; runNum++ {
		for taskNum := 1; taskNum <= 3; taskNum++ {
			exec := &learning.TaskExecution{
				PlanFile:     planFile,
				RunNumber:    runNum,
				TaskNumber:   fmt.Sprintf("%d", taskNum),
				TaskName:     fmt.Sprintf("Task %d", taskNum),
				Agent:        "test-agent",
				Prompt:       "Do something",
				Success:      true,
				Output:       "Completed",
				DurationSecs: 5,
			}
			err = store.RecordExecution(exec)
			require.NoError(t, err)
		}

		// Verify run count increments correctly
		currentRunCount := store.GetRunCount(planFile)
		assert.Equal(t, runNum, currentRunCount,
			"Run count should be %d after run %d", runNum, runNum)
	}

	// Verify final run count
	finalRunCount := store.GetRunCount(planFile)
	assert.Equal(t, 5, finalRunCount)

	// Test different plan file has independent run count
	otherPlanFile := "other-plan.md"
	otherRunCount := store.GetRunCount(otherPlanFile)
	assert.Equal(t, 0, otherRunCount, "Different plan file should have independent run count")
}

// TestLearningPipeline_MultipleTasksWithLearning tests learning across multiple task executions
func TestLearningPipeline_MultipleTasksWithLearning(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	planFile := filepath.Join(t.TempDir(), "plan.md")
	plan := &models.Plan{
		FilePath: planFile,
		Name:     "Multi-Task Plan",
		Tasks: []models.Task{
			{
				Number: "1",
				Name:   "Task 1",
				Prompt: "First task",
				Agent:  "backend-developer",
			},
			{
				Number:    "2",
				Name:      "Task 2",
				Prompt:    "Second task",
				Agent:     "frontend-developer",
				DependsOn: []string{"1"},
			},
			{
				Number:    "3",
				Name:      "Task 3",
				Prompt:    "Third task",
				Agent:     "fullstack-developer",
				DependsOn: []string{"2"},
			},
		},
		Waves: []models.Wave{
			{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
			{
				Name:           "Wave 2",
				TaskNumbers:    []string{"2"},
				MaxConcurrency: 1,
			},
			{
				Name:           "Wave 3",
				TaskNumbers:    []string{"3"},
				MaxConcurrency: 1,
			},
		},
	}

	// Create mock invoker for all tasks
	invoker := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"content":"Task 1 completed"}`,
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		},
		&agent.InvocationResult{
			Output:   `{"content":"Task 2 completed"}`,
			ExitCode: 0,
			Duration: 150 * time.Millisecond,
		},
		&agent.InvocationResult{
			Output:   `{"content":"Task 3 completed"}`,
			ExitCode: 0,
			Duration: 200 * time.Millisecond,
		},
	)

	updater := &recordingUpdater{}
	taskExec, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: planFile,
		QualityControl: models.QualityControlConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)

	waveExec := NewWaveExecutor(taskExec, nil)
	orch := NewOrchestrator(waveExec, nil)

	// Execute plan
	result, err := orch.ExecutePlan(context.Background(), plan)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify all tasks completed
	assert.Equal(t, 3, result.Completed)
	assert.Equal(t, 0, result.Failed)

	// Record executions in learning store
	runNumber := 1
	for _, task := range plan.Tasks {
		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    runNumber,
			TaskNumber:   task.Number,
			TaskName:     task.Name,
			Agent:        task.Agent,
			Prompt:       task.Prompt,
			Success:      true,
			Output:       fmt.Sprintf("%s completed", task.Name),
			DurationSecs: 100,
		}
		err = store.RecordExecution(exec)
		require.NoError(t, err)
	}

	// Verify all executions recorded
	for _, task := range plan.Tasks {
		history, err := store.GetExecutionHistory(task.Number)
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.Equal(t, task.Name, history[0].TaskName)
		assert.Equal(t, task.Agent, history[0].Agent)
		assert.True(t, history[0].Success)
	}

	// Verify run count
	runCount := store.GetRunCount(planFile)
	assert.Equal(t, 1, runCount)
}

// TestLearningPipeline_StoreUnavailable tests graceful degradation when store is unavailable
func TestLearningPipeline_StoreUnavailable(t *testing.T) {
	// This test verifies that execution can proceed even if learning store is not available
	planFile := filepath.Join(t.TempDir(), "plan.md")
	plan := &models.Plan{
		FilePath: planFile,
		Name:     "Test Plan",
		Tasks: []models.Task{
			{
				Number: "1",
				Name:   "Test Task",
				Prompt: "Do something",
				Agent:  "backend-developer",
			},
		},
		Waves: []models.Wave{
			{
				Name:           "Wave 1",
				TaskNumbers:    []string{"1"},
				MaxConcurrency: 1,
			},
		},
	}

	invoker := newStubInvoker(&agent.InvocationResult{
		Output:   `{"content":"Task completed"}`,
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	})

	updater := &recordingUpdater{}
	taskExec, err := NewTaskExecutor(invoker, nil, updater, TaskExecutorConfig{
		PlanPath: planFile,
		QualityControl: models.QualityControlConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)

	waveExec := NewWaveExecutor(taskExec, nil)
	orch := NewOrchestrator(waveExec, nil)

	// Execute without learning store (graceful degradation)
	result, err := orch.ExecutePlan(context.Background(), plan)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify execution succeeded despite no learning store
	assert.Equal(t, 1, result.Completed)
	assert.Equal(t, 0, result.Failed)
}

// TestLearningPipeline_FirstRunVsSubsequentRuns tests run number behavior across multiple runs
func TestLearningPipeline_FirstRunVsSubsequentRuns(t *testing.T) {
	store, err := learning.NewStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	planFile := "plan.md"
	taskNumber := "1"

	// First run: no history
	runCount := store.GetRunCount(planFile)
	assert.Equal(t, 0, runCount, "First run should have count 0")

	// Execute first run
	exec1 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   taskNumber,
		TaskName:     "Test Task",
		Agent:        "backend-developer",
		Prompt:       "Do something",
		Success:      true,
		Output:       "First run completed",
		DurationSecs: 50,
	}
	err = store.RecordExecution(exec1)
	require.NoError(t, err)

	// Verify first run recorded
	runCount = store.GetRunCount(planFile)
	assert.Equal(t, 1, runCount, "After first run, count should be 1")

	history, err := store.GetExecutionHistory(taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, 1, history[0].RunNumber)

	// Subsequent run: has history
	exec2 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    2,
		TaskNumber:   taskNumber,
		TaskName:     "Test Task",
		Agent:        "backend-developer",
		Prompt:       "Do something",
		Success:      true,
		Output:       "Second run completed",
		DurationSecs: 45,
	}
	err = store.RecordExecution(exec2)
	require.NoError(t, err)

	// Verify subsequent run recorded
	runCount = store.GetRunCount(planFile)
	assert.Equal(t, 2, runCount, "After second run, count should be 2")

	history, err = store.GetExecutionHistory(taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 2)

	// History should be ordered by most recent first
	assert.Equal(t, 2, history[0].RunNumber)
	assert.Equal(t, 1, history[1].RunNumber)
}
