# models

<!-- archie:ai-start -->

> Pure types package defining the shared data structures for the sink worker pipeline: SinkMessage (the unit of work flowing from the Kafka consumer through deduplication to ClickHouse flush) and ProcessingStatus/ProcessingState (OK/DROP disposition). No business logic; every sink pipeline stage imports it.

## Patterns

**SinkMessage as pipeline carrier** — All stages operate on SinkMessage bundling the raw kafka.Message, deserialized CloudEventsKafkaPayload, resolved Namespace, affected Meters, and ProcessingStatus. New pipeline data is added as fields here, not passed separately. (`msg := models.SinkMessage{Namespace: ns, KafkaMessage: km, Serialized: payload, Status: models.ProcessingStatus{State: models.OK}}`)
**GetDedupeItem bridges the dedupe package** — SinkMessage.GetDedupeItem() builds the canonical dedupe.Item from Namespace, Serialized.Id, and Serialized.Source; always use it rather than constructing dedupe.Item inline. (`item := msg.GetDedupeItem() // dedupe.Item{Namespace, ID: Serialized.Id, Source: Serialized.Source}`)
**ProcessingState as int8 iota enum with String()** — States are int8 iota constants (OK=0, DROP=1) with String() for logging; new states follow the iota pattern and add a String() case. (`const ( OK ProcessingState = iota; DROP )`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Single source of truth for all sink pipeline data types, imported by every sink stage (consumer, dedup, ClickHouse storage, flush handlers). | Serialized may be nil before deserialization — nil-check before accessing Serialized.Id/Source; ProcessingStatus.DropError should be non-nil whenever State==DROP to preserve the reason. |

## Anti-Patterns

- Adding business logic or validation — this must remain a pure types/models package.
- Constructing dedupe.Item directly from SinkMessage fields instead of calling GetDedupeItem().
- Adding a new ProcessingState iota constant without a matching ProcessingState.String() case.
- Importing this package from openmeter/streaming or openmeter/meter — they are upstream deps, not consumers; doing so creates a circular dependency.

## Decisions

- **SinkMessage holds both the raw kafka.Message and the deserialized payload as separate fields.** — Deserialization is a discrete step; keeping both lets downstream stages access Kafka metadata (offsets, partition) alongside the typed payload without re-parsing raw bytes.
- **ProcessingState is int8 iota, not a string enum.** — The sink processes high volumes; int8 comparisons are cheaper than string comparisons in the hot flush path.

## Example: Construct a SinkMessage after Kafka consumption and check deduplication disposition

```
import "github.com/openmeterio/openmeter/openmeter/sink/models"

msg := models.SinkMessage{
  Namespace:    ns,
  KafkaMessage: km,
  Serialized:   payload, // set after deserialization
  Status:       models.ProcessingStatus{State: models.OK},
}
item := msg.GetDedupeItem() // always use GetDedupeItem, not inline construction
if duplicate {
  msg.Status = models.ProcessingStatus{State: models.DROP, DropError: dedupeErr}
}
```

<!-- archie:ai-end -->
