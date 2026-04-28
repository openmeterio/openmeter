# E2E Scenario Specifications — Features

Natural-language, runner-neutral description of e2e scenarios for the
`/openmeter/features` endpoint family (and
`/openmeter/features/{id}/cost/query` for cost query). Each
`## Scenario` maps 1:1 to a target test function in any runner that
consumes this spec.

See the `e2e-nl` skill (`.agents/skills/e2e-nl/`) for format rules
(`references/format.md`) and worked examples (`references/examples.md`).

Features **do not have a draft/publish/archive lifecycle**. `DELETE` is a
soft archive, `PATCH` only accepts `unit_cost`, and there is no
`validation_errors` on the response body — validation rules fire at
create-time or update-time only. No `draft-with-errors` shape applies.

**Error-shape convention for this family:** almost all 4xx responses
use the **detail-substring** shape (`problem.detail`), not the
**domain code** shape. Schema-rule shape (`invalid_parameters[]`) only
fires for malformed query params / bodies — i.e. requests that fail
schema validation before the request body is processed.

**GET by id vs. by key.** `GET /features/{featureId}` accepts either an
id or a key. When the entity is not found, the by-id path returns the
handler's 404 (`"feature not found: <id>"`), while the by-key path may
be intercepted at the gateway layer with a different detail
(`"The requested route was not found"`). For assertions on 4xx
responses, prefer GET by id; reserve key lookup for positive
resolution scenarios.

---

## Scenario list

Index of scenarios in this file. Presence of a `## Scenario: <id>`
section below is the source of truth for "this scenario exists" — the
list is just a directory. Priority is a scheduling hint, not a
commitment. Entries marked `NEEDS-VERIFY` have a rule the code doesn't
fully pin down — confirm behavior against a live server before adding
the corresponding `## Scenario` section.

**p0 — happy path**

- `feature_lifecycle` — Create static feature, get, list, patch to add/clear manual unit_cost, delete, get-after-delete. — shape: lifecycle — priority: p0

**p1 — core validation**

- `feature_create_duplicate_key_rejected` — Posting twice with the same key returns 409. — shape: single-request — priority: p1
- `feature_create_key_must_not_be_ulid` — Posting a feature whose key parses as a ULID returns 400. — shape: single-request — priority: p1
- `feature_create_meter_not_found_rejected` — Posting a feature referencing a non-existent meter returns 404 with detail `"meter not found: <id>"` (detail-substring shape). — shape: single-request — priority: p1
- `feature_create_invalid_meter_aggregation_rejected` — Creating a feature whose meter aggregation is outside `{sum, count, unique_count, latest}` returns 400. — shape: single-request — priority: p1
- `feature_create_invalid_meter_filter_key_rejected` — Meter filter key not in meter `group_by` returns 400. — shape: single-request — priority: p1
- `feature_create_llm_unit_cost_without_meter_rejected` — LLM `unit_cost` with no associated meter returns 400. — shape: single-request — priority: p1
- `feature_create_llm_property_not_in_meter_rejected` — LLM `provider_property` / `model_property` / `token_type_property` referencing a key absent from the meter's `group_by` returns 400. — shape: single-request — priority: p1
- `feature_create_unit_cost_validation_matrix` — Matrix over manual/llm validation rules (type/payload mismatch, mutex of `provider` vs. `provider_property`, negative amount, invalid `token_type`). — shape: matrix — priority: p1
- `feature_update_unit_cost_required` — PATCH with empty body (or without `unit_cost` key) returns 400 — `unit_cost` must be explicitly specified (either a value or `null`). — shape: single-request — priority: p1
- `feature_update_llm_without_meter_rejected` — PATCH adding LLM `unit_cost` on a feature without a meter returns 400. — shape: single-request — priority: p1
- `feature_update_nonexistent_rejected` — PATCH on a non-existent feature id returns 404. — shape: single-request — priority: p1
- `feature_get_nonexistent_returns_404` — GET on a non-existent feature id returns 404. — shape: single-request — priority: p1
- `feature_delete_nonexistent_returns_404` — DELETE on a non-existent feature id returns 404. — shape: single-request — priority: p1
- `feature_get_by_key_resolves` — GET with `{featureId}` as the feature's key (not id) resolves to the feature. — shape: single-request — priority: p1

**p1 — list filters, sort, pagination**

- `feature_list_filter_by_meter_id` — `filter[meter_id][eq]=<id>` returns only features on that meter. — shape: single-request — priority: p1
- `feature_list_filter_by_key` — `filter[key][eq]=<key>` returns matching features. — shape: single-request — priority: p1
- `feature_list_filter_by_name` — `filter[name][eq]=<name>` returns matching features. — shape: single-request — priority: p1
- `feature_list_combined_filters` — Combining `filter[meter_id]` + `filter[key]` narrows intersection. — shape: matrix — priority: p1
- `feature_list_sort_by_supported_fields` — `sort=key`, `sort=name`, `sort=created_at`, `sort=updated_at` (with `:asc` / `:desc`) returns the expected order. — shape: matrix — priority: p1
- `feature_list_sort_invalid_field_rejected` — `sort=<unknown_field>` returns 400 with schema-rule shape. — shape: single-request — priority: p1
- `feature_list_filter_malformed_rejected` — Malformed filter value (unknown operator, bad ULID format on `filter[meter_id]`) returns 400 with schema-rule shape. — shape: single-request — priority: p1
- `feature_list_pagination` — `page[number]` + `page[size]` return the expected slice and response metadata. — shape: single-request — priority: p1

**p1 — cost query**

- `feature_cost_query_happy_path` — `POST /features/{id}/cost/query` with an empty body on a feature that has a meter and manual unit_cost returns 200 with `data[]` populated (at least one row; zero-usage yields `cost: "0"`, `usage: "0"`, `currency: "USD"`, and the default time window `1970-01-01T00:00:00Z` → `1970-01-01T00:01:00Z`). — shape: single-request — priority: p1
- `feature_cost_query_nonexistent_feature_returns_404` — `POST /features/{id}/cost/query` on a non-existent id returns 404 with detail `"feature not found: <id>"`. — shape: single-request — priority: p1
- `feature_cost_query_no_meter_rejected` — `POST /features/{id}/cost/query` on a feature without an associated meter returns 400 with detail `"feature <key> has no meter associated"`. — shape: single-request — priority: p1

**p2 — edge cases**

- `feature_archived_excluded_from_list` — After delete, the feature is not returned by a default list request. — shape: single-request — priority: p2
- `feature_archived_get_returns_404` — After delete, GET returns 404 with detail `"feature not found: <id>"`. (TypeSpec allows 410 `Common.Gone` but the current server always returns 404.) — shape: single-request — priority: p2
- `feature_llm_pricing_absent_when_unresolved` — GET of an LLM feature whose provider+model don't resolve in the LLM cost database returns 200 with no `pricing` block (silently absent; no error). — shape: single-request — priority: p2
- `feature_llm_pricing_enriched_on_get` — GET of an LLM feature whose provider+model resolve returns a `pricing` block populated from the LLM cost database. — shape: single-request — priority: p2 — **SKIPPED:** environment-gated; deferred until the e2e stack seeds the LLM cost DB with a known `(openai, gpt-4)` price. Do not generate a test for this scenario.

