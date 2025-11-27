#!/bin/bash
# Demo: conductor observe live
# Simulates live real-time event streaming

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe live --project conductor${NC}"
echo ""
sleep 0.5

echo -e "${CYAN}Streaming live events from conductor...${NC}"
echo "Press Ctrl+C to stop"
echo ""
echo "────────────────────────────────────────────────────────────"
sleep 1

timestamp() {
    date +"%H:%M:%S"
}

TS=$(timestamp)
echo -e "[${TS}] ${GREEN}SESSION_START${NC} session-abc123 (agent: golang-pro)"
sleep 0.8

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Read: internal/executor/task.go ${GREEN}OK${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Grep: pattern='func.*Execute' ${GREEN}OK${NC} (3 matches)"
sleep 0.6

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Read: internal/models/task.go ${GREEN}OK${NC}"
sleep 0.4

TS=$(timestamp)
echo -e "[${TS}] ${MAGENTA}BASH${NC} go test ./internal/executor/... ${GREEN}exit 0${NC}"
sleep 1.2

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Edit: internal/executor/task.go ${GREEN}OK${NC}"
sleep 0.7

TS=$(timestamp)
echo -e "[${TS}] ${MAGENTA}BASH${NC} go build ./cmd/conductor ${GREEN}exit 0${NC}"
sleep 0.9

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Edit: internal/executor/task_test.go ${GREEN}OK${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${MAGENTA}BASH${NC} go test -v ./internal/executor/... ${GREEN}exit 0${NC}"
sleep 1.5

TS=$(timestamp)
echo -e "[${TS}] ${GREEN}SESSION_END${NC} session-abc123 (duration: 4m32s, ${GREEN}success${NC})"
echo ""
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${GREEN}SESSION_START${NC} session-def456 (agent: code-reviewer)"
sleep 0.6

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Glob: **/*_test.go ${GREEN}OK${NC} (45 files)"
sleep 0.4

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Read: internal/executor/task_test.go ${GREEN}OK${NC}"
sleep 0.5

TS=$(timestamp)
echo -e "[${TS}] ${MAGENTA}BASH${NC} go test -cover ./... ${RED}exit 1${NC}"
sleep 0.8

TS=$(timestamp)
echo -e "[${TS}] ${RED}ERROR${NC} Test failure: TestExecuteTask_Timeout"
sleep 0.3

TS=$(timestamp)
echo -e "[${TS}] ${CYAN}TOOL${NC} Read: internal/executor/task_test.go:156 ${GREEN}OK${NC}"
sleep 0.5

echo ""
echo -e "${YELLOW}^C${NC}"
echo ""
echo "Live streaming stopped."
echo "Events captured: 14"
