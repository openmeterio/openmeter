package clickhouse

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// TestPoisonThroughBatchInsertConverges drives a late event through the production
// ingestion hook (Connector.BatchInsert) and proves the full invalidation lifecycle: the
// marker is written with a server-side timestamp, the affected MV's refresh is triggered,
// and once that refresh completes the marker is healed under the G1 rule (it started
// after the marker, well within the heal bound) so the cached read serves the recomputed
// bucket — poison included — byte-equal with live.
func (s *MeterCacheCHTestSuite) TestPoisonThroughBatchInsertConverges() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-poison-ingest"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	// given:
	// - two settled events cached via deploy (insert bypasses BatchInsert so the baseline
	//   has no markers)
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 2}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 7}`, now),
	)

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
	s.deployMeterCacheMV(ctx, mv)

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	totalParams := streaming.QueryParams{From: &from, To: &to}

	live, cached := s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(9), s.totalValue(live))
	s.Equal(float64(9), s.totalValue(cached))

	// when:
	// - a late event lands through BatchInsert (the production ingestion path)
	s.NoError(c.BatchInsert(ctx, []streaming.RawEvent{
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(40*time.Minute), `{"value": 1000}`, now),
	}))

	// then:
	// - one marker was written and one best-effort refresh trigger fired
	var markerCount uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		"SELECT count() FROM "+getTableName(s.Database, meterCacheInvalidationsTableName),
	).Scan(&markerCount))
	s.Equal(uint64(1), markerCount)
	s.Equal(uint64(1), c.cacheInvalidator.refreshTriggersFired.Load())
	s.Equal(uint64(0), c.cacheInvalidator.markerInsertFailures.Load())

	// when:
	// - a refresh completes measurably after the marker (the invalidator-triggered one
	//   usually is that refresh already; extra refreshes only compensate the
	//   second-resolution refresh-start bookkeeping, see refreshViewUntilMarkersHealed)
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+getTableName(s.Database, mv.name())))
	s.refreshViewUntilMarkersHealed(ctx, mv.name())

	// then:
	// - the marker is healed (the refresh started after it, delta far below the 20m heal
	//   bound) and the cached read serves the recomputed bucket including the poison
	live, cached = s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(1009), s.totalValue(live))
	s.Equal(float64(1009), s.totalValue(cached))
}

// TestPoisonBypassStaysCachedUntilRefresh proves the cache leg really serves cached
// buckets by poisoning one behind the invalidation machinery's back: a late event
// inserted with direct SQL (no BatchInsert, hence no marker) must stay invisible to the
// cached read — had the gate silently fallen back to live, cached and live would agree —
// and the next refresh must converge the bucket because the poison carries a fresh
// stored_at that the dirty-bucket lookback picks up.
func (s *MeterCacheCHTestSuite) TestPoisonBypassStaysCachedUntilRefresh() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-poison-bypass"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 2}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 7}`, now),
	)

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
	s.deployMeterCacheMV(ctx, mv)

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	totalParams := streaming.QueryParams{From: &from, To: &to}

	live, cached := s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(9), s.totalValue(live))
	s.Equal(float64(9), s.totalValue(cached))

	// when:
	// - a late event is inserted bypassing BatchInsert: no marker, no refresh trigger
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(40*time.Minute), `{"value": 1000}`, now),
	)

	// then:
	// - live sees the poison, the cached read serves the stale cached bucket: the
	//   inequality is the proof the cache leg (not a silent live fallback) produced the
	//   cached rows
	live, cached = s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(1009), s.totalValue(live))
	s.Equal(float64(9), s.totalValue(cached))

	// when:
	// - a refresh runs: the poison's stored_at is fresh, so the dirty-bucket filter
	//   recomputes its bucket even without any marker
	qualifiedView := getTableName(s.Database, mv.name())
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	// then:
	// - the cached read converges to live
	live, cached = s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(1009), s.totalValue(live))
	s.Equal(float64(1009), s.totalValue(cached))
}