---

## Baselines

Named object shapes scenarios reference by name. Each names the API
schema type (from TypeSpec under
`api/spec/packages/aip/src/features/`, surfaced in `api/openapi.yaml`).

### Baseline feature — `CreateFeatureRequest`

The minimal valid feature — no meter reference, no unit cost. Scenarios
mutate this shape per test intent (adding a meter reference, adding a
`unit_cost`, changing the `key`, etc.).

- `key`: unique; **must not parse as a valid ULID**.
- `name`: `"Test Feature"`.

Optional fields available for mutation:

- `description`: string.
- `meter`: `{ id: <meter_id>, filters?: <map of meter group-by key → filter string> }`.
- `unit_cost`: discriminated union (`manual` | `llm`) — see the two sub-baselines below.
- `labels`: map.

### Baseline manual unit cost — `BillingFeatureManualUnitCost`

Simplest valid manual cost. Amount is non-negative.

- `type`: `"manual"`.
- `amount`: `"5"`.

### Baseline LLM unit cost (static) — `BillingFeatureLLMUnitCost`

All three dimensions as static values. Requires the feature to have a
meter associated. `pricing` is read-only and populated by the server on
`GET` if the provider + model resolve against the LLM cost database.

- `type`: `"llm"`.
- `provider`: `"openai"`.
- `model`: `"gpt-4"`.
- `token_type`: `"input"`.

### Baseline meter — `CreateMeterRequest`

Minimal `count`-aggregation meter that any feature scenario can use as
its meter reference. The key is unique per run.

- `key`: unique.
- `name`: `"Test Meter"`.
- `event_type`: `"e2e_feature_test"`.
- `aggregation`: `"count"`.
- `dimensions`: omitted.

### Baseline LLM-capable meter — `CreateMeterRequest`

A meter with the three group-by dimensions an LLM unit cost may
reference (`provider`, `model`, `token_type`). Used by scenarios that
need a feature with `BillingFeatureLLMUnitCost` to validate cleanly.

- `key`: unique.
- `name`: `"Test LLM Meter"`.
- `event_type`: `"e2e_llm_test"`.
- `aggregation`: `"count"`.
- `dimensions`: `{ provider: "$.provider", model: "$.model", token_type: "$.token_type" }`.

---

## Scenario: feature_lifecycle

```yaml
id: feature_lifecycle
endpoints:
  - POST /features
  - GET /features/{id}
  - GET /features
  - PATCH /features/{id}
  - DELETE /features/{id}
entities: [feature]
tags: [lifecycle, crud]
```

**Intent:** A feature moves through the CRUD lifecycle — create static
feature, get, list, patch `unit_cost` to add then clear it, delete, and
get-after-delete returns 404.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature**.

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created` with a `Feature` body.
   - Expect `key` equals the fixture key.
   - Expect `name` is `"Test Feature"`.
   - Expect `meter` is absent.
   - Expect `unit_cost` is absent.
   - Expect `deleted_at` is absent.

   Captures:
   - `feature` ← `response.body`

2. **Get feature.** `GET /features/{feature.id}`.
   - Expect `200 OK`.
   - Expect `id` equals `{feature.id}`.
   - Expect `key` equals the fixture key.
   - Expect `name` is `"Test Feature"`.

3. **List features and find the created feature.**
   `GET /features?page[size]=1000`.
   - Expect `200 OK`.
   - Expect response `data[]` contains an entry with `id == {feature.id}`.
   - Note: page size 1000 because the shared DB may push fresh rows
     past the default page-1 window of 20.

4. **Patch feature — set manual unit_cost.**
   `PATCH /features/{feature.id}` with an `UpdateFeatureRequest`:
   - `unit_cost`: per **Baseline manual unit cost**
     (`{ type: "manual", amount: "5" }`).

   Assertions:
   - Expect `200 OK` with a `Feature` body.
   - Expect `unit_cost.type` is `"manual"`.
   - Expect `unit_cost.amount` is `"5"` (or a normalized equivalent;
     the server trims trailing zeros, so `"5.00"` round-trips as
     `"5"`).

5. **Get feature — verify unit_cost persisted.**
   `GET /features/{feature.id}`.
   - Expect `200 OK`.
   - Expect `unit_cost.type` is `"manual"`.
   - Expect `unit_cost.amount` matches the value from step 4.

6. **Patch feature — clear unit_cost.**
   `PATCH /features/{feature.id}` with an `UpdateFeatureRequest`:
   - `unit_cost`: `null`.

   Assertions:
   - Expect `200 OK`.
   - Expect `unit_cost` is absent on the response.

7. **Get feature — verify unit_cost cleared.**
   `GET /features/{feature.id}`.
   - Expect `200 OK`.
   - Expect `unit_cost` is absent.

8. **Delete feature.** `DELETE /features/{feature.id}`.
   - Expect `204 No Content`.

9. **Get feature after deletion.** `GET /features/{feature.id}`.
   - Expect `404 Not Found`.
   - Expect **detail contains** `"feature not found: {feature.id}"`.

**Notes:**

- **Validation moment:** all assertions in this scenario are at
  `create-time`, `GET-time` (reads), or `update-time` (PATCH). There is
  no publish or archive flow on features.
- **`unit_cost` presence semantics.** `UpdateFeatureRequest.unit_cost`
  uses `specified | value | null` semantics: omitting the key entirely
  is not allowed (a PATCH without `unit_cost` is rejected, covered in
  `feature_update_unit_cost_required`). Passing `null` clears the
  value; passing an object replaces it.
- **Get-after-delete status code.** The TypeSpec for `get-feature`
  declares both `Common.NotFound` (404) and `Common.Gone` (410) as
  possible responses. **Verified against a live server:** the server
  always returns 404 with detail `"feature not found: <id>"` — the
  410 path is aspirational. Pin 404 + detail-substring shape.
- **Decimal normalization.** `amount` strings are normalized on
  round-trip — trailing zeros trimmed. Assertions on numeric strings
  should match the normalized form.
- **No include-archived.** v3 has no `include_archived` query param on
  list or get; a deleted feature is unreachable via the public API.
  This is a known parity gap with v1.

---

## Scenario: feature_create_duplicate_key_rejected

```yaml
id: feature_create_duplicate_key_rejected
endpoints:
  - POST /features
