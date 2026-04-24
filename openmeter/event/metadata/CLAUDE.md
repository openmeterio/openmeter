# metadata

<!-- archie:ai-start -->

> Defines shared event identity primitives: the canonical event name format (`io.openmeter.<subsystem>.<version>.<name>`), event metadata (ID, time, source, subject), and resource path helpers for constructing `//openmeter.io/...` URIs. All event payloads across the system reference these types for consistent naming and routing.

## Patterns

**EventType triple (Subsystem/Version/Name)** — Every event kind is declared as an `EventType{Subsystem, Version, Name}` value. The canonical string form is produced by `EventType.EventName()` — never build the string manually. (`EventType{Subsystem: "ingest", Version: "v1", Name: "flush"}.EventName() // "io.openmeter.ingest.v1.flush"`)
**Resource paths via ComposeResourcePath** — All `source` / `subject` fields in `EventMetadata` must be built with `ComposeResourcePath(namespace, entityType, id)` or `ComposeResourcePathRaw(...)`. Never construct `//openmeter.io/...` paths by hand. (`ComposeResourcePath(ns, EntityEntitlement, entID) // "//openmeter.io/namespace/<ns>/entitlement/<id>"`)
**Entity constant registry** — All entity type strings (e.g. `EntityInvoice`, `EntityCustomer`) are declared as package-level constants. New entity types must be added here, not as inline strings in event producers. (`const EntitySubscriptionAddon = "subscriptionAddon"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event_type.go` | Canonical event naming: EventType struct + EventName()/VersionSubsystem() formatters, plus EventMetadata for ID/time/source/subject. | Do not bypass EventType.EventName() to build the name string; version is the second segment, not the third — the format is `<subsystem>.<version>.<name>`. |
| `resourcepath.go` | Entity-type constants and ComposeResourcePath/ComposeResourcePathRaw helpers for all `//openmeter.io/...` URIs. | New domain entities need a matching `Entity*` constant added here; missing constants cause inconsistent resource paths across event sources. |

## Anti-Patterns

- Building `io.openmeter.*` event name strings by hand instead of using EventType.EventName()
- Constructing `//openmeter.io/...` paths with fmt.Sprintf instead of ComposeResourcePath
- Adding a new entity type as an inline string literal in an event struct rather than adding an EntityXxx constant here

## Decisions

- **EventName format encodes subsystem + version + name in a fixed dot-separated URI** — Watermill consumers route and filter events by name prefix; a deterministic format prevents routing mismatches across worker binaries.
- **Resource paths follow `//openmeter.io/namespace/<ns>/<entity>/<id>` convention** — CloudEvents spec requires unambiguous source/subject URIs; the `//openmeter.io/` authority prefix scopes all paths to this system globally.

## Example: Declaring a new event type and building its metadata

```
import "github.com/openmeterio/openmeter/openmeter/event/metadata"

var MyEventType = metadata.EventType{
    Subsystem: "billing",
    Version:   "v1",
    Name:      "invoiceCreated",
}

meta := metadata.EventMetadata{
    ID:      ulid.Make().String(),
    Time:    time.Now(),
    Source:  metadata.ComposeResourcePath(ns, metadata.EntityInvoice, invoiceID),
    Subject: metadata.ComposeResourcePath(ns, metadata.EntityCustomer, customerID),
}
```

<!-- archie:ai-end -->
