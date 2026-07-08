package clickhouse

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// e2eEventsTable is the events table name shared by the WP7 end-to-end cache tests; each
// suite test runs in its own temp database, so the constant name never collides.
const e2eEventsTable = "om_events"

// cacheTestConnector bundles a cache-enabled connector with its captured logs so tests can
// assert the gate's serve/fallback decision: the connector logs "serving live" on every
// cache fallback, so its absence after a cached query proves the cache leg produced the
// rows (a silently falling-back gate would make any parity assertion pass trivially).
type cacheTestConnector struct {
	*Connector

	logs *bytes.Buffer
}

func (s *MeterCacheCHTestSuite) newCacheConnector(ctx context.Context, cache CacheConfig) cacheTestConnector {
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connector, err := New(ctx, Config{
		Logger:                 logger,
		ClickHouse:             s.ClickHouse,
		Database:               s.Database,
		EventsTableName:        e2eEventsTable,
		EnableDecimalPrecision: true,
		ProgressManager:        progressmanageradapter.NewMockProgressManager(),
		Cache:                  cache,
	})
	s.NoError(err)

	// Tests drive refreshes explicitly (SYSTEM REFRESH VIEW + SYSTEM WAIT VIEW) and then
	// query immediately; the production view-state memo (G13) would serve a pre-refresh
	// snapshot for up to its TTL, making assertions time-dependent.
	if connector.cacheGate != nil {
		connector.cacheGate.viewStateTTL = 0
	}

	return cacheTestConnector{Connector: connector, logs: logs}
}

func (s *MeterCacheCHTestSuite) insertRawEvents(ctx context.Context, events ...streaming.RawEvent) {
	insertSQL, args := InsertEventsQuery{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Events:          events,
	}.ToSQL()
	s.NoError(s.ClickHouse.Exec(ctx, insertSQL, args...))
}

func rawCacheTestEvent(namespace, eventType, subject string, at time.Time, data string, storedAt time.Time) streaming.RawEvent {
	return streaming.RawEvent{
		Namespace:  namespace,
		ID:         ulid.Make().String(),
		Type:       eventType,
		Source:     "test-source",
		Subject:    subject,
		Time:       at,
		Data:       data,
		IngestedAt: storedAt,
		StoredAt:   storedAt,
	}
}

// queryBothPaths runs the same query on the live path (Cachable=false) and the cached
// path (Cachable=true), asserting the cached run did not silently fall back to live.
func (s *MeterCacheCHTestSuite) queryBothPaths(ctx context.Context, c cacheTestConnector, namespace string, m meter.Meter, params streaming.QueryParams) (live, cached []meter.MeterQueryRow) {
	params.Cachable = false
	live, err := c.QueryMeter(ctx, namespace, m, params)
	s.NoError(err)

	c.logs.Reset()

	params.Cachable = true
	cached, err = c.QueryMeter(ctx, namespace, m, params)
	s.NoError(err)
	s.NotContains(c.logs.String(), "serving live", "cached query fell back to the live path")

	return live, cached
}

// totalValue extracts the single row value of a windowless total query.
func (s *MeterCacheCHTestSuite) totalValue(rows []meter.MeterQueryRow) float64 {
	s.Len(rows, 1)

	return rows[0].Value
}

