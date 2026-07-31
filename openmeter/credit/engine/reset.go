package engine

import (
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// reset rolls over the grants and burns down the overage if needed.
// It returns the new snapshot of the balances at the start of the next period.
func (e *engine) reset(grants []grant.Grant, snap balance.Snapshot, behavior grant.ResetBehavior, at time.Time) (balance.Snapshot, GrantBurnDownHistorySegment, error) {
	// Let's build a grantMap from our grants for easier lookup
	grantMap := make(map[string]grant.Grant)
	for _, g := range grants {
		grantMap[g.ID] = g
	}

	// First, we roll over the grants
	rolledOver := snap.Balances.Clone()

	for grantID, grantBalance := range rolledOver {
		grant, ok := grantMap[grantID]
		// Inconsistency check, should never happen
		if !ok {
			return balance.Snapshot{}, GrantBurnDownHistorySegment{}, fmt.Errorf("grant %s not found", grantID)
		}

		// grants might become inactive at the reset time, in which case they're irrelevant for the next period
		if !grant.ActiveAt(at) {
			continue
		}

		rolledOver[grantID] = grant.RolloverBalance(grantBalance)
	}

	// Then if needed, we burn down the overage
	startingOverage := 0.0
	if behavior.PreserveOverage {
		startingOverage = snap.Overage
	}

	prioritizedGrants := slices.Clone(grants)
	if err := PrioritizeGrants(prioritizedGrants); err != nil {
		return balance.Snapshot{}, GrantBurnDownHistorySegment{}, fmt.Errorf("failed to prioritize grants: %w", err)
	}

	balances := rolledOver
	overage := startingOverage
	var grantUsages []GrantUsage
	if startingOverage != 0 {
		balances, grantUsages, overage = e.burnDownGrants(rolledOver, prioritizedGrants, startingOverage)
	}

	// The reset snapshot is the point-in-time balance after both rollover and
	// settlement of preserved overage.
	resetSnapshot := balance.Snapshot{
		At:       at,
		Balances: balances,
		Overage:  overage,
	}

	// The rollover segment captures the instantaneous transition from rolled-over
	// balances to the reset snapshot.
	rolloverSegment := GrantBurnDownHistorySegment{
		ClosedPeriod:   timeutil.ClosedPeriod{From: at, To: at},
		BalanceAtStart: rolledOver,
		TerminationReasons: SegmentTerminationReason{
			Rollover: true,
		},
		TotalUsage:     0,
		OverageAtStart: startingOverage,
		Overage:        overage,
		GrantUsages:    grantUsages,
	}

	return resetSnapshot, rolloverSegment, nil
}
