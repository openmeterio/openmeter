# snapshot

<!-- archie:ai-start -->

> Defines SnapshotEvent — the versioned Watermill CloudEvent (entitlement.snapshot v2) published to the system topic on entitlement balance changes (reset, update, delete), consumed by the balance-worker and notification-service.

## Patterns

**marshaler.Event with pinned version** — SnapshotEvent implements marshaler.Event (EventName, EventMetadata, Validate); EventName uses metadata.GetEventName pinned to v2. New versions bump the version and keep the old struct. (`snapshotEventName = metadata.GetEventName(metadata.EventType{Subsystem: entitlement.EventSubsystem, Name: "entitlement.snapshot", Version: "v2"})`)
**NewSnapshotEvent constructor — no struct literals** — Use NewSnapshotEvent() which handles a nil Subject safely and populates Namespace from the entitlement. (`snapshot.NewSnapshotEvent(ent, subj, customer, feat, snapshot.ValueOperationUpdate, &calculatedAt, &value, &currentUsagePeriod)`)
**Validate enforces Value for update/reset** — Validate() errors if Value is nil for ValueOperationUpdate or ValueOperationReset; delete allows nil Value. CalculatedAt is always required. Call before publishing. (`case ValueOperationUpdate, ValueOperationReset: if e.Value == nil { errs = append(errs, errors.New("balance is required")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event.go` | SnapshotEvent, EntitlementValue (balance/usage/overage/access/config), ValueOperationType enum, NewSnapshotEvent constructor. | Subject field is deprecated — new consumer code reads Customer.ID. NewSnapshotEvent sets an empty Subject{} when subj is nil. |

## Anti-Patterns

- Creating SnapshotEvent struct literals that omit CalculatedAt — Validate() rejects it.
- Bumping the event version in place instead of adding a new versioned struct alongside the old.
- Reading the Subject field in new consumer code — use Customer.
- Publishing SnapshotEvent without calling Validate() first.

## Decisions

- **SnapshotEvent v2 carries both deprecated Subject and Customer.** — Edge workers were originally subject-based; migrating to customer-centric routing requires both fields during the transition.

<!-- archie:ai-end -->
