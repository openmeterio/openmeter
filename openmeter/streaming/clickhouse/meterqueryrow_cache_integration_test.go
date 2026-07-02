package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	progressmanager "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// enableCacheOnConnector turns on the query cache on the suite connector (small
// horizon so the parity extremes are reachable) and ensures the rollup table
// exists.
func (s *ConnectorTestSuite) enableCacheOnConnector() {
	// Mutate the existing connector's config in place: the events table and DB
	// are already created; we only need the cache table + flags.
	s.Connector.config.QueryCacheEnabled = true
	s.Connector.config.QueryCacheMinimumCacheableQueryPeriod = time.Hour
	s.Connector.config.QueryCacheMinimumCacheableUsageAge = time.Hour
	// The cache only serves decimal-precision queries (see canQueryBeCached):
	// the rollup stores Decimal128 and recombines exactly. Configure the test to
	// that mode, which is the only one the cache supports.
	s.Connector.config.EnableDecimalPrecision = true
	s.NoError(s.Connector.createMeterQueryRowCacheTable(s.T().Context()))
}

// TestQueryCacheParity is the billing-safety gate: for every admitted
// (meter, window size, filter subset, aggregation), and at the two extremes
// (all-fresh cutoff==from, all-cached cutoff==to) plus interior cutoffs, the
// cached result equals the live result.
func (s *ConnectorTestSuite) TestQueryCacheParity() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	// Seed a deterministic multi-day window with grouped, multi-subject data,
	// including an all-null value-property event (min/max Nullable coverage).
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	eventType := "parity_event"

	// Events spread across ~3 days and 2 subjects, 2 group-by dimensions.
	events := []parityEvent{
		{subject: "s1", at: base.Add(30 * time.Minute), data: `{"value": 10, "region": "us", "tier": "free"}`},
		{subject: "s1", at: base.Add(90 * time.Minute), data: `{"value": 5, "region": "us", "tier": "free"}`},
		{subject: "s2", at: base.Add(2 * time.Hour), data: `{"value": 20, "region": "eu", "tier": "pro"}`},
		{subject: "s1", at: base.Add(26 * time.Hour), data: `{"value": 7, "region": "us", "tier": "pro"}`},
		{subject: "s2", at: base.Add(27 * time.Hour), data: `{"value": 3, "region": "eu", "tier": "free"}`},
		{subject: "s2", at: base.Add(50 * time.Hour), data: `{"value": 100, "region": "eu", "tier": "pro"}`},
		{subject: "s1", at: base.Add(51 * time.Hour), data: `{"value": 1, "region": "us", "tier": "free"}`},
		// All-null value property in its own settled hour window: exercises the
		// Nullable min/max path (live returns no row; SUM/COUNT still do).
		{subject: "s1", at: base.Add(70 * time.Hour), data: `{"region": "us", "tier": "free"}`},
	}
	s.seedEvents(ctx, eventType, events)

	groupByMap := map[string]string{"region": "$.region", "tier": "$.tier"}

	// Bound sets: hour-aligned, and OFF the hour grid on BOTH ends. The off-grid
	// set exercises the partial head [from, ceil(from)) and partial tail
	// [floor(cutoff), to) live legs simultaneously across every agg/window/cutoff,
	// closing the boundary class the aligned matrix structurally cannot reach.
	type bounds struct {
		name string
		from time.Time
		to   time.Time
	}
	boundSets := []bounds{
		{name: "aligned", from: base, to: base.Add(72 * time.Hour)},
		{name: "offgrid", from: base.Add(15 * time.Minute), to: base.Add(72*time.Hour - 20*time.Minute)},
	}

	meters := []parityMeter{
		{name: "parity_sum", eventType: eventType, valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationSum, groupBy: groupByMap},
		{name: "parity_count", eventType: eventType, aggregation: meterpkg.MeterAggregationCount, groupBy: groupByMap},
		{name: "parity_min", eventType: eventType, valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationMin, groupBy: groupByMap},
		{name: "parity_max", eventType: eventType, valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationMax, groupBy: groupByMap},
	}

	// Window sizes handled by the direct-merge comparison. nil (total) is included:
	// with To set, QueryMeter overwrites windowstart/end with From/To on both the
	// live and cached path, so they align. (Total with To==nil is routed to live by
	// the gate and thus never reaches the cache.)
	windowSizes := []*meterpkg.WindowSize{
		nil,
		ptr(meterpkg.WindowSizeHour),
		ptr(meterpkg.WindowSizeDay),
		ptr(meterpkg.WindowSizeMonth),
	}

	// Filter/group-by subsets: no grouping, group by one/both dims, subject
	// group-by, subject filter, group-by value filter.
	type filterCase struct {
		name          string
		groupBy       []string
		filterSubject []string
		filterGroupBy map[string]filter.FilterString
	}
	filterCases := []filterCase{
		{name: "total-nofilter"},
		{name: "group-region", groupBy: []string{"region"}},
		{name: "group-region-tier", groupBy: []string{"region", "tier"}},
		{name: "group-subject", groupBy: []string{"subject"}},
		{name: "group-subject-region", groupBy: []string{"subject", "region"}},
		{name: "filter-subject-s1", filterSubject: []string{"s1"}},
		{name: "filter-region-us", groupBy: []string{"region"}, filterGroupBy: map[string]filter.FilterString{"region": filterEq("us")}},
		// Non-Eq operators exercise filterStringWhere's fragment splicing (the
		// cached path renders these on group_by[i] while live renders them on the
		// raw JSON extraction — they must select identical row sets).
		{name: "filter-region-in", groupBy: []string{"region"}, filterGroupBy: map[string]filter.FilterString{"region": {In: &[]string{"us", "eu"}}}},
		{name: "filter-tier-ne", groupBy: []string{"region", "tier"}, filterGroupBy: map[string]filter.FilterString{"tier": {Ne: ptr("pro")}}},
		{name: "filter-region-like", groupBy: []string{"region"}, filterGroupBy: map[string]filter.FilterString{"region": {Like: ptr("u%")}}},
	}

	for _, bs := range boundSets {
		from := bs.from
		to := bs.to

		// Cutoffs: extremes (from = all-fresh, to = all-cached) plus interior hour
		// boundaries. For the off-grid set, `from` and `to` are mid-hour, so the
		// all-fresh (cutoff=from) and all-cached (cutoff=to) extremes also exercise
		// the partial head and partial tail live legs.
		cutoffs := []time.Time{
			from, // all-fresh: cache empty (or head-only), tail is the whole range
			from.Add(24 * time.Hour),
			from.Add(48 * time.Hour),
			to, // all-cached: fresh tail empty (partial last hour still live)
		}

		for _, m := range meters {
			meter := s.newMeter(m)
			for _, ws := range windowSizes {
				for _, fc := range filterCases {
					params := streaming.QueryParams{
						From:          &from,
						To:            &to,
						WindowSize:    ws,
						GroupBy:       fc.groupBy,
						FilterSubject: fc.filterSubject,
						FilterGroupBy: fc.filterGroupBy,
					}
					live := s.liveRows(ctx, meter, params)

					for _, cutoff := range cutoffs {
						name := fmt.Sprintf("%s/%s/%s/%s/cutoff=%s", bs.name, m.name, windowSizeName(ws), fc.name, cutoff.Sub(from))
						s.Run(name, func() {
							// Fresh cache table state per case to isolate populate.
							s.truncateCacheTable(ctx)
							cached := s.cachedRows(ctx, meter, params, cutoff)
							if diff := compareMeterRows(live, cached); diff != "" {
								s.Failf("parity mismatch", "%s:\n%s", name, diff)
							}
						})
					}
				}
			}
		}
	}
}

