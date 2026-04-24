# eventbus

<!-- archie:ai-start -->

> Wraps Watermill's cqrs.EventBus to route typed domain events to one of three Kafka topics (ingest, system, balance-worker) based on event-name prefixes. The Publisher interface is the single injection point all producers use; topic routing is fully encapsulated here.

## Patterns

**Topic routing by event-name prefix** — GeneratePublishTopic checks strings.HasPrefix against ingestevents.EventVersionSubsystem+'.' and balanceworkerevents.EventVersionSubsystem+'.'; everything else goes to SystemEventsTopic. New event families must define an EventVersionSubsystem constant and ensure the prefix matches exactly. (`ingestVersionSubsystemPrefix := ingestevents.EventVersionSubsystem + "."`)
**Publisher.WithContext for inline publish-after-handler** — Use p.WithContext(ctx).PublishIfNoError(handler(ctx, event)) to avoid a separate error check when publishing from a handler that returns (Event, error). Do not call Publish directly in that pattern. (`return p.WithContext(ctx).PublishIfNoError(worker.handleEvent(ctx, event))`)
**Nil-event guard** — Publish silently ignores nil events (handler signals no-publish). Handlers that conditionally produce an event should return nil rather than a no-op event struct. (`if event == nil { return nil }`)
**Options.Validate() before construction** — Both Options and TopicMapping have Validate() methods; New() calls opts.Validate() and returns an error if any field is missing. Always validate before construction. (`if err := opts.Validate(); err != nil { return nil, err }`)
**NewMock for tests** — Use NewMock(t) in tests; it wires a noop.Publisher with fixed topic names and asserts no construction error. Never instantiate a real Kafka publisher in unit tests. (`eventBus := eventbus.NewMock(t)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus.go` | Single file for the entire package: defines TopicMapping, Options, Publisher/ContextPublisher interfaces, publisher struct, New(), and NewMock(). | Adding a 4th topic requires updating TopicMapping, Validate(), and the GeneratePublishTopic switch; the prefix must not overlap with existing EventVersionSubsystem values. |

## Anti-Patterns

- Publishing directly to a Kafka topic string — always go through eventbus.Publisher so routing is centralised.
- Defining event names without an EventVersionSubsystem constant — the prefix match in GeneratePublishTopic will default to SystemEventsTopic silently.
- Passing a nil Publisher in Options — New() will error; use noop.Publisher for tests.
- Checking event == nil at call sites — the publisher already handles nil events as a no-op.
- Creating a second publisher outside app/common Wire wiring — topic mapping must be consistent across all producers.

## Decisions

- **Three fixed topics, prefix-routed by EventVersionSubsystem** — Topic isolation matches the worker topology (sink, balance-worker, everything else) without requiring callers to know which topic to use — the event name carries the routing key.
- **Publisher interface instead of exposing cqrs.EventBus directly** — Hides Watermill internals, allows noop substitution in tests and DI, and provides the WithContext/PublishIfNoError ergonomic shortcut.

## Example: Publishing a domain event from a service handler

```
import "github.com/openmeterio/openmeter/openmeter/watermill/eventbus"

func (w *worker) handleSubscriptionEvent(ctx context.Context, ev SubscriptionEvent) (marshaler.Event, error) {
    // ... business logic ...
    return billingevents.NewInvoiceCreated(inv.ID), nil
}

// In worker loop:
return w.publisher.WithContext(ctx).PublishIfNoError(w.handleSubscriptionEvent(ctx, ev))
```

<!-- archie:ai-end -->
