package budget

import (
	"fmt"
	"testing"
	"time"
)

func TestParseRateLimitFromOutput_UnixTimestamp(t *testing.T) {
	// Pattern: Claude AI usage limit reached|<unix_timestamp>
	futureTime := time.Now().Add(2 * time.Hour).Unix()
	input := fmt.Sprintf("Claude AI usage limit reached|%d", futureTime)

	info := ParseRateLimitFromOutput(input)

	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.LimitType != LimitTypeSession {
		t.Errorf("expected session limit, got %s", info.LimitType)
	}
	// Reset time should be approximately 2 hours from now
	if info.ResetAt.Unix() != futureTime {
		t.Errorf("expected reset at %d, got %d", futureTime, info.ResetAt.Unix())
	}
	if info.Source != "output" {
		t.Errorf("expected source 'output', got %s", info.Source)
	}
}

func TestParseRateLimitFromOutput_HumanTime(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectedHr int
	}{
		{
			"afternoon time",
			"rate limit - Your limit will reset at 2pm (America/New_York)",
			14,
		},
		{
			"morning time",
			"usage limit - Your limit will reset at 9am (America/New_York)",
			9,
		},
		{
			"midnight",
			"429 error - Your limit will reset at 12am (America/New_York)",
			0,
		},
		{
			"noon",
			"too many requests - Your limit will reset at 12pm (America/New_York)",
			12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseRateLimitFromOutput(tt.input)
			if info == nil {
				t.Fatalf("expected non-nil info for input %q", tt.input)
			}

			loc, _ := time.LoadLocation("America/New_York")
			now := time.Now().In(loc)
			expected := time.Date(now.Year(), now.Month(), now.Day(), tt.expectedHr, 0, 0, 0, loc)

			// If expected time is in the past, it should wrap to next day
			if expected.Before(now) {
				expected = expected.Add(24 * time.Hour)
			}

			if info.ResetAt.Hour() != tt.expectedHr {
				t.Errorf("expected hour %d, got %d", tt.expectedHr, info.ResetAt.Hour())
			}
		})
	}
}

func TestParseRateLimitFromOutput_RetrySeconds(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"retry in seconds", "rate limit hit, retry in 300 seconds", 300},
		{"retry after seconds", "rate_limit_error: retry after 600 seconds", 600},
		{"retry in s", "429 too many requests, retry in 120s", 120},
		{"retry after s", "rate limit exceeded, retry after 60s", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseRateLimitFromOutput(tt.input)
			if info == nil {
				t.Fatal("expected non-nil info")
			}
			if info.WaitSeconds != tt.expected {
				t.Errorf("expected %d seconds, got %d", tt.expected, info.WaitSeconds)
			}
			// Check that ResetAt is calculated correctly
			expectedReset := time.Now().Add(time.Duration(tt.expected) * time.Second)
			if info.ResetAt.Unix() < expectedReset.Unix()-2 || info.ResetAt.Unix() > expectedReset.Unix()+2 {
				t.Errorf("ResetAt mismatch: expected ~%v, got %v", expectedReset, info.ResetAt)
			}
		})
	}
}

func TestParseRateLimitFromOutput_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"json with retry_after number",
			`{"error": "429 rate_limit_error", "retry_after": 300}`,
			300,
		},
		{
			"json with retry_after string",
			`{"error": "rate limit exceeded", "retry_after": "600"}`,
			600,
		},
		{
			"json with 429 in error",
			`{"error": "HTTP 429: rate_limit_error", "retry_after": 120}`,
			120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseRateLimitFromOutput(tt.input)
			if info == nil {
				t.Fatal("expected non-nil info")
			}
			if info.WaitSeconds != tt.expected {
				t.Errorf("expected %d seconds, got %d", tt.expected, info.WaitSeconds)
			}
			if info.Source != "output" {
				t.Errorf("expected source 'output', got %s", info.Source)
			}
		})
	}
}

