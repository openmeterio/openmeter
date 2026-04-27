# Worked Examples — One Per Shape Class

This reference shows each of the four shape classes drafted as a real
NL scenario. The product-catalog domain provides the context, but the
patterns generalize to any v3 endpoint family.

Use these as templates when drafting new scenarios. Match the shape
class to the kind of behavior you're pinning, then mirror the
structure (Fixtures / Steps / Notes) with the same density of detail.

| Shape | Example | When to reach for it |
|---|---|---|
| `lifecycle` | `plan_lifecycle` | A primary CRUD flow with status transitions and side effects (`effective_from`, `archived_at`, …) on each step. |
| `draft-with-errors` | `plan_invalid_draft_lifecycle` | Defects accepted at create, surfaced on GET via `validation_errors`, rejected at publish. The canonical three-moment flow. |
| `single-request` | `plan_invalid_currency` | One request, one rule, one assertion. The default for a validation rule that fires at create-time only. |
| `matrix` | `plan_addon_attach_status_matrix` | A rule that branches on two or more state variables; one row per combination. |

When in doubt, start with `single-request` and promote if multi-step
state turns out to be needed. The matrix shape is for genuinely
combinatorial rules — don't reach for it just because there are
several similar one-shot validations.

---

## Baselines (illustrative)

Named object shapes the example scenarios reference. Each baseline
names the **API schema type** it instantiates — these are TypeSpec
types under `api/spec/packages/aip/src/productcatalog/` and appear in
`api/openapi.yaml` as the stable cross-language contract.

These specific baselines belong to the product-catalog domain. Other
families (features, meters, …) define their own. The pattern — name
the API schema type, list field defaults, allow per-scenario
mutation — is what generalizes.

Keys (`key`) are unique per run. The uniqueness strategy is a runner
concern, not a spec concern — any suffix that survives re-runs
against a shared DB is fine.

### Baseline plan — `CreatePlanRequest`

- `key`: unique
- `name`: `"Test Plan"`
- `currency`: `"USD"`
- `billing_cadence`: `"P1M"`
- `phases`: a list containing one **Baseline phase** (`last = true`)

### Baseline phase — `BillingPlanPhase`

Parameter `last` (boolean) controls the duration:

- `key`: unique
- `name`: `"Test Phase"`
- `duration`: `null` when `last = true`; `"P1M"` otherwise — non-last
  phases must be bounded.
- `rate_cards`: a list containing one **Baseline flat rate card**

### Baseline flat rate card — `BillingRateCard` with `BillingPriceFlat`

- `key`: unique
- `name`: `"Test Rate Card"`
- `price`: `BillingPriceFlat { type: "flat", amount: "10" }`
- `billing_cadence`: `"P1M"`
- `payment_term`: `"in_advance"`

### Baseline addon — `CreateAddonRequest`

- `key`: unique
- `name`: `"Test Addon"`
- `currency`: `"USD"`
- `instance_type`: `"single"`
- `rate_cards`: a list containing one **Baseline flat rate card**

### Baseline plan-addon attach — `CreatePlanAddonRequest`

