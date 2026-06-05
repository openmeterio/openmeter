# events

<!-- archie:ai-start -->

> Defines the Watermill event contract for the balance worker's recalculation pipeline. The single file declares RecalculateEvent, which triggers an entitlement balance snapshot recomputation as-of a timestamp, plus the OperationType enum describing what caused the trigger.

## Patterns

**marshaler.Event compile-time assertion** — Every event type asserts interface conformance with a blank var so the compiler enforces EventName()/EventMetadata() at build time. (`var _ marshaler.Event = RecalculateEvent{}`)
**Versioned EventType triple** — Event identity is a metadata.EventType{Subsystem, Name, Version}; EventSubsystem='balanceWorker', RecalculateEventName='triggerEntitlementRecalculation', Version='v2'. The wire name is derived via metadata.GetEventName, never hardcoded. (`recalculateEventName = metadata.GetEventName(recalculateEventType)`)
**String enum with Values()+Validate()** — OperationType is a string type whose Values() lists every valid constant and Validate() checks membership via slices.Contains. New operations MUST be added to both the const block and Values(). (`func (o OperationType) Validate() error { if !slices.Contains(o.Values(), o) {...} }`)
**Error-collecting Validate()** — Validate() accumulates into var errs []error and returns errors.Join(errs...), wrapping nested validators with field context (fmt.Errorf("entitlement: %w", err)). (`errs = append(errs, fmt.Errorf("sourceOperation: %w", err)); return errors.Join(errs...)`)
**Subject path via metadata.ComposeResourcePath** — EventMetadata().Subject is built with metadata.ComposeResourcePath(namespace, metadata.EntityEntitlement, id) rather than string concatenation. (`Subject: metadata.ComposeResourcePath(e.Entitlement.Namespace, metadata.EntityEntitlement, e.Entitlement.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `recalculate.go` | Declares RecalculateEvent (Entitlement NamespacedID, AsOf, OriginalEventSource, SourceOperation, RawIngestedEvents) and the OperationType enum covering entitlement/grant lifecycle + ingest/recalculate triggers. | RawIngestedEvents carries serializer.CloudEventsKafkaPayload from kafkaingest; the event Source comes from OriginalEventSource so downstream consumers can trace the originating producer. Bumping the payload shape requires bumping recalculateEventType.Version (currently 'v2'). |

## Anti-Patterns

- Adding an OperationType constant without also adding it to Values() (Validate would reject it).
- Hardcoding the event name string instead of deriving it from metadata.GetEventName(recalculateEventType).
- Returning on the first validation failure instead of collecting into errs and errors.Join.
- Changing RecalculateEvent's JSON shape without incrementing the EventType Version, breaking deserialization of in-flight messages.

## Decisions

- **Event is at Version 'v2' and exposes EventVersionSubsystem.** — Schema has already evolved once; consumers route by version+subsystem so old and new payloads can coexist on the topic during rollout.
- **OriginalEventSource is carried on the event and surfaced as the metadata Source.** — Recalculation is triggered by many upstream operations (ingest, grant void, reset); preserving the original source keeps event provenance intact through the worker.

## Example: Declaring a versioned Watermill event with compile-time conformance and joined validation

```
var _ marshaler.Event = RecalculateEvent{}

var recalculateEventType = metadata.EventType{
	Subsystem: EventSubsystem,
	Name:      RecalculateEventName,
	Version:   "v2",
}
var recalculateEventName = metadata.GetEventName(recalculateEventType)

func (e RecalculateEvent) EventName() string { return recalculateEventName }

func (e RecalculateEvent) Validate() error {
	var errs []error
	if e.AsOf.IsZero() {
		errs = append(errs, errors.New("asOf is required"))
// ...
```

<!-- archie:ai-end -->