// refreshViewUntilMarkersHealed forces refreshes until the view's derived refresh start
// (second-resolution last_success_time minus last_success_duration_ms) lands strictly
// after the newest invalidation marker even when truncated to whole seconds — the
// clickhouse-go driver binds time.Time query arguments at second precision, so the
// gate's heal comparison is effectively second-granular and a refresh racing a marker
// within the same second looks unhealed. That is the conservative direction (production
// merely stays live until the next scheduled refresh), but a test asserting the healed
// state must drive refreshes until the ordering is measurable through the truncation.
func (s *MeterCacheCHTestSuite) refreshViewUntilMarkersHealed(ctx context.Context, viewName string) {
	qualifiedView := getTableName(s.Database, viewName)
	deadline := time.Now().Add(30 * time.Second)

	for {
		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

		var (
			lastSuccess time.Time
			durationMS  uint64
		)
		s.NoError(s.ClickHouse.QueryRow(ctx,
			"SELECT last_success_time, last_success_duration_ms FROM system.view_refreshes WHERE database = ? AND view = ?",
			s.Database, viewName,
		).Scan(&lastSuccess, &durationMS))

		var newestMarker time.Time
		s.NoError(s.ClickHouse.QueryRow(ctx,
			"SELECT max(created_at) FROM "+getTableName(s.Database, meterCacheInvalidationsTableName),
		).Scan(&newestMarker))

		refreshStart := lastSuccess.Add(-time.Duration(durationMS) * time.Millisecond)
		if refreshStart.Truncate(time.Second).After(newestMarker) {
			return
		}

		s.Less(time.Now(), deadline, "no refresh started measurably after the newest marker")
		time.Sleep(200 * time.Millisecond)
	}
}

// allCacheTestAggregations is every meter aggregation the cache supports; the parity
// matrix runs each query shape over all of them. LATEST is excluded entirely (see
// meterCacheStaticReject): it only ever needs the single newest value in the queried
// window, so there is no re-aggregation of settled history for the cache to save, and it
// always takes the live path — there is no cached leg for it to have parity with.
var allCacheTestAggregations = []meter.MeterAggregation{
	meter.MeterAggregationSum,
	meter.MeterAggregationCount,
	meter.MeterAggregationAvg,
	meter.MeterAggregationMin,
	meter.MeterAggregationMax,
	meter.MeterAggregationUniqueCount,
}

// deployCacheTestMeters creates one meter per aggregation (two JSON dimensions, value
// property except COUNT) and deploys each meter's cache MV fully (create, backfill,
// refresh, stamp).
func (s *MeterCacheCHTestSuite) deployCacheTestMeters(ctx context.Context, namespace, eventType, keyPrefix string, cache CacheConfig) map[meter.MeterAggregation]meter.Meter {
	meters := make(map[meter.MeterAggregation]meter.Meter, len(allCacheTestAggregations))

	for _, aggregation := range allCacheTestAggregations {
		m := meter.Meter{
			Key:         fmt.Sprintf("%s-%s", keyPrefix, aggregation),
			EventType:   eventType,
			Aggregation: aggregation,
			GroupBy: map[string]string{
				"group1": "$.group1",
				"group2": "$.group2",
			},
		}
		if aggregation != meter.MeterAggregationCount {
			m.ValueProperty = lo.ToPtr("$.value")
		}

		meters[aggregation] = m

		s.deployMeterCacheMV(ctx, createMeterCacheMV{
			Database:        s.Database,
			EventsTableName: e2eEventsTable,
			Namespace:       namespace,
			Meter:           m,
			Grain:           cache.WindowSize,
			RefreshInterval: cache.RefreshInterval,
			MinimumUsageAge: cache.MinimumUsageAge,
		})
	}

	return meters
}

