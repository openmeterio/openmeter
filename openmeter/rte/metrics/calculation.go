package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type CalculationOutcome string

type CalculationMethod string

const (
	CalculationMethodFull     CalculationMethod = "full"
	CalculationMethodHit      CalculationMethod = "checkpoint_hit"
	CalculationMethodAdvanced CalculationMethod = "checkpoint_advanced"
	CalculationMethodMiss     CalculationMethod = "checkpoint_miss"
)

const (
	CalculationOutcomeUpdated    CalculationOutcome = "updated"
	CalculationOutcomeSuperseded CalculationOutcome = "superseded"
	CalculationOutcomeFailed     CalculationOutcome = "failed"
)

// Calculation records execution and stored-state reads for one calculation kind.
// Labels are bounded: customer IDs or cache keys would create a time series per balance.
type Calculation struct {
	StateReads
	kind     rte.CalculationKind
	attempts metric.Int64Counter
	duration metric.Float64Histogram
}

func NewCalculation(meter metric.Meter, kind rte.CalculationKind) (Calculation, error) {
	m := Calculation{kind: kind}
	var err error
	m.attempts, err = meter.Int64Counter("rte.calculations", metric.WithDescription("Balance calculations by kind, method and outcome"))
	if err != nil {
		return m, err
	}

	m.duration, err = meter.Float64Histogram("rte.calculation.duration", durationBuckets(), metric.WithUnit("s"), metric.WithDescription("Elapsed time to calculate and store a result"))
	if err != nil {
		return m, err
	}

	m.StateReads, err = NewStateReads(meter, kind)

	return m, err
}

// Calculated records a service-call outcome, including failed and superseded attempts.
// An updated result can still be rolled back by an enclosing demand transaction.
func (m Calculation) Calculated(ctx context.Context, start time.Time, outcome CalculationOutcome, method CalculationMethod) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", string(outcome)), attribute.String("method", string(method)))
	m.attempts.Add(ctx, 1, labels)
	m.duration.Record(ctx, time.Since(start).Seconds(), labels)
}
