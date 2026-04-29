---
name: e2e-playwright
description: Generate API tests using Playwright against OpenMeter's v3 API. Use when creating TypeScript-based API tests that exercise HTTP endpoints over a live server with configurable base URL and optional API key auth. Tests produced by this skill are suitable for contract testing — they verify the HTTP contract (status codes, request/response shapes, required fields, error schemas) as defined in the OpenAPI spec.
user-invocable: true
argument-hint: "[domain to test] [user journey description]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Playwright API Testing

You are helping the user write Playwright API tests for OpenMeter's v3 API. These are TypeScript tests using Playwright's `request` context — no browser required, purely exercising the HTTP contract.

**Before writing tests**, read the relevant sections of `api/v3/openapi.yaml` to learn the exact request/response shapes, required fields, status codes, and error schemas for the endpoints you'll exercise.

## Project Layout

All Playwright tests live under `e2e/playwright/`:

```
e2e/playwright/
├── playwright.config.ts        # Base URL, auth headers, timeout config
├── helpers.ts                  # Shared test utilities (uniqueKey, etc.)
├── tests/
│   └── <domain>/
│       └── <journey>.spec.ts   # One file per user journey
└── package.json
```

If `e2e/playwright/` doesn't exist yet, create it with the scaffolding below before writing the test.

## Scaffolding

### package.json

```json
{
  "name": "openmeter-e2e-playwright",
  "private": true,
  "packageManager": "pnpm@10.33.0",
  "scripts": {
    "test": "playwright test",
    "test:headed": "playwright test --headed"
  },
  "devDependencies": {
    "@faker-js/faker": "^10.0.0",
    "@playwright/test": "^1.44.0",
    "typescript": "^5.4.0"
  }
}
```

### playwright.config.ts

```typescript
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: process.env.OPENMETER_ADDRESS ?? 'http://localhost:8888',
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...(process.env.OPENMETER_API_KEY
        ? { Authorization: `Bearer ${process.env.OPENMETER_API_KEY}` }
        : {}),
    },
    ignoreHTTPSErrors: true,
  },
})
```


## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `OPENMETER_ADDRESS` | `http://localhost:8888` | Server base URL |
| `OPENMETER_API_KEY` | _(unset)_ | Sent as `Authorization: Bearer <key>` when set |

Run tests:

```bash
# Against local dev server (no auth)
cd e2e/playwright && pnpm playwright test

# Against a remote server with auth
OPENMETER_ADDRESS=https://openmeter.cloud OPENMETER_API_KEY=om_key_xxx pnpm playwright test

# Single file
pnpm playwright test tests/customers/create-and-subscribe.spec.ts
```

## API Reference

The canonical reference is `api/v3/openapi.yaml`. Read the relevant `paths:` entries before writing a test:

- All v3 endpoints are under `/api/v3/openmeter/…` (the `servers[0].url` is `http://localhost:{port}/api/v3`, so paths in the spec are relative to that — prepend `/api/v3/openmeter` for the raw fetch).
- Response shapes are in `components/schemas/`.
- Error responses follow RFC 7807 (`application/problem+json`). On 4xx/5xx, parse with `await response.json()` and inspect `type`, `title`, `detail`, `extensions.validationErrors`, or `invalid_parameters`.
- Required vs optional fields are marked in each schema. Pay attention — missing required fields often produce 400 schema errors, not domain errors.

### Finding the right endpoint

**Step 1 — Read the TypeSpec operations file for the domain.**

These files are the source of truth before OpenAPI generation. Each is short, domain-isolated, and shows HTTP verb, `@operationId`, path parameters, and request/response type names at a glance — far easier to scan than the 9307-line openapi.yaml.

```
api/spec/packages/aip/src/customers/operations.tsp
api/spec/packages/aip/src/subscriptions/operations.tsp
api/spec/packages/aip/src/billing/operations.tsp
api/spec/packages/aip/src/meters/operations.tsp
api/spec/packages/aip/src/<domain>/operations.tsp   # one per domain
```

To list all available domains, use `codegraph_files` on `api/spec/packages/aip/src` (or `Glob` if CodeGraph is unavailable).

**Step 2 — Look up exact schema details in `api/v3/openapi.yaml`.**

Once you know the type names from the TypeSpec file, find the precise field names, required/optional markers, and enum values in `components/schemas/`. Use the `Grep` tool (not shell grep) to jump straight to a schema:

```
Grep "CustomerCreateInput" in api/v3/openapi.yaml
```

## Writing Tests

### File naming

One file per user journey under `tests/<domain>/`. Use kebab-case:

```
tests/customers/create-and-list.spec.ts
tests/subscriptions/subscribe-and-cancel.spec.ts
tests/billing/invoice-lifecycle.spec.ts
```

### Test structure

Import directly from `@playwright/test` and use the `request` fixture. Define a `BASE` constant for the API path prefix. Use `faker.string.uuid()` for unique string fields:

```typescript
import { test, expect } from '@playwright/test'
import { faker } from '@faker-js/faker'

const BASE = '/api/v3/openmeter'

test.describe('Customer > create and list', () => {
  test('creates a customer and finds it in the list', async ({ request }) => {
    const key = faker.string.uuid()

    // Create
    const createRes = await request.post(`${BASE}/customers`, {
      data: { name: 'Test Customer', key },
    })
    expect(createRes.status()).toBe(201)
    const customer = await createRes.json()
    expect(customer.key).toBe(key)
    const id: string = customer.id

    // List — bump page size so the new row is on page 1
    const listRes = await request.get(`${BASE}/customers`, {
      params: { 'page[size]': '1000' },
    })
    expect(listRes.status()).toBe(200)
    const { items } = await listRes.json()
    const found = items.find((c: { id: string }) => c.id === id)
    expect(found).toBeDefined()
  })
})
```

