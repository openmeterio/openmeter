# httphandler

<!-- archie:ai-start -->

> HTTP transport layer for the meterevent domain: exposes ListEvents (v1) and ListEventsV2 (cursor-paginated) handlers that decode api.* params into meterevent.Service inputs, call the service, and encode meterevent.Event into api.IngestedEvent (CloudEvents).

## Patterns

**httptransport.NewHandlerWithArgs three-stage handler** — Each handler is built from (decode request, business call, encoder) plus options. Decode resolves namespace via h.resolveNamespace(ctx); encode uses commonhttp.JSONResponseEncoderWithStatus[...](http.StatusOK); options append httptransport.WithOperationName(...). (`httptransport.NewHandlerWithArgs(decodeFn, func(ctx, req) (Resp, error){ ... h.metereventService.ListEventsV2(ctx, params) }, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listEventsV2"))...)`)
**Handler interface + unexported struct + New constructor** — handler.go declares Handler (composed of EventHandler), the unexported handler struct with namespaceDecoder/options/metereventService, a `var _ Handler = (*handler)(nil)` assertion, and New(...) returning Handler. (`type Handler interface { EventHandler }; var _ Handler = (*handler)(nil)`)
**Type aliases tie api types to handler signatures** — Each handler file aliases Request/Response/Params/Handler types, e.g. ListEventsV2Request = meterevent.ListEventsV2Params, ListEventsV2Response = api.IngestedEventCursorPaginatedResponse, keeping decode/encode signatures explicit. (`type ( ListEventsV2Request = meterevent.ListEventsV2Params; ListEventsV2Response = api.IngestedEventCursorPaginatedResponse )`)
**Namespace resolved from context, never from request body** — resolveNamespace(ctx) reads namespaceDecoder.GetNamespace(ctx); a missing namespace yields commonhttp.NewHTTPError(http.StatusInternalServerError, ...). All decode functions call it first. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListEventsV2Request{}, err }`)
**api<->domain mapping isolated in mapping.go via apiconverter** — convertListEventsV2Params uses apiconverter.ConvertCursorPtr/ConvertStringPtr/ConvertIDExactPtr/ConvertTimePtr; convertEvent builds a CloudEvents event.New() and convertListEventsV2Response wraps items + NextCursor via events.NextCursor.EncodePtr(). (`p.CustomerID = apiconverter.ConvertIDExactPtr(params.Filter.CustomerId); p.Time = apiconverter.ConvertTimePtr(params.Filter.Time)`)
**Per-event validation errors flow to api.IngestedEvent.ValidationError** — convertEvent joins e.ValidationErrors into a single *string via lo.EmptyableToPtr(errors.Join(...).Error()) so the service's collected validation issues surface in the response without failing the call. (`if len(e.ValidationErrors) > 0 { validationError = lo.EmptyableToPtr(errors.Join(e.ValidationErrors...).Error()) }`)
**v1 defaults applied in the decode function** — ListEvents decode applies defaults: From defaults to time.Now().Add(-meterevent.MaximumFromDuration).Add(time.Second) and Limit to meterevent.MaximumLimit via lo.FromPtrOr. (`From: lo.FromPtrOr(params.From, minimumFrom), Limit: lo.FromPtrOr(params.Limit, meterevent.MaximumLimit)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/EventHandler interfaces, handler struct, New constructor, resolveNamespace helper. | Add new endpoint methods to EventHandler and keep the `var _ Handler = (*handler)(nil)` assertion; resolveNamespace returns 500 (internal) on missing namespace, not 400. |
| `event.go` | ListEvents (v1) handler: applies From/Limit defaults, calls ListEvents, converts each event via convertEvent into []api.IngestedEvent. | minimumFrom adds an extra second to dodge validation; conversion errors abort the whole response. |
| `event_v2.go` | ListEventsV2 (cursor) handler: decodes via convertListEventsV2Params (wrapping errors in models.NewGenericValidationError), encodes via convertListEventsV2Response. | Decode-stage conversion errors must be wrapped as NewGenericValidationError (400); business-stage errors are returned raw. |
| `mapping.go` | All api<->domain conversion: convertListEventsV2Params, convertEvent (CloudEvents construction + JSON data unmarshal), convertListEventsV2Response. | convertEvent only sets data when e.Data != "" and json.Unmarshals it; use apiconverter.* helpers for pointer/cursor/id/time conversions rather than hand-rolling. |

## Anti-Patterns

- Calling the streaming connector or building business logic in the handler — handlers only decode/encode and delegate to meterevent.Service.
- Reading namespace from request params instead of resolveNamespace(ctx)/namespaceDecoder.
- Failing the request when an event has validation errors instead of surfacing them via api.IngestedEvent.ValidationError.
- Hand-writing api<->domain field conversion inline instead of using apiconverter helpers in mapping.go.
- Omitting httptransport.WithOperationName, breaking telemetry/operation naming.

## Decisions

- **v1 (ListEvents -> []IngestedEvent) and v2 (ListEventsV2 -> IngestedEventCursorPaginatedResponse) are separate handlers/files with separate type aliases.** — v2 introduces cursor pagination and a Filter object; keeping them split avoids overloading one decode path and lets each evolve independently.
- **Events are serialized as CloudEvents (event.New()) embedded in api.IngestedEvent.** — OpenMeter ingests CloudEvents; the listing API returns the original event envelope plus enrichment (CustomerId, IngestedAt, StoredAt, ValidationError).

## Example: Cursor-paginated handler wiring with namespace resolve, validation-wrapped decode, and mapping helpers

```
func (h *handler) ListEventsV2() ListEventsV2Handler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventsV2Params) (ListEventsV2Request, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventsV2Request{}, err
			}
			p, err := convertListEventsV2Params(params, ns)
			if err != nil {
				return ListEventsV2Request{}, models.NewGenericValidationError(err)
			}
			return p, nil
		},
		func(ctx context.Context, params ListEventsV2Request) (ListEventsV2Response, error) {
			events, err := h.metereventService.ListEventsV2(ctx, params)
// ...
```

<!-- archie:ai-end -->
