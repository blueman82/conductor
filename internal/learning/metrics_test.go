package learning

import (
	"sync"
	"testing"
	"time"
)

func TestNewPatternMetrics(t *testing.T) {
	pm := NewPatternMetrics()

	if pm == nil {
		t.Fatal("NewPatternMetrics() returned nil")
	}

	if pm.patterns == nil {
		t.Error("patterns map not initialized")
	}

	if pm.totalExecutions != 0 {
		t.Errorf("expected totalExecutions = 0, got %d", pm.totalExecutions)
	}

	if pm.totalPatternsFound != 0 {
		t.Errorf("expected totalPatternsFound = 0, got %d", pm.totalPatternsFound)
	}
}

func TestRecordPatternDetection(t *testing.T) {
	pm := NewPatternMetrics()

	// Record first detection
	pm.RecordPatternDetection("compilation_error", []string{"build fail", "syntax error"})

	stats := pm.GetPatternStats("compilation_error")
	if stats == nil {
		t.Fatal("expected stats for compilation_error, got nil")
	}

	if stats.DetectionCount != 1 {
		t.Errorf("expected DetectionCount = 1, got %d", stats.DetectionCount)
	}

	if len(stats.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(stats.Keywords))
	}

	// Record second detection with overlapping keywords
	pm.RecordPatternDetection("compilation_error", []string{"syntax error", "parse error"})

	stats = pm.GetPatternStats("compilation_error")
	if stats.DetectionCount != 2 {
		t.Errorf("expected DetectionCount = 2, got %d", stats.DetectionCount)
	}

	// Should have 3 unique keywords: build fail, syntax error, parse error
	if len(stats.Keywords) != 3 {
		t.Errorf("expected 3 unique keywords, got %d: %v", len(stats.Keywords), stats.Keywords)
	}

	// Verify totalPatternsFound incremented
	if pm.totalPatternsFound != 2 {
		t.Errorf("expected totalPatternsFound = 2, got %d", pm.totalPatternsFound)
	}
}

func TestMetrics_RecordExecution(t *testing.T) {
	pm := NewPatternMetrics()

	pm.RecordExecution()
	if pm.totalExecutions != 1 {
		t.Errorf("expected totalExecutions = 1, got %d", pm.totalExecutions)
	}

	pm.RecordExecution()
	pm.RecordExecution()
	if pm.totalExecutions != 3 {
		t.Errorf("expected totalExecutions = 3, got %d", pm.totalExecutions)
	}
}

