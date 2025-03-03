package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ResetUsageForOwnerParams struct {
	At              time.Time
	RetainAnchor    bool
	PreserveOverage bool
}

// Generic connector for balance related operations.
type BalanceConnector interface {
	GetBalanceOfOwner(ctx context.Context, owner models.NamespacedID, at time.Time) (*balance.Snapshot, error)
	GetBalanceHistoryOfOwner(ctx context.Context, owner models.NamespacedID, params BalanceHistoryParams) (engine.GrantBurnDownHistory, error)
	ResetUsageForOwner(ctx context.Context, owner models.NamespacedID, params ResetUsageForOwnerParams) (balanceAfterReset *balance.Snapshot, err error)
}

type BalanceHistoryParams struct {
	From time.Time
	To   time.Time
}

var _ BalanceConnector = &connector{}

func (m *connector) GetBalanceOfOwner(ctx context.Context, ownerID models.NamespacedID, at time.Time) (*balance.Snapshot, error) {
	m.logger.Debug("getting balance of owner", "owner", ownerID.ID, "at", at)

	// To include the current last minute lets round it trunc to the next minute
	if trunc := at.Truncate(time.Minute); trunc.Before(at) {
		at = trunc.Add(time.Minute)
	}

	// get last valid grantbalances
	snap, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, ownerID, at)
	if err != nil {
		return nil, err
	}

	period := timeutil.Period{
		From: snap.At,
		To:   at,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.ownerConnector.GetResetTimelineInclusive(ctx, ownerID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", period.From, period.To, ownerID.ID, err)
	}

	owner, err := m.ownerConnector.DescribeOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe owner %s: %w", owner.ID, err)
	}

	// get all relevant grants
	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, ownerID, snap.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, ownerID.ID, err)
	}
	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the current period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&snap, grants, snap.At)

	eng, err := m.buildEngineForOwner(ctx, ownerID, period)
	if err != nil {
		return nil, err
	}

	result, err := eng.Run(
		ctx,
		engine.RunParams{
			Grants:           grants,
			StartingSnapshot: snap,
			Until:            period.To,
			ResetBehavior:    owner.ResetBehavior,
			Resets:           resetTimesInclusive.After(snap.At),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", ownerID.ID, at, err)
	}

	// Let's see if a snapshot should be saved
	if err := m.snapshotEngineResult(ctx, snapshotParams{
		grants: grants,
		owner:  ownerID,
		runRes: result,
		before: m.getSnapshotBefore(clock.Now()),
	}); err != nil {
		return nil, fmt.Errorf("failed to snapshot engine result: %w", err)
	}

	// return balance
	return &result.Snapshot, nil
}

// Returns the joined GrantBurnDownHistory across usage periods.
func (m *connector) GetBalanceHistoryOfOwner(ctx context.Context, ownerID models.NamespacedID, params BalanceHistoryParams) (engine.GrantBurnDownHistory, error) {
	// To include the current last minute lets round it trunc to the next minute
	if trunc := params.To.Truncate(time.Minute); trunc.Before(params.To) {
		params.To = trunc.Add(time.Minute)
	}

	period := timeutil.Period{
		From: params.From,
		To:   params.To,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.ownerConnector.GetResetTimelineInclusive(ctx, ownerID, period)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", params.From, params.To, ownerID.ID, err)
	}

	owner, err := m.ownerConnector.DescribeOwner(ctx, ownerID)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to describe owner %s: %w", ownerID.ID, err)
	}

	// For the history result to start from the correct period start we need to start from a synthetic snapshot by calculating the balance at the period start
	snap, err := m.GetBalanceOfOwner(ctx, ownerID, period.From)
	if err != nil {
		return engine.GrantBurnDownHistory{}, err
	}

	// get all relevant grants
	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, ownerID, snap.At, period.To)
	if err != nil {
		return engine.GrantBurnDownHistory{}, err
	}

	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the first period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(snap, grants, snap.At)

	eng, err := m.buildEngineForOwner(ctx, ownerID, period)
	if err != nil {
		return engine.GrantBurnDownHistory{}, err
	}

	result, err := eng.Run(
		ctx,
		engine.RunParams{
			Grants:           grants,
			StartingSnapshot: *snap,
			Until:            period.To,
			ResetBehavior:    owner.ResetBehavior,
			Resets:           resetTimesInclusive.After(snap.At),
		},
	)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", ownerID.ID, period.From, err)
	}

	// return history
	history, err := engine.NewGrantBurnDownHistory(result.History)
	if err != nil || history == nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to create grant burn down history: %w", err)
	}
	return *history, err
}

