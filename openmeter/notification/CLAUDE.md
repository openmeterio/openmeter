# notification

<!-- archie:ai-start -->

> Domain contract layer for the notification subsystem: defines all types (Channel, Rule, Event, EventPayload, delivery status), service interfaces (Service = ChannelService + RuleService + EventService + FeatureService), and EventHandler lifecycle interface. Sub-packages own persistence, HTTP, async dispatch, and Kafka consumer — this root package is the shared vocabulary they all import.

## Patterns

**Compile-time interface assertions on every domain type** — Every Input type declares `var _ models.Validator = (*XInput)(nil)` and `var _ models.CustomValidator[XInput] = (*XInput)(nil)` to guarantee Validate() and ValidateWith() are implemented. (`var (_ models.Validator = (*CreateChannelInput)(nil); _ models.CustomValidator[CreateChannelInput] = (*CreateChannelInput)(nil))`)
**models.NewNillableGenericValidationError wrapping errors.Join** — All Validate() methods accumulate errors into a []error slice and return models.NewNillableGenericValidationError(errors.Join(errs...)) — never return raw errors or construct error types directly. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Union config/payload types with type-discriminated Validate()** — ChannelConfig, RuleConfig, and EventPayload are union structs with a *Meta type field; Validate() switches on the discriminator and delegates to the sub-type validator. Adding a new event/channel type requires updating both the union struct and the Validate() switch. (`func (c RuleConfig) Validate() error { switch c.Type { case EventTypeBalanceThreshold: return c.BalanceThreshold.Validate() ... } }`)
**EventPayloadVersionCurrent = 1 on all new event payloads** — EventPayloadMeta.Version must be set to EventPayloadVersionCurrent (1) when constructing payloads for new events. The adapter reads version to apply migration logic for older payloads. (`EventPayloadMeta{Type: EventTypeInvoiceCreated, Version: EventPayloadVersionCurrent}`)
**MaxChannelsPerRule constant enforced in Input.Validate()** — CreateRuleInput and UpdateRuleInput check len(Channels) <= MaxChannelsPerRule (5) inside Validate() — never inline the constant. (`if len(i.Channels) > MaxChannelsPerRule { errs = append(errs, fmt.Errorf(...)) }`)
**eventTypes slice as the single registry for valid EventType values** — EventType.Validate() uses lo.Contains(eventTypes, t). Adding a new EventType requires appending to the eventTypes package-level var in event.go — not just adding a const. (`var eventTypes = []EventType{EventTypeBalanceThreshold, EventTypeEntitlementReset, EventTypeInvoiceCreated, EventTypeInvoiceUpdated}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the Service interface as composition of ChannelService, RuleService, EventService, and FeatureService. All HTTP handlers and consumers depend on notification.Service — never depend on the concrete service struct. | Adding methods directly to Service instead of a sub-interface breaks the smallest-surface dependency principle. |
| `eventpayload.go` | EventPayload union type + EventPayloadVersionCurrent constant. eventPayload.Validate() is the entry-point for payload correctness checks. | New event type added to EventPayload struct without updating Validate() switch — payload passes validation silently. |
| `eventhandler.go` | EventHandler interface (EventDispatcher + EventReconciler + Start/Close). Defines DefaultReconcileInterval, DefaultDispatchTimeout, DefaultDeliveryStatePendingTimeout, DefaultDeliveryStateSendingTimeout. | Implementations must not block in Dispatch — it is a fire-and-forget contract. |
| `event.go` | eventTypes registry, EventType, Event struct, all *EventInput types. Contains the canonical list of valid event types. | New EventType const without appending to eventTypes slice — EventType.Validate() will reject it. |
| `channel.go / rule.go` | Channel and Rule domain types plus all CRUD input types. ChannelConfig and RuleConfig union types with discriminated Validate(). | Adding a new channel/rule type without handling it in the corresponding Config.Validate() switch. |
| `repository.go` | Repository interface (ChannelRepository + RuleRepository + EventRepository + entutils.TxCreator) — the persistence contract. Implemented by the adapter sub-package. | Never import the adapter sub-package from this package — Repository is the interface, adapter is the implementation. |

## Anti-Patterns

- Adding business logic (Svix calls, DB queries) to this package — it is a types+interfaces package; logic belongs in service/ or adapter/ sub-packages.
- Defining a new EventType const without adding it to the eventTypes slice in event.go — breaks EventType.Validate().
- Returning raw errors from Validate() instead of models.NewNillableGenericValidationError(errors.Join(...)) — breaks HTTP 400 mapping.
- Adding a new payload variant to EventPayload struct without updating EventPayload.Validate() switch — new payloads silently bypass validation.
- Importing openmeter/notification/adapter, /service, or /webhook/svix from this package — creates import cycles; callers depend on interfaces, not implementations.

## Decisions

- **Root package contains only types, interfaces, and constants — no implementations.** — All sub-packages (adapter, service, httpdriver, consumer, eventhandler, webhook) import this package for shared types; putting any implementation here would create import cycles.
- **EventPayloadVersionCurrent is a single int constant rather than per-event versioning.** — Versioning is global to simplify migration logic in the adapter; the adapter only needs to branch on version for event types whose schema changed (currently invoice).
- **EventHandler is composed of EventDispatcher and EventReconciler sub-interfaces.** — Allows the balance-worker and other callers that only need to dispatch (not reconcile) to depend on the smaller EventDispatcher interface.

## Example: Adding a new event type: extend eventTypes registry and union types

```
// event.go — append to registry
var eventTypes = []EventType{
    EventTypeBalanceThreshold,
    EventTypeEntitlementReset,
    EventTypeInvoiceCreated,
    EventTypeInvoiceUpdated,
    EventTypeMyNew, // add here
}

// eventpayload.go — extend union
type EventPayload struct {
    EventPayloadMeta
    BalanceThreshold *BalanceThresholdPayload `json:"balanceThreshold,omitempty"`
    EntitlementReset *EntitlementResetPayload `json:"entitlementReset,omitempty"`
    Invoice          *InvoicePayload          `json:"invoice,omitempty"`
// ...
```

<!-- archie:ai-end -->
