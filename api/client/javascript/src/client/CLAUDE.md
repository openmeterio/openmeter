# client

<!-- archie:ai-start -->

> Generated TypeScript SDK client implementing the full OpenMeter admin API surface. Each domain resource is a class wrapping an openapi-fetch Client<paths> with typed routes, sub-resource composition, and uniform error/date handling via transformResponse.

## Patterns

**Resource class wrapping openapi-fetch** — Every domain file exports one or more classes. Each class takes Client<paths, `${string}/${string}`> in its constructor and exposes async methods that call this.client.GET/POST/PUT/DELETE/PATCH with typed path literals, then return transformResponse(resp). Never access resp.data directly. (`constructor(private client: Client<paths, `${string}/${string}`>) {}
public async create(addon: AddonCreate, options?: RequestOptions) {
  const resp = await this.client.POST('/api/v1/addons', { body: addon, ...options })
  return transformResponse(resp)
}`)
**Nested sub-resource classes as public fields** — Compound domains (Billing, Apps, Customers, Notifications, Plans, Entitlements) expose sub-resources as public class fields initialized in the constructor. Each sub-class takes the same client reference. (`export class Billing {
  public profiles: BillingProfiles
  public invoices: BillingInvoices
  constructor(private client: Client<paths, ...>) {
    this.profiles = new BillingProfiles(this.client)
    this.invoices = new BillingInvoices(this.client)
  }
}`)
**Types always from schemas.ts via operations/paths** — All domain types (input shapes, path params, query params) come from './schemas.js' imports — either named types or operations['operationName']['parameters']['path'|'query'] and requestBody['content']['application/json']. Never invent local type shapes. (`import type { AddonCreate, operations, paths } from './schemas.js'
public async update(id: operations['updateAddon']['parameters']['path']['id'], addon: operations['updateAddon']['requestBody']['content']['application/json'], options?: RequestOptions)`)
**RequestOptions on every public method** — Every public method accepts options?: RequestOptions as its last parameter and spreads it into the fetch call with ...options. RequestOptions = Pick<RequestInit, 'signal'> defined in common.ts. (`public async get(id: string, options?: RequestOptions) {
  const resp = await this.client.GET('/api/v1/addons/{addonId}', { params: { path: { addonId: id } }, ...options })
  return transformResponse(resp)
}`)
**OpenMeter root class registers every resource** — index.ts creates the openapi-fetch client with date-encoding querySerializer and optional Bearer auth, then instantiates every resource class as a public field. Adding a new resource requires: (1) new domain file, (2) import in index.ts, (3) public field declaration, (4) constructor instantiation. (`this.client = createClient<paths>({ ...config, querySerializer: (q) => createQuerySerializer({ array: { explode: true, style: 'form' }, object: { explode: true, style: 'deepObject' } })(encodeDates(q)) })
this.addons = new Addons(this.client)`)
**Events ingest uses application/cloudevents-batch+json** — events.ts special-cases the ingest method: normalizes single/array input to array, applies setDefaultsForEvent (id, source, specversion, time defaults), and sends Content-Type: application/cloudevents-batch+json. The list method uses standard application/json. (`const resp = await this.client.POST('/api/v1/events', { body, headers: { 'Content-Type': 'application/cloudevents-batch+json' }, ...options })`)
**Explicit return type casts for typed responses** — When the generic transformResponse return type is too broad, methods cast the result to the exact operations response type. See meters.ts query() and queryPost(). (`return transformResponse(resp) as operations['queryMeter']['responses']['200']['content']['application/json']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.ts` | Root OpenMeter class and Config type. Creates the openapi-fetch client with date-encoding querySerializer (encodeDates + form/deepObject serialization), optional Bearer token, and instantiates all resource classes as public fields. | Adding a resource class file without registering it as a public field AND calling new in the constructor — the instance will be unreachable. Config type requires apiKey only when baseUrl is 'https://openmeter.cloud'. |
| `utils.ts` | transformResponse (throws HTTPError on 4xx/5xx, recursively decodes ISO date strings to Date objects), decodeDates, encodeDates (recursive Date→ISO string used in query serializer). | Never skip transformResponse; direct resp.data access misses date decoding and error handling entirely. |
| `common.ts` | Defines RequestOptions (Pick<RequestInit, 'signal'>), HTTPError class with fromResponse factory (reads application/problem+json for structured detail), and isHTTPError type guard. | HTTPError.fromResponse checks Content-Type: application/problem+json to extract problem.detail as the error message. Use isHTTPError to distinguish API errors from network failures. |
| `schemas.ts` | Generated file exporting paths, operations, and all domain types. Source of truth for all type shapes. | DO NOT EDIT — regenerate via make gen-api. Any manual edit will be overwritten. |
| `billing.ts` | Most complex resource: Billing class with three sub-classes (BillingProfiles, BillingInvoices, BillingCustomers). BillingInvoices exposes state-machine actions: advance, approve, retry, void, recalculateTax, snapshotQuantities, simulate, createLineItems, invoicePendingLines. | Most invoice actions take only path params (no body), but void requires body: VoidInvoiceActionInput and simulate/createLineItems/invoicePendingLines require body. Don't omit required bodies. |
| `customers.ts` | Customers class with four sub-classes: CustomerApps, CustomerEntitlements (v1 /api/v1), CustomerEntitlementsV2 (/api/v2), CustomerStripe. | customers.entitlementsV1 hits /api/v1 paths; customers.entitlements hits /api/v2 paths — they are different API versions, do not mix when adding methods. |
| `events.ts` | Events.ingest normalizes input, sets CloudEvent defaults (id via node:crypto or fallback UUID, source='@openmeter/sdk', specversion='1.0', time=now), sends as cloudevents-batch+json. | Must use Content-Type: application/cloudevents-batch+json for POST /api/v1/events. UUID generation tries node:crypto first and falls back to Math.random-based UUID for browser compatibility. |
| `events.spec.ts` | Vitest tests using @fetch-mock/vitest. Shows correct test pattern: fetchMock.mockReset() in beforeEach, route assertions with fetchMock.callHistory.done(). | Date params must be serialized to ISO strings in mock route query matchers — the SDK encodes dates before sending. The test verifies this encoding behavior. |

## Anti-Patterns

- Defining domain type shapes locally instead of importing from './schemas.js' — all shapes come from the generated file
- Accessing resp.data directly without calling transformResponse — loses date decoding and HTTPError throwing on 4xx/5xx
- Adding a resource class but not registering it as a public field AND constructor instantiation in OpenMeter (index.ts)
- Manually editing schemas.ts — it is code-generated and will be overwritten by make gen-api
- Omitting options?: RequestOptions from a public method or not spreading ...options into the fetch call — prevents AbortSignal propagation

## Decisions

- **openapi-fetch with typed paths/operations from generated schemas.ts** — Compile-time type-checking of route paths, path params, query params, and request/response bodies. Any API change in TypeSpec caught at TypeScript compile time after regeneration, preventing SDK drift.
- **Date values encoded (Date→ISO string) in query serializer and decoded (ISO→Date) in transformResponse** — OpenAPI dates are strings; SDK callers expect JS Date objects. Centralizing encode/decode in utils.ts ensures consistent Date handling across all endpoints without per-method boilerplate.
- **Each domain has its own file; index.ts is the only file that imports all of them** — Tree-shaking friendly — consumers who import individual classes only pull in their file. index.ts provides the convenience-bundled OpenMeter class for callers wanting the full client.

## Example: Add a new resource class for a new domain 'widgets' with list and create methods

```
// widgets.ts
import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { WidgetCreate, operations, paths } from './schemas.js'
import { transformResponse } from './utils.js'

export class Widgets {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  public async create(widget: WidgetCreate, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/widgets', { body: widget, ...options })
    return transformResponse(resp)
  }

  public async list(params?: operations['listWidgets']['parameters']['query'], options?: RequestOptions) {
// ...
```

<!-- archie:ai-end -->
