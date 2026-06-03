# credit

<!-- archie:ai-start -->

> Manages credit grants and balance snapshots for metered entitlements. CreditConnector (= BalanceConnector + GrantConnector) is the public facade; engine/ computes burn-down without I/O; adapter/ persists via transaction-aware Ent repos; grant/balance/hook/driver sub-packages own the domain contract, snapshots, lifecycle hook, and v1 HTTP layer. Primary constraint: all effective times are truncated to Granularity (time.Minute).

## Patterns

**transaction.Run wraps all multi-step writes** — CreateGrant/VoidGrant/ResetUsageForOwner use transaction.Run then m.GrantRepo.WithTx(ctx, tx) to stay on the ctx-bound transaction. (`transaction.Run(ctx, m.GrantRepo, func(ctx) (*grant.Grant, error) { tx, _ := entutils.GetDriverFromContext(ctx); return m.GrantRepo.WithTx(ctx, tx).CreateGrant(ctx, inp) })`)
**LockOwnerForTx before any write** — All mutating connector methods call m.OwnerConnector.LockOwnerForTx(ctx, ownerID, true) before writing grants/snapshots to prevent concurrent balance races. (`err = m.OwnerConnector.LockOwnerForTx(ctx, ownerID, true)`)
**GetLastValidSnapshotAt falls back to start-of-measurement** — On NoSavedBalanceForOwnerError, create a zero-balance snapshot from GetStartOfMeasurement + NewStartingMap(grants) — never propagate the not-found error. (`if _, ok := lo.ErrorsAs[*balance.NoSavedBalanceForOwnerError](err); ok { bal = balance.Snapshot{At: startOfMeasurement, ...} }`)
**buildEngineForOwner caches period boundaries** — Build a period cache from SortedPeriodsFromDedupedTimes before constructing the engine; resolve GetUsagePeriodStartAt from the cache inside the UsageQuerier closure, never per usage query. (`periodCache := SortedPeriodsFromDedupedTimes(times)`)
**Granularity truncation on all times** — All effective and reset times are truncated to m.Granularity (time.Minute) before engine use and before storing. (`input.EffectiveAt = input.EffectiveAt.Truncate(time.Minute)`)
**Snapshotting acquires non-blocking lock and skips on failure** — snapshotEngineResult uses a non-blocking LockOwnerForTx(false); failure to acquire must not abort the balance query that triggered it. (`if err := transaction.RunWithNoValue(...LockOwnerForTx(ctx, owner, false)); err != nil { return nil }`)
**ValidationIssue errors with HTTP status attribute** — Validation errors use models.NewValidationIssue + commonhttp.WithHTTPStatusCodeAttribute; CreateGrantInput.Validate uses ErrGrantAmountMustBePositive.WithAttr. (`var ErrGrantAmountMustBePositive = models.NewValidationIssue(ErrCodeGrantAmountMustBePositive, "amount must be positive", models.WithFieldString("amount"), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | Defines CreditConnector (= BalanceConnector + GrantConnector), CreditConnectorConfig (embedded by value), and NewCreditConnector. | CreditConnectorConfig is embedded by value — new config fields are visible to all methods automatically. |
| `balance.go` | Implements BalanceConnector: GetBalanceAt, GetBalanceForPeriod, ResetUsageForOwner, GetLastValidSnapshotAt. | GetBalanceAt/GetBalanceForPeriod truncate-up to the next minute (FIXME) — replicate in any new method. Cannot reset in the future or before the current usage period start. |
| `grant.go` | Implements GrantConnector: CreateGrant, VoidGrant; defines GrantConnector interface and CreateGrantInput. | CreateGrant truncates EffectiveAt and Recurrence.Anchor, locks the owner, then invalidates snapshots via BalanceSnapshotService.InvalidateAfter and publishes grant.NewCreatedEventV2FromGrant. |
| `helper.go` | Engine-building helpers: buildEngineForOwner, snapshotEngineResult, saveSnapshot, populate/removeInactiveGrants, SortedPeriodsFromDedupedTimes. | removeInactiveGrantsFromSnapshotAt errors if a grant in the snapshot balances map is not in the grants slice — pass the full grants slice used for engine.Run. |
| `errors.go` | Package-level ValidationIssue error vars with HTTP status attributes (ErrGrantAmountMustBePositive, ErrGrantEffectiveAtMustBeSet). | New validation errors must follow NewValidationIssue + WithHTTPStatusCodeAttribute. |
| `trace.go` | ctrace singleton with typed OTel span options (WithOwner, WithPeriod, WithEngineParams). | All new connector methods should open a span via m.Tracer.Start and use cTrace helpers. |

## Anti-Patterns

- Calling GrantRepo or BalanceSnapshotService methods without wrapping in transaction.Run during a multi-step write — bypasses the ctx-bound transaction
- Omitting LockOwnerForTx before any grant/snapshot mutation — causes balance races under concurrency
- Propagating balance.NoSavedBalanceForOwnerError to callers — always fall back to start-of-measurement snapshot
- Querying GetUsagePeriodStartAt inside the QueryUsageFn callback at runtime — build the period cache in buildEngineForOwner instead
- Passing the caller's grants slice to the engine without populating missing grants via populateBalanceSnapshotWithMissingGrantsActiveAt

## Decisions

- **Period cache built once in buildEngineForOwner, resolved in-memory by the UsageQuerier closure** — QueryUsageFn is called many times during engine.Run; caching period start times avoids O(n) DB calls per burn phase.
- **Snapshotting acquires a non-blocking lock and silently skips on failure** — Snapshotting is an optimisation; lock-acquisition failure must not abort the triggering balance query.
- **All times truncated to Granularity (time.Minute) before engine use** — ClickHouse stores events in minute-window chunks; sub-minute precision causes incorrect burn-down.

<!-- archie:ai-end -->
