# governance

<!-- archie:ai-start -->

> TypeSpec definitions for the v3 Governance domain: a batch feature-access query that resolves customer identifiers to per-feature access states with partial-error semantics and cursor pagination. All operations are private, unstable, and internal.

## Patterns

**Partial-error response envelope** — GovernanceQueryResponse combines data (successful results), errors (per-identifier failures, typed via GovernanceQueryErrorCode), and meta (Common.CursorMeta) — enabling partial success without HTTP 207. (`model GovernanceQueryResponse { data: GovernanceQueryResult[]; errors: GovernanceQueryError[]; meta: Common.CursorMeta; }`)
**BaseError extension for typed error/reason bodies** — GovernanceQueryError and GovernanceFeatureAccessReason both use model is Shared.BaseError<EnumCode>; GovernanceQueryError adds an optional customer field. (`model GovernanceQueryError is Shared.BaseError<GovernanceQueryErrorCode> { customer?: string; }`)
**Record<T> for dynamic feature maps** — GovernanceQueryResult.features is Record<GovernanceFeatureAccess> keyed by feature key, allowing a variable number of features per customer. (`features: Record<GovernanceFeatureAccess>;`)
**All operations private/unstable/internal** — GovernanceOperations carries all three extension markers — the most restrictive visibility combination in the spec. (`@extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @extension(Shared.PrivateExtension, true) @post @route("/query")`)
**Cursor pagination on a POST body query** — The query operation spreads ...Common.CursorPaginationQuery as query params alongside a @body GovernanceQueryRequest — paginating batch responses while keeping the (potentially large) request in the body. (`query(...Common.CursorPaginationQuery, @body _: GovernanceQueryRequest): GovernanceQueryResponse | Common.ErrorResponses;`)
**Unknown sentinel on error enums** — Both GovernanceFeatureAccessReasonCode and GovernanceQueryErrorCode include an Unknown: "unknown" zero-value sentinel for forward compatibility. (`enum GovernanceQueryErrorCode { Unknown: "unknown", CustomerNotFound: "customer_not_found" }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `governance.tsp` | All governance models: GovernanceQueryRequest (+Customers/Features), GovernanceQueryResponse/Result, GovernanceFeatureAccess, reason/error codes and BaseError types. | matched on GovernanceQueryResult is a string[] of all identifiers resolving to one customer — do not collapse to a single string. Customer keys are @minItems(1)@maxItems(100). |
| `operations.tsp` | GovernanceOperations interface with the single POST /query endpoint. | Must import @typespec/http and use `using TypeSpec.Http;`. The POST needs both spread cursor params and a @body. |
| `index.tsp` | Barrel importing governance.tsp and operations.tsp, reopening namespace Governance. | New .tsp files must be added here. |

## Anti-Patterns

- Returning HTTP 207 Multi-Status instead of the data+errors+meta partial-success envelope.
- Using a GET with query params for the query — the request body (up to 100 customer keys) needs POST.
- Omitting any of the three extension markers (Private/Unstable/Internal) on new governance operations.
- Adding optional fields to GovernanceQueryRequest without default values (include_credits defaults to false is the precedent).

## Decisions

- **Single POST /query with a partial-error response rather than per-customer GET endpoints.** — Batch evaluation of up to 100 customers across all features in one round-trip is a latency requirement; partial errors (CustomerNotFound) must not abort the whole response.
- **Unknown sentinel value on all error code enums.** — Forward compatibility — new server-side codes degrade gracefully to Unknown on older clients rather than failing to deserialize.

## Example: Adding a new access-denial reason

```
// governance.tsp
enum GovernanceFeatureAccessReasonCode {
  Unknown: "unknown",
  UsageLimitReached: "usage_limit_reached",
  FeatureUnavailable: "feature_unavailable",
  FeatureNotFound: "feature_not_found",
  NoCreditAvailable: "no_credit_available",
  EntitlementExpired: "entitlement_expired",
}
```

<!-- archie:ai-end -->
