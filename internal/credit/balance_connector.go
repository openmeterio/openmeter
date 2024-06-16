package credit

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/internal/streaming"
)

// Generic connector for balance related operations.
type BalanceConnector interface {
	GetBalanceOfOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (*GrantBalanceSnapshot, error)
	GetBalanceHistoryOfOwner(ctx context.Context, owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error)
	ResetUsageForOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (balanceAfterReset *GrantBalanceSnapshot, err error)
}

type BalanceHistoryParams struct {
	From time.Time
	To   time.Time
}

func NewBalanceConnector(gc GrantDBConnector, bsc BalanceSnapshotDBConnector, oc OwnerConnector, sc streaming.Connector, log *slog.Logger) BalanceConnector {
	return &balanceConnector{gc: gc, bsc: bsc, oc: oc, sc: sc, logger: log}
}

type balanceConnector struct {
	// grants and balance snapshots are managed in this same package
	gc  GrantDBConnector
	bsc BalanceSnapshotDBConnector
	// external dependencies
	oc     OwnerConnector
	sc     streaming.Connector
	logger *slog.Logger
}

var _ BalanceConnector = &balanceConnector{}

func (m *balanceConnector) GetBalanceOfOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (*GrantBalanceSnapshot, error) {
	// get last valid grantbalances
	balance, err := m.getLatestValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	periodStart, err := m.oc.GetUsagePeriodStartAt(ctx, owner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if balance.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown accross resets.
		return nil, fmt.Errorf("last valid balance snapshot %s is before current period start at %s, no snapshot was created for reset", balance.At, periodStart)
	}

	// get all relevant grants
	grants, err := m.gc.ListActiveGrantsBetween(ctx, owner, balance.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	// these grants might not be present in the starting balance so lets fill them
	for _, grant := range grants {
		if _, ok := balance.Balances[grant.ID]; !ok {
			if grant.ActiveAt(balance.At) {
				// This is only possible in case the grant becomes active exactly at the start of the current period
				balance.Balances.Set(grant.ID, grant.Amount)
			} else {
				balance.Balances.Set(grant.ID, 0.0)
			}
		}
	}

	// run engine and calculate grantbalance
	queryFn, err := m.getQueryUsageFn(ctx, owner)
	if err != nil {
		return nil, err
	}
	engine := NewEngine(queryFn)

	result, overage, segments, err := engine.Run(
		grants,
		balance.Balances,
		balance.Overage,
		Period{
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
	if saveable, err := history.GetLastSaveableAt(at); err == nil {
		// save snapshot at end of segment
		m.bsc.Save(ctx, owner, []GrantBalanceSnapshot{
			saveable.ToSnapshot(),
		})
	}

	// return balance
	return &GrantBalanceSnapshot{
		At:       at,
		Balances: result,
		Overage:  overage,
	}, nil
}

// Returns the joined GrantBurnDownHistory accross usage periods.
func (m *balanceConnector) GetBalanceHistoryOfOwner(ctx context.Context, owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error) {
	// get all usage resets inbetween queryied period
	startTimes, err := m.oc.GetPeriodStartTimesBetween(ctx, owner, params.From, params.To)
	if err != nil {
		return GrantBurnDownHistory{}, fmt.Errorf("failed to get period start times between %s and %s for owner %s: %w", params.From, params.To, owner.ID, err)
	}
	times := []time.Time{params.From}
	times = append(times, startTimes...)
	times = append(times, params.To)

	periods := SortedPeriodsFromDedupedTimes(times)
	historySegments := make([]GrantBurnDownHistorySegment, 0, len(periods))

	// collect al history segments through all periods
	for _, period := range periods {
		// get last valid grantbalances at start of period (eq balance at start of period)
		balance, err := m.getLatestValidBalanceSnapshotForOwnerAt(ctx, owner, period.From)
		if err != nil {
			return GrantBurnDownHistory{}, err
		}

		if balance.At.Before(period.From) {
			// This is an inconsistency check. It can only happen if we lost our snapshot for the reset.
			//
			// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown accross resets.
			// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
			return GrantBurnDownHistory{}, fmt.Errorf("current period start %s is before last valid balance snapshot at %s, no snapshot was created for reset", period.From, balance.At)
		}

		// get all relevant grants
		grants, err := m.gc.ListActiveGrantsBetween(ctx, owner, period.From, period.To)
		// these grants might not be present in the starting balance so lets fill them
		for _, grant := range grants {
			if _, ok := balance.Balances[grant.ID]; !ok {
				if grant.ActiveAt(period.From) {
					// This is only possible in case the grant becomes active exactly at the start of the current period
					balance.Balances.Set(grant.ID, grant.Amount)
				} else {
					balance.Balances.Set(grant.ID, 0.0)
				}
			}
		}

		if err != nil {
			return GrantBurnDownHistory{}, err
		}
		// run engine and calculate grantbalance
		queryFn, err := m.getQueryUsageFn(ctx, owner)
		if err != nil {
			return GrantBurnDownHistory{}, err
		}
		engine := NewEngine(queryFn)

		_, _, segments, err := engine.Run(
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

func (m *balanceConnector) ResetUsageForOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (*GrantBalanceSnapshot, error) {
	// definitely do in transsaction
	//      - also don't forget about locking on a per owner basis
	//      - to do connectors neeed to be able to accept a transaction object
	//      -  we can just shadow the endb transaction object
	//      - all schema specific capabilities of the generated clients can be reattached
	//      - in practiec we'll just create a new transacting instance that all clients can use

	// check if reset is possible (after last reset)
	periodStart, err := m.oc.GetUsagePeriodStartAt(ctx, owner, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", owner.ID, at, err)
	}
	if at.Before(periodStart) {
		return nil, fmt.Errorf("reset at %s is before current usage period start %s", at, periodStart)
	}

	balance, err := m.getLatestValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return nil, err
	}

	if balance.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown accross resets.
		// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
		return nil, fmt.Errorf("last valid balance snapshot %s is before current period start at %s, no snapshot was created for reset", balance.At, periodStart)
	}

	grants, err := m.gc.ListActiveGrantsBetween(ctx, owner, balance.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}

	queryFn, err := m.getQueryUsageFn(ctx, owner)
	if err != nil {
		return nil, err
	}
	engine := NewEngine(queryFn)

	endingBalance, _, _, err := engine.Run(
		grants,
		balance.Balances,
		balance.Overage,
		Period{
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

	// We have to roll over the grants and save the starting balance for the next period
	// at the reset time.
	startingBalance := endingBalance.Copy()
	for grantID, grantBalance := range endingBalance {
		grant, ok := grantMap[grantID]
		// inconcistency check, shouldn't happen
		if !ok {
			return nil, fmt.Errorf("attempting to roll over unknown grant %s", grantID)
		}
		startingBalance.Set(grantID, grant.RolloverBalance(grantBalance))
	}

	startingSnapshot := GrantBalanceSnapshot{
		At:       at,
		Balances: startingBalance,
		Overage:  0.0, // Overage is forgiven at reset
	}

	err = m.bsc.Save(ctx, owner, []GrantBalanceSnapshot{startingSnapshot})
	if err != nil {
		return nil, fmt.Errorf("failed to save balance for owner %s at %s: %w", owner.ID, at, err)
	}

	err = m.oc.EndCurrentUsagePeriod(ctx, owner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to end current usage period for owner %s at %s: %w", owner.ID, at, err)
	}

	return &startingSnapshot, nil
}

// Fetches the last valid snapshot for an owner.
//
// If no snapshot exists returns a default snapshot for measurement start to recalculate the entire history
// in case no usable snapshot was found.
func (m *balanceConnector) getLatestValidBalanceSnapshotForOwnerAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error) {
	balance, err := m.bsc.GetLatestValidAt(ctx, owner, at)
	if err != nil {
		if _, ok := err.(*GrantBalanceNoSavedBalanceForOwnerError); ok {
			// if no snapshot is found we have to calculate from start of time on all grants and usage
			m.logger.Info(fmt.Sprintf("no saved balance found for owner %s before %s, calculating from start of time", owner.ID, at))

			startOfMeasurement, err := m.oc.GetStartOfMeasurement(ctx, owner)
			if err != nil {
				return balance, fmt.Errorf("failed to get start of measurement for owner %s: %w", owner.ID, err)
			}

			grants, err := m.gc.ListActiveGrantsBetween(ctx, owner, startOfMeasurement, at)
			if err != nil {
				return balance, fmt.Errorf("failed to list grants for owner %s: %w", owner.ID, err)
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

// returns owner specific QueryUsageFn
func (m *balanceConnector) getQueryUsageFn(ctx context.Context, owner NamespacedGrantOwner) (QueryUsageFn, error) {
	meterSlug, ownerParams, err := m.oc.GetOwnerQueryParams(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get query params for owner %v: %w", owner, err)
	}
	return func(from, to time.Time) (float64, error) {
		// copy
		params := ownerParams
		params.From = &from
		params.To = &to
		rows, err := m.sc.QueryMeter(context.TODO(), owner.Namespace, meterSlug, params)
		if err != nil {
			return 0.0, fmt.Errorf("failed to query meter %s: %w", meterSlug, err)
		}
		if len(rows) > 1 {
			return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
		}
		if len(rows) == 0 {
			return 0.0, nil
		}
		return rows[0].Value, nil
	}, nil
}
