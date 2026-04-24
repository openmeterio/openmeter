# httphandler

<!-- archie:ai-start -->

> v1 HTTP handler layer for the meter domain. Exposes all meter, query, subject, and group-by endpoints via the httptransport.Handler pattern; delegates business logic to meter.ManageService and streaming.Connector.

## Patterns

**Handler interface + factory function** — Handler interface composes all sub-handler interfaces (MeterHandler). New(...) constructs the concrete handler struct; var _ Handler = (*handler)(nil) asserts compliance. (`func New(namespaceDecoder, customerService, meterService, streaming, subjectService, options...) Handler { return &handler{...} }`)
**httptransport.NewHandlerWithArgs for path-param endpoints** — Endpoints with path parameters use httptransport.NewHandlerWithArgs[Request, Response, Params]; the decoder func receives the parsed params alongside *http.Request. (`return httptransport.NewHandlerWithArgs(decoderFunc, operationFunc, commonhttp.JSONResponseEncoderWithStatus[R](http.StatusOK), options...)`)
**Namespace resolved at decode time** — Every decoder calls h.resolveNamespace(ctx) first and embeds the resolved namespace in the request struct — never passes raw http.Request to service layer. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, err }`)
**Mapping functions in mapping.go** — All api <-> domain type conversions (ToAPIMeter, ToAPIMeterQueryResult, toQueryParamsFromRequest) live in mapping.go; handler files must not inline type conversions. (`return ToAPIMeter(m), nil`)
**JSON path validation via streaming.Connector** — Before CreateMeter and UpdateMeter, validateJSONPaths/validateGroupByJSONPaths call streaming.ValidateJSONPath because ClickHouse is stricter than Go JSONPath libs. (`err := validateJSONPaths(ctx, h.streaming, request.ValueProperty, request.GroupBy)`)
**CSV response via commonhttp.CSVResponseEncoder** — CSV endpoints return a commonhttp.CSVResponse (interface with Records() + FileName()); query_csv.go implements queryMeterCSVResult satisfying that interface. (`commonhttp.CSVResponseEncoder[QueryMeterCSVResponse]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface declaration, concrete handler struct, New constructor. All service dependencies are injected here. | Adding a new endpoint requires extending MeterHandler interface + implementing the method on *handler. |
| `mapping.go` | Domain <-> API type converters and toQueryParamsFromRequest (resolves customer IDs, validates group-by, maps window timezone). | toQueryParamsFromRequest calls customerService.ListCustomers — any customer resolution error surfaces as 404, not 500. |
| `meter.go` | CRUD handler implementations for meters (List, Get, Create, Update, Delete). Also contains validateJSONPaths helpers. | UpdateMeter fetches current meter first via GetMeterByIDOrSlug to resolve ID before calling adapter; deleting this fetch breaks the update flow. |
| `query.go` | QueryMeter (GET), QueryMeterPost (POST), ListSubjects, ListGroupByValues — all delegate to streaming.Connector after resolving the meter. | ListGroupByValues defaults From to last 24 hours when both From and To are nil — must remain to avoid unbounded ClickHouse scans. |
| `query_csv.go` | CSV variants of query endpoints. queryMeterCSVResult.Records() builds header/data rows including optional subject display names from subjectService. | Subject display names are a best-effort enrichment; missing subjects fill with empty string, not an error. |

## Anti-Patterns

- Calling meter.ManageService or streaming.Connector directly in the decoder func — decoding must only extract and validate HTTP input.
- Inline type conversions in handler files instead of mapping.go functions.
- Skipping validateJSONPaths before CreateMeter/UpdateMeter — ClickHouse will reject invalid paths at query time.
- Returning 500 for customer-not-found in filter resolution — must return models.NewGenericNotFoundError.
- Using context.Background() instead of propagating ctx from the request.

## Decisions

- **handler struct holds both meter.ManageService and streaming.Connector.** — Meter query operations (QueryMeter, ListSubjects, ListGroupByValues) run directly against ClickHouse via Connector, bypassing the meter service to avoid double-hop latency.
- **POST body query (QueryMeterPost) is mapped to a GET-equivalent struct via ToRequestFromQueryParamsPOSTBody before calling toQueryParamsFromRequest.** — Single shared params-to-streaming conversion path reduces divergence between GET and POST query semantics.

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
