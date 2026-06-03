# events

<!-- archie:ai-start -->

> Defines the EventBatchedIngest domain event published to Watermill's balance-worker Kafka topic after a successful ClickHouse flush, carrying namespace, subject key, affected meter slugs, raw CloudEvent payloads, and a StoredAt timestamp so balance-worker consumers can recalculate entitlement grant burn-down without re-querying ClickHouse.

## Patterns

**marshaler.Event compliance via compile-time assertion** — Each event struct implements marshaler.Event (EventName(), EventMetadata()); a blank var _ marshaler.Event = EventBatchedIngest{} must appear alongside it or routing silently fails. (`var _ marshaler.Event = EventBatchedIngest{}`)
**Versioned event identity via metadata.EventType constant** — Identity is a package-level metadata.EventType{Subsystem, Name, Version}; EventName() returns metadata.GetEventName(eventType), never a bare literal, so GeneratePublishTopic routes by EventVersionSubsystem prefix. (`batchIngestEventType = metadata.EventType{Subsystem: EventSubsystem, Name: "events.ingested", Version: "v2"}`)
**Export EventVersionSubsystem for consumers** — Export each event type's VersionSubsystem string so balance-worker can subscribe by prefix without importing internal metadata helpers. (`EventVersionSubsystem = batchIngestEventType.VersionSubsystem()`)
**EventMetadata uses ComposeResourcePath helpers** — EventMetadata() returns Source via ComposeResourcePathRaw and Subject via ComposeResourcePath(namespaceID, entityKind, key); never hand-roll path strings. (`Subject: metadata.ComposeResourcePath(b.Namespace.ID, metadata.EntitySubjectKey, b.SubjectKey)`)
**MeterSlugs not MeterIDs in payloads** — Meter references use slugs ([]string), not internal ULIDs — IDs are absent in the open-source build and would yield zero-length lookups downstream. (`MeterSlugs []string `json:"meterSlugs"``)
**Validate() returns combined descriptive errors** — Validate() delegates to embedded validators (e.g. Namespace.Validate()) and checks every required scalar with a descriptive error; never silently skip or return nil for an empty required field. (`if b.SubjectKey == "" { return errors.New("subjectKey must be set") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `events.go` | Sole file: EventBatchedIngest, its metadata.EventType constant, GetEventName call, exported EventVersionSubsystem, the marshaler.Event assertion — the complete public API. | A new event type needs a metadata.EventType constant, a GetEventName call, an exported VersionSubsystem var, a var _ marshaler.Event assertion, and a Validate(); omitting any causes silent Kafka misrouting or consumer decode failures. |

## Anti-Patterns

- Using meter IDs instead of meter slugs — IDs are unavailable in open-source consumers.
- Hard-coding event name strings instead of metadata.GetEventName(eventType).
- Omitting the var _ marshaler.Event compile-time assertion for a new event struct.
- Storing mutable state or pointers in the event struct — it must be a pure value type for safe concurrent marshaling.
- Adding context.Context parameters — event structs are value types; context belongs at the publisher call site.

## Decisions

- **EventVersionSubsystem is exported for consumer subscription.** — Decouples producer/consumer versioning; consumers match on VersionSubsystem prefix, so a version bump (v2 -> v3) is isolated to the producer side.
- **RawEvents embeds the full serializer.CloudEventsKafkaPayload slice rather than aggregated fields.** — Avoids re-querying ClickHouse in the balance-worker; the worker has all raw usage data needed to recalculate grant burn-down in-memory.

## Example: Adding a new ingest notification event type alongside EventBatchedIngest

```
import (
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)
var (
	_ marshaler.Event = EventNewIngest{}
	newIngestEventType = metadata.EventType{Subsystem: EventSubsystem, Name: "events.new", Version: "v1"}
	newIngestEventName = metadata.GetEventName(newIngestEventType)
	NewIngestVersionSubsystem = newIngestEventType.VersionSubsystem()
)
```

<!-- archie:ai-end -->
