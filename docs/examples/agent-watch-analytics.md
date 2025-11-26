# Agent Watch Analytics Features

Advanced analytics capabilities for behavioral data analysis.

## Overview

Agent Watch includes sophisticated analytics features:

- **Pattern Detection**: Identify recurring patterns in agent behavior
- **Failure Prediction**: Predict task failures before they occur
- **Performance Scoring**: Multi-dimensional agent scoring system
- **Anomaly Detection**: Find statistical outliers in behavior
- **Behavior Clustering**: Group similar sessions together

## Pattern Detection

### Tool Sequence Analysis

Detects common sequences of tool executions (n-grams):

```go
// Example: Detect 3-tool sequences appearing 5+ times
detector := behavioral.NewPatternDetector(sessions, metrics)
sequences := detector.DetectToolSequences(5, 3)

// Results:
// ToolSequence{
//   Tools: ["Read", "Edit", "Bash"],
//   Frequency: 42,
//   AvgTime: 45s,
//   SuccessRate: 0.92
// }
```

**Use Cases:**
- Identify common workflows
- Find inefficient tool chains
- Discover successful patterns to replicate

### Bash Pattern Analysis

Identifies frequently used bash command patterns:

```go
patterns := detector.DetectBashPatterns(3)

// Results:
// CommandPattern{
//   Pattern: "git",
//   Frequency: 156,
//   SuccessRate: 0.95,
//   AvgDuration: 1.2s
// }
```

**Pattern Extraction:**
- Extracts first word/command from bash strings
- Groups similar commands together
- Tracks success rates per pattern

## Anomaly Detection

### Statistical Outliers

Uses standard deviation to identify anomalies:

```go
anomalies := detector.IdentifyAnomalies(2.0)  // 2 std devs

// Result types:
// - "duration": Session took unusually long/short
// - "error_rate": Abnormally high error count
// - "session_failure": Failed when success rate is normally high
```

### Anomaly Structure

```go
type Anomaly struct {
    Type        string    // duration, error_rate, session_failure
    Description string    // Human-readable explanation
    Severity    string    // low, medium, high
    SessionID   string    // Which session
    Timestamp   time.Time // When it occurred
    Value       float64   // Actual value
    Expected    float64   // Expected value
    Deviation   float64   // Standard deviations from mean
}
```

### Severity Levels

| Deviation | Severity |
|-----------|----------|
| >= 3.0 | high |
| >= 2.0 | medium |
| < 2.0 | low |

### Example Analysis

```bash
# Detect anomalies in last 7 days
# (Programmatic - via behavioral package)

# Anomaly output:
# Type: duration
# Description: Session duration significantly differs from baseline
# Severity: high
# SessionID: abc123
# Value: 1800000ms
# Expected: 300000ms
# Deviation: 3.5
```

## Behavior Clustering

### K-Means Clustering

Groups similar sessions using feature vectors:

```go
clusters := detector.ClusterSessions(3)  // 3 clusters

// Each cluster contains:
// - ClusterID
// - SessionIDs (grouped sessions)
// - Centroid (feature vector)
// - Description (auto-generated)
// - Size (number of sessions)
```

### Feature Vectors

Sessions are represented by:
- Duration (normalized to seconds)
- Error count
- Success (binary: 0 or 1)

### Auto-Generated Descriptions

| Cluster Pattern | Description |
|-----------------|-------------|
| High success, low errors, fast | "Fast, successful sessions" |
| High success, low errors, slow | "Slow but successful sessions" |
| High error count | "High-error sessions" |
| Low success rate | "Failed sessions" |
| Mixed | "Mixed-result sessions" |

## Failure Prediction

### Prediction Model

Predicts task failure probability based on:

1. **Tool Failure Rates**: Historical failure rates per tool
2. **Similar Sessions**: Jaccard similarity with past sessions
3. **Combination Risk**: Detection of risky tool combinations

```go
predictor := behavioral.NewFailurePredictor(sessions, metrics)

result, err := predictor.PredictFailure(session, toolUsage)
// result.Probability = 0.35
// result.Confidence = 0.72
// result.RiskLevel = "medium"
```

### Prediction Result

```go
type PredictionResult struct {
    Probability     float64  // 0.0 to 1.0
    Confidence      float64  // 0.0 to 0.9
    RiskLevel       string   // low, medium, high
    Explanation     string   // Human-readable explanation
    RiskFactors     []string // Specific risks identified
    Recommendations []string // Mitigation suggestions
}
```

### Risk Levels

| Probability | Risk Level |
|-------------|------------|
| >= 0.7 | high |
| >= 0.4 | medium |
| < 0.4 | low |

### Risk Factor Detection

Automatically identifies:
- Tools with >50% historical failure rate
- High tool diversity (>8 different tools)
- High tool usage count (>50 executions)
- Similar sessions with high failure rates

### Recommendations

Based on risk factors, suggests:
- "Review X tool usage carefully"
- "Consider breaking task into smaller subtasks"
- "Monitor execution closely due to high complexity"
- "Enable verbose logging for debugging"
- "Plan for potential retry with different approach"
- "Validate bash commands before execution"

### High-Risk Tool Combinations

Detects risky patterns:
- Heavy Bash usage (>5) with Write/Edit (>3)
- >8 different tool types (complexity)
- >50 total tool executions

## Performance Scoring

### Multi-Dimensional Scoring

Agents are scored across four dimensions:

| Dimension | Weight | Description |
|-----------|--------|-------------|
| Success | 40% | Task completion rate |
| Cost Efficiency | 25% | Token usage relative to baseline |
| Speed | 20% | Duration relative to baseline |
| Error Recovery | 15% | Ability to recover from errors |

### Score Calculation

