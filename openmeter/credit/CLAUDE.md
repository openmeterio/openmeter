# credit

<!-- archie:ai-start -->

> Manages credit grants and balance snapshots for metered entitlements. CreditConnector (= BalanceConnector + GrantConnector) is the public facade; the engine sub-package computes burn-down without I/O; adapter sub-packages persist via Ent in transaction-aware repos.

## Patterns

**transaction.Run wraps all multi-step writes** — CreateGrant and VoidGrant both use transaction.Run(ctx, m.GrantRepo, ...) and then call m.GrantRepo.WithTx(ctx, tx) inside to stay on the ctx-bound transaction. (`transaction.Run(ctx, m.GrantRepo, func(ctx context.Context) (*grant.Grant, error) { tx, _ := entutils.GetDriverFromContext(ctx); return m.GrantRepo.WithTx(ctx, tx).CreateGrant(ctx, inp) })`)
**LockOwnerForTx before any write** — All mutating connector methods call m.OwnerConnector.LockOwnerForTx(ctx, ownerID, ...) before writing grants or snapshots to prevent concurrent balance races. (`err = m.OwnerConnector.LockOwnerForTx(ctx, ownerID, true)`)
**GetLastValidSnapshotAt falls back to start-of-measurement** — When no snapshot exists (NoSavedBalanceForOwnerError), create a zero-balance snapshot from GetStartOfMeasurement + NewStartingMap(grants) — do not propagate the not-found error. (`if _, ok := lo.ErrorsAs[*balance.NoSavedBalanceForOwnerError](err); ok { startOfMeasurement, _ = m.OwnerConnector.GetStartOfMeasurement(ctx, owner); bal = balance.Snapshot{At: startOfMeasurement, ...} }`)
**buildEngineForOwner caches period boundaries** — Build a period cache from SortedPeriodsFromDedupedTimes before constructing the engine, then resolve GetUsagePeriodStartAt from the cache inside the UsageQuerier closure — never call the owner connector per usage query. (`periodCache := SortedPeriodsFromDedupedTimes(times); GetUsagePeriodStartAt: func(..., at time.Time) (time.Time, error) { for _, p := range periodCache { if p.ContainsInclusive(at) { return p.From, nil } } }`)
**snapshotEngineResult saves latest eligible segment** — Iterate segments in reverse (skip index 0 which is the input snapshot); save the first segment whose From is not after notAfter — skips LATEST aggregation type entirely. (`for i := len(segs) - 1; i >= 1; i-- { if !seg.From.After(snapParams.notAfter) { m.saveSnapshot(ctx, ...) ; break } }`)
**Granularity truncation on all times** — All effective times and reset times are truncated to m.Granularity (time.Minute) before engine use and before storing grants or snapshots. (`input.EffectiveAt = input.EffectiveAt.Truncate(granularity)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/credit/connector.go` | Defines CreditConnector interface (BalanceConnector + GrantConnector), CreditConnectorConfig, and the connector struct. NewCreditConnector is the sole constructor. | CreditConnectorConfig is embedded by value in connector — adding fields to config is automatically visible to all methods. |
| `openmeter/credit/balance.go` | Implements BalanceConnector: GetBalanceAt, GetBalanceForPeriod, ResetUsageForOwner, GetLastValidSnapshotAt. | GetBalanceAt and GetBalanceForPeriod both truncate-up to the next minute — any new method must replicate this FIXME truncation. |
| `openmeter/credit/grant.go` | Implements GrantConnector: CreateGrant, VoidGrant. Also defines GrantConnector interface and CreateGrantInput. | CreateGrantInput.Validate() uses ValidationIssue (ErrGrantAmountMustBePositive.WithAttr) — follow the same pattern for new input fields. |
| `openmeter/credit/helper.go` | Internal engine-building helpers: buildEngineForOwner, snapshotEngineResult, saveSnapshot, populateBalanceSnapshotWithMissingGrantsActiveAt, removeInactiveGrantsFromSnapshotAt, SortedPeriodsFromDedupedTimes. | removeInactiveGrantsFromSnapshotAt returns an error if a grant in the snapshot balances map is not in the grants slice — callers must pass the full grants slice used for engine.Run. |
| `openmeter/credit/errors.go` | Package-level ValidationIssue error vars (ErrGrantAmountMustBePositive, ErrGrantEffectiveAtMustBeSet) with HTTP status attributes. | New validation errors should follow the same models.NewValidationIssue + commonhttp.WithHTTPStatusCodeAttribute pattern. |
| `openmeter/credit/trace.go` | ctrace singleton providing typed OTel span start options (WithOwner, WithPeriod, WithEngineParams). | All new connector methods should add a span via m.Tracer.Start and use cTrace helpers for structured attributes. |

## Anti-Patterns

- Calling GrantRepo or BalanceSnapshotService methods without wrapping in transaction.Run when inside a multi-step write — bypasses ctx-bound transaction.
- Omitting LockOwnerForTx before any grant/snapshot mutation — causes balance races under concurrent requests.
- Propagating balance.NoSavedBalanceForOwnerError to callers — always fall back to start-of-measurement snapshot.
- Querying GetUsagePeriodStartAt inside the QueryUsageFn callback at runtime — build the period cache in buildEngineForOwner instead.
- Passing the caller's grants slice to the engine without populating missing grants via populateBalanceSnapshotWithMissingGrantsActiveAt — engine will error on unknown grant IDs in the snapshot.

## Decisions

- **Period cache built once in buildEngineForOwner, resolved in-memory by UsageQuerier closure.** — QueryUsageFn is called many times during engine.Run; caching period start times avoids O(n) DB calls per burn phase.
- **snapshotEngineResult acquires a non-blocking lock and silently skips on failure.** — Snapshotting is an optimisation; failure to acquire the lock should not abort the balance query that triggered it.
- **All times truncated to Granularity (time.Minute) before engine use.** — ClickHouse stores events in minute-window chunks; sub-minute precision causes incorrect burn-down calculations.

## Example: Adding a new multi-step write method to the credit connector

```
import (
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (m *connector) NewMutation(ctx context.Context, ownerID models.NamespacedID) error {
	ctx, span := m.Tracer.Start(ctx, "credit.NewMutation", cTrace.WithOwner(ownerID))
	defer span.End()
	_, err := transaction.Run(ctx, m.GrantRepo, func(ctx context.Context) (*interface{}, error) {
		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil { return nil, err }
		if err := m.OwnerConnector.LockOwnerForTx(ctx, ownerID, true); err != nil { return nil, err }
		return nil, m.GrantRepo.WithTx(ctx, tx).SomeWrite(ctx, ownerID)
	})
	return err
// ...
```

<!-- archie:ai-end -->