entities: [feature]
tags: [validation, create-time, single-request, conflict]
```

**Intent:** Posting two features with the same `key` in the same
namespace returns `409 Conflict` on the second attempt.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature**.

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Create again with the same key.** `POST /features` with a body
   reusing the **same `key`** as `{feature.key}` (`name` may differ).
   - Expect `409 Conflict`.
   - Expect **detail contains** `"with key {feature.key} already exists"`.

**Notes:**

- **Validation moment:** create-time. The duplicate is detected
  before any state is committed — there is no partial create.
- **Error shape:** detail-substring (`problem.detail`).

---

## Scenario: feature_create_key_must_not_be_ulid

```yaml
id: feature_create_key_must_not_be_ulid
endpoints:
  - POST /features
entities: [feature]
tags: [validation, create-time, single-request]
```

**Intent:** A feature whose `key` parses as a ULID is rejected at
create-time. This protects the GET-by-id-or-key dispatch from
ambiguity.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature** with `key`
  overridden to a ULID literal (e.g. `"01HXYZABCDEFGHJKMNPQRSTVWX"`).

**Steps:**

1. **Create with ULID-shaped key.** `POST /features` with the fixture.
   - Expect `400 Bad Request`.
   - Expect **detail contains** `"Feature key cannot be a valid ULID"`.

**Notes:**

- **Validation moment:** create-time, before any state is committed.
- **Error shape:** detail-substring.

---

## Scenario: feature_create_meter_not_found_rejected

```yaml
id: feature_create_meter_not_found_rejected
endpoints:
  - POST /features
entities: [feature]
tags: [validation, create-time, single-request, not-found]
```

**Intent:** Posting a feature whose `meter.id` does not resolve to any
meter in the namespace returns `404 Not Found`. The handler resolves
the meter reference before persisting the feature.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature** with `meter`
  overridden to `{ id: "<random-ULID-not-in-DB>" }`. Use a freshly
  generated ULID literal to guarantee no collision.

**Steps:**

1. **Create referencing missing meter.** `POST /features` with the
   fixture.
   - Expect `404 Not Found`.
   - Expect **detail contains** `"meter not found: <id>"` (the
     literal id used in the request body).

**Notes:**

- **Validation moment:** create-time, before any state is committed.
- **Error shape:** detail-substring at status 404.

---

## Scenario: feature_create_invalid_meter_aggregation_rejected

```yaml
id: feature_create_invalid_meter_aggregation_rejected
endpoints:
  - POST /meters
  - POST /features
entities: [feature, meter]
tags: [validation, create-time, single-request]
```

**Intent:** Features can only be associated with meters whose
aggregation is one of `{ sum, count, unique_count, latest }`. Posting
a feature against a meter with any other aggregation
(`avg` / `min` / `max`) returns `400 Bad Request`.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter** with
  `aggregation` overridden to `"avg"` and `value_property` set to
  `"$.value"` (avg requires a value property).
- A `CreateFeatureRequest` per **Baseline feature** with `meter`
  overridden to `{ id: {meter.id} }`.

**Steps:**

1. **Create meter with avg aggregation.** `POST /meters` with the
   meter fixture.
   - Expect `201 Created`.

   Captures:
   - `meter` ← `response.body`

2. **Create feature against the meter.** `POST /features` with the
   feature fixture (referencing `{meter.id}`).
   - Expect `400 Bad Request`.
   - Expect **detail contains** `"features can only be created for"`
     (the message also enumerates `sum, count, unique_count, latest`
     in some order).

**Notes:**

- **Validation moment:** create-time. The aggregation check happens
  after the meter reference resolves but before any feature state
  is committed.
- **Error shape:** detail-substring at status 400.

---

## Scenario: feature_create_invalid_meter_filter_key_rejected

```yaml
id: feature_create_invalid_meter_filter_key_rejected
endpoints:
  - POST /meters
  - POST /features
entities: [feature, meter]
tags: [validation, create-time, single-request]
```

**Intent:** Meter filter keys (`meter.filters`) must reference a key
that exists in the meter's `dimensions` (group-by). Posting a feature
whose `meter.filters` references an unknown key returns `400`.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter** with `dimensions`
  overridden to `{ region: "$.region" }`.
- A `CreateFeatureRequest` per **Baseline feature** with `meter`
  overridden to:
  - `id`: `{meter.id}`.
  - `filters`: `{ unknown_key: { eq: "x" } }`.

**Steps:**

1. **Create meter with a single dimension.** `POST /meters` with the
   meter fixture.
   - Expect `201 Created`.

   Captures:
   - `meter` ← `response.body`

2. **Create feature with bad filter key.** `POST /features` with the
   feature fixture.
   - Expect `400 Bad Request`.
   - Expect **detail contains** `"filter key \"unknown_key\" is not a
     valid dimension of meter"`.

**Notes:**

- **Validation moment:** create-time, before any state is committed.
- **Error shape:** detail-substring.

---

## Scenario: feature_create_llm_unit_cost_without_meter_rejected

```yaml
id: feature_create_llm_unit_cost_without_meter_rejected
endpoints:
  - POST /features
entities: [feature]
tags: [validation, create-time, single-request, llm]
```

**Intent:** Posting a feature with an `llm` unit cost but no `meter`
reference returns `400`. LLM unit cost requires a meter.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature** with:
  - `meter`: omitted.
  - `unit_cost`: per **Baseline LLM unit cost (static)**.

**Steps:**

1. **Create LLM feature without meter.** `POST /features` with the
   fixture.
   - Expect `400 Bad Request`.
   - Expect **detail contains**
     `"LLM unit cost requires a meter to be associated with the feature"`.

**Notes:**

- **Validation moment:** create-time. Inner unit-cost validation
  runs first; the meter-presence check fires next.
- **Error shape:** detail-substring.

---

## Scenario: feature_create_llm_property_not_in_meter_rejected

```yaml
id: feature_create_llm_property_not_in_meter_rejected
endpoints:
  - POST /meters
  - POST /features
entities: [feature, meter]
tags: [validation, create-time, single-request, llm]
```

**Intent:** When an LLM unit cost uses `*_property` to reference a
meter group-by key, that key must exist in the meter's `dimensions`.
Posting a feature whose LLM unit cost references an absent property
returns `400`.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter** with `dimensions`
  overridden to `{ provider: "$.provider", model: "$.model" }` (note:
  `token_type` deliberately omitted).
- A `CreateFeatureRequest` per **Baseline feature** with:
  - `meter`: `{ id: {meter.id} }`.
  - `unit_cost`: a `BillingFeatureLLMUnitCost` with:
    - `type`: `"llm"`.
    - `provider_property`: `"provider"`.
    - `model_property`: `"model"`.
    - `token_type_property`: `"token_type"` (the missing one).

**Steps:**

1. **Create meter without `token_type` dimension.** `POST /meters`.
   - Expect `201 Created`.

   Captures:
   - `meter` ← `response.body`

2. **Create LLM feature referencing missing property.**
   `POST /features` with the feature fixture.
   - Expect `400 Bad Request`.
   - Expect **detail contains**
     `"token_type_property \"token_type\" not found in meter group-by keys"`.

**Notes:**

