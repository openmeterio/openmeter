# ingest

<!-- archie:ai-start -->

> CloudEvent ingestion pipeline: the Collector interface receives single events and forwards them to Kafka; DeduplicatingCollector wraps any Collector with Redis or in-memory deduplication; ingestadapter decorates with OTel telemetry without altering ingest behavior.

## Patterns

**Collector interface wrapping via struct embedding** — DeduplicatingCollector embeds Collector and overrides Ingest() to call Deduplicator.IsUnique before delegating to the inner Collector. New decorators must follow the same embed-and-override pattern. (`ingest/dedupe.go: DeduplicatingCollector struct embeds Collector and adds Deduplicator field; Ingest calls IsUnique before delegating`)
**OTel metric instruments allocated in constructor, never in Ingest()** — ingestadapter/telemetry.go creates Int64Counters in New(), not inside Ingest(). This avoids allocation and registration overhead on every event. (`ingestadapter/telemetry.go: New() creates events_ingested and errors counters; Ingest() calls tracer.Start and records span`)
**Topic provisioned on every Ingest call in kafkaingest** — kafkaingest/collector.go calls TopicResolver.Resolve + TopicProvisioner.EnsureExists on each Ingest call, not only at namespace creation. Ensures topics exist even if the namespace was created without Kafka available. (`kafkaingest/collector.go: Ingest calls resolver.Resolve then provisioner.EnsureExists before Producer.Produce`)
**Content-type dispatch in httpdriver** — httpdriver/ingest.go decodes requests by Content-Type: single (application/cloudevents+json) or batch (application/cloudevents-batch+json). Invalid content types return ErrorInvalidContentType, not raw errors. (`httpdriver/ingest.go: switch on r.Header.Get('Content-Type') to select decoder path`)
**KafkaProducerGroup as oklog/run pair, not an internal goroutine** — kafkaingest returns a KafkaProducerGroup that must be added to the caller's run.Group to process delivery reports. Never start it as an internal goroutine inside NewCollector. (`kafkaingest/collector.go: NewCollector returns (Collector, KafkaProducerGroup, error); caller adds group to run.Group`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/ingest/ingest.go` | Collector interface definition — the only required interface for the ingest domain. | Close() is a lifecycle method; callers must call it to drain the Kafka producer before shutdown. Do not add new methods to Collector without considering all implementations. |
| `openmeter/ingest/dedupe.go` | DeduplicatingCollector — thin deduplication wrapper over any Collector using dedupe.Deduplicator. | IsUnique returns (true, nil) for unique events and (false, nil) for duplicates. A (false, err) case means the deduplicator itself failed — the event is still forwarded to avoid data loss. |
| `openmeter/ingest/service.go` | ingest.Service wraps Collector with namespace-scoped validation and exposes IngestEvents for multi-event batches. | service.go validates namespace before calling Collector.Ingest. New HTTP endpoints that accept events must go through Service, not Collector directly. |

## Anti-Patterns

- Calling Producer.Produce in kafkaingest without running KafkaProducerGroup in an oklog/run.Group — delivery errors are silently dropped.
- Creating OTel metric instruments inside Ingest() instead of in the constructor — causes allocation and registration overhead per event.
- Reading namespace from URL path params in httpdriver instead of NamespaceDecoder — bypasses static namespace injection for self-hosted deployments.
- Adding new Kafka headers inside kafkaingest/collector.go without updating the sink worker's header parsing — breaks downstream deserialization.
- Importing app/common from httpdriver — the HTTP handler must remain DI-agnostic.

## Decisions

- **OTel telemetry is in a separate ingestadapter package rather than embedded in kafkaingest.** — Keeps kafkaingest focused on Kafka protocol; ingestadapter can be swapped or omitted in tests without touching producer logic.
- **Topic is resolved and provisioned on every Ingest call, not only at namespace creation.** — Ensures topics exist even if the namespace was created without Kafka being available; removes a class of startup-order failures.

<!-- archie:ai-end -->
