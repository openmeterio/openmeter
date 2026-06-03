# meters

<!-- archie:ai-start -->

> v3 HTTP handler package for all meter CRUD and query operations; bridges generated api/v3 request/response types to meter.ManageService and streaming.Connector via the httptransport.Handler pipeline, delegating all MeterQueryRequest-to-streaming.QueryParams translation to its query/ sub-package.

## Patterns

**Type-alias triplet per operation file** — create.go/get.go/update.go/query.go each declare <Op>Request/<Op>Response/<Op>Handler; path-param ops add <Op>Params. (`type CreateMeterRequest = meter.CreateMeterInput; type CreateMeterResponse = api.Meter; type CreateMeterHandler httptransport.Handler[CreateMeterRequest, CreateMeterResponse]`)
**NewHandlerWithArgs for path-param endpoints; NewHandler for no-param** — meterID-scoped endpoints use NewHandlerWithArgs; CreateMeter uses NewHandler. (`httptransport.NewHandlerWithArgs(decoder, operation, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), opts...)`)
**Namespace resolved in decoder, never in operation** — h.resolveNamespace(ctx) is the first decoder call; errors return before the operation runs. (`ns, err := h.resolveNamespace(ctx); if err != nil { return GetMeterRequest{}, err }`)
**Reserved-dimension validation in create/update decoders** — validateDimensionsWithoutReserved(*body.Dimensions) (dimensions.go) delegates to query.IsReservedDimension and returns NewReservedDimensionError. (`if body.Dimensions != nil { if err := validateDimensionsWithoutReserved(*body.Dimensions); err != nil { return CreateMeterRequest{}, err } }`)
**Query endpoints delegate to query.BuildQueryParams** — QueryMeter and QueryMeterCSV call query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService)); never inline param-building. (`params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))`)
**ValidationIssue sentinel + WithPathString/WithAttr constructor** — errors.go declares ErrReservedDimension with HTTP 400 attribute; per-call constructors add field-specific context. (`func NewReservedDimensionError(d string) error { return ErrReservedDimension.WithPathString("dimensions", d).WithAttr("value", d) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (7 methods) and struct with resolveNamespace, service (meter.ManageService), streaming, customerService, options. | Do not put domain calls on the struct directly; they belong in per-operation files. |
| `convert.go` | Hand-written helpers (ToAPIMeterAggregation, ToAPIMeterQueryRow/Result, FromAPIUpdateMeterRequest) plus goverter annotations. | Goverter vars (FromAPICreateMeterRequest, ToAPIMeter) are set in convert.gen.go init() — never call before init. |
| `convert.gen.go` | Goverter output (DO NOT EDIT) initializing conversion vars in init(). | Re-run make generate after changing convert.go annotations. |
| `errors.go` | ErrCodeReservedDimension code + ErrReservedDimension ValidationIssue sentinel. | New validation errors follow the sentinel + .WithPathString/.WithAttr constructor pattern. |
| `dimensions.go` | validateDimensionsWithoutReserved iterates a map against query.IsReservedDimension. | Add new reserved dimensions in query/dimensions.go, not here. |
| `query_csv.go` | QueryMeterCSV + queryMeterCSVResult (commonhttp.CSVResponse); enriches rows with customer key/name via ListCustomers. | A new reserved column must be added to both the header slice and the record loop in Records(). |
| `query/` | Sub-package owning BuildQueryParams, IsReservedDimension, CustomerResolverFunc, filter-operator extraction, ISO8601<->WindowSize. | Never inline query translation in the parent files; always delegate to BuildQueryParams. |

## Anti-Patterns

- Using NewHandler instead of NewHandlerWithArgs for path-param (meterID) endpoints
- Calling h.resolveNamespace in the operation closure instead of the decoder
- Adding a reserved dimension key in this package instead of query.IsReservedDimension
- Hand-editing convert.gen.go (overwritten by go generate)
- Omitting httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()), breaking error-to-status mapping

## Decisions

- **query/ is a separate sub-package, not inline in query.go** — Isolates dimension validation, filter extraction, customer resolution, and window-size mapping so handlers stay thin and the logic is testable without an HTTP context.
- **Conversion functions are Goverter-generated vars set in init(), not methods** — Lets type-safe generated mapping live in convert.gen.go while remaining callable from hand-written handlers without a receiver.

<!-- archie:ai-end -->
