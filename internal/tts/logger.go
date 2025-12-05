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

// LogQCCriteriaResults is a no-op implementation.
func (l *TTSLogger) LogQCCriteriaResults(agentName string, results []models.CriterionResult) {
}

// LogQCIntelligentSelectionMetadata delegates to announcer.QCIntelligentSelectionRationale.
func (l *TTSLogger) LogQCIntelligentSelectionMetadata(rationale string, fallback bool, fallbackReason string) {
	l.announcer.QCIntelligentSelectionRationale(rationale)
}
