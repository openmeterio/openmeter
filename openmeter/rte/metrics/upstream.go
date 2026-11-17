package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type UpstreamOutcome string

const (
	UpstreamOutcomeError    UpstreamOutcome = "error"
	UpstreamOutcomeNoTarget UpstreamOutcome = "no_target"
	UpstreamOutcomeRouted   UpstreamOutcome = "routed"
)

// Upstream records source handling, publication lag and target-routing costs.
// A routed event may still be a duplicate or match zero active demand rows.
type Upstream struct {
	events          metric.Int64Counter
	lag             metric.Float64Histogram
	routingDuration metric.Float64Histogram
}

func NewUpstream(meter metric.Meter) (*Upstream, error) {
	events, err := meter.Int64Counter("rte.upstream.events", metric.WithDescription("Supported source events by routing outcome"))
	if err != nil {
		return nil, err
	}

	lag, err := meter.Float64Histogram("rte.upstream.lag", metric.WithUnit("s"), metric.WithDescription("Source publication to RTE processing delay"))
	if err != nil {
		return nil, err
	}

	routingDuration, err := meter.Float64Histogram("rte.upstream.routing.duration", metric.WithUnit("s"), durationBuckets(),
		metric.WithDescription("Time to resolve a source event and mark matching balances dirty"))
	if err != nil {
		return nil, err
	}

	return &Upstream{events: events, lag: lag, routingDuration: routingDuration}, nil
}

func (m *Upstream) Event(ctx context.Context, name string, outcome UpstreamOutcome) {
	m.events.Add(ctx, 1, metric.WithAttributes(attribute.String("event", name), attribute.String("outcome", string(outcome))))
}

func (m *Upstream) Lag(ctx context.Context, name string, publishedAt time.Time) {
	m.lag.Record(ctx, max(0, time.Since(publishedAt).Seconds()), metric.WithAttributes(attribute.String("event", name)))
}

// Routing measures resolution and invalidation after any usage visibility wait.
func (m *Upstream) Routing(ctx context.Context, name string, start time.Time, outcome UpstreamOutcome) {
	m.routingDuration.Record(ctx, time.Since(start).Seconds(),
		metric.WithAttributes(attribute.String("event", name), attribute.String("outcome", string(outcome))))
}
