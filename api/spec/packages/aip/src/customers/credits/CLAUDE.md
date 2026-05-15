# credits

<!-- archie:ai-start -->

> TypeSpec definitions for the customer credits API: credit grants (allocation, funding methods, lifecycle), credit balances (per-currency aggregates), credit transactions (immutable ledger entries), and credit adjustments. Compiles into v3 OpenAPI; the corresponding Go handlers are gated behind credits.enabled in api/v3/server.

## Patterns

**Shared.Resource for mutable entities, Shared.ResourceImmutable for ledger entries** — CreditGrant spreads Shared.Resource (mutable, has id/created_at/updated_at). CreditTransaction spreads Shared.ResourceImmutable to signal no updates are ever allowed. Balance models do not spread Resource at all — they are computed aggregates. (`model CreditGrant { ...Shared.Resource; ... }
model CreditTransaction { ...Shared.ResourceImmutable; ... }
model CreditBalances { retrieved_at: Shared.DateTime; balances: CreditBalance[]; }  // no Resource spread`)
**@friendlyName("Billing<Name>") on all exported models and enums** — Every exported symbol carries @friendlyName("Billing<PascalCase>") for stable generated SDK names. Exception: CreditBalance (non-billing-prefixed) is intentional for the inner model. (`@friendlyName("BillingCreditGrant") model CreditGrant { ... }
@friendlyName("BillingCreditFundingMethod") enum CreditFundingMethod { ... }`)
**Separate interfaces per resource in operations.tsp** — Operations are split into fine-grained interfaces: CustomerCreditGrantsOperations, CustomerCreditGrantExternalSettlementOperations, CustomerCreditBalancesOperations, CustomerCreditTransactionOperations, CustomerCreditAdjustmentsOperations. No omnibus interface. (`interface CustomerCreditGrantsOperations { create(...); get(...); list(...); }
interface CustomerCreditBalancesOperations { get(...); }`)
**Cursor pagination for transactions, page pagination for grants/adjustments** — CreditTransaction list uses ...Common.CursorPaginationQuery + Shared.CursorPaginatedResponse<T>. CreditGrant list uses ...Common.PagePaginationQuery + Shared.PagePaginatedResponse<T>. (`list(...Common.CursorPaginationQuery, ...): Shared.CursorPaginatedResponse<CreditTransaction>  // transactions
list(...Common.PagePaginationQuery, ...): Shared.PagePaginatedResponse<CreditGrant>  // grants`)
**@visibility(Lifecycle.Read, Lifecycle.Create) only — no Update** — No field uses Lifecycle.Update because credits have no general PATCH operation; grants have only targeted action endpoints (updateExternalSettlement). Read-only computed fields use Lifecycle.Read alone. (`@visibility(Lifecycle.Read, Lifecycle.Create) funding_method: CreditFundingMethod;
@visibility(Lifecycle.Read) status: CreditGrantStatus;`)
**Nested anonymous object literals for compound fields** — Tightly related sub-fields use inline anonymous objects (purchase, invoice, filters, available_balance) instead of named models, unless the sub-type needs to be referenced standalone. (`purchase?: { currency: Shared.CurrencyCode; per_unit_cost_basis?: Shared.Numeric; amount: Shared.Numeric; availability_policy?: CreditAvailabilityPolicy; };`)
**Commented-out stubs for unimplemented features** — Future fields (effective_at, expires_after, expires_at, OnAuthorization/OnSettlement variants) and entire interfaces (void-grant) are fully commented out. Do not uncomment without a matching Go implementation. (`// @visibility(Lifecycle.Create, Lifecycle.Read)
// effective_at?: Shared.DateTime;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.tsp` | Defines CreditGrant model, CreditFundingMethod, CreditAvailabilityPolicy, CreditGrantStatus, CreditGrantTaxConfig, CreditPurchasePaymentSettlementStatus enums. | Several commented-out fields are intentional stubs for future features — do not uncomment without a matching Go charges.Service implementation. `purchase` is an anonymous nested object, not a named model. |
| `operations.tsp` | Declares all CRUD and action interfaces for grants, balance, transactions, and adjustments. Imports all sub-type files directly. | Must import @typespec/http and declare `using TypeSpec.Http; using TypeSpec.OpenAPI;`. The void-grant interface is entirely commented out — do not expose it until Go service implements void lifecycle. balance.tsp and adjustment.tsp are imported here, not in index.tsp. |
| `balance.tsp` | Defines CreditBalances (list of per-currency CreditBalance) returned by the balance GET endpoint. | Does not spread Shared.Resource — it is a computed read-only aggregate, not a stored entity. Do not add id/created_at. |
| `ledger.tsp` | Defines CreditTransaction (immutable ledger entry spreading Shared.ResourceImmutable) and CreditTransactionType enum. | CreditTransaction spreads Shared.ResourceImmutable — do not add any mutable or Create-visible fields. Actor model is fully commented out for future use. |
| `adjustment.tsp` | Defines CreditAdjustment for manual balance corrections using PickProperties<Shared.Resource> for selective field inheritance. | Only Create-visible fields are present; no Read-only state fields. Currency is @visibility(Lifecycle.Create) only. |
| `index.tsp` | Re-exports grant.tsp, ledger.tsp, and operations.tsp. Note: balance.tsp and adjustment.tsp are imported directly in operations.tsp, not here. | New type files must be imported either in index.tsp or directly in operations.tsp before they are visible to the compiler. |

## Anti-Patterns

- Uncommenting void-grant or external-payment-initial-settlement-status stubs without a matching Go charges.Service implementation
- Adding @visibility(Lifecycle.Update) — no general update operation exists for any credit resource
- Spreading Shared.Resource into balance.tsp models — CreditBalances is a computed aggregate, not a stored entity
- Using page pagination for credit transactions — they must use cursor pagination (CursorPaginationQuery + CursorPaginatedResponse)
- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml instead of running `make gen-api`

## Decisions

- **Cursor pagination for credit transactions, page pagination for grants** — Transactions are append-only and high-volume; cursor pagination is stable under concurrent inserts. Grants are low-cardinality and need offset-based navigation for admin UIs.
- **Separate operations interfaces per resource sub-type** — Matches the Go side where each resource maps to a distinct service method group, and makes it easy to enable/disable individual endpoint groups (e.g. void-grant is fully commented out until implemented).
- **Credits API compiled in TypeSpec but gated at Go wiring layer, not at TypeSpec level** — The spec is always compiled; credits.enabled disables Go handler registration in api/v3/server and noop-wires ledger services in app/common. This keeps the spec complete while the feature is in progress.

## Example: Adding a new filter field to credit grant listing — only operations.tsp changes needed

```
// operations.tsp — extend ListCreditGrantsParamsFilter
@friendlyName("ListCreditGrantsParamsFilter")
model ListCreditGrantsParamsFilter {
  status?: CreditGrantStatus;
  currency?: Shared.CurrencyCode;
  // New field:
  feature?: Shared.ResourceKey;  // filter by feature key
}
// No changes needed in grant.tsp or index.tsp.
// Then: make gen-api && make generate
```

<!-- archie:ai-end -->
