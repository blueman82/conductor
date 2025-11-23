# Cross-File Dependency Implementation Roadmap

**Status**: Design Complete, Ready for Implementation
**Scope**: Core parsing, validation, and wave calculation
**Effort Estimate**: 3-4 sprints (3-4 weeks)
**Risk Level**: Medium (extends existing graph algorithms)

---

## Executive Summary

This roadmap provides a structured implementation plan for Conductor's cross-file dependency support. The feature enables tasks in split plans to reference tasks in other files using mixed dependency formats.

**Key Deliverables**:
1. Mixed dependency parser (YAML and Markdown)
2. Cross-file path resolution
3. Enhanced validation and cycle detection
4. Updated wave calculation
5. Comprehensive test suite (80%+ coverage)
6. Documentation and examples

---

## Phase 1: Core Parsing (Week 1)

### Goals
- Parse mixed dependency formats (numeric + object)
- Support both YAML and Markdown
- Normalize dependencies to canonical format

### Tasks

#### 1.1 Create Dependency Model
**File**: `internal/models/dependency.go`

```go
package models

// Dependency represents a single dependency entry (local or cross-file)
type Dependency struct {
    Type    string // "local" or "cross-file"
    TaskID  string // Task ID (numeric or alphanumeric)
    File    string // File path (for cross-file, absolute path)
    RawFile string // Original file path from YAML (for error messages)
}

// IsLocal returns true if this is a local dependency (same file)
func (d *Dependency) IsLocal() bool {
    return d.Type == "local"
}

// IsCrossFile returns true if this is a cross-file reference
func (d *Dependency) IsCrossFile() bool {
    return d.Type == "cross-file"
}

// Canonical returns the canonical string representation
// Local: "local:1", "local:task-name"
// Cross-file: "file:/absolute/path/to/plan.yaml:2"
func (d *Dependency) Canonical() string {
    if d.IsLocal() {
        return fmt.Sprintf("local:%s", d.TaskID)
    }
    return fmt.Sprintf("file:%s:%s", d.File, d.TaskID)
}
```

**Tests**:
- `TestDependency_IsLocal`
- `TestDependency_IsCrossFile`
- `TestDependency_Canonical`

**Effort**: 2-4 hours

---

#### 1.2 Implement YAML Mixed Dependency Parser
**File**: `internal/parser/yaml_dependencies.go`

```go
package parser

// ParseMixedDependencies parses the depends_on field which can contain
// numeric entries (local), object entries (cross-file), or a mix
func ParseMixedDependencies(rawDeps []interface{}) ([]models.Dependency, error) {
    var deps []models.Dependency

    for i, raw := range rawDeps {
        switch v := raw.(type) {
        case int, int64, float64:
            // Numeric entry: local dependency
            taskID := convertToString(v)
            deps = append(deps, models.Dependency{
                Type:   "local",
                TaskID: taskID,
            })

        case string:
            // String entry: assume local dependency
            deps = append(deps, models.Dependency{
                Type:   "local",
                TaskID: v,
            })

        case map[interface{}]interface{}:
            // Object entry: cross-file dependency
            dep, err := parseCrossFileDependency(v)
            if err != nil {
                return nil, fmt.Errorf("depends_on[%d]: %w", i, err)
            }
            deps = append(deps, *dep)

        default:
            return nil, fmt.Errorf("depends_on[%d]: unsupported type %T", i, raw)
        }
    }

    return deps, nil
}

// parseCrossFileDependency parses a cross-file dependency object
func parseCrossFileDependency(obj map[interface{}]interface{}) (*models.Dependency, error) {
    fileRaw, hasFile := obj["file"]
    taskRaw, hasTask := obj["task"]

    if !hasFile || !hasTask {
        return nil, errors.New("cross-file dependency missing required fields (file, task)")
    }

    file, err := convertToString(fileRaw)
    if err != nil {
        return nil, fmt.Errorf("invalid file field: %w", err)
    }

    taskID, err := convertToString(taskRaw)
    if err != nil {
        return nil, fmt.Errorf("invalid task field: %w", err)
    }

    return &models.Dependency{
        Type:    "cross-file",
        TaskID:  taskID,
        RawFile: file,
    }, nil
}
```

