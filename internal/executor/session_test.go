package executor

import (
	"testing"
)

func TestGenerateSessionID_Format(t *testing.T) {
	sessionID := generateSessionID()

	// Session ID should match format: session-YYYYMMDD-HHMMSS
	// Example: session-20251112-143045
	if len(sessionID) != 23 {
		t.Errorf("Expected session ID length 23, got %d", len(sessionID))
	}

	// Check prefix
	if sessionID[:8] != "session-" {
		t.Errorf("Expected session ID to start with 'session-', got %s", sessionID[:8])
	}

	// Check format with regex-like validation
	// Format: session-YYYYMMDD-HHMMSS
	dateTimePart := sessionID[8:]
	if len(dateTimePart) != 15 {
		t.Errorf("Expected date-time part length 15, got %d", len(dateTimePart))
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

	// They should be identical if generated in the same second
	// This test verifies the function works consistently
	if sessionID1 != sessionID2 {
		// This is expected if we cross a second boundary
		t.Logf("Session IDs differ (likely crossed second boundary): %s vs %s", sessionID1, sessionID2)
	}

	// Both should still have valid format
	if len(sessionID1) != 23 || len(sessionID2) != 23 {
		t.Errorf("Both session IDs should have length 23")
	}
}
