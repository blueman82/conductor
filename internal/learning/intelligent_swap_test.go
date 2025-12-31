package learning

import (
	"context"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/agent"
)

// MockKnowledgeGraph implements KnowledgeGraph for testing
type MockKnowledgeGraph struct {
	nodes         map[string]*KnowledgeNode
	edges         []KnowledgeEdge
	relatedNodes  []KnowledgeNode
	relatedErr    error
	pathNodes     []KnowledgeNode
	pathErr       error
	addNodeErr    error
	addEdgeErr    error
	deleteNodeErr error
}

func NewMockKnowledgeGraph() *MockKnowledgeGraph {
	return &MockKnowledgeGraph{
		nodes: make(map[string]*KnowledgeNode),
		edges: make([]KnowledgeEdge, 0),
	}
}

func (m *MockKnowledgeGraph) AddNode(ctx context.Context, node *KnowledgeNode) error {
	if m.addNodeErr != nil {
		return m.addNodeErr
	}
	m.nodes[node.ID] = node
	return nil
}

func (m *MockKnowledgeGraph) AddEdge(ctx context.Context, edge *KnowledgeEdge) error {
	if m.addEdgeErr != nil {
		return m.addEdgeErr
	}
	m.edges = append(m.edges, *edge)
	return nil
}

func (m *MockKnowledgeGraph) GetNode(ctx context.Context, nodeID string) (*KnowledgeNode, error) {
	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, nil
	}
	return node, nil
}

func (m *MockKnowledgeGraph) GetEdges(ctx context.Context, nodeID string, edgeTypes []EdgeType) ([]KnowledgeEdge, error) {
	var result []KnowledgeEdge
	for _, edge := range m.edges {
		if edge.SourceID == nodeID || edge.TargetID == nodeID {
			result = append(result, edge)
		}
	}
	return result, nil
}

func (m *MockKnowledgeGraph) GetRelated(ctx context.Context, nodeID string, hops int, edgeTypes []EdgeType) ([]KnowledgeNode, error) {
	if m.relatedErr != nil {
		return nil, m.relatedErr
	}
	return m.relatedNodes, nil
}

func (m *MockKnowledgeGraph) FindPath(ctx context.Context, fromID, toID string) ([]KnowledgeNode, error) {
	if m.pathErr != nil {
		return nil, m.pathErr
	}
	return m.pathNodes, nil
}

func (m *MockKnowledgeGraph) DeleteNode(ctx context.Context, nodeID string) error {
	if m.deleteNodeErr != nil {
		return m.deleteNodeErr
	}
	delete(m.nodes, nodeID)
	return nil
}

// MockLIPCollector implements LIPCollector for testing
type MockLIPCollector struct {
	events   []LIPEvent
	progress ProgressScore
	err      error
}

func NewMockLIPCollector() *MockLIPCollector {
	return &MockLIPCollector{
		events:   make([]LIPEvent, 0),
		progress: ProgressNone,
	}
}

func (m *MockLIPCollector) RecordEvent(ctx context.Context, event *LIPEvent) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, *event)
	return nil
}

func (m *MockLIPCollector) GetEvents(ctx context.Context, filter *LIPFilter) ([]LIPEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func (m *MockLIPCollector) CalculateProgress(ctx context.Context, taskExecutionID int64) (ProgressScore, error) {
	if m.err != nil {
		return ProgressNone, m.err
	}
	return m.progress, nil
}

// MockWaiterLogger implements budget.WaiterLogger for testing
type MockWaiterLogger struct {
	countdownCalls int
	announceCalls  int
}

func (m *MockWaiterLogger) LogRateLimitCountdown(remaining, total time.Duration) {
	m.countdownCalls++
}

func (m *MockWaiterLogger) LogRateLimitAnnounce(remaining, total time.Duration) {
	m.announceCalls++
}

func TestNewIntelligentAgentSwapper(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}
	timeout := 90 * time.Second

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, timeout, logger)

	if swapper == nil {
		t.Fatal("expected non-nil swapper")
	}

	if swapper.Registry != registry {
		t.Error("registry not set correctly")
	}

	if swapper.KnowledgeGraph != kg {
		t.Error("knowledge graph not set correctly")
	}

	if swapper.LIPStore != lip {
		t.Error("LIP store not set correctly")
	}

	if swapper.ClaudePath != "claude" {
		t.Errorf("expected ClaudePath 'claude', got '%s'", swapper.ClaudePath)
	}

	if swapper.Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, swapper.Timeout)
	}
}

func TestNewIntelligentAgentSwapper_CustomTimeout(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	customTimeout := 120 * time.Second
	swapper := NewIntelligentAgentSwapper(registry, kg, lip, customTimeout, logger)

	if swapper.Timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, swapper.Timeout)
	}
}

