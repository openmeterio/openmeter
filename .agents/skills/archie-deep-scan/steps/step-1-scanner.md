## Step 1: Run the scanner

**Telemetry:** persist the step start to disk (compaction-safe), then keep the shell var for readability:
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan scan
TELEMETRY_STEP1_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 1, skip this step.**

```bash
python3 .archie/scanner.py "$PROJECT_ROOT"
python3 .archie/detect_cycles.py "$PROJECT_ROOT" --full 2>/dev/null
```

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 1
```