// TestMeterHashFilterGuardsShapeCollision pins the cache leg's meter_hash filter (G8): a
// meter shape change leaves the old shape's rows in om_meter_cache under the same
// namespace and meter key until the reconciler GCs them, and reads of the new shape must
// never co-read them. The old shape is deployed second, so its row versions are newer: a
// read missing the hash filter would newest-wins-pick the old shape's value and return 3
// instead of 30 for the settled bucket.
//
// Watched RED with the guard reverted: removing the meter_hash WHERE clause from
// meterCacheReadQuery.cacheLeg fails the "current shape" assertions below with a cached
// total of 43 (the old shape's newer row wins the settled bucket: 3, plus the 40 live
// tail) instead of 70, proving cross-shape pollution.
func (s *MeterCacheCHTestSuite) TestMeterHashFilterGuardsShapeCollision() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-shape-collision"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	// The same meter key with two value properties models a meter shape edit: the current
	// shape reads $.value2, the pre-edit shape read $.value.
	currentShape := meter.Meter{
		Key:           "meter-shape",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value2"),
	}

	oldShape := meter.Meter{
		Key:           "meter-shape",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 1, "value2": 10}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 2, "value2": 20}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", now.Add(-30*time.Minute), `{"value": 4, "value2": 40}`, now),
	)

	// Deploy order is load-bearing: the old shape second, so its cache rows carry newer
	// created_at versions and would win a hash-less newest-wins pick.
	for _, m := range []meter.Meter{currentShape, oldShape} {
		s.deployMeterCacheMV(ctx, createMeterCacheMV{
			Database:        s.Database,
			EventsTableName: e2eEventsTable,
			Namespace:       namespace,
			Meter:           m,
			Grain:           CacheGrainHour,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
		})
	}

	// Sanity: both shapes' rows share (namespace, meter_key) and differ only in hash.
	var distinctHashes uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT uniqExact(meter_hash) FROM %s WHERE namespace = ? AND meter_key = ?", getTableName(s.Database, meterCacheTableName)),
		namespace, "meter-shape",
	).Scan(&distinctHashes))
	s.Equal(uint64(2), distinctHashes)

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	totalParams := streaming.QueryParams{From: &from, To: &to}

	// then:
	// - the current shape reads only its own rows (30 cached + 40 live tail)
	live, cached := s.queryBothPaths(ctx, c, namespace, currentShape, totalParams)
	s.Equal(float64(70), s.totalValue(live))
	s.Equal(float64(70), s.totalValue(cached))

	// then:
	// - the old shape also reads only its own rows (3 cached + 4 live tail)
	live, cached = s.queryBothPaths(ctx, c, namespace, oldShape, totalParams)
	s.Equal(float64(7), s.totalValue(live))
	s.Equal(float64(7), s.totalValue(cached))
}

