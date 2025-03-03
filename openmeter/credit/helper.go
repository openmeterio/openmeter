package credit

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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

// Builds the engine for a given owner caching the period boundaries for the given range (queryBounds).
// As QueryUsageFn is frequently called, getting the CurrentUsagePeriodStartTime during it's execution would impact performance, so we cache all possible values during engine building.
func (m *connector) buildEngineForOwner(ctx context.Context, owner grant.NamespacedOwner, queryBounds timeutil.Period) (engine.Engine, error) {
	// Let's validate the parameters
	if queryBounds.From.IsZero() || queryBounds.To.IsZero() {
		return nil, fmt.Errorf("query bounds must have both from and to set")
	}

	// Let's get the owner specific params
	ownerMeter, err := m.ownerConnector.GetMeter(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get query params for owner %v: %w", owner, err)
	}

	subjectKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner subject key for owner %s: %w", owner.ID, err)
	}

	// Let's collect all period start times for any time between the query bounds
	// First we get the period start time for the start of the period, then all times in between
	firstPeriodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, queryBounds.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage period start time for owner %s at %s: %w", owner.ID, queryBounds.From, err)
	}

	inbetweenPeriodStarts, err := m.ownerConnector.GetResetTimelineInclusive(ctx, owner, queryBounds)
	if err != nil {
		return nil, fmt.Errorf("failed to get period start times for owner %s between %s and %s: %w", owner.ID, queryBounds.From, queryBounds.To, err)
	}

	times := append([]time.Time{firstPeriodStart}, inbetweenPeriodStarts.GetTimes()...)
	times = append(times, queryBounds.To)

	periodCache := SortedPeriodsFromDedupedTimes(times)

	if len(periodCache) == 0 {
		// If we didn't have at least 2 different timestamps, we need to create a period from the first start time and the bound
		periodCache = []timeutil.Period{{From: firstPeriodStart, To: queryBounds.To}}
	}

	// Let's write a function that replaces GetUsagePeriodStartAt with a cache lookup
	getUsagePeriodStartAtFromCache := func(at time.Time) (time.Time, error) {
		for _, period := range periodCache {
			// We run with ContainsInclusive in Time-ASC order so we can match the end of the last period
			if period.ContainsInclusive(at) {
				return period.From, nil
			}
		}
		return time.Time{}, fmt.Errorf("no period start time found for %s, known periods: %+v", at, periodCache)
	}

	// Let's define a simple helper that validates the returned meter rows
	getValueFromRows := func(rows []meter.MeterQueryRow) (float64, error) {
		// We expect only one row
		if len(rows) > 1 {
			return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
		}
		if len(rows) == 0 {
			return 0.0, nil
		}
		return rows[0].Value, nil
	}

	eng := engine.NewEngine(engine.EngineConfig{
		Granularity: ownerMeter.Meter.WindowSize,
		QueryUsage: func(ctx context.Context, from, to time.Time) (float64, error) {
			// Let's validate we're not querying outside the bounds
			if !queryBounds.ContainsInclusive(from) || !queryBounds.ContainsInclusive(to) {
				return 0.0, fmt.Errorf("query bounds between %s and %s do not contain query from %s to %s: %t %t", queryBounds.From, queryBounds.To, from, to, queryBounds.ContainsInclusive(from), queryBounds.ContainsInclusive(to))
			}

			params := ownerMeter.DefaultParams
			params.FilterSubject = []string{subjectKey}

			// Let's query the meter based on the aggregation
			switch ownerMeter.Meter.Aggregation {
			case meter.MeterAggregationUniqueCount:
				periodStart, err := getUsagePeriodStartAtFromCache(from)
				if err != nil {
					return 0.0, err
				}

				// To get the UNIQUE_COUNT value between `from` and `to` we need to:
				// 1. Query between the period start and `to` to get the unique count at `to`
				// 2. Query between the period start and `from` to get the unique count at `from`
				// 3. Subtract the two values
				params.From = &periodStart
				params.To = &to

				var valueTo float64 = 0.0
				var valueFrom float64 = 0.0

				if !periodStart.Equal(to) {
					rows, err := m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.Meter, params)
					if err != nil {
						return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.Meter.Slug, err)
					}

					valueTo, err = getValueFromRows(rows)
					if err != nil {
						return 0.0, err
					}
				}

				params.To = &from

				// If the two times are different we need to query the value at `from`
				if !params.From.Equal(*params.To) && !periodStart.Equal(from) {
					rows, err := m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.Meter, params)
					if err != nil {
						return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.Meter.Slug, err)
					}

					valueFrom, err = getValueFromRows(rows)
					if err != nil {
						return 0.0, err
					}
				}

				// Let's do an accurate subsctraction
				vTo := alpacadecimal.NewFromFloat(valueTo)
				vFrom := alpacadecimal.NewFromFloat(valueFrom)

				return vTo.Sub(vFrom).InexactFloat64(), nil

			// For SUM and COUNT we can simply query the meter
			case meter.MeterAggregationSum, meter.MeterAggregationCount:
				// If the two times are the same we can return 0.0 as there's no usage
				if from.Equal(to) {
					return 0.0, nil
				}

				params.From = &from
				params.To = &to

				// Let's query the meter
				rows, err := m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.Meter, params)
				if err != nil {
					return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.Meter.Slug, err)
				}

				return getValueFromRows(rows)
			default:
				return 0.0, fmt.Errorf("unsupported aggregation %s", ownerMeter.Meter.Aggregation)
			}
		},
	})
	return eng, nil
}

