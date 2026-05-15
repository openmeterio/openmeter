## Step 4: Save Wave 1 output and merge

**Telemetry:**
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan merge
TELEMETRY_STEP4_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 4, skip this step.**

### If SCAN_MODE = "incremental":

The single incremental agent's output was saved to `/tmp/archie_incremental_$PROJECT_NAME.json` in Step 3. Patch the existing blueprint:

```bash
python3 .archie/merge.py "$PROJECT_ROOT" --patch /tmp/archie_incremental_$PROJECT_NAME.json
```

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 3
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 4
```

### If SCAN_MODE = "full" (default):

**If resuming via --from or --continue:** Step 4 depends on Wave 1 agent outputs in /tmp/. These may not survive a system reboot. If merge fails with missing files, re-run from step 3: `/archie-deep-scan --from 3`

**Subagent output contract (mandatory — append to each agent's prompt before spawning):**

Each Wave 1 subagent must write its own output directly to a pre-specified path. The orchestrator must NEVER copy or Write the transcript itself — attempting to access `.claude/projects/.../subagents/*.jsonl` triggers a sensitive-file permission prompt on every call.

Append this block to each Wave 1 agent's prompt, substituting `<OUTPUT_PATH>` with the path below:

```
---
OUTPUT CONTRACT (mandatory):
1. Use the Write tool to save your COMPLETE output to <OUTPUT_PATH>.
2. Write the raw output verbatim — the merge script handles JSON envelopes, code fences, and multi-block text.
3. After Writing, reply with exactly: "Wrote <OUTPUT_PATH>"
4. Do NOT print the output in your response body. /tmp/archie_* is already permissioned via Write(//tmp/archie_*).
```

Output paths per agent:
- Structure agent → `/tmp/archie_sub1_$PROJECT_NAME.json`
- Patterns agent → `/tmp/archie_sub2_$PROJECT_NAME.json`
- Technology agent → `/tmp/archie_sub3_$PROJECT_NAME.json`
- UI Layer agent (if spawned) → `/tmp/archie_sub4_$PROJECT_NAME.json`

When each subagent's confirmation reply returns, its file is already on disk — proceed directly to the merge step below. Do NOT attempt to re-extract output from the subagent's conversation — if the confirmation is missing or file absent, skip that agent's contribution and report the failure.

Then merge:

```bash
python3 .archie/merge.py "$PROJECT_ROOT" /tmp/archie_sub1_$PROJECT_NAME.json /tmp/archie_sub2_$PROJECT_NAME.json /tmp/archie_sub3_$PROJECT_NAME.json /tmp/archie_sub4_$PROJECT_NAME.json
```

This saves `$PROJECT_ROOT/.archie/blueprint_raw.json` (raw merged data). Verify the output shows non-zero component/section counts. If it says "0 sections, 0 components", the merge failed — check the agent output files.

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 4
```

