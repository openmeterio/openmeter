# webhook

<!-- archie:ai-start -->

> Contract layer for the notification webhook delivery subsystem: defines the composed Handler interface (WebhookHandler + MessageHandler + EventTypeHandler), all shared Input/Output types with validation, the typed error taxonomy, and the event-type registry. Implementations live in svix/ (production) and noop/ (Svix-unconfigured fallback); the secret/ child generates/validates whsec_ signing secrets. Callers import only this package.

## Patterns

**models.Validator on every Input type** — Every *Input struct implements models.Validator: Validate() collects field errors into []error, joins with errors.Join, and wraps with NewValidationError. A compile-time assertion `var _ models.Validator = (*XInput)(nil)` sits adjacent to the type. (`var _ models.Validator = (*CreateWebhookInput)(nil)
func (i CreateWebhookInput) Validate() error { var errs []error; ...; return NewValidationError(errors.Join(errs...)) }`)
**Typed error hierarchy with Is* helpers** — Each error category (ValidationError, NotFoundError, RetryableError, MessageAlreadyExistsError, UnrecoverableError) is a struct wrapping an inner error, with a New* constructor and an Is* predicate using the generic isError[T] helper. Handler methods never return raw errors. (`func IsValidationError(err error) bool { return isError[ValidationError](err) }
func isError[T error](err error) bool { var t T; return errors.As(err, &t) }`)
**Handler interface composition over flat interface** — Handler embeds WebhookHandler + MessageHandler + EventTypeHandler. New capabilities go on the appropriate sub-interface; both svix/ and noop/ must satisfy all three. (`type Handler interface { WebhookHandler; MessageHandler; EventTypeHandler }`)
**EventType registered in NotificationEventTypes** — New event types are package-level vars using EventType{Name, Description, GroupName} literals in events.go and appended to NotificationEventTypes. Missing the append means the type is never registered with Svix at startup. (`var EventTypeInvoiceCreated = EventType{ Name: InvoiceCreatedType, Description: InvoiceCreatedDescription, GroupName: InvoiceEventGroupName }`)
**MaxChannelsPerWebhook enforced via constant** — The hard channel limit is MaxChannelsPerWebhook (10) with ErrMaxChannelsPerWebhookExceeded in errors.go. Channel-count checks reference these constants, never inline literals. (`var ErrMaxChannelsPerWebhookExceeded = fmt.Errorf("maximum number of channels (%d) per webhook exceeded", MaxChannelsPerWebhook)`)
**Signing-secret validation delegated to secret/** — CreateWebhookInput/UpdateWebhookInput validate a provided secret via secret.ValidateSigningSecret; the secret/ sub-package owns whsec_-prefixed HMAC secret generation/validation using crypto/rand and standard base64. (`if err := secret.ValidateSigningSecret(*i.Secret); err != nil { errs = append(errs, fmt.Errorf("invalid secret: %w", err)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | All Input/Output types, Webhook/Message domain structs, ExpandParams, EventType, the three sub-interfaces and the composed Handler interface, plus MaxChannelsPerWebhook/DefaultRegistrationTimeout consts. The only file callers import. | Adding a method to Handler without updating both svix/ and noop/ breaks compilation. Every new Input needs Validate() + a var _ models.Validator assertion. |
| `errors.go` | Full error taxonomy (ValidationError, NotFoundError, RetryableError w/ RetryAfter, MessageAlreadyExistsError, UnrecoverableError, ErrNotImplemented) + the generic isError[T] helper. | Never add domain error types in svix/ or noop/ — they belong here. Svix HTTP errors map to these via internal.WrapSvixError in the svix/ sub-package, not here. |
| `events.go` | Registry of supported notification event types as package-level vars; NotificationEventTypes slice passed to RegisterEventTypes at startup (entitlements balance/reset, invoice created/updated). | A new event type needs a const block (Name+Description), a var EventType{...}, AND an append to NotificationEventTypes — missing the append silently skips Svix registration. |

## Anti-Patterns

- Adding Svix SDK imports or Svix API calls to this package — all Svix logic belongs in webhook/svix/
- Returning raw errors from Input.Validate() — always wrap with NewValidationError so HTTP 400 mapping works via GenericErrorEncoder
- Defining new error types in svix/ or noop/ — all error taxonomy lives in errors.go
- Adding a new Input struct without a var _ models.Validator compile-time assertion
- Hardcoding the channel limit (10) inline instead of referencing MaxChannelsPerWebhook / ErrMaxChannelsPerWebhookExceeded

## Decisions

- **Handler is a composed interface (WebhookHandler + MessageHandler + EventTypeHandler) rather than a flat 10+ method interface** — Lets callers depend only on the capability slice they need and lets noop/ and svix/ be verified against each sub-interface independently at compile time.
- **Typed error structs with Is* predicates instead of sentinel errors** — Enables the HTTP encoder chain to pattern-match on error category (ValidationError→400, NotFoundError→404, RetryableError→retry) without importing Svix types into the HTTP layer.
- **events.go holds all EventType definitions as package-level vars, not inside sub-packages** — Keeps the canonical event-type list visible to all consumers (notification service, test helpers) without importing the Svix implementation.

## Example: Add a new notification event type (e.g. invoice.voided)

```
// events.go
const (
    InvoiceVoidedType        = "invoice.voided"
    InvoiceVoidedDescription = "Notification event for voided invoice."
)
var EventTypeInvoiceVoided = EventType{ Name: InvoiceVoidedType, Description: InvoiceVoidedDescription, GroupName: InvoiceEventGroupName }
// append to NotificationEventTypes so it is registered with Svix at startup
var NotificationEventTypes = []EventType{ EventTypeInvoiceCreated, EventTypeInvoiceUpdated, EventTypeInvoiceVoided }
```

<!-- archie:ai-end -->
