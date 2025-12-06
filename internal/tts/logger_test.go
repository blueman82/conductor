package tts

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

// Compile-time check that TTSLogger implements executor.Logger.
var _ executor.Logger = (*TTSLogger)(nil)

func TestNewTTSLogger(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.announcer != announcer {
		t.Error("expected logger to have the provided announcer")
	}
}

func TestTTSLogger_LogWaveStart(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2", "3"},
	}

	logger.LogWaveStart(wave)

	if len(mock.spokenText) != 1 {
		t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	expected := "Starting Wave 1 with 3 tasks"
	if mock.spokenText[0] != expected {
		t.Errorf("expected %q, got %q", expected, mock.spokenText[0])
	}
}

func TestTTSLogger_LogWaveComplete(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2"},
	}
	results := []models.TaskResult{
		{Status: models.StatusGreen},
		{Status: models.StatusRed},
	}

	logger.LogWaveComplete(wave, 5*time.Second, results)

	if len(mock.spokenText) != 1 {
		t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	expected := "Wave 1 completed with 1 failures"
	if mock.spokenText[0] != expected {
		t.Errorf("expected %q, got %q", expected, mock.spokenText[0])
	}
}

func TestTTSLogger_LogWaveComplete_AllPassed(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	wave := models.Wave{
		Name:        "Wave 2",
		TaskNumbers: []string{"3", "4"},
	}
	results := []models.TaskResult{
		{Status: models.StatusGreen},
		{Status: models.StatusGreen},
	}

	logger.LogWaveComplete(wave, 10*time.Second, results)

	if len(mock.spokenText) != 1 {
		t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	expected := "Wave 2 completed, all tasks passed"
	if mock.spokenText[0] != expected {
		t.Errorf("expected %q, got %q", expected, mock.spokenText[0])
	}
}

func TestTTSLogger_LogTaskResult(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	result := models.TaskResult{
		Task:   models.Task{Number: "1"},
		Status: models.StatusGreen,
	}

	err := logger.LogTaskResult(result)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(mock.spokenText) != 0 {
		t.Errorf("expected no spoken messages for no-op method, got %d", len(mock.spokenText))
	}
}

func TestTTSLogger_LogProgress(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	results := []models.TaskResult{
		{Task: models.Task{Number: "1"}, Status: models.StatusGreen},
		{Task: models.Task{Number: "2"}, Status: models.StatusRed},
	}

	// Should not panic and should not speak
	logger.LogProgress(results)

	if len(mock.spokenText) != 0 {
		t.Errorf("expected no spoken messages for no-op method, got %d", len(mock.spokenText))
	}
}

func TestTTSLogger_LogSummary(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	result := models.ExecutionResult{
		TotalTasks: 5,
		Completed:  4,
		Failed:     1,
	}

	logger.LogSummary(result)

	if len(mock.spokenText) != 1 {
		t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	expected := "Run completed. 4 of 5 tasks passed, 1 failed"
	if mock.spokenText[0] != expected {
		t.Errorf("expected %q, got %q", expected, mock.spokenText[0])
	}
}

func TestTTSLogger_LogSummary_AllPassed(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	result := models.ExecutionResult{
		TotalTasks: 3,
		Completed:  3,
		Failed:     0,
	}

	logger.LogSummary(result)

	if len(mock.spokenText) != 1 {
		t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	expected := "Run completed. All 3 tasks passed"
	if mock.spokenText[0] != expected {
		t.Errorf("expected %q, got %q", expected, mock.spokenText[0])
	}
}

func TestTTSLogger_LogTaskAgentInvoke(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	task := models.Task{
		Number: "1",
		Name:   "Test Task",
		Agent:  "fullstack-developer",
	}
	logger.LogTaskAgentInvoke(task)

	if len(mock.spokenText) != 1 {
		t.Errorf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	if mock.spokenText[0] != "Deploying agent fullstack-developer" {
		t.Errorf("unexpected message: %s", mock.spokenText[0])
	}
}

func TestTTSLogger_LogQCAgentSelection(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Should announce QC agents
	logger.LogQCAgentSelection([]string{"code-reviewer", "qa-expert"}, "auto")

	if len(mock.spokenText) != 1 {
		t.Errorf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	if mock.spokenText[0] != "Deploying QC agents code-reviewer and qa-expert" {
		t.Errorf("unexpected message: %s", mock.spokenText[0])
	}
}

func TestTTSLogger_LogQCIndividualVerdicts(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Should announce each agent's verdict
	logger.LogQCIndividualVerdicts(map[string]string{"code-reviewer": "GREEN", "qa-expert": "RED"})

	// Note: map iteration order is not guaranteed, so we check both messages exist
	if len(mock.spokenText) != 2 {
		t.Errorf("expected 2 spoken messages, got %d", len(mock.spokenText))
	}
}

func TestTTSLogger_LogQCAggregatedResult(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Should announce the aggregated result
	logger.LogQCAggregatedResult("GREEN", "unanimous")

	if len(mock.spokenText) != 1 {
		t.Errorf("expected 1 spoken message, got %d", len(mock.spokenText))
	}
	if mock.spokenText[0] != "QC passed" {
		t.Errorf("unexpected message: %s", mock.spokenText[0])
	}
}

func TestTTSLogger_LogQCCriteriaResults(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Should speak when there are failures
	logger.LogQCCriteriaResults("agent1", []models.CriterionResult{
		{Criterion: "test1", Passed: true},
		{Criterion: "test2", Passed: false},
	})

	// Expect 1 message (only announces when there are failures)
	if len(mock.spokenText) != 1 {
		t.Errorf("expected 1 spoken message for criteria with failures, got %d", len(mock.spokenText))
	}
}

func TestTTSLogger_LogQCCriteriaResults_AllPassed(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Should not speak when all pass (to reduce noise)
	logger.LogQCCriteriaResults("agent1", []models.CriterionResult{
		{Criterion: "test1", Passed: true},
		{Criterion: "test2", Passed: true},
	})

	if len(mock.spokenText) != 0 {
		t.Errorf("expected no spoken messages when all criteria pass, got %d", len(mock.spokenText))
	}
}

func TestTTSLogger_LogQCIntelligentSelectionMetadata(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Should speak the rationale
	logger.LogQCIntelligentSelectionMetadata("Selected agents based on Go expertise", false, "")

	// Expect 1 message (announces the rationale)
	if len(mock.spokenText) != 1 {
		t.Errorf("expected 1 spoken message for intelligent selection, got %d", len(mock.spokenText))
	}
	if len(mock.spokenText) > 0 && mock.spokenText[0] != "Selected agents based on Go expertise" {
		t.Errorf("expected rationale in spoken message, got %q", mock.spokenText[0])
	}
}

func TestTTSLogger_InterfaceCompliance(t *testing.T) {
	// This test verifies that TTSLogger fully implements executor.Logger
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	var logger executor.Logger = NewTTSLogger(announcer)

	// Call all methods to ensure they don't panic
	logger.LogWaveStart(models.Wave{Name: "Wave 1"})
	logger.LogWaveComplete(models.Wave{Name: "Wave 1"}, time.Second, nil)
	_ = logger.LogTaskResult(models.TaskResult{})
	logger.LogProgress(nil)
	logger.LogSummary(models.ExecutionResult{})
	logger.LogQCAgentSelection(nil, "")
	logger.LogQCIndividualVerdicts(nil)
	logger.LogQCAggregatedResult("", "")
	logger.LogQCCriteriaResults("", nil)
	logger.LogQCIntelligentSelectionMetadata("", false, "")

	// Should have spoken only for LogWaveStart, LogWaveComplete, and LogSummary
	if len(mock.spokenText) != 3 {
		t.Errorf("expected 3 spoken messages, got %d", len(mock.spokenText))
	}
}

func TestTTSLogger_FullWorkflow(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)
	logger := NewTTSLogger(announcer)

	// Simulate a typical execution workflow
	wave1 := models.Wave{Name: "Wave 1", TaskNumbers: []string{"1", "2"}}
	wave2 := models.Wave{Name: "Wave 2", TaskNumbers: []string{"3"}}

	logger.LogWaveStart(wave1)
	logger.LogTaskResult(models.TaskResult{Task: models.Task{Number: "1"}, Status: models.StatusGreen})
	logger.LogTaskResult(models.TaskResult{Task: models.Task{Number: "2"}, Status: models.StatusGreen})
	logger.LogProgress([]models.TaskResult{{Task: models.Task{Number: "1"}}, {Task: models.Task{Number: "2"}}})
	logger.LogWaveComplete(wave1, 5*time.Second, []models.TaskResult{
		{Status: models.StatusGreen},
		{Status: models.StatusGreen},
	})

	logger.LogWaveStart(wave2)
	logger.LogTaskResult(models.TaskResult{Task: models.Task{Number: "3"}, Status: models.StatusRed})
	logger.LogWaveComplete(wave2, 3*time.Second, []models.TaskResult{
		{Status: models.StatusRed},
	})

	logger.LogSummary(models.ExecutionResult{
		TotalTasks: 3,
		Completed:  2,
		Failed:     1,
	})

	expected := []string{
		"Starting Wave 1 with 2 tasks",
		"Wave 1 completed, all tasks passed",
		"Starting Wave 2 with 1 tasks",
		"Wave 2 completed with 1 failures",
		"Run completed. 2 of 3 tasks passed, 1 failed",
	}

	if len(mock.spokenText) != len(expected) {
		t.Fatalf("expected %d messages, got %d: %v", len(expected), len(mock.spokenText), mock.spokenText)
	}

	for i, exp := range expected {
		if mock.spokenText[i] != exp {
			t.Errorf("message %d: expected %q, got %q", i, exp, mock.spokenText[i])
		}
	}
}
