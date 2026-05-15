# events

<!-- archie:ai-start -->

> Pure schema package defining RecalculateEvent — the Watermill event contract between balance recalculation producers (entitlement lifecycle hooks, ingest flush) and the balance-worker consumer. No business logic, no DB access.

## Patterns

**Implement marshaler.Event with compile-time assertion** — Every event struct must implement marshaler.Event (EventName() string, EventMetadata() metadata.EventMetadata). Enforce with var _ marshaler.Event = RecalculateEvent{}. (`var _ marshaler.Event = RecalculateEvent{}`)
**Versioned EventType via metadata.EventType struct** — Event type is declared as metadata.EventType{Subsystem, Name, Version}; the event name string is derived via metadata.GetEventName(). Bump Version on breaking schema changes — never rename the Go type instead. (`recalculateEventType = metadata.EventType{Subsystem: EventSubsystem, Name: RecalculateEventName, Version: "v2"}`)
**OperationType enum with Values() + Validate()** — New operation kinds must be added to both the const block and the Values() slice. Validate() uses slices.Contains(o.Values(), o) — omitting an entry from Values() makes it always invalid. (`func (o OperationType) Validate() error { if !slices.Contains(o.Values(), o) { return fmt.Errorf(...) } return nil }`)
**EventMetadata.Subject via ComposeResourcePath** — Subject must always be built with metadata.ComposeResourcePath(namespace, entityKind, id) — never raw string concatenation. (`Subject: metadata.ComposeResourcePath(e.Entitlement.Namespace, metadata.EntityEntitlement, e.Entitlement.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `recalculate.go` | Single source of truth for the recalculate event schema: struct fields, OperationType enum, and all marshaler.Event plumbing. | Adding fields is a wire-format change — consumers must be updated simultaneously. Version string in recalculateEventType must be bumped on breaking changes. EventVersionSubsystem is exported from this file and used by eventbus routing — do not rename. |

## Anti-Patterns

- Adding DB calls or business logic — this is a pure schema/data package
- Constructing RecalculateEvent without calling Validate() before publishing
- Hard-coding the event name string instead of using metadata.GetEventName(recalculateEventType)
- Introducing a new OperationType constant without adding it to Values()
- Renaming EventVersionSubsystem — it is the routing key used by eventbus.GeneratePublishTopic

## Decisions

- **Event schema lives in events/ sub-package rather than inline in the worker** — Producers outside the worker (entitlement service hooks, ingest flush) import this package to construct events without depending on the full worker package, avoiding circular imports.
- **Version encoded in EventType struct ('v2'), not in the Go type name** — Allows multiple consumers running different versions to coexist on the same Kafka topic during rolling deploys by filtering on EventVersionSubsystem.

## Example: Add a new OperationType and publish a valid RecalculateEvent

```
// 1. Add to const block and Values() in recalculate.go:
const OperationTypeMyNew OperationType = "my_new"
func (o OperationType) Values() []OperationType {
    return []OperationType{..., OperationTypeMyNew}
}

// 2. Construct and validate before publish:
evt := events.RecalculateEvent{
    Entitlement:         models.NamespacedID{Namespace: ns, ID: entID},
    AsOf:                time.Now(),
    OriginalEventSource: "my-producer",
    SourceOperation:     events.OperationTypeMyNew,
}
if err := evt.Validate(); err != nil {
    return fmt.Errorf("invalid event: %w", err)
// ...
```

<!-- archie:ai-end -->
