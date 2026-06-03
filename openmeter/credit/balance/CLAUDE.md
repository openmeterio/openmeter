# balance

<!-- archie:ai-start -->

> Domain types and service layer for credit balance snapshots: the in-memory Map (per-grant float64 balances), Snapshot struct, SnapshotRepo interface, and SnapshotService that transparently back-fills missing usage from streaming when a stored snapshot has zero usage.

## Patterns

**SnapshotService wraps SnapshotRepo + usage back-fill** — SnapshotService.GetLatestValidAt delegates to Repo then, if Snapshot.Usage.IsZero(), queries streaming via UsageQuerier to compute period usage. Use the service, not the raw Repo, when usage completeness matters. (`snapshot, err := svc.GetLatestValidAt(ctx, owner, at) // fills usage if zero`)
**service() sentinel guard** — SnapshotService declares an unexported service() method so the raw SnapshotRepo cannot accidentally satisfy SnapshotService. (`func (s *service) service() {}`)
**UNIQUE_COUNT needs a double streaming query** — For MeterAggregationUniqueCount, QueryUsage queries (periodStart→to) and (periodStart→from) and subtracts via alpacadecimal; all other aggregations use a single from→to query. (`vTo.Sub(vFrom).InexactFloat64()`)
**NoSavedBalanceForOwnerError for missing snapshots** — GetLatestValidAt returns NoSavedBalanceForOwnerError (not generic not-found) when no snapshot precedes the requested time; callers type-assert it. (`_, isNoSavedBalanceErr := err.(balance.NoSavedBalanceForOwnerError)`)
**Clone balance.Map before mutating** — Map is a plain map[string]float64 (reference type). Clone() before mutating to avoid aliasing shared state. (`rolledOver := snap.Balances.Clone()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `balance.go` | Value types: Map (grantID→float64) with Burn/Set/Clone/Balance/ExactlyForGrants, Snapshot (Map + Overage + At + SnapshottedUsage), NewStartingMap. | Map is a reference type — Clone before mutating. SnapshottedUsage.IsZero() drives the service back-fill path. |
| `repository.go` | SnapshotRepo interface + NoSavedBalanceForOwnerError domain error. | GetLatestValidAt comment notes the returned Snapshot may lack usage; the service layer fills it — don't call repo directly for complete usage. |
| `service.go` | SnapshotService interface + concrete service; SnapshotServiceConfig wires OwnerConnector, StreamingConnector, Repo; UsageQuerier built inside NewSnapshotService. | Do not pass UsageQuerier externally — it is constructed internally from config to bind the right closures. |
| `usage.go` | UsageQuerier interface + impl dispatching to streaming.Connector.QueryMeter with aggregation-specific logic. | alpacadecimal used for UNIQUE_COUNT subtraction; getValueFromRows errors if >1 row. Don't switch to plain float64 for the unique-count case. |

## Anti-Patterns

- Calling SnapshotRepo.GetLatestValidAt directly when complete usage is needed — go through SnapshotService.
- Mutating a balance.Map without cloning first — aliasing corrupts shared engine state.
- Using float64 arithmetic for UNIQUE_COUNT subtraction instead of alpacadecimal.
- Returning a generic error instead of NoSavedBalanceForOwnerError when no snapshot exists.
- Adding a new aggregation case without checking whether it needs a double query like UNIQUE_COUNT.

## Decisions

- **SnapshotService transparently back-fills usage for snapshots persisted without usage data.** — Earlier snapshots may lack usage fields; querying streaming at read time avoids a data migration.

## Example: Constructing a SnapshotService with all required dependencies

```
svc := balance.NewSnapshotService(balance.SnapshotServiceConfig{
	OwnerConnector:     ownerConnector,
	StreamingConnector: streamingConnector,
	Repo:               repo,
})
snapshot, err := svc.GetLatestValidAt(ctx, owner, at) // not repo.GetLatestValidAt
```

<!-- archie:ai-end -->
