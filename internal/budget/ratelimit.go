package budget

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LimitType distinguishes session (5h) from weekly limits
type LimitType string

const (
	LimitTypeSession LimitType = "session"
	LimitTypeWeekly  LimitType = "weekly"
	LimitTypeUnknown LimitType = "unknown"
)

// RateLimitInfo contains parsed rate limit details
type RateLimitInfo struct {
	DetectedAt  time.Time
	ResetAt     time.Time // When limit resets
	WaitSeconds int64
	LimitType   LimitType // "session" or "weekly"
	RawMessage  string
	Source      string // "output", "error", "block"
}

// TimeUntilReset calculates duration until the rate limit resets
func (r *RateLimitInfo) TimeUntilReset() time.Duration {
	if r.ResetAt.IsZero() {
		return 0
	}
	return time.Until(r.ResetAt)
}

// IsExpired checks if the rate limit has already expired
func (r *RateLimitInfo) IsExpired() bool {
	if r.ResetAt.IsZero() {
		return true
	}
	return time.Now().After(r.ResetAt)
}

var (
	// Pattern 1: Claude AI usage limit reached|<unix_timestamp>
	unixTimestampPattern = regexp.MustCompile(`Claude AI usage limit reached\|(\d+)`)

	// Pattern 2: Your limit will reset at 2pm (America/New_York)
	humanTimePattern = regexp.MustCompile(`limit will reset at (\d+)(am|pm)\s*\(([^)]+)\)`)

	// Pattern 3: retry in 300 seconds / retry after 300s
	retrySecondsPattern = regexp.MustCompile(`retry (?:in|after)\s+(\d+)\s*(?:seconds?|s)`)

	// Pattern 4: Generic rate limit indicators (v2.20.1: added "out of.*usage")
	rateLimitIndicator = regexp.MustCompile(`(?i)(out of.*usage|rate.?limit|usage.?limit|429|too.?many.?requests)`)

	// Pattern 5: "resets 1am (Europe/Dublin)" format from Claude CLI (v2.20.1+)
	resetsTimePattern = regexp.MustCompile(`resets\s+(\d+)(am|pm)\s*\(([^)]+)\)`)

	// Pattern 6: False positive exclusions - displayed/logged text, not actual errors (v2.28+)
	falsePositivePattern = regexp.MustCompile(`(?i)(\[RATE.?LIMIT\]|` + // Log prefixes like [RATE LIMIT]
		"`rate.?limit|" + // Markdown inline code
		`"rate.?limit|` + // Quoted strings
		`'rate.?limit|` + // Single-quoted strings
		`waiting for reset\.\.\.|` + // Historical log messages
		`until auto-resume)`) // Countdown display text
)

// ParseRateLimitFromOutput parses rate limit info from CLI stdout/stderr
func ParseRateLimitFromOutput(output string) *RateLimitInfo {
	if output == "" {
		return nil
	}

	// Check if this looks like a rate limit message
	if !rateLimitIndicator.MatchString(output) {
		return nil
	}

	info := &RateLimitInfo{
		DetectedAt: time.Now(),
		RawMessage: output,
		Source:     "output",
		LimitType:  LimitTypeUnknown,
	}

	// Try parsing patterns in order of specificity

	// Pattern 1: Unix timestamp
	if matches := unixTimestampPattern.FindStringSubmatch(output); len(matches) > 1 {
		if ts, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			info.ResetAt = time.Unix(ts, 0)
			info.WaitSeconds = info.ResetAt.Unix() - time.Now().Unix()
			info.LimitType = inferLimitType(info.WaitSeconds)
			return info
		}
	}

	// Pattern 2: Human-readable time with timezone ("limit will reset at 2pm")
	if matches := humanTimePattern.FindStringSubmatch(output); len(matches) > 3 {
		hour, _ := strconv.Atoi(matches[1])
		meridiem := matches[2]
		tzName := matches[3]

		// Convert 12-hour to 24-hour
		if meridiem == "pm" && hour != 12 {
			hour += 12
		} else if meridiem == "am" && hour == 12 {
			hour = 0
		}

		// Load timezone
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			// Fallback to UTC if timezone parsing fails
			loc = time.UTC
		}

		// Construct reset time
		now := time.Now().In(loc)
		resetAt := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, loc)

		// If reset time is in the past, assume next day
		if resetAt.Before(now) {
			resetAt = resetAt.Add(24 * time.Hour)
		}

		info.ResetAt = resetAt
		info.WaitSeconds = int64(time.Until(resetAt).Seconds())
		info.LimitType = inferLimitType(info.WaitSeconds)
		return info
	}

	// Pattern 2b: "resets 1am (Europe/Dublin)" format (v2.20.1+)
	if matches := resetsTimePattern.FindStringSubmatch(output); len(matches) > 3 {
		hour, _ := strconv.Atoi(matches[1])
		meridiem := matches[2]
		tzName := matches[3]

		// Convert 12-hour to 24-hour
		if meridiem == "pm" && hour != 12 {
			hour += 12
		} else if meridiem == "am" && hour == 12 {
			hour = 0
		}

		// Load timezone
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			// Fallback to UTC if timezone parsing fails
			loc = time.UTC
		}

		// Construct reset time
		now := time.Now().In(loc)
		resetAt := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, loc)

		// If reset time is in the past, assume next day
		if resetAt.Before(now) {
			resetAt = resetAt.Add(24 * time.Hour)
		}

		info.ResetAt = resetAt
		info.WaitSeconds = int64(time.Until(resetAt).Seconds())
		info.LimitType = inferLimitType(info.WaitSeconds)
		return info
	}

	// Pattern 3: Retry seconds
	if matches := retrySecondsPattern.FindStringSubmatch(output); len(matches) > 1 {
		if seconds, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			info.WaitSeconds = seconds
			info.ResetAt = time.Now().Add(time.Duration(seconds) * time.Second)
			info.LimitType = inferLimitType(seconds)
			return info
		}
	}

	// Pattern 4: Try JSON/JSONL parsing
	if jsonInfo := tryParseJSON(output); jsonInfo != nil {
		jsonInfo.DetectedAt = info.DetectedAt
		jsonInfo.Source = info.Source
		jsonInfo.RawMessage = info.RawMessage
		return jsonInfo
	}

	// If we detected rate limit indicator but can't parse details, infer reset time
	info.ResetAt = InferResetTime()
	info.WaitSeconds = int64(time.Until(info.ResetAt).Seconds())
	info.LimitType = LimitTypeSession
	return info
}

