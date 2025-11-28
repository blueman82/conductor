# Task Field Analysis - Documentation Index

## Quick Navigation

This comprehensive analysis traces how Conductor's executor uses Task struct fields during execution. Choose the document that best fits your needs:

---

## Documents Overview

### 1. TASK_FIELD_ANALYSIS_SUMMARY.md (START HERE)
**When to read**: First overview, high-level understanding
**Size**: ~13 KB
**Content**:
- Executive summary of key findings
- Critical fields and behaviors
- Field usage by execution stage
- Important code locations
- Testing considerations
- Conclusions and recommendations

**Best for**: Getting oriented, understanding the big picture, finding which detailed doc to read next

---

### 2. TASK_FIELD_REFERENCE.md (PRACTICAL GUIDE)
**When to read**: During development, looking for patterns
**Size**: ~12 KB
**Content**:
- Quick reference for field behaviors
- Integration context injection explanation
- Structured vs. legacy QC rules
- Domain-specific QC injection
- Multi-file coordination details
- Inter-retry agent swapping logic
- Learning system integration
- Field modification timeline
- Common code patterns
- Success criteria examples
- File-level locking explanation

**Best for**: Day-to-day development, copying common patterns, quick lookups

---

### 3. TASK_FIELD_TRACE.md (DETAILED REFERENCE)
**When to read**: Deep dive, debugging, understanding specific behaviors
**Size**: ~26 KB
**Content**:
- Complete task struct field definitions
- Field-by-field analysis with line numbers
- Parser details (YAML vs. Markdown)
- Parsing validation rules
- Pre-execution hook details
- QC integration patterns
- Learning hook integration
- Metadata field usage
- Fields modified during execution
- Fields not used (and why)
- Field dependencies and interactions
- Integration context injection details
- Parsing: where each field comes from
- Summary table of field usage

**Best for**: Debugging issues, understanding edge cases, finding exact code locations

---

### 4. TASK_FIELD_DATAFLOW.md (VISUAL GUIDE)
**When to read**: Understanding execution flow, complex interactions
**Size**: ~30 KB
**Content**:
- End-to-end ASCII flowcharts
- Stage-by-stage processing
- Decision point diagrams
- Critical decision tree logic
- Field dependency graph
- Execution state machine
- Multi-file coordination visual
- Key field modifications timeline

**Best for**: Understanding complex flows, presentation/documentation, architectural understanding

---

## Quick Decision Tree

```
I want to...

├─ Get started quickly
│  └─ Read: TASK_FIELD_ANALYSIS_SUMMARY.md

├─ Find a code example
│  └─ Read: TASK_FIELD_REFERENCE.md (Common Patterns section)

├─ Debug a specific issue
│  └─ Read: TASK_FIELD_TRACE.md
│     └─ Look for the field name in the index

├─ Understand the big picture
│  └─ Read: TASK_FIELD_DATAFLOW.md
│     └─ Then: TASK_FIELD_ANALYSIS_SUMMARY.md

├─ Find exact line numbers
│  └─ Read: TASK_FIELD_TRACE.md
│     └─ Search for "Location:" or "Line"

├─ Understand multi-file execution
│  └─ Read: TASK_FIELD_REFERENCE.md (Multi-File Coordination section)
│     └─ Then: TASK_FIELD_DATAFLOW.md (STAGE 8)

├─ Understand QC logic
│  └─ Read: TASK_FIELD_REFERENCE.md (Structured QC section)
│     └─ Then: TASK_FIELD_TRACE.md (quality control impact fields)
│     └─ Then: TASK_FIELD_DATAFLOW.md (STAGE 5)

└─ Understand learning integration
   └─ Read: TASK_FIELD_REFERENCE.md (Learning System Integration section)
      └─ Then: TASK_FIELD_TRACE.md (Learning Integration section)
      └─ Then: TASK_FIELD_DATAFLOW.md (STAGE 3, 7)
```

---

## Key Topics by Document

| Topic | Summary | Reference | Trace | Dataflow |
|-------|---------|-----------|-------|----------|
| Critical fields | ✓✓ | ✓ | ✓✓ | ✓ |
| Integration context | ✓✓ | ✓✓ | ✓✓ | ✓✓ |
| Structured QC | ✓✓ | ✓✓ | ✓✓ | ✓ |
| Domain checks | ✓ | ✓✓ | ✓ | ✓ |
| Multi-file plans | ✓ | ✓✓ | ✓ | ✓✓ |
| Agent swapping | ✓ | ✓ | ✓ | ✓ |
| Learning system | ✓ | ✓✓ | ✓✓ | ✓ |
| Code patterns | | ✓✓ | ✓ | |
| Field locations | | | ✓✓ | |
| Execution flow | ✓ | ✓ | ✓ | ✓✓ |
| Decision logic | ✓ | ✓ | ✓ | ✓✓ |
| Testing guidance | ✓ | | ✓ | |
| Recommendations | ✓ | | | |

Legend: ✓ = covered, ✓✓ = extensively covered, blank = not covered