// TestQueryCachePartialFirstWindow locks in that a mid-hour `from` produces the
// same partial first window in the cache as in the live query. Both the populate
// and the live query filter `time >= from`, so the hour window straddling `from`
// aggregates only the events at/after `from` on both sides; the cache leg's
// floored lower bound (windowstart >= tumbleStart(from)) includes that partial
// window without re-including the excluded earlier events.
func (s *ConnectorTestSuite) TestQueryCachePartialFirstWindow() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "partial_event"
	hour0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	from := hour0.Add(30 * time.Minute) // mid-hour lower bound
	to := hour0.Add(3 * time.Hour)

	// One event BEFORE from (excluded), one after (included), plus later windows.
	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: hour0.Add(15 * time.Minute), data: `{"value": 10}`}, // before from -> excluded
		{subject: "s1", at: hour0.Add(45 * time.Minute), data: `{"value": 20}`}, // in hour 0, after from
		{subject: "s1", at: hour0.Add(90 * time.Minute), data: `{"value": 5}`},  // hour 1
	})

	meter := s.newMeter(parityMeter{
		name: "partial_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})
	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}

	live := s.liveRows(ctx, meter, params)
	// cutoff mid-range so both legs contribute.
	cached := s.cachedRows(ctx, meter, params, hour0.Add(2*time.Hour))

	if diff := compareMeterRows(live, cached); diff != "" {
		s.Failf("partial-first-window parity mismatch", "%s", diff)
	}
	// The straddling hour must be 20 (only the post-from event), never 30.
	for _, r := range cached {
		if r.WindowStart.Equal(hour0) {
			s.Equal(float64(20), r.Value, "partial first window must exclude pre-from events")
		}
	}
}

// TestQueryCacheCrossQueryPartialWindow is the discriminating test for the
// shared-cache hazard: two queries over the same meter with DIFFERENT mid-hour
// `from`s that floor to the SAME hour must both match their live result. If
// populate stored partial (incomplete) hour windows, query A would write one
// partial for hour 00 and query B a different partial for the same key, and the
// read-time any() collapse would serve one query the other's partial — a silent
// wrong billing total. Storing only COMPLETE windows (head served live) makes
// every stored window identical regardless of `from`.
func (s *ConnectorTestSuite) TestQueryCacheCrossQueryPartialWindow() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "cross_event"
	hour0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := hour0.Add(3 * time.Hour)

	// Three events inside hour 0 at :10, :20, :50, plus a later window.
	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: hour0.Add(10 * time.Minute), data: `{"value": 1}`},
		{subject: "s1", at: hour0.Add(20 * time.Minute), data: `{"value": 2}`},
		{subject: "s1", at: hour0.Add(50 * time.Minute), data: `{"value": 4}`},
		{subject: "s1", at: hour0.Add(150 * time.Minute), data: `{"value": 8}`},
	})

	meter := s.newMeter(parityMeter{
		name: "cross_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	cutoff := hour0.Add(2 * time.Hour)

	// Query A: from = 00:15 (floors to hour 0). Live hour 0 = 2+4 = 6.
	fromA := hour0.Add(15 * time.Minute)
	paramsA := streaming.QueryParams{From: &fromA, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	liveA := s.liveRows(ctx, meter, paramsA)
	cachedA := s.cachedRows(ctx, meter, paramsA, cutoff)
	if diff := compareMeterRows(liveA, cachedA); diff != "" {
		s.Failf("query A parity mismatch", "%s", diff)
	}

	// Query B: from = 00:45 (also floors to hour 0), same wall-clock hour so it
	// would collide on the same stored key. Live hour 0 = 4 (only the :50 event).
	fromB := hour0.Add(45 * time.Minute)
	paramsB := streaming.QueryParams{From: &fromB, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	liveB := s.liveRows(ctx, meter, paramsB)
	cachedB := s.cachedRows(ctx, meter, paramsB, cutoff)
	if diff := compareMeterRows(liveB, cachedB); diff != "" {
		s.Failf("query B parity mismatch (cross-query partial-window pollution)", "%s", diff)
	}

	// Re-run A after B populated: A must STILL be correct (B must not have
	// overwritten hour 0 with its partial).
	cachedA2 := s.cachedRows(ctx, meter, paramsA, cutoff)
	if diff := compareMeterRows(liveA, cachedA2); diff != "" {
		s.Failf("query A re-run mismatch after B populated", "%s", diff)
	}
}

// TestQueryCacheMidHourFromLowCutoff exercises the boundary where the cutoff
// falls at or below the head-ceiling: mid-hour `from` with a cutoff <= headCeil.
// The cache range [headCeil, cutoff) is then empty, and the head [from, headCeil)
// and tail [cutoff, to) legs must NOT overlap (which would double-count events in
// [cutoff, headCeil)). Covers the all-fresh extreme (cutoff==from) with a
// mid-hour from, which the hour-aligned parity matrix cannot reach.
func (s *ConnectorTestSuite) TestQueryCacheMidHourFromLowCutoff() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "midhour_event"
	hour0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	from := hour0.Add(15 * time.Minute) // headCeil = 01:00
	to := hour0.Add(3 * time.Hour)

	// Event at 00:30 sits in [from, headCeil) — the overlap window if head and
	// tail both cover it.
	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: hour0.Add(30 * time.Minute), data: `{"value": 10}`},
		{subject: "s1", at: hour0.Add(90 * time.Minute), data: `{"value": 20}`},
	})

	meter := s.newMeter(parityMeter{
		name: "midhour_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})
	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	live := s.liveRows(ctx, meter, params)

	// Cutoffs at/below headCeil (01:00): the cache range is empty and head+tail
	// must partition the range without overlap.
	for _, cutoff := range []time.Time{from, hour0.Add(30 * time.Minute), hour0.Add(time.Hour)} {
		s.Run(cutoff.Sub(from).String(), func() {
			s.truncateCacheTable(ctx)
			cached := s.cachedRows(ctx, meter, params, cutoff)
			if diff := compareMeterRows(live, cached); diff != "" {
				s.Failf("mid-hour from / low cutoff parity mismatch", "%s", diff)
			}
			// The 00:30 event's hour must be 10, never 20 (doubled).
			for _, r := range cached {
				if r.WindowStart.Equal(hour0) {
					s.Equal(float64(10), r.Value, "head/tail overlap must not double-count hour 0")
				}
			}
		})
	}
}

// TestQueryCacheTwoMetersSameType covers the meter axis of the cache key: two
// distinct meters in the same namespace CAN share an event type (the unique
// constraint is on (namespace, key), not (namespace, event_type)). They must not
// collide in the cache — populating one must not corrupt the other's rows.
//
// The sharp case is two SUM meters over the SAME type with DIFFERENT value
// properties (e.g. sum of tokens vs sum of latency): both write sum_value under
// the same (ns,type,window,subject,group_by) sort key, so without a meter
// discriminator the read-time any(sum_value) serves one meter the other's total.
// (SUM-vs-MIN accidentally survives because any() skips the NULL non-owned
// column, so it is NOT a sufficient test of the collision.)
func (s *ConnectorTestSuite) TestQueryCacheTwoMetersSameType() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "shared_type"
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(3 * time.Hour)
	cutoff := base.Add(2 * time.Hour)

	// Distinct magnitudes for tokens vs latency so a collision is obvious.
	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: base.Add(15 * time.Minute), data: `{"tokens": 100, "latency": 3}`},
		{subject: "s1", at: base.Add(45 * time.Minute), data: `{"tokens": 200, "latency": 4}`},
		{subject: "s1", at: base.Add(90 * time.Minute), data: `{"tokens": 50, "latency": 1}`},
	})

	tokensMeter := s.newMeter(parityMeter{name: "shared_tokens", eventType: eventType, valueProperty: ptr("$.tokens"), aggregation: meterpkg.MeterAggregationSum})
	latencyMeter := s.newMeter(parityMeter{name: "shared_latency", eventType: eventType, valueProperty: ptr("$.latency"), aggregation: meterpkg.MeterAggregationSum})

	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}

	liveTokens := s.liveRows(ctx, tokensMeter, params)
	liveLatency := s.liveRows(ctx, latencyMeter, params)

	// Populate BOTH meters into the SAME cache table FIRST, then read each — with
	// NO re-population between populate and read. Re-populating right before the
	// read masks the collision (the just-inserted row wins the order-dependent
	// any() pick), so population and read are deliberately separated here.
	s.populateCache(ctx, tokensMeter, params, cutoff)
	s.populateCache(ctx, latencyMeter, params, cutoff)

	cachedTokens := s.readCached(ctx, tokensMeter, params, cutoff)
	cachedLatency := s.readCached(ctx, latencyMeter, params, cutoff)

	if diff := compareMeterRows(liveTokens, cachedTokens); diff != "" {
		s.Failf("tokens meter corrupted by latency meter sharing the event type", "%s", diff)
	}
	if diff := compareMeterRows(liveLatency, cachedLatency); diff != "" {
		s.Failf("latency meter corrupted by tokens meter sharing the event type", "%s", diff)
	}
}

