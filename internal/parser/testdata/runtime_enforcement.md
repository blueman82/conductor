# Runtime Enforcement Test Fixture

Tests PlannerComplianceSpec, TaskMetadataRuntime, DataFlowRegistry, and all enforcement modes

Enforcement modes demonstrated:
- dependency_checks: Preflight commands before agent invocation
- test_commands: Post-agent tests (hard gate, blocks on failure)
- documentation_targets: Doc section verification (soft signal)
- success_criteria[].verification: Criterion verification (soft signal)
- package_guard: Go package isolation (via depends_on serialization)

## Task 1: Task with full runtime metadata

**Type**: component
**Estimated Time**: 30m
**Agent**: golang-pro
**Status**: pending
**Files**: internal/models/plan.go, internal/models/task.go

**Success Criteria**:
- All new fields serialize via YAML round-trip
  - Verification Command: `go test -run TestYAMLSerialization ./internal/parser`
  - Expected: PASS
  - Description: Verifies YAML serialization preserves runtime metadata
- Parser populates TaskMetadataRuntime from YAML
- Validation rejects missing metadata when strict_enforcement=true
  - Verification Command: `go test -run TestStrictValidation ./internal/validation`
  - Expected: PASS
  - Description: Ensures validation works with enforcement enabled
- Empty runtime_metadata blocks are valid (not rejected)

**Test Commands**:
```bash
go test ./internal/parser -run RuntimeEnforcement -v
go test ./internal/models -run Task -v
```

**Key Points**:
1. Establish runtime enforcement patterns for Conductor v2.9+
   - Reference: internal/executor/preflight.go
   - Impact: High - foundational enforcement implementation

2. Maintain backward compatibility with plans lacking runtime_metadata
   - Note: Plans without metadata should continue to work
   - Impact: High - prevents breaking existing plans

3. Implement preflight checks as soft signal for documentation
   - Reference: docs/runtime.md
   - Impact: Medium - enables gradual adoption

### Dependency Checks:
- Verify build succeeds before task
- Verify test compilation
- Verify Go module exists

### Documentation Targets:
- Configuration section in docs/runtime.md
- Unreleased section in CHANGELOG.md

### Prompt Blocks:
- This task establishes runtime enforcement patterns for Conductor v2.9+
- Must maintain backward compatibility with plans lacking runtime_metadata
- See internal/executor/preflight.go for enforcement implementation

---

## Task 2: Task with minimal runtime metadata

**Type**: component
**Estimated Time**: 20m
**Agent**: golang-pro
**Status**: pending
**Files**: internal/parser/yaml.go
**Depends on**: Task 1

**Success Criteria**:
- Parser handles minimal runtime_metadata
- Empty documentation_targets array is valid
- Empty prompt_blocks array is valid

**Test Commands**:
```bash
go test ./internal/parser -run YAML -v
```

### Dependency Checks:
- Static analysis check with go vet

### Documentation Targets:
(empty - no documentation targets for this task)

### Prompt Blocks:
(empty - no prompt blocks for this task)

---

## Task 3: Integration task demonstrating dual criteria

**Type**: integration
**Estimated Time**: 25m
**Agent**: golang-pro
**Status**: pending
**Files**: internal/executor/preflight.go, internal/executor/task.go
**Depends on**: Task 1, Task 2

**Success Criteria**:
- RunDependencyChecks executes all checks in order
  - Verification Command: `go test -run TestRunDependencyChecks ./internal/executor`
  - Expected: PASS
  - Description: Validates preflight check execution
- VerifyDocumentationTargets returns structured results
- RunCriterionVerifications collects all verification outputs

**Integration Criteria**:
- Preflight dependency checks complete before agent invocation
- Test commands execute after agent output and block on failure
- Criterion verification results flow into QC prompt context
- Documentation target results flow into QC prompt context
- Package guard prevents concurrent modifications to same Go package

**Test Commands**:
```bash
go test ./internal/executor -run Enforcement -v
go test ./test/integration -run RuntimeEnforcement -v
```

**Key Points**:
1. Wire preflight checks into the task execution pipeline
   - Reference: docs/runtime.md
   - Impact: High - critical integration point

2. Ensure all enforcement modes work together cohesively
   - Note: Dependency checks, test commands, and criterion verification must coordinate
   - Impact: High - system reliability

3. Document integration flow between enforcement components
   - Reference: internal/executor/task.go
   - Impact: Medium - maintainability

### Dependency Checks:
- Verify executor package builds

### Documentation Targets:
- Integration section in docs/runtime.md

### Prompt Blocks:
- This task wires preflight checks into the task execution pipeline