func TestGetDetectionRate(t *testing.T) {
	tests := []struct {
		name           string
		executions     int
		detections     int
		expectedRate   float64
		setupFunc      func(*PatternMetrics)
	}{
		{
			name:         "no executions",
			executions:   0,
			detections:   0,
			expectedRate: 0.0,
			setupFunc:    func(pm *PatternMetrics) {},
		},
		{
			name:         "no patterns detected",
			executions:   5,
			detections:   0,
			expectedRate: 0.0,
			setupFunc: func(pm *PatternMetrics) {
				for i := 0; i < 5; i++ {
					pm.RecordExecution()
				}
			},
		},
		{
			name:         "50% detection rate",
			executions:   4,
			detections:   2,
			expectedRate: 0.5,
			setupFunc: func(pm *PatternMetrics) {
				pm.RecordExecution()
				pm.RecordPatternDetection("test_failure", []string{"test fail"})
				pm.RecordExecution()
				pm.RecordExecution()
				pm.RecordPatternDetection("timeout", []string{"deadline"})
				pm.RecordExecution()
			},
		},
		{
			name:         "100% detection rate",
			executions:   3,
			detections:   3,
			expectedRate: 1.0,
			setupFunc: func(pm *PatternMetrics) {
				pm.RecordExecution()
				pm.RecordPatternDetection("compilation_error", []string{"build fail"})
				pm.RecordExecution()
				pm.RecordPatternDetection("test_failure", []string{"test fail"})
				pm.RecordExecution()
				pm.RecordPatternDetection("timeout", []string{"deadline"})
			},
		},
		{
			name:         "multiple patterns per execution",
			executions:   2,
			detections:   4,
			expectedRate: 2.0,
			setupFunc: func(pm *PatternMetrics) {
				pm.RecordExecution()
				pm.RecordPatternDetection("compilation_error", []string{"build fail"})
				pm.RecordPatternDetection("test_failure", []string{"test fail"})
				pm.RecordExecution()
				pm.RecordPatternDetection("timeout", []string{"deadline"})
				pm.RecordPatternDetection("runtime_error", []string{"panic"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPatternMetrics()
			tt.setupFunc(pm)

			rate := pm.GetDetectionRate()
			if rate != tt.expectedRate {
				t.Errorf("expected rate = %.2f, got %.2f", tt.expectedRate, rate)
			}
		})
	}
}

func TestGetPatternStats(t *testing.T) {
	pm := NewPatternMetrics()

	// Test non-existent pattern
	stats := pm.GetPatternStats("non_existent")
	if stats != nil {
		t.Error("expected nil for non-existent pattern")
	}

	// Record pattern and verify stats
	pm.RecordPatternDetection("compilation_error", []string{"build fail"})
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	pm.RecordPatternDetection("compilation_error", []string{"syntax error"})

	stats = pm.GetPatternStats("compilation_error")
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	if stats.PatternType != "compilation_error" {
		t.Errorf("expected PatternType = compilation_error, got %s", stats.PatternType)
	}

	if stats.DetectionCount != 2 {
		t.Errorf("expected DetectionCount = 2, got %d", stats.DetectionCount)
	}

	if len(stats.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(stats.Keywords))
	}

	if stats.LastDetected.IsZero() {
		t.Error("expected LastDetected to be set")
	}

	// Verify returned copy is independent
	stats.DetectionCount = 999
	statsAgain := pm.GetPatternStats("compilation_error")
	if statsAgain.DetectionCount == 999 {
		t.Error("GetPatternStats should return a copy, not a reference")
	}
}

func TestGetAllPatterns(t *testing.T) {
	pm := NewPatternMetrics()

	// Empty patterns
	all := pm.GetAllPatterns()
	if len(all) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(all))
	}

	// Add multiple patterns
	pm.RecordPatternDetection("compilation_error", []string{"build fail"})
	pm.RecordPatternDetection("test_failure", []string{"test fail"})
	pm.RecordPatternDetection("timeout", []string{"deadline"})

	all = pm.GetAllPatterns()
	if len(all) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(all))
	}

	// Verify all expected patterns present
	expectedPatterns := []string{"compilation_error", "test_failure", "timeout"}
	for _, expected := range expectedPatterns {
		if _, exists := all[expected]; !exists {
			t.Errorf("expected pattern %s not found", expected)
		}
	}

	// Verify returned copies are independent
	all["compilation_error"].DetectionCount = 999
	statsAgain := pm.GetPatternStats("compilation_error")
	if statsAgain.DetectionCount == 999 {
		t.Error("GetAllPatterns should return copies, not references")
	}
}

func TestResetMetrics(t *testing.T) {
	pm := NewPatternMetrics()

	// Add some data
	pm.RecordExecution()
	pm.RecordExecution()
	pm.RecordPatternDetection("compilation_error", []string{"build fail"})
	pm.RecordPatternDetection("test_failure", []string{"test fail"})

	// Verify data exists
	if pm.totalExecutions != 2 {
		t.Errorf("setup: expected totalExecutions = 2, got %d", pm.totalExecutions)
	}
	if pm.totalPatternsFound != 2 {
		t.Errorf("setup: expected totalPatternsFound = 2, got %d", pm.totalPatternsFound)
	}
	if len(pm.patterns) != 2 {
		t.Errorf("setup: expected 2 patterns, got %d", len(pm.patterns))
	}

	// Reset
	pm.ResetMetrics()

	// Verify reset
	if pm.totalExecutions != 0 {
		t.Errorf("expected totalExecutions = 0 after reset, got %d", pm.totalExecutions)
	}
	if pm.totalPatternsFound != 0 {
		t.Errorf("expected totalPatternsFound = 0 after reset, got %d", pm.totalPatternsFound)
	}
	if len(pm.patterns) != 0 {
		t.Errorf("expected 0 patterns after reset, got %d", len(pm.patterns))
	}

	// Verify GetPatternStats returns nil for previously existing patterns
	stats := pm.GetPatternStats("compilation_error")
	if stats != nil {
		t.Error("expected nil for pattern after reset")
	}
}

