# YAML Plan Structure Reference

Complete specification for conductor-compatible YAML implementation plans.

## Root Structure

```yaml
conductor:
  # Configuration for conductor orchestrator

plan:
  # Implementation plan metadata and tasks
```

## Conductor Configuration

### Default Agent

```yaml
conductor:
  default_agent: general-purpose
```

Fallback agent for tasks without explicit assignment. Use `general-purpose` as safe default.

### Quality Control

```yaml
conductor:
  quality_control:
    enabled: true
    review_agent: quality-control
    retry_on_red: 2

    agents:
      mode: auto  # auto | explicit | mixed
      explicit_list: []
      additional: []
      blocked: []
```

**Modes:**
- `auto`: File-type based selection (.go→golang-pro, .py→python-pro)
- `explicit`: Only use `explicit_list` agents
- `mixed`: Auto-selected + `additional` agents

### Worktree Groups (Organizational Metadata)

**Purpose:** Human organization and git workflow documentation. NOT used for execution control.

**Conductor uses `depends_on` for execution order**, not worktree groups.

```yaml
conductor:
  worktree_groups:
    - group_id: "auth-chain"
      description: "Tasks 1→2→4 implement auth flow"
      tasks: [1, 2, 4]
      rationale: "Related auth tasks grouped for git workflow"
      setup_commands: |
        # Create worktree
        git worktree add ../wt-chain-1 -b feature/<slug>/chain-1
        cd ../wt-chain-1

        # Execute tasks sequentially
        # Task 1 → Task 2 → Task 4

        # Merge when complete
        git checkout main
        git merge feature/<slug>/chain-1
```

**Execution Models:**
- `sequential`: Tasks execute in order (dependency chains)
- `parallel`: Tasks execute independently (no dependencies)

## Plan Metadata

```yaml
plan:
  metadata:
    feature_name: "Descriptive Feature Name"
    created: "YYYY-MM-DD"
    target: "Brief description of feature"
    estimated_tasks: <number>
```

## Context Section

```yaml
plan:
  context:
    framework: "Framework name (e.g., FastAPI, Next.js)"
    architecture: "Pattern (e.g., Clean Architecture, MVC)"
    test_framework: "Test framework (e.g., pytest, Jest)"
    other_context:
      - "Additional context point 1"
      - "Additional context point 2"
    expectations:
      - "Write tests BEFORE implementation (TDD)"
      - "Commit frequently"
      - "Follow existing patterns"
      - "YAGNI - avoid over-engineering"
      - "DRY - don't repeat yourself"
```

## Prerequisites

```yaml
plan:
  prerequisites:
    - item: "Required tools installed"
      details: "Node.js 18+, npm, git"
      verified: false

    - item: "Task 0: Environment validation (BLOCKING)"
      details: |
        MUST complete before implementation tasks.
        Python: mypy, pytest, black configured
        TypeScript: tsc, eslint, prettier configured
        Go: go vet, golangci-lint configured
      blocking: true
      commands:
        python:
          - "python -m mypy --version"
          - "python -m pytest --collect-only"
        typescript:
          - "npx tsc --version"
          - "npx eslint --version"
        go:
          - "go version"
          - "golangci-lint --version"
      verified: false
```

## Task Structure

### Minimum Required Fields

```yaml
tasks:
  - task_number: 1
    name: "Task name"
    agent: "agent-name"
    worktree_group: "chain-1"
    files:
      - "path/to/file1.ext"
      - "path/to/file2.ext"
    depends_on: []
    estimated_time: "30m"
    description: "What and why"
```

**CRITICAL:** `files` must be flat list of strings:
```yaml
# CORRECT
files:
  - "internal/auth/jwt.go"
  - "internal/auth/jwt_test.go"

# WRONG - nested structure not supported
files:
  source: ["jwt.go"]
  test: ["jwt_test.go"]
```

### Success Criteria (Task-Level Fields)

**REQUIRED:** These fields must be at task level (NOT nested under verification):

