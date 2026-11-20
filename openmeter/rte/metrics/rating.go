package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Rating counts per-charge reuse after fresh usage has been read. A hit avoids
// repeating rating work; it does not avoid the usage query.
type Rating struct {
	lookups metric.Int64Counter
}

func NewRating(meter metric.Meter) (*Rating, error) {
	lookups, err := meter.Int64Counter("rte.rating.lookups", metric.WithDescription("Per-charge rating cache lookups"))
	if err != nil {
		return nil, err
	}

	return &Rating{lookups: lookups}, nil
}

func (m *Rating) Lookup(ctx context.Context, hit bool) {
	outcome := "miss"
	if hit {
		outcome = "hit"
	}

	m.lookups.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}
