package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// GetLastValidSnapshotAt fetches the last valid snapshot for an owner.
// If no usable snapshot exists returns a default snapshot for measurement start to recalculate the entire history.
func (m *connector) GetLastValidSnapshotAt(ctx context.Context, owner models.NamespacedID, at time.Time) (balance.Snapshot, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.GetLastValidSnapshotAt", cTrace.WithOwner(owner), trace.WithAttributes(attribute.String("at", at.String())))
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

func (m *connector) runEngineInSpan(ctx context.Context, eng engine.Engine, runParams engine.RunParams) (engine.RunResult, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.runEngine", cTrace.WithEngineParams(runParams))
	defer span.End()

	res, err := eng.Run(ctx, runParams)

	// Let's annotate the span with the calculated history periods so we understand the engine's execution
	// We can do it even if we got an error, worst case scenario we'll have an empty list of periods.
	periods := res.History.GetPeriods()

	periodsJSON, marshalErr := json.Marshal(periods)
	if marshalErr != nil {
		m.Logger.WarnContext(ctx, "failed to marshal periods for tracing", "error", err)
	} else {
		span.SetAttributes(attribute.String("periods", string(periodsJSON)))
	}

	return res, err
}

type buildEngineForOwnerParams struct {
	// The owner that will be queried
	owner grant.Owner
	// A limit to that period that can be queried
	queryBounds timeutil.ClosedPeriod
	// A timeline of all reset events that have happened inside query bounds
	inbetweenPeriodStarts timeutil.SimpleTimeline
}

// Builds the engine for a given owner caching the period boundaries for the given range (queryBounds).
// As QueryUsageFn is frequently called, getting the CurrentUsagePeriodStartTime during it's execution would impact performance, so we cache all possible values during engine building.
func (m *connector) buildEngineForOwner(ctx context.Context, params buildEngineForOwnerParams) (engine.Engine, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.buildEngineForOwner", cTrace.WithOwner(params.owner.NamespacedID), cTrace.WithPeriod(params.queryBounds))
	defer span.End()

	// Let's validate the parameters
	if params.queryBounds.From.IsZero() || params.queryBounds.To.IsZero() {
		return nil, fmt.Errorf("query bounds must have both from and to set")
	}

	// Let's collect all period start times for any time between the query bounds
	// First we get the period start time for the start of the period, then all times in between
	firstPeriodStart, err := m.OwnerConnector.GetUsagePeriodStartAt(ctx, params.owner.NamespacedID, params.queryBounds.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage period start time for owner %s at %s: %w", params.owner.NamespacedID.ID, params.queryBounds.From, err)
	}

	times := append([]time.Time{firstPeriodStart}, params.inbetweenPeriodStarts.GetTimes()...)
	times = append(times, params.queryBounds.To)

	periodCache := SortedPeriodsFromDedupedTimes(times)

	if len(periodCache) == 0 {
		// If we didn't have at least 2 different timestamps, we need to create a period from the first start time and the bound
		periodCache = []timeutil.ClosedPeriod{{From: firstPeriodStart, To: params.queryBounds.To}}
	}

	// We build a custom UsageQuerier for our usecase here. The engine should only every query the one owner we fetched above.
	usageQuerier := balance.NewUsageQuerier(balance.UsageQuerierConfig{
		StreamingConnector: m.StreamingConnector,
		DescribeOwner: func(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
			if id != params.owner.NamespacedID {
				return grant.Owner{}, fmt.Errorf("expected owner %s, got %s", params.owner.NamespacedID.ID, id.ID)
			}
			return params.owner, nil
		},
		GetDefaultParams: func(ctx context.Context, oID models.NamespacedID) (streaming.QueryParams, error) {
			if oID != params.owner.NamespacedID {
				return streaming.QueryParams{}, fmt.Errorf("expected owner %s, got %s", params.owner.NamespacedID.ID, oID.ID)
			}
			return params.owner.DefaultQueryParams, nil
		},
		GetUsagePeriodStartAt: func(_ context.Context, _ models.NamespacedID, at time.Time) (time.Time, error) {
			for _, period := range periodCache {
				// We run with ContainsInclusive in Time-ASC order so we can match the end of the last period
				if period.ContainsInclusive(at) {
					return period.From, nil
				}
			}
			return time.Time{}, fmt.Errorf("no period start time found for %s, known periods: %+v", at, periodCache)
		},
	})

	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(ctx context.Context, from, to time.Time) (float64, error) {
			ctx, span := m.Tracer.Start(
				ctx,
				"credit.QueryUsageFn",
				trace.WithAttributes(attribute.String("from", from.String())),
				trace.WithAttributes(attribute.String("to", to.String())),
			)
			defer span.End()

			// Let's validate we're not querying outside the bounds
			if !params.queryBounds.ContainsInclusive(from) || !params.queryBounds.ContainsInclusive(to) {
				return 0.0, fmt.Errorf("query bounds between %s and %s do not contain query from %s to %s: %t %t", params.queryBounds.From, params.queryBounds.To, from, to, params.queryBounds.ContainsInclusive(from), params.queryBounds.ContainsInclusive(to))
			}

			// If we're inside the period cache, we can just use the UsageQuerier
			return usageQuerier.QueryUsage(ctx, params.owner.NamespacedID, timeutil.ClosedPeriod{From: from, To: to})
		},
	})
	return eng, nil
}

