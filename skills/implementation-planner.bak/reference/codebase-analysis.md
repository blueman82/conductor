# Codebase Analysis Patterns

Comprehensive discovery patterns for understanding existing codebases and generating accurate implementation plans.

## Phase 1: Agent Discovery

**ALWAYS run agent discovery first:**

```bash
fd '\.md$' ~/.claude/agents --type f
```

Extract agent names by removing path and extension:
- `/Users/harrison/.claude/agents/golang-pro.md` → `golang-pro`
- `/Users/harrison/.claude/agents/test-automator.md` → `test-automator`

Store discovered agents for task assignment. If no agents found, use `general-purpose` as default.

## Phase 2: Greenfield vs Existing Project

Use `Glob` with pattern `"*"` to check current directory:

- **Empty or minimal files** (README, .git only) → Greenfield project
- **Existing codebase** → Proceed with thorough analysis

## Existing Codebase Analysis

### 1. Project Structure

```bash
# Directory structure
ls -la

# Identify key directories
Glob patterns:
  - "src/**/*"
  - "tests/**/*"
  - "docs/**/*"
  - "config/**/*"
```

**Look for:**
- Source code directory (`src/`, `lib/`, `pkg/`)
- Test directory (`tests/`, `__tests__/`, `*_test.go`)
- Documentation (`docs/`, `README.md`)
- Configuration (`config/`, `.env.example`)

### 2. Technology Stack Discovery

**Package managers:**
```bash
# JavaScript/TypeScript
Read: package.json, package-lock.json, tsconfig.json

# Python
Read: pyproject.toml, requirements.txt, setup.py, Pipfile

# Go
Read: go.mod, go.sum

# Ruby
Read: Gemfile, Gemfile.lock

# Rust
Read: Cargo.toml, Cargo.lock
```

**Configuration files:**
```bash
Glob patterns:
  - ".eslintrc*"
  - ".prettierrc*"
  - "tsconfig.json"
  - "pytest.ini"
  - "pyproject.toml"
  - ".golangci.yml"
```

**Identify:**
- Frameworks (React, Vue, Django, Rails, Express)
- Testing frameworks (Jest, pytest, Go testing)
- Linters/formatters (ESLint, Black, gofmt)
- Build tools (Webpack, Vite, Make)

### 3. Architecture Pattern Recognition

```bash
# Search for common patterns
Grep searches:
  - "class.*Controller" (MVC controllers)
  - "class.*Service" (Service layer)
  - "class.*Repository" (Repository pattern)
  - "interface.*Port" (Hexagonal/Clean)
  - "func New.*\(" (Go constructors)
```

