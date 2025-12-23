package budget

import (
	"context"
	"testing"
	"time"
)

// mockWaiterLogger captures countdown calls for testing
type mockWaiterLogger struct {
	calls []struct {
		remaining time.Duration
		total     time.Duration
	}
}

func (m *mockWaiterLogger) LogRateLimitCountdown(remaining, total time.Duration) {
	m.calls = append(m.calls, struct {
		remaining time.Duration
		total     time.Duration
	}{remaining, total})
}

func TestNewRateLimitWaiter(t *testing.T) {
	logger := &mockWaiterLogger{}
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, logger)

	if waiter == nil {
		t.Fatal("expected non-nil waiter")
	}
	if waiter.maxWait != 6*time.Hour {
		t.Errorf("expected maxWait 6h, got %v", waiter.maxWait)
	}
	if waiter.announceInt != 15*time.Minute {
		t.Errorf("expected announceInt 15m, got %v", waiter.announceInt)
	}
	if waiter.safetyBuffer != 60*time.Second {
		t.Errorf("expected safetyBuffer 60s, got %v", waiter.safetyBuffer)
	}
}

func TestRateLimitWaiter_ShouldWait_Nil(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	if waiter.ShouldWait(nil) {
		t.Error("ShouldWait(nil) should return false")
	}
}

func TestRateLimitWaiter_ShouldWait_WithinLimit(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(30 * time.Minute),
	}

	if !waiter.ShouldWait(info) {
		t.Error("ShouldWait should return true for 30m wait when max is 6h")
	}
}

func TestRateLimitWaiter_ShouldWait_ExceedsLimit(t *testing.T) {
	waiter := NewRateLimitWaiter(1*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(2 * time.Hour),
	}

	if waiter.ShouldWait(info) {
		t.Error("ShouldWait should return false for 2h wait when max is 1h")
	}
}

func TestRateLimitWaiter_ShouldWait_ExactLimit(t *testing.T) {
	waiter := NewRateLimitWaiter(1*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(1 * time.Hour),
	}

	if !waiter.ShouldWait(info) {
		t.Error("ShouldWait should return true for exactly max wait duration")
	}
}

func TestRateLimitWaiter_ShouldWait_Expired(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(-1 * time.Hour), // Already expired
	}

	// TimeUntilReset returns negative, which is <= maxWait
	// This is expected - we still "wait" (just the safety buffer)
	if !waiter.ShouldWait(info) {
		t.Error("ShouldWait should return true for expired reset (negative duration)")
	}
}

func TestRateLimitWaiter_TimeUntilResume_Nil(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	duration := waiter.TimeUntilResume(nil)

	if duration != 0 {
		t.Errorf("expected 0 for nil info, got %v", duration)
	}
}

func TestRateLimitWaiter_TimeUntilResume_Future(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(30 * time.Minute),
	}

	duration := waiter.TimeUntilResume(info)

	// Should be ~30 minutes + 60 seconds safety buffer
	expected := 30*time.Minute + 60*time.Second
	if duration < expected-2*time.Second || duration > expected+2*time.Second {
		t.Errorf("expected ~%v, got %v", expected, duration)
	}
}

func TestRateLimitWaiter_TimeUntilResume_Expired(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(-1 * time.Hour), // Already expired
	}

	duration := waiter.TimeUntilResume(info)

	// Should just return safety buffer
	if duration != 60*time.Second {
		t.Errorf("expected safety buffer 60s for expired, got %v", duration)
	}
}

func TestRateLimitWaiter_WaitForReset_Nil(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	err := waiter.WaitForReset(context.Background(), nil)

	if err != nil {
		t.Errorf("expected nil error for nil info, got %v", err)
	}
}

func TestRateLimitWaiter_WaitForReset_ContextCancelled(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(1 * time.Hour),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := waiter.WaitForReset(ctx, info)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimitWaiter_WaitForReset_ShortWait(t *testing.T) {
	logger := &mockWaiterLogger{}
	// Use short intervals for testing
	waiter := NewRateLimitWaiter(1*time.Hour, 50*time.Millisecond, 10*time.Millisecond, logger)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(100 * time.Millisecond),
	}

	start := time.Now()
	err := waiter.WaitForReset(context.Background(), info)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Should have waited approximately 100ms + 10ms safety buffer
	if elapsed < 100*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("expected ~110ms wait, got %v", elapsed)
	}

	// Should have logged countdown at least once (initial)
	if len(logger.calls) < 1 {
		t.Error("expected at least 1 countdown log")
	}
}

