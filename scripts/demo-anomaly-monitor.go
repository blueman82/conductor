//go:build ignore
// +build ignore

// Demo script to show GUARD v2.18 Anomaly Monitor in action
// Run with: go run scripts/demo-anomaly-monitor.go
package main

import (
	"fmt"
	"time"

	"github.com/harrison/conductor/internal/executor"
	"github.com/harrison/conductor/internal/models"
)

func main() {
	fmt.Println("=" + repeatStr("=", 60))
	fmt.Println("GUARD v2.18 Anomaly Monitor Demo")
	fmt.Println("=" + repeatStr("=", 60))
	fmt.Println()

	// Create anomaly monitor with thresholds
	config := executor.AnomalyMonitorConfig{
		ConsecutiveFailureThreshold: 3,
		ErrorRateThreshold:          0.5,
		DurationDeviationThreshold:  2.0,
	}
	monitor := executor.NewAnomalyMonitorWithConfig("Demo Wave", config)

	fmt.Println("Configuration:")
	fmt.Printf("  Consecutive Failure Threshold: %d\n", config.ConsecutiveFailureThreshold)
	fmt.Printf("  Error Rate Threshold: %.0f%%\n", config.ErrorRateThreshold*100)
	fmt.Printf("  Duration Deviation Threshold: %.1fx\n", config.DurationDeviationThreshold)
	fmt.Println()

	// Demo 1: Consecutive Failures
	fmt.Println("-" + repeatStr("-", 60))
	fmt.Println("Demo 1: Consecutive Failure Detection")
	fmt.Println("-" + repeatStr("-", 60))

	for i := 1; i <= 4; i++ {
		result := models.TaskResult{
			Task:   models.Task{Number: fmt.Sprintf("%d", i)},
			Status: models.StatusRed,
		}
		anomalies := monitor.RecordResult(result)

		fmt.Printf("Task %d: RED → ", i)
		if len(anomalies) > 0 {
			for _, a := range anomalies {
				fmt.Printf("🚨 ANOMALY: %s (%s severity)\n", a.Description, a.Severity)
			}
		} else {
			fmt.Println("No anomaly yet")
		}
	}
	fmt.Println()

	// Reset for next demo
	monitor.Reset("Duration Demo Wave")

	// Demo 2: Duration Outlier
	fmt.Println("-" + repeatStr("-", 60))
	fmt.Println("Demo 2: Duration Outlier Detection")
	fmt.Println("-" + repeatStr("-", 60))

	outlierResult := models.TaskResult{
		Task: models.Task{
			Number:        "1",
			EstimatedTime: 1 * time.Minute,
		},
		Status:   models.StatusGreen,
		Duration: 5 * time.Minute, // 5x the estimate!
	}

	anomalies := monitor.RecordResult(outlierResult)
	fmt.Printf("Task 1: Estimated 1m, Actual 5m (5x) → ")
	if len(anomalies) > 0 {
		for _, a := range anomalies {
			fmt.Printf("🚨 ANOMALY: %s (%s severity)\n", a.Description, a.Severity)
		}
	}
	fmt.Println()

	// Reset for next demo
	monitor.Reset("Error Rate Demo Wave")

	// Demo 3: High Error Rate
	fmt.Println("-" + repeatStr("-", 60))
	fmt.Println("Demo 3: High Error Rate Detection")
	fmt.Println("-" + repeatStr("-", 60))

	// 4 failures out of 5 = 80% error rate
	statuses := []string{
		models.StatusRed,
		models.StatusGreen,
		models.StatusRed,
		models.StatusRed,
		models.StatusRed,
	}

	for i, status := range statuses {
		result := models.TaskResult{
			Task:   models.Task{Number: fmt.Sprintf("%d", i+1)},
			Status: status,
		}
		anomalies := monitor.RecordResult(result)

		statusStr := "GREEN"
		if status == models.StatusRed {
			statusStr = "RED"
		}
		fmt.Printf("Task %d: %s → ", i+1, statusStr)
		if len(anomalies) > 0 {
			for _, a := range anomalies {
				if a.Type == "high_error_rate" {
					fmt.Printf("🚨 ANOMALY: %s\n", a.Description)
				}
			}
		} else {
			fmt.Println("No error rate anomaly")
		}
	}

	// Final health check
	fmt.Println()
	healthy, finalAnomalies := monitor.CheckWaveHealth()
	fmt.Printf("Wave Health: %v (anomalies: %d)\n", healthy, len(finalAnomalies))
	fmt.Println()
	fmt.Println("=" + repeatStr("=", 60))
	fmt.Println("Demo Complete!")
	fmt.Println("=" + repeatStr("=", 60))
}

func repeatStr(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