type snapshotParams struct {
	// Meter information to determine aggregation type
	meter meter.Meter
	// All grants used at engine.Run
	grants []grant.Grant
	// Owner of the snapshot
	owner models.NamespacedID
	// Snapshot is saved if the segment is not after this time & the start of the current usage period (at time of snapshot)
	notAfter time.Time
}

// It is assumed that there are no snapshots persisted during the length of the history (as engine.Run starts with a snapshot that should be the last valid snapshot)
func (m *connector) snapshotEngineResult(ctx context.Context, snapParams snapshotParams, runRes engine.RunResult) error {
	ctx, span := m.Tracer.Start(ctx, "credit.snapshotEngineResult", cTrace.WithOwner(snapParams.owner))
	defer span.End()

	// Skip snapshotting for LATEST type entitlements as the values fluctuate and snapshots can't be used
	if snapParams.meter.Aggregation == meter.MeterAggregationLatest {
		m.Logger.Debug("skipping snapshot for LATEST aggregation type entitlement", "owner", snapParams.owner, "meter", snapParams.meter.Key)
		return nil
	}

	segs := runRes.History.Segments()

	// i >= 1 because:
	// The first segment starts with the last valid snapshot and we don't want to create another snapshot for that same time
	for i := len(segs) - 1; i >= 1; i-- {
		seg := segs[i]

		// We can save a segment if its not after the current period start (this way backfilling, granting, resetting, etc... will work for the current UsagePeriod)
		if !seg.From.After(snapParams.notAfter) {
			snap, err := runRes.History.GetSnapshotAtStartOfSegment(i)
			if err != nil {
				return fmt.Errorf("failed to get snapshot at start of segment: %w", err)
			}

			if _, err := m.saveSnapshot(ctx, snapParams, snap); err != nil {
				return fmt.Errorf("failed to save snapshot: %w", err)
			}

			break
		}
	}

	return nil
}

func (m *connector) saveSnapshot(ctx context.Context, params snapshotParams, snap balance.Snapshot) (balance.Snapshot, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.saveSnapshot", cTrace.WithOwner(params.owner))
	defer span.End()

	// Let's validate the timestamp
	if !snap.At.Truncate(m.Granularity).Equal(snap.At) {
		return snap, fmt.Errorf("snapshot timestamp %s is not aligned to granularity %s", snap.At, m.Granularity)
	}

	if err := m.removeInactiveGrantsFromSnapshotAt(&snap, params.grants, snap.At); err != nil {
		return snap, fmt.Errorf("failed to remove inactive grants from snapshot: %w", err)
	}

	if err := m.BalanceSnapshotService.Save(ctx, params.owner, []balance.Snapshot{snap}); err != nil {
		return snap, fmt.Errorf("failed to save snapshot: %w", err)
	}

	m.Logger.DebugContext(ctx, "saved snapshot", "snapshot", snap, "owner", params.owner)

	return snap, nil
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
func SortedPeriodsFromDedupedTimes(ts []time.Time) []timeutil.ClosedPeriod {
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

	periods := make([]timeutil.ClosedPeriod, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		periods = append(periods, timeutil.ClosedPeriod{From: times[i-1], To: times[i]})
	}

	return periods
}
