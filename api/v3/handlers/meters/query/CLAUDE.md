# query

<!-- archie:ai-start -->

> Pure-function helper package that translates a v3 `api.MeterQueryRequest` body into a `streaming.QueryParams` for meter querying: window-size/timezone/groupBy/dimension-filter parsing, customer-ID resolution, and ISO-8601 duration conversion. No HTTP handler lives here — it is the request-mapping layer consumed by the meters (and featurecost) handlers.

## Patterns

**Single orchestrating entrypoint BuildQueryParams** — All request mapping flows through `BuildQueryParams(ctx, meter.Meter, api.MeterQueryRequest, CustomerResolverFunc) (streaming.QueryParams, error)`. Each request field (Granularity, TimeZone, GroupByDimensions, Filters.Dimensions) is conditionally parsed only when non-nil; sub-converters are pure helpers it calls. (`if body.Granularity != nil { ws, err := ConvertISO8601DurationToWindowSize(string(*body.Granularity)); ... params.WindowSize = &ws }`)
**Reserved dimensions are special-cased** — `DimensionSubject` ("subject") and `DimensionCustomerID` ("customer_id") are constants; `IsReservedDimension` / `IsSupportedGroupByDimension` gate them. In the filter switch they map to `params.FilterSubject` / `params.FilterCustomer`; everything else must exist in `m.GroupBy` and becomes a `params.FilterGroupBy` entry. (`switch k { case DimensionSubject: ...; case DimensionCustomerID: ...; default: if _, ok := m.GroupBy[k]; !ok { return params, NewInvalidDimensionFilterError(k) } }`)
**Filtered dimensions auto-added to GroupBy (deduped)** — When a subject or customer_id filter has values, the dimension is appended to `params.GroupBy` but only via `slices.Contains` guard to avoid duplicates. (`if len(subjects) > 0 && !slices.Contains(params.GroupBy, DimensionSubject) { params.GroupBy = append(params.GroupBy, DimensionSubject) }`)
**Restricted filter operators on reserved dims** — `ExtractStringsFromQueryFilter` / `ExtractStringsFromQueryFilterMapItem` accept ONLY `Eq` and `In`; any of Neq/Nin/Contains/Ncontains/And/Or (and Exists for the MapItem variant), or Eq+In together, returns `NewUnsupportedFilterOperatorError`. Non-reserved dims instead go through `request.ConvertQueryFilterStringMapItem` + `ValidateWithComplexity(maxGroupByFilterComplexityDepth)` (depth 2). (`if f.Neq != nil || f.Nin != nil || ... { return nil, NewUnsupportedFilterOperatorError(fieldPath...) }`)
**Errors are models.ValidationIssue with HTTP status + field path** — Every error is a package-level `models.NewValidationIssue(ErrCode..., msg, models.WithFieldString(...), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(...))` plus a `New...Error(...)` constructor that calls `.WithAttr` or `.WithPathString`. Never return bare `fmt.Errorf` for user-facing validation (the lone exception is `ConvertWindowSizeToISO8601Duration`'s internal unknown-WindowSize case). (`var ErrInvalidWindowSize = models.NewValidationIssue(ErrCodeInvalidWindowSize, "...", models.WithFieldString("granularity"), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**CustomerResolverFunc injected, not imported service** — Customer lookup is a `CustomerResolverFunc` typedef passed into `BuildQueryParams`; `NewCustomerResolver(customer.Service)` builds the production one (lists with `IncludeDeleted: true`, KeyBy ID, joins `NewCustomerNotFoundError` for missing IDs). Tests pass a `noopCustomerResolver` instead of a real service. (`func NewCustomerResolver(customerService customer.Service) CustomerResolverFunc { return func(ctx, ns, ids) ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `params.go` | Core `BuildQueryParams` orchestration + `maxGroupByFilterComplexityDepth = 2` constant. | Field-by-field nil checks are load-bearing; subject/customer filters mutate GroupBy as a side effect, so adding new dimension kinds means updating both the switch and the dedup-append logic. |
| `convert.go` | ISO-8601<->WindowSize maps and the `ExtractStrings...` eq/in-only extractors. | The two extractor variants differ ONLY in that the MapItem version also rejects `Exists`; keep them in sync when adding operators. |
| `dimensions.go` | Reserved-dimension constants and `IsReserved`/`IsSupportedGroupByDimension` predicates. | `IsSupportedGroupByDimension` returns true for reserved dims OR keys present in `m.GroupBy`; a new reserved dimension must be added to both the const set and `IsReservedDimension`. |
| `customers.go` | `CustomerResolverFunc` type, `NewCustomerResolver`, and `CustomersToStreaming` (identity map since `customer.Customer` satisfies `streaming.Customer`). | Resolver lists with `IncludeDeleted: true` and returns a joined error per missing ID — do not silently drop unresolved customers. |
| `errors.go` | All `ErrCode*` constants, `ValidationIssue` vars, and `New*Error` constructors with field paths / HTTP statuses. | Customer-not-found is 404, all others are 400; new errors must follow the same ValidationIssue + constructor pairing. |

## Anti-Patterns

- Returning plain `fmt.Errorf`/`errors.New` for request-validation failures instead of the package's `models.ValidationIssue`-backed `New*Error` constructors (loses HTTP status and field path).
- Accepting filter operators beyond eq/in on reserved (subject, customer_id) dimensions by bypassing `ExtractStringsFromQueryFilter(MapItem)`.
- Calling a customer service directly inside `BuildQueryParams` instead of going through the injected `CustomerResolverFunc`.
- Appending a dimension to `params.GroupBy` without the `slices.Contains` dedup guard.
- Treating an arbitrary dimension as valid without checking membership in `m.GroupBy` (must error via `NewInvalidDimensionFilterError`).

## Decisions

- **Mapping logic lives in its own `query` sub-package separate from the meters HTTP handler.** — It is pure and table-testable (convert_test/params_test cover it without HTTP), and is reused by `api/v3/handlers/featurecost`.
- **Customer resolution is abstracted behind `CustomerResolverFunc` rather than importing `customer.Service`.** — Keeps `BuildQueryParams` unit-testable with a noop resolver and decouples param-building from service wiring.
- **Reserved dimensions and non-reserved meter groupBy dimensions take different validation paths (eq/in extractor vs. full FilterString with complexity limit).** — Subject/customer map to dedicated streaming filter fields, while groupBy dimensions support richer filter expressions bounded by `maxGroupByFilterComplexityDepth`.

## Example: Adding a new user-facing validation error

```
const ErrCodeInvalidGroupBy models.ErrorCode = "invalid_group_by"

var ErrInvalidGroupBy = models.NewValidationIssue(
	ErrCodeInvalidGroupBy,
	"invalid group by dimension",
	models.WithFieldString("group_by_dimensions"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewInvalidGroupByError(dimension string) error {
	return ErrInvalidGroupBy.WithAttr("value", dimension)
}
```

<!-- archie:ai-end -->
