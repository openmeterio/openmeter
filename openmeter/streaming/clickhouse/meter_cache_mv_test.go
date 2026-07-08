package clickhouse

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func mvFixture(m meter.Meter, grain CacheGrain) createMeterCacheMV {
	return createMeterCacheMV{
		Database:        "openmeter",
		EventsTableName: "om_events",
		Namespace:       "my_namespace",
		Meter:           m,
		Grain:           grain,
		RefreshInterval: 10 * time.Minute,
		MinimumUsageAge: time.Hour,
	}
}

func TestCreateMeterCacheMVSQL(t *testing.T) {
	t.Run("golden sum hour with group bys", func(t *testing.T) {
		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
			GroupBy: map[string]string{
				"group1": "$.group1",
				"group2": "$.group2",
			},
		}, CacheGrainHour)

		sql, err := mv.toSQL()
		require.NoError(t, err)
		assert.Equal(t,
			"CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_meter_cache_mv_814585ab_a71813334efc5e0b "+
				"REFRESH EVERY 600 SECOND RANDOMIZE FOR 200 SECOND APPEND TO openmeter.om_meter_cache AS "+
				"SELECT namespace, 'meter1' AS meter_key, 12040394714864442891 AS meter_hash, "+
				"tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, subject, "+
				"[JSON_VALUE(om_events.data, '$.group1'), JSON_VALUE(om_events.data, '$.group2')] AS group_by, now64(3) AS created_at, "+
				"sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = 'my_namespace' AND om_events.type = 'event1' "+
				"AND om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC') "+
				"AND toStartOfInterval(om_events.time, INTERVAL 1 HOUR, 'UTC') IN ("+
				"SELECT DISTINCT toStartOfInterval(time, INTERVAL 1 HOUR, 'UTC') FROM openmeter.om_events "+
				"WHERE namespace = 'my_namespace' AND type = 'event1' AND stored_at >= now() - INTERVAL 5400 SECOND "+
				"UNION DISTINCT "+
				"SELECT subtractSeconds(toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC'), (number + 1) * 3600) FROM numbers(1)) "+
				"GROUP BY namespace, windowstart, subject, group_by "+
				`COMMENT '{"namespace":"my_namespace","meter_key":"meter1","event_type":"event1","meter_hash":"a71813334efc5e0b","ddl_hash":"49eab5c7b097a200"}'`,
			sql,
		)
	})

	t.Run("golden avg with event from and no group by", func(t *testing.T) {
		eventFrom, err := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
		require.NoError(t, err)

		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationAvg,
			ValueProperty: lo.ToPtr("$.value"),
			EventFrom:     &eventFrom,
		}, CacheGrainHour)

		sql, err := mv.toSQL()
		require.NoError(t, err)
		assert.Equal(t,
			"CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_meter_cache_mv_814585ab_249fdd0e66a5f244 "+
				"REFRESH EVERY 600 SECOND RANDOMIZE FOR 200 SECOND APPEND TO openmeter.om_meter_cache AS "+
				"SELECT namespace, 'meter1' AS meter_key, 2639070960583832132 AS meter_hash, "+
				"tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, subject, "+
				"emptyArrayString() AS group_by, now64(3) AS created_at, "+
				"sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value, "+
				"count(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS value_count "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = 'my_namespace' AND om_events.type = 'event1' "+
				"AND om_events.time >= 1735689600 "+
				"AND om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC') "+
				"AND toStartOfInterval(om_events.time, INTERVAL 1 HOUR, 'UTC') IN ("+
				"SELECT DISTINCT toStartOfInterval(time, INTERVAL 1 HOUR, 'UTC') FROM openmeter.om_events "+
				"WHERE namespace = 'my_namespace' AND type = 'event1' AND stored_at >= now() - INTERVAL 5400 SECOND "+
				"UNION DISTINCT "+
				"SELECT subtractSeconds(toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC'), (number + 1) * 3600) FROM numbers(1)) "+
				"GROUP BY namespace, windowstart, subject, group_by "+
				`COMMENT '{"namespace":"my_namespace","meter_key":"meter1","event_type":"event1","meter_hash":"249fdd0e66a5f244","ddl_hash":"6eca4aaec943452f"}'`,
			sql,
		)
	})

	t.Run("string literals are escaped", func(t *testing.T) {
		mv := mvFixture(meter.Meter{
			Key:         `it's "quoted" \meter`,
			EventType:   "event'1",
			Aggregation: meter.MeterAggregationCount,
		}, CacheGrainHour)
		mv.Namespace = `ns\'1`

		sql, err := mv.toSQL()
		require.NoError(t, err)
		assert.Contains(t, sql, `'it\'s "quoted" \\meter' AS meter_key`)
		assert.Contains(t, sql, `om_events.namespace = 'ns\\\'1'`)
		assert.Contains(t, sql, `om_events.type = 'event\'1'`)
	})

	t.Run("grain drives bucket arithmetic", func(t *testing.T) {
		m := meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		}

		tests := []struct {
			grain           CacheGrain
			wantWindowstart string
			wantBound       string
			// 3 x 10m refresh interval, rounded up to whole buckets, floored at one:
			// 30 minute-buckets, 1 hour-bucket, 1 day-bucket.
			wantStrip string
		}{
			{
				grain:           CacheGrainMinute,
				wantWindowstart: "tumbleStart(om_events.time, toIntervalMinute(1), 'UTC') AS windowstart",
				wantBound:       "om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 MINUTE, 'UTC')",
				wantStrip:       "(number + 1) * 60) FROM numbers(30))",
			},
			{
				grain:           CacheGrainHour,
				wantWindowstart: "tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart",
				wantBound:       "om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC')",
				wantStrip:       "(number + 1) * 3600) FROM numbers(1))",
			},
			{
				grain:           CacheGrainDay,
				wantWindowstart: "tumbleStart(om_events.time, toIntervalDay(1), 'UTC') AS windowstart",
				wantBound:       "om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 DAY, 'UTC')",
				wantStrip:       "(number + 1) * 86400) FROM numbers(1))",
			},
		}

		for _, tt := range tests {
			t.Run(string(tt.grain), func(t *testing.T) {
				sql, err := mvFixture(m, tt.grain).toSQL()
				require.NoError(t, err)
				assert.Contains(t, sql, tt.wantWindowstart)
				assert.Contains(t, sql, tt.wantBound)
				assert.Contains(t, sql, tt.wantStrip)
			})
		}
	})

	t.Run("dirty window is floored at one hour", func(t *testing.T) {
		// minimumUsageAge 10m + 3 x 5m = 25m lookback would be shorter than the floor;
		// the floor keeps slow-arriving stored_at updates coverable.
		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		}, CacheGrainMinute)
		mv.RefreshInterval = 5 * time.Minute
		mv.MinimumUsageAge = 10 * time.Minute

		sql, err := mv.toSQL()
		require.NoError(t, err)
		assert.Contains(t, sql, "stored_at >= now() - INTERVAL 3600 SECOND")
	})

	t.Run("randomize is omitted below one second", func(t *testing.T) {
		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		}, CacheGrainHour)
		mv.RefreshInterval = 2 * time.Second

		sql, err := mv.toSQL()
		require.NoError(t, err)
		assert.Contains(t, sql, "REFRESH EVERY 2 SECOND APPEND TO")
		assert.NotContains(t, sql, "RANDOMIZE")
	})

	t.Run("rejects sub-second refresh interval", func(t *testing.T) {
		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		}, CacheGrainHour)
		mv.RefreshInterval = 500 * time.Millisecond

		_, err := mv.toSQL()
		require.ErrorContains(t, err, "refresh interval must be at least one second")
	})

	t.Run("rejects invalid grain", func(t *testing.T) {
		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
		}, CacheGrain("month"))

		_, err := mv.toSQL()
		require.ErrorContains(t, err, "invalid meter cache grain: month")
	})

	t.Run("rejects reserved alias group by keys (G9)", func(t *testing.T) {
		for _, key := range []string{"namespace", "time", "windowstart", "meter_hash", "total_value", "uniq_state"} {
			t.Run(key, func(t *testing.T) {
				mv := mvFixture(meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy:       map[string]string{key: "$.x"},
				}, CacheGrainHour)

				_, err := mv.toSQL()
				require.ErrorContains(t, err, "collide with reserved SQL aliases: "+key)
			})
		}
	})

	t.Run("rejects missing value property", func(t *testing.T) {
		mv := mvFixture(meter.Meter{
			Key:         "meter1",
			EventType:   "event1",
			Aggregation: meter.MeterAggregationSum,
		}, CacheGrainHour)

		_, err := mv.toSQL()
		require.ErrorContains(t, err, "meter value property is required")
	})

	t.Run("rejects latest aggregation", func(t *testing.T) {
		// LATEST is excluded from the cache entirely (meterCacheStaticReject / (*Connector).
		// DesiredMeterCacheView both reject it before this point); the MV generator's combine
		// form must never be reachable for it either.
		mv := mvFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationLatest,
			ValueProperty: lo.ToPtr("$.value"),
		}, CacheGrainHour)

		_, err := mv.toSQL()
		require.ErrorContains(t, err, "invalid aggregation type: LATEST")
	})
}

