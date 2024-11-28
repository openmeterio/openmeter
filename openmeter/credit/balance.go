package credit

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/samber/lo"

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

	ownerSubjectKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner subject key for owner %s: %w", owner.ID, err)
	}

	// run engine and calculate grantbalance
	engineParams, err := m.getQueryUsageFn(ctx, owner, ownerSubjectKey)
	if err != nil {
		return nil, err
	}
	eng := engine.NewEngine(engineParams.QueryUsageFn, engineParams.Grantuality)

	result, overage, segments, err := eng.Run(
		ctx,
		grants,
		bal.Balances,
		bal.Overage,
		recurrence.Period{
			From: bal.At,
			To:   at,
		},
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

	ownerSubjecKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
	if err != nil {
		return engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get owner subject key for owner %s: %w", owner.ID, err)
	}

	// collect al history segments through all periods
	for _, period := range periods {
		// get last valid grantbalances at start of period (eq balance at start of period)
		balance, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, period.From)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}

		if period.From.Before(balance.At) {
			// This is an inconsistency check. It can only happen if we lost our snapshot for the reset.
			//
			// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
			// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
			return engine.GrantBurnDownHistory{}, fmt.Errorf("current period start %s is before last valid balance snapshot at %s, no snapshot was created for reset", period.From, balance.At)
		}

		// get all relevant grants
		grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, period.From, period.To)
		// These grants might not be present in the starting balance so lets fill them
		// This is only possible in case the grant becomes active exactly at the start of the current period
		m.populateBalanceSnapshotWithMissingGrantsActiveAt(&balance, grants, period.From)

		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}
		// run engine and calculate grantbalance
		engineParams, err := m.getQueryUsageFn(ctx, owner, ownerSubjecKey)
		if err != nil {
			return engine.GrantBurnDownHistory{}, err
		}
		eng := engine.NewEngine(engineParams.QueryUsageFn, engineParams.Grantuality)

		_, _, segments, err := eng.Run(
			ctx,
			grants,
			balance.Balances,
			balance.Overage,
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
		return nil, &models.GenericUserError{Message: fmt.Sprintf("cannot reset at %s in the future", params.At)}
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
		return nil, &models.GenericUserError{Message: fmt.Sprintf("reset at %s is before current usage period start %s", at, periodStart)}
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

	engineParams, err := m.getQueryUsageFn(ctx, owner, ownerMeter.SubjectKey)
	if err != nil {
		return nil, err
	}
	eng := engine.NewEngine(engineParams.QueryUsageFn, engineParams.Grantuality)

	endingBalance, endingOverage, _, err := eng.Run(
		ctx,
		grants,
		bal.Balances,
		bal.Overage,
		recurrence.Period{
			From: bal.At,
			To:   at,
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

// Fetches the last valid snapshot for an owner.
//
// If no snapshot exists returns a default snapshot for measurement start to recalculate the entire history
// in case no usable snapshot was found.
func (m *connector) getLastValidBalanceSnapshotForOwnerAt(ctx context.Context, owner grant.NamespacedOwner, at time.Time) (balance.Snapshot, error) {
	bal, err := m.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, at)
	if err != nil {
		if _, ok := err.(*balance.NoSavedBalanceForOwnerError); ok {
			// if no snapshot is found we have to calculate from start of time on all grants and usage
			m.logger.Debug(fmt.Sprintf("no saved balance found for owner %s before %s, calculating from start of time", owner.ID, at))

			startOfMeasurement, err := m.ownerConnector.GetStartOfMeasurement(ctx, owner)
			if err != nil {
				return bal, err
			}

			grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, startOfMeasurement, at)
			if err != nil {
				return bal, err
			}

			balances := balance.Map{}
			for _, grant := range grants {
				if grant.ActiveAt(startOfMeasurement) {
					// Grants that are active at the start will have full balance
					balances.Set(grant.ID, grant.Amount)
				} else {
					// Grants that are not active at the start won't have a balance
					balances.Set(grant.ID, 0.0)
				}
			}

			bal = balance.Snapshot{
				At:       startOfMeasurement,
				Balances: balances,
				Overage:  0.0, // There cannot be overage at the start of measurement
			}
		} else {
			return bal, fmt.Errorf("failed to get latest valid grant balance at %s for owner %s: %w", at, owner.ID, err)
		}
	}

	return bal, nil
}

type engineParams struct {
	QueryUsageFn engine.QueryUsageFn
	Grantuality  models.WindowSize
}

// returns owner specific QueryUsageFn
func (m *connector) getQueryUsageFn(ctx context.Context, owner grant.NamespacedOwner, subjectKey string) (*engineParams, error) {
	ownerMeter, err := m.ownerConnector.GetMeter(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get query params for owner %v: %w", owner, err)
	}
	return &engineParams{
		QueryUsageFn: func(ctx context.Context, from, to time.Time) (float64, error) {
			// copy
			params := ownerMeter.DefaultParams
			params.From = &from
			params.To = &to
			params.FilterSubject = []string{subjectKey}
			rows, err := m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.Meter, params)
			if err != nil {
				return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.Meter.Slug, err)
			}
			if len(rows) > 1 {
				return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
			}
			if len(rows) == 0 {
				return 0.0, nil
			}
			return rows[0].Value, nil
		},
		Grantuality: ownerMeter.Meter.WindowSize,
	}, nil
}