```yaml
tasks:
  - task_number: 1
    # ... other fields

    # Task-level fields for conductor QC parsing
    success_criteria:
      - "Specific measurable outcome"
      - "Verifiable condition"

    test_commands:
      - "go test ./internal/auth/ -v"
      - "npm test -- --grep 'auth'"
```

### Integration Tasks (Dual Criteria)

Integration tasks require BOTH criteria types:

```yaml
tasks:
  - task_number: 5
    type: integration  # REQUIRED marker
    name: "Wire auth to API router"
    depends_on: [1, 2, 3, 4]

    # Component-level validation
    success_criteria:
      - "Router accepts requests"
      - "Auth function has correct signature"

    # Cross-component validation
    integration_criteria:
      - "Auth middleware executes before handlers"
      - "Database transaction commits atomically"
      - "Error propagates end-to-end"
```

### Test-First Section

```yaml
test_first:
  test_file: "path/to/test.ext"

  structure:
    - "describe block for feature"
    - "test case 1: specific behavior"
    - "test case 2: edge case"

  mocks:
    - "dependency1"
    - "dependency2"

  fixtures:
    - "fixture1"

  assertions:
    - "outcome1"
    - "outcome2"

  edge_cases:
    - "edge case 1"
    - "edge case 2"

  example_skeleton: |
    # Language-specific test pattern
    # See quality-gates.md for complete examples
```

### Implementation Section

**Structure matches conductor parser:** `internal/parser/yaml.go:58-73`

**Simple example** (minimum: approach + key_points):
```yaml
implementation:
  approach: |
    Use dataclass with environment variable defaults.
    Initialize in __init__, validate required fields.

  key_points:
    - point: "Use os.getenv with defaults"
      details: "Allows environment override without breaking defaults"
      reference: "See existing config.py pattern"
```

**Complex example** (full structure with integration):
```yaml
implementation:
  approach: |
    Implement JWT middleware that runs before route handlers.
    Extract token from Authorization header, validate signature,
    attach user context to request.

  code_structure: |
    src/auth/middleware.py:
      - JWTMiddleware class
      - validate_token(token: str) -> User
      - attach_user_context(request, user)

    src/auth/exceptions.py:
      - InvalidTokenError
      - ExpiredTokenError

  key_points:
    - point: "Token validation happens before handler execution"
      details: "Middleware intercepts request, validates JWT, rejects invalid"
      reference: "FastAPI dependency injection pattern"

    - point: "Use HS256 algorithm for signing"
      details: "Symmetric key from environment, faster than RSA for API tokens"
      reference: "jwt.encode() with algorithm='HS256'"

    - point: "Attach user to request.state"
      details: "Makes authenticated user available to downstream handlers"
      reference: "request.state.user pattern from FastAPI docs"

  integration:
    imports:
      - "from jose import jwt"
      - "from fastapi import Request, HTTPException"

    services_to_inject:
      - "UserRepository (from task 2)"
      - "Config (from task 1)"

    config_values:
      - "JWT_SECRET_KEY - from environment"
      - "JWT_ALGORITHM - default HS256"
      - "JWT_EXPIRY_HOURS - default 24"

    error_handling:
      - "Invalid token → 401 Unauthorized with clear message"
      - "Expired token → 401 with 'Token expired' message"
      - "Missing token → 401 with 'Authorization required'"
      - "Database errors → 500 Internal Server Error"
```

**Field definitions:**

- `approach` (string): High-level strategy, architectural decisions, why this approach
- `code_structure` (string): File/class/function structure, pseudocode or outline
- `key_points` (array of objects):
  - `point` (required): Key decision or critical step
  - `details` (required): Why this matters, implications, context
  - `reference` (optional): File path, docs, or pattern to follow
- `integration` (object, use for tasks with dependencies):
  - `imports`: Required import statements
  - `services_to_inject`: Dependencies from other tasks
  - `config_values`: Configuration keys needed
  - `error_handling`: Error cases and responses

**When to use:**
- Always include `approach` + `key_points` minimum
- Add `code_structure` for complex implementations (3+ files, unclear architecture)
- Add `integration` subsection when task has `depends_on` (wires multiple components)

