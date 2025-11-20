package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	optimizedengine "github.com/openmeterio/openmeter/openmeter/credit/engine/optimized"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credittrace "github.com/openmeterio/openmeter/openmeter/credit/trace"
	"github.com/openmeterio/openmeter/pkg/clock"
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
	// GetBalanceAt returns the result of the engine.Run at a given time.
	// It tries to minimize execution cost by calculating from the latest valid snapshot, thus the length of the returned history WILL NOT be deterministic.
	GetBalanceAt(ctx context.Context, ownerID models.NamespacedID, at time.Time) (engine.RunResult, error)
	// GetBalanceForPeriod returns the result of the engine.Run for the provided period.
	// The returned history will exactly match the provided period.
	GetBalanceForPeriod(ctx context.Context, ownerID models.NamespacedID, period timeutil.ClosedPeriod) (engine.RunResult, error)
	// ResetUsageForOwner resets the usage for an owner at a given time.
	ResetUsageForOwner(ctx context.Context, ownerID models.NamespacedID, params ResetUsageForOwnerParams) (balanceAfterReset *balance.Snapshot, err error)
	// GetLastValidSnapshotAt fetches the last valid snapshot for an owner.
	GetLastValidSnapshotAt(ctx context.Context, owner models.NamespacedID, at time.Time) (balance.Snapshot, error)
}

var _ BalanceConnector = &connector{}

// GetBalanceSinceSnapshot returns the result of the engine.Run since a given snapshot.
func (m *connector) getBalanceSinceSnapshot(ctx context.Context, ownerID models.NamespacedID, snap balance.Snapshot, at time.Time) (engine.RunResult, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.GetBalanceSinceSnapshot", credittrace.WithOwner(ownerID), trace.WithAttributes(attribute.String("at", at.String())))
	defer span.End()

	var def engine.RunResult
	m.Logger.Debug("getting balance of owner since snapshot", "owner", ownerID.ID, "since", snap.At, "at", at)

	period := timeutil.ClosedPeriod{
		From: snap.At,
		To:   at,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.OwnerConnector.GetResetTimelineInclusive(ctx, ownerID, period)
	if err != nil {
		return def, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", period.From, period.To, ownerID.ID, err)
	}

	owner, err := m.OwnerConnector.DescribeOwner(ctx, ownerID)
	if err != nil {
		return def, fmt.Errorf("failed to describe owner %s: %w", ownerID.ID, err)
	}

	// get all relevant grants
	grants, err := m.GrantRepo.ListActiveGrantsBetween(ctx, ownerID, period.From, period.To)
	if err != nil {
		return def, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, ownerID.ID, err)
	}
	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the current period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&snap, grants, period.From)

	eng, err := optimizedengine.NewEngine(ctx, optimizedengine.Dependencies{
		OwnerConnector:     m.OwnerConnector,
		StreamingConnector: m.StreamingConnector,
		Tracer:             m.Tracer,
		Logger:             m.Logger,
	}, optimizedengine.Config{
		Owner:                 owner,
		QueryBounds:           period,
		InbetweenPeriodStarts: resetTimesInclusive,
	})
	if err != nil {
		return def, err
	}

	result, err := eng.Run(ctx, engine.RunParams{
		Grants:           grants,
		StartingSnapshot: snap,
		Until:            period.To,
		ResetBehavior:    owner.ResetBehavior,
		Resets:           resetTimesInclusive.After(period.From),
		Meter:            owner.Meter,
	})
	if err != nil {
		return def, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", ownerID.ID, at, err)
	}

	// Let's remove any grants that are not active at the query time
	err = m.removeInactiveGrantsFromSnapshotAt(&result.Snapshot, grants, at)
	if err != nil {
		return def, fmt.Errorf("failed to remove inactive grants from snapshot: %w", err)
	}

	periodStart, err := m.OwnerConnector.GetUsagePeriodStartAt(ctx, ownerID, at)
	if err != nil {
		return def, fmt.Errorf("failed to get usage period start at %s for owner %s: %w", at, ownerID.ID, err)
	}

	// Let's see if a snapshot should be saved
	// TODO: it might be the case that we don't save any snapshots as they require a history breakpoint. To solve this,
	// we should introduce artificial history breakpoints in the engine, but that would result in more streaming.Query calls, so first lets improve the visibility of what's happening.
	if err := m.snapshotEngineResult(ctx, snapshotParams{
		grants:   grants,
		owner:    ownerID,
		notAfter: m.getSnapshotNotAfter(periodStart, clock.Now()),
		meter:    owner.Meter,
	}, result); err != nil {
		return def, fmt.Errorf("failed to snapshot engine result: %w", err)
	}

	// return balance
	return result, nil
}

