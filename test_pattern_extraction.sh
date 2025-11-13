#!/bin/bash
# Comprehensive Test Suite for Pattern Extraction Keyword Expansion
# This script executes all tests and generates a detailed report

set -e

RESULTS_FILE="/tmp/conductor_test_results.txt"
> "$RESULTS_FILE"

echo "┌──────────────────────────────────────────────────────────────┐"
echo "│ Pattern Extraction Keyword Expansion - Test Execution       │"
echo "└──────────────────────────────────────────────────────────────┘"
echo ""

# Step 1: Pattern Extraction Tests
echo "Step 1: Pattern Extraction Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Testing task.go pattern extraction..."
go test ./internal/executor -run TestExtractFailurePatterns -v 2>&1 | tee -a "$RESULTS_FILE"
TASK_PATTERN_TESTS=$?

echo ""
echo "Testing analysis.go pattern extraction..."
go test ./internal/learning -run TestAnalyzeFailures -v 2>&1 | tee -a "$RESULTS_FILE"
ANALYSIS_PATTERN_TESTS=$?

# Step 2: Metrics Tests
echo ""
echo "Step 2: Metrics Collection Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test ./internal/learning -run TestNewPatternMetrics -v 2>&1 | tee -a "$RESULTS_FILE"
go test ./internal/learning -run TestRecordPatternDetection -v 2>&1 | tee -a "$RESULTS_FILE"
go test ./internal/learning -run TestGetDetectionRate -v 2>&1 | tee -a "$RESULTS_FILE"
go test ./internal/learning -run TestThreadSafety -v 2>&1 | tee -a "$RESULTS_FILE"
METRICS_TESTS=$?

# Step 3: Full Executor Tests
echo ""
echo "Step 3: Full Executor Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test ./internal/executor -v 2>&1 | tee -a "$RESULTS_FILE"
EXECUTOR_TESTS=$?

# Step 4: Full Learning Tests
echo ""
echo "Step 4: Full Learning Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test ./internal/learning -v 2>&1 | tee -a "$RESULTS_FILE"
LEARNING_TESTS=$?

# Step 5: Race Detection
echo ""
echo "Step 5: Race Detection"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Checking executor for race conditions..."
go test -race ./internal/executor 2>&1 | tee -a "$RESULTS_FILE"
EXECUTOR_RACE=$?

echo ""
echo "Checking learning for race conditions..."
go test -race ./internal/learning 2>&1 | tee -a "$RESULTS_FILE"
LEARNING_RACE=$?

# Step 6: Code Coverage
echo ""
echo "Step 6: Code Coverage Analysis"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Executor coverage:"
go test ./internal/executor -cover 2>&1 | tee -a "$RESULTS_FILE"

echo ""
echo "Learning coverage:"
go test ./internal/learning -cover 2>&1 | tee -a "$RESULTS_FILE"

echo ""
echo "Full coverage report:"
go test ./... -coverprofile=/tmp/coverage.out 2>&1 | tee -a "$RESULTS_FILE"
COVERAGE_EXIT=$?

# Step 7: Format & Lint
echo ""
echo "Step 7: Code Formatting & Linting"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Formatting executor..."
go fmt ./internal/executor/...

echo "Formatting learning..."
go fmt ./internal/learning/...

echo ""
echo "Running vet on executor..."
go vet ./internal/executor/... 2>&1 | tee -a "$RESULTS_FILE"
VET_EXECUTOR=$?

echo ""
echo "Running vet on learning..."
go vet ./internal/learning/... 2>&1 | tee -a "$RESULTS_FILE"
VET_LEARNING=$?

# Step 8: Build Binary
echo ""
echo "Step 8: Build Binary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go build ./cmd/conductor 2>&1 | tee -a "$RESULTS_FILE"
BUILD_EXIT=$?

if [ $BUILD_EXIT -eq 0 ]; then
    echo ""
    echo "Testing binary..."
    ./conductor --version
    echo ""
    ./conductor --help | head -10
fi

# Step 9: Full Test Suite
echo ""
echo "Step 9: Full Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test ./... 2>&1 | tee -a "$RESULTS_FILE"
FULL_TEST_EXIT=$?

# Generate Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Execution Complete - Results Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Count test results
TOTAL_TESTS=$(grep -c "^=== RUN" "$RESULTS_FILE" || echo "0")
PASS_TESTS=$(grep -c "^--- PASS" "$RESULTS_FILE" || echo "0")
FAIL_TESTS=$(grep -c "^--- FAIL" "$RESULTS_FILE" || echo "0")

echo "Total Test Functions: $TOTAL_TESTS"
echo "Passed: $PASS_TESTS"
echo "Failed: $FAIL_TESTS"
echo ""

# Exit codes summary
echo "Exit Codes Summary:"
echo "  Pattern Extraction (task.go):    $TASK_PATTERN_TESTS"
echo "  Pattern Extraction (analysis):   $ANALYSIS_PATTERN_TESTS"
echo "  Metrics Tests:                   $METRICS_TESTS"
echo "  Executor Tests:                  $EXECUTOR_TESTS"
echo "  Learning Tests:                  $LEARNING_TESTS"
echo "  Race Detection (executor):       $EXECUTOR_RACE"
echo "  Race Detection (learning):       $LEARNING_RACE"
echo "  Vet (executor):                  $VET_EXECUTOR"
echo "  Vet (learning):                  $VET_LEARNING"
echo "  Build:                           $BUILD_EXIT"
echo "  Full Test Suite:                 $FULL_TEST_EXIT"
echo ""

# Overall status
if [ $FULL_TEST_EXIT -eq 0 ] && [ $BUILD_EXIT -eq 0 ] && [ $VET_EXECUTOR -eq 0 ] && [ $VET_LEARNING -eq 0 ]; then
    echo "✓ ALL SYSTEMS GO - Ready for Deployment"
    exit 0
else
    echo "⚠ Some tests failed - Review required"
    exit 1
fi
