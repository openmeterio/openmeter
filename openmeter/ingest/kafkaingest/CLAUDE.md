# kafkaingest

<!-- archie:ai-start -->

> Kafka-publishing side of the ingest pipeline: turns validated CloudEvents into Kafka messages via Collector.Ingest, and provisions/deprovisions per-namespace topics via NamespaceHandler. It is the producer boundary between the ingest HTTP service and the sink-worker that drains Kafka into ClickHouse.

## Patterns

**Validating constructor with required-dependency guards** — NewCollector returns (*Collector, error) and rejects any nil dependency (producer, serializer, resolver, provisioner, logger, tracer) before constructing. New collectors must validate every injected field. (`if producer == nil { return nil, fmt.Errorf("producer is required") }`)
**Resolve-then-provision before produce** — Ingest always resolves namespace->topic via TopicResolver.Resolve, then TopicProvisioner.Provision (with TopicPartitions), and only then serializes and produces. Never produce to an unresolved/unprovisioned topic. (`topicName, _ := s.TopicResolver.Resolve(ctx, namespace); s.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{Name: topicName, Partitions: s.TopicPartitions})`)
**Key from Serializer, never hand-built** — The Kafka message Key comes from Serializer.SerializeKey and MUST match the dedupe hash; the partitioner is assumed consistent-hash. Hand-constructing keys breaks dedupe partition routing. (`key, _ := s.Serializer.SerializeKey(topicName, namespace, ev); msg := &kafka.Message{Key: key, Value: value}`)
**Standard ingest headers** — Every produced message carries namespace (HeaderKeyNamespace), specversion, ingested_at (HeaderKeyIngestedAt via ToIngestedAt/clock.Now), and the serialized OTel span context (otelx.OTelSpanContextKey). New headers must use the exported constants and clock.Now(). (`Headers: []kafka.Header{{Key: HeaderKeyNamespace, Value: []byte(namespace)}, {Key: HeaderKeyIngestedAt, Value: []byte(ToIngestedAt(clock.Now()))}}`)
**Span-wrapped Ingest with deferred error recording** — Ingest opens a tracer span (openmeter.ingest.process.event), assigns errors to a named err captured in defer, and records/sets status on span exit. New produce paths follow this named-err + deferred span.End pattern. (`ctx, span := s.Tracer.Start(ctx, "openmeter.ingest.process.event", ...); defer func(){ if err != nil { span.RecordError(err); span.SetStatus(otelcodes.Error, err.Error()) }; span.End() }()`)
**ingested_at RFC3339Nano UTC round-trip** — ingested_at is encoded with ToIngestedAt (UTC RFC3339Nano) and decoded with FromIngestedAt; these are the only sanctioned converters for that header. (`func ToIngestedAt(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }`)
**NamespaceHandler gated DeleteNamespace** — NamespaceHandler implements CreateNamespace/DeleteNamespace; DeleteNamespace is a no-op unless DeletionEnabled, and both nil-check TopicResolver before use. (`func (h NamespaceHandler) DeleteNamespace(ctx, namespace) error { if !h.DeletionEnabled { return nil } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector.go` | Collector struct + NewCollector + Ingest (produce one event) + Close (Flush 30s then Close) + KafkaProducerGroup (run.Group execute/interrupt that drains producer.Events for delivery reports, kafka.Stats metrics, and kafka.Error logging). Defines ingest headers and ToIngestedAt/FromIngestedAt. | Ingest takes Collector by value (func (s Collector)); don't add mutable state expecting it to persist. KafkaProducerGroup logs librdkafka 'local' errors (code <= -100) at warn and broker errors at error — preserve that split. Stats are unmarshalled in a goroutine with a 5s timeout context. |
| `namespace.go` | NamespaceHandler implementing the namespace handler interface (CreateNamespace provisions, DeleteNamespace deprovisions) so Kafka topics track namespace lifecycle. | DeleteNamespace silently returns nil when DeletionEnabled is false — a missing deprovision is intentional, not a bug. Both methods must guard TopicResolver != nil before resolving. |

## Anti-Patterns

- Hand-building the Kafka message Key instead of using Serializer.SerializeKey — diverges from the dedupe hash and causes deduplication race conditions across partitions.
- Producing before TopicResolver.Resolve and TopicProvisioner.Provision — risks publishing to a non-existent or wrong topic.
- Bypassing ToIngestedAt/clock.Now() (e.g. time.Now() directly) for the ingested_at header — breaks test-time freezing and the RFC3339Nano contract.
- Constructing a Collector without NewCollector (skipping nil-dependency validation).
- Adding new headers without the exported HeaderKey* constants or omitting the OTel span context header — loses trace propagation into the sink.

## Decisions

- **Topic provisioning happens inline on every Ingest call rather than once at startup.** — Namespaces (and their topics) are created dynamically; provisioning per-ingest guarantees the topic exists for newly seen namespaces without a separate bootstrap step.
- **Delivery reports, client stats, and client errors are handled in a separate KafkaProducerGroup loop, not synchronously in Ingest.** — librdkafka produce is async; the application already configures internal retries, so Ingest returns on enqueue and the events loop reports terminal delivery outcome and emits Kafka client metrics.

## Example: Resolve namespace, provision topic, serialize, and produce a CloudEvent to Kafka

```
func (s Collector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	topicName, err := s.TopicResolver.Resolve(ctx, namespace)
	if err != nil { return fmt.Errorf("failed to resolve namespace to topic name: %w", err) }
	if err = s.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{Name: topicName, Partitions: s.TopicPartitions}); err != nil {
		return fmt.Errorf("failed to provision topic: %w", err)
	}
	key, err := s.Serializer.SerializeKey(topicName, namespace, ev)
	if err != nil { return fmt.Errorf("serialize event key: %w", err) }
	value, err := s.Serializer.SerializeValue(topicName, ev)
	if err != nil { return fmt.Errorf("serialize event value: %w", err) }
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topicName, Partition: kafka.PartitionAny},
		Timestamp:      ev.Time(),
		Headers: []kafka.Header{
			{Key: HeaderKeyNamespace, Value: []byte(namespace)},
// ...
```

<!-- archie:ai-end -->
