package credit

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type ResetUsageForOwnerParams struct {
	At              time.Time
	RetainAnchor    bool
	PreserveOverage bool
}

// Generic connector for balance related operations.
type BalanceConnector interface {
	GetBalanceOfOwner(ctx context.Context, owner grant.NamespacedOwner, at time.Time) (*balance.Snapshot, error)
	GetBalanceHistoryOfOwner(ctx context.Context, owner grant.NamespacedOwner, params BalanceHistoryParams) (engine.GrantBurnDownHistory, error)
	ResetUsageForOwner(ctx context.Context, owner grant.NamespacedOwner, params ResetUsageForOwnerParams) (balanceAfterReset *balance.Snapshot, err error)
}

type BalanceHistoryParams struct {
	From time.Time
	To   time.Time
}

var _ BalanceConnector = &connector{}

func (m *connector) GetBalanceOfOwner(ctx context.Context, owner grant.NamespacedOwner, at time.Time) (*balance.Snapshot, error) {
	// To include the current last minute lets round it trunc to the next minute
	if trunc := at.Truncate(time.Minute); trunc.Before(at) {
		at = trunc.Add(time.Minute)
	}

	// get last valid grantbalances
	bal, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if bal.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
		return nil, fmt.Errorf("last valid balance snapshot %s is before current period start at %s, no snapshot was created for reset", bal.At, periodStart)
	}

	// get all relevant grants
	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, bal.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the current period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&bal, grants, bal.At)

	// Let's define the period the engine will be queried for
	queriedPeriod := recurrence.Period{
		From: bal.At,
		To:   at,
	}

	eng, err := m.buildEngineForOwner(ctx, owner, queriedPeriod)
	if err != nil {
		return nil, err
	}

	result, overage, segments, err := eng.Run(
		ctx,
		grants,
		bal.Balances,
		bal.Overage,
		queriedPeriod,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, at, err)
	}

	history, err := engine.NewGrantBurnDownHistory(segments)
	if err != nil {
		return nil, fmt.Errorf("failed to create grant burn down history: %w", err)
	}

	// FIXME: It can be the case that we never actually save anything if the history has a single segment.
	// In practice what we can save is the balance at the last activation or recurrence event
	// as those demark segments.
	//
	// If we want to we can cheat that by artificially introducing a segment through the engine at the end
	// just so it can be saved...
	//
	// FIXME: we should do this comparison not with the queried time but the current time...
	if snap, err := m.getLastSaveableSnapshotAt(history, bal, at); err == nil {
		grantMap := make(map[string]grant.Grant, len(grants))
		for _, grant := range grants {
			grantMap[grant.ID] = grant
		}
		activeBalance, err := m.excludeInactiveGrantsFromBalance(snap.Balances, grantMap, at)
		if err != nil {
			return nil, err
		}
		snap.Balances = *activeBalance
		err = m.balanceSnapshotRepo.Save(ctx, owner, []balance.Snapshot{
			*snap,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save balance for owner %s at %s: %w", owner.ID, at, err)
		}
	}

	// return balance
	return &balance.Snapshot{
		At:       at,
		Balances: result,
		Overage:  overage,
	}, nil
}

// Returns the joined GrantBurnDownHistory across usage periods.
func (m *connector) GetBalanceHistoryOfOwner(ctx context.Context, owner grant.NamespacedOwner, params BalanceHistoryParams) (engine.GrantBurnDownHistory, error) {
	// To include the current last minute lets round it trunc to the next minute
	if trunc := params.To.Truncate(time.Minute); trunc.Before(params.To) {
		params.To = trunc.Add(time.Minute)
	}
	// get all usage resets between queryied period
	startTimes, err := m.ownerConnector.GetPeriodStartTimesBetween(ctx, owner, params.From, params.To)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get period start times between %s and %s for owner %s: %w", params.From, params.To, owner.ID, err)
	}
	times := []time.Time{params.From}
	times = append(times, startTimes...)
	times = append(times, params.To)

	periods := SortedPeriodsFromDedupedTimes(times)
	historySegments := make([]engine.GrantBurnDownHistorySegment, 0, len(periods))

	// For each period we'll have to calculate separately as we cannot calculate across resets.
	// For each period, we will:
	// 1. Find the last valid snapshot before the period start (might be at or before the period start)
	// 2. Calculate the balance at the period start
	// 3. Calculate the balance through the period
	for _, period := range periods {
		// Get last valid BalanceSnapshot before (or at) the period start
		snap, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, period.From)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}

		if period.From.Before(snap.At) {
			// This is an inconsistency check. It can only happen if we lost our snapshot for the reset.
			//
			// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
			// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
			return engine.GrantBurnDownHistory{}, fmt.Errorf("current period start %s is before last valid balance snapshot at %s, no snapshot was created for reset", period.From, snap.At)
		}

		// First, let's calculate the balance from the last snapshot until the start of the period

		// get all relevant grants
		grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, snap.At, period.From)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}

		// These grants might not be present in the starting balance so lets fill them
		// This is only possible in case the grant becomes active exactly at the start of the current period
		m.populateBalanceSnapshotWithMissingGrantsActiveAt(&snap, grants, snap.At)

		periodFromSnapshotToPeriodStart := recurrence.Period{
			From: snap.At,
			To:   period.From,
		}

		eng, err := m.buildEngineForOwner(ctx, owner, periodFromSnapshotToPeriodStart)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}

		balances, overage, _, err := eng.Run(
			ctx,
			grants,
			snap.Balances,
			snap.Overage,
			periodFromSnapshotToPeriodStart,
		)
		if err != nil {
			return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, period.From, err)
		}

		fakeSnapshotForPeriodStart := balance.Snapshot{
			Balances: balances,
			Overage:  overage,
			At:       period.From,
		}

		// Second, lets calculate the balance for the period

		// get all relevant grants
		grants, err = m.grantRepo.ListActiveGrantsBetween(ctx, owner, period.From, period.To)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}

		// These grants might not be present in the starting balance so lets fill them
		// This is only possible in case the grant becomes active exactly at the start of the current period
		m.populateBalanceSnapshotWithMissingGrantsActiveAt(&fakeSnapshotForPeriodStart, grants, period.From)

		eng, err = m.buildEngineForOwner(ctx, owner, period)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}

		_, _, segments, err := eng.Run(
			ctx,
			grants,
			fakeSnapshotForPeriodStart.Balances,
			fakeSnapshotForPeriodStart.Overage,
			period,
		)
		if err != nil {
			return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, period.To, err)
		}

		// set reset as reason for last segment if current period end is a reset
		if slices.Contains(startTimes, period.To) {
			segments[len(segments)-1].TerminationReasons.UsageReset = true
		}

		historySegments = append(historySegments, segments...)
	}

	// return history
	history, err := engine.NewGrantBurnDownHistory(historySegments)
	if err != nil || history == nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to create grant burn down history: %w", err)
	}
	return *history, err
}

