package metercache

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// metercacheMetricPrefix namespaces every reconciler instrument alongside the connector's
// streaming.meter_cache.* instruments (see clickhouse.meterCacheMetricPrefix); the two
// packages intentionally share one metric namespace since both describe the same cache's
// health from different vantage points.
const metercacheMetricPrefix = "streaming.meter_cache."

// passStatus is the per-view health classification the reconcile_pass_views gauge reports
// (see passStatusFor in reconciler.go): healthy (the view is converged, no action needed),
// stale (no deployed view yet, or a deployed view still converging via repair/recreate/
// marker-expiry backfill), unstamped (a deployed view whose BackfilledAt is nil, i.e. a
// create-or-backfill sequence left mid-flight per G3), or exception. exception is the one
// pass-level status: it is tallied once when reconcile() itself aborts (ListMeters or
// ListActualViews failed) rather than per view, so a nonzero exception count on this
// per-view gauge means the whole pass failed before any view could be classified, not that
// one view is broken.
type passStatus string

const (
	passStatusHealthy   passStatus = "healthy"
	passStatusStale     passStatus = "stale"
	passStatusException passStatus = "exception"
	passStatusUnstamped passStatus = "unstamped"
)

// reconcileOp is the reconcile_ops counter's op attribute: the convergence action the
// reconciler took for one desired view, plus gc for the orphan-row cleanup that runs once
// per pass rather than once per view.
type reconcileOp string

const (
	reconcileOpCreate   reconcileOp = "create"
	reconcileOpBackfill reconcileOp = "backfill"
	reconcileOpDrop     reconcileOp = "drop"
	reconcileOpRepair   reconcileOp = "repair"
	reconcileOpGC       reconcileOp = "gc"
)

type reconcileOpOutcome string

const (
	reconcileOpOutcomeOK    reconcileOpOutcome = "ok"
	reconcileOpOutcomeError reconcileOpOutcome = "error"
)

// passSnapshot is the most recent pass's status-per-view counts, read by the observable
// gauge callback. It is stored behind an atomic.Pointer so the callback (invoked by the
// SDK's collection goroutine) never races with the reconcile loop publishing a new pass.
type passSnapshot struct {
	counts map[passStatus]int64
}

// observability holds the meter cache reconciler's OTel instruments plus the atomic
// snapshot its observable gauge callback reads. It is always non-nil on a constructed
// Reconciler — see newObservability — so reconcile() never needs a nil check before
// recording.
type observability struct {
	tracer trace.Tracer

	reconcileOps        metric.Int64Counter
	reconcileDurationMS metric.Int64Histogram
	capabilityDisabled  metric.Int64Counter

	lastPass atomic.Pointer[passSnapshot]
}

// newObservability builds the instrument set from meter and tracer, substituting no-op
// implementations when either is nil (the reconciler's cache observability is optional,
// matching the clickhouse connector's Config.Meter/Config.Tracer contract).
func newObservability(meter metric.Meter, tracer trace.Tracer) (*observability, error) {
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("noop")
	}

	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	}

	o := &observability{tracer: tracer}
	o.lastPass.Store(&passSnapshot{counts: map[passStatus]int64{}})

	reconcileOps, err := meter.Int64Counter(
		metercacheMetricPrefix+"reconcile_ops",
		metric.WithDescription("Number of meter cache view convergence operations the reconciler performed, by op and outcome"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %sreconcile_ops: %w", metercacheMetricPrefix, err)
	}
	o.reconcileOps = reconcileOps

	reconcileDurationMS, err := meter.Int64Histogram(
		metercacheMetricPrefix+"reconcile_duration_ms",
		metric.WithDescription("Duration of one meter cache reconciliation pass"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %sreconcile_duration_ms: %w", metercacheMetricPrefix, err)
	}
	o.reconcileDurationMS = reconcileDurationMS

	capabilityDisabled, err := meter.Int64Counter(
		metercacheMetricPrefix+"capability_disabled",
		metric.WithDescription("Incremented once when the capability probe permanently disables the meter cache reconciler for this process"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %scapability_disabled: %w", metercacheMetricPrefix, err)
	}
	o.capabilityDisabled = capabilityDisabled

	views, err := meter.Int64ObservableGauge(
		metercacheMetricPrefix+"reconcile_pass_views",
		metric.WithDescription("Number of views in the most recent reconciliation pass, by status"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %sreconcile_pass_views: %w", metercacheMetricPrefix, err)
	}

	_, err = meter.RegisterCallback(func(_ context.Context, obs metric.Observer) error {
		snapshot := o.lastPass.Load()

		for status, count := range snapshot.counts {
			obs.ObserveInt64(views, count, metric.WithAttributes(attribute.String("status", string(status))))
		}

		return nil
	}, views)
	if err != nil {
		return nil, fmt.Errorf("failed to register callback for %sreconcile_pass_views: %w", metercacheMetricPrefix, err)
	}

	return o, nil
}

// recordOp emits the reconcile_ops counter for one convergence action.
func (o *observability) recordOp(ctx context.Context, op reconcileOp, outcome reconcileOpOutcome) {
	o.reconcileOps.Add(ctx, 1, metric.WithAttributes(
		attribute.String("op", string(op)),
		attribute.String("outcome", string(outcome)),
	))
}

// recordPass publishes the pass's per-view status counts for the reconcile_pass_views
// observable gauge and records the pass duration histogram.
func (o *observability) recordPass(ctx context.Context, counts map[passStatus]int64, duration int64) {
	o.lastPass.Store(&passSnapshot{counts: counts})
	o.reconcileDurationMS.Record(ctx, duration)
}

// recordCapabilityDisabled increments the capability_disabled counter. It is expected to
// fire at most once per process lifetime (the reconciler latches disabled and never probes
// again), so any value above 1 across a fleet points at repeated restarts hitting the same
// unsupported deployment.
func (o *observability) recordCapabilityDisabled(ctx context.Context) {
	o.capabilityDisabled.Add(ctx, 1)
}