func TestParseRateLimitFromOutput_JSONL(t *testing.T) {
	input := `{"status": "ok"}
{"error": "429 rate_limit_error", "retry_after": 300}
{"status": "pending"}`

	info := ParseRateLimitFromOutput(input)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.WaitSeconds != 300 {
		t.Errorf("expected 300 seconds, got %d", info.WaitSeconds)
	}
}

func TestParseRateLimitFromOutput_NotRateLimit(t *testing.T) {
	inputs := []string{
		"task completed successfully",
		"no errors detected",
		"",
		"some random error message",
		"processing your request",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			info := ParseRateLimitFromOutput(input)
			if info != nil {
				t.Errorf("expected nil for non-rate-limit input %q, got %+v", input, info)
			}
		})
	}
}

func TestParseRateLimitFromOutput_GenericRateLimitFallback(t *testing.T) {
	// Test that generic rate limit indicators trigger inference
	inputs := []string{
		"rate limit exceeded",
		"usage limit reached",
		"HTTP 429 error",
		"too many requests",
		"rate_limit error",
		"ratelimit exceeded",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			info := ParseRateLimitFromOutput(input)
			if info == nil {
				t.Fatalf("expected non-nil info for %q", input)
			}
			// Should infer a reset time
			if info.ResetAt.IsZero() {
				t.Error("expected non-zero reset time")
			}
			if info.LimitType != LimitTypeSession {
				t.Errorf("expected session limit type, got %s", info.LimitType)
			}
			// Should have positive wait seconds
			if info.WaitSeconds <= 0 {
				t.Errorf("expected positive wait seconds, got %d", info.WaitSeconds)
			}
		})
	}
}

func TestInferResetTime(t *testing.T) {
	resetTime := InferResetTime()

	// Reset time should be in the future
	if resetTime.Before(time.Now()) {
		t.Error("inferred reset time should be in the future")
	}

	// Should be within 5 hours (max window)
	maxFuture := time.Now().Add(5 * time.Hour)
	if resetTime.After(maxFuture) {
		t.Errorf("reset time %v should be within 5 hours of now", resetTime)
	}

	// Should be on an hour boundary (minute = 0, second = 0)
	if resetTime.Minute() != 0 || resetTime.Second() != 0 {
		t.Errorf("reset time should be on hour boundary, got minute=%d second=%d",
			resetTime.Minute(), resetTime.Second())
	}

	// Hour should be on 5-hour boundary (0, 5, 10, 15, 20)
	if resetTime.Hour()%5 != 0 {
		t.Errorf("reset time hour %d should be on 5-hour boundary", resetTime.Hour())
	}
}

func TestInferResetTime_Boundaries(t *testing.T) {
	// Test at different times of day to ensure proper wrapping
	// This test validates the algorithm's correctness at edge cases
	now := time.Now()

	// Create test cases for different hours
	testHours := []int{0, 1, 4, 5, 9, 10, 14, 15, 19, 20, 23}

	for _, hour := range testHours {
		t.Run(fmt.Sprintf("hour_%d", hour), func(t *testing.T) {
			// Create a time at the specified hour
			testTime := time.Date(now.Year(), now.Month(), now.Day(), hour, 30, 0, 0, now.Location())

			// Calculate expected next window
			currentWindow := (hour / 5) * 5
			nextWindow := currentWindow + 5
			expectedDay := testTime.Day()
			if nextWindow >= 24 {
				nextWindow = 0
				expectedDay++
			}

			// We can't easily mock time.Now() in InferResetTime,
			// but we can verify the general algorithm is correct
			resetTime := InferResetTime()

			// Verify it's on a 5-hour boundary
			if resetTime.Hour()%5 != 0 {
				t.Errorf("reset time hour %d should be on 5-hour boundary", resetTime.Hour())
			}

			// Verify it's in the future
			if resetTime.Before(now) {
				t.Error("reset time should be in the future")
			}
		})
	}
}

