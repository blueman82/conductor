#!/bin/bash
# Demo: Error Classification Features (v2.11-v2.12)
# Shows error pattern detection during task execution with error categorization
#
# Features Demonstrated:
# - Error pattern detection (regex and Claude-based)
# - Error categories: CODE_LEVEL, ENV_LEVEL, PLAN_LEVEL
# - Confidence scores for Claude-based classification
# - Automatic suggestions for fixing errors
# - Retry logic based on error category
# - "Human intervention required" messages
# - Detection method display (regex vs Claude)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

timestamp() {
    date +"%H:%M:%S"
}

# Function to display error classification box
show_error_classification() {
    local category=$1
    local confidence=$2
    local method=$3
    local suggestion=$4
    local agent_can_fix=$5
    local human_intervention=$6

    # Color based on category
    local cat_color=$CYAN
    case "$category" in
        CODE_LEVEL)
            cat_color=$CYAN
            ;;
        ENV_LEVEL)
            cat_color=$YELLOW
            ;;
        PLAN_LEVEL)
            cat_color=$RED
            ;;
    esac

    echo -e "${BOLD}┌────────────────────────────────────────────────────────┐${NC}"
    echo -e "${BOLD}│${NC} ${CYAN}🔍 Error Pattern Detected${NC}                              ${BOLD}│${NC}"
    echo -e "${BOLD}├────────────────────────────────────────────────────────┤${NC}"
    echo -e "${BOLD}│${NC} Category:  ${cat_color}${BOLD}${category}${NC}                              ${BOLD}│${NC}"
    echo -e "${BOLD}│${NC} Detection: ${method}${NC}                         ${BOLD}│${NC}"
    if [ "$method" = "claude" ]; then
        echo -e "${BOLD}│${NC} Confidence: ${MAGENTA}${confidence}%${NC}                        ${BOLD}│${NC}"
    fi
    echo -e "${BOLD}│${NC}                                                        ${BOLD}│${NC}"
    echo -e "${BOLD}│${NC} Suggestion:                                    ${BOLD}│${NC}"
    echo -e "${BOLD}│${NC} ${suggestion}${NC}    ${BOLD}│${NC}"
    echo -e "${BOLD}│${NC}                                                        ${BOLD}│${NC}"
    echo -e "${BOLD}│${NC} Agent Can Fix: ${CYAN}${agent_can_fix}${NC}                    ${BOLD}│${NC}"
    if [ "$human_intervention" = "true" ]; then
        echo -e "${BOLD}│${NC} ${RED}Human Intervention Required${NC}                       ${BOLD}│${NC}"
    fi
    echo -e "${BOLD}└────────────────────────────────────────────────────────┘${NC}"
}

echo -e "${BOLD}$ conductor run plan.md --verbose${NC}"
echo ""
sleep 0.5

echo "Loading plan from plan.md..."
sleep 0.3
echo "Validating dependencies..."
sleep 0.3

echo ""
echo "Plan Summary:"
echo "  Total tasks: 4"
echo "  Execution waves: 3"
echo "  Timeout: 10h0m0s"
echo "  Max concurrency: 3"
echo "  Error Classification: ${CYAN}enabled${NC}"
echo ""
sleep 0.5

echo "Starting execution..."
echo ""

# ============================================================================
# WAVE 1: CODE_LEVEL Error Example
# ============================================================================

TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 1${NC}: 1 task"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 1 (Build Application) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] Running test: go test ./cmd/conductor"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${RED}[ERROR]${NC} Test output:"
echo "    --- FAIL: TestBuildCommand (0.45s)"
echo "        undefined: someFunction"
echo "        ./main.go:42: undefined variable"
echo ""
sleep 0.5

show_error_classification "CODE_LEVEL" "94" "regex" "Missing function definition. Agent should add the function to main.go" "true" "false"

echo ""
TS=$(timestamp)
echo -e "[${TS}] Classification Analysis:"
echo -e "  Pattern matched: '${CYAN}undefined.*variable${NC}' (regex pattern)"
echo -e "  Category: ${CYAN}CODE_LEVEL${NC} - Agent-fixable code issue"
echo -e "  Retry Strategy: ${GREEN}AGENT CAN RETRY${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 1...${NC} (attempt 2/3)"
sleep 1.2

TS=$(timestamp)
echo -e "[${TS}] Running test: go test ./cmd/conductor"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 1 (Build Application): ${GREEN}GREEN${NC} (3.2s, agent: golang-pro, 2 files)"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 1${NC} ${GREEN}complete${NC} (4.7s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# ============================================================================
# WAVE 2: ENV_LEVEL Error Example
# ============================================================================

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 2${NC}: 1 task"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/1]${NC} Task 2 (Setup Database) ${MAGENTA}(agent: database-optimizer)${NC}"
sleep 1.8

