---
name: e2e-nl
description: Generate a natural-language, runner-agnostic e2e test specification from a v3 API endpoint's source files (TypeSpec + handler + domain validator + converter). The output is a Markdown scenario spec that downstream skills translate into Go, Playwright, .http, Hurl, or other runners. Use this skill whenever someone wants to spec, document, or capture API contract behavior for an endpoint family before writing tests — including phrasings like "write an NL spec", "describe the contract for endpoint X", "what scenarios should we test on /foo", or "I want a test plan for the bar API". Reach for it especially when an endpoint has no e2e coverage yet, or when the user wants a runner-neutral plan that multiple downstream generators can consume.
user-invocable: true
argument-hint: "[endpoint family | path to TypeSpec file] [optional: scenario id]"
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# E2E Natural-Language Spec Generator

You produce a natural-language e2e test specification for an API
endpoint from its code. Output format follows `references/format.md`,
modeled on the worked examples in `references/examples.md`.

This skill is **step 1 in a pipeline**:

```text
endpoint code → [e2e-nl] → NL spec → [/e2e or other generator] → executable tests
```

Your output feeds the next skill. Describe behavior unambiguously
enough that a downstream generator can emit tests from the NL spec
alone.

## Before you start — read these

1. `references/format.md` — format rules, vocabulary, contract.
2. `references/examples.md` — one worked scenario per shape class.

Both files live alongside this SKILL.md inside the skill directory.
They are the contract. If either is missing, stop and tell the user
— the skill depends on them and shouldn't improvise.

`e2e_spec_plan.md` (top-level) is the broader pipeline tracker. Read
it only if the user references pipeline status, open questions, or
"where we left off." It isn't required for drafting scenarios.

## The consumer-perspective rule

**The NL spec describes API-observable behavior only.** You read up to
four internal files to understand *what* the API does; you never write
*how* it does it into the spec.

Concretely:

- ✅ "Posting a plan with `currency: 'ZZZ'` returns 400 with domain
  code `currency_invalid`." — Consumer-observable.
- ❌ "The handler calls `validateCurrency()` which iterates ISO-4217
  codes." — Implementation.
- ✅ "A plan with a phase that has no rate cards is accepted at create
  and surfaces `plan_phase_has_no_rate_cards` on GET." — Visible
  validation moment.
- ❌ "The service defers rate-card validation to `Plan.Validate()` at
  publish." — Implementation.

If a behavior is only visible through logs, DB state, or internal
telemetry, it is **not** in scope. The reader of a generated test —
and any future generator skill — should be able to verify every
assertion by issuing HTTP requests and reading responses.

## Inputs — the four source files

Given an endpoint family (e.g. "features") or a TypeSpec path, locate:

1. **TypeSpec** — `api/spec/packages/aip/src/<family>/` (v3) or
   `api/spec/packages/legacy/src/<family>/` (v1).
   *Gives:* request/response types, path + method, status codes, error
   response shapes.
2. **Handler** — `api/v3/handlers/<family>/` (v3) or
   `api/handlers/<family>/` (v1).
   *Gives:* which validators fire, which error shape is used
   (`HandleIssueIfHTTPStatusKnown` → domain code, `BaseAPIError` →
   detail substring, schema binder → invalid_parameters rule).
3. **Domain validator / rules** — `openmeter/<family>/`. Look for
   `Validate()`, `Publishable()`, validation error codes, predicate
   packages.
   *Gives:* the validation moment (create-time vs. GET-time vs.
   publish-time) and the domain-code vocabulary.
4. **Converter** — `convert.go` next to the handler.
   *Gives:* fields that map 1:1 vs. drop vs. reshape; hints at fields
   that deserve dedicated scenarios.

Report the four paths you found and the primary endpoint(s) under
test. If one is missing or ambiguous, ask before proceeding — don't
guess.

## Workflow — three modes

The skill takes an optional `[scenario id]` argument and dispatches
based on whether a `e2e/specs/<family>.md` file already exists:

