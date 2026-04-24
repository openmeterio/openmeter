# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the notification v1 API (channels, rules, events, delivery status), implementing httptransport.Handler[Req,Resp] and HandlerWithArgs patterns. Delegates all business logic to notification.Service; only responsible for request decoding, response encoding, and error mapping.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs for every endpoint** — Each CRUD operation returns a typed Handler or HandlerWithArgs. The decoder extracts namespace via h.resolveNamespace(ctx) and maps API params to service Input types. The operation calls h.service.<Method>. The encoder uses commonhttp.JSONResponseEncoderWithStatus. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[ListChannelsResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listNotificationChannels"), httptransport.WithErrorEncoder(errorEncoder()))...)`)
**Type alias pattern for Request/Response/Params** — Each handler file declares type aliases: <Verb><Noun>Request = notification.<Verb><Noun>Input, <Verb><Noun>Response = api.<ResponseType>, <Verb><Noun>Handler = httptransport.Handler[...]. (`type (ListChannelsRequest = notification.ListChannelsInput; ListChannelsResponse = api.NotificationChannelPaginatedResponse; ListChannelsHandler httptransport.HandlerWithArgs[...])`)
**Custom errorEncoder() in errors.go** — All handlers pass httptransport.WithErrorEncoder(errorEncoder()) in options. The error encoder maps notification domain errors to HTTP status codes.
**FromChannel / FromRule / FromEvent mapping functions in mapping.go** — All domain-to-API conversions are in mapping.go. Never inline domain-to-API conversion in handler files. (`item, err = FromChannel(channel); item, err = FromRule(rule)`)
**namespace resolution via h.resolveNamespace(ctx)** — Every decoder calls h.resolveNamespace(ctx) to get the tenant namespace — never read namespace from request body or path parameters directly. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListChannelsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (all endpoint methods), handler struct (service, options, resolveNamespace), and New() constructor. | New endpoints must be added to the Handler interface in handler.go before implementing the method. |
| `errors.go` | errorEncoder function mapping notification domain errors to HTTP status codes. | notification.NotFoundError must map to 404; missing an error type causes it to fall through to 500. |
| `mapping.go` | Domain-to-API conversion functions (FromChannel, FromRule, FromEvent, etc.). | mapping_test.go has coverage for payload shape; new payload types need test cases. |

## Anti-Patterns

- Putting business logic in decoder/encoder functions instead of delegating to notification.Service
- Inlining domain-to-API mapping in handler files instead of using mapping.go functions
- Reading namespace from request body instead of h.resolveNamespace(ctx)
- Omitting httptransport.WithErrorEncoder(errorEncoder()) from handler options
- Returning non-API types from handler operations (must use api.* response types)

## Decisions

- **Type aliases for Request/Response/Handler types per endpoint** — Makes handler signatures self-documenting and enables the router to reference concrete handler types without importing service input types directly.

<!-- archie:ai-end -->
