package clickhouse

import (
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func backfillFixture(m meter.Meter) meterCacheBackfill {
	return meterCacheBackfill{
		Database:        "openmeter",
		EventsTableName: "om_events",
		Namespace:       "my_namespace",
		Meter:           m,
		Grain:           CacheGrainHour,
		MinimumUsageAge: time.Hour,
	}
}

func TestMeterCacheBackfillSQL(t *testing.T) {
	t.Run("golden sum full settled history", func(t *testing.T) {
		backfill := backfillFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
			GroupBy: map[string]string{
				"group1": "$.group1",
				"group2": "$.group2",
			},
		})

		sql, err := backfill.toSQL()
		require.NoError(t, err)
		assert.Equal(t,
			"INSERT INTO openmeter.om_meter_cache (namespace, meter_key, meter_hash, windowstart, subject, group_by, created_at, sum_value) "+
				"SELECT namespace, 'meter1' AS meter_key, 12040394714864442891 AS meter_hash, "+
				"tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, subject, "+
				"[JSON_VALUE(om_events.data, '$.group1'), JSON_VALUE(om_events.data, '$.group2')] AS group_by, now64(3) AS created_at, "+
				"sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value "+
				"FROM openmeter.om_events "+
				"WHERE om_events.namespace = 'my_namespace' AND om_events.type = 'event1' "+
				"AND om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC') "+
				"GROUP BY namespace, windowstart, subject, group_by",
			sql,
		)
	})

	t.Run("insert column list follows the aggregation", func(t *testing.T) {
		tests := []struct {
			aggregation meter.MeterAggregation
			wantColumns string
		}{
			{aggregation: meter.MeterAggregationSum, wantColumns: "created_at, sum_value)"},
			{aggregation: meter.MeterAggregationCount, wantColumns: "created_at, count_value)"},
			{aggregation: meter.MeterAggregationAvg, wantColumns: "created_at, sum_value, value_count)"},
			{aggregation: meter.MeterAggregationMin, wantColumns: "created_at, min_value)"},
			{aggregation: meter.MeterAggregationMax, wantColumns: "created_at, max_value)"},
			{aggregation: meter.MeterAggregationUniqueCount, wantColumns: "created_at, uniq_state)"},
			{aggregation: meter.MeterAggregationLatest, wantColumns: "created_at, latest_state)"},
		}

		for _, tt := range tests {
			t.Run(string(tt.aggregation), func(t *testing.T) {
				m := meter.Meter{Key: "meter1", EventType: "event1", Aggregation: tt.aggregation}
				if tt.aggregation != meter.MeterAggregationCount {
					m.ValueProperty = lo.ToPtr("$.value")
				}

				sql, err := backfillFixture(m).toSQL()
				require.NoError(t, err)
				assert.Contains(t, sql, "INSERT INTO openmeter.om_meter_cache (namespace, meter_key, meter_hash, windowstart, subject, group_by, "+tt.wantColumns)
				// Backfills cover full settled history: no dirty-bucket restriction.
				assert.NotContains(t, sql, "UNION DISTINCT")
				assert.NotContains(t, sql, "stored_at")
			})
		}
	})

	t.Run("lower bound follows queryMeter.from() semantics", func(t *testing.T) {
		earlier, err := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
		require.NoError(t, err)
		later, err := time.Parse(time.RFC3339, "2025-06-01T00:00:00Z")
		require.NoError(t, err)

		m := meter.Meter{Key: "meter1", EventType: "event1", Aggregation: meter.MeterAggregationCount}

		tests := []struct {
			name      string
			eventFrom *time.Time
			chunkFrom *time.Time
			wantBound *time.Time
		}{
			{name: "neither set means unbounded history", eventFrom: nil, chunkFrom: nil, wantBound: nil},
			{name: "only event from", eventFrom: &earlier, chunkFrom: nil, wantBound: &earlier},
			{name: "only chunk from", eventFrom: nil, chunkFrom: &earlier, wantBound: &earlier},
			{name: "event from wins when later", eventFrom: &later, chunkFrom: &earlier, wantBound: &later},
			{name: "chunk from wins when later", eventFrom: &earlier, chunkFrom: &later, wantBound: &later},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				meterWithFrom := m
				meterWithFrom.EventFrom = tt.eventFrom

				backfill := backfillFixture(meterWithFrom)
				backfill.From = tt.chunkFrom

				sql, err := backfill.toSQL()
				require.NoError(t, err)

				if tt.wantBound == nil {
					assert.NotContains(t, sql, "om_events.time >= ")
				} else {
					assert.Contains(t, sql, fmt.Sprintf("om_events.time >= %d", tt.wantBound.Unix()))
				}
			})
		}
	})

	t.Run("chunk upper bound applies alongside the settled bound", func(t *testing.T) {
		chunkTo, err := time.Parse(time.RFC3339, "2025-02-01T00:00:00Z")
		require.NoError(t, err)

		backfill := backfillFixture(meter.Meter{Key: "meter1", EventType: "event1", Aggregation: meter.MeterAggregationCount})
		backfill.To = &chunkTo

		sql, err := backfill.toSQL()
		require.NoError(t, err)
		assert.Contains(t, sql, fmt.Sprintf("om_events.time < %d", chunkTo.Unix()))
		assert.Contains(t, sql, "om_events.time < toStartOfInterval(now() - INTERVAL 3600 SECOND, INTERVAL 1 HOUR, 'UTC')")
	})

	t.Run("rejects reserved alias group by keys (G9)", func(t *testing.T) {
		backfill := backfillFixture(meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: lo.ToPtr("$.value"),
			GroupBy:       map[string]string{"stored_at": "$.x"},
		})

		_, err := backfill.toSQL()
		require.ErrorContains(t, err, "collide with reserved SQL aliases: stored_at")
	})
}

