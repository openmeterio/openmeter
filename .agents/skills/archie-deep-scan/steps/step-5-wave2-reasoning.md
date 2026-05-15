## Step 5: Wave 2 — Reasoning agent

**Telemetry:**
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan wave2_synthesis
python3 .archie/telemetry.py extra "$PROJECT_ROOT" wave2_synthesis model=opus
TELEMETRY_STEP5_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 5, skip this step.**

### Findings store (accumulates across all runs)

`.archie/findings.json` is a shared, compounding store — both `/archie-scan` and `/archie-deep-scan` read from it and write back to it. Each run adds new findings, upgrades existing ones (with matching `id`), confirms recurrence, or marks resolution. Scan and deep-scan are independent — neither requires the other; you can run either command any number of times in any order.

Before Wave 2:

- If `$PROJECT_ROOT/.archie/findings.json` exists, load the `findings` array and pass it to the Reasoning agent. It will upgrade the draft entries and emit additional findings it discovers.
- If the file is absent (first-ever run, or brand-new project), the Reasoning agent produces findings from scratch and Wave 2 writes the file as part of its output.

Either way, after Wave 2 the store reflects accumulated knowledge across every prior scan and deep-scan run.

**Maintainer guardrails — extract before Wave 2 (deterministic preprocessing):**

```bash
python3 .archie/intent_layer.py extract-guardrails "$PROJECT_ROOT"
```

This scans every per-folder `CLAUDE.md` (excluding `.archie/`, `.claude/`, `node_modules/`, etc.), strips Archie's own marker blocks (`<!-- archie:ai-* -->`, `<!-- archie:scoped-* -->`) so the loop reads only maintainer prose, extracts bullets under any `## Anti-Patterns` section, and writes `.archie/maintainer_guardrails.json`. Wave 2 §11 (compound learning) reads that file rather than globbing CLAUDE.md directly — the deterministic extraction guarantees no self-amplification across runs (Archie's previous output cannot feed back into itself, by construction). On the first deep-scan ever (no per-folder CLAUDE.md exist yet), the file is written as `{"version": 1, "guardrails": []}` — Wave 2 sees an empty guardrails array and §11 is a no-op for that run.

### If SCAN_MODE = "incremental":

Spawn an **Opus subagent** (`model: "opus"`) with scoped context:
- The existing `$PROJECT_ROOT/.archie/blueprint.json` (full current architecture)
- The patched `$PROJECT_ROOT/.archie/blueprint_raw.json` (with incremental changes from Step 4)
- The current `$PROJECT_ROOT/.archie/findings.json` if it exists (the accumulated findings store)
- The changed file contents (from `changed_files` list)

Tell the scoped Reasoning agent:

> The architecture was previously analyzed (blueprint.json attached). The blueprint_raw.json was updated with incremental structural changes. If findings.json is present, it carries the accumulated findings from prior scan and deep-scan runs. These specific files changed: [list changed_files]. Review the changes and update ONLY the affected sections:
> - If changes affect a key decision, update it
> - If changes introduce a new trade-off or invalidate one, update trade_offs
> - If changes trigger, resolve, or modify findings/pitfalls in the changed areas, update them
> - Update the decision_chain only for affected branches
> Return ONLY the sections that need updating — unchanged sections will be preserved. Use the 4-field contract (`problem_statement`, `evidence`, `root_cause`, `fix_direction`) when writing finding or pitfall entries.

Instruct the Reasoning agent to write its own output (append to its prompt):

```
---
OUTPUT CONTRACT (mandatory):
1. Use the Write tool to save your COMPLETE output to /tmp/archie_sub_x_$PROJECT_NAME.json
2. Write the raw output verbatim — merge handles JSON envelopes.
3. After Writing, reply with exactly: "Wrote /tmp/archie_sub_x_$PROJECT_NAME.json"
4. Do NOT print the output in your response body.
```

The file will be on disk when the agent's confirmation returns. Then finalize with patch mode:
```bash
python3 .archie/finalize.py "$PROJECT_ROOT" --patch /tmp/archie_sub_x_$PROJECT_NAME.json
```
```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 5
```

Then skip to Step 6.

### If SCAN_MODE = "full" (default):

