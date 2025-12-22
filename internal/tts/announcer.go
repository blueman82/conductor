// Package tts provides text-to-speech functionality for conductor.
package tts

import (
	"fmt"
	"time"

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

// QCIndividualVerdicts announces verdicts from individual QC agents.
func (a *Announcer) QCIndividualVerdicts(verdicts map[string]string) {
	if len(verdicts) == 0 {
		return
	}
	// Announce each agent's verdict
	for agent, verdict := range verdicts {
		msg := fmt.Sprintf("%s says %s", agent, verdict)
		a.client.Speak(msg)
	}
}

// QCAggregatedResult announces the final QC result.
func (a *Announcer) QCAggregatedResult(verdict string, strategy string) {
	if verdict == "" {
		return
	}
	var msg string
	switch verdict {
	case "GREEN":
		msg = "QC passed"
	case "RED":
		msg = "QC failed"
	case "YELLOW":
		msg = "QC passed with warnings"
	default:
		msg = fmt.Sprintf("QC result: %s", verdict)
	}
	a.client.Speak(msg)
}

// QCIntelligentSelectionRationale announces the QC agent selection rationale.
func (a *Announcer) QCIntelligentSelectionRationale(rationale string) {
	if rationale == "" {
		return
	}
	// Speak the full rationale
	a.client.Speak(rationale)
}

// QCCriteriaResults announces criteria pass/fail summary for an agent.
// Only announces if there are failures to avoid noise.
func (a *Announcer) QCCriteriaResults(agentName string, results []models.CriterionResult) {
	if len(results) == 0 {
		return
	}

	// Count passes and failures
	passed := 0
	for _, cr := range results {
		if cr.Passed {
			passed++
		}
	}
	failed := len(results) - passed

	// Only announce if there are failures (reduce noise)
	if failed > 0 {
		var msg string
		if len(results) == 1 {
			msg = fmt.Sprintf("%s reports criterion failed", agentName)
		} else {
			msg = fmt.Sprintf("%s reports %d of %d criteria failed", agentName, failed, len(results))
		}
		a.client.Speak(msg)
	}
}

// GuardResultDisplay interface for type assertion from interface{} parameter.
// Matches the interface implemented by executor.GuardResult.
type GuardResultDisplay interface {
	GetTaskNumber() string
	GetProbability() float64
	GetConfidence() float64
	GetRiskLevel() string
	GetShouldBlock() bool
	GetBlockReason() string
	GetRecommendations() []string
}

// GuardPrediction announces GUARD protocol predictions.
func (a *Announcer) GuardPrediction(taskNumber string, result interface{}) {
	if result == nil {
		return
	}

	// Type assert to GuardResultDisplay interface
	guard, ok := result.(GuardResultDisplay)
	if !ok {
		return
	}

	riskLevel := guard.GetRiskLevel()
	probability := guard.GetProbability() * 100

	var msg string

	if guard.GetShouldBlock() {
		// Blocked task: announce with reason
		msg = fmt.Sprintf("GUARD blocked Task %s. %s. %.0f percent failure probability",
			taskNumber, guard.GetBlockReason(), probability)
	} else if riskLevel == "high" {
		msg = fmt.Sprintf("GUARD: Task %s high risk. %.0f percent failure probability",
			taskNumber, probability)
	} else if riskLevel == "medium" {
		msg = fmt.Sprintf("GUARD: Task %s medium risk. %.0f percent failure probability",
			taskNumber, probability)
	} else {
		msg = fmt.Sprintf("GUARD: Task %s low risk", taskNumber)
	}

	if msg != "" {
		a.client.Speak(msg)
	}
}

// AgentSwap announces when GUARD predictive selection swaps to a better agent.
func (a *Announcer) AgentSwap(taskNumber string, fromAgent string, toAgent string) {
	msg := fmt.Sprintf("GUARD: Swapping Task %s from %s to %s", taskNumber, fromAgent, toAgent)
	a.client.Speak(msg)
}

// WaveAnomalyDisplay interface for type assertion from interface{} parameter.
type WaveAnomalyDisplay interface {
	GetType() string
	GetDescription() string
	GetSeverity() string
	GetTaskNumber() string
	GetWaveName() string
}

// Anomaly announces when an anomaly is detected during wave execution.
func (a *Announcer) Anomaly(anomaly interface{}) {
	if anomaly == nil {
		return
	}

	// Type assert to WaveAnomalyDisplay interface
	wa, ok := anomaly.(WaveAnomalyDisplay)
	if !ok {
		return
	}

	var msg string
	switch wa.GetType() {
	case "consecutive_failures":
		msg = fmt.Sprintf("Warning: %s", wa.GetDescription())
	case "high_error_rate":
		msg = fmt.Sprintf("Warning: High error rate detected. %s", wa.GetDescription())
	case "duration_outlier":
		if wa.GetTaskNumber() != "" {
			msg = fmt.Sprintf("Warning: Task %s duration anomaly. %s", wa.GetTaskNumber(), wa.GetDescription())
		} else {
			msg = fmt.Sprintf("Warning: Duration anomaly. %s", wa.GetDescription())
		}
	default:
		msg = fmt.Sprintf("Anomaly detected: %s", wa.GetDescription())
	}

	a.client.Speak(msg)
}

// BudgetWarning announces when approaching budget limit.
func (a *Announcer) BudgetWarning(percentUsed float64) {
	msg := fmt.Sprintf("Warning: %.0f percent of budget used", percentUsed*100)
	a.client.Speak(msg)
}

// RateLimitPause announces when pausing due to rate limit.
func (a *Announcer) RateLimitPause(delay time.Duration) {
	minutes := int(delay.Minutes())
	if minutes > 0 {
		msg := fmt.Sprintf("Rate limited. Pausing for %d minutes", minutes)
		a.client.Speak(msg)
	} else {
		msg := fmt.Sprintf("Rate limited. Pausing for %d seconds", int(delay.Seconds()))
		a.client.Speak(msg)
	}
}

// RateLimitResume announces when resuming after rate limit pause.
func (a *Announcer) RateLimitResume() {
	a.client.Speak("Rate limit cleared. Resuming execution")
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
