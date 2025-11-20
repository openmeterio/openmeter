package cogspgx

import (
	"context"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/cogs"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

// PGXCOGSInterceptor is an interceptor that measures the time taken to execute queries against the database
// and records them in the cogs.postgres.round_trip_ms metric.
// Analytics can be extraced from this in combination with an estimate of network time (captured by cogs.postgres.ping_round_trip_ms)
// This approach works, because pgdriver.Interceptors are translated to pgx Tracers, at which point we already have a ready-to-use connection.
type PGXCOGSInterceptor struct {
	metrics *cogs.Metrics
}

var _ pgdriver.Interceptor = &PGXCOGSInterceptor{}

type contextKey string

const (
	contextKeyPGXCOGSInterceptor = contextKey("cogs.pgx.interceptor")
	contextKeyObserverPing       = contextKey("cogs.pgx.observer.ping")
)

// NewPGXCOGSInterceptor creates a new PGXCOGSInterceptor
func NewPGXCOGSInterceptor(metrics *cogs.Metrics) pgdriver.Interceptor {
	return &PGXCOGSInterceptor{metrics: metrics}
}

func (i *PGXCOGSInterceptor) Before(ctx context.Context) context.Context {
	start := clock.Now()
	return context.WithValue(ctx, contextKeyPGXCOGSInterceptor, start)
}

func (i *PGXCOGSInterceptor) After(ctx context.Context) {
	start, ok := ctx.Value(contextKeyPGXCOGSInterceptor).(time.Time)
	if !ok {
		slog.Default().DebugContext(ctx, "failed to get start time from context")
		return
	}

	durationMs := clock.Now().Sub(start).Milliseconds()

	ping, ok := ctx.Value(contextKeyObserverPing).(bool)
	if ok && ping {
		i.metrics.PostgresPingRoundTripMs.Record(ctx, durationMs)
		return
	}

	i.metrics.PostgresRoundTripMs.Record(ctx, durationMs)
}
