# grant

<!-- archie:ai-start -->

> Core domain types for the credit grant subsystem: the Grant value type, OwnerConnector interface, Repo interface, expiration/rollover logic, and Watermill event types for grant lifecycle. This is the contract layer that adapter, engine, and service layers all depend on — no Ent or HTTP dependencies.

## Patterns

**Grant.GetEffectivePeriod() / ActiveAt() as canonical activity window** — GetEffectivePeriod() returns a StartBoundedPeriod bounded by the earliest of ExpiresAt, DeletedAt, and VoidedAt. Grant.ActiveAt(t) delegates to this. Always use ActiveAt or GetEffectivePeriod rather than comparing fields directly. (`grant.ActiveAt(phase.from) // correct; do not compare grant.ExpiresAt < phase.from directly`)
**Repo embeds TxCreator + TxUser[Repo] for transaction propagation** — grant.Repo interface requires both entutils.TxCreator and entutils.TxUser[Repo] so callers can propagate transactions. Any implementation must satisfy all methods including Tx(), WithTx(), and Self(). (`type Repo interface { CreateGrant(...) ...; entutils.TxCreator; entutils.TxUser[Repo] }`)
**Event versioning: grantEventV2 literal + ToDomainGrant conversion** — Watermill events use a pinned literal struct (grantEventV2GrantLiteral) that mirrors Grant fields using only primitives. ToDomainGrant() converts back to Grant. New event versions must add a new versioned struct with a new name suffix, never modify existing structs. (`type CreatedEventV2 grantEventV2; func (e CreatedEventV2) EventName() string { return grantCreatedEventNameV2 }`)
**ExpiresAt pre-computed and stored at creation time** — Grant.ExpiresAt is set from grant.GetExpiration() at creation time (in RepoCreateInput.ExpiresAt). Query predicates use the stored ExpiresAt rather than computing from Expiration+EffectiveAt at query time. (`// At creation: RepoCreateInput.ExpiresAt = grant.GetExpiration(grant.EffectiveAt)`)
**RolloverBalance vs RecurrenceBalance — distinct semantics** — RolloverBalance clamps the balance to [ResetMinRollover, ResetMaxRollover]. RecurrenceBalance always resets to Amount (recurring grants do not roll over). Do not conflate the two. (`rolledOver[grantID] = grant.RolloverBalance(grantBalance) // for reset; for recurrence: grant.RecurrenceBalance()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | Grant struct, GetEffectivePeriod, ActiveAt, RolloverBalance, RecurrenceBalance, GetExpiration. | RolloverBalance clamps to [ResetMinRollover, ResetMaxRollover]; RecurrenceBalance always resets to Amount. Do not conflate the two. GetEffectivePeriod returns the first non-nil of ExpiresAt, DeletedAt, VoidedAt as the upper bound. |
| `owner_connector.go` | OwnerConnector interface (DescribeOwner, GetResetTimelineInclusive, GetUsagePeriodStartAt, etc.), Owner value type, ResetBehavior, OwnerNotFoundError. | GetResetTimelineInclusive returns the start of the period that contains each time point, not just the resets in the period — the first returned time may precede period.From. |
| `repo.go` | Repo interface, ListParams, RepoCreateInput, OrderBy constants. | Limit/Offset fields in ListParams are marked deprecated — prefer Page-based pagination in new code. OrderByDefault is OrderByCreatedAt. |
| `expiration.go` | ExpirationPeriod struct and GetExpiration(t) calculation for all duration types (day, month, year). | Unknown duration values return time.Time{} (zero value) silently — validate ExpirationPeriodDuration before persisting. |
| `events.go` | v1 Watermill events (CreatedEvent, VoidedEvent) — deprecated, embeds Grant directly in the event struct. | These v1 events embed the domain Grant struct which makes them fragile to domain changes. New events must use the v2 literal pattern in events_2.go. |
| `events_2.go` | v2 Watermill events using grantEventV2GrantLiteral (primitive fields only) + ToDomainGrant() conversion. | When adding a new grant field that must be in events, add it to grantEventV2GrantLiteral AND update ToDomainGrant(). Create a new versioned struct type (e.g. CreatedEventV3) rather than modifying existing ones. |

## Anti-Patterns

- Comparing grant.ExpiresAt or grant.DeletedAt directly instead of using Grant.ActiveAt or GetEffectivePeriod.
- Creating a new event version by modifying the existing grantEventV2 struct — add a new versioned struct instead.
- Implementing Repo without embedding TxCreator and TxUser[Repo] — the transaction propagation chain breaks.
- Using Limit/Offset in new list code — use pagination.Page instead.
- Using unknown ExpirationPeriodDuration values — GetExpiration returns zero time silently.

## Decisions

- **ExpiresAt is pre-computed and stored at creation time rather than derived dynamically.** — The engine queries grants using ExpiresAt as a predicate. Computing it at runtime in SQL would require duplicating the expiration arithmetic in the query layer; pre-storing it keeps queries simple and indexed.
- **v2 events use a dedicated literal struct (grantEventV2GrantLiteral) with only primitive fields instead of embedding the Grant domain struct.** — Embedding Grant in v1 events means domain model changes silently break event serialization compatibility. The v2 literal struct is decoupled from domain evolution and can be versioned independently.

## Example: Creating a new v2 event type for a new grant lifecycle action

```
// In events_2.go — add a new versioned type, never modify existing ones
const grantSuspendedEventNameV2 = "..."

type SuspendedEventV2 grantEventV2

var _ marshaler.Event = SuspendedEventV2{}

func (e SuspendedEventV2) EventName() string { return grantSuspendedEventNameV2 }
func (e SuspendedEventV2) Validate() error   { return grantEventV2(e).Validate() }
```

<!-- archie:ai-end -->
