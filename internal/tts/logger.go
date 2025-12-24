// Package tts provides text-to-speech functionality for conductor.
package tts

import (
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

// Compile-time interface compliance check.
var _ executor.Logger = (*TTSLogger)(nil)

// TTSLogger wraps an Announcer and implements executor.Logger.
// Delegates execution events to the announcer for voice feedback:
// - LogWaveStart, LogWaveComplete, LogSummary
// - LogQCAgentSelection, LogQCIndividualVerdicts, LogQCAggregatedResult
type TTSLogger struct {
	announcer *Announcer
}

// NewTTSLogger creates a new TTSLogger with the given Announcer.
func NewTTSLogger(announcer *Announcer) *TTSLogger {
	return &TTSLogger{
		announcer: announcer,
	}
}

// LogWaveStart delegates to announcer.WaveStart.
func (l *TTSLogger) LogWaveStart(wave models.Wave) {
	l.announcer.WaveStart(wave)
}

// LogWaveComplete delegates to announcer.WaveComplete (duration is ignored).
func (l *TTSLogger) LogWaveComplete(wave models.Wave, duration time.Duration, results []models.TaskResult) {
	l.announcer.WaveComplete(wave, results)
}

// LogTaskResult is a no-op implementation.
func (l *TTSLogger) LogTaskResult(result models.TaskResult) error {
	return nil
}

// LogProgress is a no-op implementation.
func (l *TTSLogger) LogProgress(results []models.TaskResult) {
}

// LogSummary delegates to announcer.RunComplete.
func (l *TTSLogger) LogSummary(result models.ExecutionResult) {
	l.announcer.RunComplete(result)
}

// LogTaskAgentInvoke delegates to announcer.TaskAgentInvoke.
func (l *TTSLogger) LogTaskAgentInvoke(task models.Task) {
	l.announcer.TaskAgentInvoke(task)
}

// LogQCAgentSelection delegates to announcer.QCAgentSelection.
func (l *TTSLogger) LogQCAgentSelection(agents []string, mode string) {
	l.announcer.QCAgentSelection(agents)
}

// LogQCIndividualVerdicts delegates to announcer.QCIndividualVerdicts.
func (l *TTSLogger) LogQCIndividualVerdicts(verdicts map[string]string) {
	l.announcer.QCIndividualVerdicts(verdicts)
}

// LogQCAggregatedResult delegates to announcer.QCAggregatedResult.
func (l *TTSLogger) LogQCAggregatedResult(verdict string, strategy string) {
	l.announcer.QCAggregatedResult(verdict, strategy)
}

// LogQCCriteriaResults delegates to announcer.QCCriteriaResults.
// Only announces failures to reduce noise.
func (l *TTSLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
	l.announcer.QCCriteriaResults(agentName, results)
}

// LogQCIntelligentSelectionMetadata delegates to announcer.QCIntelligentSelectionRationale.
func (l *TTSLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	l.announcer.QCIntelligentSelectionRationale(rationale)
}

// LogGuardPrediction delegates to announcer.GuardPrediction.
func (l *TTSLogger) LogGuardPrediction(taskNumber string, result interface{}) {
	l.announcer.GuardPrediction(taskNumber, result)
}

// LogAgentSwap delegates to announcer.AgentSwap.
func (l *TTSLogger) LogAgentSwap(taskNumber string, fromAgent string, toAgent string) {
	l.announcer.AgentSwap(taskNumber, fromAgent, toAgent)
}

// LogAnomaly delegates to announcer.Anomaly.
func (l *TTSLogger) LogAnomaly(anomaly interface{}) {
	l.announcer.Anomaly(anomaly)
}

// LogBudgetStatus is a no-op implementation.
// Budget status announcements are not implemented for TTS to reduce noise.
func (l *TTSLogger) LogBudgetStatus(status interface{}) {
	// No-op: budget status is too frequent for voice announcements
}

// LogBudgetWarning is a no-op implementation.
// Budget warnings could be announced in the future if needed.
func (l *TTSLogger) LogBudgetWarning(percentUsed float64) {
	// No-op: could announce "Budget at N percent" if desired
}

// LogRateLimitPause is a no-op implementation.
// Rate limit pauses could be announced in the future if needed.
func (l *TTSLogger) LogRateLimitPause(delay time.Duration) {
	// No-op: could announce "Pausing for rate limit" if desired
}

// LogRateLimitResume is a no-op implementation.
// Rate limit resumes could be announced in the future if needed.
func (l *TTSLogger) LogRateLimitResume() {
	// No-op: could announce "Resuming after rate limit" if desired
}

// LogRateLimitCountdown is a no-op implementation.
// Live visual countdown is handled by console logger.
func (l *TTSLogger) LogRateLimitCountdown(remaining, total time.Duration) {
	// No-op: live visual countdown is console-only
}

// LogRateLimitAnnounce speaks TTS announcements at the configured interval.
func (l *TTSLogger) LogRateLimitAnnounce(remaining, total time.Duration) {
	if l.announcer != nil {
		l.announcer.RateLimitCountdown(remaining, total)
	}
}
