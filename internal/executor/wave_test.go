package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

type sequentialMockExecutor struct {
	wave1Done chan struct{}
	mu        sync.Mutex
	wave1Seen int
}

func newSequentialMockExecutor() *sequentialMockExecutor {
	return &sequentialMockExecutor{
		wave1Done: make(chan struct{}),
	}
}

func (m *sequentialMockExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	if task.Number == 1 || task.Number == 2 {
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
		m.wave1Seen++
		if m.wave1Seen == 2 {
			close(m.wave1Done)
		}
		m.mu.Unlock()
	} else {
		select {
		case <-m.wave1Done:
		default:
			return models.TaskResult{Task: task, Status: models.StatusFailed}, fmt.Errorf("wave 2 task %d executed before wave 1 completion", task.Number)
		}
	}

	return models.TaskResult{
		Task:   task,
		Status: models.StatusGreen,
	}, nil
}

func TestWaveExecutor_WavesExecuteSequentially(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: 1, Name: "Task 1", Prompt: "Do task 1"},
			{Number: 2, Name: "Task 2", Prompt: "Do task 2"},
			{Number: 3, Name: "Task 3", Prompt: "Do task 3"},
			{Number: 4, Name: "Task 4", Prompt: "Do task 4"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1, 2}, MaxConcurrency: 2},
			{Name: "Wave 2", TaskNumbers: []int{3, 4}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newSequentialMockExecutor()
	waveExecutor := NewWaveExecutor(mockExecutor)

	results, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if len(results) != len(plan.Tasks) {
		t.Fatalf("expected %d results, got %d", len(plan.Tasks), len(results))
	}
}

type concurrencyMockExecutor struct {
	mu       sync.Mutex
	current  int
	maxSeen  int
	duration time.Duration
}

func newConcurrencyMockExecutor(duration time.Duration) *concurrencyMockExecutor {
	return &concurrencyMockExecutor{duration: duration}
}

func (m *concurrencyMockExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	m.mu.Lock()
	m.current++
	if m.current > m.maxSeen {
		m.maxSeen = m.current
	}
	m.mu.Unlock()

	select {
	case <-ctx.Done():
	case <-time.After(m.duration):
	}

	m.mu.Lock()
	m.current--
	m.mu.Unlock()

	return models.TaskResult{Task: task, Status: models.StatusGreen}, nil
}

func TestWaveExecutor_RespectsMaxConcurrency(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: 1, Name: "Task 1", Prompt: "Do task 1"},
			{Number: 2, Name: "Task 2", Prompt: "Do task 2"},
			{Number: 3, Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1, 2, 3}, MaxConcurrency: 2},
		},
	}

	mockExecutor := newConcurrencyMockExecutor(25 * time.Millisecond)
	waveExecutor := NewWaveExecutor(mockExecutor)

	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan returned error: %v", err)
	}

	if mockExecutor.maxSeen > plan.Waves[0].MaxConcurrency {
		t.Fatalf("expected max concurrency <= %d, got %d", plan.Waves[0].MaxConcurrency, mockExecutor.maxSeen)
	}
}

type cancellableMockExecutor struct {
	cancel func()
	mu     sync.Mutex
	calls  int
}

func (m *cancellableMockExecutor) Execute(ctx context.Context, task models.Task) (models.TaskResult, error) {
	m.mu.Lock()
	m.calls++
	callNumber := m.calls
	m.mu.Unlock()

	if callNumber == 1 && m.cancel != nil {
		m.cancel()
	}

	select {
	case <-ctx.Done():
		return models.TaskResult{Task: task, Status: models.StatusFailed, Error: ctx.Err()}, ctx.Err()
	case <-time.After(5 * time.Millisecond):
		return models.TaskResult{Task: task, Status: models.StatusGreen}, nil
	}
}

func TestWaveExecutor_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockExecutor := &cancellableMockExecutor{cancel: cancel}
	waveExecutor := NewWaveExecutor(mockExecutor)

	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: 1, Name: "Task 1", Prompt: "Do task 1"},
			{Number: 2, Name: "Task 2", Prompt: "Do task 2"},
			{Number: 3, Name: "Task 3", Prompt: "Do task 3"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1, 2, 3}, MaxConcurrency: 2},
		},
	}

	results, err := waveExecutor.ExecutePlan(ctx, plan)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}

	if len(results) == len(plan.Tasks) {
		t.Fatalf("expected fewer results due to cancellation, got %d", len(results))
	}
}

func TestWaveExecutor_ErrorsOnMissingTask(t *testing.T) {
	plan := &models.Plan{
		Tasks: []models.Task{
			{Number: 1, Name: "Task 1", Prompt: "Do task 1"},
		},
		Waves: []models.Wave{
			{Name: "Wave 1", TaskNumbers: []int{1, 99}, MaxConcurrency: 1},
		},
	}

	waveExecutor := NewWaveExecutor(&sequentialMockExecutor{wave1Done: make(chan struct{})})

	_, err := waveExecutor.ExecutePlan(context.Background(), plan)
	if err == nil {
		t.Fatalf("expected error for missing task")
	}
	if err != nil && !strings.Contains(err.Error(), "task 99") {
		t.Fatalf("expected error mentioning missing task, got %v", err)
	}
}
