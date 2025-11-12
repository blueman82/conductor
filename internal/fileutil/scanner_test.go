package fileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanDirectory(t *testing.T) {
	// Create a temporary test directory structure
	tmpDir := t.TempDir()

	// Create test directory structure:
	// tmpDir/
	//   file1.md
	//   file2.yaml
	//   file3.txt
	//   plan-001.md
	//   plan-002.yaml
	//   Setup.MD (test case-insensitive)
	//   subdir1/
	//     nested1.md
	//     nested2.yaml
	//     subdir2/
	//       deep1.md
	//       deep2.txt
	//   .hidden/
	//     hidden.md
	//   node_modules/
	//     package.json
	//   excluded/
	//     excluded.md

	testFiles := []string{
		"file1.md",
		"file2.yaml",
		"file3.txt",
		"plan-001.md",
		"plan-002.yaml",
		"Setup.MD",
		"subdir1/nested1.md",
		"subdir1/nested2.yaml",
		"subdir1/subdir2/deep1.md",
		"subdir1/subdir2/deep2.txt",
		".hidden/hidden.md",
		"node_modules/package.json",
		"excluded/excluded.md",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	tests := []struct {
		name          string
		opts          ScanOptions
		wantFileNames []string // Just the base filenames for easier assertion
		wantErrorsLen int
	}{
		{
			name: "basic non-recursive scan",
			opts: ScanOptions{
				Recursive: false,
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml"},
		},
		{
			name: "basic recursive scan",
			opts: ScanOptions{
				Recursive: true,
			},
			wantFileNames: []string{
				"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml",
				"nested1.md", "nested2.yaml", "deep1.md", "deep2.txt",
				"excluded.md", "package.json", // These are NOT auto-excluded
			},
		},
		{
			name: "filter by single extension - md",
			opts: ScanOptions{
				Extensions: []string{".md"},
				Recursive:  true,
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "plan-001.md", "nested1.md", "deep1.md", "excluded.md"},
		},
		{
			name: "filter by multiple extensions",
			opts: ScanOptions{
				Extensions: []string{".md", ".yaml"},
				Recursive:  true,
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "file2.yaml", "plan-001.md", "plan-002.yaml", "nested1.md", "nested2.yaml", "deep1.md", "excluded.md"},
		},
		{
			name: "extension without dot prefix",
			opts: ScanOptions{
				Extensions: []string{"md", "yaml"},
				Recursive:  true,
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "file2.yaml", "plan-001.md", "plan-002.yaml", "nested1.md", "nested2.yaml", "deep1.md", "excluded.md"},
		},
		{
			name: "case-insensitive extension matching",
			opts: ScanOptions{
				Extensions: []string{".MD"},
				Recursive:  false,
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "plan-001.md"},
		},
		{
			name: "pattern matching - plan prefix",
			opts: ScanOptions{
				Pattern:   "^plan-",
				Recursive: true,
			},
			wantFileNames: []string{"plan-001.md", "plan-002.yaml"},
		},
		{
			name: "pattern matching - digits only",
			opts: ScanOptions{
				Pattern:   "^\\d+$",
				Recursive: true,
			},
			wantFileNames: []string{},
		},
		{
			name: "pattern matching - file1 or file2",
			opts: ScanOptions{
				Pattern:   "^file[12]$",
				Recursive: false,
			},
			wantFileNames: []string{"file1.md", "file2.yaml"},
		},
		{
			name: "pattern with extension filter",
			opts: ScanOptions{
				Pattern:    "^plan-",
				Extensions: []string{".md"},
				Recursive:  true,
			},
			wantFileNames: []string{"plan-001.md"},
		},
		{
			name: "maxDepth 1 - current directory only",
			opts: ScanOptions{
				Recursive: true,
				MaxDepth:  1,
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml"},
		},
		{
			name: "maxDepth 2 - one level deep",
			opts: ScanOptions{
				Recursive: true,
				MaxDepth:  2,
			},
			wantFileNames: []string{
				"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml",
				"nested1.md", "nested2.yaml", "excluded.md", "package.json",
			},
		},
		{
			name: "maxDepth 0 - unlimited",
			opts: ScanOptions{
				Recursive: true,
				MaxDepth:  0,
			},
			wantFileNames: []string{
				"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml",
				"nested1.md", "nested2.yaml", "deep1.md", "deep2.txt",
				"excluded.md", "package.json",
			},
		},
		{
			name: "exclude single directory",
			opts: ScanOptions{
				Recursive:   true,
				ExcludeDirs: []string{"subdir1"},
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml", "excluded.md", "package.json"},
		},
		{
			name: "exclude multiple directories",
			opts: ScanOptions{
				Recursive:   true,
				ExcludeDirs: []string{"subdir1", "excluded"},
			},
			wantFileNames: []string{"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml", "package.json"},
		},
		{
			name: "auto-exclude hidden directories",
			opts: ScanOptions{
				Recursive: true,
			},
			wantFileNames: []string{
				"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml",
				"nested1.md", "nested2.yaml", "deep1.md", "deep2.txt",
				"excluded.md", "package.json", // .hidden/ is excluded automatically
			},
		},
		{
			name: "exclude node_modules",
			opts: ScanOptions{
				Recursive:   true,
				ExcludeDirs: []string{"node_modules"},
			},
			wantFileNames: []string{
				"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml",
				"nested1.md", "nested2.yaml", "deep1.md", "deep2.txt", "excluded.md",
			},
		},
		{
			name: "no matches",
			opts: ScanOptions{
				Pattern:   "^nonexistent$",
				Recursive: true,
			},
			wantFileNames: []string{},
		},
		{
			name: "all extensions with pattern",
			opts: ScanOptions{
				Pattern:   ".*", // Match all
				Recursive: true,
			},
			wantFileNames: []string{
				"Setup.MD", "file1.md", "file2.yaml", "file3.txt", "plan-001.md", "plan-002.yaml",
				"nested1.md", "nested2.yaml", "deep1.md", "deep2.txt",
				"excluded.md", "package.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ScanDirectory(tmpDir, tt.opts)
			if err != nil {
				t.Fatalf("ScanDirectory() error = %v", err)
			}

			if result == nil {
				t.Fatal("ScanDirectory() returned nil result")
			}

			// Check error count
			if len(result.Errors) != tt.wantErrorsLen {
				t.Errorf("ScanDirectory() errors count = %d, want %d", len(result.Errors), tt.wantErrorsLen)
				for _, e := range result.Errors {
					t.Logf("  error: %v", e)
				}
			}

			// Extract basenames from result
			gotFileNames := make([]string, len(result.Files))
			for i, path := range result.Files {
				gotFileNames[i] = filepath.Base(path)
			}

			// Check that we got the expected files
			if len(gotFileNames) != len(tt.wantFileNames) {
				t.Errorf("ScanDirectory() file count = %d, want %d", len(gotFileNames), len(tt.wantFileNames))
				t.Logf("got: %v", gotFileNames)
				t.Logf("want: %v", tt.wantFileNames)
				return
			}

			// Create maps for easier comparison
			gotMap := make(map[string]bool)
			for _, name := range gotFileNames {
				gotMap[name] = true
			}

			wantMap := make(map[string]bool)
			for _, name := range tt.wantFileNames {
				wantMap[name] = true
			}

			// Check each expected file is present
			for _, want := range tt.wantFileNames {
				if !gotMap[want] {
					t.Errorf("ScanDirectory() missing expected file: %s", want)
				}
			}

			// Check no unexpected files
			for _, got := range gotFileNames {
				if !wantMap[got] {
					t.Errorf("ScanDirectory() unexpected file: %s", got)
				}
			}
		})
	}
}

func TestScanDirectory_AbsolutePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result, err := ScanDirectory(tmpDir, ScanOptions{
		Recursive: false,
	})
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}

	// Verify the path is absolute
	if !filepath.IsAbs(result.Files[0]) {
		t.Errorf("ScanDirectory() returned relative path: %s", result.Files[0])
	}

	// Verify the file actually exists at that path
	if _, err := os.Stat(result.Files[0]); err != nil {
		t.Errorf("file at returned path does not exist: %v", err)
	}
}

func TestScanDirectory_SortedOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in non-alphabetical order
	files := []string{"zebra.md", "apple.md", "mango.md", "banana.md"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	result, err := ScanDirectory(tmpDir, ScanOptions{
		Recursive: false,
	})
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	// Extract basenames
	gotNames := make([]string, len(result.Files))
	for i, path := range result.Files {
		gotNames[i] = filepath.Base(path)
	}

	// Expected sorted order
	wantNames := []string{"apple.md", "banana.md", "mango.md", "zebra.md"}

	if len(gotNames) != len(wantNames) {
		t.Fatalf("expected %d files, got %d", len(wantNames), len(gotNames))
	}

	for i, want := range wantNames {
		if gotNames[i] != want {
			t.Errorf("files[%d] = %s, want %s", i, gotNames[i], want)
		}
	}
}

func TestScanDirectory_Errors(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() (string, ScanOptions)
		wantErr   string
	}{
		{
			name: "non-existent directory",
			setupFunc: func() (string, ScanOptions) {
				return "/nonexistent/directory/path", ScanOptions{Recursive: false}
			},
			wantErr: "failed to access directory",
		},
		{
			name: "path is a file not directory",
			setupFunc: func() (string, ScanOptions) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "file.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return filePath, ScanOptions{Recursive: false}
			},
			wantErr: "path is not a directory",
		},
		{
			name: "invalid regex pattern",
			setupFunc: func() (string, ScanOptions) {
				tmpDir := t.TempDir()
				return tmpDir, ScanOptions{
					Pattern: "[invalid(regex",
				}
			},
			wantErr: "invalid pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, opts := tt.setupFunc()
			result, err := ScanDirectory(dir, opts)

			if err == nil {
				t.Fatalf("ScanDirectory() expected error containing %q, got nil", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ScanDirectory() error = %v, want error containing %q", err, tt.wantErr)
			}

			if result != nil {
				t.Errorf("ScanDirectory() expected nil result on error, got %+v", result)
			}
		})
	}
}