// TestGrainChangeIsolatesOldGrainRows is the G4 regression: a cache grain change is a
// shape change, so rows rolled up at the previous grain must be invisible to reads at the
// current grain even while both row sets coexist in om_meter_cache (the reconciler GCs
// the old ones eventually, but reads must be correct immediately).
//
// Watched RED with the guard reverted: dropping the grain component from meterHash's
// input (meter_cache_hash.go) collapses the two deployments onto one hash — the
// distinct-hash sanity assertion below reads 1 instead of 2, and past it the hour read
// would co-read the minute-grain rows and double the total.
func (s *MeterCacheCHTestSuite) TestGrainChangeIsolatesOldGrainRows() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-grain-change"
		eventType = "api-calls"
	)

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	// Two events in distinct minute buckets of the same hour bucket: at minute grain they
	// roll up into two cache rows, at hour grain into one — co-reading both grains would
	// double the hour total.
	minuteConnector := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainMinute,
	})

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 2}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 7}`, now),
	)

	for _, grain := range []CacheGrain{CacheGrainMinute, CacheGrainHour} {
		s.deployMeterCacheMV(ctx, createMeterCacheMV{
			Database:        s.Database,
			EventsTableName: e2eEventsTable,
			Namespace:       namespace,
			Meter:           m,
			Grain:           grain,
			RefreshInterval: 10 * time.Minute,
			MinimumUsageAge: time.Hour,
		})
	}

	hourConnector := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	// Sanity: both grains' row sets coexist under distinct hashes (2 minute rows + 1 hour
	// row) — the exact situation a grain change leaves behind until GC.
	var distinctHashes, rowCount uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT uniqExact(meter_hash), count() FROM %s FINAL WHERE namespace = ?", getTableName(s.Database, meterCacheTableName)),
		namespace,
	).Scan(&distinctHashes, &rowCount))
	s.Equal(uint64(2), distinctHashes)
	s.Equal(uint64(3), rowCount)

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	totalParams := streaming.QueryParams{From: &from, To: &to}

	// then:
	// - the hour-grain connector reads only hour-hash rows
	live, cached := s.queryBothPaths(ctx, hourConnector, namespace, m, totalParams)
	s.Equal(float64(9), s.totalValue(live))
	s.Equal(float64(9), s.totalValue(cached))

	// then:
	// - the minute-grain connector reads only minute-hash rows (windowed to prove the
	//   finer buckets re-window into hour windows correctly)
	live, cached = s.queryBothPaths(ctx, minuteConnector, namespace, m, streaming.QueryParams{
		From:       &from,
		To:         &to,
		WindowSize: lo.ToPtr(meter.WindowSizeHour),
	})
	s.NotEmpty(live)
	s.ElementsMatch(live, cached)
}

// TestStaleMarkerKeepsReaderLive is the G1 regression: an invalidation marker that aged
// past what any refresh's stored_at lookback provably covered must never be considered
// healed — the reader stays live for the marked range (correctly seeing the late event in
// the raw table) until the reconciler re-backfills, while ranges no marker overlaps keep
// serving from the cache.
//
// The extended outage is modeled without waiting: the late event and its marker are
// backdated (stored_at and created_at two hours in the past, beyond the 1h30m dirty
// window and the 20m heal bound) exactly as an outage would leave them — production
// markers always carry the server-side DEFAULT timestamp, the explicit value here only
// simulates their age. The view is STOPped across the "outage" and resumed before the
// verifying refresh, mirroring the plan's stop → late event + marker → age → resume
// sequence.
//
// Watched RED with the guard reverted: removing the
// `created_at <= refreshStart - healBound` arm from meterCacheMarkerOverlapQuery.toSQL
// wrongly heals the aged marker and the marked-range read below executes the cached
// query — the unhealed_markers log assertion goes red first, and the total would read 9
// (the cached bucket that never saw the late event) instead of 109.
func (s *MeterCacheCHTestSuite) TestStaleMarkerKeepsReaderLive() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-stale-marker"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucketP := now.Add(-5 * time.Hour).Truncate(time.Hour)
	bucketQ := now.Add(-3 * time.Hour).Truncate(time.Hour)

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucketP.Add(10*time.Minute), `{"value": 2}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucketQ.Add(10*time.Minute), `{"value": 7}`, now),
	)

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
	s.deployMeterCacheMV(ctx, mv)

	qualifiedView := getTableName(s.Database, mv.name())

	// when:
	// - refreshing stops (the outage begins)
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM STOP VIEW "+qualifiedView))

	// - a late event lands during the outage; by resume time its stored_at (now-2h) has
	//   aged out of the 1h30m dirty lookback, so no future refresh will ever recompute its
	//   bucket
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucketP.Add(20*time.Minute), `{"value": 100}`, now.Add(-2*time.Hour)),
	)

	// - its marker aged with it (2h > the 20m heal bound)
	s.NoError(s.ClickHouse.Exec(ctx,
		fmt.Sprintf("INSERT INTO %s (namespace, event_type, window_lo, window_hi, created_at) VALUES (?, ?, ?, ?, ?)",
			getTableName(s.Database, meterCacheInvalidationsTableName)),
		namespace, eventType, bucketP, bucketP.Add(time.Hour), now.Add(-2*time.Hour),
	))

	// - refreshing resumes and a refresh completes
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM START VIEW "+qualifiedView))
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	to := now.Truncate(time.Hour).Add(time.Hour)

	// then:
	// - the marked range is served live (the resumed refresh could not have covered the
	//   marker) and therefore sees the late event the cache is missing
	c.logs.Reset()

	overlapping, err := c.QueryMeter(ctx, namespace, m, streaming.QueryParams{From: &bucketP, To: &to, Cachable: true})
	s.NoError(err)
	s.Contains(c.logs.String(), "reason=unhealed_markers")
	s.Equal(float64(109), s.totalValue(overlapping))

	// then:
	// - a range the marker does not overlap still serves from the cache
	live, cached := s.queryBothPaths(ctx, c, namespace, m, streaming.QueryParams{From: &bucketQ, To: &to})
	s.Equal(float64(7), s.totalValue(live))
	s.Equal(float64(7), s.totalValue(cached))
}

