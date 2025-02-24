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
	m.logger.Debug("getting balance of owner", "owner", owner.ID, "at", at)

	// To include the current last minute lets round it trunc to the next minute
	if trunc := at.Truncate(time.Minute); trunc.Before(at) {
		at = trunc.Add(time.Minute)
	}

	// get last valid grantbalances
	snap, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	period := timeutil.Period{
		From: snap.At,
		To:   at,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.ownerConnector.GetResetTimelineInclusive(ctx, owner, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", period.From, period.To, owner.ID, err)
	}

	resetBehavior, err := m.ownerConnector.GetResetBehavior(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get reset behavior for owner %s: %w", owner.ID, err)
	}

	// get all relevant grants
	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, snap.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the current period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&snap, grants, snap.At)

	eng, err := m.buildEngineForOwner(ctx, owner, period)
	if err != nil {
		return nil, err
	}

	result, err := eng.Run(
		ctx,
		engine.RunParams{
			Grants:           grants,
			StartingSnapshot: snap,
			Until:            period.To,
			ResetBehavior:    resetBehavior,
			Resets:           resetTimesInclusive.After(snap.At),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, at, err)
	}

	// Let's see if a snapshot should be saved
	if err := m.snapshotEngineResult(ctx, snapshotParams{
		grants: grants,
		owner:  owner,
		runRes: result,
		before: clock.Now().AddDate(0, 0, -7), // 7 days ago
	}); err != nil {
		return nil, fmt.Errorf("failed to snapshot engine result: %w", err)
	}

	// return balance
	return &result.Snapshot, nil
}

// Returns the joined GrantBurnDownHistory across usage periods.
func (m *connector) GetBalanceHistoryOfOwner(ctx context.Context, owner grant.NamespacedOwner, params BalanceHistoryParams) (engine.GrantBurnDownHistory, error) {
	// To include the current last minute lets round it trunc to the next minute
	if trunc := params.To.Truncate(time.Minute); trunc.Before(params.To) {
		params.To = trunc.Add(time.Minute)
	}

	period := timeutil.Period{
		From: params.From,
		To:   params.To,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.ownerConnector.GetResetTimelineInclusive(ctx, owner, period)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", params.From, params.To, owner.ID, err)
	}

	resetBehavior, err := m.ownerConnector.GetResetBehavior(ctx, owner)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get reset behavior for owner %s: %w", owner.ID, err)
	}

	// For the history result to start from the correct period start we need to start from a synthetic snapshot by calculating the balance at the period start
	snap, err := m.GetBalanceOfOwner(ctx, owner, period.From)
	if err != nil {
		return engine.GrantBurnDownHistory{}, err
	}

	// get all relevant grants
	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, snap.At, period.To)
	if err != nil {
		return engine.GrantBurnDownHistory{}, err
	}

	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the first period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(snap, grants, snap.At)

	eng, err := m.buildEngineForOwner(ctx, owner, period)
	if err != nil {
		return engine.GrantBurnDownHistory{}, err
	}

	result, err := eng.Run(
		ctx,
		engine.RunParams{
			Grants:           grants,
			StartingSnapshot: *snap,
			Until:            period.To,
			ResetBehavior:    resetBehavior,
			Resets:           resetTimesInclusive.After(snap.At),
		},
	)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, period.From, err)
	}

	// return history
	history, err := engine.NewGrantBurnDownHistory(result.History)
	if err != nil || history == nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to create grant burn down history: %w", err)
	}
	return *history, err
}

func (m *connector) ResetUsageForOwner(ctx context.Context, owner grant.NamespacedOwner, params ResetUsageForOwnerParams) (*balance.Snapshot, error) {
	// Cannot reset for the future
	if params.At.After(clock.Now()) {
		return nil, models.NewGenericValidationError(fmt.Errorf("cannot reset at %s in the future", params.At))
	}

	ownerMeter, err := m.ownerConnector.GetMeter(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner query params for owner %s: %w", owner.ID, err)
	}

	at := params.At.Truncate(ownerMeter.Meter.WindowSize.Duration())

	// check if reset is possible (not before current period)
	periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, clock.Now())
	if err != nil {
		if _, ok := err.(*grant.OwnerNotFoundError); ok {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if at.Before(periodStart) {
		return nil, models.NewGenericValidationError(fmt.Errorf("reset at %s is before current usage period start %s", at, periodStart))
	}

	bal, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, bal.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&bal, grants, bal.At)

	// Let's define the period the engine will be queried for
	queriedPeriod := timeutil.Period{
		From: bal.At,
		To:   at,
	}

	eng, err := m.buildEngineForOwner(ctx, owner, queriedPeriod)
	if err != nil {
		return nil, err
	}

	res, err := eng.Run(
		ctx,
		engine.RunParams{
			Grants:           grants,
			StartingSnapshot: bal,
			Until:            queriedPeriod.To,
		},
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
	for grantID, grantBalance := range res.Snapshot.Balances {
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
		startingOverage = res.Snapshot.Overage
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