func (s *ConnectorTestSuite) truncateCacheTable(ctx context.Context) {
	// The coverage claims must go with the rows: a surviving claim over a
	// truncated table would make the next read skip population and serve the
	// settled range as empty.
	for _, name := range []string{parityCacheTable, meterQueryRowCacheCoverageTable} {
		table := getTableName(s.Connector.config.Database, name)
		s.NoError(s.Connector.config.ClickHouse.Exec(ctx, "TRUNCATE TABLE IF EXISTS "+table))
	}
}

func windowSizeName(ws *meterpkg.WindowSize) string {
	if ws == nil {
		return "total"
	}
	return string(*ws)
}

// TestQueryCacheStoreRereadWithGap populates a settled range that has a GAP (an
// hour window with no events), then re-queries an OVERLAPPING range and asserts
// parity. This is the shape the old suite never had — where the two removed bugs
// met. It proves the rollup + merge is correct across separate populate/read
// calls with a real gap.
func (s *ConnectorTestSuite) TestQueryCacheStoreRereadWithGap() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	eventType := "gap_event"

	// Hours 0 and 3 have events; hours 1,2 are a gap; hour 5 more events.
	events := []parityEvent{
		{subject: "s1", at: base.Add(15 * time.Minute), data: `{"value": 10, "region": "us"}`},
		{subject: "s1", at: base.Add(3*time.Hour + 15*time.Minute), data: `{"value": 20, "region": "us"}`},
		{subject: "s2", at: base.Add(5*time.Hour + 15*time.Minute), data: `{"value": 30, "region": "eu"}`},
	}
	s.seedEvents(ctx, eventType, events)

	meter := s.newMeter(parityMeter{
		name: "gap_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum, groupBy: map[string]string{"region": "$.region"},
	})

	from := base
	to := base.Add(6 * time.Hour)

	// First: populate a sub-range [from, from+4h) (covers the gap), separate call.
	firstParams := streaming.QueryParams{From: &from, To: ptr(base.Add(4 * time.Hour)), WindowSize: ptr(meterpkg.WindowSizeHour), GroupBy: []string{"region"}}
	q := s.buildQueryMeter(meter, firstParams, []string{"region"})
	s.NoError(s.Connector.populateMeterQueryRowCache(ctx, q, from, base.Add(4*time.Hour)))

	// Now re-query the OVERLAPPING wider range with a cutoff inside the cached
	// portion, so the cache leg serves the gap+events and the tail serves hour 5.
	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour), GroupBy: []string{"region"}}
	live := s.liveRows(ctx, meter, params)
	cached := s.cachedRows(ctx, meter, params, base.Add(4*time.Hour))

	if diff := compareMeterRows(live, cached); diff != "" {
		s.Failf("store-reread parity mismatch", "%s", diff)
	}
}

// TestQueryCacheConcurrentPopulate proves §4.2: two concurrent populates of the
// SAME settled window must not double-count SUM/COUNT after the read-time cache
// leg collapse.
func (s *ConnectorTestSuite) TestQueryCacheConcurrentPopulate() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	eventType := "concurrent_event"
	events := []parityEvent{
		{subject: "s1", at: base.Add(15 * time.Minute), data: `{"value": 10}`},
		{subject: "s1", at: base.Add(45 * time.Minute), data: `{"value": 5}`},
	}
	s.seedEvents(ctx, eventType, events)

	meter := s.newMeter(parityMeter{
		name: "concurrent_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	from := base
	to := base.Add(2 * time.Hour)
	cutoff := base.Add(1 * time.Hour) // hour 0 is settled

	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	q := s.buildQueryMeter(meter, params, nil)

	// Populate the SAME settled window twice (simulating a race). With plain
	// MergeTree + no read-time collapse this would double the SUM.
	s.NoError(s.Connector.populateMeterQueryRowCache(ctx, q, from, cutoff))
	s.NoError(s.Connector.populateMeterQueryRowCache(ctx, q, from, cutoff))

	live := s.liveRows(ctx, meter, params)
	cached := s.cachedRows(ctx, meter, params, cutoff)

	if diff := compareMeterRows(live, cached); diff != "" {
		s.Failf("concurrent populate double-count", "%s", diff)
	}
	// Explicit: the settled hour's SUM must be 15, not 30.
	var settledValue float64 = -1
	for _, r := range cached {
		if r.WindowStart.Equal(from) {
			settledValue = r.Value
		}
	}
	s.Equal(float64(15), settledValue, "settled window SUM must not be doubled by concurrent populate")
}

// TestQueryCacheGateRouting asserts that queries the gate must NOT admit route
// to the live path (canQueryBeCached == false): minute windows, fractional-hour
// timezones, AVG, UNIQUE_COUNT, customer_id, stored-at filter, too-short span,
// and too-fresh range.
func (s *ConnectorTestSuite) TestQueryCacheGateRouting() {
	if s.T().Skipped() {
		return
	}
	s.enableCacheOnConnector()

	sumMeter := s.newMeter(parityMeter{name: "gate_sum", eventType: "gate_event", valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationSum})
	avgMeter := s.newMeter(parityMeter{name: "gate_avg", eventType: "gate_event", valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationAvg})
	uniqueMeter := s.newMeter(parityMeter{name: "gate_unique", eventType: "gate_event", valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationUniqueCount})
	dimMeter := s.newMeter(parityMeter{name: "gate_dim", eventType: "gate_event", valueProperty: ptr("$.value"), aggregation: meterpkg.MeterAggregationSum, groupBy: map[string]string{"region": "$.region"}})

	oldFrom := time.Now().UTC().Add(-30 * 24 * time.Hour)
	oldTo := time.Now().UTC().Add(-10 * 24 * time.Hour)

	kolkata, err := time.LoadLocation("Asia/Kolkata")
	s.NoError(err)

	cases := []struct {
		name   string
		meter  meterpkg.Meter
		params streaming.QueryParams
		want   bool
	}{
		{
			name:   "sum hour whole range -> cacheable",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   true,
		},
		{
			name:   "total with To set -> cacheable",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: nil},
			want:   true,
		},
		{
			name:   "total with To nil -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: nil, WindowSize: nil},
			want:   false,
		},
		{
			name:   "minute window -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeMinute)},
			want:   false,
		},
		{
			name:   "fractional-hour tz -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeDay), WindowTimeZone: kolkata},
			want:   false,
		},
		{
			name:   "avg -> live",
			meter:  avgMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "unique_count -> live",
			meter:  uniqueMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "not cachable flag -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: false, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "no from -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "stored-at filter -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour), FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Gte: ptr(oldFrom)}}},
			want:   false,
		},
		{
			name:   "customer_id group by -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour), GroupBy: []string{"customer_id"}},
			want:   false,
		},
		{
			name:   "too-fresh range -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: ptr(time.Now().UTC().Add(-30 * time.Minute)), To: ptr(time.Now().UTC()), WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			// Span (30m) below MinimumCacheableQueryPeriod (1h) while the range IS
			// settled — isolates the period guard from the age guard, so deleting
			// either one on its own turns some case red.
			name:   "settled but too-short span -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: ptr(time.Now().UTC().Add(-3 * time.Hour)), To: ptr(time.Now().UTC().Add(-150 * time.Minute)), WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			// Span (70m) above the minimum period while from is fresher than the
			// horizon — isolates the age guard from the period guard.
			name:   "long span but fresh from -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: ptr(time.Now().UTC().Add(-30 * time.Minute)), To: ptr(time.Now().UTC().Add(40 * time.Minute)), WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "windowed with To nil -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: nil, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "client id (progress tracking) -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, ClientID: ptr("client-1"), From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)},
			want:   false,
		},
		{
			name:   "customer filter -> live",
			meter:  sumMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour), FilterCustomer: []streaming.Customer{gateTestCustomer{}}},
			want:   false,
		},
		{
			name:   "filter on meter dimension -> cacheable",
			meter:  dimMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour), FilterGroupBy: map[string]filter.FilterString{"region": filterEq("us")}},
			want:   true,
		},
		{
			// The live path rejects filter keys that are not meter dimensions with a
			// validation error (including "subject"); routing to live preserves that.
			name:   "filter on unknown key -> live",
			meter:  dimMeter,
			params: streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour), FilterGroupBy: map[string]filter.FilterString{"subject": filterEq("s1")}},
			want:   false,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := s.Connector.canQueryBeCached(namespace, tc.meter, tc.params)
			s.Equal(tc.want, got, tc.name)
		})
	}

	// Decimal precision is required: the cache only serves the provably-exact
	// decimal mode. Toggle it off and confirm an otherwise-cacheable query routes
	// to live.
	s.Run("decimal precision off -> live", func() {
		s.Connector.config.EnableDecimalPrecision = false
		defer func() { s.Connector.config.EnableDecimalPrecision = true }()

		got := s.Connector.canQueryBeCached(namespace, sumMeter, streaming.QueryParams{
			Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour),
		})
		s.False(got, "decimal precision off must route to live")
	})

	// Feature gate: enabled for every namespace by default (nil checker or Noop
	// gate -> true); a gate backend that denies the namespace routes to live.
	cacheable := streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)}

	s.Run("feature gate nil (default) -> cacheable", func() {
		s.Connector.config.FeatureGate = nil
		s.True(s.Connector.canQueryBeCached(namespace, sumMeter, cacheable), "nil feature gate must default to enabled")
	})

	s.Run("feature gate noop -> cacheable", func() {
		s.Connector.config.FeatureGate = featuregate.NewFeatureGateChecker(featuregate.NewNoop(), nil, nil)
		defer func() { s.Connector.config.FeatureGate = nil }()
		s.True(s.Connector.canQueryBeCached(namespace, sumMeter, cacheable), "noop gate must default to enabled")
	})

	s.Run("feature gate denies namespace -> live", func() {
		s.Connector.config.FeatureGate = featuregate.NewFeatureGateChecker(alwaysFalseGate{}, nil, nil)
		defer func() { s.Connector.config.FeatureGate = nil }()
		s.False(s.Connector.canQueryBeCached(namespace, sumMeter, cacheable), "a denying feature gate must route to live")
	})
}

