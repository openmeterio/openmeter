package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// MeterCacheObservabilityCHTestSuite embeds CHTestSuite directly rather than
// MeterCacheCHTestSuite: stretchr/testify suite.Run executes every exported Test* method
// reachable on the suite value, including ones promoted from an embedded suite type, so
// embedding MeterCacheCHTestSuite here would re-run its entire test file set under this
// suite's name too.
type MeterCacheObservabilityCHTestSuite struct {
	CHTestSuite
}

func TestMeterCacheObservabilityClickHouse(t *testing.T) {
	suite.Run(t, new(MeterCacheObservabilityCHTestSuite))
}

// TestCachedQueryEmitsQueriesCounterAndDurationHistogram is the real-ClickHouse companion
// to the mock-based unit tests: it drives one meter query all the way through a genuinely
// deployed cache MV and proves the streaming.meter_cache.queries counter records
// result=cached and the query_duration_ms histogram observes the cached arm — the mock-based
// unit tests in meter_cache_observability_test.go only exercise the gate-reject path, which
// never reaches the ClickHouse round trip this test covers.
func (s *MeterCacheObservabilityCHTestSuite) TestCachedQueryEmitsQueriesCounterAndDurationHistogram() {
	t := s.T()
	ctx := t.Context()

	const (
		eventsTable = "om_events"
		namespace   = "cache-observability"
		eventType   = "api-calls"
	)

	telemetry, telemetryConfig := newTestTelemetry()

	cache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}

	connector, err := New(ctx, Config{
		Logger:                 slog.Default(),
		ClickHouse:             s.ClickHouse,
		Database:               s.Database,
		EventsTableName:        eventsTable,
		EnableDecimalPrecision: true,
		ProgressManager:        progressmanageradapter.NewMockProgressManager(),
		Cache:                  cache,
		Meter:                  telemetryConfig.Meter,
		Tracer:                 telemetryConfig.Tracer,
	})
	s.Require().NoError(err)

	// Tests query immediately after an explicit refresh; the production view-state memo
	// (G13) would serve a pre-refresh snapshot for up to its TTL, making this assertion
	// time-dependent.
	connector.cacheGate.viewStateTTL = 0

	now := time.Now().UTC()
	bucket := now.Add(-3 * time.Hour).Truncate(time.Hour)

	newEvent := func(at time.Time, value int) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    "subject-1",
			Time:       at,
			Data:       fmt.Sprintf(`{"value": %d}`, value),
			IngestedAt: now,
			StoredAt:   now,
		}
	}

	insertSQL, insertArgs := InsertEventsQuery{
		Database:        s.Database,
		EventsTableName: eventsTable,
		Events: []streaming.RawEvent{
			newEvent(bucket.Add(5*time.Minute), 2),
			newEvent(bucket.Add(10*time.Minute), 7),
		},
	}.ToSQL()
	s.Require().NoError(s.ClickHouse.Exec(ctx, insertSQL, insertArgs...))

	m := meter.Meter{
		Key:           "observability-meter",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	s.deployMV(ctx, createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: eventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           cache.WindowSize,
		RefreshInterval: cache.RefreshInterval,
		MinimumUsageAge: cache.MinimumUsageAge,
	})

	values, err := connector.QueryMeter(ctx, namespace, m, streaming.QueryParams{
		Cachable:   true,
		From:       lo.ToPtr(bucket),
		To:         lo.ToPtr(now),
		WindowSize: lo.ToPtr(meter.WindowSizeHour),
	})
	s.Require().NoError(err)
	s.Require().Len(values, 1)
	s.Require().Equal(float64(9), values[0].Value)

	metricData := collectMetric(t, telemetry.reader, "streaming.meter_cache.queries")
	s.Require().EqualValues(1, sumAttrValue(t, metricData, map[string]string{"result": "cached", "reject_reason": ""}))

	durationMetric := collectMetric(t, telemetry.reader, "streaming.meter_cache.query_duration_ms")
	hist, ok := durationMetric.Data.(metricdata.Histogram[int64])
	s.Require().True(ok, "query_duration_ms must be a histogram")
	s.Require().Len(hist.DataPoints, 1)

	arm, ok := hist.DataPoints[0].Attributes.Value(attribute.Key("arm"))
	s.Require().True(ok)
	s.Require().Equal("cached", arm.AsString())

	spans := telemetry.recorder.Ended()
	s.Require().Len(spans, 1)
	s.Require().Equal("streaming.meter_cache.query", spans[0].Name())
}

// deployMV deploys one cache MV fully: create, wait for the initial refresh, backfill,
// explicit refresh, wait, then stamp backfilled_at. This mirrors
// MeterCacheCHTestSuite.deployMeterCacheMV; it is not shared because the two suites are
// independent stretchr/testify suite types with no common base beyond CHTestSuite.
func (s *MeterCacheObservabilityCHTestSuite) deployMV(ctx context.Context, mv createMeterCacheMV) {
	createSQL, err := mv.toSQL()
	s.Require().NoError(err)
	s.Require().NoError(s.ClickHouse.Exec(ctx, createSQL))

	qualifiedView := getTableName(s.Database, mv.name())
	s.Require().NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	backfillSQL, err := meterCacheBackfill{
		Database:        mv.Database,
		EventsTableName: mv.EventsTableName,
		Namespace:       mv.Namespace,
		Meter:           mv.Meter,
		Grain:           mv.Grain,
		MinimumUsageAge: mv.MinimumUsageAge,
	}.toSQL()
	s.Require().NoError(err)
	s.Require().NoError(s.ClickHouse.Exec(ctx, backfillSQL))

	s.Require().NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
	s.Require().NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	metadata, err := mv.metadata()
	s.Require().NoError(err)
	metadata.BackfilledAt = lo.ToPtr(time.Now().UTC().Truncate(time.Second))

	comment, err := metadata.marshal()
	s.Require().NoError(err)
	s.Require().NoError(s.ClickHouse.Exec(ctx, fmt.Sprintf("ALTER TABLE %s MODIFY COMMENT %s", qualifiedView, sqlStringLiteral(comment))))
}
