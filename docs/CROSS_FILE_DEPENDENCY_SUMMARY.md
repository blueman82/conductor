# Cross-File Dependency Support - Design Summary

## Overview

This design document set provides a complete technical specification for implementing cross-file dependency support in Conductor v2.5+ using the annotation-based approach (Option 2).

---

## What's Included

### 1. CROSS_FILE_DEPENDENCY_DESIGN.md
**Main technical design document (15 sections)**

Comprehensive design covering:
- Data structure design (DependencyReference, NormalizedDependency types)
- YAML and Markdown parser implementation changes
- Validation and error handling strategy
- Plan merging and cross-file resolution
- Graph building integration
- Full backward compatibility approach
- Phase-based implementation plan
- Performance and security considerations
- Future enhancements (v2.6+)

Key sections:
- Section 1: Core data types for dependency representation
- Section 2: Parser implementation (YAML and Markdown)
- Section 3: Validation strategy (single-file vs multi-file)
- Section 4: Dependency graph integration
- Section 5-6: Implementation phases and YAML format examples
- Section 7-8: Error handling with detailed messages
- Section 9: Backward compatibility guarantees
- Section 10: Comprehensive testing strategy

### 2. CROSS_FILE_DEPENDENCY_IMPLEMENTATION.md
**Practical implementation guide with code snippets**

Step-by-step implementation guide including:
- Complete Go struct definitions ready to copy-paste
- Helper function implementations (parseDependencies, normalizeFilePath, etc.)
- Parser modifications with before/after code
- Plan merger changes with detailed comments
- Graph validator updates
- Test file examples and test cases

Perfect for developers implementing the feature.

### 3. CROSS_FILE_EXAMPLES.md
**Practical YAML examples and use cases**

Real-world examples showing:
- Simple two-file plan with cross-file dependencies
- Mixed format dependencies (numeric + cross-file in one array)
- Complex three-file microservices architecture
- Error scenarios and how to fix them
- Best practices for file organization
- Testing multi-file plans
- Troubleshooting guide

---

## Key Design Decisions

### Format Choice: Annotation-Based

```yaml
# Numeric (existing, backward compatible)
depends_on: [2, 3, 4]

# Cross-file reference (new)
depends_on:
  - file: "plan-01-foundation.yaml"
    task: 2

# Mixed format (new)
depends_on:
  - 2                                   # Numeric (local)
  - file: "plan-01-foundation.yaml"    # Cross-file
    task: 3
```

**Why this format?**
- Explicit and clear intent
- Self-documenting (no guessing which file)
- Type-safe with structured YAML
- Extensible for future enhancements
- Compatible with existing numeric format

### Backward Compatibility Approach

1. **DependsOn field unchanged** - Always contains string task numbers
2. **No breaking changes** - Existing plans work without modification
3. **Graceful degradation** - Single-file plans with cross-file refs just warn
4. **Two-step resolution** - Parsing creates references; merge resolves them

### Validation Strategy

**Single-File Plans:**
- Numeric dependencies validated during parsing
- Cross-file references are warned but not enforced
- No changes needed for existing single-file plans

**Multi-File Plans:**
- Cross-file references validated during merge
- File mapping built for O(1) lookup resolution
- Cycle detection includes cross-file dependencies

---

## Core Implementation Points

### 1. Data Structures (models/task.go)

```go
// New types
type DependencyReference struct {
    Type       string // "local" or "cross-file"
    TaskNumber string // For local refs
    File       string // For cross-file refs
    Task       string // For cross-file refs
}

// Extended Task struct
type Task struct {
    DependsOn     []string              // Existing field (unchanged semantically)
    DependsOnRaw  []interface{}         // Internal: raw parsing data
    CrossFileDependencies map[string]string // File:task -> resolved task number
    DependencyReferences []DependencyReference // Parsed references
}
```

### 2. Parser Changes (parser/yaml.go)

