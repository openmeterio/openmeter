package clickhouse

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type MeterCacheCHTestSuite struct {
	CHTestSuite
}

func TestMeterCacheClickHouse(t *testing.T) {
	suite.Run(t, new(MeterCacheCHTestSuite))
}

// TestGeneratedSQLRoundTrip proves every generated statement is accepted by a real
// ClickHouse and behaves per design: CREATE TABLE for both cache tables, one CREATE
// MATERIALIZED VIEW and one backfill INSERT per aggregation against a shared target, a
// SYSTEM REFRESH VIEW + SYSTEM WAIT VIEW round-trip, comment metadata surviving
// system.tables, and newest-wins reads returning live-equal values with unsettled events
// excluded.
func (s *MeterCacheCHTestSuite) TestGeneratedSQLRoundTrip() {
	t := s.T()
	ctx := t.Context()

	const (
		eventsTable = "om_events"
		namespace   = "cache-smoke"
		eventType   = "api-calls"
	)

	s.NoError(s.ClickHouse.Exec(ctx, createEventsTable{Database: s.Database, EventsTableName: eventsTable}.toSQL()))
	s.NoError(s.ClickHouse.Exec(ctx, createMeterCacheTable{Database: s.Database}.toSQL()))
	s.NoError(s.ClickHouse.Exec(ctx, createMeterCacheInvalidationsTable{Database: s.Database}.toSQL()))

	// given:
	// - two settled events (2 and 7) in one fully settled hour bucket, stored just now so
	//   the MV's dirty stored_at lookback picks their bucket up,
	// - one unsettled event (100) younger than minimumUsageAge that must never be cached.
	now := time.Now().UTC()
	bucket := now.Add(-3 * time.Hour).Truncate(time.Hour)

	newEvent := func(at time.Time, data string) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    "subject-1",
			Time:       at,
			Data:       data,
			IngestedAt: now,
			StoredAt:   now,
		}
	}

	insertSQL, insertArgs := InsertEventsQuery{
		Database:        s.Database,
		EventsTableName: eventsTable,
		Events: []streaming.RawEvent{
			newEvent(bucket.Add(5*time.Minute), `{"value": 2, "group1": "a"}`),
			newEvent(bucket.Add(10*time.Minute), `{"value": 7, "group1": "a"}`),
			newEvent(now.Add(-10*time.Minute), `{"value": 100, "group1": "a"}`),
		},
	}.ToSQL()
	s.NoError(s.ClickHouse.Exec(ctx, insertSQL, insertArgs...))

	aggregations := []meter.MeterAggregation{
		meter.MeterAggregationSum,
		meter.MeterAggregationCount,
		meter.MeterAggregationAvg,
		meter.MeterAggregationMin,
		meter.MeterAggregationMax,
		meter.MeterAggregationUniqueCount,
		meter.MeterAggregationLatest,
	}

	meters := make(map[meter.MeterAggregation]meter.Meter, len(aggregations))
	for _, aggregation := range aggregations {
		m := meter.Meter{
			Key:         fmt.Sprintf("meter-%s", aggregation),
			EventType:   eventType,
			Aggregation: aggregation,
			GroupBy:     map[string]string{"group1": "$.group1"},
		}
		if aggregation != meter.MeterAggregationCount {
			m.ValueProperty = lo.ToPtr("$.value")
		}
		meters[aggregation] = m
	}

	// when:
	// - the generated MV is created (all seven share om_meter_cache as APPEND target),
	// - the initial refresh triggered by CREATE is awaited so the explicit refresh below
	//   cannot race it,
	// - the generated backfill INSERT runs,
	// - an explicit SYSTEM REFRESH VIEW + SYSTEM WAIT VIEW round-trip completes. The
	//   backfill/refresh overlap is intentional: newest-wins must absorb it.
	for _, aggregation := range aggregations {
		mv := createMeterCacheMV{
			Database:        s.Database,
			EventsTableName: eventsTable,
			Namespace:       namespace,
			Meter:           meters[aggregation],
			Grain:           CacheGrainHour,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
		}

		createSQL, err := mv.toSQL()
		s.NoError(err)
		s.NoError(s.ClickHouse.Exec(ctx, createSQL), "create MV for %s", aggregation)

		qualifiedView := getTableName(s.Database, mv.name())
		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

		backfillSQL, err := meterCacheBackfill{
			Database:        s.Database,
			EventsTableName: eventsTable,
			Namespace:       namespace,
			Meter:           meters[aggregation],
			Grain:           CacheGrainHour,
			MinimumUsageAge: time.Hour,
		}.toSQL()
		s.NoError(err)
		s.NoError(s.ClickHouse.Exec(ctx, backfillSQL), "backfill for %s", aggregation)

		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))
	}

	// then:
	// - all seven views are registered and none recorded an exception
	var viewCount, exceptionCount uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		"SELECT count(), countIf(exception != '') FROM system.view_refreshes WHERE database = ?", s.Database,
	).Scan(&viewCount, &exceptionCount))
	s.Equal(uint64(len(aggregations)), viewCount)
	s.Equal(uint64(0), exceptionCount)

	// then:
	// - the COMMENT metadata written at CREATE survives the system.tables round-trip and
	//   is unstamped (backfilled_at is a lifecycle stamp added later via MODIFY COMMENT)
	for _, aggregation := range aggregations {
		m := meters[aggregation]

		var comment string
		s.NoError(s.ClickHouse.QueryRow(ctx,
			"SELECT comment FROM system.tables WHERE database = ? AND name = ?",
			s.Database, mvName(namespace, meterHash(m, CacheGrainHour)),
		).Scan(&comment))

		metadata, err := parseMeterCacheMVMetadata(comment)
		s.NoError(err)
		s.Equal(m.Key, metadata.MeterKey)
		s.Equal(eventType, metadata.EventType)
		s.Equal(formatCacheHash(meterHash(m, CacheGrainHour)), metadata.MeterHash)
		s.Nil(metadata.BackfilledAt)
	}

	// then:
	// - despite backfill and refresh both appending the bucket, newest-wins (FINAL on
	//   ReplacingMergeTree(created_at)) yields exactly one row per meter with the settled
	//   events aggregated and the unsettled event excluded
	cacheTable := getTableName(s.Database, meterCacheTableName)

	requireSingleBucketRow := func(aggregation meter.MeterAggregation) {
		var rowCount uint64
		var windowstart time.Time
		var groupBy []string
		s.NoError(s.ClickHouse.QueryRow(ctx,
			fmt.Sprintf("SELECT count(), any(windowstart), any(group_by) FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", cacheTable),
			namespace, meterHash(meters[aggregation], CacheGrainHour),
		).Scan(&rowCount, &windowstart, &groupBy))
		s.Equal(uint64(1), rowCount, "aggregation %s", aggregation)
		s.True(windowstart.UTC().Equal(bucket), "aggregation %s: windowstart %s != %s", aggregation, windowstart, bucket)
		s.Equal([]string{"a"}, groupBy)
	}

	scanDecimal := func(aggregation meter.MeterAggregation, column string) float64 {
		var value NullDecimal
		s.NoError(s.ClickHouse.QueryRow(ctx,
			fmt.Sprintf("SELECT %s FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", column, cacheTable),
			namespace, meterHash(meters[aggregation], CacheGrainHour),
		).Scan(&value))
		s.True(value.Valid, "aggregation %s: %s is NULL", aggregation, column)
		return value.Decimal.InexactFloat64()
	}

	for _, aggregation := range aggregations {
		requireSingleBucketRow(aggregation)
	}

	s.Equal(float64(9), scanDecimal(meter.MeterAggregationSum, "sum_value"))
	s.Equal(float64(2), scanDecimal(meter.MeterAggregationMin, "min_value"))
	s.Equal(float64(7), scanDecimal(meter.MeterAggregationMax, "max_value"))
	s.Equal(float64(9), scanDecimal(meter.MeterAggregationAvg, "sum_value"))

	var countValue uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT count_value FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", cacheTable),
		namespace, meterHash(meters[meter.MeterAggregationCount], CacheGrainHour),
	).Scan(&countValue))
	s.Equal(uint64(2), countValue)

	var valueCount uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT value_count FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", cacheTable),
		namespace, meterHash(meters[meter.MeterAggregationAvg], CacheGrainHour),
	).Scan(&valueCount))
	s.Equal(uint64(2), valueCount)

	var uniqCount uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT uniqExactMerge(uniq_state) FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", cacheTable),
		namespace, meterHash(meters[meter.MeterAggregationUniqueCount], CacheGrainHour),
	).Scan(&uniqCount))
	s.Equal(uint64(2), uniqCount)

	var latest NullDecimal
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT argMaxMerge(latest_state) FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", cacheTable),
		namespace, meterHash(meters[meter.MeterAggregationLatest], CacheGrainHour),
	).Scan(&latest))
	s.True(latest.Valid)
	s.Equal(float64(7), latest.Decimal.InexactFloat64())
}

