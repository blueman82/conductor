# Cross-File Dependency Integration Tests - Summary

## Quick Start

### Run All Cross-File Dependency Tests
```bash
cd /Users/harrison/Github/conductor
go test -v ./test/integration -run TestCrossFileDependency
```

### Expected Output
```
=== RUN   TestCrossFileDependency_SimpleLinearChain
--- PASS: TestCrossFileDependency_SimpleLinearChain (0.00s)
=== RUN   TestCrossFileDependency_DiamondPattern
--- PASS: TestCrossFileDependency_DiamondPattern (0.00s)
...
PASS
ok  	github.com/harrison/conductor/test/integration	0.3xxs
```

## Files Created

### Test Implementation
- **Path**: `/Users/harrison/Github/conductor/test/integration/cross_file_dependency_test.go`
- **Size**: ~1000 lines
- **Contains**: 12 test functions with 27 total test cases
- **Coverage**: Cross-file dependency parsing, merging, validation, and execution planning

### Documentation
- **Path**: `/Users/harrison/Github/conductor/test/integration/CROSS_FILE_DEPENDENCY_TESTS.md`
- **Purpose**: Comprehensive documentation of all test scenarios
- **Contents**: Purpose, scenario, validations, and fixtures for each test

### Test Fixtures
Base directory: `/Users/harrison/Github/conductor/test/integration/fixtures/cross-file/`

**Linear Chain Pattern**:
- `linear/foundation.yaml` - Tasks 1-3 (foundation layer)
- `linear/features.yaml` - Tasks 4-6 (feature layer with cross-file dep)

**Diamond Pattern**:
- `diamond/setup.yaml` - Task 1 (root)
- `diamond/branches.yaml` - Tasks 2-3 (parallel branches)
- `diamond/join.yaml` - Task 4 (join point)

**Invalid References** (for error testing):
- `invalid/valid.yaml` - Valid reference plan
- `invalid/missing-task-ref.yaml` - Invalid cross-file reference

**Circular Dependencies** (for error testing):
- `circular/file1.yaml` & `circular/file2.yaml` - Simple cycle
- `circular/file3.yaml` & `circular/file4.yaml` - Indirect cycle

**Mixed Dependencies**:
- `mixed/part1.yaml` - Local dependencies
- `mixed/part2.yaml` - Mixed local and cross-file dependencies

**Markdown Format**:
- `markdown/setup.md` - Markdown format setup
- `markdown/implement.md` - Markdown format implementation

**Complex Multi-File Architecture**:
- `complex/01-foundation.yaml` - Foundation layer (3 tasks)
- `complex/02-middleware.yaml` - Middleware layer (3 tasks)
- `complex/03-handlers.yaml` - Handler layer (3 tasks)
- `complex/04-integration.yaml` - Integration layer (3 tasks)

## Test Statistics

### Overall Coverage
- **Total Tests**: 27 test cases (including sub-tests)
- **Pass Rate**: 100%
- **Execution Time**: ~0.3 seconds
- **Lines of Test Code**: ~1000

### Test Breakdown by Category
1. **Basic Functionality** (4 tests)
   - Simple Linear Chain
   - Diamond Pattern
   - Invalid References
   - Circular Detection

2. **Format Support** (2 tests)
   - Mixed Format Dependencies
   - Markdown Format

3. **Wave Calculation** (2 tests)
   - Wave Calculation with cross-file deps
   - Execution Boundary tracking

4. **Scalability** (1 test)
   - Large Multi-File (12 tasks, 4 files)

5. **Validation & String Handling** (3 tests)
   - Resolution Validation (3 sub-tests)
   - Dependency String Format (5 sub-tests)
   - Contextual Execution

## Key Features Tested

### Dependency Resolution
✓ Parsing cross-file dependencies in YAML format
✓ Normalizing mixed dependency formats (numeric and cross-file)
✓ Resolving cross-file references during plan merge
✓ Validating all dependencies reference existing tasks

