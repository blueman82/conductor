package executor

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/models"
)

func TestNewAnomalyMonitor(t *testing.T) {
	monitor := NewAnomalyMonitor("Wave 1")

	if monitor == nil {
		t.Fatal("Expected non-nil monitor")
	}

	if monitor.waveName != "Wave 1" {
		t.Errorf("Expected waveName 'Wave 1', got '%s'", monitor.waveName)
	}

	// Check defaults
	if monitor.config.ConsecutiveFailureThreshold != 3 {
		t.Errorf("Expected ConsecutiveFailureThreshold 3, got %d", monitor.config.ConsecutiveFailureThreshold)
	}
	if monitor.config.ErrorRateThreshold != 0.5 {
		t.Errorf("Expected ErrorRateThreshold 0.5, got %f", monitor.config.ErrorRateThreshold)
	}
	if monitor.config.DurationDeviationThreshold != 2.0 {
		t.Errorf("Expected DurationDeviationThreshold 2.0, got %f", monitor.config.DurationDeviationThreshold)
	}
}

func TestNewAnomalyMonitorWithConfig(t *testing.T) {
	config := AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 5,
		ErrorRateThreshold:          0.7,
		DurationDeviationThreshold:  3.0,
	}
	monitor := NewAnomalyMonitorWithConfig("Wave 2", config)

	if monitor.config.ConsecutiveFailureThreshold != 5 {
		t.Errorf("Expected ConsecutiveFailureThreshold 5, got %d", monitor.config.ConsecutiveFailureThreshold)
	}
}

func TestAnomalyMonitor_ConsecutiveFailures(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 3,
		ErrorRateThreshold:          0.9, // High to avoid triggering
	})

	// First failure - no anomaly yet
	result1 := models.TaskResult{
		Task:   models.Task{Number: "1"},
		Status: models.StatusRed,
	}
	anomalies := monitor.RecordResult(result1)
	if len(anomalies) != 0 {
		t.Errorf("Expected 0 anomalies after 1 failure, got %d", len(anomalies))
	}

	// Second failure - still no anomaly
	result2 := models.TaskResult{
		Task:   models.Task{Number: "2"},
		Status: models.StatusRed,
	}
	anomalies = monitor.RecordResult(result2)
	if len(anomalies) != 0 {
		t.Errorf("Expected 0 anomalies after 2 failures, got %d", len(anomalies))
	}

	// Third failure - anomaly triggered!
	result3 := models.TaskResult{
		Task:   models.Task{Number: "3"},
		Status: models.StatusRed,
	}
	anomalies = monitor.RecordResult(result3)
	if len(anomalies) == 0 {
		t.Error("Expected consecutive_failures anomaly after 3 failures")
	}
	if anomalies[0].Type != "consecutive_failures" {
		t.Errorf("Expected type 'consecutive_failures', got '%s'", anomalies[0].Type)
	}
	if anomalies[0].TaskNumber != "3" {
		t.Errorf("Expected task number '3', got '%s'", anomalies[0].TaskNumber)
	}
}

func TestAnomalyMonitor_ConsecutiveFailuresReset(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 3,
		ErrorRateThreshold:          0.9, // High to avoid triggering
	})

	// Two failures
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "1"}, Status: models.StatusRed})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "2"}, Status: models.StatusRed})

	// Success resets counter
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "3"}, Status: models.StatusGreen})

	// Verify counter was reset
	total, failed, consecutive := monitor.GetStats()
	if consecutive != 0 {
		t.Errorf("Expected consecutive failures 0 after success, got %d", consecutive)
	}
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
	if failed != 2 {
		t.Errorf("Expected failed 2, got %d", failed)
	}
}

func TestAnomalyMonitor_HighErrorRate(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 10,  // High to avoid triggering
		ErrorRateThreshold:          0.5, // 50% error rate
	})

	// 2 failures, 1 success = 67% error rate, but only 3 tasks needed for alert
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "1"}, Status: models.StatusRed})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "2"}, Status: models.StatusRed})

	anomalies := monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "3"}, Status: models.StatusRed})

	// Should trigger high error rate
	hasHighErrorRate := false
	for _, a := range anomalies {
		if a.Type == "high_error_rate" {
			hasHighErrorRate = true
		}
	}
	if !hasHighErrorRate {
		t.Error("Expected high_error_rate anomaly")
	}
}

func TestAnomalyMonitor_DurationOutlier(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 10,  // High to avoid triggering
		ErrorRateThreshold:          0.9, // High to avoid triggering
		DurationDeviationThreshold:  2.0,
	})

	// Task with estimate of 1 minute, actual 3 minutes (3x = outlier)
	result := models.TaskResult{
		Task: models.Task{
			Number:        "1",
			EstimatedTime: 1 * time.Minute,
		},
		Status:   models.StatusGreen,
		Duration: 3 * time.Minute,
	}

	anomalies := monitor.RecordResult(result)

	hasDurationOutlier := false
	for _, a := range anomalies {
		if a.Type == "duration_outlier" {
			hasDurationOutlier = true
			if a.Severity != "medium" && a.Severity != "low" {
				// 3x is medium severity
			}
		}
	}
	if !hasDurationOutlier {
		t.Error("Expected duration_outlier anomaly for 3x duration")
	}
}

func TestAnomalyMonitor_NoDurationOutlier(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 10,
		ErrorRateThreshold:          0.9,
		DurationDeviationThreshold:  2.0,
	})

	// Task with estimate of 1 minute, actual 1.5 minutes (1.5x < 2x threshold)
	result := models.TaskResult{
		Task: models.Task{
			Number:        "1",
			EstimatedTime: 1 * time.Minute,
		},
		Status:   models.StatusGreen,
		Duration: 90 * time.Second,
	}

	anomalies := monitor.RecordResult(result)
	if len(anomalies) != 0 {
		t.Errorf("Expected no anomalies for 1.5x duration, got %d", len(anomalies))
	}
}

