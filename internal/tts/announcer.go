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

// TaskAgentInvoke announces when a task agent is being deployed.
func (a *Announcer) TaskAgentInvoke(task models.Task) {
	agentName := task.Agent
	if agentName == "" {
		agentName = "default agent"
	}
	msg := fmt.Sprintf("Deploying agent %s", agentName)
	a.client.Speak(msg)
}

// QCAgentSelection announces when QC agents are being deployed for review.
func (a *Announcer) QCAgentSelection(agents []string) {
	if len(agents) == 0 {
		return
	}
	var msg string
	if len(agents) == 1 {
		msg = fmt.Sprintf("Deploying QC agent %s", agents[0])
	} else {
		// Multiple agents - list them
		msg = fmt.Sprintf("Deploying QC agents %s", joinAgents(agents))
	}
	a.client.Speak(msg)
}

// joinAgents creates a human-readable list of agents ("a, b, and c").
func joinAgents(agents []string) string {
	if len(agents) == 0 {
		return ""
	}
	if len(agents) == 1 {
		return agents[0]
	}
	if len(agents) == 2 {
		return agents[0] + " and " + agents[1]
	}
	// 3 or more: "a, b, and c"
	result := ""
	for i, a := range agents {
		if i == len(agents)-1 {
			result += "and " + a
		} else {
			result += a + ", "
		}
	}
	return result
}
