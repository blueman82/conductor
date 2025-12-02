# Quality Gates Reference

Language-specific quality pipelines, common pitfalls, and enforcement patterns.

## Quality Pipeline by Language

### Python

```yaml
code_quality:
  python:
    full_quality_pipeline:
      command: |
        python -m black . && \
        python -m mypy src/ && \
        python -m pytest
      description: "Complete quality pipeline"
      exit_on_failure: true
```

**Common Pitfalls:**
- Using `from src.` imports (use `from bot.` with PYTHONPATH)
- Missing `__aexit__` in async context managers
- Not initializing critical IDs (e.g., `bot_user_id`)
- No error handling in async event handlers
- Using `typing.Dict/List` instead of `dict/list` (Python 3.9+)
- mypy not configured or not running

### TypeScript

```yaml
code_quality:
  typescript:
    full_quality_pipeline:
      command: |
        npx prettier --write . && \
        npx tsc --noEmit && \
        npm test
      description: "Complete quality pipeline"
      exit_on_failure: true
```

**Common Pitfalls:**
- Missing null checks with optional properties
- Using `any` type instead of proper types
- Not using optional chaining (`?.`)
- Missing nullish coalescing (`??`)

### Go

```yaml
code_quality:
  go:
    full_quality_pipeline:
      command: |
        gofmt -w . && \
        go vet ./... && \
        go test ./...
      description: "Complete quality pipeline"
      exit_on_failure: true
```

**Common Pitfalls:**
- Not checking errors (`_` instead of `err`)
- Goroutine leaks (not closing channels/contexts)
- Missing context cancellation handling
- Race conditions (run `go test -race`)

## Cross-Language Pitfalls

### Not Running Formatters

```bash
Python: black . && ruff check .
TypeScript: prettier --write . && eslint --fix .
Go: gofmt -w . && goimports -w .
```

### Tests Not in CI

```bash
# Check CI config exists
ls .github/workflows/ .gitlab-ci.yml

# Verify test commands present
cat .github/workflows/*.yml | grep -i test
```

### Hardcoded Paths/Credentials

```bash
grep -r '/Users/' .
grep -r '/home/' .
grep -r 'password.*=' .
grep -r 'api_key.*=' .
```

## Complete Test Patterns

### Python (pytest + async)

```python
import pytest
from bot.handlers import MessageHandler  # No 'from src.'
from bot.models import Message

@pytest.mark.asyncio
async def test_handler_success():
    """Test handler processes messages."""
    # Arrange
    handler = MessageHandler()
    message = Message(content="test", author_id=123)

    # Act
    result = await handler.process(message)

    # Assert
    assert result.status == "success"

@pytest.mark.asyncio
async def test_handler_error_handling():
    """Test graceful error handling."""
    handler = MessageHandler()

    # Should NOT raise
    try:
        result = await handler.process(None)
        assert result.status == "error"
    except Exception as e:
        pytest.fail(f"Should catch errors: {e}")
```

### TypeScript (Jest)

```typescript
import { MessageHandler } from '../src/handlers/MessageHandler';
import { Message } from '../src/models/Message';

describe('MessageHandler', () => {
  let handler: MessageHandler;

  beforeEach(() => {
    handler = new MessageHandler();
  });

  it('should process valid messages', async () => {
    // Arrange
    const message: Message = {
      content: 'test',
      authorId: 123,
      timestamp: new Date()
    };

    // Act
    const result = await handler.process(message);

    // Assert
    expect(result).toBeDefined();
    expect(result.status).toBe('success');
  });

  it('should handle null gracefully', async () => {
    // Act
    const result = await handler.process(null as any);

    // Assert
    expect(result.status).toBe('error');
  });
});
```

### Go (standard testing)

```go
package handlers

import (
    "context"
    "testing"
)

func TestMessageHandler_ProcessSuccess(t *testing.T) {
    // Arrange
    handler := NewMessageHandler()
    ctx := context.Background()
    message := &Message{Content: "test", AuthorID: 123}

    // Act
    result, err := handler.Process(ctx, message)

    // Assert
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    if result.Status != "success" {
        t.Errorf("expected success, got: %s", result.Status)
    }
}

func TestMessageHandler_ErrorHandling(t *testing.T) {
    handler := NewMessageHandler()

    // Act
    _, err := handler.Process(context.Background(), nil)

    // Assert - should return error, not panic
    if err == nil {
        t.Error("expected error for nil message")
    }
}
```

See doc-yaml.md lines 1170-1365 for complete pattern examples with error handling, async context managers, and enforcement.