**Look for:**
- MVC (models/, views/, controllers/)
- Clean Architecture (domain/, application/, infrastructure/)
- Layered (presentation/, business/, data/)
- Microservices (services/*, packages per service)

**Dependency injection:**
```bash
Grep patterns:
  - "constructor.*inject"
  - "@Injectable"
  - "container.register"
  - "func.*dependencies"
```

### 4. Testing Infrastructure

**Find test files:**
```bash
Glob patterns:
  - "**/*.test.*"
  - "**/*.spec.*"
  - "**/*_test.go"
  - "**/test_*.py"
```

**Read sample tests to identify:**
- Test framework (Jest, pytest, Go testing, RSpec)
- Mocking library (jest.mock, unittest.mock, gomock)
- Fixtures/factories (fixtures/, factories/, testdata/)
- Test utilities (test-utils/, helpers/)

**Test patterns:**
```bash
Read tests to find:
  - describe/it blocks (Jest, Mocha)
  - test_* functions (pytest)
  - func Test* (Go)
  - Arrange-Act-Assert structure
```

### 5. Code Quality Standards

**Linters/formatters:**
```bash
Glob: ".eslintrc*", ".prettierrc*", ".black", "ruff.toml"

Read configs to determine:
  - ESLint rules
  - Prettier config
  - Black line length
  - Go fmt requirements
```

**Naming conventions:**
```bash
Grep through codebase:
  - Variable naming (camelCase, snake_case)
  - File naming (kebab-case, PascalCase)
  - Test file naming (*.test.*, *_test.*)
```

**Error handling:**
```bash
Grep for patterns:
  - "try {" / "catch" (JS/TS exception handling)
  - "if err != nil" (Go error handling)
  - "except Exception" (Python exception handling)
```

**Logging:**
```bash
Grep for:
  - "console.log" / "logger."
  - "log.Printf" / "log.Error"
  - "logging.info" / "logger.debug"
```

### 6. File Path Verification (CRITICAL)

**BEFORE specifying file paths in tasks, ALWAYS verify:**

```bash
# Check if directories exist
ls -la internal/learning/migrations/ 2>/dev/null || echo "Directory does not exist"

# Find how similar features organize files
Glob: "**/migrations/*"
Glob: "**/*migration*"

# Check existing patterns for the feature type
Grep: "CREATE TABLE" to find where SQL lives
Read: existing similar files to see organization
```

**Common pitfalls to avoid:**
- Assuming `migrations/` directory exists (may use inline migrations)
- Assuming separate `.sql` files (may embed SQL in Go/Python)
- Assuming test files are in separate directories (may be co-located)
- Using version numbers that don't match existing patterns

**Verification checklist:**
1. ✓ Directory exists OR task creates it
2. ✓ File naming matches existing patterns
3. ✓ Similar features use same organization
4. ✓ Version/sequence numbers are correct

**Example - Migrations:**
```bash
# WRONG: Assuming separate SQL files
files: ["internal/learning/migrations/004_foo.sql"]

# RIGHT: Check first
ls internal/learning/migrations/  # Does it exist?
grep -r "CREATE TABLE" internal/learning/  # Where does SQL live?
# If inline: files: ["internal/learning/migration.go"]
```

### 7. Git History Analysis

```bash
git log --oneline -20
```

**Analyze for:**
- Commit message format (conventional commits, custom)
- Commit size (atomic vs large)
- Commit frequency
- Branch strategy (feature/, fix/)

```bash
# Check for contribution guide
Glob: "CONTRIBUTING.md", "DEVELOPMENT.md"
Read: contribution guidelines
```

## Greenfield Project Setup

If no existing codebase detected:

### 1. Technology Stack Decisions

**Document choices:**
- Language/framework selection and rationale
- Required dependencies and purposes
- Project structure to create
- Testing framework and approach

### 2. Initial Setup Tasks

**Create structure:**
```
src/           # Source code
tests/         # Test files
docs/          # Documentation
config/        # Configuration
scripts/       # Build/deploy scripts
```

**Config files needed:**
- `package.json` / `pyproject.toml` / `go.mod`
- `tsconfig.json` / `pytest.ini` / `.golangci.yml`
- `.eslintrc` / `.black` / `.editorconfig`
- `.gitignore`
- `README.md`

### 3. Foundation Commits

**Commit sequence:**
1. Initial scaffold with directory structure
2. Configuration files and tooling
3. README and documentation structure

### 4. Development Environment

**Required setup:**
- Tools and versions (Node 18+, Python 3.11+, Go 1.21+)
- Environment variables needed
- Setup verification steps

## Analysis Checklist

Use this checklist when analyzing codebases:

- [ ] Agent discovery completed
- [ ] Greenfield vs existing determined
- [ ] Directory structure mapped
- [ ] Technology stack identified
- [ ] Package managers and dependencies reviewed
- [ ] Test framework and patterns identified
- [ ] Code quality tools discovered
- [ ] Naming conventions documented
- [ ] Error handling patterns noted
- [ ] Logging approach understood
- [ ] Git history analyzed
- [ ] Similar features found (for patterns)
- [ ] Architecture pattern recognized
- [ ] Dependency injection method identified

## Pattern Matching Examples

### Finding Similar Features

```bash
# Search for existing auth implementation
Grep: "authentication", "login", "jwt"
Glob: "**/auth/**/*", "**/*auth*"

# Examine structure
Read: auth files to understand pattern
Note: file organization, naming, dependencies
```

### Understanding API Patterns

```bash
# Find API routes
Glob: "**/*.routes.*", "**/*Controller*", "**/handlers/*"

# Read sample route
Read: One route file completely
Note: routing pattern, middleware, error handling
```

### Test Pattern Discovery

```bash
# Find well-structured tests
Glob: "**/*.test.*"
Read: Highest quality test file
Note: Arrange-Act-Assert, mocking, fixtures
```

## Output Format

After analysis, document findings:

```markdown
## Codebase Analysis Results

**Technology Stack:**
- Framework: Next.js 14
- Language: TypeScript
- Testing: Jest + React Testing Library
- Linting: ESLint + Prettier

**Architecture:**
- Pattern: Feature-based modules
- Structure: src/features/{feature}/
- DI: React Context + hooks

**Testing Infrastructure:**
- Location: src/**/*.test.tsx
- Pattern: describe/it blocks
- Mocks: jest.mock()
- Fixtures: test-utils/fixtures/

**Code Quality:**
- Linter: ESLint (strict mode)
- Formatter: Prettier (80 char)
- Pre-commit: Husky + lint-staged

**Similar Features:**
- Authentication: src/features/auth/
- API pattern: src/features/*/api/route.ts
- Test pattern: src/features/auth/auth.test.tsx

**Recommendations:**
- Follow feature-based structure
- Use existing auth patterns for new auth features
- Maintain test coverage above 80%
```

This analysis informs accurate task generation, agent selection, and pattern following.
