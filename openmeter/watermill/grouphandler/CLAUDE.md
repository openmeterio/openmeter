# grouphandler

<!-- archie:ai-start -->

> Provides NoPublishingHandler, a multiplexed Watermill handler that dispatches a single Kafka message to all registered GroupEventHandlers matching the CloudEvents ce_type header, with per-type OTel metrics. Unknown event types are silently ignored (dropped, not errored).

## Patterns

**Type-keyed handler map with RWMutex** — typeHandlerMap maps marshaler.Name(event) -> []GroupEventHandler. Handlers are looked up by ce_type on every message. Multiple handlers for the same event type are fanned out via errors.Join(lo.Map(...)). (`typeHandlerMap[marshaler.Name(event)] = append(typeHandlerMap[...], handler)`)
**Silent drop on unknown event type** — If no handler matches ce_type, the message is ACKed with status=ignored metric. Never return an error for unknown event types — workers must tolerate schema evolution. (`if !ok || len(groupHandler) == 0 { h.meters.handlerMessageCount.Add(..., meterAttributeStatusIgnored); return nil }`)
**AddHandler for post-construction registration** — Use AddHandler to register additional handlers after NewNoPublishingHandler; it acquires the write lock. Used by BillingWorker to attach external handlers at startup. (`handler.AddHandler(grouphandler.NewGroupEventHandler(myFunc))`)
**NewGroupEventHandler generic constructor** — Use NewGroupEventHandler[T](func(ctx, *T) error) to create a typed handler — the generic parameter T is the event struct. Do not implement GroupEventHandler manually. (`grouphandler.NewGroupEventHandler[billingevents.InvoiceCreated](func(ctx context.Context, ev *billingevents.InvoiceCreated) error { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grouphandler.go` | Entire package in one file. Defines NoPublishingHandler (the mux), GroupEventHandler alias, NewGroupEventHandler, and OTel meters. | The first handler in the slice for a given type is used for Unmarshal (NewEvent()); all handlers share the same deserialized event pointer. If handlers mutate the event, they will race. |

## Anti-Patterns

- Returning an error for unrecognised event types — this will cause Watermill to retry and eventually DLQ valid messages from other event families on the same topic.
- Mutating the event pointer inside a GroupEventHandler when multiple handlers are registered for the same type — they share one deserialized instance.
- Registering handlers after the router has started without verifying AddHandler's lock safety (it is safe; the RWMutex guards concurrent access).
- Bypassing NoPublishingHandler and writing a raw message.NoPublishHandlerFunc — metrics and type dispatch would be missing.

## Decisions

- **errors.Join fan-out over all handlers for a type** — All handlers for an event type are considered equal observers; partial failure must surface as an error so Watermill retries, not silently swallow one handler's failure.

## Example: Registering typed event handlers and wiring into the Watermill router

```
import (
    "github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
    "github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
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
// ...
```

<!-- archie:ai-end -->
