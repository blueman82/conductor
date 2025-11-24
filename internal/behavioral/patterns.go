package behavioral

import (
	"math"
	"sort"
	"time"
)

// PatternDetector analyzes behavioral patterns in agent sessions
type PatternDetector struct {
	sessions []Session
	metrics  []BehavioralMetrics
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(sessions []Session, metrics []BehavioralMetrics) *PatternDetector {
	return &PatternDetector{
		sessions: sessions,
		metrics:  metrics,
	}
}

// ToolSequence represents a common sequence of tool executions
type ToolSequence struct {
	Tools     []string  `json:"tools"`      // Ordered list of tools
	Frequency int       `json:"frequency"`  // How often this sequence appears
	AvgTime   time.Duration `json:"avg_time"` // Average time for sequence
	SuccessRate float64 `json:"success_rate"` // Success rate of sessions with this sequence
}

// CommandPattern represents a common bash command pattern
type CommandPattern struct {
	Pattern     string  `json:"pattern"`      // Command pattern (e.g., "git *", "go test *")
	Frequency   int     `json:"frequency"`    // How often this pattern appears
	SuccessRate float64 `json:"success_rate"` // Success rate of this command
	AvgDuration time.Duration `json:"avg_duration"` // Average execution time
}

// Anomaly represents a detected anomalous behavior
type Anomaly struct {
	Type        string    `json:"type"`         // Type: duration, error_rate, tool_usage, command_failure
	Description string    `json:"description"`  // Human-readable description
	Severity    string    `json:"severity"`     // low, medium, high
	SessionID   string    `json:"session_id"`   // Session where anomaly occurred
	Timestamp   time.Time `json:"timestamp"`    // When anomaly occurred
	Value       float64   `json:"value"`        // Anomalous value
	Expected    float64   `json:"expected"`     // Expected value
	Deviation   float64   `json:"deviation"`    // Standard deviations from mean
}

// BehaviorCluster represents a group of similar sessions
type BehaviorCluster struct {
	ClusterID   int       `json:"cluster_id"`    // Cluster identifier
	SessionIDs  []string  `json:"session_ids"`   // Sessions in this cluster
	Centroid    []float64 `json:"centroid"`      // Cluster center (feature vector)
	Description string    `json:"description"`   // Human-readable cluster description
	Size        int       `json:"size"`          // Number of sessions in cluster
}

// PatternEvolution tracks how a pattern changes over time
type PatternEvolution struct {
	Pattern    string              `json:"pattern"`     // Pattern identifier
	Snapshots  []PatternSnapshot   `json:"snapshots"`   // Time-based snapshots
	Trend      string              `json:"trend"`       // increasing, decreasing, stable
}

// PatternSnapshot represents a pattern's metrics at a point in time
type PatternSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`  // When snapshot was taken
	Frequency  int       `json:"frequency"`  // Pattern frequency
	SuccessRate float64  `json:"success_rate"` // Pattern success rate
}

// DetectToolSequences identifies common sequences of tool executions
func (pd *PatternDetector) DetectToolSequences(minFrequency int, sequenceLength int) []ToolSequence {
	if sequenceLength < 2 || sequenceLength > 5 {
		sequenceLength = 3 // Default to trigrams
	}

	sequenceMap := make(map[string]*toolSequenceData)

	for _, metric := range pd.metrics {
		if len(metric.ToolExecutions) < sequenceLength {
			continue
		}

		// Extract sequences of tools
		tools := make([]string, len(metric.ToolExecutions))
		for i, te := range metric.ToolExecutions {
			tools[i] = te.Name
		}

		// Generate n-grams
		for i := 0; i <= len(tools)-sequenceLength; i++ {
			sequence := tools[i : i+sequenceLength]
			key := makeSequenceKey(sequence)

			if _, exists := sequenceMap[key]; !exists {
				sequenceMap[key] = &toolSequenceData{
					tools: sequence,
				}
			}

			sequenceMap[key].frequency++
			// Track timing and success from the corresponding session
			if i < len(pd.sessions) {
				sequenceMap[key].totalDuration += pd.sessions[i].Duration
				if pd.sessions[i].Success {
					sequenceMap[key].successCount++
				}
			}
		}
	}

	// Convert to slice and filter by frequency
	sequences := make([]ToolSequence, 0)
	for _, data := range sequenceMap {
		if data.frequency >= minFrequency {
			avgTime := time.Duration(0)
			if data.frequency > 0 {
				avgTime = time.Duration(data.totalDuration/int64(data.frequency)) * time.Millisecond
			}

			successRate := 0.0
			if data.frequency > 0 {
				successRate = float64(data.successCount) / float64(data.frequency)
			}

			sequences = append(sequences, ToolSequence{
				Tools:       data.tools,
				Frequency:   data.frequency,
				AvgTime:     avgTime,
				SuccessRate: successRate,
			})
		}
	}

	// Sort by frequency descending
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Frequency > sequences[j].Frequency
	})

	return sequences
}

// DetectBashPatterns identifies common bash command patterns
func (pd *PatternDetector) DetectBashPatterns(minFrequency int) []CommandPattern {
	patternMap := make(map[string]*commandPatternData)

	for _, metric := range pd.metrics {
		for _, cmd := range metric.BashCommands {
			// Extract command pattern (first word)
			pattern := extractCommandPattern(cmd.Command)

			if _, exists := patternMap[pattern]; !exists {
				patternMap[pattern] = &commandPatternData{}
			}

			data := patternMap[pattern]
			data.frequency++
			data.totalDuration += cmd.Duration
			if cmd.Success {
				data.successCount++
			}
		}
	}

	// Convert to slice and filter
	patterns := make([]CommandPattern, 0)
	for pattern, data := range patternMap {
		if data.frequency >= minFrequency {
			avgDuration := time.Duration(0)
			if data.frequency > 0 {
				avgDuration = data.totalDuration / time.Duration(data.frequency)
			}

			successRate := 0.0
			if data.frequency > 0 {
				successRate = float64(data.successCount) / float64(data.frequency)
			}

			patterns = append(patterns, CommandPattern{
				Pattern:     pattern,
				Frequency:   data.frequency,
				SuccessRate: successRate,
				AvgDuration: avgDuration,
			})
		}
	}

	// Sort by frequency descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

// IdentifyAnomalies detects anomalous behaviors using statistical deviation
func (pd *PatternDetector) IdentifyAnomalies(stdDevThreshold float64) []Anomaly {
	if stdDevThreshold <= 0 {
		stdDevThreshold = 2.0 // Default to 2 standard deviations
	}

	anomalies := make([]Anomaly, 0)

	// Calculate baseline statistics
	stats := pd.calculateBaselineStats()

	// Check each session for anomalies
	for _, session := range pd.sessions {
		// Duration anomalies
		durationMs := float64(session.Duration)
		if deviation := math.Abs((durationMs - stats.avgDuration) / stats.stdDuration); deviation > stdDevThreshold {
			anomalies = append(anomalies, Anomaly{
				Type:        "duration",
				Description: "Session duration significantly differs from baseline",
				Severity:    getSeverity(deviation),
				SessionID:   session.ID,
				Timestamp:   session.Timestamp,
				Value:       durationMs,
				Expected:    stats.avgDuration,
				Deviation:   deviation,
			})
		}

		// Error rate anomalies
		errorRate := float64(session.ErrorCount)
		if deviation := math.Abs((errorRate - stats.avgErrors) / stats.stdErrors); deviation > stdDevThreshold && stats.stdErrors > 0 {
			anomalies = append(anomalies, Anomaly{
				Type:        "error_rate",
				Description: "Abnormally high error count",
				Severity:    getSeverity(deviation),
				SessionID:   session.ID,
				Timestamp:   session.Timestamp,
				Value:       errorRate,
				Expected:    stats.avgErrors,
				Deviation:   deviation,
			})
		}

		// Failed session when success rate is normally high
		if !session.Success && stats.successRate > 0.8 {
			anomalies = append(anomalies, Anomaly{
				Type:        "session_failure",
				Description: "Session failed when success rate is typically high",
				Severity:    "high",
				SessionID:   session.ID,
				Timestamp:   session.Timestamp,
				Value:       0.0,
				Expected:    stats.successRate,
				Deviation:   (stats.successRate - 0.0) / 0.2, // Normalized
			})
		}
	}

	// Sort by severity and deviation
	sort.Slice(anomalies, func(i, j int) bool {
		if anomalies[i].Severity != anomalies[j].Severity {
			return severityValue(anomalies[i].Severity) > severityValue(anomalies[j].Severity)
		}
		return anomalies[i].Deviation > anomalies[j].Deviation
	})

	return anomalies
}

// ClusterSessions groups similar sessions using simple k-means clustering
func (pd *PatternDetector) ClusterSessions(numClusters int) []BehaviorCluster {
	if numClusters <= 0 || numClusters > len(pd.sessions) {
		numClusters = 3 // Default to 3 clusters
	}

	if len(pd.sessions) == 0 {
		return []BehaviorCluster{}
	}

	// Extract feature vectors from sessions
	features := make([][]float64, len(pd.sessions))
	for i, session := range pd.sessions {
		features[i] = extractFeatures(session)
	}

	// Initialize centroids using k-means++
	centroids := initializeCentroids(features, numClusters)

	// Run k-means iterations
	maxIterations := 10
	assignments := make([]int, len(features))

	for iter := 0; iter < maxIterations; iter++ {
		// Assign points to nearest centroid
		changed := false
		for i, feature := range features {
			nearest := findNearestCentroid(feature, centroids)
			if assignments[i] != nearest {
				assignments[i] = nearest
				changed = true
			}
		}

		if !changed {
			break
		}

		// Update centroids
		centroids = updateCentroids(features, assignments, numClusters)
	}

	// Build clusters
	clusters := make([]BehaviorCluster, numClusters)
	for i := range clusters {
		clusters[i] = BehaviorCluster{
			ClusterID:  i,
			SessionIDs: make([]string, 0),
			Centroid:   centroids[i],
		}
	}

	// Assign sessions to clusters
	for i, clusterID := range assignments {
		clusters[clusterID].SessionIDs = append(clusters[clusterID].SessionIDs, pd.sessions[i].ID)
		clusters[clusterID].Size++
	}

	// Generate descriptions
	for i := range clusters {
		clusters[i].Description = generateClusterDescription(&clusters[i], centroids[i])
	}

	return clusters
}

// Helper types and functions

type toolSequenceData struct {
	tools         []string
	frequency     int
	totalDuration int64
	successCount  int
}

type commandPatternData struct {
	frequency     int
	totalDuration time.Duration
	successCount  int
}

type baselineStats struct {
	avgDuration  float64
	stdDuration  float64
	avgErrors    float64
	stdErrors    float64
	successRate  float64
}

func makeSequenceKey(tools []string) string {
	key := ""
	for i, tool := range tools {
		if i > 0 {
			key += "->"
		}
		key += tool
	}
	return key
}

func extractCommandPattern(command string) string {
	// Extract first word/command
	for i, ch := range command {
		if ch == ' ' || ch == '\t' {
			return command[:i]
		}
	}
	return command
}

func (pd *PatternDetector) calculateBaselineStats() baselineStats {
	if len(pd.sessions) == 0 {
		return baselineStats{}
	}

	// Calculate averages
	var totalDuration float64
	var totalErrors float64
	var successCount int

	for _, session := range pd.sessions {
		totalDuration += float64(session.Duration)
		totalErrors += float64(session.ErrorCount)
		if session.Success {
			successCount++
		}
	}

	n := float64(len(pd.sessions))
	avgDuration := totalDuration / n
	avgErrors := totalErrors / n
	successRate := float64(successCount) / n

	// Calculate standard deviations
	var sumSqDuration float64
	var sumSqErrors float64

	for _, session := range pd.sessions {
		durDiff := float64(session.Duration) - avgDuration
		sumSqDuration += durDiff * durDiff

		errDiff := float64(session.ErrorCount) - avgErrors
		sumSqErrors += errDiff * errDiff
	}

	stdDuration := math.Sqrt(sumSqDuration / n)
	stdErrors := math.Sqrt(sumSqErrors / n)

	return baselineStats{
		avgDuration: avgDuration,
		stdDuration: stdDuration,
		avgErrors:   avgErrors,
		stdErrors:   stdErrors,
		successRate: successRate,
	}
}

func getSeverity(deviation float64) string {
	if deviation >= 3.0 {
		return "high"
	} else if deviation >= 2.0 {
		return "medium"
	}
	return "low"
}

func severityValue(severity string) int {
	switch severity {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func extractFeatures(session Session) []float64 {
	// Feature vector: [duration, error_count, success]
	successValue := 0.0
	if session.Success {
		successValue = 1.0
	}

	return []float64{
		float64(session.Duration) / 1000.0, // Normalize to seconds
		float64(session.ErrorCount),
		successValue,
	}
}

func initializeCentroids(features [][]float64, k int) [][]float64 {
	if k <= 0 || len(features) == 0 {
		return [][]float64{}
	}

	centroids := make([][]float64, k)

	// Use first k points as initial centroids (simple approach)
	for i := 0; i < k && i < len(features); i++ {
		centroids[i] = make([]float64, len(features[0]))
		copy(centroids[i], features[i])
	}

	return centroids
}

func findNearestCentroid(feature []float64, centroids [][]float64) int {
	if len(centroids) == 0 {
		return 0
	}

	minDist := math.MaxFloat64
	nearest := 0

	for i, centroid := range centroids {
		dist := euclideanDistance(feature, centroid)
		if dist < minDist {
			minDist = dist
			nearest = i
		}
	}

	return nearest
}

func euclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	sumSq := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sumSq += diff * diff
	}

	return math.Sqrt(sumSq)
}

func updateCentroids(features [][]float64, assignments []int, k int) [][]float64 {
	if len(features) == 0 || k <= 0 {
		return [][]float64{}
	}

	// Calculate new centroids as mean of assigned points
	centroids := make([][]float64, k)
	counts := make([]int, k)

	// Initialize centroids
	for i := range centroids {
		centroids[i] = make([]float64, len(features[0]))
	}

	// Sum features for each cluster
	for i, assignment := range assignments {
		counts[assignment]++
		for j := range features[i] {
			centroids[assignment][j] += features[i][j]
		}
	}

	// Divide by count to get mean
	for i := range centroids {
		if counts[i] > 0 {
			for j := range centroids[i] {
				centroids[i][j] /= float64(counts[i])
			}
		}
	}

	return centroids
}

func generateClusterDescription(cluster *BehaviorCluster, centroid []float64) string {
	if len(centroid) < 3 {
		return "Unknown pattern"
	}

	duration := centroid[0]        // seconds
	errorCount := centroid[1]
	successRate := centroid[2]

	if successRate > 0.8 && errorCount < 1.0 {
		if duration < 60 {
			return "Fast, successful sessions"
		}
		return "Slow but successful sessions"
	} else if errorCount > 2.0 {
		return "High-error sessions"
	} else if successRate < 0.3 {
		return "Failed sessions"
	}

	return "Mixed-result sessions"
}
