#!/bin/bash
# Demo: conductor observe transcript
# Simulates formatted transcript of a session

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BLUE='\033[0;34m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe transcript abc123 --project conductor${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}════════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}Session Transcript: abc123${NC}"
echo -e "${BOLD}════════════════════════════════════════════════════════════════${NC}"
echo ""
echo "Project:    conductor"
echo "Agent:      golang-pro"
echo "Started:    2024-01-15 14:23:45"
echo "Duration:   4m32s"
echo "Status:     ${GREEN}SUCCESS${NC}"
echo ""
echo -e "${DIM}────────────────────────────────────────────────────────────────${NC}"
echo ""
sleep 0.3

echo -e "${BLUE}[14:23:45]${NC} ${BOLD}User Request:${NC}"
echo "  Implement the Execute method for task execution with timeout support."
echo "  The method should handle context cancellation gracefully."
echo ""
sleep 0.3

echo -e "${BLUE}[14:23:46]${NC} ${CYAN}Tool: Read${NC}"
echo "  ${DIM}Path: internal/executor/task.go${NC}"
echo "  ${GREEN}✓${NC} Read 245 lines"
echo ""
sleep 0.3

echo -e "${BLUE}[14:23:48]${NC} ${CYAN}Tool: Grep${NC}"
echo "  ${DIM}Pattern: 'func.*Execute'${NC}"
echo "  ${GREEN}✓${NC} Found 3 matches in 2 files"
echo ""
sleep 0.3

echo -e "${BLUE}[14:24:02]${NC} ${CYAN}Tool: Read${NC}"
echo "  ${DIM}Path: internal/models/task.go${NC}"
echo "  ${GREEN}✓${NC} Read 89 lines"
echo ""
sleep 0.3

echo -e "${BLUE}[14:24:15]${NC} ${CYAN}Tool: Edit${NC}"
echo "  ${DIM}Path: internal/executor/task.go${NC}"
echo "  ${DIM}Lines: 45-78${NC}"
echo "  ${GREEN}✓${NC} Applied edit (added Execute method with timeout)"
echo ""
sleep 0.3

echo -e "${BLUE}[14:25:30]${NC} ${MAGENTA}Bash Command:${NC}"
echo "  ${DIM}\$ go test ./internal/executor/...${NC}"
echo "  ${GREEN}✓${NC} Exit code: 0"
echo "  ${DIM}Output: ok  github.com/harrison/conductor/internal/executor 0.234s${NC}"
echo ""
sleep 0.3

echo -e "${BLUE}[14:26:45]${NC} ${MAGENTA}Bash Command:${NC}"
echo "  ${DIM}\$ go build ./...${NC}"
echo "  ${RED}✗${NC} Exit code: 1"
echo "  ${RED}Error: internal/executor/task.go:67: undefined: context.Canceled${NC}"
echo ""
sleep 0.3

echo -e "${BLUE}[14:27:10]${NC} ${CYAN}Tool: Edit${NC}"
echo "  ${DIM}Path: internal/executor/task.go${NC}"
echo "  ${DIM}Lines: 5-10${NC}"
echo "  ${GREEN}✓${NC} Applied edit (added missing import)"
echo ""
sleep 0.3

echo -e "${BLUE}[14:27:55]${NC} ${MAGENTA}Bash Command:${NC}"
echo "  ${DIM}\$ go build ./...${NC}"
echo "  ${GREEN}✓${NC} Exit code: 0"
echo ""
sleep 0.3

echo -e "${BLUE}[14:28:10]${NC} ${MAGENTA}Bash Command:${NC}"
echo "  ${DIM}\$ go test -v ./internal/executor/...${NC}"
echo "  ${GREEN}✓${NC} Exit code: 0"
echo "  ${DIM}Output: === RUN   TestExecute${NC}"
echo "  ${DIM}        --- PASS: TestExecute (0.02s)${NC}"
echo "  ${DIM}        PASS${NC}"
echo ""
sleep 0.3

echo -e "${BLUE}[14:28:17]${NC} ${BOLD}Assistant Response:${NC}"
echo "  I've implemented the Execute method with proper timeout support."
echo "  The implementation:"
echo "  - Uses context.WithTimeout for deadline enforcement"
echo "  - Handles context.Canceled gracefully"
echo "  - Returns appropriate errors for timeout scenarios"
echo "  All tests pass."
echo ""
echo -e "${DIM}────────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}Summary:${NC}"
echo "  Tools used:     4 (Read: 2, Edit: 2)"
echo "  Bash commands:  4 (3 success, 1 failure)"
echo "  Files modified: 1"
echo "  Total tokens:   16,348"
echo "  Est. cost:      \$0.24"
