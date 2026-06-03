# grant

<!-- archie:ai-start -->

> Core domain types for the credit grant subsystem: the Grant value type, OwnerConnector and Repo interfaces, expiration/rollover logic, and Watermill grant lifecycle events. This is the contract layer adapter/engine/service depend on — no Ent or HTTP imports.

## Patterns

**ActiveAt / GetEffectivePeriod as canonical activity window** — GetEffectivePeriod() returns a StartBoundedPeriod bounded by the earliest of ExpiresAt, DeletedAt, VoidedAt; ActiveAt(t) delegates to it. Use these instead of comparing fields directly. (`grant.ActiveAt(phase.from) // not grant.ExpiresAt < phase.from`)
**Repo embeds TxCreator + TxUser[Repo]** — grant.Repo requires entutils.TxCreator and entutils.TxUser[Repo] so callers propagate transactions; implementations must satisfy Tx/WithTx/Self. (`type Repo interface { CreateGrant(...) ...; entutils.TxCreator; entutils.TxUser[Repo] }`)
**Versioned event literals (v2) over embedded domain** — v2 events use a pinned primitive-only struct (grantEventV2GrantLiteral) with ToDomainGrant() conversion. New versions add a new named struct; never modify existing ones. (`type CreatedEventV2 grantEventV2; func (e CreatedEventV2) EventName() string { return grantCreatedEventNameV2 }`)
**ExpiresAt pre-computed at creation** — Grant.ExpiresAt is set from GetExpiration() at creation (RepoCreateInput.ExpiresAt); query predicates use stored ExpiresAt rather than recomputing. (`RepoCreateInput.ExpiresAt = grant.GetExpiration(grant.EffectiveAt)`)
**RolloverBalance vs RecurrenceBalance — distinct semantics** — RolloverBalance clamps to [ResetMinRollover, ResetMaxRollover]; RecurrenceBalance always resets to Amount. Do not conflate. (`rolledOver[grantID] = grant.RolloverBalance(grantBalance) // recurrence uses grant.RecurrenceBalance()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | Grant struct, GetEffectivePeriod, ActiveAt, RolloverBalance, RecurrenceBalance, GetExpiration. | RolloverBalance clamps to [ResetMinRollover, ResetMaxRollover]; RecurrenceBalance always resets to Amount. GetEffectivePeriod upper bound = first non-nil of ExpiresAt/DeletedAt/VoidedAt. |
| `owner_connector.go` | OwnerConnector interface (DescribeOwner, GetResetTimelineInclusive, GetUsagePeriodStartAt, ...), Owner value type, ResetBehavior, OwnerNotFoundError. | GetResetTimelineInclusive returns the start of the containing period for each time point — the first returned time may precede period.From. |
| `repo.go` | Repo interface, ListParams, RepoCreateInput, OrderBy constants. | Limit/Offset on ListParams are deprecated — prefer Page-based pagination. OrderByDefault is OrderByCreatedAt. |
| `expiration.go` | ExpirationPeriod struct + GetExpiration(t) for day/month/year durations. | Unknown duration values return zero time.Time silently — validate ExpirationPeriodDuration before persisting. |
| `events.go` | v1 Watermill events (CreatedEvent, VoidedEvent) — deprecated, embed Grant directly. EventSubsystem = "credit". | v1 embeds the domain Grant, making serialization fragile to domain changes; use v2 literal pattern for new events. |
| `events_2.go` | v2 events using grantEventV2GrantLiteral (primitive fields) + ToDomainGrant(). | When adding a grant field needed in events, add to grantEventV2GrantLiteral AND ToDomainGrant(), and create a new versioned struct rather than editing existing ones. |

## Anti-Patterns

- Comparing grant.ExpiresAt/DeletedAt directly instead of using ActiveAt/GetEffectivePeriod.
- Creating a new event version by modifying the existing grantEventV2 struct.
- Implementing Repo without embedding TxCreator and TxUser[Repo].
- Using Limit/Offset in new list code instead of pagination.Page.
- Using unknown ExpirationPeriodDuration values — GetExpiration returns zero time silently.

## Decisions

- **ExpiresAt is pre-computed and stored at creation rather than derived dynamically.** — The engine queries grants by ExpiresAt; computing it in SQL would duplicate expiration arithmetic, so pre-storing keeps queries simple and indexable.
- **v2 events use a primitive-only literal struct instead of embedding the Grant domain struct.** — Embedding Grant (v1) means domain changes break event serialization; the v2 literal is decoupled and independently versionable.

## Example: Adding a new v2 grant event type

```
// In events_2.go — add a new versioned type, never modify existing ones
const grantSuspendedEventNameV2 = "..."

type SuspendedEventV2 grantEventV2

var _ marshaler.Event = SuspendedEventV2{}

func (e SuspendedEventV2) EventName() string { return grantSuspendedEventNameV2 }
func (e SuspendedEventV2) Validate() error   { return grantEventV2(e).Validate() }
```

<!-- archie:ai-end -->
