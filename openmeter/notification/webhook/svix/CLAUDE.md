# svix

<!-- archie:ai-start -->

> Svix-backed implementation of the notification webhook.Handler interface. Translates the domain's webhook operations (applications, endpoints, event types, messages, delivery status) into Svix SDK calls, with OTel tracing and idempotency on every mutating call.

## Patterns

**Handler methods wrap a closure in tracex** — Every webhook.Handler method on svixHandler defines an inner `fn := func(ctx) (...) {...}` and returns `tracex.Start[T](ctx, h.tracer, "svix.<op>").Wrap(fn)` (or `tracex.StartWithNoValue` for error-only). Span names are `svix.<snake_case_op>`. (`return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.create_webhook").Wrap(fn)`)
**All Svix errors funnel through internal.WrapSvixError** — Immediately after any `h.client.*` call, reassign `if err = internal.WrapSvixError(err); err != nil`. Never return a raw *svix.Error — it must become a webhook-domain error first. `Error` is aliased as `internal.SvixError`. (`endpoint, err := h.client.Endpoint.Create(ctx, app.Id, input, opts)
if err = internal.WrapSvixError(err); err != nil { return nil, fmt.Errorf("failed to create Svix endpoint: %w", err) }`)
**Idempotency key per mutating Svix call** — Create/Update/Resend/RotateSecret pass an IdempotencyKey from `idempotency.Key()` in the Svix *Options struct. Generate it, error-check, then expose it as the span attribute "idempotency_key". (`idempotencyKey, err := idempotency.Key()
... h.client.Application.GetOrCreate(ctx, input, &svix.ApplicationCreateOptions{IdempotencyKey: &idempotencyKey})`)
**Namespace IS the Svix application UID** — The notification namespace is used directly as the Svix application Name/Uid (CreateApplication) and passed as the appID arg to Message/Endpoint calls. Webhook objects get `wh.Namespace = params.Namespace` set after mapping. (`input := svix.ApplicationIn{Name: id, Uid: &id}`)
**Validate inputs first** — Public methods taking a webhook.*Input call `params.Validate()` and wrap failures with fmt.Errorf before any Svix call. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid send message params: %w", err) }`)
**NullChannel guards against unfiltered fan-out** — When creating/updating an endpoint with no Channels (and no EventTypes), set Channels to []string{NullChannel} so a filterless Svix endpoint does not receive every message. Strip NullChannel back out in WebhookFromSvixEndpointOut. (`if len(params.Channels) == 0 { params.Channels = []string{NullChannel} }`)
**Paginate Svix lists via Iterator/Done loop** — List operations (ListWebhooks, getDeliveryStatus) loop with Limit + Iterator, breaking on `out.Done`, sleeping `time.Sleep(100 * time.Millisecond)` between pages to avoid Svix rate limits. (`if !out.Done { opts.Iterator = out.Iterator; time.Sleep(100*time.Millisecond); continue }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `svix.go` | Config types (SvixConfig with Validate/IsEnabled, Config with SvixAPIClient/Logger/Tracer), constructors New + NewHandler, the svixHandler struct, NullChannel const, and `type Error = internal.SvixError` alias. `var _ webhook.Handler = (*svixHandler)(nil)` enforces the interface. | New() optionally registers event types on startup (RegisterEventTypes) honoring SkipRegistrationOnError, using context.Background() with RegistrationTimeout. Config.Validate requires non-nil client, logger, tracer. |
| `webhook.go` | Endpoint CRUD (Create/Update/Delete/Get/List Webhook), UpdateWebhookChannels, header/secret helpers (GetOrUpdateEndpointHeaders, GetOrUpdateEndpointSecret), and the WebhookFromSvixEndpointOut mapper. | CreateWebhook auto-ensures the Svix application, auto-generates a signing secret via webhooksecret.NewSigningSecretWithDefaultSize() and a ULID UID when absent. UpdateWebhookChannels re-reads, merges add/remove, enforces webhook.MaxChannelsPerWebhook. DeleteWebhook swallows webhook.IsNotFoundError as success. |
| `message.go` | SendMessage, GetMessage, ResendMessage, getDeliveryStatus, and deliveryStateFromSvixMessageStatus mapper. Bridges Svix message attempts to notification.EventDeliveryAttempt / webhook.MessageDeliveryStatus. | SendMessage maps http.StatusConflict to webhook.NewMessageAlreadyExistsError (EventID already published). getDeliveryStatus treats Svix FAIL with a non-nil NextAttempt as EventDeliveryStatusStateSending (transient). SendMessage sets WithContent:false so payload comes from the request, not Svix. |
| `application.go` | CreateApplication — GetOrCreate a Svix application keyed by namespace/id, used implicitly by webhook create/update flows. | Uses GetOrCreate (idempotent), not Create; both Name and Uid are set to the namespace id. |
| `event.go` | RegisterEventTypes — upserts Svix EventType schemas per webhook.EventType (loops, calls EventType.Update). | Uses Update (upsert) per event type; failure aborts the loop. Called from New() at handler init. |
| `annotations.go` | String constants for OTel span attribute keys (svix.message.id, svix.application.uid, svix.endpoint.url, etc.) used across all handler files. | Add new annotation keys here, not as inline string literals, to keep span attributes consistent. |
| `internal/error.go` | WrapSvixError / SvixError taxonomy — sole boundary decoding *svix.Error HTTP failures into webhook recoverable/retryable/validation/not-found errors. | Internal package — only importable from within svix/. All status-class decisions live here, not at call sites. |

## Anti-Patterns

- Returning a raw *svix.Error / unwrapped err from a handler method instead of passing it through internal.WrapSvixError first.
- Calling a Svix mutating endpoint (Create/Update/Resend/RotateSecret) without an idempotency.Key() in the *Options struct.
- Skipping the tracex.Start/StartWithNoValue wrapper on a new handler method, losing the span and consistent error handling.
- Creating or updating an endpoint with empty Channels and EventTypes without falling back to NullChannel — produces an endpoint that receives all messages.
- Hardcoding span attribute key strings inline instead of using the Annotation* constants in annotations.go.

## Decisions

- **Use the notification namespace directly as the Svix application UID rather than a separate mapping table.** — Svix applications are 1:1 with OpenMeter namespaces, so the namespace is a natural idempotent key and avoids extra persistence.
- **Centralize Svix error classification in internal.WrapSvixError and re-export only `type Error = internal.SvixError`.** — Keeps retryable-vs-fatal decisions in one auditable place; delivery/consumer layers reason in webhook-domain errors, never Svix SDK shapes.
- **Treat a Svix FAIL status with a pending NextAttempt as still-sending.** — Svix reports FAIL during transient retries; mapping it to EventDeliveryStatusStateSending avoids surfacing premature failures to the notification domain.

## Example: Adding a new svixHandler method: validate, call Svix, wrap errors, trace.

```
func (h svixHandler) DeleteWebhook(ctx context.Context, params webhook.DeleteWebhookInput) error {
	fn := func(ctx context.Context) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("failed to validate DeleteWebhookInputs: %w", err)
		}
		err := h.client.Endpoint.Delete(ctx, params.Namespace, params.ID)
		if err = internal.WrapSvixError(err); err != nil {
			if webhook.IsNotFoundError(err) {
				return nil
			}
			return fmt.Errorf("failed to delete Svix endpoint: %w", err)
		}
		return nil
	}
	return tracex.StartWithNoValue(ctx, h.tracer, "svix.delete_webhook").Wrap(fn)
// ...
```

<!-- archie:ai-end -->