// TestFutureDatedEventCachedAfterSettling is the G2 regression: an event whose stored_at
// is far in the past relative to its event time (a future-dated ingest) is invisible to
// the dirty stored_at lookback by the time its bucket settles, so only the newly-settled
// strip can bring it into the cache. The scheduled refresh must recompute the strip
// bucket and the cache row must include the event's contribution.
//
// Watched RED with the guard reverted: removing the "UNION DISTINCT ... numbers(...)"
// newly-settled-strip arm from dirtyBucketFilterExpr leaves the strip bucket
// unrecomputed (its only events' stored_at is days old) and the newest cache row below
// stays at 3 instead of 8.
func (s *MeterCacheCHTestSuite) TestFutureDatedEventCachedAfterSettling() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-future-dated"
		eventType = "api-calls"
	)

	// 30m interval → the strip unconditionally recomputes the 2 grain buckets below the
	// settled bound, tolerating the bound advancing one bucket mid-test.
	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 30 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	settledBound := now.Add(-time.Hour).Truncate(time.Hour)
	stripBucket := settledBound.Add(-time.Hour)
	oldStoredAt := now.Add(-3 * 24 * time.Hour)

	// given:
	// - the strip bucket already holds one settled event with a days-old stored_at (it
	//   reaches the cache via the deploy backfill, which has no stored_at restriction)
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", stripBucket.Add(10*time.Minute), `{"value": 3}`, oldStoredAt),
	)

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 30 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
	s.deployMeterCacheMV(ctx, mv)

	newestStripBucketSum := func() float64 {
		var value NullDecimal
		s.NoError(s.ClickHouse.QueryRow(ctx,
			fmt.Sprintf("SELECT argMax(sum_value, created_at) FROM %s WHERE namespace = ? AND meter_hash = ? AND windowstart = ?",
				getTableName(s.Database, meterCacheTableName)),
			namespace, meterHash(m, CacheGrainHour), stripBucket,
		).Scan(&value))
		s.True(value.Valid)

		return value.Decimal.InexactFloat64()
	}

	s.Equal(float64(3), newestStripBucketSum())

	// when:
	// - a future-dated event lands in the already-settled strip bucket: its event time is
	//   fresh but its stored_at is days old, so the dirty stored_at lookback (2h30m here)
	//   can never see it — only the strip recompute can
	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", stripBucket.Add(30*time.Minute), `{"value": 5}`, oldStoredAt),
	)

	qualifiedView := getTableName(s.Database, mv.name())
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
	s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))

	// then:
	// - the strip recompute appended a bucket version that includes the future-dated event
	s.Equal(float64(8), newestStripBucketSum())

	// then:
	// - reads stay in parity throughout (the strip bucket itself sits at the G5 epsilon
	//   boundary and is served by the live post leg today; the cache row matters the
	//   moment the horizon advances past it)
	from := stripBucket.Add(-time.Hour)
	to := now.Truncate(time.Hour).Add(time.Hour)

	live, cached := s.queryBothPaths(ctx, c, namespace, m, streaming.QueryParams{From: &from, To: &to})
	s.Equal(float64(8), s.totalValue(live))
	s.Equal(float64(8), s.totalValue(cached))
}