func TestRateLimitWaiter_WaitForReset_Expired(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 50*time.Millisecond, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(-1 * time.Hour), // Already expired
	}

	start := time.Now()
	err := waiter.WaitForReset(context.Background(), info)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Should have waited just the safety buffer
	if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("expected ~50ms safety buffer wait, got %v", elapsed)
	}
}

func TestRateLimitWaiter_WaitForReset_NilLogger(t *testing.T) {
	// Should work fine without a logger
	waiter := NewRateLimitWaiter(1*time.Hour, 50*time.Millisecond, 10*time.Millisecond, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(50 * time.Millisecond),
	}

	err := waiter.WaitForReset(context.Background(), info)

	if err != nil {
		t.Errorf("expected nil error with nil logger, got %v", err)
	}
}

func TestRateLimitWaiter_WaitForReset_MultipleCountdowns(t *testing.T) {
	logger := &mockWaiterLogger{}
	// Use very short intervals to ensure multiple countdown announcements
	waiter := NewRateLimitWaiter(1*time.Hour, 30*time.Millisecond, 10*time.Millisecond, logger)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(100 * time.Millisecond),
	}

	err := waiter.WaitForReset(context.Background(), info)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Should have logged multiple countdowns (initial + at least 2 ticker events)
	if len(logger.calls) < 2 {
		t.Errorf("expected at least 2 countdown logs, got %d", len(logger.calls))
	}

	// Verify countdown is decreasing
	if len(logger.calls) >= 2 {
		if logger.calls[1].remaining >= logger.calls[0].remaining {
			t.Error("expected remaining time to decrease between countdown announcements")
		}
	}

	// Verify total wait time is consistent across all calls
	if len(logger.calls) >= 2 {
		if logger.calls[0].total != logger.calls[1].total {
			t.Error("expected total wait time to remain constant")
		}
	}
}

func TestRateLimitWaiter_WaitForReset_ZeroResetAt(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 50*time.Millisecond, nil)

	info := &RateLimitInfo{
		ResetAt: time.Time{}, // Zero time
	}

	start := time.Now()
	err := waiter.WaitForReset(context.Background(), info)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Zero ResetAt means IsExpired() returns true, so just safety buffer
	if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("expected ~50ms safety buffer wait, got %v", elapsed)
	}
}

func TestRateLimitWaiter_TimeUntilResume_ZeroResetAt(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Time{}, // Zero time
	}

	duration := waiter.TimeUntilResume(info)

	// Zero ResetAt means IsExpired() returns true, so just safety buffer
	if duration != 60*time.Second {
		t.Errorf("expected safety buffer 60s for zero ResetAt, got %v", duration)
	}
}

func TestRateLimitWaiter_WaitForReset_ContextCancelledDuringWait(t *testing.T) {
	waiter := NewRateLimitWaiter(1*time.Hour, 50*time.Millisecond, 10*time.Millisecond, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(500 * time.Millisecond),
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := waiter.WaitForReset(ctx, info)
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Should have returned after ~100ms, not the full 500ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("expected cancellation after ~100ms, but waited %v", elapsed)
	}
}

func TestRateLimitWaiter_ShouldWait_NegativeDuration(t *testing.T) {
	waiter := NewRateLimitWaiter(1*time.Hour, 15*time.Minute, 60*time.Second, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(-30 * time.Minute), // 30 minutes in the past
	}

	// Negative duration should be <= maxWait, so should wait
	if !waiter.ShouldWait(info) {
		t.Error("ShouldWait should return true for negative duration (already expired)")
	}
}

func TestRateLimitWaiter_WaitForReset_ExpiredContextCancelled(t *testing.T) {
	waiter := NewRateLimitWaiter(6*time.Hour, 15*time.Minute, 100*time.Millisecond, nil)

	info := &RateLimitInfo{
		ResetAt: time.Now().Add(-1 * time.Hour), // Already expired
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := waiter.WaitForReset(ctx, info)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled even for expired, got %v", err)
	}
}
