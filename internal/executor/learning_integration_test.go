package executor

import (
	"context"
	"fmt"
	"os"
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
	err = store.RecordExecution(context.Background(), exec)
	require.NoError(t, err)

	// Verify execution was recorded
	history, err := store.GetExecutionHistory(context.Background(), planFile, "1")
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
		err = store.RecordExecution(context.Background(), exec)
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
		err = store.RecordExecution(context.Background(), exec)
		require.NoError(t, err)
	}

	// Analyze failures
	analysis, err := store.AnalyzeFailures(context.Background(), planFile, taskNumber, 2)
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
			err = store.RecordExecution(context.Background(), exec)
			require.NoError(t, err)
		})
	}

	// Analyze failures to extract patterns
	analysis, err := store.AnalyzeFailures(context.Background(), planFile, taskNumber, 2)
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
	runCount, _ := store.GetRunCount(context.Background(), planFile)
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
			err = store.RecordExecution(context.Background(), exec)
			require.NoError(t, err)
		}

		// Verify run count increments correctly
		currentRunCount, _ := store.GetRunCount(context.Background(), planFile)
		assert.Equal(t, runNum, currentRunCount,
			"Run count should be %d after run %d", runNum, runNum)
	}

	// Verify final run count
	finalRunCount, _ := store.GetRunCount(context.Background(), planFile)
	assert.Equal(t, 5, finalRunCount)

	// Test different plan file has independent run count
	otherPlanFile := "other-plan.md"
	otherRunCount, _ := store.GetRunCount(context.Background(), otherPlanFile)
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
		err = store.RecordExecution(context.Background(), exec)
		require.NoError(t, err)
	}

	// Verify all executions recorded
	for _, task := range plan.Tasks {
		history, err := store.GetExecutionHistory(context.Background(), planFile, task.Number)
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.Equal(t, task.Name, history[0].TaskName)
		assert.Equal(t, task.Agent, history[0].Agent)
		assert.True(t, history[0].Success)
	}

	// Verify run count
	runCount, _ := store.GetRunCount(context.Background(), planFile)
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
	runCount, _ := store.GetRunCount(context.Background(), planFile)
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
	err = store.RecordExecution(context.Background(), exec1)
	require.NoError(t, err)

	// Verify first run recorded
	runCount, _ = store.GetRunCount(context.Background(), planFile)
	assert.Equal(t, 1, runCount, "After first run, count should be 1")

	history, err := store.GetExecutionHistory(context.Background(), "plan.md", taskNumber)
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
	err = store.RecordExecution(context.Background(), exec2)
	require.NoError(t, err)

	// Verify subsequent run recorded
	runCount, _ = store.GetRunCount(context.Background(), planFile)
	assert.Equal(t, 2, runCount, "After second run, count should be 2")

	history, err = store.GetExecutionHistory(context.Background(), "plan.md", taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 2)

	// History should be ordered by most recent first
	assert.Equal(t, 2, history[0].RunNumber)
	assert.Equal(t, 1, history[1].RunNumber)
}

// TestCrossRun_LearningPersists verifies that learning data persists between conductor runs
func TestCrossRun_LearningPersists(t *testing.T) {
	// Create persistent temp database (not :memory:)
	dbPath := filepath.Join(t.TempDir(), "cross_run_test.db")

	// Run 1: Execute a task and record outcome
	db1, err := learning.NewStore(dbPath)
	require.NoError(t, err)

	planFile := "plan.yaml"
	taskNumber := "task-1"

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
	err = db1.RecordExecution(context.Background(), exec1)
	require.NoError(t, err)

	// Close first database connection
	db1.Close()

	// Run 2: New orchestrator instance, same database file
	db2, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer db2.Close()

	// Verify data from Run 1 is available
	history, err := db2.GetExecutionHistory(context.Background(), "plan.yaml", taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 1, "Should find execution from Run 1")

	assert.Equal(t, 1, history[0].RunNumber)
	assert.Equal(t, "backend-developer", history[0].Agent)
	assert.True(t, history[0].Success)

	// Verify run count persisted
	runCount, _ := db2.GetRunCount(context.Background(), planFile)
	assert.Equal(t, 1, runCount, "Run count should persist across runs")

	// Record second run
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
	err = db2.RecordExecution(context.Background(), exec2)
	require.NoError(t, err)

	// Verify both runs are in history
	history, err = db2.GetExecutionHistory(context.Background(), "plan.yaml", taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 2, "Should find executions from both runs")
}