func TestScanDirectory_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := ScanDirectory(tmpDir, ScanOptions{
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("ScanDirectory() on empty dir returned %d files, want 0", len(result.Files))
	}

	if len(result.Errors) != 0 {
		t.Errorf("ScanDirectory() on empty dir returned %d errors, want 0", len(result.Errors))
	}
}

func TestScanDirectory_NoMatchingExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different extensions
	files := []string{"file1.md", "file2.yaml", "file3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	result, err := ScanDirectory(tmpDir, ScanOptions{
		Extensions: []string{".json", ".xml"},
		Recursive:  false,
	})
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("ScanDirectory() with non-matching extensions returned %d files, want 0", len(result.Files))
	}
}

func TestScanDirectory_ComplexPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with various patterns
	files := []string{
		"task-01-setup.md",
		"task-02-implement.md",
		"task-03-test.md",
		"readme.md",
		"notes.md",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	tests := []struct {
		name          string
		pattern       string
		wantFileNames []string
	}{
		{
			name:          "match task-* pattern",
			pattern:       "^task-\\d{2}-.*",
			wantFileNames: []string{"task-01-setup.md", "task-02-implement.md", "task-03-test.md"},
		},
		{
			name:          "match files ending with notes or readme",
			pattern:       "(readme|notes)$",
			wantFileNames: []string{"notes.md", "readme.md"}, // Sorted alphabetically
		},
		{
			name:          "match files containing 'test'",
			pattern:       ".*test.*",
			wantFileNames: []string{"task-03-test.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ScanDirectory(tmpDir, ScanOptions{
				Pattern:   tt.pattern,
				Recursive: false,
			})
			if err != nil {
				t.Fatalf("ScanDirectory() error = %v", err)
			}

			gotNames := make([]string, len(result.Files))
			for i, path := range result.Files {
				gotNames[i] = filepath.Base(path)
			}

			if len(gotNames) != len(tt.wantFileNames) {
				t.Errorf("ScanDirectory() file count = %d, want %d", len(gotNames), len(tt.wantFileNames))
				t.Logf("got: %v", gotNames)
				t.Logf("want: %v", tt.wantFileNames)
				return
			}

			// Results are already sorted, so we can compare element by element
			for i, want := range tt.wantFileNames {
				if gotNames[i] != want {
					t.Errorf("files[%d] = %s, want %s", i, gotNames[i], want)
				}
			}
		})
	}
}

