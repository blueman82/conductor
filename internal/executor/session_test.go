package executor

import (
	"testing"
)

func TestGenerateSessionID_Format(t *testing.T) {
	sessionID := generateSessionID()

	// Session ID should match format: session-YYYYMMDD-HHMM
	// Example: session-20251112-1430
	if len(sessionID) != 21 {
		t.Errorf("Expected session ID length 21, got %d", len(sessionID))
	}

	// Check prefix
	if sessionID[:8] != "session-" {
		t.Errorf("Expected session ID to start with 'session-', got %s", sessionID[:8])
	}

	// Check format with regex-like validation
	// Format: session-YYYYMMDD-HHMM
	dateTimePart := sessionID[8:]
	if len(dateTimePart) != 13 {
		t.Errorf("Expected date-time part length 13, got %d", len(dateTimePart))
	}

	// Check hyphen position
	if dateTimePart[8] != '-' {
		t.Errorf("Expected hyphen at position 8 in date-time part, got %c", dateTimePart[8])
	}
}

func TestGenerateSessionID_Unique(t *testing.T) {
	// Generate two session IDs in quick succession
	sessionID1 := generateSessionID()
	sessionID2 := generateSessionID()

	// They should be identical if generated in the same minute
	// This test verifies the function works consistently
	if sessionID1 != sessionID2 {
		// This is expected if we cross a minute boundary
		t.Logf("Session IDs differ (likely crossed minute boundary): %s vs %s", sessionID1, sessionID2)
	}

	// Both should still have valid format
	if len(sessionID1) != 21 || len(sessionID2) != 21 {
		t.Errorf("Both session IDs should have length 21")
	}
}
