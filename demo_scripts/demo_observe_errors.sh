#!/bin/bash
# Demo: conductor observe errors
# Simulates error pattern analysis

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe errors${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Error Pattern Analysis ===${NC}"
echo ""
printf "%-30s %-12s %-12s\n" "Error Type" "Count" "Percentage"
echo "────────────────────────────────────────────────────────────"
printf "%-30s ${RED}%-12s${NC} %-12s\n" "Compilation Error" "45" "32.1%"
printf "%-30s ${RED}%-12s${NC} %-12s\n" "Test Failure" "38" "27.1%"
printf "%-30s ${YELLOW}%-12s${NC} %-12s\n" "Timeout" "23" "16.4%"
printf "%-30s ${YELLOW}%-12s${NC} %-12s\n" "Permission Denied" "12" "8.6%"
printf "%-30s ${YELLOW}%-12s${NC} %-12s\n" "File Not Found" "10" "7.1%"
printf "%-30s ${YELLOW}%-12s${NC} %-12s\n" "Network Error" "8" "5.7%"
printf "%-30s %-12s %-12s\n" "Other" "4" "2.9%"
echo ""
echo "Total: 140 errors across 147 sessions (0.95 errors/session)"
echo ""
echo -e "${BOLD}--- Error Frequency by Tool ---${NC}"
printf "%-20s %-12s %-12s %-15s\n" "Tool" "Total Runs" "Errors" "Error Rate"
echo "────────────────────────────────────────────────────────────────────────"
printf "%-20s %-12s ${RED}%-12s${NC} %-15s\n" "Bash" "543" "78" "14.4%"
printf "%-20s %-12s ${YELLOW}%-12s${NC} %-15s\n" "Edit" "987" "28" "2.8%"
printf "%-20s %-12s ${YELLOW}%-12s${NC} %-15s\n" "Write" "654" "21" "3.2%"
printf "%-20s %-12s %-12s %-15s\n" "Task" "234" "18" "7.7%"
printf "%-20s %-12s %-12s %-15s\n" "FetchUrl" "98" "8" "8.2%"
echo ""
echo -e "${BOLD}--- Recent Errors ---${NC}"
printf "%-20s %-15s %-15s %s\n" "Time" "Session" "Tool" "Error"
echo "────────────────────────────────────────────────────────────────────────────────────"
printf "%-20s %-15s %-15s ${RED}%s${NC}\n" "2024-01-15 14:26" "abc123" "Bash" "go build: undefined reference"
printf "%-20s %-15s %-15s ${RED}%s${NC}\n" "2024-01-15 14:10" "ghi789" "Bash" "test timeout exceeded"
printf "%-20s %-15s %-15s ${YELLOW}%s${NC}\n" "2024-01-15 13:45" "xyz987" "Edit" "file locked by another process"
printf "%-20s %-15s %-15s ${RED}%s${NC}\n" "2024-01-15 13:30" "def456" "Bash" "npm ERR! peer dep conflict"
printf "%-20s %-15s %-15s ${YELLOW}%s${NC}\n" "2024-01-15 12:15" "uvw654" "FetchUrl" "connection timed out"
echo ""
echo -e "${BOLD}--- Error Trends (Last 7 Days) ---${NC}"
echo ""
echo "  Mon:  ████████████ 12"
echo "  Tue:  ███████████████████ 19"
echo "  Wed:  ██████████████████████████ 26"
echo "  Thu:  ███████████████████████ 23"
echo "  Fri:  ████████████████████████████████ 32"
echo "  Sat:  ██████████ 10"
echo "  Sun:  ██████████████████ 18"
echo ""
echo "Peak error day: Friday (likely due to deployment attempts)"
