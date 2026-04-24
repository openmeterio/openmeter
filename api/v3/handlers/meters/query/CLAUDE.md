# query

<!-- archie:ai-start -->

> Shared query-parameter translation sub-package for the v3 meter query endpoint. Converts api.MeterQueryRequest fields into streaming.QueryParams, validates dimensions/filters against meter definitions, and resolves customer IDs — insulating the HTTP handler from all domain-mapping logic.

## Patterns

**ValidationIssue error factory** — Every domain error is a package-level var of type models.ValidationIssue constructed with models.NewValidationIssue(...) plus WithFieldString / WithCriticalSeverity / commonhttp.WithHTTPStatusCodeAttribute. Public constructors like NewInvalidWindowSizeError attach runtime values via .WithAttr or .WithPathString. (`var ErrInvalidWindowSize = models.NewValidationIssue(ErrCodeInvalidWindowSize, "...", models.WithFieldString("granularity"), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))
func NewInvalidWindowSizeError(duration string) error { return ErrInvalidWindowSize.WithAttr("value", duration) }`)
**Whitelist-only filter operator extraction** — ExtractStringsFromQueryFilter / ExtractStringsFromQueryFilterMapItem explicitly reject every operator except eq and in by checking non-nil fields (Neq, Nin, Contains, Ncontains, And, Or, Exists) and returning NewUnsupportedFilterOperatorError. Do not add permissive pass-through paths. (`if f.Neq != nil || f.Nin != nil || f.Contains != nil { return nil, NewUnsupportedFilterOperatorError(fieldPath...) }`)
**Reserved-dimension switch in BuildQueryParams** — The dimensions loop in BuildQueryParams uses a switch on the key: case DimensionSubject, case DimensionCustomerID, default (meter GroupBy lookup). New reserved dimensions must be added as a case here and to IsReservedDimension in dimensions.go. (`switch k { case DimensionSubject: ... case DimensionCustomerID: ... default: if _, ok := m.GroupBy[k]; !ok { return params, NewInvalidDimensionFilterError(k) } }`)
**Deduplication of GroupBy entries via slices.Contains** — Every code path that appends to params.GroupBy (subject filter, customer filter, explicit GroupByDimensions) first calls slices.Contains to prevent duplicates. Always guard with this check before appending. (`if !slices.Contains(params.GroupBy, DimensionSubject) { params.GroupBy = append(params.GroupBy, DimensionSubject) }`)
**CustomerResolverFunc dependency injection** — Customer lookup is injected as CustomerResolverFunc (a function type), not as a direct service reference. BuildQueryParams receives it as a parameter so callers (HTTP handler) control the resolver — enabling noop injection in tests. (`func BuildQueryParams(ctx context.Context, m meter.Meter, body api.MeterQueryRequest, resolveCustomers CustomerResolverFunc) (streaming.QueryParams, error)`)
**ISO 8601 ↔ WindowSize via lookup tables** — Bidirectional conversion is done through package-level maps (iso8601ToWindowSize, windowSizeToISO8601). Adding a new window size requires updating both maps and the error message string in ErrInvalidWindowSize. (`var iso8601ToWindowSize = map[string]meter.WindowSize{"PT1M": meter.WindowSizeMinute, ...}`)
**Filter complexity depth cap** — Non-reserved GroupBy dimension filters are validated with f.ValidateWithComplexity(maxGroupByFilterComplexityDepth) (currently 2). Raising this constant widens the attack surface for nested filter expressions. (`const maxGroupByFilterComplexityDepth = 2
if err := f.ValidateWithComplexity(maxGroupByFilterComplexityDepth); err != nil { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `params.go` | Core translation function BuildQueryParams — the single entry point for converting MeterQueryRequest to streaming.QueryParams with full validation. | The dimensions map iteration order is non-deterministic in Go; GroupBy slice order may vary across calls — tests must not assert exact ordering. |
| `errors.go` | All typed validation errors for this package. Each error has a package-level sentinel var and a constructor that attaches runtime context via WithAttr/WithPathString. | HTTP status codes are embedded via commonhttp.WithHTTPStatusCodeAttribute — do not return raw fmt.Errorf for user-facing errors; use or extend these sentinels. |
| `convert.go` | Stateless conversion helpers for ISO 8601 durations, WindowSize, and QueryFilterString extraction. Only eq/in operators are supported. | ExtractStringsFromQueryFilterMapItem checks f.Exists (extra field vs QueryFilterString) — the two Extract functions are NOT interchangeable. |
| `customers.go` | CustomerResolverFunc type + NewCustomerResolver factory + CustomersToStreaming conversion. Bridges customer.Service to streaming.Customer. | NewCustomerResolver uses IncludeDeleted: true when listing customers; missing IDs are collected via errors.Join — both behaviours are intentional. |
| `dimensions.go` | Defines DimensionSubject and DimensionCustomerID constants, IsReservedDimension, and IsSupportedGroupByDimension. Single source of truth for reserved dimension names. | IsSupportedGroupByDimension checks meter.GroupBy map — the meter must be passed with its GroupBy populated or all custom dimensions will be rejected. |

## Anti-Patterns

- Returning raw fmt.Errorf for user-visible validation failures — always use a models.ValidationIssue constructor from errors.go.
- Adding a new dimension key directly to BuildQueryParams without also updating IsReservedDimension in dimensions.go.
- Appending to params.GroupBy without a slices.Contains deduplication guard.
- Injecting customer.Service directly into BuildQueryParams — always pass it as a CustomerResolverFunc to keep the function testable without wiring.
- Supporting additional filter operators (Neq, Nin, Contains) — the whitelist is intentional; extend only after updating ExtractStringsFromQueryFilter AND ExtractStringsFromQueryFilterMapItem consistently.

## Decisions

- **CustomerResolverFunc injection instead of interface** — A function type is the minimal dependency surface; tests inject a one-liner lambda without needing a mock struct, and the HTTP handler passes NewCustomerResolver(svc) at wiring time.
- **Strict whitelist for filter operators (eq/in only)** — ClickHouse query complexity and index usage are bounded; rejecting Neq/Nin/Contains/And/Or at the translation layer prevents unbounded scan patterns from reaching the analytics store.
- **Package-level ValidationIssue sentinels with runtime WithAttr/WithPathString constructors** — Sentinel vars allow callers to errors.Is check the type; WithAttr attaches context without allocating a new type, keeping error handling consistent with the rest of the v3 handler stack.

## Example: Adding a new reserved dimension (e.g. 'workspace_id') that resolves like customer_id

```
// dimensions.go — add constant and update IsReservedDimension
const DimensionWorkspaceID = "workspace_id"
func IsReservedDimension(dimension string) bool {
    switch dimension {
    case DimensionSubject, DimensionCustomerID, DimensionWorkspaceID:
        return true
    }
    return false
}

// errors.go — add typed error
var ErrWorkspaceNotFound = models.NewValidationIssue(
    "workspace_not_found", "workspace not found",
    models.WithFieldString("filters", "dimensions", DimensionWorkspaceID),
    models.WithCriticalSeverity(),
// ...
```

<!-- archie:ai-end -->
