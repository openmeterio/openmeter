# Worked Examples — One Per Shape Class

This reference shows each shape class drafted as a real httpYac
`.http` file emitted from an NL scenario in `e2e/specs/features.md`.

Use these as templates when emitting new files. Match the shape class
to the kind of behavior the scenario is pinning, then mirror the
structure with the same density of detail.

| Shape | Example | When to reach for it |
|---|---|---|
| `lifecycle` | `feature_lifecycle` | A primary CRUD flow with chained captures, multiple read-back steps, and a final not-found assertion. |
| `single-request` (detail-substring) | `feature_create_key_must_not_be_ulid` | One request, one rule, one assertion with `?? body contains`. The default for any 4xx with the detail-substring shape. |
| `single-request` (schema-rule) | `feature_list_filter_malformed_rejected` | A 4xx whose error shape is `invalid_parameters[].rule`. Always uses a JS `test()` block. |
| `matrix` | `feature_create_unit_cost_validation_matrix` | A rule with multiple rows, each a self-contained request region. Demonstrates the no-`@loop` pattern. |

The `draft-with-errors` shape doesn't have an example here because
features don't have draft lifecycle semantics. When `plans.md` is
translated, the worked example for that shape lives in this file's
peer in the plans family — emit it then, alongside the plan
scenarios that exercise it.

---

## Project root file (illustrative)

Emitted **once** when bootstrapping `e2e/http/`. Subsequent scenario
emissions leave it untouched. There is no env file — configuration is
shell-env based, mirroring the Go e2e convention
(`OPENMETER_ADDRESS=... go test ./e2e/...`).

### `e2e/http/httpyac.config.js`

```js
// Project-root marker for httpYac. The presence of this file scopes
// httpYac's project root to e2e/http/.
//
// Configuration is shell-env based. Each .http file references
// {{process.env.OPENMETER_ADDRESS}}; httpYac resolves it from process.env at
// send time. No env files, no `--env <name>`.
//
// Add response-log scrubbing here if real auth tokens flow through
// tests. v1 of the skill ships without scrubbing — add intentionally
// when the auth surface is settled.

module.exports = {};
```

---

## Example: `lifecycle` shape — `feature_lifecycle`

**NL spec excerpt** (from `e2e/specs/features.md`):

```yaml
id: feature_lifecycle
endpoints:
  - POST /features
  - GET /features/{id}
  - GET /features
  - PATCH /features/{id}
  - DELETE /features/{id}
```

Steps: create → get → list → patch (set unit_cost) → get → patch
(clear unit_cost) → get → delete → get-after-delete (404 with detail
substring).

**Emitted `.http`** (`e2e/http/features/feature_lifecycle.http`):