func TestSelectAgent_NilContext(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	_, err := swapper.SelectAgent(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil swap context")
	}

	if err.Error() != "swap context cannot be nil" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestExtractFileExtensions(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "single go file",
			files:    []string{"internal/executor/task.go"},
			expected: []string{"go"},
		},
		{
			name:     "multiple go files",
			files:    []string{"main.go", "internal/parser/parser.go"},
			expected: []string{"go"},
		},
		{
			name:     "mixed extensions",
			files:    []string{"main.go", "config.yaml", "README.md"},
			expected: []string{"go", "yaml", "md"},
		},
		{
			name:     "no extension",
			files:    []string{"Makefile", "Dockerfile"},
			expected: []string{},
		},
		{
			name:     "empty files",
			files:    []string{},
			expected: []string{},
		},
		{
			name:     "typescript files",
			files:    []string{"src/app.ts", "src/utils.tsx", "types.d.ts"},
			expected: []string{"ts", "tsx"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFileExtensions(tt.files)

			// Convert to map for comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, ext := range result {
				resultMap[ext] = true
			}

			expectedMap := make(map[string]bool)
			for _, ext := range tt.expected {
				expectedMap[ext] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("expected %d extensions, got %d", len(expectedMap), len(resultMap))
			}

			for ext := range expectedMap {
				if !resultMap[ext] {
					t.Errorf("expected extension '%s' not found", ext)
				}
			}
		})
	}
}

func TestNormalizeFileID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "internal/executor/task.go",
			expected: "file:internal/executor/task.go",
		},
		{
			input:    "./src/main.ts",
			expected: "file:src/main.ts",
		},
		{
			input:    "MAIN.GO",
			expected: "file:main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeFileID(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestAgentSwapSchema(t *testing.T) {
	schema := AgentSwapSchema()

	if schema == "" {
		t.Error("expected non-empty schema")
	}

	// Check for required fields
	if !contains(schema, "recommended_agent") {
		t.Error("schema should contain recommended_agent")
	}
	if !contains(schema, "rationale") {
		t.Error("schema should contain rationale")
	}
	if !contains(schema, "confidence") {
		t.Error("schema should contain confidence")
	}
	if !contains(schema, "alternatives") {
		t.Error("schema should contain alternatives")
	}
}

func TestApplyGuardrails_NilRecommendation(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	result := swapper.applyGuardrails(nil)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.RecommendedAgent != "" {
		t.Errorf("expected empty recommended agent, got '%s'", result.RecommendedAgent)
	}

	if result.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0, got %f", result.Confidence)
	}
}

func TestApplyGuardrails_ConfidenceClamping(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	tests := []struct {
		input    float64
		expected float64
	}{
		{input: -0.5, expected: 0.0},
		{input: 0.0, expected: 0.0},
		{input: 0.5, expected: 0.5},
		{input: 1.0, expected: 1.0},
		{input: 1.5, expected: 1.0},
	}

	for _, tt := range tests {
		recommendation := &AgentSwapRecommendation{
			RecommendedAgent: "", // Empty so no registry check
			Rationale:        "test",
			Confidence:       tt.input,
		}

		result := swapper.applyGuardrails(recommendation)

		if result.Confidence != tt.expected {
			t.Errorf("input %f: expected confidence %f, got %f", tt.input, tt.expected, result.Confidence)
		}
	}
}

func TestGetAvailableAgents_NilRegistry(t *testing.T) {
	swapper := &IntelligentAgentSwapper{
		Registry: nil,
	}

	agents := swapper.getAvailableAgents()

	if len(agents) != 0 {
		t.Errorf("expected empty list for nil registry, got %v", agents)
	}
}

