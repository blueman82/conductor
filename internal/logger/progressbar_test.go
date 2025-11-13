package logger

import (
	"strings"
	"sync"
	"testing"
)

// TestProgressBarRender verifies correct ASCII bar rendering
func TestProgressBarRender(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		expected string
	}{
		{
			name:     "empty progress",
			current:  0,
			total:    10,
			width:    10,
			expected: "[          ] 0/10 (0%)",
		},
		{
			name:     "half progress",
			current:  5,
			total:    10,
			width:    10,
			expected: "[=====     ] 5/10 (50%)",
		},
		{
			name:     "full progress",
			current:  10,
			total:    10,
			width:    10,
			expected: "[==========] 10/10 (100%)",
		},
		{
			name:     "quarter progress",
			current:  2,
			total:    8,
			width:    8,
			expected: "[==      ] 2/8 (25%)",
		},
		{
			name:     "small width",
			current:  1,
			total:    4,
			width:    4,
			expected: "[=   ] 1/4 (25%)",
		},
		{
			name:     "large width",
			current:  30,
			total:    100,
			width:    20,
			expected: "[======              ] 30/100 (30%)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, tt.width, false)
			pb.Update(tt.current)
			result := pb.Render()

			if result != tt.expected {
				t.Errorf("Render() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestProgressBarWidth tests different bar widths
func TestProgressBarWidth(t *testing.T) {
	tests := []struct {
		name  string
		width int
		total int
	}{
		{"width 5", 5, 10},
		{"width 10", 10, 10},
		{"width 20", 20, 10},
		{"width 1", 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, tt.width, false)
			pb.Update(tt.total / 2)
			result := pb.Render()

			// Verify bar is present and contains correct width
			if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
				t.Errorf("Render() missing brackets: %q", result)
			}

			// Count characters between brackets
			start := strings.Index(result, "[")
			end := strings.Index(result, "]")
			if start >= 0 && end > start {
				barContent := result[start+1 : end]
				if len(barContent) != tt.width {
					t.Errorf("Bar width = %d, want %d. Content: %q", len(barContent), tt.width, barContent)
				}
			}
		})
	}
}

// TestProgressBarColors tests color rendering
func TestProgressBarColors(t *testing.T) {
	tests := []struct {
		name        string
		enableColor bool
		shouldHave  string
	}{
		{
			name:        "with color",
			enableColor: true,
			shouldHave:  "\033[", // ANSI escape code prefix
		},
		{
			name:        "without color",
			enableColor: false,
			shouldHave:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(10, 10, tt.enableColor)
			pb.Update(5)
			result := pb.Render()

			if tt.enableColor {
				if !strings.Contains(result, "\033[") {
					t.Errorf("Render() with color should contain ANSI codes, got: %q", result)
				}
			} else {
				// Plain output should not contain ANSI codes
				if strings.Contains(result, "\033[") {
					t.Errorf("Render() without color should not contain ANSI codes, got: %q", result)
				}
			}
		})
	}
}

// TestProgressBarUpdate tests progress updates
func TestProgressBarUpdate(t *testing.T) {
	tests := []struct {
		name          string
		initialTotal  int
		updateValue   int
		expectedCurr  int
		expectedTotal int
	}{
		{
			name:          "update to half",
			initialTotal:  10,
			updateValue:   5,
			expectedCurr:  5,
			expectedTotal: 10,
		},
		{
			name:          "update to full",
			initialTotal:  10,
			updateValue:   10,
			expectedCurr:  10,
			expectedTotal: 10,
		},
		{
			name:          "update to zero",
			initialTotal:  10,
			updateValue:   0,
			expectedCurr:  0,
			expectedTotal: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.initialTotal, 10, false)
			pb.Update(tt.updateValue)

			if pb.Current() != tt.expectedCurr {
				t.Errorf("Current() = %d, want %d", pb.Current(), tt.expectedCurr)
			}

			if pb.Total() != tt.expectedTotal {
				t.Errorf("Total() = %d, want %d", pb.Total(), tt.expectedTotal)
			}
		})
	}
}

// TestProgressBarIncrement tests Increment method
func TestProgressBarIncrement(t *testing.T) {
	pb := NewProgressBar(10, 10, false)

	if pb.Current() != 0 {
		t.Errorf("Initial Current() = %d, want 0", pb.Current())
	}

	pb.Increment()
	if pb.Current() != 1 {
		t.Errorf("After Increment(), Current() = %d, want 1", pb.Current())
	}

	pb.Increment()
	pb.Increment()
	if pb.Current() != 3 {
		t.Errorf("After 3 Increments, Current() = %d, want 3", pb.Current())
	}
}

