# notification-service

<!-- archie:ai-start -->

> Binary entrypoint for the notification-service: wires domain services via Wire, then manually constructs the Watermill Kafka subscriber and notification consumer in main.go and runs them in an oklog/run group. Unlike other workers, the Watermill subscriber and consumer.New call are in main.go, not delegated to a common.Runner.

## Patterns

**Manual run group assembly in main.go** — notification-service builds its own run.Group in main.go (telemetry server + notificationConsumer.Run + signal handler) rather than using common.Runner. The Application struct exposes BrokerOptions, MessagePublisher, Meter, Tracer, TelemetryServer for assembly in main.go. (`group.Add(func() error { return notificationConsumer.Run(ctx) }, func(err error) { _ = notificationConsumer.Close() })`)
**Subscriber constructed in main.go, not in Wire** — watermillkafka.NewSubscriber is called in main.go using app.BrokerOptions and conf.Notification.Consumer.ConsumerGroupName — the subscriber is not part of the Wire graph. (`wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{ Broker: app.BrokerOptions, ConsumerGroupName: conf.Notification.Consumer.ConsumerGroupName })`)
**consumer.Options wires app fields to consumer** — consumer.Options{} aggregates SystemEventsTopic, Router (router.Options with subscriber/publisher/logger/meter/tracer), Marshaler, and Notification service from app fields. (`consumer.Options{ SystemEventsTopic: conf.Events.SystemEvents.Topic, Router: router.Options{Subscriber: wmSubscriber, Publisher: app.MessagePublisher, ...}, Notification: app.Notification }`)
**Application exposes raw primitives (BrokerOptions, Meter, Tracer) as fields** — Unlike other workers that embed common.Runner, notification-service Application exposes metric.Meter, trace.Tracer, message.Publisher directly so main.go can assemble consumer.Options. (`type Application struct { ... BrokerOptions watermillkafka.BrokerOptions; MessagePublisher message.Publisher; Meter metric.Meter; Tracer trace.Tracer; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Builds Kafka subscriber, constructs consumer.Options, instantiates consumer.New, assembles and runs the oklog run.Group. This is intentionally more manual than other workers. | consumer.Options.Marshaler must be app.EventPublisher.Marshaler() — mismatching marshalers causes message deserialization failures. |
| `wire.go` | Application struct exposes more raw primitives than other workers (BrokerOptions, MessagePublisher, Meter, Tracer) because main.go needs them for manual consumer construction. | common.NotificationServiceProvisionTopics must be listed to ensure the correct Kafka topic provisioning for notification-service. |
| `wire_gen.go` | Generated — DO NOT EDIT. Note: TelemetryServer is a field on Application (not on common.Runner) because notification-service manages it in its own run.Group. | StreamingConnector is wired even though notification-service doesn't query usage — it's a transitive dependency of the notification service chain. |

## Anti-Patterns

- Moving Kafka subscriber construction into Wire — it is intentionally in main.go so consumer group name can come from config at startup
- Using app.MessagePublisher as the eventbus.Publisher for marshaling — use app.EventPublisher.Marshaler() for event deserialization
- Adding business notification logic to main.go — it belongs in openmeter/notification/consumer
- Manually editing wire_gen.go

## Decisions

- **notification-service constructs its Watermill subscriber in main.go rather than Wire.** — The consumer group name is a runtime config value tied to deployment identity; keeping subscriber construction in main.go makes the config-to-subscriber mapping explicit and avoids parameterising Wire with runtime strings.

## Example: Assembling notification consumer in main.go after Wire initialization

```
wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
    Broker:            app.BrokerOptions,
    ConsumerGroupName: conf.Notification.Consumer.ConsumerGroupName,
})
consumerOptions := consumer.Options{
    SystemEventsTopic: conf.Events.SystemEvents.Topic,
    Router: router.Options{
        Subscriber:  wmSubscriber,
        Publisher:   app.MessagePublisher,
        Logger:      logger,
        MetricMeter: app.Meter,
        Tracer:      app.Tracer,
        Config:      conf.Notification.Consumer,
    },
    Marshaler:    app.EventPublisher.Marshaler(),
// ...
```

<!-- archie:ai-end -->
