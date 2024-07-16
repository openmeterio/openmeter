package credit

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"time"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type ResetUsageForOwnerParams struct {
	At           time.Time
	RetainAnchor bool
}

// Generic connector for balance related operations.
type BalanceConnector interface {
	GetBalanceOfOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (*GrantBalanceSnapshot, error)
	GetBalanceHistoryOfOwner(ctx context.Context, owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error)
	ResetUsageForOwner(ctx context.Context, owner NamespacedGrantOwner, params ResetUsageForOwnerParams) (balanceAfterReset *GrantBalanceSnapshot, err error)
}

type BalanceHistoryParams struct {
	From time.Time
	To   time.Time
}

func NewBalanceConnector(
	grantRepo GrantRepo,
	balanceSnapshotConnector BalanceSnapshotRepo,
	ownerConnector OwnerConnector,
	streamingConnector streaming.Connector,
	logger *slog.Logger,
) BalanceConnector {
	return &balanceConnector{
		grantRepo:                grantRepo,
		balanceSnapshotConnector: balanceSnapshotConnector,
		ownerConnector:           ownerConnector,
		streamingConnector:       streamingConnector,
		logger:                   logger,

		// TODO: make configurable
		snapshotGracePeriod: time.Hour,
	}
}

type balanceConnector struct {
	// grants and balance snapshots are managed in this same package
	grantRepo                GrantRepo
	balanceSnapshotConnector BalanceSnapshotRepo
	// external dependencies
	ownerConnector     OwnerConnector
	streamingConnector streaming.Connector
	logger             *slog.Logger

	snapshotGracePeriod time.Duration
}

var _ BalanceConnector = &balanceConnector{}

func (m *balanceConnector) GetBalanceOfOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (*GrantBalanceSnapshot, error) {
	// To include the current last minute lets round it trunc to the next minute
	if trunc := at.Truncate(time.Minute); trunc.Before(at) {
		at = trunc.Add(time.Minute)
	}

	// get last valid grantbalances
	balance, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if balance.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
		return nil, fmt.Errorf("last valid balance snapshot %s is before current period start at %s, no snapshot was created for reset", balance.At, periodStart)
	}

	// get all relevant grants
	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, balance.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the current period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&balance, grants, balance.At)

	ownerSubjectKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner subject key for owner %s: %w", owner.ID, err)
	}

	// run engine and calculate grantbalance
	engineParams, err := m.getQueryUsageFn(ctx, owner, ownerSubjectKey)
	if err != nil {
		return nil, err
	}
	engine := NewEngine(engineParams.QueryUsageFn, engineParams.Grantuality)

	result, overage, segments, err := engine.Run(
		ctx,
		grants,
		balance.Balances,
		balance.Overage,
		recurrence.Period{
			From: balance.At,
			To:   at,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, at, err)
	}

	history, err := NewGrantBurnDownHistory(segments)
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
	if snap, err := m.getLastSaveableSnapshotAt(history, balance, at); err == nil {
		grantMap := make(map[string]Grant, len(grants))
		for _, grant := range grants {
			grantMap[grant.ID] = grant
		}
		activeBalance, err := m.excludeInactiveGrantsFromBalance(snap.Balances, grantMap, at)
		if err != nil {
			return nil, err
		}
		snap.Balances = *activeBalance
		err = m.balanceSnapshotConnector.Save(ctx, owner, []GrantBalanceSnapshot{
			*snap,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save balance for owner %s at %s: %w", owner.ID, at, err)
		}
	}

	// return balance
	return &GrantBalanceSnapshot{
		At:       at,
		Balances: result,
		Overage:  overage,
	}, nil
}