// TestCrossRun_AgentAdaptation verifies that the system adapts agent selection based on historical data
func TestCrossRun_AgentAdaptation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "adaptation_test.db")

	planFile := "plan.yaml"
	taskNumber := "task-1"

	// Run 1: Record failures with agent-a
	db1, err := learning.NewStore(dbPath)
	require.NoError(t, err)

	// Attempt 1: Fail with agent-a
	exec1 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   taskNumber,
		TaskName:     "Problematic Task",
		Agent:        "agent-a",
		Prompt:       "Fix the bug",
		Success:      false,
		Output:       "compilation_error: undefined variable",
		ErrorMessage: "Task failed",
		DurationSecs: 30,
	}
	err = db1.RecordExecution(context.Background(), exec1)
	require.NoError(t, err)

	// Attempt 2: Fail again with agent-a
	exec2 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    2,
		TaskNumber:   taskNumber,
		TaskName:     "Problematic Task",
		Agent:        "agent-a",
		Prompt:       "Fix the bug",
		Success:      false,
		Output:       "compilation_error: type mismatch",
		ErrorMessage: "Task failed again",
		DurationSecs: 35,
	}
	err = db1.RecordExecution(context.Background(), exec2)
	require.NoError(t, err)

	db1.Close()

	// Run 2: New orchestrator instance, should adapt based on failures
	db2, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer db2.Close()

	// Analyze failures
	analysis, err := db2.AnalyzeFailures(context.Background(), planFile, taskNumber, 2)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Verify agent-a has poor performance
	assert.Equal(t, 2, analysis.TotalAttempts)
	assert.Equal(t, 2, analysis.FailedAttempts)
	assert.Contains(t, analysis.TriedAgents, "agent-a")
	assert.True(t, analysis.ShouldTryDifferentAgent)

	// Suggested agent should NOT be agent-a
	assert.NotEmpty(t, analysis.SuggestedAgent)
	assert.NotEqual(t, "agent-a", analysis.SuggestedAgent,
		"System should suggest different agent after multiple failures")

	// Verify failure patterns detected
	assert.Contains(t, analysis.CommonPatterns, "compilation_error")

	// Now record success with different agent
	exec3 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    3,
		TaskNumber:   taskNumber,
		TaskName:     "Problematic Task",
		Agent:        "agent-b",
		Prompt:       "Fix the bug",
		Success:      true,
		Output:       "Task completed successfully",
		DurationSecs: 40,
	}
	err = db2.RecordExecution(context.Background(), exec3)
	require.NoError(t, err)

	// Verify new history shows agent-b success
	history, err := db2.GetExecutionHistory(context.Background(), "plan.yaml", taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 3)

	// Most recent should be successful with agent-b
	assert.True(t, history[0].Success)
	assert.Equal(t, "agent-b", history[0].Agent)
}

