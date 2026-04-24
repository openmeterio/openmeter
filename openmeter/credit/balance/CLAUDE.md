# balance

<!-- archie:ai-start -->

> Domain types and service layer for credit balance snapshots: defines the in-memory Map (per-grant float64 balances), Snapshot struct, SnapshotRepo interface, and SnapshotService that fills missing usage data from streaming when a stored snapshot has zero usage.

## Patterns

**SnapshotService wraps SnapshotRepo and adds usage back-fill** — SnapshotService.GetLatestValidAt delegates to Repo.GetLatestValidAt and, if the returned Snapshot.Usage.IsZero(), queries streaming via UsageQuerier to compute the actual period usage. New code must not call Repo directly if usage completeness matters. (`service.GetLatestValidAt(ctx, owner, at) // automatically fills usage if zero`)
**service() sentinel method prevents Repo from satisfying SnapshotService** — SnapshotService declares an unexported service() method. This is an explicit guard to prevent callers from accidentally using the raw SnapshotRepo as a SnapshotService. (`func (s *service) service() {}`)
**UsageQuerier handles UNIQUE_COUNT aggregation separately** — For MeterAggregationUniqueCount, the querier performs two streaming queries (period-start→to and period-start→from) and subtracts using alpacadecimal for accuracy. All other aggregations use a single from→to query. (`case meter.MeterAggregationUniqueCount: /* double query and subtract */`)
**NoSavedBalanceForOwnerError for missing snapshots** — When no snapshot exists for the owner before the requested time, SnapshotRepo.GetLatestValidAt returns NoSavedBalanceForOwnerError (not a generic not-found). Callers must type-assert this error to distinguish 'no history' from 'DB error'. (`_, isNoSavedBalanceErr := err.(balance.NoSavedBalanceForOwnerError)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `balance.go` | Core value types: Map (grant-ID→float64), Snapshot (Map + Overage + At + SnapshottedUsage). Map.Burn/Set/Clone/Balance are the canonical mutation helpers. | Map is a plain map[string]float64 — Clone before mutating in the engine to avoid aliasing bugs. |
| `repository.go` | SnapshotRepo interface definition + NoSavedBalanceForOwnerError domain error type. | GetLatestValidAt comment notes the returned Snapshot may lack usage data — the service layer fills it. |
| `service.go` | SnapshotService interface + concrete service constructor. SnapshotServiceConfig wires OwnerConnector, StreamingConnector, and Repo together. | UsageQuerier is built inside NewSnapshotService via inline config — do not pass it externally. |
| `usage.go` | UsageQuerier interface and implementation. Dispatches to streaming.Connector.QueryMeter with aggregation-specific logic. | alpacadecimal is used for UNIQUE_COUNT subtraction to avoid float rounding. Do not switch to plain float64 arithmetic for this case. |

## Anti-Patterns

- Calling SnapshotRepo.GetLatestValidAt directly when you need complete usage data — always go through SnapshotService.
- Mutating a balance.Map without cloning it first — Map is a reference type and aliasing causes hard-to-track bugs in the engine.
- Using float64 arithmetic for UNIQUE_COUNT subtraction — use alpacadecimal as usage.go does.
- Returning a generic error instead of NoSavedBalanceForOwnerError when no snapshot is found — callers depend on type-assertion.

## Decisions

- **SnapshotService adds a back-fill path for snapshots stored without usage data.** — Earlier snapshots may have been persisted without usage fields. The service transparently queries streaming to compute missing usage rather than requiring a migration.

<!-- archie:ai-end -->
