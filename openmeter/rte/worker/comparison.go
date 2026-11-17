package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	rtemetrics "github.com/openmeterio/openmeter/openmeter/rte/metrics"
)

func (w *worker) handleComparison(ctx context.Context, event *rte.BalanceComparisonEvent) error {
	lag := max(time.Since(event.ObservedAt), 0)
	w.comparison.Lag(ctx, event, lag)

	if lag >= rte.ComparisonObservationTTL {
		w.comparison.Execution(ctx, event, rtemetrics.ComparisonOutcomeExpired)
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
		w.comparison.Execution(ctx, event, rtemetrics.ComparisonOutcomeError)
		w.logger.DebugContext(ctx, "RTE comparison failed", "kind", event.Kind, "operation", event.Operation, "error", err)

		return nil
	}

	w.comparison.Compared(ctx, event, duration)

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
