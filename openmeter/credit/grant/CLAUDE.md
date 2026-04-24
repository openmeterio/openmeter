# grant

<!-- archie:ai-start -->

> Core domain types for the credit grant subsystem: the Grant value type, OwnerConnector interface, Repo interface, expiration/rollover logic, and Watermill event types for grant lifecycle. This package is the contract layer that adapter, engine, and service layers all depend on.

## Patterns

**Grant.GetEffectivePeriod() is the canonical activity window** — Grant.GetEffectivePeriod() returns a StartBoundedPeriod bounded by the earliest of ExpiresAt, DeletedAt, and VoidedAt. Grant.ActiveAt(t) delegates to this. Always use ActiveAt or GetEffectivePeriod rather than comparing fields directly. (`grant.ActiveAt(phase.from) // correct; do not compare grant.ExpiresAt < phase.from directly`)
**Repo embeds TxCreator + TxUser[Repo] for transaction propagation** — grant.Repo interface requires both entutils.TxCreator and entutils.TxUser[Repo] so callers can propagate transactions. Any implementation must satisfy all methods in transaction.go. (`type Repo interface { CreateGrant(...) ... entutils.TxCreator; entutils.TxUser[Repo] }`)
**Event versioning with grantEventV2 literal + ToDomainGrant** — Watermill events use a pinned literal struct (grantEventV2GrantLiteral) that mirrors Grant fields. ToDomainGrant() converts back to Grant. New event versions must follow this pattern with a new version suffix. (`type CreatedEventV2 grantEventV2; func (e CreatedEventV2) EventName() string { return grantCreatedEventNameV2 }`)
**ExpiresAt is pre-computed and stored, not derived at query time** — Grant.ExpiresAt is set from grant.GetExpiration() at creation time (makeGrant in tests, RepoCreateInput.ExpiresAt from callers). Query predicates use the stored ExpiresAt rather than computing from Expiration+EffectiveAt at query time. (`grant.ExpiresAt = grant.GetExpiration() // must be set before persisting`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | Grant struct, GetEffectivePeriod, ActiveAt, RolloverBalance, RecurrenceBalance, GetExpiration. | RolloverBalance clamps to [ResetMinRollover, ResetMaxRollover]; RecurrenceBalance always resets to Amount (recurring grants do not roll over). Do not conflate the two. |
| `owner_connector.go` | OwnerConnector interface (DescribeOwner, GetResetTimelineInclusive, GetUsagePeriodStartAt, etc.), Owner value type, ResetBehavior, OwnerNotFoundError. | GetResetTimelineInclusive returns the start of the period that contains each time point, not just the resets in the period — the first returned time may precede period.From. |
| `repo.go` | Repo interface, ListParams, RepoCreateInput, OrderBy constants. | Limit/Offset fields in ListParams are marked deprecated — prefer Page-based pagination. OrderByDefault is OrderByCreatedAt. |
| `expiration.go` | ExpirationPeriod struct and GetExpiration(t) calculation for all duration types. | Unknown duration values return time.Time{} (zero value) silently — validate ExpirationPeriodDuration before persisting. |

## Anti-Patterns

- Comparing grant.ExpiresAt or grant.DeletedAt directly instead of using Grant.ActiveAt or GetEffectivePeriod.
- Creating a new event version by modifying the existing grantEventV2 struct — add a new versioned struct instead.
- Implementing Repo without embedding TxCreator and TxUser[Repo] — the transaction propagation chain breaks.
- Using Limit/Offset in new code — use pagination.Page instead.

## Decisions

- **ExpiresAt is pre-computed and stored at creation time rather than derived dynamically.** — The engine queries grants using ExpiresAt as a predicate. Computing it at runtime in SQL would require duplicating the expiration arithmetic in the query layer; pre-storing it keeps queries simple and indexed.

<!-- archie:ai-end -->
