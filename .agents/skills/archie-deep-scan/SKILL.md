---
name: archie-deep-scan
description: Comprehensive architecture baseline scan (15-20 min). Two-wave AI analysis producing blueprint.json, per-folder CLAUDE.md, AI-synthesized rules, health metrics, and drift detection. Use for first-time baselines or major refactors; use /archie-scan for incremental health checks.
---

# Archie Deep Scan — Comprehensive Architecture Baseline

Run a comprehensive architecture analysis. Produces full blueprint, per-folder CLAUDE.md, rules, and health metrics.

**Modes:**
- `/archie-deep-scan` — full baseline from step 1 (default, proven workflow)
- `/archie-deep-scan --incremental` — only process files changed since last deep scan (fast, 3-6 min)
- `/archie-deep-scan --from N` — resume from step N (runs N through 9)
- `/archie-deep-scan --continue` — resume from where the last run stopped

**Prerequisites:** Run `npx @bitraptors/archie` first to install the scripts. If `.archie/scanner.py` doesn't exist, tell the user to run `npx @bitraptors/archie` and try again.

## Update notice (run before anything else, silent unless action needed)

```bash
python3 .archie/update_check.py check 2>/dev/null
```

If output is non-empty:
- `UPGRADE_AVAILABLE old new` → tell the user once at the top of your reply: `"Archie {new} is available (installed: {old}). Upgrade: npx @bitraptors/archie@latest \"$PWD\""`. Then continue with the scan — do not block.
- `JUST_UPGRADED old new` → say `"Archie upgraded {old} → {new}."` once, then proceed.

If output is empty: proceed silently. This is informational only.

## Telemetry consent (one-time, run before anything else)

Read and follow `.claude/commands/_shared/telemetry-consent.md`. It checks whether this machine has been asked about anonymous usage telemetry and, if not, presents a one-time `AskUserQuestion` opt-in. It self-skips after the first answer and on non-interactive sessions.

**CRITICAL CONSTRAINT: Never write inline Python.**
Do NOT use `python3 -c "..."` or any ad-hoc scripting to inspect, parse, or transform JSON. Every operation has a dedicated command:
- Normalize blueprint: `python3 .archie/finalize.py "$PROJECT_ROOT" --normalize-only`
- Append health history: `python3 .archie/measure_health.py "$PROJECT_ROOT" --append-history --scan-type deep`
- Inspect any JSON file: `python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" <filename>`
- Query a specific field: `python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" scan.json --query .frontend_ratio`

If you need data not covered by these commands, proceed without it or ask the user. NEVER improvise Python.

## Preamble: Determine starting step

Check the user's message (ARGUMENTS) for flags:

**If `--from N` is present** (e.g., `/archie-deep-scan --from 5`):
1. Set `START_STEP = N` (the number after --from) and `RESUME_ACTION=resume` (so the Resume Prelude rehydrates shell variables from `deep_scan_state.run_context`).
2. Validate prerequisites exist:
```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" check-prereqs N
```
3. If check fails, tell the user which files are missing and which earlier step to run.
4. If check passes, proceed. Do NOT call `deep-scan-state init` — it would wipe the state the Resume Prelude needs to read.

**If `--continue` is present:**
1. Read state (no prompt — `--continue` is an explicit opt-in):
```bash
LAST=$(python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" deep_scan_state.json --query .last_completed 2>/dev/null)
STATUS=$(python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" deep_scan_state.json --query .status 2>/dev/null)
[ -z "$LAST" ] || [ "$LAST" = "null" ] && LAST=0
```
2. If `LAST == 0` or `STATUS == "completed"`: print "No interrupted run found. Starting fresh from step 1." Set `START_STEP=1`, `RESUME_ACTION=fresh`, and run `deep-scan-state init` below.
3. Otherwise: Set `START_STEP = LAST + 1`, `RESUME_ACTION=resume`. Print `"Resuming deep scan from step {START_STEP}."`. Skip the init call.

