// Package executor provides task execution orchestration for Conductor.
package executor

import (
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/models"
)

// WaveAnomaly represents an anomaly detected during wave execution
type WaveAnomaly struct {
	Type        string // "consecutive_failures", "high_error_rate", "duration_outlier"
	Description string // Human-readable description
	Severity    string // "low", "medium", "high"
	TaskNumber  string // Task where anomaly occurred
	WaveName    string // Wave where anomaly occurred
}

// Getter methods for interface compliance (logger decoupling)

// GetType returns the anomaly type
func (wa WaveAnomaly) GetType() string { return wa.Type }

// GetDescription returns the human-readable description
func (wa WaveAnomaly) GetDescription() string { return wa.Description }

// GetSeverity returns the severity level
func (wa WaveAnomaly) GetSeverity() string { return wa.Severity }

// GetTaskNumber returns the task number where anomaly occurred
func (wa WaveAnomaly) GetTaskNumber() string { return wa.TaskNumber }

// GetWaveName returns the wave name where anomaly occurred
func (wa WaveAnomaly) GetWaveName() string { return wa.WaveName }

// AnomalyMonitorConfig holds configuration for anomaly detection
type AnomalyMonitorConfig struct {
	// ConsecutiveFailureThreshold triggers alert after N consecutive failures (default: 3)
	ConsecutiveFailureThreshold int

	// ErrorRateThreshold triggers alert when error rate exceeds this percentage (0.0-1.0, default: 0.5)
	ErrorRateThreshold float64

	// DurationDeviationThreshold triggers alert when duration exceeds N times the estimate (default: 2.0)
	DurationDeviationThreshold float64
}

// DefaultAnomalyMonitorConfig returns sensible defaults
func DefaultAnomalyMonitorConfig() AnomalyMonitorConfig {
	return AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 3,
		ErrorRateThreshold:          0.5,
		DurationDeviationThreshold:  2.0,
	}
}

// AnomalyMonitor watches for anomalous patterns during wave execution
type AnomalyMonitor struct {
	config              AnomalyMonitorConfig
	waveName            string
	consecutiveFailures int
	totalTasks          int
	failedTasks         int
	results             []models.TaskResult
}

// NewAnomalyMonitor creates a new AnomalyMonitor with default configuration
func NewAnomalyMonitor(waveName string) *AnomalyMonitor {
	return &AnomalyMonitor{
		config:   DefaultAnomalyMonitorConfig(),
		waveName: waveName,
		results:  make([]models.TaskResult, 0),
	}
}

// NewAnomalyMonitorWithConfig creates a new AnomalyMonitor with custom configuration
func NewAnomalyMonitorWithConfig(waveName string, config AnomalyMonitorConfig) *AnomalyMonitor {
	return &AnomalyMonitor{
		config:   config,
		waveName: waveName,
		results:  make([]models.TaskResult, 0),
	}
}

