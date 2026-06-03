---
name: v3-client-shim
description: Add or update an endpoint in the OpenMeter legacy JavaScript client's v3 compatibility shim (api/client/javascript/src/v3/). Use whenever a v3 API operation needs a typed JS client method — a new server endpoint to expose via `om.v3.*`, a changed request/response, regenerating the v3 types or Zod schemas, or wiring a new v3 resource class. Trigger this even if the user only says "add the v3 plans endpoint to the JS SDK", "wire up the new v3 customer call", or "regenerate the v3 client types" without naming the shim.
user-invocable: true
argument-hint: "[v3 endpoint or resource to add]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# v3 client shim — adding endpoints

## What this is

The legacy JS client at `api/client/javascript` (`@openmeter/sdk`) exposes the
**v3 API** through a compatibility shim under `src/v3/`, reached via a lazy
`om.v3` getter on the `OpenMeter` class:

```ts
const om = new OpenMeter({ apiKey })
await om.v3.plans.create({ key: 'starter', name: 'Starter', currency: 'USD', billing_cadence: 'P1M', phases: [] })
```

It reuses the **same generator stack as the v1 client** — `openapi-typescript`
for types, `orval` for Zod, and thin hand-written resource classes over
`openapi-fetch` — but pointed at the v3 OpenAPI spec. It is an interim fallback
to the generated v3 SDK in `api/spec/`; the win is that it's simple and uses
machinery the team already maintains.

**The one governing principle — "Option A":** the shim surfaces the v3 wire
shape **verbatim**. The v3 wire is `snake_case` (`billing_cadence`,
`created_at`); callers see exactly that. There is **no field renaming and no
per-type transform layer**. The only runtime transform is dates, handled by the
shared v1 `decodeDates`/`encodeDates` walker. If you ever feel the urge to add
camelCase↔snake_case mapping, stop (see Gotchas — it corrupts free-form maps and
rebuilds the exact thing this shim exists to avoid).

## File map (everything lives in `api/client/javascript`)

| Path | Role | Edit by hand? |
|---|---|---|
| `scripts/generate-v3.ts` | openapi-typescript → `src/v3/schemas.ts` | rarely |
| `orval.v3.config.ts` | orval → `src/v3/zod/index.ts` | rarely |
| `src/v3/schemas.ts` | generated TS types (committed) | **no — generated** |
| `src/v3/zod/index.ts` | generated Zod schemas (committed) | **no — generated** |
| `src/v3/index.ts` | `V3` class: builds the client, holds resource instances | yes (new classes) |
| `src/v3/<resource>.ts` | thin resource classes (plans, features, customers, …) | **yes** |
| `src/v3/v3.spec.ts` | smoke test | yes |
| `src/client/index.ts` | `OpenMeter` class + the `om.v3` getter | rarely |
| `src/client/utils.ts` | shared `transformResponse`, `decodeDates`, `encodeDates` | no |

## Workflow

Run all `pnpm`/`node`/`tsc` commands from `api/client/javascript`.

### 1. Make sure the operation is in the spec

Both v3 generators read the bundled **`api/v3/openapi.yaml`**. That file is
produced upstream from the TypeSpec in `api/spec/` (via `make gen-api` at the
repo root). If the endpoint was just added on the server and isn't in
`api/v3/openapi.yaml` yet, regenerate the spec first — the shim is strictly
downstream of it.

### 2. Regenerate the shim's types + Zod

```bash
pnpm run generate:client:v3   # → src/v3/schemas.ts
pnpm run generate:zod:v3      # → src/v3/zod/index.ts
```

(`pnpm run generate` runs all four: v1 client/zod + v3 client/zod.) Commit the
regenerated files — they're tracked, like their v1 counterparts.

### 3. Look up the operation's exact types

Open `src/v3/schemas.ts` and find the operation in the `operations` interface.
**The key is the kebab-case `operationId`** — e.g. `create-plan`, `get-customer`,
`ingest-metering-events`. From its block, read:

- the **path** and any **path params** (`path: { planId: ... }`),
- the **request body** type (`requestBody.content['application/json']` →
  `components['schemas']['CreatePlanRequest']`),
- the **2xx response** content type.

Import the **named component type** (`CreatePlanRequest`,
`BillingSubscriptionCreate`, `CreateCustomerRequest`, …). Look it up — don't
guess from v1 names; v3 uses request-wrapper names (`CreatePlanRequest`, not
`PlanCreate`).

### 4. Add the method (or a new class)

