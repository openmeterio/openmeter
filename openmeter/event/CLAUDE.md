# event

<!-- archie:ai-start -->

> Organisational namespace for cross-subsystem event primitives. openmeter/event/metadata owns canonical event naming (io.openmeter.<subsystem>.<version>.<name>), resource path construction (//openmeter.io/...), and entity type constants. openmeter/event/models holds minimal shared payload value types (FeatureKeyAndID, NamespaceID) used in event payloads across domain boundaries.

## Patterns

**EventType triple for naming** — All event names are constructed via metadata.EventType{Subsystem, Version, Name} and metadata.GetEventName(); never build io.openmeter.* strings manually. (`metadata.GetEventName(metadata.EventType{Subsystem: MeterEventSubsystem, Name: MeterCreateEventName, Version: "v1"})`)
**Resource path construction** — All //openmeter.io/... URIs must be built with metadata.ComposeResourcePath(namespace, entityConst, id); never use fmt.Sprintf. (`metadata.ComposeResourcePath(meter.Namespace, metadata.EntityMeter, meter.ID)`)
**Entity constant registry in metadata** — New entity types must be added as EntityXxx constants in openmeter/event/metadata, not inlined as strings in event structs. (`metadata.EntityMeter, metadata.EntityCustomer, metadata.EntitySubscription`)
**Inline Validate() on every model** — Every type in openmeter/event/models must implement Validate() — called at event deserialization boundaries to catch missing required fields. (`func (m NamespaceID) Validate() error { if m.ID == "" { return errors.New(...) } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/event/metadata/event_type.go` | Defines EventType, EventSubsystem, EventName, EventMetadata, and GetEventName — the canonical naming contract for all events. | Adding new subsystems must follow the io.openmeter.<subsystem>.<version>.<name> pattern; version must be explicit (e.g. 'v1'). |
| `openmeter/event/metadata/resourcepath.go` | Defines ComposeResourcePath and all EntityXxx constants for constructing //openmeter.io/... URIs. | New domain entities must register an EntityXxx constant here before being used in event source/subject fields. |
| `openmeter/event/models/models.go` | Holds cross-subsystem minimal payload types (FeatureKeyAndID, NamespaceID). Must not import domain packages to avoid circular dependencies. | Never add domain-specific rich types (Invoice, Subscription) here — they belong in their own domain packages. |

## Anti-Patterns

- Building io.openmeter.* event name strings with string concatenation instead of EventType.EventName().
- Constructing //openmeter.io/... resource paths with fmt.Sprintf instead of ComposeResourcePath.
- Adding a new entity type as an inline string literal in an event struct instead of an EntityXxx constant in metadata.
- Importing openmeter domain packages from openmeter/event/models — creates circular dependencies.
- Skipping Validate() calls on values received in event payloads — missing required fields go undetected.

## Decisions

- **Separate metadata and models sub-packages** — metadata owns naming/routing primitives while models holds cross-domain payload types; keeping them separate prevents the naming package from accumulating domain-specific types.

<!-- archie:ai-end -->
