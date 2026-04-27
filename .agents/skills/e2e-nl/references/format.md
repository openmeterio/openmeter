# NL Spec Format

How to write a natural-language e2e test specification — runner-neutral,
contract-only, generator-friendly.

This reference is the format contract for the `e2e-nl` skill. Worked
examples of every shape class live in `examples.md`.

---

## Goal

A natural-language test specification that:

1. Reads like a design doc — reviewable by non-Go engineers, readable in
   60 seconds.
2. Is the **source of truth**: shipped tests map to it, not the other
   way around.
3. **Is runner-agnostic by construction.** The spec describes observable
   HTTP behavior — verbs, paths, status codes, response-body shapes,
   `problem+json` error shapes. Each downstream generator (Go,
   Playwright, `.http`, Hurl, …) translates the same spec into its
   runtime. Language-specific or framework-specific details never belong
   in the spec.
4. Is mechanically translatable to executable tests by an LLM or a
   small generator.

---

## The runner-agnosticism rule

**Central principle.** The NL spec is a test plan against an API
contract, not against any particular runner. Each scenario describes
wire-level behavior — HTTP request/response, `problem+json` body shape,
JSON field assertions. Runner-specific translation is always a
downstream concern.

### Forbidden in the spec

- **Runner-specific code or helper names** — no `newV3Client`, no
  `validPlanRequest`, no `test.describe`, no `.http` variables. These
  are transient scaffolding.
- **Runner-specific construction mechanics** — no
  `FromBillingPriceFlat`, no `nullable.NewNullableWithValue`, no
  `new Request(...)`. Describe the JSON body shape; leave wire-format
  plumbing to each generator.
- **Runner-specific naming conventions** — no `TestV3<…>`, no
  Playwright `test(…)` titles, no `### Section` markers for `.http`.
  The spec commits to the `id` slug only; each generator derives its
  own idiomatic name.
- **Assumptions about test isolation, parallelism, cleanup, or client
  lifecycle.** These are runner choices. "Fresh fixtures per row" is
  observable behavior; "fresh client per row" is not.

### Encouraged in the spec

- **JSON-visible vocabulary.** `Expect status is active` (wire-level
  string), not `Expect plan.Status == BillingPlanStatusActive` (Go
  type).
- **HTTP-native verbs and paths.** `POST /plans`, `GET /plans/{id}`.
- **API schema type names** from TypeSpec / OpenAPI
  (`CreatePlanRequest`, `BillingRateCard`). These appear in every
  generator's toolchain.

---

## File layout

One Markdown file per endpoint family — convention is
`e2e/specs/<family>.md`, co-located with the e2e tests.

```
# E2E Scenario Specifications — <Family>

<one-paragraph preamble: scope, lifecycle/no-lifecycle, family-wide
error-shape conventions>

---

## Scenario list

<plain bullets, one per scenario, grouped by priority>

---

## Baselines

<named object shapes scenarios reference by name>

---

## Scenario: <id>

<fenced YAML + Intent + Fixtures + Steps + Notes>

## Scenario: <id>
…
```

The `## Scenario: <id>` heading is the canonical drafted marker — the
heading text is exactly the scenario's `id` slug, not a human title.
Presence of such a section is ground truth for "this scenario is
drafted." No checkbox, no status tracker.

---

## The scenario contract

Every scenario **MUST** provide:

- `id` (fenced YAML) — a stable snake_case slug, unique within the
  spec file.
- `endpoints` (fenced YAML) — a list of `METHOD /path` entries the
  scenario exercises.
- **Intent** — one sentence describing the behavior the scenario pins.
- **Fixtures** — the API-level preconditions described as Baseline
  references (or inline JSON when a Baseline isn't warranted).
- **Steps** — numbered HTTP actions with:
  - The verb + path + request body reference.
  - The expected HTTP status code.
  - Response-field assertions in JSON-visible vocabulary.
  - For 4xx steps: the expected error shape (**domain code** /
    **detail substring** / **schema rule**) and the code / substring
    / rule string.

Every scenario **MAY** provide:

- `entities`, `tags`, `references`, `status` (fenced YAML).
- Per-step `Captures:` blocks (see below).
- **Notes** section covering validation-moment choices, omitted rows,
  caveats, or `NEEDS-VERIFY` items awaiting server confirmation.

Downstream generators own: test file location, language-level test
function naming, helper / client / fixture-builder code, concurrency
and isolation strategies, discriminated-union construction, cleanup
mechanics. None of this belongs in the spec.

---

## Controlled vocabularies

### Error-shape vocabulary

Every expected 4xx names one of three shapes. Which shape a server uses
is a server-side choice; pinning it in the spec keeps the generated
test honest.

| Shape | How the server returns it | Spec phrasing |
|---|---|---|
| **Domain code** | `extensions.validationErrors[].code` | `Expect domain code "<code>".` |
| **Detail substring** | `problem.detail` (free text) | `Expect detail contains "<substring>".` |
| **Schema rule** | `invalid_parameters[].rule` | `Expect schema rule "<rule>".` |

### Validation-moment vocabulary

Draftable entities (plans, addons, plan-addons) can fire validation at
three moments. Spec each assertion against the moment it fires.

- **create-time** — the POST itself is rejected.
- **GET-time** — POST accepts as draft; GET returns the entity with
  `validation_errors` populated.
- **publish-time** — POST accepts, GET may show errors, and the
  publish POST is rejected.

Non-draftable entities (e.g. features) fire only at create-time or
update-time. Pin the moment in the scenario's Notes if it's not
obvious from the steps.

---

## Captures directive

When a step's response produces state later steps need (typically a
generated `id`), add a `Captures:` block at the end of the step. Two
equivalent forms:

