# events

<!-- archie:ai-start -->

> Defines the EventBatchedIngest domain event published to Watermill's balance-worker Kafka topic after a successful ClickHouse flush, carrying namespace, subject key, affected meter slugs, raw CloudEvent payloads, and a StoredAt timestamp so downstream balance-worker consumers can recalculate entitlement grant burn-down without re-querying ClickHouse.

## Patterns

**marshaler.Event interface compliance via compile-time assertion** — Every event struct must implement marshaler.Event (EventName() string + EventMetadata() metadata.EventMetadata). A blank var _ marshaler.Event = EventBatchedIngest{} assertion must appear alongside each new struct. Missing it causes silent routing failure in the Watermill eventbus. (`var _ marshaler.Event = EventBatchedIngest{}`)
**Versioned event identity via metadata.EventType constant** — Event identity is declared as a package-level metadata.EventType{Subsystem, Name, Version} constant. EventName() must return metadata.GetEventName(eventType) — never a bare string literal. This ensures GeneratePublishTopic routes correctly by EventVersionSubsystem prefix. (`batchIngestEventType = metadata.EventType{Subsystem: EventSubsystem, Name: "events.ingested", Version: "v2"}`)
**Export EventVersionSubsystem for consumer-side subscription** — Each event type must export its VersionSubsystem string so balance-worker and other consumers can subscribe by subsystem prefix without importing internal metadata helpers. Computed via batchIngestEventType.VersionSubsystem(). (`EventVersionSubsystem = batchIngestEventType.VersionSubsystem()`)
**EventMetadata uses metadata.ComposeResourcePath helpers** — EventMetadata() must return Source via ComposeResourcePathRaw and Subject via ComposeResourcePath(namespaceID, entityKind, key). Never hand-roll path strings — inconsistent paths break CloudEvents routing and OTel span correlation. (`Subject: metadata.ComposeResourcePath(b.Namespace.ID, metadata.EntitySubjectKey, b.SubjectKey)`)
**MeterSlugs not MeterIDs in payloads** — Any field referencing meters must use slugs ([]string), not internal ULIDs. Meter IDs are absent in the open-source build; downstream balance-worker consumers would silently receive zero-length lookups. (`MeterSlugs []string `json:"meterSlugs"``)
**Validate() returns combined descriptive errors** — Validate() must delegate to embedded type validators (e.g. Namespace.Validate()) and check every required scalar field with a descriptive errors.New string. Never silently skip validation or return nil for an empty required field. (`if b.SubjectKey == "" { return errors.New("subjectKey must be set") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `events.go` | Single file defining EventBatchedIngest, its metadata.EventType constant, GetEventName call, exported EventVersionSubsystem, and the marshaler.Event compile-time assertion. This is the complete public API of this package. | Adding a new event type requires: a new metadata.EventType constant, a GetEventName call, an exported VersionSubsystem var, a var _ marshaler.Event assertion, and a Validate() implementation. Omitting any of these causes silent Kafka misrouting or consumer decoding failures. |

## Anti-Patterns

- Using meter IDs instead of meter slugs in event payloads — IDs are unavailable in open-source consumers
- Hard-coding event name strings instead of using metadata.GetEventName(eventType)
- Omitting the var _ marshaler.Event compile-time assertion when adding a new event struct
- Storing mutable state or pointers in the event struct — it must be a pure value type for safe concurrent marshaling
- Adding context.Context parameters — event structs are value types; context propagation belongs in the publisher call site

## Decisions

- **EventVersionSubsystem is exported so balance-worker consumers can subscribe by subsystem prefix without importing internal metadata helpers.** — Decouples producer and consumer versioning; consumers match on VersionSubsystem prefix, so a version bump (v2 -> v3) is isolated to the producer side and does not require consumer recompilation.
- **RawEvents embeds the full serializer.CloudEventsKafkaPayload slice rather than derived aggregated fields.** — Avoids re-querying ClickHouse in the balance-worker; the worker has all raw usage data it needs to recalculate grant burn-down without an additional read path, keeping the flush-to-balance pipeline fully in-memory.

## Example: Adding a new ingest notification event type alongside EventBatchedIngest

```
import (
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

var (
	_ marshaler.Event = EventNewIngest{}
	newIngestEventType = metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "events.new",
		Version:   "v1",
	}
	newIngestEventName       = metadata.GetEventName(newIngestEventType)
	NewIngestVersionSubsystem = newIngestEventType.VersionSubsystem()
// ...
```

<!-- archie:ai-end -->
