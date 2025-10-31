package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (e *engine) Run(ctx context.Context, params RunParams) (RunResult, error) {
	resParams := params.Clone()

	// Let's build the timeline
	times := []time.Time{
		params.StartingSnapshot.At,
	}

	// If the start time is a reset, that reset shouldn't be included in the timeline
	if params.Resets.GetBoundingPeriod().ContainsInclusive(params.StartingSnapshot.At) {
		return RunResult{}, fmt.Errorf("provided reset times must occur after the starting snapshot, got %s", params.StartingSnapshot.At)
	}

	times = append(times, params.Resets.GetTimes()...)

	times = append(times, params.Until)

	timeline := timeutil.NewSimpleTimeline(times)

	boundingPeriod := timeline.GetBoundingPeriod()

	// Let's validate that the timeline represents the correct period
	// This would occur if the provided reset times fall outside of the provided period.
	if boundingPeriod.From.Compare(params.StartingSnapshot.At) != 0 || boundingPeriod.To.Compare(params.Until) != 0 {
		return RunResult{}, fmt.Errorf("timeline does not represent the correct period, expected %s - %s, got %s - %s", params.StartingSnapshot.At, params.Until, boundingPeriod.From, boundingPeriod.To)
	}

	snapshot := params.StartingSnapshot
	historySegments := make([]GrantBurnDownHistorySegment, 0)

	for idx, period := range timeline.GetClosedPeriods() {
		// Let's reset the snapshot usage information as we're entering a new period (between resets)
		if idx > 0 {
			snapshot.Usage = balance.SnapshottedUsage{
				Since: period.From,
				Usage: 0.0,
			}
		}

		// We need to find the grants that are relevant for this period.
		// We do this filtering so that history isn't polluted with grants that are irrelevant.
		relevantGrants := e.filterRelevantGrants(params.Grants, snapshot.Balances, period)

		runRes, err := e.runBetweenResets(ctx, inbetweenRunParams{
			Grants:           relevantGrants,
			Until:            period.To,
			StartingSnapshot: snapshot,
			Meter:            params.Meter,
		})
		if err != nil {
			return RunResult{}, fmt.Errorf("failed to run calculation for period %s - %s: %w", period.From, period.To, err)
		}

		snapshot = runRes.Snapshot
		historySegments = append(historySegments, runRes.History.Segments()...)

		if idx != len(timeline.GetClosedPeriods())-1 {
			// We need to reset at each period, except the last one.
			// If the ending time is also a reset, there will be a 0 length period at the end.
			snap, err := e.reset(relevantGrants, runRes.Snapshot, params.ResetBehavior, period.To)
			if err != nil {
				return RunResult{}, fmt.Errorf("failed to reset at end of period %s - %s: %w", period.From, period.To, err)
			}

			snapshot = snap

			// We need to mark the history segment as one resulting from a reset for all periods except the last one
			if len(historySegments) > 0 {
				historySegments[len(historySegments)-1].TerminationReasons.UsageReset = true
			}
		}
	}

	history, err := NewGrantBurnDownHistory(historySegments, params.StartingSnapshot.Usage)
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to create grant burn down history: %w", err)
	}

	return RunResult{
		Snapshot:  snapshot,
		History:   history,
		RunParams: resParams,
	}, nil
}

type inbetweenRunParams struct {
	// List of all grants that are active at the relevant period at some point.
	Grants []grant.Grant
	// End of the period to burn down the grants for.
	Until time.Time
	// Starting snapshot of the balances at the START OF THE PERIOD.
	StartingSnapshot balance.Snapshot
	// Meter for the current run.
	Meter meter.Meter
}

