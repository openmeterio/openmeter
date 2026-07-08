package clickhouse

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func newTestInvalidator(t *testing.T, cache CacheConfig) (*meterCacheInvalidator, *MockClickHouse, *testTelemetry) {
	t.Helper()

	telemetry, telemetryConfig := newTestTelemetry()
	mockCH := NewMockClickHouse()

	observability, err := newMeterCacheObservability(Config{
		Meter:  telemetryConfig.Meter,
		Tracer: telemetryConfig.Tracer,
	})
	require.NoError(t, err)

	invalidator := newMeterCacheInvalidator(Config{
		Logger:     slog.Default(),
		ClickHouse: mockCH,
		Database:   "testdb",
		Cache:      cache,
	}, observability)

	return invalidator, mockCH, telemetry
}

// sumAttrValue finds the counter data point matching every expected attribute and returns
// its value, failing the test if no matching data point was recorded — the same "assert
// emission, not attachment" requirement as the query-path observability tests.
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

// TestInvalidateLateEvents_ClassifyFailureEmitsMarkerFailureAndSpanError forces
// lateEventWindows to fail (an invalid cache grain) and proves the classify-stage failure
// reaches the marker_failures counter with stage=classify, and the invalidate span records
// the error and sets its status to codes.Error — this is the G11 silent-staleness alert
// signal for the classification half of the pipeline.
func TestInvalidateLateEvents_ClassifyFailureEmitsMarkerFailureAndSpanError(t *testing.T) {
	invalidator, _, telemetry := newTestInvalidator(t, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrain("bogus-grain"),
	})

	invalidator.invalidateLateEvents(t.Context(), []streaming.RawEvent{
		{Namespace: "ns1", Type: "api-calls", Time: time.Now().UTC().Add(-3 * time.Hour)},
	})

	require.Equal(t, uint64(1), invalidator.markerInsertFailures.Load())

	m := collectMetric(t, telemetry.reader, "streaming.meter_cache.marker_failures")
	require.EqualValues(t, 1, sumAttrValue(t, m, map[string]string{"stage": "classify"}))

	spans := telemetry.recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "streaming.meter_cache.invalidate", spans[0].Name())
	require.NotEmpty(t, spans[0].Events(), "span must record the classification error")
	require.Equal(t, codes.Error, spans[0].Status().Code, "span status must be set on an error exit for status-based error dashboards")
}

// TestInvalidateLateEvents_NoLateEventsEmitsNoSpan proves the steady-state majority path —
// a batch with no late events — never starts the streaming.meter_cache.invalidate span.
// BatchInsert calls invalidateLateEvents on every insert while the cache is enabled, so
// starting a span unconditionally would emit one per insert; the span must exist only when
// there is something to report (a classification error or at least one marker window).
func TestInvalidateLateEvents_NoLateEventsEmitsNoSpan(t *testing.T) {
	invalidator, _, telemetry := newTestInvalidator(t, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	invalidator.invalidateLateEvents(t.Context(), []streaming.RawEvent{
		{Namespace: "ns1", Type: "api-calls", Time: time.Now().UTC()},
	})

	require.Equal(t, uint64(0), invalidator.markerInsertFailures.Load())
	require.Empty(t, telemetry.recorder.Ended(), "a batch with no late events must not start the invalidate span")
}

// TestInvalidateLateEvents_InsertFailureEmitsMarkerFailure forces the marker INSERT to fail
// and proves the insert-stage failure reaches marker_failures with stage=insert, distinct
// from the classify stage.
func TestInvalidateLateEvents_InsertFailureEmitsMarkerFailure(t *testing.T) {
	invalidator, mockCH, telemetry := newTestInvalidator(t, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	mockCH.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return len(sql) > 0
	}), mock.Anything).Return(errors.New("insert failed")).Once()

	mockCH.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(NewMockRows(), errors.New("listing failed")).Maybe()

	invalidator.invalidateLateEvents(t.Context(), []streaming.RawEvent{
		{Namespace: "ns1", Type: "api-calls", Time: time.Now().UTC().Add(-3 * time.Hour)},
	})

	require.Equal(t, uint64(1), invalidator.markerInsertFailures.Load())

	m := collectMetric(t, telemetry.reader, "streaming.meter_cache.marker_failures")
	require.EqualValues(t, 1, sumAttrValue(t, m, map[string]string{"stage": "insert"}))
}

// TestTriggerRefreshes_ListErrorEmitsRefreshTriggerOutcome proves a failed view listing
// (the prerequisite for any refresh trigger) is counted as outcome=list_error.
func TestTriggerRefreshes_ListErrorEmitsRefreshTriggerOutcome(t *testing.T) {
	invalidator, mockCH, telemetry := newTestInvalidator(t, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	mockCH.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(NewMockRows(), errors.New("listing failed")).Once()

	invalidator.triggerRefreshes(t.Context(), []invalidationWindow{
		{Namespace: "ns1", EventType: "api-calls", WindowLo: time.Now(), WindowHi: time.Now()},
	})

	m := collectMetric(t, telemetry.reader, "streaming.meter_cache.refresh_triggers")
	require.EqualValues(t, 1, sumAttrValue(t, m, map[string]string{"outcome": "list_error"}))
}

// TestNewMeterCacheInvalidator_NilObservabilityDependenciesDoNotPanic proves the
// invalidator's instrumented paths run without panicking when built from an observability
// instance constructed with nil Meter/Tracer.
func TestNewMeterCacheInvalidator_NilObservabilityDependenciesDoNotPanic(t *testing.T) {
	observability, err := newMeterCacheObservability(Config{})
	require.NoError(t, err)

	emptyRows := NewMockRows()
	emptyRows.On("Next").Return(false)
	emptyRows.On("Err").Return(nil)
	emptyRows.On("Close").Return(nil)

	mockCH := NewMockClickHouse()
	mockCH.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	mockCH.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil).Once()

	invalidator := newMeterCacheInvalidator(Config{
		Logger:     slog.Default(),
		ClickHouse: mockCH,
		Database:   "testdb",
		Cache: CacheConfig{
			Enabled:         true,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
			WindowSize:      CacheGrainHour,
		},
	}, observability)

	require.NotPanics(t, func() {
		invalidator.invalidateLateEvents(t.Context(), []streaming.RawEvent{
			{Namespace: "ns1", Type: "api-calls", Time: time.Now().UTC().Add(-3 * time.Hour)},
		})
	})
}
