# metadata

<!-- archie:ai-start -->

> Canonical event-identity primitives for the whole system: the EventType triple (Subsystem/Version/Name) producing deterministic 'io.openmeter.<subsystem>.<version>.<name>' strings, EventMetadata (ID/time/source/subject), and ComposeResourcePath helpers for '//openmeter.io/...' URIs. Every Watermill producer depends on these for consistent Kafka routing.

## Patterns

**EventType triple declaration** — Each event kind is a package-level var EventType{Subsystem, Version, Name}; get the canonical string via EventName(), never fmt.Sprintf. VersionSubsystem() returns the two-segment routing prefix used by eventbus.GeneratePublishTopic. (`var MyEventType = metadata.EventType{Subsystem: "billing", Version: "v1", Name: "invoiceCreated"} // io.openmeter.billing.v1.invoiceCreated`)
**Resource paths via ComposeResourcePath** — source/subject fields must use ComposeResourcePath(namespace, entityType..., id) (or Raw); the '//openmeter.io/' authority is injected automatically. Never build URIs with fmt.Sprintf. (`source := metadata.ComposeResourcePath(ns, metadata.EntityInvoice, invoiceID)`)
**Entity constant registry** — Entity type strings (EntityInvoice, EntityCustomer, ...) are package-level constants in resourcepath.go; a new domain entity needs a matching EntityXxx constant before any ComposeResourcePath use. (`const EntitySubscriptionAddon = "subscriptionAddon"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event_type.go` | EventType with EventName() and VersionSubsystem() formatters, plus EventMetadata for ID/time/source/subject. | Format is subsystem.version.name — version is the SECOND segment; routing in eventbus.GeneratePublishTopic relies on the VersionSubsystem() prefix matching exactly. |
| `resourcepath.go` | All EntityXxx constants and ComposeResourcePath/ComposeResourcePathRaw producing '//openmeter.io/...' URIs. | A new entity requires a new constant; missing one leads to inline literals and inconsistent paths breaking audit/correlation across binaries. |

## Anti-Patterns

- Building 'io.openmeter.*' event names with fmt.Sprintf instead of EventType.EventName().
- Constructing '//openmeter.io/...' paths with fmt.Sprintf instead of ComposeResourcePath.
- Adding a new entity type as an inline literal in an event struct instead of an EntityXxx constant.
- Declaring EventType with version in the wrong position (it is subsystem.version.name).

## Decisions

- **EventName encodes subsystem+version+name in a fixed dot-separated URI under 'io.openmeter.'.** — eventbus.GeneratePublishTopic routes by matching the EventName() prefix to known EventVersionSubsystem constants; a central deterministic format prevents routing mismatches across worker binaries.
- **Resource paths follow '//openmeter.io/namespace/<ns>/<entity>/<id>' via ComposeResourcePath.** — CloudEvents requires unambiguous source/subject URIs; the authority prefix scopes paths globally and enables correlation across audit logs, traces, and webhooks.

## Example: Declare a new event type and build its EventMetadata

```
import "github.com/openmeterio/openmeter/openmeter/event/metadata"

var InvoiceCreatedEventType = metadata.EventType{Subsystem: "billing", Version: "v1", Name: "invoiceCreated"}

meta := metadata.EventMetadata{
    ID:      ulid.Make().String(),
    Time:    time.Now(),
    Source:  metadata.ComposeResourcePath(ns, metadata.EntityInvoice, invoiceID),
    Subject: metadata.ComposeResourcePath(ns, metadata.EntityCustomer, customerID),
}
// Routing prefix: InvoiceCreatedEventType.VersionSubsystem()
```

<!-- archie:ai-end -->