// TestLateEventInvalidation drives events through Connector.BatchInsert against a real
// ClickHouse and proves the invalidation hook contract: an on-time batch leaves no trace,
// a late batch writes one server-timestamped marker per (namespace, event type) and
// triggers the affected MV's refresh, and further late batches within the throttle window
// still write markers but do not re-trigger.
func (s *MeterCacheCHTestSuite) TestLateEventInvalidation() {
	t := s.T()
	ctx := t.Context()

	const (
		eventsTable = "om_events"
		namespace   = "cache-invalidation"
		eventType   = "api-calls"
	)

	// given:
	// - a cache-enabled connector (New provisions the events table and both cache tables),
	// - one SUM meter with its cache MV deployed and its CREATE-time refresh awaited so
	//   later refresh accounting is unambiguous
	connector, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: eventsTable,
		ProgressManager: progressmanageradapter.NewMockProgressManager(),
		Cache: CacheConfig{
			Enabled:         true,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
			WindowSize:      CacheGrainHour,
		},
	})
	s.NoError(err)

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: eventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}

	createSQL, err := mv.toSQL()
	s.NoError(err)
	s.NoError(s.ClickHouse.Exec(ctx, createSQL))

	qualifiedView := getTableName(s.Database, mv.name())
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	now := time.Now().UTC()

	newEvent := func(at time.Time, data string) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    "subject-1",
			Time:       at,
			Data:       data,
			IngestedAt: now,
			StoredAt:   now,
		}
	}

	invalidationsTable := getTableName(s.Database, meterCacheInvalidationsTableName)

	// when:
	// - a batch containing only an on-time event is inserted
	s.NoError(connector.BatchInsert(ctx, []streaming.RawEvent{
		newEvent(now.Add(-10*time.Minute), `{"value": 100}`),
	}))

	// then:
	// - no marker is written and no refresh is triggered
	var markerCount uint64
	s.NoError(s.ClickHouse.QueryRow(ctx, "SELECT count() FROM "+invalidationsTable).Scan(&markerCount))
	s.Equal(uint64(0), markerCount)
	s.Equal(uint64(0), connector.cacheInvalidator.refreshTriggersFired.Load())

	// when:
	// - a batch with two late events in two different settled buckets is inserted
	bucketA := now.Add(-4 * time.Hour).Truncate(time.Hour)
	bucketB := now.Add(-3 * time.Hour).Truncate(time.Hour)

	s.NoError(connector.BatchInsert(ctx, []streaming.RawEvent{
		newEvent(bucketA.Add(5*time.Minute), `{"value": 2}`),
		newEvent(bucketB.Add(10*time.Minute), `{"value": 7}`),
	}))

	// then:
	// - exactly one marker spans both buckets, and its created_at was stamped by the
	//   ClickHouse clock (the structural guarantee that the INSERT carries no client
	//   timestamp is TestInsertInvalidationMarkersToSQL; here we prove the DEFAULT
	//   actually produced a sane server-side value)
	var (
		markerNamespace, markerEventType string
		windowLo, windowHi               time.Time
		createdAt, serverNow             time.Time
	)
	s.NoError(s.ClickHouse.QueryRow(ctx,
		"SELECT namespace, event_type, window_lo, window_hi, created_at, now64(3) FROM "+invalidationsTable,
	).Scan(&markerNamespace, &markerEventType, &windowLo, &windowHi, &createdAt, &serverNow))
	s.Equal(namespace, markerNamespace)
	s.Equal(eventType, markerEventType)
	s.True(windowLo.UTC().Equal(bucketA), "window_lo %s != %s", windowLo, bucketA)
	s.True(windowHi.UTC().Equal(bucketB.Add(time.Hour)), "window_hi %s != %s", windowHi, bucketB.Add(time.Hour))
	s.Less(serverNow.Sub(createdAt).Abs(), time.Minute)

	// then:
	// - exactly one best-effort refresh was triggered, with no failure counted
	s.Equal(uint64(1), connector.cacheInvalidator.refreshTriggersFired.Load())
	s.Equal(uint64(0), connector.cacheInvalidator.markerInsertFailures.Load())
	s.Equal(uint64(0), connector.cacheInvalidator.refreshTriggerFailures.Load())

	// then:
	// - the triggered refresh converges the late buckets into the cache (both are settled
	//   and carry a fresh stored_at, so the dirty filter picks them up)
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	var cachedRows uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT count() FROM %s FINAL WHERE namespace = ? AND meter_hash = ?", getTableName(s.Database, meterCacheTableName)),
		namespace, meterHash(m, CacheGrainHour),
	).Scan(&cachedRows))
	s.Equal(uint64(2), cachedRows)

	// when:
	// - another late batch arrives within the throttle window
	s.NoError(connector.BatchInsert(ctx, []streaming.RawEvent{
		newEvent(bucketB.Add(20*time.Minute), `{"value": 1}`),
	}))

	// then:
	// - markers are never throttled (correctness), refresh triggers are (best-effort)
	s.NoError(s.ClickHouse.QueryRow(ctx, "SELECT count() FROM "+invalidationsTable).Scan(&markerCount))
	s.Equal(uint64(2), markerCount)
	s.Equal(uint64(1), connector.cacheInvalidator.refreshTriggersFired.Load())
}