- **Validation moment:** create-time. `UnitCost.ValidateWithMeter`
  runs after the basic shape checks pass and the meter is resolved.
- **Error shape:** detail-substring.
- The same shape applies symmetrically for `provider_property` and
  `model_property`. One row covers the family; the matrix scenario
  exercises the others if needed.

---

## Scenario: feature_create_unit_cost_validation_matrix

```yaml
id: feature_create_unit_cost_validation_matrix
endpoints:
  - POST /meters
  - POST /features
entities: [feature, meter]
tags: [matrix, validation, create-time, llm, manual]
```

**Intent:** Pin the create-time validation rules for `unit_cost`
across both manual and LLM shapes — required-field, mutex, range,
and enum checks — that the public API can actually surface. Each row
exercises one rule against a fresh fixture.

**Fixtures (built fresh per row):**
- A `CreateMeterRequest` per **Baseline LLM-capable meter** (created
  fresh for any row whose feature references a meter).
- A `CreateFeatureRequest` per **Baseline feature** with:
  - `meter`: `{ id: {meter.id} }` (every row that needs an LLM
    unit cost also creates the meter above).
  - `unit_cost`: per the row's column.

**Rows:**

| # | Mutation from baseline (`type` + payload) | Expect | Shape |
|---|---|---|---|
| 1 | manual baseline (`type: manual`, `amount: "5"`); no meter | `201 Created` | — |
| 2 | LLM baseline (static provider/model/token_type); meter attached | `201 Created` | — |
| 3 | manual with `amount: "-1"`; no meter | `400 Bad Request` | detail equals `"validation error: manual unit cost amount must be non-negative"` |
| 4 | LLM with both `provider: "openai"` and `provider_property: "provider"`; meter attached | `400 Bad Request` | detail equals `"validation error: provider_property and provider are mutually exclusive"` |
| 5 | LLM with both `model: "gpt-4"` and `model_property: "model"`; meter attached | `400 Bad Request` | detail equals `"validation error: model_property and model are mutually exclusive"` |
| 6 | LLM with both `token_type: "input"` and `token_type_property: "token_type"`; meter attached | `400 Bad Request` | detail equals `"validation error: token_type_property and token_type are mutually exclusive"` |
| 7 | LLM with no `provider` and no `provider_property` (model + token_type still set); meter attached | `400 Bad Request` | detail equals `"validation error: either provider_property or provider is required for LLM unit cost"` |
| 8 | LLM with no `model` and no `model_property` (provider + token_type still set); meter attached | `400 Bad Request` | detail equals `"validation error: either model_property or model is required for LLM unit cost"` |
| 9 | LLM with no `token_type` and no `token_type_property` (provider + model still set); meter attached | `400 Bad Request` | detail equals `"validation error: either token_type_property or token_type is required for LLM unit cost"` |
| 10 | LLM baseline with `token_type: "banana"`; meter attached | `400 Bad Request` | schema-rule: `invalid_parameters[0].field == "unit_cost"`, `rule == "allOf"` |

**Steps per row:**

1. **(If the row needs a meter)** `POST /meters` with the meter
   fixture.
   - Expect `201 Created`.
   - Captures: `meter` ← `response.body`.
2. **Create feature.** `POST /features` with the feature fixture for
   the row.
   - Expect the row's `Expect` status.
   - If 4xx and the row's shape is **detail-equality** (rows 3–9),
     expect `response.parsedBody.detail` to equal the full pinned
     string (including the `"validation error: "` prefix).
   - If 4xx and the row's shape is **schema-rule** (row 10), expect
     `response.parsedBody.invalid_parameters[0].field` and `.rule` to
     match the pinned values.
   - If 2xx, expect the response body parses as a `Feature`.

**Notes:**

- **Validation moment:** create-time. All rows fire before insert.
- **Two error shapes in this matrix.**
  - Rows 3–9 are **handler rejections.** The OpenAPI binder layer
    (when present) forwards the request — the body schema is
    structurally valid — and the OpenMeter handler runs the
    mutex / required / range rules, returning a domain error. The
    framework wraps the bare message in a `"validation error:
    <msg>"` detail field — pin the **full prefixed string** via
    equality, since each row triggers exactly one rule and the
    detail is deterministic.
  - Row 10 is a **binder-layer rejection** in deployments where an
    OpenAPI schema validator runs ahead of the handler.
    `token_type` is constrained by an OpenAPI enum; the binder's
    `allOf` validator rejects before the handler runs. The
    handler's `"invalid token_type"` message is **unreachable** in
    that topology. The expected shape is schema-rule
    (`invalid_parameters[].rule == "allOf"`), not detail-equality.
    In a deployment that runs the handler directly with no schema
    validator in front, row 10 reverts to detail-substring shape
    carrying the handler's `"invalid token_type"` message —
    re-pin per deployment topology.
- **Joined errors.** If a future row removes more than one required
  dimension, the server may join multiple messages into a single
  `detail`, breaking the equality assertion. Either split into
  separate rows (one rule per row) or fall back to a
  `detail.includes('<distinctive substring>')` check — but **not**
  `?? body contains "..."`, which is unreliable on httpyac 6.16.7
  for `application/problem+json` bodies (see
  `e2e-httpyac/references/format.md`).
- **Rows intentionally omitted:** the internal "Manual configuration
  must not be set when type is llm" / "llm must not be set when type
  is manual" checks. The API discriminator-based converter never
  produces those mixed-shape inputs, so they aren't reachable through
  the public surface.
- **Pinned 2026-04-28** against an OpenMeter deployment running
  behind an OpenAPI binder gateway. Re-pin if the deployment
  topology changes (binder added, removed, or upgraded) or the
  framework wrapper is upgraded.

---

## Scenario: feature_update_unit_cost_required

```yaml
id: feature_update_unit_cost_required
endpoints:
  - POST /features
  - PATCH /features/{id}
entities: [feature]
tags: [validation, update-time, single-request]
```

**Intent:** `PATCH /features/{id}` requires the body to **explicitly
specify** the `unit_cost` field (either a value or `null`). An empty
body — or one without `unit_cost` — is rejected.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature**.

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Patch with empty body.** `PATCH /features/{feature.id}` with
   request body `{}`.
   - Expect `400 Bad Request`.
   - Expect **detail contains** `"unitCost is required"`.

**Notes:**

- **Validation moment:** update-time. The body shape is rejected
  before any state is committed.
- **Error shape:** detail-substring.
- **Tri-state semantics:** `unit_cost` uses
  *unspecified | value | null* tracking. `unspecified` (key absent)
  is rejected; `null` clears; an object replaces.

---

## Scenario: feature_update_llm_without_meter_rejected

```yaml
id: feature_update_llm_without_meter_rejected
endpoints:
  - POST /features
  - PATCH /features/{id}
entities: [feature]
tags: [validation, update-time, single-request, llm]
```

