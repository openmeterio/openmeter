package metercache

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
)

// testTelemetry bundles a ManualReader-backed MeterProvider and a SpanRecorder-backed
// TracerProvider, mirroring the clickhouse package's own testTelemetry: tests must assert
// that an instrument actually recorded, not merely that construction succeeded (a
// detached instrument looks like a healthy quiet system).
type testTelemetry struct {
	reader   *sdkmetric.ManualReader
	recorder *tracetest.SpanRecorder
}

func newTestTelemetry() *testTelemetry {
	return &testTelemetry{
		reader:   sdkmetric.NewManualReader(),
		recorder: tracetest.NewSpanRecorder(),
	}
}

// newInstrumentedTestReconciler builds a pass-only reconciler like newTestReconciler, but
// wired to real (ManualReader/SpanRecorder-backed) instruments instead of no-ops, so tests
// can assert emission.
func newInstrumentedTestReconciler(t *testing.T, connector Connector, meters meter.Service, telemetry *testTelemetry) *Reconciler {
	t.Helper()

	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(telemetry.reader))
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(telemetry.recorder))

	observability, err := newObservability(meterProvider.Meter("test"), tracerProvider.Tracer("test"))
	require.NoError(t, err)

	return &Reconciler{
		logger:        slog.Default(),
		connector:     connector,
		meters:        meters,
		observability: observability,
	}
}

func collectMetric(t *testing.T, reader *sdkmetric.ManualReader, name string) metricdata.Metrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}

	require.Failf(t, "metric not emitted", "expected metric %q to have been recorded", name)

	return metricdata.Metrics{}
}

func sumAttrValue(t *testing.T, m metricdata.Metrics, expected map[string]string) int64 {
	t.Helper()

	sum, ok := m.Data.(metricdata.Sum[int64])
	require.Truef(t, ok, "%s must be a counter", m.Name)

	for _, dp := range sum.DataPoints {
		matched := true

		for key, want := range expected {
			got, ok := dp.Attributes.Value(attribute.Key(key))
			if !ok || got.AsString() != want {
				matched = false
				break
			}
		}

		if matched {
			return dp.Value
		}
	}

	require.Failf(t, "no matching data point", "metric %s has no data point with attributes %v", m.Name, expected)

	return 0
}

func gaugeAttrValue(t *testing.T, m metricdata.Metrics, expected map[string]string) (int64, bool) {
	t.Helper()

	gauge, ok := m.Data.(metricdata.Gauge[int64])
	require.Truef(t, ok, "%s must be a gauge", m.Name)

	for _, dp := range gauge.DataPoints {
		matched := true

		for key, want := range expected {
			got, ok := dp.Attributes.Value(attribute.Key(key))
			if !ok || got.AsString() != want {
				matched = false
				break
			}
		}

		if matched {
			return dp.Value, true
		}
	}

	return 0, false
}

// TestReconcile_EmitsCreateOpsAndPassMetrics proves a pass that creates a missing view
// records reconcile_ops with op=create and op=backfill (EnsureMeterCache performs both
// atomically, see reconcileOpsFor), op=gc for the orphan-row cleanup, records a
// reconcile_pass span, and records the reconcile_pass_views gauge with the new view under
// status=stale (passStatusFor classifies a nil actualView as stale: fakeConnector's
// ListActualViews reflects the pre-ensure state, since the fake does not model
// EnsureMeterCache's effect on actual view state).
func TestReconcile_EmitsCreateOpsAndPassMetrics(t *testing.T) {
	telemetry := newTestTelemetry()

	m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
	connector := &fakeConnector{}

	r := newInstrumentedTestReconciler(t, connector, &fakeMeterService{meters: []meter.Meter{m}}, telemetry)
	require.NoError(t, r.reconcile(t.Context()))

	ops := collectMetric(t, telemetry.reader, "streaming.meter_cache.reconcile_ops")
	require.EqualValues(t, 1, sumAttrValue(t, ops, map[string]string{"op": "create", "outcome": "ok"}))
	require.EqualValues(t, 1, sumAttrValue(t, ops, map[string]string{"op": "backfill", "outcome": "ok"}))
	require.EqualValues(t, 1, sumAttrValue(t, ops, map[string]string{"op": "gc", "outcome": "ok"}))

	durationMetric := collectMetric(t, telemetry.reader, "streaming.meter_cache.reconcile_duration_ms")
	hist, ok := durationMetric.Data.(metricdata.Histogram[int64])
	require.True(t, ok, "reconcile_duration_ms must be a histogram")
	require.Len(t, hist.DataPoints, 1)

	viewsMetric := collectMetric(t, telemetry.reader, "streaming.meter_cache.reconcile_pass_views")
	count, ok := gaugeAttrValue(t, viewsMetric, map[string]string{"status": "stale"})
	require.True(t, ok, "a missing-view pass must report at least one stale-status view")
	require.EqualValues(t, 1, count)

	spans := telemetry.recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "streaming.meter_cache.reconcile_pass", spans[0].Name())
}

// TestReconcile_DropOpEmitsOutcome proves dropping an undesired view (a meter no longer
// exists) is credited to reconcile_ops with op=drop, outcome=ok.
func TestReconcile_DropOpEmitsOutcome(t *testing.T) {
	telemetry := newTestTelemetry()

	m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
	view := deployedView("ns-1", m, "", time.Now())
	connector := &fakeConnector{actual: []clickhouse.MeterCacheView{view}}

	r := newInstrumentedTestReconciler(t, connector, &fakeMeterService{}, telemetry)
	require.NoError(t, r.reconcile(t.Context()))

	ops := collectMetric(t, telemetry.reader, "streaming.meter_cache.reconcile_ops")
	require.EqualValues(t, 1, sumAttrValue(t, ops, map[string]string{"op": "drop", "outcome": "ok"}))
}

// TestReconcile_CapabilityDisabledEmitsCounter proves the capability probe's permanent
// disablement is counted, distinct from a transient probe failure.
func TestReconcile_CapabilityDisabledEmitsCounter(t *testing.T) {
	telemetry := newTestTelemetry()

	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(telemetry.reader))
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(telemetry.recorder))

	observability, err := newObservability(meterProvider.Meter("test"), tracerProvider.Tracer("test"))
	require.NoError(t, err)

	connector := &fakeConnector{probeErr: clickhouse.ErrMeterCacheUnsupported}

	r := &Reconciler{
		logger:            slog.Default(),
		connector:         connector,
		meters:            &fakeMeterService{},
		observability:     observability,
		enabled:           true,
		reconcileInterval: time.Millisecond,
		stopCh:            make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	err = r.lead(ctx)
	require.NoError(t, err)
	require.True(t, r.disabled.Load())

	m := collectMetric(t, telemetry.reader, "streaming.meter_cache.capability_disabled")
	sum, ok := m.Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.Len(t, sum.DataPoints, 1)
	require.EqualValues(t, 1, sum.DataPoints[0].Value)
}

// TestNewObservability_NilDependenciesDoNotPanic proves the reconciler's instrumented
// paths run without panicking when observability is built from nil Meter/Tracer — the
// deployment shape where telemetry was never wired.
func TestNewObservability_NilDependenciesDoNotPanic(t *testing.T) {
	observability, err := newObservability(nil, nil)
	require.NoError(t, err)

	m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
	connector := &fakeConnector{}

	r := &Reconciler{
		logger:        slog.Default(),
		connector:     connector,
		meters:        &fakeMeterService{meters: []meter.Meter{m}},
		observability: observability,
	}

	require.NotPanics(t, func() {
		require.NoError(t, r.reconcile(t.Context()))
	})
}
