# Contributing to Conductor

Thank you for your interest in contributing to Conductor! This guide provides all the information you need to contribute effectively.

## Code of Conduct

Be respectful, inclusive, and constructive in all interactions.

## Getting Started

### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/conductor.git
cd conductor

# Add upstream remote
git remote add upstream https://github.com/harrison/conductor.git
```

### 2. Create a Feature Branch

```bash
# Update main
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/your-feature-name
# or for bug fixes:
git checkout -b fix/your-bug-name
```

### 3. Set Up Development Environment

See [SETUP.md](SETUP.md) for complete development setup instructions.

## Development Workflow

### Test-Driven Development

Conductor follows strict TDD principles:

1. **Write failing test first**
   ```go
   func TestNewFeature(t *testing.T) {
       // Test your feature
       result := NewFeature()
       expected := "something"
       if result != expected {
           t.Errorf("expected %s, got %s", expected, result)
       }
   }
   ```

2. **Implement minimal code to pass**
   ```go
   func NewFeature() string {
       return "something"
   }
   ```

3. **Refactor for clarity**
   - Improve variable names
   - Add documentation
   - Extract helper functions
   - Remove duplication

4. **Commit after each complete task**
   ```bash
   git add .
   git commit -m "feat: implement new feature with tests"
   ```

### Before Submitting

**1. Run All Tests**
```bash
make test
```

Coverage should remain at or above 70% (target: 85%+):
```bash
make test-coverage
```

**2. Format Code**
```bash
make fmt
```

**3. Run Linter**
```bash
make lint
```

If you don't have golangci-lint installed:
```bash
brew install golangci-lint
```

**4. Vet Code**
```bash
go vet ./...
```

**5. Test with Race Detection** (for concurrent code)
```bash
make test-race
```

## Commit Message Guidelines

Use conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build, dependencies, etc.

### Examples

```bash
git commit -m "feat(parser): add YAML frontmatter support"
git commit -m "fix(executor): handle timeout edge case in wave execution"
git commit -m "docs(readme): add learning system quick example"
git commit -m "test(learning): add pattern analyzer integration tests"
git commit -m "refactor(agent): simplify discovery registry logic"
```

## Pull Request Process

### 1. Prepare Your Branch

```bash
# Update from upstream
git fetch upstream
git rebase upstream/main

# Squash commits if needed
git rebase -i upstream/main
```

### 2. Push Your Branch

```bash
git push origin feature/your-feature-name
```

### 3. Create Pull Request

In GitHub:
- Use clear title describing the change
- Reference related issues (`Fixes #123`)
- Describe what changed and why
- Include test results/coverage

### 4. Respond to Review

- Address feedback promptly
- Discuss disagreements respectfully
- Update branch after making changes:
  ```bash
  git add .
  git commit -m "Address review feedback"
  git push origin feature/your-feature-name
  ```

## Areas for Contribution

### High Priority

- Bug fixes (see Issues labeled `bug`)
- Performance optimizations
- Test coverage improvements
- Documentation enhancements
- User-facing features

### Feature Ideas

- Additional export formats (e.g., HTML, PDF for plans)
- Enhanced metrics and reporting
- Improved error messages
- Additional learning insights
- CI/CD integration examples
- Plan validation enhancements

### Documentation

- Example plans for different use cases
- Tutorial improvements
- API documentation
- Troubleshooting guides
- Architecture deep-dives

## Testing Guidelines

### Test Coverage

Target: 85%+ for critical paths, 70%+ overall

Check coverage:
```bash
go test ./... -cover
make test-coverage
```

### Writing Tests

Use table-driven tests for multiple scenarios:

```go
func TestParser(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Plan
        wantErr bool
    }{
        {
            name:  "valid markdown plan",
            input: "## Task 1: Setup\n**Files**: file.go",
            want:  &Plan{Tasks: []Task{{Name: "Setup"}}},
        },
        {
            name:    "invalid format",
            input:   "invalid",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("wantErr %v, got %v", tt.wantErr, err)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("want %v, got %v", tt.want, got)
            }
        })
    }
}
```

### Test Organization

- Test files alongside source: `foo.go` → `foo_test.go`
- Fixtures in `internal/{package}/testdata/`
- Mock interfaces for dependencies
- Use helper functions for common setup

## Code Style

### Go Conventions

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting (automated with `make fmt`)
- Prefer clear names over clever code
- Add comments for exported functions

### Examples

```go
// Good: Clear, descriptive naming
type WaveExecutor struct {
    maxConcurrency int
    tasks []Task
}

func (we *WaveExecutor) Execute(ctx context.Context) error {
    // Implementation
}

// Avoid: Unclear abbreviations
type WE struct {
    mc int
}
```

### Documentation

- All exported functions need doc comments
- Include examples for complex functions
- Document parameters and return values
- Add architecture notes for complex packages

```go
// WaveExecutor executes tasks in parallel waves based on dependencies.
// Tasks within a wave execute concurrently up to maxConcurrency limit.
func (we *WaveExecutor) Execute(ctx context.Context) error {
    // ...
}
```

## Project Structure Guidelines

When adding new functionality:

1. **Minimal Dependencies**: Keep external dependencies minimal
2. **Interfaces**: Use interfaces for testability
3. **Package Organization**: One responsibility per package
4. **Naming**: Clear, descriptive package and function names

Example package structure:
```
internal/newfeature/
├── feature.go       # Core functionality
├── feature_test.go  # Tests
└── testdata/        # Test fixtures
```

## Debugging Tips

### Verbose Logging

Enable verbose output during testing:
```bash
go test -v ./internal/executor/
```

### Add Debug Output

Use `log.Printf` during development (remove before committing):
```go
log.Printf("DEBUG: wave %d has %d tasks", waveNum, len(tasks))
```

### Debugging Tests

Run single test with verbose output:
```bash
go test -v ./internal/executor/ -run TestWaveCalculation
```

### Coverage Analysis

Generate and view coverage:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Performance Considerations

- Conductor should feel responsive for plans up to 100+ tasks
- Keep task execution synchronous (Claude CLI invocation)
- Use goroutines for wave parallelization only
- Profile before optimizing: `go test -bench ./...`

## Security Considerations

- Don't commit secrets (API keys, credentials)
- Validate user input from plan files
- Sanitize file paths to prevent traversal attacks
- Use `context.WithTimeout` for all external calls

## Version Management

Don't manually edit VERSION file. Use Make targets:

```bash
make build-patch   # 1.0.0 → 1.0.1
make build-minor   # 1.0.0 → 1.1.0
make build-major   # 1.0.0 → 2.0.0
```

The VERSION file is the single source of truth.

## Questions or Need Help?

- Check [SETUP.md](SETUP.md) for development setup
- See [CLAUDE.md](CLAUDE.md) for architecture details
- Review [docs/](docs/) for complete documentation
- Open an issue for questions or clarification

## License

By contributing, you agree that your contributions will be licensed under the same MIT License as the project.

Thank you for contributing to Conductor!
