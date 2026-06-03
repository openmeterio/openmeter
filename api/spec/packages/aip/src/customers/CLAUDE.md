# customers

<!-- archie:ai-start -->

> Root TypeSpec folder for the v3 customer API: core customer CRUD, billing overlay, Stripe session operations, and the credits/ and charges/ sub-domains. Coordinates sub-domain imports via index.tsp and is the entry point for the Customers namespace; compiles to api/v3/openapi.yaml and SDKs via make gen-api.

## Patterns

**Shared.Resource spread for mutable entities** — Core CRUD entities (Customer) spread ...Shared.Resource to inherit id, name, created_at, updated_at, deleted_at, metadata. (`model Customer { ...Shared.Resource; @visibility(Lifecycle.Create, Lifecycle.Read) key: Shared.ExternalResourceKey; }`)
**@visibility lists every applicable lifecycle** — Each field explicitly lists Create/Read/Update where applicable; omitting Update for a mutable field excludes it from PUT/upsert payloads. (`@visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update) primary_email?: string;`)
**@friendlyName("Billing<Name>") on every exported type** — All exported models, enums, unions, and scalars carry a Billing-prefixed @friendlyName to stabilize SDK names. (`@friendlyName("BillingCustomer") model Customer { ... }`)
**Shared generic request/response wrappers** — Operations use Shared.CreateRequest<T>, UpsertRequest<T>, CreateResponse<T>, GetResponse<T>, UpsertResponse<T> rather than bare model types. (`create(@body customer: Shared.CreateRequest<Customer>): Shared.CreateResponse<Customer> | Common.ErrorResponses;`)
**deepObject filter + PagePaginationQuery for lists** — List operations spread ...Common.PagePaginationQuery and accept filter via @query(#{ style: "deepObject", explode: true }). (`list(...Common.PagePaginationQuery, @query(#{ style: "deepObject", explode: true }) filter?: ListCustomersParamsFilter)`)
**Models in domain .tsp, HTTP in operations.tsp** — customer.tsp and billing.tsp define models only; operations.tsp imports @typespec/http and declares all interfaces. Stripe sub-operations use @route sub-paths. (`@post @route("/stripe/checkout-sessions") createCheckoutSession(...)`)
**Sub-domains as child folders imported via index.tsp** — credits/ and charges/ each have their own index.tsp imported in the parent index.tsp; new sub-domains follow the same pattern. (`// index.tsp: import "./credits/index.tsp"; import "./charges/index.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.tsp` | Customer, Address, CustomerUsageAttribution, CustomerReference, CustomerKeyReference, UsageAttributionSubjectKey. No HTTP imports. | Missing @visibility defaults to all lifecycles; not spreading Shared.Resource on new CRUD entities breaks consistency; no HTTP decorators here. |
| `billing.tsp` | CustomerBillingData referencing Billing.BillingProfileReference and Apps.AppCustomerData — bridges customer, billing, and apps namespaces. | Keep billing concerns here, not in customer.tsp; it imports apps and billing index.tsp by design. |
| `operations.tsp` | CustomersOperations (CRUD + list), CustomerBillingOperations (get/upsert billing, app-data, Stripe checkout/portal sessions), and related request models. | Each operation needs @operationId and @summary; Stripe sub-ops use @route; responses must include Common.ErrorResponses. |
| `index.tsp` | Root import file pulling in customer.tsp, operations.tsp, credits/index.tsp, charges/index.tsp. | New sub-domain folders must be imported here or they are silently excluded from compilation. |

## Anti-Patterns

- Adding HTTP decorators (@get, @post, @path, @query) in customer.tsp or billing.tsp — they belong only in operations.tsp
- Omitting @visibility on fields — they default to all lifecycle phases and leak write-only/system fields into payloads
- Omitting @friendlyName on new models or enums — the generated SDK name becomes unstable or collides
- Using inline pagination params instead of ...Common.PagePaginationQuery
- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml instead of running make gen-api

## Decisions

- **Customer billing and app data live in billing.tsp, not customer.tsp** — Keeps core customer identity free of billing/app coupling; billing.tsp bridges namespaces explicitly so the dependency is traceable.
- **credits/ and charges/ are child sub-folders imported via index.tsp** — Maintains logical separation of sub-domains while keeping the compilation path unified through the parent index.
- **Stripe operations use @route sub-paths within CustomerBillingOperations** — Co-locates Stripe session operations with the billing interface they belong to, avoiding a proliferation of top-level interfaces.

## Example: Adding a new customer sub-resource with list and get operations

```
// new-sub-resource.tsp (no HTTP imports)
import "../shared/index.tsp";
namespace Customers;

@friendlyName("BillingCustomerContract")
model CustomerContract {
  ...Shared.Resource;
  @visibility(Lifecycle.Create, Lifecycle.Read)
  title: string;
}

// operations.tsp addition
import "./new-sub-resource.tsp";
interface CustomerContractOperations {
  @get @operationId("get-customer-contract") @summary("Get customer contract")
// ...
```

<!-- archie:ai-end -->
