# Runtime Enforcement Documentation

This file serves as a documentation target for runtime enforcement integration tests.

# Configuration

The runtime enforcement system provides several toggles to control task execution behavior:

- `enforce_dependency_checks`: Run dependency check commands before agent invocation
- `enforce_test_commands`: Run test commands after agent output but before QC
- `verify_criteria`: Run per-criterion verification commands
- `enforce_package_guard`: Prevent concurrent Go package modifications
- `enforce_doc_targets`: Verify documentation targets before QC

All toggles default to `true` for plans with `RuntimeMetadata`.

# Integration

Integration tasks combine multiple enforcement features:

1. **Preflight Phase**: Dependency checks run before agent starts
2. **Execution Phase**: Agent performs task with package guard active
3. **Postflight Phase**: Test commands and criterion verification run
4. **QC Phase**: Documentation targets verified, results passed to QC

This ensures a complete validation pipeline for each task.