func TestThreadSafety(t *testing.T) {
	pm := NewPatternMetrics()

	// Number of goroutines and operations
	numGoroutines := 10
	opsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent RecordExecution
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				pm.RecordExecution()
			}
		}()
	}

	// Concurrent RecordPatternDetection
	patterns := []string{"compilation_error", "test_failure", "timeout"}
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				pattern := patterns[j%len(patterns)]
				pm.RecordPatternDetection(pattern, []string{"keyword"})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = pm.GetDetectionRate()
				_ = pm.GetAllPatterns()
				_ = pm.GetPatternStats("compilation_error")
			}
		}()
	}

	wg.Wait()

	// Verify final counts
	expectedExecutions := int64(numGoroutines * opsPerGoroutine)
	if pm.totalExecutions != expectedExecutions {
		t.Errorf("expected totalExecutions = %d, got %d", expectedExecutions, pm.totalExecutions)
	}

	// Pattern detections: numGoroutines * opsPerGoroutine
	expectedDetections := int64(numGoroutines * opsPerGoroutine)
	if pm.totalPatternsFound != expectedDetections {
		t.Errorf("expected totalPatternsFound = %d, got %d", expectedDetections, pm.totalPatternsFound)
	}

	// Verify all 3 patterns exist
	allPatterns := pm.GetAllPatterns()
	if len(allPatterns) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(allPatterns))
	}
}

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name     string
		initial  []string
		items    []string
		expected []string
	}{
		{
			name:     "empty slice",
			initial:  []string{},
			items:    []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "no duplicates",
			initial:  []string{"a", "b"},
			items:    []string{"c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "all duplicates",
			initial:  []string{"a", "b"},
			items:    []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "partial duplicates",
			initial:  []string{"a", "b"},
			items:    []string{"b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "duplicate in items",
			initial:  []string{"a"},
			items:    []string{"b", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendUnique(tt.initial, tt.items...)

			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
			}

			// Check all expected items present
			for _, exp := range tt.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected item %s not found in result", exp)
				}
			}
		})
	}
}

func TestPatternStatsLastDetected(t *testing.T) {
	pm := NewPatternMetrics()

	// Record first detection
	before := time.Now()
	pm.RecordPatternDetection("compilation_error", []string{"build fail"})
	after := time.Now()

	stats := pm.GetPatternStats("compilation_error")
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	// Verify LastDetected is within expected range
	if stats.LastDetected.Before(before) || stats.LastDetected.After(after) {
		t.Errorf("LastDetected %v not within range [%v, %v]", stats.LastDetected, before, after)
	}

	// Record second detection
	time.Sleep(10 * time.Millisecond)
	before2 := time.Now()
	pm.RecordPatternDetection("compilation_error", []string{"syntax error"})
	after2 := time.Now()

	stats2 := pm.GetPatternStats("compilation_error")
	if stats2.LastDetected.Before(before2) || stats2.LastDetected.After(after2) {
		t.Errorf("LastDetected %v not within range [%v, %v]", stats2.LastDetected, before2, after2)
	}

	// Second detection should be later than first
	if !stats2.LastDetected.After(stats.LastDetected) {
		t.Error("second LastDetected should be after first")
	}
}
