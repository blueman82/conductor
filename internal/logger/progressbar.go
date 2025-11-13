package logger

import (
	"fmt"
	"sync"
)

// ProgressBar represents an ASCII progress bar with color support
type ProgressBar struct {
	current     int
	total       int
	width       int
	enableColor bool
	prefix      string
	mu          sync.RWMutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total, width int, enableColor bool) *ProgressBar {
	if width < 1 {
		width = 10
	}
	return &ProgressBar{
		current:     0,
		total:       total,
		width:       width,
		enableColor: enableColor,
		prefix:      "",
	}
}

// Update sets the current progress value
func (pb *ProgressBar) Update(current int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.current = current
}

// Increment increments the current progress by 1
func (pb *ProgressBar) Increment() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.current++
}

// Current returns the current progress value
func (pb *ProgressBar) Current() int {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.current
}

// Total returns the total progress value
func (pb *ProgressBar) Total() int {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.total
}

// Percentage returns the progress percentage (0-100)
func (pb *ProgressBar) Percentage() int {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if pb.total == 0 {
		return 0
	}

	perc := (pb.current * 100) / pb.total
	if perc > 100 {
		perc = 100
	}
	if perc < 0 {
		perc = 0
	}
	return perc
}

// SetPrefix sets a custom prefix for the progress bar
func (pb *ProgressBar) SetPrefix(prefix string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.prefix = prefix
}

// Render generates the ASCII progress bar string
func (pb *ProgressBar) Render() string {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	// Calculate percentage and filled characters
	var perc int
	if pb.total == 0 {
		perc = 0
	} else {
		perc = (pb.current * 100) / pb.total
		if perc > 100 {
			perc = 100
		}
		if perc < 0 {
			perc = 0
		}
	}

	// Calculate number of filled characters
	filled := (perc * pb.width) / 100
	if filled > pb.width {
		filled = pb.width
	}
	if filled < 0 {
		filled = 0
	}

	// Build the bar string
	bar := "["
	for i := 0; i < pb.width; i++ {
		if i < filled {
			bar += "="
		} else {
			bar += " "
		}
	}
	bar += "]"

	// Add counter and percentage
	result := fmt.Sprintf("%s%s %d/%d (%d%%)", pb.prefix, bar, pb.current, pb.total, perc)

	// Apply color if enabled
	if pb.enableColor && perc < 100 {
		result = fmt.Sprintf("\033[36m%s\033[0m", result) // Cyan for in-progress
	} else if pb.enableColor && perc == 100 {
		result = fmt.Sprintf("\033[32m%s\033[0m", result) // Green for complete
	}

	return result
}
