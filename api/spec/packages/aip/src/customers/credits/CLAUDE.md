# credits

<!-- archie:ai-start -->

> TypeSpec definitions for the customer credits API: credit grants (allocation, funding methods, lifecycle), credit balances, credit transactions (ledger), and credit adjustments. Compiles into v3 OpenAPI spec; gated behind credits.enabled in Go wiring.

## Patterns

**Shared.Resource spread for mutable resources, Shared.ResourceImmutable for transactions** — `CreditGrant` spreads `Shared.Resource` (mutable); `CreditTransaction` spreads `Shared.ResourceImmutable` to signal no updates are allowed. (`model CreditGrant { ...Shared.Resource; ... }  model CreditTransaction { ...Shared.ResourceImmutable; ... }`)
**Nested anonymous object literals for compound fields** — Tightly related sub-fields use inline anonymous objects (`purchase?: { currency: ...; amount: ...; }`) instead of named models, unless the sub-type needs to be referenced elsewhere. (`purchase?: { currency: Shared.CurrencyCode; per_unit_cost_basis?: Shared.Numeric; amount: Shared.Numeric; };`)
**@friendlyName("Billing<Name>") on every exported model/enum** — All exported symbols carry `@friendlyName("Billing<PascalCase>")` for stable generated SDK names. (`@friendlyName("BillingCreditGrant") model CreditGrant { ... }`)
**Separate interfaces per resource in operations.tsp** — Operations are split into fine-grained interfaces: `CustomerCreditGrantsOperations`, `CustomerCreditBalancesOperations`, `CustomerCreditTransactionOperations`, `CustomerCreditAdjustmentsOperations`. No omnibus interface. (`interface CustomerCreditGrantsOperations { create(...); get(...); list(...); }`)
**Cursor pagination for transactions, page pagination for grants** — `CreditTransaction` list uses `...Common.CursorPaginationQuery` + `Shared.CursorPaginatedResponse<T>`. `CreditGrant` list uses `...Common.PagePaginationQuery` + `Shared.PagePaginatedResponse<T>`. (`list(...Common.CursorPaginationQuery, ...): Shared.CursorPaginatedResponse<CreditTransaction>`)
**@visibility(Lifecycle.Read, Lifecycle.Create) — no Update visibility** — No field uses Lifecycle.Update because credits do not have a general PATCH operation; grants have only a targeted `updateExternalSettlement` action. (`@visibility(Lifecycle.Read, Lifecycle.Create) funding_method: CreditFundingMethod;`)

## Key Files

| File             | Role                                                                                                                                                                            | Watch For                                                                                                                                                                                                                               |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `grant.tsp`      | Defines `CreditGrant` model, `CreditFundingMethod`, `CreditAvailabilityPolicy`, `CreditGrantStatus`, `CreditGrantTaxConfig`, and `CreditPurchasePaymentSettlementStatus` enums. | Several commented-out fields exist (effective_at, expires_after, expires_at, OnAuthorization/OnSettlement enum variants) — these are intentional stubs for future features; do not uncomment without a corresponding Go implementation. |
| `operations.tsp` | Declares all CRUD and action interfaces for grants, balance, transactions, and adjustments. Imports all sub-type files.                                                         | Must import `@typespec/http` and declare `using TypeSpec.Http; using TypeSpec.OpenAPI;`. The void-grant interface is entirely commented out — do not expose it until the Go service implements the void lifecycle.                      |
| `balance.tsp`    | Defines `CreditBalances` (list of per-currency `CreditBalance`) returned by the balance GET endpoint.                                                                           | Does not spread Shared.Resource — it is a computed read-only aggregate, not a stored entity. Do not add id/created_at.                                                                                                                  |
| `ledger.tsp`     | Defines `CreditTransaction` (immutable ledger entry) and `CreditTransactionType` enum. Also has commented-out actor model for future attribution.                               | `CreditTransaction` spreads `Shared.ResourceImmutable` — do not add any mutable or Create-visible fields here.                                                                                                                          |
| `adjustment.tsp` | Defines `CreditAdjustment` for manual balance corrections. Spreads a subset of Shared.Resource fields via `PickProperties`.                                                     | Only Create-visible fields are present; no Read-only state fields. Currency is @visibility(Lifecycle.Create) only.                                                                                                                      |
| `index.tsp`      | Re-exports grant, ledger, and operations in order. Note: balance.tsp and adjustment.tsp are NOT re-exported from index.tsp — they are imported directly in operations.tsp.      | New type files must be imported either in index.tsp or directly in operations.tsp before they become available to the compiler.                                                                                                         |

## Anti-Patterns

- Uncommenting void-grant or external-payment-initial-settlement-status stubs without a matching Go charges.Service implementation
- Adding @visibility(Lifecycle.Update) — no general update operation exists for credit resources
- Spreading Shared.Resource into balance.tsp models — balance is a computed aggregate, not a stored entity
- Mixing pagination types: transactions must use cursor pagination, grants/adjustments use page pagination
- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml instead of running `make gen-api`

## Decisions

- **Cursor pagination for credit transactions, page pagination for grants** — Transactions are append-only and high-volume; cursor pagination is stable under concurrent inserts. Grants are low-cardinality and need offset-based navigation.
- **Separate operations interfaces per resource sub-type** — Matches the Go side where each resource maps to a distinct service method group, and makes it easy to enable/disable individual endpoint groups (e.g. void-grant is fully commented out until implemented).
- **credits API gated at wiring layer, not at TypeSpec level** — The TypeSpec spec is always compiled; the credits.enabled flag disables Go handler registration in api/v3/server and noop-wires ledger services in app/common. This keeps the spec complete while the feature is in progress.

## Example: Adding a new filtered list operation for credit grants by feature

```
// In operations.tsp — extend ListCreditGrantsParamsFilter
@friendlyName("ListCreditGrantsParamsFilter")
model ListCreditGrantsParamsFilter {
  status?: CreditGrantStatus;
  currency?: Shared.CurrencyCode;
  // New field:
  feature?: Shared.ResourceKey;
}
// No changes needed in grant.tsp or index.tsp.
// Run: make gen-api && make generate
```

<!-- archie:ai-end -->