// RecordResult records a task result and checks for anomalies
// Returns any anomalies detected after this result
func (am *AnomalyMonitor) RecordResult(result models.TaskResult) []WaveAnomaly {
	am.results = append(am.results, result)
	am.totalTasks++

	anomalies := make([]WaveAnomaly, 0)

	// Check if this is a failure
	isFailed := result.Status == models.StatusRed ||
		result.Status == models.StatusFailed ||
		result.Error != nil

	if isFailed {
		am.consecutiveFailures++
		am.failedTasks++

		// Check for consecutive failures anomaly
		if am.consecutiveFailures >= am.config.ConsecutiveFailureThreshold {
			anomalies = append(anomalies, WaveAnomaly{
				Type:        "consecutive_failures",
				Description: fmt.Sprintf("%d consecutive task failures detected", am.consecutiveFailures),
				Severity:    am.getConsecutiveFailureSeverity(),
				TaskNumber:  result.Task.Number,
				WaveName:    am.waveName,
			})
		}

		// Check for high error rate anomaly
		errorRate := float64(am.failedTasks) / float64(am.totalTasks)
		if am.totalTasks >= 3 && errorRate >= am.config.ErrorRateThreshold {
			anomalies = append(anomalies, WaveAnomaly{
				Type:        "high_error_rate",
				Description: fmt.Sprintf("%.0f%% error rate in wave (%.0f threshold)", errorRate*100, am.config.ErrorRateThreshold*100),
				Severity:    am.getErrorRateSeverity(errorRate),
				TaskNumber:  result.Task.Number,
				WaveName:    am.waveName,
			})
		}
	} else {
		// Reset consecutive failures on success
		am.consecutiveFailures = 0
	}

	// Check for duration outlier if estimate is available
	if result.Task.EstimatedTime > 0 && result.Duration > 0 {
		estimate := result.Task.EstimatedTime
		actual := result.Duration
		deviation := float64(actual) / float64(estimate)

		if deviation >= am.config.DurationDeviationThreshold {
			anomalies = append(anomalies, WaveAnomaly{
				Type: "duration_outlier",
				Description: fmt.Sprintf("Task took %.1fx longer than estimated (%s vs %s estimated)",
					deviation, formatDuration(actual), formatDuration(estimate)),
				Severity:   am.getDurationSeverity(deviation),
				TaskNumber: result.Task.Number,
				WaveName:   am.waveName,
			})
		}
	}

	return anomalies
}

// CheckWaveHealth returns overall wave health assessment
func (am *AnomalyMonitor) CheckWaveHealth() (healthy bool, anomalies []WaveAnomaly) {
	anomalies = make([]WaveAnomaly, 0)

	if am.totalTasks == 0 {
		return true, anomalies
	}

	// Check final error rate
	errorRate := float64(am.failedTasks) / float64(am.totalTasks)
	if errorRate >= am.config.ErrorRateThreshold {
		anomalies = append(anomalies, WaveAnomaly{
			Type:        "high_error_rate",
			Description: fmt.Sprintf("Wave completed with %.0f%% failure rate", errorRate*100),
			Severity:    am.getErrorRateSeverity(errorRate),
			TaskNumber:  "",
			WaveName:    am.waveName,
		})
	}

	// Check if there were any consecutive failure streaks that ended the wave
	if am.consecutiveFailures >= am.config.ConsecutiveFailureThreshold {
		anomalies = append(anomalies, WaveAnomaly{
			Type:        "consecutive_failures",
			Description: fmt.Sprintf("Wave ended with %d consecutive failures", am.consecutiveFailures),
			Severity:    "high",
			TaskNumber:  "",
			WaveName:    am.waveName,
		})
	}

	healthy = len(anomalies) == 0
	return healthy, anomalies
}

// Reset resets the monitor for a new wave
func (am *AnomalyMonitor) Reset(waveName string) {
	am.waveName = waveName
	am.consecutiveFailures = 0
	am.totalTasks = 0
	am.failedTasks = 0
	am.results = make([]models.TaskResult, 0)
}

// GetStats returns current monitoring statistics
func (am *AnomalyMonitor) GetStats() (total int, failed int, consecutive int) {
	return am.totalTasks, am.failedTasks, am.consecutiveFailures
}

// getConsecutiveFailureSeverity determines severity based on consecutive failure count
func (am *AnomalyMonitor) getConsecutiveFailureSeverity() string {
	if am.consecutiveFailures >= 5 {
		return "high"
	} else if am.consecutiveFailures >= 4 {
		return "medium"
	}
	return "low"
}

// getErrorRateSeverity determines severity based on error rate
func (am *AnomalyMonitor) getErrorRateSeverity(rate float64) string {
	if rate >= 0.8 {
		return "high"
	} else if rate >= 0.6 {
		return "medium"
	}
	return "low"
}

// getDurationSeverity determines severity based on duration deviation
func (am *AnomalyMonitor) getDurationSeverity(deviation float64) string {
	if deviation >= 5.0 {
		return "high"
	} else if deviation >= 3.0 {
		return "medium"
	}
	return "low"
}

// formatDuration formats a duration for human readability
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