Wave 1 gathered facts: components, patterns, technology, deployment, UI layer. Now spawn a single Opus subagent (`model: "opus"`) that reads ALL Wave 1 output and produces deep architectural reasoning.

Tell the Reasoning agent:

> Read `$PROJECT_ROOT/.archie/blueprint_raw.json` — it contains the full analysis from Wave 1 agents: components, communication patterns, technology stack, deployment, frontend. It may also carry a top-level `findings` array holding **draft findings from Wave 1 agents** (for example, the Structure agent's workspace-level observations — cross-workspace cycles, monorepo constraint violations). Pick those drafts up and include them in your own findings output, upgrading them to canonical (fill `root_cause` and `fix_direction`, keep their `problem_statement`/`evidence`/`applies_to`, set `depth: "canonical"`, `source: "deep:synthesis"`). Also read `$PROJECT_ROOT/.archie/findings.json` **if it exists** — it is the accumulated findings store across every prior scan and deep-scan run, each entry shaped as `{id, problem_statement, evidence, root_cause, fix_direction, depth, source, ...}`. If the file is absent, proceed without it and produce findings from scratch. Also read key source files: entry points, main configs, core abstractions.
>
> With the COMPLETE picture of what was built and how, produce deep architectural reasoning. You will upgrade any draft findings in the accumulated store, emit new findings you discover, AND emit pitfalls (classes of problem rooted in architectural decisions). Both findings and pitfalls share the same 4-field core (`problem_statement`, `evidence`, `root_cause`, `fix_direction`); pitfalls differ in altitude (class-of-problem, not instance) and ownership (blueprint-durable, not per-run).
>
> ### 1. Decision Chain
> Trace the root constraint(s) that shaped this architecture. Build a dependency tree:
> - What is the ROOT constraint? (e.g., "local-first tool requiring filesystem access")
> - What does it FORCE? (each forced decision)
> - What does EACH forced decision FORCE in turn?
> - Continue until you reach leaf decisions
> - For EACH node, include `violation_keywords`: specific code patterns or package names that would violate this decision (e.g., for "SQLite only" → `["pg", "mongoose", "prisma", "typeorm", "postgres"]`)
>
> Every decision in the chain must be grounded in code you can see in the blueprint or source files. Do NOT invent theoretical constraints.
>
> ### 2. Architectural Style Decision
> THE top-level architecture choice. You can see the full component list, pattern list, and tech stack — explain WHY this architecture, not just WHAT. Reference specific components and patterns from the blueprint.
> - **title**: e.g., "Full-stack monolith with subprocess orchestration"
> - **chosen**: What was chosen and how it manifests
> - **rationale**: WHY — reference specific components, patterns, and tech stack items from the blueprint
> - **alternatives_rejected**: What alternatives were NOT chosen and WHY they were ruled out by the constraints
>
> ### 3. Key Decisions (3-7)
>
> Before writing key decisions, run these three probes across the codebase. They surface load-bearing decisions that are often invisible to pure structural analysis (components, layers, tech stack). Each probe is **shape-based and framework-neutral** — it asks about *how* the code commits, not about any specific pattern. A probe can produce 0-N decisions; only emit what is actually present. Name each commitment in the **codebase's own vocabulary** (exact class/module/file names), not generic restatements.
>
> **Probe A — Complexity-budget:** identify every place where this codebase spends meaningful complexity on something a naive implementation of the same product wouldn't need. For each:
> - Name the commitment (in the codebase's own terms).
> - Sketch the naive alternative in one sentence.
> - What does this choice **enable** (capability unlocked)?
> - What does it **foreclose** (path closed off)?
> - Point at the specific files/modules where the commitment is implemented. If you can't, drop it.
>
> **Probe B — Invariants & gates:** locate every rule the codebase enforces on itself — anything that constrains, sequences, authorizes, versions, rate-limits, or otherwise gates other operations. For each: state the invariant in plain language, identify whether it is enforced at a single seam or scattered, and name what violates it (or where violation would slip through).
>
> **Probe C — Seams:** locate every place designed for substitution or extension — abstract interfaces with multiple concrete implementations, registry- or config-driven dispatch, protocol boundaries, plugin surfaces, hook/callback systems. For each: what **varies** across the seam, what is held **stable**, and what is the **mechanism** for adding a new implementation.
>
> **Working from Wave 1 output.** Wave 1 has already read the codebase (skeletons and, where needed, source). Run each probe primarily against `blueprint_raw.json` — specifically the `communication`, `components`, and `technology` sections, plus any raw agent output captured there. The probes are synthesis questions over Wave 1's data, not a fresh read pass.
>
> Only read source directly when Wave 1's output is genuinely insufficient to answer a probe (e.g., Wave 1 named a seam but didn't record what varies across it, or flagged a gate without naming the invariant). Judge file-by-file — no blanket re-read. If a signal clearly exists in the codebase but Wave 1's data is thin, prefer to record that as a gap in `pitfalls` (so the next scan catches more) over re-doing Wave 1's job here.
>
> **Emitting decisions.** Consolidate what the probes surface into `key_decisions` (target 3-7). Each with: title, chosen, rationale, alternatives_rejected.
> - **rationale** must reference specific components, patterns, or tech from the blueprint AND cite the concrete file/module that implements the commitment.
> - **forced_by**: what constraint or other decision made this one necessary
> - **enables**: what this decision makes possible downstream
>
> A codebase with few commitments (template apps, naive CRUD) may genuinely have only 2-3 meaningful decisions — do not invent filler. A codebase heavy in protocols, gates, and seams will have more than 7 candidates; in that case keep the most load-bearing and note the rest in `trade_offs` or `pitfalls` as appropriate.
>
> ### 4. Trade-offs (3-5)
> Each with: accept, benefit, caused_by (which decision created this trade-off), violation_signals (code patterns that would indicate someone is undoing this trade-off, e.g., removing Puppeteer → `["uninstall puppeteer", "remove puppeteer", "playwright"]`)
>
> ### 5. Out-of-Scope
> What this codebase does NOT do. For each item, optionally note which decision makes it out of scope.
>
> ### 6. Findings (primary: new; secondary: upgrade existing)
> Findings describe **instances**: concrete problems observed in specific files. **Pitfalls** describe **classes**: architectural traps with no current call-site instance (yet) but rooted in a decision/pattern that makes them likely. The two streams are not interchangeable — mis-filing a class-of-problem as a finding tells a maintainer to "fix this now" when there's nothing to fix, eroding trust in the report.
>
> **REQUIRED — triggering_call_site (new field, blocks emission as a finding).** Every finding MUST carry a `triggering_call_site` string: a verbatim code quote at `<file>:<line>` showing **a real caller in the corpus that actually triggers the failure mode under current code**. Not a function whose signature *could* trigger it — a caller whose actual argument or surrounding context demonstrates the problem firing right now. If you cannot quote such a call site (because the cited invariant is universally enforced, the suspect helper is only ever called via a tx-bound adapter, the missing wrapper exists at every call site, etc.), the entry is a **risk class**, not a current problem.
>
> Risk classes go into `pitfalls` (forward-looking, durable in the blueprint), NOT `findings` (active, fix-this-now in `findings.json`). Make the choice **explicitly at emission time**: ask *"can I quote a verbatim caller in this corpus that fires the failure mode?"* — yes ⇒ finding; no ⇒ pitfall. The viewer renders findings under "Architectural Problems" (treated as a triage queue) and pitfalls under "Pitfalls" (forward-looking guardrails) — mis-filing degrades both signals.
>
> The `f_0001` shape we are guarding against: AI sees an AGENTS.md mandate, finds two helpers with the suspect signature, lists a fallback mechanism that *would* trigger silent failure if the helpers were misused — but never walks one level out to verify whether any caller actually misuses them. Every fact is true; the conclusion is unverified. The triggering-call-site rule forces that verification step at synthesis time.
>
> **APPROACH — anchor synthesis to documented invariants.** Instead of speculatively asking *"what could go wrong?"*, walk the documented invariants in AGENTS.md, root `CLAUDE.md`, per-folder `CLAUDE.md` (Anti-Patterns and Patterns sections — `.archie/maintainer_guardrails.json` if available), and `blueprint.pitfalls`. For each invariant, ask: *"is there code in this corpus that violates it? Quote it verbatim."* If yes ⇒ that quote is the `triggering_call_site` of a finding. If the invariant is real but uniformly enforced ⇒ no finding (and no need to re-emit a pitfall already in the store). This adversarial framing converts the loose "find problems" task into a falsifiable evidence-gathering pass.
>
> **Primary goal — emit NEW findings.** You have the overall picture (all Wave 1 output plus source files). Your highest-leverage work is surfacing problems that are NOT already in findings.json — things only visible from the whole-system view: cross-component coupling, pattern breakdowns that individual agents miss, constraint violations implied by the decision chain, gaps between what the blueprint claims and what the code does. Spend the bulk of your cognitive budget here. For each new finding: next-free `f_NNNN` id, `first_seen` = today, `confirmed_in_scan` = 1, `depth: "canonical"`, `source: "deep:synthesis"`, AND a non-empty `triggering_call_site`.
>
> **Novelty check before emitting.** Before you add a "new" finding, verify it is genuinely new: scan the existing store for any entry with overlapping `problem_statement` meaning OR overlapping `applies_to` files. If the same problem is already tracked under a different wording, DO NOT mint a new id — instead upgrade the existing entry (see below). A new finding must describe something the store doesn't already cover.
>
> **Secondary goal — upgrade existing drafts.** If findings.json has entries (especially `depth: "draft"` from scan:triage), upgrade them: preserve `id`, `first_seen`, `applies_to`, `evidence` (you may append new evidence). Rewrite `root_cause` with architectural grounding (name the decision, pattern, or constraint — not generic explanation) and rewrite `fix_direction` as an **ordered list of sequenced steps** referencing specific components and file paths. Set `depth: "canonical"`, `source: "deep:synthesis"`, increment `confirmed_in_scan` by 1. Upgrading is housekeeping — quick, targeted; don't re-derive evidence from scratch.
>
> Quality bar (both new and upgraded): `problem_statement` specific, `evidence` with concrete references, `root_cause` architecturally grounded, `fix_direction` sequenced and actionable. Soft floor of 3 total findings in the updated store; if fewer meet the bar, say so explicitly in your output. If the store is already comprehensive and you cannot find anything new, say "no new findings emerged from the overall-picture pass" in your output rather than restating existing ones under different ids.
>
> ### 7. Pitfalls
> Pitfalls describe **classes** of problem — architectural traps rooted in decisions or patterns, covering both current manifestations and latent risks. They are durable blueprint entries, not per-run observations. Each pitfall uses the same 4-field core as findings:
> - `id` (`pf_NNNN`, stable across runs — reuse existing ids when upgrading)
> - `problem_statement` — one sentence describing the class of problem
> - `evidence` — list of observations (cite architectural decisions, pattern recurrences across multiple findings, component absences)
> - `root_cause` — the decision/pattern/constraint making this class of problem likely
> - `fix_direction` — **ordered list** of strategic steps (migration order, seam to introduce, rule to establish)
> - `applies_to` — component/folder paths (broader than finding-level file paths)
> - `severity`, `confidence`, `source: "deep:synthesis"`, `depth: "canonical"`, `first_seen`, `confirmed_in_scan`
>
> Where a finding's `root_cause` is structural/recurring, also emit a corresponding pitfall and set the finding's `pitfall_id` to it. A single pitfall may have multiple confirming findings.
>
> **Novelty check for pitfalls.** If the blueprint already contains pitfalls, reuse their `id`s when you upgrade them (preserve `first_seen`, bump `confirmed_in_scan`). Before minting a new `pf_NNNN`, verify no existing pitfall covers the same class of problem — the store is durable across runs, and a pitfall that re-emerges under a new id loses its history. Spend your cognitive budget surfacing NEW classes of problem (architectural traps visible only from the whole-system view) rather than restating existing ones.
>
> Quality bar: only emit pitfalls whose `root_cause` traces to something visible in the blueprint (decision, pattern, component absence). Soft floor of 3; if fewer meet the bar, say so. If no new pitfalls emerged, say so explicitly rather than duplicating existing ones under new wording.
>
> Only describe problems grounded in actual code and observed decisions. Do NOT recommend alternatives the code doesn't use.
>
> ### 8. Architecture Diagram
> Mermaid `graph TD` with 8-12 nodes. You have the full component list and communication patterns from the blueprint — use actual component names and real data flows.
>
> ### 9. Implementation Guidelines (5-8)
> Capabilities using third-party libraries. Cross-reference the tech stack and pattern list from the blueprint. For each:
> - **capability**: Human-readable name
> - **category**: auth | notifications | media | storage | networking | analytics | persistence | ui | payments | location | state_management | navigation | testing
> - **libraries**: Libraries used with versions (from tech stack)
> - **pattern_description**: Architecture pattern, main service/class, data flow
> - **key_files**: Actual file paths (MUST exist in file_tree)
> - **usage_example**: Realistic code snippet that a developer would actually write. **Multi-line is the default**: use **real `\n` newlines** in the JSON string. Reserve a one-liner ONLY for patterns that are *genuinely* one-line — a single function call like `logger.track(Event.X)`, a single annotation, a single import. **Anything with multiple statements, control flow, multiple parameters past the first, a `;` separator, an inline `// comment`, or that exceeds ~80 characters MUST be multi-line.** A one-liner crammed with `;` chains, mid-line `// inline comment`, or 3+ chained statements is wrong even if it parses — split it across lines as if writing real code in an editor. The renderer (`<pre><code>` block) preserves newlines correctly, so a 10-line example will render as 10 lines, not as a 200-character horizontal scroll.
> - **applicable_when**: The verifiable invariant that makes this pattern correct in THIS codebase. MUST cite a concrete code artifact at `<file>:<line>` — pick whichever invariant shape fits the language and paradigm: schema annotation (unique/foreign-key/NOT-NULL/index), type signature (Result/Option/exhaustive enum/generic bound), lifecycle state (hook under a Provider, handler registered before bus start), ownership/concurrency (borrow scope, lock-held interval, transaction-active context), or structural contract (single registration point + iterating consumer, sealed hierarchy + exhaustive match). The requirement is that the citation is **falsifiable against the corpus**, not its invariant shape. NOT prose. "Per-customer operations" / "in the auth flow" / "during request handling" are NOT preconditions — they pattern-match across domains where the invariant doesn't hold. If the capability has no codebase-specific invariant (e.g. generic logging, generic HTTP middleware), leave empty (`""`). **REQUIRED SHAPE — category-then-evidence:** lead with a categorical noun phrase naming the **class of callers/situations** the invariant guards (substitutable across components — if the predicate only makes sense about ONE specific file, you wrote trivia), then back it with the `<file>:<line>` citation. *BAD:* `"BabyWeatherAnalyticsManager uses internal AtomicBoolean singleton and its own initialize() — exposed through analyticsModule but not constructed via Koin injection."` (one-file fact, no category to match against). *GOOD:* `"Component manages its own initialization lifecycle (manual initialize() outside DI; Koin only exposes the already-built instance) — BabyWeatherAnalyticsManager (analyticsModule), LocalisationHelper (DomainModules.kt:18-21)."` (named class of cases + citations as evidence).
> - **do_not_apply_when**: Array of concrete anti-indicators (each citable against code) and each following the same **category-then-evidence** shape as `applicable_when`: a categorical noun phrase + citation, never a per-file description. You have the cross-cutting view — when this capability's shape (lock key, registry key, validator key, hook usage, lifetime contract) WOULD be wrong elsewhere in the corpus, name those classes of cases (with citations as evidence). Empty array if the pattern is universally safe.
> - **scope**: Array of identifiers naming where this pattern is RELEVANT WHEN EDITING — producers, consumers, boundary participants. Each identifier may be **(a) a component name from `components.components[].name`**, OR **(b) a concrete code symbol** — class, interface, object, enum, or Koin `val Foo = module {}` declaration — that the resolver can map back to a component via the file it's declared in. Concrete code symbols are often more useful at edit time (a developer recognises `NetworkDatasourceImpl` faster than `Domain and Data Layer`); both forms are accepted, mix freely. Avoid prose like `"All Fragments under page_*"` — the resolver cannot map prose, and unresolved values are dropped silently. In per-package mode, leave empty `[]` (the package boundary is the scope). Empty array means "applies repo-wide". Conservative default: leave empty unless there is verifiable evidence the pattern is component-bound (no regression vs. today).
>
>     **Threshold rule — use `scope: []` for near-universal patterns.** If a pattern would resolve to **at least half** of the blueprint's components (e.g. MVVM convention used in every Fragment+ViewModel pair, a tracing decorator on every ViewModel, a Repository pattern used by every data layer consumer), set `scope: []` and let it live repo-wide. Enumerating consumers in such cases just duplicates the same content into every per-folder `CLAUDE.md`. The renderer treats fan-outs ≥50% as repo-wide and skips per-folder injection regardless, so over-enumeration produces no extra signal — only the global rule file `guidelines.md` / `patterns.md` (loaded for every edit) carries it. Reserve scope enumeration for patterns that genuinely cluster in a minority of components.
> - **tips**: Gotchas specific to this implementation
>
> ### 10. Communication patterns enrichment (cross-cutting)
>
> Wave 1's Patterns agent produced `communication.patterns` with `applicable_when`, `do_not_apply_when`, and `scope`. Re-emit that array with `do_not_apply_when` enriched from your cross-corpus view: when a pattern's shape is used somewhere in the corpus WITHOUT the invariant holding, name those places. For each pattern, also verify `scope` is **relevance-based, not location-based** — list every component that interacts with the pattern at edit-time (producers, consumers, transactional-boundary participants), not just where the source file lives. If you have nothing to add over Wave 1, copy the array through verbatim.
>
> ### 11. Compound learning — fold maintainer-curated anti-patterns into `do_not_apply_when`
>
> Maintainers sometimes hand-edit per-folder `CLAUDE.md` files with anti-pattern guardrails — e.g. *"No <pattern> here — <local invariant fact contradicts it>."* These are gold for `do_not_apply_when` because someone who knows the codebase has already condensed the invariant into prose.
>
> **Read the deterministic extractor's output, not the raw CLAUDE.md files.** Before this step, the deep-scan pipeline runs `intent_layer.py extract-guardrails` (see Step 7), which strips Archie's own marker blocks (`<!-- archie:ai-* -->`, `<!-- archie:scoped-* -->`) and writes the cleaned bullets to `.archie/maintainer_guardrails.json`. **Read that file** — do NOT glob `CLAUDE.md` directly. The extractor guarantees only maintainer prose is in scope; the JSON shape is `{guardrails: [{source: "<rel-path>/CLAUDE.md", items: [text, text, ...]}]}`.
>
> For each bullet, fuzzy-match its pattern name to your `implementation_guidelines[].capability` or `communication.patterns[].name`. If a clean match exists, include the bullet's reasoning (with citation `(see <source>)`) in the matched entry's `do_not_apply_when` array. **Do NOT invent matches** — if no clean match exists, skip it.
>
> **Regeneration semantics — emit the FULL `do_not_apply_when` array each run, not a delta.** The array you write is the complete list for that pattern, comprising:
>
> 1. Anti-indicators you derived from the corpus this run (the inverse of `applicable_when`, plus call-sites where the same shape would be wrong).
> 2. Maintainer guardrails currently present in `.archie/maintainer_guardrails.json` that fuzzy-match this pattern.
>
> Do **not** preserve previous-blueprint entries that no longer have a current source. The previous blueprint's `do_not_apply_when` is informational only — entries from prior runs that are no longer corpus-derived AND no longer in the extractor's output have aged out and must drop. This prevents the array from growing monotonically across runs even when the underlying violation has been fixed.
>
> Return JSON:
> ```json
> {
>   "decisions": {
>     "architectural_style": {"title": "", "chosen": "", "rationale": "", "alternatives_rejected": []},
>     "key_decisions": [{"title": "", "chosen": "", "rationale": "", "alternatives_rejected": [], "forced_by": "", "enables": ""}],
>     "trade_offs": [{"accept": "", "benefit": "", "caused_by": "", "violation_signals": []}],
>     "out_of_scope": [],
>     "decision_chain": {"root": "", "forces": [{"decision": "", "rationale": "", "violation_keywords": [], "forces": []}]}
>   },
>   "findings": [
>     {
>       "id": "f_NNNN",
>       "problem_statement": "",
>       "evidence": [],
>       "triggering_call_site": "<rel/path/to/file.ext>:<line>\\n<verbatim code quote of the caller that fires the failure mode here>",
>       "root_cause": "",
>       "fix_direction": ["step 1", "step 2", "step 3"],
>       "severity": "error|warn|info",
>       "confidence": 0.9,
>       "applies_to": [],
>       "source": "deep:synthesis",
>       "depth": "canonical",
>       "pitfall_id": "pf_NNNN (optional)",
>       "first_seen": "YYYY-MM-DDTHHMM",
>       "confirmed_in_scan": 1,
>       "status": "active"
>     }
>   ],
>   "pitfalls": [
>     {
>       "id": "pf_NNNN",
>       "problem_statement": "",
>       "evidence": [],
>       "root_cause": "",
>       "fix_direction": ["step 1", "step 2", "step 3"],
>       "severity": "error|warn",
>       "confidence": 0.9,
>       "applies_to": [],
>       "source": "deep:synthesis",
>       "depth": "canonical",
>       "first_seen": "YYYY-MM-DD",
>       "confirmed_in_scan": 1
>     }
>   ],
>   "architecture_diagram": "graph TD\n  A[...] --> B[...]",
>   "implementation_guidelines": [
>     {"capability": "", "category": "", "libraries": [], "pattern_description": "", "key_files": [], "usage_example": "", "applicable_when": "", "do_not_apply_when": [], "scope": [], "tips": []}
>   ],
>   "communication": {
>     "patterns": [
>       {"name": "", "when_to_use": "", "how_it_works": "", "examples": [], "applicable_when": "", "do_not_apply_when": [], "scope": []}
>     ]
>   }
> }
> ```

The Reasoning agent also gets the GROUNDING RULES from Step 3.

Instruct the Reasoning agent to write its own output (append to its prompt):

```
---
OUTPUT CONTRACT (mandatory):
1. Use the Write tool to save your COMPLETE output to /tmp/archie_sub_x_$PROJECT_NAME.json
2. Write the raw output verbatim — finalize handles JSON envelopes.
3. After Writing, reply with exactly: "Wrote /tmp/archie_sub_x_$PROJECT_NAME.json"
4. Do NOT print the output in your response body.
```

After the agent's confirmation returns, finalize:

```bash
python3 .archie/finalize.py "$PROJECT_ROOT" /tmp/archie_sub_x_$PROJECT_NAME.json
```

This single command: merges the Reasoning agent's output into the blueprint, normalizes the schema, renders CLAUDE.md + AGENTS.md + rule files, installs hooks, and validates. Review the validation output — warnings are informational, not blocking.

**Backward-check the findings against actual code.** After finalize writes `.archie/findings.json`, run the Haiku verifier and apply hysteresis. The verifier reads each finding's required `triggering_call_site` field, walks one level out from the cited caller, and decides per finding: `keep` (failure fires there — real finding), `demote` (call site exists but failure doesn't fire — risk class, not current problem), or `drop` (premise unsound for this codebase). The hysteresis layer then applies the verdict with cross-run stability — single-scan flips on unchanged code don't propagate (kills LLM-noise flicker), but a git-diff anchor (a file in the finding's `triggering_call_site` was touched in the last 5 commits) lets a real transition land immediately.

```bash
python3 .archie/verify_findings.py "$PROJECT_ROOT"
python3 .archie/apply_verdicts.py "$PROJECT_ROOT"
```

Both scripts are idempotent and graceful: if the claude CLI is unreachable, every Haiku call times out, or `findings.json` is empty, both no-op cleanly. Findings whose status flips to `demoted` or `dropped` here will be filtered out of any user-facing rendering automatically (status-driven filter) — only `status: active` reaches the report.

After finalize completes, regenerate the dependency graph (the blueprint now has component definitions, which enables cross-component edge detection):

```bash
python3 .archie/detect_cycles.py "$PROJECT_ROOT" --full 2>/dev/null
```

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 5
```

