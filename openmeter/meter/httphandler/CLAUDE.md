# httphandler

<!-- archie:ai-start -->

> v1 HTTP handler layer for the meter domain. Exposes meter CRUD, query (JSON and CSV), subject list, and group-by endpoints via the httptransport.Handler pattern; delegates to meter.ManageService and streaming.Connector.

## Patterns

**Handler interface + factory** — Handler composes sub-handler interfaces (MeterHandler); New(...) builds the concrete struct; var _ Handler = (*handler)(nil) asserts compliance. (`func New(namespaceDecoder, customerService, meterService, streaming, subjectService, options...) Handler { return &handler{...} }`)
**NewHandlerWithArgs for path-param endpoints** — Endpoints with path params use httptransport.NewHandlerWithArgs[Request, Response, Params]; the decoder receives parsed params alongside *http.Request. (`return httptransport.NewHandlerWithArgs(decoderFunc, operationFunc, commonhttp.JSONResponseEncoderWithStatus[R](http.StatusOK), options...)`)
**Namespace resolved at decode time** — Every decoder calls h.resolveNamespace(ctx) first and embeds the resolved namespace in the request struct — never passes raw http.Request to the service layer. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, err }`)
**Mapping functions live in mapping.go** — All api<->domain conversions (ToAPIMeter, ToAPIMeterQueryResult, toQueryParamsFromRequest) live in mapping.go; handler files must not inline conversions. (`return ToAPIMeter(m), nil`)
**JSON-path validation before meter mutations** — validateJSONPaths/validateGroupByJSONPaths call streaming.ValidateJSONPath before CreateMeter/UpdateMeter because ClickHouse rejects invalid paths at query time. (`err := validateJSONPaths(ctx, h.streaming, request.ValueProperty, request.GroupBy)`)
**CSV response via commonhttp.CSVResponseEncoder** — CSV endpoints return commonhttp.CSVResponse (Records() + FileName()); queryMeterCSVResult in query_csv.go implements it. (`commonhttp.CSVResponseEncoder[QueryMeterCSVResponse]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/MeterHandler interface, concrete handler struct, New constructor with all service deps. | A new endpoint requires extending MeterHandler and implementing the method on *handler. |
| `mapping.go` | Domain<->API converters and toQueryParamsFromRequest (resolves customer IDs, validates group-by, maps window timezone). | getFilterCustomer surfaces customer-not-found as models.NewGenericNotFoundError, not 500. |
| `meter.go` | CRUD handlers (List, Get, Create, Update, Delete) and validateJSONPaths helpers. | UpdateMeter fetches the current meter first to resolve ID; removing it breaks the update flow. |
| `query.go` | QueryMeter (GET), QueryMeterPost (POST), ListSubjects, ListGroupByValues — delegate to streaming.Connector after resolving the meter. | ListGroupByValues defaults From to last 24 hours when both From and To are nil — keep it to avoid unbounded ClickHouse scans. |
| `query_csv.go` | CSV variants; queryMeterCSVResult.Records() builds header/data rows with optional subject display names. | Subject display names are best-effort; missing subjects fill empty string, not an error. |

## Anti-Patterns

- Calling meter.ManageService or streaming.Connector in the decoder func — decoding must only extract/validate HTTP input
- Inline type conversions in handler files instead of mapping.go
- Skipping validateJSONPaths before CreateMeter/UpdateMeter — ClickHouse rejects invalid JSONPath at query time
- Returning 500 for customer-not-found in filter resolution — return models.NewGenericNotFoundError
- Using context.Background() instead of the request ctx

## Decisions

- **handler holds both meter.ManageService and streaming.Connector directly** — Query operations run directly against ClickHouse via Connector, bypassing the meter service to avoid double-hop latency.
- **POST body query mapped via ToRequestFromQueryParamsPOSTBody before toQueryParamsFromRequest** — A single shared params-to-streaming conversion path reduces divergence between GET and POST query semantics.

## Example: Typical handler: decode path param + namespace, delegate, map response

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
