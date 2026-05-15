## Step 10: Write telemetry

Each prior step persisted its start timestamp to `.archie/telemetry/_current_run.json` via `telemetry.py mark` — so the final writer reads entirely from disk (no shell variables required, no /tmp timing file to assemble). This is what makes mid-run `/compact` safe: even if the orchestrator's conversation was compacted, every step's timing is on disk.

If the Intent Layer was skipped (INTENT_LAYER=no), mark it so explicitly:

```bash
if [ "$INTENT_LAYER" = "no" ]; then
  python3 .archie/telemetry.py extra "$PROJECT_ROOT" intent_layer skipped=true
fi
```

Then flush the in-flight file into the final `.archie/telemetry/deep-scan_<timestamp>.json`:

```bash
python3 .archie/telemetry.py finish "$PROJECT_ROOT"
python3 .archie/telemetry.py write  "$PROJECT_ROOT"
```

`write` auto-closes any still-open step with `now`, emits the final timestamped JSON, then deletes `_current_run.json` so the next deep-scan starts fresh. If telemetry fails for any reason, do not abort — telemetry is informational only.

**Legacy fallback:** the old `/tmp/archie_timing.json` + `telemetry.py <root> --command … --timing-file …` invocation still works for any downstream tool that expects it, but the disk-persisted flow above is the compaction-safe canonical path.
