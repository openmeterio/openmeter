# E2E Scenario Specifications — Meters

Natural-language, runner-agnostic description of e2e scenarios for
the `meters` endpoint(s). Each `## Scenario` describes wire-level
behavior (HTTP verb, path, status code, response shape,
`problem+json` error shape) that any downstream runner can translate
to an executable test.

Meters are simple CRUD resources with no draft/publish lifecycle.
Validation fires at create-time (or update-time). The primary
validation constraints are: `aggregation` determines whether
`value_property` is required or forbidden; dimension keys must not
collide with reserved names; JSONPath values must start with `$`.

Error shapes: most validation failures are `GenericValidationError`
→ 400. Reserved-dimension violations use a `ValidationIssue` with
code `reserved_dimension` surfaced as a schema rule in
`invalid_parameters[].rule`.

See the `e2e-nl` skill (`.agents/skills/e2e-nl/`) for format rules
(`references/format.md`) and worked examples (`references/examples.md`).

---

## Scenario list

**p0 — happy path**

- `meter_lifecycle` — full CRUD: create (sum) → get → list → filter by key → update name/description/dimensions → delete → GET returns 200 with `deleted_at` set — shape: lifecycle — priority: p0

**p1 — core validation**

- `meter_create_count_aggregation` — count aggregation: omit value_property → 201; count + value_property → 400; sum without value_property → 400 — shape: single-request — priority: p1
- `meter_create_reserved_dimension` — dimension key `subject` or `customer_id` → 400, schema rule `reserved_dimension` — shape: single-request — priority: p1
- `meter_list_filter_by_name` — `filter[name]=<name>` returns only matching meters — shape: single-request — priority: p1

**p2 — edge cases**

- `meter_get_not_found` — GET non-existent meterId → 404 — shape: single-request — priority: p2
- `meter_duplicate_key` — same key twice → conflict NEEDS-VERIFY: exact status code (409 vs 400) — shape: single-request — priority: p2
- `meter_create_invalid_dimension_key` — dimension key with invalid chars (e.g. `"bad-key"`) → 400 NEEDS-VERIFY: exact error shape — shape: single-request — priority: p2
- `meter_create_dimension_value_equals_value_property` — dimension value same as `value_property` → 400 NEEDS-VERIFY: exact error shape — shape: single-request — priority: p2

---

## Baselines

### Baseline meter — `CreateMeterRequest`

- `key`: unique per run (e.g. append a timestamp+random suffix)
- `name`: `"Test Meter"`
- `aggregation`: `"sum"`
- `event_type`: `"api_call"`
- `value_property`: `"$.tokens"`
- `dimensions`: `{ "model": "$.model" }`

### Baseline count meter — `CreateMeterRequest`

Same as Baseline meter but:
- `aggregation`: `"count"`
- `value_property`: absent (omitted)
- `dimensions`: absent

---

## Scenario: meter_lifecycle

```yaml
id: meter_lifecycle
endpoints:
  - POST /meters
  - GET /meters/{id}
  - GET /meters
  - PUT /meters/{id}
  - DELETE /meters/{id}
entities: [meter]
tags: [lifecycle, crud]
```

**Intent:** A meter moves through full CRUD — create, get, list, filter by key,
update (name / description / dimensions), delete — and after deletion remains retrievable
with `deleted_at` set (soft-delete, same as plans).

**Fixtures:**
- A `CreateMeterRequest` per **Baseline meter**.

**Steps:**

1. **Create meter.** `POST /meters` with the fixture.
   - Expect `201 Created`.
   - Expect `key` equals the fixture key.
   - Expect `aggregation` is `"sum"`.
   - Expect `value_property` is `"$.tokens"`.
   - Expect `dimensions` contains key `"model"` with value `"$.model"`.
   - Expect `status` is absent (meters have no status field).

   Captures:
   - `meter` ← `response.body`

2. **Get by ID.** `GET /meters/{meter.id}`.
   - Expect `200 OK`.
   - Expect `id` equals `{meter.id}`.
   - Expect `name` equals the fixture name.
   - Expect `event_type` is `"api_call"`.

3. **List and find.** `GET /meters?page[size]=100`.
   - Expect `200 OK`.
   - Expect `data[]` contains an entry with `id == {meter.id}`.

4. **Filter by key.** `GET /meters?filter[key]={meter.key}`.
   - Expect `200 OK`.
   - Expect every item in `data[]` has `key == {meter.key}`.
   - Expect `data[]` contains an entry with `id == {meter.id}`.

5. **Update name, description, and dimensions.** `PUT /meters/{meter.id}` with:
   - `name`: `"Updated Meter"` (changed)
   - `description`: `"updated description"`
   - `dimensions`: `{ "model": "$.model", "type": "$.type" }` (extra dimension added)

   Assertions:
   - Expect `200 OK`.
   - Expect `name` is `"Updated Meter"`.
   - Expect `description` is `"updated description"`.
   - Expect `dimensions` contains both `"model"` and `"type"`.
   - Expect `aggregation` is still `"sum"` (immutable field is unchanged).
   - Expect `value_property` is still `"$.tokens"` (immutable field is unchanged).

6. **Delete.** `DELETE /meters/{meter.id}`.
   - Expect `204 No Content`.

7. **Get after delete (soft-delete).** `GET /meters/{meter.id}`.
   - Expect `200 OK`.
   - Expect `deleted_at` is non-null.

**Notes:**
- Meters are **soft-deleted** — the resource remains accessible after DELETE and `deleted_at`
  is set. This mirrors plan behavior and differs from features (which return 404 after delete).
- `aggregation`, `event_type`, and `value_property` are immutable after create; the update
  request type (`UpdateMeterRequest`) only carries `name`, `description`, `dimensions`,
  and `labels`.
- Step 4 uses `filter[key]=<exact-key>` per the TypeSpec doc: "To filter meters by key
  add the following query param: `filter[key]=my-meter-key`".
