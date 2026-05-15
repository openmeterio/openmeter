# balance

<!-- archie:ai-start -->

> Domain types and service layer for credit balance snapshots: defines the in-memory Map (per-grant float64 balances), Snapshot struct, SnapshotRepo interface, and SnapshotService that transparently fills missing usage data from streaming when a stored snapshot has zero usage.

## Patterns

**SnapshotService wraps SnapshotRepo and adds usage back-fill** — SnapshotService.GetLatestValidAt delegates to Repo.GetLatestValidAt and, if the returned Snapshot.Usage.IsZero(), queries streaming via UsageQuerier to compute the actual period usage. New code must not call Repo directly if usage completeness matters. (`service.GetLatestValidAt(ctx, owner, at) // automatically fills usage if zero; do not call Repo.GetLatestValidAt directly`)
**service() sentinel prevents Repo from satisfying SnapshotService** — SnapshotService declares an unexported service() method. This is an explicit guard to prevent callers from accidentally substituting the raw SnapshotRepo as a SnapshotService. (`func (s *service) service() {}`)
**UNIQUE_COUNT aggregation requires double streaming query with alpacadecimal subtraction** — For MeterAggregationUniqueCount, QueryUsage performs two streaming queries (period-start→to and period-start→from) and subtracts using alpacadecimal. All other aggregations use a single from→to query. (`case meter.MeterAggregationUniqueCount: /* query to=period.To, then query to=period.From, subtract with alpacadecimal */`)
**NoSavedBalanceForOwnerError for missing snapshots** — When no snapshot exists before the requested time, SnapshotRepo.GetLatestValidAt returns NoSavedBalanceForOwnerError (not a generic not-found). Callers must type-assert this error to distinguish 'no history' from 'DB error'. (`_, isNoSavedBalanceErr := err.(balance.NoSavedBalanceForOwnerError)`)
**Clone balance.Map before mutating** — balance.Map is a plain map[string]float64 (reference type). Always Clone() before mutating in the engine to avoid aliasing bugs that corrupt shared state. (`rolledOver := snap.Balances.Clone() // never mutate snap.Balances directly`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `balance.go` | Core value types: Map (grant-ID→float64), Snapshot (Map + Overage + At + SnapshottedUsage). Map.Burn/Set/Clone/Balance are the canonical mutation helpers. | Map is a plain map — Clone before mutating in the engine to avoid aliasing bugs. |
| `repository.go` | SnapshotRepo interface definition + NoSavedBalanceForOwnerError domain error type. | GetLatestValidAt comment notes the returned Snapshot may lack usage data — the service layer fills it, do not call repo directly for complete usage. |
| `service.go` | SnapshotService interface + concrete service constructor. SnapshotServiceConfig wires OwnerConnector, StreamingConnector, and Repo together. UsageQuerier is built inside NewSnapshotService. | Do not pass UsageQuerier externally — it is always constructed internally from the config to ensure the correct closure bindings. |
| `usage.go` | UsageQuerier interface and implementation. Dispatches to streaming.Connector.QueryMeter with aggregation-specific logic. | alpacadecimal is used for UNIQUE_COUNT subtraction to avoid float rounding. Do not switch to plain float64 arithmetic for this case. |

## Anti-Patterns

- Calling SnapshotRepo.GetLatestValidAt directly when you need complete usage data — always go through SnapshotService.
- Mutating a balance.Map without cloning it first — Map is a reference type and aliasing causes hard-to-track bugs in the engine.
- Using float64 arithmetic for UNIQUE_COUNT subtraction — use alpacadecimal as usage.go does.
- Returning a generic error instead of NoSavedBalanceForOwnerError when no snapshot is found — callers depend on type-assertion.
- Implementing a new aggregation case without checking whether it requires a double query like UNIQUE_COUNT.

## Decisions

- **SnapshotService adds a back-fill path for snapshots stored without usage data.** — Earlier snapshots may have been persisted without usage fields. The service transparently queries streaming to compute missing usage rather than requiring a data migration.

## Example: Constructing a SnapshotService with all required dependencies

```
svc := balance.NewSnapshotService(balance.SnapshotServiceConfig{
	OwnerConnector:     ownerConnector,
	StreamingConnector: streamingConnector,
	Repo:               repo,
})
// Always use svc.GetLatestValidAt — not repo.GetLatestValidAt — for complete usage data
snapshot, err := svc.GetLatestValidAt(ctx, owner, at)
```

<!-- archie:ai-end -->