// alwaysFalseGate denies every namespace/flag — used to prove the feature gate
// can restrict the cache per namespace (the cloud posture).
type alwaysFalseGate struct{}

func (alwaysFalseGate) EvaluateBool(_, _ string, _ bool) (bool, error) { return false, nil }

// TestQueryCacheEndToEndPublicPath drives the fully-wired public QueryMeter with
// the cache enabled, against a query the gate admits (old range, hour window,
// decimal precision). It proves the whole path — canQueryBeCached → lazy
// populate-on-read → merge → scan — produces the same rows as the same query on
// a cache-disabled connector. This is the coverage the direct-merge harness
// (which bypasses the gate and QueryMeter) cannot give.
func (s *ConnectorTestSuite) TestQueryCacheEndToEndPublicPath() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "e2e_event"
	// Range must be older than the 1h horizon and span >= 1h; use a settled range
	// well in the past so cutoff (now-1h) is after `to` (all-cached).
	to := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Hour)
	from := to.Add(-6 * time.Hour)

	events := []parityEvent{
		{subject: "s1", at: from.Add(30 * time.Minute), data: `{"value": 10, "region": "us"}`},
		{subject: "s2", at: from.Add(2*time.Hour + 15*time.Minute), data: `{"value": 20, "region": "eu"}`},
		{subject: "s1", at: from.Add(4*time.Hour + 5*time.Minute), data: `{"value": 5, "region": "us"}`},
	}
	s.seedEvents(ctx, eventType, events)

	meter := s.newMeter(parityMeter{
		name: "e2e_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum, groupBy: map[string]string{"region": "$.region"},
	})

	// Sanity: the gate must admit this query.
	params := streaming.QueryParams{
		Cachable: true, From: &from, To: &to,
		WindowSize: ptr(meterpkg.WindowSizeHour), GroupBy: []string{"region"},
	}
	s.True(s.Connector.canQueryBeCached(namespace, meter, params), "query must be admitted by the gate")

	// Cache-enabled public path.
	cached, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)

	// Cache-disabled comparison: same connector with the flag off (live path).
	s.Connector.config.QueryCacheEnabled = false
	live, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)
	s.Connector.config.QueryCacheEnabled = true

	if diff := compareMeterRows(live, cached); diff != "" {
		s.Failf("end-to-end public path parity mismatch", "%s", diff)
	}
	s.NotEmpty(cached, "expected non-empty result")
}

// TestQueryCacheInvalidationOnLateEvent proves §4.3: inserting an event older
// than the freshness horizon invalidates (wipes) the affected namespace's cached
// rows, while a fresh event does not.
func (s *ConnectorTestSuite) TestQueryCacheInvalidationOnLateEvent() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "invalidation_event"
	from := time.Now().UTC().Add(-10 * 24 * time.Hour).Truncate(time.Hour)
	cutoff := from.Add(4 * time.Hour)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: from.Add(30 * time.Minute), data: `{"value": 10}`},
	})

	meter := s.newMeter(parityMeter{
		name: "invalidation_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	params := streaming.QueryParams{From: &from, To: ptr(from.Add(4 * time.Hour)), WindowSize: ptr(meterpkg.WindowSizeHour)}
	q := s.buildQueryMeter(meter, params, nil)
	s.NoError(s.Connector.populateMeterQueryRowCache(ctx, q, from, cutoff))
	s.Positive(s.countCacheRows(ctx), "cache should be populated")

	// A fresh event (now) must NOT invalidate (it does not mutate a settled window).
	s.NoError(s.Connector.BatchInsert(ctx, []streaming.RawEvent{{
		Namespace: namespace, ID: ulid.Make().String(), Time: time.Now().UTC(), Type: eventType,
		Source: "test", Subject: "s1", Data: `{"value": 1}`, IngestedAt: time.Now().UTC(), StoredAt: time.Now().UTC(),
	}}))
	s.Positive(s.countCacheRows(ctx), "fresh event must not invalidate the cache")

	// A late event (older than the horizon) MUST invalidate the namespace.
	s.NoError(s.Connector.BatchInsert(ctx, []streaming.RawEvent{{
		Namespace: namespace, ID: ulid.Make().String(), Time: from.Add(45 * time.Minute), Type: eventType,
		Source: "test", Subject: "s1", Data: `{"value": 99}`, IngestedAt: time.Now().UTC(), StoredAt: time.Now().UTC(),
	}}))
	// Lightweight DELETE is async-applied but mutations block until done by
	// default; give the count a moment via a direct read.
	s.Zero(s.countCacheRows(ctx), "late event must wipe the namespace's cache")
}

func (s *ConnectorTestSuite) countCacheRows(ctx context.Context) int {
	table := getTableName(s.Connector.config.Database, meterQueryRowCacheTable)
	rows, err := s.Connector.config.ClickHouse.Query(ctx, "SELECT count() FROM "+table+" WHERE namespace = ?", namespace)
	s.NoError(err)
	defer rows.Close()
	var count uint64
	for rows.Next() {
		s.NoError(rows.Scan(&count))
	}
	return int(count)
}

// TestQueryCacheOffByDefault asserts that with the cache disabled, QueryMeter
// takes the live path and no cache table is created.
func (s *ConnectorTestSuite) TestQueryCacheOffByDefault() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()

	// Fresh connector with cache OFF (the suite default).
	connector, err := New(ctx, Config{
		Logger:          slog.Default(),
		ClickHouse:      s.ClickHouse,
		Database:        s.Database,
		EventsTableName: "off_events",
		ProgressManager: progressmanager.NewMockProgressManager(),
	})
	s.NoError(err)

	// The cache table must NOT exist.
	table := getTableName(s.Database, meterQueryRowCacheTable)
	rows, err := s.ClickHouse.Query(ctx, "EXISTS TABLE "+table)
	s.NoError(err)
	var exists uint8
	for rows.Next() {
		s.NoError(rows.Scan(&exists))
	}
	rows.Close()
	s.Equal(uint8(0), exists, "cache table must not exist when cache is disabled")

	// canQueryBeCached must be false regardless of params.
	oldFrom := time.Now().UTC().Add(-30 * 24 * time.Hour)
	oldTo := time.Now().UTC().Add(-10 * 24 * time.Hour)
	meter := meterpkg.Meter{
		ManagedResource: models.ManagedResource{ID: ulid.Make().String(), NamespacedModel: models.NamespacedModel{Namespace: namespace}},
		Key:             "off_sum", EventType: "off_event", ValueProperty: ptr("$.value"), Aggregation: meterpkg.MeterAggregationSum,
	}
	s.False(connector.canQueryBeCached(namespace, meter, streaming.QueryParams{Cachable: true, From: &oldFrom, To: &oldTo, WindowSize: ptr(meterpkg.WindowSizeHour)}))
}

// gateTestCustomer is a minimal streaming.Customer for gate tests.
type gateTestCustomer struct{}

