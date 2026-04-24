# kafka

<!-- archie:ai-start -->

> Provides NewPublisher and NewSubscriber factories over IBM Sarama + watermill-kafka, with SASL/SSL auth, OTel tracing, partition-key routing, and OTel metrics bridging. All Sarama internals are hidden behind these two constructors; callers receive a watermill.Publisher or message.Subscriber.

## Patterns

**BrokerOptions as shared config root** — Both PublisherOptions and SubscriberOptions embed BrokerOptions. Call Validate() before any construction — it checks ClientID, Logger, and MetricMeter are non-nil. (`opts.Broker.Validate() is called at the top of both NewPublisher and NewSubscriber.`)
**Role suffix on ClientID** — createKafkaConfig(role) appends the role string ('publisher' or 'subscriber') to ClientID so Kafka broker logs distinguish producers from consumers. (`config.ClientID = fmt.Sprintf("%s-%s", o.ClientID, role)`)
**OTel tracing via kafka.NewOTELSaramaTracer()** — PublisherConfig always sets Tracer: kafka.NewOTELSaramaTracer(). Relies on the global trace provider; do not pass a custom tracer. (`wmConfig.Tracer = kafka.NewOTELSaramaTracer()`)
**Partition-key routing via marshalerWithPartitionKey** — Publisher always uses marshalerWithPartitionKey{} (wraps DefaultMarshaler). Set msg.Metadata[PartitionKeyMetadataKey] = subject to route to a partition. Subscriber uses DefaultMarshaler. (`wmConfig.Marshaler = marshalerWithPartitionKey{}`)
**SaramaMetricRenamer drops per-broker/topic cardinality** — Metrics containing 'for-broker' or 'for-topic' substrings are dropped; others are prefixed with 'sarama.' and tagged with {role}. New metrics must fit this filter or add an explicit Drop rule. (`SaramaMetricRenamer(role) is passed to metrics.NewRegistry as NameTransformFn.`)
**Topic provisioning before publisher creation** — TopicProvisioner.Provision(ctx, topics...) is called inside NewPublisher before kafka.NewPublisher. Topics must exist before the publisher is returned. (`if err = in.TopicProvisioner.Provision(ctx, in.ProvisionTopics...); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | Defines BrokerOptions and createKafkaConfig — the shared sarama.Config builder. All SASL/TLS, metric registry, and ClientID logic lives here. | MetricMeter must not be nil — metrics.NewRegistry returns an error if it is. createKafkaConfig always sets config.ApiVersionsRequest = false to suppress log spam. |
| `publisher.go` | NewPublisher: validates options, calls createKafkaConfig("publisher"), provisions topics, then constructs kafka.Publisher. | Always use marshalerWithPartitionKey{} — using kafka.DefaultMarshaler{} silently breaks partition routing. |
| `subscriber.go` | NewSubscriber: validates options, calls createKafkaConfig("subscriber"), sets defaultMaxProcessingTime (5 min), constructs kafka.Subscriber. | Subscriber uses kafka.DefaultMarshaler{} (not marshalerWithPartitionKey) — that is intentional. |
| `marshaler.go` | marshalerWithPartitionKey wraps DefaultMarshaler to promote PartitionKeyMetadataKey from message metadata to sarama.ProducerMessage.Key. AddPartitionKeyFromSubject is the helper to set the metadata. | The metadata key is stripped from Kafka headers after promotion to avoid double-encoding. |
| `metrics.go` | SaramaMetricRenamer returns a TransformMetricsNameToOtel function that drops per-broker/topic metrics and prefixes survivors with 'sarama.'. | Per-broker and per-topic metric names are dropped unconditionally — do not rely on them for observability. |
| `saslscram.go` | XDGSCRAMClient implements sarama.SCRAMClient for SCRAM-SHA-256/512 SASL mechanisms. | Only used when SecurityProtocol == SASL_SSL and SaslMechanisms is SCRAM-SHA-256 or SCRAM-SHA-512. |

## Anti-Patterns

- Building *sarama.Config directly instead of calling createKafkaConfig — SASL/TLS, metric registry, and ClientID suffix are all missing
- Using kafka.DefaultMarshaler{} in PublisherConfig — partition-key routing silently stops working
- Skipping TopicProvisioner.Provision before publishing — topics may not exist in the Kafka cluster
- Setting a nil MetricMeter in BrokerOptions — metrics.NewRegistry returns an error and construction fails
- Adding per-broker or per-topic metric labels without a Drop rule in SaramaMetricRenamer — creates unbounded OTel label cardinality

## Decisions

- **marshalerWithPartitionKey wraps DefaultMarshaler rather than replacing it** — Reuses all existing serialisation logic; only adds the key-promotion step, keeping the delta minimal and auditable.
- **OTel metrics registry injected into sarama.Config.MetricRegistry** — Bridges Sarama's go-metrics interface to OTel without patching Sarama; SaramaMetricRenamer drops high-cardinality names before they reach the exporter.
- **Topic provisioning is a Publisher-side responsibility, not caller responsibility** — Callers pass ProvisionTopics in PublisherOptions; NewPublisher ensures topics exist before the publisher is usable, preventing publish-time errors.

## Example: Construct a Publisher with SASL_SSL and OTel metrics

```
import (
    "context"
    kafkadriver "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
    pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

pub, err := kafkadriver.NewPublisher(ctx, kafkadriver.PublisherOptions{
    Broker: kafkadriver.BrokerOptions{
        KafkaConfig: cfg.Kafka,
        ClientID:    "billing-worker",
        Logger:      logger,
        MetricMeter: meter,
    },
    ProvisionTopics:  []pkgkafka.TopicConfig{{Name: "system-events", ...}},
    TopicProvisioner: provisioner,
// ...
```

<!-- archie:ai-end -->
