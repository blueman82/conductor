# Task Field Analysis - Delivery Summary

## Comprehensive Analysis Complete

A complete trace of how Conductor's executor uses Task struct fields during execution has been documented in 5 comprehensive guides totaling ~2,350 lines of documentation.

---

## Deliverables

### 1. TASK_FIELD_DOCS_INDEX.md
**Your starting point** - Quick navigation guide with topic index, scenario walkthrough, and cross-references

### 2. TASK_FIELD_ANALYSIS_SUMMARY.md (13 KB)
High-level overview with key findings, critical behaviors, and recommendations

### 3. TASK_FIELD_REFERENCE.md (12 KB)
Practical quick-reference guide for development patterns and common scenarios

### 4. TASK_FIELD_TRACE.md (26 KB)
Detailed field-by-field analysis with exact code locations and line numbers

### 5. TASK_FIELD_DATAFLOW.md (30 KB)
Complete execution flow diagrams, decision trees, and data flow visualizations

---

## Key Findings

### Critical Task Fields
```
Number          → Task identification (required)
Name            → Human identification (required)
Prompt          → Implementation details - MODIFIED before invocation (required)
Agent           → Agent selection (optional, defaults to config.DefaultAgent)
```

### Fields That Trigger Specific Behaviors

| Field | Behavior | Scope |
|-------|----------|-------|
| DependsOn | Integration context injection | Modifies Prompt before invocation |
| Type="integration" | Includes IntegrationCriteria in QC | Dual-level validation |
| SuccessCriteria | Switches to structured QC mode | Per-criterion verdict with unanimous consensus |
| IntegrationCriteria | Cross-component validation | Only if Type="integration" |
| Files | Domain-specific QC injection | Auto-injects language-specific checks |
| SourceFile | Per-file locking in multi-file plans | Serializes updates to same file |
| TestCommands | Listed in QC prompt | Informational only |

### Fields Modified During Execution
- **Prompt**: Enhanced with learning context + integration context (BEFORE invocation)
- **Agent**: May be adapted by learning or swapped on RED verdict
- **Metadata**: Stores QC verdict and failure patterns
- **Status**: Updated throughout execution lifecycle
- **CompletedAt**: Set when task completes

### Fields NOT Used (Parsed but unused)
- EstimatedTime
- WorktreeGroup
- ExecutionStartTime/EndTime/Duration (display only)
- FilesModified/Created/Deleted (display only)

---

## Critical Behaviors Documented

### 1. Integration Context Injection
**Trigger**: DependsOn not empty OR Type=="integration"
**Effect**: Modifies Prompt to include dependency file context BEFORE agent invocation
**Location**: task.go lines 517-519

### 2. Structured QC vs. Legacy QC
**Trigger**: SuccessCriteria present
**Effect**: Switches to per-criterion verification with unanimous consensus
**Location**: qc.go lines 160-227

### 3. Dual-Criteria Integration Validation
**Trigger**: Type=="integration"
**Effect**: Includes IntegrationCriteria + SuccessCriteria in QC
**Location**: qc.go lines 149-155, 178-185

### 4. Domain-Specific QC Injection
**Based on**: File extensions (.go, .sql, .ts, .py, etc.)
**Effect**: Auto-injects language-specific review criteria
**Location**: qc.go lines 18-66, 208

### 5. Per-File Locking (Multi-File Plans)
**Trigger**: SourceFile set for multi-file plans
**Effect**: Serializes updates to same file, parallel updates to different files
**Location**: task.go lines 490-498

### 6. Inter-Retry Agent Swapping
**Trigger**: RED verdict + conditions met
**Effect**: Changes Agent field for next retry attempt
**Location**: task.go lines 708-728

### 7. Learning System Integration
**Three hooks**: preTaskHook, qcReviewHook, postTaskHook
**Effect**: Adaptation + learning persistence
**Location**: task.go lines 213-440

---

## Architecture Insights

### Prompt Modification Chain
```
Original Prompt (from parser)
    ↓
[Maybe] enhancePromptWithLearning() - Add failure context
    ↓
[Maybe] buildIntegrationPrompt() - Add dependency context
    ↓
[Final Prompt] → Agent invocation
```

Both modifications happen BEFORE agent invocation - critical for providing context.

### QC Determination Logic
```
if SuccessCriteria not empty:
    USE structured mode
        → Unified criteria array (success + integration)
        → Per-criterion verification
        → Unanimous consensus
        → CriteriaResults[] array required
else:
    USE legacy mode
        → Single verdict
        → Agent verdict is final
```

### Multi-Agent Consensus Rules
```
Structured QC (with criteria):
    Criterion PASS = ALL agents agree PASS
    Criterion FAIL = ANY agent disagrees
    Final verdict = if ANY criterion fails → RED
    UNLESS all criteria pass but agent caught external issues → RED

Legacy QC (no criteria):
    Strictest wins: RED > YELLOW > GREEN
```

---

## Code Location Reference

| Concept | File | Key Lines |
|---------|------|-----------|
| Task struct | models/task.go | 9-38 |
| Integration context | executor/integration_prompt.go | 11-46 |
| Default executor | executor/task.go | 482 (Execute method) |
| Pre-execution hook | executor/task.go | 213-255 |
| Integration injection | executor/task.go | 517-519 |
| QC structured prompt | executor/qc.go | 160-227 |
| Domain-specific checks | executor/qc.go | 18-66 |
| Multi-agent QC | executor/qc.go | 433-512 |
| Criteria aggregation | executor/qc.go | 826-934 |
| Agent swapping | executor/task.go | 721-727 |
| Per-file locking | executor/task.go | 490-498 |
| Learning hooks | executor/task.go | 213-440 |
| YAML parser | parser/yaml.go | 162-174 |