// TestCachedReadParityMatrix is the WP7 parity matrix: for every supported aggregation,
// cached and live results must be row-for-row equal across the query-shape axes of the
// test plan — window sizes (total, =grain hour, day, month), grid alignment (on-grid,
// mid-hour from, mid-hour to, both off-grid), group-by shapes (none, subject, two JSON
// dimensions, subset), filters (FilterSubject, FilterGroupBy $eq/$ne/$in), timezones (UTC
// and America/New_York), and data shapes (all-null buckets, empty buckets, missing value
// property, duplicate values across the cache/live boundary). LATEST is not part of this
// matrix: it is excluded from the cache entirely and always reads live (see
// allCacheTestAggregations).
//
// Every cached run asserts the absence of the "serving live" fallback log, so parity can
// never pass because the gate silently served the live path twice.
func (s *MeterCacheCHTestSuite) TestCachedReadParityMatrix() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-parity-matrix"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	newYork, err := time.LoadLocation("America/New_York")
	s.NoError(err)

	// given:
	// - three settled hour buckets (A, B, D) with an intentionally empty bucket between B
	//   and D, covering two subjects and two JSON dimensions,
	// - a JSON-null value and a missing value property in bucket A,
	// - the series (subject-2, b, y) having only a NULL value in bucket D: sum/min/max/avg
	//   drop that row, UNIQUE_COUNT emits it with 0,
	// - the value 7 repeated in cached buckets and again in the live tail (UNIQUE_COUNT
	//   cross-leg dedupe: states must merge, never sum),
	// - one event between the cache horizon and now-1h (post leg) and two tail events.
	now := time.Now().UTC()
	bucketA := now.Add(-6 * time.Hour).Truncate(time.Hour)
	bucketB := now.Add(-5 * time.Hour).Truncate(time.Hour)
	bucketD := now.Add(-3 * time.Hour).Truncate(time.Hour)

	event := func(subject string, at time.Time, data string) streaming.RawEvent {
		return rawCacheTestEvent(namespace, eventType, subject, at, data, now)
	}

	// Inserted directly (not through BatchInsert) so no invalidation markers are written:
	// the matrix exercises the clean read path, not marker healing.
	s.insertRawEvents(ctx,
		event("subject-1", bucketA.Add(5*time.Minute), `{"value": 2, "group1": "a", "group2": "x"}`),
		event("subject-1", bucketA.Add(10*time.Minute), `{"value": 7, "group1": "a", "group2": "y"}`),
		event("subject-2", bucketA.Add(20*time.Minute), `{"value": 5, "group1": "b", "group2": "x"}`),
		event("subject-1", bucketA.Add(25*time.Minute), `{"value": null, "group1": "b", "group2": "x"}`),
		event("subject-2", bucketA.Add(30*time.Minute), `{"group1": "a", "group2": "y"}`),
		event("subject-1", bucketB.Add(5*time.Minute), `{"value": 7, "group1": "a", "group2": "x"}`),
		event("subject-2", bucketB.Add(15*time.Minute), `{"value": 3.5, "group1": "b", "group2": "y"}`),
		event("subject-1", bucketB.Add(40*time.Minute), `{"value": 1, "group1": "a", "group2": "x"}`),
		event("subject-2", bucketD.Add(10*time.Minute), `{"value": null, "group1": "b", "group2": "y"}`),
		event("subject-1", bucketD.Add(35*time.Minute), `{"value": 11, "group1": "a", "group2": "y"}`),
		event("subject-1", now.Add(-90*time.Minute), `{"value": 13, "group1": "b", "group2": "x"}`),
		event("subject-2", now.Add(-30*time.Minute), `{"value": 7, "group1": "a", "group2": "x"}`),
		event("subject-1", now.Add(-10*time.Minute), `{"value": 17, "group1": "b", "group2": "y"}`),
	)

	meters := s.deployCacheTestMeters(ctx, namespace, eventType, "meter", c.config.Cache)

	fromOn := bucketA
	toOn := now.Truncate(time.Hour).Add(time.Hour)
	fromMid := bucketA.Add(25 * time.Minute)
	toMid := now.Add(-7 * time.Minute)

	queryCases := []struct {
		name   string
		params streaming.QueryParams
	}{
		{
			name:   "total on grid no group by",
			params: streaming.QueryParams{From: &fromOn, To: &toOn},
		},
		{
			name: "total both off grid subject group by",
			params: streaming.QueryParams{
				From:    &fromMid,
				To:      &toMid,
				GroupBy: []string{"subject"},
			},
		},
		{
			name: "hour on grid two dims and subject",
			params: streaming.QueryParams{
				From:       &fromOn,
				To:         &toOn,
				WindowSize: lo.ToPtr(meter.WindowSizeHour),
				GroupBy:    []string{"subject", "group1", "group2"},
			},
		},
		{
			name: "hour mid from subset group by",
			params: streaming.QueryParams{
				From:       &fromMid,
				To:         &toOn,
				WindowSize: lo.ToPtr(meter.WindowSizeHour),
				GroupBy:    []string{"subject", "group1"},
			},
		},
		{
			name: "hour mid to no group by",
			params: streaming.QueryParams{
				From:       &fromOn,
				To:         &toMid,
				WindowSize: lo.ToPtr(meter.WindowSizeHour),
			},
		},
		{
			name: "hour subject filter",
			params: streaming.QueryParams{
				From:          &fromOn,
				To:            &toOn,
				WindowSize:    lo.ToPtr(meter.WindowSizeHour),
				FilterSubject: []string{"subject-1"},
			},
		},
		{
			name: "hour group by filter eq",
			params: streaming.QueryParams{
				From:       &fromOn,
				To:         &toOn,
				WindowSize: lo.ToPtr(meter.WindowSizeHour),
				GroupBy:    []string{"group1"},
				FilterGroupBy: map[string]filter.FilterString{
					"group1": {Eq: lo.ToPtr("a")},
				},
			},
		},
		{
			name: "total group by filters in and ne",
			params: streaming.QueryParams{
				From:    &fromOn,
				To:      &toOn,
				GroupBy: []string{"group1", "group2"},
				FilterGroupBy: map[string]filter.FilterString{
					"group1": {In: lo.ToPtr([]string{"a", "b"})},
					"group2": {Ne: lo.ToPtr("y")},
				},
			},
		},
		{
			name: "day utc subject group by",
			params: streaming.QueryParams{
				From:       &fromOn,
				To:         &toOn,
				WindowSize: lo.ToPtr(meter.WindowSizeDay),
				GroupBy:    []string{"subject"},
			},
		},
		{
			name: "day new york both off grid",
			params: streaming.QueryParams{
				From:           &fromMid,
				To:             &toMid,
				WindowSize:     lo.ToPtr(meter.WindowSizeDay),
				WindowTimeZone: newYork,
				GroupBy:        []string{"subject", "group2"},
			},
		},
		{
			name: "month utc no group by",
			params: streaming.QueryParams{
				From:       &fromOn,
				To:         &toOn,
				WindowSize: lo.ToPtr(meter.WindowSizeMonth),
			},
		},
		{
			name: "month new york subject group by",
			params: streaming.QueryParams{
				From:           &fromMid,
				To:             &toOn,
				WindowSize:     lo.ToPtr(meter.WindowSizeMonth),
				WindowTimeZone: newYork,
				GroupBy:        []string{"subject"},
			},
		},
		{
			name: "hour new york multi subject filter",
			params: streaming.QueryParams{
				From:           &fromOn,
				To:             &toMid,
				WindowSize:     lo.ToPtr(meter.WindowSizeHour),
				WindowTimeZone: newYork,
				FilterSubject:  []string{"subject-1", "subject-2"},
				GroupBy:        []string{"subject"},
			},
		},
		{
			name: "total off grid dim filter subset group by",
			params: streaming.QueryParams{
				From:    &fromMid,
				To:      &toMid,
				GroupBy: []string{"group1"},
				FilterGroupBy: map[string]filter.FilterString{
					"group2": {Eq: lo.ToPtr("x")},
				},
			},
		},
	}

	for _, aggregation := range allCacheTestAggregations {
		for _, queryCase := range queryCases {
			s.Run(fmt.Sprintf("%s %s", aggregation, queryCase.name), func() {
				live, cached := s.queryBothPaths(ctx, c, namespace, meters[aggregation], queryCase.params)

				s.NotEmpty(live, "parity would be trivial on an empty result")
				s.ElementsMatch(live, cached)
			})
		}
	}
}

