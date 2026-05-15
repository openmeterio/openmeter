# governance

<!-- archie:ai-start -->

> TypeSpec definitions for the Governance domain: batch feature-access query that resolves customer identifiers to access states across multiple features, with partial-error semantics and cursor pagination. All operations are private, unstable, and internal.

## Patterns

**Partial-error response model** — GovernanceQueryResponse combines data (successful results), errors (per-identifier failures), and meta (cursor pagination) in a single response — enabling partial success without HTTP 207. Errors use a typed enum (GovernanceQueryErrorCode). (`model GovernanceQueryResponse {
  data: GovernanceQueryResult[];
  errors: GovernanceQueryError[];
  meta: Common.CursorMeta;
}`)
**BaseError extension for error models** — GovernanceQueryError and GovernanceFeatureAccessReason both extend Shared.BaseError<TCode> with a typed enum code. Use model is Shared.BaseError<EnumType> pattern for structured error bodies. (`model GovernanceFeatureAccessReason is Shared.BaseError<GovernanceFeatureAccessReasonCode>;
model GovernanceQueryError is Shared.BaseError<GovernanceQueryErrorCode> { customer?: string; }`)
**Record<T> for dynamic feature maps** — GovernanceQueryResult.features is Record<GovernanceFeatureAccess> keyed by feature key — allowing a variable number of features per customer without fixed schema. (`features: Record<GovernanceFeatureAccess>;`)
**All operations are private/unstable/internal** — GovernanceOperations carries all three extension markers. This is the most restrictive visibility combination in the codebase. (`@extension(Shared.UnstableExtension, true)
@extension(Shared.InternalExtension, true)
@extension(Shared.PrivateExtension, true)
@post @route("/query") query(...)`)
**Cursor pagination on a POST body query** — The governance query operation spreads ...Common.CursorPaginationQuery as query params alongside a @body request — enabling pagination of batch query responses. (`query(
  ...Common.CursorPaginationQuery,
  @body _: GovernanceQueryRequest,
): GovernanceQueryResponse | Common.ErrorResponses;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `governance.tsp` | Defines all governance domain models: GovernanceQueryRequest, GovernanceQueryResponse, GovernanceQueryResult, GovernanceFeatureAccess, reason codes, and error types. | matched field on GovernanceQueryResult is a string[] of all customer keys/subjects that resolved to the same customer — do not collapse to a single string. The Unknown enum variant on both error enums is a forward-compatibility sentinel, not a default. |
| `operations.tsp` | Declares GovernanceOperations interface with the single query POST endpoint. | Must import @typespec/http and use `using TypeSpec.Http;`. The operation is a POST with both spread cursor params and a @body — both are required for paginated batch queries. |
| `index.tsp` | Barrel: imports governance.tsp and operations.tsp, reopens namespace Governance; | New .tsp files must be added to index.tsp. |

## Anti-Patterns

- Returning HTTP 207 Multi-Status instead of the data+errors+meta partial-success envelope pattern
- Using a GET with query params for the governance query — the request body can be large (up to 100 customer keys) and needs POST
- Omitting any of the three extension markers (Private/Unstable/Internal) on new governance operations
- Adding optional fields to GovernanceQueryRequest without specifying default values (include_credits defaults to false is the existing precedent)

## Decisions

- **Single POST /query endpoint with partial-error response rather than per-customer GET endpoints** — Batch evaluation of up to 100 customers across all features in one round-trip is a latency requirement; partial errors (CustomerNotFound) are expected and must not abort the entire response.
- **Unknown sentinel value on all error code enums** — Forward compatibility — new error codes added in future server versions degrade gracefully to Unknown on older clients rather than failing to deserialize.

## Example: Add a new access denial reason to GovernanceFeatureAccessReasonCode

```
// In governance.tsp:
enum GovernanceFeatureAccessReasonCode {
  Unknown: "unknown",
  UsageLimitReached: "usage_limit_reached",
  FeatureUnavailable: "feature_unavailable",
  FeatureNotFound: "feature_not_found",
  NoCreditAvailable: "no_credit_available",

  // New reason:
  EntitlementExpired: "entitlement_expired",
}
```

<!-- archie:ai-end -->
