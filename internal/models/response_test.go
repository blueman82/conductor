package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAgentResponse_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		resp    AgentResponse
		want    string
	}{
		{
			name: "complete response",
			resp: AgentResponse{
				Status:   "success",
				Summary:  "Task completed",
				Output:   "Full output",
				Errors:   []string{},
				Files:    []string{"file1.go"},
				Metadata: map[string]interface{}{"duration_ms": float64(1000)},
			},
			want: `{"status":"success","summary":"Task completed","output":"Full output","errors":[],"files_modified":["file1.go"],"metadata":{"duration_ms":1000}}`,
		},
		{
			name: "minimal response",
			resp: AgentResponse{
				Status:   "success",
				Summary:  "Done",
				Output:   "result",
				Errors:   []string{},
				Files:    []string{},
				Metadata: map[string]interface{}{},
			},
			want: `{"status":"success","summary":"Done","output":"result","errors":[],"files_modified":[],"metadata":{}}`,
		},
		{
			name: "failed response with errors",
			resp: AgentResponse{
				Status:   "failed",
				Summary:  "Compilation error",
				Output:   "Build failed",
				Errors:   []string{"undefined variable", "missing import"},
				Files:    []string{},
				Metadata: map[string]interface{}{},
			},
			want: `{"status":"failed","summary":"Compilation error","output":"Build failed","errors":["undefined variable","missing import"],"files_modified":[],"metadata":{}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.resp)
			if err != nil {
				t.Errorf("Marshal() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestAgentResponse_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AgentResponse
		wantErr bool
	}{
		{
			name:  "valid JSON",
			input: `{"status":"success","summary":"Done","output":"result","errors":[],"files_modified":["test.go"],"metadata":{}}`,
			want: AgentResponse{
				Status:   "success",
				Summary:  "Done",
				Output:   "result",
				Errors:   []string{},
				Files:    []string{"test.go"},
				Metadata: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:  "valid JSON with metadata",
			input: `{"status":"success","summary":"Done","output":"result","errors":[],"files_modified":[],"metadata":{"duration_ms":500}}`,
			want: AgentResponse{
				Status:   "success",
				Summary:  "Done",
				Output:   "result",
				Errors:   []string{},
				Files:    []string{},
				Metadata: map[string]interface{}{"duration_ms": float64(500)},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			want:    AgentResponse{},
			wantErr: true,
		},
		{
			name:  "missing optional fields",
			input: `{"status":"success","summary":"Done","output":"result"}`,
			want: AgentResponse{
				Status:  "success",
				Summary: "Done",
				Output:  "result",
			},
			wantErr: false,
		},
		{
			name:  "extra unknown fields ignored",
			input: `{"status":"success","summary":"Done","output":"result","unknown_field":"ignored","errors":[],"files_modified":[],"metadata":{}}`,
			want: AgentResponse{
				Status:   "success",
				Summary:  "Done",
				Output:   "result",
				Errors:   []string{},
				Files:    []string{},
				Metadata: map[string]interface{}{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AgentResponse
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQCResponse_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    QCResponse
		wantErr bool
	}{
		{
			name:  "valid QC response",
			input: `{"verdict":"GREEN","feedback":"All tests pass","issues":[],"recommendations":["Good work"],"should_retry":false,"suggested_agent":""}`,
			want: QCResponse{
				Verdict:         "GREEN",
				Feedback:        "All tests pass",
				Issues:          []Issue{},
				Recommendations: []string{"Good work"},
				ShouldRetry:     false,
				SuggestedAgent:  "",
			},
			wantErr: false,
		},
		{
			name:  "RED verdict with issues",
			input: `{"verdict":"RED","feedback":"Tests failing","issues":[{"severity":"critical","description":"Nil pointer panic","location":"task.go:42"}],"recommendations":["Add nil check"],"should_retry":true,"suggested_agent":"debugger"}`,
			want: QCResponse{
				Verdict:  "RED",
				Feedback: "Tests failing",
				Issues: []Issue{
					{
						Severity:    "critical",
						Description: "Nil pointer panic",
						Location:    "task.go:42",
					},
				},
				Recommendations: []string{"Add nil check"},
				ShouldRetry:     true,
				SuggestedAgent:  "debugger",
			},
			wantErr: false,
		},
		{
			name:  "YELLOW verdict",
			input: `{"verdict":"YELLOW","feedback":"Minor issues","issues":[{"severity":"warning","description":"Missing doc comment","location":"func.go:10"}],"recommendations":["Add documentation"],"should_retry":false,"suggested_agent":""}`,
			want: QCResponse{
				Verdict:  "YELLOW",
				Feedback: "Minor issues",
				Issues: []Issue{
					{
						Severity:    "warning",
						Description: "Missing doc comment",
						Location:    "func.go:10",
					},
				},
				Recommendations: []string{"Add documentation"},
				ShouldRetry:     false,
				SuggestedAgent:  "",
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			want:    QCResponse{},
			wantErr: true,
		},
		{
			name:  "missing optional fields",
			input: `{"verdict":"GREEN","feedback":"OK"}`,
			want: QCResponse{
				Verdict:  "GREEN",
				Feedback: "OK",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got QCResponse
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgentResponse_Validate(t *testing.T) {
	tests := []struct {
		name    string
		resp    AgentResponse
		wantErr bool
	}{
		{
			name: "valid success response",
			resp: AgentResponse{
				Status:  "success",
				Summary: "Done",
				Output:  "result",
			},
			wantErr: false,
		},
		{
			name: "valid failed response",
			resp: AgentResponse{
				Status:  "failed",
				Summary: "Error",
				Output:  "failure",
			},
			wantErr: false,
		},
		{
			name: "missing status",
			resp: AgentResponse{
				Summary: "Done",
				Output:  "result",
			},
			wantErr: true,
		},
		{
			name: "invalid status value",
			resp: AgentResponse{
				Status:  "pending",
				Summary: "Done",
				Output:  "result",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQCResponse_Validate(t *testing.T) {
	tests := []struct {
		name    string
		resp    QCResponse
		wantErr bool
	}{
		{
			name: "valid GREEN verdict",
			resp: QCResponse{
				Verdict:  "GREEN",
				Feedback: "OK",
			},
			wantErr: false,
		},
		{
			name: "valid RED verdict",
			resp: QCResponse{
				Verdict:  "RED",
				Feedback: "Failed",
			},
			wantErr: false,
		},
		{
			name: "valid YELLOW verdict",
			resp: QCResponse{
				Verdict:  "YELLOW",
				Feedback: "Warning",
			},
			wantErr: false,
		},
		{
			name: "missing verdict",
			resp: QCResponse{
				Feedback: "OK",
			},
			wantErr: true,
		},
		{
			name: "invalid verdict value",
			resp: QCResponse{
				Verdict:  "BLUE",
				Feedback: "OK",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
