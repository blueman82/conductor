package behavioral

import (
	"testing"
	"time"
)

func TestFilterCriteria_Validate(t *testing.T) {
	tests := []struct {
		name    string
		criteria FilterCriteria
		wantErr bool
		errMsg  string
	}{
		{
			name:     "empty criteria is valid",
			criteria: FilterCriteria{},
			wantErr:  false,
		},
		{
			name: "valid event type tool",
			criteria: FilterCriteria{
				EventType: "tool",
			},
			wantErr: false,
		},
		{
			name: "valid event type bash",
			criteria: FilterCriteria{
				EventType: "bash",
			},
			wantErr: false,
		},
		{
			name: "valid event type file",
			criteria: FilterCriteria{
				EventType: "file",
			},
			wantErr: false,
		},
		{
			name: "invalid event type",
			criteria: FilterCriteria{
				EventType: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid event type",
		},
		{
			name: "valid time range",
			criteria: FilterCriteria{
				Since: time.Now().Add(-24 * time.Hour),
				Until: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid time range - since after until",
			criteria: FilterCriteria{
				Since: time.Now(),
				Until: time.Now().Add(-24 * time.Hour),
			},
			wantErr: true,
			errMsg:  "since time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.criteria.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FilterCriteria.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if len(err.Error()) < len(tt.errMsg) || err.Error()[:len(tt.errMsg)] != tt.errMsg {
					t.Errorf("FilterCriteria.Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestApplyFiltersToSessions(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	sessions := []Session{
		{ID: "1", Project: "conductor", AgentName: "golang-pro", Timestamp: now, ErrorCount: 0},
		{ID: "2", Project: "myapp", AgentName: "python-pro", Timestamp: yesterday, ErrorCount: 2},
		{ID: "3", Project: "conductor", AgentName: "rust-engineer", Timestamp: twoDaysAgo, ErrorCount: 1},
		{ID: "4", Project: "webapp", AgentName: "golang-pro", Timestamp: now, ErrorCount: 0},
	}

	tests := []struct {
		name     string
		sessions []Session
		criteria FilterCriteria
		want     int
	}{
		{
			name:     "no filter returns all",
			sessions: sessions,
			criteria: FilterCriteria{},
			want:     4,
		},
		{
			name:     "search by project",
			sessions: sessions,
			criteria: FilterCriteria{Search: "conductor"},
			want:     2,
		},
		{
			name:     "search case insensitive",
			sessions: sessions,
			criteria: FilterCriteria{Search: "CONDUCTOR"},
			want:     2,
		},
		{
			name:     "search by agent",
			sessions: sessions,
			criteria: FilterCriteria{Search: "golang"},
			want:     2,
		},
		{
			name:     "errors only",
			sessions: sessions,
			criteria: FilterCriteria{ErrorsOnly: true},
			want:     2,
		},
		{
			name:     "time range since yesterday",
			sessions: sessions,
			criteria: FilterCriteria{Since: yesterday.Add(-1 * time.Hour)},
			want:     3, // All sessions from yesterday onwards (sessions 1, 2, 4)
		},
		{
			name:     "time range until yesterday",
			sessions: sessions,
			criteria: FilterCriteria{Until: yesterday.Add(1 * time.Hour)},
			want:     2,
		},
		{
			name:     "combined filters - search and errors",
			sessions: sessions,
			criteria: FilterCriteria{Search: "conductor", ErrorsOnly: true},
			want:     1,
		},
		{
			name:     "combined filters - no matches",
			sessions: sessions,
			criteria: FilterCriteria{Search: "nonexistent"},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyFiltersToSessions(tt.sessions, tt.criteria)
			if len(got) != tt.want {
				t.Errorf("ApplyFiltersToSessions() returned %d sessions, want %d", len(got), tt.want)
			}
		})
	}
}

func TestApplyFiltersToToolExecutions(t *testing.T) {
	tools := []ToolExecution{
		{Name: "Read", Count: 10, TotalErrors: 0},
		{Name: "Write", Count: 5, TotalErrors: 1},
		{Name: "Edit", Count: 8, TotalErrors: 0},
		{Name: "Bash", Count: 15, TotalErrors: 3},
	}

	tests := []struct {
		name     string
		tools    []ToolExecution
		criteria FilterCriteria
		want     int
	}{
		{
			name:     "no filter returns all",
			tools:    tools,
			criteria: FilterCriteria{},
			want:     4,
		},
		{
			name:     "search by tool name",
			tools:    tools,
			criteria: FilterCriteria{Search: "read"},
			want:     1,
		},
		{
			name:     "errors only",
			tools:    tools,
			criteria: FilterCriteria{ErrorsOnly: true},
			want:     2,
		},
		{
			name:     "event type tool",
			tools:    tools,
			criteria: FilterCriteria{EventType: "tool"},
			want:     4,
		},
		{
			name:     "event type bash filters out all",
			tools:    tools,
			criteria: FilterCriteria{EventType: "bash"},
			want:     0,
		},
		{
			name:     "combined search and errors",
			tools:    tools,
			criteria: FilterCriteria{Search: "bash", ErrorsOnly: true},
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyFiltersToToolExecutions(tt.tools, tt.criteria)
			if len(got) != tt.want {
				t.Errorf("ApplyFiltersToToolExecutions() returned %d tools, want %d", len(got), tt.want)
			}
		})
	}
}

func TestApplyFiltersToBashCommands(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	commands := []BashCommand{
		{Command: "go test", Success: true, Timestamp: now},
		{Command: "go build", Success: false, Timestamp: yesterday},
		{Command: "git status", Success: true, Timestamp: now},
		{Command: "git commit", Success: false, Timestamp: yesterday},
	}

	tests := []struct {
		name     string
		commands []BashCommand
		criteria FilterCriteria
		want     int
	}{
		{
			name:     "no filter returns all",
			commands: commands,
			criteria: FilterCriteria{},
			want:     4,
		},
		{
			name:     "search by command",
			commands: commands,
			criteria: FilterCriteria{Search: "git"},
			want:     2,
		},
		{
			name:     "errors only",
			commands: commands,
			criteria: FilterCriteria{ErrorsOnly: true},
			want:     2,
		},
		{
			name:     "event type bash",
			commands: commands,
			criteria: FilterCriteria{EventType: "bash"},
			want:     4,
		},
		{
			name:     "event type tool filters out all",
			commands: commands,
			criteria: FilterCriteria{EventType: "tool"},
			want:     0,
		},
		{
			name:     "time range since yesterday",
			commands: commands,
			criteria: FilterCriteria{Since: yesterday.Add(-1 * time.Hour)},
			want:     4,
		},
		{
			name:     "combined filters",
			commands: commands,
			criteria: FilterCriteria{Search: "go", ErrorsOnly: true},
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyFiltersToBashCommands(tt.commands, tt.criteria)
			if len(got) != tt.want {
				t.Errorf("ApplyFiltersToBashCommands() returned %d commands, want %d", len(got), tt.want)
			}
		})
	}
}

func TestApplyFiltersToFileOperations(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	files := []FileOperation{
		{Type: "read", Path: "/src/main.go", Success: true, Timestamp: now},
		{Type: "write", Path: "/src/test.go", Success: false, Timestamp: yesterday},
		{Type: "edit", Path: "/docs/README.md", Success: true, Timestamp: now},
		{Type: "delete", Path: "/tmp/temp.txt", Success: false, Timestamp: yesterday},
	}

	tests := []struct {
		name     string
		files    []FileOperation
		criteria FilterCriteria
		want     int
	}{
		{
			name:     "no filter returns all",
			files:    files,
			criteria: FilterCriteria{},
			want:     4,
		},
		{
			name:     "search by path",
			files:    files,
			criteria: FilterCriteria{Search: "main"},
			want:     1,
		},
		{
			name:     "search by type",
			files:    files,
			criteria: FilterCriteria{Search: "read"},
			want:     2, // Matches both "read" type and "README.md" path
		},
		{
			name:     "errors only",
			files:    files,
			criteria: FilterCriteria{ErrorsOnly: true},
			want:     2,
		},
		{
			name:     "event type file",
			files:    files,
			criteria: FilterCriteria{EventType: "file"},
			want:     4,
		},
		{
			name:     "event type bash filters out all",
			files:    files,
			criteria: FilterCriteria{EventType: "bash"},
			want:     0,
		},
		{
			name:     "time range",
			files:    files,
			criteria: FilterCriteria{Since: yesterday.Add(-1 * time.Hour)},
			want:     4,
		},
		{
			name:     "combined filters",
			files:    files,
			criteria: FilterCriteria{Search: "src", ErrorsOnly: true},
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyFiltersToFileOperations(tt.files, tt.criteria)
			if len(got) != tt.want {
				t.Errorf("ApplyFiltersToFileOperations() returned %d files, want %d", len(got), tt.want)
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		timeRange string
		wantErr   bool
		validate  func(time.Time) bool
	}{
		{
			name:      "empty string returns zero time",
			timeRange: "",
			wantErr:   false,
			validate: func(t time.Time) bool {
				return t.IsZero()
			},
		},
		{
			name:      "today keyword",
			timeRange: "today",
			wantErr:   false,
			validate: func(t time.Time) bool {
				today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				return t.Equal(today)
			},
		},
		{
			name:      "yesterday keyword",
			timeRange: "yesterday",
			wantErr:   false,
			validate: func(t time.Time) bool {
				yesterday := now.AddDate(0, 0, -1)
				expectedYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
				return t.Equal(expectedYesterday)
			},
		},
		{
			name:      "1 hour ago",
			timeRange: "1h",
			wantErr:   false,
			validate: func(t time.Time) bool {
				diff := now.Sub(t)
				return diff >= 55*time.Minute && diff <= 65*time.Minute
			},
		},
		{
			name:      "24 hours ago",
			timeRange: "24h",
			wantErr:   false,
			validate: func(t time.Time) bool {
				diff := now.Sub(t)
				return diff >= 23*time.Hour && diff <= 25*time.Hour
			},
		},
		{
			name:      "7 days ago",
			timeRange: "7d",
			wantErr:   false,
			validate: func(t time.Time) bool {
				expected := now.AddDate(0, 0, -7)
				diff := expected.Sub(t)
				return diff >= -1*time.Hour && diff <= 1*time.Hour
			},
		},
		{
			name:      "30 days ago",
			timeRange: "30d",
			wantErr:   false,
			validate: func(t time.Time) bool {
				expected := now.AddDate(0, 0, -30)
				diff := expected.Sub(t)
				return diff >= -1*time.Hour && diff <= 1*time.Hour
			},
		},
		{
			name:      "ISO date",
			timeRange: "2025-01-15",
			wantErr:   false,
			validate: func(t time.Time) bool {
				expected, _ := time.Parse("2006-01-02", "2025-01-15")
				return t.Equal(expected)
			},
		},
		{
			name:      "ISO datetime",
			timeRange: "2025-01-15T14:30:00",
			wantErr:   false,
			validate: func(t time.Time) bool {
				expected, _ := time.Parse("2006-01-02T15:04:05", "2025-01-15T14:30:00")
				return t.Equal(expected)
			},
		},
		{
			name:      "RFC3339",
			timeRange: "2025-01-15T14:30:00Z",
			wantErr:   false,
			validate: func(t time.Time) bool {
				expected, _ := time.Parse(time.RFC3339, "2025-01-15T14:30:00Z")
				return t.Equal(expected)
			},
		},
		{
			name:      "invalid format",
			timeRange: "invalid",
			wantErr:   true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimeRange(tt.timeRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimeRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil && !tt.validate(got) {
				t.Errorf("ParseTimeRange() = %v, validation failed", got)
			}
		})
	}
}

func TestValidateTimeRange(t *testing.T) {
	tests := []struct {
		name      string
		timeRange string
		wantErr   bool
	}{
		{
			name:      "empty is valid",
			timeRange: "",
			wantErr:   false,
		},
		{
			name:      "today is valid",
			timeRange: "today",
			wantErr:   false,
		},
		{
			name:      "1h is valid",
			timeRange: "1h",
			wantErr:   false,
		},
		{
			name:      "ISO date is valid",
			timeRange: "2025-01-15",
			wantErr:   false,
		},
		{
			name:      "invalid format",
			timeRange: "bad-format",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeRange(tt.timeRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
