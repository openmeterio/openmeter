# client

<!-- archie:ai-start -->

> Generated TypeScript SDK client for OpenMeter. Each domain resource (Addons, Apps, Billing, Customers, Events, etc.) is a class that wraps an openapi-fetch Client<paths>, calls the typed OpenAPI routes, and routes all responses through transformResponse. The OpenMeter class in index.ts is the public entrypoint that instantiates all resource classes.

## Patterns

**Resource class wrapping openapi-fetch** — Each domain class (e.g. Addons, Billing, Customers) takes a Client<paths, ...> in its constructor and exposes async methods. Every method calls this.client.GET/POST/PUT/DELETE/PATCH with typed path literals from schemas.ts, then returns transformResponse(resp). (`constructor(private client: Client<paths, `${string}/${string}`>) {}
public async create(addon: AddonCreate, options?: RequestOptions) {
  const resp = await this.client.POST('/api/v1/addons', { body: addon, ...options })
  return transformResponse(resp)
}`)
**Nested sub-resource classes** — Compound domains expose sub-resources as public class fields initialized in the constructor, e.g. Billing.profiles (BillingProfiles), Billing.invoices (BillingInvoices), Apps.marketplace (AppMarketplace), Customers.apps (CustomerApps). (`export class Billing {
  public profiles: BillingProfiles
  public invoices: BillingInvoices
  constructor(private client: Client<paths, ...>) {
    this.profiles = new BillingProfiles(this.client)
    this.invoices = new BillingInvoices(this.client)
  }
}`)
**Path and query params typed from operations** — Parameter types are taken directly from operations['operationName']['parameters']['path'|'query'] and request bodies from operations['operationName']['requestBody']['content']['application/json']. Never invent param shapes. (`public async get(id: operations['getApp']['parameters']['path']['id'], options?: RequestOptions)`)
**RequestOptions = Pick<RequestInit, 'signal'>** — Every public method accepts an optional options?: RequestOptions as last parameter, which is spread into the fetch call with ...options. RequestOptions is defined in common.ts as Pick<RequestInit, 'signal'>. (`export type RequestOptions = Pick<RequestInit, 'signal'>`)
**All types imported from schemas.ts** — All domain types (AddonCreate, BillingProfileCreate, CustomerCreate, etc.) and the paths/operations types come from './schemas.js'. Never define domain shapes locally — always import from schemas. (`import type { AddonCreate, operations, paths } from './schemas.js'`)
**transformResponse for all API calls** — Every method ends with return transformResponse(resp). This throws HTTPError for 4xx/5xx responses and decodes ISO date strings into Date objects recursively. Never access resp.data directly. (`return transformResponse(resp)`)
**OpenMeter root class wires all resources** — index.ts creates the openapi-fetch client with date-encoding querySerializer and Bearer auth header, then instantiates every resource class. Adding a new resource requires: (1) new file, (2) import in index.ts, (3) public field + constructor instantiation. Exports all schemas and common via re-export. (`this.client = createClient<paths>({ ...config, querySerializer: (q) => createQuerySerializer({array:{explode:true,style:'form'}, object:{explode:true,style:'deepObject'}})(encodeDates(q)) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.ts` | Root OpenMeter class and Config type. Single entry point. Creates the openapi-fetch client with query serialization (dates encoded, arrays form-explode, objects deepObject), Bearer token injection, and instantiates all resource classes. | Adding a resource class without registering it as a public field and calling new in the constructor — the instance will be unreachable. |
| `utils.ts` | transformResponse (throws HTTPError on errors, decodes dates), decodeDates (recursive ISO-to-Date), encodeDates (recursive Date-to-ISO, used in query serializer). | Never skip transformResponse; direct resp.data access misses date decoding and error handling. |
| `common.ts` | Defines RequestOptions (Pick<RequestInit, 'signal'>), HTTPError class with fromResponse factory, and isHTTPError type guard. Errors from application/problem+json use problem.detail as message. | HTTPError.fromResponse reads Content-Type: application/problem+json to extract structured detail. Callers catching errors should use isHTTPError to distinguish API errors from network failures. |
| `schemas.ts` | Generated file exporting paths, operations, and all domain types. DO NOT EDIT — regenerate via make gen-api. | Any manual edit will be overwritten. Always import types from here rather than redefining. |
| `events.ts` | Special-cases ingest: normalizes single/array input to array, sets defaults (id, source, specversion, time) via setDefaultsForEvent, sends Content-Type: application/cloudevents-batch+json. UUID generation falls back gracefully in non-Node environments. | Must use Content-Type: application/cloudevents-batch+json for POST /api/v1/events, not application/json. |
| `events.spec.ts` | Vitest tests using @fetch-mock/vitest to mock HTTP calls. Shows correct test pattern: fetchMock.mockReset() in beforeEach, route assertions with fetchMock.callHistory.done(). | Date params must be serialized to ISO strings in the mock route query matcher — the SDK encodes dates before sending. |
| `billing.ts` | Most complex resource: three sub-classes (BillingProfiles, BillingInvoices, BillingCustomers). BillingInvoices exposes state-machine actions: advance, approve, retry, void, recalculateTax, snapshotQuantities, simulate, createLineItems, invoicePendingLines. | Invoice actions use POST with path params only (no body) except void (requires body: VoidInvoiceActionInput) and simulate/createLineItems/invoicePendingLines (require body). |
| `customers.ts` | Customers class with four sub-classes: CustomerApps, CustomerEntitlements (v1, /api/v1), CustomerEntitlementsV2 (/api/v2), CustomerStripe. entitlementsV1 field uses old /api/v1 paths; entitlements field uses /api/v2. | customers.entitlementsV1 vs customers.entitlements — they hit different API versions. Don't mix up the two when adding methods. |

## Anti-Patterns

- Defining domain type shapes locally instead of importing from './schemas.js'
- Accessing resp.data directly without calling transformResponse — loses date decoding and error handling
- Adding a resource class but not registering it as a public field in OpenMeter's constructor in index.ts
- Manually editing schemas.ts — it is code-generated and will be overwritten by make gen-api
- Omitting the options?: RequestOptions parameter from a public method or not spreading ...options into the fetch call — prevents abort signal propagation

## Decisions

- **openapi-fetch is used with typed paths/operations from generated schemas.ts** — Compile-time type-checking of route paths, path params, query params, and request/response bodies. Any API change caught at TypeScript compile time after regeneration.
- **Date values are encoded (Date->ISO string) in the query serializer and decoded (ISO string->Date) in transformResponse** — OpenAPI dates are strings; SDK callers expect JS Date objects. Centralizing encode/decode in utils.ts ensures consistent behavior across all endpoints.
- **Each domain has its own file exporting one or more classes; index.ts is the only file that imports them all** — Tree-shaking friendly — consumers who import individual classes only pull in their file. index.ts provides the convenience-bundled OpenMeter class for callers who want the full client.

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

  public async list(
// ...
```

<!-- archie:ai-end -->
