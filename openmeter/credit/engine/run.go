package engine

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Burns down all grants in the defined period by the usage amounts.
//
// When the engine outputs a balance, it doesn't discriminate what should be in that balance.
// If a grant is inactive at the end of the period, it will still be in the output.
func (e *engine) Run(ctx context.Context, grants []grant.Grant, startingBalances balance.Map, overage float64, period timeutil.Period) (balance.Map, float64, []GrantBurnDownHistorySegment, error) {
	if !startingBalances.ExactlyForGrants(grants) {
		return nil, 0, nil, fmt.Errorf("provided grants and balances don't pair up, grants: %+v, balances: %+v", grants, startingBalances)
	}

	e.grants = grants
	phases, err := e.getPhases(period)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get burn phases: %w", err)
	}

	err = PrioritizeGrants(e.grants)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to prioritize grants: %w", err)
	}

	// Only respect balances that we know the grants of, otherwise we cannot guarantee
	// that the output balance is correct for said grants.
	balancesAtPhaseStart := startingBalances.Copy()

	rePrioritize := false
	recurredGrants := []string{}

	grantMap := make(map[string]grant.Grant)
	for _, grant := range e.grants {
		grantMap[grant.ID] = grant
	}

	segments := make([]GrantBurnDownHistorySegment, 0, len(phases))

	for _, phase := range phases {
		// reprioritize grants if needed
		if rePrioritize {
			err = PrioritizeGrants(e.grants)
			if err != nil {
				return nil, 0, nil, fmt.Errorf("failed to prioritize grants: %w", err)
			}
			rePrioritize = false
		}

		// reset recurring grant balances
		if len(recurredGrants) > 0 {
			for _, grantID := range recurredGrants {
				grant, ok := grantMap[grantID]
				if !ok {
					return nil, 0, nil, fmt.Errorf("failed to get grant with id %s", grantID)
				}
				balancesAtPhaseStart.Set(grant.ID, grant.RecurrenceBalance(balancesAtPhaseStart[grantID]))
			}
		}

		// get active and inactive grants in the phase
		activeGrants := make([]grant.Grant, 0, len(e.grants))
		for _, grant := range e.grants {
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

		segment := GrantBurnDownHistorySegment{
			Period:         timeutil.Period{From: phase.from, To: phase.to},
			BalanceAtStart: balancesAtPhaseStart.Copy(),
			OverageAtStart: overage,
			TerminationReasons: SegmentTerminationReason{
				PriorityChange: phase.priorityChange,
				Recurrence:     phase.grantsRecurredAtEnd,
			},
		}

		// query feature usage in the burning phase
		usage, err := e.QueryUsage(ctx, phase.from, phase.to)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to get feature usage for period %s - %s: %w", period.From, period.To, err)
		}
		balancesAtPhaseStart, segment.GrantUsages, overage, err = BurnDownGrants(balancesAtPhaseStart, activeGrants, usage+overage)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to burn down grants in period %s - %s: %w", period.From, period.To, err)
		}

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
	return balancesAtPhaseStart, overage, segments, nil
}

// Burns down the grants of the priority sorted list. Manages overage.
//
// FIXME: calculations happen on inexact representations as float64, this can lead to rounding errors.
func BurnDownGrants(startingBalances balance.Map, prioritized []grant.Grant, usage float64) (balance.Map, []GrantUsage, float64, error) {
	balances := startingBalances.Copy()
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

	return balances, uses, getFloat(exactUsage), nil
}