func (m *connector) ResetUsageForOwner(ctx context.Context, ownerID models.NamespacedID, params ResetUsageForOwnerParams) (*balance.Snapshot, error) {
	// Cannot reset for the future
	if params.At.After(clock.Now()) {
		return nil, models.NewGenericValidationError(fmt.Errorf("cannot reset at %s in the future", params.At))
	}

	owner, err := m.ownerConnector.DescribeOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe owner %s: %w", ownerID.ID, err)
	}

	at := params.At.Truncate(owner.Meter.WindowSize.Duration())

	// check if reset is possible (not before current period)
	periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, ownerID, clock.Now())
	if err != nil {
		if _, ok := err.(*grant.OwnerNotFoundError); ok {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", ownerID.ID, at, err)
	}
	if at.Before(periodStart) {
		return nil, models.NewGenericValidationError(fmt.Errorf("reset at %s is before current usage period start %s", at, periodStart))
	}

	resetsSinceTime, err := m.ownerConnector.GetResetTimelineInclusive(ctx, ownerID, timeutil.Period{From: at, To: clock.Now()})
	if err != nil {
		return nil, fmt.Errorf("failed to get reset times since %s for owner %s: %w", at, ownerID.ID, err)
	}

	if rts := resetsSinceTime.After(at).GetTimes(); len(rts) > 0 {
		lastReset := rts[len(rts)-1]
		return nil, models.NewGenericValidationError(fmt.Errorf("reset at %s is before last reset at %s", at, lastReset))
	}

	bal, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, ownerID, at)
	if err != nil {
		return nil, err
	}

	period := timeutil.Period{
		From: bal.At,
		To:   at,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.ownerConnector.GetResetTimelineInclusive(ctx, ownerID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", period.From, period.To, ownerID.ID, err)
	}

	// Let's also add the at time to the resets
	resetTimes := append(resetTimesInclusive.GetTimes(), at)
	resetTimeline := timeutil.NewSimpleTimeline(resetTimes)

	// This gets overwritten by the inputs
	resetBehavior := owner.ResetBehavior
	resetBehavior.PreserveOverage = params.PreserveOverage

	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, ownerID, bal.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, ownerID.ID, err)
	}
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&bal, grants, bal.At)

	// Let's define the period the engine will be queried for
	queriedPeriod := timeutil.Period{
		From: bal.At,
		To:   at,
	}

	eng, err := m.buildEngineForOwner(ctx, ownerID, queriedPeriod)
	if err != nil {
		return nil, err
	}

	res, err := eng.Run(
		ctx,
		engine.RunParams{
			Grants:           grants,
			StartingSnapshot: bal,
			Until:            queriedPeriod.To,
			ResetBehavior:    resetBehavior,
			Resets:           resetTimeline.After(bal.At),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for reset: %w", err)
	}

	// Some grants in the snapshot might have been terminated at the reset time, in which case they're irrelevant for the next period
	startingSnap := res.Snapshot
	err = m.removeInactiveGrantsFromSnapshotAt(&startingSnap, grants, at)
	if err != nil {
		return nil, fmt.Errorf("failed to remove inactive grants from snapshot: %w", err)
	}

	_, err = transaction.Run(ctx, m.transactionManager, func(ctx context.Context) (*balance.Snapshot, error) {
		//lint:ignore SA1019 we need to use the transaction here
		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return nil, err
		}

		err = m.ownerConnector.LockOwnerForTx(ctx, ownerID)
		if err != nil {
			return nil, fmt.Errorf("failed to lock owner %s: %w", ownerID.ID, err)
		}

		err = m.balanceSnapshotRepo.WithTx(ctx, tx).Save(ctx, ownerID, []balance.Snapshot{startingSnap})
		if err != nil {
			return nil, fmt.Errorf("failed to save balance for owner %s at %s: %w", ownerID.ID, at, err)
		}

		err = m.ownerConnector.EndCurrentUsagePeriod(ctx, ownerID, grant.EndCurrentUsagePeriodParams{
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

	return &startingSnap, nil
}