func (m *connector) GetBalanceAt(ctx context.Context, ownerID models.NamespacedID, at time.Time) (engine.RunResult, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.GetBalanceAt", credittrace.WithOwner(ownerID), trace.WithAttributes(attribute.String("at", at.String())))
	defer span.End()

	m.Logger.Debug("getting balance of owner", "owner", ownerID.ID, "at", at)

	var def engine.RunResult

	// FIXME: remove truncation
	// To include the current last minute lets round it trunc to the next minute
	if trunc := at.Truncate(time.Minute); trunc.Before(at) {
		at = trunc.Add(time.Minute)
	}

	// get last valid grantbalances
	snap, err := m.GetLastValidSnapshotAt(ctx, ownerID, at)
	if err != nil {
		return def, err
	}

	return m.getBalanceSinceSnapshot(ctx, ownerID, snap, at)
}

func (m *connector) GetBalanceForPeriod(ctx context.Context, ownerID models.NamespacedID, period timeutil.ClosedPeriod) (engine.RunResult, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.GetBalanceForPeriod", credittrace.WithOwner(ownerID), credittrace.WithPeriod(period))
	defer span.End()

	m.Logger.Debug("calculating history for owner", "owner", ownerID.ID, "period", period)

	var def engine.RunResult

	// FIXME: remove truncation
	// To include the current last minute lets round it trunc to the next minute
	if trunc := period.To.Truncate(time.Minute); trunc.Before(period.To) {
		period.To = trunc.Add(time.Minute)
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.OwnerConnector.GetResetTimelineInclusive(ctx, ownerID, period)
	if err != nil {
		return def, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", period.From, period.To, ownerID.ID, err)
	}

	owner, err := m.OwnerConnector.DescribeOwner(ctx, ownerID)
	if err != nil {
		return def, fmt.Errorf("failed to describe owner %s: %w", ownerID.ID, err)
	}

	// For the history result to start from the correct period start we need to start from a synthetic snapshot by calculating the balance at the period start
	res, err := m.GetBalanceAt(ctx, ownerID, period.From)
	if err != nil {
		return def, err
	}

	// get all relevant grants
	grants, err := m.GrantRepo.ListActiveGrantsBetween(ctx, ownerID, res.Snapshot.At, period.To)
	if err != nil {
		return def, err
	}

	snap := res.Snapshot

	// These grants might not be present in the starting balance so lets fill them
	// This is only possible in case the grant becomes active exactly at the start of the first period
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&snap, grants, snap.At)

	eng, err := optimizedengine.NewEngine(ctx, optimizedengine.Dependencies{
		OwnerConnector:     m.OwnerConnector,
		StreamingConnector: m.StreamingConnector,
		Tracer:             m.Tracer,
		Logger:             m.Logger,
	}, optimizedengine.Config{
		Owner:                 owner,
		QueryBounds:           period,
		InbetweenPeriodStarts: resetTimesInclusive,
	})
	if err != nil {
		return def, fmt.Errorf("failed to build engine for owner %s: %w", ownerID.ID, err)
	}

	result, err := eng.Run(ctx, engine.RunParams{
		Grants:           grants,
		StartingSnapshot: snap,
		Until:            period.To,
		ResetBehavior:    owner.ResetBehavior,
		Resets:           resetTimesInclusive.After(snap.At),
		Meter:            owner.Meter,
	})
	if err != nil {
		return def, fmt.Errorf("failed to calculate balance for owner %s at %s: %w", ownerID.ID, period.From, err)
	}

	return result, nil
}

