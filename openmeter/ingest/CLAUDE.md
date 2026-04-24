# ingest

<!-- archie:ai-start -->

> CloudEvent ingestion pipeline: the Collector interface receives single events and forwards them to Kafka; DeduplicatingCollector wraps any Collector with Redis or in-memory deduplication; ingestadapter decorates with OTel telemetry. The httpdriver package translates multi-format HTTP requests into ingest.Service calls.

## Patterns

**Collector interface wrapping via struct embedding** — DeduplicatingCollector embeds Collector and overrides Ingest() to call Deduplicator.IsUnique before delegating to the inner Collector. New decorators must follow the same embed-and-override pattern. (`ingest/dedupe.go: DeduplicatingCollector struct embeds Collector and adds Deduplicator field`)
**OTel span per Ingest call in ingestadapter** — ingestadapter/telemetry.go wraps every Ingest call with tracer.Start + span.RecordError and increments two counters (events_ingested, errors). Metric instruments are created in the constructor, never inside Ingest(). (`ingestadapter/telemetry.go: New() creates Int64Counters; Ingest() calls tracer.Start and records span`)
**Topic provisioned on every Ingest call** — kafkaingest/collector.go calls TopicResolver.Resolve + TopicProvisioner.EnsureExists on each Ingest, not just at namespace creation. This ensures topics exist even if the namespace was created without Kafka being available. (`kafkaingest/collector.go: Ingest calls resolver.Resolve then provisioner.EnsureExists before Producer.Produce`)
**Content-type dispatch in httpdriver** — httpdriver/ingest.go decodes requests by Content-Type: application/cloudevents+json (single), application/cloudevents-batch+json (batch). Invalid content types are returned as ErrorInvalidContentType, not raw errors. (`httpdriver/ingest.go: switch on r.Header.Get('Content-Type') to select decoder path`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/ingest/ingest.go` | Collector interface definition — the only required interface for the ingest domain. | Close() is a lifecycle method; callers must call it to drain the Kafka producer before shutdown. Do not add new methods to Collector without considering all implementations. |
| `openmeter/ingest/dedupe.go` | DeduplicatingCollector — thin deduplication wrapper over any Collector using dedupe.Deduplicator. | IsUnique returns (true, nil) for unique events and (false, nil) for duplicates. A (false, err) case means the deduplicator itself failed — the event is still forwarded to avoid data loss. |
| `openmeter/ingest/service.go` | ingest.Service wraps Collector with namespace-scoped validation and exposes IngestEvents for multi-event batches. | service.go validates namespace before calling Collector.Ingest. New HTTP endpoints that accept events must go through Service, not Collector directly. |

## Anti-Patterns

- Calling Producer.Produce in kafkaingest without running KafkaProducerGroup — delivery errors are silently dropped.
- Creating OTel metric instruments inside Ingest() instead of in the constructor — causes allocation and registration overhead per event.
- Reading namespace from URL path params in httpdriver instead of NamespaceDecoder — bypasses the static namespace injection for self-hosted deployments.
- Adding new Kafka headers inside kafkaingest/collector.go without updating the sink worker's header parsing — breaks downstream deserialization.

## Decisions

- **OTel telemetry is in a separate ingestadapter package rather than embedded in kafkaingest.** — Keeps kafkaingest focused on Kafka protocol; ingestadapter can be swapped or omitted in tests without touching producer logic.

<!-- archie:ai-end -->