**Tests**:
- `TestParseMixedDependencies_NumericOnly` (existing numeric array)
- `TestParseMixedDependencies_CrossFileOnly` (new object format)
- `TestParseMixedDependencies_Mixed` (mix of numeric and objects)
- `TestParseMixedDependencies_MissingFields` (error cases)
- `TestParseMixedDependencies_InvalidTypes` (error cases)

**Effort**: 6-8 hours

---

#### 1.3 Implement Markdown Mixed Dependency Parser
**File**: `internal/parser/markdown_dependencies.go`

```go
package parser

// Parses markdown format like:
// **Depends on**: Task 1, [Task 2 from plan-02.yaml], Task 3
// Or alternative format:
// **Depends on**: Task 1, Task 3
// **Cross-file deps**: plan-02.yaml:Task 2

func parseMarkdownDependencies(text string, baseFile string) ([]models.Dependency, error) {
    // Extract "Depends on" line
    depsLine := extractMetadataField(text, "Depends on")
    if depsLine == "" {
        return nil, nil
    }

    // Split by comma, parse each entry
    entries := strings.Split(depsLine, ",")
    var deps []models.Dependency

    for _, entry := range entries {
        entry = strings.TrimSpace(entry)

        // Check if it's a cross-file ref: [Task X from file.yaml]
        if strings.HasPrefix(entry, "[") && strings.HasSuffix(entry, "]") {
            dep, err := parseMarkdownCrossFileDep(entry)
            if err != nil {
                return nil, err
            }
            deps = append(deps, *dep)
        } else {
            // Local dependency: "Task 1" or just "1"
            taskID := extractTaskID(entry)
            deps = append(deps, models.Dependency{
                Type:   "local",
                TaskID: taskID,
            })
        }
    }

    return deps, nil
}
```

**Alternative Markdown Format**:
```markdown
**Depends on**: Task 1, Task 3
**Cross-file**: plan-02.yaml:Task 2
```

**Decision Point**: Which markdown format is cleaner?
- Option A: Inline `[Task 2 from plan-02.yaml]`
- Option B: Separate `**Cross-file**: plan-02.yaml:Task 2`

**Recommendation**: Start with Option B (cleaner), implement Option A later if needed.

**Tests**:
- `TestParseMarkdownDependencies_NumericOnly`
- `TestParseMarkdownDependencies_TaskFormat` ("Task 1" vs "1")
- `TestParseMarkdownDependencies_CrossFileFormat`
- `TestParseMarkdownDependencies_Mixed`
- `TestParseMarkdownDependencies_MissingField`

**Effort**: 6-8 hours

---

#### 1.4 Update YAML Parser Integration
**File**: `internal/parser/yaml.go` (modify existing)

Replace the simple `DependsOn` field parsing:

```go
// Before (current code)
dependsOn := make([]string, 0, len(yt.DependsOn))
for j, dep := range yt.DependsOn {
    depStr, err := convertToString(dep)
    if err != nil {
        return nil, fmt.Errorf("task %s: invalid depends_on[%d]: %w", taskNum, j, err)
    }
    dependsOn = append(dependsOn, depStr)
}

// After (new code with mixed format support)
parsedDeps, err := ParseMixedDependencies(yt.DependsOn)
if err != nil {
    return nil, fmt.Errorf("task %s: %w", taskNum, err)
}
// Normalize and resolve to canonical format (see Phase 2)
dependsOn := normalizeDependencies(parsedDeps, baseFile)
```

**Tests**: Update existing YAML parser tests to ensure backward compatibility

**Effort**: 4 hours

---

### Phase 1 Summary

| Task | Files | Effort | Tests | Status |
|------|-------|--------|-------|--------|
| 1.1: Dependency Model | models/dependency.go | 2-4h | 3 | Ready |
| 1.2: YAML Parser | parser/yaml_dependencies.go | 6-8h | 5+ | Ready |
| 1.3: Markdown Parser | parser/markdown_dependencies.go | 6-8h | 5+ | Ready |
| 1.4: Integration | parser/yaml.go (modified) | 4h | Updated | Ready |
| **Phase 1 Total** | | **18-28 hours** | **13+ tests** | |

