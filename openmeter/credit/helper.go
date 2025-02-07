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
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
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

// Builds the engine for a given owner caching the period boundaries for the given range (queryBounds).
// As QueryUsageFn is frequently called, getting the CurrentUsagePeriodStartTime during it's execution would impact performance, so we cache all possible values during engine building.
func (m *connector) buildEngineForOwner(ctx context.Context, owner grant.NamespacedOwner, queryBounds recurrence.Period) (engine.Engine, error) {
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

	inbetweenPeriodStarts, err := m.ownerConnector.GetPeriodStartTimesBetween(ctx, owner, queryBounds.From, queryBounds.To)
	if err != nil {
		return nil, fmt.Errorf("failed to get period start times for owner %s between %s and %s: %w", owner.ID, queryBounds.From, queryBounds.To, err)
	}

	times := append([]time.Time{firstPeriodStart}, inbetweenPeriodStarts...)

	periodCache := SortedPeriodsFromDedupedTimes(times)

	if len(periodCache) == 0 {
		// If we didn't have at least 2 different timestamps, we need to create a period from the first start time and the bound
		periodCache = []recurrence.Period{{From: firstPeriodStart, To: queryBounds.To}}
	}

	// Let's write a function that replaces GetUsagePeriodStartAt with a cache lookup
	getUsagePeriodStartAtFromCache := func(at time.Time) (time.Time, error) {
		for _, period := range periodCache {
			if period.ContainsInclusive(at) {
				return period.From, nil
			}
		}
		return time.Time{}, fmt.Errorf("no period start time found for %s", at)
	}

	// Let's define a simple helper that validates the returned meter rows
	getValueFromRows := func(rows []models.MeterQueryRow) (float64, error) {
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
			case models.MeterAggregationUniqueCount:
				periodStartAtFrom, err := getUsagePeriodStartAtFromCache(from)
				if err != nil {
					return 0.0, err
				}

				periodStartAtTo, err := getUsagePeriodStartAtFromCache(to)
				if err != nil {
					return 0.0, err
				}

				// The engine cannot query across periods so if the two times don't match we got a problem
				if !periodStartAtFrom.Equal(periodStartAtTo) {
					return 0.0, fmt.Errorf("cannot query across periods: %s and %s", periodStartAtFrom, periodStartAtTo)
				}

				// To get the UNIQUE_COUNT value between `from` and `to` we need to:
				// 1. Query between the period start and `to` to get the unique count at `to`
				// 2. Query between the period start and `from` to get the unique count at `from`
				// 3. Subtract the two values
				params.From = &periodStartAtFrom
				params.To = &to

				rows, err := m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.Meter, params)
				if err != nil {
					return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.Meter.Slug, err)
				}

				valueTo, err := getValueFromRows(rows)
				if err != nil {
					return 0.0, err
				}

				params.To = &from

				rows, err = m.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.Meter, params)
				if err != nil {
					return 0.0, fmt.Errorf("failed to query meter %s: %w", ownerMeter.Meter.Slug, err)
				}

				valueFrom, err := getValueFromRows(rows)
				if err != nil {
					return 0.0, err
				}

				// Let's do an accurate subsctraction
				vTo := alpacadecimal.NewFromFloat(valueTo)
				vFrom := alpacadecimal.NewFromFloat(valueFrom)

				return vTo.Sub(vFrom).InexactFloat64(), nil

			// For SUM and COUNT we can simply query the meter
			case models.MeterAggregationSum, models.MeterAggregationCount:
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

	periods := make([]recurrence.Period, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		periods = append(periods, recurrence.Period{From: times[i-1], To: times[i]})
	}

	return periods
}
