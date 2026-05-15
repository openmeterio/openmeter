## Step 2: Read scan results

**Telemetry:**
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan read
TELEMETRY_STEP2_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 2, skip this step.**

Read `$PROJECT_ROOT/.archie/scan.json`. Note total files, detected frameworks, top-level directories, and `frontend_ratio`.

Also read `$PROJECT_ROOT/.archie/dependency_graph.json` if it exists — it provides the resolved directory-level dependency graph with node metrics (in-degree, out-degree, file count) and cycle data. Wave 1 agents can reference this for quantitative dependency analysis.

**UI layer detection:** Only spawn the dedicated UI Layer agent if `frontend_ratio` >= 0.20 (20%+ of source files are UI/frontend). A small SwiftUI menubar or a minor React admin panel in an otherwise backend/CLI/library project does NOT warrant a dedicated UI agent — the Structure agent will cover it.

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 2
```

