#!/usr/bin/env bash
#
# ci-local.sh - Local CI validation for Conductor
#
# Mirrors GitHub Actions CI pipeline locally with parallel test execution.
# Run before pushing to catch issues early.
#
# Usage: ./check [options]
#   -h, --help        Show this help message
#   -v, --verbose     Verbose test output
#   --fix             Auto-fix formatting issues (gofmt -w)
#   --quick           Skip tests, only run fmt/vet checks
#   -p, --parallel N  Set parallel jobs (default: 4)
#

set -euo pipefail

# Resolve symlinks to get the real script location
SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -L "$SCRIPT_PATH" ]; do
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    [[ $SCRIPT_PATH != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
PARALLEL_JOBS=${CI_PARALLEL:-4}
TIMEOUT=${CI_TIMEOUT:-5m}
VERBOSE=false
FIX=false
QUICK=false

log_step() { echo -e "${BLUE}==>${NC} $1"; }
log_success() { echo -e "${GREEN}✓${NC} $1"; }
log_error() { echo -e "${RED}✗${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $1"; }

show_help() {
    cat << EOF
${BOLD}Conductor Local CI Pipeline${NC}

Mirrors GitHub Actions CI locally with parallel test execution.

${BOLD}Usage:${NC}
  ./check [options]

${BOLD}Options:${NC}
  -h, --help          Show this help message
  -v, --verbose       Verbose test output
  --fix               Auto-fix formatting issues (gofmt -w)
  --quick             Skip tests, only run fmt/vet checks
  -p, --parallel N    Set parallel jobs (default: 4)

${BOLD}Environment Variables:${NC}
  CI_PARALLEL=N       Set parallel jobs (default: 4)
  CI_TIMEOUT=DURATION Set test timeout (default: 5m)

${BOLD}Examples:${NC}
  ./check                    # Full CI pipeline
  ./check --fix              # Fix formatting, then run CI
  ./check --quick            # Fast check (no tests)
  ./check -p 8               # Run with 8 parallel jobs
  ./check -v                 # Verbose test output

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --fix)
            FIX=true
            shift
            ;;
        --quick)
            QUICK=true
            shift
            ;;
        -p|--parallel)
            PARALLEL_JOBS="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Track timing
SECONDS=0

cd "$PROJECT_ROOT"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Conductor Local CI Pipeline${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Step 1: Check formatting (fast, do first)
log_step "Checking formatting..."
UNFORMATTED=$(gofmt -l . 2>&1 || true)
if [ -n "$UNFORMATTED" ]; then
    if [ "$FIX" = true ]; then
        log_warn "Fixing formatting..."
        gofmt -w .
        log_success "Formatting fixed"
    else
        log_error "Files need formatting:"
        echo "$UNFORMATTED"
        echo ""
        echo "Run: ./check --fix"
        exit 1
    fi
else
    log_success "Formatting OK"
fi

# Step 2: go vet (fast static analysis)
log_step "Running go vet..."
if ! go vet ./... 2>&1; then
    log_error "go vet failed"
    exit 1
fi
log_success "go vet passed"

# Step 3: Verify dependencies
log_step "Verifying dependencies..."
if ! go mod verify 2>&1; then
    log_error "Dependency verification failed"
    exit 1
fi
log_success "Dependencies verified"

# Step 4: Run tests with race detection and parallelism
if [ "$QUICK" = true ]; then
    log_warn "Skipping tests (--quick mode)"
else
    log_step "Running tests (parallel=$PARALLEL_JOBS, race=on, timeout=$TIMEOUT)..."

    # Suppress the ld warning by redirecting stderr for known harmless warnings
    # The LC_DYSYMTAB warning is a macOS linker issue that doesn't affect functionality
    export GOFLAGS="-p=$PARALLEL_JOBS"

    TEST_ARGS=("-race" "-short" "-timeout=$TIMEOUT" "-parallel=$PARALLEL_JOBS")
    if [ "$VERBOSE" = true ]; then
        TEST_ARGS+=("-v")
    fi

    if ! go test ./... "${TEST_ARGS[@]}" 2>&1 | grep -v "ld: warning.*LC_DYSYMTAB"; then
        log_error "Tests failed"
        exit 1
    fi
    log_success "All tests passed"
fi

# Step 5: Build verification (optional, fast)
log_step "Verifying build..."
if ! go build -o /dev/null ./cmd/conductor 2>&1; then
    log_error "Build failed"
    exit 1
fi
log_success "Build OK"

# Summary
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  All CI checks passed in ${SECONDS}s${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
