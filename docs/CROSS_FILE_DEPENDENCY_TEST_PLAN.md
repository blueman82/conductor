# Cross-File Dependency Support: Comprehensive Test Plan

**Status**: Design Phase
**Version**: 1.0
**Date**: 2025-11-23

## Executive Summary

This document provides comprehensive test coverage design for Conductor's cross-file dependency feature. The feature enables tasks in split plans (multiple plan files) to reference tasks in other files using mixed dependency formats:

```yaml
depends_on:
  - 3                                    # Local numeric (current file)
  - file: "plan-01-foundation.yaml"     # Cross-file reference
    task: 2
```

This test plan covers unit tests, integration tests, error handling, backward compatibility, and edge cases across both Markdown and YAML formats.

## Table of Contents

1. [Feature Overview](#feature-overview)
2. [Dependency Format Specification](#dependency-format-specification)
3. [Unit Test Design](#unit-test-design)
4. [Integration Test Design](#integration-test-design)
5. [Test Fixtures](#test-fixtures)
6. [Error Cases](#error-cases)
7. [Coverage Targets](#coverage-targets)

---

## Feature Overview

### What Cross-File Dependencies Enable

- **Modular Plans**: Split implementation plans across multiple files (setup.md, features.md, deployment.md)
- **File References**: Tasks in one file can depend on tasks in other files
- **Backward Compatibility**: Single-file numeric dependencies still work
- **Clear Dependency Graph**: Conductor resolves all dependencies across files before calculating execution waves
- **Better Organization**: Large plans become more maintainable and readable

### Design Principles

1. **Backward Compatible**: Numeric-only `depends_on` continues to work
2. **Explicit Format**: Cross-file refs are structurally distinct (object with `file` and `task` fields)
3. **File Resolution**: Paths resolved relative to the plan directory
4. **Validation Early**: Errors caught during parsing, not execution
5. **Clear Error Messages**: Users understand what file/task cannot be resolved

---

## Dependency Format Specification

### Format Variants

#### 1. Local Numeric (Current File)
```yaml
depends_on:
  - 1
  - 2
  - 3
```
**Resolution**: Task numbers interpreted as local task IDs within current file.

#### 2. Cross-File Reference
```yaml
depends_on:
  - file: "plan-01-setup.yaml"
    task: 2
  - file: "./deployment/plan-03-infra.yaml"
    task: 5
```
**Resolution**:
- `file` path resolved relative to current plan file directory
- `task` matches task number in referenced file
- Error if file doesn't exist or task not found

#### 3. Mixed Format
```yaml
depends_on:
  - 1                                    # Local: Task 1 in same file
  - file: "plan-02-features.yaml"       # Cross-file: Task 3 in plan-02
    task: 3
  - 5                                    # Local: Task 5 in same file
```
**Resolution**: Parse each entry, determine if numeric or object, resolve accordingly.

#### 4. Markdown Format (if supported)

Current Markdown format:
```markdown
**Depends on**: Task 1, Task 2, Task 3
```

Extended Markdown format for cross-file:
```markdown
**Depends on**: Task 1, [Task 2 from plan-02-features.yaml], Task 3
**Cross-file deps**: plan-02-features.yaml:Task 2
```
*Alternative syntax to be designed during implementation.*

### Format Validation Rules

1. **Numeric entries**: Single integer or "Task N" format
2. **Object entries**: Must have both `file` and `task` fields
3. **File paths**:
   - Can be relative (relative to current file) or absolute
   - Must refer to existing files
   - No symlink cycles allowed
4. **Task references**:
   - Task ID must exist in referenced file
   - Can be numeric, alphanumeric, or "Task N" format
5. **No circular references**: Enforced by existing cycle detection

---

## Unit Test Design

### Test Module: `internal/parser/dependency_parsing_test.go`

#### Test Category 1: Parsing Mixed Dependency Formats

**Test Suite**: `TestParseMixedDependencies`

```go
tests := []struct {
    name           string
    input          []interface{}        // Raw YAML input
    wantDeps       []models.Dependency   // Expected parsed dependencies
    wantErr        bool
    errContains    string
}{
    {
        name: "numeric only dependencies",
        input: []interface{}{1, 2, 3},
        wantDeps: []models.Dependency{
            {Type: "local", TaskID: "1"},
            {Type: "local", TaskID: "2"},
            {Type: "local", TaskID: "3"},
        },
        wantErr: false,
    },
    {
        name: "single cross-file dependency",
        input: []interface{}{
            map[string]interface{}{
                "file": "plan-02-features.yaml",
                "task": 3,
            },
        },
        wantDeps: []models.Dependency{
            {Type: "cross-file", File: "plan-02-features.yaml", TaskID: "3"},
        },
        wantErr: false,
    },
    {
        name: "mixed local and cross-file",
        input: []interface{}{
            1,
            map[string]interface{}{
                "file": "plan-02-features.yaml",
                "task": 2,
            },
            3,
        },
        wantDeps: []models.Dependency{
            {Type: "local", TaskID: "1"},
            {Type: "cross-file", File: "plan-02-features.yaml", TaskID: "2"},
            {Type: "local", TaskID: "3"},
        },
        wantErr: false,
    },
    {
        name: "cross-file with relative path",
        input: []interface{}{
            map[string]interface{}{
                "file": "./deployment/plan-03-infra.yaml",
                "task": "infra-5",
            },
        },
        wantDeps: []models.Dependency{
            {Type: "cross-file", File: "./deployment/plan-03-infra.yaml", TaskID: "infra-5"},
        },
        wantErr: false,
    },
    {
        name: "missing file field in cross-file dep",
        input: []interface{}{
            map[string]interface{}{
                "task": 2,
                // Missing "file" field
            },
        },
        wantErr: true,
        errContains: "cross-file dependency missing required field",
    },
    {
        name: "missing task field in cross-file dep",
        input: []interface{}{
            map[string]interface{}{
                "file": "plan-02-features.yaml",
                // Missing "task" field
            },
        },
        wantErr: true,
        errContains: "cross-file dependency missing required field",
    },
    {
        name: "invalid object in depends_on (extra fields)",
        input: []interface{}{
            map[string]interface{}{
                "file": "plan-02.yaml",
                "task": 2,
                "extra": "invalid", // Extra fields should be ignored
            },
        },
        wantDeps: []models.Dependency{
            {Type: "cross-file", File: "plan-02.yaml", TaskID: "2"},
        },
        wantErr: false,
    },
    {
        name: "numeric string in local dependency",
        input: []interface{}{"1", "2", "3"},
        wantDeps: []models.Dependency{
            {Type: "local", TaskID: "1"},
            {Type: "local", TaskID: "2"},
            {Type: "local", TaskID: "3"},
        },
        wantErr: false,
    },
    {
        name: "alphanumeric task IDs in cross-file",
        input: []interface{}{
            map[string]interface{}{
                "file": "plan-02.yaml",
                "task": "auth-setup",
            },
        },
        wantDeps: []models.Dependency{
            {Type: "cross-file", File: "plan-02.yaml", TaskID: "auth-setup"},
        },
        wantErr: false,
    },
}
```

#### Test Category 2: File Path Resolution

**Test Suite**: `TestResolveFilePathDependency`

```go
tests := []struct {
    name          string
    baseFile      string              // Current plan file path
    refFile       string              // Referenced file path
    wantResolvedPath string            // Expected resolved path
    fileExists    bool                // Whether resolved file exists
    wantErr       bool
}{
    {
        name:         "relative path in same directory",
        baseFile:     "/plans/setup.yaml",
        refFile:      "features.yaml",
        wantResolvedPath: "/plans/features.yaml",
        fileExists:   true,
        wantErr:      false,
    },
    {
        name:         "relative path with subdirectory",
        baseFile:     "/plans/plan-01.yaml",
        refFile:      "./deployment/plan-03.yaml",
        wantResolvedPath: "/plans/deployment/plan-03.yaml",
        fileExists:   true,
        wantErr:      false,
    },
    {
        name:         "relative path going up directory",
        baseFile:     "/project/plans/sub/plan-02.yaml",
        refFile:      "../plan-01.yaml",
        wantResolvedPath: "/project/plans/plan-01.yaml",
        fileExists:   true,
        wantErr:      false,
    },
    {
        name:         "absolute path reference",
        baseFile:     "/plans/setup.yaml",
        refFile:      "/other-plans/features.yaml",
        wantResolvedPath: "/other-plans/features.yaml",
        fileExists:   true,
        wantErr:      false,
    },
    {
        name:         "non-existent file returns error",
        baseFile:     "/plans/setup.yaml",
        refFile:      "nonexistent.yaml",
        wantResolvedPath: "/plans/nonexistent.yaml",
        fileExists:   false,
        wantErr:      true,
        errContains: "file not found",
    },
    {
        name:         "directory instead of file",
        baseFile:     "/plans/setup.yaml",
        refFile:      "subdir",
        wantResolvedPath: "/plans/subdir",
        fileExists:   false, // Is a directory, not a file
        wantErr:      true,
    },
}
```

#### Test Category 3: Cross-File Dependency Validation

**Test Suite**: `TestValidateCrossFileDependencies`

```go
tests := []struct {
    name        string
    plans       map[string]*models.Plan  // filename -> parsed plan
    wantErr     bool
    errMsg      string
}{
    {
        name: "valid cross-file dependencies",
        plans: map[string]*models.Plan{
            "plan-01-setup.yaml": {
                Tasks: []models.Task{
                    {Number: "1", Name: "Initialize", DependsOn: []string{}},
                    {Number: "2", Name: "Setup DB", DependsOn: []string{"1"}},
                },
            },
            "plan-02-features.yaml": {
                Tasks: []models.Task{
                    {Number: "3", Name: "Auth", DependsOn: []string{"file:plan-01-setup.yaml:2"}},
                    {Number: "4", Name: "API", DependsOn: []string{"3"}},
                },
            },
        },
        wantErr: false,
    },
    {
        name: "missing cross-file task reference",
        plans: map[string]*models.Plan{
            "plan-01-setup.yaml": {
                Tasks: []models.Task{
                    {Number: "1", Name: "Initialize", DependsOn: []string{}},
                },
            },
            "plan-02-features.yaml": {
                Tasks: []models.Task{
                    {Number: "2", Name: "Feature", DependsOn: []string{"file:plan-01-setup.yaml:999"}},
                },
            },
        },
        wantErr: true,
        errMsg: "task 999 not found in file plan-01-setup.yaml",
    },
    {
        name: "circular dependency across files",
        plans: map[string]*models.Plan{
            "plan-01.yaml": {
                Tasks: []models.Task{
                    {Number: "1", Name: "Task 1", DependsOn: []string{"file:plan-02.yaml:2"}},
                },
            },
            "plan-02.yaml": {
                Tasks: []models.Task{
                    {Number: "2", Name: "Task 2", DependsOn: []string{"file:plan-01.yaml:1"}},
                },
            },
        },
        wantErr: true,
        errMsg: "circular dependency detected",
    },
    {
        name: "self-reference in cross-file (same file)",
        plans: map[string]*models.Plan{
            "plan-01.yaml": {
                Tasks: []models.Task{
                    {Number: "1", Name: "Task 1", DependsOn: []string{"file:plan-01.yaml:1"}},
                },
            },
        },
        wantErr: true,
        errMsg: "self-reference",
    },
}
```

#### Test Category 4: Dependency Normalization

**Test Suite**: `TestNormalizeDependencies`

```go
tests := []struct {
    name           string
    rawDeps        []interface{}         // Raw YAML input
    baseFile       string                // Current file
    wantNormalized []string              // Expected normalized format (canonical)
    wantErr        bool
}{
    {
        name:           "numeric converted to canonical local format",
        rawDeps:        []interface{}{1, 2, 3},
        baseFile:       "/plans/plan-01.yaml",
        wantNormalized: []string{"local:1", "local:2", "local:3"},
        wantErr:        false,
    },
    {
        name: "cross-file converted to canonical format",
        rawDeps: []interface{}{
            map[string]interface{}{
                "file": "plan-02.yaml",
                "task": 2,
            },
        },
        baseFile:       "/plans/plan-01.yaml",
        wantNormalized: []string{"file:/plans/plan-02.yaml:2"},
        wantErr:        false,
    },
    {
        name: "mixed normalized correctly",
        rawDeps: []interface{}{
            1,
            map[string]interface{}{
                "file": "plan-02.yaml",
                "task": 2,
            },
            3,
        },
        baseFile: "/plans/plan-01.yaml",
        wantNormalized: []string{
            "local:1",
            "file:/plans/plan-02.yaml:2",
            "local:3",
        },
        wantErr: false,
    },
}
```

---

## Integration Test Design

### Test Module: `internal/parser/cross_file_integration_test.go`

#### Test Category 1: Multi-File Plan Loading

**Test Suite**: `TestLoadAndResolveCrossFileDepdencies`

```go
tests := []struct {
    name          string
    planFiles     map[string]string     // filename -> file content
    basePath      string                // Base directory for plans
    wantTaskCount int                   // Total tasks loaded
    wantErr       bool
    validate      func(*testing.T, *models.Plan)
}{
    {
        name: "two-file plan with cross-file dependency",
        planFiles: map[string]string{
            "plan-01-setup.yaml": `
plan:
  metadata:
    feature_name: "Setup"
  tasks:
    - task_number: 1
      name: "Initialize"
      depends_on: []
      description: "Init"
    - task_number: 2
      name: "Configure"
      depends_on: [1]
      description: "Config"
`,
            "plan-02-features.yaml": `
plan:
  metadata:
    feature_name: "Features"
  tasks:
    - task_number: 3
      name: "Authentication"
      depends_on:
        - file: "plan-01-setup.yaml"
          task: 2
      description: "Auth"
`,
        },
        wantTaskCount: 3,
        wantErr:       false,
        validate: func(t *testing.T, plan *models.Plan) {
            // Verify Task 3 has correct resolved dependency
            task3 := findTask(plan.Tasks, "3")
            if task3 == nil {
                t.Fatal("Task 3 not found")
            }
            // Check that cross-file dependency is resolved/normalized
            if len(task3.DependsOn) != 1 {
                t.Errorf("Task 3 should have 1 dependency, got %d", len(task3.DependsOn))
            }
        },
    },
    {
        name: "three-file chain with sequential dependencies",
        planFiles: map[string]string{
            "plan-01-foundation.yaml": `...`,
            "plan-02-services.yaml": `...`,
            "plan-03-deployment.yaml": `...`,
        },
        wantTaskCount: 9, // 3 tasks per file
        wantErr:       false,
    },
    {
        name: "diamond dependency pattern across files",
        planFiles: map[string]string{
            "plan-01-base.yaml": `...`,
            "plan-02-auth.yaml": `...`,
            "plan-03-api.yaml": `...`,
            "plan-04-integration.yaml": `...`,
        },
        wantTaskCount: 4,
        wantErr:       false,
    },
}
```

#### Test Category 2: Dependency Graph Resolution

**Test Suite**: `TestResolveFullDependencyGraph`

```go
tests := []struct {
    name             string
    planContent      map[string]string
    wantWaveCount    int           // Expected number of execution waves
    wantWaveContent  [][]string    // Expected task numbers per wave
    wantErr          bool
}{
    {
        name: "cross-file dependencies affect wave calculation",
        planContent: {
            "plan-01.yaml": {
                Task 1: no deps
                Task 2: depends on Task 1
            },
            "plan-02.yaml": {
                Task 3: depends on Task 2 (cross-file)
                Task 4: depends on Task 3
            },
        },
        wantWaveCount: 4,
        wantWaveContent: [][]string{
            {"1"},      // Wave 1: Task 1 only
            {"2"},      // Wave 2: Task 2 (depends on 1)
            {"3"},      // Wave 3: Task 3 (depends on 2 in other file)
            {"4"},      // Wave 4: Task 4 (depends on 3)
        },
        wantErr: false,
    },
    {
        name: "parallel tasks across files in same wave",
        planContent: {
            "plan-01.yaml": {
                Task 1: no deps
            },
            "plan-02.yaml": {
                Task 2: depends on Task 1 (cross-file)
                Task 3: depends on Task 1 (cross-file)
            },
        },
        wantWaveCount: 3,
        wantWaveContent: [][]string{
            {"1"},      // Wave 1: Task 1
            {"2", "3"}, // Wave 2: Tasks 2 and 3 (both depend on 1, can run in parallel)
        },
        wantErr: false,
    },
}
```

#### Test Category 3: Backward Compatibility

**Test Suite**: `TestBackwardCompatibilityNumericOnly`

```go
tests := []struct {
    name    string
    setup   func() *models.Plan
    wantErr bool
    validate func(*testing.T, *models.Plan)
}{
    {
        name: "single-file plan with numeric dependencies still works",
        setup: func() *models.Plan {
            return &models.Plan{
                Tasks: []models.Task{
                    {Number: "1", Name: "T1", DependsOn: []string{}},
                    {Number: "2", Name: "T2", DependsOn: []string{"1"}},
                    {Number: "3", Name: "T3", DependsOn: []string{"1", "2"}},
                },
            }
        },
        wantErr: false,
        validate: func(t *testing.T, plan *models.Plan) {
            if len(plan.Tasks) != 3 {
                t.Errorf("expected 3 tasks, got %d", len(plan.Tasks))
            }
            // Waves should be: Wave 1: [1], Wave 2: [2], Wave 3: [3]
            waves, err := executor.CalculateWaves(plan.Tasks)
            if err != nil {
                t.Fatalf("CalculateWaves failed: %v", err)
            }
            if len(waves) != 3 {
                t.Errorf("expected 3 waves, got %d", len(waves))
            }
        },
    },
    {
        name: "old-style split plans without cross-file refs still work",
        setup: func() *models.Plan {
            // Simulates two files loaded separately, merged without cross-file refs
            return &models.Plan{
                Tasks: []models.Task{
                    // From plan-01.yaml
                    {Number: "1", Name: "T1", DependsOn: []string{}, SourceFile: "plan-01.yaml"},
                    {Number: "2", Name: "T2", DependsOn: []string{"1"}, SourceFile: "plan-01.yaml"},
                    // From plan-02.yaml
                    {Number: "3", Name: "T3", DependsOn: []string{"2"}, SourceFile: "plan-02.yaml"},
                },
            }
        },
        wantErr: false,
    },
}
```

#### Test Category 4: Single-File Validation

**Test Suite**: `TestValidateSingleFilePlan`

```go
// When validating a single plan file in isolation:
// - Cross-file references should be flagged as warnings (or optional)
// - Single-file numeric dependencies should work normally
// - No requirement to have all referenced files present

tests := []struct{
    name    string
    plan    *models.Plan
    wantWarning bool
    wantErr bool
}{
    {
        name: "single file with only numeric deps - no warnings",
        plan: &models.Plan{
            Tasks: []models.Task{
                {Number: "1", DependsOn: []string{}},
                {Number: "2", DependsOn: []string{"1"}},
            },
        },
        wantWarning: false,
        wantErr: false,
    },
    {
        name: "single file with unresolved cross-file refs - warning only",
        plan: &models.Plan{
            Tasks: []models.Task{
                {Number: "1", DependsOn: []string{"file:other.yaml:2"}},
            },
        },
        wantWarning: true,
        wantErr: false, // Single-file validation doesn't error on unresolved cross-file refs
    },
}
```

#### Test Category 5: Multi-File Validation

**Test Suite**: `TestValidateMultiFilePlan`

```go
// When validating multiple plan files together:
// - All cross-file references must be resolvable
// - All referenced tasks must exist
// - Circular dependencies across files must be detected
// - Clear error messages indicating which file/task is problematic

tests := []struct{
    name    string
    plans   map[string]*models.Plan
    wantErr bool
    errMsg  string
}{
    {
        name: "multi-file with valid cross-file refs",
        plans: {
            "plan-01.yaml": { Task 1, Task 2 },
            "plan-02.yaml": { Task 3 depends on plan-01.yaml:Task 2 },
        },
        wantErr: false,
    },
    {
        name: "multi-file with missing task",
        plans: {
            "plan-01.yaml": { Task 1 },
            "plan-02.yaml": { Task 2 depends on plan-01.yaml:Task 999 },
        },
        wantErr: true,
        errMsg: "task 999 not found in plan-01.yaml referenced from plan-02.yaml",
    },
    {
        name: "multi-file with circular dependency",
        plans: {
            "plan-01.yaml": { Task 1 depends on plan-02.yaml:Task 2 },
            "plan-02.yaml": { Task 2 depends on plan-01.yaml:Task 1 },
        },
        wantErr: true,
        errMsg: "circular dependency detected",
    },
}
```

---

## Test Fixtures

### Fixture 1: Two-File Linear Chain

**File: `test-fixtures/split-plan-linear/plan-01-setup.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Setup Phase"
  tasks:
    - task_number: 1
      name: "Initialize Repository"
      files: [go.mod, main.go]
      depends_on: []
      estimated_time: "15m"
      description: "Initialize Go module and entry point"

    - task_number: 2
      name: "Setup CLI Framework"
      files: [cmd/root.go, cmd/version.go]
      depends_on: [1]
      estimated_time: "30m"
      description: "Add Cobra CLI framework"
```

**File: `test-fixtures/split-plan-linear/plan-02-features.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Features Phase"
  tasks:
    - task_number: 3
      name: "Implement Core Feature"
      files: [internal/feature/feature.go]
      depends_on:
        - file: "plan-01-setup.yaml"
          task: 2
      estimated_time: "1h"
      description: "Core feature implementation"

    - task_number: 4
      name: "Add Tests"
      files: [internal/feature/feature_test.go]
      depends_on: [3]
      estimated_time: "45m"
      description: "Test suite for feature"
```

**Expected Waves**:
- Wave 1: [1]
- Wave 2: [2]
- Wave 3: [3]
- Wave 4: [4]

---

### Fixture 2: Three-File Diamond Pattern

**File: `test-fixtures/split-plan-diamond/plan-01-foundation.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Foundation"
  tasks:
    - task_number: 1
      name: "Database Setup"
      files: [internal/db/db.go]
      depends_on: []
      estimated_time: "30m"
      description: "Initialize database"
```

**File: `test-fixtures/split-plan-diamond/plan-02-services.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Services"
  tasks:
    - task_number: 2
      name: "Auth Service"
      files: [internal/auth/auth.go]
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 1
      estimated_time: "1h"
      description: "Authentication service"

    - task_number: 3
      name: "API Service"
      files: [internal/api/api.go]
      depends_on:
        - file: "plan-01-foundation.yaml"
          task: 1
      estimated_time: "1h"
      description: "REST API service"
```

**File: `test-fixtures/split-plan-diamond/plan-03-integration.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Integration"
  tasks:
    - task_number: 4
      name: "Wire Services Together"
      files: [cmd/main.go]
      depends_on:
        - file: "plan-02-services.yaml"
          task: 2
        - file: "plan-02-services.yaml"
          task: 3
      estimated_time: "30m"
      description: "Integrate auth and API services"
```

**Expected Waves**:
- Wave 1: [1]
- Wave 2: [2, 3] (parallel)
- Wave 3: [4]

---

### Fixture 3: Complex Multi-File with Subdirectories

**File: `test-fixtures/split-plan-complex/plan-01-foundation.yaml`**
```yaml
# Foundation tasks (no dependencies)
```

**File: `test-fixtures/split-plan-complex/features/plan-02-auth.yaml`**
```yaml
tasks:
  - task_number: 2
    depends_on:
      - file: "../plan-01-foundation.yaml"
        task: 1
```

**File: `test-fixtures/split-plan-complex/features/plan-03-api.yaml`**
```yaml
tasks:
  - task_number: 3
    depends_on:
      - file: "../plan-01-foundation.yaml"
        task: 1
      - file: "plan-02-auth.yaml"
        task: 2
```

**File: `test-fixtures/split-plan-complex/deployment/plan-04-deploy.yaml`**
```yaml
tasks:
  - task_number: 4
    depends_on:
      - file: "../features/plan-03-api.yaml"
        task: 3
```

---

### Fixture 4: Markdown Format Fixtures

**File: `test-fixtures/split-plan-markdown/plan-01-setup.md`**
```markdown
# Setup Phase

## Task 1: Initialize Repository

**File(s)**: go.mod, main.go
**Depends on**: None
**Estimated time**: 15m

Initialize Go module and entry point.
```

**File: `test-fixtures/split-plan-markdown/plan-02-features.md`**
```markdown
# Features Phase

## Task 2: Implement Feature

**File(s)**: internal/feature/feature.go
**Depends on**: plan-01-setup.md:Task 1
**Estimated time**: 1h

Feature implementation that depends on setup from plan-01.
```

---

### Fixture 5: Mixed Format (Numeric and Cross-File)

**File: `test-fixtures/split-plan-mixed/plan-01.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Phase 1"
  tasks:
    - task_number: 1
      name: "Task 1"
      depends_on: []

    - task_number: 2
      name: "Task 2"
      depends_on: [1]  # Local numeric

    - task_number: 3
      name: "Task 3"
      depends_on: [1, 2]  # Mixed: local numeric only in same file
```

**File: `test-fixtures/split-plan-mixed/plan-02.yaml`**
```yaml
plan:
  metadata:
    feature_name: "Phase 2"
  tasks:
    - task_number: 4
      name: "Task 4"
      depends_on:
        - file: "plan-01.yaml"
          task: 1  # Cross-file
        - 3        # ERROR: Should be local, but interpreted as reference to Task 3 in current file
```

**Note**: This fixture demonstrates the importance of clarifying numeric semantics:
- **Option A**: Numeric always means local to current file (recommended)
- **Option B**: Numeric is ambiguous in multi-file context (error)

---

## Error Cases

### Error Case 1: Missing File

**Scenario**: Task references non-existent file
```yaml
depends_on:
  - file: "nonexistent.yaml"
    task: 2
```

**Expected Error**:
```
Error: Task "4" in "plan-02.yaml" references missing file "nonexistent.yaml"
Location: plan-02.yaml, Task 4
Resolution: Verify file name and path (relative to plan-02.yaml directory)
```

---

### Error Case 2: Missing Task in Referenced File

**Scenario**: File exists but task not found
```yaml
# plan-02.yaml
depends_on:
  - file: "plan-01.yaml"
    task: 999
```

**Expected Error**:
```
Error: Task "2" in "plan-02.yaml" depends on Task "999" in "plan-01.yaml", but Task "999" not found
Available tasks in plan-01.yaml: 1, 2, 3
Location: plan-02.yaml, Task 2
Resolution: Check referenced task number in plan-01.yaml
```

---

### Error Case 3: Circular Dependency Across Files

**Scenario**: Dependency forms cycle
```yaml
# plan-01.yaml, Task 1
depends_on:
  - file: "plan-02.yaml"
    task: 2

# plan-02.yaml, Task 2
depends_on:
  - file: "plan-01.yaml"
    task: 1
```

**Expected Error**:
```
Error: Circular dependency detected
Cycle: plan-01.yaml:Task 1 -> plan-02.yaml:Task 2 -> plan-01.yaml:Task 1
Resolution: Remove or rearrange dependencies to break the cycle
```

---

### Error Case 4: Malformed Dependency Object

**Scenario**: Cross-file dependency missing required field
```yaml
depends_on:
  - file: "plan-02.yaml"
    # Missing "task" field
```

**Expected Error**:
```
Error: Malformed cross-file dependency in Task "1" of "plan-01.yaml"
Missing required field: "task"
Expected format: { file: "filename.yaml", task: "task-id" }
```

---

### Error Case 5: Invalid File Path

**Scenario**: File path contains invalid characters or is a directory
```yaml
depends_on:
  - file: "../../../../../../etc/passwd"  # Path traversal attempt
    task: 1
```

**Expected Error/Handling**:
```
Error: Invalid file path in cross-file dependency
Path: "../../../../../../etc/passwd"
Issue: Path traversal outside project root not allowed
Resolution: Use relative paths within project directory
```

---

### Error Case 6: Ambiguous Numeric References in Multi-File

**Scenario**: Task in multi-file plan with numeric-only dependency (ambiguous)
```yaml
# plan-02.yaml, Task 5
depends_on:
  - 1  # Is this Task 1 in plan-02.yaml or plan-01.yaml?
```

**Recommended Behavior**:
- **Numeric-only = local file only** (clear and unambiguous)
- If cross-file reference intended, must use explicit format
- Alternative: Warn user that numeric is ambiguous in multi-file context

---

## Coverage Targets

### Unit Test Coverage

| Module | Target | Current | Status |
|--------|--------|---------|--------|
| `parser.ParseMixedDependencies` | 95% | 0% | Not yet implemented |
| `parser.ResolveCrossFileRef` | 90% | 0% | Not yet implemented |
| `parser.ValidateCrossFileDeps` | 90% | 0% | Not yet implemented |
| `executor.ResolveFullDepGraph` | 85% | N/A | Extends existing |
| `executor.ValidateCyclicDeps` | 85% | N/A | Extends existing |

### Integration Test Coverage

| Scenario | Target | Current | Status |
|----------|--------|---------|--------|
| Two-file linear chain | 100% | 0% | Not yet implemented |
| Three-file diamond | 100% | 0% | Not yet implemented |
| Complex subdirectory | 100% | 0% | Not yet implemented |
| Circular detection | 100% | 0% | Not yet implemented |
| Backward compatibility | 100% | 0% | Not yet implemented |
| Single-file validation | 100% | 0% | Not yet implemented |
| Multi-file validation | 100% | 0% | Not yet implemented |

### Error Handling Coverage

| Error Type | Coverage |
|------------|----------|
| Missing file | Unit + Integration |
| Missing task | Unit + Integration |
| Circular dependency | Unit + Integration |
| Malformed object | Unit |
| Invalid path | Unit |
| Numeric ambiguity | Unit + Documentation |

### Overall Coverage Target

**Phase 1 (Unit)**: 85%+ coverage for new parser functions
**Phase 2 (Integration)**: 80%+ coverage for multi-file scenarios
**Phase 3 (E2E)**: All error cases validated with clear messages

---

## Test Execution Strategy

### Phase 1: Parser Unit Tests (Priority: High)

1. Run `TestParseMixedDependencies` - ensure parsing logic correct
2. Run `TestResolveFilePathDependency` - ensure path resolution works
3. Run `TestValidateCrossFileDependencies` - ensure validation catches errors
4. Run `TestNormalizeDependencies` - ensure canonical format works

**Success Criteria**:
- All tests pass
- Edge cases covered
- Error messages clear and actionable
- 85%+ code coverage

### Phase 2: Parser Integration Tests (Priority: High)

1. Run `TestLoadAndResolveCrossFileDepdencies` - end-to-end loading
2. Run `TestResolveFullDependencyGraph` - wave calculation with cross-file
3. Run `TestBackwardCompatibilityNumericOnly` - existing functionality preserved
4. Run `TestValidateSingleFilePlan` - single-file still works
5. Run `TestValidateMultiFilePlan` - multi-file validation works

**Success Criteria**:
- All tests pass
- Existing tests still pass (backward compatible)
- 80%+ integration coverage
- Wave calculations correct

### Phase 3: Executor Tests (Priority: Medium)

1. Update `TestCalculateWaves` to include cross-file scenarios
2. Verify `DetectCycle` works with cross-file references
3. Verify wave execution respects cross-file dependencies

**Success Criteria**:
- Executor handles cross-file deps transparently
- Wave calculation correct
- Cycle detection works across files

### Phase 4: E2E CLI Tests (Priority: Medium)

1. `conductor validate plan-01.yaml plan-02.yaml` - validates cross-file refs
2. `conductor run plan-01.yaml plan-02.yaml` - executes with correct ordering
3. Error scenarios produce helpful messages

---

## Implementation Checklist

- [ ] Design dependency format (numeric vs object structure)
- [ ] Implement `ParseMixedDependencies()` function
- [ ] Implement `ResolveCrossFileRef()` function
- [ ] Add `Dependency` model struct
- [ ] Implement YAML parsing for mixed formats
- [ ] Implement Markdown support (if applicable)
- [ ] Add path resolution logic
- [ ] Add cross-file validation
- [ ] Extend cycle detection for cross-file
- [ ] Add test fixtures (all 5 types)
- [ ] Implement unit tests (Phase 1)
- [ ] Implement integration tests (Phase 2)
- [ ] Update executor for cross-file (Phase 3)
- [ ] Update CLI validation command
- [ ] Add documentation and examples
- [ ] Add migration guide for existing plans
- [ ] Full test coverage validation

---

## Appendix A: Data Structures

### Proposed Dependency Model

```go
// Dependency represents a single dependency entry (local or cross-file)
type Dependency struct {
    Type    string // "local" or "cross-file"
    TaskID  string // Task ID (numeric or alphanumeric)
    File    string // File path (for cross-file only, resolved to absolute)
    RawFile string // Original file path from YAML (for error messages)
}

// Canonical representation for internal use
// Local: "local:1", "local:task-name"
// Cross-file: "file:/absolute/path/to/plan.yaml:2"
```

### Validation Result Model

```go
type ValidationResult struct {
    Valid      bool
    Errors     []ValidationError
    Warnings   []ValidationWarning
}

type ValidationError struct {
    File        string // Source file
    Task        string // Task number
    Dependency  string // The problematic dependency
    Message     string // Error message
    Suggestion  string // How to fix it
}
```

---

## Appendix B: Test Naming Convention

All cross-file dependency tests follow naming pattern:
```
Test[Feature][Scenario][Aspect]

Examples:
- TestParseMixedDependencies_Numeric_Only
- TestParseMixedDependencies_CrossFile_WithRelativePath
- TestResolveCrossFileRef_FileNotFound_ReturnsError
- TestValidateCrossFileDeps_CircularDependency_DetectedCorrectly
- TestResolveFullDependencyGraph_LinearChain_WaveOrderCorrect
- TestLoadAndResolveCrossFileDependencies_TwoFileLinear_IntegrationOK
```

---

## Document History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-11-23 | Initial comprehensive test plan |

