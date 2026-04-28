---
name: create-e2e-test
description: Generate Insomnia e2e tests for any OpenMeter v3 API endpoint. Derives test shape from api/v3/openapi.yaml, api/v3/handlers/, openmeter/ domain modules, and pkg/ utilities.
user-invocable: true
argument-hint: "[test name] [description of what to test]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Create E2E Test

You are helping the user add a new Insomnia-based e2e test for OpenMeter's v3 API.

## Input

The user provides:
- **Test name** — short label (e.g. `"Billing: create invoice"`)
- **Description** — what the test should exercise

Parse both from `args`. If unclear, ask once.

## Step 1 — Research the endpoint

Before writing any test code, read the relevant sources in this order:

### 1a. OpenAPI spec — `api/v3/openapi.yaml`

This is the ground truth for paths, methods, required fields, and status codes.

```bash
# All resource groups
grep -E '^  /openmeter/' api/v3/openapi.yaml | sort

# Paths for one domain
grep -n '/openmeter/currencies' api/v3/openapi.yaml

# Schema for request/response
grep -n -A 30 'CreateXxxRequest:' api/v3/openapi.yaml
grep -n -A 20 'BillingXxx:' api/v3/openapi.yaml
```

Key things to extract:
- **Path** — e.g. `/openmeter/plans` → suffix after `/openmeter` is what you append to `BASE`
- **Methods + status codes** — POST→201, GET→200, PUT→200, DELETE→204, etc.
- **Request body schema** — required fields, field types, enums
- **Response body schema** — `id`, output-only fields, discriminator types
- **Pagination** — all list responses use `{ data: [...], meta: {...} }` (not `items`)

The `BASE` constant in test scripts is always `…/api/v3/openmeter`, so a path like `/openmeter/plans/{id}` becomes `/plans/${id}` in test code.

### 1b. TypeSpec source — `api/spec/packages/aip/src/<domain>/`

TypeSpec is the source of truth that generates the OpenAPI. Read it when the OpenAPI schema is dense or the field semantics are unclear.

```
api/spec/packages/aip/src/
├── billing/
├── currencies/
├── customers/
├── entitlements/
├── features/
├── invoices/
├── meters/
├── productcatalog/   ← plans, addons, plan-addons
├── subscriptions/
├── tax/
└── shared/
```

### 1c. Handler — `api/v3/handlers/<domain>/`

Each domain has a handler package:

```
api/v3/handlers/
├── addons/           plans/        currencies/
├── apps/             subscriptions/ taxcodes/
├── customers/        meters/       features/
└── billingprofiles/  llmcost/      events/
```

Read `handler.go` for the interface, then the operation files (`create.go`, `list.go`, `get.go`, `update.go`, `delete.go`, etc.) to understand:
- What the handler decodes from the request (body fields, path params, query params)
- What the handler returns (status code, response type)
- How errors map to HTTP codes (conflict → 409, not found → 404, bad request → 400)

Read `convert.go` to understand how domain types map to API response shapes (especially discriminated unions).

### 1d. Domain module — `openmeter/<domain>/`

```
openmeter/
├── billing/          customer/     entitlement/
├── currencies/       meter/        productcatalog/
├── subscription/     taxcode/      notification/
└── …
```

Read `models.go` (or equivalent) to understand:
- What fields are required by `Validate()`
- What constraints exist that the OpenAPI schema may not fully capture
- Enum values and their string representations

Read `service.go` to see the service interface — what operations exist.

### 1e. Shared packages — `pkg/`

```
pkg/
├── pagination/    ← Page, Result[T], pagination query params
├── models/        ← ManagedModel (id, created_at, updated_at), NamespacedID
├── currencyx/     ← currency codes and utilities
├── filter/        ← AIP filter helpers
└── …
```

`pkg/pagination` defines the page-based query params: `page[size]=N&page[number]=N`.

## Step 2 — Choose collection file

Tests live in `e2e/insomnia/`. The runner (`run.sh`) auto-discovers all `*.json` files.

| Scenario | Action |
|----------|--------|
| Test fits an existing collection | Add a new `unit_test` (or `unit_test_suite`) to that file |
| Test is for a new domain | Create `e2e/insomnia/openmeter-<domain>.json` |

Existing collections:
- `openmeter-v3-product-catalog.json` — plans, addons, plan-addons
- `openmeter-v3-currencies.json` — currencies, cost bases

## Step 3 — Write the test

### Collection skeleton (new file)

