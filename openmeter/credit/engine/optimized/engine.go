package optimizedengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credittrace "github.com/openmeterio/openmeter/openmeter/credit/trace"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Dependencies struct {
	OwnerConnector     grant.OwnerConnector
	StreamingConnector streaming.Connector
	Tracer             trace.Tracer
	Logger             *slog.Logger
}

func (d Dependencies) Validate() error {
	if d.OwnerConnector == nil {
		return errors.New("owner connector is required")
	}

	if d.StreamingConnector == nil {
		return errors.New("streaming connector is required")
	}

	if d.Tracer == nil {
		return errors.New("tracer is required")
	}

	if d.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

type Config struct {
	// We're only ever calculating for a single owner at a time
	Owner                 grant.Owner
	QueryBounds           timeutil.ClosedPeriod
	InbetweenPeriodStarts timeutil.SimpleTimeline
}

func (c Config) Validate() error {
	if c.QueryBounds.From.IsZero() || c.QueryBounds.To.IsZero() {
		return errors.New("query bounds must have both from and to set")
	}

	return nil
}

// Builds the engine for a given owner caching the period boundaries for the given range (queryBounds).
// As QueryUsageFn is frequently called, getting the CurrentUsagePeriodStartTime during it's execution would impact performance, so we cache all possible values during engine building.
func NewEngine(ctx context.Context, deps Dependencies, config Config) (engine.Engine, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate dependencies: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	ctx, span := deps.Tracer.Start(ctx, "credit.buildEngineForOwner", credittrace.WithOwner(config.Owner.NamespacedID), credittrace.WithPeriod(config.QueryBounds))
	defer span.End()

	// Let's validate the parameters
	if config.QueryBounds.From.IsZero() || config.QueryBounds.To.IsZero() {
		return nil, fmt.Errorf("query bounds must have both from and to set")
	}

	// Let's collect all period start times for any time between the query bounds
	// First we get the period start time for the start of the period, then all times in between
	firstPeriodStart, err := deps.OwnerConnector.GetUsagePeriodStartAt(ctx, config.Owner.NamespacedID, config.QueryBounds.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage period start time for owner %s at %s: %w", config.Owner.NamespacedID.ID, config.QueryBounds.From, err)
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

	// Let's add tracing
	usageQuerier = &usageQuerierWrapper{
		UsageQuerier: usageQuerier,
		Tracer:       deps.Tracer,
	}

	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(ctx context.Context, from, to time.Time) (float64, error) {
			// Let's validate we're not querying outside the bounds
			if !config.QueryBounds.ContainsInclusive(from) || !config.QueryBounds.ContainsInclusive(to) {
				return 0.0, fmt.Errorf("query bounds between %s and %s do not contain query from %s to %s: %t %t", config.QueryBounds.From, config.QueryBounds.To, from, to, config.QueryBounds.ContainsInclusive(from), config.QueryBounds.ContainsInclusive(to))
			}

			// If we're inside the period cache, we can just use the UsageQuerier
			return usageQuerier.QueryUsage(ctx, config.Owner.NamespacedID, timeutil.ClosedPeriod{From: from, To: to})
		},
	})

	return &engineWrapper{
		Engine: eng,
		Tracer: deps.Tracer,
		Logger: deps.Logger,
	}, nil
}
