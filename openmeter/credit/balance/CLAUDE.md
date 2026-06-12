# balance

<!-- archie:ai-start -->

> Defines the credit balance model (balance.Map of grantID->float64, Snapshot, SnapshottedUsage) and the SnapshotService that loads snapshots and backfills usage from the streaming connector. Pure computation + repo interface; no ent imports.

## Patterns

**Service wraps Repo and seals itself** — SnapshotService has the same methods as SnapshotRepo plus an unexported service() method so a repo can never accidentally satisfy the service interface. (`type SnapshotService interface { ...; service() }`)
**Usage backfill on read** — GetLatestValidAt: if the stored Snapshot.Usage.IsZero(), the service recomputes usage via OwnerConnector.GetUsagePeriodStartAt + UsageQuerier.QueryUsage and fills it in before returning. (`if res.Usage.IsZero() { usage, _ := s.UsageQuerier.QueryUsage(ctx, owner, timeutil.ClosedPeriod{From: periodStart, To: res.At}); res.Usage = SnapshottedUsage{Usage: usage, Since: periodStart} }`)
**Aggregation-aware usage query** — UsageQuerier.QueryUsage switches on owner.Meter.Aggregation: UNIQUE_COUNT does two period-start-anchored queries and subtracts (using alpacadecimal); SUM/COUNT/LATEST query the period directly; unknown aggregation errors. (`switch owner.Meter.Aggregation { case meter.MeterAggregationUniqueCount: ...; case meter.MeterAggregationSum, ...: ...; default: return 0, fmt.Errorf("unsupported aggregation %s", ...) }`)
**balance.Map value-type helpers** — Map is a plain map[string]float64 with Clone/Burn/Set/Balance/ExactlyForGrants; ExactlyForGrants asserts the map's grant set exactly matches a grant slice (used as an engine invariant). (`if !startingBalances.ExactlyForGrants(grants) { return error }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `balance.go` | Map type + helpers, SnapshottedUsage, Snapshot (Balances/Overage/At/Usage), NewStartingMap. | Map.Burn does balance - amount with no zero floor — overage handling lives in the engine, not here; Snapshot.Balance() sums all entries including inactive grants. |
| `service.go` | SnapshotService interface + service struct; NewSnapshotService builds an internal UsageQuerier from OwnerConnector callbacks. | Service delegates Invalidate/Save straight to Repo; only GetLatestValidAt adds backfill logic. |
| `usage.go` | UsageQuerier interface + impl; translates owner meter aggregation into streaming.QueryParams windows. | UNIQUE_COUNT requires the two-query subtraction; getValueFromRows errors if >1 row and returns 0 for 0 rows. |
| `repository.go` | SnapshotRepo interface + NoSavedBalanceForOwnerError. | Repo.GetLatestValidAt may return a Snapshot without usage data — callers must use the service for backfill. |

## Anti-Patterns

- Importing openmeter/ent here — this package is persistence-agnostic; the adapter implements SnapshotRepo.
- Letting a Repo satisfy SnapshotService — the sealing service() method forbids it.
- Querying UNIQUE_COUNT meters with a single window instead of the period-start subtraction.
- Treating Snapshot.Balance() as 'active balance' — it includes inactive/expired grants still present in the Map.

## Decisions

- **Usage may be stored zero in a snapshot and recomputed lazily at read time.** — Older snapshots predate stored usage; backfilling from streaming keeps GetLatestValidAt correct without a data migration.
- **UNIQUE_COUNT usage is derived by subtracting two cumulative point-in-time queries.** — UNIQUE_COUNT is non-additive over arbitrary windows, so a period delta must be computed from period-start anchored counts.

## Example: Backfilling usage when a snapshot was stored without it

```
func (s *service) GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error) {
	res, err := s.Repo.GetLatestValidAt(ctx, owner, at)
	if err != nil { return Snapshot{}, err }
	if res.Usage.IsZero() {
		periodStart, err := s.OwnerConnector.GetUsagePeriodStartAt(ctx, owner, res.At)
		if err != nil { return Snapshot{}, err }
		usage, err := s.UsageQuerier.QueryUsage(ctx, owner, timeutil.ClosedPeriod{From: periodStart, To: res.At})
		if err != nil { return Snapshot{}, err }
		res.Usage = SnapshottedUsage{Usage: usage, Since: periodStart}
	}
	return res, nil
}
```

<!-- archie:ai-end -->
