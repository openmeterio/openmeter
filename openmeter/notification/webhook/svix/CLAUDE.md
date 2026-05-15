# svix

<!-- archie:ai-start -->

> Svix-backed implementation of webhook.Handler: manages Svix applications (one per namespace), webhook endpoints (channels), event type registration, message delivery, and delivery status polling. All Svix API calls are encapsulated here; callers interact only via the webhook.Handler interface.

## Patterns

**tracex span wrapping on every method** — Every exported svixHandler method wraps its body in a closure passed to tracex.Start[T] or tracex.StartWithNoValue, producing a named OTel span using 'svix.<operation>' convention. (`return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.create_webhook").Wrap(fn)`)
**internal.WrapSvixError on every Svix API call** — Every h.client.* call result is immediately reassigned through internal.WrapSvixError(err). Never propagate *svix.Error directly; always wrap before returning or branching. (`if err = internal.WrapSvixError(err); err != nil { return nil, fmt.Errorf("...: %w", err) }`)
**Idempotency key on every mutating Svix call** — CreateApplication, SendMessage, CreateWebhook, ResendMessage, and RotateSecret each generate a fresh idempotency key via idempotency.Key() and pass it to the Svix API options struct. (`idempotencyKey, err := idempotency.Key(); h.client.Message.Create(ctx, ns, input, &svix.MessageCreateOptions{IdempotencyKey: &idempotencyKey})`)
**NullChannel sentinel for unfiltered endpoint prevention** — When creating or updating a webhook with no channels or event types, inject NullChannel ('__null_channel') so the endpoint never receives unintended messages. (`if len(params.Channels) == 0 { params.Channels = []string{NullChannel} }`)
**Namespace maps 1:1 to Svix application UID** — The namespace string is used as both the Svix application name and UID. CreateApplication uses GetOrCreate with Uid=namespace to ensure idempotent provisioning. (`input := svix.ApplicationIn{Name: id, Uid: &id}; h.client.Application.GetOrCreate(ctx, input, ...)`)
**Cursor-based pagination with rate-limit sleep** — ListWebhooks, getDeliveryStatus, and ListAttemptedDestinations iterate Svix pages using Done/Iterator; a 100ms sleep is inserted between pages to respect rate limits. (`if !out.Done { opts.Iterator = out.Iterator; time.Sleep(100 * time.Millisecond); continue }`)
**Annotation constants for OTel span attributes** — All OTel span attributes use the typed constants from annotations.go (AnnotationApplicationUID, AnnotationEndpointUID, etc.) rather than inline strings. (`attribute.String(AnnotationApplicationUID, params.Namespace)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `svix.go` | Package entry point: defines SvixConfig, Config, Error type alias, New() constructor (registers event types on startup), NewHandler(), and the svixHandler struct. New() registers event types; NewHandler is the bare constructor. | New() uses context.Background() with a timeout for event type registration — this is the one sanctioned context.Background() usage in this package. Do not replicate this pattern elsewhere. |
| `webhook.go` | Implements CreateWebhook, UpdateWebhook, UpdateWebhookChannels, DeleteWebhook, GetWebhook, ListWebhooks — the core CRUD surface. CreateWebhook auto-generates a ULID endpoint UID and signing secret when not supplied. | UpdateWebhookChannels is a read-modify-write on channels (GetWebhook then UpdateWebhook) — not atomic. Concurrent updates can race. |
| `message.go` | Implements SendMessage, GetMessage, ResendMessage, and the internal getDeliveryStatus cursor-loop. SendMessage maps HTTP 409 Conflict from Svix to webhook.NewMessageAlreadyExistsError. | getDeliveryStatus makes two separate paginated calls (ListByMsg for attempts, ListAttemptedDestinations for per-endpoint status) and merges them by Svix endpoint ID — not by OpenMeter channel ID. |
| `application.go` | Implements CreateApplication which is called before every webhook or message operation to ensure the Svix application exists for the namespace via GetOrCreate. | CreateApplication is called defensively in CreateWebhook and UpdateWebhook; it is not cached — each call makes a Svix network round-trip. |
| `event.go` | Implements RegisterEventTypes using Svix EventType.Update (upsert) for each type — idempotent, safe to call repeatedly. | Uses EventType.Update (not Create) so existing event types are updated in place, not duplicated. |
| `annotations.go` | Defines all OTel span attribute key constants used across the package. | Add new constants here; never use raw strings for span attributes. |
| `internal/` | Single-responsibility error translation: maps *svix.Error HTTP status codes to typed SvixError categories (retryable, unrecoverable, not-found, validation). Entry point is WrapSvixError. | Never add business logic or Svix API calls here. Never return *svix.Error outside this package. |

## Anti-Patterns

- Returning *svix.Error directly to callers — always pass through internal.WrapSvixError first
- Skipping idempotency key generation on mutating Svix calls (Create, Rotate, Resend)
- Creating or updating an endpoint with empty channels and no event types without injecting NullChannel
- Adding Svix API logic inside internal/ — it is error-mapping only
- Using context.Background() in new methods — propagate the caller's ctx through the full call path

## Decisions

- **NullChannel sentinel prevents unintended message fan-out** — Svix delivers all messages to endpoints without channel or event-type filters. NullChannel is injected as a dummy filter so newly created endpoints receive nothing until explicitly subscribed to real channels.
- **internal sub-package isolates Svix error HTTP-status mapping** — HTTP status codes (429 retry, 422 validation, 404 not-found) drive retry vs unrecoverable classification; centralising this prevents scattered type assertions against *svix.Error across the package.
- **Every method body is wrapped in a tracex span closure** — Uniform OTel tracing with named spans ('svix.*') ensures every Svix call is observable in distributed traces without manual span management at each call site.

## Example: Adding a new svixHandler method that calls a Svix API

```
func (h svixHandler) PauseEndpoint(ctx context.Context, params webhook.PauseWebhookInput) error {
	fn := func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationApplicationUID, params.Namespace),
			attribute.String(AnnotationEndpointUID, params.ID),
		}
		span.AddEvent("pausing endpoint", trace.WithAttributes(spanAttrs...))

		idempotencyKey, err := idempotency.Key()
		if err != nil {
			return fmt.Errorf("failed to generate idempotency key: %w", err)
		}
		err = h.client.Endpoint.Pause(ctx, params.Namespace, params.ID, &svix.EndpointPauseOptions{
			IdempotencyKey: &idempotencyKey,
// ...
```

<!-- archie:ai-end -->