- `name`: `"Test Plan Addon"`
- `addon`: `{ id: {addon.id} }` (referencing the addon bound by the
  scenario's fixtures)
- `from_plan_phase`: `{plan.phases[0].key}` (or whichever phase key
  the scenario targets)

---

## Example: `lifecycle` shape — Plan lifecycle

```yaml
id: plan_lifecycle
endpoints:
  - POST /plans
  - GET /plans/{id}
  - GET /plans
  - PUT /plans/{id}
  - POST /plans/{id}/publish
  - POST /plans/{id}/archive
  - DELETE /plans/{id}
entities: [plan]
tags: [lifecycle, crud]
```

**Intent:** A plan moves through the full draft → active → archived →
deleted lifecycle, and `effective_from` / `effective_to` transition
predictably at publish and archive.

**Fixtures:**
- A `CreatePlanRequest` per **Baseline plan**.

**Steps:**

1. **Create plan in draft.** `POST /plans` with the fixture.
   - Expect `201 Created`.
   - Expect `key` equals the fixture key.
   - Expect `version` is `1`.
   - Expect `status` is `draft`.
   - Expect `effective_from` and `effective_to` are null.

   Captures:
   - `plan` ← `response.body`

2. **Get plan (draft).** `GET /plans/{plan.id}`.
   - Expect `200 OK`.
   - Expect `id` equals `{plan.id}`.
   - Expect `status` is `draft`.
   - Expect `version` is `1`.
   - Expect `effective_from` is null.

3. **List plans and find the created plan.**
   `GET /plans?page[size]=1000`.
   - Expect `200 OK`.
   - Expect response `data[]` contains an entry with `id == {plan.id}`.
   - Note: page size 1000 because the shared DB may push fresh rows
     past the default page-1 window of 20.

4. **Update plan — rename phase and add a second rate card.**
   `PUT /plans/{plan.id}` with an `UpsertPlanRequest` describing the
   **full desired state** (PUT is full-replace, not delta):
   - `name`: unchanged from the create body.
   - `phases`: a list of one phase with:
     - `key`: the original phase `key`.
     - `name`: `"Phase Renamed"`.
     - `duration`: the original `duration` (`null` for a last phase).
     - `rate_cards`: the original rate card plus one additional rate
       card per **Baseline flat rate card** (two total).

   Assertions:
   - Expect `200 OK`.
   - Expect `phases[0].key` equals the original phase key (phase keys
     are immutable).
   - Expect `phases[0].name` is `"Phase Renamed"`.
   - Expect `phases[0].rate_cards` has length `2`.
   - Expect `status` still `draft`.

5. **Get plan (verify update persisted).** `GET /plans/{plan.id}`.
   - Expect `200 OK`.
   - Expect `phases[0].name` is `"Phase Renamed"`.
   - Expect `phases[0].rate_cards` has length `2`.

6. **Publish plan.** `POST /plans/{plan.id}/publish`.
   - Expect `200 OK`.
   - Expect `status` is `active`.
   - Expect `effective_from` is non-null.
   - Expect `effective_to` is null.

7. **Archive plan.** `POST /plans/{plan.id}/archive`.
   - Expect `200 OK`.
   - Expect `status` is `archived`.
   - Expect `effective_to` is non-null.

8. **Delete plan.** `DELETE /plans/{plan.id}`.
   - Expect `204 No Content`.

9. **Get plan after deletion.** `GET /plans/{plan.id}`.
   - Expect `200 OK`.
   - Expect `deleted_at` is non-null.
   - Note: deleted plans remain retrievable; they don't 404. Compare
     with `feature_lifecycle` where delete-then-GET returns 404 — this
     is a per-resource server choice, pin it scenario-by-scenario.

---

## Example: `draft-with-errors` shape — Plan invalid-draft lifecycle

This is the canonical three-moment flow (create → GET → publish).
Use it as the template for any new draftable-entity invalid-draft
test.

```yaml
id: plan_invalid_draft_lifecycle
endpoints:
  - POST /plans
  - GET /plans/{id}
  - POST /plans/{id}/publish
  - PUT /plans/{id}
entities: [plan]
tags: [validation, draft, publish-time, canonical-three-moment]
```

**Intent:** A plan with a phase that has no rate cards is accepted at
create (stored as draft with validation errors), surfaces its errors
on GET, is rejected at publish with a domain code, and can be fixed
via PUT and republished.

**Fixtures:**
- A `CreatePlanRequest` per **Baseline plan** whose single phase uses
  **Baseline phase** with `last = true` but **overrides `rate_cards`
  to `[]`** (the empty list is the defect under test).

**Steps:**

1. **Create accepts the invalid draft.** `POST /plans` with the
   fixture.
   - Expect `201 Created`.
   - Note: this is a validation-moment choice — an empty-rate-card
     phase is accepted at create, not rejected.

   Captures:
   - `plan` ← `response.body`

2. **GET surfaces `validation_errors`.** `GET /plans/{plan.id}`.
   - Expect `200 OK`.
   - Expect `validation_errors` is non-null and non-empty.
   - Expect `validation_errors[].code` includes domain code
     `"plan_phase_has_no_rate_cards"`.

3. **Publish is rejected with the same code.**
   `POST /plans/{plan.id}/publish`.
   - Expect `400 Bad Request`.
   - Expect domain code `"plan_phase_has_no_rate_cards"`.

4. **Fix by adding a rate card.** `PUT /plans/{plan.id}` with an
   `UpsertPlanRequest` describing the full desired state:
   - `name`: unchanged from the create body.
   - `phases`: a list of one phase with the original `key`, `name`,
     and `duration`, but `rate_cards` now containing one rate card
     per **Baseline flat rate card** (the empty-rate-cards defect is
     gone).

   Assertions:
   - Expect `200 OK`.

5. **Publish succeeds after fix.** `POST /plans/{plan.id}/publish`.
   - Expect `200 OK`.
   - Expect `status` is `active`.

**Notes:**

- Three validation moments are exercised in one scenario —
  create-time accepts, GET-time surfaces, publish-time rejects.
  Calling out the moment per assertion keeps the generated test
  honest.

---

## Example: `single-request` shape — Plan invalid currency

The simplest validation scenario — one request, one rule, one
assertion. Reach for this when a rule is pinned to create-time and
doesn't carry draft / lifecycle semantics.

```yaml
id: plan_invalid_currency
endpoints:
  - POST /plans
entities: [plan]
tags: [validation, create-time, single-request]
```

**Intent:** A plan submitted with an unknown ISO-4217 currency is
rejected at create-time with a structured 400 carrying the
`currency_invalid` domain code.

**Fixtures:**
- A `CreatePlanRequest` per **Baseline plan** with `currency`
  overridden to `"ZZZ"` (not a valid ISO-4217 code).

**Steps:**

1. **Create with invalid currency.** `POST /plans` with the fixture.
   - Expect `400 Bad Request`.
   - Expect domain code `"currency_invalid"`.

**Notes:**

- This is a **create-time** validation — the handler rejects the
  request before accepting any draft. Contrast with
  `plan_invalid_draft_lifecycle`, where defects are accepted at
  create and only surface on GET / publish.

---

## Example: `matrix` shape — Plan-addon attach status matrix

Use the matrix shape when a rule branches on two or more state
variables. One Markdown table per row, plus one "Steps per row"
block.

```yaml
id: plan_addon_attach_status_matrix
endpoints:
  - POST /plans/{id}/addons
entities: [plan, addon, plan-addon]
tags: [matrix, validation, attach, status-matrix]
```

**Intent:** Attaching an addon to a plan is only allowed when
`plan.status ∈ {draft, scheduled}` **and** `addon.status == active`.
All other combinations are rejected with a plain BaseAPIError whose
`detail` carries the reason.

**Fixtures (built fresh per row):**
- A plan at the row's target `plan.status`. Create from
  **Baseline plan**, then advance:
  - `draft` → stop after create.
  - `active` → create, then `POST /plans/{id}/publish`.
  - `archived` → create, publish, then `POST /plans/{id}/archive`.
  - (`scheduled` is not reachable — see Notes.)
- An addon at the row's target `addon.status`. Create from
  **Baseline addon**, then advance with the same pattern.
- A `CreatePlanAddonRequest` per **Baseline plan-addon attach** with
  `addon.id = {addon.id}` and
  `from_plan_phase = {plan.phases[0].key}`.

**Rows:**

| # | plan.status | addon.status | Expect | Detail contains |
|---|---|---|---|---|
| 1 | draft | active | `201 Created` | — |
| 2 | active | active | `400 Bad Request` | `"invalid active status, allowed statuses: [draft scheduled]"` |
| 3 | archived | active | `400 Bad Request` | `"invalid archived status, allowed statuses: [draft scheduled]"` |
| 4 | draft | draft | `400 Bad Request` | `"invalid draft status, allowed statuses: [active]"` |
| 5 | draft | archived | `400 Bad Request` | `"invalid archived status, allowed statuses: [active]"` |

**Steps per row:**

1. **Attach.** `POST /plans/{plan.id}/addons` with the fixture body.
   - Expect the row's `Expect` status.
   - If 4xx, expect **detail contains** the row's `Detail contains`
     substring — BaseAPIError shape (`problem.detail`, not
     `extensions.validationErrors`).
   - If 2xx, expect the response body parses as a `PlanAddon`.

**Notes:**

- The `scheduled` plan row is intentionally omitted: the v3 publish
  endpoint hardcodes `effective_from = clock.Now()`, so a plan
  cannot be driven to `scheduled` via the public API. Add the row if
  publish takes a future `effective_from` body.
- Status-mismatch rejections use the **detail substring** shape, not
  **domain code**. Don't assume every domain rule surfaces through
  `extensions.validationErrors` — here the `attach` handler wraps
  the validator error as a BaseAPIError.
- Row fixtures are built fresh per row — "fresh fixtures" is
  observable behavior. "Fresh client per row" is a runner choice
  and stays out of the spec.
