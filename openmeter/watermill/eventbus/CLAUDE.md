# eventbus

<!-- archie:ai-start -->

> Wraps Watermill's cqrs.EventBus to route typed domain events to one of three Kafka topics (ingest, system, balance-worker) based on event-name prefix matching against EventVersionSubsystem constants. The Publisher interface is the single injection point all producers use; topic routing is fully encapsulated and callers never reference topic names directly.

## Patterns

**Topic routing by event-name prefix** — GeneratePublishTopic checks strings.HasPrefix against ingestevents.EventVersionSubsystem+'.' and balanceworkerevents.EventVersionSubsystem+'.'; everything else falls to SystemEventsTopic with no error. New event families must declare an EventVersionSubsystem constant matching one of these prefixes exactly. (`ingestVersionSubsystemPrefix := ingestevents.EventVersionSubsystem + "."`)
**PublishIfNoError for handler-inline publishing** — Use p.WithContext(ctx).PublishIfNoError(handler(ctx, event)) to avoid a separate error check when a handler returns (marshaler.Event, error). Returns nil if the event is nil (handler signals no-publish). (`return w.publisher.WithContext(ctx).PublishIfNoError(w.handleEvent(ctx, ev))`)
**Options.Validate before construction** — Both Options and TopicMapping have Validate() methods called by New(). All three topic names and a non-nil Publisher and Logger are required. New() returns an error on validation failure; never skip validation. (`if err := opts.Validate(); err != nil { return nil, err }`)
**NewMock for unit tests** — Use eventbus.NewMock(t) in tests; it wires a noop.Publisher with fixed topic names. Never instantiate a real Kafka-backed publisher in unit tests. (`eventBus := eventbus.NewMock(t)`)
**Nil-event guard — return nil to suppress publish** — publisher.Publish silently ignores nil events. Handlers that conditionally want to skip publishing should return nil rather than an empty struct. (`if event == nil { return nil } // inside publisher.Publish`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus.go` | Entire package in one file: TopicMapping, Options, Publisher/ContextPublisher interfaces, publisher struct, New(), NewMock(). | Adding a 4th topic requires updating TopicMapping, Validate(), and the GeneratePublishTopic switch; the new EventVersionSubsystem prefix must not overlap with existing ones or routing silently misroutes. |

## Anti-Patterns

- Publishing directly to a Kafka topic string — always use eventbus.Publisher so routing is centralised.
- Defining event names without an EventVersionSubsystem constant — GeneratePublishTopic defaults to SystemEventsTopic silently for unrecognised prefixes.
- Passing a nil Publisher in Options — New() will return an error; use noop.Publisher for tests.
- Checking event == nil at call sites — the publisher already guards against nil events.
- Creating a second Publisher instance outside app/common Wire wiring — TopicMapping must be consistent across all producers in a binary.

## Decisions

- **Three fixed topics, prefix-routed by EventVersionSubsystem** — Topic isolation matches the worker topology (sink-worker, balance-worker, everything else) without requiring callers to know which topic to use — the event name carries the routing key, keeping producers decoupled from consumer topology.
- **Publisher interface instead of exposing cqrs.EventBus directly** — Hides Watermill internals, allows noop substitution in tests and DI, and provides the WithContext/PublishIfNoError ergonomic shortcut that avoids repetitive error-check boilerplate in handler loops.

## Example: Publishing a domain event from a service handler using PublishIfNoError

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
