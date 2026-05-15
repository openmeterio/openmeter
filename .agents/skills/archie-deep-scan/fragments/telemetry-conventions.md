## Telemetry conventions

Every Step 1–9 records its start timestamp to disk via `telemetry.py mark` as its first action. The mark auto-closes the previous step's `completed_at`, mirroring the "next step's start = prior step's end" convention. Step 9 finishes its own completion with `telemetry.py finish`. After Step 10, `telemetry.py write` consumes the persisted `.archie/telemetry/_current_run.json` and emits the final `deep-scan_<timestamp>.json`.

Shell-variable fallback: the existing `TELEMETRY_STEPN_START` shell variables are still set for readability, but they are **not load-bearing** — the disk file is the source of truth. This makes the pipeline safe to `/compact` mid-run without losing timing data.

