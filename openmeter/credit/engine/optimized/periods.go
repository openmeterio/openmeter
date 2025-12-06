package optimizedengine

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Caches the period boundaries for the given range (queryBounds) and owner.
// As QueryUsageFn is frequently called, getting the CurrentUsagePeriodStartTime during it's execution would impact performance, so we cache all possible values during engine building.
func optimizePeriodFetching(ctx context.Context, deps Dependencies, config Config) (balance.UsageQuerier, OptimizationGuard, error) {
	// Let's validate the parameters
	if config.QueryBounds.From.IsZero() || config.QueryBounds.To.IsZero() {
		return nil, nil, fmt.Errorf("query bounds must have both from and to set")
	}

	// Let's collect all period start times for any time between the query bounds
	// First we get the period start time for the start of the period, then all times in between
	firstPeriodStart, err := deps.OwnerConnector.GetUsagePeriodStartAt(ctx, config.Owner.NamespacedID, config.QueryBounds.From)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get usage period start time for owner %s at %s: %w", config.Owner.NamespacedID.ID, config.QueryBounds.From, err)
	}

	times := append([]time.Time{firstPeriodStart}, config.InbetweenPeriodStarts.GetTimes()...)
	times = append(times, config.QueryBounds.To)

	periodCache := timeutil.NewSimpleTimeline(lo.UniqBy(times, func(t time.Time) int64 {
		// We unique by unixnano because time.Time == time.Time comparison is finicky
		return t.UnixNano()
	})).GetClosedPeriods()

	if len(periodCache) == 0 {
		// If we didn't have at least 2 different timestamps, we need to create a period from the first start time and the bound
		periodCache = []timeutil.ClosedPeriod{{From: firstPeriodStart, To: config.QueryBounds.To}}
	}

	// We build a custom UsageQuerier for our usecase here. The engine should only ever query the one owner we fetched above.
	usageQuerier := balance.NewUsageQuerier(balance.UsageQuerierConfig{
		StreamingConnector: deps.StreamingConnector,
		DescribeOwner: func(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
			if id != config.Owner.NamespacedID {
				return grant.Owner{}, fmt.Errorf("expected owner %s, got %s", config.Owner.NamespacedID.ID, id.ID)
			}
			return config.Owner, nil
		},
		GetDefaultParams: func(ctx context.Context, oID models.NamespacedID) (streaming.QueryParams, error) {
			if oID != config.Owner.NamespacedID {
				return streaming.QueryParams{}, fmt.Errorf("expected owner %s, got %s", config.Owner.NamespacedID.ID, oID.ID)
			}
			return config.Owner.DefaultQueryParams, nil
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

	guard := func(ctx context.Context, from, to time.Time) error {
		// Let's validate we're not querying outside the bounds
		if !config.QueryBounds.ContainsInclusive(from) || !config.QueryBounds.ContainsInclusive(to) {
			return fmt.Errorf("query bounds between %s and %s do not contain query from %s to %s: %t %t", config.QueryBounds.From, config.QueryBounds.To, from, to, config.QueryBounds.ContainsInclusive(from), config.QueryBounds.ContainsInclusive(to))
		}

		return nil
	}

	return usageQuerier, guard, nil
}
