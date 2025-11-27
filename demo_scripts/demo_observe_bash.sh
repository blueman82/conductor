#!/bin/bash
# Demo: conductor observe bash
# Simulates bash command pattern analysis

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe bash${NC}"
echo ""
sleep 0.5

echo ""
echo -e "${BOLD}=== Bash Command Analysis ===${NC}"
echo ""
printf "%-35s %-10s %-12s %-12s\n" "Command Pattern" "Count" "Success" "Avg Time"
echo "────────────────────────────────────────────────────────────────────────────"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "go test ./..." "156" "92.3%" "3.2s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "go build ./..." "134" "88.8%" "4.5s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "git status" "98" "100.0%" "0.1s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "git diff" "87" "100.0%" "0.2s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "npm install" "45" "84.4%" "12.3s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "npm run build" "42" "81.0%" "8.7s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "make test" "38" "89.5%" "5.1s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "python -m pytest" "32" "87.5%" "6.4s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "docker build" "28" "78.6%" "45.2s"
printf "%-35s %-10s ${GREEN}%-12s${NC} %-12s\n" "cargo test" "15" "93.3%" "8.9s"
echo ""
echo "Total: 543 bash commands executed"
echo ""
echo -e "${BOLD}--- Exit Code Distribution ---${NC}"
printf "%-15s %-10s %-10s\n" "Exit Code" "Count" "Percentage"
echo "────────────────────────────────────────────"
printf "%-15s %-10s ${GREEN}%-10s${NC}\n" "0 (success)" "465" "85.6%"
printf "%-15s %-10s ${RED}%-10s${NC}\n" "1 (error)" "56" "10.3%"
printf "%-15s %-10s ${RED}%-10s${NC}\n" "2 (misuse)" "12" "2.2%"
printf "%-15s %-10s ${YELLOW}%-10s${NC}\n" "124 (timeout)" "8" "1.5%"
printf "%-15s %-10s ${RED}%-10s${NC}\n" "other" "2" "0.4%"
echo ""
echo -e "${BOLD}--- Frequently Failed Commands ---${NC}"
printf "%-35s %-10s %s\n" "Command" "Failures" "Common Error"
echo "────────────────────────────────────────────────────────────────────────────"
printf "%-35s %-10s %s\n" "go build ./..." "15" "compilation error"
printf "%-35s %-10s %s\n" "go test ./..." "12" "test failure"
printf "%-35s %-10s %s\n" "npm run build" "8" "module not found"
printf "%-35s %-10s %s\n" "docker build" "6" "dockerfile error"
printf "%-35s %-10s %s\n" "npm install" "5" "peer dependency conflict"
