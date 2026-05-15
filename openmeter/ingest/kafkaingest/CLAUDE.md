# kafkaingest

<!-- archie:ai-start -->

> Implements the ingest.Collector interface by producing CloudEvents to per-namespace Kafka topics using confluent-kafka-go. Owns topic resolution and provisioning on every Ingest call, OTel span lifecycle per event, Kafka header construction (namespace, specversion, ingested_at, OTel span context), and the KafkaProducerGroup oklog/run pair for async delivery reporting.

## Patterns

**Serializer injection for key and value encoding** — Collector never encodes events directly — it delegates to serializer.Serializer. SerializeKey encodes a dedupe.Item (namespace+source+id) as JSON for sink-worker deduplication; SerializeValue encodes a flat CloudEventsKafkaPayload. Both must be called in order inside every Ingest invocation. (`key, _ := s.Serializer.SerializeKey(topicName, namespace, ev)
value, _ := s.Serializer.SerializeValue(topicName, ev)`)
**Topic resolved and provisioned inside every Ingest call** — TopicResolver.Resolve maps namespace → topic name; TopicProvisioner.Provision ensures the topic exists before Produce is called. Both happen per call, not at construction time. The provisioner is expected to be idempotent (no-op on existing topics). (`topicName, err := s.TopicResolver.Resolve(ctx, namespace)
_ = s.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{Name: topicName, Partitions: s.TopicPartitions})`)
**OTel span wraps full Ingest lifecycle with deferred error recording** — Ingest opens a span at entry and captures the local err variable in a deferred closure that calls span.RecordError + span.SetStatus on any non-nil error. Never return an error path without this span capturing it. (`ctx, span := s.Tracer.Start(ctx, "openmeter.ingest.process.event", ...)
defer func() {
  if err != nil { span.RecordError(err); span.SetStatus(otelcodes.Error, err.Error()) }
  span.End()
}()`)
**Fixed Kafka header set per message** — Every kafka.Message must include four headers: 'namespace' ([]byte), 'specversion', 'ingested_at' (RFC3339Nano UTC via ToIngestedAt), and otelx.OTelSpanContextKey (serialized span context). The sink worker reads these headers to reconstruct context and attribute events to the correct tenant. (`Headers: []kafka.Header{
  {Key: HeaderKeyNamespace, Value: []byte(namespace)},
  {Key: "specversion", Value: []byte(ev.SpecVersion())},
  {Key: HeaderKeyIngestedAt, Value: []byte(ToIngestedAt(clock.Now()))},
  {Key: otelx.OTelSpanContextKey, Value: spanCtx},
}`)
**KafkaProducerGroup as oklog/run pair, not an internal goroutine** — KafkaProducerGroup returns (execute func() error, interrupt func(error)) for use in cmd/* oklog/run.Group. The execute loop drains producer.Events() handling *kafka.Message delivery reports, *kafka.Stats (metrics), and kafka.Error (broker errors). Must be started by the caller — never start the producer without running this group. (`execute, interrupt := kafkaingest.KafkaProducerGroup(ctx, producer, logger, kafkaMetrics)
runGroup.Add(execute, interrupt)`)
**NamespaceHandler reuses same Resolver and Provisioner as Collector** — NamespaceHandler.CreateNamespace calls Resolve then Provision; DeleteNamespace is a no-op unless DeletionEnabled=true. The TopicResolver nil check is done at call time, not construction, so a zero-value NamespaceHandler will panic in production. (`namespaceManager.RegisterHandler(kafkaingest.NamespaceHandler{
  TopicResolver: resolver, TopicProvisioner: provisioner,
  Partitions: 4, DeletionEnabled: false,
})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector.go` | Core Collector struct with Ingest (produce path) and Close methods, plus KafkaProducerGroup for async delivery reporting. All Kafka produce logic lives here. | Producer.Produce is fire-and-forget; delivery results arrive asynchronously via KafkaProducerGroup. Missing the group means silent delivery failures. The local 'err' variable is captured by the deferred span closure — do not shadow it with := inside the function body. |
| `namespace.go` | NamespaceHandler implements openmeter/namespace.Handler for Kafka ingest topics. Provisions on CreateNamespace; deprovisions only when DeletionEnabled=true. | TopicResolver nil-check is deferred to call time — a zero-value NamespaceHandler panics at runtime, not setup. Register this handler before initNamespace() in cmd/server/main.go or the default namespace misses provisioning. |
| `serializer/json.go` | JSONSerializer implements SerializeKey (dedupe.Item JSON → key bytes) and SerializeValue (CloudEventsKafkaPayload JSON → value bytes). Wire format consumed by sink worker's FromKafkaPayloadToCloudEvents. | CloudEventsKafkaPayload.Time is unix int64 — timezone is intentionally dropped. Data is stored as a JSON string inside the payload, not raw bytes. Do not change this without updating the sink worker's deserializer. |
| `serializer/serializer.go` | Serializer interface definition with GetKeySchemaId/GetValueSchemaId schema-registry hooks. Current JSON impl returns 0 for both schema IDs. | If adding a schema-registry-aware serializer, all four methods must return real schema IDs or the sink worker will mis-deserialize messages. |
| `topicresolver/namespacedtopic.go` | NamespacedTopicResolver maps namespace → Kafka topic name via fmt.Sprintf(template, namespace). Default template produces 'om_%s_events'. | Constructor validates non-empty template but not %s count. A template with zero or two %s placeholders silently produces wrong topic names. |
| `topicresolver/resolver.go` | Resolver interface. Accepts context.Context for future async/DB-backed implementations. | Always propagate caller's ctx to Resolve — never pass context.Background() in application code. |

## Anti-Patterns

- Calling Producer.Produce without running KafkaProducerGroup in an oklog/run.Group — delivery errors are silently dropped
- Hard-coding topic names instead of calling TopicResolver.Resolve per namespace
- Adding new Kafka headers inside Ingest without also updating the sink worker's header parsing
- Constructing Collector or NamespaceHandler with a nil TopicResolver — nil check is deferred to call time and will panic in production
- Storing timezone-aware time in CloudEventsKafkaPayload.Time — the field is unix int64 and timezone is intentionally stripped by the serializer

## Decisions

- **Topic is resolved and provisioned on every Ingest call, not only at namespace creation** — Ensures idempotent topic existence even when the NamespaceHandler was not called first (e.g., in self-hosted single-namespace deployments). The provisioner is expected to be idempotent (no-op on existing topics) so the per-call overhead is a single Kafka admin API check.
- **Key encodes dedupe.Item (namespace+source+id) rather than event ID alone** — Enables consistent Kafka partitioning per logical event identity so the sink worker's Redis deduplication hash matches the Kafka message key without a separate lookup, preventing double-counting on consumer restart.
- **KafkaProducerGroup is an oklog/run pair returned to the caller rather than a goroutine started inside NewCollector** — Lets cmd/* binaries control the delivery-reporting lifecycle via the shared run.Group, ensuring graceful shutdown and avoiding goroutine leaks — consistent with the oklog/run pattern used throughout all cmd/* entrypoints.

## Example: Wiring Collector and registering NamespaceHandler in app/common

```
import (
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

resolver := topicresolver.NewNamespacedTopicResolver("om_%s_events")
collector, err := kafkaingest.NewCollector(producer, serializer.NewJSONSerializer(), resolver, provisioner, 4, logger, tracer)
// Register namespace handler BEFORE initNamespace() so default namespace gets provisioned:
namespaceManager.RegisterHandler(kafkaingest.NamespaceHandler{
	TopicResolver:    resolver,
	TopicProvisioner: provisioner,
	Partitions:       4,
	DeletionEnabled:  false,
// ...
```

<!-- archie:ai-end -->
