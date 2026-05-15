## Step 7: Intent Layer — per-folder CLAUDE.md

**Telemetry:**
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan intent_layer
python3 .archie/telemetry.py extra "$PROJECT_ROOT" intent_layer model=sonnet skipped=false
TELEMETRY_STEP7_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 7, skip this step.**

**If `INTENT_LAYER=no` (user opted out in Step E), skip this entire step.** Print a one-line note to the user: *"Intent Layer skipped (no per-folder CLAUDE.md generated). Root CLAUDE.md + rule files still written. You can run `/archie-intent-layer` later if you change your mind."* Then proceed to Step 8. The `intent_layer` telemetry step will record zero elapsed time (its `started_at == completed_at`) and carry `"skipped": true` (see Step 10).

**If `INTENT_LAYER=yes`, execute this step fully. Do NOT ask the user whether to run, skip, or reduce scope. Do NOT offer alternatives. Run all batches as instructed below.**

### Shared pipeline

**This step runs the exact same pipeline as the standalone `/archie-intent-layer` command.** The canonical description lives there (Phases 1–4): prepare the DAG → loop `next-ready` / `suggest-batches` / Sonnet subagent per batch / `save-enrichment` → `merge` enrichments into per-folder CLAUDE.md files.

**Load the canonical prose into context before starting** — slash-command bodies are not cross-loaded automatically, so you must Read the file yourself:

```
Read .claude/commands/archie-intent-layer.md
```

Then execute Phases 1–4 from that file, using `PROJECT_ROOT` in place of `$PWD`, with the deep-scan-specific deltas below layered on top. Do NOT reinterpret or re-derive the pipeline logic — follow what the file says.

### Deep-scan-specific deltas

1. **Skip the precondition check (Phase 0).** The blueprint was just produced in Steps 5–6, so the hard-requirement check in `/archie-intent-layer` Phase 0 is a no-op in this context. Do not re-run it.

2. **Auto-resume when this deep-scan itself is resuming.** If the Preamble set `RESUME_ACTION=resume` (from `--continue`, `--from N`, or the user picking "Resume" at the bare-invocation prompt), pass `RESUME_INTENT=continue` to the Intent Layer. Phase 0.25 in the intent-layer skips its own reconciliation prompt and auto-resumes from the on-disk done list.

   For fresh deep-scan runs (`RESUME_ACTION=fresh`), pass `RESUME_INTENT=ask`. Phase 0.25 will either see no partial state (the Preamble's Fresh-start path already reset it) or — in the rare case where enrichments survived the reset — ask the user. The Fresh-start path in the Preamble explicitly resets enrich_state + enrichments/, so this should be a no-op in practice.

3. **Skip the mode selector (Phase 0.5).** The deep-scan already decided `SCAN_MODE` in its own preamble — don't ask the user again. In Phase 1 of `/archie-intent-layer`, treat `MODE=incremental` as equivalent to `SCAN_MODE=incremental`, and `MODE=full` as `SCAN_MODE=full`.

3. **SCAN_MODE = "incremental" → pass `--only-folders`.** When the preamble set `SCAN_MODE=incremental`, the Phase 1 `prepare` call becomes:

   ```bash
   python3 .archie/intent_layer.py prepare "$PROJECT_ROOT" --only-folders AFFECTED_FOLDER1,AFFECTED_FOLDER2,...
   ```

   Use the comma-separated `affected_folders` list from the detect-changes output you captured earlier in the deep-scan run. `next-ready` will then only return dirty folders and their ancestors — waves will be much smaller than a full scan.

3. **Batch-processing compact checkpoint.**

   **✓ Compact Checkpoint B** — between Intent Layer waves. After every wave's `save-enrichment` commands have all returned (so no subagent is in flight and all completed folders are persisted in `enrich_state.json`), this is a safe compaction boundary. Suggested frequency: every 3 waves when the project has >20 folders. On small projects (<20 folders) ignore this checkpoint. Procedure: `/compact` → `/archie-deep-scan --continue` → Resume Prelude sees `last_completed=6` and re-enters Step 7, which calls `next-ready` and resumes from the next wave using disk state alone.

4. **Project blueprint scoped patterns into per-folder CLAUDE.md.** After Phase 3 (`merge`) writes the AI-generated per-folder CLAUDE.md files, project the blueprint's scoped `implementation_guidelines` and `communication.patterns` into the matching component-root CLAUDE.md files. This is the hard-filter delivery: scoped rules land in the component's CLAUDE.md (between `<!-- archie:scoped-start --> ... <!-- archie:scoped-end -->` markers) and Claude Code's per-folder autoloading does the path-based filtering — they never load on out-of-scope edits.

   ```bash
   python3 .archie/intent_layer.py inject-scoped "$PROJECT_ROOT"
   ```

   Idempotent: re-running replaces the marker block in place. If a pattern's scope shrinks between runs, the stale block is cleared from out-of-scope CLAUDE.md files. Patterns with `scope: []` (repo-wide) are NOT projected — they already live in global rules and would only duplicate.

5. **Mark step complete at the end.** After the inject-scoped step, record completion:

   ```bash
   python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 7
   ```

---

### ✓ Compact Checkpoint C — after Intent Layer

Only meaningful when `INTENT_LAYER=yes`. Step 7 has just pushed dozens-to-hundreds of Sonnet subagent transcripts into conversation context; those are now fully persisted to `.archie/enrichments/*.json` and merged into per-folder `CLAUDE.md` files. Compacting here gives Step 9 (Drift Assessment) a fresh context, which matters because drift assessment reads blueprint + drift_report + CLAUDE.md files and benefits from focused attention.

If `INTENT_LAYER=no` (opted out in Step E), skip this checkpoint — Checkpoint A already covered it.

Procedure when firing: `/compact` → `/archie-deep-scan --continue` → Resume Prelude sees `last_completed=7` and jumps to Step 8 (Cleanup is cheap, so continuing through 8→9 in a fresh context costs nothing).

---

