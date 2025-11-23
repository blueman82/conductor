# Quality Enforcement Guide

This guide provides comprehensive enforcement mechanisms for code quality standards through pre-commit hooks and CI/CD gates. Reference this in your implementation plans to automate quality checks.

## Purpose

Automated enforcement of code quality standards to prevent common errors from reaching production. These mechanisms should be set up at the start of the project and enforced consistently.

---

## Pre-Commit Hooks

Git pre-commit hooks run automatically before every commit, preventing commits that violate quality standards. These are local checks that provide immediate feedback to developers.

### Python Setup (pre-commit framework)

**Tool**: `pre-commit` framework

**Installation:**
```bash
# Install pre-commit
pip install pre-commit

# Add pre-commit to requirements-dev.txt
echo "pre-commit>=3.5.0" >> requirements-dev.txt
```

**Configuration File**: `.pre-commit-config.yaml`

**Configuration Example:**
```yaml
# .pre-commit-config.yaml
repos:
  # Code formatting with black
  - repo: https://github.com/psf/black
    rev: 23.10.0
    hooks:
      - id: black
        language_version: python3.11

  # Import sorting
  - repo: https://github.com/pycqa/isort
    rev: 5.12.0
    hooks:
      - id: isort
        args: ["--profile", "black"]

  # Linting with ruff
  - repo: https://github.com/astral-sh/ruff-pre-commit
    rev: v0.1.4
    hooks:
      - id: ruff
        args: ["--fix"]

  # Type checking with mypy
  - repo: https://github.com/pre-commit/mirrors-mypy
    rev: v1.6.0
    hooks:
      - id: mypy
        additional_dependencies: [types-all]
        args: [--config-file=pyproject.toml]

  # Check for 'from src.' imports (CRITICAL)
  - repo: local
    hooks:
      - id: check-src-imports
        name: Check for incorrect 'from src.' imports
        entry: bash -c 'if grep -r "from src\." . --include="*.py"; then echo "ERROR: Found \"from src.\" imports. Use proper module imports."; exit 1; fi'
        language: system
        pass_filenames: false

  # Check for missing __aexit__ in async context managers
  - repo: local
    hooks:
      - id: check-async-context-managers
        name: Check async context managers have both __aenter__ and __aexit__
        entry: bash -c 'files=$(grep -l "async def __aenter__" $(find . -name "*.py")); for f in $files; do if ! grep -q "__aexit__" "$f"; then echo "ERROR: $f has __aenter__ but missing __aexit__"; exit 1; fi; done'
        language: system
        pass_filenames: false
```

**Setup Commands:**
```bash
# Install git hooks
pre-commit install

# Run against all files (one time)
pre-commit run --all-files

# Hooks now run automatically on every commit
```

**Bypass (Emergency Only):**
```bash
# RARE: Only bypass for emergency hotfixes
git commit --no-verify -m "hotfix: critical security patch"
```

### TypeScript Setup (husky + lint-staged)

**Tools**: `husky` + `lint-staged`

**Installation:**
```bash
# Install husky and lint-staged
npm install --save-dev husky lint-staged

# Initialize husky
npx husky install

# Add postinstall script to package.json
npm pkg set scripts.prepare="husky install"
```

**Configuration File**: `package.json`

**Configuration Example:**
```json
{
  "lint-staged": {
    "*.{ts,tsx}": [
      "prettier --write",
      "eslint --fix",
      "bash -c 'tsc --noEmit'"
    ],
    "*.{json,md,yml}": [
      "prettier --write"
    ]
  }
}
```

**Hook File**: `.husky/pre-commit`

**Hook Content:**
```bash
#!/usr/bin/env sh
. "$(dirname -- "$0")/_/husky.sh"

# Run lint-staged
npx lint-staged

# Run tests if any test files changed
git diff --cached --name-only | grep -q '\.test\.ts$' && npm test
```

**Setup Commands:**
```bash
# Create pre-commit hook
npx husky add .husky/pre-commit "npx lint-staged"

# Make executable
chmod +x .husky/pre-commit
```

### Go Setup (git pre-commit hook script)

**Hook File**: `.git/hooks/pre-commit`

**Hook Content:**
```bash
#!/bin/sh
# Go pre-commit hook

# Format code
echo "Running gofmt..."
gofmt -w .

echo "Running goimports..."
goimports -w .

# Re-add formatted files
git add -u

# Run go vet
echo "Running go vet..."
if ! go vet ./...; then
  echo "go vet failed. Fix issues before committing."
  exit 1
fi

# Run tests
echo "Running tests..."
if ! go test ./...; then
  echo "Tests failed. Fix tests before committing."
  exit 1
fi

# Run race detector
echo "Running race detector..."
if ! go test -race ./...; then
  echo "Race conditions detected. Fix before committing."
  exit 1
fi

echo "Pre-commit checks passed!"
```