TS=$(timestamp)
echo -e "[${TS}] Running test: psql --version"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${RED}[ERROR]${NC} Test output:"
echo "    /bin/sh: psql: command not found"
echo ""
sleep 0.5

show_error_classification "ENV_LEVEL" "96" "claude" "PostgreSQL not installed. Install psql via 'brew install postgresql' or apt-get" "false" "true"

echo ""
TS=$(timestamp)
echo -e "[${TS}] Classification Analysis:"
echo -e "  Pattern analyzed by: ${MAGENTA}Claude AI${NC} (semantic analysis)"
echo -e "  Confidence: ${MAGENTA}96%${NC} - High confidence classification"
echo -e "  Category: ${YELLOW}ENV_LEVEL${NC} - Environmental configuration issue"
echo -e "  Retry Strategy: ${RED}AGENT CANNOT FIX${NC}"
echo -e "  Intervention: ${RED}HUMAN ACTION REQUIRED${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${RED}[BLOCKED]${NC} Task 2 cannot proceed - environment setup needed"
echo -e "[${TS}] Operator must run: ${YELLOW}brew install postgresql${NC}"
echo -e "[${TS}] After setup, use: ${CYAN}conductor run plan.md --skip-completed${NC}"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] {{PAUSE FOR OPERATOR}} Waiting for environment setup..."
sleep 1

# Simulate operator fixing the environment
echo ""
TS=$(timestamp)
echo -e "[${TS}] ${GREEN}[OPERATOR]${NC} Environment setup complete"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] {{RESUME}} Retrying Task 2..."
sleep 1.2

TS=$(timestamp)
echo -e "[${TS}] Running test: psql --version"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} psql (PostgreSQL) 15.2"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 2 (Setup Database): ${GREEN}GREEN${NC} (4.5s, agent: database-optimizer, 3 files)"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 2${NC} ${GREEN}complete${NC} (5.7s) - 1/1 completed (${GREEN}1 GREEN${NC})"
echo ""

# ============================================================================
# WAVE 3: PLAN_LEVEL Error Example
# ============================================================================

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] Starting ${BOLD}Wave 3${NC}: 2 tasks"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[1/2]${NC} Task 3 (Deploy to Staging) ${MAGENTA}(agent: deployment-engineer)${NC}"
sleep 2

TS=$(timestamp)
echo -e "[${TS}] Running test: npm run deploy:staging"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${RED}[ERROR]${NC} Test output:"
echo "    Error: Cannot find bundle 'staging-prod' in deployment plan"
echo "    Available bundles: [development, testing, staging]"
echo ""
sleep 0.5

show_error_classification "PLAN_LEVEL" "91" "claude" "Plan references 'staging-prod' bundle which doesn't exist. Update plan.md to use 'staging' or add 'staging-prod' bundle definition" "false" "true"

echo ""
TS=$(timestamp)
echo -e "[${TS}] Classification Analysis:"
echo -e "  Pattern analyzed by: ${MAGENTA}Claude AI${NC} (semantic analysis)"
echo -e "  Confidence: ${MAGENTA}91%${NC} - High confidence classification"
echo -e "  Category: ${RED}PLAN_LEVEL${NC} - Requires plan file update"
echo -e "  Retry Strategy: ${RED}AGENT CANNOT FIX${NC}"
echo -e "  Intervention: ${RED}HUMAN ACTION REQUIRED${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${RED}[BLOCKED]${NC} Task 3 requires plan update"
echo -e "[${TS}] Action: Edit plan.md, Task 3"
echo -e "[${TS}]   Change: ${RED}staging-prod${NC} → ${GREEN}staging${NC}"
echo -e "[${TS}]   Or: Add 'staging-prod' bundle to deployment configuration"
echo -e "[${TS}] After update, use: ${CYAN}conductor run plan.md --skip-completed${NC}"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] {{PAUSE FOR OPERATOR}} Waiting for plan update..."
sleep 1

# Simulate operator fixing the plan
echo ""
TS=$(timestamp)
echo -e "[${TS}] ${GREEN}[OPERATOR]${NC} Plan updated: changed 'staging-prod' to 'staging'"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] {{RESUME}} Retrying Task 3..."
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] Running test: npm run deploy:staging"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Deployment to staging environment successful"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 3 (Deploy to Staging): ${GREEN}GREEN${NC} (5.1s, agent: deployment-engineer, 4 files)"
sleep 0.3

