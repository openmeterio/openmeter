# query

<!-- archie:ai-start -->

> Shared query-parameter translation sub-package for the v3 meter query endpoint. Converts api.MeterQueryRequest into streaming.QueryParams, validates dimensions/filters against meter definitions, and resolves customer IDs — insulating the HTTP handler from all domain mapping.

## Patterns

**ValidationIssue error factory in errors.go** — Every user-facing validation error is a package-level models.ValidationIssue var built with NewValidationIssue + WithFieldString/WithCriticalSeverity/commonhttp.WithHTTPStatusCodeAttribute. Public constructors attach runtime context via .WithAttr or .WithPathString. (`var ErrInvalidWindowSize = models.NewValidationIssue(...); func NewInvalidWindowSizeError(d string) error { return ErrInvalidWindowSize.WithAttr("value", d) }`)
**Whitelist-only filter operators (eq/in)** — ExtractStringsFromQueryFilter and ExtractStringsFromQueryFilterMapItem reject every operator except eq and in by checking non-nil fields (Neq, Nin, Contains, Ncontains, And, Or, and Exists for MapItem) and returning NewUnsupportedFilterOperatorError. eq and in together is also rejected. (`if f.Neq != nil || f.Nin != nil || f.Contains != nil { return nil, NewUnsupportedFilterOperatorError(fieldPath...) }`)
**Reserved-dimension switch in BuildQueryParams** — The dimensions loop switches on key: DimensionSubject, DimensionCustomerID, default (meter GroupBy lookup). A new reserved dimension must be added as a case here AND to IsReservedDimension in dimensions.go. (`switch k { case DimensionSubject: ...; case DimensionCustomerID: ...; default: if _, ok := m.GroupBy[k]; !ok { return params, NewInvalidDimensionFilterError(k) } }`)
**Deduplicated GroupBy via slices.Contains** — Every path that appends to params.GroupBy (subject filter, customer filter, explicit GroupByDimensions) first calls slices.Contains to prevent duplicates. (`if !slices.Contains(params.GroupBy, DimensionSubject) { params.GroupBy = append(params.GroupBy, DimensionSubject) }`)
**CustomerResolverFunc dependency injection** — Customer lookup is a CustomerResolverFunc type passed into BuildQueryParams, not a direct service reference — enabling noop injection in tests. NewCustomerResolver(svc) wires the real lookup with IncludeDeleted: true. (`func BuildQueryParams(ctx, m meter.Meter, body api.MeterQueryRequest, resolveCustomers CustomerResolverFunc) (streaming.QueryParams, error)`)
**ISO8601 <-> WindowSize lookup tables** — Bidirectional conversion uses package-level maps iso8601ToWindowSize / windowSizeToISO8601. A new window size requires updating both maps and the ErrInvalidWindowSize message. (`var iso8601ToWindowSize = map[string]meter.WindowSize{"PT1M": meter.WindowSizeMinute, "PT1H": ..., "P1D": ..., "P1M": ...}`)
**Filter complexity depth cap** — Non-reserved GroupBy dimension filters are validated with f.ValidateWithComplexity(maxGroupByFilterComplexityDepth) (currently 2). Raising the constant widens the nested-filter attack surface. (`const maxGroupByFilterComplexityDepth = 2`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `params.go` | BuildQueryParams — the single entry point converting MeterQueryRequest to streaming.QueryParams with full validation. | Go map iteration order is non-deterministic; GroupBy slice order may vary — tests must not assert exact ordering. |
| `errors.go` | All typed validation errors: window size, time zone, group_by, dimension filter, unsupported operator, customer-not-found. | HTTP status is embedded via commonhttp.WithHTTPStatusCodeAttribute — never return raw fmt.Errorf for user-facing errors; extend these sentinels. |
| `convert.go` | Stateless helpers: ISO8601<->WindowSize and the two Extract functions (eq/in only). | ExtractStringsFromQueryFilterMapItem also rejects f.Exists — the two Extract functions are NOT interchangeable. |
| `customers.go` | CustomerResolverFunc type, NewCustomerResolver factory, CustomersToStreaming. | NewCustomerResolver lists with IncludeDeleted: true and collects missing IDs via errors.Join — both behaviors are intentional. |
| `dimensions.go` | DimensionSubject/DimensionCustomerID constants, IsReservedDimension, IsSupportedGroupByDimension — single source of reserved dimension names. | IsSupportedGroupByDimension checks meter.GroupBy — the meter must carry its GroupBy populated or custom dimensions are rejected. |

## Anti-Patterns

- Returning raw fmt.Errorf for user-visible validation failures — use a models.ValidationIssue constructor from errors.go.
- Adding a dimension key to BuildQueryParams without updating IsReservedDimension.
- Appending to params.GroupBy without a slices.Contains dedup guard.
- Injecting customer.Service directly into BuildQueryParams instead of passing a CustomerResolverFunc.
- Supporting extra filter operators without updating BOTH Extract functions consistently.

## Decisions

- **CustomerResolverFunc function type instead of an interface.** — Minimal dependency surface — tests inject a one-line lambda; the handler passes NewCustomerResolver(svc) at wiring time.
- **Strict whitelist of eq/in filter operators.** — Bounds ClickHouse query complexity and index usage; rejecting Neq/Nin/Contains/And/Or at translation prevents unbounded scans reaching the analytics store.
- **Package-level ValidationIssue sentinels with runtime WithAttr/WithPathString constructors.** — Sentinels allow errors.Is checks; WithAttr attaches context without allocating a new type, consistent with the rest of the v3 stack.

## Example: Adding a new reserved dimension that resolves like customer_id

```
// dimensions.go
const DimensionWorkspaceID = "workspace_id"
func IsReservedDimension(d string) bool { switch d { case DimensionSubject, DimensionCustomerID, DimensionWorkspaceID: return true }; return false }
// params.go: add a `case DimensionWorkspaceID:` branch in the BuildQueryParams switch
```

<!-- archie:ai-end -->
