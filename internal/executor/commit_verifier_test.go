package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	Outputs map[string]string // command -> output
	Errors  map[string]error  // command -> error
}

func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Outputs: make(map[string]string),
		Errors:  make(map[string]error),
	}
}

func (m *MockCommandRunner) Run(ctx context.Context, command string) (string, error) {
	if err, ok := m.Errors[command]; ok && err != nil {
		return m.Outputs[command], err
	}
	return m.Outputs[command], nil
}

func TestCommitVerification_Fields(t *testing.T) {
	v := &CommitVerification{
		Found:      true,
		CommitHash: "abc1234",
		FullHash:   "abc1234567890def",
		Message:    "feat: add new feature",
		Mismatch:   "",
	}

	if !v.Found {
		t.Error("Expected Found to be true")
	}
	if v.CommitHash != "abc1234" {
		t.Errorf("Expected CommitHash 'abc1234', got %q", v.CommitHash)
	}
	if v.FullHash != "abc1234567890def" {
		t.Errorf("Expected FullHash 'abc1234567890def', got %q", v.FullHash)
	}
	if v.Message != "feat: add new feature" {
		t.Errorf("Expected Message 'feat: add new feature', got %q", v.Message)
	}
	if v.Mismatch != "" {
		t.Errorf("Expected Mismatch to be empty, got %q", v.Mismatch)
	}
}

func TestNewGitLogVerifier(t *testing.T) {
	v := NewGitLogVerifier()
	if v == nil {
		t.Fatal("Expected non-nil verifier")
	}
	if v.LookbackCommits != 10 {
		t.Errorf("Expected LookbackCommits 10, got %d", v.LookbackCommits)
	}
	if v.CommandRunner != nil {
		t.Error("Expected nil CommandRunner by default")
	}
}

func TestNewGitLogVerifierWithRunner(t *testing.T) {
	runner := NewMockCommandRunner()
	v := NewGitLogVerifierWithRunner(runner)
	if v == nil {
		t.Fatal("Expected non-nil verifier")
	}
	if v.LookbackCommits != 10 {
		t.Errorf("Expected LookbackCommits 10, got %d", v.LookbackCommits)
	}
	if v.CommandRunner != runner {
		t.Error("Expected CommandRunner to be set")
	}
}

func TestGitLogVerifier_Verify_NilSpec(t *testing.T) {
	runner := NewMockCommandRunner()
	v := NewGitLogVerifierWithRunner(runner)

	result, err := v.Verify(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found to be false for nil spec")
	}
	if result.Mismatch != "commit spec is nil" {
		t.Errorf("Expected 'commit spec is nil' mismatch, got %q", result.Mismatch)
	}
}

func TestGitLogVerifier_Verify_EmptySpec(t *testing.T) {
	runner := NewMockCommandRunner()
	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found to be false for empty spec")
	}
	if result.Mismatch != "commit spec is empty" {
		t.Errorf("Expected 'commit spec is empty' mismatch, got %q", result.Mismatch)
	}
}

func TestGitLogVerifier_Verify_FoundCommit(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git log output: short hash + message
	runner.Outputs[`git log --oneline --grep="feat: add new feature" -n 10`] = "abc1234 feat: add new feature\n"
	// Mock git rev-parse for full hash
	runner.Outputs["git rev-parse abc1234"] = "abc1234567890abcdef1234567890abcdef123456\n"

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "add new feature",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Found {
		t.Errorf("Expected Found to be true, mismatch: %s", result.Mismatch)
	}
	if result.CommitHash != "abc1234" {
		t.Errorf("Expected CommitHash 'abc1234', got %q", result.CommitHash)
	}
	if result.Message != "feat: add new feature" {
		t.Errorf("Expected Message 'feat: add new feature', got %q", result.Message)
	}
	if result.FullHash != "abc1234567890abcdef1234567890abcdef123456" {
		t.Errorf("Expected full hash, got %q", result.FullHash)
	}
}

func TestGitLogVerifier_Verify_NotFound(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git log output: empty (no match)
	runner.Outputs[`git log --oneline --grep="feat: nonexistent feature" -n 10`] = ""

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "nonexistent feature",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found to be false")
	}
	expectedMismatch := `no commit found matching "feat: nonexistent feature"`
	if result.Mismatch != expectedMismatch {
		t.Errorf("Expected mismatch %q, got %q", expectedMismatch, result.Mismatch)
	}
}

func TestGitLogVerifier_Verify_GitError(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git error (not in repo)
	runner.Errors[`git log --oneline --grep="feat: add feature" -n 10`] = errors.New("fatal: not a git repository")
	runner.Outputs[`git log --oneline --grep="feat: add feature" -n 10`] = ""

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "add feature",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found to be false on git error")
	}
	if result.Mismatch == "" {
		t.Error("Expected non-empty Mismatch on git error")
	}
}

func TestGitLogVerifier_Verify_MessageWithoutType(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git log output for message-only commit
	runner.Outputs[`git log --oneline --grep="update documentation" -n 10`] = "def5678 update documentation\n"
	runner.Outputs["git rev-parse def5678"] = "def5678901234567890abcdef\n"

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Message: "update documentation",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Found {
		t.Errorf("Expected Found to be true, mismatch: %s", result.Mismatch)
	}
	if result.CommitHash != "def5678" {
		t.Errorf("Expected CommitHash 'def5678', got %q", result.CommitHash)
	}
	if result.Message != "update documentation" {
		t.Errorf("Expected Message 'update documentation', got %q", result.Message)
	}
}

