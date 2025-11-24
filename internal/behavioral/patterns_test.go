package behavioral

import (
	"testing"
	"time"
)

func TestNewPatternDetector(t *testing.T) {
	sessions := []Session{
		{ID: "1", Success: true, Duration: 1000},
		{ID: "2", Success: false, Duration: 2000},
	}
	metrics := []BehavioralMetrics{
		{TotalSessions: 2},
	}

	detector := NewPatternDetector(sessions, metrics)

	if detector == nil {
		t.Fatal("Expected non-nil detector")
	}
	if len(detector.sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(detector.sessions))
	}
	if len(detector.metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(detector.metrics))
	}
}

func TestDetectToolSequences(t *testing.T) {
	tests := []struct {
		name           string
		metrics        []BehavioralMetrics
		sessions       []Session
		minFrequency   int
		sequenceLength int
		expectedMin    int
	}{
		{
			name: "basic sequence detection",
			metrics: []BehavioralMetrics{
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read"},
						{Name: "Write"},
						{Name: "Bash"},
					},
				},
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read"},
						{Name: "Write"},
						{Name: "Bash"},
					},
				},
			},
			sessions: []Session{
				{ID: "1", Success: true, Duration: 1000},
				{ID: "2", Success: true, Duration: 1000},
			},
			minFrequency:   2,
			sequenceLength: 2,
			expectedMin:    1,
		},
		{
			name: "no sequences meet threshold",
			metrics: []BehavioralMetrics{
				{
					ToolExecutions: []ToolExecution{
						{Name: "Read"},
					},
				},
			},
			sessions:       []Session{{ID: "1"}},
			minFrequency:   5,
			sequenceLength: 2,
			expectedMin:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewPatternDetector(tt.sessions, tt.metrics)
			sequences := detector.DetectToolSequences(tt.minFrequency, tt.sequenceLength)

			if len(sequences) < tt.expectedMin {
				t.Errorf("Expected at least %d sequences, got %d", tt.expectedMin, len(sequences))
			}

			// Verify sequences are sorted by frequency
			for i := 1; i < len(sequences); i++ {
				if sequences[i].Frequency > sequences[i-1].Frequency {
					t.Error("Sequences not sorted by frequency descending")
				}
			}
		})
	}
}

func TestDetectBashPatterns(t *testing.T) {
	tests := []struct {
		name         string
		metrics      []BehavioralMetrics
		minFrequency int
		expectedMin  int
	}{
		{
			name: "common git commands",
			metrics: []BehavioralMetrics{
				{
					BashCommands: []BashCommand{
						{Command: "git status", Success: true, Duration: 100 * time.Millisecond},
						{Command: "git commit", Success: true, Duration: 200 * time.Millisecond},
					},
				},
				{
					BashCommands: []BashCommand{
						{Command: "git status", Success: true, Duration: 100 * time.Millisecond},
						{Command: "git push", Success: true, Duration: 300 * time.Millisecond},
					},
				},
			},
			minFrequency: 2,
			expectedMin:  1,
		},
		{
			name: "mixed commands",
			metrics: []BehavioralMetrics{
				{
					BashCommands: []BashCommand{
						{Command: "go test", Success: true, Duration: 1000 * time.Millisecond},
						{Command: "go build", Success: false, Duration: 500 * time.Millisecond},
						{Command: "ls -la", Success: true, Duration: 10 * time.Millisecond},
					},
				},
			},
			minFrequency: 1,
			expectedMin:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewPatternDetector([]Session{}, tt.metrics)
			patterns := detector.DetectBashPatterns(tt.minFrequency)

			if len(patterns) < tt.expectedMin {
				t.Errorf("Expected at least %d patterns, got %d", tt.expectedMin, len(patterns))
			}

			// Verify patterns are sorted by frequency
			for i := 1; i < len(patterns); i++ {
				if patterns[i].Frequency > patterns[i-1].Frequency {
					t.Error("Patterns not sorted by frequency descending")
				}
			}

			// Verify success rates are valid
			for _, pattern := range patterns {
				if pattern.SuccessRate < 0 || pattern.SuccessRate > 1 {
					t.Errorf("Invalid success rate: %f", pattern.SuccessRate)
				}
			}
		})
	}
}