type snapshotParams struct {
	// All grants used at engine.Run
	grants []grant.Grant
	// Owner of the snapshot
	owner grant.NamespacedOwner
	// Result of the engine.Run
	runRes engine.RunResult
	// Snapshot is saved if the segment is before this time & the start of the current usage period (at time of snapshot)
	before time.Time
}

// It is assumed that there are no snapshots persisted during the length of the history (as engine.Run starts with a snapshot that should be the last valid snapshot)
func (m *connector) snapshotEngineResult(ctx context.Context, params snapshotParams) error {
	history, err := engine.NewGrantBurnDownHistory(params.runRes.History)
	if err != nil {
		return fmt.Errorf("failed to create grant burn down history: %w", err)
	}

	currentPeriodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, params.owner, params.runRes.Snapshot.At)
	if err != nil {
		return fmt.Errorf("failed to get current usage period start for owner %s at %s: %w", params.owner.ID, params.runRes.Snapshot.At, err)
	}

	segs := history.Segments()

	// i >= 1 because:
	// The first segment starts with the last valid snapshot and we don't want to create another snapshot for that same time
	for i := len(segs) - 1; i >= 1; i-- {
		seg := segs[i]

		// We can save a segment if its not after the current period start (this way backfilling, granting, resetting, etc... will work for the current UsagePeriod)
		if !seg.From.After(currentPeriodStart) && !seg.From.After(params.before) {
			snap := seg.ToSnapshot()
			if err := m.removeInactiveGrantsFromSnapshotAt(&snap, params.grants, currentPeriodStart); err != nil {
				return fmt.Errorf("failed to remove inactive grants from snapshot: %w", err)
			}

			if err := m.balanceSnapshotRepo.Save(ctx, params.owner, []balance.Snapshot{snap}); err != nil {
				return fmt.Errorf("failed to save snapshot: %w", err)
			}

			m.logger.DebugContext(ctx, "saved snapshot", "snapshot", snap, "owner", params.owner)

			break
		}
	}

	return nil
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

// Removes grants that are not active at the given time from the snapshot.
func (m *connector) removeInactiveGrantsFromSnapshotAt(snapshot *balance.Snapshot, grants []grant.Grant, at time.Time) error {
	grantMap := make(map[string]grant.Grant)
	for _, grant := range grants {
		grantMap[grant.ID] = grant
	}

	filtered := balance.Map{}
	for grantID, grantBalance := range snapshot.Balances {
		grant, ok := grantMap[grantID]
		if !ok {
			return fmt.Errorf("grant %s not found when removing inactive grants", grantID)
		}

		if grant.ActiveAt(at) {
			filtered.Set(grantID, grantBalance)
		}
	}

	snapshot.Balances = filtered

	return nil
}

// Returns a list of non-overlapping periods between the sorted times.
func SortedPeriodsFromDedupedTimes(ts []time.Time) []timeutil.Period {
	times := lo.UniqBy(ts, func(t time.Time) int64 {
		// We unique by unixnano because time.Time == time.Time comparison is finicky
		return t.UnixNano()
	})

	if len(times) < 2 {
		return nil
	}

	// sort
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	periods := make([]timeutil.Period, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		periods = append(periods, timeutil.Period{From: times[i-1], To: times[i]})
	}

	return periods
}
