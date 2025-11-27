#!/bin/bash
# Demo: conductor observe session
# Simulates session listing and detailed session analysis

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe session${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Recent Sessions ===${NC}"
echo ""
printf "%-8s %-10s %-12s %-20s %-20s %s\n" "ID" "Status" "Duration" "Started" "Agent" "Task"
echo "────────────────────────────────────────────────────────────────────────────────────────────────────"
printf "%-8s ${GREEN}%-10s${NC} %-12s %-20s %-20s %s\n" "1" "SUCCESS" "4m32s" "2024-01-15 14:23:45" "golang-pro" "Implement API handlers"
printf "%-8s ${GREEN}%-10s${NC} %-12s %-20s %-20s %s\n" "2" "SUCCESS" "2m15s" "2024-01-15 14:18:12" "code-reviewer" "Review PR #123"
printf "%-8s ${RED}%-10s${NC} %-12s %-20s %-20s %s\n" "3" "FAILED" "6m48s" "2024-01-15 14:10:00" "database-optimizer" "Optimize query"
printf "%-8s ${GREEN}%-10s${NC} %-12s %-20s %-20s %s\n" "4" "SUCCESS" "3m22s" "2024-01-15 13:55:30" "golang-pro" "Add unit tests"
printf "%-8s ${GREEN}%-10s${NC} %-12s %-20s %-20s %s\n" "5" "SUCCESS" "1m58s" "2024-01-15 13:45:15" "typescript-pro" "Fix type errors"
echo ""
echo "(Showing 5 sessions. Use --limit/-n to see more)"
echo "Use 'conductor observe session <ID>' for detailed analysis."
echo ""

# Detailed session analysis
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor observe session abc123 --project conductor${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Session: abc123 ===${NC}"
echo ""
echo "Project:      conductor"
echo "Agent:        golang-pro"
echo "Status:       ${GREEN}SUCCESS${NC}"
echo "Duration:     4m32s"
echo "Started:      2024-01-15 14:23:45"
echo "Ended:        2024-01-15 14:28:17"
echo ""
echo -e "${BOLD}--- Token Usage ---${NC}"
echo "Input Tokens:   12,456"
echo "Output Tokens:  3,892"
echo "Total Tokens:   16,348"
echo "Est. Cost:      \$0.24"
echo ""
echo -e "${BOLD}--- Tool Timeline ---${NC}"
printf "%-12s %-15s %-8s %s\n" "Time" "Tool" "Status" "Details"
echo "────────────────────────────────────────────────────────────────"
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:23:46" "Read" "OK" "internal/executor/task.go"
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:23:48" "Grep" "OK" "pattern: 'func.*Execute'"
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:24:02" "Read" "OK" "internal/models/task.go"
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:24:15" "Edit" "OK" "internal/executor/task.go"
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:25:30" "Bash" "OK" "go test ./internal/executor/..."
printf "%-12s %-15s ${RED}%-8s${NC} %s\n" "14:26:45" "Bash" "FAIL" "go build ./..."
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:27:10" "Edit" "OK" "internal/executor/task.go (fix)"
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:27:55" "Bash" "OK" "go build ./..."
printf "%-12s %-15s ${GREEN}%-8s${NC} %s\n" "14:28:10" "Bash" "OK" "go test ./..."
echo ""
echo -e "${BOLD}--- File Operations ---${NC}"
printf "%-40s %-10s %-10s\n" "File" "Operation" "Count"
echo "────────────────────────────────────────────────────────────"
printf "%-40s %-10s %-10s\n" "internal/executor/task.go" "Edit" "2"
printf "%-40s %-10s %-10s\n" "internal/executor/task.go" "Read" "1"
printf "%-40s %-10s %-10s\n" "internal/models/task.go" "Read" "1"