```http
# feature_lifecycle
#
# Source: e2e/specs/features.md ## Scenario: feature_lifecycle

@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3
@d__feature_lifecycle__feature_key = lifecycle_{{$timestamp}}_{{$randomInt 0 1000000}}

###
# @name r__feature_lifecycle__create_feature
# @title create feature
POST {{api_base}}/openmeter/features
Content-Type: application/json

{
  "key": "{{d__feature_lifecycle__feature_key}}",
  "name": "Test Feature"
}

?? status == 201
?? js response.parsedBody.key == {{d__feature_lifecycle__feature_key}}
?? js response.parsedBody.name == Test Feature
?? js typeof response.parsedBody.meter == undefined
?? js typeof response.parsedBody.unit_cost == undefined
{{
  exports.v__feature_lifecycle__feature_id = response.parsedBody.id;
}}

###
# @name r__feature_lifecycle__get_feature
# @title get feature by id
# @ref r__feature_lifecycle__create_feature
GET {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}

?? status == 200
?? js response.parsedBody.id == {{v__feature_lifecycle__feature_id}}
?? js response.parsedBody.key == {{d__feature_lifecycle__feature_key}}
?? js response.parsedBody.name == Test Feature

###
# @name r__feature_lifecycle__list_features_find_created
# @title list features and find the created feature
# @ref r__feature_lifecycle__create_feature
GET {{api_base}}/openmeter/features?page[size]=1000

?? status == 200
{{
  const { ok } = require('assert');
  test('created feature is present in list', () => {
    const items = response.parsedBody.data || response.parsedBody.items || [];
    ok(
      items.some(f => f.id === v__feature_lifecycle__feature_id),
      'expected list to contain feature id ' + v__feature_lifecycle__feature_id
    );
  });
}}

###
# @name r__feature_lifecycle__patch_set_unit_cost
# @title patch feature - set manual unit_cost
# @ref r__feature_lifecycle__create_feature
PATCH {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}
Content-Type: application/json

{ "unit_cost": { "type": "manual", "amount": "5" } }

?? status == 200
?? js response.parsedBody.unit_cost.type == manual
{{
  const { ok } = require('assert');
  test('unit_cost.amount is normalized to "5"', () => {
    // Server trims trailing zeros: "5.00" round-trips as "5".
    ok(
      response.parsedBody.unit_cost.amount === '5',
      'expected unit_cost.amount === "5", got ' + response.parsedBody.unit_cost.amount
    );
  });
}}

###
# @name r__feature_lifecycle__get_after_set
# @title get feature - verify unit_cost persisted
# @ref r__feature_lifecycle__patch_set_unit_cost
GET {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}

?? status == 200
?? js response.parsedBody.unit_cost.type == manual
?? js response.parsedBody.unit_cost.amount == 5

###
# @name r__feature_lifecycle__patch_clear_unit_cost
# @title patch feature - clear unit_cost
# @ref r__feature_lifecycle__create_feature
PATCH {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}
Content-Type: application/json

{ "unit_cost": null }

?? status == 200
?? js typeof response.parsedBody.unit_cost == undefined

###
# @name r__feature_lifecycle__get_after_clear
# @title get feature - verify unit_cost cleared
# @ref r__feature_lifecycle__patch_clear_unit_cost
GET {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}

?? status == 200
?? js typeof response.parsedBody.unit_cost == undefined

###
# @name r__feature_lifecycle__delete_feature
# @title delete feature
# @ref r__feature_lifecycle__create_feature
DELETE {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}

?? status == 204

###
# @name r__feature_lifecycle__get_after_delete
# @title get feature after delete returns 404
# @ref r__feature_lifecycle__delete_feature
GET {{api_base}}/openmeter/features/{{v__feature_lifecycle__feature_id}}

?? status == 404
?? body contains "feature not found: {{v__feature_lifecycle__feature_id}}"
```

**Commentary:**

- The file's first comment block names the scenario id and spec
  source path. A reviewer landing in the file knows where to look
  for context within ten seconds.
- `@d__feature_lifecycle__feature_key` is set once at file scope.
  Every request that needs the key reads the same value, so the
  asserted-against name persists across the whole lifecycle.
- The capture is **explicit** — `exports.v__feature_lifecycle__feature_id`
  in a post-response script — not implicit via `@name`. The newline
  immediately after `{{` is required.
- Decimal normalization (`"5.00"` → `"5"`) is asserted via a JS
  `test()` block with a clear failure message. A naive
  `?? body contains "amount\":\"5\""` would be too brittle.
- The list-find step uses a JS `test()` because it's an array
  membership check — the kind of assertion `??` shapes don't cover
  cleanly.
- The final 404 step uses the **detail-substring** shape because the
  NL spec phrases it as `Expect detail contains "feature not found:
  {feature.id}"`. The `{feature.id}` placeholder maps to
  `{{v__feature_lifecycle__feature_id}}`.
- `# @ref` chains every region back to its prerequisite. httpYac
  can run this file standalone (`httpyac send … --filter only-failed`)
  and resolve the ref graph.

---

## Example: `single-request` shape (detail-substring) — `feature_create_key_must_not_be_ulid`

