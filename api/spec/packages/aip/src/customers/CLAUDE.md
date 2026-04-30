# customers

<!-- archie:ai-start -->

> TypeSpec definitions for the customer API: core customer CRUD, billing data, Stripe checkout/portal sessions, credits sub-API, and charges sub-API. Root folder coordinates sub-domain imports via index.tsp.

## Patterns

**Shared.Resource spread for mutable entities** — Core entities (Customer) spread ...Shared.Resource to inherit id, name, created_at, updated_at, deleted_at, and metadata fields consistently. (`model Customer { ...Shared.Resource; @visibility(Lifecycle.Create, Lifecycle.Read) key: Shared.ExternalResourceKey; ... }`)
**@visibility on every field with all applicable lifecycles** — Each field explicitly lists all of Lifecycle.Create, Lifecycle.Read, Lifecycle.Update that apply. Omitting Update for mutable fields causes them to be excluded from upsert payloads. (`@visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update) primary_email?: string;`)
**@friendlyName("Billing<Name>") on every model, enum, union** — All exported TypeSpec types must have @friendlyName prefixed with 'Billing' to stabilize generated SDK names. (`@friendlyName("BillingCustomer") model Customer { ... }`)
**Shared.CreateRequest/UpsertRequest/CreateResponse/GetResponse/UpsertResponse wrappers** — Operations use Shared generic request/response wrappers rather than bare model types or ad-hoc response shapes. (`create(@body customer: Shared.CreateRequest<Customer>): Shared.CreateResponse<Customer> | Common.ErrorResponses;`)
**deepObject filter + PagePaginationQuery for list operations** — List operations spread ...Common.PagePaginationQuery and accept filter via @query(#{ style: "deepObject", explode: true }). (`list(...Common.PagePaginationQuery, @query(#{ style: "deepObject", explode: true }) filter?: ListCustomersParamsFilter)`)
**Models in domain .tsp, HTTP ops in operations.tsp** — customer.tsp and billing.tsp define models only. operations.tsp imports @typespec/http and declares all interface operations. (`customer.tsp has no HTTP imports; operations.tsp: using TypeSpec.Http; interface CustomersOperations { @post create(...) }`)
**Sub-domains as child folders imported via index.tsp** — credits/ and charges/ are imported in index.tsp. New sub-domains follow the same pattern: add a folder with its own index.tsp and import it here. (`index.tsp: import "./credits/index.tsp"; import "./charges/index.tsp";`)

## Key Files

| File             | Role                                                                                                                                                  | Watch For                                                                                                                                   |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `customer.tsp`   | Defines Customer, Address, CustomerUsageAttribution, CustomerReference, CustomerKeyReference, UsageAttributionSubjectKey. No HTTP imports.            | Missing @visibility on new fields will default to all lifecycles. Spreading Shared.Resource is mandatory for CRUD entities.                 |
| `billing.tsp`    | Defines CustomerBillingData model referencing Billing.BillingProfileReference and Apps.AppCustomerData. Bridges customer and billing/apps namespaces. | Imports apps and billing index.tsp; adding new billing-related customer fields here keeps billing concerns co-located.                      |
| `operations.tsp` | Declares CustomersOperations (CRUD + list), CustomerBillingOperations (get/upsert billing, app-data, Stripe sessions), and related request models.    | Each operation needs @operationId and @summary. Stripe operations use @route sub-paths. Error responses must include Common.ErrorResponses. |
| `index.tsp`      | Root import file for the customers folder; pulls in customer.tsp, operations.tsp, credits/index.tsp, charges/index.tsp.                               | New sub-domain folders must be imported here or they are excluded from compilation.                                                         |

## Anti-Patterns

- Adding HTTP decorators (@get, @post, @path, @query) in customer.tsp or billing.tsp — they belong only in operations.tsp
- Omitting @visibility on fields — fields default to all lifecycle phases, leaking write-only or system fields into create/update payloads
- Omitting @friendlyName on new models or enums — the generated SDK name will be unstable or collide
- Using inline pagination params instead of ...Common.PagePaginationQuery — causes drift from the standard pagination contract
- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml — always regenerate via `make gen-api`

## Decisions

- **Customer billing and app data live in billing.tsp, not customer.tsp** — Keeps core customer identity (customer.tsp) free of billing/app coupling; billing.tsp bridges namespaces explicitly so the dependency is traceable.
- **credits/ and charges/ are child sub-folders imported via index.tsp** — Maintains logical separation of sub-domains while keeping the TypeSpec compilation path unified through the parent index.
- **Stripe-specific operations use @route sub-paths within CustomerBillingOperations** — Co-locates Stripe session operations with the billing interface they logically belong to, avoiding a proliferation of top-level interfaces.

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
  @get
// ...
```

<!-- archie:ai-end -->
