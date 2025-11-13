#!/bin/bash
# Comprehensive test runner for Pattern Extraction Keyword Expansion

cd /Users/harrison/Github/conductor

echo "=========================================="
echo "Pattern Extraction Test Suite"
echo "=========================================="
echo ""

# Run all executor tests
echo "Running executor tests..."
go test ./internal/executor -v -count=1

echo ""
echo "=========================================="
echo "Running learning tests..."
go test ./internal/learning -v -count=1

echo ""
echo "=========================================="
echo "Running race detection on executor..."
go test -race ./internal/executor -count=1

echo ""
echo "Running race detection on learning..."
go test -race ./internal/learning -count=1

echo ""
echo "=========================================="
echo "Coverage report..."
go test ./internal/executor -cover
go test ./internal/learning -cover

echo ""
echo "=========================================="
echo "Building binary..."
go build ./cmd/conductor

if [ $? -eq 0 ]; then
    echo "Build successful!"
    ./conductor --version
else
    echo "Build failed!"
    exit 1
fi

echo ""
echo "=========================================="
echo "Running full test suite..."
go test ./... -count=1

echo ""
echo "=========================================="
echo "Test execution complete!"
