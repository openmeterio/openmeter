# svix

<!-- archie:ai-start -->

> Svix-backed implementation of webhook.Handler: manages one Svix application per namespace, webhook endpoints (channels), event-type registration, message delivery, and delivery-status polling. All Svix API calls are encapsulated here behind the webhook.Handler interface; the internal/ child isolates Svix HTTP-error translation.

## Patterns

**tracex span wrapping on every method** — Each exported svixHandler method builds its body as a closure passed to tracex.Start[T] or tracex.StartWithNoValue, producing a named OTel span 'svix.<operation>'. Span attributes use the typed constants from annotations.go, never inline strings. (`return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.create_webhook").Wrap(fn)`)
**internal.WrapSvixError on every Svix call** — Reassign every h.client.* error through internal.WrapSvixError(err) immediately; never propagate or branch on a raw *svix.Error. To detect specific statuses, errors.As the wrapped Error and inspect HTTPStatus. (`if err = internal.WrapSvixError(err); err != nil { return nil, fmt.Errorf("...: %w", err) }`)
**Idempotency key on every mutating call** — CreateApplication, SendMessage, CreateWebhook, ResendMessage, and RotateSecret each generate a fresh idempotency.Key() and pass it via the Svix options struct. (`idempotencyKey, _ := idempotency.Key(); h.client.Message.Create(ctx, ns, in, &svix.MessageCreateOptions{IdempotencyKey: &idempotencyKey})`)
**NullChannel sentinel to prevent unfiltered fan-out** — When creating/updating an endpoint with no channels and no event types, inject NullChannel ('__null_channel') so the endpoint receives nothing until explicitly subscribed. (`if len(params.EventTypes) == 0 && len(params.Channels) == 0 { params.Channels = []string{NullChannel} }`)
**Namespace maps 1:1 to Svix application UID** — The namespace string is both the Svix application name and UID; CreateApplication uses GetOrCreate with Uid=namespace for idempotent provisioning and is called defensively before webhook/message ops. (`input := svix.ApplicationIn{Name: id, Uid: &id}; h.client.Application.GetOrCreate(ctx, input, ...)`)
**Cursor pagination with rate-limit sleep** — ListWebhooks and getDeliveryStatus iterate Svix pages via Done/Iterator, sleeping 100ms between pages to respect rate limits. (`if !out.Done { opts.Iterator = out.Iterator; time.Sleep(100 * time.Millisecond); continue }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `svix.go` | Package entry: SvixConfig/Config, NullChannel const, Error = internal.SvixError alias, New() (registers event types on startup), NewHandler() (bare constructor), and svixHandler struct. | New() uses context.Background() with a timeout for startup event-type registration — the one sanctioned context.Background() here; do not replicate elsewhere. |
| `webhook.go` | Endpoint CRUD: CreateWebhook (auto-generates ULID UID + signing secret when absent), UpdateWebhook, UpdateWebhookChannels, DeleteWebhook, GetWebhook, ListWebhooks. | UpdateWebhookChannels is a non-atomic read-modify-write (GetWebhook then UpdateWebhook); concurrent updates can race. DeleteWebhook swallows not-found as success. |
| `message.go` | SendMessage (maps HTTP 409 to webhook.NewMessageAlreadyExistsError), GetMessage, ResendMessage, and getDeliveryStatus cursor loop. | getDeliveryStatus makes two paginated calls (ListByMsg + ListAttemptedDestinations) merged by Svix endpoint ID, not OpenMeter channel ID. |
| `application.go` | CreateApplication via GetOrCreate, called before every webhook/message op to ensure the namespace's app exists. | Not cached — each call is a Svix network round-trip. |
| `event.go` | RegisterEventTypes via EventType.Update (upsert), idempotent and safe to call repeatedly. | Uses Update not Create, so existing event types are updated in place, not duplicated. |
| `annotations.go` | Typed OTel span attribute key constants used across the package. | Add new constants here; never use raw strings for span attributes. |
| `internal/error.go` | Sole entry WrapSvixError maps *svix.Error HTTP status to typed SvixError categories (retryable/unrecoverable/not-found/validation); 422 uses SvixValidationErrorBody, others SvixErrorBody, rate-limit carries RetryAfter. | Error-mapping only — never add Svix API calls or business logic; never return *svix.Error outside this package; never construct SvixError{} literals externally. |

## Anti-Patterns

- Returning *svix.Error directly to callers instead of passing through internal.WrapSvixError.
- Skipping idempotency key generation on a mutating Svix call (Create/Rotate/Resend).
- Creating/updating an endpoint with empty channels and no event types without injecting NullChannel.
- Adding Svix API logic or business logic inside internal/ — it is error-mapping only.
- Using context.Background() in new methods instead of propagating the caller's ctx (New()'s startup registration is the only exception).

## Decisions

- **NullChannel sentinel prevents unintended message fan-out.** — Svix delivers all messages to endpoints lacking channel/event-type filters; the dummy filter lets an endpoint be provisioned before it is subscribed to real channels.
- **internal sub-package isolates Svix HTTP-status error mapping.** — Status codes (429 retry, 422 validation, 404 not-found) drive retry vs unrecoverable classification; centralising avoids scattered *svix.Error type assertions.
- **Every method body is wrapped in a tracex span closure with 'svix.*' naming.** — Gives uniform, observable distributed tracing for every Svix call without manual per-call span management.

## Example: Adding a new svixHandler method that calls a Svix API

```
func (h svixHandler) PauseEndpoint(ctx context.Context, params webhook.PauseWebhookInput) error {
    fn := func(ctx context.Context) error {
        span := trace.SpanFromContext(ctx)
        span.AddEvent("pausing endpoint", trace.WithAttributes(
            attribute.String(AnnotationApplicationUID, params.Namespace),
            attribute.String(AnnotationEndpointUID, params.ID),
        ))
        idempotencyKey, err := idempotency.Key()
        if err != nil {
            return fmt.Errorf("failed to generate idempotency key: %w", err)
        }
        err = h.client.Endpoint.Pause(ctx, params.Namespace, params.ID, &svix.EndpointPauseOptions{IdempotencyKey: &idempotencyKey})
        return internal.WrapSvixError(err)
    }
    return tracex.StartWithNoValue(ctx, h.tracer, "svix.pause_endpoint").Wrap(fn)
// ...
```

<!-- archie:ai-end -->
