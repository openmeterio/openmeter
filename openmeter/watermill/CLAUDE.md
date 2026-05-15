# watermill

<!-- archie:ai-start -->

> Kafka-backed pub-sub abstraction layer for all async domain event passing between OpenMeter binaries. Owns three named topics (ingest, system, balance-worker), topic routing by event-name prefix, CloudEvents 1.0 wire format, standard Watermill router middleware stack, and driver selection (kafka/noop). All domain event producers and consumers depend on sub-packages here — never on raw Watermill or confluent-kafka-go primitives.

## Patterns

**Always publish via eventbus.Publisher — never to a Kafka topic string** — Topic routing (ingest vs system vs balance-worker) is determined by event-name prefix inside eventbus.GeneratePublishTopic. Publishing directly to a Kafka topic bypasses this routing and breaks consumer group isolation. (`publisher.Publish(ctx, &billingevents.InvoiceCreated{...}) // routes automatically by event-name prefix`)
**All workers use router.NewDefaultRouter as their router factory** — NewDefaultRouter wires the fixed middleware stack (PoisonQueue, DLQ telemetry, CorrelationID, Recoverer, exponential-backoff Retry, ProcessingTimeout+RestoreContext, HandlerMetrics). Never construct *message.Router directly. (`router, err := router.NewDefaultRouter(router.Options{Tracer: t, MetricMeter: m, ...})`)
**Unknown event types silently dropped via NoPublishingHandler** — grouphandler.NoPublishingHandler dispatches to registered GroupEventHandlers keyed by CloudEvents ce_type header. Unregistered event types are dropped (not errored) to support rolling deploys — never return errors for unknown event types. (`noPublishHandler := grouphandler.NewNoPublishingHandler(
    marshaler,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error {
        return svc.OnInvoiceCreated(ctx, ev)
    }),
)
router.AddNoPublisherHandler("handler", topics.System, subscriber, noPublishHandler)`)
**EventVersionSubsystem prefix determines topic routing** — eventbus.GeneratePublishTopic matches event name prefixes against subsystem constants. New events must have EventName() starting with a recognised subsystem prefix; otherwise they silently route to SystemEventsTopic. (`// ingestevents.EventVersionSubsystem prefix → IngestEventsTopic
// balanceworkerevents.EventVersionSubsystem prefix → BalanceWorkerEventsTopic
// any other prefix → SystemEventsTopic (default, not error)`)
**CloudEvents 1.0 wire format — Event interface with EventName, EventMetadata, Validate** — All events must implement marshaler.Event (EventName, EventMetadata, Validate). Marshal sets ce_type from EventName(); never set CloudEventsHeaderType manually in message metadata. Validate() is called on both producer and consumer sides. (`type MyEvent struct { Namespace string; ID string }
func (e *MyEvent) EventName() string { return myEventName } // stable, prefix-matched
func (e *MyEvent) EventMetadata() metadata.EventMetadata { ... }
func (e *MyEvent) Validate() error { return nil }`)
**MaxRetries off-by-one: 0 means one attempt then DLQ** — router.Options.MaxRetries=0 means zero retries (one attempt total, then DLQ). Use MaxRetries=1 for one attempt plus one retry. Never set MaxRetries=0 intending 'no retries with no DLQ' — the message will still go to DLQ on failure. (`// MaxRetries=0 → 1 attempt → DLQ on failure
// MaxRetries=1 → 1 attempt + 1 retry → DLQ on second failure`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus/eventbus.go` | Publisher interface and concrete EventBus with topic routing by event-name prefix. Single injection point for all event producers. | Event names without a registered subsystem prefix silently route to SystemEventsTopic — verify prefix mapping when adding new event families. |
| `grouphandler/grouphandler.go` | NoPublishingHandler multiplexer. Unknown event types are dropped. AddHandler is safe to call after construction (RWMutex-guarded). | Never return errors for unknown event types — causes infinite Watermill retries and DLQ poisoning. |
| `marshaler/marshaler.go` | CloudEvents 1.0 serializer/deserializer. Calls Validate() on deserialized events. ULID auto-ID if ID is empty; time.Now() if Time is zero. | Validate() implementations that are too strict can reject valid events from older producers during rolling deploys. |
| `router/router.go` | NewDefaultRouter: fixed middleware order. MaxRetries off-by-one: config value 0 = 1 attempt, 1 = 2 attempts. | Never add Timeout middleware without RestoreContext immediately before it (watermill#467 workaround) — cancelled contexts break retry logic. |
| `driver/` | Organizational parent for kafka/ (IBM Sarama-backed) and noop/ driver implementations. Wire selects driver via config flag. | No source files belong directly in driver/ — only in kafka/ or noop/ sub-packages. |
| `nopublisher/nopublisher.go` | Adapter utilities between Watermill's NoPublishHandlerFunc and HandlerFunc. | HandlerFuncToNoPublisherHandler panics with ErrMessagesProduced if the adapted handler returns messages — use only when the handler never produces. |

## Anti-Patterns

- Constructing a *message.Router directly instead of router.NewDefaultRouter — middleware stack (DLQ, OTel, Retry) will be missing.
- Defining event names without a stable EventVersionSubsystem prefix — routing silently falls through to SystemEventsTopic.
- Adding source files directly in openmeter/watermill/driver/ — all driver code must live in kafka/ or noop/ sub-packages.
- Setting MaxRetries=0 in router.Options intending 'no retries' — the subtraction logic results in 1 attempt then DLQ.
- Publishing directly to a raw Kafka topic string from domain code — bypasses eventbus routing and breaks consumer group isolation.

## Decisions

- **Three fixed Kafka topics with prefix-based routing inside eventbus rather than per-event topic configuration.** — Topic isolation matches the worker topology (ingest/balance/system); centralising routing in eventbus prevents producers from accidentally writing to the wrong consumer group.
- **CloudEvents 1.0 as the wire format for all Watermill messages.** — Provides a standard schema for ce_type header-based routing in grouphandler and interoperability with Svix webhook payloads.
- **Unknown event types are silently dropped in NoPublishingHandler rather than errored.** — Allows independent deployment of producers and consumers — a consumer can safely ignore event types it has not yet registered without poisoning its DLQ.

## Example: Registering a new event handler in a worker's NoPublishingHandler

```
import (
    "github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
    billingevents "github.com/openmeterio/openmeter/openmeter/billing/events"
)

noPublishHandler := grouphandler.NewNoPublishingHandler(
    marshaler,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error {
        return svc.OnInvoiceCreated(ctx, ev) // use msg.Context() — never context.Background()
    }),
    // add more handlers here; unknown types are silently dropped
)
router.AddNoPublisherHandler("my-handler", topics.System, subscriber, noPublishHandler)
```

<!-- archie:ai-end -->
