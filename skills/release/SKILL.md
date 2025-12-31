---
name: release
description: |
  Create software releases with version bumping, changelog updates, git tags, and GitHub releases.
  Use when: "create a release", "bump version", "tag release", "publish version", "/release".
  Handles VERSION files, package.json, Cargo.toml, pyproject.toml, and go.mod projects.
triggers:
  - "release"
  - "create release"
  - "bump version"
  - "tag version"
  - "publish release"
  - "cut a release"
  - "new version"
---

# Release Skill

Create consistent, safe software releases across any project type.

## Invocation

```bash
/release                     # Interactive - detect and prompt
/release patch               # Bump patch version (0.0.X)
/release minor               # Bump minor version (0.X.0)
/release major               # Major version bump (X.0.0)
/release v2.5.0              # Set explicit version
/release --dry-run           # Preview without executing
```

## Workflow Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     RELEASE WORKFLOW                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Phase 1: Pre-flight Checks                                     │
│    ├─ Clean working tree?                                       │
│    ├─ On releasable branch? (main/master/release/*)             │
│    ├─ Tests passing?                                            │
│    └─ Detect project type & current version                     │
│                                                                 │
│  Phase 2: Version Bump                                          │
│    ├─ Calculate new version                                     │
│    ├─ Update version file(s)                                    │
│    └─ Update version references in docs                         │
│                                                                 │
│  Phase 3: Changelog & Docs                                      │
│    ├─ Generate changelog entry (if CHANGELOG.md exists)         │
│    ├─ Update README.md badges/versions (if applicable)          │
│    └─ Update CLAUDE.md version (if applicable)                  │
│                                                                 │
│  Phase 4: Build & Verify                                        │
│    ├─ Run build command (make build, npm build, etc.)           │
│    ├─ Run linters/formatters (gofmt, eslint, etc.)              │
│    └─ Verify build succeeded                                    │
│                                                                 │
│  Phase 5: Git Operations                                        │
│    ├─ Stage all changes                                         │
│    ├─ Commit: "chore(release): vX.Y.Z"                          │
│    ├─ Create annotated tag: vX.Y.Z                              │
│    └─ Push commits and tags                                     │
│                                                                 │
│  Phase 6: GitHub Release (optional)                             │
│    ├─ Generate release notes from commits                       │
│    └─ Create GitHub release via gh CLI                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Pre-flight Checks

### 1.1 Working Tree Status

```bash
git status --porcelain
```

**STOP if uncommitted changes exist.** Ask user to commit or stash first.

Exception: If `--dry-run` flag, continue but warn.

### 1.2 Branch Check

```bash
git branch --show-current
```

**Allowed branches:**
- `main`, `master` - production releases
- `release/*` - release branches
- `develop` - pre-releases only (append `-beta.N` or `-rc.N`)

**If on feature branch:**
1. Ask user: "Merge to main first?"
2. If yes: `git checkout main && git merge <feature-branch> --ff-only`
3. Continue release from main

### 1.3 Test Suite

Run tests if test command is detectable:

| Project Type | Test Command |
|--------------|--------------|
| Go | `go test ./...` |
| Node.js | `npm test` or `yarn test` |
| Rust | `cargo test` |
| Python | `pytest` or `python -m pytest` |
| Make-based | `make test` (if target exists) |

**STOP if tests fail.** Do not proceed with broken tests.

### 1.4 Project Detection

Detect project type by scanning for these files (in order):

| File | Type | Version Location | Build Command |
|------|------|------------------|---------------|
| `VERSION` | Generic | File contents | `make build` (if Makefile) |
| `package.json` | Node.js | `.version` field | `npm run build` |
| `Cargo.toml` | Rust | `version = "X.Y.Z"` | `cargo build --release` |
| `pyproject.toml` | Python | `version = "X.Y.Z"` | `python -m build` |
| `go.mod` | Go | Git tags only | `go build ./cmd/...` |
| `build.gradle` | Java/Gradle | `version 'X.Y.Z'` | `./gradlew build` |
| `pom.xml` | Java/Maven | `<version>` tag | `mvn package` |

**Extract current version** from the detected source.

If no version file found, check latest git tag:
```bash
git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
```

---

## Phase 2: Version Bump

### 2.1 Parse Current Version

Extract semantic version components: `MAJOR.MINOR.PATCH[-PRERELEASE]`

Examples:
- `2.31.0` → major=2, minor=31, patch=0
- `1.0.0-beta.1` → major=1, minor=0, patch=0, pre=beta.1

### 2.2 Calculate New Version

| Bump Type | Current | New |
|-----------|---------|-----|
| `patch` | 2.31.0 | 2.31.1 |
| `minor` | 2.31.0 | 2.32.0 |
| `major` | 2.31.0 | 3.0.0 |
| explicit `v2.35.0` | 2.31.0 | 2.35.0 |

**Pre-release handling:**
- If current is `2.0.0-beta.1` and bump is `patch` → `2.0.0-beta.2`
- To exit pre-release: use explicit version or `--graduate`

### 2.3 Update Version Files

Update ALL version locations found:

**VERSION file:**
```bash
echo "X.Y.Z" > VERSION
```

**package.json:**
```bash
npm version X.Y.Z --no-git-tag-version
# Or edit directly with jq/sed
```

**Cargo.toml:**
```toml
version = "X.Y.Z"
```

**pyproject.toml:**
```toml
version = "X.Y.Z"
```

**Go projects:** Version comes from git tag, no file update needed.

---

## Phase 3: Changelog & Documentation

### 3.1 CHANGELOG.md (if exists)

Generate changelog entry from commits since last tag:

```bash
git log $(git describe --tags --abbrev=0)..HEAD --oneline --no-merges
```

**Format (Keep a Changelog style):**

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New feature descriptions (from feat: commits)

### Changed
- Change descriptions (from refactor:, chore: commits)

### Fixed
- Bug fix descriptions (from fix: commits)
```

**Categorize by conventional commit prefix:**
- `feat:` → Added
- `fix:` → Fixed
- `refactor:`, `perf:` → Changed
- `docs:` → Documentation
- `chore:`, `build:`, `ci:` → Maintenance (optional section)

Insert new section after `# Changelog` header, before previous releases.

### 3.2 README.md Updates

Search for version badges or references:

```markdown
![Version](https://img.shields.io/badge/version-2.31.0-blue)
```

Update to new version if found.

### 3.3 CLAUDE.md Updates

If CLAUDE.md exists and contains version reference, update it:

```markdown
**Current**: v2.31.0 - Description here
```

Update to:
```markdown
**Current**: vX.Y.Z - Description here
```

---

## Phase 4: Build & Verify

### 4.1 Detect Build Command

| Condition | Command |
|-----------|---------|
| Makefile with `build` target | `make build` |
| Makefile with `build-patch/minor/major` | Use appropriate target |
| package.json with `build` script | `npm run build` |
| Cargo.toml | `cargo build --release` |
| go.mod | `go build ./...` |
| pyproject.toml | `python -m build` |

### 4.2 Execute Build

Run the detected build command. **STOP if build fails.**

```bash
# Example for Make-based projects
make build
```

### 4.3 Verify Artifacts

Check that expected artifacts exist:
- Binary in expected location
- No error output
- Exit code 0

### 4.4 Run Linters/Formatters (CI Parity)

Run the same checks CI will run to avoid post-push failures:

| Language | Commands |
|----------|----------|
| Go | `gofmt -l . && go vet ./...` |
| Node.js | `npm run lint` (if exists) |
| Rust | `cargo fmt --check && cargo clippy` |
| Python | `ruff check . && black --check .` |

**STOP if linting fails.** Fix before proceeding.

```bash
# Example for Go projects
gofmt -w .  # Auto-fix formatting
go vet ./...
```

---

## Phase 5: Git Operations

### 5.1 Stage Changes

```bash
git add -A
```

### 5.2 Create Release Commit

```bash
git commit -m "chore(release): v${VERSION}

Release version ${VERSION}

Changes in this release:
$(git log $(git describe --tags --abbrev=0 2>/dev/null || echo "")..HEAD~1 --oneline --no-merges | head -20)
"
```

### 5.3 Create Annotated Tag

```bash
git tag -a "v${VERSION}" -m "Release v${VERSION}"
```

**Tag format:** Always prefix with `v` (e.g., `v2.32.0`)

### 5.4 Push to Remote

```bash
git push origin $(git branch --show-current)
git push origin "v${VERSION}"
```

**Ask for confirmation before push** unless `--yes` flag provided.

---

## Phase 6: GitHub Release (Optional)

### 6.1 Check for gh CLI

```bash
gh --version
```

Skip this phase if `gh` not installed or not authenticated.

### 6.2 Generate Release Notes

```bash
gh api repos/{owner}/{repo}/releases/generate-notes \
  -f tag_name="v${VERSION}" \
  -f target_commitish="$(git branch --show-current)" \
  --jq '.body'
```

Or generate from commit log if API unavailable.

### 6.3 Create Release

```bash
gh release create "v${VERSION}" \
  --title "v${VERSION}" \
  --notes "${RELEASE_NOTES}" \
  --latest
```

**For pre-releases:**
```bash
gh release create "v${VERSION}" --prerelease
```

---

## Flags Reference

| Flag | Effect |
|------|--------|
| `--dry-run` | Show what would happen, don't execute |
| `--skip-tests` | Skip test suite execution |
| `--skip-build` | Skip build step |
| `--skip-changelog` | Don't update CHANGELOG.md |
| `--skip-push` | Create commit/tag locally, don't push |
| `--skip-gh` | Skip GitHub release creation |
| `--yes`, `-y` | Skip all confirmations |
| `--prerelease` | Mark as pre-release (beta/rc) |

---

## Safety Mechanisms

1. **Dirty tree protection** - Won't release with uncommitted changes
2. **Branch protection** - Warns on non-release branches
3. **Test gate** - Fails if tests don't pass
4. **Build verification** - Fails if build breaks
5. **Confirmation prompts** - Asks before destructive operations
6. **Dry-run mode** - Preview entire workflow safely

---

## Examples

### Basic Patch Release
```
User: /release patch
Claude:
1. Detected Go project with VERSION file (current: 2.31.0)
2. New version: 2.31.1
3. Running tests... ✓
4. Updating VERSION file... ✓
5. Building... ✓
6. Creating commit and tag... ✓
7. Push to origin? [Y/n]
```

### Minor Release with Changelog
```
User: /release minor
Claude:
1. Current version: 1.5.2 → New: 1.6.0
2. Found 12 commits since v1.5.2
3. Generated CHANGELOG.md entry with 3 features, 2 fixes
4. [Shows preview]
5. Proceed? [Y/n]
```

### Dry Run
```
User: /release major --dry-run
Claude:
[DRY RUN] Would perform:
- Bump version: 2.31.0 → 3.0.0
- Update: VERSION, CLAUDE.md
- Run: make build
- Commit: "chore(release): v3.0.0"
- Tag: v3.0.0
- Push to origin/main
- Create GitHub release
No changes made.
```

---

## Error Recovery

| Error | Recovery |
|-------|----------|
| Tests failed | Fix tests, re-run `/release` |
| Build failed | Fix build, re-run `/release` |
| Push rejected | `git pull --rebase`, then `git push` |
| Tag exists | Delete with `git tag -d vX.Y.Z`, re-run |
| GH release failed | Run `gh release create` manually |

If release partially completed:
```bash
# Undo commit (if not pushed)
git reset --soft HEAD~1

# Delete tag (if created)
git tag -d vX.Y.Z

# Start fresh
/release
```

---

## Project-Specific Notes

### Go Projects
- Version lives in git tags, not files
- If VERSION file exists, update it too
- Build: `go build ./cmd/...` or `make build`

### Node.js Projects
- Use `npm version` for atomic version + tag
- Or update package.json + package-lock.json manually
- Build: `npm run build` if script exists

### Rust Projects
- Update Cargo.toml version field
- Also update Cargo.lock: `cargo update`
- Build: `cargo build --release`

### Python Projects
- Update pyproject.toml or setup.py
- Build: `python -m build`
- Consider PyPI publish: `twine upload`

### Monorepos
- May need to update multiple package.json/Cargo.toml
- Consider workspace version commands
- Tag format: `package-name@X.Y.Z` or `vX.Y.Z`