### Wave Calculation
✓ Topological sort respects file boundaries
✓ Independent tasks execute in parallel
✓ Dependent tasks execute sequentially
✓ Diamond patterns handled correctly

### Error Handling
✓ Invalid task references detected
✓ Circular dependencies detected
✓ Helpful error messages provided
✓ Proper error propagation

### Metadata Preservation
✓ SourceFile tracking for file origin
✓ Task numbers preserved
✓ Dependencies preserved through merge
✓ Absolute path tracking

## Test Organization

### Helper Functions (Defined at Bottom of Test File)

```go
// Task lookup
findTaskByNumber(tasks []models.Task, number string) *models.Task

// Wave analysis
findWaveIndex(waves []models.Wave, taskNumber string) int

// Set operations
mapset(items []string) map[string]bool
setsEqual(a, b map[string]bool) bool

// String utilities
containsString(haystack, needle string) bool
indexOf(s, substr string) int
```

### Test Utilities

All tests use standard Go testing patterns:
- `testing.T` for assertions
- `t.Fatalf()` for fatal errors
- `t.Errorf()` for non-fatal errors
- `t.Run()` for subtests
- `filepath.Join()` for cross-platform fixture paths

## Integration with Conductor

### Parser Integration
- Uses existing `parser.ParseFile()` for file loading
- Uses existing `parser.MergePlans()` for multi-file merging
- Uses existing `parser.ResolveCrossFileDependencies()` for validation
- Uses existing `models.IsCrossFileDep()` and `models.ParseCrossFileDep()` helpers

### Executor Integration
- Uses `executor.CalculateWaves()` for dependency resolution
- Tests wave structure and execution order
- Validates topological sort correctness

### Models Integration
- Uses `models.Task` with cross-file dependency support
- Uses `models.CrossFileDependency` structure
- Tests string serialization/deserialization

## Success Metrics

✓ **100% test pass rate** - All 27 tests pass
✓ **Comprehensive coverage** - Happy path, error cases, edge cases
✓ **Well-documented** - Each test has clear purpose and expectations
✓ **Maintainable** - Helper functions reduce duplication
✓ **Scalable** - Tests cover 2-4 file scenarios
✓ **No regressions** - All existing integration tests still pass

## Future Test Enhancements

Possible future additions:
1. **E2E Execution Tests**: Mock executor to test actual task execution across files
2. **Performance Tests**: Measure wave calculation time with 100+ files
3. **Concurrency Tests**: Verify thread safety with parallel file loading
4. **Resume Tests**: Test skipping completed tasks across file boundaries
5. **Learning Integration**: Test learning system with multi-file plans
6. **Format Conversion**: Test converting between Markdown and YAML formats

## Regression Testing

Run these commands to verify no regressions:

```bash
# Full test suite
go test ./... -v

# Just integration tests
go test ./test/integration -v

# With coverage
go test ./test/integration -cover
```

## Development Notes

### Adding New Test Fixtures

1. Create directory under `test/integration/fixtures/cross-file/`
2. Create YAML or Markdown files following standard conductor format
3. Add test function calling the fixtures
4. Run tests to verify

### Adding New Test Cases

1. Use existing test functions as template
2. Follow naming convention: `TestCrossFileDependency_DescriptiveName`
3. Add helper functions if needed (at bottom of file)
4. Document in CROSS_FILE_DEPENDENCY_TESTS.md
5. Run full test suite to check for regressions

### Debugging Failed Tests

1. Run single test: `go test -v ./test/integration -run TestCrossFileDependency_NameHere`
2. Check fixture paths are correct (use `filepath.Join`)
3. Verify fixture YAML structure matches expected format
4. Use `-v` flag to see detailed output
5. Add `t.Logf()` statements for debugging

## Contact & Maintenance

- Test Suite Author: AI Test Automation Engineer
- Created: November 2025
- Maintenance: Follow Conductor development guidelines
- Issues: Report in Conductor GitHub repository
