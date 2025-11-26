# Documentation Updates: Cross-File Dependencies (v2.6+)

Date: 2025-11-23
Version: Conductor v2.5.2 (v2.6 features documented for upcoming release)

## Overview

This document summarizes all documentation updates made to support Conductor's cross-file dependency feature (v2.6+). The cross-file dependency system enables explicit task references across multiple plan files with automatic validation.

## Files Updated

### 1. CLAUDE.md (Developer Reference)

**Location:** `/Users/harrison/Github/conductor/CLAUDE.md`

**Changes Made:**

#### Section: Project Overview (Line 9)
- Updated "Current Status" to reflect v2.5.2 with cross-file dependency support
- Added mention of cross-file dependencies alongside other major features
- Updated version from v2.4.0 to v2.5.2

**Old:**
```
Current Status: Production-ready v2.4.0 with comprehensive multi-agent orchestration...
```

**New:**
```
Current Status: Production-ready v2.5.2 with comprehensive multi-agent orchestration,
multi-file plan support with cross-file dependencies (v2.6+)...
```

#### Section: Task Metadata Parsing (Lines 499-562)
- Added `DependsOn` field description to include "local and cross-file, v2.6+"
- Added comprehensive "Cross-File Dependencies (v2.6+)" subsection
- Documented local vs cross-file reference syntax
- Provided YAML format examples
- Provided Markdown format pseudo-syntax documentation
- Documented SourceFile field (internal use)

**Added Content:**
```markdown
**Cross-File Dependencies (v2.6+)**:
- Local references: Task number (integer) refers to tasks in the same file
- Cross-file references: Object with `file` (path) and `task` (number) keys
- Syntax varies by format:

**YAML Format**:
tasks:
  - id: 1
    name: Setup Database
    depends_on: []

  - id: 2
    name: Auth Service
    depends_on:
      - 1                              # Local task in same file
      - file: foundation.yaml          # Cross-file reference
        task: 3                         # Task 3 from foundation.yaml
```

#### Section: Multi-File Plan Examples (Lines 875-1037)
- Renamed "Example 1" to "Split Backend Plan (Implicit Ordering)"
- Added "Example 2: Split Backend Plan with Explicit Cross-File Dependencies (v2.6+)"
  - Shows YAML multi-file setup with explicit cross-file dependencies
  - Demonstrates `file:` and `task:` notation
  - Includes execution and validation examples
- Added "Example 3: Microservices Plan with Multiple Files"
  - Three separate plans with complex cross-file dependencies
  - Shows auth-service.yaml, api-service.yaml, deployment.yaml
  - Demonstrates cross-file dependencies flowing through services
- Added "Example 4: Worktree Groups with Cross-File Dependencies"
  - Explains how worktree groups work with cross-file dependencies
  - Notes parallelism within groups enforced by cross-file links

#### Section: Production Status (Lines 582-644)
- Updated version from v2.4.0 to v2.5.2
- Updated major features list to include "Multi-file plan support with explicit cross-file dependencies (v2.6+)"
- Added "v2.5 Enhancements" subsection (integration tasks)
- Added "v2.6+ Enhancements (In Development)" subsection with:
  - Explicit cross-file dependency notation
  - Automatic validation of cross-file links
  - Enhanced plan merger
  - SourceFile field for tracking
  - Backward compatibility notes

### 2. README.md (User Guide)

**Location:** `/Users/harrison/Github/conductor/README.md`

**Changes Made:**

#### Section: Key Features (Lines 57-112)
- Updated "Dependency Management" subsection to include:
  - Local dependencies note
  - Cross-file dependencies with explicit notation (v2.6+)
  - Automatic validation of all dependency links
- Added dedicated "Multi-File Plans" feature (Lines 72-77) with:
  - Objective splitting mention
  - Cross-file dependency management (v2.6+)
  - Worktree groups
  - File-to-task mapping
  - Backward compatibility note