**NL spec excerpt:**

```yaml
id: feature_create_key_must_not_be_ulid
endpoints:
  - POST /features
```

Step: POST a feature whose `key` is a valid ULID literal — expect
`400 Bad Request` with detail containing
`"Feature key cannot be a valid ULID"`.

**Emitted `.http`** (`e2e/http/features/feature_create_key_must_not_be_ulid.http`):

```http
# feature_create_key_must_not_be_ulid
#
# Source: e2e/specs/features.md ## Scenario: feature_create_key_must_not_be_ulid

@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3
@d__feature_create_key_must_not_be_ulid__ulid_key = 01HXYZABCDEFGHJKMNPQRSTVWX

###
# @name r__feature_create_key_must_not_be_ulid__create_with_ulid_key
# @title POST /features with a ULID-shaped key returns 400
POST {{api_base}}/openmeter/features
Content-Type: application/json

{
  "key": "{{d__feature_create_key_must_not_be_ulid__ulid_key}}",
  "name": "Test Feature"
}

?? status == 400
?? body contains "Feature key cannot be a valid ULID"
```

**Commentary:**

- The simplest shape — one request, one status assertion, one
  detail-substring assertion. No captures, no `@ref`, no JS blocks.
- The ULID key is a **fixed literal**, not a dynamic value, because
  the test wants to exercise a specific shape (a ULID) and any random
  ULID is equally good. Using a literal makes the test
  deterministic across runs.
- The `@d__` prefix still applies — all data variables follow the
  same naming convention regardless of complexity.

---

## Example: `single-request` shape (schema-rule) — `feature_list_filter_malformed_rejected`

**NL spec excerpt:**

```yaml
id: feature_list_filter_malformed_rejected
endpoints:
  - GET /features
tags: [list, filter, validation, single-request]
```

Two rows: a malformed ULID filter (row 1) and an unknown filter
operator (row 2). Both expect 400 with `invalid_parameters[]`
populated. Spec pins exact `field` / `rule` values per row; row 2
also pins a `reason` substring, since its `field` binds at the
umbrella `filter` level and the offending sub-key only surfaces in
`reason`.

**Emitted `.http`** (`e2e/http/features/feature_list_filter_malformed_rejected.http`):

```http
# feature_list_filter_malformed_rejected
#
# Source: e2e/specs/features.md ## Scenario: feature_list_filter_malformed_rejected

@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3

###
# @name r__feature_list_filter_malformed_rejected__row1_bad_ulid
# @title row 1: filter[meter_id][eq]=not-a-ulid -> 400 with invalid_parameters
GET {{api_base}}/openmeter/features?filter[meter_id][eq]=not-a-ulid

?? status == 400
?? js response.parsedBody.invalid_parameters[0].field == meter_id
?? js response.parsedBody.invalid_parameters[0].rule == anyOf

###
# @name r__feature_list_filter_malformed_rejected__row2_unknown_operator
# @title row 2: filter[meter_id][zz]=<ulid> -> 400 with invalid_parameters
GET {{api_base}}/openmeter/features?filter[meter_id][zz]=01HXYZABCDEFGHJKMNPQRSTVWX

?? status == 400
?? js response.parsedBody.invalid_parameters[0].field == filter
?? js response.parsedBody.invalid_parameters[0].rule == format
?? body contains "filter[meter_id][zz]: unsupported operator"
```

**Commentary:**

- Two-row scenario, two regions — same pattern as a matrix, just
  smaller. Each region is independent (no `@ref` between them).
- DSL-first: indexed `[0].field` / `[0].rule` access via `?? js`
  beats a `test()` block with `some()` here because the response
  consistently has a single `invalid_parameters` entry. Each `??`
  line emits its own JUnit entry. If a future binder version
  starts emitting multiple entries with non-deterministic
  ordering, swap to `test()` with `some()` (see `format.md`'s
  schema-rule example).
