package budget

import (
	"context"
	"time"
)

// visualUpdateInterval is the fixed interval for live counter updates (1 second)
const visualUpdateInterval = 1 * time.Second

// WaiterLogger interface for countdown announcements
type WaiterLogger interface {
	LogRateLimitCountdown(remaining, total time.Duration) // Called every 1s for live visual update
	LogRateLimitAnnounce(remaining, total time.Duration)  // Called at announce_interval for TTS
}

// RateLimitWaiter handles intelligent waiting for rate limit resets
type RateLimitWaiter struct {
	maxWait      time.Duration // Max wait before save-and-exit (default: 6h)
	announceInt  time.Duration // TTS announcement interval (default: 15m)
	safetyBuffer time.Duration // Extra wait after reset (default: 60s)
	logger       WaiterLogger  // For countdown announcements (can be nil)
}

// NewRateLimitWaiter creates a waiter with the given configuration
func NewRateLimitWaiter(maxWait, announceInterval, safetyBuffer time.Duration, logger WaiterLogger) *RateLimitWaiter {
	return &RateLimitWaiter{
		maxWait:      maxWait,
		announceInt:  announceInterval,
		safetyBuffer: safetyBuffer,
		logger:       logger,
	}
}

// ShouldWait returns true if we should wait for reset, false if wait is too long
// If info is nil, returns false
func (w *RateLimitWaiter) ShouldWait(info *RateLimitInfo) bool {
	if info == nil {
		return false
	}
	waitTime := info.TimeUntilReset()
	return waitTime <= w.maxWait
}

// WaitForReset blocks until rate limit reset with periodic countdown announcements
// Returns nil on successful wait, context error if cancelled
// Adds safety buffer after reset time
func (w *RateLimitWaiter) WaitForReset(ctx context.Context, info *RateLimitInfo) error {
	if info == nil {
		return nil
	}

	// Handle case where reset time has already passed
	if info.IsExpired() {
		// Just wait the safety buffer
		select {
		case <-time.After(w.safetyBuffer):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Calculate total wait time including safety buffer
	totalWait := w.TimeUntilResume(info)
	startTime := time.Now()
	endTime := startTime.Add(totalWait)

	// Create ticker for periodic announcements
	ticker := time.NewTicker(w.announceInt)
	defer ticker.Stop()

	// Initial announcement
	if w.logger != nil {
		w.logger.LogRateLimitCountdown(totalWait, totalWait)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case now := <-ticker.C:
			remaining := endTime.Sub(now)
			if remaining <= 0 {
				// Wait complete
				return nil
			}

			// Announce countdown
			if w.logger != nil {
				w.logger.LogRateLimitCountdown(remaining, totalWait)
			}

		case <-time.After(time.Until(endTime)):
			// Final wait complete
			return nil
		}
	}
}

// TimeUntilResume returns the total time to wait including safety buffer
func (w *RateLimitWaiter) TimeUntilResume(info *RateLimitInfo) time.Duration {
	if info == nil {
		return 0
	}

	// If already expired, just return safety buffer
	if info.IsExpired() {
		return w.safetyBuffer
	}

	// Otherwise, add safety buffer to reset time
	return info.TimeUntilReset() + w.safetyBuffer
}