- Added dedicated "Cross-File Dependencies" feature (Lines 108-112) with:
  - File and task notation explanation
  - Automatic validation note
  - Microservices support mention

#### Section: Installation - Quick Install (Lines 132-148)
- Updated download URLs from v2.4.0 to v2.5.2
- Updated verification output from v2.4.0 to v2.5.2

#### Section: Installation - Build from Source (Line 168)
- Updated expected version output from v2.4.0 to v2.5.2

#### Section: Multi-File Plans (Lines 728-780)
- Updated introductory text to mention cross-file dependencies (v2.6+)
- Added examples in bash commands showing cross-file validation
- Expanded "Features" subsection:
  - Added "Explicit cross-file dependencies with validation (v2.6+)" with sub-bullets
  - Added "Implicit ordering" note for v2.5 backward compatibility
  - Kept existing features (multi-file loading, objective splitting, etc.)
- Added "Cross-File Dependency Example (v2.6+)" subsection with:
  - infrastructure.yaml sample
  - services.yaml with cross-file dependency
  - YAML syntax demonstration
  - Execution examples
- Updated documentation links to include CROSS_FILE_DEPENDENCIES.md

#### Section: Project Status (Line 786)
- Updated version from v2.5.0 to v2.5.2

### 3. docs/CROSS_FILE_DEPENDENCIES.md (Complete Reference)

**Location:** `/Users/harrison/Github/conductor/docs/CROSS_FILE_DEPENDENCIES.md`

**Status:** Already comprehensive and complete

**Contents Include:**
- Overview with use cases
- Complete syntax guide (YAML and Markdown)
- Multiple detailed examples (3-file backend, microservices, advanced patterns)
- Best practices for organizing multi-file plans
- Validation and error handling
- Backward compatibility notes
- Migration guide
- Troubleshooting section

This file serves as the definitive reference for cross-file dependency syntax and usage.

### 4. docs/MIGRATION_CROSS_FILE_DEPS.md (Migration Guide)

**Location:** `/Users/harrison/Github/conductor/docs/MIGRATION_CROSS_FILE_DEPS.md`

**Status:** Already comprehensive and complete

**Contents Include:**
- Clear explanation of why migration is beneficial
- Three migration levels (no migration, partial, complete)
- Step-by-step migration process with phases
- Before/after examples
- Common migration patterns
- Testing strategies
- Troubleshooting guide
- Rollback procedures

This file provides a clear path for users to migrate from implicit to explicit cross-file dependencies.

## Documentation Architecture

### User-Facing Documentation
- **README.md** - Quick start and overview (updated)
- **docs/conductor.md** - Complete reference with all features
- **docs/CROSS_FILE_DEPENDENCIES.md** - Dedicated cross-file dependency guide
- **docs/MIGRATION_CROSS_FILE_DEPS.md** - Migration guide for existing users

### Developer Documentation
- **CLAUDE.md** - Developer reference (updated)
- Includes implementation details, architecture, and code examples

## Key Documentation Updates

### 1. Version Updates
- Updated all version references from v2.4.0 to v2.5.2
- Added v2.6+ notes for features in development
- Clarified version boundaries for each feature set

### 2. Cross-File Dependency Examples
- **CLAUDE.md**: Added 4 multi-file examples showing:
  1. Implicit ordering (backward compatible)
  2. Explicit cross-file dependencies (new feature)
  3. Complex microservices scenario
  4. Integration with worktree groups

- **README.md**: Added concise YAML example demonstrating:
  1. infrastructure.yaml setup
  2. services.yaml with cross-file reference
  3. Execution command

### 3. Feature Documentation
- Expanded "Dependency Management" section to clarify:
  - Local dependencies (within single file)
  - Cross-file dependencies (between files)
  - Automatic validation
  - Use cases

- Added dedicated "Multi-File Plans" feature explanation:
  - Objective splitting strategies
  - Cross-file dependency management
  - Worktree groups
  - Resumable execution