**Intent:** PATCHing an LLM `unit_cost` onto a feature that has no
associated meter returns `400`. Same rule as create-time, applied at
update.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature** (no meter).
- An `UpdateFeatureRequest` body with `unit_cost` per **Baseline LLM
  unit cost (static)**.

**Steps:**

1. **Create static feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Patch with LLM unit cost.** `PATCH /features/{feature.id}` with
   the update body.
   - Expect `400 Bad Request`.
   - Expect **detail contains**
     `"LLM unit cost requires a meter to be associated with the feature"`.

**Notes:**

- **Validation moment:** update-time, before any state is committed.
- **Error shape:** detail-substring.

---

## Scenario: feature_update_nonexistent_rejected

```yaml
id: feature_update_nonexistent_rejected
endpoints:
  - PATCH /features/{id}
entities: [feature]
tags: [validation, update-time, single-request, not-found]
```

**Intent:** PATCHing an id that doesn't resolve returns `404`.

**Fixtures:**
- An `UpdateFeatureRequest` body with `unit_cost: null` (a syntactically
  valid clear). Use a freshly generated ULID literal in the path —
  guaranteed not to exist in the DB.

**Steps:**

1. **Patch unknown id.** `PATCH /features/<random-ULID-not-in-DB>` with
   the body.
   - Expect `404 Not Found`.
   - Expect **detail contains** `"feature not found: <id>"` (the
     literal id used in the path).

**Notes:**

- **Validation moment:** update-time. The feature is resolved before
  any update is applied; missing → 404.
- **Error shape:** detail-substring at status 404.

---

## Scenario: feature_get_nonexistent_returns_404

```yaml
id: feature_get_nonexistent_returns_404
endpoints:
  - GET /features/{id}
entities: [feature]
tags: [single-request, not-found]
```

**Intent:** GET on an unknown id returns `404` with the standard
detail substring.

**Fixtures:** *(none — the request is path-only)*.

**Steps:**

1. **Get unknown id.** `GET /features/<random-ULID-not-in-DB>`.
   - Expect `404 Not Found`.
   - Expect **detail contains** `"feature not found: <id>"`.

**Notes:**

- **Error shape:** detail-substring.
- **GET by id vs. by key.** Lookup by key on a missing feature may
  surface a different gateway-level detail (`"The requested route
  was not found"`). This scenario uses a ULID-shaped path segment to
  exercise the handler's own 404, not the gateway.

---

## Scenario: feature_delete_nonexistent_returns_404

```yaml
id: feature_delete_nonexistent_returns_404
endpoints:
  - DELETE /features/{id}
entities: [feature]
tags: [single-request, not-found]
```

**Intent:** DELETE on an unknown id returns `404`.

**Fixtures:** *(none — the request is path-only)*.

**Steps:**

1. **Delete unknown id.** `DELETE /features/<random-ULID-not-in-DB>`.
   - Expect `404 Not Found`.
   - Expect **detail contains** `"feature not found: <id>"`.

**Notes:**

- **Error shape:** detail-substring.

---

## Scenario: feature_get_by_key_resolves

```yaml
id: feature_get_by_key_resolves
endpoints:
  - POST /features
  - GET /features/{featureId}
entities: [feature]
tags: [single-request, lookup-by-key, deployment-gated, skipped]
status: skipped
```

**Status: SKIPPED.** Deployment-gated. Verified 2026-04-28: in
deployments fronted by Kong (or any gateway whose route patterns
require `{featureId}` to look like a ULID), a `GET
/features/<bare-key>` request is rejected at the gateway with `404
"The requested entity was not found"` and never reaches the OpenMeter
handler. The handler's by-id-or-key dispatch is therefore unreachable
through the public surface in those topologies. Re-enable when
running against a deployment that routes arbitrary path segments to
the handler.

