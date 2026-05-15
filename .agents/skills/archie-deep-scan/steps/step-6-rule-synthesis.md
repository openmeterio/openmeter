## Step 6: AI Rule Synthesis

**Telemetry:**
```bash
python3 .archie/telemetry.py mark "$PROJECT_ROOT" deep-scan rule_synthesis
python3 .archie/telemetry.py extra "$PROJECT_ROOT" rule_synthesis model=sonnet
TELEMETRY_STEP6_START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**If START_STEP > 6, skip this step.**

### If SCAN_MODE = "incremental":

The blueprint was patched in Step 5. Spawn a **Sonnet subagent** (`model: "sonnet"`) with this additional instruction prepended to the standard prompt below:

> The existing rules are in `.archie/rules.json`. Only propose rules for patterns discovered in the changed files. Do not regenerate existing rules. If a change invalidates an existing rule, flag it with `"status": "invalidated"` in the output.

### All modes (full and incremental):

The blueprint contains architectural facts. This step synthesizes them into **architectural rules** — insights that the AI reviewer uses to evaluate plans and code changes.

Spawn a **Sonnet subagent** (`model: "sonnet"`) with this prompt:

> Read `$PROJECT_ROOT/.archie/blueprint.json` ONCE (do not re-read it). It contains the full architecture: components, decisions (with decision chains and violation keywords), patterns, trade-offs (with violation signals), pitfalls (with causal chains), technology stack, development_rules, infrastructure_rules, and architecture_rules (file_placement_rules + naming_conventions).
>
> **You are the SOLE producer of `proposed_rules.json`.** Every rule shape the user can adopt/reject/edit through the viewer's Rules section originates from this synthesis. The other deep-scan agents (structure, technology, patterns, reasoning) write architectural FACTS into the blueprint — your job is to turn those facts into agent-facing enforcement rules in the unified schema.
>
> Produce 30-60 architectural rules. Each rule captures an enforcement intent a coding agent must respect when planning or making changes. Coverage MUST span every blueprint section that carries enforcement signal — not just decisions and pitfalls. See "Coverage" below.
>
> **Primary enforcement is AI-powered:** the AI reviewer reads each rule's `why` and `example` on every plan approval and pre-commit, and evaluates whether changes violate the rule's *intent*. The hook also surfaces these inline at edit time when the rule applies, so the agent sees the canonical reasoning + example without any extra lookup.
>
> **Secondary enforcement is mechanical (optional):** if a rule can also be expressed as a regex, add `check` + `forbidden_patterns`/`required_in_content` fields so the pre-edit hook catches obvious violations instantly. Most rules won't have this — that's fine. Don't force regex where it doesn't fit.
>
> Return ONLY valid JSON: `{"rules": [...]}`.
>
> ## Rule schema (Phase 1 inline shape)
>
> **Required fields** (every rule):
> ```json
> {
>   "id": "dep-001",
>   "kind": "decision",
>   "topic": "layering",
>   "severity_class": "decision_violation",
>   "description": "What is forbidden/required (one imperative sentence — see Description shape below)",
>   "why": "Inlined reasoning copied from the blueprint section that motivates this rule (decision text, pitfall description, tradeoff signal, or pattern rationale). 2-4 sentences. The agent sees this verbatim.",
>   "example": "Inlined canonical code from implementation_guidelines.usage_example when applicable. Empty string if no code example fits.",
>   "source": "deep_scan"
> }
> ```
>
> **Recommended fields** (every rule that has a motivating decision or pitfall — these add the depth that makes a rule useful at edit-time, beyond the one-line directive):
> ```json
> {
>   "forced_by": "The constraint that drove this decision — one sentence. Copy from decisions.key_decisions[*].forced_by, or paraphrase the root constraint from the decision_chain when the rule is pitfall-driven. Empty string if not applicable (mechanical rules, naming conventions).",
>   "enables": "What capability this preserves — one sentence. Copy from decisions.key_decisions[*].enables. Empty string if not applicable.",
>   "alternative": "What to do instead — one imperative sentence pointing the agent at the project's correct path (e.g. 'Add a new Koin module under page_<feature>/ModulesX.kt and load it in BabyWeatherApplication.startKoin'). Empty string when the rule is purely 'do not X' with no alternative path."
> }
> ```
>
> The pre-validate hook surfaces all of these to the agent at edit time, in this order: `description`, `why`, `forced_by`, `enables`, `alternative`, `example`. Together they answer: *what to do*, *why this exists*, *what constraint forced it*, *what we'd lose without it*, *what to do instead*, *what the right shape looks like*. Skip a field by emitting `""` — the hook elides empty strings.
>
> ## Description shape (the title the user sees)
>
> The `description` is the rule's *title*. It must be one short imperative sentence. **Do not** cram version pins, configuration flags, or project-name references into it — those belong in `why`. The agent gets `description` first; rationale and version detail live two lines below.
>
> Bad (rationale crammed into title):
> > *"Do not introduce Hilt, Dagger, or any DI framework other than Koin — Koin 3.1.3 with allowOverride(true) is the sole DI container."*
>
> Good (directive only; version + configuration moved to `why`):
> > *"Dependency injection must go through Koin — do not introduce Hilt, Dagger, or other DI frameworks."*
> > `why`: "Koin 3.1.3 is configured project-wide via `startKoin { allowOverride(true); modules(...) }` in `BabyWeatherApplication`, deliberately enabling test-time module overrides..."
> > `forced_by`: "Test-time module override capability required by the test harness."
> > `enables`: "Per-test Koin module substitution without instrumentation; fast unit tests for ViewModels and Repositories."
> > `alternative`: "Add new bindings as a Koin `module { single { ... } }` block under the relevant `page_<feature>/ModulesX.kt` and load it in `BabyWeatherApplication.startKoin`."
>
> **The `kind` field** — names the conceptual *type* of rule, independent of which blueprint section motivated it. The viewer groups rules in the UI by kind so a user curating proposed rules can scan all file-placement rules together, all naming conventions together, etc. Pick exactly one:
>
> - `decision` — clarifies a `decisions.key_decisions[*]` invariant. Pair with `severity_class: "decision_violation"`.
> - `pitfall` — guards against a `pitfalls[*]` causal chain. Pair with `severity_class: "pitfall_triggered"`.
> - `semantic_pattern` — captures a `components.patterns` or `implementation_guidelines` shape. Pair with `severity_class: "pattern_divergence"` (or `tradeoff_undermined` when it formalizes a `trade_offs[*].violation_signals`).
> - `file_placement` — derived from `blueprint.architecture_rules.file_placement_rules` (or directly observed): which kind of file belongs under which directory. Pair with `severity_class: "pattern_divergence"` by default; bump to `mechanical_violation` when a regex is reliable.
> - `naming_convention` — derived from `blueprint.architecture_rules.naming_conventions`: which file/identifier names must follow which pattern. Pair with `severity_class: "mechanical_violation"` when expressible as a file-basename regex.
> - `coding_practice` — derived from `blueprint.development_rules` (or `infrastructure_rules` when build/CI-flavored): general guidance the agent should remember at edit time. Pair with `severity_class: "pattern_divergence"` (informational).
>
> `kind` and `severity_class` are not redundant: `kind` is *what the rule is about* (UI grouping), `severity_class` is *how the hook responds* (enforcement behavior).
>
> **The `topic` field** — a short kebab-case slug naming the conceptual area the rule governs. The renderer groups rules by topic into per-file markdown topic pages under `.claude/rules/enforcement/by-topic/<topic>.md`, so an agent can load only the topic relevant to the current task instead of the full enforcement set.
>
> Prefer one of these recommended cross-platform topics:
>
> - `data-access` — fetching, persisting, caching, ORMs, network
> - `concurrency` — async/reactive primitives, threads, schedulers
> - `ui` — view layer, components, styling, layout
> - `navigation` — routing, deep links, screen transitions
> - `layering` — file placement, dependency direction, layer rules
> - `services` — singletons, DI, cross-cutting service patterns
> - `state-management` — global state, stores, reactive sources
> - `dependencies` — package managers, build, secrets handling
> - `security` — auth, secrets, GDPR/PII, crypto
> - `testing` — test harness, fixtures, anti-patterns
> - `resources` — assets, i18n, localized strings
> - `error-handling` — error propagation, fallbacks, retries
>
> You MAY introduce a project-specific topic when a coherent group of 3 or more rules clearly belongs together under a name not in the list (examples: `mapping`, `payments`, `auth`, `realtime`, `migrations`, `accessibility`). Use a kebab-case slug.
>
> **Severity classes** — pick exactly one based on which blueprint section motivates the rule:
> - `decision_violation` — rule clarifies a `decisions.key_decisions[*]` invariant. Hook **blocks** (exit 2) on violation.
> - `pitfall_triggered` — rule guards against a `pitfalls[*]` trap. Hook **blocks** (exit 2).
> - `tradeoff_undermined` — rule formalizes a `decisions.trade_offs[*].violation_signals` signal. Hook **warns** (exit 0, prominent).
> - `pattern_divergence` — rule captures a `components.patterns` or `implementation_guidelines` style. Hook **informs** (exit 0, quiet).
> - `mechanical_violation` — rule is regex-checkable housekeeping (don't-edit-generated, file-naming-regex). Hook **blocks** (exit 2). Use this with the optional mechanical fields below.
>
> **The `why` field is the most important field.** Inline 2-4 sentences directly from the blueprint section that motivated the rule — do NOT paraphrase or summarize, copy the language Sonnet wrote in Wave 2 verbatim or near-verbatim. The agent reads this at edit time as the rejection/warning explanation, so the inlined text *is* the agent's understanding of why the rule exists. Examples (good):
> - "We chose SQLite for the local-first constraint. Introducing any ORM or remote database would undermine the zero-config deployment model and force connection management the architecture doesn't support."
> - "ViewModels must stay framework-agnostic because the decision chain roots in testability — if a ViewModel references Android Context, it can't be unit-tested without instrumentation, which breaks the fast-feedback development loop."
>
> **The `example` field** carries the canonical code shape, copied from `implementation_guidelines.usage_example` when present. If the rule is structural (a layering or pattern rule with a Wave-2 example), copy that example verbatim. If the rule is purely about *what NOT to do* (forbidden imports, mechanical no-edit) and there's no positive example, leave `example` as an empty string.
>
> **`source` field** — always emit `"source": "deep_scan"` for rules produced in this step. The post-process step also stamps it defensively if you forget.
>
> **Prompt-time matching** (RECOMMENDED for every rule):
> - `"keywords"`: 2-5 terms an AI would use when describing a task this rule governs (e.g. `["datetime", "timestamp"]`, `["migration"]`, `["handler", "endpoint"]`). The `UserPromptSubmit` hook matches these against the user's prompt and surfaces the rule BEFORE the agent writes code. Without keywords the hook falls back to noisy description-token extraction — always emit keywords.
>
> **Optional mechanical fields** (add ONLY when a meaningful regex exists; pair with `severity_class: "mechanical_violation"`):
> - `"check"`: one of `forbidden_import`, `required_pattern`, `forbidden_content`, `architectural_constraint`, `file_naming`
> - `"applies_to"`: directory prefix (string-prefix match, NOT a glob) — e.g. `"backend/"`
> - `"file_pattern"`: glob on basename for `required_pattern` / `architectural_constraint`, OR regex on basename for `file_naming`
> - `"forbidden_patterns"`: array of regexes; rule fires if any matches the content
> - `"required_in_content"`: array of literal strings; rule fires if NONE appears in the content
>
> When `check` is present:
> - `forbidden_import`: requires `applies_to` + `forbidden_patterns`
> - `required_pattern`: requires `file_pattern` (glob) + `required_in_content`
> - `forbidden_content`: requires `forbidden_patterns`, optional `applies_to`
> - `architectural_constraint`: requires `file_pattern` (glob) + `forbidden_patterns`
> - `file_naming`: requires `applies_to` (path glob) + `file_pattern` (regex on basename)
>
> ## Examples
>
> Architectural rule (most rules look like this — semantic, no mechanical check):
> ```json
> {
>   "id": "arch-001",
>   "kind": "decision",
>   "topic": "layering",
>   "severity_class": "decision_violation",
>   "description": "Business logic must not depend on UI framework classes",
>   "why": "The decision chain roots in testability. Business logic that references framework classes can't be unit-tested without instrumentation, which breaks the fast-feedback loop and makes refactoring risky.",
>   "forced_by": "Pure-Kotlin unit-test policy — every ViewModel covered by a JVM test, no Robolectric, no instrumentation.",
>   "enables": "Sub-second feedback loop on `./gradlew :app:testDebugUnitTest`; refactors stay safe because business rules have characterization tests.",
>   "alternative": "Inject a thin port (e.g. `interface Clock { fun now(): Instant }`) and bind the framework-aware implementation in the Koin module.",
>   "example": "class CartViewModel(private val cart: CartRepository) { fun checkout(): Result<Order> = cart.placeOrder() }",
>   "source": "deep_scan",
>   "keywords": ["viewmodel", "domain", "business logic", "context"]
> }
> ```
>
> Pitfall rule (blocks because walking into a documented trap):
> ```json
> {
>   "id": "ctx-001",
>   "kind": "pitfall",
>   "topic": "concurrency",
>   "severity_class": "pitfall_triggered",
>   "description": "Never use context.TODO() or context.Background() inside request handlers",
>   "why": "Pitfall #7: handlers that swallow the request context lose cancellation, deadlines, and tracing. Downstream calls become orphans on client disconnect, leaving partial state in DB. Always thread through the handler's incoming ctx.",
>   "forced_by": "Caller-driven cancellation contract — every handler must propagate the request's deadline so downstream RPCs and DB calls cancel together.",
>   "enables": "Bounded resource usage on client disconnect; correct OpenTelemetry trace stitching across services.",
>   "alternative": "Accept `ctx context.Context` as the first parameter and pass it down unchanged; if the call must outlive the request, branch via `context.WithoutCancel(ctx)` and document why.",
>   "example": "func (h *handler) Charge(ctx context.Context, in ChargeIn) error { return h.svc.Process(ctx, in) }",
>   "source": "deep_scan",
>   "keywords": ["context", "handler", "request"]
> }
> ```
>
> Tradeoff signal rule (warns — agent might have a reason):
> ```json
> {
>   "id": "cache-001",
>   "kind": "semantic_pattern",
>   "topic": "data-access",
>   "severity_class": "tradeoff_undermined",
>   "description": "Avoid sync I/O inside the cache hot path",
>   "why": "We chose an in-memory cache for sub-ms reads. Tradeoff signal: any blocking I/O in Get/Set undermines the latency budget that justified the choice. Async or skip the cache.",
>   "example": "",
>   "source": "deep_scan",
>   "keywords": ["cache", "io", "latency"]
> }
> ```
>
> Mechanical rule (regex check, blocks instantly):
> ```json
> {
>   "id": "dep-001",
>   "kind": "decision",
>   "topic": "layering",
>   "severity_class": "mechanical_violation",
>   "description": "Domain layer must not import from presentation layer",
>   "why": "The domain is the stable core. UI depends on domain, never the reverse. Inverting this makes every UI refactor a domain change.",
>   "example": "",
>   "source": "deep_scan",
>   "keywords": ["domain", "presentation", "import"],
>   "check": "forbidden_import",
>   "applies_to": "domain/",
>   "forbidden_patterns": ["from presentation", "import.*\\.ui\\."]
> }
> ```
>
> File-placement rule (derived from `architecture_rules.file_placement_rules`):
> ```json
> {
>   "id": "place-001",
>   "kind": "file_placement",
>   "topic": "layering",
>   "severity_class": "pattern_divergence",
>   "description": "Fragments must live under `app/src/main/java/.../page_<feature_name>/fragment/`",
>   "why": "The page_<feature> package is the unit of feature ownership. Co-locating fragment, viewmodel, and cells under one package keeps the feature swappable. Fragments elsewhere lose this locality and the DI graph can't see them.",
>   "example": "// LoginFragment.kt lives at app/src/main/java/com/foo/bar/page_login/fragment/LoginFragment.kt",
>   "source": "deep_scan",
>   "keywords": ["fragment", "page", "feature placement"],
>   "applies_to": "app/src/main/java/com/foo/bar/page_",
>   "file_pattern": "*Fragment.kt"
> }
> ```
>
> Naming-convention rule (derived from `architecture_rules.naming_conventions`):
> ```json
> {
>   "id": "name-001",
>   "kind": "naming_convention",
>   "topic": "layering",
>   "severity_class": "mechanical_violation",
>   "description": "Feature ViewModels are named `<Feature>ViewModel.kt`",
>   "why": "ViewModels carry feature scope. The `<Feature>ViewModel` name lets the Koin module + the DI graph + grep all find the same class — drop the convention and feature swaps break.",
>   "example": "",
>   "source": "deep_scan",
>   "keywords": ["viewmodel", "naming"],
>   "check": "file_naming",
>   "applies_to": "app/src/main/java/com/foo/bar/page_*",
>   "file_pattern": ".*ViewModel\\.kt$"
> }
> ```
>
> Coding-practice rule (derived from `development_rules` or `infrastructure_rules` — informational, not blocking):
> ```json
> {
>   "id": "practice-001",
>   "kind": "coding_practice",
>   "topic": "data-access",
>   "severity_class": "pattern_divergence",
>   "description": "Every Repository exposes only `Flow<T>` — no suspend functions or callbacks on the public surface",
>   "why": "Repositories are the boundary into the data layer. Flows give consumers cancellation + back-pressure for free; suspend functions force every caller to invent its own coroutine scope. The decision was made when adopting Koin singletons.",
>   "example": "interface SettingsRepository { val settings: Flow<Settings> ; fun update(s: Settings) }",
>   "source": "deep_scan",
>   "keywords": ["repository", "flow", "suspend"]
> }
> ```
>
> ## Phase 2 — `triggers` block (RECOMMENDED for every rule)
>
> The pre-validate hook narrows candidate rules at edit time using a small structured block called `triggers`. When you can express the rule's *applicability* and *violation signal* as a path glob + content regex, write them here. The hook will fire the rule deterministically without calling any AI. Rules without `triggers` are still candidates for Phase 3 (plan/commit-time semantic comparison) but skip the hot edit-time path.
>
> ```json
> "triggers": {
>   "path_glob": ["openmeter/billing/charges/**/adapter/**"],
>   "code_shape": [
>     {
>       "kind": "regex_in_content",
>       "must_match": ["func \\w+ \\(.*\\*entdb\\.Client.*\\)"],
>       "must_not_match": ["entutils\\.Tx\\("]
>     }
>   ]
> }
> ```
>
> - `path_glob`: array of glob patterns. `*` matches within a path segment, `**` matches across. Trailing `/` matches as a directory prefix. The rule is a candidate for an edit only if the file's relative path matches at least one glob.
> - `code_shape`: array of structured matchers. Each entry currently uses `kind: "regex_in_content"`. The shape fires when **any** `must_match` pattern matches the diff/content AND **none** of the `must_not_match` patterns matches.
> - Both arrays are AND-combined: path_glob narrows file applicability, code_shape narrows content. Either alone is fine — omit the other key entirely if not relevant.
> - **Trigger-only rules are valid.** A rule with `triggers` but no `check` field uses the trigger as the violation detector — if the trigger fires, the rule is violated, severity per `severity_class`. Use this for layering / pattern / decision rules where the regex IS the structural test.
> - **Trigger + check** is also fine: triggers narrow candidacy, the existing `check` field runs the deterministic check.
> - **No triggers** = Phase 3 only. The rule stays semantic; the edit-time hook ignores it; the plan/commit classifier reasons about it.
>
> ## What to produce:
>
> **Coverage — sweep every blueprint section that carries enforcement signal.** You are the only producer of `proposed_rules.json`. If a section names a constraint or convention but you don't emit a rule for it, the user has no way to curate it through the viewer. Walk these in order:
>
> | Blueprint section | Emit `kind` | Typical `severity_class` |
> |---|---|---|
> | `decisions.key_decisions[*]` | `decision` | `decision_violation` |
> | `pitfalls[*]` | `pitfall` | `pitfall_triggered` |
> | `decisions.trade_offs[*].violation_signals[*]` | `semantic_pattern` | `tradeoff_undermined` |
> | `components.patterns[*]` / `implementation_guidelines[*]` | `semantic_pattern` | `pattern_divergence` |
> | `architecture_rules.file_placement_rules[*]` | `file_placement` | `pattern_divergence` (or `mechanical_violation` if a regex captures it cleanly) |
> | `architecture_rules.naming_conventions[*]` | `naming_convention` | `mechanical_violation` (file-basename regex) or `pattern_divergence` |
> | `development_rules[*]` | `coding_practice` | `pattern_divergence` |
> | `infrastructure_rules[*]` | `coding_practice` | `pattern_divergence` |
>
> Do NOT skip a section because "those aren't 'real' rules" — if it's in the blueprint, the agent should know about it, and the only way the agent learns about it is for you to emit it into proposed_rules.json so the user adopts it. Aim for ≥1 emitted rule per non-empty section.
>
> **Deep architectural rules** — invariants an AI coding agent might accidentally violate. These are the most valuable. Derive them from decision chains, trade-offs, pitfalls, and pattern descriptions. Examples: "ViewModel must never reference View/Context", "Repository must use IO dispatcher", "Fragments must use DI delegation not direct construction".
>
> **Structural rules** — dependency direction between layers/components, forbidden technologies (from decisions/trade-offs).
>
> **File-placement and naming-convention rules** — derived directly from the corresponding `architecture_rules` sections. These are user-curated through the same Adopt / Reject / Disable / Enable lifecycle as every other rule; do not write them back into `blueprint.architecture_rules` — emit them HERE.
>
> ## Critical:
> - Every rule must be specific to THIS project — never generic programming advice
> - Focus on what an AI coding agent would get wrong without knowing this codebase
> - **`kind` is required.** Pick one of: `decision`, `pitfall`, `semantic_pattern`, `file_placement`, `naming_convention`, `coding_practice`. The viewer groups rules by this for user curation.
> - **`severity_class` is required.** Pick the one that matches which blueprint section motivated the rule.
> - **`topic` is required.** Pick from the recommended list (or a project-specific topic when justified) — the renderer groups rules into per-topic files using this slug.
> - **`description` is the title** — one imperative sentence, no version pins, no config flags, no project-name references. See "Description shape" above for the canonical bad-vs-good comparison.
> - **`why` is required and must be inlined from the blueprint** — copy the language verbatim or near-verbatim, do not paraphrase. This is what the agent reads at edit-time. Version pins (`Koin 3.1.3`), configuration flags (`allowOverride(true)`), and project-name references belong here, NOT in `description`.
> - **`forced_by`, `enables`, `alternative` are recommended for every `decision` and `pitfall` rule.** Copy `forced_by` and `enables` from the source `decisions.key_decisions[*]` entry. Write `alternative` as one imperative sentence pointing at the project's correct path. Use `""` when not applicable; never omit the key.
> - **`example` is required as a key** but may be an empty string when the rule is purely about what NOT to do and no positive code shape applies.
> - **`source: "deep_scan"`** on every rule. The post-process stamps it defensively but prefer to emit it.
> - **Add `triggers` whenever the rule has a structural signature** (file path + content pattern). Trigger regexes are more robust than the older `forbidden_patterns` because path_glob + content combine cleanly. Rules with no structural signature (purely semantic) stay trigger-less and fire only at plan/commit time via Phase 3.
> - If you include `forbidden_patterns` or `triggers.code_shape`, every regex must be valid Python `re` syntax.
> - Include an `"id"` field for each rule (e.g., "dep-001", "arch-001", "ban-001")
> - The `description` must explain WHAT is forbidden in one sentence
> - Do NOT force mechanical fields — most rules will be `decision_violation` / `pitfall_triggered` / `tradeoff_undermined` / `pattern_divergence` with no `check`. Only add the regex fields when a meaningful pattern exists, and pair them with `severity_class: "mechanical_violation"`.

**IMPORTANT: If `.archie/rules.json` already exists (from previous scans), read it first. The new rules must be MERGED with existing rules — do not overwrite user-adopted rules.**

Instruct the agent to write its own output (append to its prompt):

```
---
OUTPUT CONTRACT (mandatory):
1. Use the Write tool to save your COMPLETE output to /tmp/archie_rules_$PROJECT_NAME.json
2. Write the raw output verbatim — extract_output.py handles JSON envelopes.
3. After Writing, reply with exactly: "Wrote /tmp/archie_rules_$PROJECT_NAME.json"
4. Do NOT print the output in your response body.
```

After the agent's confirmation returns, extract:

```bash
python3 .archie/extract_output.py rules /tmp/archie_rules_$PROJECT_NAME.json "$PROJECT_ROOT/.archie/rules.json"
```

**IMPORTANT: Do NOT try to extract or parse JSON yourself. Do NOT copy the agent's transcript. Always use the pre-installed scripts on the file the agent already wrote.**

Build the Phase 2 trigger index so the pre-validate hook can narrow candidates fast on every edit:

```bash
python3 .archie/rule_index.py build "$PROJECT_ROOT"
```

Refresh the rendered topic files now that `rules.json` exists. This re-emits the `.claude/rules/enforcement/` directory (index.md + by-topic/ + universal.md) and refreshes the other topic files / CLAUDE.md / AGENTS.md idempotently — merge markers preserve any hand-edits:

```bash
python3 .archie/renderer.py "$PROJECT_ROOT"
```

```bash
python3 .archie/intent_layer.py deep-scan-state "$PROJECT_ROOT" complete-step 6
```

---

### ✓ Compact Checkpoint A — before Intent Layer

**This is the highest-leverage compaction point in the whole pipeline.** Steps 1–6 are fully persisted (blueprint, findings, pitfalls, rules, proposed_rules, telemetry marks). Wave 1 subagent transcripts, Wave 2 Opus synthesis, and Rule Synthesis are redundant with the disk state — holding them in conversation context is pure waste for the massive Intent Layer pass that follows (which spawns one Sonnet subagent per folder batch).

If the orchestrator's context is over ~70%, pause here, run `/compact`, and resume with `/archie-deep-scan --continue`. The Resume Prelude reads `deep_scan_state.json` (last_completed=6) and jumps straight to Step 7 after rehydrating shell vars from the persisted run_context.

Skipping this checkpoint is safe — auto-compact will fire if needed — but compacting here is strictly cheaper and cleaner.

---

