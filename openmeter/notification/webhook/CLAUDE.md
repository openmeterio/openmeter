# webhook

<!-- archie:ai-start -->

> Defines the webhook.Handler interface (WebhookHandler + MessageHandler + EventTypeHandler), all shared input/output types, the error taxonomy, and the event-type registry for the notification delivery subsystem. It is the contract layer — implementations (svix/, noop/) live in sub-packages; callers depend only on this package.

## Patterns

**models.Validator on every Input type** — Every *Input struct implements models.Validator with a Validate() method that collects field errors into []error, joins with errors.Join, and wraps with NewValidationError. The compile-time assertion var _ models.Validator = (*XInput)(nil) must appear adjacent to the type. (`var _ models.Validator = (*CreateWebhookInput)(nil)
func (i CreateWebhookInput) Validate() error {
    var errs []error
    if i.Namespace == "" { errs = append(errs, errors.New("namespace is required")) }
    return NewValidationError(errors.Join(errs...))
}`)
**Typed error hierarchy with Is* helpers** — Every error category (ValidationError, NotFoundError, RetryableError, MessageAlreadyExistsError, UnrecoverableError) is a distinct struct wrapping an inner error, with a public New* constructor and Is* predicate using errors.As via the generic isError[T] helper. Never return raw errors from handler methods. (`func IsValidationError(err error) bool { return isError[ValidationError](err) }`)
**Handler interface composition** — Handler embeds WebhookHandler + MessageHandler + EventTypeHandler. New capabilities are added to the appropriate sub-interface, not directly to Handler. (`type Handler interface {
    WebhookHandler
    MessageHandler
    EventTypeHandler
}`)
**EventType as a value type with group/schema metadata** — EventType carries Name, Description, GroupName, Schemas map, and Deprecated. New event types are declared as package-level vars using EventType literals and appended to NotificationEventTypes in events.go. (`var EventTypeInvoiceCreated = EventType{
    Name: InvoiceCreatedType,
    Description: InvoiceCreatedDescription,
    GroupName: InvoiceEventGroupName,
}`)
**MaxChannelsPerWebhook constant enforced at validation** — The hard limit (10) is expressed as MaxChannelsPerWebhook and ErrMaxChannelsPerWebhookExceeded in errors.go. Channel-count checks in implementations must reference these constants, not inline literals. (`var ErrMaxChannelsPerWebhookExceeded = fmt.Errorf("maximum number of channels (%d) per webhook exceeded", MaxChannelsPerWebhook)`)
**RetryableError carries retry duration** — Transient failures from Svix (rate limits, etc.) must be returned as RetryableError with a non-zero retryAfter so callers can back off correctly. DefaultRetryAfter (15s) is used when after==0. (`return webhook.NewRetryableError(err, svixRateLimitRetryAfter)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines all Input/Output types, the Webhook and Message domain structs, ExpandParams, and the Handler interface composition. This is the only file callers need to import. | Adding a new method to Handler without updating both the svix/ and noop/ implementations causes compile failures. Every new Input type needs a Validate() method and a var _ models.Validator assertion. |
| `errors.go` | Defines the full error taxonomy: ValidationError, NotFoundError, RetryableError, MessageAlreadyExistsError, UnrecoverableError, ErrNotImplemented, ErrMaxChannelsPerWebhookExceeded. Contains the generic isError[T] helper. | Never add domain error types in sub-packages (svix/, noop/) — they belong here. Svix-specific HTTP errors are mapped to these types via internal.WrapSvixError in the svix/ sub-package. |
| `events.go` | Registry of all supported notification event types as package-level vars. NotificationEventTypes slice is passed to RegisterEventTypes at startup. | Adding a new event type requires: a const block for Name+Description, a var EventType{...} value, and appending to NotificationEventTypes. Missing the NotificationEventTypes append means the event type is never registered with Svix. |

## Anti-Patterns

- Adding Svix API calls or Svix SDK imports to this package — all Svix logic belongs in webhook/svix/
- Returning raw errors from Input.Validate() — always wrap with NewValidationError so HTTP 400 mapping works
- Defining error types in svix/ or noop/ sub-packages — all error taxonomy lives in errors.go
- Adding a new Input struct without a var _ models.Validator compile-time assertion
- Hardcoding the channel limit (10) inline instead of referencing MaxChannelsPerWebhook

## Decisions

- **Handler is a composed interface (WebhookHandler + MessageHandler + EventTypeHandler) rather than a flat interface** — Allows callers to depend only on the capability slice they need and lets noop/svix implementations be verified against each sub-interface independently.
- **Typed error structs with Is* predicates instead of sentinel errors** — Enables the HTTP encoder chain to pattern-match on error category (ValidationError → 400, NotFoundError → 404, RetryableError → retry logic) without importing Svix types into the HTTP layer.
- **events.go holds all EventType definitions as package-level vars, not inside sub-packages** — Ensures the canonical event-type list is visible to all consumers (notification service, test helpers) without importing the Svix implementation.

## Example: Add a new notification event type (e.g. invoice.voided)

```
// events.go
const (
    InvoiceVoidedType        = "invoice.voided"
    InvoiceVoidedDescription = "Notification event for voided invoice."
)

var EventTypeInvoiceVoided = EventType{
    Name:        InvoiceVoidedType,
    Description: InvoiceVoidedDescription,
    GroupName:   InvoiceEventGroupName,
}

// append to NotificationEventTypes:
var NotificationEventTypes = []EventType{
    EventTypeEntitlementsBalanceThreshold,
// ...
```

<!-- archie:ai-end -->
