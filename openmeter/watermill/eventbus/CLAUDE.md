# eventbus

<!-- archie:ai-start -->

> Outbound event-publishing facade over Watermill's CQRS EventBus. Maps domain events to Kafka topics by event-name prefix and provides the Publisher/ContextPublisher interfaces the whole codebase uses to emit CloudEvents.

## Patterns

**Topic routing by event-name prefix** — GeneratePublishTopic switches on strings.HasPrefix(EventName, <subsystem>+".") to pick IngestEventsTopic, BalanceWorkerEventsTopic, else SystemEventsTopic. New topic-routed subsystems add a prefix case here. (`case strings.HasPrefix(params.EventName, ingestVersionSubsystemPrefix): return opts.TopicMapping.IngestEventsTopic, nil`)
**Options/TopicMapping Validate() before construction** — New(Options) returns (Publisher, error) and calls opts.Validate() first; TopicMapping.Validate() requires all three topics non-empty. Never construct the bus bypassing Validate. (`func New(opts Options) (Publisher, error) { if err := opts.Validate(); err != nil { return nil, err } ... }`)
**nil event is a no-op publish** — Publisher.Publish returns nil immediately when event==nil, so handlers can signal 'nothing to publish' by returning a nil event without the caller branching. (`if event == nil { return nil }`)
**WithContext().PublishIfNoError chaining** — ContextPublisher inlines publish-on-success: PublishIfNoError(event, err) returns err if non-nil else publishes. Used to fold a handler's (event,err) result into one return. (`return p.WithContext(ctx).PublishIfNoError(worker.handleEvent(ctx, event))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `eventbus.go` | Entire package: TopicMapping, Options, Publisher/ContextPublisher interfaces, publisher impl, New, NewMock. | NewMock uses slog.Default() — acceptable in test-only constructor; production New requires an explicit Logger in Options (Validate rejects nil). |

## Anti-Patterns

- Adding a new Kafka topic without extending both TopicMapping (+Validate) and the GeneratePublishTopic switch.
- Calling cqrs.NewEventBusWithConfig directly instead of going through New, bypassing Options validation and marshaler wiring.
- Using NewMock or slog.Default() in production wiring — inject a real *slog.Logger via Options.

## Decisions

- **Route by event-name prefix rather than per-event topic config.** — Subsystems (ingest, balanceworker) own a topic; everything else falls through to SystemEventsTopic, keeping topic config to three fields.

<!-- archie:ai-end -->