```go
// New functions
func parseDependencies(rawDeps []interface{}, task *models.Task, currentFile string) error
func isValidTaskNumber(taskNum string) bool
func normalizeFilePath(filePath, currentFile string) (string, error)
```

### 3. Plan Merging (parser/parser.go)

Modified MergePlans() to:
1. Build file-to-task mapping during collection
2. Resolve cross-file references using the mapping
3. Add resolved task numbers to DependsOn array
4. Return fully resolved plan ready for execution

### 4. Graph Building (executor/graph.go)

Modified BuildDependencyGraph() to:
1. Call task.NormalizedDependsOn() to get all dependencies
2. Build graph with both local and resolved cross-file refs
3. ValidateTasks() validates all dependencies exist

---

## Error Handling

All errors include:
1. **What went wrong** - Clear problem description
2. **Where it happened** - Task number and file
3. **What's valid** - Examples of correct format
4. **How to fix it** - Actionable steps

**Example:**
```
Error: task 5 (User API): cross-file reference to task 2 in plan-01-foundation.yaml not found
Available tasks in plan-01-foundation.yaml: 1, 3, 4
Hint: Verify task number exists in the referenced file
```

---

## Files That Need Modification

### Core Changes (5 files)
1. **internal/models/task.go**
   - Add DependencyReference type
   - Add NormalizedDependency type
   - Extend Task struct with 3 new fields
   - Add 3 helper methods

2. **internal/parser/yaml.go**
   - Add parseDependencies() function
   - Add isValidTaskNumber() function
   - Add normalizeFilePath() function
   - Update YAMLParser.Parse() to use new parser

3. **internal/parser/parser.go**
   - Update MergePlans() to build fileToTask map
   - Add cross-file resolution logic
   - Better error messages

4. **internal/executor/graph.go**
   - Update ValidateTasks() signature
   - Update BuildDependencyGraph() to use NormalizedDependsOn()
   - Add ValidateCrossFileDependencies() function

5. **internal/parser/markdown.go**
   - Update parseTaskMetadata() documentation
   - No functional changes needed for MVP

### Test Files (4 new files)
1. **internal/parser/cross_file_yaml_test.go** - YAML parsing tests
2. **internal/parser/cross_file_integration_test.go** - MergePlans tests
3. **internal/executor/cross_file_graph_test.go** - Graph building tests
4. **test/integration/cross_file_e2e_test.go** - End-to-end tests

---

## Implementation Timeline

### Phase 1: Data Structures (2-3 hours)
- Add types to models/task.go
- Add helper methods
- Document new fields

### Phase 2: YAML Parser (3-4 hours)
- Implement parseDependencies()
- Implement validation helpers
- Update YAMLParser.Parse()
- Write unit tests

### Phase 3: Plan Merging (2-3 hours)
- Update MergePlans() for resolution
- Add error handling
- Write merge tests

### Phase 4: Graph Integration (1-2 hours)
- Update BuildDependencyGraph()
- Update ValidateTasks()
- Write graph tests

### Phase 5: Testing & Polish (3-4 hours)
- Create test data files
- Write integration tests
- Test error scenarios
- Update documentation

**Total: 11-16 hours (1-2 days for experienced developer)**

---

## Testing Strategy

### Unit Tests (40+ tests)
- YAML parsing: numeric, structured, cross-file, mixed formats
- Validation: invalid task numbers, invalid paths, etc.
- File path normalization: relative paths, rejected absolute paths
- Plan merging: resolution, unresolved refs, duplicate tasks

### Integration Tests (15+ tests)
- Multi-file merge with cross-file deps
- Dependency graph with cross-file deps
- Cycle detection with cross-file deps
- Error scenarios and recovery

### End-to-End Tests (5+ tests)
- Parse → Merge → Validate → Calculate Waves → Execute
- Verify cross-file deps in wave calculation
- Verify correct execution order

### Test Coverage Target
- **Critical paths**: 95%+ (dependency resolution)
- **Parsers**: 85%+ (multiple input formats)
- **Overall**: 90%+ (maintain project standard)

