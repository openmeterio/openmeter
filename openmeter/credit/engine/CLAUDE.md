# engine

<!-- archie:ai-start -->

> Pure, stateless grant burn-down engine. Engine.Run takes a starting balance Snapshot, grants, a reset Timeline and a usage-query function, then produces the ending Snapshot plus a GrantBurnDownHistory of per-segment usage. No DB/IO except the injected QueryUsage callback.

## Patterns

**Reset-period then burn-phase decomposition** — Run splits [StartingSnapshot.At, Until] at reset times into closed periods (runBetweenResets per period, reset() between them), and within each period getPhases splits on grant recurrence and priority changes into burnPhases. (`for idx, period := range timeline.GetClosedPeriods() { runRes := e.runBetweenResets(...); if idx != last { snap := e.reset(...); segments[last].TerminationReasons.UsageReset = true } }`)
**Decimal-exact burndown over float storage** — burnDownGrants converts usage/balances to alpacadecimal for subtraction to limit float error, returning float64 results and per-grant GrantUsage with TerminationReason (EXHAUSTED vs SEGMENT_TERMINATION). (`exactUsage := alpacadecimal.NewFromFloat(usage); if exactBalance.LessThanOrEqual(exactUsage) { exactUsage = exactUsage.Sub(exactBalance); ... }`)
**Deterministic grant prioritization** — PrioritizeGrants stable-sorts by (1) priority asc, (2) earlier expiration first, (3) tie-break created_at then id — applied initially and re-applied whenever a phase has priorityChange. (`PrioritizeGrants(grants) // priority < then expiration < then created_at,id`)
**Recurrence/activity drive balance resets, not the order** — Recurrence sets balance to grant.RecurrenceBalance (full Amount); grants newly active at phase.from get full Amount; grants becoming inactive get balance 0; LATEST-aggregation meters treat Amount as a per-point limit (reset each phase). (`if params.Meter.Aggregation == meter.MeterAggregationLatest { for _, g := range activeGrants { balancesAtPhaseStart[g.ID] = g.Amount } }`)
**Balance/grant-set invariant** — runBetweenResets requires StartingSnapshot.Balances.ExactlyForGrants(Grants); filterRelevantGrants narrows grants to those overlapping the period or present in the balance map before each run. (`if !params.StartingSnapshot.Balances.ExactlyForGrants(params.Grants) { return error }`)
**History chunked by resets** — GrantBurnDownHistory stores ordered non-overlapping segments + usageAtStart; ChunkByResets/GetUsageInPeriodUntilSegment recompute SnapshottedUsage relative to the last UsageReset segment. (`for _, seg := range g.segments { current.segments = append(...); if seg.TerminationReasons.UsageReset { chunks = append(chunks, current); current = ...usageAtReset(seg.To) } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Engine interface, RunParams/RunResult, NewEngine(EngineConfig{QueryUsage}). | Output balance includes inactive grants (documented); QueryUsageFn returns float64 (TODO to make decimal); RunParams.Clone deep-copies grants and resets. |
| `run.go` | Run (period loop + reset) and runBetweenResets (phase loop + burndown), burnDownGrants, inbetweenRunParams. | Reset times must occur strictly after StartingSnapshot.At and within [At, Until] or Run errors; PeriodEndRecurrenceBoundaryBehavior is Inclusive only for the final period; burndown has a documented float rounding FIXME. |
| `burnphase.go` | getPhases: merges activity changes and recurrence times into ordered burnPhases, handling grantsRecurredAtStart and terminal zero-length phases. | Empty/zero-length phases are dropped by appendPhase unless phase.to.After(phase.from); a terminal recurrence appends a 0-length phase at period.To. |
| `grant.go` | getGrantActivityChanges, getGrantRecurrenceTimes, filterRelevantGrants, PrioritizeGrants. | Activity changes are truncated to the minute and UTC-normalized (FIXME noted); recurrence iteration uses endBoundaryBehavior and grant.ActiveAt to bound times. |
| `history.go` | GrantBurnDownHistorySegment, GrantBurnDownHistory, NewGrantBurnDownHistory (validates no overlap), ChunkByResets, TotalGrantUsage. | MarshalJSON serializes only segments (usageAtStart is internal); GetUsageInPeriodUntilSegment resets the running usage at the last UsageReset segment. |
| `reset.go` | reset(): rolls over active grants via RolloverBalance, optionally preserves overage, re-prioritizes and burns down at the reset boundary. | Inactive-at-reset grants are skipped from rollover; missing grant in grantMap is an internal inconsistency error. |

## Anti-Patterns

- Doing IO inside the engine — all usage comes through the injected QueryUsage callback.
- Passing balances whose key set doesn't ExactlyForGrants the grants slice (runBetweenResets rejects it).
- Subtracting usage from balances with raw float64 instead of alpacadecimal in burndown logic.
- Assuming a grant absent from the period is excluded from the output Snapshot — inactive grants remain in the Map with 0/rolled-over balance.
- Skipping re-prioritization after a phase.priorityChange or recurrence application.

## Decisions

- **The engine is split into reset-bounded periods, then recurrence/priority burn phases.** — Resets roll over balances and may preserve overage (different semantics than recurrence), while phases capture priority/recurrence changes that alter burn order — separating them keeps burndown deterministic and resumable across multiple Run calls.
- **Same period computed across multiple Run invocations yields identical results (fuzz-tested).** — Balance computation must be incrementally resumable from any snapshot, so the engine is required to be deterministic and split-invariant.
- **LATEST-aggregation meters reset grant balance to Amount each phase.** — For point-in-time meters grant Amounts are limits, not consumable budgets, so usage is deducted from the full amount per phase.

## Example: Splitting the run into reset periods and resetting between them

```
for idx, period := range timeline.GetClosedPeriods() {
	relevantGrants := e.filterRelevantGrants(params.Grants, snapshot.Balances, period)
	runRes, err := e.runBetweenResets(ctx, inbetweenRunParams{
		Grants: relevantGrants, Until: period.To, StartingSnapshot: snapshot, Meter: params.Meter,
		PeriodEndRecurrenceBoundaryBehavior: lo.Ternary(idx == len(timeline.GetClosedPeriods())-1, timeutil.Inclusive, timeutil.Exclusive),
	})
	if err != nil { return RunResult{}, err }
	snapshot = runRes.Snapshot
	historySegments = append(historySegments, runRes.History.Segments()...)
	if idx != len(timeline.GetClosedPeriods())-1 {
		snapshot, _ = e.reset(relevantGrants, runRes.Snapshot, params.ResetBehavior, period.To)
		historySegments[len(historySegments)-1].TerminationReasons.UsageReset = true
	}
}
```

<!-- archie:ai-end -->