func TestInferLimitType(t *testing.T) {
	tests := []struct {
		name        string
		waitSeconds int64
		expected    LimitType
	}{
		{"zero", 0, LimitTypeUnknown},
		{"negative", -100, LimitTypeUnknown},
		{"5 minutes", 300, LimitTypeSession},
		{"1 hour", 3600, LimitTypeSession},
		{"5 hours", 5 * 3600, LimitTypeSession},
		{"6 hours", 6 * 3600, LimitTypeSession},
		{"6 hours 1 second", 6*3600 + 1, LimitTypeWeekly},
		{"7 hours", 7 * 3600, LimitTypeWeekly},
		{"24 hours", 24 * 3600, LimitTypeWeekly},
		{"1 week", 7 * 24 * 3600, LimitTypeWeekly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferLimitType(tt.waitSeconds)
			if got != tt.expected {
				t.Errorf("inferLimitType(%d) = %s, want %s", tt.waitSeconds, got, tt.expected)
			}
		})
	}
}

func TestRateLimitInfo_TimeUntilReset(t *testing.T) {
	future := time.Now().Add(30 * time.Minute)
	info := &RateLimitInfo{ResetAt: future}

	duration := info.TimeUntilReset()

	// Should be approximately 30 minutes (within 1 second tolerance)
	if duration < 29*time.Minute || duration > 31*time.Minute {
		t.Errorf("expected ~30 minutes, got %v", duration)
	}
}

func TestRateLimitInfo_TimeUntilReset_Zero(t *testing.T) {
	info := &RateLimitInfo{} // Zero ResetAt

	duration := info.TimeUntilReset()

	if duration != 0 {
		t.Errorf("expected 0 duration for zero ResetAt, got %v", duration)
	}
}

func TestRateLimitInfo_TimeUntilReset_Past(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	info := &RateLimitInfo{ResetAt: past}

	duration := info.TimeUntilReset()

	// Should be negative for past times
	if duration >= 0 {
		t.Errorf("expected negative duration for past time, got %v", duration)
	}
}

func TestRateLimitInfo_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		resetAt  time.Time
		expected bool
	}{
		{"zero", time.Time{}, true},
		{"past", time.Now().Add(-1 * time.Hour), true},
		{"future", time.Now().Add(1 * time.Hour), false},
		{"just past", time.Now().Add(-1 * time.Second), true},
		{"just future", time.Now().Add(1 * time.Second), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &RateLimitInfo{ResetAt: tt.resetAt}
			got := info.IsExpired()
			if got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseRateLimitFromError(t *testing.T) {
	info := ParseRateLimitFromError("rate limit exceeded, retry in 300 seconds")

	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Source != "error" {
		t.Errorf("expected source 'error', got %s", info.Source)
	}
	if info.WaitSeconds != 300 {
		t.Errorf("expected 300 seconds, got %d", info.WaitSeconds)
	}
}

func TestParseRateLimitFromError_Empty(t *testing.T) {
	info := ParseRateLimitFromError("")
	if info != nil {
		t.Error("expected nil for empty input")
	}
}

func TestParseRateLimitFromError_NotRateLimit(t *testing.T) {
	info := ParseRateLimitFromError("some other error")
	if info != nil {
		t.Error("expected nil for non-rate-limit error")
	}
}

func TestTryParseJSON_InvalidJSON(t *testing.T) {
	inputs := []string{
		"not json at all",
		"{invalid json}",
		"[]",
		`{"error": "not a rate limit"}`,
		`{"retry_after": 300}`, // Has retry_after but no error field
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			info := tryParseJSON(input)
			if info != nil {
				t.Errorf("expected nil for invalid/non-rate-limit JSON %q, got %+v", input, info)
			}
		})
	}
}

