package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type ComparisonOutcome string

const (
	ComparisonOutcomeExpired ComparisonOutcome = "expired"
	ComparisonOutcomeError   ComparisonOutcome = "error"
	ComparisonOutcomeSuccess ComparisonOutcome = "success"
)

// Comparison records worker-side shadow replay outcomes and paired timings.
// Queue delay is kept separate from execution time.
type Comparison struct {
	duration   metric.Float64Histogram
	lag        metric.Float64Histogram
	executions metric.Int64Counter
}

func NewComparison(meter metric.Meter) (*Comparison, error) {
	duration, err := meter.Float64Histogram("rte.comparison.duration", metric.WithUnit("ms"),
		metric.WithDescription("Matched production and RTE operation durations, excluding replay lag"),
		metric.WithExplicitBucketBoundaries(0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000))
	if err != nil {
		return nil, err
	}

	lag, err := meter.Float64Histogram("rte.comparison.lag", metric.WithUnit("ms"),
		metric.WithDescription("Delay from production observation to RTE processing"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 50, 100, 500, 1000, 5000, 10000, 30000, 60000, 300000))
	if err != nil {
		return nil, err
	}

	executions, err := meter.Int64Counter("rte.comparison.executions", metric.WithDescription("Shadow operations by outcome"))
	if err != nil {
		return nil, err
	}

	return &Comparison{duration: duration, lag: lag, executions: executions}, nil
}

func (m *Comparison) Lag(ctx context.Context, event *rte.BalanceComparisonEvent, lag time.Duration) {
	m.lag.Record(ctx, float64(lag)/float64(time.Millisecond), metric.WithAttributes(
		attribute.String("kind", string(event.Kind)), attribute.String("operation", string(event.Operation))))
}

func (m *Comparison) Execution(ctx context.Context, event *rte.BalanceComparisonEvent, outcome ComparisonOutcome) {
	m.executions.Add(ctx, 1, metric.WithAttributes(
		attribute.String("kind", string(event.Kind)), attribute.String("operation", string(event.Operation)), attribute.String("outcome", string(outcome))))
}

// Compared records both timings only after a successful replay, keeping the
// direct and RTE samples paired. Replay lag is measured separately.
func (m *Comparison) Compared(ctx context.Context, event *rte.BalanceComparisonEvent, duration time.Duration) {
	kind := attribute.String("kind", string(event.Kind))
	op := attribute.String("operation", string(event.Operation))
	m.duration.Record(ctx, float64(event.DirectDuration)/float64(time.Millisecond), metric.WithAttributes(kind, op, attribute.String("variant", "direct")))
	m.duration.Record(ctx, float64(duration)/float64(time.Millisecond), metric.WithAttributes(kind, op, attribute.String("variant", "rte")))
	m.Execution(ctx, event, ComparisonOutcomeSuccess)
}
