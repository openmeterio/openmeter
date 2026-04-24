# kafkaingest

<!-- archie:ai-start -->

> Implements the ingest.Collector interface by producing CloudEvents to per-namespace Kafka topics. Owns topic provisioning on first ingest, OTel tracing per event, and the producer event-loop goroutine (KafkaProducerGroup) for delivery reporting and metrics.

## Patterns

**Serializer injection** — Collector never encodes events directly; it delegates to serializer.Serializer for key and value bytes. SerializeKey encodes a dedupe.Item (namespace+source+id); SerializeValue encodes a CloudEventsKafkaPayload. Both must be called in order on every Ingest call. (`key, _ := s.Serializer.SerializeKey(topicName, namespace, ev); value, _ := s.Serializer.SerializeValue(topicName, ev)`)
**Topic resolved and provisioned on each Ingest call** — TopicResolver.Resolve maps namespace → topic name; TopicProvisioner.Provision ensures the topic exists before Produce is called. Both happen inside every Ingest call, not at construction time. (`topicName, err := s.TopicResolver.Resolve(ctx, namespace); _ = s.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{Name: topicName, Partitions: s.TopicPartitions})`)
**OTel span wraps full Ingest lifecycle** — Ingest opens a span at entry, records errors and sets status on all error paths via a deferred closure, and annotates checkpoints with span.AddEvent. Never return an error without the span capturing it. (`ctx, span := s.Tracer.Start(ctx, "openmeter.ingest.process.event", ...); defer func() { if err != nil { span.RecordError(err); span.SetStatus(otelcodes.Error, err.Error()) }; span.End() }()`)
**Kafka headers carry namespace + OTel context** — Every kafka.Message must include the 'namespace', 'specversion', 'ingested_at', and otelx.OTelSpanContextKey headers. The sink worker reads these headers to reconstruct context and attribute events. (`Headers: []kafka.Header{{Key: HeaderKeyNamespace, Value: []byte(namespace)}, {Key: otelx.OTelSpanContextKey, Value: spanCtx}}`)
**NamespaceHandler delegates to same Resolver+Provisioner** — namespace.Handler lifecycle methods (CreateNamespace/DeleteNamespace) reuse the same TopicResolver and TopicProvisioner as the Collector. DeletionEnabled guards DeProvision calls. (`h.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{Name: topicName, Partitions: h.Partitions})`)
**KafkaProducerGroup as oklog/run pair** — KafkaProducerGroup returns (execute, interrupt) for an oklog/run group. The execute loop drains producer.Events() handling *kafka.Message, *kafka.Stats, and kafka.Error. Never start the producer without running this group or delivery errors will go unreported. (`execute, interrupt := kafkaingest.KafkaProducerGroup(ctx, producer, logger, kafkaMetrics)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector.go` | Core Collector struct and Ingest/Close methods plus KafkaProducerGroup. All Kafka produce logic lives here. | Producer.Produce is fire-and-forget; delivery results come via KafkaProducerGroup event loop. Missing the group means silent delivery failures. |
| `namespace.go` | NamespaceHandler implements openmeter/namespace.Handler for Kafka ingest topics. Provisioned on CreateNamespace, deprovisioned only if DeletionEnabled=true. | TopicResolver nil-check is done at call time, not construction — this means a zero-value NamespaceHandler will panic at runtime, not at setup. |
| `serializer/json.go` | JSONSerializer: SerializeKey encodes dedupe.Item as JSON, SerializeValue encodes CloudEventsKafkaPayload. Wire format consumed by sink worker's FromKafkaPayloadToCloudEvents. | CloudEventsKafkaPayload.Time is unix int64 — timezone is intentionally dropped. Data is a JSON string, not raw bytes. |
| `serializer/serializer.go` | Serializer interface definition. GetKeySchemaId/GetValueSchemaId are schema-registry hooks; current JSON impl returns 0. | If adding a schema-registry-aware serializer, all four methods must return real schema IDs or the sink worker will mis-deserialize. |
| `topicresolver/namespacedtopic.go` | NamespacedTopicResolver maps namespace → topic via fmt.Sprintf(template, namespace). Default template produces 'om_%s_events'. | Constructor validates non-empty template but not %s count; a template with zero or two %s placeholders will silently produce wrong topic names. |
| `topicresolver/resolver.go` | Resolver interface. Requires context.Context for future async/DB-backed implementations. | Always propagate caller's ctx; never pass context.Background() to Resolve. |

## Anti-Patterns

- Calling Producer.Produce without running KafkaProducerGroup — delivery errors will be silently dropped
- Hard-coding topic names instead of calling TopicResolver.Resolve per namespace
- Adding new Kafka headers inside Ingest without also updating the sink worker's header parsing
- Constructing Collector or NamespaceHandler with a nil TopicResolver — nil check is deferred to call time and will panic in production
- Storing timezone-aware time in CloudEventsKafkaPayload.Time — the field is unix int64 and timezone is intentionally dropped by the serializer

## Decisions

- **Topic is provisioned on every Ingest call, not just at namespace creation** — Ensures idempotent topic existence even if the namespace handler was not called first, at the cost of a Kafka admin API call per event (provisioner is expected to cache or no-op on existing topics).
- **Key encodes dedupe.Item (namespace+source+id), not just event ID** — Enables consistent Kafka partitioning per logical event identity so the sink worker's deduplication hash matches the Kafka key without a separate lookup.
- **KafkaProducerGroup is a separate oklog/run pair, not a goroutine started inside NewCollector** — Lets the caller control lifecycle via the run group and avoids goroutine leaks when the collector is shut down — matches the oklog/run pattern used throughout all cmd/* binaries.

## Example: Wiring Collector in app/common and registering NamespaceHandler

```
import (
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

resolver := topicresolver.NewNamespacedTopicResolver("om_%s_events")
collector, err := kafkaingest.NewCollector(producer, serializer.NewJSONSerializer(), resolver, provisioner, 4, logger, tracer)
// register namespace handler so CreateNamespace provisions the topic:
namespaceManager.RegisterHandler(kafkaingest.NamespaceHandler{
	TopicResolver:    resolver,
	TopicProvisioner: provisioner,
	Partitions:       4,
	DeletionEnabled:  false,
// ...
```

<!-- archie:ai-end -->
