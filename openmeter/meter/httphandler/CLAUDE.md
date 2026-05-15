# httphandler

<!-- archie:ai-start -->

> v1 HTTP handler layer for the meter domain. Exposes meter CRUD, query (JSON and CSV), subject list, and group-by endpoints using the httptransport.Handler pattern; delegates business logic to meter.ManageService and streaming.Connector.

## Patterns

**Handler interface + factory function** — Handler interface composes sub-handler interfaces (MeterHandler). New(...) constructs the concrete handler struct; var _ Handler = (*handler)(nil) asserts compliance. (`func New(namespaceDecoder, customerService, meterService, streaming, subjectService, options...) Handler { return &handler{...} }`)
**httptransport.NewHandlerWithArgs for path-param endpoints** — Endpoints with path parameters use httptransport.NewHandlerWithArgs[Request, Response, Params]; the decoder func receives the parsed params alongside *http.Request. (`return httptransport.NewHandlerWithArgs(decoderFunc, operationFunc, commonhttp.JSONResponseEncoderWithStatus[R](http.StatusOK), options...)`)
**Namespace resolved at decode time** — Every decoder calls h.resolveNamespace(ctx) first and embeds the resolved namespace in the request struct — never passes raw http.Request to service layer. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, err }`)
**Mapping functions live in mapping.go** — All api <-> domain type conversions (ToAPIMeter, ToAPIMeterQueryResult, toQueryParamsFromRequest) live in mapping.go; handler files must not inline type conversions. (`return ToAPIMeter(m), nil`)
**JSON path validation via streaming.Connector before meter mutations** — validateJSONPaths and validateGroupByJSONPaths call streaming.ValidateJSONPath before CreateMeter/UpdateMeter because ClickHouse rejects invalid paths at query time. (`err := validateJSONPaths(ctx, h.streaming, request.ValueProperty, request.GroupBy)`)
**CSV response via commonhttp.CSVResponseEncoder** — CSV endpoints return commonhttp.CSVResponse (interface with Records() + FileName()). queryMeterCSVResult in query_csv.go implements this interface. (`commonhttp.CSVResponseEncoder[QueryMeterCSVResponse]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface declaration, concrete handler struct, New constructor with all service dependencies. | Adding a new endpoint requires extending the MeterHandler interface and implementing the method on *handler. |
| `mapping.go` | Domain <-> API type converters and toQueryParamsFromRequest (resolves customer IDs, validates group-by, maps window timezone). | toQueryParamsFromRequest calls customerService.ListCustomers — customer-not-found must surface as GenericNotFoundError, not 500. |
| `meter.go` | CRUD handler implementations (List, Get, Create, Update, Delete) and validateJSONPaths helpers. | UpdateMeter fetches current meter first to resolve ID; removing this fetch breaks the update flow. |
| `query.go` | QueryMeter (GET), QueryMeterPost (POST), ListSubjects, ListGroupByValues — all delegate to streaming.Connector after resolving the meter. | ListGroupByValues defaults From to last 24 hours when both From and To are nil — must remain to avoid unbounded ClickHouse scans. |
| `query_csv.go` | CSV variants of query endpoints; queryMeterCSVResult.Records() builds header/data rows with optional subject display names. | Subject display names are best-effort enrichment; missing subjects fill with empty string, not an error. |

## Anti-Patterns

- Calling meter.ManageService or streaming.Connector directly in the decoder func — decoding must only extract and validate HTTP input.
- Inline type conversions in handler files instead of mapping.go functions.
- Skipping validateJSONPaths before CreateMeter/UpdateMeter — ClickHouse will reject invalid JSONPath expressions at query time.
- Returning 500 for customer-not-found in filter resolution — must return models.NewGenericNotFoundError.
- Using context.Background() instead of propagating ctx from the request.

## Decisions

- **handler struct holds both meter.ManageService and streaming.Connector directly.** — Meter query operations (QueryMeter, ListSubjects, ListGroupByValues) run directly against ClickHouse via Connector, bypassing the meter service to avoid double-hop latency.
- **POST body query (QueryMeterPost) is mapped via ToRequestFromQueryParamsPOSTBody before calling toQueryParamsFromRequest.** — Single shared params-to-streaming conversion path reduces divergence between GET and POST query semantics.

## Example: Typical handler method: decode path param + namespace, delegate to service, map response

```
func (h *handler) GetMeter() GetMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, idOrSlug string) (GetMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return GetMeterRequest{}, err }
			return GetMeterRequest{namespace: ns, idOrSlug: idOrSlug}, nil
		},
		func(ctx context.Context, request GetMeterRequest) (GetMeterResponse, error) {
			m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: request.namespace, IDOrSlug: request.idOrSlug})
			if err != nil { return GetMeterResponse{}, err }
			return ToAPIMeter(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetMeterResponse](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("getMeter"))...,
	)
// ...
```

<!-- archie:ai-end -->
