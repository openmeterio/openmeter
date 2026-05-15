# charges

<!-- archie:ai-start -->

> TypeSpec definitions for the customer charges API — defines the `Charge` discriminated union (flat_fee, usage_based), shared `ChargeBase<T>` generic model, charge enums, and the list operation with filter/sort/expand. Compiles into v3 OpenAPI spec and generated SDKs; types live in charges.tsp, HTTP operations in operations.tsp.

## Patterns

**ChargeBase<T> spread for concrete models** — All concrete charge models (FlatFeeCharge, UsageBasedCharge) spread `ChargeBase<ChargeType.X>` to inherit shared read-only fields. Type-specific fields are added only on the concrete model, never on ChargeBase itself. (`model FlatFeeCharge { ...ChargeBase<ChargeType.FlatFee>; payment_term: ProductCatalog.PricePaymentTerm; }`)
**Discriminated union with envelope:none** — `Charge` is a `union` with `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })`. Variant keys match enum values verbatim (flat_fee, usage_based). New charge types require adding to both ChargeType enum and the union. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Charge { flat_fee: FlatFeeCharge, usage_based: UsageBasedCharge }`)
**@visibility(Lifecycle.Read) for response-only, Lifecycle.Create for writable fields** — All fields on ChargeBase are `@visibility(Lifecycle.Read)` — they are computed/server-assigned. Fields that can be set on creation use `@visibility(Lifecycle.Read, Lifecycle.Create)`. No Update visibility — charges have no PATCH operation. (`@visibility(Lifecycle.Read) status: ChargeStatus;  @visibility(Lifecycle.Read, Lifecycle.Create) payment_term: ProductCatalog.PricePaymentTerm;`)
**@friendlyName("Billing<Name>") on all models and enums** — Every exported model and enum must carry `@friendlyName("Billing<PascalCase>")` so the generated SDK name is stable and prefixed with Billing. (`@friendlyName("BillingChargeType") enum ChargeType { FlatFee: "flat_fee", UsageBased: "usage_based" }`)
**HTTP decorators and operations belong only in operations.tsp** — charges.tsp holds types only. operations.tsp imports `@typespec/http`, declares `using TypeSpec.Http; using TypeSpec.OpenAPI;`, and owns `@get`, `@path`, `@query`, `@operationId` usage. (`// operations.tsp only: @get @operationId("list-customer-charges") list(...): Shared.PagePaginatedResponse<Charge>`)
**deepObject explode for filter query params** — Filter query params use `@query(#{ style: "deepObject", explode: true })` to support bracket-style filters like `filter[status][oeq]=created`. (`@query(#{ style: "deepObject", explode: true }) filter?: ListCustomerChargesParamsFilter`)
**index.tsp re-exports all sibling files** — Every .tsp file added to this folder must be imported in index.tsp in dependency order or it is invisible to the compiler. (`import "./charges.tsp";
import "./operations.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charges.tsp` | Defines all charge models (ChargeBase<T>, FlatFeeCharge, UsageBasedCharge, ChargeTotals), enums (ChargeType, ChargeStatus), the Charge discriminated union, and ChargesExpand enum. | Do not add HTTP decorators here. Concrete models must spread ChargeBase<T> — do not duplicate shared fields. Do not use Lifecycle.Update. |
| `operations.tsp` | Declares CustomerChargesOperations interface with the list operation, filter/sort/expand params, and PagePaginatedResponse return type. | Must import @typespec/http and declare `using TypeSpec.Http;`. Response must use Shared.PagePaginatedResponse<Charge>, not a raw array. @path customerId is required. |
| `index.tsp` | Re-exports charges.tsp then operations.tsp in order; consumed by parent package index. | New .tsp files must be added here in dependency order or the compiler will not see them. |

## Anti-Patterns

- Adding HTTP decorators (@get, @post, @path, @query) in charges.tsp — they belong only in operations.tsp
- Defining type-specific fields on ChargeBase instead of on FlatFeeCharge / UsageBasedCharge
- Omitting @friendlyName on new models or enums — the generated SDK name will be unstable
- Using @visibility(Lifecycle.Update) — charges have no PATCH operation
- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml instead of running `make gen-api`

## Decisions

- **Discriminated union with envelope:none and discriminatorPropertyName:type** — Keeps the wire format flat (no extra wrapper object) while still allowing the OpenAPI discriminator to route deserialization to the correct concrete model.
- **ChargeBase<T extends ChargeType> generic model** — Avoids duplicating the large set of shared read-only fields across FlatFeeCharge and UsageBasedCharge while keeping the type parameter visible in the generated schema.
- **No update operation and no Lifecycle.Update visibility** — Charges are managed by the billing engine lifecycle; callers cannot freely PATCH them. This is intentional and mirrors the Go charges.Service interface which has no general Update method.

## Example: Adding a new charge type (e.g. credit_purchase) — changes required in charges.tsp only

```
// charges.tsp
@friendlyName("BillingChargeType")
enum ChargeType {
  FlatFee: "flat_fee",
  UsageBased: "usage_based",
  CreditPurchase: "credit_purchase",  // add here
}

@friendlyName("BillingCreditPurchaseCharge")
@summary("Credit purchase charge")
model CreditPurchaseCharge {
  ...ChargeBase<ChargeType.CreditPurchase>;
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Purchase amount")
  amount: Shared.Numeric;
// ...
```

<!-- archie:ai-end -->
