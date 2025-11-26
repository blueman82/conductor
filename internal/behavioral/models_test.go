package behavioral

import (
	"testing"
	"time"
)

func TestSessionValidate(t *testing.T) {
	tests := []struct {
		name    string
		session Session
		wantErr bool
	}{
		{
			name: "valid session",
			session: Session{
				ID:        "test-123",
				Project:   "test-project",
				Timestamp: time.Now(),
				Status:    "completed",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			session: Session{
				Project:   "test-project",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing project",
			session: Session{
				ID:        "test-123",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			session: Session{
				ID:      "test-123",
				Project: "test-project",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionGetDuration(t *testing.T) {
	session := Session{
		Duration: 5000, // 5000ms = 5s
	}

	expected := 5 * time.Second
	got := session.GetDuration()

	if got != expected {
		t.Errorf("Session.GetDuration() = %v, want %v", got, expected)
	}
}

func TestBehavioralMetricsValidate(t *testing.T) {
	tests := []struct {
		name    string
		metrics BehavioralMetrics
		wantErr bool
	}{
		{
			name: "valid metrics",
			metrics: BehavioralMetrics{
				TotalSessions: 10,
				SuccessRate:   0.8,
				ErrorRate:     0.2,
				TotalCost:     1.50,
			},
			wantErr: false,
		},
		{
			name: "negative sessions",
			metrics: BehavioralMetrics{
				TotalSessions: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid success rate",
			metrics: BehavioralMetrics{
				SuccessRate: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid error rate",
			metrics: BehavioralMetrics{
				ErrorRate: -0.1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metrics.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BehavioralMetrics.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBehavioralMetricsAggregateMetrics(t *testing.T) {
	sessions := []Session{
		{
			ID:         "s1",
			Project:    "test",
			Timestamp:  time.Now(),
			Success:    true,
			Duration:   1000,
			ErrorCount: 0,
			AgentName:  "agent-1",
		},
		{
			ID:         "s2",
			Project:    "test",
			Timestamp:  time.Now(),
			Success:    false,
			Duration:   2000,
			ErrorCount: 3,
			AgentName:  "agent-2",
		},
		{
			ID:         "s3",
			Project:    "test",
			Timestamp:  time.Now(),
			Success:    true,
			Duration:   3000,
			ErrorCount: 1,
			AgentName:  "agent-1",
		},
	}

	metrics := &BehavioralMetrics{}
	metrics.AggregateMetrics(sessions)

	if metrics.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", metrics.TotalSessions)
	}

	expectedSuccessRate := 2.0 / 3.0
	if metrics.SuccessRate != expectedSuccessRate {
		t.Errorf("SuccessRate = %f, want %f", metrics.SuccessRate, expectedSuccessRate)
	}

	expectedAvgDuration := 2 * time.Second // (1000 + 2000 + 3000) / 3 = 2000ms
	if metrics.AverageDuration != expectedAvgDuration {
		t.Errorf("AverageDuration = %v, want %v", metrics.AverageDuration, expectedAvgDuration)
	}

	if metrics.TotalErrors != 4 {
		t.Errorf("TotalErrors = %d, want 4", metrics.TotalErrors)
	}

	if metrics.AgentPerformance["agent-1"] != 2 {
		t.Errorf("AgentPerformance[agent-1] = %d, want 2", metrics.AgentPerformance["agent-1"])
	}
}

func TestToolExecutionValidate(t *testing.T) {
	tests := []struct {
		name    string
		tool    ToolExecution
		wantErr bool
	}{
		{
			name: "valid tool",
			tool: ToolExecution{
				Name:        "Read",
				Count:       10,
				SuccessRate: 0.9,
				ErrorRate:   0.1,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tool: ToolExecution{
				Count: 10,
			},
			wantErr: true,
		},
		{
			name: "invalid success rate",
			tool: ToolExecution{
				Name:        "Write",
				SuccessRate: 1.2,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tool.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToolExecution.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToolExecutionCalculateRates(t *testing.T) {
	tool := &ToolExecution{
		Count:        10,
		TotalSuccess: 8,
		TotalErrors:  2,
	}

	tool.CalculateRates()

	if tool.SuccessRate != 0.8 {
		t.Errorf("SuccessRate = %f, want 0.8", tool.SuccessRate)
	}
	if tool.ErrorRate != 0.2 {
		t.Errorf("ErrorRate = %f, want 0.2", tool.ErrorRate)
	}
}

func TestBashCommandIsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		want     bool
	}{
		{"success", 0, true},
		{"failure", 1, false},
		{"error", 127, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := BashCommand{
				Command:  "test",
				ExitCode: tt.exitCode,
			}
			if got := bc.IsSuccess(); got != tt.want {
				t.Errorf("BashCommand.IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileOperationIsWrite(t *testing.T) {
	tests := []struct {
		name string
		op   FileOperation
		want bool
	}{
		{"write", FileOperation{Type: "write", Path: "/test"}, true},
		{"edit", FileOperation{Type: "edit", Path: "/test"}, true},
		{"read", FileOperation{Type: "read", Path: "/test"}, false},
		{"delete", FileOperation{Type: "delete", Path: "/test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsWrite(); got != tt.want {
				t.Errorf("FileOperation.IsWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileOperationIsRead(t *testing.T) {
	tests := []struct {
		name string
		op   FileOperation
		want bool
	}{
		{"read", FileOperation{Type: "read", Path: "/test"}, true},
		{"write", FileOperation{Type: "write", Path: "/test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsRead(); got != tt.want {
				t.Errorf("FileOperation.IsRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenUsageTotalTokens(t *testing.T) {
	tu := TokenUsage{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	if got := tu.TotalTokens(); got != 1500 {
		t.Errorf("TokenUsage.TotalTokens() = %d, want 1500", got)
	}
}

func TestTokenUsageCalculateCost(t *testing.T) {
	tu := &TokenUsage{
		InputTokens:  1_000_000, // 1M input
		OutputTokens: 1_000_000, // 1M output
	}

	tu.CalculateCost()

	// $3 for 1M input + $15 for 1M output = $18
	expected := 18.0
	if tu.CostUSD != expected {
		t.Errorf("TokenUsage.CalculateCost() = %f, want %f", tu.CostUSD, expected)
	}
}

func TestAggregateMetrics(t *testing.T) {
	metrics := []BehavioralMetrics{
		{
			TotalSessions:   5,
			AverageDuration: 2 * time.Second,
			TokenUsage: TokenUsage{
				InputTokens:  1000,
				OutputTokens: 500,
				CostUSD:      0.50,
			},
			TotalErrors: 2,
			AgentPerformance: map[string]int{
				"agent-1": 3,
			},
		},
		{
			TotalSessions:   3,
			AverageDuration: 3 * time.Second,
			TokenUsage: TokenUsage{
				InputTokens:  2000,
				OutputTokens: 1000,
				CostUSD:      1.00,
			},
			TotalErrors: 1,
			AgentPerformance: map[string]int{
				"agent-2": 2,
			},
		},
	}

	result := AggregateMetrics(metrics)

	if result["session_count"] != 8 {
		t.Errorf("session_count = %v, want 8", result["session_count"])
	}

	if result["total_cost_usd"] != 1.50 {
		t.Errorf("total_cost_usd = %v, want 1.50", result["total_cost_usd"])
	}

	if result["total_errors"] != 3 {
		t.Errorf("total_errors = %v, want 3", result["total_errors"])
	}

	if result["total_input_tokens"] != int64(3000) {
		t.Errorf("total_input_tokens = %v, want 3000", result["total_input_tokens"])
	}
}

func TestModels(t *testing.T) {
	// Integration test verifying all models work together
	session := Session{
		ID:         "test-session",
		Project:    "test-project",
		Timestamp:  time.Now(),
		Status:     "completed",
		AgentName:  "test-agent",
		Duration:   5000,
		Success:    true,
		ErrorCount: 0,
	}

	if err := session.Validate(); err != nil {
		t.Fatalf("Session.Validate() failed: %v", err)
	}

	toolExec := ToolExecution{
		Name:         "Read",
		Count:        10,
		TotalSuccess: 9,
		TotalErrors:  1,
	}
	toolExec.CalculateRates()

	if err := toolExec.Validate(); err != nil {
		t.Fatalf("ToolExecution.Validate() failed: %v", err)
	}

	bashCmd := BashCommand{
		Command:      "go test",
		ExitCode:     0,
		OutputLength: 1024,
		Duration:     2 * time.Second,
		Success:      true,
		Timestamp:    time.Now(),
	}

	if err := bashCmd.Validate(); err != nil {
		t.Fatalf("BashCommand.Validate() failed: %v", err)
	}

	fileOp := FileOperation{
		Type:      "write",
		Path:      "/test/file.go",
		SizeBytes: 2048,
		Success:   true,
		Timestamp: time.Now(),
		Duration:  100,
	}

	if err := fileOp.Validate(); err != nil {
		t.Fatalf("FileOperation.Validate() failed: %v", err)
	}

	tokenUsage := TokenUsage{
		InputTokens:  1000,
		OutputTokens: 500,
		ModelName:    "claude-sonnet-4-5",
	}
	tokenUsage.CalculateCost()

	if err := tokenUsage.Validate(); err != nil {
		t.Fatalf("TokenUsage.Validate() failed: %v", err)
	}

	metrics := BehavioralMetrics{
		TotalSessions:   1,
		SuccessRate:     1.0,
		AverageDuration: session.GetDuration(),
		TotalCost:       tokenUsage.CostUSD,
		ToolExecutions:  []ToolExecution{toolExec},
		BashCommands:    []BashCommand{bashCmd},
		FileOperations:  []FileOperation{fileOp},
		TokenUsage:      tokenUsage,
		ErrorRate:       0.0,
		TotalErrors:     0,
		AgentPerformance: map[string]int{
			"test-agent": 1,
		},
	}

	if err := metrics.Validate(); err != nil {
		t.Fatalf("BehavioralMetrics.Validate() failed: %v", err)
	}
}
