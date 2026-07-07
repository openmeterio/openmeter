package clickhouse

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestWindowExprs(t *testing.T) {
	tests := []struct {
		name       string
		windowSize meter.WindowSize
		tz         string
		want       []string
	}{
		{
			name:       "minute",
			windowSize: meter.WindowSizeMinute,
			tz:         "UTC",
			want: []string{
				"tumbleStart(om_events.time, toIntervalMinute(1), 'UTC') AS windowstart",
				"tumbleEnd(om_events.time, toIntervalMinute(1), 'UTC') AS windowend",
			},
		},
		{
			name:       "hour",
			windowSize: meter.WindowSizeHour,
			tz:         "UTC",
			want: []string{
				"tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart",
				"tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend",
			},
		},
		{
			name:       "day derives windowend from windowstart",
			windowSize: meter.WindowSizeDay,
			tz:         "Asia/Shanghai",
			want: []string{
				"tumbleStart(om_events.time, toIntervalDay(1), 'Asia/Shanghai') AS windowstart",
				"windowstart + toIntervalDay(1) AS windowend",
			},
		},
		{
			name:       "month keeps the toDateTime cast",
			windowSize: meter.WindowSizeMonth,
			tz:         "Europe/Budapest",
			want: []string{
				"toDateTime(tumbleStart(om_events.time, toIntervalMonth(1), 'Europe/Budapest'), 'Europe/Budapest') AS windowstart",
				"toDateTime(tumbleEnd(om_events.time, toIntervalMonth(1), 'Europe/Budapest'), 'Europe/Budapest') AS windowend",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := windowExprs(tt.windowSize, "om_events.time", tt.tz)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("invalid window size", func(t *testing.T) {
		_, err := windowExprs(meter.WindowSize("SECOND"), "om_events.time", "UTC")
		require.ErrorContains(t, err, "invalid window size type: SECOND")
	})
}

func TestValueExprPlain(t *testing.T) {
	tests := []struct {
		name                   string
		aggregation            meter.MeterAggregation
		enableDecimalPrecision bool
		want                   string
	}{
		{
			name:        "count ignores value property",
			aggregation: meter.MeterAggregationCount,
			want:        "count(*) AS value",
		},
		{
			name:        "unique count aggregates the raw string regardless of decimal precision",
			aggregation: meter.MeterAggregationUniqueCount,
			want:        "uniqExact(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null')) AS value",
		},
		{
			name:                   "unique count with decimal precision",
			aggregation:            meter.MeterAggregationUniqueCount,
			enableDecimalPrecision: true,
			want:                   "uniqExact(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null')) AS value",
		},
		{
			name:        "latest float",
			aggregation: meter.MeterAggregationLatest,
			want:        "argMax(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null), om_events.time) AS value",
		},
		{
			name:                   "latest decimal",
			aggregation:            meter.MeterAggregationLatest,
			enableDecimalPrecision: true,
			want:                   "argMax(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19), om_events.time) AS value",
		},
		{
			name:        "sum float",
			aggregation: meter.MeterAggregationSum,
			want:        "sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value",
		},
		{
			name:                   "sum decimal",
			aggregation:            meter.MeterAggregationSum,
			enableDecimalPrecision: true,
			want:                   "sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS value",
		},
		{
			name:                   "avg decimal",
			aggregation:            meter.MeterAggregationAvg,
			enableDecimalPrecision: true,
			want:                   "avg(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS value",
		},
		{
			name:        "min float",
			aggregation: meter.MeterAggregationMin,
			want:        "min(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value",
		},
		{
			name:        "max float",
			aggregation: meter.MeterAggregationMax,
			want:        "max(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := meter.Meter{Aggregation: tt.aggregation}
			if tt.aggregation != meter.MeterAggregationCount {
				m.ValueProperty = lo.ToPtr("$.value")
			}

			got, err := valueExprPlain(m, "om_events.data", "om_events.time", tt.enableDecimalPrecision)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("invalid aggregation", func(t *testing.T) {
		_, err := valueExprPlain(meter.Meter{Aggregation: meter.MeterAggregation("MEDIAN")}, "om_events.data", "om_events.time", false)
		require.ErrorContains(t, err, "invalid aggregation type: MEDIAN")
	})

	t.Run("missing value property", func(t *testing.T) {
		_, err := valueExprPlain(meter.Meter{Aggregation: meter.MeterAggregationSum}, "om_events.data", "om_events.time", false)
		require.ErrorContains(t, err, "meter value property is required for SUM aggregation")
	})
}

func TestValueExprsCombine(t *testing.T) {
	tests := []struct {
		name        string
		aggregation meter.MeterAggregation
		want        []string
	}{
		{
			name:        "sum",
			aggregation: meter.MeterAggregationSum,
			want:        []string{"sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value"},
		},
		{
			name:        "count",
			aggregation: meter.MeterAggregationCount,
			want:        []string{"count(*) AS count_value"},
		},
		{
			name:        "avg stores the sum plus non-null value count pair",
			aggregation: meter.MeterAggregationAvg,
			want: []string{
				"sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS sum_value",
				"count(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS value_count",
			},
		},
		{
			name:        "min",
			aggregation: meter.MeterAggregationMin,
			want:        []string{"min(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS min_value"},
		},
		{
			name:        "max",
			aggregation: meter.MeterAggregationMax,
			want:        []string{"max(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null'), 19)) AS max_value"},
		},
		{
			name:        "unique count stores a mergeable state over the raw string",
			aggregation: meter.MeterAggregationUniqueCount,
			want:        []string{"uniqExactState(nullIf(JSON_VALUE(om_events.data, '$.value'), 'null')) AS uniq_state"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := meter.Meter{Aggregation: tt.aggregation}
			if tt.aggregation != meter.MeterAggregationCount {
				m.ValueProperty = lo.ToPtr("$.value")
			}

			got, err := valueExprsCombine(m, "om_events.data", "om_events.time")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("invalid aggregation", func(t *testing.T) {
		_, err := valueExprsCombine(meter.Meter{Aggregation: meter.MeterAggregation("MEDIAN")}, "om_events.data", "om_events.time")
		require.ErrorContains(t, err, "invalid aggregation type: MEDIAN")
	})

	t.Run("latest is not a valid combine-form aggregation", func(t *testing.T) {
		// LATEST is excluded from the cache entirely (meterCacheStaticReject); the combine
		// form must never be generated for it.
		_, err := valueExprsCombine(meter.Meter{Aggregation: meter.MeterAggregationLatest, ValueProperty: lo.ToPtr("$.value")}, "om_events.data", "om_events.time")
		require.ErrorContains(t, err, "invalid aggregation type: LATEST")
	})

	t.Run("missing value property", func(t *testing.T) {
		_, err := valueExprsCombine(meter.Meter{Aggregation: meter.MeterAggregationUniqueCount}, "om_events.data", "om_events.time")
		require.ErrorContains(t, err, "meter value property is required for UNIQUE_COUNT aggregation")
	})
}

func TestGroupBySelectExprs(t *testing.T) {
	meterGroupBy := map[string]string{
		"group1": "$.group1",
		"group2": "$.nested.group2",
	}

	t.Run("subject, customer_id and JSON dimensions", func(t *testing.T) {
		selectColumns, groupByColumns := groupBySelectExprs(
			[]string{"subject", "customer_id", "group1", "group2"},
			meterGroupBy,
			"om_events.subject",
			"om_events.data",
		)

		// customer_id contributes no SELECT expression: its select column (a
		// subject-to-customer map lookup) is attached by selectCustomerIdColumn.
		assert.Equal(t, []string{
			"om_events.subject",
			"JSON_VALUE(om_events.data, '$.group1') as group1",
			"JSON_VALUE(om_events.data, '$.nested.group2') as group2",
		}, selectColumns)
		assert.Equal(t, []string{"subject", "customer_id", "group1", "group2"}, groupByColumns)
	})

	t.Run("empty group by", func(t *testing.T) {
		selectColumns, groupByColumns := groupBySelectExprs(nil, meterGroupBy, "om_events.subject", "om_events.data")
		assert.Empty(t, selectColumns)
		assert.Empty(t, groupByColumns)
	})

	t.Run("JSON path literal is escaped", func(t *testing.T) {
		selectColumns, _ := groupBySelectExprs(
			[]string{"weird"},
			map[string]string{"weird": `$.it's`},
			"om_events.subject",
			"om_events.data",
		)
		assert.Equal(t, []string{`JSON_VALUE(om_events.data, '$.it\'s') as weird`}, selectColumns)
	})
}

func TestReservedAliasCheck(t *testing.T) {
	t.Run("accepts ordinary group by keys", func(t *testing.T) {
		require.NoError(t, reservedAliasCheck([]string{"model", "region", "valuex", "statement"}))
	})

	t.Run("accepts no keys", func(t *testing.T) {
		require.NoError(t, reservedAliasCheck(nil))
	})

	tests := []struct {
		name string
		key  string
	}{
		{name: "om_events column", key: "namespace"},
		{name: "om_events time column", key: "time"},
		{name: "cache column", key: "meter_hash"},
		{name: "query output alias", key: "value"},
		{name: "window alias", key: "windowstart"},
		{name: "value column family suffix", key: "total_value"},
		{name: "state column family suffix", key: "uniq_state"},
	}

	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			err := reservedAliasCheck([]string{"model", tt.key})
			require.ErrorContains(t, err, "meter group by keys collide with reserved SQL aliases: "+tt.key)
		})
	}

	t.Run("reports all offenders sorted", func(t *testing.T) {
		err := reservedAliasCheck([]string{"windowstart", "region", "data"})
		require.ErrorContains(t, err, "meter group by keys collide with reserved SQL aliases: data, windowstart")
	})
}
