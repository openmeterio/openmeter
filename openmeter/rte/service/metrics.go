package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type calculationOutcome string

const (
	calculationOutcomeUpdated    calculationOutcome = "updated"
	calculationOutcomeSuperseded calculationOutcome = "superseded"
	calculationOutcomeFailed     calculationOutcome = "failed"
)

type readOutcome string

const (
	readOutcomeHit     readOutcome = "hit"
	readOutcomeMissing readOutcome = "missing"
	readOutcomeFailed  readOutcome = "failed"
)

// The calculation kind and outcomes are bounded. A customer, entitlement or
// cache key would create a separate time series for every tracked balance.
type calculationMetrics struct {
	kind         rte.CalculationKind
	attempts     metric.Int64Counter
	duration     metric.Float64Histogram
	reads        metric.Int64Counter
	readDuration metric.Float64Histogram
}

func newCalculationMetrics(meter metric.Meter, kind rte.CalculationKind) (calculationMetrics, error) {
	m := calculationMetrics{kind: kind}
	var err error
	m.attempts, err = meter.Int64Counter("rte.calculations", metric.WithDescription("Exact calculations executed, by outcome"))
	if err != nil {
		return m, err
	}

	m.duration, err = meter.Float64Histogram("rte.calculation.duration", rte.DurationBuckets(), metric.WithUnit("s"), metric.WithDescription("Elapsed time to calculate and store a result"))
	if err != nil {
		return m, err
	}

	m.reads, err = meter.Int64Counter("rte.state.reads", metric.WithDescription("Stored balance reads, by outcome"))
	if err != nil {
		return m, err
	}

	m.readDuration, err = meter.Float64Histogram("rte.state.read.duration", rte.DurationBuckets(), metric.WithUnit("s"), metric.WithDescription("Elapsed time to read stored state"))

	return m, err
}

func (m calculationMetrics) calculated(ctx context.Context, start time.Time, outcome calculationOutcome) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", string(outcome)))
	m.attempts.Add(ctx, 1, labels)
	m.duration.Record(ctx, time.Since(start).Seconds(), labels)
}

func (m calculationMetrics) read(ctx context.Context, start time.Time, outcome readOutcome) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", string(outcome)))
	m.reads.Add(ctx, 1, labels)
	m.readDuration.Record(ctx, time.Since(start).Seconds(), labels)
}
