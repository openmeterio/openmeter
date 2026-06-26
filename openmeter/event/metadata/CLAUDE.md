# metadata

<!-- archie:ai-start -->

> Defines the canonical CloudEvents-style identity for every domain event in OpenMeter: the structured EventType (subsystem/version/name) that produces wire event names like `io.openmeter.<subsystem>.<version>.<name>`, plus the resource-path scheme used to build CloudEvents `source`/`subject` URIs. It is the shared vocabulary imported by nearly every event-emitting domain package, so its conventions are load-bearing across the whole event bus.

## Patterns

**EventType identity triple** — Every event declares an EventType with Subsystem, Name, and Version; the wire name is derived, never hand-written. Compute names via EventName()/GetEventName(spec) rather than string-formatting elsewhere. (`var grantCreatedType = metadata.EventType{Subsystem: "credit", Name: "grant.created", Version: "v1"}; name := metadata.GetEventName(grantCreatedType)`)
**Resource paths via Compose helpers** — CloudEvents source/subject URIs are always built with ComposeResourcePath(namespace, items...) (which prepends `namespace/<ns>`) or ComposeResourcePathRaw. Never concatenate `//openmeter.io/...` by hand. (`source := metadata.ComposeResourcePath(ns, metadata.EntityEntitlement, entitlementID, metadata.EntityGrant, grantID)`)
**Entity-segment constants** — Path segments use the exported Entity* constants (EntityEntitlement, EntityInvoice, EntitySubscription, EntityGrant, EntityEvent, ...). Add a new constant here rather than inlining a literal segment string at a call site. (`metadata.ComposeResourcePath(ns, metadata.EntityCustomer, customerID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event_type.go` | EventType struct + EventName()/VersionSubsystem()/GetEventName() name builders, and the EventMetadata envelope (ID, Time, Subject, Source). String type aliases EventSubsystem/EventName/EventVersion enforce intent at field level. | EventName format `io.openmeter.<subsystem>.<version>.<name>` is the consumed contract — changing it renames every event and breaks Kafka topic routing/consumers. The doc comment on EventMetadata.Subject/Source is the authoritative example of correct path shapes; keep source/subject consistent with it. |
| `resourcepath.go` | Entity type constants and ComposeResourcePath/ComposeResourcePathRaw builders for `//openmeter.io/...` URIs. | ComposeResourcePath auto-injects the `namespace/<ns>` prefix; passing your own `namespace` segment double-prefixes. Use the Raw variant only when you intentionally need a non-namespace-rooted path (e.g. EntityEvent ingestion source). |

## Anti-Patterns

- Hand-formatting `io.openmeter.*` event names or `//openmeter.io/*` resource paths instead of using EventName()/ComposeResourcePath.
- Adding a new path segment as an inline string literal instead of an Entity* constant.
- Changing the EventName/VersionSubsystem format strings — they are wire contracts consumed by event subscribers.
- Adding dependencies or business logic here; this package must stay a leaf with only fmt/time/strings imports.

## Decisions

- **Event identity is a structured triple (Subsystem/Version/Name) with a derived name rather than free-form strings.** — Versioning is first-class (VersionSubsystem()) so payload schemas can evolve per subsystem, and a single derivation point keeps all emitters/consumers in sync.
- **Keep metadata as a tiny dependency-free leaf package.** — It is imported by ~20 domain packages; any heavier dependency here would create wide coupling and import cycles.

## Example: Build an event name and its CloudEvents source/subject for a grant event

```
import "github.com/openmeterio/openmeter/openmeter/event/metadata"

et := metadata.EventType{Subsystem: "credit", Name: "grant.created", Version: "v1"}
name := metadata.GetEventName(et) // io.openmeter.credit.v1.grant.created
meta := metadata.EventMetadata{
    Source:  metadata.ComposeResourcePath(ns, metadata.EntityEntitlement, entID, metadata.EntityGrant, grantID),
    Subject: metadata.ComposeResourcePath(ns, "subject", subjectID),
}
```

<!-- archie:ai-end -->