func (m *connector) ResetUsageForOwner(ctx context.Context, ownerID models.NamespacedID, params ResetUsageForOwnerParams) (*balance.Snapshot, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.ResetUsageForOwner", credittrace.WithOwner(ownerID), trace.WithAttributes(attribute.String("at", params.At.String())))
	defer span.End()

	// Cannot reset for the future
	if params.At.After(clock.Now()) {
		return nil, models.NewGenericValidationError(fmt.Errorf("cannot reset at %s in the future", params.At))
	}

	owner, err := m.OwnerConnector.DescribeOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe owner %s: %w", ownerID.ID, err)
	}

	// FIXME: remove truncation
	at := params.At.Truncate(time.Minute)

	// check if reset is possible (not before current period)
	periodStart, err := m.OwnerConnector.GetUsagePeriodStartAt(ctx, ownerID, clock.Now())
	if err != nil {
		if _, ok := lo.ErrorsAs[*grant.OwnerNotFoundError](err); ok {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", ownerID.ID, at, err)
	}
	if at.Before(periodStart) {
		return nil, models.NewGenericValidationError(fmt.Errorf("reset at %s is before current usage period start %s", at, periodStart))
	}

	resetsSinceTime, err := m.OwnerConnector.GetResetTimelineInclusive(ctx, ownerID, timeutil.ClosedPeriod{From: at, To: clock.Now()})
	if err != nil {
		return nil, fmt.Errorf("failed to get reset times since %s for owner %s: %w", at, ownerID.ID, err)
	}

	if rts := resetsSinceTime.After(at).GetTimes(); len(rts) > 0 {
		lastReset := rts[len(rts)-1]
		return nil, models.NewGenericValidationError(fmt.Errorf("reset at %s is before last reset at %s", at, lastReset))
	}

	bal, err := m.GetLastValidSnapshotAt(ctx, ownerID, at)
	if err != nil {
		return nil, err
	}

	period := timeutil.ClosedPeriod{
		From: bal.At,
		To:   at,
	}

	// get all usage resets between queryied period
	resetTimesInclusive, err := m.OwnerConnector.GetResetTimelineInclusive(ctx, ownerID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get reset times between %s and %s for owner %s: %w", period.From, period.To, ownerID.ID, err)
	}

	// Let's also add the at time to the resets
	resetTimes := append(resetTimesInclusive.GetTimes(), at)
	resetTimeline := timeutil.NewSimpleTimeline(resetTimes)

	// This gets overwritten by the inputs
	resetBehavior := owner.ResetBehavior
	resetBehavior.PreserveOverage = params.PreserveOverage

	grants, err := m.GrantRepo.ListActiveGrantsBetween(ctx, ownerID, bal.At, at)
	if err != nil {
		return nil, fmt.Errorf("failed to list active grants at %s for owner %s: %w", at, ownerID.ID, err)
	}
	m.populateBalanceSnapshotWithMissingGrantsActiveAt(&bal, grants, bal.At)

	// Let's define the period the engine will be queried for
	queriedPeriod := timeutil.ClosedPeriod{
		From: bal.At,
		To:   at,
	}

	eng, err := optimizedengine.NewEngine(ctx, optimizedengine.Dependencies{
		OwnerConnector:     m.OwnerConnector,
		StreamingConnector: m.StreamingConnector,
		Tracer:             m.Tracer,
		Logger:             m.Logger,
	}, optimizedengine.Config{
		Owner:                 owner,
		QueryBounds:           queriedPeriod,
		InbetweenPeriodStarts: resetTimesInclusive,
	})
	if err != nil {
		return nil, err
	}

	res, err := eng.Run(ctx, engine.RunParams{
		Grants:           grants,
		StartingSnapshot: bal,
		Until:            queriedPeriod.To,
		ResetBehavior:    resetBehavior,
		Resets:           resetTimeline.After(bal.At),
		Meter:            owner.Meter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance for reset: %w", err)
	}

	snap := res.Snapshot

	_, err = transaction.Run(ctx, m.TransactionManager, func(ctx context.Context) (*balance.Snapshot, error) {
		err = m.OwnerConnector.LockOwnerForTx(ctx, ownerID)
		if err != nil {
			return nil, fmt.Errorf("failed to lock owner %s: %w", ownerID.ID, err)
		}

		// Let's save the snapshot
		snap, err = m.saveSnapshot(ctx, snapshotParams{
			grants:   grants,
			owner:    ownerID,
			notAfter: at,
			meter:    owner.Meter,
		}, snap)
		if err != nil {
			return nil, fmt.Errorf("failed to save snapshot: %w", err)
		}

		// Let's end the current usage period
		err = m.OwnerConnector.EndCurrentUsagePeriod(ctx, ownerID, grant.EndCurrentUsagePeriodParams{
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

	return &snap, nil
}

// GetLastValidSnapshotAt fetches the last valid snapshot for an owner.
// If no usable snapshot exists returns a default snapshot for measurement start to recalculate the entire history.
func (m *connector) GetLastValidSnapshotAt(ctx context.Context, owner models.NamespacedID, at time.Time) (balance.Snapshot, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.GetLastValidSnapshotAt", credittrace.WithOwner(owner), trace.WithAttributes(attribute.String("at", at.String())))
	defer span.End()

	bal, err := m.BalanceSnapshotService.GetLatestValidAt(ctx, owner, at)
	if err != nil {
		if _, ok := lo.ErrorsAs[*balance.NoSavedBalanceForOwnerError](err); ok {
			// if no snapshot is found we have to calculate from start of time on all grants and usage
			m.Logger.Debug(fmt.Sprintf("no saved balance found for owner %s before %s, calculating from start of time", owner.ID, at))

			startOfMeasurement, err := m.OwnerConnector.GetStartOfMeasurement(ctx, owner)
			if err != nil {
				return bal, err
			}

			grants, err := m.GrantRepo.ListActiveGrantsBetween(ctx, owner, startOfMeasurement, at)
			if err != nil {
				return bal, err
			}

			bal = balance.Snapshot{
				At:       startOfMeasurement,
				Balances: balance.NewStartingMap(grants, startOfMeasurement),
				Overage:  0.0, // There cannot be overage at the start of measurement
			}
		} else {
			return bal, fmt.Errorf("failed to get latest valid grant balance at %s for owner %s: %w", at, owner.ID, err)
		}
	}

	return bal, nil
}
