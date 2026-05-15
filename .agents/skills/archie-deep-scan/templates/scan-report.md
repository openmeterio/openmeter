# Archie Scan Report
> Deep scan baseline | <today's date in YYYY-MM-DD HH:MM UTC> | <total_functions> functions / <total_loc> LOC analyzed | baseline run

## Architecture Overview

<2-3 paragraphs from Part 2: architecture style, key components, most important decisions. Prose, not bullets.>

## Health Scores

| Metric | Current | Previous | Trend | What it means |
|--------|--------:|---------:|------:|---------------|
| Erosion    | <erosion>    | <prev or "—"> | <up/down/flat> | <one-liner interpretation> |
| Gini       | <gini>       | <prev or "—"> | <trend> | <one-liner> |
| Top-20%    | <top20>      | <prev or "—"> | <trend> | <one-liner> |
| Verbosity  | <verbosity>  | <prev or "—"> | <trend> | <one-liner> |
| LOC        | <total_loc>  | <prev or "—"> | <trend> | <one-liner> |

<one paragraph summarizing what the numbers say together>

### Complexity Trajectory
<short list of the top 5-8 high-CC functions from health.json with file:line and CC values, and what they suggest about risk concentration>

## Findings

Ranked by severity, grouped by novelty.

### NEW (first observed this scan)
<numbered list of findings — each: **[severity] Title.** Description. Confidence N.>

### RECURRING (previously documented, still present)
<only if prior report exists; otherwise omit this subsection>

### RESOLVED
<only if prior report exists; otherwise omit. "None" if nothing resolved.>

## Proposed Rules

<Any new rules proposed by Step 6 synthesis that are not yet in rules.json. Reference proposed_rules.json.>
```

Sources for Findings:
- `drift_report.json` — mechanical and deep drift findings from Phase 1 and 2
- `blueprint.json` — `pitfalls` (each causal chain becomes a finding), `decisions.trade_offs` with violated `violation_signals` (if any appear in drift_report)
- Top complexity offenders from `health.json` (only if CC ≥ 15 or a cluster — don't list every high-CC function as a finding)

Severity mapping:
- `error` — decision violations, inverted dependencies, cycles across architectural boundaries
- `warn` — pattern erosion, god-objects, pitfalls currently manifesting, trade-offs actively undermined
- `info` — structural observations (dependency magnets, high fan-in nodes) that aren't currently broken

Confidence: carry forward from drift findings when available; otherwise use 0.8-0.95 for findings grounded in direct code reading, lower for inferred ones.

Verify the write:
```bash
test -s "$PROJECT_ROOT/.archie/scan_report.md" && wc -l "$PROJECT_ROOT/.archie/scan_report.md"
```

Expected: non-empty file with at least 30 lines.

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 9
```

Save baseline marker for future incremental runs (use "full" or "incremental" based on SCAN_MODE):
```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" save-baseline SCAN_MODE
```
(Replace SCAN_MODE with the actual mode — "full" or "incremental")

End with: **"Archie is now active. Architecture rules will be enforced on every code change. Run `/archie-scan` for fast health checks. Run `/archie-deep-scan --incremental` after code changes to update the architecture analysis."**

