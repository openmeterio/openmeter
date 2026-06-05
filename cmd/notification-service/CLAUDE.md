# notification-service

<!-- archie:ai-start -->

> main.go entrypoint for the notification-service binary, which consumes notification system events from Kafka and delivers them. Unlike the mixin-driven workers, main() builds the run.Group and notification consumer by hand from a partial Wire Application.

## Patterns

**Hand-built run.Group** — main() constructs a watermillkafka.Subscriber, builds consumer.Options with router.Options, calls consumer.New, then assembles run.Group (telemetry server, consumer, SignalHandler) and group.Run(run.WithReverseShutdownOrder()) directly rather than via common.Runner. (`notificationConsumer, err := consumer.New(consumerOptions)`)
**Application provides building blocks not lifecycle** — Application exposes BrokerOptions, MessagePublisher, EventPublisher, Notification, Meter, Tracer, TelemetryServer for main() to wire the consumer manually; still embeds GlobalInitializer/Migrator. (`app.BrokerOptions, app.MessagePublisher, app.EventPublisher.Marshaler(), app.Notification`)
**Consumer wired to system events topic** — consumer.Options.SystemEventsTopic = conf.Events.SystemEvents.Topic; ConsumerGroupName from conf.Notification.Consumer.ConsumerGroupName. (`SystemEventsTopic: conf.Events.SystemEvents.Topic`)
**Shared config bootstrap + panic funnel** — Identical viper/pflag config load, conf.Validate(), defer log.PanicLogger(log.WithExit), and Migrate before running. (`defer log.PanicLogger(log.WithExit)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Bootstrap, build Kafka subscriber + notification consumer, run telemetry + consumer in a run.Group. | Subscriber/consumer are constructed in main, not Wire; group.Run uses run.WithReverseShutdownOrder() so add components in dependency order. |
| `wire.go` | Provider list producing the Application building blocks (notification, meter, streaming, eventbus, watermill kafka driver). | Application here is a provider container, not a Runner; lifecycle is owned by main.go. |
| `wire_gen.go` | Generated injector; DO NOT EDIT. | Regenerate via make generate after provider changes. |
| `version.go` | ldflags version metadata. | Identical to other binaries. |

## Anti-Patterns

- Editing wire_gen.go instead of wire.go
- Adding new run.Group members without preserving reverse-shutdown ordering
- Putting delivery/consumer logic in main.go instead of openmeter/notification/consumer
- Skipping app.Migrate(ctx) before starting the consumer

## Decisions

- **The consumer and run.Group are assembled in main.go rather than via common.Runner** — Notification delivery needs explicit control over the Kafka subscriber, marshaler, and shutdown ordering of telemetry vs consumer.

<!-- archie:ai-end -->
