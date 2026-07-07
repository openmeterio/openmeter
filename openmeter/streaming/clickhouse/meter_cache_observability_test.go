package clickhouse

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// testTelemetry bundles a ManualReader-backed MeterProvider and a SpanRecorder-backed
// TracerProvider so tests can assert both that an instrument fired and its attributes,
// and that the expected spans were recorded. See the R4 lesson referenced across the
// meter cache observability code: a detached instrument silently looks like a healthy
// quiet system, so tests must assert emission, not merely that construction succeeded.
type testTelemetry struct {
	reader   *sdkmetric.ManualReader
	recorder *tracetest.SpanRecorder
}

func newTestTelemetry() (*testTelemetry, Config) {
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	return &testTelemetry{reader: reader, recorder: recorder}, Config{
		Meter:  meterProvider.Meter("test"),
		Tracer: tracerProvider.Tracer("test"),
	}
}

// collectMetric returns the collected data points for one instrument name, failing the
// test if the instrument never recorded (the exact failure mode a detached instrument
// produces: Collect succeeds, but the metric is simply absent).
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

func newMeterCacheTestConfig(cache CacheConfig, telemetry Config) Config {
	return Config{
		Logger:           slog.Default(),
		ClickHouse:       NewMockClickHouse(),
		Database:         "testdb",
		EventsTableName:  "events",
		ProgressManager:  progressmanager.NewMockProgressManager(),
		SkipCreateTables: true,
		Cache:            cache,
		Meter:            telemetry.Meter,
		Tracer:           telemetry.Tracer,
	}
}

// TestQueryMeterCached_EmitsLiveRejectMetricsAndSpan proves a gate-rejected Cachable query
// records the queries counter with result=live_reject and the reject reason, and starts the
// streaming.meter_cache.query span — the health signal operators dashboard for "why isn't
// this meter using the cache".
func TestQueryMeterCached_EmitsLiveRejectMetricsAndSpan(t *testing.T) {
	telemetry, telemetryConfig := newTestTelemetry()

	cache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	connector, err := New(t.Context(), newMeterCacheTestConfig(cache, telemetryConfig))
	require.NoError(t, err)

	query := queryMeter{
		Database:        "testdb",
		EventsTableName: "events",
		Namespace:       "ns1",
		Meter: meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationLatest, // LATEST is always a static reject
			ValueProperty: lo.ToPtr("$.value"),
		},
		To: lo.ToPtr(time.Now()),
	}

	values, served := connector.queryMeterCached(t.Context(), query, streaming.QueryParams{Cachable: true, To: query.To})
	require.False(t, served)
	require.Nil(t, values)

	m := collectMetric(t, telemetry.reader, "streaming.meter_cache.queries")
	sum, ok := m.Data.(metricdata.Sum[int64])
	require.True(t, ok, "queries must be a counter")
	require.Len(t, sum.DataPoints, 1)

	dp := sum.DataPoints[0]
	result, ok := dp.Attributes.Value(attribute.Key("result"))
	require.True(t, ok)
	require.Equal(t, "live_reject", result.AsString())

	reason, ok := dp.Attributes.Value(attribute.Key("reject_reason"))
	require.True(t, ok)
	require.Equal(t, string(cacheRejectReasonLatestAggregation), reason.AsString())

	spans := telemetry.recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "streaming.meter_cache.query", spans[0].Name())
}

