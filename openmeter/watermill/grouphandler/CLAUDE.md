# grouphandler

<!-- archie:ai-start -->

> Provides NoPublishingHandler, a multiplexed Watermill consumer handler that dispatches a single Kafka message to all registered GroupEventHandlers matching the CloudEvents ce_type header, with per-type OTel metrics. Unknown event types are silently ACKed (dropped) to support rolling deploys.

## Patterns

**NewGroupEventHandler generic constructor** — Use NewGroupEventHandler[T](func(ctx, *T) error) to register a typed handler; T is the concrete event struct. Do not implement GroupEventHandler manually. (`grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error { return svc.OnInvoiceCreated(ctx, ev) })`)
**Silent drop on unknown event type** — If no handler matches ce_type, the message is ACKed and the ignored metric incremented; never return an error for unknown types so workers tolerate schema evolution. (`if !ok || len(groupHandler) == 0 { h.meters.handlerMessageCount.Add(..., meterAttributeStatusIgnored); return nil }`)
**errors.Join fan-out over all handlers** — All handlers for the same event type receive the same deserialized event pointer via errors.Join(lo.Map(...)); any failure causes Watermill retry — no partial success. (`err := errors.Join(lo.Map(groupHandler, func(h GroupEventHandler, _ int) error { return h.Handle(msg.Context(), event) })...)`)
**AddHandler for post-construction registration** — Use AddHandler to register additional handlers after NewNoPublishingHandler returns; it acquires the write lock safely. (`handler.AddHandler(grouphandler.NewGroupEventHandler(myOtherHandler))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grouphandler.go` | Entire package: NoPublishingHandler mux, GroupEventHandler alias, NewGroupEventHandler generic, OTel meters. | The first handler in the slice for a type is used for Unmarshal (NewEvent()); all handlers share one deserialized pointer — mutating it races across handlers. |

## Anti-Patterns

- Returning an error for unrecognised event types — causes retry and DLQ of valid messages from other families on the same topic.
- Mutating the event pointer inside a handler when multiple handlers share the same type — they share one deserialized instance.
- Bypassing NoPublishingHandler with a raw message.NoPublishHandlerFunc — per-type OTel metrics and type-dispatch would be missing.
- Passing a non-pointer event struct (with pointer receivers) to NewGroupEventHandler — NewEvent() deserialization fails.

## Decisions

- **errors.Join fan-out rather than early-exit on handler failure.** — All handlers for an event type are equal observers; partial failure must surface so Watermill retries rather than silently swallowing one handler's failure.

## Example: Registering typed handlers and wiring into the router

```
import (
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	billingevents "github.com/openmeterio/openmeter/openmeter/billing/worker/events"
)
handler, err := grouphandler.NewNoPublishingHandler(
	m.Marshaler(), metricMeter,
	grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error { return svc.OnInvoiceCreated(ctx, ev) }),
)
handler.AddHandler(grouphandler.NewGroupEventHandler(myOtherHandler))
```

<!-- archie:ai-end -->