func TestScanDirectory_DeepNesting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested structure
	// tmpDir/
	//   level1/
	//     file1.md
	//     level2/
	//       file2.md
	//       level3/
	//         file3.md
	//         level4/
	//           file4.md

	deepPath := filepath.Join(tmpDir, "level1", "level2", "level3", "level4")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("failed to create deep directory: %v", err)
	}

	files := []struct {
		path string
		name string
	}{
		{filepath.Join(tmpDir, "level1"), "file1.md"},
		{filepath.Join(tmpDir, "level1", "level2"), "file2.md"},
		{filepath.Join(tmpDir, "level1", "level2", "level3"), "file3.md"},
		{filepath.Join(tmpDir, "level1", "level2", "level3", "level4"), "file4.md"},
	}

	for _, f := range files {
		path := filepath.Join(f.path, f.name)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	tests := []struct {
		name          string
		maxDepth      int
		wantFileNames []string
	}{
		{
			name:          "maxDepth 1 - only level1",
			maxDepth:      1,
			wantFileNames: []string{},
		},
		{
			name:          "maxDepth 2 - up to level1",
			maxDepth:      2,
			wantFileNames: []string{"file1.md"},
		},
		{
			name:          "maxDepth 3 - up to level2",
			maxDepth:      3,
			wantFileNames: []string{"file1.md", "file2.md"},
		},
		{
			name:          "maxDepth 4 - up to level3",
			maxDepth:      4,
			wantFileNames: []string{"file1.md", "file2.md", "file3.md"},
		},
		{
			name:          "maxDepth 0 - unlimited",
			maxDepth:      0,
			wantFileNames: []string{"file1.md", "file2.md", "file3.md", "file4.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ScanDirectory(tmpDir, ScanOptions{
				Recursive: true,
				MaxDepth:  tt.maxDepth,
			})
			if err != nil {
				t.Fatalf("ScanDirectory() error = %v", err)
			}

			gotNames := make([]string, len(result.Files))
			for i, path := range result.Files {
				gotNames[i] = filepath.Base(path)
			}

			if len(gotNames) != len(tt.wantFileNames) {
				t.Errorf("ScanDirectory() file count = %d, want %d", len(gotNames), len(tt.wantFileNames))
				t.Logf("got: %v", gotNames)
				t.Logf("want: %v", tt.wantFileNames)
				return
			}

			// Create maps for comparison
			gotMap := make(map[string]bool)
			for _, name := range gotNames {
				gotMap[name] = true
			}

			for _, want := range tt.wantFileNames {
				if !gotMap[want] {
					t.Errorf("ScanDirectory() missing expected file: %s", want)
				}
			}
		})
	}
}