// TestQueryMeterCached_EmitsLiveFallbackMetricsAndSpanError proves an infrastructure
// failure in the gate's dynamic checks (the view-state lookup) — as opposed to a static
// reject — records the queries counter with result=live_fallback, and the
// streaming.meter_cache.query span records the error, sets its status to codes.Error, and
// carries both live_fallback=true and result=live_fallback (the same result attribute the
// cached and live_reject exits set, so span filtering can rely on it being present on every
// exit path). This is the cache-degradation signal: if it silently detached, an operator's
// dashboard would look like a healthy, quiet system while every opted-in query is actually
// falling back to the live path.
func TestQueryMeterCached_EmitsLiveFallbackMetricsAndSpanError(t *testing.T) {
	telemetry, telemetryConfig := newTestTelemetry()

	cache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	config := newMeterCacheTestConfig(cache, telemetryConfig)
	// The static reject checks require decimal precision to be enabled (see
	// meterCacheStaticReject); this test exercises the dynamic (ClickHouse-backed) checks,
	// so the query must clear every static check first.
	config.EnableDecimalPrecision = true

	connector, err := New(t.Context(), config)
	require.NoError(t, err)

	// Forces the gate past the static reject checks into the dynamic (ClickHouse-backed)
	// checks, then fails there — the infrastructure-failure branch of cacheEligibility,
	// distinct from a static reject reason.
	connector.cacheGate.fetchViewState = func(context.Context, string) (meterCacheViewState, error) {
		return meterCacheViewState{}, errors.New("view state lookup failed")
	}

	query := queryMeter{
		Database:        "testdb",
		EventsTableName: "events",
		Namespace:       "ns1",
		Meter: meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		},
		From:       lo.ToPtr(time.Now().Add(-time.Hour)),
		To:         lo.ToPtr(time.Now()),
		WindowSize: lo.ToPtr(meter.WindowSizeHour),
	}

	values, served := connector.queryMeterCached(t.Context(), query, streaming.QueryParams{
		Cachable:   true,
		From:       query.From,
		To:         query.To,
		WindowSize: query.WindowSize,
	})
	require.False(t, served)
	require.Nil(t, values)

	m := collectMetric(t, telemetry.reader, "streaming.meter_cache.queries")
	require.EqualValues(t, 1, sumAttrValue(t, m, map[string]string{"result": "live_fallback", "reject_reason": ""}))

	spans := telemetry.recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "streaming.meter_cache.query", spans[0].Name())
	require.NotEmpty(t, spans[0].Events(), "span must record the eligibility check error")
	require.Equal(t, codes.Error, spans[0].Status().Code, "span status must be set on an error exit for status-based error dashboards")

	var sawLiveFallback, sawResultAttr bool
	for _, attr := range spans[0].Attributes() {
		if attr.Key == "live_fallback" && attr.Value.AsBool() {
			sawLiveFallback = true
		}

		if attr.Key == "result" && attr.Value.AsString() == string(meterCacheQueryResultLiveFallback) {
			sawResultAttr = true
		}
	}
	require.True(t, sawLiveFallback, "span must carry live_fallback=true")
	require.True(t, sawResultAttr, "span must carry result=live_fallback so every exit path shares the same result attribute")
}

// TestNewMeterCacheObservability_NilDependenciesDoNotPanic proves the connector is
// constructible, and its instrumented paths runnable, with nil Config.Meter and
// Config.Tracer — the deployment shape where telemetry was never wired.
func TestNewMeterCacheObservability_NilDependenciesDoNotPanic(t *testing.T) {
	cache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	connector, err := New(t.Context(), Config{
		Logger:           slog.Default(),
		ClickHouse:       NewMockClickHouse(),
		Database:         "testdb",
		EventsTableName:  "events",
		ProgressManager:  progressmanager.NewMockProgressManager(),
		SkipCreateTables: true,
		Cache:            cache,
		// Meter and Tracer deliberately left nil.
	})
	require.NoError(t, err)
	require.NotNil(t, connector.observability)

	query := queryMeter{
		Database:        "testdb",
		EventsTableName: "events",
		Namespace:       "ns1",
		Meter: meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationLatest,
			ValueProperty: lo.ToPtr("$.value"),
		},
		To: lo.ToPtr(time.Now()),
	}

	require.NotPanics(t, func() {
		_, served := connector.queryMeterCached(t.Context(), query, streaming.QueryParams{Cachable: true, To: query.To})
		require.False(t, served)
	})
}