- Row 2 uses `?? body contains "..."` as a **third** assertion to
  pin the offending sub-key, since `field == "filter"` alone
  doesn't say "this was about `meter_id`'s operator." The
  `reason` text carries that detail; substring-on-body is the
  cheapest way to assert it.
- Spec started with `NEEDS-VERIFY` markers on both `rule` values.
  A live run against a deployment with an OpenAPI binder layer in
  front of OpenMeter pinned the actual values (`anyOf`, `format`)
  and revealed the umbrella-field binding for row 2 — both
  observations were fed back into `e2e/specs/features.md`. The
  spec-loose → run-real → tighten-spec → tighten-emit loop is the
  canonical workflow when the spec author can't predict the
  binder's exact rule vocabulary.

---

## Example: `matrix` shape — `feature_create_unit_cost_validation_matrix`

**NL spec excerpt:**

```yaml
id: feature_create_unit_cost_validation_matrix
endpoints:
  - POST /meters
  - POST /features
```

10 rows. Rows 4–10 all need an LLM-capable meter; row 1 (manual
baseline) and row 3 (negative amount) don't. Rows 1–2 are `201
success`; rows 3–9 are `400` handler rejections with the
detail-equality shape (`response.parsedBody.detail` equals
`"validation error: <bare msg>"`); row 10 is a `400` binder
rejection (the OpenAPI schema validator runs ahead of the handler
in this deployment) with the schema-rule shape
(`invalid_parameters[0].rule == "allOf"`).

The full 10-row file is too long to inline here — see
`e2e/http/features/feature_create_unit_cost_validation_matrix.http`
for the complete emitted output. Below is the structure plus the
first three rows and the binder-intercepted final row.

**Emitted `.http`** (excerpt):

```http
# feature_create_unit_cost_validation_matrix
#
# Source: e2e/specs/features.md ## Scenario: feature_create_unit_cost_validation_matrix
#
# Rows 4-10 reuse a single shared LLM-capable meter created at the
# top of the file. The meter is read-only across rows, so sharing
# is safe. Rows 1 and 3 don't need a meter and run independently.

@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3
@d__matrix__meter_key = matrix_meter_{{$timestamp}}_{{$randomInt 0 1000000}}

###
# @name r__matrix__create_shared_meter
# @title create shared LLM-capable meter (used by rows 4-10)
POST {{api_base}}/openmeter/meters
Content-Type: application/json

{
  "key": "{{d__matrix__meter_key}}",
  "name": "Test LLM Meter",
  "event_type": "e2e_llm_test",
  "aggregation": "count",
  "dimensions": {
    "provider": "$.provider",
    "model": "$.model",
    "token_type": "$.token_type"
  }
}

?? status == 201
{{
  exports.v__matrix__meter_id = response.parsedBody.id;
}}

###
# @name r__matrix__row1_manual_baseline_succeeds
# @title row 1: manual baseline (amount=5, no meter) -> 201
POST {{api_base}}/openmeter/features
Content-Type: application/json

{
  "key": "matrix_r1_{{$timestamp}}_{{$randomInt 0 1000000}}",
  "name": "matrix",
  "unit_cost": { "type": "manual", "amount": "5" }
}

?? status == 201

###
# @name r__matrix__row2_llm_baseline_succeeds
# @title row 2: LLM baseline with shared meter -> 201
# @ref r__matrix__create_shared_meter
POST {{api_base}}/openmeter/features
Content-Type: application/json

{
  "key": "matrix_r2_{{$timestamp}}_{{$randomInt 0 1000000}}",
  "name": "matrix",
  "meter": { "id": "{{v__matrix__meter_id}}" },
  "unit_cost": {
    "type": "llm",
    "provider": "openai",
    "model": "gpt-4",
    "token_type": "input"
  }
}

?? status == 201

###
# @name r__matrix__row3_negative_amount_rejected
# @title row 3: manual amount=-1 -> 400 (non-negative rule)
POST {{api_base}}/openmeter/features
Content-Type: application/json

