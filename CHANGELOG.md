# Changelog

All notable changes to Conductor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-01-12

### Added

**Adaptive Learning System**
- Learning store with persistent execution history tracking
- Session management with auto-generated session IDs
- Run number tracking per plan file for cross-run intelligence
- Pattern analysis for identifying common failure types (compilation errors, test failures, dependency issues)
- Pre-task hook that loads failure history and injects context into task prompts
- Post-task hook that stores execution outcomes for future learning
- Automatic agent adaptation after repeated failures
- Graceful degradation when learning system encounters errors

**Learning CLI Commands**
- `conductor learning stats` - View learning statistics and insights
- `conductor learning show <session-id>` - Display detailed session information
- `conductor learning export` - Export learning data to JSON for external analysis
- `conductor learning clear` - Clear learning history with confirmation prompt

**Configuration**
- Learning configuration section in `.conductor/config.yaml`
- `learning.enabled` flag to toggle learning system (default: true)
- `learning.dir` to specify storage directory (default: .conductor/learning)
- `learning.max_history` to limit stored executions per task (default: 100)

**Multi-File Plan Support**
- File-aware plan merging with cross-file dependency validation
- Source file tracking for task origin identification
- Worktree group validation in single-file plans
- Improved error handling for multi-file plan operations

**Developer Experience**
- Colorful progress indicators for multi-file loading
- Warning system for potentially problematic file patterns
- Numerical sorting for tasks within execution waves
- Enhanced display package with progress indicators and utilities

### Changed

- Orchestrator now integrates with learning store for history tracking
- TaskExecutor includes three hook points: pre-task, post-task, and result hooks
- Configuration schema extended with learning subsystem settings
- Parser enhanced with `FilterPlanFiles` utility for plan file discovery
- Display output now uses cyan for progress indicators (previously blue)

### Fixed

- Apply `--skip-completed` filter correctly in `--dry-run` mode
- Populate `SourceFile` field in `parser.MergePlans` for multi-file plans
- Use `Task.SourceFile` for file-specific locking and updates
- Add `-p` flag for non-interactive claude CLI invocation
- Resolve file conflicts in plan file parsing
- Fix non-deterministic task ordering in tests
- Remove false warnings for agent discovery in test directories

### Technical Details

**Architecture**
- Learning data stored in JSON format at `.conductor/learning/{session-id}.json`
- Each session captures: task metadata, execution outcomes, durations, files modified, error messages
- Pattern recognition engine analyzes historical data for actionable insights
- Hooks seamlessly integrate learning into the existing execution pipeline

**Test Coverage**
- Comprehensive test suite with 86.4% coverage (465+ tests passing)
- End-to-end validation suite for learning system
- Cross-run learning integration tests
- Race detection and coverage reporting

### Breaking Changes

None. Version 2.0.0 is fully backward compatible with 1.x plans and configurations.
The learning system is opt-in by default and degrades gracefully if disabled or unavailable.

---

## [1.x] - Previous Releases

For changes in version 1.x, see git history. Version 1.x included:
- Core orchestration engine with dependency resolution
- Multi-agent task execution with Claude CLI integration
- Quality control reviews with retry logic
- YAML and Markdown plan file support
- Resumable execution with state tracking
- Comprehensive logging and error handling