Mirror the existing resource classes exactly. Path params are typed as `string`
(they're ULIDs); the body uses the named component type; list query params come
from the `operations` index. **Use the v3 path verbatim** (`/openmeter/...`) —
the `${baseUrl}/api/v3` prefix is already handled by the client.

```ts
// POST with body
public async create(plan: CreatePlanRequest, options?: RequestOptions) {
  const resp = await this.client.POST('/openmeter/plans', { body: plan, ...options })
  return transformResponse(resp)
}

// GET with path param
public async get(planId: string, options?: RequestOptions) {
  const resp = await this.client.GET('/openmeter/plans/{planId}', {
    params: { path: { planId } },
    ...options,
  })
  return transformResponse(resp)
}

// GET list with query
public async list(
  params?: operations['list-plans']['parameters']['query'],
  options?: RequestOptions,
) {
  const resp = await this.client.GET('/openmeter/plans', { params: { query: params }, ...options })
  return transformResponse(resp)
}
```

**Extending an existing class** (e.g. add `update`/`delete` to `plans`): just add
the method. **Creating a new resource class**: write `src/v3/<resource>.ts`
following the same shape, then in `src/v3/index.ts` import it, declare a public
field, and instantiate it in the `V3` constructor off `this.client`. It is then
automatically reachable as `om.v3.<resource>` — no change needed in
`src/client/index.ts`.

For the full end-to-end walkthrough (extending a class **and** adding a brand-new
one, with the `V3` wiring), read **`references/add-endpoint.md`**.

### 5. Verify

```bash
node_modules/.bin/tsc --noEmit                      # 0 errors
node_modules/.bin/biome check --write src/v3/<file> # format + lint the hand-written file(s)
node_modules/.bin/vitest --run src/v3/v3.spec.ts    # extend the spec with the new op first
```

Add a case to `src/v3/v3.spec.ts` covering the new method (request shaping +
response). The spec stubs the transport with `@fetch-mock/vitest`; see the
existing cases and the fetch-mock notes in Gotchas.

## Gotchas (hard-won — read before editing)

- **operationIds are kebab-case** in the `operations` interface keys
  (`create-plan`), even though the JS methods are camelCase.
- **Don't re-prefix paths.** Use `/openmeter/...` verbatim; the client targets
  `${config.baseUrl}/api/v3`, composed in `src/v3/index.ts`.
- **Pagination and errors are free.** v3 `page` is a deepObject query
  (`page[size]`/`page[number]`) and works via the reused querySerializer; v3
  errors are `application/problem+json` and throw via the shared
  `transformResponse`/`HTTPError`. Nothing to wire.
- **Never re-export `src/v3/schemas.ts` or the v3 Zod from the package root** —
  it collides wholesale with the v1 `export *` (`paths`, `operations`, shared
  names). If consumers need v3 types, add a dedicated `@openmeter/sdk/v3` subpath
  export instead.
- **biome forbids assignment-in-expression.** The lazy `om.v3` getter uses an
  `if (!this._v3) { this._v3 = new V3(...) }` guard, not `??=` in a return
  (`noAssignInExpressions`). Follow that pattern for any new lazy getter.
- **fetch-mock v12 quirks** (in `v3.spec.ts`): `callHistory.calls()` takes **no**
  name filter (passing one is misread as a URL matcher); and a bare URL matcher
  requires an exact match, so use a `begin:<url>` matcher when the request
  carries a query string.
- **Don't add a transform/rename layer** (Option A). A generic camel↔snake walker
  would corrupt free-form `metadata` / event `data` / `additionalProperties`
  maps, and a schema-aware one re-creates the per-type transform layer the shim
  exists to avoid.
- **`sort` param caveat:** v3 `sort` is `style: form, explode: false`
  (comma-joined) but the shared serializer uses `explode: true`. If a list
  endpoint actually relies on `sort`, verify the encoding and add a per-param
  override if needed.
- **`decodeDates` footgun:** the shared walker coerces any ISO-8601-looking
  string to a `Date`, including inside free-form `metadata`/`data`. Pre-existing
  behavior; be aware when a payload carries such fields.
- **Only the hand-written `src/v3/*.ts` files are edited by hand.**
  `schemas.ts` and `zod/index.ts` are generated and committed. `scripts/*.ts`
  and `*.config.ts` are excluded from `tsc`; `*.spec.ts` is excluded from `tsc`
  but run by `vitest`.
- **The v3 spec is unstable.** Every v3 operation is flagged `x-unstable: true`,
  so paths, types, and verbs can change between spec versions. After
  regenerating, expect occasional churn in `schemas.ts` and re-verify the
  hand-written methods still compile.
- **Some capabilities are simply absent from the v3 spec** and therefore cannot
  be shimmed: notifications, portal tokens, subjects, debug metrics, and the app
  marketplace install flow exist in the v1 client but have no v3 operations. If
  asked to add one of these to `om.v3.*`, confirm it exists in
  `api/v3/openapi.yaml` first — if it doesn't, it belongs on v1.
