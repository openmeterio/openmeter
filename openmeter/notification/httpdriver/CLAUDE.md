# httpdriver

<!-- archie:ai-start -->

> v1 HTTP transport for notification channels, rules, and events, mounted by openmeter/server/router. Decodes requests, resolves namespace, calls notification.Service, and maps domain<->api via FromX/AsX functions.

## Patterns

**httptransport handler trio per endpoint** — Each operation defines Request/Response/Handler type aliases then a (h *handler) method returning httptransport.NewHandler[WithArgs] with a decode fn, a business fn, a JSON encoder, and AppendOptions(WithOperationName, WithErrorEncoder(errorEncoder())). (`type (ListRulesRequest = notification.ListRulesInput; ListRulesHandler httptransport.HandlerWithArgs[ListRulesRequest, ListRulesResponse, ListRulesParams])`)
**Namespace resolved via h.resolveNamespace(ctx)** — Every decode fn first calls h.resolveNamespace(ctx) (wraps namespaceDecoder.GetNamespace, 500 on miss) and threads the namespace into the *Input. (`ns, err := h.resolveNamespace(ctx); req := ListChannelsRequest{Namespaces: []string{ns}, ...}`)
**Discriminated rule create/update via ValueByDiscriminator** — CreateRule/UpdateRule call body.ValueByDiscriminator() then switch on the api.NotificationRule*CreateRequest concrete type to pick the matching AsRule<Kind>CreateRequest/UpdateRequest mapper. (`switch v := value.(type) { case api.NotificationRuleBalanceThresholdCreateRequest: req = AsRuleBalanceThresholdCreateRequest(v, ns) ... }`)
**Domain<->API mapping in mapping.go (FromX / AsX)** — FromChannel/FromRule/FromEvent map domain to api; AsChannelWebhook*/AsRule*Request map api to *Input. FromRule switches on rule.Type and uses api union setters (rule.FromNotificationRuleBalanceThreshold(...)). (`err = rule.FromNotificationRuleBalanceThreshold(FromRuleBalanceThreshold(r))`)
**Centralized error encoder** — errorEncoder() chains commonhttp.HandleErrorIfTypeMatches for notification.NotFoundError(404), feature.FeatureNotFoundError(400), models.GenericValidationError(400), webhook.Validation/NotFound(500), notification.UpdateAfterDeleteError(409). All handlers attach it via WithErrorEncoder. (`commonhttp.HandleErrorIfTypeMatches[notification.NotFoundError](ctx, http.StatusNotFound, err, w)`)
**TestRule generates a synthetic event** — TestRule loads the rule, calls internal.TestEventGenerator.Generate for the rule type, then CreateEvent with annotation AnnotationRuleTestEvent=true. (`testEvent, _ := h.testEventGenerator.Generate(ctx, internal.EventGeneratorInput{Namespace: request.Namespace, EventType: rule.Type})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (Channel/Rule/Event sub-interfaces), handler struct, New() wiring service+TestEventGenerator+namespaceDecoder | New requires billing.Service (only to build the TestEventGenerator); resolveNamespace returns 500 on missing namespace. |
| `mapping.go` | All Domain<->API converters; FromRule dispatches per EventType | Adding a new rule/event type requires a new AsRule*Create+Update pair AND a FromRule* + FromRule switch case, or you get a GenericValidationError 'invalid rule type'. |
| `rule.go` | Rule CRUD + TestRule handlers using ValueByDiscriminator | The create/update switch has no default — an unhandled discriminator yields a zero-value req silently; add the case when extending. |
| `channel.go` | Channel CRUD handlers; only ChannelTypeWebhook supported | FromChannel returns GenericValidationError for non-webhook types; DeleteChannel uses EmptyResponseEncoder/204. |
| `event.go` | ListEvents/GetEvent/ResendEvent handlers; list params map subject/feature/rule/channel/from/to | ResendEvent returns 202 Accepted with EmptyResponseEncoder; body.Channels is optional. |
| `errors.go` | errorEncoder() status mapping for all notification handlers | New domain error types are invisible to clients (default 500) until added here. |
| `mapping_test.go` | Guards FromEventAsInvoiceCreated/UpdatedPayload pass-through and nil handling | Nil InvoicePayload must produce an error, not a zero api.Invoice. |

## Anti-Patterns

- Hand-decoding namespace instead of h.resolveNamespace(ctx)
- Adding a rule/event type without updating both AsRule* mappers and the FromRule switch
- Returning a new domain error type without registering it in errors.go (becomes a 500)
- Bypassing httptransport.NewHandler and writing the ResponseWriter directly
- Omitting WithOperationName/WithErrorEncoder from AppendOptions

## Decisions

- **Request/Response are type aliases of notification.*Input / api.* rather than bespoke DTOs** — Keeps the transport layer a thin adapter — the service input type IS the request, minimizing mapping surface.
- **Single shared errorEncoder() across all handlers** — Uniform HTTP status mapping for domain errors regardless of which endpoint surfaced them.

## Example: A list endpoint: decode params + namespace, call service, map domain to API

```
func (h *handler) ListChannels() ListChannelsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListChannelsParams) (ListChannelsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return ListChannelsRequest{}, err }
			return ListChannelsRequest{Namespaces: []string{ns}, Page: pagination.Page{PageSize: lo.FromPtrOr(params.PageSize, notification.DefaultPageSize)}}, nil
		},
		func(ctx context.Context, req ListChannelsRequest) (ListChannelsResponse, error) {
			resp, err := h.service.ListChannels(ctx, req)
			/* map resp.Items via FromChannel */ return ListChannelsResponse{}, err
		},
		commonhttp.JSONResponseEncoderWithStatus[ListChannelsResponse](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("listNotificationChannels"), httptransport.WithErrorEncoder(errorEncoder()))...,
	)
}
```

<!-- archie:ai-end -->