// TestProgressBarEdgeCases tests boundary conditions
func TestProgressBarEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		expect  string
	}{
		{
			name:    "zero total",
			current: 0,
			total:   0,
			expect:  "[", // Should handle gracefully
		},
		{
			name:    "current > total",
			current: 15,
			total:   10,
			expect:  "[==========]", // Should cap at 100%
		},
		{
			name:    "negative current",
			current: -5,
			total:   10,
			expect:  "[", // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, 10, false)
			pb.Update(tt.current)
			result := pb.Render()

			if !strings.Contains(result, tt.expect) {
				t.Errorf("Render() = %q, should contain %q", result, tt.expect)
			}
		})
	}
}

// TestProgressBarPercentage tests percentage calculation
func TestProgressBarPercentage(t *testing.T) {
	tests := []struct {
		name         string
		current      int
		total        int
		expectedPerc int
	}{
		{"0%", 0, 10, 0},
		{"25%", 2, 8, 25},
		{"50%", 5, 10, 50},
		{"100%", 10, 10, 100},
		{">100%", 15, 10, 100},          // Should cap at 100
		{"1/3", 1, 3, 33},               // Should floor
		{"zero total", 0, 0, 0},         // Should return 0 for zero total
		{"negative current", -5, 10, 0}, // Should floor to 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, 10, false)
			pb.Update(tt.current)
			perc := pb.Percentage()

			if perc != tt.expectedPerc {
				t.Errorf("Percentage() = %d, want %d", perc, tt.expectedPerc)
			}
		})
	}
}

// TestProgressBarFormat tests custom format strings
func TestProgressBarFormat(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		shouldHave string
	}{
		{
			name:       "with prefix",
			prefix:     "Task 1: ",
			shouldHave: "Task 1: ",
		},
		{
			name:       "empty prefix",
			prefix:     "",
			shouldHave: "[",
		},
		{
			name:       "long prefix",
			prefix:     "Processing long task name: ",
			shouldHave: "Processing long task name: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(10, 10, false)
			pb.SetPrefix(tt.prefix)
			pb.Update(5)
			result := pb.Render()

			if !strings.Contains(result, tt.shouldHave) {
				t.Errorf("Render() = %q, should contain %q", result, tt.shouldHave)
			}
		})
	}
}

// TestProgressBarNewProgressBarEdgeCases tests NewProgressBar with edge case widths
func TestProgressBarNewProgressBarEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		expectedOK bool
	}{
		{"positive width", 10, true},
		{"width 1", 1, true},
		{"zero width (should default to 10)", 0, true},
		{"negative width (should default to 10)", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(10, tt.width, false)
			if pb == nil {
				t.Errorf("NewProgressBar() = nil")
				return
			}
			// Check that width is at least 1
			if pb.width < 1 {
				t.Errorf("width = %d, want >= 1", pb.width)
			}
		})
	}
}

// TestProgressBarColorAndNoColor tests rendering with and without color
func TestProgressBarColorAndNoColor(t *testing.T) {
	tests := []struct {
		name        string
		current     int
		total       int
		enableColor bool
	}{
		{"color, incomplete", 5, 10, true},
		{"color, complete", 10, 10, true},
		{"no color, incomplete", 5, 10, false},
		{"no color, complete", 10, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, 10, tt.enableColor)
			pb.Update(tt.current)
			result := pb.Render()

			if len(result) == 0 {
				t.Errorf("Render() returned empty string")
			}
		})
	}
}

// TestProgressBarConcurrency tests thread-safe concurrent updates
func TestProgressBarConcurrency(t *testing.T) {
	pb := NewProgressBar(100, 10, false)
	var wg sync.WaitGroup
	numGoroutines := 10

	// Spawn multiple goroutines to increment concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				pb.Increment()
				// Also test Read operations
				_ = pb.Current()
				_ = pb.Percentage()
				_ = pb.Render()
			}
		}()
	}

	wg.Wait()

	// Should have processed all increments (10 goroutines * 10 increments = 100)
	if pb.Current() != 100 {
		t.Errorf("After concurrent updates, Current() = %d, want 100", pb.Current())
	}
}

// TestProgressBarRaceCondition tests for data races with -race flag
func TestProgressBarRaceCondition(t *testing.T) {
	pb := NewProgressBar(1000, 10, false)
	var wg sync.WaitGroup

	// Reader goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = pb.Current()
				_ = pb.Total()
				_ = pb.Percentage()
				_ = pb.Render()
			}
		}()
	}

	// Writer goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				pb.Update(j % 100)
				pb.Increment()
			}
		}()
	}

	wg.Wait()
}
