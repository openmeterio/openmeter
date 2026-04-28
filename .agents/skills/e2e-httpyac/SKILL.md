---
name: e2e-httpyac
description: Generate executable httpYac `.http` e2e tests from a natural-language scenario specification produced by `/e2e-nl`. The output is one `.http` file per scenario, runnable via `httpyac send` against a live OpenMeter server with JUnit / JSON reporting. Use this skill when an NL spec at `e2e/specs/<family>.md` is in place and you want runnable wire-level tests with a lighter ceremony than the Go e2e suite — particularly useful for exploratory contract tests, manual debugging, and CI smoke runs against deployed environments. Reach for it for phrasings like "translate the features spec to httpYac", "generate `.http` tests for `<scenario>`", "I want runnable contract tests from the NL spec".
user-invocable: true
argument-hint: "[family or e2e/specs path] [optional: scenario id]"
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# E2E httpYac Test Generator

You translate a natural-language e2e scenario specification into an
executable httpYac `.http` file. Output format follows
`references/format.md`, modeled on the worked examples in
`references/examples.md`.

This skill is **step 2 in a pipeline**:

```text
endpoint code → [/e2e-nl] → NL spec → [/e2e-httpyac] → executable .http
```

Sibling generators may emit Go (`/e2e`), Playwright, Hurl, etc. The
contract between every generator and `/e2e-nl` is the NL spec; no
generator reads any other generator's output.

## Before you start — read these

1. `references/format.md` — format contract for emitted `.http` files
   (naming convention, assertion vocabulary, capture pattern, error
   shapes, forbidden directives).
2. `references/examples.md` — one worked example per shape class.

Both files live alongside this `SKILL.md` inside the skill directory.
They are the contract. If either is missing, stop and tell the user —
the skill depends on them and shouldn't improvise.

## What this skill does **not** do

- It doesn't read endpoint code (`api/spec/`, `api/v3/handlers/`, …).
  That's `/e2e-nl`'s job. If the NL spec is missing, run `/e2e-nl`
  first.
- It doesn't write Go e2e tests. That's `/e2e`'s job.
- It doesn't decide whether a scenario should exist. The NL spec is
  the source of truth; this skill emits exactly what's drafted there.
- It doesn't validate execution against a live server. After writing
  the file, you tell the user the install + run command. The user runs
  it; debugging the runtime is a separate task.

## Inputs — the NL spec

Given an endpoint family (e.g. "features") or an explicit path:

1. Locate the spec at `e2e/specs/<family>.md`.
2. Confirm it has a `## Scenario: <id>` heading for the requested
   scenario id (or, if no id was given, list the available ids and
   ask).
3. Read the scenario's YAML, Intent, Fixtures, Steps, and Notes
   blocks. Read the file's Baselines and family-wide preamble for
   context.

Report the spec path and the scenario id under translation. If
either is missing or ambiguous, ask before proceeding — don't guess.

## Workflow — five modes

