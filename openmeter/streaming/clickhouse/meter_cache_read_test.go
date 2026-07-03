package clickhouse

import (
	"math/rand"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestMeterCacheBounds(t *testing.T) {
	parse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return ts
	}

	refreshStart := parse("2025-03-01T12:10:30Z")
	age := time.Hour

	tests := []struct {
		name string

		from  *time.Time
		to    time.Time
		grain CacheGrain

		wantLo *time.Time
		wantHi time.Time
		wantOK bool
	}{
		{
			// horizon = 11:10:30 > to, so to governs: floor(to) − 1h
			name:   "on grid from and to below horizon",
			from:   lo.ToPtr(parse("2025-03-01T00:00:00Z")),
			to:     parse("2025-03-01T06:00:00Z"),
			grain:  CacheGrainHour,
			wantLo: lo.ToPtr(parse("2025-03-01T00:00:00Z")),
			wantHi: parse("2025-03-01T05:00:00Z"),
			wantOK: true,
		},
		{
			name:   "off grid from is ceiled and off grid to floored",
			from:   lo.ToPtr(parse("2025-03-01T00:12:00Z")),
			to:     parse("2025-03-01T06:45:00Z"),
			grain:  CacheGrainHour,
			wantLo: lo.ToPtr(parse("2025-03-01T01:00:00Z")),
			wantHi: parse("2025-03-01T05:00:00Z"),
			wantOK: true,
		},
		{
			// G5 epsilon at the horizon: horizon 11:10:30 floors to 11:00, minus one
			// grain = 10:00 — bucket [10:00, 11:00) is never served even though the
			// refresh may have computed it, because refreshStart is an estimate.
			name:   "to above horizon is capped with one grain epsilon",
			from:   lo.ToPtr(parse("2025-03-01T00:00:00Z")),
			to:     parse("2025-03-01T12:00:00Z"),
			grain:  CacheGrainHour,
			wantLo: lo.ToPtr(parse("2025-03-01T00:00:00Z")),
			wantHi: parse("2025-03-01T10:00:00Z"),
			wantOK: true,
		},
		{
			// exact-horizon variant: refreshStart − age exactly on grid still backs off a
			// full grain
			name:   "on grid horizon still backs off one grain",
			from:   lo.ToPtr(parse("2025-03-01T00:00:00Z")),
			to:     parse("2025-03-01T11:10:30Z"),
			grain:  CacheGrainHour,
			wantLo: lo.ToPtr(parse("2025-03-01T00:00:00Z")),
			wantHi: parse("2025-03-01T10:00:00Z"),
			wantOK: true,
		},
		{
			name:   "nil from is unbounded below",
			from:   nil,
			to:     parse("2025-03-01T06:00:00Z"),
			grain:  CacheGrainHour,
			wantLo: nil,
			wantHi: parse("2025-03-01T05:00:00Z"),
			wantOK: true,
		},
		{
			name:   "zero width range is fully live",
			from:   lo.ToPtr(parse("2025-03-01T05:10:00Z")),
			to:     parse("2025-03-01T06:00:00Z"),
			grain:  CacheGrainHour,
			wantOK: false,
		},
		{
			name:   "range entirely above horizon is fully live",
			from:   lo.ToPtr(parse("2025-03-01T11:30:00Z")),
			to:     parse("2025-03-01T12:00:00Z"),
			grain:  CacheGrainHour,
			wantOK: false,
		},
		{
			name:   "day grain floors to utc midnight",
			from:   lo.ToPtr(parse("2025-02-01T05:00:00Z")),
			to:     parse("2025-02-20T00:00:00Z"),
			grain:  CacheGrainDay,
			wantLo: lo.ToPtr(parse("2025-02-02T00:00:00Z")),
			wantHi: parse("2025-02-19T00:00:00Z"),
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounds, ok, err := meterCacheBounds(tt.from, tt.to, refreshStart, age, tt.grain)
			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)

			if !tt.wantOK {
				return
			}

			if tt.wantLo == nil {
				assert.Nil(t, bounds.CacheLo)
			} else {
				require.NotNil(t, bounds.CacheLo)
				assert.True(t, bounds.CacheLo.Equal(*tt.wantLo), "cacheLo %s != %s", bounds.CacheLo, tt.wantLo)
			}

			assert.True(t, bounds.CacheHi.Equal(tt.wantHi), "cacheHi %s != %s", bounds.CacheHi, tt.wantHi)
		})
	}

	t.Run("invalid grain", func(t *testing.T) {
		_, _, err := meterCacheBounds(nil, refreshStart, refreshStart, age, CacheGrain("week"))
		require.Error(t, err)
	})
}