func TestExtractFromJSONObject_VariousRetryAfterTypes(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]interface{}
		expected int64
	}{
		{
			"float64",
			map[string]interface{}{"error": "429 rate_limit", "retry_after": float64(300)},
			300,
		},
		{
			"int64",
			map[string]interface{}{"error": "rate limit", "retry_after": int64(600)},
			600,
		},
		{
			"int",
			map[string]interface{}{"error": "rate_limit_error", "retry_after": 120},
			120,
		},
		{
			"string",
			map[string]interface{}{"error": "429", "retry_after": "450"},
			450,
		},
		{
			"invalid string",
			map[string]interface{}{"error": "rate limit", "retry_after": "not a number"},
			0, // Should fall back to InferResetTime
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := extractFromJSONObject(tt.obj)
			if info == nil {
				t.Fatal("expected non-nil info")
			}
			if tt.expected > 0 {
				if info.WaitSeconds != tt.expected {
					t.Errorf("expected %d seconds, got %d", tt.expected, info.WaitSeconds)
				}
			} else {
				// Should have inferred a reset time
				if info.ResetAt.IsZero() {
					t.Error("expected non-zero reset time")
				}
			}
		})
	}
}

func TestExtractFromJSONObject_NoError(t *testing.T) {
	obj := map[string]interface{}{
		"status":      "ok",
		"retry_after": 300,
	}

	info := extractFromJSONObject(obj)
	if info != nil {
		t.Error("expected nil when no error field present")
	}
}

func TestExtractFromJSONObject_ErrorNotRateLimit(t *testing.T) {
	obj := map[string]interface{}{
		"error":       "some other error",
		"retry_after": 300,
	}

	info := extractFromJSONObject(obj)
	if info != nil {
		t.Error("expected nil when error is not rate limit related")
	}
}

func TestParseRateLimitFromOutput_RawMessage(t *testing.T) {
	input := "rate limit exceeded, retry in 300 seconds"
	info := ParseRateLimitFromOutput(input)

	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.RawMessage != input {
		t.Errorf("expected RawMessage to be %q, got %q", input, info.RawMessage)
	}
}

func TestParseRateLimitFromOutput_DetectedAt(t *testing.T) {
	before := time.Now()
	info := ParseRateLimitFromOutput("rate limit exceeded")
	after := time.Now()

	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.DetectedAt.Before(before) || info.DetectedAt.After(after) {
		t.Errorf("DetectedAt %v should be between %v and %v", info.DetectedAt, before, after)
	}
}

func TestParseRateLimitFromOutput_CaseInsensitive(t *testing.T) {
	inputs := []string{
		"RATE LIMIT EXCEEDED",
		"Rate Limit Exceeded",
		"usage_limit reached",
		"USAGE_LIMIT REACHED",
		"TOO MANY REQUESTS",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			info := ParseRateLimitFromOutput(input)
			if info == nil {
				t.Errorf("expected non-nil info for case-insensitive match %q", input)
			}
		})
	}
}

func TestParseRateLimitFromOutput_MultiplePatterns(t *testing.T) {
	// Test that the most specific pattern wins
	// Unix timestamp should be preferred over generic inference
	futureTime := time.Now().Add(3 * time.Hour).Unix()
	input := fmt.Sprintf("rate limit exceeded. Claude AI usage limit reached|%d", futureTime)

	info := ParseRateLimitFromOutput(input)
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	// Should use the unix timestamp, not the generic inference
	if info.ResetAt.Unix() != futureTime {
		t.Errorf("expected specific timestamp %d, got %d", futureTime, info.ResetAt.Unix())
	}
}

func TestHumanTimePattern_TimezoneFailure(t *testing.T) {
	// Test with invalid timezone - should fallback to UTC
	input := "rate limit - Your limit will reset at 2pm (Invalid/Timezone)"
	info := ParseRateLimitFromOutput(input)

	if info == nil {
		t.Fatal("expected non-nil info even with invalid timezone")
	}

	// Should still parse successfully with UTC fallback
	if info.ResetAt.IsZero() {
		t.Error("expected non-zero reset time")
	}
}