func TestScanDirectory_MixedOptions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create comprehensive test structure
	testFiles := []string{
		"plan-001.md",
		"plan-002.yaml",
		"readme.md",
		"config.yaml",
		"notes.txt",
		"subdir/plan-003.md",
		"subdir/data.json",
		"excluded/plan-004.md",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	result, err := ScanDirectory(tmpDir, ScanOptions{
		Pattern:     "^plan-\\d+$",
		Extensions:  []string{".md", ".yaml"},
		Recursive:   true,
		ExcludeDirs: []string{"excluded"},
		MaxDepth:    0,
	})
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	wantFileNames := []string{"plan-001.md", "plan-002.yaml", "plan-003.md"}
	gotNames := make([]string, len(result.Files))
	for i, path := range result.Files {
		gotNames[i] = filepath.Base(path)
	}

	if len(gotNames) != len(wantFileNames) {
		t.Errorf("ScanDirectory() file count = %d, want %d", len(gotNames), len(wantFileNames))
		t.Logf("got: %v", gotNames)
		t.Logf("want: %v", wantFileNames)
		return
	}

	gotMap := make(map[string]bool)
	for _, name := range gotNames {
		gotMap[name] = true
	}

	for _, want := range wantFileNames {
		if !gotMap[want] {
			t.Errorf("ScanDirectory() missing expected file: %s", want)
		}
	}
}