// TestMeterCacheBoundsTilingProperty asserts the R4 tiling invariant on randomized
// inputs: whenever the bounds admit a cache leg, the three legs
// [from, cacheLo) ∪ [cacheLo, cacheHi) ∪ [cacheHi, to) tile [from, to) exactly — shared
// endpoints, aligned cache bucket bounds, non-empty cache leg, always-live tail — and
// whenever they do not, there genuinely is no whole settled bucket to serve.
func TestMeterCacheBoundsTilingProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	grains := []CacheGrain{CacheGrainMinute, CacheGrainHour, CacheGrainDay}
	base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 2000; i++ {
		grain := grains[rng.Intn(len(grains))]
		spec, err := grainSpecFor(grain)
		require.NoError(t, err)

		grainDuration := time.Duration(spec.seconds) * time.Second

		from := base.Add(time.Duration(rng.Int63n(int64(30 * 24 * time.Hour))))
		to := from.Add(time.Second + time.Duration(rng.Int63n(int64(60*24*time.Hour))))
		refreshStart := from.Add(time.Duration(rng.Int63n(int64(90 * 24 * time.Hour))))
		age := time.Duration(1+rng.Int63n(int64(6*time.Hour)/int64(time.Second))) * time.Second

		fromArg := &from
		if rng.Intn(10) == 0 {
			fromArg = nil
		}

		bounds, ok, err := meterCacheBounds(fromArg, to, refreshStart, age, grain)
		require.NoError(t, err)

		horizon := refreshStart.Add(-age)
		hi := to
		if horizon.Before(hi) {
			hi = horizon
		}

		if !ok {
			// The independent predicate: no whole grain bucket fits between the ceiled
			// lower bound and the epsilon-backed-off upper bound.
			require.NotNil(t, fromArg, "unbounded ranges always admit a cache leg")

			ceiledFrom := from.Truncate(grainDuration)
			if ceiledFrom.Before(from) {
				ceiledFrom = ceiledFrom.Add(grainDuration)
			}

			require.False(t, hi.Truncate(grainDuration).Add(-grainDuration).After(ceiledFrom))

			continue
		}

		// Cache bucket bounds are grain aligned.
		require.True(t, bounds.CacheHi.Equal(bounds.CacheHi.Truncate(grainDuration)))

		// G5 epsilon: the last served bucket ends at least one grain before the
		// estimated settled horizon (or the query end, whichever is lower).
		require.False(t, bounds.CacheHi.Add(grainDuration).After(hi))

		// The tail [cacheHi, to) is never empty: the freshest data is always live.
		require.True(t, bounds.CacheHi.Before(to))

		if fromArg == nil {
			require.Nil(t, bounds.CacheLo)

			continue
		}

		require.NotNil(t, bounds.CacheLo)
		require.True(t, bounds.CacheLo.Equal(bounds.CacheLo.Truncate(grainDuration)))

		// Tiling: from <= cacheLo < cacheHi < to with shared leg endpoints means
		// [from, cacheLo) ∪ [cacheLo, cacheHi) ∪ [cacheHi, to) = [from, to) exactly.
		require.False(t, bounds.CacheLo.Before(from))
		require.True(t, bounds.CacheLo.Sub(from) < grainDuration, "pre leg wider than one grain")
		require.True(t, bounds.CacheHi.After(*bounds.CacheLo))
	}
}

