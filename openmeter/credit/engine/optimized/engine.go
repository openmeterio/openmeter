package optimizedengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credittrace "github.com/openmeterio/openmeter/openmeter/credit/trace"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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

// Builds an optimized version of the engine
func NewEngine(ctx context.Context, deps Dependencies, config Config) (engine.Engine, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate dependencies: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	ctx, span := deps.Tracer.Start(ctx, "credit.buildEngineForOwner", credittrace.WithOwner(config.Owner.NamespacedID), credittrace.WithPeriod(config.QueryBounds))
	defer span.End()

	usageQuerier, guard, err := optimizePeriodFetching(ctx, deps, config)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize period fetching: %w", err)
	}

	// Let's add tracing
	usageQuerier = &usageQuerierWrapper{
		UsageQuerier: usageQuerier,
		Tracer:       deps.Tracer,
	}

	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(ctx context.Context, from, to time.Time) (float64, error) {
			if err := guard(ctx, from, to); err != nil {
				return 0.0, err
			}

			return usageQuerier.QueryUsage(ctx, config.Owner.NamespacedID, timeutil.ClosedPeriod{From: from, To: to})
		},
	})

	return &engineWrapper{
		Engine: eng,
		Tracer: deps.Tracer,
		Logger: deps.Logger,
	}, nil
}
