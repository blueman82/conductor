package cmd

import (
	"testing"

	"github.com/harrison/conductor/internal/learning"
)

func TestFormatFileAnalysisTable(t *testing.T) {
	tests := []struct {
		name  string
		files []learning.FileStats
		limit int
		verify func(string) bool
	}{
		{
			name:  "empty files",
			files: []learning.FileStats{},
			limit: 20,
			verify: func(s string) bool {
				return contains(s, "No file operation data available")
			},
		},
		{
			name: "single file operation",
			files: []learning.FileStats{
				{
					FilePath:      "/path/to/file.go",
					OperationType: "read",
					OpCount:       100,
					SuccessCount:  100,
					FailureCount:  0,
					AvgDurationMs: 25.0,
					TotalBytes:    50000,
					SuccessRate:   1.0,
				},
			},
			limit: 20,
			verify: func(s string) bool {
				return contains(s, "/path/to/file.go") &&
					contains(s, "read") &&
					contains(s, "File Operation Analysis")
			},
		},
		{
			name: "multiple file operations with limit",
			files: []learning.FileStats{
				{
					FilePath:      "/src/main.go",
					OperationType: "read",
					OpCount:       200,
					SuccessCount:  200,
					FailureCount:  0,
					AvgDurationMs: 30.0,
					TotalBytes:    100000,
					SuccessRate:   1.0,
				},
				{
					FilePath:      "/src/main.go",
					OperationType: "write",
					OpCount:       50,
					SuccessCount:  48,
					FailureCount:  2,
					AvgDurationMs: 80.0,
					TotalBytes:    25000,
					SuccessRate:   0.96,
				},
				{
					FilePath:      "/config/config.yaml",
					OperationType: "read",
					OpCount:       30,
					SuccessCount:  30,
					FailureCount:  0,
					AvgDurationMs: 15.0,
					TotalBytes:    5000,
					SuccessRate:   1.0,
				},
			},
			limit: 2,
			verify: func(s string) bool {
				return contains(s, "Showing 2 of 3 file operations")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileAnalysisTable(tt.files, tt.limit)
			if !tt.verify(result) {
				t.Errorf("formatFileAnalysisTable output verification failed\nGot:\n%s", result)
			}
		})
	}
}

func TestDisplayFileAnalysis_MissingDatabase(t *testing.T) {
	// This test verifies that the function properly handles a missing database
	// In actual usage, this would fail due to missing config, which is expected
	err := DisplayFileAnalysis("", 20)
	if err == nil {
		t.Error("Expected error for missing database, got nil")
	}
}

func TestFileStatsCalculations(t *testing.T) {
	tests := []struct {
		name          string
		opCount       int
		successCount  int
		expectedRate  float64
	}{
		{
			name:          "100% success",
			opCount:       50,
			successCount:  50,
			expectedRate:  1.0,
		},
		{
			name:          "90% success",
			opCount:       10,
			successCount:  9,
			expectedRate:  0.9,
		},
		{
			name:          "0% success",
			opCount:       5,
			successCount:  0,
			expectedRate:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := learning.FileStats{
				FilePath:      "test.txt",
				OperationType: "read",
				OpCount:       tt.opCount,
				SuccessCount:  tt.successCount,
				FailureCount:  tt.opCount - tt.successCount,
			}

			if fs.OpCount > 0 {
				fs.SuccessRate = float64(fs.SuccessCount) / float64(fs.OpCount)
			}

			if fs.SuccessRate != tt.expectedRate {
				t.Errorf("Expected success rate %.2f, got %.2f", tt.expectedRate, fs.SuccessRate)
			}
		})
	}
}