Two top-level operations: **single-scenario emit** (the default)
and **family regenerate** (explicit user intent — phrasings like
"regenerate the features family", "wipe and rebuild the .http files
for plans", "delete and regenerate"). Modes dispatch on whether the
project root and target file already exist:

| Mode | Project root | Scenario file | Scenario id | Action |
|---|---|---|---|---|
| bootstrap | no | n/a | any | Create `e2e/http/httpyac.config.js`, then continue as below. |
| draft new | yes | no | given | Emit `e2e/http/<family>/<scenario_id>.http`. |
| list scenarios | yes | no | not given | List ids from the NL spec and ask which to draft (or whether to regenerate the family). |
| update existing | yes | yes | given | Refuse to overwrite silently. Diff against what you'd emit; ask whether to update, replace, or skip. |
| **regenerate family** | yes | any | n/a, **regenerate intent explicit** | Emit every `## Scenario:` heading from the NL spec into `e2e/http/<family>/`, overwriting prior files. **Confirm with the user first** — hand-edits to existing files will be lost. Skipped scenarios still emit (with `# @disabled`); never silently drop. |

Single-scenario emit is the right default for incremental work.
Family-regenerate is the right move when the NL spec has changed
materially, when the skill rules change (new error-shape template,
new naming rule), or when you want a clean reset after
experimentation.

## Idempotency contract

The skill is designed so that **deleting any subset of `.http`
files under `e2e/http/<family>/` and re-running family-regenerate
produces functionally equivalent output**. Equivalence is the
contract; byte-for-byte identity is not — a careful diff may catch
trivial formatting variation, but assertions, captures, request
shapes, and naming will match.

What's deterministic:

- **Region order:** matches NL spec Step order.
- **Request names** (`r__<scenario>__<step_slug>`): scenario id and
  step slug both derive from the NL spec verbatim. The step slug
  is a lowercase-snake-case rendering of the NL spec's Step title
  text — see `references/format.md` for the exact rule.
- **Capture variable names** (`v__<scenario>__<capture>`): the
  capture name matches the NL spec's `Captures:` block label.
- **Assertion choice:** mechanically driven by NL spec phrasing
  (see `references/format.md`'s assertion vocabulary table and
  error-shape templates). No model creativity in the assertion
  layer.
- **File header:** scenario id + spec source path comment using
  the fixed template in `references/format.md`.

What's *runtime* dynamic but *emit-time* deterministic:

- `{{$timestamp}}_{{$randomInt 0 1000000}}` suffixes for unique
  fixture keys: the literal text in the file is identical across
  emits; only the resolved value changes per `httpyac send`.

What's *not* idempotent (and how to avoid losing it on regenerate):

- **Hand-edits to a file's assertions or commentary.** If you
  pinned a `NEEDS-VERIFY` value, tightened an assertion, or added
  a regression-marker comment, family-regenerate will overwrite
  those edits. Workflow:
  1. Pin findings into the NL spec first (the canonical source of
     truth) — remove `NEEDS-VERIFY` markers, update detail
     substrings to match observed server output, etc.
  2. Then regenerate. The regenerate carries the findings forward.
  - Or, when regenerating, accept the overwrite and re-apply the
    delta in a follow-up edit pass.
- **`httpyac.config.js`.** Bootstrapped once; never touched again
  by the skill. Hand-edits (auth scrubbing hooks, custom default
  headers) are preserved across regenerates.

## Output file convention

```
e2e/
  http/
    httpyac.config.js              # project root marker
    features/
      feature_lifecycle.http
      feature_create_duplicate_key_rejected.http
      …                            # one file per ## Scenario: <id> in features.md
    plans/                         # future families
```

Rules:

- **One scenario per file.** No `### Scenario` separators packing
  multiple scenarios into one file. httpYac has no scope on `@name`
  across imports — collisions are silent and the first match wins.
  One file per scenario sidesteps this entirely.
- **File name equals the scenario id verbatim.** `<scenario_id>.http`.
- **Subdirectory per family** mirrors `e2e/specs/<family>.md`.
- **Project root** is `e2e/http/`. Do **not** write outside it.

## Project root setup (first run only)

If `e2e/http/httpyac.config.js` doesn't exist, bootstrap:

- `e2e/http/httpyac.config.js` — project-root marker. See
  `references/format.md` for the canonical content.

Configuration is shell-env based (mirrors the Go e2e convention
`OPENMETER_ADDRESS=... go test ./e2e/...`). The skill emits
`@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3` in every file; httpYac resolves
the value from `process.env` at send time. No env files, no
`--env <name>` flag, no `package.json`. Those are out of scope; the
`SKILL.md`'s Handoff section documents the canonical CLI invocation
that any wrapper can call.

## Naming convention (hard rule)

Every emitted file uses these prefixes, with the **scenario id** as
the namespace:

| Prefix | Used for | Example |
|---|---|---|
| `r__` | Request name (`# @name`) | `r__feature_lifecycle__create_feature` |
| `v__` | Exported variable (post-response script) | `v__feature_lifecycle__feature_id` |
| `d__` | File-global immutable data variable | `d__feature_lifecycle__feature_key` |
| upper-snake | Environment variable (read from `process.env`) | `OPENMETER_ADDRESS` |

All names are valid JavaScript identifiers (lowercase snake_case,
letters/digits/underscores only). Dynamic values
(`{{$timestamp}}`, `{{$randomInt}}`, `{{$uuid}}`) appear in **values**,
never in names.

The `r__` / `v__` / `d__` distinction matters for review: a reader
should be able to grep an emitted file and see immediately which
identifiers are requests, which are runtime captures, and which are
literal fixture data.

## Error-shape triage

Every 4xx step in the NL spec names one of four shapes. Pick the
emit template by **shape**, not by scenario intent.

| Shape | NL phrasing | Emit template |
|---|---|---|
| **Detail equality** | `Expect detail equals "<full>"` | `?? js response.parsedBody.detail == <full>` |
| **Detail substring** | `Expect detail contains "<sub>"` | `?? js response.parsedBody.detail.includes('<sub>')` |
| **Domain code** | `Expect domain code "<code>"` | JS `test(...)` block asserting `response.parsedBody.extensions.validationErrors.some(e => e.code === "<code>")` |
| **Schema rule** | `Expect schema rule "<rule>"` (or `field`/`rule` pinned in spec) | `?? js response.parsedBody.invalid_parameters[0].field == <field>` + `?? js response.parsedBody.invalid_parameters[0].rule == <rule>` |

The NL spec's phrasing is unambiguous; the emit template follows
mechanically. See `references/format.md` for the full templates,
the `body contains` caveat (don't use it on
`application/problem+json` bodies), and the DSL-first principle.
`references/examples.md` carries one worked scenario per shape.

### Binder layer vs handler — the same input may surface as
different shapes depending on deployment topology

OpenMeter's API contract is defined by `api/openapi.yaml`. Some
deployments place an OpenAPI binder gateway (any conformant binder
— Kong, oapi-codegen middleware, Spectral validator, etc.) in
front of the handler. When a binder is present, schema-level
rejections (pattern, enum, format, type) fire **at the binder**
and surface with the **schema-rule** shape — the handler's domain
check for the same input never runs.

Practical consequence: a scenario the NL spec describes as
detail-substring (because that's what the handler emits in
isolation) may surface as schema-rule when run through a binder
gateway. Workflow when first generating a scenario:

1. Emit per the NL spec's stated shape.
2. Run once against the target deployment (`httpyac send …
   --output-failed response`).
3. If the response shape doesn't match what the spec said, the
   binder is intercepting. Re-classify the scenario's shape, pin
   the actual `field`/`rule` values, and update **both** the NL
   spec (with a deployment-topology note) and the emitted file.
4. Document the topology in the scenario's Notes section so
   future maintainers know the test is binder-shape-pinned.

`references/examples.md` shows this loop applied to
`feature_list_filter_malformed_rejected` and the matrix scenario's
row 10.

## Things to avoid

**Format-level:**

- **No `@loop`.** Matrix scenarios emit one request region per row.
  See `references/format.md` ("Matrix scenarios — N regions, not a
  loop") for the rationale and template.
- **No implicit `@name`-as-id captures.** `@name` exposes only the
  parsed body, not headers or status. Always emit an explicit
  post-response `exports.v__... = response.parsedBody.id` block.
- **No editor-only directives.** No `@note`, `@openWith`, `@save`,
  `$input`, `$password`, `$pick`. They break in CLI/CI runs.
- **No dynamic `@disabled`.** Static `@disabled` for `status: skipped`
  scenarios is fine; don't emit conditional disable expressions.
- **No `@import` of another scenario's file.** Cross-scenario state
  leaks via the shared first-match-wins `@name` namespace. Each
  scenario file is self-contained.
- **No clever lazy-variable chains (`@var := ...`).** Always emit
  fixed (`=`) values for data variables, and rely on explicit
  `exports` for runtime state. Lifecycle files become
  significantly easier to debug this way.

**Spec-fidelity:**

- **Don't invent assertions.** If the NL spec doesn't pin a field,
  don't assert it. Drift between spec and emitted test is what the
  pipeline exists to prevent.
- **Don't tighten `NEEDS-VERIFY` markers.** The spec author left
  them loose for a reason. Emit a JS assertion that's tolerant of
  the un-pinned value, and copy the `NEEDS-VERIFY: <reason>` text
  into a comment in the `.http` file.
- **Don't skip steps.** A multi-step lifecycle emits every step,
  including read-backs. Read-backs are how the test proves the
  mutation persisted; dropping them turns the test into a write-only
  smoke check.
- **Don't reorder steps.** httpYac runs requests in file order;
  reordering breaks captures.

## Handoff — installation and execution

The skill emits the `.http` file. The user runs it. Don't try to
execute httpYac inside the skill's environment — install the runner
on the user's machine and document the CLI invocation.

```bash
# httpYac is provided by the repo's Nix dev shell (see flake.nix).
# Enter the shell once via direnv (`direnv reload`) or
# `nix develop --impure .#ci`; `httpyac` will be on PATH.
# CI uses the same shell (`nix develop --impure .#ci -c httpyac ...`).

# run a single scenario
OPENMETER_ADDRESS=http://localhost:8888 \
  httpyac send e2e/http/features/feature_lifecycle.http \
  --all --output-failed response

# run a family (CI / JUnit)
OPENMETER_ADDRESS=http://localhost:8888 \
  httpyac send "e2e/http/features/**/*.http" \
  --all --bail --junit \
  --output-failed response --parallel 1 \
  > reports/features.junit.xml
```

The `--parallel 1` flag is intentional: lifecycle scenarios depend
on intra-file ordering. Cross-file parallelism is a CI-wrapper
concern (e.g. `xargs -P 4`), not a runner concern.

`--output-failed response` is the workaround for the JSON reporter
stripping `parsedBody` and `rawBody` from response objects — it
re-includes them on failed assertions for forensic triage.

## Smoke test

Run the skill against `feature_lifecycle` (already drafted in
`e2e/specs/features.md`) and diff the produced file against the
worked example in `references/examples.md`. Large overlap confirms
grounding; systematic differences surface gaps in either the skill
or the reference.

## Done criterion

A scenario is **emitted** when the file at
`e2e/http/<family>/<scenario_id>.http` parses (`httpyac` reports no
syntax errors), and the user has confirmed it runs against a real
server. The skill itself stops after writing the file; live
validation is the user's pass.

## Handoff to /e2e-nl

If the user requests a scenario the spec doesn't cover, point them at
`/e2e-nl` to draft it first. Don't invent NL content from endpoint
code in this skill — that's the upstream skill's job, and bypassing
it breaks the pipeline contract.