```json
{
  "__export_format": 4,
  "__export_date": "2025-01-01T00:00:00.000Z",
  "__export_source": "insomnia.desktop.app:v10.0.0",
  "resources": [
    {
      "_id": "wrk_<domain>",
      "_type": "workspace",
      "name": "OpenMeter <Domain>",
      "scope": "collection",
      "parentId": null
    },
    {
      "_id": "env_<domain>_base",
      "_type": "environment",
      "name": "Base Environment",
      "data": {},
      "parentId": "wrk_<domain>",
      "color": null,
      "isPrivate": false,
      "metaSortKey": -1
    },
    {
      "_id": "env_<domain>_local",
      "_type": "environment",
      "name": "Local Dev",
      "data": { "base_url": "http://localhost:8888" },
      "parentId": "env_<domain>_base",
      "color": "#00AA00",
      "isPrivate": false,
      "metaSortKey": -2
    },
    {
      "_id": "uts_<suite>",
      "_type": "unit_test_suite",
      "name": "<Suite Name>",
      "parentId": "wrk_<domain>"
    },
    {
      "_id": "ut_<test>",
      "_type": "unit_test",
      "name": "<Test Name>",
      "requestId": null,
      "parentId": "uts_<suite>",
      "code": "/* test script */"
    }
  ]
}
```

**ID rules**: unique across all resources; use prefixes `wrk_`, `env_`, `uts_`, `ut_` + short slug.

**Environment hierarchy**: `workspace → base env → sub-env`. `--env "Local Dev"` selects the sub-env; `process.env` overrides apply on top.

### Test script (inso v12 API)

Every `unit_test.code` is self-contained JavaScript. Globals: `fetch`, `chai` (`chai.expect`), `process.env`.

**Standard preamble** — copy verbatim for every test:

```javascript
const { expect } = chai;
const BASE = (process.env.OM_BASE_URL || 'http://localhost:8888').replace(/\/$/, '') + '/api/v3/openmeter';
const KEY = process.env.OM_API_KEY || '';
const HDRS = { 'Content-Type': 'application/json', ...(KEY ? { 'Authorization': `Bearer ${KEY}` } : {}) };
const sfx = `${Date.now()}_${Math.floor(Math.random() * 9999)}`;

async function api(method, path, body) {
  const resp = await fetch(`${BASE}${path}`, {
    method,
    headers: HDRS,
    ...(body !== undefined ? { body: JSON.stringify(body) } : {})
  });
  let json = null;
  try { json = await resp.json(); } catch (_) {}
  return { status: resp.status, json };
}
```

**Assertion rules**:
- Always use chai BDD with a step label: `expect(val, '1. Create → 201').to.equal(201)`
- Number steps sequentially: `'1. …'`, `'2. …'`, `'3. …'`
- `to.not.be.ok` not `to.be.null` — API omits null fields entirely (field is `undefined` in JS)
- `to.be.at.least(400)` + `to.be.below(500)` — when exact 4xx code is uncertain
- State (IDs, keys) flows naturally between steps — no need for outer `let` reassignment other than `r`

**Lifecycle template**:

```javascript
// 1. Create
let r = await api('POST', '/<resources>', { key: `test_${sfx}`, name: `Test ${sfx}` });
expect(r.status, '1. Create → 201').to.equal(201);
const id = r.json.id;
expect(id, '1. id present').to.be.ok;

// 2. Get
r = await api('GET', `/<resources>/${id}`);
expect(r.status, '2. Get → 200').to.equal(200);
expect(r.json.name, '2. name').to.equal(`Test ${sfx}`);

// 3. List — verify appears
r = await api('GET', '/<resources>?page[size]=100');
expect(r.status, '3. List → 200').to.equal(200);
expect((r.json.data || []).some(x => x.id === id), '3. in list').to.be.true;

// 4. Update
r = await api('PUT', `/<resources>/${id}`, { name: `Updated ${sfx}` });
expect(r.status, '4. Update → 200').to.equal(200);
expect(r.json.name, '4. updated name').to.equal(`Updated ${sfx}`);

// 5. Delete
r = await api('DELETE', `/<resources>/${id}`);
expect(r.status, '5. Delete → 204').to.equal(204);

// 6. Get after delete → 404
r = await api('GET', `/<resources>/${id}`);
expect(r.status, '6. Gone → 404').to.equal(404);
```

**Pagination**: list responses are always `{ data: [...], meta: { page: { ... } } }`. Use `r.json.data` not `r.json.items`.