- Added dedicated "Cross-File Dependencies" feature:
  - Explicit notation (file + task)
  - Automatic validation
  - Microservices support

### 4. Best Practices and Guidance
- Task Metadata Parsing section explains:
  - When to use cross-file references
  - Syntax in both YAML and Markdown
  - Internal SourceFile tracking
  - Compatibility with other features

### 5. Migration Path
- Complete backward compatibility maintained
- Existing implicit ordering still works
- Migration is optional but recommended for:
  - Complex multi-file plans
  - Large teams
  - Microservices architectures

## Consistency Across Documentation

### Terminology
- **Cross-file dependency**: Explicit reference to task in another file
- **Local dependency**: Reference to task in same file
- **Implicit ordering**: File order determines execution (v2.5)
- **Explicit notation**: `file:` and `task:` keys (v2.6+)

### Version Markers
- `(v2.6+)` - Features introduced in v2.6
- `(v2.5+)` - Features introduced in v2.5
- `(In Development)` - Features being added to v2.6

### Code Examples
- All examples use consistent formatting
- YAML examples are complete and executable
- Markdown pseudo-syntax clearly marked as reference
- Comments explain complex dependencies

## Validation and Testing Recommendations

### Documentation Quality Checklist
- [x] All version numbers are consistent (v2.5.2)
- [x] Cross-file dependency syntax is consistent across files
- [x] Examples are clear and follow the same pattern
- [x] Backward compatibility is mentioned in all relevant sections
- [x] Migration guidance is clear and actionable
- [x] Feature descriptions are accurate and complete
- [x] All links to related documentation are present

### Suggested User Testing
1. Users should read README.md "Multi-File Plans" section
2. Refer to docs/CROSS_FILE_DEPENDENCIES.md for detailed syntax
3. Use docs/MIGRATION_CROSS_FILE_DEPS.md if migrating existing plans
4. Consult CLAUDE.md multi-file examples for developer reference

## Documentation Maintenance Notes

### Future Updates Required
- When v2.6 releases, update version markers from "(v2.6+)" to just note v2.6
- Add any final implementation details discovered during v2.6 development
- Update examples if any edge cases are discovered
- Add performance notes if applicable

### Documentation Files to Monitor
- **docs/conductor.md** - Main reference (may need updates for v2.6 specifics)
- **docs/CROSS_FILE_DEPENDENCIES.md** - Already comprehensive, stable
- **docs/MIGRATION_CROSS_FILE_DEPS.md** - Already comprehensive, stable
- **CLAUDE.md** - May need implementation detail updates during v2.6 development
- **README.md** - Should remain stable unless feature changes

## Summary

All user-facing and developer-facing documentation has been comprehensively updated to reflect Conductor's cross-file dependency support. The updates include:

1. **2 primary documentation files updated** (README.md, CLAUDE.md)
2. **2 dedicated reference files** maintained and verified (CROSS_FILE_DEPENDENCIES.md, MIGRATION_CROSS_FILE_DEPS.md)
3. **4 detailed multi-file examples** added across documentation
4. **Version consistency** maintained throughout (v2.5.2 as current, v2.6+ for features in development)
5. **Backward compatibility** clearly documented
6. **Migration guidance** provided for existing users

The documentation is now complete, consistent, and ready to guide users through cross-file dependency features while maintaining clarity about which features are available in current vs. upcoming releases.

## Files Modified

1. `/Users/harrison/Github/conductor/CLAUDE.md` - Developer reference
2. `/Users/harrison/Github/conductor/README.md` - User guide

## Files Verified (No Changes Needed)

1. `/Users/harrison/Github/conductor/docs/CROSS_FILE_DEPENDENCIES.md` - Already comprehensive
2. `/Users/harrison/Github/conductor/docs/MIGRATION_CROSS_FILE_DEPS.md` - Already comprehensive
3. `/Users/harrison/Github/conductor/docs/conductor.md` - Already has cross-file references