{
  "key": "matrix_r3_{{$timestamp}}_{{$randomInt 0 1000000}}",
  "name": "matrix",
  "unit_cost": { "type": "manual", "amount": "-1" }
}

?? status == 400
?? js response.parsedBody.detail == validation error: manual unit cost amount must be non-negative

# … rows 4-9 follow the same handler-rejection pattern, each
# pinning the full "validation error: <msg>" detail. Row 10 is
# binder-intercepted and uses the schema-rule shape:

###
# @name r__matrix__row10_invalid_token_type_rejected
# @title row 10: LLM with token_type="banana" -> 400 (binder allOf rejection)
# @ref r__matrix__create_shared_meter
#
# The OpenAPI binder validates the enum on token_type before the
# request reaches the handler; the handler's "invalid token_type"
# message is unreachable in deployments with a binder gateway in
# front. Schema-rule shape.
POST {{api_base}}/openmeter/features
Content-Type: application/json

{
  "key": "matrix_r10_{{$timestamp}}_{{$randomInt 0 1000000}}",
  "name": "matrix",
  "meter": { "id": "{{v__matrix__meter_id}}" },
  "unit_cost": {
    "type": "llm",
    "provider": "openai",
    "model": "gpt-4",
    "token_type": "banana"
  }
}

?? status == 400
?? js response.parsedBody.invalid_parameters[0].field == unit_cost
?? js response.parsedBody.invalid_parameters[0].rule == allOf
```

**Commentary:**

- **No `@loop`.** Each row is a separate `r__` region with a
  descriptive slug. The JUnit reporter shows
  `r__matrix__row3_negative_amount_rejected` as the failing test —
  not `iteration #3 of POST /features`.
- **Shared prerequisite.** The LLM-capable meter is created **once**
  at the top, and rows 4–10 depend on it via `# @ref
  r__matrix__create_shared_meter`. The NL spec says "every row that
  needs an LLM unit cost also creates the meter above" — the emit
  consolidates this to one creation, since the meter is read-only
  across rows. This is a deliberate deviation from "fresh fixtures
  per row"; the deviation is documented in the file's preamble
  comment so a reviewer can spot it.
- **Heterogeneous bodies, written in full.** Each row's `unit_cost`
  is a different shape — manual vs LLM, with vs without `meter`,
  required-field-missing vs mutex-violation. No shared base body,
  no conditional fields. The cost is verbosity; the benefit is
  spec-row-to-region 1:1 readability.
- **Detail-equality, not body-contains, for handler rejections.**
  Rows 3–9 use `?? js response.parsedBody.detail == validation
  error: <bare msg>` — pinning the full prefixed string from the
  domain validator's framework wrapper. Avoid `?? body contains
  "..."`: it's documented (`format.md`) as unreliable for
  `application/problem+json` bodies on httpyac 6.16.7.
- **Row 10 is binder-intercepted.** `token_type: "banana"`
  violates the OpenAPI enum, so the binder layer's `allOf`
  validator rejects before the handler runs — the handler's
  `"invalid token_type"` message is unreachable in deployments
  that put an OpenAPI schema validator in front of OpenMeter. The
  row uses the schema-rule shape
  (`invalid_parameters[0].field == "unit_cost"`,
  `rule == "allOf"`), matching the canonical pattern in
  `feature_create_key_must_not_be_ulid` and
  `feature_list_filter_malformed_rejected`. Re-pin if the test is
  re-targeted at a deployment that runs the handler directly with
  no schema validator in front; in that case row 10 reverts to
  detail-substring shape carrying the handler's
  `"invalid token_type"` message.
- **Row 1 has no detail substring** — it's a 201, not a 4xx. Just
  `?? status == 201`. Don't invent assertions when the spec is
  silent.
- **Unique-suffixed keys per row.** Each row's request body uses
  `{{$timestamp}}_{{$randomInt 0 1000000}}` so re-runs against a
  shared DB don't collide.
