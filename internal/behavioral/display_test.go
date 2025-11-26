package behavioral

import (
	"strings"
	"testing"
	"time"
)

func TestNewPaginator(t *testing.T) {
	tests := []struct {
		name         string
		items        []interface{}
		pageSize     int
		wantPages    int
		wantCurrPage int
	}{
		{
			name:         "empty items",
			items:        []interface{}{},
			pageSize:     10,
			wantPages:    0,
			wantCurrPage: 1,
		},
		{
			name:         "exactly one page",
			items:        makeItems(10),
			pageSize:     10,
			wantPages:    1,
			wantCurrPage: 1,
		},
		{
			name:         "multiple full pages",
			items:        makeItems(100),
			pageSize:     50,
			wantPages:    2,
			wantCurrPage: 1,
		},
		{
			name:         "partial last page",
			items:        makeItems(55),
			pageSize:     50,
			wantPages:    2,
			wantCurrPage: 1,
		},
		{
			name:         "invalid page size defaults to 50",
			items:        makeItems(100),
			pageSize:     0,
			wantPages:    2,
			wantCurrPage: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPaginator(tt.items, tt.pageSize)

			if p.GetTotalPages() != tt.wantPages {
				t.Errorf("GetTotalPages() = %d, want %d", p.GetTotalPages(), tt.wantPages)
			}

			if p.GetCurrentPageNum() != tt.wantCurrPage {
				t.Errorf("GetCurrentPageNum() = %d, want %d", p.GetCurrentPageNum(), tt.wantCurrPage)
			}
		})
	}
}

func TestPaginatorNavigation(t *testing.T) {
	items := makeItems(125)
	p := NewPaginator(items, 50)

	// Should start at page 1
	if p.GetCurrentPageNum() != 1 {
		t.Errorf("Expected page 1, got %d", p.GetCurrentPageNum())
	}

	// Should have 3 pages total (50 + 50 + 25)
	if p.GetTotalPages() != 3 {
		t.Errorf("Expected 3 pages, got %d", p.GetTotalPages())
	}

	// Next page should work
	if !p.NextPage() {
		t.Error("NextPage() should return true")
	}
	if p.GetCurrentPageNum() != 2 {
		t.Errorf("Expected page 2, got %d", p.GetCurrentPageNum())
	}

	// Another next page should work
	if !p.NextPage() {
		t.Error("NextPage() should return true")
	}
	if p.GetCurrentPageNum() != 3 {
		t.Errorf("Expected page 3, got %d", p.GetCurrentPageNum())
	}

	// Next page at end should fail
	if p.NextPage() {
		t.Error("NextPage() should return false at last page")
	}
	if p.GetCurrentPageNum() != 3 {
		t.Errorf("Expected page 3, got %d", p.GetCurrentPageNum())
	}

	// Previous page should work
	if !p.PrevPage() {
		t.Error("PrevPage() should return true")
	}
	if p.GetCurrentPageNum() != 2 {
		t.Errorf("Expected page 2, got %d", p.GetCurrentPageNum())
	}

	// Another previous page should work
	if !p.PrevPage() {
		t.Error("PrevPage() should return true")
	}
	if p.GetCurrentPageNum() != 1 {
		t.Errorf("Expected page 1, got %d", p.GetCurrentPageNum())
	}

	// Previous page at start should fail
	if p.PrevPage() {
		t.Error("PrevPage() should return false at first page")
	}
	if p.GetCurrentPageNum() != 1 {
		t.Errorf("Expected page 1, got %d", p.GetCurrentPageNum())
	}
}

func TestPaginatorGetCurrentPage(t *testing.T) {
	items := makeItems(125)
	p := NewPaginator(items, 50)

	// First page should have 50 items
	page1 := p.GetCurrentPage()
	if len(page1) != 50 {
		t.Errorf("Page 1 should have 50 items, got %d", len(page1))
	}

	// Second page should have 50 items
	p.NextPage()
	page2 := p.GetCurrentPage()
	if len(page2) != 50 {
		t.Errorf("Page 2 should have 50 items, got %d", len(page2))
	}

	// Third page should have 25 items
	p.NextPage()
	page3 := p.GetCurrentPage()
	if len(page3) != 25 {
		t.Errorf("Page 3 should have 25 items, got %d", len(page3))
	}
}

func TestPaginatorHasNextPrevPage(t *testing.T) {
	items := makeItems(100)
	p := NewPaginator(items, 50)

	// At page 1
	if p.HasPrevPage() {
		t.Error("HasPrevPage() should be false at page 1")
	}
	if !p.HasNextPage() {
		t.Error("HasNextPage() should be true at page 1")
	}

	// Move to page 2
	p.NextPage()
	if !p.HasPrevPage() {
		t.Error("HasPrevPage() should be true at page 2")
	}
	if p.HasNextPage() {
		t.Error("HasNextPage() should be false at last page")
	}
}

func TestFormatToolExecutions(t *testing.T) {
	tools := []ToolExecution{
		{
			Name:         "Read",
			Count:        100,
			SuccessRate:  0.95,
			ErrorRate:    0.05,
			AvgDuration:  100 * time.Millisecond,
			TotalSuccess: 95,
			TotalErrors:  5,
		},
		{
			Name:         "Write",
			Count:        50,
			SuccessRate:  0.80,
			ErrorRate:    0.20,
			AvgDuration:  200 * time.Millisecond,
			TotalSuccess: 40,
			TotalErrors:  10,
		},
	}

	rows := formatToolExecutions(tools, false)

	// Should have header + separator + 2 data rows
	if len(rows) != 4 {
		t.Errorf("Expected 4 rows, got %d", len(rows))
	}

	// Check header
	if !strings.Contains(rows[0], "Name") {
		t.Error("Header should contain 'Name'")
	}

	// Check data rows contain tool names
	if !strings.Contains(rows[2], "Read") {
		t.Error("Row should contain 'Read'")
	}
	if !strings.Contains(rows[3], "Write") {
		t.Error("Row should contain 'Write'")
	}
}