// TestUnstampedBackfillServesLive is the G3 regression on a real ClickHouse: a healthy,
// refreshing MV whose comment has no backfilled_at stamp must be refused by the reader —
// its cache rows only cover recently refreshed buckets, so serving it would silently drop
// older history. Stamping the very same view flips the gate to cache-serving without any
// other change.
func (s *MeterCacheCHTestSuite) TestUnstampedBackfillServesLive() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-unstamped"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 2}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 7}`, now),
	)

	// given:
	// - a healthy view (created, refreshed, even backfilled) that was never stamped: the
	//   exact state a leader crash between backfill and MODIFY COMMENT leaves behind
	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
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

	backfillSQL, err := meterCacheBackfill{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		MinimumUsageAge: time.Hour,
	}.toSQL()
	s.NoError(err)
	s.NoError(s.ClickHouse.Exec(ctx, backfillSQL))

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)

	// then:
	// - the cached read refuses the unstamped view and serves live (correct values via
	//   the raw table)
	c.logs.Reset()

	rows, err := c.QueryMeter(ctx, namespace, m, streaming.QueryParams{From: &from, To: &to, Cachable: true})
	s.NoError(err)
	s.Contains(c.logs.String(), "reason=backfill_unstamped")
	s.Equal(float64(9), s.totalValue(rows))

	// when:
	// - the stamp lands (what EnsureMeterCache does last)
	metadata, err := mv.metadata()
	s.NoError(err)
	metadata.BackfilledAt = lo.ToPtr(time.Now().UTC().Truncate(time.Second))

	comment, err := metadata.marshal()
	s.NoError(err)
	s.NoError(s.ClickHouse.Exec(ctx, fmt.Sprintf("ALTER TABLE %s MODIFY COMMENT %s", qualifiedView, sqlStringLiteral(comment))))

	// then:
	// - the same query is now served from the cache with identical values
	live, cached := s.queryBothPaths(ctx, c, namespace, m, streaming.QueryParams{From: &from, To: &to})
	s.Equal(float64(9), s.totalValue(live))
	s.Equal(float64(9), s.totalValue(cached))
}

// TestHorizonBucketParity is the G5 regression: with events in every hour bucket up to
// now, the cached read must be row-for-row equal with live across the cache horizon —
// the epsilon bucket (one grain below the horizon) and everything above it are served by
// the live post leg, everything below by the cache leg, and the seams must be invisible
// in the result.
func (s *MeterCacheCHTestSuite) TestHorizonBucketParity() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-horizon"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()

	// One event in every hour bucket of the last six hours with distinct powers of two:
	// any bucket lost or double-counted at the cache/live seam changes the total to a
	// value no other combination of buckets can produce.
	events := make([]streaming.RawEvent, 0, 6)
	for i := 1; i <= 6; i++ {
		bucket := now.Add(-time.Duration(i) * time.Hour).Truncate(time.Hour)
		events = append(events, rawCacheTestEvent(
			namespace, eventType, "subject-1",
			bucket.Add(30*time.Minute),
			fmt.Sprintf(`{"value": %d}`, 1<<i),
			now,
		))
	}
	s.insertRawEvents(ctx, events...)

	s.deployMeterCacheMV(ctx, createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	})

	from := now.Add(-6 * time.Hour).Truncate(time.Hour)
	to := now.Truncate(time.Hour).Add(time.Hour)

	// then:
	// - windowed rows match bucket by bucket across the horizon
	live, cached := s.queryBothPaths(ctx, c, namespace, m, streaming.QueryParams{
		From:       &from,
		To:         &to,
		WindowSize: lo.ToPtr(meter.WindowSizeHour),
	})
	s.Len(live, 6)
	s.ElementsMatch(live, cached)

	// then:
	// - the total across the horizon equals the exact sum of all six buckets
	live, cached = s.queryBothPaths(ctx, c, namespace, m, streaming.QueryParams{From: &from, To: &to})
	s.Equal(float64(126), s.totalValue(live))
	s.Equal(float64(126), s.totalValue(cached))
}