// TestCrossRun_RunNumberIncrement verifies that run numbers increment correctly per plan file
func TestCrossRun_RunNumberIncrement(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "run_number_test.db")

	// Run 1 for plan-a.yaml
	db1, err := learning.NewStore(dbPath)
	require.NoError(t, err)

	exec1 := &learning.TaskExecution{
		PlanFile:     "plan-a.yaml",
		RunNumber:    1,
		TaskNumber:   "task-1",
		TaskName:     "Task A1",
		Agent:        "agent-a",
		Prompt:       "Do A1",
		Success:      true,
		Output:       "A1 completed",
		DurationSecs: 10,
	}
	err = db1.RecordExecution(context.Background(), exec1)
	require.NoError(t, err)

	db1.Close()

	// Run 2 for plan-a.yaml (should be run number 2)
	db2, err := learning.NewStore(dbPath)
	require.NoError(t, err)

	exec2 := &learning.TaskExecution{
		PlanFile:     "plan-a.yaml",
		RunNumber:    2,
		TaskNumber:   "task-2",
		TaskName:     "Task A2",
		Agent:        "agent-b",
		Prompt:       "Do A2",
		Success:      true,
		Output:       "A2 completed",
		DurationSecs: 15,
	}
	err = db2.RecordExecution(context.Background(), exec2)
	require.NoError(t, err)

	db2.Close()

	// Run 1 for plan-b.yaml (different plan, run number 1)
	db3, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer db3.Close()

	exec3 := &learning.TaskExecution{
		PlanFile:     "plan-b.yaml",
		RunNumber:    1,
		TaskNumber:   "task-3",
		TaskName:     "Task B1",
		Agent:        "agent-c",
		Prompt:       "Do B1",
		Success:      true,
		Output:       "B1 completed",
		DurationSecs: 20,
	}
	err = db3.RecordExecution(context.Background(), exec3)
	require.NoError(t, err)

	// Verify run counts are correct per plan file
	runCountA, _ := db3.GetRunCount(context.Background(), "plan-a.yaml")
	assert.Equal(t, 2, runCountA, "plan-a.yaml should have 2 runs")

	runCountB, _ := db3.GetRunCount(context.Background(), "plan-b.yaml")
	assert.Equal(t, 1, runCountB, "plan-b.yaml should have 1 run")

	// Verify each plan's executions are separate
	historyA1, err := db3.GetExecutionHistory(context.Background(), "plan-a.yaml", "task-1")
	require.NoError(t, err)
	require.Len(t, historyA1, 1)
	assert.Equal(t, "plan-a.yaml", historyA1[0].PlanFile)

	historyB1, err := db3.GetExecutionHistory(context.Background(), "plan-b.yaml", "task-3")
	require.NoError(t, err)
	require.Len(t, historyB1, 1)
	assert.Equal(t, "plan-b.yaml", historyB1[0].PlanFile)
}

// TestCrossRun_DatabaseDeleted verifies graceful handling when database is deleted between runs
func TestCrossRun_DatabaseDeleted(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "deleted_test.db")

	// Run 1: Create database and record data
	db1, err := learning.NewStore(dbPath)
	require.NoError(t, err)

	exec1 := &learning.TaskExecution{
		PlanFile:     "plan.yaml",
		RunNumber:    1,
		TaskNumber:   "task-1",
		TaskName:     "Test Task",
		Agent:        "agent-a",
		Prompt:       "Do something",
		Success:      true,
		Output:       "Completed",
		DurationSecs: 10,
	}
	err = db1.RecordExecution(context.Background(), exec1)
	require.NoError(t, err)

	db1.Close()

	// Verify database exists
	_, err = os.Stat(dbPath)
	require.NoError(t, err, "Database file should exist")

	// Delete the database file
	err = os.Remove(dbPath)
	require.NoError(t, err)

	// Verify database is gone
	_, err = os.Stat(dbPath)
	require.True(t, os.IsNotExist(err), "Database file should be deleted")

	// Run 2: Should handle missing database gracefully by recreating it
	db2, err := learning.NewStore(dbPath)
	require.NoError(t, err, "Should recreate database gracefully")
	defer db2.Close()

	// Database should be recreated
	_, err = os.Stat(dbPath)
	require.NoError(t, err, "Database file should be recreated")

	// Stats should be empty (no historical data)
	history, err := db2.GetExecutionHistory(context.Background(), "plan.yaml", "task-1")
	require.NoError(t, err)
	assert.Empty(t, history, "Should have no history after database deletion")

	runCount, _ := db2.GetRunCount(context.Background(), "plan.yaml")
	assert.Equal(t, 0, runCount, "Run count should be 0 after database deletion")

	// Should be able to record new data
	exec2 := &learning.TaskExecution{
		PlanFile:     "plan.yaml",
		RunNumber:    1,
		TaskNumber:   "task-1",
		TaskName:     "Test Task",
		Agent:        "agent-a",
		Prompt:       "Do something",
		Success:      true,
		Output:       "Completed after recreation",
		DurationSecs: 12,
	}
	err = db2.RecordExecution(context.Background(), exec2)
	require.NoError(t, err, "Should be able to record in recreated database")

	// Verify new data was recorded
	history, err = db2.GetExecutionHistory(context.Background(), "plan.yaml", "task-1")
	require.NoError(t, err)
	require.Len(t, history, 1)
}