| Family file | Scenario id | Action |
|---|---|---|
| absent | not given | Draft the **scenario list + Baselines + first p0 scenario** to seed the file. |
| absent | given | Draft the **scenario list + Baselines + the named scenario**. The scenario list is still required even when a specific scenario is requested first — it's the index. |
| present | given | Append **only** the named `## Scenario: <id>` section. Do not modify the list, Baselines, or prior scenarios. |
| present | not given | Ask the user which scenario id to draft next. The list shows what's available. |

When seeding a new family file:

1. Read the four source files.
2. Extract observable behavior:
   - Endpoints (method + path).
   - Request / response types.
   - Status codes (2xx and 4xx).
   - Error shapes (`extensions.validationErrors[].code` /
     `problem.detail` substring / `invalid_parameters[].rule`).
   - Lifecycle transitions and validation moments.
   - Matrix branch points (status × status, type × quantity, etc.).
   - Uniqueness / conflict constraints.
3. Produce a **scenario list**. Plain bullets, no checkboxes. Format
   per `references/format.md` ("Scenario list conventions"):
   ```text
   - `<id>` — <one-line intent> — shape: <class> — priority: <p0|p1|p2>
   ```
   Append `NEEDS-VERIFY: <reason>` for any candidate the code doesn't
   fully pin down.
4. Define any **new Baselines** the scenarios will reference. Reuse
   no baselines from another family file — each family file is
   self-contained (see Baselines below).
5. Draft the first scenario in full (per the dispatch table above).
6. Return to the user with the list + the seeded scenario.

## Done criterion

A family file is **minimally testable** once its `p0` scenarios are
drafted. p1 scenarios are core validation; drafting them is recommended
but the user picks the cadence. p2 scenarios are edge cases; draft them
when a real run hits one.

Don't draft scenarios indefinitely on your own — after each draft,
hand control back to the user with the list of remaining scenarios,
not a freshly-drafted one.

## Shape-class signals in the code

| Shape | Code signal | Example |
|---|---|---|
| **lifecycle** | Create + update + publish/archive/delete handlers; domain has a `Status` enum and `effective_from/to` semantics. | Plan lifecycle. |
| **draft-with-errors** | `Validate()` vs. `Publishable()` split; `validation_errors` on the response body. | `plan_invalid_draft_lifecycle`. |
| **matrix** | Rule branches on two or more state variables. | Attach `plan.status × addon.status`. |
| **single-request** | One-shot rule at create-time; no lifecycle follow-up. | Invalid currency. |

When in doubt, start with `single-request` and promote if multi-step
state turns out to be needed.

### Worked dispatch — picking the shape

> The features handler rejects a `feature` whose `key` parses as a
> ULID. The validator returns immediately at create-time; there is no
> draft state, no second moment. There is no matrix here either —
> only one variable (the `key`) decides. → `single-request`.
>
> The plans handler accepts a plan with an empty-rate-card phase at
> create, but `Validate()` populates `validation_errors` on GET and
> `Publishable()` rejects at publish. → `draft-with-errors` (the
> three-moment template).
>
> Attaching a plan-addon checks both `plan.status` and `addon.status`
> and rejects every disallowed combination with a different detail
> substring. Two state variables, several rows. → `matrix`.

## Baselines

Each `e2e/specs/<family>.md` defines its **own** Baselines section. Do
not import baselines from another family file. The shapes a feature
spec uses (`CreateFeatureRequest`, `BillingFeatureManualUnitCost`)
have nothing to do with the shapes a plan spec uses
(`CreatePlanRequest`, `BillingRateCard`), even when the names rhyme.
Cross-file baseline imports couple specs together and break the
"one file per family" rule.

When the endpoint family needs an object shape **not** yet in its
file's Baselines section, produce **both**:

1. **A proposed new Baseline block** for that file's Baselines
   section. Name the API schema type (`CreateFeatureRequest`,
   `Meter`, etc.) and list field defaults.