// Returns the joined GrantBurnDownHistory across usage periods.
func (m *balanceConnector) GetBalanceHistoryOfOwner(ctx context.Context, owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error) {
	// To include the current last minute lets round it trunc to the next minute
	if trunc := params.To.Truncate(time.Minute); trunc.Before(params.To) {
		params.To = trunc.Add(time.Minute)
	}
	// get all usage resets between queryied period
	startTimes, err := m.ownerConnector.GetPeriodStartTimesBetween(ctx, owner, params.From, params.To)
	if err != nil {
		return GrantBurnDownHistory{}, fmt.Errorf("failed to get period start times between %s and %s for owner %s: %w", params.From, params.To, owner.ID, err)
	}
	times := []time.Time{params.From}
	times = append(times, startTimes...)
	times = append(times, params.To)

	periods := SortedPeriodsFromDedupedTimes(times)
	historySegments := make([]GrantBurnDownHistorySegment, 0, len(periods))

	ownerSubjecKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
	if err != nil {
		return GrantBurnDownHistory{}, fmt.Errorf("failed to get owner subject key for owner %s: %w", owner.ID, err)
	}

	// collect al history segments through all periods
	for _, period := range periods {
		// get last valid grantbalances at start of period (eq balance at start of period)
		balance, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, period.From)
		if err != nil {
			return GrantBurnDownHistory{}, err
		}

		if period.From.Before(balance.At) {
			// This is an inconsistency check. It can only happen if we lost our snapshot for the reset.
			//
			// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
			// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
			return GrantBurnDownHistory{}, fmt.Errorf("current period start %s is before last valid balance snapshot at %s, no snapshot was created for reset", period.From, balance.At)
		}

		// get all relevant grants
		grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, period.From, period.To)
		// These grants might not be present in the starting balance so lets fill them
		// This is only possible in case the grant becomes active exactly at the start of the current period
		m.populateBalanceSnapshotWithMissingGrantsActiveAt(&balance, grants, period.From)

		if err != nil {
			return GrantBurnDownHistory{}, err
		}
		// run engine and calculate grantbalance
		engineParams, err := m.getQueryUsageFn(ctx, owner, ownerSubjecKey)
		if err != nil {
			return GrantBurnDownHistory{}, err
		}
		engine := NewEngine(engineParams.QueryUsageFn, engineParams.Grantuality)

		_, _, segments, err := engine.Run(
			ctx,
			grants,
			balance.Balances,
			balance.Overage,
			period,
		)
		if err != nil {
			return GrantBurnDownHistory{}, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", owner.ID, period.To, err)
		}

		// set reset as reason for last segment if current period end is a reset
		if slices.Contains(startTimes, period.To) {
			segments[len(segments)-1].TerminationReasons.UsageReset = true
		}

		historySegments = append(historySegments, segments...)
	}

	// return history
	return GrantBurnDownHistory{
		segments: historySegments,
	}, nil
}