func TestIdentifyAnomalies(t *testing.T) {
	tests := []struct {
		name            string
		sessions        []Session
		stdDevThreshold float64
		expectedMin     int
	}{
		{
			name: "duration anomaly",
			sessions: []Session{
				{ID: "1", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "2", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "3", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "4", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "5", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "6", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "7", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "8", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "9", Duration: 50000, ErrorCount: 0, Success: true, Timestamp: time.Now()}, // Anomaly
			},
			stdDevThreshold: 2.0,
			expectedMin:     1,
		},
		{
			name: "error rate anomaly",
			sessions: []Session{
				{ID: "1", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "2", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "3", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "4", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "5", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "6", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "7", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "8", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "9", Duration: 1000, ErrorCount: 50, Success: false, Timestamp: time.Now()}, // Anomaly
			},
			stdDevThreshold: 2.0,
			expectedMin:     1,
		},
		{
			name: "no anomalies",
			sessions: []Session{
				{ID: "1", Duration: 1000, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "2", Duration: 1100, ErrorCount: 0, Success: true, Timestamp: time.Now()},
				{ID: "3", Duration: 900, ErrorCount: 0, Success: true, Timestamp: time.Now()},
			},
			stdDevThreshold: 3.0,
			expectedMin:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewPatternDetector(tt.sessions, []BehavioralMetrics{})
			anomalies := detector.IdentifyAnomalies(tt.stdDevThreshold)

			if len(anomalies) < tt.expectedMin {
				t.Errorf("Expected at least %d anomalies, got %d", tt.expectedMin, len(anomalies))
			}

			// Verify anomalies have required fields
			for _, anomaly := range anomalies {
				if anomaly.Type == "" {
					t.Error("Anomaly missing type")
				}
				if anomaly.Description == "" {
					t.Error("Anomaly missing description")
				}
				if anomaly.Severity == "" {
					t.Error("Anomaly missing severity")
				}
				if anomaly.SessionID == "" {
					t.Error("Anomaly missing session ID")
				}
				if anomaly.Deviation < 0 {
					t.Error("Anomaly deviation should be positive")
				}
			}
		})
	}
}

func TestClusterSessions(t *testing.T) {
	tests := []struct {
		name        string
		sessions    []Session
		numClusters int
		expectedLen int
	}{
		{
			name: "basic clustering",
			sessions: []Session{
				{ID: "1", Duration: 1000, ErrorCount: 0, Success: true},
				{ID: "2", Duration: 1100, ErrorCount: 0, Success: true},
				{ID: "3", Duration: 5000, ErrorCount: 5, Success: false},
			},
			numClusters: 2,
			expectedLen: 2,
		},
		{
			name: "single cluster",
			sessions: []Session{
				{ID: "1", Duration: 1000, ErrorCount: 0, Success: true},
			},
			numClusters: 1,
			expectedLen: 1,
		},
		{
			name:        "empty sessions",
			sessions:    []Session{},
			numClusters: 3,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewPatternDetector(tt.sessions, []BehavioralMetrics{})
			clusters := detector.ClusterSessions(tt.numClusters)

			if len(clusters) != tt.expectedLen {
				t.Errorf("Expected %d clusters, got %d", tt.expectedLen, len(clusters))
			}

			// Verify cluster properties
			totalSessions := 0
			for _, cluster := range clusters {
				if cluster.Description == "" {
					t.Error("Cluster missing description")
				}
				if len(cluster.SessionIDs) != cluster.Size {
					t.Errorf("Cluster size mismatch: %d vs %d", len(cluster.SessionIDs), cluster.Size)
				}
				totalSessions += cluster.Size
			}

			// Verify all sessions are assigned
			if len(tt.sessions) > 0 && totalSessions != len(tt.sessions) {
				t.Errorf("Expected %d total sessions in clusters, got %d", len(tt.sessions), totalSessions)
			}
		})
	}
}

