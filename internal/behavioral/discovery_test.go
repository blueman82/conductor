package behavioral

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsValidSessionFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "valid session file",
			filename: "agent-12345678-1234-1234-1234-123456789abc.jsonl",
			want:     true,
		},
		{
			name:     "valid session file uppercase UUID",
			filename: "agent-ABCDEF12-ABCD-ABCD-ABCD-ABCDEFABCDEF.jsonl",
			want:     true,
		},
		{
			name:     "invalid prefix",
			filename: "session-12345678-1234-1234-1234-123456789abc.jsonl",
			want:     false,
		},
		{
			name:     "invalid UUID format",
			filename: "agent-12345678-1234-1234-123456789abc.jsonl",
			want:     false,
		},
		{
			name:     "wrong extension",
			filename: "agent-12345678-1234-1234-1234-123456789abc.json",
			want:     false,
		},
		{
			name:     "no extension",
			filename: "agent-12345678-1234-1234-1234-123456789abc",
			want:     false,
		},
		{
			name:     "empty string",
			filename: "",
			want:     false,
		},
		{
			name:     "random file",
			filename: "README.md",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSessionFile(tt.filename)
			if got != tt.want {
				t.Errorf("isValidSessionFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{
			name:     "valid lowercase UUID",
			filename: "agent-12345678-1234-1234-1234-123456789abc.jsonl",
			want:     "12345678-1234-1234-1234-123456789abc",
			wantErr:  false,
		},
		{
			name:     "valid uppercase UUID",
			filename: "agent-ABCDEF12-ABCD-ABCD-ABCD-ABCDEFABCDEF.jsonl",
			want:     "ABCDEF12-ABCD-ABCD-ABCD-ABCDEFABCDEF",
			wantErr:  false,
		},
		{
			name:     "invalid format",
			filename: "invalid.jsonl",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "wrong prefix",
			filename: "session-12345678-1234-1234-1234-123456789abc.jsonl",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractSessionID(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractSessionID(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractSessionID(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExpandHomeDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "tilde only",
			path:    "~",
			want:    homeDir,
			wantErr: false,
		},
		{
			name:    "tilde with path",
			path:    "~/.claude/projects",
			want:    filepath.Join(homeDir, ".claude/projects"),
			wantErr: false,
		},
		{
			name:    "absolute path",
			path:    "/absolute/path",
			want:    "/absolute/path",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			want:    "relative/path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandHomeDir(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandHomeDir(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("expandHomeDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseSessionPath(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	filename := "agent-12345678-1234-1234-1234-123456789abc.jsonl"
	testFile := filepath.Join(projectDir, filename)

	// Create test file
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get file info
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Parse session path
	sessionInfo, err := parseSessionPath(testFile, info)
	if err != nil {
		t.Fatalf("parseSessionPath() error = %v", err)
	}

	// Validate results
	if sessionInfo.Project != "test-project" {
		t.Errorf("Project = %v, want test-project", sessionInfo.Project)
	}
	if sessionInfo.Filename != filename {
		t.Errorf("Filename = %v, want %v", sessionInfo.Filename, filename)
	}
	if sessionInfo.SessionID != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("SessionID = %v, want 12345678-1234-1234-1234-123456789abc", sessionInfo.SessionID)
	}
	if sessionInfo.FileSize != 4 {
		t.Errorf("FileSize = %v, want 4", sessionInfo.FileSize)
	}
	if sessionInfo.FilePath == "" {
		t.Error("FilePath should not be empty")
	}
	if sessionInfo.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestDiscoverProjectSessions(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test session files
	sessions := []string{
		"agent-11111111-1111-1111-1111-111111111111.jsonl",
		"agent-22222222-2222-2222-2222-222222222222.jsonl",
		"agent-33333333-3333-3333-3333-333333333333.jsonl",
	}

	for i, session := range sessions {
		testFile := filepath.Join(projectDir, session)
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Set different mod times for sorting test
		modTime := time.Now().Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(testFile, modTime, modTime); err != nil {
			t.Fatalf("Failed to set mod time: %v", err)
		}
	}

	// Create invalid files (should be ignored)
	invalidFiles := []string{
		"README.md",
		"invalid.jsonl",
		"session-12345678-1234-1234-1234-123456789abc.jsonl",
	}
	for _, invalid := range invalidFiles {
		testFile := filepath.Join(projectDir, invalid)
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create invalid file: %v", err)
		}
	}

	// Discover sessions
	discovered, err := DiscoverProjectSessions(tmpDir, "test-project")
	if err != nil {
		t.Fatalf("DiscoverProjectSessions() error = %v", err)
	}

	// Validate count (only valid session files)
	if len(discovered) != len(sessions) {
		t.Errorf("Discovered %d sessions, want %d", len(discovered), len(sessions))
	}

	// Validate sorting (newest first)
	if len(discovered) > 1 {
		for i := 0; i < len(discovered)-1; i++ {
			if discovered[i].CreatedAt.Before(discovered[i+1].CreatedAt) {
				t.Error("Sessions not sorted by creation time (newest first)")
			}
		}
	}

	// Validate fields
	for _, session := range discovered {
		if session.Project != "test-project" {
			t.Errorf("Session project = %v, want test-project", session.Project)
		}
		if session.SessionID == "" {
			t.Error("Session ID should not be empty")
		}
		if session.FilePath == "" {
			t.Error("File path should not be empty")
		}
	}
}

func TestDiscoverProjectSessions_NonexistentProject(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := DiscoverProjectSessions(tmpDir, "nonexistent-project")
	if err == nil {
		t.Error("Expected error for nonexistent project, got nil")
	}
}

func TestDiscoverSessions(t *testing.T) {
	// Create temp directory structure with multiple projects
	tmpDir := t.TempDir()

	projects := []string{"project1", "project2"}
	expectedSessions := 0

	for _, project := range projects {
		projectDir := filepath.Join(tmpDir, project)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		// Create 2 session files per project
		for i := 1; i <= 2; i++ {
			validFilename := filepath.Join(projectDir,
				"agent-12345678-1234-1234-1234-12345678900"+string(rune('0'+i))+".jsonl")
			if err := os.WriteFile(validFilename, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			expectedSessions++
		}
	}

	// Discover all sessions
	discovered, err := DiscoverSessions(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}

	// Validate count
	if len(discovered) != expectedSessions {
		t.Errorf("Discovered %d sessions, want %d", len(discovered), expectedSessions)
	}

	// Validate sorting (newest first)
	if len(discovered) > 1 {
		for i := 0; i < len(discovered)-1; i++ {
			if discovered[i].CreatedAt.Before(discovered[i+1].CreatedAt) {
				t.Error("Sessions not sorted by creation time (newest first)")
			}
		}
	}
}

func TestDiscoverSessions_NonexistentDirectory(t *testing.T) {
	_, err := DiscoverSessions("/nonexistent/directory/path")
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestDiscoverSessions_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	discovered, err := DiscoverSessions(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSessions() error = %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("Expected 0 sessions in empty directory, got %d", len(discovered))
	}
}
