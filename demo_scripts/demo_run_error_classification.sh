#!/bin/bash
# Demo: conductor run error-classification-plan.yaml --verbose
# Shows v2.11-v2.12 error pattern detection with CODE_LEVEL, ENV_LEVEL, PLAN_LEVEL categories
#
# Demonstrates:
# - Regex-based error detection
# - Claude AI-based classification
# - Confidence scores and categorization

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

timestamp() {
    date +"%H:%M:%S"
}

echo -e "${BOLD}$ conductor run error-classification-plan.yaml --verbose${NC}"
echo ""

echo "Plan Summary:"
echo "  Total tasks: 3"
echo "  Error detection: ENABLED"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# Task 1: Compile Error (CODE_LEVEL)
TS=$(timestamp)
echo -e "[${TS}] Starting Wave 1: 3 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [1/3] Task 1 (Compile Code) (agent: python-pro)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] ✗ Task 1 (Compile Code): RED (agent: python-pro)"
sleep 0.3

echo ""
echo -e "${YELLOW}Error Classification:${NC}"
echo -e "  Category: ${RED}CODE_LEVEL${NC} (Agent can fix with retry)"
echo -e "  Pattern: SyntaxError - invalid syntax"
echo -e "  Detection: Regex (100% confidence)"
echo -e "  Suggestion: Review Python syntax and fix compilation errors"
sleep 0.5

# Task 2: Missing Dependency (ENV_LEVEL)
echo ""
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [2/3] Task 2 (Install Package) (agent: python-pro)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] ✗ Task 2 (Install Package): RED (agent: python-pro)"
sleep 0.3

echo ""
echo -e "${YELLOW}Error Classification:${NC}"
echo -e "  Category: ${YELLOW}ENV_LEVEL${NC} (Environment dependency missing - manual fix needed)"
echo -e "  Pattern: ModuleNotFoundError - numpy not installed"
echo -e "  Detection: Claude AI (89% confidence)"
echo -e "  Suggestion: Install numpy via pip: pip install numpy"
sleep 0.5

# Task 3: Configuration Error (PLAN_LEVEL)
echo ""
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS [3/3] Task 3 (Run Tests) (agent: python-pro)"
sleep 1.0

TS=$(timestamp)
echo -e "[${TS}] ✗ Task 3 (Run Tests): RED (agent: python-pro)"
sleep 0.3

echo ""
echo -e "${YELLOW}Error Classification:${NC}"
echo -e "  Category: ${RED}PLAN_LEVEL${NC} (Plan misconfiguration - manual fix needed)"
echo -e "  Pattern: ConfigurationError - missing test configuration file"
echo -e "  Detection: Claude AI (92% confidence)"
echo -e "  Suggestion: Review plan configuration and provide test_config.yaml"
sleep 0.5

echo ""
TS=$(timestamp)
echo -e "[${TS}] === Execution Summary ==="
echo "Total tasks: 3"
echo "Completed: 3"
echo "Failed: 3"
echo ""
echo "Error Patterns:"
echo "  CODE_LEVEL (agent fixable): 1"
echo "  ENV_LEVEL (manual fix): 1"
echo "  PLAN_LEVEL (manual fix): 1"
echo ""
echo "Learning context stored for adaptive retries"
