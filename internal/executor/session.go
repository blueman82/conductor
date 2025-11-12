package executor

import (
	"fmt"
	"time"
)

// generateSessionID creates a unique session identifier with timestamp format
// Format: session-YYYYMMDD-HHMM
// Example: session-20251112-1430
func generateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("session-%s", now.Format("20060102-1504"))
}