func (gateTestCustomer) GetUsageAttribution() streaming.CustomerUsageAttribution {
	return streaming.NewCustomerUsageAttribution("cus_1", nil, []string{"s1"})
}

// TestQueryCacheMeterShapeChange covers the meter-mutation axis of the cache
// key: meter definitions are mutable (UpdateMeter can add or remove group-by
// dimensions), and the group_by array is aligned to the CURRENT sorted paths.
// Rows written before and after a definition change must never combine into one
// aggregate — without the meter_hash fingerprint in the key and read filter,
// both shapes coexist under different sort keys and the cache leg double-counts
// every settled hour (SUM returns exactly 2x).
func (s *ConnectorTestSuite) TestQueryCacheMeterShapeChange() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "shape_event"
	base := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(4 * time.Hour)
	cutoff := base.Add(3 * time.Hour)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: base.Add(10 * time.Minute), data: `{"value": 10, "model": "gpt4", "region": "us"}`},
		{subject: "s1", at: base.Add(40 * time.Minute), data: `{"value": 20, "model": "gpt4", "region": "eu"}`},
		{subject: "s1", at: base.Add(80 * time.Minute), data: `{"value": 5, "model": "mini", "region": "us"}`},
	})

	// Shape v1: group by model only.
	meterV1 := s.newMeter(parityMeter{
		name: "shape_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum, groupBy: map[string]string{"model": "$.model"},
	})
	// Shape v2: SAME slug (definition updated), region dimension added.
	meterV2 := meterV1
	meterV2.GroupBy = map[string]string{"model": "$.model", "region": "$.region"}

	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour), GroupBy: []string{"model"}}

	// Populate BOTH shapes into the shared table (v1 first, then the "updated"
	// definition), with no truncate between — exactly what a meter update
	// followed by the next cached query produces.
	s.populateCache(ctx, meterV1, params, cutoff)
	s.populateCache(ctx, meterV2, params, cutoff)

	liveV2 := s.liveRows(ctx, meterV2, params)
	cachedV2 := s.readCached(ctx, meterV2, params, cutoff)
	if diff := compareMeterRows(liveV2, cachedV2); diff != "" {
		s.Failf("updated meter shape corrupted by pre-update rows", "%s", diff)
	}

	// The pre-update shape must also stay correct (its rows are keyed to its own
	// fingerprint).
	liveV1 := s.liveRows(ctx, meterV1, params)
	cachedV1 := s.readCached(ctx, meterV1, params, cutoff)
	if diff := compareMeterRows(liveV1, cachedV1); diff != "" {
		s.Failf("pre-update meter shape corrupted by post-update rows", "%s", diff)
	}
}

// TestQueryCachePopulateFailureFallsBackToLive covers the populate failure
// path: the settled range is served exclusively from the cache table, so a
// failed populate must NOT proceed to the merge (which would silently drop the
// whole settled range and undercount). The connector must fall back to the full
// live query and still return the correct result. Dropping the cache table
// makes the populate INSERT fail deterministically.
func (s *ConnectorTestSuite) TestQueryCachePopulateFailureFallsBackToLive() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "fallback_event"
	to := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	from := to.Add(-6 * time.Hour)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: from.Add(30 * time.Minute), data: `{"value": 10}`},
		{subject: "s1", at: from.Add(3 * time.Hour), data: `{"value": 20}`},
	})

	meter := s.newMeter(parityMeter{
		name: "fallback_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	params := streaming.QueryParams{Cachable: true, From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	s.True(s.Connector.canQueryBeCached(namespace, meter, params), "query must be admitted by the gate")

	// Live control BEFORE breaking the cache table.
	live, err := s.Connector.QueryMeter(ctx, namespace, meter, streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)})
	s.NoError(err)
	s.NotEmpty(live)

	// Break population: the cached path must fall back to live, not undercount.
	table := getTableName(s.Connector.config.Database, meterQueryRowCacheTable)
	s.NoError(s.Connector.config.ClickHouse.Exec(ctx, "DROP TABLE "+table+" SYNC"))

	cached, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err, "populate failure must not fail the query")
	if diff := compareMeterRows(live, cached); diff != "" {
		s.Failf("populate-failure fallback must serve the full live result", "%s", diff)
	}

	// The failed populate must not have claimed coverage: a claim over rows
	// that were never written would make the next read skip population and
	// serve the settled range as empty.
	s.Zero(s.countCoverageRows(ctx, meter.Key), "no coverage claim may be stored after a failed populate")

	// Restore the table for any later suite activity.
	s.NoError(s.Connector.createMeterQueryRowCacheTable(ctx))
}

// TestQueryCacheNewestRowWins pins the read-time collapse semantics: when the
// write-once invariant is violated (a populate racing the late-event
// invalidation DELETE persists a stale row alongside a later corrected row),
// the cache leg must deterministically serve the NEWEST row — including when
// the newest value is legitimately NULL, which a bare argMax would skip in
// favor of the stale non-NULL one.
func (s *ConnectorTestSuite) TestQueryCacheNewestRowWins() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "newest_event"
	base := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(2 * time.Hour)
	cutoff := to // all-cached: the settled range is served purely from the table

	meter := s.newMeter(parityMeter{
		name: "newest_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	table := getTableName(s.Connector.config.Database, meterQueryRowCacheTable)
	hash := meterShapeHash(meter)

	insert := func(ws time.Time, sum *int64, createdAt string) {
		var sumSQL string
		if sum == nil {
			sumSQL = "NULL"
		} else {
			sumSQL = fmt.Sprintf("toDecimal128(%d, 19)", *sum)
		}
		s.NoError(s.Connector.config.ClickHouse.Exec(ctx, fmt.Sprintf(
			`INSERT INTO %s (namespace, type, meter_slug, meter_hash, windowstart, subject, group_by, sum_value, count_value, min_value, max_value, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, [], %s, 1, NULL, NULL, toDateTime64(?, 3))`, table, sumSQL),
			namespace, eventType, meter.Key, hash, ws, "s1", createdAt))
	}

	// Hour 0: stale row (100) then corrected row (60) 1ms later -> 60 must win.
	insert(base, ptr(int64(100)), "2026-12-01 12:00:00.001")
	insert(base, ptr(int64(60)), "2026-12-01 12:00:00.002")
	// Hour 1: stale non-NULL (7) then corrected NULL -> the row must vanish
	// (NULL value rows are skipped on scan, matching live).
	insert(base.Add(time.Hour), ptr(int64(7)), "2026-12-01 12:00:00.001")
	insert(base.Add(time.Hour), nil, "2026-12-01 12:00:00.002")

	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	rows := s.readCached(ctx, meter, params, cutoff)

	s.Require().Len(rows, 1, "hour 1's newest row is NULL and must be skipped; expected only hour 0. rows=%s", dumpRows(rows))
	s.Equal(base, rows[0].WindowStart.UTC())
	s.Equal(float64(60), rows[0].Value, "the NEWEST stored rollup must win, not the stale one")
}

// TestQueryCacheDSTTimezoneParity compares cached vs live for a gate-admitted
// whole-hour DST timezone across the US spring-forward transition (2026-03-08
// in America/New_York): day windows are 23 hours long that day and month
// windows span the offset change, exercising the re-tumbling of hourly-UTC
// rollup rows into tz-local windows.
func (s *ConnectorTestSuite) TestQueryCacheDSTTimezoneParity() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	ny, err := time.LoadLocation("America/New_York")
	s.Require().NoError(err)

	eventType := "dst_event"
	// [Mar 6 .. Mar 10) UTC, hour-aligned: spans the Mar 8 02:00 EST -> 03:00 EDT jump.
	from := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: time.Date(2026, 3, 6, 12, 30, 0, 0, time.UTC), data: `{"value": 1, "region": "us"}`},
		{subject: "s1", at: time.Date(2026, 3, 8, 6, 15, 0, 0, time.UTC), data: `{"value": 2, "region": "us"}`}, // 01:15 EST, pre-jump
		{subject: "s1", at: time.Date(2026, 3, 8, 7, 45, 0, 0, time.UTC), data: `{"value": 4, "region": "eu"}`}, // 03:45 EDT, post-jump
		{subject: "s2", at: time.Date(2026, 3, 8, 23, 5, 0, 0, time.UTC), data: `{"value": 8, "region": "us"}`}, // 19:05 EDT
		{subject: "s1", at: time.Date(2026, 3, 9, 15, 0, 0, 0, time.UTC), data: `{"value": 16, "region": "us"}`},
	})

	meter := s.newMeter(parityMeter{
		name: "dst_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum, groupBy: map[string]string{"region": "$.region"},
	})

	windowSizes := []*meterpkg.WindowSize{ptr(meterpkg.WindowSizeHour), ptr(meterpkg.WindowSizeDay), ptr(meterpkg.WindowSizeMonth)}
	cutoffs := []time.Time{from, time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC), to} // all-fresh, mid-jump, all-cached

	for _, ws := range windowSizes {
		for _, groupBy := range [][]string{nil, {"region"}, {"subject"}} {
			params := streaming.QueryParams{From: &from, To: &to, WindowSize: ws, WindowTimeZone: ny, GroupBy: groupBy}
			live := s.liveRows(ctx, meter, params)

			for _, cutoff := range cutoffs {
				name := fmt.Sprintf("%s/groupby=%v/cutoff=%s", windowSizeName(ws), groupBy, cutoff.Sub(from))
				s.Run(name, func() {
					s.truncateCacheTable(ctx)
					cached := s.cachedRows(ctx, meter, params, cutoff)
					if diff := compareMeterRows(live, cached); diff != "" {
						s.Failf("DST timezone parity mismatch", "%s:\n%s", name, diff)
					}
				})
			}
		}
	}
}