**If `--incremental` is present:**
1. Check if `.archie/blueprint.json` exists. If not: print "No existing blueprint — running full baseline instead." Set SCAN_MODE = "full", START_STEP = 1, and proceed as default.
2. If blueprint exists, detect changes:
```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" detect-changes
```
3. Read the JSON output:
   - If `mode` is "full" (threshold exceeded or no previous scan): print the `reason` and say "Running full baseline." Set SCAN_MODE = "full", START_STEP = 1.
   - If `mode` is "incremental" and `changed_count` is 0: print "No files changed since last deep scan. Nothing to do." Exit.
   - If `mode` is "incremental": Set SCAN_MODE = "incremental". Save `changed_files` and `affected_folders` from the output. Print "Incremental deep scan: N files changed. Analyzing changes only." Set START_STEP = 1.
4. Initialize state:
```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" init
```

**If no flags (default — detect interrupted run, otherwise full baseline):**

Before assuming a fresh run, check whether a previous deep-scan was left unfinished. If so, offer the user a choice instead of silently wiping their work.

1. Read state:
```bash
LAST=$(python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" deep_scan_state.json --query .last_completed 2>/dev/null)
STATUS=$(python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" deep_scan_state.json --query .status 2>/dev/null)
[ -z "$LAST" ] || [ "$LAST" = "null" ] && LAST=0
```

2. **If `LAST == 0` or `STATUS == "completed"`** → no interrupted run. Proceed as fresh:
   - Set `SCAN_MODE=full`, `START_STEP=1`, `RESUME_ACTION=fresh`.
   - Initialize state: `python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" init`.

3. **Otherwise** (`LAST > 0` and `STATUS == "in_progress"`) → an interrupted run exists. Figure out which step stopped it and where enrichment state stands:
   ```bash
   ENRICH_DONE=$(python3 .archie/intent_layer.py inspect "$PROJECT_ROOT" enrich_state.json --query '.done|length' 2>/dev/null)
   [ -z "$ENRICH_DONE" ] || [ "$ENRICH_DONE" = "null" ] && ENRICH_DONE=0
   ```
   Build a human-readable label for the last completed step:
   | LAST | step_name |
   |---|---|
   | 1 | scanner |
   | 2 | read accumulated knowledge |
   | 3 | Wave 1 analytical agents |
   | 4 | Wave 1 merge |
   | 5 | Wave 2 reasoning agent |
   | 6 | AI rule synthesis |
   | 7 | Intent Layer |
   | 8 | Cleanup |
   | 9 | Drift detection |

   Call `AskUserQuestion`:
   - **question:** (build dynamically) `"A previous deep-scan stopped after Step {LAST} ({step_name})."` — and if `ENRICH_DONE > 0`, append `" The Intent Layer got {ENRICH_DONE} folders in before stopping."` — then `"What do you want to do?"`
   - **header:** "Resume"
   - **multiSelect:** false
   - **options** (exactly these two labels):
     1. label `Resume` — description `Continue from Step {LAST+1}. Preserves all completed work, including any partial Intent Layer batches. Recommended.`
     2. label `Fresh start` — description `Discard all progress and restart from Step 1. Erases the interrupted blueprint, Wave 1 outputs, and the partial Intent Layer state. Use only if the codebase changed significantly.`

   Map the answer:
   - `Resume` → Set `START_STEP = LAST + 1`, `RESUME_ACTION=resume`. Skip `init`. Print `"Resuming from Step {START_STEP}."`.
   - `Fresh start` → Reset everything and start over:
     ```bash
     python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" init
     python3 .archie/intent_layer.py reset-state "$PROJECT_ROOT"
     rm -f /tmp/archie_enrichment_*.json
     ```
     `reset-state` wipes both `.archie/enrich_state.json` and the `.archie/enrichments/` directory — no `rm -rf` needed in the slash-command layer (keeps the command inside the default Bash permission allowlist so this runs prompt-free).
     Set `SCAN_MODE=full`, `START_STEP=1`, `RESUME_ACTION=fresh`. Print `"Starting fresh. Previous progress discarded."`.

