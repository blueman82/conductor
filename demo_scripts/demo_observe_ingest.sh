#!/bin/bash
# Demo: conductor observe ingest
# Simulates JSONL ingestion daemon

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}$ conductor observe ingest${NC}"
echo ""
sleep 0.5

echo "Processing files in ~/.claude/projects/..."
sleep 1
echo "  Processed: 156 events"
sleep 0.5
echo "  Processed: 342 events"
sleep 0.5
echo "  Processed: 567 events"
sleep 0.3
echo ""
echo "=================================================="
echo "Ingestion Summary"
echo "=================================================="
echo "Files Tracked:     23"
echo "Events Processed:  567"
echo "Sessions Created:  12"
echo "Errors:            0"
echo "Uptime:            2.345s"
echo ""

# Demo with --watch flag
echo -e "${BOLD}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BOLD}$ conductor observe ingest --watch --verbose${NC}"
echo ""
sleep 0.5

echo "Ingestion Configuration:"
echo "  Root Directory: /Users/demo/.claude/projects"
echo "  Pattern: *.jsonl"
echo "  Batch Size: 50"
echo "  Batch Timeout: 500ms"
echo "  Watch Mode: true"
echo ""
sleep 0.3
echo -e "Ingestion daemon started, watching /Users/demo/.claude/projects"
echo "Press Ctrl+C to stop..."
echo ""
sleep 1

echo -e "[${CYAN}INFO${NC}] Scanning for existing JSONL files..."
sleep 0.5
echo -e "[${CYAN}INFO${NC}] Found 23 files to process"
sleep 0.3
echo -e "[${CYAN}INFO${NC}] Processing: conductor/session-abc123.jsonl (156 events)"
sleep 0.5
echo -e "[${CYAN}INFO${NC}] Processing: my-webapp/session-def456.jsonl (234 events)"
sleep 0.5
echo -e "[${CYAN}INFO${NC}] Processing: api-service/session-ghi789.jsonl (177 events)"
sleep 0.8

echo ""
echo -e "[${GREEN}WATCH${NC}] Waiting for new events..."
sleep 1
echo -e "[${GREEN}WATCH${NC}] New file detected: conductor/session-new123.jsonl"
sleep 0.5
echo -e "[${CYAN}INFO${NC}] Processing: conductor/session-new123.jsonl (45 events)"
sleep 0.8
echo -e "[${GREEN}WATCH${NC}] File updated: conductor/session-new123.jsonl (+12 events)"
sleep 0.5

echo ""
echo -e "${YELLOW}^C${NC}"
echo ""
echo "Shutting down..."
sleep 0.3
echo ""
echo "=================================================="
echo "Ingestion Summary"
echo "=================================================="
echo "Files Tracked:     24"
echo "Events Processed:  624"
echo "Sessions Created:  13"
echo "Errors:            0"
echo "Uptime:            12.456s"
