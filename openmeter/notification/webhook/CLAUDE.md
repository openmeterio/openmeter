# webhook

<!-- archie:ai-start -->

> Contract layer for the notification webhook delivery subsystem: defines the composed Handler interface (WebhookHandler + MessageHandler + EventTypeHandler), all shared Input/Output types with validation, the typed error taxonomy, and the event-type registry. Implementations live in svix/ and noop/ sub-packages; callers import only this package.

## Patterns

**models.Validator on every Input type** — Every *Input struct implements models.Validator with a Validate() method that collects field errors into []error, joins with errors.Join, and wraps with NewValidationError. A compile-time assertion `var _ models.Validator = (*XInput)(nil)` must appear adjacent to the type. (`var _ models.Validator = (*CreateWebhookInput)(nil)
func (i CreateWebhookInput) Validate() error {
    var errs []error
    if i.Namespace == "" { errs = append(errs, errors.New("namespace is required")) }
    return NewValidationError(errors.Join(errs...))
}`)
**Typed error hierarchy with Is* helpers** — Every error category (ValidationError, NotFoundError, RetryableError, MessageAlreadyExistsError, UnrecoverableError) is a distinct struct wrapping an inner error, with a New* constructor and Is* predicate using the generic isError[T] helper. Never return raw errors from handler methods. (`func IsValidationError(err error) bool { return isError[ValidationError](err) }
func isError[T error](err error) bool { var t T; return errors.As(err, &t) }`)
**Handler interface composition over flat interface** — Handler embeds WebhookHandler + MessageHandler + EventTypeHandler. New capabilities are added to the appropriate sub-interface, not directly to Handler. Both svix/ and noop/ must satisfy all three sub-interfaces. (`type Handler interface {
    WebhookHandler
    MessageHandler
    EventTypeHandler
}`)
**EventType as a value type registered in NotificationEventTypes** — New event types are declared as package-level vars using EventType{Name, Description, GroupName} literals in events.go and appended to NotificationEventTypes. Missing the append means the event type is never registered with Svix at startup. (`var EventTypeInvoiceCreated = EventType{
    Name:        InvoiceCreatedType,
    Description: InvoiceCreatedDescription,
    GroupName:   InvoiceEventGroupName,
}`)
**MaxChannelsPerWebhook constant enforced at validation** — The hard channel limit is expressed as MaxChannelsPerWebhook (10) and ErrMaxChannelsPerWebhookExceeded in errors.go. Channel-count checks in implementations must reference these constants, never inline literals. (`var ErrMaxChannelsPerWebhookExceeded = fmt.Errorf("maximum number of channels (%d) per webhook exceeded", MaxChannelsPerWebhook)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines all Input/Output types, the Webhook and Message domain structs, ExpandParams, and the Handler interface composition. This is the only file callers need to import. | Adding a new method to Handler without updating both svix/ and noop/ implementations causes compile failures. Every new Input type needs a Validate() method and a var _ models.Validator compile-time assertion. |
| `errors.go` | Defines the full error taxonomy and the generic isError[T] helper. All error types used by svix/, noop/, and callers originate here. | Never add domain error types in svix/ or noop/ — they belong here. Svix HTTP errors are mapped to these types via internal.WrapSvixError in the svix/ sub-package, not here. |
| `events.go` | Registry of all supported notification event types as package-level vars. NotificationEventTypes slice is passed to RegisterEventTypes at startup. | Adding a new event type requires: a const block for Name+Description, a var EventType{...} value, and an append to NotificationEventTypes. Missing the append silently skips Svix registration. |

## Anti-Patterns

- Adding Svix SDK imports or Svix API calls to this package — all Svix logic belongs in webhook/svix/
- Returning raw errors from Input.Validate() — always wrap with NewValidationError so HTTP 400 mapping works via GenericErrorEncoder
- Defining new error types in svix/ or noop/ sub-packages — all error taxonomy lives in errors.go
- Adding a new Input struct without a var _ models.Validator compile-time assertion
- Hardcoding the channel limit (10) inline instead of referencing MaxChannelsPerWebhook / ErrMaxChannelsPerWebhookExceeded

## Decisions

- **Handler is a composed interface (WebhookHandler + MessageHandler + EventTypeHandler) rather than a flat 10+ method interface** — Allows callers to depend only on the capability slice they need and lets noop/ and svix/ implementations be verified against each sub-interface independently at compile time.
- **Typed error structs with Is* predicates instead of sentinel errors** — Enables the HTTP encoder chain to pattern-match on error category (ValidationError → 400, NotFoundError → 404, RetryableError → retry logic) without importing Svix types into the HTTP layer.
- **events.go holds all EventType definitions as package-level vars, not inside sub-packages** — Ensures the canonical event-type list is visible to all consumers (notification service, test helpers) without requiring import of the Svix implementation.

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
