## Step 9: Drift Detection & Architectural Assessment

**Telemetry:**
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan drift
TELEMETRY_STEP9_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 9, skip this step.**

### Phase 0: Health measurement

```bash
python3 .archie/measure_health.py "$PROJECT_ROOT" > "$PROJECT_ROOT/.archie/health.json" 2>/dev/null
```

Save health scores to history for trending:

```bash
python3 .archie/measure_health.py "$PROJECT_ROOT" --append-history --scan-type deep
```

### Phase 1: Mechanical drift scan

```bash
python3 .archie/drift.py "$PROJECT_ROOT"
```

### Phase 2: Deep architectural drift (AI)

Identify files to analyze:
```bash
git -C "$PROJECT_ROOT" log --name-only --pretty=format: --since="30 days ago" -- '*.kt' '*.java' '*.swift' '*.ts' '*.tsx' '*.py' '*.go' '*.rs' | sort -u | head -100
```
If that returns nothing (new repo or no recent changes), use all source files from the scan:
```bash
python3 .archie/extract_output.py recent-files "$PROJECT_ROOT/.archie/scan.json"
```

For each file (batch into groups of ~15), collect:
- The file's content
- Its folder's CLAUDE.md **if it exists** (per-folder patterns, anti-patterns — these were generated in Step 7, but may be missing if Step 7 was skipped or partially completed)
- Its parent folder's CLAUDE.md **if it exists**

Read `$PROJECT_ROOT/.archie/blueprint.json` — specifically `decisions.key_decisions`, `decisions.decision_chain`, `decisions.trade_offs` (with `violation_signals`), `pitfalls` (with `stems_from`), `communication.patterns`, `development_rules`.

Read `$PROJECT_ROOT/.archie/drift_report.json` (mechanical findings from Phase 1).

Spawn a **Sonnet subagent** (`model: "sonnet"`) with the file contents, their folder CLAUDE.md files, and the blueprint context. Tell it:

> You are an architecture reviewer. You have the project's architectural blueprint (decisions, trade-offs, pitfalls, patterns), per-folder CLAUDE.md files describing expected patterns, mechanical drift findings (already detected), and source files to review.
>
> Find **deep architectural violations** — problems that pattern matching cannot catch. For each finding, return:
> - `folder`: the folder path
> - `file`: the specific file
> - `type`: one of `decision_violation`, `pattern_erosion`, `trade_off_undermined`, `pitfall_triggered`, `responsibility_leak`, `abstraction_bypass`, `semantic_duplication`
> - `severity`: `error` or `warn`
> - `decision_or_pattern`: which architectural decision, pattern, or pitfall this violates (reference by name from the blueprint)
> - `evidence`: the specific code (function name, class, line pattern) that demonstrates the violation
> - `message`: one sentence explaining what's wrong and why it matters
>
> Focus on:
> 1. **Decision violations** — code that contradicts a key architectural decision
> 2. **Pattern erosion** — code that doesn't follow the patterns described in its folder's CLAUDE.md
> 3. **Trade-off undermining** — code that works against an accepted trade-off (check `violation_signals`)
> 4. **Pitfall triggers** — code that falls into a documented pitfall (check `stems_from` chains)
> 5. **Responsibility leaks** — a component doing work that belongs to another component
> 6. **Abstraction bypass** — code reaching through a layer instead of using the intended interface
> 7. **Semantic duplication** — functions/methods with different signatures but essentially the same logic. AI agents frequently copy-paste a function, tweak the name/parameters, and leave the body identical or near-identical. Look for: functions with similar names (e.g., `getText`/`getTexts`, `loadUser`/`fetchUser`), functions in different files that do the same thing with slightly different types, helper functions reimplemented instead of shared. For each, use type `semantic_duplication` and explain what's duplicated and which function should be the canonical one.
>
> Do NOT report: style/formatting/naming (the script handles those), generic best-practice violations not grounded in THIS project's blueprint, or issues already in the mechanical drift report.
>
> Return JSON: `{"deep_findings": [...]}`

Instruct the reviewer subagent to write its own output (append to its prompt):

```
---
OUTPUT CONTRACT (mandatory):
1. Use the Write tool to save your COMPLETE output to /tmp/archie_deep_drift.json
2. Write the raw output verbatim — extract_output.py handles JSON envelopes.
3. After Writing, reply with exactly: "Wrote /tmp/archie_deep_drift.json"
4. Do NOT print the output in your response body.
```

After the agent's confirmation returns, extract and clean up:

```bash
python3 .archie/extract_output.py deep-drift /tmp/archie_deep_drift.json "$PROJECT_ROOT/.archie/drift_report.json"
rm -f /tmp/archie_deep_drift.json
```

### Phase 3: Present the combined assessment

Read `$PROJECT_ROOT/.archie/blueprint.json` and `$PROJECT_ROOT/.archie/drift_report.json` (now contains both mechanical and deep findings). This is the final output — make it valuable.

