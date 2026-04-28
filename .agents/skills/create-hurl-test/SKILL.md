---
name: create-hurl-test
description: Generate a Hurl (.hurl) end-to-end test for an OpenMeter API endpoint. Covers both v3 (AIP-style filtering, api/v3/openapi.yaml) and v1 (api/openapi.yaml, api/spec/packages/legacy/) APIs. Output files go in e2e/hurl/ named openmeter-<api-version>-<domain>.hurl. Use this when someone wants to create an HTTP-level e2e test using the Hurl tool.
user-invocable: true
argument-hint: "<test name> <description>"
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# Create Hurl E2E Test

Generate a [Hurl](https://hurl.dev/) end-to-end test for an OpenMeter API endpoint.

**Args:**
- `$1` — test name (e.g. `"Meter Lifecycle"`)
- `$2` — description of what to test (e.g. `"create, get, list, update, delete a meter"`)

Parse both from `args`. If unclear, ask once.

## Hurl Format Primer

A `.hurl` file is a sequence of HTTP entries separated by blank lines. Each entry has:

```hurl
# Optional comment
METHOD url
[Headers]
Header-Name: value

[Request body — for POST/PUT]
```json
{ "key": "value" }
```

HTTP status_code
[Asserts]
jsonpath "$.field" == "value"

[Captures]
var_name: jsonpath "$.id"
```

Key syntax rules:
- Variables: `{{variable_name}}` — injected via CLI `--variable key=value` or captured from previous responses
- `[QueryStringParams]` section handles `deepObject` style filters (`filter[key]=value`) — always prefer this over URL encoding
- `[Captures]` binds response values to variables for later requests
- `[Asserts]` holds multiple assertions; jsonpath queries use `$.field` syntax
- Requests within one file run sequentially; blank lines between entries
- Comments: `#` prefix

## File Location and Naming

```
e2e/hurl/openmeter-v3-<domain>.hurl    # v3 API (api/v3/openapi.yaml)
e2e/hurl/openmeter-v1-<domain>.hurl    # v1 API (api/openapi.yaml)
```

Create `e2e/hurl/` if it doesn't exist.

## Unique Keys / Run Isolation

Hurl has no built-in random generation. Pass a unique suffix at CLI time:

```bash
hurl --test \
  --variable base_url=http://localhost:8888 \
  --variable api_key="" \
  --variable run_id=$(date +%s%N | head -c 13) \
  e2e/hurl/openmeter-v3-meters.hurl
```

In the `.hurl` file, append `{{run_id}}` to any field that must be unique per run:

```
"key": "test_meter_{{run_id}}"
"name": "Test Meter {{run_id}}"
```

## Standard Variables

Every test file uses these variables (passed via CLI `--variable`):

| Variable | Default at CLI | Purpose |
|---|---|---|
| `base_url` | `http://localhost:8888` | Server base URL (no trailing slash) |
| `api_key` | `""` (empty = no auth) | Bearer token; empty value omits auth effectively |
| `run_id` | `$(date +%s%N \| head -c 13)` | Unique suffix to avoid key collisions |

Auth header in every request (empty value is a valid no-op for local dev):

```hurl
Authorization: Bearer {{api_key}}
Content-Type: application/json
```

## Step 1 — Research the Endpoint

Read sources in this order before writing any Hurl code:

### 1a. OpenAPI spec

**v3:** `api/v3/openapi.yaml`
**v1:** `api/openapi.yaml`

Extract:
- Path (e.g. `/openmeter/meters`) → append to `{{base_url}}/api/v3`
- Methods + status codes (POST→201, GET→200, PUT→200, DELETE→204, etc.)
- Required fields and their types
- Filter params and their `style` (deepObject for v3 AIP filters)
- Pagination shape: always `{ data: [...], meta: { page: { ... } } }` for v3 list responses

```bash
# Find paths for a domain
grep -n '/openmeter/meters' api/v3/openapi.yaml

# Find schema
grep -n -A 30 'CreateMeterRequest:' api/v3/openapi.yaml
```

### 1b. TypeSpec source

**v3:** `api/spec/packages/aip/src/<domain>/`
**v1:** `api/spec/packages/legacy/src/<domain>/`

Read when OpenAPI field semantics are unclear.

### 1c. Handler

**v3:** `api/v3/handlers/<domain>/`

Key files: `create.go`, `list.go`, `get.go`, `update.go`, `delete.go`, `convert.go`, `errors.go`

Tells you:
- Which validators fire (look for `validateDimensions*`, `request.ParseBody`, etc.)
- Error shape used: `apierrors.GenericErrorEncoder()` → domain code via `extensions.validationErrors[].code`; `BaseAPIError` → detail substring via `problem.detail`; schema binder → `invalid_parameters[].rule`
- Which fields are accepted vs. ignored

### 1d. Domain module

`openmeter/<domain>/`

Look for `Validate()`, validation error codes, enum values, constraint logic.

## Step 2 — Choose or Create the Hurl File

| Scenario | Action |
|---|---|
| Domain has no `.hurl` file yet | Create `e2e/hurl/openmeter-v3-<domain>.hurl` |
| File exists, adding a new test | Append below the last entry (blank line separator) |

## Step 3 — Write the Hurl Test

### Base URL Path

v3 path = `{{base_url}}/api/v3` + OpenAPI path suffix without `/openmeter` prefix:

> `/openmeter/meters` → `{{base_url}}/api/v3/openmeter/meters`

### Lifecycle Template (v3)

```hurl
# ── 1. Create ─────────────────────────────────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "test_meter_{{run_id}}",
  "name": "Test Meter {{run_id}}",
  "aggregation": "sum",
  "event_type": "api_request",
  "value_property": "$.duration_ms"
}
```
HTTP 201
[Captures]
meter_id: jsonpath "$.id"
meter_key: jsonpath "$.key"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.key" == "test_meter_{{run_id}}"
jsonpath "$.status" == "active"
jsonpath "$.deleted_at" not exists

# ── 2. Get by ID ──────────────────────────────────────────────────────────────
GET {{base_url}}/api/v3/openmeter/meters/{{meter_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{meter_id}}"
jsonpath "$.key" == "test_meter_{{run_id}}"

# ── 3. List — verify appears ──────────────────────────────────────────────────
# Use JSONPath filter expression, NOT $.data[*].id contains "{{id}}"
# $.data[*].id returns a bare string (not array) when exactly 1 item exists,
# making the contains predicate do a substring check instead of membership check.
GET {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
[QueryStringParams]
page[size]: 1000
HTTP 200
[Asserts]
jsonpath "$.data" isCollection
jsonpath "$.data[?(@.id=='{{meter_id}}')].id" == "{{meter_id}}"

# ── 4. List — filter by key ───────────────────────────────────────────────────
GET {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
[QueryStringParams]
filter[key]: test_meter_{{run_id}}
page[size]: 10
HTTP 200
[Asserts]
jsonpath "$.data" count == 1
jsonpath "$.data[0].id" == "{{meter_id}}"

# ── 5. Update ─────────────────────────────────────────────────────────────────
PUT {{base_url}}/api/v3/openmeter/meters/{{meter_id}}
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "name": "Updated Meter {{run_id}}"
}
```
HTTP 200
[Asserts]
jsonpath "$.name" == "Updated Meter {{run_id}}"
jsonpath "$.key" == "test_meter_{{run_id}}"

# ── 6. Delete ─────────────────────────────────────────────────────────────────
DELETE {{base_url}}/api/v3/openmeter/meters/{{meter_id}}
Authorization: Bearer {{api_key}}
HTTP 204

# ── 7. Get after delete — soft delete returns 200 + deleted_at ───────────────
GET {{base_url}}/api/v3/openmeter/meters/{{meter_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.deleted_at" isString
```

### Validation Test Template (single-request)

```hurl
# ── Validation: count aggregation must not have value_property ───────────────
POST {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "test_count_bad_{{run_id}}",
  "name": "Count Bad {{run_id}}",
  "aggregation": "count",
  "event_type": "api_request",
  "value_property": "$.duration_ms"
}
```
HTTP 400
```

### Asserting Error Shapes

v3 uses three error shapes — pick the right one per handler:

| Shape | How server returns it | Hurl assertion |
|---|---|---|
| Domain code | `extensions.validationErrors[].code` | `jsonpath "$.extensions.validationErrors[0].code" == "reserved_dimension"` |
| Detail substring | `problem.detail` (free text) | `jsonpath "$.detail" contains "reserved"` |
| Schema rule | `invalid_parameters[].rule` | `jsonpath "$.extensions.invalidParameters[0].rule" == "pattern"` |

Check the handler's `errors.go` for defined error codes (e.g. `ErrCodeReservedDimension = "reserved_dimension"`).

**Detail substring gotcha:** `apierrors.NewConflictError(ctx, err, "message")` — the second string argument is NOT the `problem.detail` value. The actual `detail` comes from the underlying error's `Error()` string (e.g. `"conflict error: currency with code X already exists"`). Always use a short substring that appears in the real error, not the handler's label string. When in doubt, run the request once and read the actual `detail` before asserting.

### Asserting List Membership

**Never use `$.data[*].id contains "{{id}}"`.** When the array has exactly 1 element,
jsonpath returns a bare string (not a collection), making `contains` do a substring check
instead of a membership check — it silently passes or fails for the wrong reason.

Use a JSONPath filter expression instead:

```hurl
# ✅ Correct — assert field on filtered result; works regardless of array size
# 0 matches → null → fails; 1 match → string → passes; N matches → array → == fails
jsonpath "$.data[?(@.id=='{{resource_id}}')].id" == "{{resource_id}}"

# ✅ Also correct when you know position (e.g. filter already narrowed to 1 result)
jsonpath "$.data[0].id" == "{{resource_id}}"

# ❌ count predicate on filter result — fails when filter returns single object (not list)
jsonpath "$.data[?(@.id=='{{resource_id}}')]" count == 1

# ❌ Broken for single-item arrays — DO NOT USE
jsonpath "$.data[*].id" contains "{{resource_id}}"
```

Also note: `includes` predicate is deprecated in favour of `contains` — always use `contains`.

## Step 4 — Run Command

Add a run command comment at the top of each new file:

```hurl
# Run: hurl --test \
#   --variable base_url=http://localhost:8888 \
#   --variable api_key="" \
#   --variable run_id=$(date +%s%N | head -c 13) \
#   e2e/hurl/openmeter-v3-meters.hurl
```

Run the test:

```bash
hurl --test \
  --variable base_url=http://localhost:8888 \
  --variable api_key="" \
  --variable run_id=$(date +%s%N | head -c 13) \
  e2e/hurl/openmeter-v3-<domain>.hurl
```

Run all hurl files:

```bash
hurl --test \
  --variable base_url=http://localhost:8888 \
  --variable api_key="" \
  --variable run_id=$(date +%s%N | head -c 13) \
  e2e/hurl/*.hurl
```

## Domain-Specific Knowledge

### Meters (v3) — `POST /openmeter/meters`

Required fields: `key`, `name`, `aggregation`, `event_type`

| Aggregation | `value_property` |
|---|---|
| `count` | Must be absent/omitted — error if present |
| `sum`, `avg`, `min`, `max`, `unique_count`, `latest` | Required — error if absent |

Reserved dimensions (rejected at create and update — domain code `reserved_dimension`):
- `subject`
- `customer_id`

Updatable fields: `name`, `description`, `dimensions`, `labels` (key/aggregation/event_type/value_property are immutable).

**Soft-delete:** `DELETE` returns 204, subsequent `GET` returns 200 with `deleted_at` non-null. Does NOT return 404.

### Plans (v3) — `POST /openmeter/plans`

Soft-delete: same as meters — GET after DELETE returns 200 + `deleted_at`.

Draft/publish lifecycle: plans have `status: draft/active/archived`. Publish via `POST /openmeter/plans/{id}/publish`.

Validation errors surface via `validation_errors[]` on GET (draft-with-errors shape).

### Features (v3) — `POST /openmeter/features`

**Hard-delete:** GET after DELETE returns 404. Contrast with meters/plans.

### Filter Params (v3 AIP style)

Use `[QueryStringParams]` section — not inline URL query string — for deepObject style params:

```hurl
GET {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
[QueryStringParams]
filter[key]: my-meter-key
filter[name]: My Meter
page[size]: 20
page[number]: 1
```

Named string type fields (`*BillingCurrencyType`, etc.) use `parseStringPtrTyped` in the filter parser — they work the same as `*string` from the wire perspective.

### Pagination (v3)

All v3 list responses:
```json
{
  "data": [...],
  "meta": {
    "page": { "size": 20, "number": 1, "total": 42 }
  }
}
```

Default page size is 20. Use `page[size]=1000` when checking if a specific item appears in a potentially large shared dataset.

### Error Response Shape (v3)

```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "human readable message",
  "extensions": {
    "validationErrors": [
      { "code": "reserved_dimension", "field": "dimensions.subject" }
    ]
  }
}
```

## Checklist

1. Read `api/v3/openapi.yaml` (or `api/openapi.yaml` for v1) — paths, methods, fields, status codes
2. Read `api/v3/handlers/<domain>/` — error shapes, which validators fire, convert.go
3. Read `openmeter/<domain>/` — Validate() constraints not always visible in OpenAPI
4. Check for existing `.hurl` file in `e2e/hurl/`; append or create
5. Use `{{run_id}}` suffix on all unique fields
6. Use `[QueryStringParams]` for all filter/pagination params (deepObject style)
7. Assert correct soft-delete vs hard-delete behavior per resource type
8. Validate JSON syntax: `hurl --check <file>`
9. Add run command comment at top of file