func TestAnomalyMonitor_CheckWaveHealth(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 3,
		ErrorRateThreshold:          0.5,
	})

	// All green - healthy
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "1"}, Status: models.StatusGreen})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "2"}, Status: models.StatusGreen})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "3"}, Status: models.StatusGreen})

	healthy, anomalies := monitor.CheckWaveHealth()
	if !healthy {
		t.Error("Expected wave to be healthy")
	}
	if len(anomalies) != 0 {
		t.Errorf("Expected 0 anomalies, got %d", len(anomalies))
	}
}

func TestAnomalyMonitor_CheckWaveHealthUnhealthy(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 10,
		ErrorRateThreshold:          0.5,
	})

	// 2 failures, 1 success = 67% failure rate
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "1"}, Status: models.StatusRed})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "2"}, Status: models.StatusGreen})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "3"}, Status: models.StatusRed})

	healthy, anomalies := monitor.CheckWaveHealth()
	if healthy {
		t.Error("Expected wave to be unhealthy")
	}

	hasHighErrorRate := false
	for _, a := range anomalies {
		if a.Type == "high_error_rate" {
			hasHighErrorRate = true
		}
	}
	if !hasHighErrorRate {
		t.Error("Expected high_error_rate in wave health check")
	}
}

func TestAnomalyMonitor_Reset(t *testing.T) {
	monitor := NewAnomalyMonitor("Wave 1")

	// Record some results
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "1"}, Status: models.StatusRed})
	monitor.RecordResult(models.TaskResult{Task: models.Task{Number: "2"}, Status: models.StatusRed})

	// Reset for new wave
	monitor.Reset("Wave 2")

	if monitor.waveName != "Wave 2" {
		t.Errorf("Expected waveName 'Wave 2', got '%s'", monitor.waveName)
	}

	total, failed, consecutive := monitor.GetStats()
	if total != 0 || failed != 0 || consecutive != 0 {
		t.Errorf("Expected all stats to be 0 after reset, got total=%d, failed=%d, consecutive=%d", total, failed, consecutive)
	}
}

func TestWaveAnomaly_Getters(t *testing.T) {
	anomaly := WaveAnomaly{
		Type:        "consecutive_failures",
		Description: "3 consecutive failures",
		Severity:    "medium",
		TaskNumber:  "5",
		WaveName:    "Wave 3",
	}

	if anomaly.GetType() != "consecutive_failures" {
		t.Errorf("GetType: expected 'consecutive_failures', got '%s'", anomaly.GetType())
	}
	if anomaly.GetDescription() != "3 consecutive failures" {
		t.Errorf("GetDescription: expected '3 consecutive failures', got '%s'", anomaly.GetDescription())
	}
	if anomaly.GetSeverity() != "medium" {
		t.Errorf("GetSeverity: expected 'medium', got '%s'", anomaly.GetSeverity())
	}
	if anomaly.GetTaskNumber() != "5" {
		t.Errorf("GetTaskNumber: expected '5', got '%s'", anomaly.GetTaskNumber())
	}
	if anomaly.GetWaveName() != "Wave 3" {
		t.Errorf("GetWaveName: expected 'Wave 3', got '%s'", anomaly.GetWaveName())
	}
}

func TestAnomalyMonitor_SeverityLevels(t *testing.T) {
	tests := []struct {
		name             string
		consecutiveCount int
		expectedSeverity string
	}{
		{"3 consecutive", 3, "low"},
		{"4 consecutive", 4, "medium"},
		{"5 consecutive", 5, "high"},
		{"10 consecutive", 10, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
				ConsecutiveFailureThreshold: 3,
				ErrorRateThreshold:          0.99, // High to avoid
			})

			var lastAnomalies []WaveAnomaly
			for i := 0; i < tt.consecutiveCount; i++ {
				lastAnomalies = monitor.RecordResult(models.TaskResult{
					Task:   models.Task{Number: string(rune('1' + i))},
					Status: models.StatusRed,
				})
			}

			// Find consecutive_failures anomaly
			var severity string
			for _, a := range lastAnomalies {
				if a.Type == "consecutive_failures" {
					severity = a.Severity
				}
			}

			if severity != tt.expectedSeverity {
				t.Errorf("Expected severity '%s' for %d consecutive failures, got '%s'",
					tt.expectedSeverity, tt.consecutiveCount, severity)
			}
		})
	}
}

func TestAnomalyMonitor_StatusFailed(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 1, // Trigger on first failure
		ErrorRateThreshold:          0.99,
	})

	// Test with StatusFailed (different from StatusRed)
	result := models.TaskResult{
		Task:   models.Task{Number: "1"},
		Status: models.StatusFailed,
	}

	anomalies := monitor.RecordResult(result)
	if len(anomalies) == 0 {
		t.Error("Expected anomaly for StatusFailed")
	}
}

func TestAnomalyMonitor_ErrorField(t *testing.T) {
	monitor := NewAnomalyMonitorWithConfig("Wave 1", AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 1,
		ErrorRateThreshold:          0.99,
	})

	// Test with Error field set (even if status is not explicitly failed)
	result := models.TaskResult{
		Task:  models.Task{Number: "1"},
		Error: &anomalyTestError{msg: "test error"},
	}

	anomalies := monitor.RecordResult(result)
	if len(anomalies) == 0 {
		t.Error("Expected anomaly when Error field is set")
	}
}

// anomalyTestError is a simple error type for testing anomaly monitor
type anomalyTestError struct {
	msg string
}

func (e *anomalyTestError) Error() string {
	return e.msg
}