// TestCreateMeterCacheMVMatrix sweeps the generator over every aggregation x group-by
// shape x grain x EventFrom combination and asserts the structural invariants of the
// emitted DDL. APPEND is the load-bearing one: a single non-APPEND cache MV refresh
// atomically wipes the entire shared om_meter_cache table.
//
// Watched RED: with `%s APPEND TO %s` changed to `%s TO %s` in createMeterCacheMV.toSQL
// and the runtime guard disabled (`if false && !strings.Contains(...)`), this matrix and
// the golden tests fail on the missing APPEND.
func TestCreateMeterCacheMVMatrix(t *testing.T) {
	eventFrom, err := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	require.NoError(t, err)

	// LATEST is excluded from the cache entirely (meterCacheStaticReject) and covered by its
	// own rejection test above, not this matrix.
	aggregations := []meter.MeterAggregation{
		meter.MeterAggregationSum,
		meter.MeterAggregationCount,
		meter.MeterAggregationAvg,
		meter.MeterAggregationMin,
		meter.MeterAggregationMax,
		meter.MeterAggregationUniqueCount,
	}

	wantCombineAliases := map[meter.MeterAggregation][]string{
		meter.MeterAggregationSum:         {"sum_value"},
		meter.MeterAggregationCount:       {"count_value"},
		meter.MeterAggregationAvg:         {"sum_value", "value_count"},
		meter.MeterAggregationMin:         {"min_value"},
		meter.MeterAggregationMax:         {"max_value"},
		meter.MeterAggregationUniqueCount: {"uniq_state"},
	}

	groupByShapes := map[string]map[string]string{
		"none":     nil,
		"one dim":  {"group1": "$.group1"},
		"two dims": {"group1": "$.group1", "group2": "$.nested.group2"},
	}

	for _, aggregation := range aggregations {
		for grainName, grain := range map[string]CacheGrain{"minute": CacheGrainMinute, "hour": CacheGrainHour, "day": CacheGrainDay} {
			for shapeName, groupBy := range groupByShapes {
				for _, withEventFrom := range []bool{false, true} {
					name := fmt.Sprintf("%s/%s/%s/eventFrom=%t", aggregation, grainName, shapeName, withEventFrom)

					t.Run(name, func(t *testing.T) {
						m := meter.Meter{
							Key:         "meter1",
							EventType:   "event1",
							Aggregation: aggregation,
							GroupBy:     groupBy,
						}
						if aggregation != meter.MeterAggregationCount {
							m.ValueProperty = lo.ToPtr("$.value")
						}
						if withEventFrom {
							m.EventFrom = &eventFrom
						}

						mv := mvFixture(m, grain)

						sql, err := mv.toSQL()
						require.NoError(t, err)

						// APPEND is mandatory on every emitted cache MV, no exceptions.
						require.Contains(t, sql, " APPEND TO openmeter.om_meter_cache AS ")
						require.True(t, strings.HasPrefix(sql, "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_meter_cache_mv_"))

						// Settled bound and the dirty union strip are always present.
						require.Contains(t, sql, "toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1")
						require.Contains(t, sql, "UNION DISTINCT")
						require.Contains(t, sql, "stored_at >= now() - INTERVAL 5400 SECOND")

						for _, alias := range wantCombineAliases[aggregation] {
							require.Contains(t, sql, " AS "+alias)
						}

						if withEventFrom {
							require.Contains(t, sql, fmt.Sprintf("om_events.time >= %d", eventFrom.Unix()))
						} else {
							require.NotContains(t, sql, "om_events.time >= ")
						}

						// The comment always carries parseable, unstamped metadata.
						comment := sql[strings.LastIndex(sql, "COMMENT '")+len("COMMENT '"):]
						comment = strings.TrimSuffix(comment, "'")
						metadata, err := parseMeterCacheMVMetadata(comment)
						require.NoError(t, err)
						require.Equal(t, "meter1", metadata.MeterKey)
						require.Equal(t, "event1", metadata.EventType)
						require.Equal(t, formatCacheHash(meterHash(m, grain)), metadata.MeterHash)
						require.Nil(t, metadata.BackfilledAt)
					})
				}
			}
		}
	}
}
