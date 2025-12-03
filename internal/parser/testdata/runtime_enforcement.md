# Runtime Enforcement Comprehensive Test Plan

## Task 1: Task with full runtime metadata

**Type**: component
**Estimated Time**: 30m
**Agent**: golang-pro
**Status**: pending
**Files**: internal/models/plan.go, internal/models/task.go

**Success Criteria**:
- All new fields serialize via YAML round-trip
  and maintain data integrity across conversions
- Parser populates TaskMetadataRuntime from YAML
- Validation rejects missing metadata when strict_enforcement=true
  and provides helpful error messages
- Empty runtime_metadata blocks are valid (not rejected)

**Test Commands**:
```bash
go test ./internal/parser -run RuntimeEnforcement -v
go test ./internal/models -run Task -v
```

**Structured Criteria**:
- All new fields serialize via YAML round-trip
  - Verification Command: `go test -run TestYAMLSerialization ./internal/parser`
  - Expected: PASS
  - Description: Verifies YAML serialization preserves runtime metadata

- Validation rejects missing metadata when strict_enforcement=true
  - Verification Command: `go test -run TestStrictValidation ./internal/validation`
  - Expected: PASS
  - Description: Confirms strict mode enforcement works correctly

### Dependency Checks:
- Verify build succeeds before task execution
- Verify test compilation for models package
- Confirm Go module exists and is properly configured

### Documentation Targets:
- API reference in docs/runtime.md configuration section
- Architecture diagram in CHANGELOG.md unreleased section

### Prompt Blocks:
- This task establishes runtime enforcement patterns for Conductor v2.9+
- Must maintain backward compatibility with plans lacking runtime_metadata

**Key Points**:
1. All new fields must serialize via YAML round-trip
   - Reference: internal/models/task.go
   - Impact: High - core serialization requirement

2. Parser must populate TaskMetadataRuntime from YAML
   - Note: Essential for runtime enforcement pipeline
   - Reference: internal/parser/yaml.go

3. Backward compatibility is critical
   - Impact: High - existing plans must continue working
   - Reference: internal/executor/task.go

---

## Task 2: Task with minimal runtime metadata

**Type**: component
**Estimated Time**: 20m
**Agent**: golang-pro
**Status**: pending
**Files**: internal/parser/yaml.go

**Success Criteria**:
- Parser handles minimal runtime_metadata
- Empty documentation_targets array is valid
- Empty prompt_blocks array is valid
  and does not cause parsing errors

**Test Commands**:
- go test ./internal/parser -run YAML -v
- go vet ./...

### Dependency Checks:
- Static analysis check for package consistency

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
  and reports results accurately
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

**Structured Criteria**:
- RunDependencyChecks executes all checks in order
  - Verification Command: `go test -run TestRunDependencyChecks ./internal/executor`
  - Expected: PASS
  - Description: Validates preflight check execution order and collection

- Documentation target verification returns results
  - Verification Command: `go test -run TestDocumentationTargets ./internal/executor`
  - Expected: PASS
  - Description: Confirms soft signal collection for documentation targets

### Dependency Checks:
- Verify executor package builds successfully
- Confirm all dependencies resolved

### Documentation Targets:
- Runtime enforcement integration guide in docs/runtime.md
- Integration patterns reference

### Prompt Blocks:
- This task wires preflight checks into the task execution pipeline
- Integration verification demonstrates cross-component enforcement

**Key Points**:
1. Preflight checks run before agent invocation
   - Reference: internal/executor/preflight.go
   - Impact: High - critical execution flow requirement

2. Test commands enforce hard gates post-agent
   - Note: Block execution on failure, not just warnings
   - Reference: internal/executor/wave.go

3. Verification results flow through QC context
   - Impact: High - affects quality control feedback
   - Reference: internal/executor/qc.go
