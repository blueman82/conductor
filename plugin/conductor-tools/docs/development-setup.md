# Development Environment Setup Guide

This guide provides comprehensive development environment validation for Python, TypeScript, and Go projects. Reference this in your implementation plan's prerequisites section.

## Purpose

Validate that the development environment is correctly configured BEFORE starting implementation tasks. This prevents common setup issues that cause wasted time and frustration during development.

---

## Python Environment Setup

### Type Checking (mypy)

**Tool**: `mypy`
**Config File**: `pyproject.toml` or `mypy.ini`

**Required Settings:**
- `mypy_path`: `"src"` (or appropriate source directory)
- `python_version`: Matching runtime version (e.g., `"3.11"`)
- `strict_optional`: `true`
- `warn_return_any`: `true`
- `disallow_untyped_defs`: `true` (for new code)

**Validation Command:**
```bash
python -m mypy src/
```

**Expected Output:**
```
Success: no issues found in N source files
```

**Config Example (`pyproject.toml`):**
```toml
[tool.mypy]
python_version = "3.11"
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true
mypy_path = "src"

[[tool.mypy.overrides]]
module = "tests.*"
disallow_untyped_defs = false
```

### Testing (pytest)

**Tool**: `pytest`
**Config File**: `pyproject.toml` or `pytest.ini`

**Required Settings:**
- `pythonpath`: `["src"]` (ensures imports work from test files)
- `testpaths`: `["tests"]`
- `python_files`: `["test_*.py"]`
- `python_classes`: `["Test*"]`
- `python_functions`: `["test_*"]`

**Validation Command:**
```bash
python -m pytest --collect-only
```

**Expected Output:**
```
collected N items
```

**Config Example (`pyproject.toml`):**
```toml
[tool.pytest.ini_options]
pythonpath = ["src"]
testpaths = ["tests"]
python_files = ["test_*.py"]
python_classes = ["Test*"]
python_functions = ["test_*"]
addopts = "-v --strict-markers --cov=src --cov-report=term-missing"
```

### Project Structure

**Required Files:**
- `pyproject.toml` (or `setup.py`)
- `requirements.txt` or `poetry.lock`
- `tests/conftest.py` (shared fixtures)
- `.gitignore` (including `__pycache__`, `.venv`)

**Import Pattern (CRITICAL):**
```python
# ✅ CORRECT import pattern (assuming src/ in PYTHONPATH)
from bot.handlers import MessageHandler
from bot.models import User
from utils.logging import setup_logger

# ❌ AVOID these patterns:
from src.bot.handlers import MessageHandler  # Don't include 'src' prefix
import sys; sys.path.append('../src')  # Don't manipulate sys.path
```

**Validation Commands:**
```bash
# Verify imports work
python -c "from bot.handlers import MessageHandler"

# Verify tests can import
cd tests && python -c "from bot.handlers import MessageHandler"
```

### Formatting and Linting

**Formatter**: `black`

**Config Example (`pyproject.toml`):**
```toml
[tool.black]
line-length = 100
target-version = ['py311']
```

**Linter**: `ruff`

**Config Example (`pyproject.toml`):**
```toml
[tool.ruff]
select = ["E", "F", "I", "N", "W"]
line-length = 100
target-version = "py311"
```

**Validation Commands:**
```bash
python -m black --check .
python -m ruff check .
```

---

## TypeScript Environment Setup

### Type Checking (tsc)

**Tool**: TypeScript compiler
**Config File**: `tsconfig.json`

**Required Settings:**
- `strict`: `true`
- `noImplicitAny`: `true`
- `strictNullChecks`: `true`
- `esModuleInterop`: `true`
- `skipLibCheck`: `false`

**Validation Command:**
```bash
npx tsc --noEmit
```

**Expected Output:**
```
No errors
```

**Config Example (`tsconfig.json`):**
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "esModuleInterop": true,
    "skipLibCheck": false,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "outDir": "./dist",
    "rootDir": "./src"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

### Testing (Jest/Vitest)

**Tool**: Jest or Vitest
**Config File**: `jest.config.js` or `vitest.config.ts`

**Validation Command:**
```bash
npm test -- --listTests
```

**Expected Output:**
```
List of test files
```

### Linting (ESLint)

**Tool**: ESLint
**Config File**: `.eslintrc.js` or `eslint.config.js`

**Required Rules:**
- `@typescript-eslint/no-explicit-any`
- `@typescript-eslint/no-non-null-assertion`
- `@typescript-eslint/strict-boolean-expressions`

**Validation Command:**
```bash
npx eslint . --ext .ts,.tsx
```

**Config Example (`.eslintrc.js`):**
```javascript
module.exports = {
  parser: '@typescript-eslint/parser',
  plugins: ['@typescript-eslint'],
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking'
  ],
  rules: {
    '@typescript-eslint/no-explicit-any': 'error',
    '@typescript-eslint/no-non-null-assertion': 'error'
  }
};
```

---

## Go Environment Setup

### Type Checking (go vet)

**Tool**: `go vet`

**Validation Command:**
```bash
go vet ./...
```

**Expected Output:**
```
(No output = success)
```

### Module Setup

**Required File**: `go.mod`

**Validation Command:**
```bash
go list -m
```

**Expected Output:**
```
github.com/user/project
```

**Initialization (if needed):**
```bash
go mod init github.com/user/project
```

### Formatting

**Tools**: `gofmt` and `goimports`

**Validation Commands:**
```bash
# Should output nothing if all formatted
gofmt -l .
goimports -l .
```

**Auto-fix:**
```bash
gofmt -w .
goimports -w .
```

### Linting (golangci-lint)

**Tool**: `golangci-lint`
**Config File**: `.golangci.yml`

**Validation Command:**
```bash
golangci-lint run
```

**Config Example (`.golangci.yml`):**
```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
run:
  timeout: 5m
  tests: false
```

---

## Quick Validation Checklist

Use this checklist to validate your environment is ready:

### Python
- [ ] `python -m mypy --version` works
- [ ] `python -m pytest --collect-only` works
- [ ] `cat pyproject.toml | grep mypy_path` shows correct path
- [ ] Test imports work: `python -c "from <module> import <class>"`

### TypeScript
- [ ] `npx tsc --version` works
- [ ] `npx eslint --version` works
- [ ] `cat tsconfig.json | grep strict` shows `true`
- [ ] `npm test` discovers tests

### Go
- [ ] `go version` works
- [ ] `golangci-lint --version` works
- [ ] `go list -m` shows module path
- [ ] `go test ./...` runs

---

## Troubleshooting

### Python: "ModuleNotFoundError"
**Solution**: Verify `pythonpath` in `pyproject.toml` or `PYTHONPATH` environment variable

### TypeScript: "Cannot find module"
**Solution**: Check `tsconfig.json` paths configuration and `node_modules` installation

### Go: "package not found"
**Solution**: Run `go mod tidy` to sync dependencies

---

## References

- See `ai_docs/quality-gates-reference.yaml` for quality gate pipelines
- See `ai_docs/enforcement-guide.md` for pre-commit hook setup