2. **An inlined shape** in the scenario's Fixtures block, so the
   scenario reads standalone.

Present both alternatives to the user and let them pick. Rule of
thumb: shapes used by ≥2 scenarios in the family earn a Baseline;
one-offs stay inlined.

See `references/examples.md` for a worked Baselines section.

## Output file

Default path: `e2e/specs/<family>.md`, co-located with the e2e tests
in the project (e.g. `e2e/specs/features.md`). If the user specifies
a different path, use that. Create the `e2e/specs/` directory if it
doesn't exist.

- **If the file doesn't exist** — create with the preamble,
  scenario list, Baselines section (only with new baselines added
  here), and the first drafted scenario.
- **If the file exists** — append only the named `## Scenario: <id>`
  section. Do not modify the list, Baselines, or prior scenarios.
  If the new scenario needs an object shape that isn't yet a
  Baseline, surface that as a separate decision for the user
  (per the Baselines section above): either inline the shape in the
  scenario's Fixtures block, or propose a new Baseline as a separate
  edit. Never add a Baseline silently during append-mode.

Always include a short preamble above the list:

```markdown
# E2E Scenario Specifications — <Family>

Natural-language, runner-agnostic description of e2e scenarios for
the `<family>` endpoint(s). Each `## Scenario` describes wire-level
behavior (HTTP verb, path, status code, response shape,
`problem+json` error shape) that any downstream runner can translate
to an executable test.

See the `e2e-nl` skill (`.agents/skills/e2e-nl/`) for format rules
(`references/format.md`) and worked examples (`references/examples.md`).
```

## Things to avoid

**Runner-agnosticism — the central rule.** The spec must read the
same whether the downstream generator is Go, Playwright, `.http`,
Hurl, or something that doesn't exist yet. Violations:

- **No runner-specific names.** `TestV3<…>`, `validPlanRequest`,
  `uniqueKey`, `v3helpers_test.go`, `test.describe`, `.http` section
  markers — none of this belongs in the spec.
- **No runner-specific construction mechanics.**
  `FromBillingPriceFlat`, `nullable.NewNullableWithValue`,
  `new Request(...)` — describe the JSON body shape; every generator
  handles its own plumbing.
- **No assumptions about client lifecycle, isolation, parallelism,
  or fixture cleanup.** Those are runner choices. "Fresh fixtures
  per row" is an observable behavior; "fresh client per row" is
  not.

**Consumer perspective:**

- **No internal package paths, service methods, ent queries.** Read
  them to understand behavior; never write them into the spec.
- **No inferred behavior ungrounded in the code.** Flag uncertainty
  with `NEEDS-VERIFY` and probe against a live server — do not
  fabricate rules.

**Output discipline:**

- **Every 4xx step names a validation moment** (create-time /
  GET-time / publish-time / update-time) and an error shape (domain
  code / detail substring / schema rule). Both vocabularies are in
  `references/format.md`.
- **Do not modify previously drafted scenarios on follow-up
  invocations.** Append only. A scenario's presence is its drafted
  state; there is no checkbox to tick.

## Smoke test

Run the skill against a known-covered endpoint family (plans) and
diff the produced spec against the worked examples in
`references/examples.md`. Large overlap confirms grounding;
systematic differences surface gaps in either the skill or the
reference.

## Handoff

Once an NL spec is complete (all p0 scenarios drafted, p1/p2 as
desired), any number of downstream generator skills can consume it:

- The existing `/e2e` skill (`.agents/skills/e2e/SKILL.md`) emits Go
  e2e tests.
- Future skills may emit Playwright, `.http`, Hurl, or other runners
  from the same source file.

The NL spec is the contract between this skill and all consumers —
no direct interaction, no runner-specific fields added here. If a
downstream skill needs information the spec doesn't carry (e.g.,
"where do generated tests live for runner X"), that's the
downstream skill's business to resolve by its own convention, not
this skill's to bake in.