// TestLatestAggregationAlwaysRoutesLive proves LATEST is excluded from the cache at the
// gate, not merely undeployed: even with Cachable set, a LATEST query never reaches the
// cache leg — it logs the gate's rejection reason and its result is exactly what an
// otherwise-identical query with the cache disabled entirely would produce.
//
// Watched RED: commenting out the meterCacheStaticReject LATEST check (so the gate falls
// through to cacheRejectReasonViewMissing instead, since no LATEST MV is ever deployed)
// still leaves this test green on the "serving live" and result-equality assertions —
// those degrade gracefully either way. Only the reject-reason assertion below distinguishes
// "excluded by design" from "happens to have no view deployed", which is why
// TestMeterCacheStaticReject's dedicated LATEST case (asserting the exact reject reason) is
// the real red/green vehicle for the gate change; this test is the complementary proof that
// the exclusion holds end to end against a real ClickHouse.
func (s *MeterCacheCHTestSuite) TestLatestAggregationAlwaysRoutesLive() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-latest-routing"
		eventType = "api-calls"
	)

	cache := CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	}
	c := s.newCacheConnector(ctx, cache)
	noCache := s.newCacheConnector(ctx, CacheConfig{})

	m := meter.Meter{
		Key:           "meter-latest-routing",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationLatest,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-3 * time.Hour).Truncate(time.Hour)

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 2}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 7}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-2", now.Add(-30*time.Minute), `{"value": 13}`, now),
	)

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	params := streaming.QueryParams{From: &from, To: &to, WindowSize: lo.ToPtr(meter.WindowSizeHour)}

	// A cache-disabled connector's gate never runs at all — cacheGate is nil and QueryMeter
	// always serves live for it — so it is the independent ground truth for "no cache leg
	// was ever consulted", distinct from the Cachable=true run under test.
	paramsLive := params
	paramsLive.Cachable = false
	wantRows, err := noCache.QueryMeter(ctx, namespace, m, paramsLive)
	s.NoError(err)
	s.NotEmpty(wantRows, "routing proof would be trivial on an empty result")

	c.logs.Reset()

	paramsCachable := params
	paramsCachable.Cachable = true
	gotRows, err := c.QueryMeter(ctx, namespace, m, paramsCachable)
	s.NoError(err)

	s.Contains(c.logs.String(), "serving live", "a Cachable LATEST query must be rejected by the gate, not silently served some other way")
	s.Contains(c.logs.String(), string(cacheRejectReasonLatestAggregation), "the gate must reject LATEST specifically, not fall through to a different reason")
	s.Equal(wantRows, gotRows)
}