// TestCrossRun_SessionIsolation verifies that different sessions maintain proper data separation
func TestCrossRun_SessionIsolation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session_test.db")

	db, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer db.Close()

	planFile := "plan.yaml"
	taskNumber := "task-1"

	// Session 1: Run with agent-a (success)
	exec1 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    1,
		TaskNumber:   taskNumber,
		TaskName:     "Task 1",
		Agent:        "agent-a",
		Prompt:       "Do something",
		Success:      true,
		Output:       "Session 1 completed",
		DurationSecs: 20,
	}
	err = db.RecordExecution(context.Background(), exec1)
	require.NoError(t, err)

	// Session 2: Run with agent-b (failure)
	exec2 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    2,
		TaskNumber:   taskNumber,
		TaskName:     "Task 1",
		Agent:        "agent-b",
		Prompt:       "Do something",
		Success:      false,
		Output:       "Session 2 failed",
		ErrorMessage: "Error occurred",
		DurationSecs: 25,
	}
	err = db.RecordExecution(context.Background(), exec2)
	require.NoError(t, err)

	// Session 3: Run with agent-a again (success)
	exec3 := &learning.TaskExecution{
		PlanFile:     planFile,
		RunNumber:    3,
		TaskNumber:   taskNumber,
		TaskName:     "Task 1",
		Agent:        "agent-a",
		Prompt:       "Do something",
		Success:      true,
		Output:       "Session 3 completed",
		DurationSecs: 18,
	}
	err = db.RecordExecution(context.Background(), exec3)
	require.NoError(t, err)

	// Verify all sessions recorded
	history, err := db.GetExecutionHistory(context.Background(), "plan.yaml", taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 3, "Should have 3 session executions")

	// Verify cross-session learning
	runCount, _ := db.GetRunCount(context.Background(), planFile)
	assert.Equal(t, 3, runCount, "Should track all 3 runs")

	// Verify each execution maintained its data
	assert.Equal(t, 3, history[0].RunNumber) // Most recent
	assert.True(t, history[0].Success)
	assert.Equal(t, "agent-a", history[0].Agent)

	assert.Equal(t, 2, history[1].RunNumber) // Middle
	assert.False(t, history[1].Success)
	assert.Equal(t, "agent-b", history[1].Agent)

	assert.Equal(t, 1, history[2].RunNumber) // Oldest
	assert.True(t, history[2].Success)
	assert.Equal(t, "agent-a", history[2].Agent)
}