func TestBackfillMonthChunks(t *testing.T) {
	parse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return ts
	}

	t.Run("splits on utc month boundaries", func(t *testing.T) {
		chunks := backfillMonthChunks(parse("2025-01-15T10:30:00Z"), parse("2025-03-02T00:00:00Z"))
		assert.Equal(t, []backfillChunk{
			{From: parse("2025-01-15T10:30:00Z"), To: parse("2025-02-01T00:00:00Z")},
			{From: parse("2025-02-01T00:00:00Z"), To: parse("2025-03-01T00:00:00Z")},
			{From: parse("2025-03-01T00:00:00Z"), To: parse("2025-03-02T00:00:00Z")},
		}, chunks)
	})

	t.Run("range within one month is a single chunk", func(t *testing.T) {
		chunks := backfillMonthChunks(parse("2025-01-05T00:00:00Z"), parse("2025-01-20T00:00:00Z"))
		assert.Equal(t, []backfillChunk{
			{From: parse("2025-01-05T00:00:00Z"), To: parse("2025-01-20T00:00:00Z")},
		}, chunks)
	})

	t.Run("empty and inverted ranges yield no chunks", func(t *testing.T) {
		at := parse("2025-01-05T00:00:00Z")
		assert.Nil(t, backfillMonthChunks(at, at))
		assert.Nil(t, backfillMonthChunks(at.Add(time.Hour), at))
	})

	t.Run("chunks always tile the range exactly", func(t *testing.T) {
		// Property over a spread of awkward ranges: chunk edges must meet with no gap or
		// overlap, ends must match the inputs, interior boundaries must be month starts.
		ranges := [][2]time.Time{
			{parse("2024-11-30T23:59:59Z"), parse("2025-03-01T00:00:01Z")},
			{parse("2025-01-01T00:00:00Z"), parse("2026-01-01T00:00:00Z")},
			{parse("2025-02-28T12:00:00Z"), parse("2025-03-01T00:00:00Z")},
			{parse("2023-12-31T23:00:00-05:00"), parse("2024-02-15T08:00:00+09:00")},
		}

		for _, r := range ranges {
			chunks := backfillMonthChunks(r[0], r[1])
			require.NotEmpty(t, chunks)

			assert.True(t, chunks[0].From.Equal(r[0].UTC()))
			assert.True(t, chunks[len(chunks)-1].To.Equal(r[1].UTC()))

			for i, chunk := range chunks {
				require.True(t, chunk.From.Before(chunk.To))

				if i > 0 {
					require.True(t, chunks[i-1].To.Equal(chunk.From))
					require.Equal(t, 1, chunk.From.Day())
					require.Equal(t, 0, chunk.From.Hour())
				}
			}
		}
	})
}