func TestBuildSwapPrompt_MinimalContext(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	swapCtx := &SwapContext{
		TaskNumber:   "1",
		TaskName:     "Test Task",
		CurrentAgent: "golang-pro",
	}

	prompt, err := swapper.buildSwapPrompt(context.Background(), swapCtx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	// Check for XML root wrapper
	if !contains(prompt, "<agent_swap_context>") {
		t.Error("prompt should contain <agent_swap_context> root wrapper")
	}
	if !contains(prompt, "</agent_swap_context>") {
		t.Error("prompt should contain closing </agent_swap_context> tag")
	}

	// Check for key XML sections
	if !contains(prompt, "<task_context>") {
		t.Error("prompt should contain <task_context> section")
	}
	if !contains(prompt, "<number>1</number>") {
		t.Error("prompt should contain task number in XML format")
	}
	if !contains(prompt, "<current_agent status=\"failed\">golang-pro</current_agent>") {
		t.Error("prompt should contain current agent with status attribute")
	}
	if !contains(prompt, "<available_agents>") {
		t.Error("prompt should contain <available_agents> section")
	}
	if !contains(prompt, "<instructions>") {
		t.Error("prompt should contain <instructions> section")
	}
}

func TestBuildSwapPrompt_FullContext(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	swapCtx := &SwapContext{
		TaskNumber:      "5",
		TaskName:        "Implement feature X",
		TaskDescription: "Add new feature X with proper error handling",
		Files:           []string{"internal/feature/x.go", "internal/feature/x_test.go"},
		CurrentAgent:    "typescript-pro",
		ErrorContext:    "compilation error: undefined type Foo",
		AttemptNumber:   2,
	}

	prompt, err := swapper.buildSwapPrompt(context.Background(), swapCtx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for file context in XML format
	if !contains(prompt, "<file_context>") {
		t.Error("prompt should contain <file_context> section")
	}
	if !contains(prompt, "<file>internal/feature/x.go</file>") {
		t.Error("prompt should contain file names in XML format")
	}
	if !contains(prompt, "<ext>go</ext>") {
		t.Error("prompt should contain file extensions in XML format")
	}

	// Check for error context in XML format
	if !contains(prompt, "<error_context source=\"failed_attempt\">") {
		t.Error("prompt should contain <error_context> section with source attribute")
	}
	if !contains(prompt, "compilation error") {
		t.Error("prompt should contain error message")
	}

	// Check for task description in XML format
	if !contains(prompt, "<task_description>") {
		t.Error("prompt should contain <task_description> section")
	}
}

func TestBuildSwapPrompt_TruncatesLongContent(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	// Create very long error context
	longError := ""
	for i := 0; i < 3000; i++ {
		longError += "x"
	}

	swapCtx := &SwapContext{
		TaskNumber:   "1",
		TaskName:     "Test Task",
		CurrentAgent: "test-agent",
		ErrorContext: longError,
	}

	prompt, err := swapper.buildSwapPrompt(context.Background(), swapCtx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be truncated
	if !contains(prompt, "(truncated)") {
		t.Error("expected long error context to be truncated")
	}
}

func TestGetKnowledgeGraphContext_WithRelatedAgents(t *testing.T) {
	registry := agent.NewRegistry("")
	kg := NewMockKnowledgeGraph()
	lip := NewMockLIPCollector()
	logger := &MockWaiterLogger{}

	// Setup mock knowledge graph with related agent nodes
	kg.relatedNodes = []KnowledgeNode{
		{
			ID:       "agent:golang-pro",
			NodeType: NodeTypeAgent,
			Properties: map[string]interface{}{
				"name": "golang-pro",
			},
		},
		{
			ID:       "agent:backend-developer",
			NodeType: NodeTypeAgent,
			Properties: map[string]interface{}{
				"name": "backend-developer",
			},
		},
	}

	swapper := NewIntelligentAgentSwapper(registry, kg, lip, 90*time.Second, logger)

	files := []string{"internal/executor/task.go"}
	context := swapper.getKnowledgeGraphContext(context.Background(), files)

	if context == "" {
		t.Error("expected non-empty context with related agents")
	}

	if !contains(context, "golang-pro") {
		t.Error("context should mention golang-pro agent")
	}
}

func TestAgentSwapRecommendation_Structure(t *testing.T) {
	rec := AgentSwapRecommendation{
		RecommendedAgent: "golang-pro",
		Rationale:        "Go file requires Go expertise",
		Confidence:       0.85,
		Alternatives:     []string{"backend-developer", "fullstack-developer"},
	}

	if rec.RecommendedAgent != "golang-pro" {
		t.Errorf("unexpected recommended agent: %s", rec.RecommendedAgent)
	}

	if rec.Confidence != 0.85 {
		t.Errorf("unexpected confidence: %f", rec.Confidence)
	}

	if len(rec.Alternatives) != 2 {
		t.Errorf("expected 2 alternatives, got %d", len(rec.Alternatives))
	}
}

func TestSwapContext_Structure(t *testing.T) {
	ctx := SwapContext{
		TaskNumber:      "5",
		TaskName:        "Implement feature",
		TaskDescription: "Full description here",
		Files:           []string{"file1.go", "file2.go"},
		CurrentAgent:    "typescript-pro",
		ErrorContext:    "Error occurred",
		AttemptNumber:   2,
	}

	if ctx.TaskNumber != "5" {
		t.Errorf("unexpected task number: %s", ctx.TaskNumber)
	}

	if ctx.AttemptNumber != 2 {
		t.Errorf("unexpected attempt number: %d", ctx.AttemptNumber)
	}

	if len(ctx.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(ctx.Files))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
