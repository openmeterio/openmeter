# Worked examples: adding v3 endpoints

Two end-to-end walkthroughs against `api/client/javascript`. The first extends an
existing resource class; the second adds a brand-new one and wires it into the
`V3` container. Both assume the operation already exists in `api/v3/openapi.yaml`
and that you've regenerated types/zod (`pnpm run generate:client:v3 &&
pnpm run generate:zod:v3`).

---

## Example 1 — extend an existing class: add `update` + `delete` to plans

### Find the operations

In `src/v3/schemas.ts`, the `operations` interface has kebab-case keys. The
relevant blocks look like:

```ts
'update-plan': {
  parameters: { path: { planId: components['schemas']['ULID'] } }
  requestBody: { content: { 'application/json': components['schemas']['UpsertPlanRequest'] } }
  responses: { 200: { content: { 'application/json': components['schemas']['BillingPlan'] } } }
}
'delete-plan': {
  parameters: { path: { planId: components['schemas']['ULID'] } }
  requestBody?: never
  responses: { 204: { content?: never } }
}
```

So: `update-plan` is `PUT /openmeter/plans/{planId}` with body `UpsertPlanRequest`;
`delete-plan` is `DELETE /openmeter/plans/{planId}`, no body, 204.

### Add the methods

In `src/v3/plans.ts`, add `UpsertPlanRequest` to the type import, then:

```ts
/** Update (replace) a plan */
public async update(
  planId: string,
  plan: UpsertPlanRequest,
  options?: RequestOptions,
) {
  const resp = await this.client.PUT('/openmeter/plans/{planId}', {
    body: plan,
    params: { path: { planId } },
    ...options,
  })
  return transformResponse(resp)
}

/** Delete a plan */
public async delete(planId: string, options?: RequestOptions) {
  const resp = await this.client.DELETE('/openmeter/plans/{planId}', {
    params: { path: { planId } },
    ...options,
  })
  return transformResponse(resp)
}
```

That's it — `plans` is already wired into `V3`, so `om.v3.plans.update(...)`
works immediately.

---

## Example 2 — add a new resource class: tax codes

The tax operations (`create-tax-code`, `get-tax-code`, `list-tax-codes`,
`upsert-tax-code`, `delete-tax-code`) live under `/openmeter/tax/codes` and have
no resource class yet.

### Look up the types

From `src/v3/schemas.ts`:
- `create-tax-code`: `POST /openmeter/tax/codes`, body `CreateTaxCodeRequest`
- `list-tax-codes`: `GET /openmeter/tax/codes`, query `operations['list-tax-codes']['parameters']['query']`
- `get-tax-code`: `GET /openmeter/tax/codes/{taxCodeId}`
- `upsert-tax-code`: `PUT /openmeter/tax/codes/{taxCodeId}`, body `UpsertTaxCodeRequest`
- `delete-tax-code`: `DELETE /openmeter/tax/codes/{taxCodeId}`

(Verify the exact paths and component names in the generated file — the above is
illustrative.)

### Write `src/v3/tax.ts`

```ts
import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateTaxCodeRequest,
  UpsertTaxCodeRequest,
  operations,
  paths,
} from './schemas.js'

/**
 * Tax codes (v3)
 *
 * Thin wrapper over the v3 tax endpoints. Bodies use the v3 wire shape verbatim
 * (snake_case).
 */
export class Tax {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  public async createCode(body: CreateTaxCodeRequest, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/tax/codes', { body, ...options })
    return transformResponse(resp)
  }

  public async listCodes(
    params?: operations['list-tax-codes']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/tax/codes', {
      params: { query: params },
      ...options,
    })
    return transformResponse(resp)
  }

  public async getCode(taxCodeId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/tax/codes/{taxCodeId}', {
      params: { path: { taxCodeId } },
      ...options,
    })
    return transformResponse(resp)
  }

  public async upsertCode(
    taxCodeId: string,
    body: UpsertTaxCodeRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/tax/codes/{taxCodeId}', {
      body,
      params: { path: { taxCodeId } },
      ...options,
    })
    return transformResponse(resp)
  }

  public async deleteCode(taxCodeId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/tax/codes/{taxCodeId}', {
      params: { path: { taxCodeId } },
      ...options,
    })
    return transformResponse(resp)
  }
}
```

### Wire it into `V3` (`src/v3/index.ts`)

Add the import (alphabetical), declare a public field, and instantiate it in the
constructor after `this.client` is built:

```ts
import { Tax } from './tax.js'
// ...
export class V3 {
  public client: Client<paths, `${string}/${string}`>
  // ...existing fields...
  public tax: Tax

  constructor(config: Config) {
    // ...this.client = createClient(...)...
    // ...existing resource instantiations...
    this.tax = new Tax(this.client)
  }
}
```

No change to `src/client/index.ts` is needed — the `om.v3` getter exposes the
whole `V3` instance, so `om.v3.tax.listCodes()` works as soon as the field
exists.

### Add a smoke test (`src/v3/v3.spec.ts`)

```ts
it<Context>('create tax code: POST <baseUrl>/api/v3/openmeter/tax/codes', async ({
  baseUrl,
  client,
  task,
}) => {
  const route = `${baseUrl}/api/v3/openmeter/tax/codes`
  const created = { id: '01J...', key: 'standard' }
  fetchMock.route(route, { body: created, status: 201 }, { method: 'POST', name: task.name })

  const resp = await client.v3.tax.createCode({ key: 'standard', /* snake_case fields */ })

  expect(resp).toMatchObject({ id: created.id })
  expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  // verify the body went out verbatim (snake_case, no renaming):
  const sent = JSON.parse(String(fetchMock.callHistory.calls()[0]?.options?.body))
  expect(sent.key).toBe('standard')
})
```

Note the fetch-mock v12 details: `calls()` takes no name argument, and for a GET
with a query string use a `begin:<url>` matcher instead of the exact URL.

### Verify

```bash
node_modules/.bin/tsc --noEmit
node_modules/.bin/biome check --write src/v3/tax.ts src/v3/index.ts src/v3/v3.spec.ts
node_modules/.bin/vitest --run src/v3/v3.spec.ts
```

---

## Quick reference: openapi-fetch call shapes

| HTTP | Call |
|---|---|
| POST body | `this.client.POST('/openmeter/x', { body, ...options })` |
| GET path | `this.client.GET('/openmeter/x/{id}', { params: { path: { id } }, ...options })` |
| GET query | `this.client.GET('/openmeter/x', { params: { query: params }, ...options })` |
| PUT path+body | `this.client.PUT('/openmeter/x/{id}', { body, params: { path: { id } }, ...options })` |
| DELETE | `this.client.DELETE('/openmeter/x/{id}', { params: { path: { id } }, ...options })` |
| POST path+body | `this.client.POST('/openmeter/x/{id}/action', { body, params: { path: { id } }, ...options })` |

Every method ends with `return transformResponse(resp)` — it throws `HTTPError`
on ≥400 and runs `decodeDates` on the body. Path params: `string`. Body: named
component type from `schemas.ts`. Query: `operations['<op-id>']['parameters']['query']`.
