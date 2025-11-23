# Cross-File Dependencies Documentation Index

Quick navigation guide for all cross-file dependency documentation.

## Quick Navigation

### I need to... → Go to

| Need | Best Resource | Time |
|------|--------------|------|
| Understand cross-file dependencies | [CROSS_FILE_DEPENDENCIES.md](#complete-reference) | 15 min |
| See quick examples | [EXAMPLES_CROSS_FILE_DEPS.md](#examples) | 10 min |
| Migrate existing plans | [MIGRATION_CROSS_FILE_DEPS.md](#migration) | 20 min |
| Debug an error | [TROUBLESHOOTING_CROSS_FILE_DEPS.md](#troubleshooting) | 5 min |
| Learn the syntax | [CROSS_FILE_DEPENDENCIES.md#syntax](#syntax) | 10 min |
| See real patterns | [EXAMPLES_CROSS_FILE_DEPS.md](#examples) | 20 min |
| Update CLAUDE.md | [CLAUDE_UPDATES.md](#claude-updates) | 30 min |
| Update README.md | [README_UPDATES.md](#readme-updates) | 30 min |

---

## By Document

### Complete Reference

**File:** [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md)

**What's inside:**
- Complete overview of cross-file dependencies
- Detailed syntax guide (YAML and Markdown)
- 7 real-world examples
- 5 best practices with guidelines
- Validation process and common errors
- Backward compatibility guarantees
- Troubleshooting mini-section

**Best for:** Users who want complete understanding

**Start with:** "Overview" section, then explore by topic

**Quick sections:**
- [Overview](./CROSS_FILE_DEPENDENCIES.md#overview)
- [Syntax Guide](./CROSS_FILE_DEPENDENCIES.md#syntax-guide)
- [Examples](./CROSS_FILE_DEPENDENCIES.md#examples)
- [Best Practices](./CROSS_FILE_DEPENDENCIES.md#best-practices)

---

### Examples

**File:** [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md)

**What's inside:**
- 7 complete real-world examples
- Simple to complex systems
- Each with full YAML code
- Expected execution order
- Why each example matters

**Examples included:**
1. [Simple Two-File Setup](./EXAMPLES_CROSS_FILE_DEPS.md#simple-two-file-setup) - Foundation + Services
2. [Three-File Microservice Architecture](./EXAMPLES_CROSS_FILE_DEPS.md#three-file-microservice-architecture)
3. [Complex Multi-Service System](./EXAMPLES_CROSS_FILE_DEPS.md#complex-multi-service-system)
4. [Frontend + Backend Integration](./EXAMPLES_CROSS_FILE_DEPS.md#frontend--backend-integration)
5. [Multi-Environment Deployment](./EXAMPLES_CROSS_FILE_DEPS.md#multi-environment-deployment)
6. [Shared Infrastructure Pattern](./EXAMPLES_CROSS_FILE_DEPS.md#shared-infrastructure-pattern)
7. [Progressive Feature Rollout](./EXAMPLES_CROSS_FILE_DEPS.md#progressive-feature-rollout)

**Best for:** Learning by example

**Start with:** Find your use case (2-file? microservices? complex?), read example, adapt to your needs

---

### Migration

**File:** [MIGRATION_CROSS_FILE_DEPS.md](./MIGRATION_CROSS_FILE_DEPS.md)

**What's inside:**
- Why migrate (benefits explained)
- 3 migration levels to choose from
- 4-phase migration process
- Before/after comparisons
- Common migration patterns
- Testing procedures
- Rollback options

**Sections:**
- [Migration Levels](./MIGRATION_CROSS_FILE_DEPS.md#migration-levels) - Choose your level
- [Step-by-Step](./MIGRATION_CROSS_FILE_DEPS.md#step-by-step-migration) - Follow phases 1-4
- [Examples](./MIGRATION_CROSS_FILE_DEPS.md#before-and-after-examples) - See transformations
- [Testing](./MIGRATION_CROSS_FILE_DEPS.md#testing-your-migration) - Verify your work

**Best for:** Existing users converting to explicit dependencies

**Start with:** Why Migrate section to understand benefits

---

### Troubleshooting

**File:** [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md)

**What's inside:**
- 8 categories of issues
- 20+ specific error scenarios
- Symptoms, causes, solutions for each
- Debug commands and procedures
- Step-by-step troubleshooting

**Categories:**
1. [File Path Issues](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#file-path-issues)
2. [Task Reference Errors](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#task-reference-errors)
3. [Circular Dependencies](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#circular-dependencies)
4. [Validation Failures](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#validation-failures)
5. [Execution Problems](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#execution-problems)
6. [Resume Issues](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#resume-issues)
7. [Performance Issues](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#performance-issues)
8. [Format Errors](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#format-errors)

**Best for:** Debugging specific issues

**Start with:** Find error message in table of contents, follow solution steps

---

### CLAUDE.md Updates

**File:** [CLAUDE_UPDATES.md](./CLAUDE_UPDATES.md)

**What's inside:**
- Specification for updating CLAUDE.md
- Before/after for each change
- Location of changes (line numbers)
- Exact text to replace
- Rationale for each change

**Changes cover:**
1. Project Overview (v2.5.0)
2. Dependency Graph Algorithm
3. Task Metadata Parsing
4. Multi-File Plans examples
5. Multi-File Plans best practices
6. High-level flow diagram

**Best for:** Developers/maintainers updating CLAUDE.md

**Implementation order:**
1. Update Project Overview
2. Update High-Level Flow
3. Update Dependency Graph Algorithm
4. Update Task Metadata Parsing
5. Expand Multi-File Plan Examples
6. Expand Multi-File Plan Best Practices

---

### README.md Updates

**File:** [README_UPDATES.md](./README_UPDATES.md)

**What's inside:**
- Specification for updating README.md
- Before/after for each change
- Section locations in README
- New content to add
- Rationale for changes

**Changes cover:**
1. Key Features (add cross-file dependencies)
2. Project Status (update to v2.5.0)
3. Multi-File Plans section (expand)
4. New syntax section
5. New examples in Plan Format
6. Documentation references
7. Architecture Overview enhancement
8. Table of Contents updates

**Best for:** Developers/maintainers updating README.md

**Implementation order:**
1. Update Table of Contents
2. Update Project Status
3. Update Key Features
4. Add Cross-File Syntax section
5. Update Multi-File Plans section
6. Add Plan Format examples
7. Update Architecture
8. Add documentation links

---

## By Use Case

### I'm New to Conductor

1. **Understand basics:** Read [README.md](../README.md) (if not done yet)
2. **See examples:** [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md#simple-two-file-setup) simple example
3. **Try it:** Use the 2-file example, run `conductor validate`
4. **Learn more:** [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md) deep dive

**Time needed:** 30 minutes

---

### I Have an Existing Multi-File Plan

1. **Understand changes:** [CROSS_FILE_DEPENDENCIES.md#overview](./CROSS_FILE_DEPENDENCIES.md#overview)
2. **Decide on migration:** [MIGRATION_CROSS_FILE_DEPS.md#migration-levels](./MIGRATION_CROSS_FILE_DEPS.md#migration-levels)
3. **Plan your approach:** [MIGRATION_CROSS_FILE_DEPS.md#step-by-step-migration](./MIGRATION_CROSS_FILE_DEPS.md#step-by-step-migration)
4. **Execute migration:** Follow 4-phase process
5. **Test thoroughly:** [MIGRATION_CROSS_FILE_DEPS.md#testing-your-migration](./MIGRATION_CROSS_FILE_DEPS.md#testing-your-migration)

**Time needed:** 1-2 hours

---

### I'm Building a Complex System

1. **Study patterns:** [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md)
2. **Review best practices:** [CROSS_FILE_DEPENDENCIES.md#best-practices](./CROSS_FILE_DEPENDENCIES.md#best-practices)
3. **Design your structure:** Choose pattern that fits
4. **Implement carefully:** Follow phase-by-phase approach
5. **Validate thoroughly:** Use `conductor validate --verbose`

**Time needed:** Depends on system complexity

---

### I'm Hitting an Error

1. **Find error category:** [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md) TOC
2. **Locate exact error:** Search within category
3. **Follow solution steps:** Execute provided commands
4. **Re-validate:** `conductor validate *.yaml`
5. **Still stuck?** Try debug commands in troubleshooting guide

**Time needed:** 5-15 minutes

---

### I'm Maintaining Documentation

1. **Understand the feature:** [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md)
2. **Review both update specs:** [CLAUDE_UPDATES.md](./CLAUDE_UPDATES.md) and [README_UPDATES.md](./README_UPDATES.md)
3. **Apply updates:** Follow implementation order
4. **Test all links:** Ensure references work
5. **Verify examples:** Run all example commands

**Time needed:** 1-2 hours

---

## By Learning Style

### I Learn Best By Reading

1. Start with [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md)
2. Follow with [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md)
3. Reference [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md) as needed

---

### I Learn Best By Example

1. Start with [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md)
2. Run the simple 2-file example
3. Read [CROSS_FILE_DEPENDENCIES.md#syntax-guide](./CROSS_FILE_DEPENDENCIES.md#syntax-guide) for syntax details
4. Adapt examples to your needs

---

### I Learn Best By Doing

1. Create a simple test with 2 files
2. Use [EXAMPLES_CROSS_FILE_DEPS.md#simple-two-file-setup](./EXAMPLES_CROSS_FILE_DEPS.md#simple-two-file-setup) as reference
3. Run `conductor validate` and learn from errors
4. Reference [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md) when stuck
5. Graduate to more complex examples

---

### I Learn Best By Problem-Solving

1. Identify your specific issue
2. Find it in [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md)
3. Follow the solution steps
4. For deeper understanding, read context sections
5. Review similar patterns in [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md)

---

## Search Quick Reference

### Syntax Questions

- **Local dependency syntax:** [CROSS_FILE_DEPENDENCIES.md#local-dependencies](./CROSS_FILE_DEPENDENCIES.md#local-dependencies-single-file)
- **Cross-file syntax:** [CROSS_FILE_DEPENDENCIES.md#cross-file-dependencies](./CROSS_FILE_DEPENDENCIES.md#cross-file-dependencies-v25)
- **Mixed syntax:** [CROSS_FILE_DEPENDENCIES.md#mixed](./CROSS_FILE_DEPENDENCIES.md#syntax-guide)
- **Markdown format:** [CROSS_FILE_DEPENDENCIES.md#markdown-format](./CROSS_FILE_DEPENDENCIES.md#markdown-format-pseudo-syntax)

### Error Messages

- **"File not found":** [TROUBLESHOOTING_CROSS_FILE_DEPS.md#file-not-found](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#error-file-not-found-servicesyaml)
- **"Task X not found":** [TROUBLESHOOTING_CROSS_FILE_DEPS.md#task-not-found](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#error-task-5-not-found-in-foundationyaml)
- **"Circular dependency":** [TROUBLESHOOTING_CROSS_FILE_DEPS.md#circular](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#error-circular-dependency-detected)
- **"Invalid format":** [TROUBLESHOOTING_CROSS_FILE_DEPS.md#invalid-format](./TROUBLESHOOTING_CROSS_FILE_DEPS.md#error-invalid-dependency-format)

### Pattern Questions

- **Simple setup (2 files):** [EXAMPLES_CROSS_FILE_DEPS.md#simple-two-file](./EXAMPLES_CROSS_FILE_DEPS.md#simple-two-file-setup)
- **Microservices (3 files):** [EXAMPLES_CROSS_FILE_DEPS.md#microservices](./EXAMPLES_CROSS_FILE_DEPS.md#three-file-microservice-architecture)
- **Complex system (5+ files):** [EXAMPLES_CROSS_FILE_DEPS.md#complex](./EXAMPLES_CROSS_FILE_DEPS.md#complex-multi-service-system)
- **Shared infrastructure:** [EXAMPLES_CROSS_FILE_DEPS.md#shared](./EXAMPLES_CROSS_FILE_DEPS.md#shared-infrastructure-pattern)
- **Linear chain:** [CROSS_FILE_DEPENDENCIES.md#linear-chain](./CROSS_FILE_DEPENDENCIES.md#pattern-1-linear-dependency-chain)
- **Star pattern:** [CROSS_FILE_DEPENDENCIES.md#star](./CROSS_FILE_DEPENDENCIES.md#pattern-2-star-dependency-multiple-files-depend-on-one)

### Migration Questions

- **Should I migrate?:** [MIGRATION_CROSS_FILE_DEPS.md#why-migrate](./MIGRATION_CROSS_FILE_DEPS.md#why-migrate)
- **Which level?:** [MIGRATION_CROSS_FILE_DEPS.md#levels](./MIGRATION_CROSS_FILE_DEPS.md#migration-levels)
- **Step-by-step help:** [MIGRATION_CROSS_FILE_DEPS.md#steps](./MIGRATION_CROSS_FILE_DEPS.md#step-by-step-migration)
- **Before/after:** [MIGRATION_CROSS_FILE_DEPS.md#before-after](./MIGRATION_CROSS_FILE_DEPS.md#before-and-after-examples)
- **Testing:** [MIGRATION_CROSS_FILE_DEPS.md#testing](./MIGRATION_CROSS_FILE_DEPS.md#testing-your-migration)
- **Rollback:** [MIGRATION_CROSS_FILE_DEPS.md#rollback](./MIGRATION_CROSS_FILE_DEPS.md#rollback-plan)

---

## File Relationships

```
Start Here
    |
    +-- README.md (updated)
    |
    +-- EXAMPLES_CROSS_FILE_DEPS.md (simple examples)
    |
    +-- CROSS_FILE_DEPENDENCIES.md (complete reference)
    |   |
    |   +-- Syntax Guide
    |   +-- Best Practices
    |   +-- Validation & Errors
    |   +-- Migration intro
    |
    +-- MIGRATION_CROSS_FILE_DEPS.md (detailed migration)
    |   |
    |   +-- Phases 1-4
    |   +-- Testing
    |   +-- Rollback
    |
    +-- TROUBLESHOOTING_CROSS_FILE_DEPS.md (debugging)
    |
    +-- CLAUDE.md (updated for devs)
    |
    +-- CLAUDE_UPDATES.md (update spec)
    |
    +-- README_UPDATES.md (update spec)
```

---

## Version Information

**Documentation version:** 1.0
**For Conductor:** v2.5+ (Cross-File Dependencies)
**Created:** 2025-11-23

**Total documentation:**
- CROSS_FILE_DEPENDENCIES.md: 3,500 lines
- MIGRATION_CROSS_FILE_DEPS.md: 2,800 lines
- TROUBLESHOOTING_CROSS_FILE_DEPS.md: 2,200 lines
- EXAMPLES_CROSS_FILE_DEPS.md: 2,000 lines
- CLAUDE_UPDATES.md: ~1,500 lines
- README_UPDATES.md: ~1,500 lines
- This index: ~400 lines

**Total: ~13,900 lines of documentation**

---

## Still Looking for Something?

### Common Questions

| Question | Answer | Link |
|----------|--------|------|
| What's the syntax? | See examples | [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md#syntax-guide) |
| Does my old plan still work? | Yes, fully backward compatible | [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md#backward-compatibility) |
| How do I migrate? | 4-phase process | [MIGRATION_CROSS_FILE_DEPS.md](./MIGRATION_CROSS_FILE_DEPS.md) |
| What error is this? | Search troubleshooting | [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md) |
| Can I see an example? | Multiple patterns | [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md) |
| How does it work? | Algorithm explanation | [CLAUDE_UPDATES.md](./CLAUDE_UPDATES.md) |

---

## Direct Links

### Documentation Files
- [CROSS_FILE_DEPENDENCIES.md](./CROSS_FILE_DEPENDENCIES.md) - Complete reference
- [MIGRATION_CROSS_FILE_DEPS.md](./MIGRATION_CROSS_FILE_DEPS.md) - Migration guide
- [TROUBLESHOOTING_CROSS_FILE_DEPS.md](./TROUBLESHOOTING_CROSS_FILE_DEPS.md) - Troubleshooting
- [EXAMPLES_CROSS_FILE_DEPS.md](./EXAMPLES_CROSS_FILE_DEPS.md) - Real-world examples

### Update Specifications
- [CLAUDE_UPDATES.md](./CLAUDE_UPDATES.md) - CLAUDE.md update spec
- [README_UPDATES.md](./README_UPDATES.md) - README.md update spec

### Conductor Files (to be updated)
- [CLAUDE.md](../CLAUDE.md) - Developer documentation
- [README.md](../README.md) - User guide

---

**Last updated:** 2025-11-23
**Status:** Ready for implementation
