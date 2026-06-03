# entitlements

<!-- archie:ai-start -->

> v3 (AIP) TypeSpec for the customer entitlement access query API: EntitlementType enum, EntitlementAccessResult model, and a single GET list operation scoped to a customerId path param. Read-only — entitlement management lives in the v1 spec.

## Patterns

**Named wrapper model for non-paginated arrays** — ListCustomerEntitlementAccessResponse wraps a data: EntitlementAccessResult[] field in a named model (...ResponseData) rather than returning a bare array, for SDK consistency. (`model ListCustomerEntitlementAccessResponseData { data: EntitlementAccessResult[]; }`)
**All result fields are @visibility(Lifecycle.Read)** — EntitlementAccessResult is a pure read model; every field carries @visibility(Lifecycle.Read), with no create/update visibility. (`@visibility(Lifecycle.Read) has_access: boolean;`)
**Customer-scoped operation without pagination** — The list operation takes only @path customerId: Shared.ULID — no namespace param, no Common.PagePaginationQuery; all features are returned in one call. (`list(@path customerId: Shared.ULID): ListCustomerEntitlementAccessResponse | Common.NotFound | Common.ErrorResponses;`)
**BillingEntitlement* friendlyName prefix** — Generated type names use the Billing* prefix via @friendlyName (BillingEntitlementType, BillingEntitlementAccessResult). (`@friendlyName("BillingEntitlementType") enum EntitlementType { Metered: "metered", ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlements.tsp` | EntitlementType enum (Metered, Static, Boolean). | @friendlyName uses the 'BillingEntitlementType' prefix — keep the Billing* prefix convention for this spec. |
| `operations.tsp` | ListCustomerEntitlementAccessResponse/...Data wrapper models + CustomerEntitlementsOperations interface with a single customer-scoped list operation. | Operation is @path customerId-scoped — it does NOT use namespace or pagination params. |
| `access.tsp` | EntitlementAccessResult model (type, feature_key, has_access, optional config). | config is only populated for static entitlements; has_access is always true for boolean/static — document this invariant in JSDoc for any new type. |

## Anti-Patterns

- Adding write operations (create/update/delete) — management is in the v1 spec; this folder is read-only access query
- Adding pagination to the list operation — the response intentionally returns all feature access in one call
- Returning a bare array instead of the wrapper model with a 'data' key

## Decisions

- **Flat response list without pagination** — The number of features per customer is bounded and small; pagination overhead would complicate client-side feature-gate checks.

## Example: Customer-scoped read-only list operation

```
interface CustomerEntitlementsOperations {
  @get
  @operationId("list-customer-entitlement-access")
  @summary("List customer entitlement access")
  list(@path customerId: Shared.ULID): ListCustomerEntitlementAccessResponse | Common.NotFound | Common.ErrorResponses;
}
```

<!-- archie:ai-end -->
