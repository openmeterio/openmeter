package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type CalculationOutcome string

const (
	CalculationOutcomeUpdated    CalculationOutcome = "updated"
	CalculationOutcomeSuperseded CalculationOutcome = "superseded"
	CalculationOutcomeFailed     CalculationOutcome = "failed"
)

type ReadOutcome string

const (
	ReadOutcomeHit     ReadOutcome = "hit"
	ReadOutcomeMissing ReadOutcome = "missing"
	ReadOutcomeFailed  ReadOutcome = "failed"
)

// Calculation records execution and stored-state reads for one calculation kind.
// Labels are bounded: customer IDs or cache keys would create a time series per balance.
type Calculation struct {
	kind         rte.CalculationKind
	attempts     metric.Int64Counter
	duration     metric.Float64Histogram
	reads        metric.Int64Counter
	readDuration metric.Float64Histogram
}

func NewCalculation(meter metric.Meter, kind rte.CalculationKind) (Calculation, error) {
	m := Calculation{kind: kind}
	var err error
	m.attempts, err = meter.Int64Counter("rte.calculations", metric.WithDescription("Exact calculations executed, by outcome"))
	if err != nil {
		return m, err
	}

	m.duration, err = meter.Float64Histogram("rte.calculation.duration", durationBuckets(), metric.WithUnit("s"), metric.WithDescription("Elapsed time to calculate and store a result"))
	if err != nil {
		return m, err
	}

	m.reads, err = meter.Int64Counter("rte.state.reads", metric.WithDescription("Stored balance reads, by outcome"))
	if err != nil {
		return m, err
	}

	m.readDuration, err = meter.Float64Histogram("rte.state.read.duration", durationBuckets(), metric.WithUnit("s"), metric.WithDescription("Elapsed time to read stored state"))

	return m, err
}

// Calculated records a service-call outcome, including failed and superseded attempts.
// An updated result can still be rolled back by an enclosing demand transaction.
func (m Calculation) Calculated(ctx context.Context, start time.Time, outcome CalculationOutcome) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", string(outcome)))
	m.attempts.Add(ctx, 1, labels)
	m.duration.Record(ctx, time.Since(start).Seconds(), labels)
}

// Read records stored-state access through the calculation service. Direct store
// reads, including the ordinary database-backed client, do not call this recorder.
func (m Calculation) Read(ctx context.Context, start time.Time, outcome ReadOutcome) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", string(outcome)))
	m.reads.Add(ctx, 1, labels)
	m.readDuration.Record(ctx, time.Since(start).Seconds(), labels)
}
