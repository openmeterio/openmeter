# eventbus

<!-- archie:ai-start -->

> Wraps Watermill's cqrs.EventBus to route typed domain events to one of three Kafka topics (ingest, system, balance-worker) by event-name prefix matching against EventVersionSubsystem constants. The Publisher interface is the single injection point for all producers; topic routing is fully encapsulated.

## Patterns

**Topic routing by event-name prefix** — GeneratePublishTopic checks strings.HasPrefix against ingestevents.EventVersionSubsystem+'.' and balanceworkerevents.EventVersionSubsystem+'.'; everything else falls to SystemEventsTopic with no error. New event families must declare a matching EventVersionSubsystem constant. (`ingestVersionSubsystemPrefix := ingestevents.EventVersionSubsystem + "."`)
**PublishIfNoError for handler-inline publishing** — Use p.WithContext(ctx).PublishIfNoError(handler(ctx, event)) to avoid a separate error check when a handler returns (marshaler.Event, error). Nil event signals no-publish. (`return w.publisher.WithContext(ctx).PublishIfNoError(w.handleEvent(ctx, ev))`)
**Options.Validate before construction** — Options and TopicMapping have Validate() methods called by New(); all three topic names, a non-nil Publisher and Logger are required. (`if err := opts.Validate(); err != nil { return nil, err }`)
**NewMock for unit tests** — Use eventbus.NewMock(t) — it wires a noop.Publisher with fixed topic names. Never instantiate a real Kafka-backed publisher in unit tests. (`eventBus := eventbus.NewMock(t)`)
**Nil-event guard suppresses publish** — publisher.Publish silently ignores nil events; handlers that conditionally skip publishing return nil rather than an empty struct. (`if event == nil { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus.go` | Entire package: TopicMapping, Options, Publisher/ContextPublisher interfaces, publisher struct, New(), NewMock(). | Adding a 4th topic requires updating TopicMapping, Validate(), and the GeneratePublishTopic switch; the new EventVersionSubsystem prefix must not overlap existing ones or routing silently misroutes. |

## Anti-Patterns

- Publishing directly to a Kafka topic string instead of via eventbus.Publisher.
- Defining event names without an EventVersionSubsystem constant — defaults to SystemEventsTopic silently.
- Passing a nil Publisher in Options — New() errors; use noop.Publisher for tests.
- Checking event == nil at call sites — the publisher already guards against nil events.
- Creating a second Publisher instance outside app/common Wire wiring — TopicMapping must be consistent across producers.

## Decisions

- **Three fixed topics, prefix-routed by EventVersionSubsystem.** — Topic isolation matches worker topology (sink-worker, balance-worker, everything else); the event name carries the routing key, decoupling producers from consumer topology.
- **Publisher interface instead of exposing cqrs.EventBus directly.** — Hides Watermill internals, allows noop substitution in tests/DI, and provides WithContext/PublishIfNoError to avoid error-check boilerplate.

## Example: Publishing a domain event using PublishIfNoError

```
import "github.com/openmeterio/openmeter/openmeter/watermill/eventbus"

func (w *worker) handleSubscriptionEvent(ctx context.Context, ev SubscriptionEvent) (marshaler.Event, error) {
	return billingevents.NewInvoiceCreated(inv.ID), nil
}
// In worker loop:
return w.publisher.WithContext(ctx).PublishIfNoError(w.handleSubscriptionEvent(ctx, ev))
```

<!-- archie:ai-end -->
