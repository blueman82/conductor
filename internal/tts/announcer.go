// Package tts provides text-to-speech functionality for conductor.
package tts

import (
	"fmt"

	"github.com/harrison/conductor/internal/models"
)

// Speaker interface defines the contract for TTS capabilities.
// This allows for dependency injection and easy testing.
type Speaker interface {
	Speak(text string)
}

// Announcer wraps a Speaker and generates human-readable messages for execution events.
type Announcer struct {
	client Speaker
}

// NewAnnouncer creates a new Announcer with the given Speaker.
func NewAnnouncer(client Speaker) *Announcer {
	return &Announcer{
		client: client,
	}
}

// WaveStart announces the start of a wave execution.
func (a *Announcer) WaveStart(wave models.Wave) {
	msg := fmt.Sprintf("Starting %s with %d tasks", wave.Name, len(wave.TaskNumbers))
	a.client.Speak(msg)
}

// WaveComplete announces the completion of a wave, including failure count if any.
func (a *Announcer) WaveComplete(wave models.Wave, results []models.TaskResult) {
	// Count RED statuses in results
	redCount := 0
	for _, result := range results {
		if result.Status == models.StatusRed {
			redCount++
		}
	}

	var msg string
	if redCount > 0 {
		msg = fmt.Sprintf("%s completed with %d failures", wave.Name, redCount)
	} else {
		msg = fmt.Sprintf("%s completed, all tasks passed", wave.Name)
	}
	a.client.Speak(msg)
}

// QCFailure announces when a task fails quality control.
func (a *Announcer) QCFailure(task models.Task) {
	msg := fmt.Sprintf("Task %s failed quality control", task.Number)
	a.client.Speak(msg)
}

// RunComplete announces the completion of the entire run.
func (a *Announcer) RunComplete(result models.ExecutionResult) {
	var msg string
	if result.Failed > 0 {
		msg = fmt.Sprintf("Run completed. %d of %d tasks passed, %d failed", result.Completed, result.TotalTasks, result.Failed)
	} else {
		msg = fmt.Sprintf("Run completed. All %d tasks passed", result.TotalTasks)
	}
	a.client.Speak(msg)
}
