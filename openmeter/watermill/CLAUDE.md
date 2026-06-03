# watermill

<!-- archie:ai-start -->

> Kafka-backed pub-sub abstraction for all async domain events between OpenMeter binaries. Owns three named topics (ingest, system, balance-worker), event-name-prefix topic routing, the CloudEvents 1.0 wire format, the standard router middleware stack, and kafka/noop driver selection. Every producer and consumer depends on its sub-packages — never on raw Watermill or confluent-kafka-go.

## Patterns

**Publish only via eventbus.Publisher — never a topic string** — eventbus/ wraps cqrs.EventBus; GeneratePublishTopic routes by EventName() prefix matched against EventVersionSubsystem constants (ingest -> IngestEventsTopic, balance-worker -> BalanceWorkerEventsTopic, default -> SystemEventsTopic). The Publisher interface is the single injection point; a second instance outside app/common Wire risks an inconsistent TopicMapping. (`publisher.Publish(ctx, &billingevents.InvoiceCreated{...}) // routed by event-name prefix`)
**Workers build routers only via router.NewDefaultRouter** — router/ wires a fixed ordered middleware stack: PoisonQueue (outermost), DLQ telemetry+OTel, CorrelationID, Recoverer, exponential-backoff Retry, optional ProcessingTimeout preceded by RestoreContext (watermill#467 workaround), HandlerMetrics (innermost). Never construct *message.Router directly. (`r, err := router.NewDefaultRouter(router.Options{Tracer: t, MetricMeter: m, ...})`)
**Unknown ce_types silently dropped in NoPublishingHandler** — grouphandler/ dispatches one Kafka message to all GroupEventHandlers matching the CloudEvents ce_type header, with errors.Join fan-out and per-type OTel metrics. Unregistered types are ACKed (dropped), never errored — returning an error retries and DLQ-poisons valid sibling families during rolling deploys. (`noPub := grouphandler.NewNoPublishingHandler(marshaler, grouphandler.NewGroupEventHandler(func(ctx, ev *billingevents.InvoiceCreated) error { return svc.OnInvoiceCreated(ctx, ev) }))`)
**CloudEvents 1.0 events implement marshaler.Event** — marshaler/ serializes each event to a CloudEvents JSON envelope (ce_type/ce_time/ce_source/ce_subject). EventName() must be stable (drives routing and dispatch); Marshal sets ce_type from it. ULID auto-ID when empty, time.Now() when zero. Validate() runs on both producer and consumer sides; WithSource injects source without mutating the struct. (`func (e *MyEvent) EventName() string { return myEventName } // stable, prefix-matched`)
**MaxRetries off-by-one: 0 = one attempt then DLQ** — router.Options.MaxRetries=0 means zero retries (one attempt, then DLQ); use 1 for one attempt plus one retry. Never set 0 expecting 'no DLQ'. (`// MaxRetries=0 -> 1 attempt -> DLQ; MaxRetries=1 -> 1 attempt + 1 retry -> DLQ`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus/eventbus.go` | Publisher interface + EventBus; topic routing by event-name prefix; NewMock for tests. | Event names without a registered subsystem prefix silently route to SystemEventsTopic; nil Publisher in Options errors in New(). |
| `grouphandler/grouphandler.go` | NoPublishingHandler ce_type multiplexer; AddHandler is RWMutex-guarded and safe post-construction. | Never error on unknown event types; don't mutate a shared deserialized event pointer when handlers share a type. |
| `marshaler/marshaler.go` | CloudEvents 1.0 serializer/deserializer; calls Validate() on deserialize; WithSource wrapper. | Too-strict Validate() rejects valid older-producer events during rolling deploys; never set CloudEventsHeaderType manually. |
| `router/router.go` | NewDefaultRouter fixed middleware order + Options.Validate. | Add Timeout only with RestoreContext immediately before it; missing Tracer/MetricMeter nil-panics HandlerMetrics. |
| `driver/` | Organizational parent for kafka/ (IBM Sarama, prod) and noop/ (disabled-feature Wire paths). Selection in app/common only. | No .go files belong directly under driver/; a new driver needs a noop counterpart. |
| `nopublisher/nopublisher.go` | Adapters between NoPublishHandlerFunc and HandlerFunc signatures. | HandlerFuncToNoPublisherHandler fails with ErrMessagesProduced if the adapted handler ever produces — use only for pure consumers. |

## Anti-Patterns

- Constructing a *message.Router directly instead of router.NewDefaultRouter — loses DLQ, OTel, and Retry middleware.
- Defining an EventName() without a stable EventVersionSubsystem prefix — silently routes to SystemEventsTopic.
- Adding source files directly under openmeter/watermill/driver/ — code belongs in kafka/ or noop/.
- Setting MaxRetries=0 expecting no DLQ — that is one attempt then DLQ.
- Publishing to a raw Kafka topic string from domain code — bypasses eventbus routing and consumer-group isolation.

## Decisions

- **Three fixed topics with prefix-based routing inside eventbus.** — Topic isolation matches the ingest/balance/system worker topology; centralising routing stops producers writing to the wrong consumer group.
- **CloudEvents 1.0 as the universal wire format.** — Standard ce_type-header routing for grouphandler and interoperability with Svix webhook payloads.
- **Unknown event types are dropped, not errored, in NoPublishingHandler.** — Producers and consumers deploy independently — a consumer can ignore types it has not yet registered without poisoning its DLQ.

## Example: Registering a new event handler in a worker's NoPublishingHandler

```
import (
    "github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
    billingevents "github.com/openmeterio/openmeter/openmeter/billing/events"
)

noPub := grouphandler.NewNoPublishingHandler(
    marshaler,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error {
        return svc.OnInvoiceCreated(ctx, ev) // use msg.Context(), never context.Background()
    }),
    // additional handlers; unknown ce_types are silently dropped
)
router.AddNoPublisherHandler("my-handler", topics.System, subscriber, noPub)
```

<!-- archie:ai-end -->