// TestNewestWinsRecomputedBucket is the 15/5/3 newest-wins regression from the R4 live
// demo, driven through the real machinery: a bucket is cached at sum 15, then the
// underlying events are rewritten twice (delete + reinsert with fresh stored_at, so each
// refresh recomputes and re-appends the bucket). Every read must return exactly the
// newest recompute — 5 then 3 — never 20/23 (summing appended versions, the
// AggregatingMergeTree failure mode the design rejected) and never a stale 15/5.
func (s *MeterCacheCHTestSuite) TestNewestWinsRecomputedBucket() {
	t := s.T()
	ctx := t.Context()

	const (
		namespace = "cache-newest-wins"
		eventType = "api-calls"
	)

	c := s.newCacheConnector(ctx, CacheConfig{
		Enabled:         true,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
		WindowSize:      CacheGrainHour,
	})

	m := meter.Meter{
		Key:           "meter-sum",
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	now := time.Now().UTC()
	bucket := now.Add(-4 * time.Hour).Truncate(time.Hour)

	s.insertRawEvents(ctx,
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(5*time.Minute), `{"value": 7}`, now),
		rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(10*time.Minute), `{"value": 8}`, now),
	)

	mv := createMeterCacheMV{
		Database:        s.Database,
		EventsTableName: e2eEventsTable,
		Namespace:       namespace,
		Meter:           m,
		Grain:           CacheGrainHour,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
	s.deployMeterCacheMV(ctx, mv)

	from := bucket
	to := now.Truncate(time.Hour).Add(time.Hour)
	totalParams := streaming.QueryParams{From: &from, To: &to}

	qualifiedView := getTableName(s.Database, mv.name())

	rewriteBucketEvents := func(value int) {
		// Lightweight delete + reinsert models history rewrites (dedupe, GDPR erasure,
		// corrections); the fresh stored_at makes the next refresh recompute the bucket.
		s.NoError(s.ClickHouse.Exec(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE namespace = ?", getTableName(s.Database, e2eEventsTable)),
			namespace,
		))

		s.insertRawEvents(ctx,
			rawCacheTestEvent(namespace, eventType, "subject-1", bucket.Add(15*time.Minute), fmt.Sprintf(`{"value": %d}`, value), time.Now().UTC()),
		)

		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+qualifiedView))
		s.NoError(s.ClickHouse.Exec(ctx, "SYSTEM WAIT VIEW "+qualifiedView))
	}

	// then: the initial state reads 15 on both paths
	live, cached := s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(15), s.totalValue(live))
	s.Equal(float64(15), s.totalValue(cached))

	// when/then: first recompute — 5, not 20 (15+5 would mean appended versions summed)
	rewriteBucketEvents(5)

	live, cached = s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(5), s.totalValue(live))
	s.Equal(float64(5), s.totalValue(cached))

	// when/then: second recompute — 3, not 23/8/15
	rewriteBucketEvents(3)

	live, cached = s.queryBothPaths(ctx, c, namespace, m, totalParams)
	s.Equal(float64(3), s.totalValue(live))
	s.Equal(float64(3), s.totalValue(cached))

	// then:
	// - all three appended versions coexist in storage and FINAL collapses them to the
	//   newest, proving the reads above picked by version, not by luck of a merge
	var versions, finalRows uint64
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT count() FROM %s WHERE namespace = ? AND windowstart = ?", getTableName(s.Database, meterCacheTableName)),
		namespace, bucket,
	).Scan(&versions))
	s.NoError(s.ClickHouse.QueryRow(ctx,
		fmt.Sprintf("SELECT count() FROM %s FINAL WHERE namespace = ? AND windowstart = ?", getTableName(s.Database, meterCacheTableName)),
		namespace, bucket,
	).Scan(&finalRows))
	s.GreaterOrEqual(versions, uint64(1))
	s.Equal(uint64(1), finalRows)
}