### Lifecycle tests (ordered steps sharing state)

When the journey is "create → update → publish → archive → delete", use sequential `test` blocks inside a single `describe` block. Share state through the outer scope:

```typescript
import { test, expect } from '@playwright/test'
import { faker } from '@faker-js/faker'

const BASE = '/api/v3/openmeter'

test.describe('Plan > full lifecycle', () => {
  let planId: string

  test('create draft plan', async ({ request }) => {
    const res = await request.post(`${BASE}/plans`, { data: validPlanBody(faker.string.uuid()) })
    expect(res.status()).toBe(201)
    const plan = await res.json()
    expect(plan.status).toBe('draft')
    planId = plan.id
  })

  test('publish plan', async ({ request }) => {
    expect(planId).toBeTruthy()
    const res = await request.post(`${BASE}/plans/${planId}/publish`, { data: {} })
    expect(res.status()).toBe(200)
    const plan = await res.json()
    expect(plan.status).toBe('active')
  })

  test('archive plan', async ({ request }) => {
    expect(planId).toBeTruthy()
    const res = await request.post(`${BASE}/plans/${planId}/archive`, { data: {} })
    expect(res.status()).toBe(200)
    const plan = await res.json()
    expect(plan.status).toBe('archived')
  })
})
```

> Lifecycle subtests run in declaration order. If one step fails, later steps that depend on `planId` will also fail — this is intentional, not a problem.

### Table-driven validation (independent cases)

For input-validation matrices, use a loop over cases. Each row gets a fresh request context:

```typescript
import { test, expect } from '@playwright/test'
import { faker } from '@faker-js/faker'

const BASE = '/api/v3/openmeter'

const invalidBodies = [
  { label: 'missing name', body: { key: 'k1' }, expectedStatus: 400 },
  { label: 'empty key', body: { name: 'N', key: '' }, expectedStatus: 400 },
]

for (const { label, body, expectedStatus } of invalidBodies) {
  test(`rejects ${label}`, async ({ request }) => {
    const res = await request.post(`${BASE}/customers`, { data: body })
    expect(res.status()).toBe(expectedStatus)
    const problem = await res.json()
    expect(problem.type).toBeDefined()
  })
}
```

### Asserting error shapes

v3 returns three error shapes. Parse the body and pick the right assertion:

```typescript
const problem = await res.json()

// 1. Domain validation (extensions.validationErrors[])
//    Produced by handlers that return models.ValidationIssue
const codes = (problem.extensions?.validationErrors ?? []).map((e: any) => e.code)
expect(codes).toContain('plan_phase_duplicated_key')

// 2. Free-text Detail (BaseAPIError)
//    Produced by apierrors.NewBadRequestError with a plain message
expect(problem.detail).toContain('only Plans in [draft scheduled] can be updated')

// 3. Schema / binder errors (invalid_parameters[])
//    Produced before any handler runs (bad enum, missing required field)
const rules = (problem.invalid_parameters ?? []).map((p: any) => p.rule)
expect(rules).toContain('required')
```

> Tip: if unsure which shape applies, `console.log(await res.json())` on a failing test — the shape tells you which assertion to use.

### Unique keys and collision avoidance

The server DB persists across test re-runs. Always generate unique fixture data with faker:

```typescript
import { faker } from '@faker-js/faker'

const key = faker.string.uuid()   // "550e8400-e29b-41d4-a716-446655440000"
```

Never hardcode a value that could collide with a previous run or a parallel worker.

### Eventual consistency (events / ingestion)

If the journey includes ingesting usage events, the processing is async through Kafka. Poll for the expected result:

```typescript
import { test, expect } from '@playwright/test'

const BASE = '/api/v3/openmeter'

test('meter value reflects ingested events', async ({ request }) => {
  // ... ingest ...

  // Poll until the meter reflects the event (up to 10s)
  await expect.poll(
    async () => {
      const res = await request.get(`${BASE}/meters/${meterSlug}/query`, {
        params: { subject: customerId },
      })
      expect(res.status()).toBe(200)
      const { data } = await res.json()
      return data[0]?.value ?? 0
    },
    { timeout: 10_000, intervals: [500, 1000, 2000] },
  ).toBeGreaterThan(0)
})
```

## Conventions

- **Import from `@playwright/test`** directly — no custom fixture layer.
- **Define `const BASE = '/api/v3/openmeter'`** at the top of each test file for the path prefix.
- **Use `faker.string.uuid()`** for any unique string field (names, keys, slugs). Never hardcode.
- **Read `api/v3/openapi.yaml`** for the endpoint before writing the request. Wrong field names produce 400s that look like test bugs.
- **Page size**: when listing to find a freshly-created entity, pass `'page[size]': '1000'` to avoid pagination miss on a shared DB.
- **Decimal round-trip**: the server trims trailing zeros (`"0.10"` → `"0.1"`). Compare the normalized form or parse as number.
- **Draft lifecycle**: some resources (plans, addons) accept invalid drafts and only reject at publish. If a create returns 201 unexpectedly, check the response body for `validation_errors` and pivot to the draft-with-errors assertion path.

## Running & Debugging

```bash
cd e2e/playwright

# Install dependencies (first time)
pnpm install
pnpm playwright install

# Run all tests
pnpm playwright test

# Run a specific file
pnpm playwright test tests/customers/

# Show full request/response on failure
DEBUG=pw:api pnpm playwright test

# With env overrides
OPENMETER_ADDRESS=http://localhost:8888 OPENMETER_API_KEY=om_key_xxx pnpm playwright test
```
