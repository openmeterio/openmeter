# engine

<!-- archie:ai-start -->

> Pure-logic grant burn-down engine: from a starting snapshot, grants, a QueryUsageFn callback, and optional reset times it produces an ending Snapshot plus a GrantBurnDownHistory. No Ent or HTTP dependencies — all I/O is the QueryUsageFn callback.

## Patterns

**Engine.Run iterates periods between resets** — Run builds a timeline from StartingSnapshot.At + reset times + Until, runs runBetweenResets per closed period, applies reset() between periods, and merges history segments into one GrantBurnDownHistory. (`for idx, period := range timeline.GetClosedPeriods() { runRes, _ := e.runBetweenResets(...); if idx != last { snap, _ = e.reset(...) } }`)
**burnPhase subdivision within a reset-free period** — runBetweenResets subdivides each period into burnPhases via getPhases(): a new phase starts on grant activity change (effectiveAt/expiresAt/deletedAt/voidedAt) or grant recurrence; each phase burns down independently. (`phases, _ := e.getPhases(grants, period, boundary); for _, phase := range phases { e.burnDownGrants(...) }`)
**PrioritizeGrants three-pass stable sort** — PrioritizeGrants applies three stable sorts in reverse importance: createdAt+id, then expiration (earlier first), then priority (lower first). Must run before burnDownGrants and re-run on priorityChange. (`PrioritizeGrants(grants) // lower priority number burned first; earlier expiry burned first`)
**alpacadecimal for burn arithmetic** — burnDownGrants keeps intermediate balances as alpacadecimal.Decimal; only the final value calls InexactFloat64() to avoid float64 rounding. (`exactBalance := alpacadecimal.NewFromFloat(grantBalance); if exactBalance.LessThanOrEqual(exactUsage) { ... }`)
**RunParams.Clone() before storing in RunResult** — Run clones params at the top so RunResult.RunParams is immutable and independent of the caller's slices. (`resParams := params.Clone()`)
**Resets must be strictly after StartingSnapshot.At** — Run errors if any reset time equals the snapshot time; the reset timeline must be (snapshot.At, Until]. (`if params.Resets.GetBoundingPeriod().ContainsInclusive(params.StartingSnapshot.At) { return RunResult{}, fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Engine interface, RunParams/RunResult, QueryUsageFn type, NewEngine. QueryUsageFn is the only external I/O. | MeterAggregationLatest treats grant amounts as limits (balances reset to grant.Amount at every phase start) — distinct from budget behaviour. |
| `run.go` | Engine.Run, runBetweenResets, and innermost burnDownGrants. | runBetweenResets validates grants pair exactly with balances (ExactlyForGrants) and errors otherwise; the grants slice is copied before sorting. |
| `burnphase.go` | getPhases merges activity-change and recurrence times into a sorted, deduplicated []burnPhase. | Times are Truncate(time.Minute); preserve this when adding new change-time sources. |
| `grant.go` | Helpers: getGrantActivityChanges, getGrantRecurrenceTimes, filterRelevantGrants, PrioritizeGrants. | PrioritizeGrants is exported (used by reset.go); its three-pass stable sort is load-bearing — do not collapse into one comparator. |
| `history.go` | GrantBurnDownHistory / GrantBurnDownHistorySegment value types; GetSnapshotAtStartOfSegment reconstructs a Snapshot at a boundary. | Overage() assumes at least one segment exists — never call on empty history. |
| `reset.go` | engine.reset() applies RolloverBalance per grant and optionally burns carry-over overage. | Grants inactive at reset time are skipped during rollover — their balance is effectively zeroed for the next period. |

## Anti-Patterns

- Introducing Ent or HTTP imports — the engine must remain a pure calculation library.
- Using plain float64 arithmetic for grant burn subtraction instead of alpacadecimal.
- Passing a balance.Map that does not exactly match the grants slice to runBetweenResets.
- Replacing PrioritizeGrants' three-pass stable sort with a single-pass comparator.
- Omitting Clone() on the grants slice before passing to the engine (it sorts in place).

## Decisions

- **The engine is stateless and takes a QueryUsageFn callback rather than a streaming.Connector.** — Decoupling from streaming makes the engine independently testable with a mock function and lets callers cache/pre-aggregate.
- **Burn phases are computed per-period (between resets) rather than globally.** — Resets change effective balances, so priority order and recurrence schedules must be re-evaluated after each reset; a global computation would conflate periods.

## Example: Running the engine for a period

```
import (
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
)

eng := engine.NewEngine(engine.EngineConfig{QueryUsage: queryFn})
res, err := eng.Run(ctx, engine.RunParams{
	Meter:  meterDef,
	Grants: activeGrants,
	StartingSnapshot: balance.Snapshot{Balances: startBalances, At: periodStart},
	Until: periodEnd,
})
```

<!-- archie:ai-end -->
