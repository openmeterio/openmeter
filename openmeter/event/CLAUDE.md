# event

<!-- archie:ai-start -->

> Organisational namespace for cross-subsystem event primitives shared by every Watermill producer. openmeter/event/metadata owns canonical event naming (io.openmeter.<subsystem>.<version>.<name>), resource-path construction (//openmeter.io/...), and EntityXxx constants; openmeter/event/models holds minimal validated payload value types (FeatureKeyAndID, NamespaceID). Both are leaf packages with no domain imports.

## Patterns

**EventType triple for naming** — Build all event names via metadata.EventType{Subsystem, Version, Name} + metadata.GetEventName; never assemble io.openmeter.* strings by hand. This name is what eventbus prefix-routing keys on. (`metadata.GetEventName(metadata.EventType{Subsystem: MeterEventSubsystem, Name: MeterCreateEventName, Version: "v1"})`)
**Resource path construction** — Build every //openmeter.io/... URI with metadata.ComposeResourcePath(namespace, entityConst, id); never fmt.Sprintf. (`metadata.ComposeResourcePath(meter.Namespace, metadata.EntityMeter, meter.ID)`)
**Entity constant registry** — New entity types are registered as EntityXxx constants in metadata before use in event source/subject fields — never inline string literals in event structs. (`metadata.EntityMeter, metadata.EntityCustomer, metadata.EntitySubscription`)
**Inline Validate() on every model type** — Every exported type in openmeter/event/models implements Validate(), called at event deserialization boundaries to catch missing required fields. (`func (m NamespaceID) Validate() error { if m.ID == "" { return errors.New(...) } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metadata/event_type.go` | EventType, EventSubsystem, EventName, EventMetadata, GetEventName — the canonical naming contract for all events. | New subsystems must follow io.openmeter.<subsystem>.<version>.<name>; version must be explicit (e.g. 'v1'). |
| `metadata/resourcepath.go` | ComposeResourcePath plus all EntityXxx constants for //openmeter.io/... URIs. | Register a new EntityXxx constant before using a new entity in event source/subject. |
| `models/models.go` | Cross-subsystem minimal payload types (FeatureKeyAndID, NamespaceID), each with Validate(). | Never add domain-specific rich types (Invoice, Subscription) — they belong in their own packages and would create cycles. |

## Anti-Patterns

- Building io.openmeter.* names by concatenation instead of EventType + GetEventName.
- Constructing //openmeter.io/... paths with fmt.Sprintf instead of ComposeResourcePath.
- Adding a new entity as an inline string literal instead of an EntityXxx constant in metadata.
- Importing openmeter/* domain packages from openmeter/event/models — creates circular dependencies.
- Skipping Validate() on payload values received from this package.

## Decisions

- **Naming/routing (metadata) and payload value types (models) are kept in separate sub-packages.** — Prevents the naming package from accumulating domain-specific types and keeps both as cycle-free leaves.

<!-- archie:ai-end -->
