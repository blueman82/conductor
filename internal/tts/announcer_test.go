package tts

import (
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// mockSpeaker captures the text passed to Speak for verification in tests.
type mockSpeaker struct {
	spokenText []string
}

func (m *mockSpeaker) Speak(text string) {
	m.spokenText = append(m.spokenText, text)
}

func TestNewAnnouncer(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)

	if announcer == nil {
		t.Fatal("expected non-nil announcer")
	}
	if announcer.client != mock {
		t.Error("expected announcer to have the provided speaker")
	}
}

func TestAnnouncer_WaveStart(t *testing.T) {
	tests := []struct {
		name     string
		wave     models.Wave
		expected string
	}{
		{
			name: "wave with multiple tasks",
			wave: models.Wave{
				Name:        "Wave 1",
				TaskNumbers: []string{"1", "2", "3"},
			},
			expected: "Starting Wave 1 with 3 tasks",
		},
		{
			name: "wave with single task",
			wave: models.Wave{
				Name:        "Wave 2",
				TaskNumbers: []string{"4"},
			},
			expected: "Starting Wave 2 with 1 tasks",
		},
		{
			name: "wave with no tasks",
			wave: models.Wave{
				Name:        "Wave 3",
				TaskNumbers: []string{},
			},
			expected: "Starting Wave 3 with 0 tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSpeaker{}
			announcer := NewAnnouncer(mock)

			announcer.WaveStart(tt.wave)

			if len(mock.spokenText) != 1 {
				t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
			}
			if mock.spokenText[0] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, mock.spokenText[0])
			}
		})
	}
}

func TestAnnouncer_WaveComplete(t *testing.T) {
	tests := []struct {
		name     string
		wave     models.Wave
		results  []models.TaskResult
		expected string
	}{
		{
			name: "all tasks passed",
			wave: models.Wave{
				Name:        "Wave 1",
				TaskNumbers: []string{"1", "2"},
			},
			results: []models.TaskResult{
				{Status: models.StatusGreen},
				{Status: models.StatusGreen},
			},
			expected: "Wave 1 completed, all tasks passed",
		},
		{
			name: "one task failed",
			wave: models.Wave{
				Name:        "Wave 2",
				TaskNumbers: []string{"3", "4"},
			},
			results: []models.TaskResult{
				{Status: models.StatusGreen},
				{Status: models.StatusRed},
			},
			expected: "Wave 2 completed with 1 failures",
		},
		{
			name: "multiple tasks failed",
			wave: models.Wave{
				Name:        "Wave 3",
				TaskNumbers: []string{"5", "6", "7"},
			},
			results: []models.TaskResult{
				{Status: models.StatusRed},
				{Status: models.StatusRed},
				{Status: models.StatusGreen},
			},
			expected: "Wave 3 completed with 2 failures",
		},
		{
			name: "yellow status not counted as failure",
			wave: models.Wave{
				Name:        "Wave 4",
				TaskNumbers: []string{"8", "9"},
			},
			results: []models.TaskResult{
				{Status: models.StatusYellow},
				{Status: models.StatusGreen},
			},
			expected: "Wave 4 completed, all tasks passed",
		},
		{
			name: "empty results",
			wave: models.Wave{
				Name:        "Wave 5",
				TaskNumbers: []string{},
			},
			results:  []models.TaskResult{},
			expected: "Wave 5 completed, all tasks passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSpeaker{}
			announcer := NewAnnouncer(mock)

			announcer.WaveComplete(tt.wave, tt.results)

			if len(mock.spokenText) != 1 {
				t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
			}
			if mock.spokenText[0] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, mock.spokenText[0])
			}
		})
	}
}

func TestAnnouncer_QCFailure(t *testing.T) {
	tests := []struct {
		name     string
		task     models.Task
		expected string
	}{
		{
			name: "task with numeric ID",
			task: models.Task{
				Number: "1",
				Name:   "Implement feature",
			},
			expected: "Task 1 failed quality control",
		},
		{
			name: "task with alphanumeric ID",
			task: models.Task{
				Number: "2a",
				Name:   "Setup database",
			},
			expected: "Task 2a failed quality control",
		},
		{
			name: "task with complex ID",
			task: models.Task{
				Number: "10.5",
				Name:   "Refactor",
			},
			expected: "Task 10.5 failed quality control",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSpeaker{}
			announcer := NewAnnouncer(mock)

			announcer.QCFailure(tt.task)

			if len(mock.spokenText) != 1 {
				t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
			}
			if mock.spokenText[0] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, mock.spokenText[0])
			}
		})
	}
}

func TestAnnouncer_RunComplete(t *testing.T) {
	tests := []struct {
		name     string
		result   models.ExecutionResult
		expected string
	}{
		{
			name: "all tasks passed",
			result: models.ExecutionResult{
				TotalTasks: 5,
				Completed:  5,
				Failed:     0,
			},
			expected: "Run completed. All 5 tasks passed",
		},
		{
			name: "some tasks failed",
			result: models.ExecutionResult{
				TotalTasks: 10,
				Completed:  7,
				Failed:     3,
			},
			expected: "Run completed. 7 of 10 tasks passed, 3 failed",
		},
		{
			name: "single task passed",
			result: models.ExecutionResult{
				TotalTasks: 1,
				Completed:  1,
				Failed:     0,
			},
			expected: "Run completed. All 1 tasks passed",
		},
		{
			name: "all tasks failed",
			result: models.ExecutionResult{
				TotalTasks: 3,
				Completed:  0,
				Failed:     3,
			},
			expected: "Run completed. 0 of 3 tasks passed, 3 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSpeaker{}
			announcer := NewAnnouncer(mock)

			announcer.RunComplete(tt.result)

			if len(mock.spokenText) != 1 {
				t.Fatalf("expected 1 spoken message, got %d", len(mock.spokenText))
			}
			if mock.spokenText[0] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, mock.spokenText[0])
			}
		})
	}
}

func TestAnnouncer_MultipleEvents(t *testing.T) {
	mock := &mockSpeaker{}
	announcer := NewAnnouncer(mock)

	wave := models.Wave{
		Name:        "Wave 1",
		TaskNumbers: []string{"1", "2"},
	}

	task := models.Task{
		Number: "1",
		Name:   "First task",
	}

	results := []models.TaskResult{
		{Status: models.StatusGreen},
		{Status: models.StatusRed},
	}

	execResult := models.ExecutionResult{
		TotalTasks: 2,
		Completed:  1,
		Failed:     1,
	}

	// Trigger multiple announcements
	announcer.WaveStart(wave)
	announcer.QCFailure(task)
	announcer.WaveComplete(wave, results)
	announcer.RunComplete(execResult)

	// Verify all messages were spoken in order
	expected := []string{
		"Starting Wave 1 with 2 tasks",
		"Task 1 failed quality control",
		"Wave 1 completed with 1 failures",
		"Run completed. 1 of 2 tasks passed, 1 failed",
	}

	if len(mock.spokenText) != len(expected) {
		t.Fatalf("expected %d messages, got %d", len(expected), len(mock.spokenText))
	}

	for i, exp := range expected {
		if mock.spokenText[i] != exp {
			t.Errorf("message %d: expected %q, got %q", i, exp, mock.spokenText[i])
		}
	}
}

func TestAnnouncer_ClientImplementsSpeaker(t *testing.T) {
	// Verify that *Client implements Speaker interface
	var _ Speaker = (*Client)(nil)
}
