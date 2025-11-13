package learning

import (
	"sync"
	"time"
)

// PatternMetrics tracks pattern extraction statistics
type PatternMetrics struct {
	mu sync.RWMutex

	// Per-pattern statistics
	patterns map[string]*PatternStats

	// Overall statistics
	totalExecutions    int64
	totalPatternsFound int64
}

// PatternStats tracks metrics for a single pattern type
type PatternStats struct {
	PatternType    string
	DetectionCount int64   // Times this pattern was detected
	ExecutionCount int64   // Times this pattern appeared in output
	DetectionRate  float64 // DetectionCount / ExecutionCount
	LastDetected   time.Time
	Keywords       []string // Keywords that triggered detection
}

// NewPatternMetrics creates a new metrics collector
func NewPatternMetrics() *PatternMetrics {
	return &PatternMetrics{
		patterns: make(map[string]*PatternStats),
	}
}

// RecordPatternDetection records when a pattern was detected
func (pm *PatternMetrics) RecordPatternDetection(patternType string, keywords []string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.patterns[patternType]; !exists {
		pm.patterns[patternType] = &PatternStats{
			PatternType: patternType,
			Keywords:    []string{},
		}
	}

	stats := pm.patterns[patternType]
	stats.DetectionCount++
	stats.LastDetected = time.Now()
	stats.Keywords = appendUnique(stats.Keywords, keywords...)

	pm.totalPatternsFound++
}

// RecordExecution records when task execution analysis occurs
func (pm *PatternMetrics) RecordExecution() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.totalExecutions++
}

// GetDetectionRate returns the rate of pattern detection
func (pm *PatternMetrics) GetDetectionRate() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.totalExecutions == 0 {
		return 0.0
	}

	return float64(pm.totalPatternsFound) / float64(pm.totalExecutions)
}

// GetPatternStats returns statistics for a specific pattern
func (pm *PatternMetrics) GetPatternStats(patternType string) *PatternStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if stats, exists := pm.patterns[patternType]; exists {
		// Return a copy to avoid race conditions
		copy := *stats
		copy.Keywords = make([]string, len(stats.Keywords))
		copySlice := copy.Keywords
		for i, k := range stats.Keywords {
			copySlice[i] = k
		}
		return &copy
	}
	return nil
}

// GetAllPatterns returns statistics for all patterns
func (pm *PatternMetrics) GetAllPatterns() map[string]*PatternStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return a copy
	result := make(map[string]*PatternStats)
	for k, v := range pm.patterns {
		copy := *v
		copy.Keywords = make([]string, len(v.Keywords))
		copySlice := copy.Keywords
		for i, kw := range v.Keywords {
			copySlice[i] = kw
		}
		result[k] = &copy
	}
	return result
}

// appendUnique appends items to slice if not already present
func appendUnique(slice []string, items ...string) []string {
	for _, item := range items {
		found := false
		for _, v := range slice {
			if v == item {
				found = true
				break
			}
		}
		if !found {
			slice = append(slice, item)
		}
	}
	return slice
}

// ResetMetrics clears all metrics (for testing)
func (pm *PatternMetrics) ResetMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.patterns = make(map[string]*PatternStats)
	pm.totalExecutions = 0
	pm.totalPatternsFound = 0
}