// ParseRateLimitFromError parses rate limit info from error messages
func ParseRateLimitFromError(errMsg string) *RateLimitInfo {
	if errMsg == "" {
		return nil
	}

	info := ParseRateLimitFromOutput(errMsg)
	if info != nil {
		info.Source = "error"
	}
	return info
}

// InferResetTime calculates reset time when not explicitly provided
// Uses 5-hour billing window floored to hour boundary
func InferResetTime() time.Time {
	now := time.Now()

	// Floor to current hour
	flooredNow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// Calculate hours since midnight
	hoursSinceMidnight := flooredNow.Hour()

	// Find next 5-hour boundary (0, 5, 10, 15, 20)
	currentWindow := (hoursSinceMidnight / 5) * 5
	nextWindow := currentWindow + 5

	// If next window exceeds 24 hours, wrap to next day
	if nextWindow >= 24 {
		nextWindow = 0
		flooredNow = flooredNow.Add(24 * time.Hour)
	}

	resetAt := time.Date(flooredNow.Year(), flooredNow.Month(), flooredNow.Day(), nextWindow, 0, 0, 0, flooredNow.Location())

	return resetAt
}

// inferLimitType determines limit type based on wait duration
// If wait > 6 hours, classify as weekly limit
func inferLimitType(waitSeconds int64) LimitType {
	const sixHoursInSeconds = 6 * 60 * 60

	if waitSeconds <= 0 {
		return LimitTypeUnknown
	}

	if waitSeconds > sixHoursInSeconds {
		return LimitTypeWeekly
	}

	return LimitTypeSession
}

// tryParseJSON attempts to extract rate limit info from JSON/JSONL
func tryParseJSON(data string) *RateLimitInfo {
	// Try parsing as single JSON object
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(data), &obj); err == nil {
		return extractFromJSONObject(obj)
	}

	// Try parsing as JSONL (line-by-line)
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			if info := extractFromJSONObject(obj); info != nil {
				return info
			}
		}
	}

	return nil
}

// extractFromJSONObject extracts rate limit info from a parsed JSON object
func extractFromJSONObject(obj map[string]interface{}) *RateLimitInfo {
	// Look for common rate limit fields
	errorField, hasError := obj["error"]
	retryAfter, hasRetryAfter := obj["retry_after"]

	// Check if this is a rate limit error
	isRateLimit := false
	if hasError {
		if errStr, ok := errorField.(string); ok {
			isRateLimit = strings.Contains(errStr, "429") ||
				strings.Contains(strings.ToLower(errStr), "rate_limit") ||
				strings.Contains(strings.ToLower(errStr), "rate limit")
		}
	}

	if !isRateLimit {
		return nil
	}

	info := &RateLimitInfo{
		DetectedAt: time.Now(),
		LimitType:  LimitTypeUnknown,
	}

	// Extract retry_after if present
	if hasRetryAfter {
		switch v := retryAfter.(type) {
		case float64:
			info.WaitSeconds = int64(v)
		case int64:
			info.WaitSeconds = v
		case int:
			info.WaitSeconds = int64(v)
		case string:
			if seconds, err := strconv.ParseInt(v, 10, 64); err == nil {
				info.WaitSeconds = seconds
			}
		}

		if info.WaitSeconds > 0 {
			info.ResetAt = time.Now().Add(time.Duration(info.WaitSeconds) * time.Second)
			info.LimitType = inferLimitType(info.WaitSeconds)
			return info
		}
	}

	// If no retry_after, infer reset time
	info.ResetAt = InferResetTime()
	info.WaitSeconds = int64(time.Until(info.ResetAt).Seconds())
	info.LimitType = LimitTypeSession
	return info
}