func (m *connector) ResetUsageForOwner(ctx context.Context, owner grant.NamespacedOwner, params ResetUsageForOwnerParams) (*balance.Snapshot, error) {
	// Cannot reset for the future
	if params.At.After(clock.Now()) {
		return nil, &models.GenericUserError{Inner: fmt.Errorf("cannot reset at %s in the future", params.At)}
	}

	ownerMeter, err := m.ownerConnector.GetMeter(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner query params for owner %s: %w", owner.ID, err)
	}

	at := params.At.Truncate(ownerMeter.Meter.WindowSize.Duration())

	// check if reset is possible (after last reset)
	periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, clock.Now())
	if err != nil {
		if _, ok := err.(*grant.OwnerNotFoundError); ok {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if at.Before(periodStart) {
		return nil, &models.GenericUserError{Inner: fmt.Errorf("reset at %s is before current usage period start %s", at, periodStart)}
	}

	bal, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	if bal.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
		// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
		return nil, fmt.Errorf("last valid balance snapshot %s is before current period start at %s, no snapshot was created for reset", bal.At, periodStart)
	}

	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, bal.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&bal, grants, bal.At)

	// Let's define the period the engine will be queried for
	queriedPeriod := recurrence.Period{
		From: bal.At,
		To:   at,
	}

	eng, err := m.buildEngineForOwner(ctx, owner, queriedPeriod)
	if err != nil {
		return nil, err
	}

	endingBalance, endingOverage, _, err := eng.Run(
		ctx,
		grants,
		bal.Balances,
		bal.Overage,
		queriedPeriod,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for reset: %w", err)
	}

	grantMap := make(map[string]grant.Grant, len(grants))
	for _, grant := range grants {
		grantMap[grant.ID] = grant
	}

	// We have to roll over the grants and save the starting balance for the next period at the reset time.
	// Engine treates the output balance as a period end (exclusive), but we need to treat it as a period start (inclusive).
	startingBalance := balance.Map{}
	for grantID, grantBalance := range endingBalance {
		grant, ok := grantMap[grantID]
		// inconsistency check, shouldn't happen
		if !ok {
			return nil, fmt.Errorf("attempting to roll over unknown grant %s", grantID)
		}

		// grants might become inactive at the reset time, in which case they're irrelevant for the next period
		if !grant.ActiveAt(at) {
			continue
		}

		startingBalance.Set(grantID, grant.RolloverBalance(grantBalance))
	}

	startingOverage := 0.0
	if params.PreserveOverage {
		startingOverage = endingOverage
	}

	gCopy := make([]grant.Grant, len(grants))
	copy(gCopy, grants)
	err = engine.PrioritizeGrants(gCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to burn down overage from previous period: failed to prioritize grants: %w", err)
	}

	startingBalance, _, startingOverage, err = engine.BurnDownGrants(startingBalance, grants, startingOverage)
	if err != nil {
		return nil, fmt.Errorf("failed to burn down overage from previous period: %w", err)
	}

	startingSnapshot := balance.Snapshot{
		At:       at,
		Balances: startingBalance,
		Overage:  startingOverage,
	}

	_, err = transaction.Run(ctx, m.transactionManager, func(ctx context.Context) (*balance.Snapshot, error) {
		//lint:ignore SA1019 we need to use the transaction here
		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return nil, err
		}

		err = m.ownerConnector.LockOwnerForTx(ctx, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to lock owner %s: %w", owner.ID, err)
		}

		err = m.balanceSnapshotRepo.WithTx(ctx, tx).Save(ctx, owner, []balance.Snapshot{startingSnapshot})
		if err != nil {
			return nil, fmt.Errorf("failed to save balance for owner %s at %s: %w", owner.ID, at, err)
		}

		err = m.ownerConnector.EndCurrentUsagePeriod(ctx, owner, grant.EndCurrentUsagePeriodParams{
			At:           at,
			RetainAnchor: params.RetainAnchor,
		})
		if err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	return &startingSnapshot, nil
}
