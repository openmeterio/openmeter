# events

<!-- archie:ai-start -->

> Defines the EventBatchedIngest domain event struct published to Watermill's balance topic after a successful ClickHouse flush, carrying the namespace, subject key, affected meter slugs, raw CloudEvents payloads, and a StoredAt timestamp so downstream balance-worker consumers can recalculate entitlement burn-down.

## Patterns

**marshaler.Event interface compliance** — Every event struct must implement marshaler.Event via EventName() string and EventMetadata() metadata.EventMetadata. A blank var _ marshaler.Event = EventBatchedIngest{} compile-time assertion enforces this. (`var _ marshaler.Event = EventBatchedIngest{}`)
**Versioned event type via metadata.EventType** — Event identity is declared as a metadata.EventType{Subsystem, Name, Version} constant; EventName() returns metadata.GetEventName(type) — never a bare string literal. (`batchIngestEventType = metadata.EventType{Subsystem: EventSubsystem, Name: "events.ingested", Version: "v2"}`)
**EventMetadata uses metadata.ComposeResourcePath helpers** — EventMetadata() returns source via ComposeResourcePathRaw and subject via ComposeResourcePath(namespaceID, entityKind, key) — never hand-rolled path strings. (`Subject: metadata.ComposeResourcePath(b.Namespace.ID, metadata.EntitySubjectKey, b.SubjectKey)`)
**MeterSlugs not MeterIDs** — Downstream consumers must reference meters by slug, not by internal ID, because meter IDs are absent in the open-source build. New fields referencing meters must use slugs. (`MeterSlugs []string `json:"meterSlugs"``)
**Validate() returns combined error** — Validate() delegates to embedded types (Namespace.Validate()) and checks required scalar fields. Return a descriptive errors.New string, never silently skip validation. (`if b.SubjectKey == "" { return errors.New("subjectKey must be set") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `events.go` | Single file defining EventBatchedIngest, its event-name constants, EventVersionSubsystem export, and the marshaler.Event compile-time assertion. | Adding a new event type requires a new EventType constant, a GetEventName call, a matching EventVersionSubsystem export, and a fresh var _ marshaler.Event assertion. Forgetting any of these causes silent routing failure in the Watermill router. |

## Anti-Patterns

- Using meter IDs instead of meter slugs in event payloads — IDs are unavailable in open-source consumers
- Hand-coding event name strings instead of using metadata.GetEventName(eventType)
- Omitting the var _ marshaler.Event compile-time assertion when adding a new event struct
- Introducing context.Background() or context.TODO() — events are value types; context is not needed here
- Storing mutable state in the event struct — it must remain a pure value type for safe marshaling

## Decisions

- **EventVersionSubsystem is exported so the balance-worker consumer can subscribe by subsystem prefix without importing internal metadata helpers.** — Decouples producer and consumer versioning; consumers match on VersionSubsystem, not on the full event name, so a version bump is isolated to the producer side.
- **RawEvents embeds the full serializer.CloudEventsKafkaPayload slice rather than derived fields.** — Avoids re-querying ClickHouse in the balance-worker; the worker has all usage data it needs to recalculate grant burn-down without an additional read path.

## Example: Adding a new ingest notification event type

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
	newIngestEventName    = metadata.GetEventName(newIngestEventType)
	NewIngestVersionSubsystem = newIngestEventType.VersionSubsystem()
// ...
```

<!-- archie:ai-end -->
