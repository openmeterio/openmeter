package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

// The calculation kind and outcomes are bounded. A customer, entitlement or
// cache key would create a separate time series for every tracked balance.
type calculationMetrics struct {
	kind     rte.CalculationKind
	attempts metric.Int64Counter
	duration metric.Float64Histogram
	reads    metric.Int64Counter
}

func newCalculationMetrics(meter metric.Meter, kind rte.CalculationKind) (calculationMetrics, error) {
	m := calculationMetrics{kind: kind}
	var err error
	m.attempts, err = meter.Int64Counter("rte.calculations", metric.WithDescription("Exact calculations executed, by outcome"))
	if err != nil {
		return m, err
	}

	m.duration, err = meter.Float64Histogram("rte.calculation.duration", metric.WithUnit("s"), metric.WithDescription("Elapsed time to calculate and store a result"))
	if err != nil {
		return m, err
	}

	m.reads, err = meter.Int64Counter("rte.state.reads", metric.WithDescription("Stored balance reads, by outcome"))

	return m, err
}

func (m calculationMetrics) calculated(ctx context.Context, start time.Time, outcome string) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", outcome))
	m.attempts.Add(ctx, 1, labels)
	m.duration.Record(ctx, time.Since(start).Seconds(), labels)
}

func (m calculationMetrics) read(ctx context.Context, outcome string) {
	m.reads.Add(ctx, 1, metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", outcome)))
}