// TestQueryCacheShadowParityCheck pins the production correctness net: the
// shadow verifier must report clean cache state as a match, detect a corrupted
// cached row as a mismatch, and self-heal by invalidating the namespace so the
// next query repopulates from raw data.
func (s *ConnectorTestSuite) TestQueryCacheShadowParityCheck() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "shadow_event"
	base := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(3 * time.Hour)
	cutoff := to // all-cached: corruption in the table is fully visible to the merge

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: base.Add(20 * time.Minute), data: `{"value": 10}`},
		{subject: "s1", at: base.Add(100 * time.Minute), data: `{"value": 5}`},
	})

	meter := s.newMeter(parityMeter{
		name: "shadow_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	params := streaming.QueryParams{From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}
	q := s.buildQueryMeter(meter, params, nil)

	// Clean state: populate + read, the verifier must report a match.
	cached := s.cachedRows(ctx, meter, params, cutoff)
	s.False(s.Connector.verifyCachedResultParity(ctx, q, cached), "clean cache must verify as a match")

	// Corrupt the cached hour 0 with a newer, wrong rollup (newest-wins makes the
	// corruption deterministic) and re-read: the served result is now wrong, and
	// the verifier must catch it and wipe the namespace.
	table := getTableName(s.Connector.config.Database, meterQueryRowCacheTable)
	s.NoError(s.Connector.config.ClickHouse.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (namespace, type, meter_slug, meter_hash, windowstart, subject, group_by, sum_value, count_value, min_value, max_value, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, [], toDecimal128(999, 19), 1, NULL, NULL, now64(3) + toIntervalHour(1))`, table),
		namespace, eventType, meter.Key, meterShapeHash(meter), base, "s1"))

	corrupted := s.readCached(ctx, meter, params, cutoff)
	s.True(s.Connector.verifyCachedResultParity(ctx, q, corrupted), "corrupted cache must verify as a mismatch")
	s.Zero(s.countCacheRows(ctx), "a parity mismatch must self-heal by invalidating the namespace cache")

	// After self-healing, the next populate + read verifies clean again.
	healed := s.cachedRows(ctx, meter, params, cutoff)
	s.False(s.Connector.verifyCachedResultParity(ctx, q, healed), "repopulated cache must verify as a match again")
}

// TestQueryCacheTelemetry proves the observability wiring actually emits: with
// a recording tracer and an in-memory metric reader attached, a cache-served
// query must produce the query/populate spans and the queries counter, and a
// parity check must produce its span with the outcome attribute. This guards
// the "monitor correctness in production" contract — a silently detached
// instrument would otherwise look identical to a healthy quiet system.
func (s *ConnectorTestSuite) TestQueryCacheTelemetry() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	// Attach a recording tracer + in-memory metrics to the suite connector.
	spanRecorder := tracetest.NewSpanRecorder()
	s.Connector.tracer = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder)).Tracer("test")

	metricReader := sdkmetric.NewManualReader()
	metrics, err := newQueryCacheMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader)).Meter("test"))
	s.Require().NoError(err)
	s.Connector.queryCacheMetrics = metrics

	defer func() {
		s.Connector.tracer = tracenoop.NewTracerProvider().Tracer("test")
		s.Connector.queryCacheMetrics = nil
	}()

	eventType := "telemetry_event"
	to := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Hour)
	from := to.Add(-4 * time.Hour)
	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: from.Add(30 * time.Minute), data: `{"value": 10}`},
	})

	meter := s.newMeter(parityMeter{
		name: "telemetry_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})
	params := streaming.QueryParams{Cachable: true, From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}

	// Cache-served query through the public path.
	rows, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)
	s.NotEmpty(rows)

	// Parity check (synchronous call, same as the sampled shadow goroutine runs).
	q := s.buildQueryMeter(meter, params, nil)
	s.False(s.Connector.verifyCachedResultParity(ctx, q, rows))

	// Spans: query + populate from the cached query, parity_check from the verify.
	spanNames := map[string]bool{}
	for _, span := range spanRecorder.Ended() {
		spanNames[span.Name()] = true
	}
	s.True(spanNames["streaming.query_cache.query"], "cached query span missing; got %v", spanNames)
	s.True(spanNames["streaming.query_cache.populate"], "populate span missing; got %v", spanNames)
	s.True(spanNames["streaming.query_cache.parity_check"], "parity check span missing; got %v", spanNames)

	// Metrics: the queries counter and both duration histograms must have data.
	var rm metricdata.ResourceMetrics
	s.Require().NoError(metricReader.Collect(ctx, &rm))
	metricNames := map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			metricNames[m.Name] = true
		}
	}
	s.True(metricNames["streaming.query_cache.queries"], "queries counter missing; got %v", metricNames)
	s.True(metricNames["streaming.query_cache.query_duration_ms"], "query duration histogram missing; got %v", metricNames)
	s.True(metricNames["streaming.query_cache.populate_duration_ms"], "populate duration histogram missing; got %v", metricNames)
	s.True(metricNames["streaming.query_cache.parity_checks"], "parity checks counter missing; got %v", metricNames)
}

// TestQueryCacheCrossTimezoneSharing proves cached rows are shared safely
// across timezones — the reason WindowTimeZone is deliberately NOT part of the
// cache key. The rollup stores timezone-agnostic hourly-UTC partials; the
// query's timezone only re-windows them at read time, and the gate admits only
// whole-hour-offset zones, where every tz-local window boundary (including DST
// transition instants) falls on a UTC hour boundary.
//
// Two invariants are pinned:
//  1. Populate output is byte-identical regardless of the querying timezone —
//     if someone ever makes populate tz-aware, cross-tz sharing breaks and this
//     fails.
//  2. Rows populated ONCE serve UTC, America/New_York and Europe/Budapest
//     queries (hour/day/month windows, mid and all-cached cutoffs) with exact
//     live parity, across the 2026 US spring-forward, EU spring-forward and US
//     fall-back transitions (the fall-back range contains the repeated local
//     hour: two distinct UTC hours labeled 01:xx locally).
func (s *ConnectorTestSuite) TestQueryCacheCrossTimezoneSharing() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	ny, err := time.LoadLocation("America/New_York")
	s.Require().NoError(err)
	budapest, err := time.LoadLocation("Europe/Budapest")
	s.Require().NoError(err)

	eventType := "crosstz_event"

	ranges := []struct {
		name string
		from time.Time
		to   time.Time
	}{
		// US spring forward 2026-03-08 (07:00Z): NY day is 23h.
		{name: "us-spring", from: time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), to: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
		// EU spring forward 2026-03-29 (01:00Z): Budapest day is 23h.
		{name: "eu-spring", from: time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC), to: time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)},
		// US fall back 2026-11-01 (06:00Z): NY day is 25h, local 01:xx occurs twice.
		{name: "us-fall", from: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), to: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)},
	}

	// Events land around each transition instant, on both sides of it, plus the
	// repeated local hour on the fall-back day (05:30Z and 06:30Z are both
	// "01:30" in New York on 2026-11-01, in different UTC hour buckets).
	events := []parityEvent{
		{subject: "s1", at: time.Date(2026, 3, 6, 12, 30, 0, 0, time.UTC), data: `{"value": 1, "region": "us"}`},
		{subject: "s1", at: time.Date(2026, 3, 8, 6, 15, 0, 0, time.UTC), data: `{"value": 2, "region": "us"}`},
		{subject: "s2", at: time.Date(2026, 3, 8, 7, 45, 0, 0, time.UTC), data: `{"value": 4, "region": "eu"}`},
		{subject: "s1", at: time.Date(2026, 3, 29, 0, 30, 0, 0, time.UTC), data: `{"value": 8, "region": "eu"}`},
		{subject: "s2", at: time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC), data: `{"value": 16, "region": "eu"}`},
		{subject: "s1", at: time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC), data: `{"value": 32, "region": "us"}`},
		{subject: "s2", at: time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC), data: `{"value": 64, "region": "us"}`},
		{subject: "s1", at: time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC), data: `{"value": 128, "region": "eu"}`},
	}
	s.seedEvents(ctx, eventType, events)

	meter := s.newMeter(parityMeter{
		name: "crosstz_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum, groupBy: map[string]string{"region": "$.region"},
	})

	// Invariant 1: populate under a New York query and under a UTC query must
	// write byte-identical rows.
	r := ranges[0]
	nyParams := streaming.QueryParams{From: &r.from, To: &r.to, WindowSize: ptr(meterpkg.WindowSizeDay), WindowTimeZone: ny}
	utcParams := streaming.QueryParams{From: &r.from, To: &r.to, WindowSize: ptr(meterpkg.WindowSizeDay)}

	s.truncateCacheTable(ctx)
	s.populateCache(ctx, meter, nyParams, r.to)
	nySnapshot := s.snapshotCacheRows(ctx)

	s.truncateCacheTable(ctx)
	s.populateCache(ctx, meter, utcParams, r.to)
	utcSnapshot := s.snapshotCacheRows(ctx)

	s.Require().NotEmpty(nySnapshot)
	s.Equal(utcSnapshot, nySnapshot, "populate must write identical rows regardless of the query timezone")

	// Invariant 2: rows populated once serve every admitted timezone with live
	// parity, across all three DST transitions. No re-population between reads.
	timezones := []*time.Location{nil, ny, budapest} // nil = UTC
	windowSizes := []*meterpkg.WindowSize{ptr(meterpkg.WindowSizeHour), ptr(meterpkg.WindowSizeDay), ptr(meterpkg.WindowSizeMonth)}

	for _, r := range ranges {
		s.truncateCacheTable(ctx)
		s.populateCache(ctx, meter, streaming.QueryParams{From: &r.from, To: &r.to, WindowSize: ptr(meterpkg.WindowSizeHour)}, r.to)

		mid := r.from.Add(r.to.Sub(r.from) / 2).Truncate(time.Hour)
		for _, tz := range timezones {
			for _, ws := range windowSizes {
				for _, cutoff := range []time.Time{mid, r.to} {
					params := streaming.QueryParams{From: &r.from, To: &r.to, WindowSize: ws, WindowTimeZone: tz, GroupBy: []string{"region"}}
					name := fmt.Sprintf("%s/tz=%v/%s/cutoff=%s", r.name, tz, windowSizeName(ws), cutoff.Sub(r.from))
					s.Run(name, func() {
						live := s.liveRows(ctx, meter, params)
						cached := s.readCached(ctx, meter, params, cutoff)
						if diff := compareMeterRows(live, cached); diff != "" {
							s.Failf("cross-timezone cache sharing parity mismatch", "%s:\n%s", name, diff)
						}
					})
				}
			}
		}
	}
}

// TestQueryCacheCoverageSkipsRepopulation pins the gap-tracking contract on
// the public path: a repeat read over a range whose coverage claim exists must
// NOT re-scan raw events. The only external observation of "did it
// re-populate" is mutating the raw data without invalidating the cache
// (bypassing BatchInsert) and checking whether the mutation leaks into the
// next cached result: it must not — until a real late event arrives through
// BatchInsert, whose invalidation must wipe the claim along with the rows and
// force a full, mutation-visible repopulation.
func (s *ConnectorTestSuite) TestQueryCacheCoverageSkipsRepopulation() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "coverage_skip_event"
	to := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Hour)
	from := to.Add(-6 * time.Hour)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: from.Add(30 * time.Minute), data: `{"value": 10}`},
		{subject: "s1", at: from.Add(3 * time.Hour), data: `{"value": 20}`},
	})

	meter := s.newMeter(parityMeter{
		name: "coverage_skip_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})
	params := streaming.QueryParams{Cachable: true, From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}

	// Age out the marker the seeding BatchInsert wrote, so the claim below is
	// honored immediately instead of being distrusted for the skew margin.
	s.clearInvalidationMarkers(ctx)

	// First read: populates and claims coverage; must match live.
	first, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)
	if diff := compareMeterRows(s.liveRows(ctx, meter, params), first); diff != "" {
		s.Failf("first cached read must match live", "%s", diff)
	}

	// Mutate the settled raw data WITHOUT invalidation. The live result now
	// includes the mutation (sanity check that it is a real discriminator)...
	s.insertRawEventBypassingInvalidation(ctx, eventType, "s1", from.Add(90*time.Minute), `{"value": 999}`)
	s.NotEmpty(compareMeterRows(first, s.liveRows(ctx, meter, params)), "the raw mutation must be visible to a live scan")

	// ...but the covered repeat read must serve the existing rollups untouched.
	second, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)
	if diff := compareMeterRows(first, second); diff != "" {
		s.Failf("covered repeat read re-populated (raw mutation leaked into the cache)", "%s", diff)
	}

	// A real late event goes through BatchInsert: its invalidation must wipe
	// the claim with the rows, so the next read repopulates and sees BOTH the
	// mutation and the late event — full live parity again.
	s.NoError(s.Connector.BatchInsert(ctx, []streaming.RawEvent{{
		Namespace: namespace, ID: ulid.Make().String(), Time: from.Add(150 * time.Minute), Type: eventType,
		Source: "test", Subject: "s1", Data: `{"value": 7}`, IngestedAt: time.Now().UTC(), StoredAt: time.Now().UTC(),
	}}))
	s.Zero(s.countCoverageRows(ctx, meter.Key), "late-event invalidation must wipe the coverage claim")

	third, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)
	if diff := compareMeterRows(s.liveRows(ctx, meter, params), third); diff != "" {
		s.Failf("post-invalidation read must repopulate to full live parity", "%s", diff)
	}
}

// TestQueryCacheCoverageWidensRange covers the partial-coverage read: a wide
// query over a range whose middle is already claimed must populate ONLY the
// missing part. Both directions are asserted through raw-data mutations:
// the gap's events must appear (the gap was populated) while a mutation inside
// the already-covered part must not (that part was served from the existing
// rollups).
func (s *ConnectorTestSuite) TestQueryCacheCoverageWidensRange() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "coverage_widen_event"
	to := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Hour)
	from := to.Add(-96 * time.Hour)
	narrowFrom := from.Add(48 * time.Hour)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: from.Add(90 * time.Minute), data: `{"value": 1}`}, // early (gap) region
		{subject: "s1", at: from.Add(30 * time.Hour), data: `{"value": 2}`},   // early (gap) region
		{subject: "s1", at: narrowFrom.Add(2 * time.Hour), data: `{"value": 4}`},
		{subject: "s1", at: narrowFrom.Add(32 * time.Hour), data: `{"value": 8}`},
	})

	meter := s.newMeter(parityMeter{
		name: "coverage_widen_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	// Age out the marker the seeding BatchInsert wrote (see clearInvalidationMarkers).
	s.clearInvalidationMarkers(ctx)

	total := func(f time.Time) float64 {
		params := streaming.QueryParams{Cachable: true, From: &f, To: &to}
		rows, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
		s.Require().NoError(err)
		s.Require().Len(rows, 1)
		return rows[0].Value
	}

	// Narrow read claims [narrowFrom, to): 4 + 8.
	s.Equal(float64(12), total(narrowFrom))

	// Mutate raw data inside the covered narrow range, without invalidation.
	s.insertRawEventBypassingInvalidation(ctx, eventType, "s1", narrowFrom.Add(4*time.Hour), `{"value": 1000}`)

	// Wide read: the early gap must be populated (1 + 2 appear) while the
	// covered part is served from the existing rollups (1000 must not appear).
	// 15 proves both at once: 1012 would mean the covered part re-populated,
	// 12 would mean the gap was skipped.
	s.Equal(float64(15), total(from))

	// Fully covered now: the repeat wide read is stable.
	s.Equal(float64(15), total(from))
}

// TestQueryCacheCoverageTiedClaimsStayAtomic pins the coverage read's
// tuple-atomicity: two claims for the same meter written with the SAME
// created_at (racing disjoint first-claims can land in the same millisecond)
// must resolve to ONE of the stored intervals — never a stitched interval
// taking covered_from from one row and covered_until from the other, which
// would assert coverage over the unpopulated gap between them.
func (s *ConnectorTestSuite) TestQueryCacheCoverageTiedClaimsStayAtomic() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	meter := s.newMeter(parityMeter{
		name: "coverage_tie_sum", eventType: "coverage_tie_event", valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})

	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	claimA := cacheCoverage{From: base, Until: base.Add(10 * time.Hour), FirstWrittenAt: base, PopulatedAt: base}
	claimB := cacheCoverage{From: base.Add(12 * time.Hour), Until: base.Add(20 * time.Hour), FirstWrittenAt: base, PopulatedAt: base}

	table := getTableName(s.Connector.config.Database, meterQueryRowCacheCoverageTable)
	for _, claim := range []cacheCoverage{claimA, claimB} {
		s.Require().NoError(s.Connector.config.ClickHouse.Exec(ctx, fmt.Sprintf(
			`INSERT INTO %s (namespace, meter_slug, meter_hash, covered_from, covered_until, first_written_at, populated_at, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, toDateTime64('2026-06-02 00:00:00.000', 3))`, table),
			namespace, meter.Key, meterShapeHash(meter), claim.From, claim.Until, claim.FirstWrittenAt, claim.PopulatedAt))
	}

	q := s.buildQueryMeter(meter, streaming.QueryParams{}, nil)
	got, _, err := s.Connector.readMeterQueryRowCacheCoverage(ctx, q)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	isA := got.From.Equal(claimA.From) && got.Until.Equal(claimA.Until)
	isB := got.From.Equal(claimB.From) && got.Until.Equal(claimB.Until)
	s.True(isA || isB, "tied claims must resolve to one stored interval, got stitched [%s, %s)", got.From, got.Until)
}

// TestQueryCacheCoverageInvalidationBeatsRacingClaim pins the fix for the
// claim/invalidation race: a cached read that planned BEFORE an invalidation
// can commit its coverage claim AFTER the invalidation's deletes (an INSERT
// cannot be ordered against a concurrent DELETE), leaving a claim that vouches
// for wiped rollups. The invalidation marker must make every read distrust
// such a claim — its populated_at predates the marker — so the next read
// repopulates instead of serving the settled range as empty.
func (s *ConnectorTestSuite) TestQueryCacheCoverageInvalidationBeatsRacingClaim() {
	if s.T().Skipped() {
		return
	}
	ctx := s.T().Context()
	s.enableCacheOnConnector()

	eventType := "coverage_race_event"
	to := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Hour)
	from := to.Add(-4 * time.Hour)

	s.seedEvents(ctx, eventType, []parityEvent{
		{subject: "s1", at: from.Add(30 * time.Minute), data: `{"value": 10}`},
	})

	meter := s.newMeter(parityMeter{
		name: "coverage_race_sum", eventType: eventType, valueProperty: ptr("$.value"),
		aggregation: meterpkg.MeterAggregationSum,
	})
	params := streaming.QueryParams{Cachable: true, From: &from, To: &to, WindowSize: ptr(meterpkg.WindowSizeHour)}

	// Normal first read: populates and claims.
	_, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)

	// The racing reader's plan starts HERE — before the invalidation below.
	stalePlanStart := time.Now().UTC()

	// A late event arrives through the real path: the invalidation inserts the
	// namespace marker and wipes the rollups + claims.
	s.NoError(s.Connector.BatchInsert(ctx, []streaming.RawEvent{{
		Namespace: namespace, ID: ulid.Make().String(), Time: from.Add(90 * time.Minute), Type: eventType,
		Source: "test", Subject: "s1", Data: `{"value": 5}`, IngestedAt: time.Now().UTC(), StoredAt: time.Now().UTC(),
	}}))

	// The racing reader now lands its claim AFTER the invalidation — the exact
	// interleaving the marker exists for. Its row is the newest in the table.
	q := s.buildQueryMeter(meter, params, nil)
	s.NoError(s.Connector.storeMeterQueryRowCacheCoverage(ctx, q, cacheCoverage{
		From: from, Until: to, FirstWrittenAt: stalePlanStart, PopulatedAt: stalePlanStart,
	}))

	// The next read must distrust the resurrected claim (its populated_at
	// predates the marker), repopulate, and serve the full result including
	// the late event — never the wiped range as empty.
	got, err := s.Connector.QueryMeter(ctx, namespace, meter, params)
	s.NoError(err)
	s.NotEmpty(got, "the settled range must not be served from the wiped cache as empty")
	if diff := compareMeterRows(s.liveRows(ctx, meter, params), got); diff != "" {
		s.Failf("read after a racing claim resurrection must repopulate to live parity", "%s", diff)
	}
}

// insertRawEventBypassingInvalidation writes an event straight into the events
// table with raw SQL, skipping BatchInsert's late-event cache invalidation.
// Coverage tests use it to mutate settled raw data WITHOUT wiping the cache —
// the only external way to observe whether a read re-populated (the mutation
// shows up) or served the existing rollups (it does not).
func (s *ConnectorTestSuite) insertRawEventBypassingInvalidation(ctx context.Context, eventType, subject string, at time.Time, data string) {
	table := getTableName(s.Connector.config.Database, s.Connector.config.EventsTableName)
	s.Require().NoError(s.Connector.config.ClickHouse.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (namespace, id, type, subject, source, time, data, ingested_at, stored_at, store_row_id)
		 VALUES (?, ?, ?, ?, 'coverage-test', ?, ?, ?, ?, '')`, table),
		namespace, ulid.Make().String(), eventType, subject, at, data, at, at))
}