// TestCrossRun_MultipleFailuresAndRecovery verifies learning across multiple failed attempts
func TestCrossRun_MultipleFailuresAndRecovery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "recovery_test.db")

	planFile := "plan.yaml"
	taskNumber := "task-1"

	// Simulate 4 separate runs with different outcomes
	scenarios := []struct {
		runNumber int
		agent     string
		success   bool
		output    string
	}{
		{1, "agent-a", false, "compilation_error: undefined variable"},
		{2, "agent-a", false, "compilation_error: type mismatch"},
		{3, "agent-b", false, "test_failure: TestFoo failed"},
		{4, "agent-c", true, "All tests passed"},
	}

	for _, scenario := range scenarios {
		db, err := learning.NewStore(dbPath)
		require.NoError(t, err)

		exec := &learning.TaskExecution{
			PlanFile:     planFile,
			RunNumber:    scenario.runNumber,
			TaskNumber:   taskNumber,
			TaskName:     "Problematic Task",
			Agent:        scenario.agent,
			Prompt:       "Fix the bug",
			Success:      scenario.success,
			Output:       scenario.output,
			DurationSecs: 30,
		}

		if !scenario.success {
			exec.ErrorMessage = fmt.Sprintf("Run %d failed", scenario.runNumber)
		}

		err = db.RecordExecution(context.Background(), exec)
		require.NoError(t, err)

		db.Close()
	}

	// Final verification: Check learning data
	db, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer db.Close()

	history, err := db.GetExecutionHistory(context.Background(), "plan.yaml", taskNumber)
	require.NoError(t, err)
	require.Len(t, history, 4)

	// Verify final run succeeded
	assert.True(t, history[0].Success)
	assert.Equal(t, "agent-c", history[0].Agent)
	assert.Equal(t, 4, history[0].RunNumber)

	// Verify failure analysis
	analysis, err := db.AnalyzeFailures(context.Background(), planFile, taskNumber, 2)
	require.NoError(t, err)

	assert.Equal(t, 4, analysis.TotalAttempts)
	assert.Equal(t, 3, analysis.FailedAttempts)
	assert.Contains(t, analysis.TriedAgents, "agent-a")
	assert.Contains(t, analysis.TriedAgents, "agent-b")
	assert.Contains(t, analysis.TriedAgents, "agent-c")

	// Should detect compilation_error pattern
	assert.Contains(t, analysis.CommonPatterns, "compilation_error")
}