func TestMeterCacheReadQueryToSQL(t *testing.T) {
	parse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return ts
	}

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	baseMeter := meter.Meter{
		Key:           "meter1",
		EventType:     "event1",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy: map[string]string{
			"group1": "$.group1",
			"group2": "$.group2",
		},
	}

	t.Run("golden sum windowed day in new york with filters", func(t *testing.T) {
		from := parse("2025-01-01T10:30:00Z")
		to := parse("2025-03-01T00:15:00Z")
		cacheLo := parse("2025-01-01T11:00:00Z")
		cacheHi := parse("2025-02-28T22:00:00Z")
		windowSize := meter.WindowSizeDay

		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:               "openmeter",
				EventsTableName:        "om_events",
				Namespace:              "my_namespace",
				Meter:                  baseMeter,
				From:                   &from,
				To:                     &to,
				FilterSubject:          []string{"subject1"},
				FilterGroupBy:          map[string]filter.FilterString{"group1": {Eq: lo.ToPtr("a")}},
				GroupBy:                []string{"group1", "subject"},
				WindowSize:             &windowSize,
				WindowTimeZone:         newYork,
				EnableDecimalPrecision: true,
			},
			Grain:   CacheGrainHour,
			CacheLo: &cacheLo,
			CacheHi: cacheHi,
		}

		sql, args, err := q.toSQL()
		require.NoError(t, err)

		assert.Equal(t,
			"SELECT tumbleStart(windowstart_bucket, toIntervalDay(1), 'America/New_York') AS windowstart, windowstart + toIntervalDay(1) AS windowend, sum(picked_sum_value) AS value, group1, subject "+
				"FROM ("+
				"(SELECT windowstart AS windowstart_bucket, group_by[1] AS group1, subject, tupleElement(argMax(tuple(sum_value), created_at), 1) AS picked_sum_value "+
				"FROM openmeter.om_meter_cache "+
				"WHERE namespace = ? AND meter_key = ? AND meter_hash = ? AND windowstart >= ? AND windowstart < ? AND subject IN (?) AND group_by[1] = ? "+
				"GROUP BY windowstart, subject, group_by) "+
				"UNION ALL "+
				"(SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart_bucket, JSON_VALUE(om_events.data, '$.group1') as group1, om_events.subject, sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? AND JSON_VALUE(om_events.data, '$.group1') = ? "+
				"GROUP BY windowstart_bucket, group1, subject) "+
				"UNION ALL "+
				"(SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart_bucket, JSON_VALUE(om_events.data, '$.group1') as group1, om_events.subject, sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? AND JSON_VALUE(om_events.data, '$.group1') = ? "+
				"GROUP BY windowstart_bucket, group1, subject)"+
				") AS legs "+
				"GROUP BY windowstart, windowend, group1, subject ORDER BY windowstart",
			sql,
		)

		assert.Equal(t, []interface{}{
			// cache leg [cacheLo, cacheHi)
			"my_namespace", "meter1", meterHash(baseMeter, CacheGrainHour), cacheLo.Unix(), cacheHi.Unix(),
			[]string{"subject1"},
			"a",
			// pre live leg [from, cacheLo)
			"my_namespace", "event1",
			[]string{"subject1"},
			from.Unix(), cacheLo.Unix(), "a",
			// post live leg [cacheHi, to)
			"my_namespace", "event1",
			[]string{"subject1"},
			cacheHi.Unix(), to.Unix(), "a",
		}, args)
	})

	t.Run("golden unique count total", func(t *testing.T) {
		from := parse("2025-01-01T10:30:00Z")
		to := parse("2025-03-01T00:15:00Z")
		cacheLo := parse("2025-01-01T11:00:00Z")
		cacheHi := parse("2025-02-28T22:00:00Z")

		m := baseMeter
		m.Aggregation = meter.MeterAggregationUniqueCount

		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:               "openmeter",
				EventsTableName:        "om_events",
				Namespace:              "my_namespace",
				Meter:                  m,
				From:                   &from,
				To:                     &to,
				EnableDecimalPrecision: true,
			},
			Grain:   CacheGrainHour,
			CacheLo: &cacheLo,
			CacheHi: cacheHi,
		}

		sql, args, err := q.toSQL()
		require.NoError(t, err)

		// UNIQUE_COUNT stays state-level across legs: cached uniqExact states and live
		// uniqExactState legs are merged, never summed.
		assert.Equal(t,
			"SELECT toDateTime(1735727400) AS windowstart, toDateTime(1740788100) AS windowend, uniqExactMerge(picked_uniq_state) AS value "+
				"FROM ("+
				"(SELECT windowstart AS windowstart_bucket, tupleElement(argMax(tuple(uniq_state), created_at), 1) AS picked_uniq_state "+
				"FROM openmeter.om_meter_cache "+
				"WHERE namespace = ? AND meter_key = ? AND meter_hash = ? AND windowstart >= ? AND windowstart < ? "+
				"GROUP BY windowstart, subject, group_by) "+
				"UNION ALL "+
				"(SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart_bucket, uniqExactState(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null')) AS uniq_state "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? "+
				"GROUP BY windowstart_bucket) "+
				"UNION ALL "+
				"(SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart_bucket, uniqExactState(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null')) AS uniq_state "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? "+
				"GROUP BY windowstart_bucket)"+
				") AS legs",
			sql,
		)

		assert.Equal(t, []interface{}{
			"my_namespace", "meter1", meterHash(m, CacheGrainHour), cacheLo.Unix(), cacheHi.Unix(),
			"my_namespace", "event1", from.Unix(), cacheLo.Unix(),
			"my_namespace", "event1", cacheHi.Unix(), to.Unix(),
		}, args)
	})

	t.Run("golden avg windowed hour on grid has no pre leg", func(t *testing.T) {
		from := parse("2025-01-01T10:00:00Z")
		to := parse("2025-01-02T00:00:00Z")
		cacheHi := parse("2025-01-01T22:00:00Z")
		windowSize := meter.WindowSizeHour

		m := baseMeter
		m.Aggregation = meter.MeterAggregationAvg

		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:               "openmeter",
				EventsTableName:        "om_events",
				Namespace:              "my_namespace",
				Meter:                  m,
				From:                   &from,
				To:                     &to,
				GroupBy:                []string{"subject"},
				WindowSize:             &windowSize,
				QuerySettings:          map[string]string{"max_execution_time": "600"},
				EnableDecimalPrecision: true,
			},
			Grain:   CacheGrainHour,
			CacheLo: &from,
			CacheHi: cacheHi,
		}

		sql, args, err := q.toSQL()
		require.NoError(t, err)

		// AVG combines as Σsum ÷ Σvalue_count — an average of averages would be wrong —
		// and the AVG pick carries both columns in one argMax tuple so they always come
		// from the same row version.
		assert.Equal(t,
			"SELECT tumbleStart(windowstart_bucket, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(windowstart_bucket, toIntervalHour(1), 'UTC') AS windowend, toFloat64(sum(picked_sum_value)) / sum(picked_value_count) AS value, subject "+
				"FROM ("+
				"(SELECT windowstart AS windowstart_bucket, subject, tupleElement(argMax(tuple(sum_value, value_count), created_at), 1) AS picked_sum_value, tupleElement(argMax(tuple(sum_value, value_count), created_at), 2) AS picked_value_count "+
				"FROM openmeter.om_meter_cache "+
				"WHERE namespace = ? AND meter_key = ? AND meter_hash = ? AND windowstart >= ? AND windowstart < ? "+
				"GROUP BY windowstart, subject, group_by) "+
				"UNION ALL "+
				"(SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart_bucket, om_events.subject, sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value, count(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS value_count "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? "+
				"GROUP BY windowstart_bucket, subject)"+
				") AS legs "+
				"GROUP BY windowstart, windowend, subject ORDER BY windowstart SETTINGS max_execution_time = 600",
			sql,
		)

		assert.Equal(t, []interface{}{
			"my_namespace", "meter1", meterHash(m, CacheGrainHour), from.Unix(), cacheHi.Unix(),
			"my_namespace", "event1", cacheHi.Unix(), to.Unix(),
		}, args)
	})

	t.Run("golden latest windowed unbounded from", func(t *testing.T) {
		to := parse("2025-01-02T00:00:00Z")
		cacheHi := parse("2025-01-01T22:00:00Z")
		windowSize := meter.WindowSizeHour

		m := baseMeter
		m.Aggregation = meter.MeterAggregationLatest

		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:               "openmeter",
				EventsTableName:        "om_events",
				Namespace:              "my_namespace",
				Meter:                  m,
				To:                     &to,
				WindowSize:             &windowSize,
				EnableDecimalPrecision: true,
			},
			Grain:   CacheGrainHour,
			CacheHi: cacheHi,
		}

		sql, args, err := q.toSQL()
		require.NoError(t, err)

		// G14's two argMax semantics side by side: the cache leg's argMax by created_at
		// is the newest-wins re-read, while argMaxState(value, time)/argMaxMerge carry
		// the live LATEST semantics.
		assert.Equal(t,
			"SELECT tumbleStart(windowstart_bucket, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(windowstart_bucket, toIntervalHour(1), 'UTC') AS windowend, argMaxMerge(picked_latest_state) AS value "+
				"FROM ("+
				"(SELECT windowstart AS windowstart_bucket, tupleElement(argMax(tuple(latest_state), created_at), 1) AS picked_latest_state "+
				"FROM openmeter.om_meter_cache "+
				"WHERE namespace = ? AND meter_key = ? AND meter_hash = ? AND windowstart < ? "+
				"GROUP BY windowstart, subject, group_by) "+
				"UNION ALL "+
				"(SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart_bucket, argMaxState(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19), om_events.time) AS latest_state "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? "+
				"GROUP BY windowstart_bucket)"+
				") AS legs "+
				"GROUP BY windowstart, windowend ORDER BY windowstart",
			sql,
		)

		assert.Equal(t, []interface{}{
			"my_namespace", "meter1", meterHash(m, CacheGrainHour), cacheHi.Unix(),
			"my_namespace", "event1", cacheHi.Unix(), to.Unix(),
		}, args)
	})

	t.Run("total without from is rejected", func(t *testing.T) {
		to := parse("2025-01-02T00:00:00Z")

		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:               "openmeter",
				EventsTableName:        "om_events",
				Namespace:              "my_namespace",
				Meter:                  baseMeter,
				To:                     &to,
				EnableDecimalPrecision: true,
			},
			Grain:   CacheGrainHour,
			CacheHi: parse("2025-01-01T22:00:00Z"),
		}

		_, _, err := q.toSQL()
		require.Error(t, err)
	})

	t.Run("nil to is rejected", func(t *testing.T) {
		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter:           baseMeter,
			},
			Grain: CacheGrainHour,
		}

		_, _, err := q.toSQL()
		require.Error(t, err)
	})

	t.Run("query group by outside the meter fails leg build", func(t *testing.T) {
		from := parse("2025-01-01T10:00:00Z")
		to := parse("2025-01-02T00:00:00Z")
		windowSize := meter.WindowSizeHour

		q := meterCacheReadQuery{
			queryMeter: queryMeter{
				Database:               "openmeter",
				EventsTableName:        "om_events",
				Namespace:              "my_namespace",
				Meter:                  baseMeter,
				From:                   &from,
				To:                     &to,
				GroupBy:                []string{"group3"},
				WindowSize:             &windowSize,
				EnableDecimalPrecision: true,
			},
			Grain:   CacheGrainHour,
			CacheLo: &from,
			CacheHi: parse("2025-01-01T22:00:00Z"),
		}

		_, _, err := q.toSQL()
		require.Error(t, err)
	})
}