---

## Topic Quick Links

### Integration Context Injection
- **Overview**: Summary → "Integration Context Injection"
- **Implementation**: Reference → "Integration Context Injection (Pre-Execution)"
- **Details**: Trace → "Integration Context Injection"
- **Flow**: Dataflow → "STAGE 3"

### Structured QC vs Legacy QC
- **Overview**: Summary → "Structured QC vs Legacy QC"
- **Usage**: Reference → "Structured QC vs. Legacy QC"
- **Details**: Trace → "Quality Control Impact Fields"
- **Flow**: Dataflow → "STAGE 5"

### Multi-File Plans
- **Overview**: Summary → "Multi-File Plan Coordination"
- **Coordination**: Reference → "Multi-File Plan Coordination"
- **Details**: Trace → "SourceFile"
- **Flow**: Dataflow → "STAGE 8"

### Agent Swapping
- **Overview**: Summary → "Inter-Retry Agent Swapping"
- **Pattern**: Reference → "Inter-Retry Agent Swapping"
- **Details**: Trace → "Agent"
- **Flow**: Dataflow → "STAGE 6 (Verdict Processing)"

### Learning Integration
- **Overview**: Summary → "Learning System Integration"
- **Patterns**: Reference → "Learning System Integration"
- **Details**: Trace → "Learning Integration"
- **Flow**: Dataflow → "STAGE 3" and "STAGE 7"

### Field Modifications
- **Overview**: Summary → "Fields Modified During Execution"
- **Timeline**: Reference → "Fields Modified During Execution"
- **Details**: Trace → "Field Access by Executor Component"
- **Flow**: Dataflow → "KEY FIELD MODIFICATIONS"

### Domain-Specific Checks
- **Overview**: Summary → "Domain-Specific QC Injection"
- **Details**: Reference → "Domain-Specific QC Injection"
- **Code**: Trace → "Domain-Specific QC Injection"
- **Flow**: Dataflow → "STAGE 5 (Quality Control Review)"

### Per-File Locking
- **Overview**: Summary → "Per-File Locking in Multi-File Plans"
- **Pattern**: Reference → "File-Level Locking in Multi-File Plans"
- **Implementation**: Trace → "SourceFile"
- **Flow**: Dataflow → "STAGE 8" and "STAGE 4"

---

## Code Location Index

| Concept | File | Reference Doc |
|---------|------|---|
| Task struct | `internal/models/task.go` | Trace |
| Integration prompt | `internal/executor/integration_prompt.go` | Trace + Reference |
| Default task executor | `internal/executor/task.go` | Trace (all major behaviors) |
| Quality controller | `internal/executor/qc.go` | Trace (QC fields) |
| YAML parser | `internal/parser/yaml.go` | Trace (Parsing section) |
| Markdown parser | `internal/parser/markdown.go` | Trace (Parsing section) |
| Wave execution | `internal/executor/wave.go` | Trace (Status field) |
| Learning store | `internal/learning/` | Trace (Learning Integration) |

---

## Critical Behaviors

### Behavior 1: Integration Context Injection
- **Entry point**: `DefaultTaskExecutor.Execute()` line 517-519
- **Trigger**: `DependsOn` not empty OR `Type == "integration"`
- **Effect**: Modifies `Prompt` field BEFORE agent invocation
- **Find details in**: Trace "Integration Context Injection"

### Behavior 2: Structured QC Mode
- **Entry point**: `QualityController.Review()` line 348-356
- **Trigger**: `SuccessCriteria` array not empty
- **Effect**: Switches to per-criterion verification with unanimous consensus
- **Find details in**: Reference "Structured QC vs. Legacy QC"

### Behavior 3: Domain-Specific QC Injection
- **Entry point**: `qc.go` line 208
- **Trigger**: File extensions in `Files` field
- **Effect**: Auto-injects language-specific review criteria
- **Find details in**: Reference "Domain-Specific QC Injection"

### Behavior 4: Per-File Locking
- **Entry point**: `DefaultTaskExecutor.Execute()` line 490-498
- **Trigger**: Multi-file plan with `SourceFile` set
- **Effect**: Serializes file updates, allows parallel execution
- **Find details in**: Reference "File-Level Locking in Multi-File Plans"

### Behavior 5: Agent Swapping
- **Entry point**: `DefaultTaskExecutor.Execute()` line 708-728
- **Trigger**: RED verdict + swap enabled + threshold reached
- **Effect**: May change `Agent` field for next retry attempt
- **Find details in**: Reference "Inter-Retry Agent Swapping"

### Behavior 6: Learning Adaptation
- **Entry point**: `DefaultTaskExecutor.preTaskHook()` line 213-255
- **Trigger**: Learning store configured + failures detected
- **Effect**: Adapts `Agent` and enhances `Prompt` from history
- **Find details in**: Reference "Learning System Integration"

---

## Common Development Scenarios

