# kafka

<!-- archie:ai-start -->

> Kafka transport driver for Watermill, building Sarama-backed publishers and subscribers from app config. Wraps watermill-kafka with OpenMeter-specific concerns: CloudEvent partition keys, SASL/SCRAM auth, Sarama logging redirection, topic provisioning, and OTel metrics via the metrics/ subpackage.

## Patterns

**Options struct with Validate() then constructor** — Each public constructor (NewPublisher, NewSubscriber) takes an Options struct embedding BrokerOptions and calls in.Validate() first thing, returning an error before building any Sarama/Watermill object. (`func NewSubscriber(in SubscriberOptions) (message.Subscriber, error) { if err := in.Validate(); err != nil { return nil, err } ... }`)
**Centralized Sarama config via createKafkaConfig(role)** — All Sarama config (SASL, TLS, metric registry, producer retries, logger) is built in BrokerOptions.createKafkaConfig(role) and handed to watermill via OverwriteSaramaConfig. role is 'publisher' or 'subscriber' and is required. (`saramaConfig, err := in.Broker.createKafkaConfig("publisher")`)
**Injected dependencies, no global fallbacks** — BrokerOptions requires a non-nil *slog.Logger and otelmetric.Meter; Validate() rejects nil. Logger is passed to watermill via watermill.NewSlogLogger and to the metrics registry via metrics.LoggingErrorHandler(o.Logger). (`if o.Logger == nil { return errors.New("logger is required") }`)
**CloudEvent subject becomes Kafka partition key** — AddPartitionKeyFromSubject copies cloudEvent.Subject() into message metadata under PartitionKeyMetadataKey; marshalerWithPartitionKey then promotes that metadata to kafkaMsg.Key and strips the header. Publisher uses marshalerWithPartitionKey{} as Marshaler. (`watermillIn.Metadata[PartitionKeyMetadataKey] = cloudEvent.Subject()`)
**Sarama global logger redirected to slog adaptor** — createKafkaConfig sets sarama.Logger and sarama.DebugLogger to SaramaLoggerAdaptor wrapping o.Logger.Info / .Debug. DebugLogger must always be set or debug logs fall back onto the info Logger. (`sarama.DebugLogger = &SaramaLoggerAdaptor{loggerFunc: logger.Debug}`)
**Metric name transform via SaramaMetricRenamer(role)** — Metrics are prefixed 'sarama.', tagged with a role attribute, and high-cardinality/low-value series (for-broker, for-topic, protocol-requests-rate, compression-) are dropped via TransformedMetric.Drop=true. (`ErrorHandler: metrics.LoggingErrorHandler(o.Logger), NameTransformFn: SaramaMetricRenamer(role)`)
**Topic provisioning before publisher returns** — NewPublisher calls in.TopicProvisioner.Provision(ctx, in.ProvisionTopics...) before constructing the kafka.Publisher, so topics exist before the first publish. (`if err = in.TopicProvisioner.Provision(ctx, in.ProvisionTopics...); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | BrokerOptions + createKafkaConfig(role): the single place all Sarama config is assembled (SASL/SCRAM, TLS, metric registry, producer settings, loggers). | ApiVersionsRequest is deliberately false (log-flood workaround); TLS Config left as &tls.Config{} because Sarama breaks on forced TLS 1.3; producer MaxMessageBytes is 2MB. Don't reintroduce these as defaults blindly. |
| `publisher.go` | NewPublisher: builds watermill kafka.Publisher with marshalerWithPartitionKey and the OTel Sarama tracer; provisions topics. | kafka.NewOTELSaramaTracer() relies on the GLOBAL trace provider, not an injected one. |
| `subscriber.go` | NewSubscriber: builds watermill kafka.Subscriber bound to a required ConsumerGroupName. | Consumer.MaxProcessingTime is hardcoded to 5m (defaultMaxProcessingTime) to avoid partition reassignment; uses DefaultMarshaler (no partition-key handling on the consume side). |
| `marshaler.go` | marshalerWithPartitionKey + AddPartitionKeyFromSubject: maps a CloudEvent subject to the Kafka message key. | PartitionKeyMetadataKey header is filtered out of the outgoing message after being promoted to Key; changing the key derivation changes partition assignment / ordering guarantees. |
| `metrics.go` | SaramaMetricRenamer(role): returns the name-transform fn consumed by the metrics subpackage registry. | ingorePrefixes drops whole metric families; for-broker/for-topic are dropped to avoid high cardinality. New metrics must avoid matching these drop rules. |
| `saslscram.go` | XDGSCRAMClient (Begin/Step/Done) + SHA256/SHA512 hash generators, vendored from the Sarama example for SCRAM auth. | Wired only when SecurityProtocol == 'SASL_SSL' and SaslMechanisms is SCRAM-SHA-256/512 in createKafkaConfig. |
| `logger.go` | SaramaLoggerAdaptor: bridges Sarama's Print/Printf/Println logger interface to a single LoggerFunc. | Thin fmt.Sprint/Sprintf bridge; level (Info vs Debug) is chosen by which slog method is bound at construction in broker.go. |

## Anti-Patterns

- Constructing Sarama config outside createKafkaConfig(role) — bypasses SASL/TLS/metrics/logger wiring and the role suffix on ClientID.
- Using slog.Default() instead of the injected BrokerOptions.Logger; Validate() exists specifically to forbid nil loggers/meters.
- Skipping in.Validate() in a new constructor before touching Sarama/Watermill objects.
- Setting only sarama.Logger and not sarama.DebugLogger — debug output then floods the info logger.
- Adding partition-key logic on the subscriber path; only the publisher uses marshalerWithPartitionKey, the subscriber uses DefaultMarshaler.

## Decisions

- **OverwriteSaramaConfig injects a fully custom Sarama config into watermill-kafka rather than relying on watermill defaults.** — OpenMeter needs control over SASL/SCRAM, TLS quirks, the OTel metric registry, producer message size, and Sarama's global loggers — none expressible through watermill's surface.
- **Partition key is derived from the CloudEvent subject and carried through message metadata.** — Guarantees per-subject (e.g. per-customer/meter) ordering and co-location across partitions for usage events.
- **ApiVersionsRequest disabled and TLS left at library defaults instead of forcing 1.3.** — Documented workarounds for known Sarama bugs (ApiVersionsRequest V3 log floods; TLS 1.3 'protocol version not supported').

## Example: Build a Kafka publisher that partitions by CloudEvent subject

```
import (
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	wmkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
)

pub, err := wmkafka.NewPublisher(ctx, wmkafka.PublisherOptions{
	Broker: wmkafka.BrokerOptions{
		KafkaConfig: cfg.Kafka,
		ClientID:    "openmeter",
		Logger:      logger,
		MetricMeter: meter,
	},
	ProvisionTopics:  []pkgkafka.TopicConfig{ /* ... */ },
	TopicProvisioner: provisioner,
})
// ...
```

<!-- archie:ai-end -->
