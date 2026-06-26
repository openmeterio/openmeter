# models

<!-- archie:ai-start -->

> Shared data-transfer types for the sink-worker pipeline. Defines SinkMessage (a Kafka event in flight) and its ProcessingStatus/ProcessingState so sink, flushhandler, and ingestnotification packages exchange state without a circular import.

## Patterns

**Leaf type-only package** — This package holds plain data structs and value-method helpers only — no services, adapters, or I/O. It exists to break import cycles between openmeter/sink and its sub-packages, so keep it dependency-light (it imports only dedupe, serializer, meter, kafka). (`type SinkMessage struct { Namespace string; KafkaMessage *kafka.Message; Serialized *serializer.CloudEventsKafkaPayload; ... }`)
**Stringer on int8 enum with explicit iota constants** — ProcessingState is an int8 enum with OK/DROP declared via iota and a String() method whose default branch formats unknown(%d). New states must be added to both the const block and the switch in String(). (`const ( OK ProcessingState = iota; DROP )`)
**Value-receiver derivation helper** — GetDedupeItem is a value-receiver method that derives a dedupe.Item from the message's Serialized payload (Id, Source) plus Namespace. It assumes Serialized is non-nil. (`func (m SinkMessage) GetDedupeItem() dedupe.Item { return dedupe.Item{Namespace: m.Namespace, ID: m.Serialized.Id, Source: m.Serialized.Source} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Entire package: SinkMessage struct, ProcessingStatus struct, ProcessingState enum + String(), and GetDedupeItem() helper. | GetDedupeItem dereferences m.Serialized without a nil check — callers must populate Serialized before deduping. IngestedAt/StoredAt are *time.Time and may be nil. DropError carries the reason a message is DROPped; it is only meaningful when State==DROP. |

## Anti-Patterns

- Adding service logic, persistence, or Kafka consumption here — this is a type-only leaf package; behavior belongs in openmeter/sink.
- Importing openmeter/sink or its sub-packages from here — that reintroduces the import cycle this package was created to avoid.
- Adding a new ProcessingState const without updating the String() switch, leaving it to fall through to unknown(%d).
- Calling GetDedupeItem() on a SinkMessage whose Serialized is nil — it will panic.

## Decisions

- **Carry processing outcome as a ProcessingStatus struct ({State, DropError}) rather than a bare error.** — Lets the pipeline distinguish OK vs intentional DROP and attach a drop reason without conflating it with fatal processing errors.
- **Keep these types in a standalone models package instead of in openmeter/sink.** — openmeter/sink, flushhandler, and ingestnotification all need SinkMessage; a shared leaf package avoids cyclic imports between them.

## Example: Construct a SinkMessage and derive its dedupe key

```
import (
	"github.com/openmeterio/openmeter/openmeter/dedupe"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

msg := sinkmodels.SinkMessage{
	Namespace:  ns,
	Serialized: payload,
	Status:     sinkmodels.ProcessingStatus{State: sinkmodels.OK},
}
var item dedupe.Item = msg.GetDedupeItem()
```

<!-- archie:ai-end -->