// TestRetryFlow_NoSwapWithoutIntelligentSwapper tests that without IntelligentAgentSwapper, no swap occurs
func TestRetryFlow_NoSwapWithoutIntelligentSwapper(t *testing.T) {
	// Setup: Create DB for learning store
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := learning.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Mock invoker that tracks which agent was used
	// ExitCode must be 0 for QC to review - non-zero exits immediately
	mockInv := newStubInvoker(
		&agent.InvocationResult{Output: `{"status": "failed", "errors": ["compilation error"]}`, ExitCode: 0},
		&agent.InvocationResult{Output: `{"status": "success", "summary": "Fixed"}`, ExitCode: 0},
	)

	// Mock QC reviewer that returns RED then GREEN
	mockReviewer := &stubReviewer{
		results: []*ReviewResult{
			{
				Flag:     models.StatusRed,
				Feedback: "Code has compilation errors",
			},
			{
				Flag:     models.StatusGreen,
				Feedback: "All criteria met",
			},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	// Mock updater
	updater := &recordingUpdater{}

	// Create executor with learning store but NO IntelligentAgentSwapper
	executor, err := NewTaskExecutor(mockInv, mockReviewer, updater, TaskExecutorConfig{
		QualityControl: models.QualityControlConfig{
			Enabled:    true, // Enable QC
			RetryOnRed: 2,    // Enable retries on RED verdict
		},
	})
	require.NoError(t, err)
	executor.LearningStore = store
	executor.SwapDuringRetries = true
	// Note: IntelligentAgentSwapper is NOT set

	// Execute task with initial agent
	task := models.Task{
		Number: "1",
		Name:   "Test no swap without swapper",
		Prompt: "Fix compilation error",
		Agent:  "backend-developer", // Initial agent
	}

	result, err := executor.Execute(context.Background(), task)

	// Verify execution completed
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Key assertion: Without IntelligentAgentSwapper, agent should NOT be swapped
	// All calls should use the same agent
	agentsUsed := []string{}
	for _, call := range mockInv.calls {
		agentsUsed = append(agentsUsed, call.Agent)
	}

	assert.GreaterOrEqual(t, len(agentsUsed), 2, "Expected at least 2 agent invocations")
	for _, agent := range agentsUsed {
		assert.Equal(t, "backend-developer", agent, "Without IntelligentAgentSwapper, agent should not swap")
	}

	// Verify final result still has original agent
	assert.Equal(t, "backend-developer", result.Task.Agent, "Task should end with original agent")
}

// TestRetryFlow_FeedbackStorageFormats tests dual feedback storage
func TestRetryFlow_FeedbackStorageFormats(t *testing.T) {
	// Setup: Create plan file and task executor
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")

	// Write minimal plan file
	planContent := `# Test Plan

## Task 1: Test Task
**Files**: internal/test.go
**Agent**: backend-developer

Implement test functionality.
`
	err := os.WriteFile(planFile, []byte(planContent), 0644)
	require.NoError(t, err)

	// Create mock invoker that simulates:
	// 1. First attempt: fails with structured output
	// 2. Second attempt: succeeds
	mockInvoker := newStubInvoker(
		&agent.InvocationResult{
			Output:   `{"status": "failed", "errors": ["validation error"]}`,
			ExitCode: 0,
			Duration: 50 * time.Millisecond,
		},
		&agent.InvocationResult{
			Output:   `{"status": "success", "summary": "Task completed"}`,
			ExitCode: 0,
			Duration: 75 * time.Millisecond,
		},
	)

	// Create mock QC reviewer that returns structured feedback
	mockReviewer := &stubReviewer{
		results: []*ReviewResult{
			{
				Flag:      models.StatusRed,
				Feedback:  "Output validation failed: missing required fields",
				AgentName: "quality-control",
			},
			{
				Flag:      models.StatusGreen,
				Feedback:  "All validation checks passed",
				AgentName: "quality-control",
			},
		},
		retryDecisions: map[int]bool{0: true, 1: false},
	}

	// Create mock updater to verify UpdateTaskFeedback calls
	recordingUpdater := &recordingUpdater{}

	// Create task executor with QC enabled
	executor, err := NewTaskExecutor(mockInvoker, mockReviewer, recordingUpdater, TaskExecutorConfig{
		PlanPath: planFile,
		QualityControl: models.QualityControlConfig{
			Enabled:    true,
			RetryOnRed: 2,
		},
	})
	require.NoError(t, err)

	// Create test task
	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		Prompt: "Implement test functionality",
		Agent:  "backend-developer",
		Files:  []string{"internal/test.go"},
	}

	// Execute task
	result, err := executor.Execute(context.Background(), task)

	// Verify execution completed successfully
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusGreen, result.Status)
	assert.Equal(t, 1, result.RetryCount, "Should have retried once")

	// Verify execution history is populated
	assert.Len(t, result.ExecutionHistory, 2, "Should have 2 execution attempts")

	// Verify first attempt
	assert.Equal(t, 1, result.ExecutionHistory[0].Attempt)
	assert.Equal(t, "backend-developer", result.ExecutionHistory[0].Agent)
	assert.Equal(t, models.StatusRed, result.ExecutionHistory[0].Verdict)
	assert.Contains(t, result.ExecutionHistory[0].QCFeedback, "validation failed", "QC feedback should be stored")

	// Verify second attempt
	assert.Equal(t, 2, result.ExecutionHistory[1].Attempt)
	assert.Equal(t, "backend-developer", result.ExecutionHistory[1].Agent)
	assert.Equal(t, models.StatusGreen, result.ExecutionHistory[1].Verdict)
	assert.Contains(t, result.ExecutionHistory[1].QCFeedback, "passed", "QC feedback should be stored")

	// Verify ReviewFeedback contains concatenated feedback from last attempt
	assert.NotEmpty(t, result.ReviewFeedback, "ReviewFeedback should be populated")

	// Verify updater was called to persist feedback
	assert.Greater(t, len(recordingUpdater.calls), 0, "UpdateTaskFeedback should have been called")

	// Verify final status is completed
	assert.Equal(t, models.StatusGreen, result.Status)
}