```
# Capture the whole response body as an object (preferred when later
# steps need multiple fields):
Captures:
- `plan` ← `response.body`

# Capture a specific field by JSON path:
Captures:
- `plan_id` ← `response.body.id`
```

The left side is a snake_case local name. The right side is rooted at
one of:

- `response.body` — the whole response body as an object.
- `response.body.<json-path>` — a specific field.
- `response.headers.<header-name>` — a response header value.
- `response.status` — the numeric status code.

Later steps reference captured values with **braces**: `{plan.id}`
(dotted field access on an object capture) or `{plan_id}` (a flat
capture). The braces disambiguate bound values from literal text.

```
2. **Get plan.** `GET /plans/{plan.id}`.
```

Fixture-produced values (e.g. "a plan at status `active`") are bound
by the Fixtures section and referenced the same way — `{plan.id}`,
`{plan.phases[0].key}`, `{addon.id}`. No explicit `Captures:` block
is needed; Fixture block names are implicit captures.

Every runner can implement this: Go assigns to a local variable,
Playwright binds with `const`, `.http` uses `@var = {{response.body.$.id}}`,
Hurl uses `[Captures]`. The spec stays neutral.

---

## Null vs. absent

JSON distinguishes between `key: null` (present with null value) and
key-missing (omitted). Pin this explicitly rather than papering over
it:

- `is null` — the key is present in the body with explicit `null`.
- `is absent` — the key is not present in the body (server
  `omitempty`).
- `is null or absent` — used only when the server is genuinely
  ambiguous; document in Notes when you reach for it.

Default for optional-field positive assertions: prefer `is absent` for
this codebase (most Go services omit-when-nil). Downstream generators
can collapse both to a single nil check when the target language
doesn't distinguish (e.g. Go pointer types).

---

## The `references:` field

Optional YAML list of freeform strings pointing at generated test
artifacts in any runner. The list form is canonical even for a single
entry — generators see one shape regardless of runner count.

```yaml
references:
  - e2e/plans_v3_test.go::TestV3PlanLifecycle
  - tests/plans.spec.ts::"plan lifecycle"
  - tests/http/plans.http#plan_lifecycle
```

Each entry is freeform — the path-and-anchor convention is whatever
each runner finds idiomatic. Omit the field entirely when no test has
been generated. `references:` is for traceability only, not contract;
the `id` slug is the stable identifier.

---

## The `status:` field

Optional YAML field that signals to downstream generators whether the
scenario should be exercised. Omitting it is the default — the
scenario is in scope for every generator that consumes this spec.

| Value | Meaning |
|---|---|
| omitted (default) | Scenario is in scope. Generators emit a test. |
| `skipped` | Scenario is **not** to be exercised. Generators MUST NOT emit a test. |

`skipped` is for scenarios that describe contract behavior the
generator can't reach today — typically because of an environmental
gate (missing seed data, an external service the e2e stack doesn't
provision, a feature flag off in test). The scenario stays in the
spec so the contract is documented; the `status: skipped` directive
suppresses generation.

When marking a scenario `skipped`:

- Explain the gate in the scenario body — usually a leading
  **Status: SKIPPED.** callout under the YAML — and describe what it
  takes to re-enable.
- Do not delete the scenario. The contract is still part of the
  spec; the marker just defers the test.

```yaml
id: feature_llm_pricing_enriched_on_get
endpoints:
  - GET /features/{id}
status: skipped
```

The field is intentionally minimal — only `skipped` is defined.
Generators encountering an unknown value should treat the scenario
as in scope (default) and surface a warning rather than fail.

---

## Baselines

Each `e2e/specs/<family>.md` defines its **own** Baselines section.
Baselines are **not shared across files**: an addons spec and a
features spec each own their own baselines, even if the names
collide. Cross-file imports couple specs together and break the
"one file per family" rule.

A Baseline names an API schema type
(`CreatePlanRequest`, `CreateFeatureRequest`, `BillingRateCard`, …)
from TypeSpec / `api/openapi.yaml` and lists the field defaults a
scenario can mutate. Scenarios reference baselines by name and
describe per-scenario mutations rather than inlining JSON.

Baselines earn their place when **two or more scenarios** in the same
family use the shape. One-off shapes stay inlined in the scenario's
Fixtures block.

See `examples.md` for worked baseline definitions and how scenarios
reference them.

---

## Scenario list conventions

The Scenario list is a flat enumeration at the top of the file. One
plain bullet per scenario:

```
- `<id>` — <one-line intent> — shape: <class> — priority: <p0|p1|p2>
```

- `id`: stable snake_case slug, unique in the file.
- shape class: `lifecycle` | `draft-with-errors` | `matrix` |
  `single-request`. (See `examples.md` for one worked example per
  class.)
- priority:
  - **p0** — happy-path lifecycle or primary CRUD flow.
  - **p1** — core validation, error-shape proofs, state-matrix rules.
  - **p2** — edge cases, deprecated paths, rare documented errors.
- Append `NEEDS-VERIFY: <reason>` to the line for any candidate where
  a behavior isn't pinned by the code and needs live-server
  confirmation.

Group by priority with subheadings (`**p0 — happy path**`,
`**p1 — core validation**`, …) when the list is long enough to scan.
The `## Scenario: <id>` section that follows in the file is what
actually marks a scenario as drafted; the list is just an index.