# Task 4 with mixed detection (regex found, Claude confirms)
TS=$(timestamp)
echo -e "[${TS}] ⏳ IN PROGRESS ${CYAN}[2/2]${NC} Task 4 (Verify Integration) ${MAGENTA}(agent: golang-pro)${NC}"
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] Running test: go test ./integration/..."
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${RED}[ERROR]${NC} Test output:"
echo "    panic: runtime error: invalid memory address or nil pointer dereference"
echo "    [signal SIGSEGV: segmentation fault code=0x1 addr=0x0 pc=0x456b8c]"
echo ""
sleep 0.5

show_error_classification "CODE_LEVEL" "88" "regex" "Nil pointer dereference in integration tests. Check null safety in database connection pooling or add defensive checks before dereferencing" "true" "false"

echo ""
TS=$(timestamp)
echo -e "[${TS}] Classification Analysis:"
echo -e "  Pattern matched: ${CYAN}regex pattern${NC} for nil pointer dereference"
echo -e "  Detection method: Fast regex (no Claude lookup needed)"
echo -e "  Category: ${CYAN}CODE_LEVEL${NC} - Agent-fixable code issue"
echo -e "  Retry Strategy: ${GREEN}AGENT CAN RETRY${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${YELLOW}Retrying Task 4...${NC} (attempt 2/3)"
sleep 1.3

TS=$(timestamp)
echo -e "[${TS}] Running test: go test ./integration/..."
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} All integration tests passed"
sleep 0.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}✓${NC} Task 4 (Verify Integration): ${GREEN}GREEN${NC} (2.8s, agent: golang-pro, 3 files)"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${BOLD}Wave 3${NC} ${GREEN}complete${NC} (9.4s) - 2/2 completed (${GREEN}2 GREEN${NC})"
echo ""

# ============================================================================
# Summary
# ============================================================================

sleep 0.5
TS=$(timestamp)
echo -e "[${TS}] ${BOLD}=== Execution Summary ===${NC}"
echo -e "[${TS}] Total tasks: 4"
echo -e "[${TS}] ${GREEN}Completed: 4${NC}"
echo -e "[${TS}] Failed: 0"
echo -e "[${TS}] Duration: 29.8s"
echo ""

echo -e "[${TS}] ${BOLD}Error Classification Summary:${NC}"
echo -e "[${TS}]   ${CYAN}CODE_LEVEL${NC}: 2 errors (agent fixed via retry)"
echo -e "[${TS}]   ${YELLOW}ENV_LEVEL${NC}: 1 error (human intervention required)"
echo -e "[${TS}]   ${RED}PLAN_LEVEL${NC}: 1 error (plan update required)"
echo ""

echo -e "[${TS}] ${BOLD}Detection Methods:${NC}"
echo -e "[${TS}]   ${GREEN}Regex${NC}: 1 error (fast pattern matching)"
echo -e "[${TS}]   ${MAGENTA}Claude${NC}: 2 errors (high confidence 91-96%)"
echo ""

echo -e "[${TS}] ${BOLD}Classification Impact:${NC}"
echo -e "[${TS}]   Retries triggered: 3 (CODE_LEVEL errors only)"
echo -e "[${TS}]   Human interventions: 2 (ENV_LEVEL + PLAN_LEVEL)"
echo -e "[${TS}]   Total error patterns detected: 4"
echo ""

echo -e "[${TS}] ${BOLD}Agent Usage:${NC}"
echo -e "[${TS}]   ${CYAN}golang-pro${NC}: 2 tasks"
echo -e "[${TS}]   ${CYAN}database-optimizer${NC}: 1 tasks"
echo -e "[${TS}]   ${CYAN}deployment-engineer${NC}: 1 tasks"
echo -e "[${TS}] Files Modified: ${GREEN}12${NC} files"
echo -e "[${TS}] Average Duration: 7.5s/task"
echo ""

echo -e "[${TS}] ${BOLD}Error Classification Config:${NC}"
echo -e "[${TS}]   Enabled: ${GREEN}true${NC}"
echo -e "[${TS}]   Claude classification: ${GREEN}enabled${NC}"
echo -e "[${TS}]   Regex fallback: ${GREEN}enabled${NC}"
echo -e "[${TS}]   Cache TTL: 24h"
echo -e "[${TS}]   Confidence threshold: 0.85"
echo ""

echo "Execution completed successfully!"
echo "Error classifications logged to: .conductor/logs/error_classifications.json"
echo ""
echo -e "${DIM}For more details, run:${NC}"
echo -e "  ${CYAN}conductor learning export${NC} (to see execution history)"
echo -e "  ${CYAN}cat .conductor/logs/*.log${NC} (to review detailed logs)"