```go
scorer := behavioral.NewPerformanceScorer(sessions, metrics)
score := scorer.ScoreAgent("backend-developer")

// AgentScore{
//   AgentName: "backend-developer",
//   SuccessScore: 0.92,
//   CostEffScore: 0.78,
//   SpeedScore: 0.85,
//   ErrorRecovScore: 0.70,
//   CompositeScore: 0.84,
//   SampleSize: 25,
//   Domain: "backend"
// }
```

### Custom Weights

```go
scorer.SetWeights(behavioral.ScoreWeights{
    Success:    0.50,  // Prioritize success
    CostEff:    0.20,
    Speed:      0.15,
    ErrorRecov: 0.15,
})
```

### Agent Ranking

```go
rankings := scorer.RankAgents()

// []RankedAgent{
//   {AgentName: "frontend-developer", Score: 0.91, Rank: 1},
//   {AgentName: "backend-developer", Score: 0.84, Rank: 2},
//   {AgentName: "test-automator", Score: 0.79, Rank: 3},
// }
```

### Domain-Specific Comparison

```go
domains := scorer.CompareWithinDomain()

// map[string][]RankedAgent{
//   "backend": [{Agent1, 0.92, 1}, {Agent2, 0.85, 2}],
//   "frontend": [{Agent3, 0.88, 1}],
//   "devops": [{Agent4, 0.75, 1}],
// }
```

### Domain Inference

Automatically infers agent domain from name:
- `backend`, `api`, `database` -> "backend"
- `frontend`, `ui`, `react` -> "frontend"
- `devops`, `deploy`, `infra` -> "devops"
- `test`, `qa` -> "testing"
- `security`, `audit` -> "security"
- Other -> "general"

### Sample Size Adjustment

Scores are adjusted for confidence based on sample size:
- >= 10 samples: Full score
- < 10 samples: Regressed toward mean (0.5)

```go
// Confidence factor = sampleSize / 10
// Adjusted = score * confidence + 0.5 * (1 - confidence)
```

## Statistics Aggregation

### AggregateStats

Combines metrics into summary statistics:

```go
type AggregateStats struct {
    TotalSessions     int
    TotalAgents       int
    TotalOperations   int
    SuccessRate       float64
    ErrorRate         float64
    AverageDuration   time.Duration
    TotalCost         float64
    TotalInputTokens  int64
    TotalOutputTokens int64
    TopTools          []ToolStats
    AgentBreakdown    map[string]int
}
```

### Top Tools Analysis

```go
topTools := stats.GetTopTools(10)

// []ToolStats{
//   {Name: "Read", Count: 450, SuccessRate: 0.99, ErrorRate: 0.01},
//   {Name: "Write", Count: 120, SuccessRate: 0.96, ErrorRate: 0.04},
// }
```

## Practical Applications

### Pre-Task Risk Assessment

Before running a task, predict failure probability:

```go
// 1. Analyze planned tool usage
toolUsage := []string{"Read", "Bash", "Write", "Bash", "Edit"}

// 2. Predict failure
result, _ := predictor.PredictFailure(nil, toolUsage)

// 3. Take action based on risk
if result.RiskLevel == "high" {
    log.Warn("High risk task: %s", result.Explanation)
    for _, rec := range result.Recommendations {
        log.Info("Recommendation: %s", rec)
    }
}
```

### Agent Selection

Choose best agent for task domain:

```go
// Get rankings within domain
domains := scorer.CompareWithinDomain()

// Select top-ranked backend agent
backendAgents := domains["backend"]
if len(backendAgents) > 0 {
    bestAgent := backendAgents[0].AgentName
    // Use bestAgent for backend tasks
}
```

### Quality Control Enhancement

Provide behavioral context to QC reviews:

```go
// Gather context for QC
toolSequences := detector.DetectToolSequences(3, 3)
anomalies := detector.IdentifyAnomalies(2.0)
prediction := predictor.PredictFailure(session, toolUsage)

// Include in QC prompt
qcContext := fmt.Sprintf(`
Behavioral Context:
- Common sequences: %v
- Anomalies detected: %d
- Failure probability: %.2f
- Risk factors: %v
`, toolSequences, len(anomalies), prediction.Probability, prediction.RiskFactors)
```

### Historical Trend Analysis

Track patterns over time:

```go
// Analyze weekly trends
weeklyMetrics := collectWeeklyMetrics()
for _, week := range weeklyMetrics {
    stats := behavioral.CalculateStats(week)
    fmt.Printf("Week %d: Success=%.2f, Errors=%.2f\n",
        week.Number, stats.SuccessRate, stats.ErrorRate)
}
```

### Cost Optimization

Identify cost-inefficient patterns:

```go
// Get agent cost efficiency scores
for _, agent := range uniqueAgents {
    score := scorer.ScoreAgent(agent)
    if score.CostEffScore < 0.5 {
        fmt.Printf("Warning: %s has low cost efficiency (%.2f)\n",
            agent, score.CostEffScore)
    }
}
```

## Integration with Conductor

### QC Enhancement

Agent Watch provides behavioral context to QC reviews:

```yaml
# .conductor/config.yaml
learning:
  qc_reads_plan_context: true
  qc_reads_db_context: true
```

### Adaptive Learning

Behavioral data feeds into Conductor's adaptive learning:

1. **Pre-Task**: Load behavioral history for context
2. **Execution**: Track tool usage and outcomes
3. **Post-Task**: Store metrics in database
4. **Future Tasks**: Use patterns to improve agent selection

### Inter-Retry Agent Swapping

Performance scores inform agent swapping decisions:

```go
// On task failure, consider swapping agent
if score.CompositeScore < threshold {
    // Learning system recommends alternative agent
    alternativeAgent := learning.SelectBetterAgent(task, currentAgent)
}
```