---

## Backward Compatibility Guarantees

### Existing Single-File Plans
✓ **No changes required** - continue to work as-is
```yaml
depends_on: [2, 3, 4]  # Works without modification
```

### Existing Multi-File Plans (split-plan-large-*.yaml)
✓ **Fully compatible** - cross-file refs are optional
```yaml
depends_on: [4, 5]     # Still works for inter-file deps
# Can now also use:
depends_on:
  - 4
  - file: "plan-01.yaml"
    task: 2
```

### Existing Code
✓ **Fully compatible** - graph building unchanged
- Code reading task.DependsOn works unchanged
- New helper method NormalizedDependsOn() available for clarity
- BuildDependencyGraph() handles both formats transparently

---

## Performance Impact

- **Parsing**: +1-2ms per plan file (one-time cost)
- **File path normalization**: Cached, not recomputed
- **Cross-file resolution**: O(1) map lookup during merge
- **Graph building**: No change to Kahn's algorithm complexity
- **Memory**: Minimal (DependsOnRaw cleared after parsing)

**Negligible impact on overall execution time**

---

## Security Considerations

1. **Path Validation**
   - Absolute paths rejected (must be relative)
   - filepath.Abs() prevents directory escape
   - Validates relative to current file's directory only

2. **Task Number Validation**
   - Alphanumeric only (no special characters)
   - Prevents injection attacks
   - Type validated in YAML parsing

3. **File Access**
   - Files must be readable (handled by OS)
   - No executing untrusted files
   - Validation before execution

---

## Future Enhancements (v2.6+)

1. **Markdown Support for Cross-File Refs**
   - Currently YAML-only for MVP
   - Future: Support markdown list syntax

2. **File Aliasing**
   - Shorter syntax for frequently referenced files
   - Example: `task: 2` in aliased file context

3. **Package-Level Dependencies**
   - Depend on all tasks in a file
   - Example: `file: "plan-01.yaml"` (no task number)

4. **Transitive Dependencies**
   - Automatically include indirect dependencies
   - Example: If A depends on B, and B depends on C, A gets C

5. **Dependency Groups**
   - Group related cross-file dependencies
   - Better organization and documentation

6. **Visualization**
   - Graph visualization showing cross-file dependencies
   - Export to GraphViz or similar

---

## Document Organization

| Document | Purpose | Audience |
|----------|---------|----------|
| CROSS_FILE_DEPENDENCY_DESIGN.md | Complete technical specification | Architects, Senior Engineers |
| CROSS_FILE_DEPENDENCY_IMPLEMENTATION.md | Code snippets and implementation guide | Developers implementing feature |
| CROSS_FILE_EXAMPLES.md | YAML examples and best practices | Users and developers |
| CROSS_FILE_DEPENDENCY_SUMMARY.md (this) | Quick reference and overview | Project managers, reviewers |

---

## Implementation Checklist

### Preparation
- [ ] Review all design documents thoroughly
- [ ] Understand existing code structure
- [ ] Set up test environment

### Phase 1: Data Structures
- [ ] Add types to models/task.go
- [ ] Add helper methods to Task
- [ ] Update Task struct documentation

### Phase 2: YAML Parser
- [ ] Implement helper functions
- [ ] Update YAMLParser.Parse()
- [ ] Write unit tests for parsing

### Phase 3: Plan Merging
- [ ] Update MergePlans() function
- [ ] Add validation functions
- [ ] Write merge tests

### Phase 4: Graph Integration
- [ ] Update BuildDependencyGraph()
- [ ] Update ValidateTasks()
- [ ] Write graph tests

### Phase 5: Testing & Documentation
- [ ] Create test data files
- [ ] Write integration tests
- [ ] Test error messages
- [ ] Update CLAUDE.md

### Pre-Release
- [ ] Run full test suite
- [ ] Verify coverage targets met
- [ ] Manual testing with complex plans
- [ ] Performance validation
- [ ] Security review

