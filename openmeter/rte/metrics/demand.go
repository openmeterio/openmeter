package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type DemandOperation string

const (
	DemandOperationPull        DemandOperation = "pull"
	DemandOperationInvalidate  DemandOperation = "invalidate"
	DemandOperationProcess     DemandOperation = "process"
	DemandOperationMaintenance DemandOperation = "maintenance"
)

type DemandOutcome string

const (
	DemandOutcomeError      DemandOutcome = "error"
	DemandOutcomeCacheHit   DemandOutcome = "cache_hit"
	DemandOutcomeCalculated DemandOutcome = "calculated"
	DemandOutcomeMarked     DemandOutcome = "marked"
	DemandOutcomeDuplicate  DemandOutcome = "duplicate"
	DemandOutcomeEmpty      DemandOutcome = "empty"
	DemandOutcomePruned     DemandOutcome = "pruned"
	DemandOutcomeRetired    DemandOutcome = "retired"
	DemandOutcomeRetry      DemandOutcome = "retry"
)

// Demand records freshness requests, invalidation matches and processing attempts.
// Pull and process counts overlap when EnsureFresh needs to refresh a key.
type Demand struct {
	operations     metric.Int64Counter
	ensureDuration metric.Float64Histogram
	invalidated    metric.Int64Counter
	queueWait      metric.Float64Histogram
	processing     metric.Float64Histogram
}

func NewDemand(meter metric.Meter) (*Demand, error) {
	operations, err := meter.Int64Counter("rte.demand.operations", metric.WithDescription("Demand operations by operation and outcome"))
	if err != nil {
		return nil, err
	}

	invalidated, err := meter.Int64Counter("rte.demand.invalidated", metric.WithDescription("Active calculation keys marked dirty"))
	if err != nil {
		return nil, err
	}

	ensureDuration, err := meter.Float64Histogram("rte.demand.ensure_fresh.duration", durationBuckets(), metric.WithUnit("s"),
		metric.WithDescription("Elapsed time to register demand and refresh a balance when needed"))
	if err != nil {
		return nil, err
	}

	queueWait, err := meter.Float64Histogram("rte.demand.queue_wait.duration", metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(.001, .01, .1, .5, 1, 2.5, 5, 10, 30, 60, 300, 1800, 3600),
		metric.WithDescription("Time eligible background work waited before the worker claimed it"))
	if err != nil {
		return nil, err
	}

	processing, err := meter.Float64Histogram("rte.demand.processing.duration", durationBuckets(), metric.WithUnit("s"),
		metric.WithDescription("Background processing from claim through transaction completion"))
	if err != nil {
		return nil, err
	}

	return &Demand{
		operations:     operations,
		invalidated:    invalidated,
		ensureDuration: ensureDuration,
		queueWait:      queueWait,
		processing:     processing,
	}, nil
}

func (m *Demand) Operation(ctx context.Context, operation DemandOperation, outcome DemandOutcome) {
	m.operations.Add(ctx, 1, metric.WithAttributes(attribute.String("operation", string(operation)), attribute.String("outcome", string(outcome))))
}

// EnsureFresh records the pull count and its duration together, including clean pulls.
func (m *Demand) EnsureFresh(ctx context.Context, kind rte.CalculationKind, start time.Time, outcome DemandOutcome) {
	m.Operation(ctx, DemandOperationPull, outcome)
	m.ensureDuration.Record(ctx, time.Since(start).Seconds(),
		metric.WithAttributes(attribute.String("kind", string(kind)), attribute.String("outcome", string(outcome))))
}

func (m *Demand) Invalidated(ctx context.Context, matched int) {
	if matched > 0 {
		m.invalidated.Add(ctx, int64(matched))
	}
}

// BackgroundAttempt records a finished claim, including a failed calculation deferred for retry.
func (m *Demand) BackgroundAttempt(ctx context.Context, timing rte.DemandAttemptTiming, outcome DemandOutcome) {
	labels := metric.WithAttributes(attribute.String("kind", string(timing.Kind)), attribute.String("outcome", string(outcome)))
	m.queueWait.Record(ctx, timing.QueueWait.Seconds(), labels)
	m.processing.Record(ctx, time.Since(timing.StartedAt).Seconds(), labels)
}
