## Resume Prelude (runs whenever `RESUME_ACTION=resume`)

Execute this block **before any other step** when resuming. It rehydrates every shell variable the pipeline depends on from disk, so the orchestrator does NOT need to have carried them forward in its conversation context.

`RESUME_ACTION=resume` is set by the Preamble in any of these cases:
- `--continue` flag was passed (explicit opt-in)
- `--from N` flag was passed (explicit opt-in; also sets `START_STEP=N`)
- **No flag was passed, partial state was detected, and the user chose "Resume"** at the interactive prompt (the bare-invocation path added in v2.4)

In all three cases, the variables `SCOPE`, `INTENT_LAYER`, `SCAN_MODE`, `MONOREPO_TYPE`, `WORKSPACES`, `PROJECT_NAME` need to be rehydrated from `deep_scan_state.run_context` — they were set during the original run and persisted to disk. Without this block, the resumed run has no idea what scope was chosen, whether Intent Layer was opt-in, etc. `PROJECT_ROOT` is NOT persisted (would leak machine-specific paths into committable state) — it's set to `$PWD` at resume time. In the common case (slash command invoked from the repo root, which is how Claude Code is typically used) that's correct. If `$PWD` differs from where the `.archie/` directory lives — e.g. the user invoked `/archie-deep-scan --continue` from a subdirectory — the `inspect` call below will fail to find state at `$PWD/.archie/deep_scan_state.json`, `LAST` stays at 0, and the Resume Prelude falls through to a fresh Phase 0 with no corruption. Symlinked `.archie/` directories resolve correctly because `$PWD/.archie/*` follows symlinks on read.

Safe to run on a fresh invocation too — the `LAST=0` branch short-circuits back to normal Phase 0 flow.

```bash
# 1. Read last_completed via dedicated CLI. A fresh run returns "null" → normalise to 0.
LAST=$(python3 .archie/intent_layer.py inspect "$PWD" deep_scan_state.json --query .last_completed 2>/dev/null)
[ -z "$LAST" ] || [ "$LAST" = "null" ] && LAST=0

# 2. If we have real state, rehydrate every shell variable from run_context.
# PROJECT_ROOT always comes from $PWD — it is intentionally not persisted to
# disk so `.archie/deep_scan_state.json` stays machine-agnostic.
if [ "$LAST" -gt 0 ]; then
    PROJECT_ROOT="$PWD"
    SCOPE=$(python3 .archie/intent_layer.py inspect "$PWD" deep_scan_state.json --query .run_context.scope 2>/dev/null)
    [ "$SCOPE" = "null" ] && SCOPE=""
    INTENT_LAYER=$(python3 .archie/intent_layer.py inspect "$PWD" deep_scan_state.json --query .run_context.intent_layer 2>/dev/null)
    [ "$INTENT_LAYER" = "null" ] || [ -z "$INTENT_LAYER" ] && INTENT_LAYER=yes
    SCAN_MODE=$(python3 .archie/intent_layer.py inspect "$PWD" deep_scan_state.json --query .run_context.scan_mode 2>/dev/null)
    [ "$SCAN_MODE" = "null" ] || [ -z "$SCAN_MODE" ] && SCAN_MODE=full
    MONOREPO_TYPE=$(python3 .archie/intent_layer.py inspect "$PWD" deep_scan_state.json --query .run_context.monorepo_type 2>/dev/null)
    [ "$MONOREPO_TYPE" = "null" ] || [ -z "$MONOREPO_TYPE" ] && MONOREPO_TYPE=none
    # WORKSPACES as newline-separated (matches the scope picker's original shape).
    WORKSPACES=$(python3 .archie/intent_layer.py inspect "$PWD" deep_scan_state.json --query .run_context.workspaces --list 2>/dev/null)
    PROJECT_NAME="${PROJECT_ROOT##*/}"

    # 3. Compute START_STEP. --from N overrides; otherwise resume at LAST+1.
    if [ -n "$FROM_STEP" ]; then
        START_STEP=$FROM_STEP
    else
        START_STEP=$((LAST + 1))
    fi

    # 4. Consistency check: telemetry _current_run.json should contain at least
    # one step entry per completed step. Warn loudly if the two states diverge
    # — that signals either corruption or manual intervention. Do not abort;
    # telemetry is informational, the scan itself is still resumable from
    # blueprint/findings on disk.
    TELEMETRY_STEPS=$(python3 .archie/telemetry.py steps-count "$PWD" 2>/dev/null)
    [ -z "$TELEMETRY_STEPS" ] && TELEMETRY_STEPS=0
    if [ "$TELEMETRY_STEPS" -lt "$LAST" ]; then
        echo "WARNING: deep_scan_state says last_completed=$LAST but telemetry has only $TELEMETRY_STEPS step marks. Final per-step timing may be incomplete. Scan output itself is not affected." >&2
    fi

    echo "Resuming from persisted state: SCOPE=$SCOPE SCAN_MODE=$SCAN_MODE INTENT_LAYER=$INTENT_LAYER last_completed=$LAST start_step=$START_STEP" >&2

    # 5. Skip Phase 0 (scope resolution) — we already have the answers on disk.
    # Jump directly to Step $START_STEP in the main pipeline below.
else
    # Fresh run: LAST=0 or no run_context. Fall through to Phase 0 normally.
    : # no-op; flow continues with scope resolution below.
fi
```

**Notes on accuracy:**

- Rehydrating from disk is lossless by construction — `save-context` wrote exactly these fields, and `save-context` runs both in Step F (fresh runs) and is safe to call again if anything changes mid-run.
- The consistency check is defensive. Under normal compact-and-resume flow, telemetry step count ≥ last_completed always holds because each step marks its start *before* calling `complete-step N`. A warning here signals something outside the happy path (manual state edit, aborted step, corrupted file).
- `WORKSPACES` is rehydrated as a newline-separated string, matching what Step C's scope picker originally produced. Downstream iteration patterns (`while IFS= read`; `printf '%s\n' "$WORKSPACES" | ...`) work identically.
- If `--from N` is supplied, the orchestrator sets `FROM_STEP=N` before this block runs.

