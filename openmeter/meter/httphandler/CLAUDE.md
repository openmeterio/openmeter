# httphandler

<!-- archie:ai-start -->

> v1 HTTP driver (package httpdriver) for meter CRUD plus meter querying (JSON + CSV), subjects, and group-by values. Bridges api.* request/response types to meter.ManageService and the streaming.Connector.

## Patterns

**httptransport handler factories** — Each operation is a method on *handler returning a typed handler built with httptransport.NewHandler / NewHandlerWithArgs: a decode func (ctx, r, params)->Request and a business func (ctx, Request)->Response, plus an encoder and WithOperationName option. All handler methods are declared on the Handler interface. (`func (h *handler) ListMeters() ListMetersHandler { return httptransport.NewHandlerWithArgs(decode, handle, commonhttp.JSONResponseEncoderWithStatus[ListMetersResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listMeters"))...) }`)
**Namespace from decoder, not request** — Every decode func calls h.resolveNamespace(ctx) which reads namespaceDecoder.GetNamespace(ctx); failure is a 500 commonhttp.NewHTTPError. Never read namespace from the body/path. (`ns, err := h.resolveNamespace(ctx)`)
**API<->domain mapping in mapping.go** — Conversions follow FromAPI/ToAPI naming: ToAPIMeter, ToAPIMeterQueryResult/Row, ToRequestFromQueryParamsPOSTBody, and toQueryParamsFromRequest. Handlers never construct streaming.QueryParams inline — they go through toQueryParamsFromRequest. (`params, err := h.toQueryParamsFromRequest(ctx, meter, ToRequestFromQueryParamsPOSTBody(request.params))`)
**GET and POST query share one param shape** — GET QueryMeter converts api.QueryMeterParams to the POST body via ToRequestFromQueryParamsPOSTBody, then both feed toQueryParamsFromRequest. CSV variants reuse QueryMeterParams/QueryMeterRequest type aliases. (`type QueryMeterCSVRequest = QueryMeterRequest`)
**ClickHouse JSONPath validation on write** — CreateMeter calls validateJSONPaths and UpdateMeter calls validateGroupByJSONPaths against streaming.ValidateJSONPath before persisting, because ClickHouse is stricter than Go JSONPath libs. Invalid paths return models.NewGenericValidationError. (`err := validateJSONPaths(ctx, h.streaming, request.MeterCreate.ValueProperty, request.MeterCreate.GroupBy)`)
**Subject/customer/group-by validation against meter config** — toQueryParamsFromRequest rejects group-by keys not in m.GroupBy (except special `subject`/`customer_id`), auto-adds subject/customer_id to GroupBy when filtered, and forbids AdvancedMeterGroupByFilters together with FilterGroupBy. (`if ok := groupBy == "subject" || groupBy == "customer_id" || m.GroupBy[groupBy] != ""; !ok { return params, models.NewGenericValidationError(...) }`)
**CSV responses via commonhttp.CSVResponse** — CSV handlers return a queryMeterCSVResult implementing Records()/FileName(); subject display names are resolved via subjectService.List keyed by subject key. Use commonhttp.CSVResponseEncoder. (`response := NewQueryMeterCSVResult(meter.Key, params.GroupBy, rows, subjectsByKey)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/MeterHandler interfaces, handler struct, New constructor, resolveNamespace | Depends on meter.ManageService (not meter.Service), customer.Service, streaming.Connector, subject.Service — all injected |
| `meter.go` | CRUD handlers (List/Get/Create/Update/Delete) + JSONPath validation helpers | CreateMeter defaults Name to Slug; UpdateMeter re-fetches current meter and only updates mutable fields; DeleteMeter resolves to ID then deletes |
| `query.go` | QueryMeter (GET), QueryMeterPost, ListSubjects, ListGroupByValues handlers | ListGroupByValues defaults From to last 24h when both From/To nil; query handlers always GetMeterByIDOrSlug first |
| `query_csv.go` | CSV query handlers and queryMeterCSVResult Records()/FileName() | CSV column order: window_start, window_end, [subject], [subject_display_name], groupBy..., value; subject is filtered out of groupByKeys |
| `mapping.go` | ToAPIMeter/QueryRow mappers and toQueryParamsFromRequest/getFilterCustomer | AdvancedMeterGroupByFilters and FilterGroupBy are mutually exclusive; customer IDs are resolved+validated via customerService.ListCustomers |

## Anti-Patterns

- Reading namespace from request body or path instead of resolveNamespace(ctx)
- Constructing streaming.QueryParams inline instead of via toQueryParamsFromRequest
- Skipping ClickHouse JSONPath validation when creating/updating meters
- Accepting group-by keys not present in meter.GroupBy (besides subject/customer_id)
- Allowing AdvancedMeterGroupByFilters and FilterGroupBy together

## Decisions

- **POST and GET query share one param representation by converting GET params into the POST body shape** — Single toQueryParamsFromRequest path avoids duplicating validation/group-by logic across transports
- **JSONPath validation is delegated to the streaming connector (ClickHouse)** — ClickHouse parsing is stricter than Go JSONPath libraries, so validating against the real engine prevents storing meters that fail at query time

## Example: Query handler: resolve namespace, fetch meter, map params, query streaming

```
func (h *handler) QueryMeterPost() QueryMeterPostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, idOrSlug QueryMeterPostParams) (QueryMeterPostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return QueryMeterPostRequest{}, err }
			var body api.QueryMeterPostJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil { return QueryMeterPostRequest{}, err }
			return QueryMeterPostRequest{namespace: ns, idOrSlug: idOrSlug, params: body}, nil
		},
		func(ctx context.Context, req QueryMeterPostRequest) (QueryMeterPostResponse, error) {
			m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: req.namespace, IDOrSlug: req.idOrSlug})
			if err != nil { return nil, err }
			params, err := h.toQueryParamsFromRequest(ctx, m, req.params)
			if err != nil { return nil, err }
			rows, err := h.streaming.QueryMeter(ctx, req.namespace, m, params)
// ...
```

<!-- archie:ai-end -->