**Intent:** `GET /features/{featureId}` accepts either a ULID id or
the feature's `key` in the path. Lookup by key resolves to the same
feature.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature**.

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Get by key.** `GET /features/{feature.key}` (the path segment is
   the feature's `key`, not its `id`).
   - Expect `200 OK`.
   - Expect `id` equals `{feature.id}`.
   - Expect `key` equals `{feature.key}`.

**Notes:**

- **Why ULID-shaped keys are forbidden** (see
  `feature_create_key_must_not_be_ulid`): if a key parsed as a ULID,
  the by-id-or-key dispatch in `GetByIdOrKey` would be ambiguous.
  This scenario is the positive read of the same dispatch.
- **Gateway-routing dependency.** The Kong-fronted deployment we
  pin tests against returns `404 "The requested entity was not
  found"` (Kong's own response) for `GET /features/<bare-key>`. The
  handler-level 404 (`"feature not found: <id>"`) is reachable only
  via a ULID-shaped path segment.

---

## Scenario: feature_list_filter_by_meter_id

```yaml
id: feature_list_filter_by_meter_id
endpoints:
  - POST /meters
  - POST /features
  - GET /features
entities: [feature, meter]
tags: [list, filter, single-request]
```

**Intent:** `GET /features?filter[meter_id][eq]=<id>` returns only
features whose `meter.id` equals the given ULID.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter** (call it meter A).
- A second `CreateMeterRequest` per **Baseline meter** (meter B).
- A `CreateFeatureRequest` per **Baseline feature** with
  `meter: { id: {meter_a.id} }` (feature A).
- A second `CreateFeatureRequest` per **Baseline feature** with
  `meter: { id: {meter_b.id} }` (feature B).

**Steps:**

1. **Provision both meters and both features.** Four `POST` calls,
   capturing each response body. Each `POST` expects `201 Created`.

2. **List filtered by meter A.**
   `GET /features?filter[meter_id][eq]={meter_a.id}&page[size]=1000`.
   - Expect `200 OK`.
   - Expect response `data[]` contains an entry with
     `id == {feature_a.id}`.
   - Expect response `data[]` does **not** contain
     `id == {feature_b.id}`.

**Notes:**

- **Filter operator.** ULID fields support `eq`, `neq`, `oeq`. Bare
  ULID (`filter[meter_id]=<id>`) is also accepted by the schema.
  This scenario pins the explicit `eq` form; downstream generators
  may exercise alternates.
- Page size is bumped to 1000 to handle a shared-DB drift past
  page-1.

---

## Scenario: feature_list_filter_by_key

```yaml
id: feature_list_filter_by_key
endpoints:
  - POST /features
  - GET /features
entities: [feature]
tags: [list, filter, single-request]
```

**Intent:** `GET /features?filter[key][eq]=<key>` returns the matching
feature.

**Fixtures:**
- Two `CreateFeatureRequest`s per **Baseline feature**, each with a
  distinct unique `key`.

**Steps:**

1. **Create both features.** Two `POST /features` calls; each
   expects `201 Created`. Capture both response bodies as
   `feature_a` and `feature_b`.

2. **List filtered by feature A's key.**
   `GET /features?filter[key][eq]={feature_a.key}&page[size]=1000`.
   - Expect `200 OK`.
   - Expect `data[]` contains an entry with `id == {feature_a.id}`.
   - Expect `data[]` does **not** contain `id == {feature_b.id}`.

**Notes:**

- **Filter operator.** `key` is a `StringFieldFilter`; supports `eq`,
  `neq`, `contains`, `ocontains`, `oeq`, `gt`/`gte`/`lt`/`lte`,
  `exists`, or a bare string (interpreted as `eq`).

---

## Scenario: feature_list_filter_by_name

```yaml
id: feature_list_filter_by_name
endpoints:
  - POST /features
  - GET /features
entities: [feature]
tags: [list, filter, single-request]
```

**Intent:** `GET /features?filter[name][eq]=<name>` returns the
matching feature.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature** with `name`
  overridden to a unique distinctive value (e.g.
  `"Test Feature ListByName <unique-suffix>"`).

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **List filtered by name.**
   `GET /features?filter[name][eq]={feature.name}&page[size]=1000`
   (URL-encode the name).
   - Expect `200 OK`.
   - Expect `data[]` contains exactly the feature with
     `id == {feature.id}` (the unique-suffix guarantees no other
     match in a shared DB).

**Notes:**

- The unique suffix on `name` is what keeps the list narrow on a
  shared DB; without it the assertion would have to allow noise.

---

## Scenario: feature_list_combined_filters

```yaml
id: feature_list_combined_filters
endpoints:
  - POST /meters
  - POST /features
  - GET /features
entities: [feature, meter]
tags: [list, filter, matrix]
```

**Intent:** Combining multiple filter fields narrows the result to
the intersection. `filter[meter_id]` AND `filter[key]` together
return only features matching both.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter** (meter A) and a
  second one (meter B).
- Four `CreateFeatureRequest`s per **Baseline feature**:
  - feature_a1 with meter A and key `"alpha_<suffix1>"`.
  - feature_a2 with meter A and key `"beta_<suffix2>"`.
  - feature_b1 with meter B and key `"alpha_<suffix3>"`.
  - feature_b2 with meter B and key `"beta_<suffix4>"`.

**Rows:**

| # | Filter | Expected ids in `data[]` |
|---|---|---|
| 1 | `filter[meter_id][eq]={meter_a.id}` only | `{feature_a1.id}` and `{feature_a2.id}` |
| 2 | `filter[key][eq]={feature_a1.key}` only | `{feature_a1.id}` only |
| 3 | both `filter[meter_id][eq]={meter_a.id}` and `filter[key][eq]={feature_b1.key}` | empty (intersection) |

**Steps per row:**

1. **List with row's filters.**
   `GET /features?<row's query>&page[size]=1000`.
   - Expect `200 OK`.
   - For each id listed in the `Expected ids` cell, expect an entry
     with `id == <captured id>` to be present in `data[]`.
   - For ids not in the cell that exist in the fixture set, expect
     them **absent** from `data[]`.

**Notes:**

- The "empty intersection" row pins AND-semantics: filters compose,
  they don't union.
- Page size 1000 to absorb shared-DB noise.

---

## Scenario: feature_list_sort_by_supported_fields

```yaml
id: feature_list_sort_by_supported_fields
endpoints:
  - POST /features
  - GET /features
entities: [feature]
tags: [list, sort, matrix]
```

**Intent:** The `sort` query param accepts `key`, `name`,
`created_at` (default), and `updated_at`. Each value returns features
in ascending order on the named field.

**Fixtures:**
- Three `CreateFeatureRequest`s per **Baseline feature**, each with
  distinct unique-suffix `key` and `name` values, **created in
  predictable order** (e.g. A, then B, then C, far enough apart in
  time that `created_at` strictly orders them).

**Rows:**

| # | `sort` value | Expected order of fixture entries (by `id`) |
|---|---|---|
| 1 | `key` | sorted ascending by `key` |
| 2 | `name` | sorted ascending by `name` |
| 3 | `created_at` (or omitted — the default) | A, B, C |
| 4 | `updated_at` | sorted ascending by `updated_at` |

**Steps per row:**

1. **List with row's `sort`.**
   `GET /features?sort=<row's value>&page[size]=1000`.
   - Expect `200 OK`.
   - Filter `data[]` to entries whose `id` is in
     `{ {feature_a.id}, {feature_b.id}, {feature_c.id} }`, then
     compare the resulting order against the row's expected order.

**Notes:**

- **Why filter the result.** A shared DB carries other features.
  The row asserts the relative order of the three fixtures, not the
  absolute contents of `data[]`.
- **Updated-at ordering.** If `updated_at` rows depend on a known
  modification time, perform a no-op PATCH (e.g.
  `unit_cost: null` on a feature with no unit_cost) to bump
  `updated_at` predictably before the assertion.
- **No `:asc` / `:desc` suffix.** Verified 2026-04-28 against a live
  server: `sort=key:desc` (and the other `:desc` permutations)
  returns 400 `"invalid feature order by: key:desc"` — the suffix is
  not parsed and the entire string is treated as the field name.
  Descending-sort scenarios are deferred until the server grows
  syntax for it (whether `:desc` suffix, `-key` prefix, or a separate
  `order` query param). Re-add rows once the syntax is pinned.

---

## Scenario: feature_list_sort_invalid_field_rejected

```yaml
id: feature_list_sort_invalid_field_rejected
endpoints:
  - GET /features
entities: [feature]
tags: [list, sort, validation, single-request]
```

**Intent:** `sort=<unknown_field>` returns `400 Bad Request`.

**Fixtures:** *(none)*.

**Steps:**

1. **List with bad sort field.** `GET /features?sort=banana`.
   - Expect `400 Bad Request`.
   - Expect **detail contains** `"invalid feature order by: banana"`.

**Notes:**

- **Error shape:** detail-substring at status 400.
- The valid set is `{ key, name, created_at, updated_at }`. The
  server does not parse a direction suffix — it treats the entire
  `sort` value as a field name (verified 2026-04-28). See
  `feature_list_sort_by_supported_fields` for context.

---

## Scenario: feature_list_filter_malformed_rejected

```yaml
id: feature_list_filter_malformed_rejected
endpoints:
  - GET /features
entities: [feature]
tags: [list, filter, validation, single-request]
```

**Intent:** A filter expression the OpenAPI binder cannot parse
returns `400 Bad Request` with `invalid_parameters[]` populated.
This is the schema-rule shape, not the domain shape.

**Rows:**

| # | Query | Expect `field` | Expect `rule` | Expect `reason` (substring) |
|---|---|---|---|---|
| 1 | `filter[meter_id][eq]=not-a-ulid` | `"meter_id"` | `"anyOf"` | — |
| 2 | `filter[meter_id][zz]=01HXYZ...` | `"filter"` | `"format"` | `"filter[meter_id][zz]: unsupported operator"` |

**Steps per row:**

1. **List with malformed filter.** `GET /features?<row's query>`.
   - Expect `400 Bad Request`.
   - Expect `invalid_parameters[0].field` matches the row's expected
     value, `invalid_parameters[0].rule` matches the row's expected
     rule string, and (row 2 only) the response body contains the
     row's expected reason substring.

**Notes:**

- **Error shape:** schema-rule (`invalid_parameters[]`), not
  detail-substring. Schema validation rejects before the request
  body is processed.
- **Why `anyOf` for row 1.** `filter[meter_id]` is typed as
  `ULIDFieldFilter` — a TypeSpec union of either a bare ULID string
  or an operator object (`{eq?, neq?, ...}`). Unions compile to
  OpenAPI `anyOf`. When the value matches neither branch, the
  binder reports the umbrella rule `anyOf` rather than which
  branch was closest — by design, `anyOf` rejections don't say
  "you almost matched branch 2."
- **Umbrella-field binding for row 2.** When a sub-operator under
  `filter[<key>]` is unknown, the binder reports the rejection at
  the **umbrella `filter` field**, not at `filter.meter_id`. The
  offending sub-key (e.g. `filter[meter_id][zz]`) only appears in
  the `reason` string. Assertions that want to pin "this rejection
  was about meter_id" must look at `reason` (or `body contains`),
  not `field`.
- **Pinned 2026-04-28** against an OpenMeter deployment running
  behind an OpenAPI binder gateway. Re-pin if the binder layer is
  replaced or upgraded.

---

## Scenario: feature_list_pagination

```yaml
id: feature_list_pagination
endpoints:
  - POST /features
  - GET /features
entities: [feature]
tags: [list, pagination, single-request]
```

**Intent:** `page[number]` + `page[size]` slice the result and
populate the response `meta` block. Page size 1 over three fixtures
yields three pages.

**Fixtures:**
- Three `CreateFeatureRequest`s per **Baseline feature**, each with
  a distinct `key` and a **shared `name` prefix** (e.g.
  `"e2e_pagination_<suffix>"`). The shared prefix makes the per-row
  filter narrow without depending on absolute DB state.

**Steps:**

1. **Provision three features.** Three `POST /features` calls; each
   `201 Created`. Capture the three response bodies.

2. **List page 1, size 1, filtered to fixtures.**
   `GET /features?filter[name][contains]=<shared-prefix>&page[number]=1&page[size]=1`.
   - Expect `200 OK`.
   - Expect `data[]` has length `1`.
   - Expect `meta.page.total` is `3`.
   - Expect `meta.page.number` is `1`.
   - Expect `meta.page.size` is `1`.

3. **List page 2.** Same filter, `page[number]=2&page[size]=1`.
   - Expect `200 OK`.
   - Expect `data[]` has length `1`.
   - Expect `meta.page.number` is `2`.
   - Expect the entry differs from page 1's entry by `id`.

4. **List page 3.** Same filter, `page[number]=3&page[size]=1`.
   - Expect `200 OK`.
   - Expect `data[]` has length `1`.
   - Expect `meta.page.number` is `3`.
   - Expect the entry differs from pages 1 and 2 by `id`.

5. **Assert union covers fixtures.** The three captured ids equal the
   union of the three pages' single-entry ids (in some order).

**Notes:**

- **Why `contains` on the name** instead of `eq` per-feature? `eq`
  collapses to one row (no pagination to test). The shared-prefix
  `contains` filter narrows to exactly the three fixtures.
- **Pagination meta shape:** `meta.page.{number, size, total}` per
  the `PaginatedMeta` / `PageMeta` schema in the OpenAPI spec.

---

## Scenario: feature_cost_query_happy_path

```yaml
id: feature_cost_query_happy_path
endpoints:
  - POST /meters
  - POST /features
  - POST /features/{id}/cost/query
entities: [feature, meter]
tags: [cost-query, single-request]
```

**Intent:** `POST /features/{id}/cost/query` with an empty body on a
metered feature with a manual unit cost returns `200 OK` and a
`FeatureCostQueryResult` with at least one row. Zero usage yields
`cost: "0"`, `usage: "0"`, `currency: "USD"`, and the default time
window `1970-01-01T00:00:00Z` → `1970-01-01T00:01:00Z`.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter**.
- A `CreateFeatureRequest` per **Baseline feature** with:
  - `meter`: `{ id: {meter.id} }`.
  - `unit_cost`: per **Baseline manual unit cost**.

**Steps:**

1. **Create meter.** `POST /meters` with the meter fixture.
   - Expect `201 Created`.

   Captures:
   - `meter` ← `response.body`

2. **Create metered feature with manual unit cost.**
   `POST /features` with the feature fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

3. **Query cost with empty body.**
   `POST /features/{feature.id}/cost/query` with body `{}`.
   - Expect `200 OK`.
   - Expect `data` is a non-empty list (at least one row).
   - Expect `data[0].cost` is `"0"` (or `null` if the row has no
     resolved pricing — both are valid for zero usage; pin
     whichever the live server emits).
   - Expect `data[0].usage` is `"0"`.
   - Expect `data[0].currency` is `"USD"`.
   - Expect `data[0].from` is `"1970-01-01T00:00:00Z"`.
   - Expect `data[0].to` is `"1970-01-01T00:01:00Z"`.

**Notes:**

- **Validation moment:** none — this is a positive read.
- **Empty body.** The endpoint accepts a missing body via
  `request.ParseOptionalBody`; an empty `{}` is the canonical empty
  shape. The default time window when neither `from` nor `to` is
  provided is the unix epoch + 60s — confirmed against a live
  server.
- **Manual unit cost makes `cost` resolvable.** With an LLM unit
  cost, `cost` may be `null` for unresolved pricing. Manual costs
  always resolve as `usage × amount` (here `0 × 5 = 0`).

---

## Scenario: feature_cost_query_nonexistent_feature_returns_404

```yaml
id: feature_cost_query_nonexistent_feature_returns_404
endpoints:
  - POST /features/{id}/cost/query
entities: [feature]
tags: [cost-query, single-request, not-found]
```

**Intent:** `POST /features/{id}/cost/query` on an unknown feature id
returns `404` with the standard detail substring.

**Fixtures:** *(none — the request is path-only)*.

**Steps:**

1. **Query cost on missing feature.**
   `POST /features/<random-ULID-not-in-DB>/cost/query` with body `{}`.
   - Expect `404 Not Found`.
   - Expect **detail contains** `"feature not found: <id>"`.

**Notes:**

- **Error shape:** detail-substring.
- The feature is resolved first, so the 404 fires before any meter
  or query-param processing — body shape doesn't matter for this
  failure mode.

---

## Scenario: feature_cost_query_no_meter_rejected

```yaml
id: feature_cost_query_no_meter_rejected
endpoints:
  - POST /features
  - POST /features/{id}/cost/query
entities: [feature]
tags: [cost-query, validation, single-request]
```

**Intent:** `POST /features/{id}/cost/query` on a feature with no
associated meter returns `400 Bad Request`. Cost is meaningless
without usage; the handler rejects up front.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature** (no meter).

**Steps:**

1. **Create static feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Query cost on unmetered feature.**
   `POST /features/{feature.id}/cost/query` with body `{}`.
   - Expect `400 Bad Request`.
   - Expect **detail contains**
     `"feature {feature.key} has no meter associated"`.

**Notes:**

- **Validation moment:** request-time, after feature resolution and
  before any query processing.
- **Error shape:** detail-substring at status 400.

---

## Scenario: feature_archived_excluded_from_list

```yaml
id: feature_archived_excluded_from_list
endpoints:
  - POST /features
  - DELETE /features/{id}
  - GET /features
entities: [feature]
tags: [list, archive, single-request]
```

**Intent:** After `DELETE`, the feature is not returned by a default
`GET /features` call. v3 has no `include_archived` query param, so
archived features are unreachable via the public API.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature**.

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Delete feature.** `DELETE /features/{feature.id}`.
   - Expect `204 No Content`.

3. **List features and confirm absence.**
   `GET /features?page[size]=1000`.
   - Expect `200 OK`.
   - Expect `data[]` does **not** contain an entry with
     `id == {feature.id}`.

**Notes:**

- **No include-archived param.** Known parity gap with v1: the v1 list
  endpoint accepted `include_archived=true`; v3 does not.

---

## Scenario: feature_archived_get_returns_404

```yaml
id: feature_archived_get_returns_404
endpoints:
  - POST /features
  - DELETE /features/{id}
  - GET /features/{id}
entities: [feature]
tags: [archive, single-request, not-found]
```

**Intent:** After `DELETE`, `GET /features/{id}` returns `404` with
the standard detail substring. The TypeSpec declares `Common.Gone`
(410) as a possible response, but the live server always returns 404.

**Fixtures:**
- A `CreateFeatureRequest` per **Baseline feature**.

**Steps:**

1. **Create feature.** `POST /features` with the fixture.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

2. **Delete feature.** `DELETE /features/{feature.id}`.
   - Expect `204 No Content`.

3. **Get archived feature.** `GET /features/{feature.id}`.
   - Expect `404 Not Found`.
   - Expect **detail contains**
     `"feature not found: {feature.id}"`.

**Notes:**

- **404, not 410.** TypeSpec lists both
  `Common.NotFound` (404) and `Common.Gone` (410); the server pins
  404. Confirmed against a live server. If the server later starts
  returning 410, this scenario is the regression test.
- **Error shape:** detail-substring.

---

## Scenario: feature_llm_pricing_absent_when_unresolved

```yaml
id: feature_llm_pricing_absent_when_unresolved
endpoints:
  - POST /meters
  - POST /features
  - GET /features/{id}
entities: [feature, meter]
tags: [llm, get, single-request]
```

**Intent:** GET on an LLM feature whose `provider` + `model` cannot
be resolved against the LLM cost database returns `200 OK` with the
`pricing` block silently absent on the feature's `unit_cost`. No
error, no warning — just an unenriched response.

**Fixtures:**
- A `CreateMeterRequest` per **Baseline LLM-capable meter**.
- A `CreateFeatureRequest` per **Baseline feature** with:
  - `meter`: `{ id: {meter.id} }`.
  - `unit_cost`: a `BillingFeatureLLMUnitCost` with:
    - `type`: `"llm"`.
    - `provider`: `"unknown_provider_<unique>"`.
    - `model`: `"unknown_model_<unique>"`.
    - `token_type`: `"input"`.

**Steps:**

1. **Create meter.** `POST /meters`.
   - Expect `201 Created`.

   Captures:
   - `meter` ← `response.body`

2. **Create LLM feature with unresolvable provider/model.**
   `POST /features`.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

3. **GET the feature.** `GET /features/{feature.id}`.
   - Expect `200 OK`.
   - Expect `unit_cost.type` is `"llm"`.
   - Expect `unit_cost.provider` equals the fixture provider.
   - Expect `unit_cost.model` equals the fixture model.
   - Expect `unit_cost.pricing` is **absent**.

**Notes:**

- **Failure mode is silent.** `resolveLLMPricing` returns `nil` when
  the provider or model can't be looked up; the response is built
  without the pricing block.
- **Error shape:** none — this is a positive scenario over an
  intentionally unresolvable input.

---

## Scenario: feature_llm_pricing_enriched_on_get

```yaml
id: feature_llm_pricing_enriched_on_get
endpoints:
  - POST /meters
  - POST /features
  - GET /features/{id}
entities: [feature, meter]
tags: [llm, get, single-request, needs-verify, skipped]
status: skipped
```

**Status: SKIPPED.** Environment-gated; deferred until the e2e stack
provisions a deterministic LLM-cost seed (a known `(openai, gpt-4)`
price). Do not generate a test from this scenario. Re-enable by
removing this block once seeding is in place.

**Intent:** GET on an LLM feature whose `provider` + `model` resolve
against the LLM cost database returns `200 OK` with `unit_cost.pricing`
populated from the database (`input_per_token`, `output_per_token`, …).

**Fixtures:**
- A `CreateMeterRequest` per **Baseline LLM-capable meter**.
- A `CreateFeatureRequest` per **Baseline feature** with:
  - `meter`: `{ id: {meter.id} }`.
  - `unit_cost`: per **Baseline LLM unit cost (static)** — that is,
    `provider: "openai"`, `model: "gpt-4"`, `token_type: "input"`.
- **Environmental precondition:** the LLM cost database must contain
  a price for `(provider: openai, model: gpt-4)` in the test
  namespace. Without that seed the scenario degrades to
  `feature_llm_pricing_absent_when_unresolved`.

**Steps:**

1. **Create meter.** `POST /meters`.
   - Expect `201 Created`.

   Captures:
   - `meter` ← `response.body`

2. **Create LLM feature with resolvable provider/model.**
   `POST /features`.
   - Expect `201 Created`.

   Captures:
   - `feature` ← `response.body`

3. **GET the feature.** `GET /features/{feature.id}`.
   - Expect `200 OK`.
   - Expect `unit_cost.type` is `"llm"`.
   - Expect `unit_cost.pricing` is present.
   - Expect `unit_cost.pricing.input_per_token` parses as a non-negative
     decimal string.
   - Expect `unit_cost.pricing.output_per_token` parses as a
     non-negative decimal string.

**Notes:**

- **NEEDS-VERIFY: requires seed data in the e2e environment's LLM
  cost DB.** Until the e2e stack provisions a known
  (`openai`, `gpt-4`) price, this scenario isn't deterministically
  exercisable from the API alone. Two resolutions:
  1. Add a fixture step that seeds the price via a public LLM-cost
     endpoint (if one exists).
  2. Tag this scenario as environment-gated and skip when the seed
     is missing.
- **Error shape:** none — positive scenario.
