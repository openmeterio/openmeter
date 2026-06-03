# ingest

<!-- archie:ai-start -->

> CloudEvent ingestion pipeline: the Collector interface receives single events and forwards them to Kafka, with composable decorators — DeduplicatingCollector (Redis or in-memory dedup) and ingestadapter (OTel telemetry) — and ingest.Service adding namespace-scoped validation for batch ingestion.

## Patterns

**Collector decorated by struct embedding** — DeduplicatingCollector embeds Collector and overrides Ingest() to call Deduplicator.IsUnique before delegating; ingestadapter wraps Collector for OTel spans/counters. New decorators must follow this embed-and-override pattern and remain behavior-preserving. (`ingest/dedupe.go: DeduplicatingCollector embeds Collector; Ingest calls IsUnique before delegating`)
**kafkaingest produces to per-namespace topics, resolving+provisioning per call** — kafkaingest/collector.go resolves the topic via TopicResolver and provisions via TopicProvisioner on every Ingest call (not just at namespace creation), encodes key from dedupe.Item, sets the fixed Kafka header set, and returns a KafkaProducerGroup the caller must add to its oklog/run.Group for delivery reporting. (`NewCollector returns (Collector, KafkaProducerGroup, error); caller adds group to run.Group`)
**httpdriver dispatches by Content-Type, stays DI-agnostic** — httpdriver/ingest.go selects single vs batch decoding by Content-Type (application/json, cloudevents+json, cloudevents-batch+json), returns domain-local errors (ErrorInvalidEvent/ErrorInvalidContentType) through an errorEncoder chain, resolves namespace via NamespaceDecoder, and delegates all logic to ingest.Service. (`switch on r.Header.Get('Content-Type') to select the decoder path`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/ingest/ingest.go` | Collector interface (Ingest, Close) — the only required interface for the domain. | Close() drains the Kafka producer before shutdown; do not add methods to Collector without considering all implementations and decorators. |
| `openmeter/ingest/dedupe.go` | DeduplicatingCollector wrapper over any Collector via dedupe.Deduplicator. | (false, err) means the deduplicator itself failed — the event is still forwarded to avoid data loss; only (false, nil) is a true duplicate. |
| `openmeter/ingest/service.go` | ingest.Service: namespace-scoped validation wrapper exposing IngestEvents for batches. | Validates namespace before Collector.Ingest; new endpoints must go through Service, not Collector directly. |

## Anti-Patterns

- Calling Producer.Produce in kafkaingest without running KafkaProducerGroup in an oklog/run.Group — delivery errors are silently dropped.
- Creating OTel metric instruments inside Ingest() instead of the constructor — allocation/registration overhead per event.
- Reading namespace from URL path params in httpdriver instead of NamespaceDecoder — bypasses static namespace injection for self-hosted deployments.
- Adding new Kafka headers in kafkaingest without updating the sink worker's header parsing — breaks downstream deserialization.
- Importing app/common from httpdriver — the HTTP handler must stay DI-agnostic.

## Decisions

- **OTel telemetry is a separate ingestadapter package rather than embedded in kafkaingest.** — Keeps kafkaingest focused on the Kafka protocol; the decorator can be swapped or omitted in tests without touching producer logic.
- **Topic is resolved and provisioned on every Ingest call, not only at namespace creation.** — Ensures topics exist even if the namespace was created while Kafka was unavailable, removing a class of startup-order failures.

<!-- archie:ai-end -->