func TestMeterCacheCombinerExprs(t *testing.T) {
	// The per-aggregation pairs of newest-wins pick (cache leg) and cross-leg combiner
	// (outer query); additive aggregations re-aggregate, AVG divides sums by counts,
	// UNIQUE_COUNT and LATEST stay state-level.
	tests := []struct {
		aggregation meter.MeterAggregation
		wantPicks   []string
		wantValue   string
	}{
		{
			aggregation: meter.MeterAggregationSum,
			wantPicks:   []string{"tupleElement(argMax(tuple(sum_value), created_at), 1) AS picked_sum_value"},
			wantValue:   "sum(picked_sum_value) AS value",
		},
		{
			aggregation: meter.MeterAggregationCount,
			wantPicks:   []string{"tupleElement(argMax(tuple(count_value), created_at), 1) AS picked_count_value"},
			wantValue:   "sum(picked_count_value) AS value",
		},
		{
			aggregation: meter.MeterAggregationAvg,
			wantPicks: []string{
				"tupleElement(argMax(tuple(sum_value, value_count), created_at), 1) AS picked_sum_value",
				"tupleElement(argMax(tuple(sum_value, value_count), created_at), 2) AS picked_value_count",
			},
			wantValue: "toFloat64(sum(picked_sum_value)) / sum(picked_value_count) AS value",
		},
		{
			aggregation: meter.MeterAggregationMin,
			wantPicks:   []string{"tupleElement(argMax(tuple(min_value), created_at), 1) AS picked_min_value"},
			wantValue:   "min(picked_min_value) AS value",
		},
		{
			aggregation: meter.MeterAggregationMax,
			wantPicks:   []string{"tupleElement(argMax(tuple(max_value), created_at), 1) AS picked_max_value"},
			wantValue:   "max(picked_max_value) AS value",
		},
		{
			aggregation: meter.MeterAggregationUniqueCount,
			wantPicks:   []string{"tupleElement(argMax(tuple(uniq_state), created_at), 1) AS picked_uniq_state"},
			wantValue:   "uniqExactMerge(picked_uniq_state) AS value",
		},
		{
			aggregation: meter.MeterAggregationLatest,
			wantPicks:   []string{"tupleElement(argMax(tuple(latest_state), created_at), 1) AS picked_latest_state"},
			wantValue:   "argMaxMerge(picked_latest_state) AS value",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.aggregation), func(t *testing.T) {
			picks, err := meterCacheNewestWinsExprs(tt.aggregation)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPicks, picks)

			value, err := meterCacheCombinedValueExpr(tt.aggregation)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, value)
		})
	}

	t.Run("invalid aggregation", func(t *testing.T) {
		_, err := meterCacheNewestWinsExprs(meter.MeterAggregation("INVALID"))
		require.Error(t, err)

		_, err = meterCacheCombinedValueExpr(meter.MeterAggregation("INVALID"))
		require.Error(t, err)
	})
}