---

## Key Files Reference

### Design Documents (This Directory)
- `CROSS_FILE_DEPENDENCY_DESIGN.md` - Main design doc (15 sections, 700+ lines)
- `CROSS_FILE_DEPENDENCY_IMPLEMENTATION.md` - Code implementation guide (500+ lines)
- `CROSS_FILE_EXAMPLES.md` - YAML examples and use cases (300+ lines)
- `CROSS_FILE_DEPENDENCY_SUMMARY.md` - This summary document

### Implementation Files

**Models:**
- `internal/models/task.go` - Task struct and dependency types

**Parsers:**
- `internal/parser/yaml.go` - YAML parsing with dependency handling
- `internal/parser/parser.go` - Plan merging and cross-file resolution
- `internal/parser/markdown.go` - Markdown parsing (minimal changes)

**Execution:**
- `internal/executor/graph.go` - Dependency graph building

**Tests:**
- `internal/parser/cross_file_yaml_test.go` - YAML parsing tests
- `internal/parser/cross_file_integration_test.go` - Merge tests
- `internal/executor/cross_file_graph_test.go` - Graph tests
- `test/integration/cross_file_e2e_test.go` - E2E tests

---

## Success Criteria

The implementation is complete when:

1. **Backward Compatibility**: Existing plans work without modification
2. **Format Support**: All three formats parse correctly
   - Numeric: `[2, 3, 4]`
   - Structured local: `[{task: 2}]`
   - Cross-file: `[{file: "...", task: 2}]`
   - Mixed: All above in same array
3. **Validation**: Clear error messages for all failure cases
4. **Resolution**: Cross-file references resolve to task numbers during merge
5. **Execution**: Dependency graph includes cross-file dependencies
6. **Testing**: 90%+ test coverage with comprehensive scenarios
7. **Documentation**: Design, implementation, and examples documented

---

## Quick Start for Implementers

1. **Read in order:**
   - This summary (you are here)
   - CROSS_FILE_DEPENDENCY_DESIGN.md (Sections 1-4)
   - CROSS_FILE_DEPENDENCY_IMPLEMENTATION.md (Part 1-3)

2. **Implement in order:**
   - Part 1: Models (1-2 hours)
   - Part 2: YAML Parser (3-4 hours)
   - Part 3: Plan Merging (2-3 hours)
   - Part 4: Graph Integration (1-2 hours)

3. **Test as you go:**
   - Unit tests for each component
   - Integration tests after merging
   - E2E tests for complete pipeline

4. **Reference during coding:**
   - CROSS_FILE_DEPENDENCY_IMPLEMENTATION.md for code snippets
   - CROSS_FILE_EXAMPLES.md for test data examples

---

## Questions & Answers

**Q: Will this break existing plans?**
A: No. All existing plans continue to work without modification. The numeric format is fully backward compatible.

**Q: Can I mix numeric and cross-file references?**
A: Yes. The design supports mixed formats in a single depends_on array.

**Q: What if I reference a task that doesn't exist?**
A: Clear error message: "task X in plan-Y.yaml not found. Available tasks: A, B, C"

**Q: Can I use absolute paths?**
A: No, only relative paths for security. This prevents directory escape attacks.

**Q: Will cross-file dependencies affect performance?**
A: Negligible. Resolution is O(1) map lookup during merge (one-time cost).

**Q: Is this the final design?**
A: This is the MVP design for v2.5. Future enhancements planned for v2.6+ (markdown support, file aliasing, package-level deps, etc.)

---

## Conclusion

This design provides a robust, backward-compatible implementation of cross-file dependency support in Conductor. The annotation-based format is clear, self-documenting, and extensible. Implementation is straightforward with detailed code examples provided.

The phased approach allows for incremental testing and validation. Error handling ensures users get clear guidance when something goes wrong. Full backward compatibility means no migration needed for existing plans.

**Status**: Ready for implementation (v2.5 target)
