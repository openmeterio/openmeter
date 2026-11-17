package worker

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
)

type comparisonMetrics struct {
	duration   metric.Float64Histogram
	lag        metric.Float64Histogram
	executions metric.Int64Counter
}

func newComparisonMetrics(meter metric.Meter) (*comparisonMetrics, error) {
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

	return &comparisonMetrics{duration: duration, lag: lag, executions: executions}, nil
}

func (w *worker) handleComparison(ctx context.Context, event *rte.BalanceComparisonEvent) error {
	kind := attribute.String("kind", string(event.Kind))
	op := attribute.String("operation", string(event.Operation))
	lag := max(time.Since(event.ObservedAt), 0)
	w.comparison.lag.Record(ctx, float64(lag)/float64(time.Millisecond), metric.WithAttributes(kind, op))

	if lag >= rte.ComparisonObservationTTL {
		w.comparison.executions.Add(ctx, 1, metric.WithAttributes(kind, op, attribute.String("outcome", "expired")))
		return nil
	}

	key, err := event.DecodeKey()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	err = w.replayComparison(ctx, event, key)
	duration := time.Since(start)
	if err != nil {
		// Retrying would turn one observed request into multiple calculations
		// and block source events behind a failed experiment.
		w.comparison.executions.Add(ctx, 1, metric.WithAttributes(kind, op, attribute.String("outcome", "error")))
		w.logger.DebugContext(ctx, "RTE comparison failed", "kind", event.Kind, "operation", event.Operation, "error", err)

		return nil
	}

	w.comparison.duration.Record(ctx, float64(event.DirectDuration)/float64(time.Millisecond), metric.WithAttributes(kind, op, attribute.String("variant", "direct")))
	w.comparison.duration.Record(ctx, float64(duration)/float64(time.Millisecond), metric.WithAttributes(kind, op, attribute.String("variant", "rte")))
	w.comparison.executions.Add(ctx, 1, metric.WithAttributes(kind, op, attribute.String("outcome", "success")))

	return nil
}

func (w *worker) replayComparison(ctx context.Context, event *rte.BalanceComparisonEvent, key rte.Key) error {
	if event.Operation == rte.ComparisonOperationRead {
		if err := w.demand.EnsureFresh(ctx, rte.EnsureFreshInput{
			Target: event.CalculationTarget, PullTTL: time.Minute, MaxAge: 30 * time.Second,
		}); err != nil {
			return err
		}
	}

	switch key := key.(type) {
	case creditbalance.Key:
		return replayBalance(ctx, w.service.CreditBalances(), key, event.Operation)
	case meteredentitlement.Key:
		return replayBalance(ctx, w.service.MeteredEntitlements(), key, event.Operation)
	default:
		return fmt.Errorf("unsupported RTE comparison key %T", key)
	}
}

func replayBalance[K rte.Key, V any](ctx context.Context, balance rte.Calculation[K, V], key K, operation rte.ComparisonOperation) error {
	if operation == rte.ComparisonOperationRead {
		_, err := balance.Get(ctx, key)
		return err
	}

	_, err := balance.Calculate(ctx, key)

	return err
}