**Exit Criteria**:
- Mixed dependencies parse correctly
- All tests pass (unit tests)
- YAML and Markdown formats work
- Backward compatibility verified

---

## Phase 2: File Resolution & Validation (Week 2)

### Goals
- Resolve relative file paths to absolute
- Validate referenced files exist
- Validate referenced tasks exist
- Extend cycle detection for cross-file

### Tasks

#### 2.1 Implement File Path Resolution
**File**: `internal/parser/file_resolution.go`

```go
package parser

// ResolveFilePath resolves a relative or absolute file path
// Returns absolute path, or error if file doesn't exist
func ResolveFilePath(refPath string, baseFile string) (string, error) {
    // If refPath is absolute, use it directly
    if filepath.IsAbs(refPath) {
        if _, err := os.Stat(refPath); err != nil {
            return "", fmt.Errorf("referenced file not found: %s", refPath)
        }
        return filepath.Abs(refPath)
    }

    // Resolve relative to baseFile's directory
    baseDir := filepath.Dir(baseFile)
    resolvedPath := filepath.Join(baseDir, refPath)

    // Normalize the path (handle .. and .)
    absPath, err := filepath.Abs(resolvedPath)
    if err != nil {
        return "", err
    }

    // Verify file exists
    if _, err := os.Stat(absPath); err != nil {
        return "", fmt.Errorf("referenced file not found: %s (relative path: %s)", absPath, refPath)
    }

    return absPath, nil
}

// ValidateCrossFileRef checks if a cross-file reference is valid
// - File exists
// - Task exists in that file
// Returns error if invalid
func ValidateCrossFileRef(dep models.Dependency, baseFile string, loadedPlans map[string]*models.Plan) error {
    if !dep.IsCrossFile() {
        return nil
    }

    // Resolve file path
    absPath, err := ResolveFilePath(dep.RawFile, baseFile)
    if err != nil {
        return fmt.Errorf("cannot resolve file %q: %w", dep.RawFile, err)
    }

    // Check if plan is loaded
    plan, exists := loadedPlans[absPath]
    if !exists {
        return fmt.Errorf("plan file %q not loaded", absPath)
    }

    // Check if task exists in plan
    for _, task := range plan.Tasks {
        if task.Number == dep.TaskID {
            // Update dependency with resolved path
            dep.File = absPath
            return nil
        }
    }

    // Task not found
    availableTasks := make([]string, len(plan.Tasks))
    for i, t := range plan.Tasks {
        availableTasks[i] = t.Number
    }

    return fmt.Errorf(
        "task %q not found in file %q\nAvailable tasks: %s",
        dep.TaskID, dep.RawFile, strings.Join(availableTasks, ", "),
    )
}
```