// Burns down all grants in the defined period by the usage amounts.
//
// When the engine outputs a balance, it doesn't discriminate what should be in that balance.
// If a grant is inactive at the end of the period, it will still be in the output.
func (e *engine) runBetweenResets(ctx context.Context, params inbetweenRunParams) (RunResult, error) {
	period := timeutil.ClosedPeriod{From: params.StartingSnapshot.At, To: params.Until}

	if !params.StartingSnapshot.Balances.ExactlyForGrants(params.Grants) {
		return RunResult{}, fmt.Errorf("provided grants and balances don't pair up, grants: %+v, balances: %+v", params.Grants, params.StartingSnapshot.Balances)
	}

	grants := make([]grant.Grant, len(params.Grants))
	copy(grants, params.Grants)

	phases, err := e.getPhases(grants, period)
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to get burn phases: %w", err)
	}

	err = PrioritizeGrants(grants)
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to prioritize grants: %w", err)
	}

	// Only respect balances that we know the grants of, otherwise we cannot guarantee
	// that the output balance is correct for said grants.
	balancesAtPhaseStart := params.StartingSnapshot.Balances.Clone()

	rePrioritize := false
	recurredGrants := []string{}

	grantMap := make(map[string]grant.Grant)
	for _, grant := range grants {
		grantMap[grant.ID] = grant
	}

	segments := make([]GrantBurnDownHistorySegment, 0, len(phases))

	overage := params.StartingSnapshot.Overage

	for _, phase := range phases {
		// reprioritize grants if needed
		if rePrioritize {
			err = PrioritizeGrants(grants)
			if err != nil {
				return RunResult{}, fmt.Errorf("failed to prioritize grants: %w", err)
			}
			rePrioritize = false
		}

		// reset recurring grant balances
		if len(recurredGrants) > 0 {
			for _, grantID := range recurredGrants {
				grant, ok := grantMap[grantID]
				if !ok {
					return RunResult{}, fmt.Errorf("failed to get grant with id %s", grantID)
				}
				balancesAtPhaseStart.Set(grant.ID, grant.RecurrenceBalance(balancesAtPhaseStart[grantID]))
			}
		}

		// get active and inactive grants in the phase
		activeGrants := make([]grant.Grant, 0, len(grants))
		for _, grant := range grants {
			if grant.ActiveAt(phase.from) {
				activeGrants = append(activeGrants, grant)
			} else {
				// grants inactivating have 0 balance
				balancesAtPhaseStart[grant.ID] = 0
			}
		}

		// if a grant becomes active at the start of this period then their balance becomes the full amount
		for _, grant := range activeGrants {
			if grant.EffectiveAt.Equal(phase.from) {
				balancesAtPhaseStart[grant.ID] = grant.Amount
			}
		}

		// If we usae LATEST aggregation, Grant Amounts are treated as "limits" instead of as "budgets",
		// so we always deduct the point-in-time values from the original granted amounts
		if params.Meter.Aggregation == meter.MeterAggregationLatest {
			for _, grant := range activeGrants {
				balancesAtPhaseStart[grant.ID] = grant.Amount
			}
		}

		segment := GrantBurnDownHistorySegment{
			ClosedPeriod:   timeutil.ClosedPeriod{From: phase.from, To: phase.to},
			BalanceAtStart: balancesAtPhaseStart.Clone(),
			OverageAtStart: overage,
			TerminationReasons: SegmentTerminationReason{
				PriorityChange: phase.priorityChange,
				Recurrence:     phase.grantsRecurredAtEnd,
			},
		}

		// query feature usage in the burning phase
		usage, err := e.QueryUsage(ctx, phase.from, phase.to)
		if err != nil {
			return RunResult{}, fmt.Errorf("failed to get feature usage for period %s - %s: %w", period.From, period.To, err)
		}
		balancesAtPhaseStart, segment.GrantUsages, overage = e.burnDownGrants(balancesAtPhaseStart, activeGrants, usage+overage)

		segment.TotalUsage = usage
		segment.Overage = overage

		segments = append(segments, segment)

		// check if priority changed or grants need to recurr
		if phase.priorityChange {
			rePrioritize = true
		}
		if len(phase.grantsRecurredAtEnd) > 0 {
			recurredGrants = phase.grantsRecurredAtEnd
		}
	}

	history, err := NewGrantBurnDownHistory(segments, params.StartingSnapshot.Usage)
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to create grant burn down history: %w", err)
	}

	return RunResult{
		Snapshot: balance.Snapshot{
			Balances: balancesAtPhaseStart,
			Overage:  overage,
			At:       period.To,
			Usage: balance.SnapshottedUsage{
				Since: params.StartingSnapshot.Usage.Since,
				Usage: params.StartingSnapshot.Usage.Usage + history.TotalUsageInHistory(),
			},
		},
		History: history,
	}, nil
}

// Burns down the grants of the priority sorted list. Manages overage.
//
// FIXME: calculations happen on inexact representations as float64, this can lead to rounding errors.
func (m *engine) burnDownGrants(startingBalances balance.Map, prioritized []grant.Grant, usage float64) (balance.Map, []GrantUsage, float64) {
	balances := startingBalances.Clone()
	uses := make([]GrantUsage, 0, len(prioritized))
	exactUsage := alpacadecimal.NewFromFloat(usage)

	getFloat := func(d alpacadecimal.Decimal) float64 {
		return d.InexactFloat64()
	}

	for _, grant := range prioritized {
		grantBalance := balances[grant.ID]
		// if grant has no balance, skip
		if grantBalance == 0 {
			continue
		}
		exactBalance := alpacadecimal.NewFromFloat(grantBalance)
		// if grant balance is less than usage, burn the grant and subtract the balance from usage
		if exactBalance.LessThanOrEqual(exactUsage) {
			balances.Set(grant.ID, 0) // 0 usage to avoid arithmetic errors
			exactUsage = exactUsage.Sub(exactBalance)
			uses = append(uses, GrantUsage{
				GrantID:           grant.ID,
				Usage:             grantBalance,
				TerminationReason: GrantUsageTerminationReasonExhausted,
			})
			// if grant balance is more than usage, burn the grant with the usage
		} else {
			balances.Burn(grant.ID, getFloat(exactUsage))
			uses = append(uses, GrantUsage{
				GrantID:           grant.ID,
				Usage:             getFloat(exactUsage),
				TerminationReason: GrantUsageTerminationReasonSegmentTermination,
			})
			exactUsage = alpacadecimal.NewFromFloat(0)
		}
	}

	return balances, uses, getFloat(exactUsage)
}