// TestRetryFlow_ConfigVariations tests retry behavior with different config flags
func TestRetryFlow_ConfigVariations(t *testing.T) {
	tests := []struct {
		name             string
		setupExecutor    func(*DefaultTaskExecutor)
		expectedAgentSeq []string
		shouldSwap       bool
		description      string
	}{
		{
			name: "SwapDuringRetries disabled prevents agent swap",
			setupExecutor: func(te *DefaultTaskExecutor) {
				te.SwapDuringRetries = false // Disable swapping
			},
			expectedAgentSeq: []string{"backend-developer", "backend-developer"}, // No swap
			shouldSwap:       false,
			description:      "When SwapDuringRetries=false, agent should never swap even on RED",
		},
		{
			name: "LearningStore nil still allows swap via IntelligentAgentSwapper",
			setupExecutor: func(te *DefaultTaskExecutor) {
				te.LearningStore = nil // Disable learning store
				te.SwapDuringRetries = true
			},
			expectedAgentSeq: []string{"backend-developer", "golang-pro"}, // Still swaps via IntelligentAgentSwapper
			shouldSwap:       true,
			description:      "When LearningStore=nil, agent still swaps via IntelligentAgentSwapper",
		},
		{
			name: "SwapDuringRetries enabled allows agent swap",
			setupExecutor: func(te *DefaultTaskExecutor) {
				te.SwapDuringRetries = true
			},
			expectedAgentSeq: []string{"backend-developer", "golang-pro"}, // Swap to suggested agent
			shouldSwap:       true,
			description:      "With SwapDuringRetries=true, agent should swap on first retry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: Create DB for learning store
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")
			store, err := learning.NewStore(dbPath)
			require.NoError(t, err)
			defer store.Close()

			// Mock invoker that tracks agents used
			// Returns RED result then GREEN result
			mockInv := newStubInvoker(
				&agent.InvocationResult{Output: `{"status": "failed", "errors": ["compilation error"]}`, ExitCode: 0},
				&agent.InvocationResult{Output: `{"status": "success", "summary": "Fixed with new agent"}`, ExitCode: 0},
			)

			// Mock QC reviewer that returns RED with suggested agent, then GREEN
			mockReviewer := &stubReviewer{
				results: []*ReviewResult{
					{
						Flag:           models.StatusRed,
						Feedback:       "Code has compilation errors",
						SuggestedAgent: "golang-pro", // QC suggests better agent
					},
					{
						Flag:     models.StatusGreen,
						Feedback: "All criteria met",
					},
				},
				retryDecisions: map[int]bool{0: true, 1: false},
			}

			// Mock updater
			updater := &recordingUpdater{}

			// Create executor with learning store
			executor, err := NewTaskExecutor(mockInv, mockReviewer, updater, TaskExecutorConfig{
				QualityControl: models.QualityControlConfig{
					Enabled:    true,
					RetryOnRed: 2,
				},
			})
			require.NoError(t, err)

			// Apply config variations
			executor.LearningStore = store
			tt.setupExecutor(executor)

			// Execute task with initial agent
			task := models.Task{
				Number: "1",
				Name:   "Test config variation",
				Prompt: "Fix compilation error",
				Agent:  "backend-developer",
			}

			result, err := executor.Execute(context.Background(), task)

			// Verify execution completed
			assert.NoError(t, err, tt.description)
			assert.NotNil(t, result, tt.description)

			// Verify agent sequence matches expected behavior
			agentsUsed := []string{}
			for _, call := range mockInv.calls {
				agentsUsed = append(agentsUsed, call.Agent)
			}

			assert.GreaterOrEqual(t, len(agentsUsed), len(tt.expectedAgentSeq),
				"Expected at least %d agent invocations", len(tt.expectedAgentSeq))

			if len(agentsUsed) >= len(tt.expectedAgentSeq) {
				for i, expectedAgent := range tt.expectedAgentSeq {
					assert.Equal(t, expectedAgent, agentsUsed[i],
						"%s: Agent at position %d should be %s", tt.description, i, expectedAgent)
				}
			}

			// Verify final result agent state
			if tt.shouldSwap {
				// When swap occurs, final agent should be the suggested agent
				assert.Equal(t, "golang-pro", result.Task.Agent,
					"%s: Task should end with suggested agent after swap", tt.description)
			} else {
				// When no swap, agent should remain unchanged or be original
				// (depends on config, but agent should not be swapped to suggestion)
				assert.NotEqual(t, "golang-pro", result.Task.Agent,
					"%s: Task should NOT swap to suggested agent", tt.description)
			}
		})
	}
}
