# grouphandler

<!-- archie:ai-start -->

> Provides NoPublishingHandler, a multiplexed Watermill consumer handler that dispatches a single Kafka message to all registered GroupEventHandlers matching the CloudEvents ce_type header, with per-type OTel metrics. Unknown event types are silently ACKed (dropped, not errored) to support rolling deploys.

## Patterns

**NewGroupEventHandler generic constructor** — Use NewGroupEventHandler[T](func(ctx context.Context, event *T) error) to register a typed handler. The generic parameter T is the concrete event struct. Do not implement GroupEventHandler manually. (`grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error { return svc.OnInvoiceCreated(ctx, ev) })`)
**Silent drop on unknown event type** — If no handler matches ce_type, the message is ACKed and the ignored metric is incremented. Never return an error for unknown event types — workers must tolerate schema evolution during rolling deploys. (`if !ok || len(groupHandler) == 0 { h.meters.handlerMessageCount.Add(..., meterAttributeStatusIgnored); return nil }`)
**errors.Join fan-out over all handlers for a type** — All handlers for the same event type receive the same deserialized event pointer via errors.Join(lo.Map(...)). A failure in any handler surfaces as an error causing Watermill retry; partial success is not possible. (`err := errors.Join(lo.Map(groupHandler, func(h GroupEventHandler, _ int) error { return h.Handle(msg.Context(), event) })...)`)
**AddHandler for post-construction registration** — Use AddHandler to register additional handlers after NewNoPublishingHandler returns; it acquires the write lock safely. Used by BillingWorker to attach external handlers at startup. (`handler.AddHandler(grouphandler.NewGroupEventHandler(myOtherHandler))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grouphandler.go` | Entire package in one file. Defines NoPublishingHandler (the mux), GroupEventHandler alias, NewGroupEventHandler generic, and OTel meters. | The first handler in the slice for a type is used for Unmarshal (NewEvent()); all handlers share the same deserialized event pointer. If handlers mutate the event, they will race and corrupt each other's view. |

## Anti-Patterns

- Returning an error for unrecognised event types — causes Watermill to retry and eventually DLQ valid messages from other event families on the same topic.
- Mutating the event pointer inside a GroupEventHandler when multiple handlers are registered for the same type — they share one deserialized instance.
- Bypassing NoPublishingHandler and writing a raw message.NoPublishHandlerFunc — per-type OTel metrics and type-dispatch would be missing.
- Passing a non-pointer event struct that uses pointer receivers to NewGroupEventHandler — type assertion for NewEvent() will fail to deserialize correctly.

## Decisions

- **errors.Join fan-out rather than early-exit on handler failure** — All handlers for an event type are equal observers; partial failure must surface as an error so Watermill retries, not silently swallow one handler's failure while others succeed.

## Example: Registering typed event handlers and wiring into the Watermill router

```
import (
    "github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
    billingevents "github.com/openmeterio/openmeter/openmeter/billing/worker/events"
)

handler, err := grouphandler.NewNoPublishingHandler(
    m.Marshaler(),
    metricMeter,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error {
        return svc.OnInvoiceCreated(ctx, ev)
    }),
)
// Later, post-construction:
handler.AddHandler(grouphandler.NewGroupEventHandler(myOtherHandler))
```

<!-- archie:ai-end -->