func TestGitLogVerifier_Verify_MultipleCommits(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git log output with multiple matches (returns first)
	runner.Outputs[`git log --oneline --grep="fix: resolve bug" -n 10`] = "111aaaa fix: resolve bug in parser\n222bbbb fix: resolve bug in lexer\n"
	runner.Outputs["git rev-parse 111aaaa"] = "111aaaabbbbcccc\n"

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "fix",
		Message: "resolve bug",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Found {
		t.Errorf("Expected Found to be true, mismatch: %s", result.Mismatch)
	}
	// Should return first match
	if result.CommitHash != "111aaaa" {
		t.Errorf("Expected CommitHash '111aaaa', got %q", result.CommitHash)
	}
	if result.Message != "fix: resolve bug in parser" {
		t.Errorf("Expected Message 'fix: resolve bug in parser', got %q", result.Message)
	}
}

func TestGitLogVerifier_VerifyExact_ExactMatch(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git log output with exact match
	runner.Outputs[`git log --oneline --grep="feat: add feature X" -n 10`] = "aaa1111 feat: add feature X\n"
	runner.Outputs["git rev-parse aaa1111"] = "aaa1111222233334444\n"

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "add feature X",
	}

	result, err := v.VerifyExact(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Found {
		t.Errorf("Expected Found to be true for exact match, mismatch: %s", result.Mismatch)
	}
}

func TestGitLogVerifier_VerifyExact_PartialMatch(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock git log output where grep finds a match but message is not exact
	runner.Outputs[`git log --oneline --grep="feat: add feature" -n 10`] = "bbb2222 feat: add feature with extra stuff\n"
	runner.Outputs["git rev-parse bbb2222"] = "bbb222233334444\n"

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "add feature",
	}

	result, err := v.VerifyExact(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// VerifyExact should fail because message is not exact
	if result.Found {
		t.Error("Expected Found to be false for partial match in VerifyExact")
	}
	if result.Mismatch == "" {
		t.Error("Expected non-empty Mismatch for partial match")
	}
	// Should still have the commit hash from the partial match
	if result.CommitHash != "bbb2222" {
		t.Errorf("Expected CommitHash 'bbb2222', got %q", result.CommitHash)
	}
}

func TestGitLogVerifier_Verify_CustomLookback(t *testing.T) {
	runner := NewMockCommandRunner()

	// Mock with custom lookback
	runner.Outputs[`git log --oneline --grep="chore: update deps" -n 5`] = "ccc3333 chore: update deps\n"
	runner.Outputs["git rev-parse ccc3333"] = "ccc333344445555\n"

	v := NewGitLogVerifierWithRunner(runner)
	v.LookbackCommits = 5

	spec := &models.CommitSpec{
		Type:    "chore",
		Message: "update deps",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Found {
		t.Errorf("Expected Found to be true, mismatch: %s", result.Mismatch)
	}
}

func TestGitLogVerifier_Verify_ContextCancelled(t *testing.T) {
	runner := NewMockCommandRunner()

	// Context will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Mock should return context error
	runner.Errors[`git log --oneline --grep="feat: cancelled" -n 10`] = context.Canceled

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "cancelled",
	}

	result, err := v.Verify(ctx, spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found to be false when context cancelled")
	}
}

func TestGitLogVerifier_Verify_DurationTracked(t *testing.T) {
	runner := NewMockCommandRunner()

	runner.Outputs[`git log --oneline --grep="test: timing" -n 10`] = ""

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "test",
		Message: "timing",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Duration should be tracked (even for fast operations)
	if result.Duration < 0 {
		t.Error("Expected non-negative Duration")
	}
}

// TestCommitVerifierInterface verifies GitLogVerifier implements CommitVerifier.
func TestCommitVerifierInterface(t *testing.T) {
	var _ CommitVerifier = (*GitLogVerifier)(nil)
}

func TestGitLogVerifier_Verify_MalformedOutput(t *testing.T) {
	runner := NewMockCommandRunner()

	// Malformed output (no space separator)
	runner.Outputs[`git log --oneline --grep="feat: malformed" -n 10`] = "abc123nospace\n"

	v := NewGitLogVerifierWithRunner(runner)

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "malformed",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found to be false for malformed output")
	}
	if result.Mismatch == "" {
		t.Error("Expected non-empty Mismatch for malformed output")
	}
}

func TestGitLogVerifier_Verify_ZeroLookback(t *testing.T) {
	runner := NewMockCommandRunner()

	// Zero lookback should default to 10
	runner.Outputs[`git log --oneline --grep="feat: zero" -n 10`] = "zzz0000 feat: zero lookback\n"
	runner.Outputs["git rev-parse zzz0000"] = "zzz0000111\n"

	v := NewGitLogVerifierWithRunner(runner)
	v.LookbackCommits = 0 // Should default to 10

	spec := &models.CommitSpec{
		Type:    "feat",
		Message: "zero",
	}

	result, err := v.Verify(context.Background(), spec, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Found {
		t.Errorf("Expected Found to be true, mismatch: %s", result.Mismatch)
	}
}