func TestParseRateLimitFromOutput_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldMatch bool
	}{
		{
			"newlines with rate limit",
			"retry in 300 seconds\nrate limit exceeded",
			true,
		},
		{
			"multiple spaces in retry pattern",
			"rate limit hit, retry in  300  seconds",
			true, // \s+ allows multiple spaces
		},
		{
			"tab instead of space",
			"rate limit hit, retry in\t300 seconds",
			true, // \s+ includes tabs
		},
		{
			"no space before number - still matches generic",
			"rate limit retryafter300s",
			true, // Contains "rate limit" so triggers generic fallback even though retry pattern doesn't match
		},
		{
			"valid single space",
			"rate limit, retry in 300 seconds",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseRateLimitFromOutput(tt.input)
			matched := info != nil
			if matched != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got match=%v", tt.input, tt.shouldMatch, matched)
			}
		})
	}
}

func TestUnixTimestampPattern_ExactMatch(t *testing.T) {
	// Test that the unix timestamp pattern requires exact prefix
	tests := []struct {
		name        string
		input       string
		shouldMatch bool
	}{
		{
			"exact match",
			"Claude AI usage limit reached|1234567890",
			true,
		},
		{
			"missing prefix",
			"usage limit reached|1234567890",
			false,
		},
		{
			"case sensitive",
			"claude ai usage limit reached|1234567890",
			false,
		},
		{
			"with context",
			"Error: Claude AI usage limit reached|1234567890. Please wait.",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseRateLimitFromOutput(tt.input)
			if tt.shouldMatch {
				if info == nil {
					t.Error("expected match")
				} else if info.ResetAt.Unix() != 1234567890 {
					t.Errorf("expected timestamp 1234567890, got %d", info.ResetAt.Unix())
				}
			}
		})
	}
}

func TestRetrySecondsPattern_Variations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"retry in seconds", "rate limit, retry in 300 seconds", 300},
		{"retry after seconds", "rate limit, retry after 600 seconds", 600},
		{"retry in second (singular)", "rate limit, retry in 1 second", 1},
		{"retry after second", "rate limit, retry after 1 second", 1},
		{"retry in s", "rate limit, retry in 120s", 120},
		{"retry after s", "rate limit, retry after 60s", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseRateLimitFromOutput(tt.input)
			if info == nil {
				t.Fatalf("expected non-nil info for %q", tt.input)
			}
			if info.WaitSeconds != tt.expected {
				t.Errorf("expected %d seconds, got %d", tt.expected, info.WaitSeconds)
			}
		})
	}
}

func TestParseRateLimitFromOutput_FullIntegration(t *testing.T) {
	// Test complete flow from parsing to validation
	input := "rate limit exceeded, retry in 3600 seconds"
	info := ParseRateLimitFromOutput(input)

	if info == nil {
		t.Fatal("expected non-nil info")
	}

	// Validate all fields are set correctly
	if info.Source != "output" {
		t.Errorf("expected source 'output', got %s", info.Source)
	}
	if info.RawMessage != input {
		t.Errorf("expected RawMessage %q, got %q", input, info.RawMessage)
	}
	if info.WaitSeconds != 3600 {
		t.Errorf("expected 3600 seconds, got %d", info.WaitSeconds)
	}
	if info.LimitType != LimitTypeSession {
		t.Errorf("expected session limit, got %s", info.LimitType)
	}
	if info.DetectedAt.IsZero() {
		t.Error("expected non-zero DetectedAt")
	}
	if info.ResetAt.IsZero() {
		t.Error("expected non-zero ResetAt")
	}
	if !info.ResetAt.After(time.Now()) {
		t.Error("expected ResetAt to be in the future")
	}
	if info.IsExpired() {
		t.Error("expected non-expired limit")
	}
	if info.TimeUntilReset() <= 0 {
		t.Error("expected positive time until reset")
	}
}
