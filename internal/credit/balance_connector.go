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
	GetBalanceOfOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (float64, error)
	GetBalanceHistoryOfOwner(ctx context.Context, owner NamespacedGrantOwner, params BalanceHistoryParams) (GrantBurnDownHistory, error)
	ResetUsageForOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error
}

type BalanceHistoryParams struct {
	From time.Time
	To   time.Time
}

func NewBalanceConnector(gc GrantConnector, gbc GrantBalanceConnector, oc OwnerConnector, sc streaming.Connector, log slog.Logger) BalanceConnector {
	return &balanceConnector{gc: gc, gbc: gbc, oc: oc, sc: sc, l: log}
}

type balanceConnector struct {
	gc  GrantConnector
	gbc GrantBalanceConnector
	oc  OwnerConnector
	sc  streaming.Connector
	l   slog.Logger
}

var _ BalanceConnector = &balanceConnector{}

func (m *balanceConnector) GetBalanceOfOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (float64, error) {
	// get last valid grantbalances
	balance, err := m.getLatestValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	// get all relevant grants
	grants, err := m.gc.ListActiveGrantsBetween(ctx, owner, balance.At, at)
	if err != nil {
		return 0, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}
	// run engine and calculate grantbalance
	queryFn, err := m.getQueryUsageFn(ctx, owner)
	if err != nil {
		return 0, err
	}
	engine := NewEngine(queryFn)

	result, _, segments, err := engine.Run(
		grants,
		balance.Balances,
		balance.Overage,
		Period{
			From: balance.At,
			To:   at,
		},
	)

	history, err := NewGrantBurnDownHistory(segments)
	if err != nil {
		return 0, fmt.Errorf("failed to create grant burn down history: %w", err)
	}

	// TODO: don't just save last segment, save entire history
	if saveable, err := history.GetLastSaveableAt(at); err == nil {
		// save snapshot at end of segment
		m.gbc.Save(ctx, owner, []GrantBalanceSnapshot{
			{
				At:       saveable.To,
				Balances: saveable.ApplyUsage(),
				Overage:  saveable.Overage,
			},
		})
	}

	// return balance
	return result.Balance(), nil
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

	periods := PeriodsFromTimes(times)
	historySegments := make([]GrantBurnDownHistorySegment, 0, len(periods))

	// collect al history segments through all periods
	for _, period := range periods {
		// get last valid grantbalances
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

func (m *balanceConnector) ResetUsageForOwner(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error {
	// definitely do in transsaction
	//      - also don't forget about locking on a per owner basis
	//      - to do connectors neeed to be able to accept a transaction object
	//      -  we can just shadow the endb transaction object
	//      - all schema specific capabilities of the generated clients can be reattached
	//      - in practiec we'll just create a new transacting instance that all clients can use

	// check if reset is possible (after last reset)
	periodStart, err := m.oc.GetCurrentUsagePeriodStart(ctx, owner)
	if err != nil {
		return fmt.Errorf("failed to get current usage period start for owner %s: %w", owner.ID, err)
	}
	if at.Before(periodStart) {
		return fmt.Errorf("reset at %s is before current usage period start %s", at, periodStart)
	}

	balance, err := m.getLatestValidBalanceSnapshotForOwnerAt(ctx, owner, at)
	if err != nil {
		return err
	}

	if balance.At.Before(periodStart) {
		// This is an inconsistency check. It can only happen if we lost our snapshot for the last reset.
		//
		// The engine doesn't manage rollovers at usage reset so it cannot be used to calculate GrantBurnDown accross resets.
		// FIXME: this is theoretically possible, we need to handle it, add capability to ledger.
		return fmt.Errorf("current period start %s is before last valid balance snapshot at %s, no snapshot was created for reset", periodStart, balance.At)
	}

	grants, err := m.gc.ListActiveGrantsBetween(ctx, owner, balance.At, at)
	if err != nil {
		return fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, owner.ID, err)
	}

	queryFn, err := m.getQueryUsageFn(ctx, owner)
	if err != nil {
		return err
	}
	engine := NewEngine(queryFn)

	endingBalance, overage, _, err := engine.Run(
		grants,
		balance.Balances,
		balance.Overage,
		Period{
			From: balance.At,
			To:   at,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to calculate balance for reset: %w", err)
	}

	// TODO: ROLLOVER at usage reset!

	// we don't have a grace period at reset, we aways save the exact balance and overage
	// for the provided timestamp
	err = m.gbc.Save(ctx, owner, []GrantBalanceSnapshot{
		{
			At:       at,
			Balances: endingBalance,
			Overage:  overage,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to save balance for owner %s at %s: %w", owner.ID, at, err)
	}

	err = m.oc.EndCurrentUsagePeriod(ctx, owner, at)
	if err != nil {
		return fmt.Errorf("failed to end current usage period for owner %s at %s: %w", owner.ID, at, err)
	}
	return nil
}

// Fetches the last valid snapshot for an owner.
//
// If no snapshot exists returns a default snapshot for measurement start to recalculate the entire history
// in case no usable snapshot was found.
func (m *balanceConnector) getLatestValidBalanceSnapshotForOwnerAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error) {
	balance, err := m.gbc.GetLatestValidAt(ctx, owner, at)
	if err != nil {
		if _, ok := err.(GrantBalanceNoSavedBalanceForOwnerError); ok {
			// if no snapshot is found we have to calculate from start of time on all grants and usage
			m.l.Info(fmt.Sprintf("no saved balance found for owner %s before %s, calculating from start of time", owner.ID, at))

			grants, err := m.gc.ListGrants(ctx, ListGrantsParams{
				Namespace:      owner.Namespace,
				OwnerID:        &owner.ID,
				IncludeDeleted: true,
			})
			if err != nil {
				return balance, fmt.Errorf("failed to list grants for owner %s: %w", owner.ID, err)
			}
			startOfMeasurement, err := m.oc.GetStartOfMeasurement(ctx, owner)
			if err != nil {
				return balance, fmt.Errorf("failed to get start of measurement for owner %s: %w", owner.ID, err)
			}

			balance = GrantBalanceSnapshot{
				At:       startOfMeasurement,
				Balances: NewGrantBalanceMapFromStartingGrants(grants),
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
		rows, err := m.sc.QueryMeter(context.TODO(), owner.Namespace, meterSlug, &params)
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
