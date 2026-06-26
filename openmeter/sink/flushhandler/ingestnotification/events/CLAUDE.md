# events

<!-- archie:ai-start -->

> Defines the Watermill event contract emitted after a batch of usage events is flushed to ClickHouse: EventBatchedIngest, the 'ingest/events.ingested' (v2) message that notifies downstream consumers (notably the balance worker) which subject/meters changed. Its sole constraint is to satisfy the marshaler.Event interface so the eventbus can serialize it as a CloudEvent.

## Patterns

**marshaler.Event implementation** — Every event type must implement marshaler.Event: EventName() string, EventMetadata() metadata.EventMetadata, Validate() error. Assert conformance at compile time with a blank var. (`var _ marshaler.Event = EventBatchedIngest{}`)
**Versioned EventType registration** — Declare a package-level metadata.EventType with Subsystem/Name/Version, then derive the wire name via metadata.GetEventName(...). EventName() returns this derived constant, never a hand-typed string. (`batchIngestEventType = metadata.EventType{Subsystem: EventSubsystem, Name: "events.ingested", Version: "v2"}; batchIngestEventName = metadata.GetEventName(batchIngestEventType)`)
**EventMetadata via metadata.ComposeResourcePath** — Build Source/Subject from the typed helpers metadata.ComposeResourcePathRaw / ComposeResourcePath with metadata.EntitySubjectKey, not raw string concatenation, so resource paths stay namespace-scoped and consistent. (`Subject: metadata.ComposeResourcePath(b.Namespace.ID, metadata.EntitySubjectKey, b.SubjectKey)`)
**Slug-based meter identity (no IDs)** — Reference meters by MeterSlugs ([]string), never meter IDs. Meter IDs are not present in the open-source build, so event payloads consumed by OSS code must not depend on them. (`MeterSlugs []string `json:"meterSlugs"``)
**Defensive Validate()** — Validate() delegates to Namespace.Validate() first, then checks required scalar fields (SubjectKey non-empty), returning a plain error on the first failure. (`if b.SubjectKey == "" { return errors.New("subjectKey must be set") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `events.go` | Declares EventSubsystem ('ingest'), the EventBatchedIngest payload struct, its versioned EventType, and the three marshaler.Event methods. Also exports EventVersionSubsystem for consumer subscription routing. | Changing struct fields or the Version (currently 'v2') is a wire-format break for the balance worker and any subscriber filtering on EventVersionSubsystem — bump the Version and coordinate consumers rather than mutating in place. RawEvents carries serializer.CloudEventsKafkaPayload and StoredAt carries the flush cutoff timestamp consumers use; do not repurpose. |

## Anti-Patterns

- Hand-coding the event name string instead of deriving it from the metadata.EventType via metadata.GetEventName.
- Adding meter IDs (or any OSS-absent identifier) to the payload; consumers in open-source must work with MeterSlugs only.
- Mutating existing JSON fields or the Version without bumping it — silently breaks balanceworker deserialization.
- Building Source/Subject strings by manual concatenation instead of metadata.ComposeResourcePath helpers.
- Adding business logic, Kafka publishing, or persistence here — this package is a pure event contract; producers live in ingestnotification, consumers in entitlement/balanceworker.

## Decisions

- **Carry MeterSlugs rather than meter IDs in the payload.** — Meter IDs do not exist in the open-source version, so any OSS consumer (e.g. balanceworker) must be able to act on the event without them.
- **Embed an explicit versioned EventType ('v2') and expose EventVersionSubsystem.** — Lets the eventbus/consumers subscribe and route by version+subsystem, enabling backward-incompatible payload evolution via version bumps instead of in-place mutation.

## Example: Defining a new flush-notification event that the eventbus can marshal as a CloudEvent

```
package events

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

var (
	_ marshaler.Event = EventBatchedIngest{}

	batchIngestEventType = metadata.EventType{Subsystem: EventSubsystem, Name: "events.ingested", Version: "v2"}
	batchIngestEventName = metadata.GetEventName(batchIngestEventType)
// ...
```

<!-- archie:ai-end -->