// TestCachedReadParityAcrossDST extends the parity matrix's timezone axis over both 2025
// US DST transitions (2025-03-09 spring forward, 2025-11-02 fall back, both fully settled
// history served entirely by the backfilled cache): day and month windows in
// America/New_York re-window UTC cache buckets across a UTC-offset change, including an
// event in the repeated local hour of fall-back and events on the far side of local
// midnight where the NY calendar day differs from the UTC day.
func (s *MeterCacheCHTestSuite) TestCachedReadParityAcrossDST() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-parity-dst"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	newYork, err := time.LoadLocation("America/New_York")
	s.NoError(err)

	storedAt := time.Now().UTC()

	event := func(subject string, at time.Time, data string) streaming.RawEvent {
		return rawCacheTestEvent(namespace, eventType, subject, at, data, storedAt)
	}

	utc := func(value string) time.Time {
		at, err := time.Parse(time.RFC3339, value)
		s.NoError(err)

		return at.UTC()
	}

	s.insertRawEvents(ctx,
		// Spring forward (2025-03-09 07:00Z): 04:30Z is still NY day Mar 8, 06:30Z is the
		// last EST hour, 07:30Z the first EDT hour (local 02:00-03:00 never happened).
		event("subject-1", utc("2025-03-09T04:30:00Z"), `{"value": 2, "group1": "a", "group2": "x"}`),
		event("subject-1", utc("2025-03-09T06:30:00Z"), `{"value": 7, "group1": "a", "group2": "y"}`),
		event("subject-2", utc("2025-03-09T07:30:00Z"), `{"value": 5, "group1": "b", "group2": "x"}`),
		event("subject-1", utc("2025-03-09T08:30:00Z"), `{"value": null, "group1": "b", "group2": "y"}`),
		// Mid-summer: 03:30Z is NY day Jun 14 but UTC day Jun 15 (day-boundary divergence).
		event("subject-2", utc("2025-06-15T03:30:00Z"), `{"value": 3.5, "group1": "a", "group2": "x"}`),
		// Fall back (2025-11-02 06:00Z): 05:30Z is local 01:30 EDT, 06:30Z is local 01:30
		// EST — the repeated local hour; 07:30Z is unambiguous again. The value 7 repeats
		// across the transition for UNIQUE_COUNT dedupe.
		event("subject-1", utc("2025-11-02T04:30:00Z"), `{"value": 7, "group1": "a", "group2": "x"}`),
		event("subject-2", utc("2025-11-02T05:30:00Z"), `{"value": 11, "group1": "b", "group2": "x"}`),
		event("subject-1", utc("2025-11-02T06:30:00Z"), `{"value": 13, "group1": "a", "group2": "y"}`),
		event("subject-2", utc("2025-11-02T07:30:00Z"), `{"group1": "b", "group2": "y"}`),
	)

	meters := s.deployCacheTestMeters(ctx, namespace, eventType, "dst-meter", c.config.Cache)

	fromFull := utc("2025-03-01T00:00:00Z")
	toFull := utc("2025-11-30T00:00:00Z")
	fromSpring := utc("2025-03-09T00:00:00Z")
	toSpring := utc("2025-03-10T00:00:00Z")
	fromFall := utc("2025-11-02T00:00:00Z")
	toFall := utc("2025-11-03T00:00:00Z")

	queryCases := []struct {
		name   string
		params streaming.QueryParams
	}{
		{
			name: "day new york across both transitions",
			params: streaming.QueryParams{
				From:           &fromFull,
				To:             &toFull,
				WindowSize:     lo.ToPtr(meter.WindowSizeDay),
				WindowTimeZone: newYork,
				GroupBy:        []string{"subject"},
			},
		},
		{
			name: "month new york across both transitions",
			params: streaming.QueryParams{
				From:           &fromFull,
				To:             &toFull,
				WindowSize:     lo.ToPtr(meter.WindowSizeMonth),
				WindowTimeZone: newYork,
				GroupBy:        []string{"subject", "group1"},
			},
		},
		{
			name: "day utc across both transitions",
			params: streaming.QueryParams{
				From:       &fromFull,
				To:         &toFull,
				WindowSize: lo.ToPtr(meter.WindowSizeDay),
			},
		},
		{
			name: "hour new york spring forward day",
			params: streaming.QueryParams{
				From:           &fromSpring,
				To:             &toSpring,
				WindowSize:     lo.ToPtr(meter.WindowSizeHour),
				WindowTimeZone: newYork,
				GroupBy:        []string{"subject", "group2"},
			},
		},
		{
			name: "hour new york fall back day",
			params: streaming.QueryParams{
				From:           &fromFall,
				To:             &toFall,
				WindowSize:     lo.ToPtr(meter.WindowSizeHour),
				WindowTimeZone: newYork,
			},
		},
	}

	for _, aggregation := range allCacheTestAggregations {
		for _, queryCase := range queryCases {
			s.Run(fmt.Sprintf("%s %s", aggregation, queryCase.name), func() {
				live, cached := s.queryBothPaths(ctx, c, namespace, meters[aggregation], queryCase.params)

				s.NotEmpty(live, "parity would be trivial on an empty result")
				s.ElementsMatch(live, cached)
			})
		}
	}
}