func TestMakeSequenceKey(t *testing.T) {
	tests := []struct {
		name     string
		tools    []string
		expected string
	}{
		{
			name:     "two tools",
			tools:    []string{"Read", "Write"},
			expected: "Read->Write",
		},
		{
			name:     "three tools",
			tools:    []string{"Read", "Edit", "Bash"},
			expected: "Read->Edit->Bash",
		},
		{
			name:     "single tool",
			tools:    []string{"Read"},
			expected: "Read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeSequenceKey(tt.tools)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExtractCommandPattern(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "git command",
			command:  "git status",
			expected: "git",
		},
		{
			name:     "go command",
			command:  "go test ./...",
			expected: "go",
		},
		{
			name:     "single word",
			command:  "ls",
			expected: "ls",
		},
		{
			name:     "with flags",
			command:  "ls -la /tmp",
			expected: "ls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommandPattern(tt.command)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetSeverity(t *testing.T) {
	tests := []struct {
		name      string
		deviation float64
		expected  string
	}{
		{
			name:      "high severity",
			deviation: 3.5,
			expected:  "high",
		},
		{
			name:      "medium severity",
			deviation: 2.5,
			expected:  "medium",
		},
		{
			name:      "low severity",
			deviation: 1.5,
			expected:  "low",
		},
		{
			name:      "boundary high",
			deviation: 3.0,
			expected:  "high",
		},
		{
			name:      "boundary medium",
			deviation: 2.0,
			expected:  "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSeverity(tt.deviation)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExtractFeatures(t *testing.T) {
	tests := []struct {
		name     string
		session  Session
		expected []float64
	}{
		{
			name: "successful session",
			session: Session{
				Duration:   5000,
				ErrorCount: 2,
				Success:    true,
			},
			expected: []float64{5.0, 2.0, 1.0},
		},
		{
			name: "failed session",
			session: Session{
				Duration:   3000,
				ErrorCount: 10,
				Success:    false,
			},
			expected: []float64{3.0, 10.0, 0.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFeatures(tt.session)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d features, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Feature %d: expected %f, got %f", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 0.0,
		},
		{
			name:     "different vectors",
			a:        []float64{0.0, 0.0},
			b:        []float64{3.0, 4.0},
			expected: 5.0,
		},
		{
			name:     "mismatched lengths",
			a:        []float64{1.0, 2.0},
			b:        []float64{1.0},
			expected: 1.7976931348623157e+308, // math.MaxFloat64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := euclideanDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestGenerateClusterDescription(t *testing.T) {
	tests := []struct {
		name     string
		centroid []float64
		contains string
	}{
		{
			name:     "fast successful",
			centroid: []float64{30.0, 0.5, 0.9},
			contains: "Fast",
		},
		{
			name:     "slow successful",
			centroid: []float64{120.0, 0.5, 0.9},
			contains: "Slow",
		},
		{
			name:     "high error",
			centroid: []float64{60.0, 5.0, 0.5},
			contains: "error",
		},
		{
			name:     "failed sessions",
			centroid: []float64{60.0, 1.0, 0.1},
			contains: "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &BehaviorCluster{}
			result := generateClusterDescription(cluster, tt.centroid)
			if result == "" {
				t.Error("Expected non-empty description")
			}
		})
	}
}

func TestCalculateBaselineStats(t *testing.T) {
	sessions := []Session{
		{Duration: 1000, ErrorCount: 0, Success: true},
		{Duration: 2000, ErrorCount: 1, Success: true},
		{Duration: 1500, ErrorCount: 0, Success: false},
	}

	detector := NewPatternDetector(sessions, []BehavioralMetrics{})
	stats := detector.calculateBaselineStats()

	if stats.avgDuration <= 0 {
		t.Error("Expected positive average duration")
	}
	if stats.stdDuration < 0 {
		t.Error("Expected non-negative standard deviation")
	}
	if stats.successRate < 0 || stats.successRate > 1 {
		t.Error("Success rate should be between 0 and 1")
	}
}
