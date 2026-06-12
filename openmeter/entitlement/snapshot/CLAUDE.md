# snapshot

<!-- archie:ai-start -->

> Defines the entitlement balance SnapshotEvent (Watermill marshaler.Event, v2) emitted by the balance worker when an entitlement value changes — the payload notification/consumer and downstream workers consume.

## Patterns

**marshaler.Event with versioned name** — SnapshotEvent implements EventName/EventMetadata/Validate; the name is built once via metadata.GetEventName(EventType{Subsystem: entitlement.EventSubsystem, Name: "entitlement.snapshot", Version: "v2"}). (`snapshotEventName = metadata.GetEventName(metadata.EventType{Subsystem: entitlement.EventSubsystem, Name: "entitlement.snapshot", Version: "v2"})`)
**Operation enum gates payload** — ValueOperationType (reset/update/delete) has Values()/Validate(); Validate() requires Value to be non-nil for update/reset, empty for delete. (`case ValueOperationUpdate, ValueOperationReset: if e.Value == nil { errs = append(errs, errors.New("balance is required ...")) }`)
**errors.Join validation** — Validate() accumulates into var errs []error and returns errors.Join(errs...) rather than failing on the first field. (`return errors.Join(errs...)`)
**Constructor derives namespace** — NewSnapshotEvent derives Namespace from ent.Namespace and tolerates a nil deprecated *subject.Subject. (`Namespace: models.NamespaceID{ID: ent.Namespace}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event.go` | ValueOperationType enum, EntitlementValue (balance/overage/usage pointers), SnapshotEvent and NewSnapshotEvent | Subject is deprecated and may be empty (validation intentionally skips it); EventMetadata branches on whether Customer.ID is set; bump the v2 version string if the shape changes |

## Anti-Patterns

- Re-validating/relying on Subject — it is deprecated and may be empty for customers without usage attribution
- Mutating the event shape without bumping the v2 version in snapshotEventName
- Returning on first validation error instead of joining all issues

## Decisions

- **EntitlementValue uses pointer fields for metered-only data** — Balance/Overage/Usage/Config are nil for entitlement types that don't have them, distinguishing 'absent' from zero.

## Example: Validating a snapshot event

```
func (e SnapshotEvent) Validate() error {
	var errs []error
	if err := e.Operation.Validate(); err != nil { errs = append(errs, err) }
	if e.Entitlement.ID == "" { errs = append(errs, errors.New("entitlementId is required")) }
	if err := e.Namespace.Validate(); err != nil { errs = append(errs, err) }
	switch e.Operation {
	case ValueOperationUpdate, ValueOperationReset:
		if e.Value == nil { errs = append(errs, errors.New("balance is required for balance update/reset")) }
	}
	return errors.Join(errs...)
}
```

<!-- archie:ai-end -->
