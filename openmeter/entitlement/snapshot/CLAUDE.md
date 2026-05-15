# snapshot

<!-- archie:ai-start -->

> Defines SnapshotEvent — the versioned Watermill CloudEvent published to the system topic when an entitlement balance changes (reset, update, delete), consumed by the balance-worker and notification-service.

## Patterns

**marshaler.Event implementation with pinned version** — SnapshotEvent implements marshaler.Event (EventName, EventMetadata, Validate). EventName uses metadata.GetEventName with a pinned version string (v2). New snapshot event versions must bump the version and keep old struct for backward compatibility. (`snapshotEventName = metadata.GetEventName(metadata.EventType{Subsystem: entitlement.EventSubsystem, Name: "entitlement.snapshot", Version: "v2"})`)
**NewSnapshotEvent constructor — never struct literals** — Use NewSnapshotEvent() to build SnapshotEvent. It handles the deprecated Subject field safely when subj is nil and ensures all required fields are populated. (`snapshot.NewSnapshotEvent(ent, subj, customer, feat, snapshot.ValueOperationUpdate, &calculatedAt, &value, &currentUsagePeriod)`)
**Validate enforces Value for update/reset operations** — Validate() returns an error if Value is nil for ValueOperationUpdate or ValueOperationReset. Delete operations allow nil Value. Always call Validate() before publishing. (`case ValueOperationUpdate, ValueOperationReset: if e.Value == nil { errs = append(errs, errors.New("balance is required")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event.go` | Defines SnapshotEvent, EntitlementValue (balance/usage/overage/access fields), ValueOperationType enum, and NewSnapshotEvent constructor. | Subject field is deprecated — new consumer code must read Customer.ID instead. NewSnapshotEvent sets Subject to empty Subject{} when subj arg is nil. |

## Anti-Patterns

- Creating SnapshotEvent with struct literals that omit CalculatedAt — Validate() will reject it.
- Bumping the event version in-place instead of adding a new versioned struct alongside the old one.
- Reading Subject field in new consumer code — use Customer instead.
- Publishing SnapshotEvent without calling Validate() first.

## Decisions

- **SnapshotEvent v2 carries both Subject (deprecated) and Customer to maintain backward compatibility.** — The balance-worker and edge workers were originally built around subjects; migrating to customer-centric routing required keeping both fields during the transition period.

<!-- archie:ai-end -->
