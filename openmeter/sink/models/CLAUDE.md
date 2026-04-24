# models

<!-- archie:ai-start -->

> Defines the shared data types used by the sink worker pipeline: SinkMessage (the unit of work flowing from Kafka consumer through deduplication to ClickHouse flush) and ProcessingStatus/ProcessingState (OK/DROP disposition tracking). This is a pure types package with no business logic.

## Patterns

**SinkMessage as pipeline carrier** — All sink pipeline stages operate on SinkMessage, which bundles the raw kafka.Message, its deserialized CloudEventsKafkaPayload, the resolved Namespace, affected Meters, and ProcessingStatus. New pipeline data must be added as fields here, not passed separately. (`msg := models.SinkMessage{Namespace: ns, KafkaMessage: km, Serialized: payload, Status: models.ProcessingStatus{State: models.OK}}`)
**GetDedupeItem bridges dedupe package** — SinkMessage.GetDedupeItem() produces a dedupe.Item using Namespace, Serialized.Id, and Serialized.Source — the canonical deduplication key. Always use this method rather than constructing dedupe.Item inline. (`item := msg.GetDedupeItem() // returns dedupe.Item{Namespace, ID, Source}`)
**ProcessingState as iota enum with String()** — States are int8 iota constants (OK=0, DROP=1). String() is implemented for logging. New states must follow the same iota pattern and add a case to String(). (`const (\n\tOK ProcessingState = iota\n\tDROP\n)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Single source of truth for sink pipeline data types. All sink stages (consumer, deduplication, storage, flush handlers) import this package. | SinkMessage.Serialized may be nil before deserialization completes — callers must nil-check before accessing Serialized.Id or Serialized.Source. ProcessingStatus.DropError should be non-nil when State==DROP. |

## Anti-Patterns

- Adding business logic or validation to this package — it must remain a pure types/models package
- Constructing dedupe.Item directly from SinkMessage fields instead of calling GetDedupeItem()
- Adding a new ProcessingState without implementing it in ProcessingState.String()
- Importing this package from openmeter/streaming or openmeter/meter (creates circular dependency — those packages are dependencies of this one)

## Decisions

- **SinkMessage holds both raw kafka.Message and deserialized CloudEventsKafkaPayload as separate fields** — Deserialization is a discrete pipeline step; keeping both lets downstream stages access Kafka metadata (offsets, partition) alongside the typed event payload without re-parsing.
- **ProcessingStatus.State is int8 iota, not a string enum** — The sink worker processes high volumes of messages; int8 comparisons are cheaper than string comparisons in hot paths.

<!-- archie:ai-end -->
