# kafka

<!-- archie:ai-start -->

> Low-level Kafka driver providing NewPublisher and NewSubscriber factories over IBM Sarama + watermill-kafka, with SASL/SSL auth, OTel tracing, partition-key routing, and OTel-bridged metrics. All Sarama internals are hidden behind these two constructors; the metrics/ child bridges Sarama's go-metrics to an OTel meter.

## Patterns

**BrokerOptions as shared config root** — Both PublisherOptions and SubscriberOptions embed BrokerOptions; call Validate() before any construction — it requires non-nil KafkaConfig, ClientID, Logger, and MetricMeter. (`if err := o.Broker.Validate(); err != nil { return nil, err }`)
**Role suffix on ClientID** — createKafkaConfig(role) appends 'publisher' or 'subscriber' to ClientID so broker logs distinguish producers from consumers. (`config.ClientID = fmt.Sprintf("%s-%s", o.ClientID, role)`)
**OTel tracing via the global provider** — PublisherConfig always sets Tracer: kafka.NewOTELSaramaTracer(), relying on the global trace provider — do not pass a custom tracer. (`wmConfig.Tracer = kafka.NewOTELSaramaTracer()`)
**Partition-key routing via marshalerWithPartitionKey** — The Publisher always uses marshalerWithPartitionKey{} (wraps DefaultMarshaler); set msg.Metadata[PartitionKeyMetadataKey]=subject (helper AddPartitionKeyFromSubject) to route. The Subscriber intentionally uses DefaultMarshaler. (`wmConfig.Marshaler = marshalerWithPartitionKey{}`)
**SaramaMetricRenamer drops high-cardinality metrics** — Metrics containing 'for-broker' or 'for-topic' are dropped; survivors are prefixed 'sarama.' and tagged with {role}. New metrics must fit this filter or add an explicit Drop rule. (`SaramaMetricRenamer(role) // passed to metrics.NewRegistry as NameTransformFn`)
**Topic provisioning is Publisher-side** — TopicProvisioner.Provision(ctx, ProvisionTopics...) runs inside NewPublisher before kafka.NewPublisher, so topics exist before the publisher is returned. (`if err = in.TopicProvisioner.Provision(ctx, in.ProvisionTopics...); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | BrokerOptions + createKafkaConfig — the shared sarama.Config builder owning all SASL/TLS, metric-registry, and ClientID logic. | MetricMeter must be non-nil (metrics.NewRegistry errors otherwise); createKafkaConfig always sets ApiVersionsRequest=false to suppress log spam. |
| `publisher.go` | NewPublisher: validates, createKafkaConfig("publisher"), provisions topics, constructs kafka.Publisher. | Must use marshalerWithPartitionKey{} — DefaultMarshaler silently breaks partition routing. |
| `subscriber.go` | NewSubscriber: validates, createKafkaConfig("subscriber"), sets defaultMaxProcessingTime (5 min), constructs kafka.Subscriber. | Uses kafka.DefaultMarshaler{} intentionally (no partition key); ConsumerGroupName must not be empty. |
| `marshaler.go` | marshalerWithPartitionKey promotes PartitionKeyMetadataKey from message metadata to sarama.ProducerMessage.Key; AddPartitionKeyFromSubject sets the metadata. | The metadata key is stripped from Kafka headers after promotion to avoid double-encoding. |
| `metrics.go` | SaramaMetricRenamer returns the OTel name-transform that drops per-broker/topic metrics and prefixes survivors with 'sarama.'. | Per-broker/per-topic metric names are dropped unconditionally — do not rely on them for observability. |
| `saslscram.go` | XDGSCRAMClient implementing sarama.SCRAMClient for SCRAM-SHA-256/512. | Only used when SecurityProtocol==SASL_SSL and SaslMechanisms is SCRAM-SHA-256/512. |
| `logger.go` | SaramaLoggerAdaptor bridging sarama.Logger/DebugLogger to slog, set inside createKafkaConfig. | These are global sarama variables — setting them affects every sarama client in the process. |

## Anti-Patterns

- Building *sarama.Config directly instead of calling createKafkaConfig — loses SASL/TLS, metric registry, and ClientID suffix.
- Using kafka.DefaultMarshaler{} in PublisherConfig — partition-key routing silently stops working.
- Skipping TopicProvisioner.Provision before publishing — topics may not exist in the cluster.
- Setting a nil MetricMeter in BrokerOptions — metrics.NewRegistry errors and construction fails.
- Adding per-broker/per-topic metric labels without a Drop rule in SaramaMetricRenamer — unbounded OTel label cardinality.

## Decisions

- **marshalerWithPartitionKey wraps DefaultMarshaler rather than replacing it.** — Reuses all existing serialisation logic and only adds the key-promotion step, keeping the delta minimal and auditable.
- **The OTel metrics registry is injected into sarama.Config.MetricRegistry via the metrics/ child.** — Bridges Sarama's go-metrics interface to OTel without patching Sarama; SaramaMetricRenamer drops high-cardinality names before export.
- **Topic provisioning is a Publisher-side responsibility.** — Callers pass ProvisionTopics; NewPublisher ensures topics exist before the publisher is usable, preventing publish-time errors.

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
    ProvisionTopics:  []pkgkafka.TopicConfig{{Name: "system-events"}},
    TopicProvisioner: provisioner,
// ...
```

<!-- archie:ai-end -->
