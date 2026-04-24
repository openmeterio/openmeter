# engine

<!-- archie:ai-start -->

> Pure-logic grant burn-down engine: given a starting balance snapshot, a list of grants, a usage query function, and optional reset times, produces an ending Snapshot plus a GrantBurnDownHistory. No Ent or HTTP dependencies — all I/O goes through the QueryUsageFn callback.

## Patterns

**Engine.Run drives periods between resets, calling runBetweenResets per segment** — Run builds a timeline from StartingSnapshot.At + reset times + Until, iterates closed periods, calls runBetweenResets for each, then applies reset() between periods. History segments from all periods are merged into a single GrantBurnDownHistory. (`for idx, period := range timeline.GetClosedPeriods() { runRes, _ := e.runBetweenResets(...); if idx != last { snap, _ = e.reset(...) } }`)
**burnPhase subdivision within a reset-free period** — runBetweenResets further subdivides the period into burnPhases using getPhases(): a new phase starts when grant activity changes (effectiveAt/expiresAt/deletedAt/voidedAt) or a grant recurs. Each phase is burned down independently. (`phases, _ := e.getPhases(grants, period); for _, phase := range phases { ... e.burnDownGrants(...) }`)
**PrioritizeGrants establishes burn order: priority < expiration < (createdAt, id)** — PrioritizeGrants sorts grants using three stable sorts applied in reverse importance order: createdAt+id (tie-breaker), expiration date (earlier first), then priority value (lower = higher importance). Call this before burnDownGrants and re-call when phase.priorityChange is true. (`PrioritizeGrants(grants) // lower priority number = burned first; earlier expiry = burned first`)
**alpacadecimal for arithmetic precision** — burnDownGrants uses alpacadecimal.NewFromFloat for all subtraction to avoid float64 rounding errors. Intermediate results are kept as Decimal; only the final value calls InexactFloat64(). (`exactBalance := alpacadecimal.NewFromFloat(grantBalance); if exactBalance.LessThanOrEqual(exactUsage) { ... }`)
**RunParams.Clone() before storing in RunResult** — Run clones its params at the top (resParams := params.Clone()) so RunResult.RunParams is immutable and independent of the caller's slice. (`resParams := params.Clone(); return RunResult{Snapshot: snapshot, History: history, RunParams: resParams}, nil`)
**Resets must be strictly after StartingSnapshot.At** — Run validates that no reset time equals the snapshot time and returns an error if it does. Callers must ensure the reset timeline is (snapshot.At, Until]. (`if params.Resets.GetBoundingPeriod().ContainsInclusive(params.StartingSnapshot.At) { return RunResult{}, fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Engine interface, RunParams/RunResult types, NewEngine constructor. QueryUsageFn is the only external I/O dependency. | MeterAggregationLatest causes grant amounts to be treated as limits (balancesAtPhaseStart reset to grant.Amount for every active grant at phase start) — distinct from budget behaviour. |
| `run.go` | Engine.Run and runBetweenResets implementations. burnDownGrants is the innermost loop. | runBetweenResets validates that grants and balances pair up (ExactlyForGrants) and returns an error if not — callers must ensure the balance map contains exactly the passed grants. |
| `burnphase.go` | getPhases merges activity-change times and grant-recurrence times into a sorted, deduplicated slice of burnPhase structs. | Times are truncated to minute precision (Truncate(time.Minute)) to avoid sub-minute inconsistencies — preserve this when adding new change-time sources. |
| `grant.go` | Engine helper methods: getGrantActivityChanges, getGrantRecurrenceTimes, filterRelevantGrants, PrioritizeGrants. | PrioritizeGrants is exported and used externally (reset.go). Its sort is three-pass stable — do not collapse into a single comparator. |
| `history.go` | GrantBurnDownHistory and GrantBurnDownHistorySegment value types. GetSnapshotAtStartOfSegment reconstructs a Snapshot at any segment boundary. | GrantBurnDownHistory.Overage() assumes at least one segment exists — callers must not call it on an empty history. |
| `reset.go` | engine.reset() applies rollover rules (RolloverBalance) per grant and optionally burns down carry-over overage. | Grants inactive at reset time are skipped during rollover; their balance is effectively zeroed for the next period. |

## Anti-Patterns

- Introducing Ent or HTTP imports into this package — the engine must remain a pure calculation library.
- Using plain float64 arithmetic for grant burn subtraction — use alpacadecimal to prevent rounding errors.
- Passing a balance.Map that does not exactly match the grants slice to runBetweenResets — it will return an error.
- Calling PrioritizeGrants with a single-pass custom comparator — the three-pass stable sort order is load-bearing for correctness.
- Omitting Clone() on the grants slice before passing to runBetweenResets — the engine sorts grants in-place.

## Decisions

- **The engine is stateless and accepts a QueryUsageFn callback rather than a streaming.Connector directly.** — Decoupling from the streaming package makes the engine independently testable with a simple mock function and allows callers to apply caching or pre-aggregation before calling the engine.
- **Burn phases are computed per-period (between resets) rather than globally.** — Resets change the effective balances of grants, so the priority order and recurrence schedule must be re-evaluated from scratch after each reset. A global phase computation would conflate periods.

## Example: Running the engine for a period with one reset

```
import (
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

eng := engine.NewEngine(engine.EngineConfig{QueryUsage: queryFn})
res, err := eng.Run(ctx, engine.RunParams{
	Meter:  meterDef,
	Grants: activeGrants,
	StartingSnapshot: balance.Snapshot{
		Balances: startBalances,
		At:       periodStart,
	},
// ...
```

<!-- archie:ai-end -->
