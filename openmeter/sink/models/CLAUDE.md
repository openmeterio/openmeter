# models

<!-- archie:ai-start -->

> Pure types package defining the shared data structures for the sink worker pipeline — SinkMessage (the unit of work flowing from Kafka consumer through deduplication to ClickHouse flush) and ProcessingStatus/ProcessingState (OK/DROP disposition tracking). Contains no business logic; all sink pipeline stages import this package.

## Patterns

**SinkMessage as pipeline carrier** — All sink pipeline stages operate on SinkMessage, which bundles the raw kafka.Message, its deserialized CloudEventsKafkaPayload, the resolved Namespace, affected Meters, and ProcessingStatus. New pipeline data must be added as fields here, not passed separately between stages. (`msg := models.SinkMessage{Namespace: ns, KafkaMessage: km, Serialized: payload, Status: models.ProcessingStatus{State: models.OK}}`)
**GetDedupeItem bridges dedupe package** — SinkMessage.GetDedupeItem() produces a dedupe.Item using Namespace, Serialized.Id, and Serialized.Source — the canonical deduplication key. Always use this method rather than constructing dedupe.Item inline to ensure the key composition stays consistent. (`item := msg.GetDedupeItem() // returns dedupe.Item{Namespace: m.Namespace, ID: m.Serialized.Id, Source: m.Serialized.Source}`)
**ProcessingState as int8 iota enum with String()** — States are int8 iota constants (OK=0, DROP=1). String() is implemented for logging. New states must follow the same iota pattern and add a case to String() — omitting the String() case produces 'unknown(N)' in logs. (`const (
	OK ProcessingState = iota
	DROP
)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Single source of truth for all sink pipeline data types. All sink stages (Kafka consumer, deduplication, ClickHouse storage, flush handlers) import this package. | SinkMessage.Serialized may be nil before deserialization completes — callers must nil-check before accessing Serialized.Id or Serialized.Source. ProcessingStatus.DropError should be non-nil whenever State==DROP to preserve the reason for dropping. |

## Anti-Patterns

- Adding business logic or validation to this package — it must remain a pure types/models package with no side effects
- Constructing dedupe.Item directly from SinkMessage fields instead of calling GetDedupeItem()
- Adding a new ProcessingState iota constant without adding a matching case to ProcessingState.String()
- Importing this package from openmeter/streaming or openmeter/meter — those packages are upstream dependencies of sink/models, not consumers; doing so creates a circular dependency

## Decisions

- **SinkMessage holds both the raw kafka.Message and the deserialized CloudEventsKafkaPayload as separate fields** — Deserialization is a discrete pipeline step; keeping both lets downstream stages access Kafka metadata (offsets, partition) alongside the typed event payload without re-parsing the raw bytes.
- **ProcessingState is int8 iota rather than a string enum** — The sink worker processes high volumes of messages; int8 comparisons are cheaper than string comparisons in the hot flush path.

## Example: Constructing a SinkMessage after Kafka consumption and checking deduplication disposition

```
import (
	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

msg := models.SinkMessage{
	Namespace:    ns,
	KafkaMessage: km,
	serialized:   payload, // set after deserialization step
	Status:       models.ProcessingStatus{State: models.OK},
}

// Bridge to deduplication layer — always use GetDedupeItem, not inline construction
item := msg.GetDedupeItem()
if duplicate {
	msg.Status = models.ProcessingStatus{
// ...
```

<!-- archie:ai-end -->