// deployMeterCacheMV provisions one meter's cache the way the WP6 reconciler will:
// CREATE MATERIALIZED VIEW, wait for its initial refresh, run the backfill INSERT,
// force one more refresh so view_refreshes reflects a post-backfill success, and only
// then stamp backfilled_at (G3) via MODIFY COMMENT.
func (s *MeterCacheCHTestSuite) deployMeterCacheMV(ctx context.Context, mv createMeterCacheMV) {
	createSQL, err := mv.toSQL()
	s.NoError(err)
	s.NoError(s.ClickHouse.Exec(ctx, createSQL))

	qualifiedView := getTableName(s.Database, mv.name())
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	backfillSQL, err := meterCacheBackfill{
		Database:        mv.Database,
		EventsTableName: mv.EventsTableName,
		Namespace:       mv.Namespace,
		Meter:           mv.Meter,
		Grain:           mv.Grain,
		MinimumUsageAge: mv.MinimumUsageAge,
	}.toSQL()
	s.NoError(err)
	s.NoError(s.ClickHouse.Exec(ctx, backfillSQL))

	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	metadata, err := mv.metadata()
	s.NoError(err)
	metadata.BackfilledAt = lo.ToPtr(time.Now().UTC().Truncate(time.Second))

	comment, err := metadata.marshal()
	s.NoError(err)
	s.NoError(s.ClickHouse.Exec(ctx, fmt.Sprintf("ALTER TABLE %s MODIFY COMMENT %s", qualifiedView, sqlStringLiteral(comment))))
}

