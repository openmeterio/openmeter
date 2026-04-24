# snapshot

<!-- archie:ai-start -->

> Defines SnapshotEvent — the versioned Watermill event published to the system topic when an entitlement balance changes (reset, update, delete). Consumed by the balance-worker and notification-service.

## Patterns

**marshaler.Event implementation** — SnapshotEvent implements marshaler.Event (EventName, EventMetadata, Validate). EventName uses metadata.GetEventName with a pinned version string (v2). New snapshot event versions must bump the version and keep old versions for backward compatibility. (`snapshotEventName = metadata.GetEventName(metadata.EventType{Subsystem: entitlement.EventSubsystem, Name: "entitlement.snapshot", Version: "v2"})`)
**NewSnapshotEvent constructor** — Use NewSnapshotEvent() to build SnapshotEvent rather than struct literals — it handles the deprecated Subject field safely when subj is nil. (`snapshot.NewSnapshotEvent(ent, subj, customer, feat, snapshot.ValueOperationUpdate, &calculatedAt, &value, &currentUsagePeriod)`)
**Validate enforces Value for update/reset operations** — Validate() returns an error if Value is nil for ValueOperationUpdate or ValueOperationReset. Delete operations allow nil Value. (`case ValueOperationUpdate, ValueOperationReset: if e.Value == nil { errs = append(errs, errors.New("balance is required for balance update/reset")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event.go` | Defines SnapshotEvent, EntitlementValue (balance/usage/overage/access fields), ValueOperationType enum, and NewSnapshotEvent constructor. | Subject field is deprecated — new consumers should use Customer.ID. Set Subject to empty Subject{} if no usage attribution. |

## Anti-Patterns

- Creating SnapshotEvent with struct literals that omit CalculatedAt — Validate() will reject it.
- Bumping the event version in-place instead of adding a new versioned struct.
- Reading Subject field in new consumer code — use Customer instead.

## Decisions

- **SnapshotEvent v2 carries both Subject (deprecated) and Customer to maintain backward compatibility with existing consumers.** — The balance-worker and edge workers were originally built around subjects; migrating them to customer-centric routing required keeping both fields during the transition.

<!-- archie:ai-end -->