4. Regardless of branch above, `RESUME_ACTION` is now set. It gates the Resume Prelude (below) and the Step 7 delta (passes `RESUME_INTENT` to the Intent Layer).

**For every step below:**
- If the step number < START_STEP, skip it entirely.
- If SCAN_MODE is not set, it defaults to "full" (all existing behavior unchanged).
- **Do NOT ask the user any questions during Steps 1–10. Do NOT offer to skip, reduce scope, or present alternatives for any step. Execute every step fully as documented.** This rule applies ONLY to Steps 1–10. It does NOT apply to Phase 0 / Activation: the scope prompt (Step C) and Intent Layer prompt (Step E) in `scope_resolution.md` are mandatory decision gates and MUST still be asked — see below.


## Activation — read these before running any step

Before executing any step, Read these files in order. They establish the
conventions and Phase 0 variables (`SCOPE`, `WORKSPACES`, `MONOREPO_TYPE`,
`PROJECT_ROOT`, `PROJECT_NAME`) that every step assumes are in place.

All paths below are relative to the project root (your cwd). The fragments
live alongside this orchestrator under `.claude/skills/archie-deep-scan/`.

1. `.claude/skills/archie-deep-scan/fragments/telemetry-conventions.md` — telemetry mark / finish / write contract used by every step.
2. `.claude/skills/archie-deep-scan/fragments/compact-resume-contract.md` — how the pipeline survives `/compact` mid-run via `.archie/deep_scan_state.json`.
3. **If `RESUME_ACTION=resume`:** `.claude/skills/archie-deep-scan/fragments/resume-prelude.md` — rehydrates shell variables from persisted state.
4. `.claude/commands/_shared/scope_resolution.md` — Phase 0 scope resolution. Establishes `PROJECT_ROOT`, `PROJECT_NAME`, `SCOPE`, `WORKSPACES`, `MONOREPO_TYPE`.

## Step-by-step routing

Before starting any Step N, Read the file in the "Load this file" column.
The router does not contain step content — each step is a self-contained file.
If `START_STEP > N` (the Preamble decided to skip earlier steps), do not Read or run those steps.

| Step | What it does | Load this file before starting |
|---|---|---|
| 1 | Run the scanner | `.claude/skills/archie-deep-scan/steps/step-1-scanner.md` |
| 2 | Read accumulated knowledge from prior runs | `.claude/skills/archie-deep-scan/steps/step-2-read-scan.md` |
| 3 | Wave 1 — spawn parallel analytical agents | `.claude/skills/archie-deep-scan/steps/step-3-wave1/orchestration.md` |
| 4 | Save & merge Wave 1 output | `.claude/skills/archie-deep-scan/steps/step-4-merge.md` |
| 5 | Wave 2 — reasoning agent (Opus) | `.claude/skills/archie-deep-scan/steps/step-5-wave2-reasoning.md` |
| 6 | AI rule synthesis | `.claude/skills/archie-deep-scan/steps/step-6-rule-synthesis.md` |
| 7 | Intent Layer — per-folder CLAUDE.md | `.claude/skills/archie-deep-scan/steps/step-7-intent-layer.md` |
| 8 | Cleanup | `.claude/skills/archie-deep-scan/steps/step-8-cleanup.md` |
| 9 | Drift detection & architectural assessment | `.claude/skills/archie-deep-scan/steps/step-9-drift.md` |
| 10 | Final telemetry flush | `.claude/skills/archie-deep-scan/steps/step-10-telemetry.md` |

Step 3's `orchestration.md` in turn references four sub-agent prompt files plus a shared `grounding-rules.md` (all under `.claude/skills/archie-deep-scan/steps/step-3-wave1/`) — read those as the orchestration instructs.
