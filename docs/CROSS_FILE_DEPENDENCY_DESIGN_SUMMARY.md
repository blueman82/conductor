# Cross-File Dependency Support: Design Complete Summary

**Status**: Design Phase Complete - Ready for Implementation
**Date**: 2025-11-23
**Total Deliverables**: 3 comprehensive documents + 8 test fixtures

---

## Overview

This summary consolidates the complete test design and implementation roadmap for Conductor's cross-file dependency feature. The feature enables tasks in split/multi-file plans to reference tasks in other plan files using an explicit, unambiguous syntax.

---

## What's Included

### 1. Comprehensive Test Plan
**File**: `docs/CROSS_FILE_DEPENDENCY_TEST_PLAN.md`

**Contents**:
- Feature overview and design principles
- Dependency format specification (numeric, cross-file, mixed)
- Unit test design (4 test categories, 20+ test cases)
- Integration test design (5 test categories, 10+ scenarios)
- Test fixtures specification (5 fixture types)
- Error cases (6 comprehensive error scenarios)
- Coverage targets (85%+ overall)

**Test Coverage**: 49+ unit/integration tests specified

---

### 2. Practical Examples & Usage Guide
**File**: `docs/CROSS_FILE_DEPENDENCY_EXAMPLES.md`

**Contents**:
- Quick start guide (problem → solution)
- 3 basic examples (linear, diamond, subdirectories)
- 3 advanced patterns (multi-stage, conditional, refactoring)
- 4 common issues with troubleshooting
- Testing guide for fixtures
- Best practices and naming conventions
- Migration guide from single-file to multi-file

**Practical Value**: Ready-to-use examples developers can copy

---

### 3. Implementation Roadmap
**File**: `docs/CROSS_FILE_DEPENDENCY_IMPLEMENTATION_ROADMAP.md`

**Contents**:
- 4-phase implementation plan (4 weeks, 84-118 hours)
- Detailed task breakdown with effort estimates
- Specific file locations and code structure
- 49+ test specifications with implementation code
- Risk assessment and mitigation strategies
- Success criteria and exit conditions

**Implementation Phases**:
- Phase 1: Core Parsing (18-28h)
- Phase 2: File Resolution & Validation (28-36h)
- Phase 3: Wave Calculation & Executor (20-26h)
- Phase 4: CLI Integration & Polish (18-28h)

---

### 4. Test Fixtures
**Location**: `internal/parser/testdata/cross-file-fixtures/`

**Fixture Sets**:
1. **Linear Chain**: 2 files, 4 tasks, sequential
2. **Diamond Pattern**: 3 files, 4 tasks, parallel execution
3. **Complex Subdirectories**: 4 files, nested structure, relative paths
4. **Markdown Format**: 2 files, Markdown syntax
5. **Mixed Format**: 2 files, numeric + cross-file dependencies

**Total**: 8 complete, ready-to-use fixture files

---

## Key Design Decisions

### 1. Dependency Format

**Chosen Format** (backward compatible):
```yaml
depends_on:
  - 1                                  # Local numeric (same file)
  - 2
  - file: "plan-02-features.yaml"     # Cross-file (explicit)
    task: 3
```

**Rationale**:
- Numeric-only remains compatible with existing code
- Cross-file uses explicit object structure (unambiguous)
- Clear distinction: numeric = local, object = cross-file

---

### 2. Path Resolution

**Strategy**: Relative paths resolved to absolute based on current file location

```yaml
# In plan-02/plan-03-api.yaml
depends_on:
  - file: "../plan-01-foundation.yaml"  # Relative to plan-02/
    task: 1
```

---

### 3. Validation Levels

**Single-File**: Checks numeric deps, warnings on cross-file refs
**Multi-File**: Checks all cross-file refs, errors on missing files/tasks

---

## Test Coverage Summary

| Category | Tests | Coverage |
|----------|-------|----------|
| Parsing | 13+ | 90% |
| Validation | 25 | 85% |
| Executor | 11 | 80% |
| CLI | - | 75% |
| **Total** | **49+** | **85%+** |

---

## Implementation Timeline

| Week | Phase | Hours | Focus |
|------|-------|-------|-------|
| 1 | Parsing | 18-28h | Mixed format parser |
| 2 | Validation | 28-36h | File resolution & validation |
| 3 | Executor | 20-26h | Wave calculation |
| 4 | Polish | 18-28h | CLI & documentation |
| **TOTAL** | | **84-118h** | **Feature complete** |

---

## Files to Create/Modify

**New Files**:
```
internal/models/dependency.go
internal/parser/yaml_dependencies.go
internal/parser/markdown_dependencies.go
internal/parser/file_resolution.go
internal/parser/dependency_normalization.go
internal/parser/validation.go
```

**Modified Files**:
```
internal/parser/yaml.go
internal/executor/graph.go
internal/cmd/validate.go
```

---

## Resources Provided

### Documentation
1. `CROSS_FILE_DEPENDENCY_TEST_PLAN.md` - Comprehensive testing strategy
2. `CROSS_FILE_DEPENDENCY_EXAMPLES.md` - Practical usage guide
3. `CROSS_FILE_DEPENDENCY_IMPLEMENTATION_ROADMAP.md` - Implementation plan

### Test Fixtures (8 files)
- `split-plan-linear/` - 2 files
- `split-plan-diamond/` - 3 files
- `split-plan-complex/` - 4 files
- `split-plan-markdown/` - 2 files
- `split-plan-mixed/` - 2 files

---

## Success Criteria

✓ All 49+ tests passing
✓ 85%+ code coverage
✓ No breaking changes
✓ Clear error messages
✓ <100ms performance for typical plans
✓ Documentation complete

---

## Next Steps

1. **Design Review** - Review all three documents, approve format
2. **Implementation Kickoff** - Set up environment, create branches
3. **Phase 1 Execution** - Implement parsing, target 90% coverage
4. **Phases 2-4** - Follow roadmap, daily tracking, weekly reviews

---

## Document References

All detailed specifications available in:
- Test strategies: `CROSS_FILE_DEPENDENCY_TEST_PLAN.md`
- Usage examples: `CROSS_FILE_DEPENDENCY_EXAMPLES.md`
- Implementation: `CROSS_FILE_DEPENDENCY_IMPLEMENTATION_ROADMAP.md`

**Status**: Design complete, ready for implementation.