**Collision avoidance**: append `sfx` to any `key`, `name`, or `code` field that must be unique per test run.

**No delete endpoint?** Skip the cleanup step. Note in a comment that the resource persists.

**Soft-delete vs hard-delete**: not all resources 404 after DELETE. Always check before asserting:

| Resource | After DELETE |
|---|---|
| Plans, meters | `GET` → 200 with `deleted_at` set (soft-delete) |
| Features | `GET` → 404 (hard-delete) |

When in doubt, check the domain model for `ManagedResource` (has `deleted_at` → soft) vs a hard removal in the adapter.

## Step 4 — Validate and run

```bash
# Validate JSON
python3 -c "import json; json.load(open('e2e/insomnia/<file>.json')); print('OK')"

# Run all collections
make test-insomnia

# Run one collection
inso --ci -w e2e/insomnia/<file>.json run test "<Suite Name>" --env "Local Dev"
```

Prerequisites: `make up` (Postgres, Kafka, ClickHouse), `make server`.

## Reference: Go e2e → Insomnia translation

Existing Go e2e tests under `e2e/` map directly to Insomnia test steps:

| Go | Insomnia JS |
|----|-------------|
| `client.CreateXxxWithResponse(ctx, body)` | `api('POST', '/xxx', body)` |
| `resp.JSON201.Id` | `r.json.id` |
| `require.Equal(t, expected, actual)` | `expect(actual, 'label').to.equal(expected)` |
| `t.Cleanup(func() { delete... })` | final DELETE step |
| `NewTestNamespace(t)` / namespace isolation | `sfx` suffix on keys/names |

Go test files to check first:
```
e2e/addons_v3_test.go        → addons
e2e/plans_v3_test.go         → plans
e2e/planaddons_v3_test.go    → plan-addons
e2e/entitlement_test.go      → entitlements
e2e/productcatalog_test.go   → product catalog
e2e/e2e_test.go              → meters, ingest, subjects (v1 SDK style)
```

## Debugging: filter query params returning 400

If a `GET` with `filter[field]=value` returns 400 unexpectedly, the cause is almost always in `api/v3/filters/parse.go`, not in kin-openapi or the handler.

`filters.Parse` dispatches on the Go type of each filter field:

| Filter field Go type | Handler |
|---|---|
| `*string` | `parseStringPtr` |
| `*CurrencyCode` (`= string` alias) | `parseStringPtr` — alias resolves to `*string` |
| `*BillingCurrencyType` (named `type T string`) | `parseStringPtrTyped` (added in #4180) |
| `*bool`, `*int`, etc. | dedicated parsers |
| other | error: "unsupported filter field type" → 400 |

The `default` case returns an error for any unsupported type. If a new domain adds a filter field with a named string type (e.g. `type MyStatus string`) and `parseStringPtrTyped` is not reached, the list endpoint will return 400 for any filtered request. Check that the field type falls into one of the supported branches; if not, `parseStringPtrTyped` needs to cover it (or the field type should use a plain `string` alias).

## Error code reference

Handler error-wrapping varies by domain. Common patterns:

| Handler | Wrapping | Effective codes |
|---|---|---|
| `CreateCurrency` | Wraps ALL service errors as `ConflictError` | Validation → 400 (unwraps through `BaseAPIError.Unwrap()`) ; duplicate code → 409 |
| `CreateCostBasis` | No extra wrapping; uses default `commonhttp.GenericErrorEncoder()` | Validation → 400, FK / duplicate → 409 |

When asserting error codes in validation tests, trace the full error chain: service → handler wrapping → error encoder. `BaseAPIError.Unwrap()` exposes `UnderlyingError`, so `errors.As` finds `*GenericValidationError` through a `*ConflictError` wrapper, which is why validation errors still resolve to 400 even when the handler wraps everything as conflict.

## Checklist

1. Read `api/v3/openapi.yaml` — confirm paths, methods, required fields, status codes
2. Read `api/v3/handlers/<domain>/` — handler decode/encode logic, error mapping, convert.go
3. Read `openmeter/<domain>/models.go` — Validate() constraints not visible in OpenAPI
4. Check `e2e/*.go` — reuse scenario structure if a Go test already covers the flow
5. Pick or create a collection file in `e2e/insomnia/`
6. Write test with standard preamble, `sfx` suffix, numbered assertions
7. Validate JSON, then run with `make test-insomnia` or direct `inso` invocation
