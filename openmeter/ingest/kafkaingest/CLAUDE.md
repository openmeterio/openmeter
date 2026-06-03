# kafkaingest

<!-- archie:ai-start -->

> Implements ingest.Collector by producing CloudEvents to per-namespace Kafka topics via confluent-kafka-go. Owns per-call topic resolution/provisioning, OTel span lifecycle per event, the fixed Kafka header set, and the KafkaProducerGroup oklog/run pair for async delivery reporting. Its child sub-packages split by concern: serializer/ (key/value encoding) and topicresolver/ (namespace→topic mapping).

## Patterns

**Serializer injection for key and value** — Collector never encodes events itself — it delegates to serializer.Serializer inside every Ingest call: SerializeKey encodes a dedupe.Item (namespace+source+id) for sink dedup, SerializeValue encodes a flat CloudEventsKafkaPayload. (`key, _ := s.Serializer.SerializeKey(topicName, namespace, ev); value, _ := s.Serializer.SerializeValue(topicName, ev)`)
**Topic resolved + provisioned per Ingest call** — TopicResolver.Resolve maps namespace→topic; TopicProvisioner.Provision (idempotent, no-op on existing) ensures the topic exists before Produce. Both run per call, not at construction. (`topicName, _ := s.TopicResolver.Resolve(ctx, namespace); _ = s.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{Name: topicName, Partitions: s.TopicPartitions})`)
**OTel span wraps the full Ingest with deferred error recording** — Ingest opens a span at entry and a deferred closure captures the local 'err' variable, calling span.RecordError + span.SetStatus on any non-nil error. Do not shadow 'err' with := inside the body. (`ctx, span := s.Tracer.Start(ctx, "openmeter.ingest.process.event", ...); defer func() { if err != nil { span.RecordError(err); span.SetStatus(otelcodes.Error, err.Error()) }; span.End() }()`)
**Fixed Kafka header set per message** — Every kafka.Message carries four headers: 'namespace' ([]byte), 'specversion', 'ingested_at' (RFC3339Nano UTC via ToIngestedAt), and otelx.OTelSpanContextKey. The sink worker reads these to reconstruct ctx and attribute tenancy. (`Headers: []kafka.Header{{Key: HeaderKeyNamespace, Value: []byte(namespace)}, {Key: "specversion", Value: []byte(ev.SpecVersion())}, {Key: HeaderKeyIngestedAt, Value: []byte(ToIngestedAt(clock.Now()))}, {Key: otelx.OTelSpanContextKey, Value: spanCtx}}`)
**KafkaProducerGroup as oklog/run pair** — KafkaProducerGroup returns (execute, interrupt) for the caller's run.Group; execute drains producer.Events() handling *kafka.Message delivery reports, *kafka.Stats, and kafka.Error. The caller must start it — Produce alone drops delivery results. (`execute, interrupt := kafkaingest.KafkaProducerGroup(ctx, producer, logger, kafkaMetrics); runGroup.Add(execute, interrupt)`)
**NamespaceHandler reuses Resolver + Provisioner** — NamespaceHandler.CreateNamespace calls Resolve then Provision; DeleteNamespace is a no-op unless DeletionEnabled=true. The TopicResolver nil-check is deferred to call time, so a zero-value handler panics in production. (`namespaceManager.RegisterHandler(kafkaingest.NamespaceHandler{TopicResolver: resolver, TopicProvisioner: provisioner, Partitions: 4, DeletionEnabled: false})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector.go` | Collector struct (NewCollector validates non-nil deps), Ingest produce path, Close, and KafkaProducerGroup; ToIngestedAt/FromIngestedAt header time helpers. | Producer.Produce is fire-and-forget; without KafkaProducerGroup running, delivery failures are silent. Do not shadow the deferred-captured 'err'. |
| `namespace.go` | NamespaceHandler implementing namespace.Handler for ingest topics; provisions on create, deprovisions only when DeletionEnabled=true. | Zero-value handler panics at call time (deferred nil-check). Register before initNamespace() in cmd/server/main.go or the default namespace misses provisioning. |
| `serializer/json.go` | JSONSerializer: SerializeKey (dedupe.Item→key bytes) and SerializeValue (CloudEventsKafkaPayload→value bytes); wire format consumed by the sink worker's FromKafkaPayloadToCloudEvents. | CloudEventsKafkaPayload.Time is unix int64 (timezone intentionally dropped); Data is a JSON string, not raw bytes. Changes must be mirrored in the sink deserializer. |
| `serializer/serializer.go` | Serializer interface with GetKeySchemaId/GetValueSchemaId hooks; the JSON impl returns 0 for both. | A schema-registry-aware serializer must return real IDs from all four methods or the sink worker mis-deserializes. |
| `topicresolver/namespacedtopic.go` | NamespacedTopicResolver maps namespace→topic via fmt.Sprintf(template, namespace); default 'om_%s_events'. | Constructor validates non-empty template but not %s count — zero or two placeholders silently produce wrong topic names. |
| `topicresolver/resolver.go` | Resolver interface, ctx-accepting for future async/DB-backed implementations. | Always propagate the caller's ctx to Resolve — never context.Background() in application code. |

## Anti-Patterns

- Calling Producer.Produce without running KafkaProducerGroup in an oklog/run.Group — delivery errors are silently dropped.
- Hard-coding topic names instead of calling TopicResolver.Resolve per namespace.
- Adding new Kafka headers in Ingest without updating the sink worker's header parsing.
- Constructing Collector/NamespaceHandler with a nil TopicResolver — the nil-check is deferred and panics in production.
- Storing timezone-aware time in CloudEventsKafkaPayload.Time — it is unix int64 with timezone intentionally stripped.

## Decisions

- **Topic is resolved and provisioned on every Ingest call, not only at namespace creation.** — Guarantees idempotent topic existence even when NamespaceHandler was not called first (self-hosted single-namespace), at the cost of one idempotent admin check per call.
- **Key encodes dedupe.Item (namespace+source+id), not the event ID alone.** — Partitions by logical event identity so the sink worker's Redis dedup hash matches the Kafka key with no extra lookup, preventing double-counting on consumer restart.
- **KafkaProducerGroup is an oklog/run pair returned to the caller, not a goroutine in NewCollector.** — Lets cmd/* control delivery-reporting lifecycle via the shared run.Group for graceful shutdown and no goroutine leaks, consistent with all cmd/* entrypoints.

## Example: Wiring Collector and registering NamespaceHandler in app/common

```
import (
    "github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
    "github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
    "github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
)

resolver := topicresolver.NewNamespacedTopicResolver("om_%s_events")
collector, err := kafkaingest.NewCollector(producer, serializer.NewJSONSerializer(), resolver, provisioner, 4, logger, tracer)
// Register BEFORE initNamespace() so the default namespace is provisioned:
namespaceManager.RegisterHandler(kafkaingest.NamespaceHandler{
    TopicResolver:    resolver,
    TopicProvisioner: provisioner,
    Partitions:       4,
    DeletionEnabled:  false,
})
```

<!-- archie:ai-end -->