// clearInvalidationMarkers removes the namespace's invalidation markers.
// seedEvents ingests old events through BatchInsert, which legitimately
// invalidates the namespace — and claims written within the clock-skew margin
// of a marker are deliberately distrusted (one redundant populate in
// production). Tests that assert a covered read does NOT repopulate would see
// that redundant populate as a failure, so they age the seeding marker out
// first, as if the seed ingestion happened long ago.
func (s *ConnectorTestSuite) clearInvalidationMarkers(ctx context.Context) {
	table := getTableName(s.Connector.config.Database, meterQueryRowCacheCoverageTable)
	s.Require().NoError(s.Connector.config.ClickHouse.Exec(ctx,
		"DELETE FROM "+table+" WHERE namespace = ? AND meter_slug = ''", namespace))
}

func (s *ConnectorTestSuite) countCoverageRows(ctx context.Context, meterSlug string) int {
	table := getTableName(s.Connector.config.Database, meterQueryRowCacheCoverageTable)
	rows, err := s.Connector.config.ClickHouse.Query(ctx,
		"SELECT count() FROM "+table+" WHERE namespace = ? AND meter_slug = ?", namespace, meterSlug)
	s.NoError(err)
	defer rows.Close()
	var count uint64
	for rows.Next() {
		s.NoError(rows.Scan(&count))
	}
	return int(count)
}

// snapshotCacheRows returns a canonical dump of every cache row except
// created_at (which legitimately differs between populates).
func (s *ConnectorTestSuite) snapshotCacheRows(ctx context.Context) []string {
	table := getTableName(s.Connector.config.Database, meterQueryRowCacheTable)
	rows, err := s.Connector.config.ClickHouse.Query(ctx, `
		SELECT namespace, type, meter_slug, meter_hash, toString(windowstart), subject,
		       toString(group_by), toString(sum_value), toString(count_value),
		       toString(min_value), toString(max_value)
		FROM `+table+`
		ORDER BY namespace, type, meter_slug, meter_hash, windowstart, subject, group_by`)
	s.Require().NoError(err)
	defer rows.Close()

	var out []string
	for rows.Next() {
		cols := make([]string, 11)
		dest := make([]interface{}, len(cols))
		for i := range cols {
			dest[i] = &cols[i]
		}
		s.Require().NoError(rows.Scan(dest...))
		out = append(out, fmt.Sprintf("%v", cols))
	}
	s.Require().NoError(rows.Err())
	return out
}