func TestFormatBashCommands(t *testing.T) {
	commands := []BashCommand{
		{
			Command:      "ls -la",
			ExitCode:     0,
			OutputLength: 1024,
			Duration:     50 * time.Millisecond,
			Success:      true,
		},
		{
			Command:      "grep pattern file.txt",
			ExitCode:     1,
			OutputLength: 0,
			Duration:     25 * time.Millisecond,
			Success:      false,
		},
	}

	rows := formatBashCommands(commands, false)

	// Should have header + separator + 2 data rows
	if len(rows) != 4 {
		t.Errorf("Expected 4 rows, got %d", len(rows))
	}

	// Check header
	if !strings.Contains(rows[0], "Command") {
		t.Error("Header should contain 'Command'")
	}

	// Check data rows
	if !strings.Contains(rows[2], "ls -la") {
		t.Error("Row should contain 'ls -la'")
	}
}

func TestFormatFileOperations(t *testing.T) {
	ops := []FileOperation{
		{
			Type:      "read",
			Path:      "/path/to/file.go",
			SizeBytes: 2048,
			Success:   true,
			Duration:  10,
		},
		{
			Type:      "write",
			Path:      "/path/to/output.go",
			SizeBytes: 4096,
			Success:   true,
			Duration:  20,
		},
	}

	rows := formatFileOperations(ops, false)

	// Should have header + separator + 2 data rows
	if len(rows) != 4 {
		t.Errorf("Expected 4 rows, got %d", len(rows))
	}

	// Check header
	if !strings.Contains(rows[0], "Type") {
		t.Error("Header should contain 'Type'")
	}

	// Check data rows
	if !strings.Contains(rows[2], "read") {
		t.Error("Row should contain 'read'")
	}
	if !strings.Contains(rows[3], "write") {
		t.Error("Row should contain 'write'")
	}
}

func TestFormatSessions(t *testing.T) {
	sessions := []Session{
		{
			ID:         "session-001",
			Project:    "conductor",
			Status:     "completed",
			Duration:   5000,
			Success:    true,
			ErrorCount: 0,
		},
		{
			ID:         "session-002",
			Project:    "test-project",
			Status:     "failed",
			Duration:   3000,
			Success:    false,
			ErrorCount: 5,
		},
	}

	rows := formatSessions(sessions, false)

	// Should have header + separator + 2 data rows
	if len(rows) != 4 {
		t.Errorf("Expected 4 rows, got %d", len(rows))
	}

	// Check header
	if !strings.Contains(rows[0], "Session ID") {
		t.Error("Header should contain 'Session ID'")
	}

	// Check data rows
	if !strings.Contains(rows[2], "session-001") {
		t.Error("Row should contain 'session-001'")
	}
	if !strings.Contains(rows[3], "session-002") {
		t.Error("Row should contain 'session-002'")
	}
}

func TestFormatTable(t *testing.T) {
	// Test with tool executions
	tools := []ToolExecution{
		{Name: "Read", Count: 10},
	}
	rows := FormatTable(tools, false)
	if len(rows) == 0 {
		t.Error("FormatTable should return rows for tool executions")
	}

	// Test with empty slice
	emptyTools := []ToolExecution{}
	rows = FormatTable(emptyTools, false)
	if len(rows) != 1 || !strings.Contains(rows[0], "No tool executions found") {
		t.Error("FormatTable should return 'No tool executions found' for empty slice")
	}
}

func TestPrintNavigationBar(t *testing.T) {
	tests := []struct {
		name        string
		currentPage int
		totalPages  int
		wantEmpty   bool
		wantPrev    bool
		wantNext    bool
	}{
		{
			name:        "single page",
			currentPage: 1,
			totalPages:  1,
			wantEmpty:   true,
		},
		{
			name:        "first of many pages",
			currentPage: 1,
			totalPages:  5,
			wantPrev:    false,
			wantNext:    true,
		},
		{
			name:        "middle page",
			currentPage: 3,
			totalPages:  5,
			wantPrev:    true,
			wantNext:    true,
		},
		{
			name:        "last page",
			currentPage: 5,
			totalPages:  5,
			wantPrev:    true,
			wantNext:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nav := PrintNavigationBar(tt.currentPage, tt.totalPages, false)

			if tt.wantEmpty {
				if nav != "" {
					t.Error("Expected empty navigation bar for single page")
				}
				return
			}

			// Check page indicator
			if !strings.Contains(nav, "Page") {
				t.Error("Navigation bar should contain 'Page'")
			}

			// Check prev/next hints
			hasPrev := strings.Contains(nav, "Previous")
			hasNext := strings.Contains(nav, "Next")

			if hasPrev != tt.wantPrev {
				t.Errorf("Expected Previous=%v, got %v", tt.wantPrev, hasPrev)
			}
			if hasNext != tt.wantNext {
				t.Errorf("Expected Next=%v, got %v", tt.wantNext, hasNext)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

// Helper function to create test items
func makeItems(count int) []interface{} {
	items := make([]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = i
	}
	return items
}
