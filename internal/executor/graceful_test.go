package executor

import (
	"testing"

	"github.com/harrison/conductor/internal/models"
)

// gracefulTestLogger captures calls for testing the graceful helpers
type gracefulTestLogger struct {
	warnfCalls []string
	infofCalls []string
}

func (m *gracefulTestLogger) Warnf(format string, args ...interface{}) {
	m.warnfCalls = append(m.warnfCalls, format)
}

func (m *gracefulTestLogger) Infof(format string, args ...interface{}) {
	m.infofCalls = append(m.infofCalls, format)
}

func (m *gracefulTestLogger) Info(message string) {}

func (m *gracefulTestLogger) Debugf(format string, args ...interface{}) {}

func (m *gracefulTestLogger) LogTestCommands(entries []models.TestCommandResult) {}

func (m *gracefulTestLogger) LogCriterionVerifications(entries []models.CriterionVerificationResult) {}

func (m *gracefulTestLogger) LogDocTargetVerifications(entries []models.DocTargetResult) {}

func (m *gracefulTestLogger) LogErrorPattern(pattern interface{}) {}

func (m *gracefulTestLogger) LogDetectedError(detected interface{}) {}

func TestGracefulWarn_NilLogger(t *testing.T) {
	// Should not panic with nil logger
	GracefulWarn(nil, "test message: %v", "error")
}

func TestGracefulWarn_WithLogger(t *testing.T) {
	logger := &gracefulTestLogger{}
	GracefulWarn(logger, "test message: %v", "error")

	if len(logger.warnfCalls) != 1 {
		t.Errorf("expected 1 Warnf call, got %d", len(logger.warnfCalls))
	}
	if logger.warnfCalls[0] != "test message: %v" {
		t.Errorf("expected format 'test message: %%v', got %q", logger.warnfCalls[0])
	}
}

func TestGracefulInfo_NilLogger(t *testing.T) {
	// Should not panic with nil logger
	GracefulInfo(nil, "test message: %v", "value")
}

func TestGracefulInfo_WithLogger(t *testing.T) {
	logger := &gracefulTestLogger{}
	GracefulInfo(logger, "test message: %v", "value")

	if len(logger.infofCalls) != 1 {
		t.Errorf("expected 1 Infof call, got %d", len(logger.infofCalls))
	}
	if logger.infofCalls[0] != "test message: %v" {
		t.Errorf("expected format 'test message: %%v', got %q", logger.infofCalls[0])
	}
}
