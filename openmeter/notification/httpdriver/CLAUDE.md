# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the notification v1 API (channels, rules, events, delivery status), implementing httptransport.Handler[Req,Resp] and HandlerWithArgs. Delegates all business logic to notification.Service; only decodes requests, encodes responses, and maps errors.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs for every endpoint** — Each CRUD operation returns a typed Handler. Decoder extracts namespace via h.resolveNamespace(ctx) and maps API params to service Input; operation calls h.service.<Method>; encoder uses commonhttp.JSONResponseEncoderWithStatus. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[ListChannelsResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listNotificationChannels"), httptransport.WithErrorEncoder(errorEncoder()))...)`)
**Type alias pattern for Request/Response/Params per endpoint** — Each handler file declares type aliases: <Verb><Noun>Request = notification.<Verb><Noun>Input, Response = api.<ResponseType>, Handler = httptransport.Handler[...]. (`type (ListChannelsRequest = notification.ListChannelsInput; ListChannelsResponse = api.NotificationChannelPaginatedResponse; ListChannelsHandler httptransport.HandlerWithArgs[...])`)
**Custom errorEncoder() in errors.go** — All handlers pass httptransport.WithErrorEncoder(errorEncoder()); the encoder maps notification domain errors (NotFoundError) to correct HTTP status codes. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(errorEncoder()))`)
**FromChannel / FromRule / FromEvent mapping functions in mapping.go** — All domain-to-API conversions live in mapping.go. Never inline domain-to-API conversion in handler files. (`item, err = FromChannel(channel); item, err = FromRule(rule)`)
**namespace resolution via h.resolveNamespace(ctx)** — Every decoder calls h.resolveNamespace(ctx) to get the tenant namespace — never read namespace from request body or path parameters. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListChannelsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (all endpoint methods), handler struct (service, options, resolveNamespace), New() constructor. | New endpoints must be added to the Handler interface before implementing the method. |
| `errors.go` | errorEncoder mapping notification domain errors to HTTP status codes. | notification.NotFoundError must map to 404; missing an error type falls through to 500. |
| `mapping.go` | Domain-to-API converters (FromChannel, FromRule, FromEvent) and API-to-domain helpers (AsChannelWebhookCreateRequest, AsRuleBalanceThresholdCreateRequest). | FromRule dispatches on rule.Type — add a case for every new EventType. mapping_test.go covers payload shape; new payload types need test cases. |
| `rule.go` | Rule CRUD handlers plus TestRule (generates a synthetic event via internal.TestEventGenerator and calls CreateEvent). | TestRule accesses h.testEventGenerator — ensure it is injected in New() when wiring. |

## Anti-Patterns

- Putting business logic in decoder/encoder functions instead of delegating to notification.Service
- Inlining domain-to-API mapping in handler files instead of using mapping.go functions
- Reading namespace from request body instead of h.resolveNamespace(ctx)
- Omitting httptransport.WithErrorEncoder(errorEncoder()) from handler options
- Returning non-API types from handler operations — encoder expects api.* response types

## Decisions

- **Type aliases for Request/Response/Handler types per endpoint** — Makes handler signatures self-documenting and lets the router reference concrete handler types without importing service input types directly.

## Example: Add a new list endpoint following the established handler pattern

```
type (
	ListFooRequest  = notification.ListFooInput
	ListFooResponse = api.NotificationFooPaginatedResponse
	ListFooParams   = api.ListNotificationFooParams
	ListFooHandler  httptransport.HandlerWithArgs[ListFooRequest, ListFooResponse, ListFooParams]
)

func (h *handler) ListFoo() ListFooHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListFooParams) (ListFooRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return ListFooRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }
			return ListFooRequest{Namespaces: []string{ns}}, nil
		},
		func(ctx context.Context, request ListFooRequest) (ListFooResponse, error) { /* ... */ },
// ...
```

<!-- archie:ai-end -->