**Setup Commands:**
```bash
# Create hook file
cat > .git/hooks/pre-commit << 'EOF'
[paste hook_content above]
EOF

# Make executable
chmod +x .git/hooks/pre-commit

# Test hook
./.git/hooks/pre-commit
```

---

## CI/CD Gates

Continuous Integration checks that run on every push and pull request. These enforce quality standards at the repository level and prevent merging code that violates standards.

### GitHub Actions

#### Python Workflow

**File**: `.github/workflows/python-quality.yml`

**Content:**
```yaml
name: Python Quality Checks
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install -r requirements-dev.txt

      - name: Check formatting with black
        run: python -m black --check .

      - name: Check imports
        run: |
          if grep -r "from src\." . --include="*.py"; then
            echo "ERROR: Found 'from src.' imports"
            exit 1
          fi

      - name: Type check with mypy
        run: python -m mypy src/

      - name: Lint with ruff
        run: python -m ruff check .

      - name: Run tests with coverage
        run: python -m pytest --cov=src --cov-report=xml --cov-report=term

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.xml

      - name: Check coverage threshold
        run: |
          coverage report --fail-under=80
```

#### TypeScript Workflow

**File**: `.github/workflows/typescript-quality.yml`

**Content:**
```yaml
name: TypeScript Quality Checks
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Check formatting with prettier
        run: npx prettier --check .

      - name: Type check with tsc
        run: npx tsc --noEmit

      - name: Lint with eslint
        run: npx eslint . --ext .ts,.tsx

      - name: Run tests with coverage
        run: npm test -- --coverage

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
```

#### Go Workflow

**File**: `.github/workflows/go-quality.yml`

**Content:**
```yaml
name: Go Quality Checks
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Check formatting
        run: |
          if [ -n "$(gofmt -l .)" ]; then
            echo "Code not formatted. Run: gofmt -w ."
            exit 1
          fi

      - name: Run go vet
        run: go vet ./...

      - name: Run staticcheck
        uses: dominikh/staticcheck-action@v1.3.0
        with:
          version: "2023.1.6"

      - name: Run tests
        run: go test -v ./...

      - name: Run tests with race detector
        run: go test -race ./...

      - name: Run tests with coverage
        run: go test -coverprofile=coverage.out ./...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
```

### Branch Protection Rules

Configure branch protection rules in GitHub/GitLab to enforce quality gates before code can be merged.

**GitHub Settings:**
- Location: `Settings > Branches > Branch protection rules`

**Required Settings:**
- "Require pull request reviews before merging"
- "Require status checks to pass before merging"
- "Require branches to be up to date before merging"
- "Require conversation resolution before merging"

**Status Checks to Require:**
- Python: `Python Quality Checks / quality`
- TypeScript: `TypeScript Quality Checks / quality`
- Go: `Go Quality Checks / quality`

**Additional Restrictions:**
- "Require linear history"
- "Do not allow bypassing the above settings"

---

## Blocking vs Warning Philosophy

Clear policy on which checks block commits/merges and which are warnings.

### Blocking Checks

**Description**: These MUST pass - no exceptions without documented justification

**Local (pre-commit):**
- Code formatting (black, prettier, gofmt)
- Type checking (mypy, tsc, go vet)
- Critical pattern violations (from src. imports, missing __aexit__)
- Unit tests (pytest, npm test, go test)

**CI/CD:**
- All pre-commit checks
- Integration tests
- Race detection (for Go)
- Coverage threshold (80%+)
- Security vulnerability scanning

### Warning Checks

**Description**: Should fix but don't block - reviewed during PR

**Local (pre-commit):**
- Linting suggestions (ruff, eslint, golangci-lint)
- Code complexity warnings
- TODO/FIXME comments

**CI/CD:**
- Performance benchmarks (if degraded)
- Documentation coverage
- Dependency updates available

### Exemption Process

**When Needed:**
- Emergency hotfixes (security vulnerabilities)
- Third-party library incompatibilities
- Temporary workarounds (must have follow-up issue)

**How to Request:**
1. Document reason in commit message or PR description
2. Add inline code comment explaining exemption
3. Create follow-up issue to resolve properly
4. Get approval from tech lead or senior engineer

**Example Commit Message:**
```
fix(auth): bypass rate limit for admin endpoints

EXEMPTION: Disabling eslint rule @typescript-eslint/no-explicit-any
for AdminRequest type due to third-party library (auth-provider v2.1)
not providing proper type definitions.

Follow-up: Issue #1234 to add custom type definitions for auth-provider
```

---

## References

- See `ai_docs/quality-gates-reference.yaml` for gate definitions
- See `ai_docs/development-setup.md` for environment configuration
