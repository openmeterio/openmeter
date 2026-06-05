# watermill

<!-- archie:ai-start -->

> Structural folder owning OpenMeter's event-bus integration over the Watermill library. It is the seam between domain code and Kafka: children split by responsibility — eventbus (outbound publish facade), grouphandler (inbound fan-out dispatch), marshaler (CloudEvents codec), router (consumer middleware stack), driver (transport impls), and nopublisher (handler adapters).

## Patterns

**CloudEvents is the canonical wire format** — marshaler defines the Event interface (EventName/EventMetadata/Validate) that every published event implements; routing/identity travels in ce_* metadata headers, never in the JSON body. eventbus and router both key off ce_type. (`Marshal serializes domain events as JSON CloudEvents 1.0 with ce_type/ce_source headers mirrored into Watermill metadata.`)
**Topic routing by event-name prefix** — eventbus maps domain events to Kafka topics via a TopicMapping + a GeneratePublishTopic switch on the event-name prefix, rather than per-event topic config. (`New topic => extend TopicMapping (+Validate) and the GeneratePublishTopic switch together.`)
**Options/Config Validate() gates construction everywhere** — eventbus.New, router.NewDefaultRouter, and drivers all validate their Options/Config before building, and inject an explicit *slog.Logger rather than slog.Default(). (`Calling cqrs.NewEventBusWithConfig or message.Router directly bypasses this validation and the standard middleware/marshaler wiring.`)
**Fixed, ordered consumer middleware stack** — router.NewDefaultRouter assembles a fixed stack (DLQ/poison-queue, OTel metrics+tracing, correlation, recover, retry, timeout) with a MaxRetries-1 correction and Timeout wrapped by RestoreContext; every Kafka consumer is built through it. (`Reordering middleware breaks the DLQ/timeout/retry/context-restore interplay (issue 467).`)
**Transport selected above the driver folder** — driver/ holds interchangeable Watermill Publisher/Subscriber implementations (kafka real driver, noop null publisher); eventbus chooses between them. The disabled path is a real noop driver, not a nil publisher or per-call-site guards. (`Publishing disabled => wire driver/noop, not nil-check at each publish site.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus/eventbus.go` | Publisher/ContextPublisher facade over Watermill CQRS EventBus; topic routing and nil-event no-op publish. | Adding a topic requires both TopicMapping.Validate and the GeneratePublishTopic switch. |
| `grouphandler/grouphandler.go` | Type-keyed fan-out from one Watermill handler to many GroupEventHandlers, errors.Join over results, OTel metrics per status. | Only NewEvent() of the first registered handler is unmarshaled per type; don't register conflicting concrete structs for one event type, and ack (don't error) unhandled types. |
| `marshaler/marshaler.go` | CloudEvents<->Watermill codec, Event interface contract, ID/Time defaulting, WithSource decorator. | Non-Event structs fail Marshal ('invalid event type'); empty Source is rejected by NewCloudEvent. |
| `router/router.go` | NewDefaultRouter assembling the standard middleware stack consumed by balance/billing/notification workers. | DLQ push is outermost but skipped when IsClosed(); preserve MaxRetries-1 and Timeout+RestoreContext pairing. |

## Anti-Patterns

- Bypassing eventbus.New / router.NewDefaultRouter and constructing cqrs.NewEventBusWithConfig or message.Router directly, losing Options validation, marshaler wiring, DLQ/metrics/retry.
- Adding a Kafka topic without extending both TopicMapping (+Validate) and GeneratePublishTopic.
- Publishing a struct that does not implement the marshaler Event interface, or relying on the JSON body instead of ce_type metadata for routing.
- Reordering/removing router middleware, dropping RestoreContext around Timeout, or forgetting the MaxRetries-1 correction.
- Using NewMock or slog.Default() in production wiring instead of injecting a real *slog.Logger.

## Decisions

- **All inter-service events use CloudEvents 1.0 with metadata mirrored into Watermill headers.** — ce_* headers give every consumer/router a uniform, body-independent way to route and validate events.
- **A single NewDefaultRouter encodes the project's whole consumer middleware policy.** — Centralizing DLQ, retry, timeout, tracing and correlation guarantees every Kafka worker behaves identically; escape hatch is building your own router.
- **Drivers are split into kafka and noop sibling packages, with the disabled path as a real noop driver.** — Selection happens once in eventbus instead of nil-checks scattered across publish call sites.

<!-- archie:ai-end -->
