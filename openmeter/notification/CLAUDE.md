# notification

<!-- archie:ai-start -->

> Notification domain root: defines all shared vocabulary (Channel, Rule, Event, EventPayload/RuleConfig/ChannelConfig union types, EventDeliveryStatus) plus the composite Service and EventHandler interfaces. Implementation is split across sub-packages — only types, interfaces, and constants live in the root, so any concrete code here would create import cycles between adapter, service, consumer, and eventhandler.

## Patterns

**Root is types+interfaces only; subtrees implement** — service.go/event.go/channel.go/rule.go/eventpayload.go/repository.go define contracts; adapter/ (Ent), service/ (orchestration), httpdriver/ (v1 HTTP), consumer/ (Kafka), eventhandler/ (dispatch+reconcile), webhook/ (Svix/noop) each import the root for shared types. Never import a sub-package from the root. (`type Service interface { FeatureService; ChannelService; RuleService; EventService }`)
**Every Input declares compile-time Validator assertions** — Each *Input type carries `var _ models.Validator` and `var _ models.CustomValidator[T]` and accumulates errors via models.NewNillableGenericValidationError(errors.Join(errs...)) — never raw errors. (`func (i CreateRuleInput) Validate() error { /* errs ... */ return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Type-discriminated union Validate()** — ChannelConfig, RuleConfig, EventPayload switch on a *Meta.Type discriminator (EventType) and delegate. Adding a new event/channel type requires updating the union struct, its Validate() switch, AND the eventTypes registry in event.go. (`switch c.Type { case EventTypeBalanceThreshold: return c.BalanceThreshold.Validate(); ... }`)
**eventTypes slice is the single EventType registry** — EventType.Validate() uses lo.Contains(eventTypes, t). A new EventType const must be appended to eventTypes in event.go, not just declared. (`var eventTypes = []EventType{EventTypeBalanceThreshold, EventTypeEntitlementReset, EventTypeInvoiceCreated, EventTypeInvoiceUpdated}`)
**EventPayloadVersionCurrent on all new payloads** — EventPayloadMeta.Version = EventPayloadVersionCurrent (1) on construction; the adapter branches on version to migrate older invoice payloads. (`EventPayloadMeta{Type: EventTypeInvoiceCreated, Version: EventPayloadVersionCurrent}`)
**Cross-cutting limits enforced via named constants** — MaxChannelsPerRule (5) is checked inside Create/UpdateRuleInput.Validate(); webhook/ enforces MaxChannelsPerWebhook. Never inline the numeric limit. (`if len(i.Channels) > MaxChannelsPerRule { errs = append(errs, fmt.Errorf(...)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Composite Service = FeatureService + ChannelService + RuleService + EventService; the dependency surface for all handlers and consumers. | Adding methods directly to Service instead of a sub-interface breaks smallest-surface dependency. |
| `event.go` | eventTypes registry, EventType, Event struct, *EventInput types. | New EventType const without appending to eventTypes — Validate() rejects it. |
| `eventpayload.go` | EventPayload union + EventPayloadVersionCurrent; Validate() is the payload-correctness entry point. | New payload variant without updating the Validate() switch silently bypasses validation. |
| `eventhandler.go` | EventHandler = EventDispatcher + EventReconciler + Start/Close; default reconcile/dispatch timeouts. | Dispatch is fire-and-forget — implementations must not block. |
| `repository.go` | Persistence contract (Channel/Rule/EventRepository + entutils.TxCreator) implemented only by adapter/. | Importing the adapter sub-package here causes an import cycle. |
| `channel.go / rule.go` | Channel/Rule + their *Config union types with discriminated Validate() and ListChannelsInput/ListRulesInput. | New channel/rule type not handled in the Config.Validate() switch silently passes. |

## Anti-Patterns

- Adding business logic (Svix calls, Ent queries) to the root package — it is types+interfaces only.
- Declaring a new EventType const without appending it to the eventTypes slice in event.go.
- Returning raw errors from Validate() instead of models.NewNillableGenericValidationError(errors.Join(...)) — breaks HTTP 400 mapping via GenericErrorEncoder.
- Extending an EventPayload/RuleConfig/ChannelConfig union without updating its Validate() switch.
- Importing openmeter/notification/adapter, /service, /consumer, or /webhook/svix from the root — creates import cycles.

## Decisions

- **Root holds only types, interfaces, and constants.** — All sub-packages import the root for shared vocabulary; any implementation here would create cycles between adapter, service, consumer, and eventhandler.
- **Single EventPayloadVersionCurrent int rather than per-event versioning.** — Keeps adapter migration logic simple — it branches on version only for event families whose schema changed (currently invoice).
- **EventHandler split into EventDispatcher and EventReconciler.** — Lets the Watermill consumer depend on the smaller EventDispatcher while the standalone service drives the full reconcile loop.

## Example: Adding a new event type: extend the registry and union payload

```
// event.go
var eventTypes = []EventType{EventTypeBalanceThreshold, EventTypeEntitlementReset, EventTypeInvoiceCreated, EventTypeInvoiceUpdated, EventTypeMyNew}

// eventpayload.go
type EventPayload struct {
    EventPayloadMeta
    BalanceThreshold *BalanceThresholdPayload `json:"balanceThreshold,omitempty"`
    MyNew            *MyNewPayload            `json:"myNew,omitempty"`
}
func (p EventPayload) Validate() error {
    switch p.Type {
    case EventTypeMyNew:
        if p.MyNew == nil { return models.NewGenericValidationError(errors.New("missing myNew payload")) }
        return p.MyNew.Validate()
    }
// ...
```

<!-- archie:ai-end -->