// TestCachedReadParity is the WP5 parity spot-check: for SUM, UNIQUE_COUNT and AVG, a
// cache-served Connector.QueryMeter must return exactly the rows the live path returns —
// windowed and total, on-grid and off-grid, UTC and a whole-hour-offset timezone — and a
// poison write proves the cached rows really came from the cache leg, not from a silent
// live fallback (the full aggregation × window × filter matrix lands with WP7).
//
// Watched RED with the guard reverted: short-circuiting queryMeterCached to
// `return nil, false` (forcing the silent-fallback failure mode this test exists to
// catch) fails the first cached subtest on the "serving live" log assertion.
func (s *MeterCacheCHTestSuite) TestCachedReadParity() {
	t := s.T()
	ctx := t.Context()

	const (
		eventsTable = "om_events"
		namespace   = "cache-parity"
		eventType   = "api-calls"
	)

	// The connector logs "serving live" on every cache fallback path; capturing the logs
	// lets each cached query assert it was actually served from the cache, so a silently
	// falling-back gate cannot make the parity checks pass trivially.
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connector, err := New(ctx, Config{
		Logger:                 logger,
		ClickHouse:             s.ClickHouse,
		Database:               s.Database,
		EventsTableName:        eventsTable,
		EnableDecimalPrecision: true,
		ProgressManager:        progressmanageradapter.NewMockProgressManager(),
		Cache: CacheConfig{
			Enabled:         true,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
			WindowSize:      CacheGrainHour,
		},
	})
	s.NoError(err)

	newYork, err := time.LoadLocation("America/New_York")
	s.NoError(err)

	// given:
	// - settled events in two fully settled hour buckets (bucketOld, bucketMid) covering
	//   two subjects, two dimension values, a JSON-null value, a missing value property,
	//   and the value 7 repeated across buckets and later in the live tail (the
	//   UNIQUE_COUNT cross-leg dedupe case),
	// - one event between the cache horizon and the freshness cutoff (post-leg only),
	// - two events in the always-live tail.
	now := time.Now().UTC()
	bucketOld := now.Add(-5 * time.Hour).Truncate(time.Hour)
	bucketMid := now.Add(-4 * time.Hour).Truncate(time.Hour)

	newEvent := func(subject string, at time.Time, data string) streaming.RawEvent {
		return streaming.RawEvent{
			Namespace:  namespace,
			ID:         ulid.Make().String(),
			Type:       eventType,
			Source:     "test-source",
			Subject:    subject,
			Time:       at,
			Data:       data,
			IngestedAt: now,
			StoredAt:   now,
		}
	}

	// Inserted directly (not through BatchInsert) so no invalidation markers are written:
	// this test exercises the clean read path, not marker healing.
	insertSQL, insertArgs := InsertEventsQuery{
		Database:        s.Database,
		EventsTableName: eventsTable,
		Events: []streaming.RawEvent{
			newEvent("subject-1", bucketOld.Add(5*time.Minute), `{"value": 2, "group1": "a"}`),
			newEvent("subject-1", bucketOld.Add(10*time.Minute), `{"value": 7, "group1": "a"}`),
			newEvent("subject-2", bucketOld.Add(20*time.Minute), `{"value": 5, "group1": "b"}`),
			newEvent("subject-1", bucketOld.Add(25*time.Minute), `{"value": null, "group1": "b"}`),
			newEvent("subject-2", bucketOld.Add(30*time.Minute), `{"group1": "a"}`),
			newEvent("subject-1", bucketMid.Add(5*time.Minute), `{"value": 7, "group1": "a"}`),
			newEvent("subject-2", bucketMid.Add(15*time.Minute), `{"value": 3.5, "group1": "b"}`),
			newEvent("subject-1", bucketMid.Add(40*time.Minute), `{"value": 1, "group1": "a"}`),
			newEvent("subject-1", now.Add(-90*time.Minute), `{"value": 11, "group1": "a"}`),
			newEvent("subject-1", now.Add(-30*time.Minute), `{"value": 13, "group1": "a"}`),
			newEvent("subject-2", now.Add(-10*time.Minute), `{"value": 7, "group1": "b"}`),
		},
	}.ToSQL()
	s.NoError(s.ClickHouse.Exec(ctx, insertSQL, insertArgs...))

	aggregations := []meter.MeterAggregation{
		meter.MeterAggregationSum,
		meter.MeterAggregationUniqueCount,
		meter.MeterAggregationAvg,
	}

	meters := make(map[meter.MeterAggregation]meter.Meter, len(aggregations))
	for _, aggregation := range aggregations {
		meters[aggregation] = meter.Meter{
			Key:           fmt.Sprintf("meter-%s", aggregation),
			EventType:     eventType,
			Aggregation:   aggregation,
			ValueProperty: lo.ToPtr("$.value"),
			GroupBy:       map[string]string{"group1": "$.group1"},
		}

		s.deployMeterCacheMV(ctx, createMeterCacheMV{
			Database:        s.Database,
			EventsTableName: eventsTable,
			Namespace:       namespace,
			Meter:           meters[aggregation],
			Grain:           CacheGrainHour,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
		})
	}

	windowSizeHour := meter.WindowSizeHour
	windowSizeDay := meter.WindowSizeDay

	fromOnGrid := bucketOld
	toOnGrid := now.Truncate(time.Hour).Add(time.Hour)
	fromOffGrid := bucketOld.Add(7 * time.Minute)
	toOffGrid := now.Add(-3 * time.Minute)

	queryCases := []struct {
		name   string
		params streaming.QueryParams
	}{
		{
			name: "windowed hour utc on grid",
			params: streaming.QueryParams{
				From:       &fromOnGrid,
				To:         &toOnGrid,
				WindowSize: &windowSizeHour,
				GroupBy:    []string{"subject", "group1"},
			},
		},
		{
			name: "windowed hour utc off grid",
			params: streaming.QueryParams{
				From:       &fromOffGrid,
				To:         &toOffGrid,
				WindowSize: &windowSizeHour,
				GroupBy:    []string{"subject", "group1"},
			},
		},
		{
			name: "windowed day new york off grid",
			params: streaming.QueryParams{
				From:           &fromOnGrid,
				To:             &toOffGrid,
				WindowSize:     &windowSizeDay,
				WindowTimeZone: newYork,
				GroupBy:        []string{"subject"},
			},
		},
		{
			name: "total off grid",
			params: streaming.QueryParams{
				From: &fromOffGrid,
				To:   &toOffGrid,
			},
		},
		{
			name: "total on grid with subject filter",
			params: streaming.QueryParams{
				From:          &fromOnGrid,
				To:            &toOnGrid,
				FilterSubject: []string{"subject-1"},
			},
		},
	}

	queryBothPaths := func(m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, []meter.MeterQueryRow) {
		params.Cachable = false
		live, err := connector.QueryMeter(ctx, namespace, m, params)
		s.NoError(err)

		logBuffer.Reset()

		params.Cachable = true
		cached, err := connector.QueryMeter(ctx, namespace, m, params)
		s.NoError(err)
		s.NotContains(logBuffer.String(), "serving live", "cached query fell back to the live path")

		return live, cached
	}

	// when/then:
	// - each aggregation × query shape returns byte-equal rows on both paths, with the
	//   cached run proven not to have fallen back
	for _, aggregation := range aggregations {
		for _, queryCase := range queryCases {
			s.Run(fmt.Sprintf("%s %s", aggregation, queryCase.name), func() {
				live, cached := queryBothPaths(meters[aggregation], queryCase.params)

				s.NotEmpty(live, "parity would be trivial on an empty result")
				s.ElementsMatch(live, cached)
			})
		}
	}

	// given:
	// - a late event written into a settled, cached bucket bypassing BatchInsert (no
	//   invalidation marker) and with no refresh afterwards
	poisonSQL, poisonArgs := InsertEventsQuery{
		Database:        s.Database,
		EventsTableName: eventsTable,
		Events: []streaming.RawEvent{
			newEvent("subject-1", bucketOld.Add(40*time.Minute), `{"value": 1000, "group1": "a"}`),
		},
	}.ToSQL()
	s.NoError(s.ClickHouse.Exec(ctx, poisonSQL, poisonArgs...))

	// then:
	// - the live path sees the poison, the cached path serves the stale cached bucket.
	//   This is the proof the cache leg produced the parity rows above: had the cached
	//   path silently served live, both paths would now agree and the inequality below
	//   would go red.
	sumParams := streaming.QueryParams{
		From:       &fromOnGrid,
		To:         &toOnGrid,
		WindowSize: &windowSizeHour,
		GroupBy:    []string{"subject", "group1"},
	}

	live, cached := queryBothPaths(meters[meter.MeterAggregationSum], sumParams)

	findPoisonedBucketValue := func(rows []meter.MeterQueryRow) float64 {
		for _, row := range rows {
			if row.WindowStart.Equal(bucketOld) && lo.FromPtr(row.Subject) == "subject-1" && lo.FromPtr(row.GroupBy["group1"]) == "a" {
				return row.Value
			}
		}

		s.Failf("poisoned bucket row not found", "rows: %v", rows)

		return 0
	}

	s.Equal(float64(1009), findPoisonedBucketValue(live))
	s.Equal(float64(9), findPoisonedBucketValue(cached))
}