func (m *balanceConnector) ResetUsageForOwner(ctx context.Context, owner NamespacedGrantOwner, params ResetUsageForOwnerParams) (*GrantBalanceSnapshot, error) {
	// Cannot reset for the future
	if params.At.After(clock.Now()) {
		return nil, &models.GenericUserError{Message: fmt.Sprintf("cannot reset at %s in the future", params.At)}
	}

	ownerMeter, err := m.ownerConnector.GetMeter(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner query params for owner %s: %w", owner.ID, err)
	}

	at := params.At.Truncate(ownerMeter.WindowSize.Duration())

	// check if reset is possible (after last reset)
	periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, clock.Now())
	if err != nil {
		if _, ok := err.(*OwnerNotFoundError); ok {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if at.Before(periodStart) {
		return nil, &models.GenericUserError{Message: fmt.Sprintf("reset at %s is before current usage period start %s", at, periodStart)}
	}

	balance, err := m.getLastValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	if balance.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown across resets.
		// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
		return nil, fmt.Errorf("last valid balance snapshot %s is before current period start at %s, no snapshot was created for reset", balance.At, periodStart)
	}

	grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, balance.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&balance, grants, balance.At)

	engineParams, err := m.getQueryUsageFn(ctx, owner, ownerMeter.SubjectKey)
	if err != nil {
		return nil, err
	}
	engine := NewEngine(engineParams.QueryUsageFn, engineParams.Grantuality)

	endingBalance, _, _, err := engine.Run(
		ctx,
		grants,
		balance.Balances,
		balance.Overage,
		recurrence.Period{
			From: balance.At,
			To:   at,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for reset: %w", err)
	}

	grantMap := make(map[string]Grant, len(grants))
	for _, grant := range grants {
		grantMap[grant.ID] = grant
	}

	// We have to roll over the grants and save the starting balance for the next period at the reset time.
	// Engine treates the output balance as a period end (exclusive), but we need to treat it as a period start (inclusive).
	startingBalance := GrantBalanceMap{}
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

	startingSnapshot := GrantBalanceSnapshot{
		At:       at,
		Balances: startingBalance,
		Overage:  0.0, // Overage is forgiven at reset
	}

	// FIXME: this is a bad hack to be able to pass around transactions on the connector level.
	// We should introduce an abstraction, maybe an AtomicOperation with something like an AtimicityGuarantee
	// (these would practically mirror entutils.TxUser & entutils.TxDriver) and then write an implementation of the
	// using the ent transactions we have.
	_, err = entutils.StartAndRunTx(ctx, m.balanceSnapshotConnector, func(txCtx context.Context, tx *entutils.TxDriver) (*GrantBalanceSnapshot, error) {
		err := m.ownerConnector.LockOwnerForTx(ctx, tx, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to lock owner %s: %w", owner.ID, err)
		}

		err = m.balanceSnapshotConnector.WithTx(txCtx, tx).Save(ctx, owner, []GrantBalanceSnapshot{startingSnapshot})
		if err != nil {
			return nil, fmt.Errorf("failed to save balance for owner %s at %s: %w", owner.ID, at, err)
		}

		err = m.ownerConnector.EndCurrentUsagePeriodTx(ctx, tx, owner, EndCurrentUsagePeriodParams{
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
func (m *balanceConnector) getLastValidBalanceSnapshotForOwnerAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error) {
	balance, err := m.balanceSnapshotConnector.GetLatestValidAt(ctx, owner, at)
	if err != nil {
		if _, ok := err.(*GrantBalanceNoSavedBalanceForOwnerError); ok {
			// if no snapshot is found we have to calculate from start of time on all grants and usage
			m.logger.Debug(fmt.Sprintf("no saved balance found for owner %s before %s, calculating from start of time", owner.ID, at))

			startOfMeasurement, err := m.ownerConnector.GetStartOfMeasurement(ctx, owner)
			if err != nil {
				return balance, err
			}

			grants, err := m.grantRepo.ListActiveGrantsBetween(ctx, owner, startOfMeasurement, at)
			if err != nil {
				return balance, err
			}

			balances := GrantBalanceMap{}
			for _, grant := range grants {
				if grant.ActiveAt(startOfMeasurement) {
					// Grants that are active at the start will have full balance
					balances.Set(grant.ID, grant.Amount)
				} else {
					// Grants that are not active at the start won't have a balance
					balances.Set(grant.ID, 0.0)
				}
			}

			balance = GrantBalanceSnapshot{
				At:       startOfMeasurement,
				Balances: balances,
				Overage:  0.0, // There cannot be overage at the start of measurement
			}
		} else {
			return balance, fmt.Errorf("failed to get latest valid grant balance at %s for owner %s: %w", at, owner.ID, err)
		}
	}

	return balance, nil
}

type engineParams struct {
	QueryUsageFn QueryUsageFn
	Grantuality  models.WindowSize
}

// returns owner specific QueryUsageFn
func (m *balanceConnector) getQueryUsageFn(ctx context.Context, owner NamespacedGrantOwner, subjectKey string) (*engineParams, error) {
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
			rows, err := m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.MeterSlug, params)
			if err != nil {
				return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.MeterSlug, err)
			}
			if len(rows) > 1 {
				return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
			}
			if len(rows) == 0 {
				return 0.0, nil
			}
			return rows[0].Value, nil
		},
		Grantuality: ownerMeter.WindowSize,
	}, nil
}

// Returns a snapshot from the last segment that can be saved, taking the following into account:
//
//  1. We can save a segment if it is older than graceperiod.
//  2. At the end of a segment history changes: s1.endBalance <> s2.startBalance. This means only the
//     starting values can be saved credibly.
func (m *balanceConnector) getLastSaveableSnapshotAt(history *GrantBurnDownHistory, lastValidBalance GrantBalanceSnapshot, at time.Time) (*GrantBalanceSnapshot, error) {
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

func (m *balanceConnector) excludeInactiveGrantsFromBalance(balances GrantBalanceMap, grants map[string]Grant, at time.Time) (*GrantBalanceMap, error) {
	filtered := &GrantBalanceMap{}
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
func (m *balanceConnector) populateBalanceSnapshotWithMissingGrantsActiveAt(snapshot *GrantBalanceSnapshot, grants []Grant, at time.Time) {
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

	// copy
	times := make([]time.Time, len(ts))
	copy(times, ts)

	// dedupe
	times = slicesx.Dedupe(times)

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