### Verification Section

```yaml
verification:
  automated_tests:
    command: "pytest tests/ -v"
    expected_output: "All tests passing"

  success_criteria:  # Human-readable (duplicates task-level field)
    - "All tests pass"
    - "Type checker passes"
    - "Code compiles"
```

### Code Quality Section (REQUIRED)

Every task MUST include language-specific quality pipeline:

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

  typescript:
    full_quality_pipeline:
      command: |
        npx prettier --write . && \
        npx tsc --noEmit && \
        npm test
      description: "Complete quality pipeline"
      exit_on_failure: true

  go:
    full_quality_pipeline:
      command: |
        gofmt -w . && \
        go vet ./... && \
        go test ./...
      description: "Complete quality pipeline"
      exit_on_failure: true
```

### Commit Section

```yaml
commit:
  type: "feat"  # feat | fix | test | refactor | docs | chore
  message: "descriptive message"
  body: "Optional explanation"
  files:
    - "path/to/file1.ext"
    - "path/to/file2.ext"
```

**NOTE:** `commit` is planning field. Conductor adds runtime fields:
- `status`: "pending" | "in_progress" | "completed"
- `completed_at`: "YYYY-MM-DD"
- `git_commit`: "actual-commit-hash"

## Testing Strategy

```yaml
testing_strategy:
  unit_tests:
    location: "tests/**/*.test.ts"
    naming_convention: "*.test.ts"
    run_command: "npm test"
    coverage_target: "80%"
    coverage_command: "npm run test:coverage"

  integration_tests:
    location: "tests/integration"
    what_to_test:
      - "API interactions"
      - "Database operations"
    setup_required:
      - "Test database"
      - "Mock services"
    run_command: "npm run test:integration"

  e2e_tests:
    enabled: true
    location: "tests/e2e"
    critical_flows:
      - "User registration flow"
    tools: "Playwright"
    run_command: "npm run test:e2e"
```

## Commit Strategy

```yaml
commit_strategy:
  total_commits: 5

  commits:
    - sequence: 1
      type: "test"
      description: "Add tests for feature X"
      files: ["tests/feature-x.test.ts"]
      why_separate: "TDD - tests first"

    - sequence: 2
      type: "feat"
      description: "Implement feature X"
      files: ["src/feature-x.ts"]
      why_separate: "Core implementation"
```

## Resources Section

```yaml
resources:
  existing_code:
    - type: "Similar feature"
      path: "src/features/similar/"
      note: "Study structure"

  documentation:
    - type: "Framework docs"
      link: "https://docs.example.com"
      relevance: "Core concepts"

  validation_checklist:
    - item: "All tests pass"
      command: "npm test"
      checked: false
```

## Field Placement Rules

### Task-Level vs Nested

**Task-level fields** (conductor parses directly):
- `success_criteria` - QC verification
- `test_commands` - Automated validation
- `integration_criteria` - Integration tasks only

**Nested fields** (human documentation):
- `verification.automated_tests`
- `verification.success_criteria` (duplicate)
- `test_first.structure`
- `implementation.approach`

### Reserved Runtime Fields

**DO NOT** manually define these (conductor manages):
- `status`
- `completed_at` / `completed_date`
- `git_commit`

Use `commit` object for planning, conductor adds `git_commit` after execution.

## YAML Best Practices

1. **Indentation**: 2 spaces (no tabs)
2. **Multiline**: Use `|` for literal, `>` for folded
3. **Quotes**: Quote strings with special chars (`:`, `#`, `@`)
4. **Lists**: `-` for items, consistent indent
5. **Validation**: Must parse with standard YAML parsers

## File Organization

**Single file** (1-15 tasks):
```
docs/plans/feature-slug.yaml
```

**Split files** (16+ tasks):
```
docs/plans/feature-slug/
├── plan-01-foundation.yaml
├── plan-02-integration.yaml
└── plan-03-testing.yaml
```

Split at ~2000 lines, worktree group boundaries, valid YAML per file.