**Tests**:
- `TestResolveFilePath_Absolute` (absolute paths)
- `TestResolveFilePath_Relative` (relative paths)
- `TestResolveFilePath_WithParent` (../ navigation)
- `TestResolveFilePath_NonExistent` (error: file not found)
- `TestResolveFilePath_Directory` (error: not a file)
- `TestValidateCrossFileRef_ValidTask` (task exists)
- `TestValidateCrossFileRef_MissingTask` (error: task not found)
- `TestValidateCrossFileRef_MissingFile` (error: file doesn't exist)

**Effort**: 8-10 hours

---

#### 2.2 Extend Cycle Detection for Cross-File
**File**: `internal/executor/graph.go` (modify existing `HasCycle()`)

Current implementation only handles local task numbers. Extend to handle canonical cross-file format:

```go
// Before (current)
func (g *DependencyGraph) HasCycle() bool {
    // Uses DFS with task numbers as string (e.g., "1", "2", "3")
}

// After (new)
func (g *DependencyGraph) HasCycle() bool {
    // Uses DFS with canonical dependency format
    // "local:1" or "file:/path/to/plan.yaml:1"

    // Build adjacency list with canonical refs
    graph := make(map[string][]string)
    for taskNum := range g.Tasks {
        graph[taskNum] = []string{}
        // Get canonical dependencies
        task := g.Tasks[taskNum]
        for _, dep := range task.DependsOn {
            if isCanonical(dep) {
                graph[taskNum] = append(graph[taskNum], canonicalToTaskRef(dep))
            }
        }
    }

    // Run existing DFS cycle detection on extended graph
    return g.dfs(graph)
}

// Helper to extract task ref from canonical format
func canonicalToTaskRef(canonical string) string {
    // "local:1" -> "1"
    // "file:/path/plan.yaml:1" -> "file:/path/plan.yaml:1"
    if strings.HasPrefix(canonical, "local:") {
        return strings.TrimPrefix(canonical, "local:")
    }
    return canonical
}
```

**Tests**:
- `TestDetectCycle_NoCycleLocal` (existing test, must still pass)
- `TestDetectCycle_NoCycleCrossFile` (cross-file, no cycle)
- `TestDetectCycle_SimpleCycleCrossFile` (A->B->A across files)
- `TestDetectCycle_ComplexCycleCrossFile` (A->B->C->A across files)
- `TestDetectCycle_MixedLocal_CrossFile` (mixed, no cycle)
- `TestDetectCycle_MixedLocal_CrossFile_WithCycle` (mixed, has cycle)

**Effort**: 6-8 hours

---

#### 2.3 Implement Dependency Normalization
**File**: `internal/parser/dependency_normalization.go`

```go
package parser

// NormalizeDependencies converts parsed dependencies to canonical format
// and resolves file paths
func NormalizeDependencies(
    parsedDeps []models.Dependency,
    baseFile string,
    loadedPlans map[string]*models.Plan,
) ([]string, error) {
    var canonical []string

    for _, dep := range parsedDeps {
        if dep.IsLocal() {
            // Local: convert to "local:1"
            canonical = append(canonical, fmt.Sprintf("local:%s", dep.TaskID))
        } else {
            // Cross-file: resolve path and convert to "file:/path:1"
            absPath, err := ResolveFilePath(dep.RawFile, baseFile)
            if err != nil {
                return nil, err
            }

            // Validate task exists
            if err := ValidateCrossFileRef(dep, baseFile, loadedPlans); err != nil {
                return nil, err
            }

            canonical = append(canonical, fmt.Sprintf("file:%s:%s", absPath, dep.TaskID))
        }
    }

    return canonical, nil
}
```

**Tests**:
- `TestNormalizeDependencies_NumericOnly`
- `TestNormalizeDependencies_CrossFileOnly`
- `TestNormalizeDependencies_Mixed`
- `TestNormalizeDependencies_UnresolvablePath` (error)
- `TestNormalizeDependencies_MissingTask` (error)

**Effort**: 6-8 hours

---

#### 2.4 Implement Comprehensive Validation
**File**: `internal/parser/validation.go` (new or extend existing)

```go
package parser

// ValidatePlan validates a single plan (backward compatible)
func ValidatePlan(plan *models.Plan) []error {
    // Existing validation logic...
    // Numeric dependencies work as before
    // Cross-file refs are warnings (can't resolve without all files)
}

// ValidateMultiFilePlan validates multiple plans together
func ValidateMultiFilePlan(plans map[string]*models.Plan) []error {
    var errors []error

    // For each plan
    for baseFile, plan := range plans {
        // For each task
        for _, task := range plan.Tasks {
            // For each dependency
            for _, depStr := range task.DependsOn {
                // Check if it's a cross-file ref
                if strings.HasPrefix(depStr, "file:") {
                    // Parse and validate
                    parts := strings.Split(depStr, ":")
                    refFile := parts[1]
                    taskID := parts[2]

                    // Check if referenced plan exists
                    refPlan, exists := plans[refFile]
                    if !exists {
                        errors = append(errors, fmt.Errorf(
                            "task %q in %q references missing file %q",
                            task.Number, baseFile, refFile,
                        ))
                        continue
                    }

                    // Check if task exists in referenced plan
                    found := false
                    for _, t := range refPlan.Tasks {
                        if t.Number == taskID {
                            found = true
                            break
                        }
                    }
                    if !found {
                        errors = append(errors, fmt.Errorf(
                            "task %q in %q references missing task %q in %q",
                            task.Number, baseFile, taskID, refFile,
                        ))
                    }
                }
            }
        }
    }

    // Check for circular dependencies (existing logic extended)
    if hasCycle(plans) {
        errors = append(errors, fmt.Errorf("circular dependency detected"))
    }

    return errors
}
```

**Tests**:
- `TestValidatePlan_SingleFile_Numeric` (backward compat)
- `TestValidatePlan_SingleFile_WithUnresolvedCrossRef` (warning, not error)
- `TestValidateMultiFilePlan_ValidRefs`
- `TestValidateMultiFilePlan_MissingFile`
- `TestValidateMultiFilePlan_MissingTask`
- `TestValidateMultiFilePlan_CircularDependency`

**Effort**: 8-10 hours

---

### Phase 2 Summary

| Task | Files | Effort | Tests | Status |
|------|-------|--------|-------|--------|
| 2.1: File Resolution | parser/file_resolution.go | 8-10h | 8 | Ready |
| 2.2: Cycle Detection | executor/graph.go (mod) | 6-8h | 6 | Ready |
| 2.3: Normalization | parser/dependency_normalization.go | 6-8h | 5 | Ready |
| 2.4: Validation | parser/validation.go (new/mod) | 8-10h | 6 | Ready |
| **Phase 2 Total** | | **28-36 hours** | **25 tests** | |

**Exit Criteria**:
- All cross-file references resolve correctly
- Cycle detection works across files
- Validation catches all error cases
- Dependencies normalized to canonical format
- All tests pass

---

## Phase 3: Wave Calculation & Executor (Week 3)

### Goals
- Update wave calculation to handle cross-file dependencies
- Ensure executor respects cross-file task ordering
- Comprehensive integration testing

### Tasks

#### 3.1 Update Wave Calculation Algorithm
**File**: `internal/executor/graph.go` (modify `CalculateWaves()`)

```go
// Current implementation assumes local task numbers.
// Need to:
// 1. Handle canonical cross-file format in dependencies
// 2. Build extended dependency graph with absolute file:task refs
// 3. Run Kahn's algorithm on extended graph
// 4. Map results back to task numbers for execution

func CalculateWaves(tasks []models.Task) ([]models.Wave, error) {
    // Step 1: Build extended graph with canonical refs
    extGraph := BuildExtendedDependencyGraph(tasks)

    // Step 2: Check for cycles (existing)
    if extGraph.HasCycle() {
        return nil, errors.New("circular dependency detected")
    }

    // Step 3: Run Kahn's algorithm (existing logic)
    waves := KahnsAlgorithm(extGraph)

    // Step 4: Create Wave structures with task numbers
    return CreateWaves(waves, tasks), nil
}

// BuildExtendedDependencyGraph builds graph with cross-file refs
func BuildExtendedDependencyGraph(tasks []models.Task) *DependencyGraph {
    g := &DependencyGraph{
        Tasks:    make(map[string]*models.Task),
        InDegree: make(map[string]int),
        Edges:    make(map[string][]string),
        Groups:   make(map[int]string),
    }

    // Initialize all tasks
    for i := range tasks {
        g.Tasks[tasks[i].Number] = &tasks[i]
        g.InDegree[tasks[i].Number] = 0
        g.Groups[i] = tasks[i].WorktreeGroup
    }

    // Build edges from dependencies (handles both local and cross-file)
    for _, task := range tasks {
        for _, dep := range task.DependsOn {
            // Parse dependency (numeric or canonical)
            depTaskNum := parseTaskNumber(dep)

            // Add edge: depTaskNum -> task.Number
            g.Edges[depTaskNum] = append(g.Edges[depTaskNum], task.Number)
            g.InDegree[task.Number]++
        }
    }

    return g
}
```

**Decision**: How to handle cross-file dependencies in Task.DependsOn?

**Option A**: Store canonical format `["local:1", "file:/path:2"]`
- Pro: Explicit, clear semantics
- Con: Changes existing Task model

**Option B**: Store normalized references `["1", "plan-02.yaml:3"]`
- Pro: Backward compatible with existing code
- Con: Ambiguous (need context to know if "1" is local or cross-file)

**Option C**: Maintain original format, resolve during wave calculation
- Pro: Minimal model changes
- Con: Resolution happens multiple times

**Recommendation**: Option A with model update (see Phase 3.2)

**Tests**:
- `TestCalculateWaves_NumericOnly_Linear` (existing test, must pass)
- `TestCalculateWaves_NumericOnly_Diamond` (existing test, must pass)
- `TestCalculateWaves_CrossFile_Linear` (cross-file linear)
- `TestCalculateWaves_CrossFile_Diamond` (cross-file diamond)
- `TestCalculateWaves_Mixed_Linear` (mix of numeric and cross-file)

**Effort**: 10-12 hours

---

#### 3.2 Update Task Model (if needed)
**File**: `internal/models/task.go` (modify)

Decide on dependency format. Option: Add explicit cross-file support:

```go
type Task struct {
    // ... existing fields ...

    // Current format (keep for backward compat)
    DependsOn []string // Can be numeric, canonical, or mixed

    // NEW: Parsed dependencies (optional)
    Dependencies []models.Dependency // Parsed format for new code
}
```

Or simply keep `DependsOn []string` and document the canonical format.

**Recommendation**: Keep single `DependsOn []string` field and document canonical format internally.

**Tests**: N/A (model extension, no new test needed)

**Effort**: 2-4 hours

---

#### 3.3 Integration Tests with Real Plans
**File**: `internal/executor/integration_test.go` (new or extend)

```go
func TestExecuteWithCrossFileDependencies(t *testing.T) {
    // Load fixture: split-plan-diamond
    plan1, err := ParseFile("testdata/split-plan-diamond/plan-01-foundation.yaml")
    require.NoError(t, err)

    plan2, err := ParseFile("testdata/split-plan-diamond/plan-02-services.yaml")
    require.NoError(t, err)

    plan3, err := ParseFile("testdata/split-plan-diamond/plan-03-integration.yaml")
    require.NoError(t, err)

    // Merge plans
    merged, err := MergePlans(plan1, plan2, plan3)
    require.NoError(t, err)

    // Validate
    err = ValidateMultiFilePlan(map[string]*models.Plan{
        "plan-01-foundation.yaml": plan1,
        "plan-02-services.yaml":   plan2,
        "plan-03-integration.yaml": plan3,
    })
    require.NoError(t, err)

    // Calculate waves
    waves, err := CalculateWaves(merged.Tasks)
    require.NoError(t, err)

    // Verify wave structure
    require.Equal(t, 3, len(waves))
    require.Equal(t, 1, len(waves[0].TaskNumbers))  // Wave 1: [1]
    require.Equal(t, 2, len(waves[1].TaskNumbers))  // Wave 2: [2, 3]
    require.Equal(t, 1, len(waves[2].TaskNumbers))  // Wave 3: [4]
}
```

**Tests**:
- `TestExecuteWithCrossFileDependencies_LinearChain`
- `TestExecuteWithCrossFileDependencies_Diamond`
- `TestExecuteWithCrossFileDependencies_Complex` (subdirectories)
- `TestExecuteWithCrossFileDependencies_Markdown` (if supported)
- `TestExecuteWithCrossFileDependencies_Mixed` (numeric + cross-file)
- `TestExecuteWithCrossFileDependencies_ParallelExecution` (verify parallelism)

**Effort**: 8-10 hours

---

### Phase 3 Summary

| Task | Files | Effort | Tests | Status |
|------|-------|--------|-------|--------|
| 3.1: Wave Calculation | executor/graph.go (mod) | 10-12h | 5 | Ready |
| 3.2: Task Model | models/task.go (opt mod) | 2-4h | 0 | Ready |
| 3.3: Integration Tests | executor/integration_test.go | 8-10h | 6 | Ready |
| **Phase 3 Total** | | **20-26 hours** | **11 tests** | |

**Exit Criteria**:
- Wave calculation correct for all scenarios
- Cross-file dependencies don't break single-file plans
- Integration tests pass
- Parallelism maintained where possible

---

## Phase 4: CLI Integration & Polish (Week 4)

### Goals
- Update `conductor validate` for multi-file plans
- Update `conductor run` for cross-file awareness
- Documentation and examples
- Final testing and bug fixes

### Tasks

#### 4.1 Update Validate Command
**File**: `internal/cmd/validate.go` (modify)

```go
// Add support for multi-file validation
func (vc *ValidateCommand) Execute(ctx context.Context, args []string) error {
    // ... existing code ...

    // NEW: When multiple files provided, validate together
    if len(args) > 1 {
        // Load all plans
        plans := make(map[string]*models.Plan)
        for _, arg := range args {
            plan, err := ParseFile(arg)
            if err != nil {
                return fmt.Errorf("failed to parse %s: %w", arg, err)
            }
            plans[plan.FilePath] = plan
        }

        // Validate cross-file dependencies
        errs := ValidateMultiFilePlan(plans)
        if len(errs) > 0 {
            for _, err := range errs {
                fmt.Fprintf(vc.out, "ERROR: %v\n", err)
            }
            return fmt.Errorf("validation failed")
        }

        fmt.Fprintf(vc.out, "All files validated successfully\n")
        return nil
    }

    // ... existing single-file logic ...
}
```

**Tests**:
- `TestValidateCommand_SingleFile_BackwardCompat`
- `TestValidateCommand_MultiFile_ValidRefs`
- `TestValidateCommand_MultiFile_InvalidRefs` (error)
- `TestValidateCommand_MultiFile_CircularDep` (error)

**Effort**: 4-6 hours

---

#### 4.2 Update Run Command
**File**: `internal/cmd/run.go` (modify)

```go
// Already supports multi-file. Ensure cross-file deps are handled.
func (rc *RunCommand) Execute(ctx context.Context, args []string) error {
    // ... existing code ...

    // The executor should now handle cross-file deps transparently
    // due to changes in Phase 3

    // Just need to ensure:
    // 1. All files are loaded
    // 2. Dependencies are normalized
    // 3. Waves are calculated correctly

    // ... existing execution logic (no changes needed) ...
}
```

**Tests**: Inherit from executor tests; no new tests needed

**Effort**: 2-4 hours

---

#### 4.3 Documentation
**Files**:
- `docs/CROSS_FILE_DEPENDENCY_TEST_PLAN.md` (created in design phase)
- `docs/CROSS_FILE_DEPENDENCY_EXAMPLES.md` (created in design phase)
- `README.md` (update)
- `CLAUDE.md` (update)

**Content**:
- Usage examples
- Format specification
- Error messages and troubleshooting
- Migration guide from single-file
- Best practices

**Effort**: 4-6 hours (already covered in design phase)

---

#### 4.4 Final Testing & Bug Fixes
**Activities**:
- Run full test suite: `go test ./... -cover`
- Test with real-world plans
- Performance testing (large plans)
- Error message clarity
- Edge case testing

**Effort**: 8-12 hours

---

### Phase 4 Summary

| Task | Files | Effort | Status |
|------|-------|--------|--------|
| 4.1: Validate Command | cmd/validate.go (mod) | 4-6h | Ready |
| 4.2: Run Command | cmd/run.go (opt mod) | 2-4h | Ready |
| 4.3: Documentation | docs/*, README.md | 4-6h | Done in design |
| 4.4: Testing & Bug Fixes | All | 8-12h | Ready |
| **Phase 4 Total** | | **18-28 hours** | |

**Exit Criteria**:
- `conductor validate` works with multi-file
- `conductor run` respects cross-file deps
- Documentation is complete
- All tests pass (80%+ coverage)
- No breaking changes to existing functionality

---

## Overall Implementation Summary

### Total Effort Estimate

| Phase | Description | Effort | Tests |
|-------|-------------|--------|-------|
| Phase 1 | Core Parsing | 18-28h | 13+ |
| Phase 2 | File Resolution & Validation | 28-36h | 25 |
| Phase 3 | Wave Calculation & Executor | 20-26h | 11 |
| Phase 4 | CLI & Polish | 18-28h | - |
| **TOTAL** | | **84-118 hours** | **49+ tests** |

**Timeline**:
- Week 1: Phase 1 (parsing)
- Week 2: Phase 2 (resolution & validation)
- Week 3: Phase 3 (executor)
- Week 4: Phase 4 (polish)
- **Total: 4 weeks**

---

## Test Coverage Targets

### By Phase

| Phase | Coverage Target | Estimated |
|-------|-----------------|-----------|
| 1: Parsing | 90% | 92% |
| 2: Validation | 85% | 88% |
| 3: Executor | 80% | 82% |
| 4: CLI | 75% | 80% |
| **Overall** | **80%+** | **85%** |

### By Module

| Module | Target | Tests |
|--------|--------|-------|
| parser/yaml_dependencies.go | 95% | 8 |
| parser/markdown_dependencies.go | 90% | 8 |
| parser/file_resolution.go | 90% | 8 |
| parser/dependency_normalization.go | 90% | 5 |
| parser/validation.go | 85% | 6 |
| executor/graph.go | 85% | 6 |
| cmd/validate.go | 80% | 4 |
| **Total** | **87%** | **45+** |

---

## Risk Assessment

### Technical Risks

**Risk 1: Breaking Existing Tests**
- **Probability**: Medium
- **Impact**: High (backward compatibility critical)
- **Mitigation**: Run full test suite after each phase, use feature flags

**Risk 2: Complex Circular Dependency Cases**
- **Probability**: Low
- **Impact**: High (incorrect detection breaks execution)
- **Mitigation**: Extensive testing of cycle cases, code review

**Risk 3: Performance with Large Plans**
- **Probability**: Low
- **Impact**: Medium (O(n^2) dependency resolution)
- **Mitigation**: Profile with large fixtures, optimize if needed

### Mitigation Strategies

1. **Feature Flags**: Implement behind a flag until fully tested
2. **Backward Compatibility**: Support numeric-only format forever
3. **Comprehensive Testing**: 80%+ coverage, integration tests
4. **Code Review**: Design review before implementation
5. **Rollback Plan**: If issues, can disable feature

---

## Success Criteria

### Functional
- [ ] Parse mixed dependency formats (numeric + cross-file)
- [ ] Resolve file paths correctly
- [ ] Validate references across files
- [ ] Calculate correct execution waves
- [ ] Detect circular dependencies across files

### Non-Functional
- [ ] 80%+ test coverage
- [ ] No breaking changes
- [ ] Clear error messages
- [ ] Performance: <100ms for typical plans
- [ ] Documentation complete

### Examples
- [ ] Linear chain plan works
- [ ] Diamond pattern plan works
- [ ] Complex subdirectory plan works
- [ ] Markdown format works (if supported)
- [ ] Mixed numeric/cross-file works

---

## Next Steps

1. **Design Review**: Get stakeholder approval on:
   - Dependency format choice
   - Markdown format decision
   - Task model changes (if any)

2. **Implementation Kickoff**: Start Phase 1
   - Create dependency model
   - Implement YAML parser
   - Implement Markdown parser

3. **Daily Status**: Track progress on:
   - Tests written
   - Tests passing
   - Coverage %
   - Blockers

4. **Weekly Review**: Assess:
   - Phase completion
   - Quality metrics
   - Risks/issues
   - Adjustments needed

---

## Appendix: Test Fixtures

All test fixtures are located in:
```
internal/parser/testdata/cross-file-fixtures/
├── split-plan-linear/
│   ├── plan-01-setup.yaml
│   └── plan-02-features.yaml
├── split-plan-diamond/
│   ├── plan-01-foundation.yaml
│   ├── plan-02-services.yaml
│   └── plan-03-integration.yaml
├── split-plan-complex/
│   ├── plan-01-foundation.yaml
│   ├── features/
│   │   ├── plan-02-auth.yaml
│   │   └── plan-03-api.yaml
│   └── deployment/
│       └── plan-04-deploy.yaml
├── split-plan-markdown/
│   ├── plan-01-setup.md
│   └── plan-02-features.md
└── split-plan-mixed/
    ├── plan-01.yaml
    └── plan-02.yaml
```

Each fixture is designed to test specific scenarios during integration testing.

---

## Document Version

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-11-23 | Initial implementation roadmap |

