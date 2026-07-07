package clickhouse

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// meterCacheMetricPrefix namespaces every meter cache instrument so they group together in
// any metrics backend regardless of what else the process emits.
const meterCacheMetricPrefix = "streaming.meter_cache."

// meterCacheObservability holds the OTel instruments the meter cache's cached read path,
// invalidation pipeline, and lifecycle reconciler emit. It is always constructed — with a
// nil Config.Meter/Config.Tracer the underlying instruments are no-ops — so callers never
// need a nil check before recording: see newMeterCacheObservability.
type meterCacheObservability struct {
	tracer trace.Tracer

	// queries counts every Cachable QueryMeter call that consulted the gate, tagged with
	// the gate's decision (result=cached|live_fallback|live_reject) and, for rejections,
	// the gate's reject reason. The pure live path for non-Cachable queries deliberately
	// never touches this counter (see queryMeterCached): instrumenting it would add
	// overhead to the byte-identical, unopted-in majority of traffic for a signal that
	// only matters for opted-in queries.
	queries metric.Int64Counter

	// queryDurationMS measures only the cached arm's execution (cache leg SQL plus the
	// outer merge across live pre/post legs); the plain live path is intentionally
	// unmeasured for the same reason queries is.
	queryDurationMS metric.Int64Histogram

	// markerFailures is the silent-staleness alert signal (G11): a lost classify or
	// insert failure is the only failure mode in the invalidation pipeline that can leave
	// cached reads serving stale buckets indefinitely, so this is the counter operators
	// must alert on, not merely observe.
	markerFailures metric.Int64Counter

	// refreshTriggers counts every best-effort SYSTEM REFRESH VIEW attempt the invalidator
	// makes after writing invalidation markers, tagged with its outcome.
	refreshTriggers metric.Int64Counter
}

// newMeterCacheObservability builds the instrument set from config.Meter and config.Tracer,
// substituting no-op implementations when either is nil. Substituting here — rather than
// nil-checking at every call site — keeps the cache's instrumentation calls unconditional:
// a deployment that never wires telemetry pays the (negligible) cost of a no-op instrument
// call instead of every recording site needing to guard against a nil dependency.
func newMeterCacheObservability(config Config) (*meterCacheObservability, error) {
	meter := config.Meter
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("noop")
	}

	tracer := config.Tracer
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	}

	queries, err := meter.Int64Counter(
		meterCacheMetricPrefix+"queries",
		metric.WithDescription("Number of Cachable meter queries the cache gate decided on, by result"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %squeries: %w", meterCacheMetricPrefix, err)
	}

	queryDurationMS, err := meter.Int64Histogram(
		meterCacheMetricPrefix+"query_duration_ms",
		metric.WithDescription("Duration of meter queries served from the cache, including the live leg merge"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %squery_duration_ms: %w", meterCacheMetricPrefix, err)
	}

	markerFailures, err := meter.Int64Counter(
		meterCacheMetricPrefix+"marker_failures",
		metric.WithDescription("Number of invalidation marker classify/insert failures; any non-zero rate risks silently stale cached reads"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %smarker_failures: %w", meterCacheMetricPrefix, err)
	}

	refreshTriggers, err := meter.Int64Counter(
		meterCacheMetricPrefix+"refresh_triggers",
		metric.WithDescription("Number of best-effort SYSTEM REFRESH VIEW triggers fired after late-event invalidation, by outcome"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: %srefresh_triggers: %w", meterCacheMetricPrefix, err)
	}

	return &meterCacheObservability{
		tracer:          tracer,
		queries:         queries,
		queryDurationMS: queryDurationMS,
		markerFailures:  markerFailures,
		refreshTriggers: refreshTriggers,
	}, nil
}
