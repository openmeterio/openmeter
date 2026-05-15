## Compact-and-resume contract

At every "✓ Step N complete" boundary, all state needed to resume lives on disk:

- `.archie/deep_scan_state.json` — last completed step + `run_context` (scope, intent_layer, scan_mode, workspaces, monorepo_type, start_step). Note: `project_root` is deliberately NOT persisted — the Resume Prelude sets `PROJECT_ROOT="$PWD"` directly, which avoids leaking machine-specific absolute paths into committable state.
- `.archie/telemetry/_current_run.json` — every step's start/completed timestamp + extras
- `.archie/archie_config.json` — persisted scope picker answer (whole/per-package/hybrid/single)
- `.archie/blueprint_raw.json`, `.archie/blueprint.json`, `.archie/findings.json` — pipeline output as it accumulates
- `.archie/enrich_state.json`, `.archie/enrich_batches.json` — Intent Layer DAG scheduler state (survives mid-Step-7 compaction)

After a `/compact`, running `/archie-deep-scan --continue` re-enters via the **Resume Prelude** below, which rehydrates every shell variable from disk before jumping to the next step. No conversation memory required.

