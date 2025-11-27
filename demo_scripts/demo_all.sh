#!/bin/bash
# Demo: Run all conductor demos
# Master script that runs all demo scenarios

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Demo mapping: number -> script, title
declare -a DEMOS=(
    "demo_validate.sh|Plan Validation (conductor validate)"
    "demo_run_dry.sh|Dry-Run Execution (conductor run --dry-run)"
    "demo_run.sh|Full Execution (conductor run)"
    "demo_run_failure.sh|Execution with Failures"
    "demo_learning.sh|Learning System (conductor learning)"
    "demo_observe_stats.sh|Observe Stats (conductor observe stats)"
    "demo_observe_project.sh|Observe Project (conductor observe project)"
    "demo_observe_session.sh|Observe Session (conductor observe session)"
    "demo_observe_tools.sh|Observe Tools (conductor observe tools)"
    "demo_observe_bash.sh|Observe Bash (conductor observe bash)"
    "demo_observe_files.sh|Observe Files (conductor observe files)"
    "demo_observe_errors.sh|Observe Errors (conductor observe errors)"
    "demo_observe_ingest.sh|Observe Ingest (conductor observe ingest)"
    "demo_observe_live.sh|Observe Live (conductor observe live)"
    "demo_observe_transcript.sh|Observe Transcript (conductor observe transcript)"
    "demo_observe_export.sh|Observe Export (conductor observe export)"
)

run_demo() {
    local script=$1
    local title=$2
    
    clear
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}${BOLD}DEMO: ${title}${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    bash "${SCRIPT_DIR}/${script}"
}

run_single_demo() {
    local num=$1
    local idx=$((num - 1))
    
    if [[ $idx -lt 0 || $idx -ge ${#DEMOS[@]} ]]; then
        echo -e "${RED}Invalid selection: $num${NC}"
        return 1
    fi
    
    IFS='|' read -r script title <<< "${DEMOS[$idx]}"
    run_demo "$script" "$title"
}

run_all_demos() {
    for i in "${!DEMOS[@]}"; do
        IFS='|' read -r script title <<< "${DEMOS[$i]}"
        run_demo "$script" "$title"
        
        if [[ $i -lt $((${#DEMOS[@]} - 1)) ]]; then
            echo ""
            echo -e "${CYAN}Press Enter to continue to next demo (or 'q' to quit)...${NC}"
            read -r input
            if [[ "$input" == "q" || "$input" == "Q" ]]; then
                echo "Exiting demo."
                return
            fi
        fi
    done
    
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}${BOLD}Demo Complete!${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

show_menu() {
    clear
    echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║                    CONDUCTOR DEMO                            ║${NC}"
    echo -e "${BOLD}║     Autonomous Multi-Agent Orchestration for Claude Code     ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "This demo will showcase all Conductor commands and features."
    echo ""
    echo -e "${BOLD}Available demos:${NC}"
    
    for i in "${!DEMOS[@]}"; do
        IFS='|' read -r script title <<< "${DEMOS[$i]}"
        printf "  %2d. %s\n" $((i + 1)) "$title"
    done
    
    echo ""
    echo -e "${BOLD}Options:${NC}"
    echo "   a. Run all demos in sequence"
    echo "   q. Quit"
    echo ""
}

# Main loop
while true; do
    show_menu
    echo -ne "${CYAN}Enter selection (1-16, 'a' for all, 'q' to quit): ${NC}"
    read -r choice
    
    case "$choice" in
        [qQ])
            echo "Goodbye!"
            exit 0
            ;;
        [aA])
            run_all_demos
            echo ""
            echo -e "${CYAN}Press Enter to return to menu...${NC}"
            read -r
            ;;
        [1-9]|1[0-6])
            run_single_demo "$choice"
            echo ""
            echo -e "${CYAN}Press Enter to return to menu...${NC}"
            read -r
            ;;
        *)
            echo -e "${RED}Invalid selection. Please enter 1-16, 'a', or 'q'.${NC}"
            sleep 1
            ;;
    esac
done