func TestMeterCacheMarkerOverlapQueryToSQL(t *testing.T) {
	parse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return ts
	}

	cacheLo := parse("2025-01-01T11:00:00Z")
	cacheHi := parse("2025-02-28T22:00:00Z")
	refreshStart := parse("2025-03-01T00:10:00Z")
	backfilledAt := parse("2025-01-15T08:00:00Z")

	t.Run("golden with bounded cache leg", func(t *testing.T) {
		sql, args := meterCacheMarkerOverlapQuery{
			Database:     "openmeter",
			Namespace:    "my_namespace",
			EventType:    "event1",
			CacheLo:      &cacheLo,
			CacheHi:      cacheHi,
			RefreshStart: refreshStart,
			HealBound:    20 * time.Minute,
			BackfilledAt: backfilledAt,
		}.toSQL()

		assert.Equal(t,
			"SELECT count() FROM openmeter.om_meter_cache_invalidations "+
				"WHERE namespace = ? AND event_type = ? AND window_lo < ? AND window_hi > ? AND created_at >= ? AND (created_at >= ? OR created_at <= ?)",
			sql,
		)

		// The heal complement: a marker is unhealed when the view's backfill started
		// before it was written, and it was written at or after the latest refresh
		// started or so long before that refresh that its stored_at lookback provably no
		// longer covered the late events (G1).
		assert.Equal(t, []interface{}{
			"my_namespace", "event1", cacheHi.Unix(), cacheLo.Unix(),
			backfilledAt, refreshStart, refreshStart.Add(-20 * time.Minute),
		}, args)
	})

	t.Run("golden with unbounded cache leg", func(t *testing.T) {
		sql, args := meterCacheMarkerOverlapQuery{
			Database:     "openmeter",
			Namespace:    "my_namespace",
			EventType:    "event1",
			CacheHi:      cacheHi,
			RefreshStart: refreshStart,
			HealBound:    20 * time.Minute,
			BackfilledAt: backfilledAt,
		}.toSQL()

		assert.Equal(t,
			"SELECT count() FROM openmeter.om_meter_cache_invalidations "+
				"WHERE namespace = ? AND event_type = ? AND window_lo < ? AND created_at >= ? AND (created_at >= ? OR created_at <= ?)",
			sql,
		)

		assert.Equal(t, []interface{}{
			"my_namespace", "event1", cacheHi.Unix(),
			backfilledAt, refreshStart, refreshStart.Add(-20 * time.Minute),
		}, args)
	})
}