### Scenario 1: I'm adding a new field to Task
1. Read: Trace → "Task Struct Definition"
2. Add to: `internal/models/task.go`
3. Update: Parser (YAML and/or Markdown)
4. Integrate: Executor hooks if needed
5. Consider: Testing implications (Summary → "Testing Considerations")

### Scenario 2: QC verdicts are wrong
1. Read: Reference → "QC Verdict Rules"
2. Check: Trace → "Quality Control Impact Fields"
3. Debug using: Dataflow → "STAGE 5"
4. Look for: Criteria aggregation logic (qc.go lines 826-934)

### Scenario 3: Agent isn't swapping as expected
1. Read: Reference → "Inter-Retry Agent Swapping"
2. Check: Trace → "Agent" field details
3. Verify: SwapDuringRetries enabled, MinFailuresBeforeAdapt reached
4. Debug using: Dataflow → "STAGE 6 (Agent Swap Decision)"

### Scenario 4: Multi-file plans have corrupted files
1. Read: Reference → "File-Level Locking in Multi-File Plans"
2. Check: Trace → "SourceFile" field
3. Verify: FileLockManager is acquiring locks
4. Debug using: Dataflow → "STAGE 8"

### Scenario 5: Integration context isn't being injected
1. Read: Reference → "Integration Context Injection (Pre-Execution)"
2. Check: Trace → "Integration Context Injection"
3. Verify: DependsOn is set OR Type=="integration"
4. Debug using: Dataflow → "STAGE 3"

### Scenario 6: Learning isn't working
1. Read: Reference → "Learning System Integration"
2. Check: Trace → "Learning Integration"
3. Verify: LearningStore is configured
4. Debug using: Dataflow → "STAGE 3" and "STAGE 7"

---

## File Statistics

| Document | Size | Lines | Focus |
|----------|------|-------|-------|
| TASK_FIELD_ANALYSIS_SUMMARY.md | 13 KB | ~300 | Overview & conclusions |
| TASK_FIELD_REFERENCE.md | 12 KB | ~450 | Practical patterns |
| TASK_FIELD_TRACE.md | 26 KB | ~800 | Detailed reference |
| TASK_FIELD_DATAFLOW.md | 30 KB | ~750 | Visual flows |
| **Total** | **81 KB** | **~2,300** | Complete analysis |

---

## Related Documentation

These documents complement the task field analysis:

- **CLAUDE.md** - Project overview, architecture, development commands
- **Go code files** - Source of truth for actual behavior
- **Tests** - Validate field usage and behaviors
- **Git history** - Track field changes over versions

---

## How to Update These Docs

If Task fields or executor behavior changes:

1. Update source code
2. Update **Trace** document with new code locations
3. Update **Reference** document with new patterns
4. Update **Dataflow** document with new flow diagrams
5. Update **Summary** with new findings
6. Update this index if new documents added

---

## Questions & Answers

**Q: Where do I find the exact line of code that does X?**
A: Use TASK_FIELD_TRACE.md - search for the field name or behavior

**Q: I need a code example of pattern Y**
A: Use TASK_FIELD_REFERENCE.md - look in "Common Patterns" section

**Q: I want to understand how feature Z flows through execution**
A: Use TASK_FIELD_DATAFLOW.md - follow the stage-by-stage flow

**Q: What are the key behaviors I need to know?**
A: Use TASK_FIELD_ANALYSIS_SUMMARY.md - read "Key Findings" section

**Q: Which fields actually affect execution?**
A: Use TASK_FIELD_REFERENCE.md → "Unified Field Behaviors" table

**Q: What gets modified during execution?**
A: Use TASK_FIELD_REFERENCE.md → "Fields Modified During Execution" table

**Q: How do integration tasks work?**
A: Use TASK_FIELD_REFERENCE.md → "Success Criteria Examples"

**Q: Why is multi-file execution safe?**
A: Use TASK_FIELD_REFERENCE.md → "File-Level Locking in Multi-File Plans"

---

## Glossary of Key Terms

- **Integration Context**: Dependency file list injected into Prompt
- **Structured QC**: Per-criterion verification with unanimous consensus
- **Domain-Specific Checks**: Language-specific criteria auto-injected based on file extensions
- **Integration Task**: Task marked with Type="integration" for dual-criteria validation
- **Agent Swapping**: Changing Agent field between retry attempts
- **Per-File Locking**: File-level synchronization for multi-file plan safety
- **Unanimous Consensus**: All agents must agree for criterion PASS (not just majority)
- **Learning Hooks**: Pre/QC-review/post-task integration points for adaptive behavior

---

## Version Information

- **Analysis Date**: 2025-11-22
- **Conductor Version**: 2.5.2
- **Go Version**: 1.21+
- **Scope**: Internal executor package field usage

---

## Contact & Contributions

For questions or updates to this analysis:
1. Review the relevant document
2. Check the source code (locations in Trace)
3. Run relevant tests
4. Update documentation
5. Commit changes with reference to this analysis

---

**Last Updated**: 2025-11-22
**Status**: Complete and comprehensive
