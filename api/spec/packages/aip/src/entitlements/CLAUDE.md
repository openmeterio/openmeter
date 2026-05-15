# entitlements

<!-- archie:ai-start -->

> TypeSpec definitions for the customer entitlement access query API: EntitlementType enum, EntitlementAccessResult model, and a single GET list operation scoped to a customerId path param. This is a read-only surface — entitlement management lives in the v1 spec.

## Patterns

**Response wrapper model for non-paginated arrays** — ListCustomerEntitlementAccessResponse wraps data: EntitlementAccessResult[] in a named model rather than returning a bare array, matching SDK consistency expectations. (`model ListCustomerEntitlementAccessResponseData { data: EntitlementAccessResult[]; }
model ListCustomerEntitlementAccessResponse { @Http.statusCode _: 200; @body body: ListCustomerEntitlementAccessResponseData; }`)
**All result fields are @visibility(Lifecycle.Read)** — EntitlementAccessResult is a pure read model; every field carries @visibility(Lifecycle.Read). No create/update visibility. (`@visibility(Lifecycle.Read) has_access: boolean;
@visibility(Lifecycle.Read) feature_key: Shared.ResourceKey;`)
**Customer-scoped operation without pagination** — The list operation takes only @path customerId — no namespace param, no Common.PagePaginationQuery. All features returned in one call. (`interface CustomerEntitlementsOperations { @get list(@path customerId: Shared.ULID): ListCustomerEntitlementAccessResponse | Common.NotFound | Common.ErrorResponses; }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlements.tsp` | Defines EntitlementType enum (Metered, Static, Boolean). Only add here if a new entitlement type is introduced. | The @friendlyName uses 'BillingEntitlementType' prefix — keep consistent with the Billing* prefix convention for this spec. |
| `operations.tsp` | CustomerEntitlementsOperations interface with a single list operation scoped to a customerId path param. | This operation is customer-scoped (@path customerId) — it does NOT use namespace or pagination params. |
| `access.tsp` | EntitlementAccessResult model. The config field is optional and only populated for static entitlements. | has_access is always true for boolean/static types — document this invariant in JSDoc for any new type. |

## Anti-Patterns

- Adding write operations (create/update/delete) — entitlement management is in the v1 spec, this folder is read-only access query only
- Adding pagination to the list operation — the response intentionally returns all feature access in one call
- Returning a bare array instead of the wrapper model with a 'data' key

## Decisions

- **Flat response list without pagination** — The number of features per customer is bounded and small; pagination overhead would complicate client-side feature-gate checks.

<!-- archie:ai-end -->
