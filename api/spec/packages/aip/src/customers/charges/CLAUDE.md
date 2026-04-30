# charges

<!-- archie:ai-start -->

> TypeSpec definitions for customer charge API (flat-fee and usage-based charge types). Defines the discriminated union `Charge`, its models, enums, list-operation with filter/sort/expand, and compiles into the v3 OpenAPI spec and generated SDKs.

## Patterns

**Generic ChargeBase spread** — Charge type models spread `ChargeBase<T>` (itself spreading `Shared.Resource`) to inherit all shared fields. Type-specific fields are added directly on the concrete model. (`model FlatFeeCharge { ...ChargeBase<ChargeType.FlatFee>; payment_term: ...; }`)
**Discriminated union via @discriminated** — `Charge` is a `union` with `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })`. Variants are named by their discriminator value (`flat_fee`, `usage_based`). (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Charge { flat_fee: FlatFeeCharge, usage_based: UsageBasedCharge }`)
**@visibility(Lifecycle.Read) for all response-only fields** — Fields only returned in responses (status, invoice_at, customer, etc.) carry `@visibility(Lifecycle.Read)`. Fields writeable on create carry `@visibility(Lifecycle.Read, Lifecycle.Create)`. (`@visibility(Lifecycle.Read) status: ChargeStatus;`)
**@friendlyName on every model and enum** — All models and enums carry `@friendlyName("Billing<Name>")` so generator-level names are stable and prefixed with `Billing`. (`@friendlyName("BillingChargeType") enum ChargeType { ... }`)
**Operations file owns @get interface, imports HTTP decorators** — `operations.tsp` imports `@typespec/http`, `@typespec/openapi`, declares `using TypeSpec.Http; using TypeSpec.OpenAPI;`, defines filter models and the operations interface. Types stay in `charges.tsp`. (`interface CustomerChargesOperations { @get @operationId("list-customer-charges") list(...): Shared.PagePaginatedResponse<Charge> | ...; }`)
**deepObject explode for filter query params** — Filter query params use `@query(#{ style: "deepObject", explode: true })` to support bracket-style filters like `filter[status][oeq]=created`. (`@query(#{ style: "deepObject", explode: true }) filter?: ListCustomerChargesParamsFilter`)

## Key Files

| File             | Role                                                                                                                                                       | Watch For                                                                                                                                                                                                    |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `charges.tsp`    | Defines all charge models and enums (`ChargeType`, `ChargeStatus`, `ChargeBase<T>`, `FlatFeeCharge`, `UsageBasedCharge`, `Charge` union, `ChargesExpand`). | Do not add HTTP decorators here; those belong in operations.tsp. Concrete models must spread `ChargeBase<T>` — do not duplicate shared fields.                                                               |
| `operations.tsp` | Declares `CustomerChargesOperations` interface with `list` operation, filter/sort/expand query params, and pagination response type.                       | Must import `@typespec/http` and declare `using TypeSpec.Http;` — otherwise `@get`, `@path`, `@query` decorators are unknown. Response type must use `Shared.PagePaginatedResponse<Charge>` not a raw array. |
| `index.tsp`      | Re-exports charges.tsp and operations.tsp in order; consumed by the parent package index.                                                                  | New .tsp files added to this folder must be imported here or they are invisible to the compiler.                                                                                                             |

## Anti-Patterns

- Adding HTTP decorators (@get, @post, @path, @query) in charges.tsp — they belong only in operations.tsp
- Defining type-specific fields directly on ChargeBase instead of on FlatFeeCharge / UsageBasedCharge
- Omitting @friendlyName on new models or enums — the generated SDK name will be unstable
- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml — always regenerate via `make gen-api`
- Using @visibility(Lifecycle.Update) — charges have no update operation; use Read or Create only

## Decisions

- **Discriminated union with envelope:none and discriminatorPropertyName:type** — Keeps the wire format flat (no extra wrapper object) while still allowing the OpenAPI discriminator to route deserialization to the correct model.
- **ChargeBase<T extends ChargeType> generic model** — Avoids duplicating the large set of shared read-only fields across FlatFeeCharge and UsageBasedCharge while keeping the type parameter visible in the generated schema.

## Example: Adding a new charge type (e.g. credit_purchase)

```
// In charges.tsp
enum ChargeType { ..., CreditPurchase: "credit_purchase" }

@friendlyName("BillingCreditPurchaseCharge")
@summary("Credit purchase charge")
model CreditPurchaseCharge {
  ...ChargeBase<ChargeType.CreditPurchase>;
  // type-specific fields here
  @visibility(Lifecycle.Read, Lifecycle.Create)
  amount: Shared.Numeric;
}

// Update the union in charges.tsp
union Charge {
  flat_fee: FlatFeeCharge,
// ...
```

<!-- archie:ai-end -->