#### Part 1: What was generated

List the generated artefacts with counts:
- Blueprint sections populated (out of total)
- Components discovered
- Enforcement rules generated
- Per-folder CLAUDE.md files created
- Rule files in `.claude/rules/`

#### Part 2: Architecture Summary

From the blueprint, summarize in 5-10 lines:
- **Architecture style** (from `meta.architecture_style`)
- **Key components** (top 5-7 from `components.components` — name + one-line responsibility)
- **Technology stack highlights** (from `technology.stack` — framework, language, key libs)
- **Key decisions** (from `decisions.key_decisions` — the 2-3 most impactful, one line each)

#### Part 3: Architecture Health Assessment

Rate and explain each dimension (use these exact labels: Strong / Adequate / Weak / Not assessed):

1. **Separation of concerns** — Are layers/modules clearly bounded? Do components have single responsibilities? Any god classes or circular dependencies?
2. **Dependency direction** — Do dependencies flow in one direction? Are domain/core layers independent of infrastructure? Any inverted or tangled dependencies?
3. **Pattern consistency** — Is the same pattern used consistently across similar components? Are there one-off deviations that break the uniformity?
4. **Testability** — Is the architecture conducive to testing? Can components be tested in isolation? Are external dependencies injectable?
5. **Change impact radius** — When a component changes, how many others are affected? Are changes localised or do they ripple?

Base every rating on actual evidence from the blueprint and drift findings — reference specific components, patterns, or findings. If the blueprint lacks data for a dimension, say "Not assessed" rather than guessing.

#### Part 4: Architectural Drift

Present ALL findings — mechanical and deep together, organized by severity (errors first).

**Deep architectural findings** (from AI analysis):
- For each: the file, which decision/pattern it violates, the evidence, and why it matters
- Group related findings (e.g., multiple files violating the same decision)

**Mechanical findings** (from script):
- Pattern divergences, dependency violations, naming violations, structural outliers, anti-pattern clusters
- For each: what diverged, why it matters, suggested action

If 0 findings, say so — that's a positive signal.

#### Part 5: Top Risks & Recommendations

Synthesize from pitfalls, trade-offs, drift findings (both mechanical and deep), and your observations. List the **3-5 most important architectural risks**, ordered by impact:
- What the risk is (one sentence)
- Where it manifests (specific components/files/drift findings)
- What to watch for going forward

#### Part 6: Semantic Duplication

**This is a critical section.** The mechanical verbosity score (0-1) only catches exact line-for-line clones. AI agents frequently create near-identical functions with slightly different names, signatures, or types — the verbosity metric completely misses these.

Present the `semantic_duplication` findings from the deep drift analysis. If the drift agent found none, **do your own quick check now**: scan the skeletons for functions with similar names (e.g., `getText`/`getTexts`, `loadUser`/`fetchUser`, `formatDate` in multiple files, `handleError` reimplemented per-module). Read suspicious pairs and confirm whether the logic is duplicated.

For each confirmed duplicate group:
- The canonical function (the one that should be the shared version)
- The duplicates: which files, what differs (just the signature? types? minor logic?)
- Whether they could be consolidated

Present in the health table as:
```
| Semantic duplication | N groups found | See Part 6 for details |
```

If genuinely none found after checking, say "No semantic duplication detected after AI analysis."

**Health scores** from Phase 0 have been saved to `.archie/health_history.json` for trending. Note: the verbosity metric is mechanical (exact line clones only) — the semantic duplication analysis in Part 6 above is the AI-powered complement. Run `/archie-scan` regularly to track how these metrics change over time.

### Phase 4: Persist findings to `.archie/scan_report.md`

The Phase 3 synthesis above is valuable but ephemeral — it only exists in the chat output. `/archie-share` (and future trending runs of `/archie-scan`) need the findings on disk. Write the same content to `.archie/scan_report.md` in the format `/archie-scan` produces.

Check whether a prior scan report exists (for resolved/new/recurring classification):
```bash
test -f "$PROJECT_ROOT/.archie/scan_report.md" && echo "PRIOR_REPORT_EXISTS" || echo "FIRST_BASELINE"
```

If `FIRST_BASELINE` (no prior scan_report.md): all findings are tagged **NEW (baseline)**. If `PRIOR_REPORT_EXISTS`: compare against the prior file's Findings section and classify each as **NEW**, **RECURRING**, or **RESOLVED**.

Read `$PROJECT_ROOT/.archie/health.json` for precise numeric values and `$PROJECT_ROOT/.archie/health_history.json` to compute trends (previous run values vs. current).

Write `$PROJECT_ROOT/.archie/scan_report.md` using the template at
`.claude/skills/archie-deep-scan/templates/scan-report.md` (path relative
to the project root). Read that file first if you haven't already, then
substitute the project-specific values into the placeholders before writing
the final report.
