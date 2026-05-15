# metadata

<!-- archie:ai-start -->

> Defines canonical event identity primitives for the entire system: the EventType triple (Subsystem/Version/Name) that produces deterministic 'io.openmeter.<subsystem>.<version>.<name>' strings, EventMetadata for ID/time/source/subject fields, and ComposeResourcePath helpers for '//openmeter.io/...' URIs. All Watermill event producers across binaries depend on these types for consistent routing.

## Patterns

**EventType triple declaration** — Every new event kind is declared as a package-level var EventType{Subsystem, Version, Name}. The canonical string is always obtained via EventType.EventName() — never via fmt.Sprintf. VersionSubsystem() returns the two-segment prefix used by eventbus.GeneratePublishTopic for Kafka routing. (`var MyEventType = metadata.EventType{Subsystem: "billing", Version: "v1", Name: "invoiceCreated"}
// canonical name: "io.openmeter.billing.v1.invoiceCreated"
// routing prefix: "io.openmeter.billing.v1"`)
**Resource paths via ComposeResourcePath** — All source/subject fields in EventMetadata must use ComposeResourcePath(namespace, entityType..., id) or ComposeResourcePathRaw(). The authority prefix '//openmeter.io/' is injected automatically. Never use fmt.Sprintf to build these URIs. (`source := metadata.ComposeResourcePath(ns, metadata.EntityInvoice, invoiceID)
// result: "//openmeter.io/namespace/<ns>/invoice/<invoiceID>"`)
**Entity constant registry** — All entity type strings (EntityInvoice, EntityCustomer, EntityEntitlement, etc.) are declared as package-level string constants in resourcepath.go. New domain entities require a matching EntityXxx constant added here before use in any ComposeResourcePath call. (`const EntitySubscriptionAddon = "subscriptionAddon"
// Then use: metadata.ComposeResourcePath(ns, metadata.EntitySubscriptionAddon, addonID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event_type.go` | Declares EventType struct with EventName() ('io.openmeter.<subsystem>.<version>.<name>') and VersionSubsystem() ('io.openmeter.<subsystem>.<version>') formatters, plus EventMetadata for ID/time/source/subject fields on every event. | The format is subsystem.version.name — version is the SECOND segment, not the third. Never build the event name string manually; routing in eventbus.GeneratePublishTopic relies on the VersionSubsystem() prefix matching exactly. |
| `resourcepath.go` | Defines all EntityXxx constants and the ComposeResourcePath / ComposeResourcePathRaw helpers that produce '//openmeter.io/...' URIs used in CloudEvents source and subject fields. | Adding a new domain entity requires a new EntityXxx constant here. Missing constants lead to inline string literals in event producers, causing inconsistent resource paths that break audit and correlation across binaries. |

## Anti-Patterns

- Building 'io.openmeter.*' event name strings with fmt.Sprintf instead of EventType.EventName()
- Constructing '//openmeter.io/...' paths with fmt.Sprintf instead of ComposeResourcePath
- Adding a new entity type as an inline string literal in an event struct rather than a new EntityXxx constant in resourcepath.go
- Declaring EventType values with the version in the wrong position (format is subsystem.version.name, not subsystem.name.version)

## Decisions

- **EventName format encodes subsystem + version + name in a fixed dot-separated URI under 'io.openmeter.'** — Watermill's eventbus.GeneratePublishTopic routes events to the correct Kafka topic by matching the EventName() prefix against known EventVersionSubsystem constants. A deterministic, centrally-defined format prevents routing mismatches across independently-deployed worker binaries.
- **Resource paths follow '//openmeter.io/namespace/<ns>/<entity>/<id>' convention via ComposeResourcePath** — CloudEvents spec requires unambiguous source/subject URIs. The '//openmeter.io/' authority prefix scopes all paths to this system globally and enables consistent correlation across audit logs, traces, and webhook payloads.

## Example: Declaring a new event type and building its EventMetadata

```
import "github.com/openmeterio/openmeter/openmeter/event/metadata"

var InvoiceCreatedEventType = metadata.EventType{
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
// Routing prefix for eventbus: InvoiceCreatedEventType.VersionSubsystem()
```

<!-- archie:ai-end -->
