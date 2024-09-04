package pgxpoolobserver

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ObservePoolMetrics registers a callback that observes the metrics of the provided pgxpool.Pool.
// the implementation is based on https://github.com/cmackenzie1/pgxpool-prometheus
func ObservePoolMetrics(meter metric.Meter, pool *pgxpool.Pool, additionalAttributes ...attribute.KeyValue) error {
	allMetrics := []metric.Observable{}

	acquireCountMetric, err := meter.Int64ObservableCounter(
		"pgxpool.acquire_count",
		metric.WithDescription("The cumulative count of successful acquires from the pool."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, acquireCountMetric)

	acquiredDurationMetric, err := meter.Int64ObservableGauge(
		"pgxpool.acquire_duration",
		metric.WithDescription("The total duration of all successful acquires from the pool in ms."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, acquiredDurationMetric)

	avgAcquiredDurationMetric, err := meter.Int64ObservableGauge(
		"pgxpool.acquire_duration_avg",
		metric.WithDescription("The average duration of all successful acquires from the pool in ms."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, avgAcquiredDurationMetric)

	acquiredConnsMetric, err := meter.Int64ObservableGauge(
		"pgxpool.acquired_conns",
		metric.WithDescription("The number of currently acquired connections in the pool."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, acquiredConnsMetric)

	canceledAcquireCountMetric, err := meter.Int64ObservableCounter(
		"pgxpool.canceled_acquire_count",
		metric.WithDescription("The cumulative count of acquires from the pool that were canceled by a context."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, canceledAcquireCountMetric)

	constructingConnsMetric, err := meter.Int64ObservableGauge(
		"pgxpool.constructing_conns",
		metric.WithDescription("The number of conns with construction in progress in the pool."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, constructingConnsMetric)

	emptyAcquireCountMetric, err := meter.Int64ObservableCounter(
		"pgxpool.empty_acquire_count",
		metric.WithDescription("The cumulative count of successful acquires from the pool that waited for a resource to be released or constructed because the pool was empty."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, emptyAcquireCountMetric)

	idleConnsMetric, err := meter.Int64ObservableGauge(
		"pgxpool.idle_conns",
		metric.WithDescription("The number of currently idle conns in the pool."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, idleConnsMetric)

	maxConns, err := meter.Int64ObservableGauge(
		"pgxpool.max_conns",
		metric.WithDescription("The maximum size of the pool."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, maxConns)

	totalConns, err := meter.Int64ObservableGauge(
		"pgxpool.total_conns",
		metric.WithDescription("The total number of resources currently in the pool. The value is the sum of ConstructingConns, AcquiredConns, and IdleConns."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, totalConns)

	newConnsCount, err := meter.Int64ObservableCounter(
		"pgxpool.new_conns_count",
		metric.WithDescription("The cumulative count of new connections opened."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, newConnsCount)

	maxLifetimeDestroyCount, err := meter.Int64ObservableCounter(
		"pgxpool.max_lifetime_destroy_count",
		metric.WithDescription("The cumulative count of connections closed due to reaching their maximum lifetime (MaxConnLifetime)."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, maxLifetimeDestroyCount)

	maxIdleDestroyCount, err := meter.Int64ObservableCounter(
		"pgxpool.max_idle_destroy_count",
		metric.WithDescription("The cumulative count of connections closed due to reaching their maximum idle time (MaxConnIdleTime)."),
	)
	if err != nil {
		return err
	}
	allMetrics = append(allMetrics, maxIdleDestroyCount)

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		stat := pool.Stat()
		o.ObserveInt64(acquireCountMetric, stat.AcquireCount(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(acquiredDurationMetric, stat.AcquireDuration().Milliseconds(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(avgAcquiredDurationMetric, stat.AcquireDuration().Milliseconds()/stat.AcquireCount(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(acquiredConnsMetric, int64(stat.AcquiredConns()), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(canceledAcquireCountMetric, stat.CanceledAcquireCount(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(constructingConnsMetric, int64(stat.ConstructingConns()), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(emptyAcquireCountMetric, stat.EmptyAcquireCount(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(idleConnsMetric, int64(stat.IdleConns()), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(maxConns, int64(stat.MaxConns()), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(totalConns, int64(stat.TotalConns()), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(newConnsCount, stat.NewConnsCount(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(maxLifetimeDestroyCount, stat.MaxLifetimeDestroyCount(), metric.WithAttributes(additionalAttributes...))
		o.ObserveInt64(maxIdleDestroyCount, stat.MaxIdleDestroyCount(), metric.WithAttributes(additionalAttributes...))

		return nil
	}, allMetrics...)
	if err != nil {
		return err
	}

	return nil
}
