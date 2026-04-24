# events

<!-- archie:ai-start -->

> Defines the Watermill event type for triggering entitlement balance recalculation (RecalculateEvent). Single file: recalculate.go. This is the schema contract between producers (entitlement lifecycle handlers, ingest flush) and the balance worker consumer.

## Patterns

**Implement marshaler.Event interface** — Every event struct must implement marshaler.Event via EventName() string and EventMetadata() metadata.EventMetadata. Verified by compile-time assertion: `var _ marshaler.Event = RecalculateEvent{}`. (`var _ marshaler.Event = RecalculateEvent{}`)
**Versioned event type via metadata.EventType** — Event type is declared as metadata.EventType{Subsystem, Name, Version} and the string name is derived via metadata.GetEventName(). Version is a 'v2'-style string. Bumping version is the migration path for breaking schema changes. (`recalculateEventType = metadata.EventType{Subsystem: EventSubsystem, Name: RecalculateEventName, Version: "v2"}`)
**OperationType enum with Values()+Validate()** — New operation kinds must be added to the OperationType const block AND to the Values() slice so Validate() rejects unknown values via slices.Contains. (`OperationTypeIngest OperationType = "ingest"`)
**EventMetadata subject uses ComposeResourcePath** — EventMetadata.Subject is always built with metadata.ComposeResourcePath(namespace, entityKind, id) — never a raw string concatenation. (`Subject: metadata.ComposeResourcePath(e.Entitlement.Namespace, metadata.EntityEntitlement, e.Entitlement.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `recalculate.go` | Single source of truth for the recalculate event schema. Contains event struct, operation type enum, and all marshaler.Event plumbing. | Adding fields to RecalculateEvent is a wire-format change — consumers must be updated before or simultaneously. Version string in recalculateEventType must be bumped on breaking changes. |

## Anti-Patterns

- Adding business logic or DB calls into this package — it is a pure data/schema package
- Bypassing the Validate() method when constructing RecalculateEvent before publishing
- Hard-coding the event name string instead of using recalculateEventName derived from metadata.GetEventName()
- Introducing a new OperationType constant without adding it to Values()

## Decisions

- **Event schema lives in a dedicated sub-package (events/) rather than inline in the worker** — Producers outside the worker (e.g., entitlement service hooks, ingest flush handlers) import this package to construct events without depending on the full worker.
- **Version is encoded in the EventType struct ('v2'), not in the Go type name** — Allows multiple consumers running different versions to coexist on the same topic during rolling deploys by filtering on EventVersionSubsystem.

<!-- archie:ai-end -->