---

## Usage Guidance

### When to consult each document:

**TASK_FIELD_DOCS_INDEX.md**
- First time reading these docs
- Looking for a specific topic
- Need navigation help

**TASK_FIELD_ANALYSIS_SUMMARY.md**
- Getting oriented with high-level overview
- Understanding key behaviors
- Looking for recommendations

**TASK_FIELD_REFERENCE.md**
- Implementing a feature
- Need code patterns
- Looking for quick examples

**TASK_FIELD_TRACE.md**
- Debugging specific issue
- Need exact code locations
- Understanding edge cases

**TASK_FIELD_DATAFLOW.md**
- Understanding execution flow
- Visualizing complex interactions
- Presentation/documentation

---

## Execution Timeline

The comprehensive analysis followed this path:

1. Read task.go (DefaultTaskExecutor.Execute method - 280 lines)
2. Read qc.go (QualityController implementation - 800+ lines)
3. Read integration_prompt.go (context injection)
4. Read models/task.go (struct definition)
5. Read models/response.go and models/plan.go (related structures)
6. Read parser files (YAML and Markdown extraction)
7. Analyzed field usage, dependencies, and behaviors
8. Created comprehensive documentation suite

---

## Coverage

### Fields Analyzed: 30+ fields
- Core identification (Number, Name)
- Execution context (Files, DependsOn, Agent)
- Status tracking (Status, CompletedAt)
- Structured QC (SuccessCriteria, IntegrationCriteria, Type, TestCommands)
- Multi-file coordination (SourceFile, WorktreeGroup)
- Extensibility (Metadata)
- Display/metrics (ExecutionStartTime, ExecutionDuration, etc.)

### Execution Stages Covered: 8 stages
1. Parsing & Validation
2. Graph Building & Wave Calculation
3. Pre-Execution Hooks
4. Execution & Invocation
5. Quality Control Review
6. Verdict Processing & Retry Logic
7. Post-Execution Hooks
8. Multi-File Plan Coordination

### Behaviors Documented: 10+ behaviors
- Integration context injection
- Structured QC mode selection
- Dual-criteria validation
- Domain-specific criteria injection
- Per-file locking
- Agent swapping
- Learning adaptation
- Failure pattern extraction
- Metadata storage
- Plan file updates

---

## Quality Assurance

Each document includes:
- ✓ Exact code locations (file + line numbers)
- ✓ Field-by-field analysis
- ✓ Behavior documentation
- ✓ Code examples
- ✓ ASCII diagrams
- ✓ Decision trees
- ✓ Cross-references
- ✓ Testing guidance
- ✓ Use-case scenarios
- ✓ Glossary of terms

---

## Recommendations for Next Steps

### Immediate Use
1. Review TASK_FIELD_DOCS_INDEX.md for navigation
2. Read TASK_FIELD_ANALYSIS_SUMMARY.md for overview
3. Use TASK_FIELD_REFERENCE.md during development
4. Consult TASK_FIELD_TRACE.md when debugging

### Future Enhancements
Consider adding to documentation as features evolve:
1. EstimatedTime field usage (currently parsed but unused)
2. WorktreeGroup execution control (currently organizational only)
3. Structured logging for SourceFile tracking
4. Metadata extensions for custom hints
5. ML-based failure pattern analysis

### Testing Coverage
Priority test scenarios identified:
1. Integration context with various dependency structures
2. Structured QC with unanimous consensus
3. Agent swapping with learning history
4. Per-file locking with concurrent execution
5. Domain-specific criteria for each language
6. Metadata propagation through execution

---

## Statistics

| Metric | Value |
|--------|-------|
| Total documentation | 2,341 lines |
| Total size | ~94 KB |
| Documents | 5 files |
| Fields analyzed | 30+ |
| Code locations referenced | 50+ |
| Execution stages covered | 8 |
| Behaviors documented | 10+ |
| Code examples | 30+ |
| Diagrams | 10+ |

---

## Quick Start

**Start here**: `/Users/harrison/Github/conductor/TASK_FIELD_DOCS_INDEX.md`

From there, use the decision tree to navigate to the right document:
- New to this? → Summary
- Need to code? → Reference
- Debugging? → Trace
- Understanding flow? → Dataflow
- Lost? → Index

---

## Document File Locations

```
/Users/harrison/Github/conductor/
├── TASK_FIELD_DOCS_INDEX.md           (Navigation & Index)
├── TASK_FIELD_ANALYSIS_SUMMARY.md     (Overview & Findings)
├── TASK_FIELD_REFERENCE.md            (Practical Guide)
├── TASK_FIELD_TRACE.md                (Detailed Reference)
└── TASK_FIELD_DATAFLOW.md             (Flow Diagrams)
```

All files are committed to the repository and ready for team access.

---

## Conclusion

This comprehensive analysis provides a complete map of how Conductor's executor uses Task struct fields during execution. Every field is accounted for, every behavior is documented with code locations, and multiple navigation paths are provided for different use cases.

The analysis is production-ready and suitable for:
- Development guidance
- Debugging assistance
- Architecture understanding
- Team documentation
- Feature development reference
- Code review reference

**Status**: Complete, comprehensive, and ready for use.
