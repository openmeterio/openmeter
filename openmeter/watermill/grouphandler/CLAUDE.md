# grouphandler

<!-- archie:ai-start -->

> Fan-out event dispatcher: a single Watermill NoPublishHandlerFunc that demultiplexes one message to all GroupEventHandlers registered for that CloudEvent type, recording OTel message-count and processing-time metrics per status.

## Patterns

**Type-keyed handler map** — Handlers are stored in typeHandlerMap[eventName] keyed by marshaler.Name(handler.NewEvent()). Handle() looks up by NameFromMessage(msg); unknown/empty groups are counted 'ignored' and return nil (message ack'd). (`groupHandler, ok := h.typeHandlerMap[eventName]; if !ok || len(groupHandler) == 0 { ...Ignored; return nil }`)
**Single unmarshal, shared event instance** — The event is unmarshaled once from groupHandler[0].NewEvent() then passed to every handler; all handlers for a type must accept the same concrete event struct. (`event := groupHandler[0].NewEvent(); h.marshaler.Unmarshal(msg, event)`)
**errors.Join over all handlers** — All handlers run via lo.Map and their errors are joined; any non-nil result fails the whole message (status 'failed', returned for retry/DLQ). Handlers are not short-circuited. (`err := errors.Join(lo.Map(groupHandler, func(h GroupEventHandler, _ int) error { return h.Handle(msg.Context(), event) })...)`)
**Metered constructor returns (*T, error)** — NewNoPublishingHandler calls getMeters(metricMeter) and propagates meter-creation errors; never construct NoPublishingHandler literal without meters. (`meters, err := getMeters(metricMeter); if err != nil { return nil, err }`)
**Concurrency-safe registration** — AddHandler takes mux.Lock and Handle takes mux.RLock, so handlers can be added while the router is running. (`h.mux.Lock(); defer h.mux.Unlock(); h.typeHandlerMap[...] = append(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grouphandler.go` | NoPublishingHandler (Handle/AddHandler), NewGroupEventHandler generic wrapper, NewNoPublishingHandler, meters + getMeters. | Handle returns nil for unregistered event types (ack'd, not DLQ'd). A returned error sends the message to retry/DLQ — keep handlers idempotent since all run on every retry. |

## Anti-Patterns

- Returning an error for an event you simply don't handle — let the type-map miss ack it as 'ignored' instead.
- Registering handlers for the same event type that expect different concrete event structs (only NewEvent() of the first is unmarshaled).
- Mutating typeHandlerMap without the mux, or constructing NoPublishingHandler without getMeters.

## Decisions

- **One Watermill handler fans out to many group handlers in-process.** — Avoids one Kafka consumer per handler; multiple subscribers to the same event type share a single subscription and message decode.

## Example: Register multiple handlers under one Watermill consumer

```
h, err := grouphandler.NewNoPublishingHandler(marshaler, metricMeter,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, e *snapshot.SnapshotEvent) error { return doA(ctx, e) }),
    grouphandler.NewGroupEventHandler(func(ctx context.Context, e *snapshot.SnapshotEvent) error { return doB(ctx, e) }),
)
if err != nil { return err }
router.AddNoPublisherHandler("name", topic, subscriber, h.Handle)
```

<!-- archie:ai-end -->
