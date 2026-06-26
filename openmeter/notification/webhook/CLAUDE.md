# webhook

<!-- archie:ai-start -->

> Defines the transport-agnostic contract for notification webhook delivery: the webhook.Handler interface (Webhook CRUD + Message send/get/resend + EventType registration), input/output domain types, and a typed error taxonomy. Concrete delivery lives in the svix child; noop is the disabled-backend fallback. The root itself is pure interface + types + errors with no Svix dependency.

## Patterns

**Handler interface composed from sub-interfaces** — The public Handler embeds WebhookHandler, MessageHandler, and EventTypeHandler. New delivery operations must be added to one of the three sub-interfaces, not as loose top-level interfaces, so every backend (svix, noop) implements them uniformly. (`type Handler interface { WebhookHandler; MessageHandler; EventTypeHandler }`)
**Input structs are models.Validator with joined errors** — Every *Input type asserts var _ models.Validator = (*XInput)(nil) and its Validate() collects into var errs []error, returning NewValidationError(errors.Join(errs...)). Never return on first invalid field. (`func (i CreateWebhookInput) Validate() error { var errs []error; if i.URL == "" { errs = append(errs, errors.New("url is required")) }; return NewValidationError(errors.Join(errs...)) }`)
**Typed error constructors + Is* predicates** — errors.go defines NotFoundError, ValidationError, RetryableError, UnrecoverableError, MessageAlreadyExistsError via New*/Is* pairs over a generic isError[T] using errors.As. Callers classify with the Is* predicates, never type-switch directly. (`if webhook.IsMessageAlreadyExistsError(err) { /* idempotent skip */ }`)
**Namespace as first field, threaded everywhere** — Every Input/output struct (Webhook, Message, SendMessageInput, ...) carries a Namespace string for multi-tenancy; the svix backend maps it directly to the Svix application UID. (`type SendMessageInput struct { Namespace string; EventID string; ... }`)
**Event type catalog is centralized constants** — events.go is the single source of truth for notification event types (NotificationEventTypes slice + EventTypeEntitlements*/EventTypeInvoice* values keyed by GroupName). New event types are registered here, not redefined per backend. (`var NotificationEventTypes = []EventType{ EventTypeEntitlementsBalanceThreshold, EventTypeEntitlementsReset, EventTypeInvoiceCreated, EventTypeInvoiceUpdated }`)
**Secret validation delegated to webhook/secret** — CreateWebhookInput/UpdateWebhookInput.Validate call secret.ValidateSigningSecret for non-empty secrets rather than inlining whsec_ format checks. (`if err := secret.ValidateSigningSecret(*i.Secret); err != nil { errs = append(errs, fmt.Errorf("invalid secret: %w", err)) }`)
**Channel cap enforced via shared constant + sentinel error** — MaxChannelsPerWebhook (10) and ErrMaxChannelsPerWebhookExceeded are defined once here; backends reference them rather than redefining the limit. (`const MaxChannelsPerWebhook = 10`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Declares the Handler interface (and its three component interfaces) plus all Webhook/Message/EventType domain and Input types with their Validate methods. | DeleteWebhookInput is a type alias of GetWebhookInput; Payload is map[string]any. Adding a method here means every implementer (svix, noop) must be updated or the build breaks via their compile-time assertions. |
| `errors.go` | Typed error taxonomy with New*/Is* helpers built on generic isError[T] (errors.As). | All New* constructors return nil when given a nil err. RetryableError defaults to DefaultRetryAfter (15s) when after==0. GetMessage/ResendMessage inputs validate that ID OR EventID is present, not both. |
| `events.go` | Canonical registry of notification EventType values and the NotificationEventTypes slice used to register types with the delivery backend. | FIXME comments note JSON schemas are not yet attached; a new event type must be both declared and appended to NotificationEventTypes or it will not be registered. |

## Anti-Patterns

- Adding a webhook operation as a new top-level interface instead of extending WebhookHandler/MessageHandler/EventTypeHandler — breaks the uniform Handler contract.
- Returning on the first failed field in a Validate() method instead of joining all errs via errors.Join + NewValidationError.
- Type-switching on concrete error types instead of using the Is* predicates (IsNotFoundError, IsRetryableError, etc.).
- Hardcoding the channel limit (10) or whsec_ secret format in callers instead of using MaxChannelsPerWebhook / the secret package.
- Importing the svix SDK or Svix-specific logic into this root package — concrete delivery belongs in webhook/svix; this layer stays transport-agnostic.

## Decisions

- **Split Handler into WebhookHandler + MessageHandler + EventTypeHandler.** — Keeps responsibilities separable so implementers and tests reason about webhook lifecycle, message delivery, and type registration independently while still satisfying one Handler.
- **Keep the delivery contract, types, and errors in the root with no Svix dependency.** — Allows noop and svix backends to be swapped via DI (app/common) without callers depending on a specific provider.

## Example: Defining a validated webhook input and classifying the resulting error

```
var _ models.Validator = (*SendMessageInput)(nil)

func (i SendMessageInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if len(i.Payload) == 0 {
		errs = append(errs, errors.New("payload must not be empty"))
	}
	return NewValidationError(errors.Join(errs...))
}

// caller side:
if _, err := handler.SendMessage(ctx, in); webhook.IsMessageAlreadyExistsError(err) {
// ...
```

<!-- archie:ai-end -->
