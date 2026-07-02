package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// TestMeterShapeHash pins the fingerprint that isolates cache rows per meter
// extraction shape. Any definition change that alters what a row's value or
// group_by array MEANS must change the hash (so old rows orphan instead of
// double-counting), while semantically irrelevant differences must not.
func TestMeterShapeHash(t *testing.T) {
	base := meterpkg.Meter{
		Key:           "m1",
		EventType:     "t",
		Aggregation:   meterpkg.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       map[string]string{"model": "$.model", "region": "$.region"},
	}

	require.Equal(t, meterShapeHash(base), meterShapeHash(base), "hash must be deterministic")

	// Same content, different map construction order — Go map iteration order is
	// random, so this also guards the sorted-key alignment.
	reordered := base
	reordered.GroupBy = map[string]string{"region": "$.region", "model": "$.model"}
	require.Equal(t, meterShapeHash(base), meterShapeHash(reordered), "group-by map order must not change the hash")

	addedDim := base
	addedDim.GroupBy = map[string]string{"model": "$.model", "region": "$.region", "tier": "$.tier"}
	require.NotEqual(t, meterShapeHash(base), meterShapeHash(addedDim), "adding a group-by dimension must change the hash")

	changedPath := base
	changedPath.GroupBy = map[string]string{"model": "$.model_id", "region": "$.region"}
	require.NotEqual(t, meterShapeHash(base), meterShapeHash(changedPath), "changing a dimension's JSON path must change the hash")

	changedValue := base
	changedValue.ValueProperty = lo.ToPtr("$.tokens")
	require.NotEqual(t, meterShapeHash(base), meterShapeHash(changedValue), "changing the value property must change the hash")

	changedAgg := base
	changedAgg.Aggregation = meterpkg.MeterAggregationMax
	require.NotEqual(t, meterShapeHash(base), meterShapeHash(changedAgg), "changing the aggregation must change the hash")

	// A meter deleted and recreated under the same slug with a different event
	// type must not inherit the old rows or coverage claim: the cache leg
	// filters rows by type, so an inherited claim would vouch for rows the
	// read can never see.
	changedType := base
	changedType.EventType = "t2"
	require.NotEqual(t, meterShapeHash(base), meterShapeHash(changedType), "changing the event type must change the hash")
}

func TestIsCacheableWindowSize(t *testing.T) {
	ws := func(s meterpkg.WindowSize) *meterpkg.WindowSize { return &s }

	require.True(t, isCacheableWindowSize(nil), "total (nil) is cacheable")
	require.True(t, isCacheableWindowSize(ws(meterpkg.WindowSizeHour)))
	require.True(t, isCacheableWindowSize(ws(meterpkg.WindowSizeDay)))
	require.True(t, isCacheableWindowSize(ws(meterpkg.WindowSizeMonth)))
	require.False(t, isCacheableWindowSize(ws(meterpkg.WindowSizeMinute)), "minute is finer than the hourly grain")
}

func TestIsWholeHourTimeZone(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 30)

	require.True(t, isWholeHourTimeZone(nil, from, to), "nil zone (UTC) is whole-hour")
	require.True(t, isWholeHourTimeZone(time.UTC, from, to), "UTC is whole-hour")

	// Whole-hour zone year-round.
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	require.True(t, isWholeHourTimeZone(ny, from, to), "America/New_York is a whole-hour offset")

	// Fractional-hour zone — cannot be composed from hourly-UTC windows.
	kolkata, err := time.LoadLocation("Asia/Kolkata")
	require.NoError(t, err)
	require.False(t, isWholeHourTimeZone(kolkata, from, to), "Asia/Kolkata (+05:30) is fractional-hour")

	kathmandu, err := time.LoadLocation("Asia/Kathmandu")
	require.NoError(t, err)
	require.False(t, isWholeHourTimeZone(kathmandu, from, to), "Asia/Kathmandu (+05:45) is fractional-hour")
}

func TestAggCacheColumn(t *testing.T) {
	cases := []struct {
		agg       meterpkg.MeterAggregation
		column    string
		recombine string
		ok        bool
	}{
		{meterpkg.MeterAggregationSum, "sum_value", "sum", true},
		{meterpkg.MeterAggregationCount, "count_value", "sum", true},
		{meterpkg.MeterAggregationMin, "min_value", "min", true},
		{meterpkg.MeterAggregationMax, "max_value", "max", true},
		{meterpkg.MeterAggregationAvg, "", "", false},
		{meterpkg.MeterAggregationUniqueCount, "", "", false},
		{meterpkg.MeterAggregationLatest, "", "", false},
	}

	for _, tc := range cases {
		col, rec, ok := aggCacheColumn(tc.agg)
		require.Equal(t, tc.ok, ok, "agg %s ok", tc.agg)
		require.Equal(t, tc.column, col, "agg %s column", tc.agg)
		require.Equal(t, tc.recombine, rec, "agg %s recombine", tc.agg)
	}
}
