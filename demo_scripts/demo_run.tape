# VHS Tape File - Conductor Demo
# Install: brew install vhs
# Run: vhs demo_run.tape

Output demo_run.gif

Set FontSize 14
Set Width 900
Set Height 600
Set Theme "Dracula"
Set Padding 20

# Speed up the demo (lower = faster)
Set TypingSpeed 50ms

Type "conductor run plan.md --verbose"
Enter
Sleep 500ms

# Simulated output (we'll use the actual script)
# For real recording, replace with: Type "./demo_run.sh"

Sleep 300ms
Type "Loading plan from plan.md..."
Enter
Sleep 200ms
Type "Validating dependencies..."
Enter
Sleep 300ms

Enter
Type "Plan Summary:"
Enter
Type "  Total tasks: 6"
Enter
Type "  Execution waves: 3"
Enter
Type "  Timeout: 10h0m0s"
Enter
Type "  Max concurrency: 3"
Enter
Enter
Sleep 300ms

Type "Starting execution..."
Enter
Enter

# Wave 1
Type "[14:23:45] Starting Wave 1: 2 tasks"
Enter
Sleep 200ms
Type "[14:23:45] ⏳ IN PROGRESS [1/2] Task 1 (Initialize Project) (agent: golang-pro)"
Enter
Sleep 800ms
Type "[14:23:47] ✓ Task 1 (Initialize Project): GREEN (2.3s, agent: golang-pro, 3 files)"
Enter
Sleep 200ms
Type "[14:23:47] ⏳ IN PROGRESS [2/2] Task 2 (Setup Database) (agent: database-optimizer)"
Enter
Sleep 800ms
Type "[14:23:50] ✓ Task 2 (Setup Database): GREEN (3.1s, agent: database-optimizer, 2 files)"
Enter
Sleep 200ms
Type "[14:23:50] Wave 1 complete (5.4s) - 2/2 completed (2 GREEN)"
Enter
Enter

# Wave 2
Sleep 300ms
Type "[14:23:50] Starting Wave 2: 3 tasks"
Enter
Sleep 200ms
Type "[14:23:50] ⏳ IN PROGRESS [1/3] Task 3 (User Auth) (agent: security-auditor)"
Enter
Sleep 600ms
Type "[14:23:52] [QC] Selected agents: [code-reviewer, security-auditor] (mode: intelligent)"
Enter
Sleep 200ms
Type "[14:23:52] [QC] Final verdict: GREEN (strictest-wins)"
Enter
Type "[14:23:52] ✓ Task 3 (User Auth): GREEN (4.2s, agent: security-auditor, 5 files)"
Enter
Sleep 200ms
Type "[14:23:52] ⏳ IN PROGRESS [2/3] Task 4 (API Endpoints) (agent: golang-pro)"
Enter
Sleep 600ms
Type "[14:23:54] ✓ Task 4 (API Endpoints): GREEN (2.8s, agent: golang-pro, 4 files)"
Enter
Sleep 200ms
Type "[14:23:54] ⏳ IN PROGRESS [3/3] Task 5 (Data Models) (agent: golang-pro)"
Enter
Sleep 600ms
Type "[14:23:56] [QC] Selected agents: [code-reviewer] (mode: auto)"
Enter
Sleep 200ms
Type "[14:23:56] [QC] Final verdict: YELLOW (strictest-wins)"
Enter
Type "[14:23:56] ⚠ Task 5 (Data Models): YELLOW (3.5s, agent: golang-pro, 2 files)"
Enter
Sleep 200ms
Type "[14:23:56] Wave 2 complete (10.5s) - 3/3 completed (2 GREEN, 1 YELLOW)"
Enter
Enter

# Wave 3 with retry
Sleep 300ms
Type "[14:23:56] Starting Wave 3: 1 tasks"
Enter
Sleep 200ms
Type "[14:23:56] ⏳ IN PROGRESS [1/1] Task 6 (Integration Tests) (agent: golang-pro)"
Enter
Sleep 600ms
Type "[14:23:58] [QC] Selected agents: [code-reviewer, golang-pro] (mode: intelligent)"
Enter
Sleep 200ms
Type "[14:23:58] [QC] Final verdict: RED (strictest-wins)"
Enter
Type "[14:23:58] ✗ Task 6 (Integration Tests): RED (4.1s)"
Enter
Type "[14:23:58]   Feedback: Test coverage below 80% threshold."
Enter
Sleep 400ms
Type "[14:23:58] Retrying Task 6... (attempt 2/3)"
Enter
Sleep 600ms
Type "[14:24:00] [QC] Final verdict: GREEN (strictest-wins)"
Enter
Type "[14:24:00] ✓ Task 6 (Integration Tests): GREEN (3.8s, agent: golang-pro, 3 files)"
Enter
Sleep 200ms
Type "[14:24:00] Wave 3 complete (9.9s) - 1/1 completed (1 GREEN)"
Enter
Enter

# Summary
Sleep 400ms
Type "[14:24:00] === Execution Summary ==="
Enter
Type "[14:24:00] Total tasks: 6"
Enter
Type "[14:24:00] Completed: 6"
Enter
Type "[14:24:00] Failed: 0"
Enter
Type "[14:24:00] Duration: 25s"
Enter
Type "[14:24:00] Status Breakdown:"
Enter
Type "[14:24:00]   GREEN: 5"
Enter
Type "[14:24:00]   YELLOW: 1"
Enter
Type "[14:24:00] Agent Usage:"
Enter
Type "[14:24:00]   golang-pro: 4 tasks"
Enter
Type "[14:24:00]   security-auditor: 1 tasks"
Enter
Type "[14:24:00]   database-optimizer: 1 tasks"
Enter
Type "[14:24:00] Files Modified: 19 files"
Enter
Type "[14:24:00] Average Duration: 4.3s/task"
Enter
Enter
Type "Execution completed successfully!"
Enter
Type "Logs written to: .conductor/logs"
Enter

Sleep 2s