// Returns a snapshot from the last segment that can be saved, taking the following into account:
//
//  1. We can save a segment if it is older than graceperiod.
//  2. At the end of a segment history changes: s1.endBalance <> s2.startBalance. This means only the
//     starting values can be saved credibly.
func (m *connector) getLastSaveableSnapshotAt(history *engine.GrantBurnDownHistory, lastValidBalance balance.Snapshot, at time.Time) (*balance.Snapshot, error) {
	segments := history.Segments()

	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		if segment.From.Add(m.snapshotGracePeriod).Before(at) {
			s := segment.ToSnapshot()
			if s.At.After(lastValidBalance.At) {
				return &s, nil
			} else {
				return nil, fmt.Errorf("the last saveable snapshot at %s is before the previous last valid snapshot", s.At)
			}
		}
	}

	return nil, fmt.Errorf("no segment can be saved at %s with gracePeriod %s", at, m.snapshotGracePeriod)
}

func (m *connector) excludeInactiveGrantsFromBalance(balances balance.Map, grants map[string]grant.Grant, at time.Time) (*balance.Map, error) {
	filtered := &balance.Map{}
	for grantID, grantBalance := range balances {
		grant, ok := grants[grantID]
		// inconsistency check, shouldn't happen
		if !ok {
			return nil, fmt.Errorf("attempting to roll over unknown grant %s", grantID)
		}

		// grants might become inactive at the reset time, in which case they're irrelevant for the next period
		if !grant.ActiveAt(at) {
			continue
		}

		filtered.Set(grantID, grantBalance)
	}
	return filtered, nil
}

// Fills in the snapshot's GrantBalanceMap with the provided grants so the Engine can use them.
func (m *connector) populateBalanceSnapshotWithMissingGrantsActiveAt(snapshot *balance.Snapshot, grants []grant.Grant, at time.Time) {
	for _, grant := range grants {
		if _, ok := snapshot.Balances[grant.ID]; !ok {
			if grant.ActiveAt(at) {
				snapshot.Balances.Set(grant.ID, grant.Amount)
			} else {
				snapshot.Balances.Set(grant.ID, 0.0)
			}
		}
	}
}

// Returns a list of non-overlapping periods between the sorted times.
func SortedPeriodsFromDedupedTimes(ts []time.Time) []recurrence.Period {
	if len(ts) < 2 {
		return nil
	}

	times := lo.Uniq(ts)

	// sort
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	periods := make([]recurrence.Period, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		periods = append(periods, recurrence.Period{From: times[i-1], To: times[i]})
	}

	return periods
}
