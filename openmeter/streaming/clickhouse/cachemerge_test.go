package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func TestMergeMeterQueryRows(t *testing.T) {
	tests := []struct {
		name        string
		meterDef    meterpkg.Meter
		queryParams streaming.QueryParams
		cachedRows  []meterpkg.MeterQueryRow
		freshRows   []meterpkg.MeterQueryRow
		wants       []meterpkg.MeterQueryRow
	}{
		{
			name: "empty cached rows",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{},
			cachedRows:  []meterpkg.MeterQueryRow{},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
			},
			wants: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
			},
		},
		{
			name: "with window size, rows are concatenated",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				WindowSize: lo.ToPtr(meterpkg.WindowSizeHour),
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T01:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       20,
					Subject:     lo.ToPtr("subject1"),
				},
			},
			wants: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
				{
					WindowStart: parseTime(t, "2023-01-01T01:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       20,
					Subject:     lo.ToPtr("subject1"),
				},
			},
		},
		{
			name: "without window size, sum aggregation",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T01:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       20,
					Subject:     lo.ToPtr("subject1"),
				},
			},
			wants: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       30,
					Subject:     lo.ToPtr("subject1"),
				},
			},
		},
		{
			name: "without window size, different subjects",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T01:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       20,
					Subject:     lo.ToPtr("subject2"),
				},
			},
			wants: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
				},
				{
					WindowStart: parseTime(t, "2023-01-01T01:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       20,
					Subject:     lo.ToPtr("subject2"),
				},
			},
		},
		{
			name: "without window size, with group by values",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T01:00:00Z"),
					Value:       10,
					Subject:     lo.ToPtr("subject1"),
					GroupBy: map[string]*string{
						"group1": lo.ToPtr("group1-value"),
						"group2": lo.ToPtr("group2-value"),
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T01:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       20,
					Subject:     lo.ToPtr("subject1"),
					GroupBy: map[string]*string{
						"group1": lo.ToPtr("group1-value"),
						"group2": lo.ToPtr("group2-value"),
					},
				},
			},
			wants: []meterpkg.MeterQueryRow{
				{
					WindowStart: parseTime(t, "2023-01-01T00:00:00Z"),
					WindowEnd:   parseTime(t, "2023-01-01T02:00:00Z"),
					Value:       30,
					Subject:     lo.ToPtr("subject1"),
					GroupBy: map[string]*string{
						"group1": lo.ToPtr("group1-value"),
						"group2": lo.ToPtr("group2-value"),
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := mergeMeterQueryRows(testCase.meterDef, testCase.queryParams, append(testCase.cachedRows, testCase.freshRows...))
			assert.Equal(t, testCase.wants, result)

			if testCase.meterDef.Aggregation == meterpkg.MeterAggregationSum && len(testCase.queryParams.GroupBy) > 0 && testCase.queryParams.WindowSize == nil {
				// If we're aggregating, check that values are summed
				if len(result) == 1 && len(testCase.cachedRows) > 0 && len(testCase.freshRows) > 0 {
					expectedSum := testCase.cachedRows[0].Value + testCase.freshRows[0].Value
					assert.Equal(t, expectedSum, result[0].Value)
				}
			}
		})
	}
}

func TestCreateGroupKeyFromRow(t *testing.T) {
	subject := "test-subject"

	testRow := meterpkg.MeterQueryRow{
		Subject: &subject,
		GroupBy: map[string]*string{
			"group1": lo.ToPtr("group1-value"),
			"group2": lo.ToPtr("group2-value"),
		},
	}

	tests := []struct {
		name        string
		queryParams streaming.QueryParams
		expectedKey string
	}{
		{
			name: "subject only",
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			expectedKey: "subject=test-subject;group=subject=nil;",
		},
		{
			name: "with group by fields",
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			expectedKey: "subject=test-subject;group=group1=group1-value;group=group2=group2-value;group=subject=nil;",
		},
		{
			name: "with missing group by field",
			queryParams: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group3"},
			},
			expectedKey: "subject=test-subject;group=group1=group1-value;group=group3=nil;group=subject=nil;",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := createGroupKeyFromRowWithQueryParams(testRow, testCase.queryParams)
			assert.Equal(t, testCase.expectedKey, result)
		})
	}
}

// TODO: implement
func TestCreateGroupKeyFromRowWithQueryParams(t *testing.T) {
}

func TestAggregateRowsByAggregationType(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	// Rows have the same subject and groupBy values
	testRows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": lo.ToPtr("group1-value"),
			},
		},
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       20,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": lo.ToPtr("group1-value"),
			},
		},
	}

	tests := []struct {
		name        string
		aggregation meterpkg.MeterAggregation
		rows        []meterpkg.MeterQueryRow
		wantValue   float64
		wantSubject string
	}{
		{
			name:        "sum aggregation",
			aggregation: meterpkg.MeterAggregationSum,
			rows:        testRows,
			wantValue:   30, // 10 + 20
			wantSubject: subject,
		},
		{
			name:        "count aggregation",
			aggregation: meterpkg.MeterAggregationCount,
			rows:        testRows,
			wantValue:   30, // count should be the same as sum
			wantSubject: subject,
		},
		{
			name:        "min aggregation",
			aggregation: meterpkg.MeterAggregationMin,
			rows:        testRows,
			wantValue:   10, // min of 10 and 20
			wantSubject: subject,
		},
		{
			name:        "max aggregation",
			aggregation: meterpkg.MeterAggregationMax,
			rows:        testRows,
			wantValue:   20, // max of 10 and 20
			wantSubject: subject,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := aggregateRowsByAggregationType(testCase.aggregation, testCase.rows)

			assert.Equal(t, testCase.wantValue, result.Value)
			require.NotNil(t, result.Subject)
			assert.Equal(t, testCase.wantSubject, *result.Subject)

			// Window range should span from earliest to latest
			assert.Equal(t, windowStart1, result.WindowStart)
			assert.Equal(t, windowEnd2, result.WindowEnd)

			// GroupBy values should be preserved
			require.Contains(t, result.GroupBy, "group1")
			require.NotNil(t, result.GroupBy["group1"])
			assert.Equal(t, group1Value, *result.GroupBy["group1"])
		})
	}
}

func TestDedupeQueryRows(t *testing.T) {
	group1Key := "group1"
	group2Key := "group2"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	rows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     lo.ToPtr("subject1"),
			GroupBy: map[string]*string{
				group1Key: lo.ToPtr("group-1"),
			},
		},
		// Duplicate row
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     lo.ToPtr("subject1"),
			GroupBy: map[string]*string{
				group1Key: lo.ToPtr("group-1"),
			},
		},
		// Row with different group by value
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     lo.ToPtr("subject1"),
			GroupBy: map[string]*string{
				group2Key: lo.ToPtr("group-2"),
			},
		},
		// Row with different time
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       10,
			Subject:     lo.ToPtr("subject1"),
			GroupBy: map[string]*string{
				group1Key: lo.ToPtr("group-1"),
			},
		},
		// Row with different subject
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     lo.ToPtr("subject2"),
			GroupBy: map[string]*string{
				group1Key: lo.ToPtr("group-1"),
			},
		},
	}

	deduplicatedRows, err := dedupeQueryRows(rows, []string{group1Key, group2Key})
	require.NoError(t, err)

	assert.Equal(t, 4, len(deduplicatedRows))
	assert.Equal(t, deduplicatedRows, []meterpkg.MeterQueryRow{
		rows[0],
		rows[2],
		rows[3],
		rows[4],
	})

	// Test duplicates with inconsistent value
	rows[0].Value = 20
	_, err = dedupeQueryRows(rows, []string{group1Key, group2Key})
	require.Error(t, err)
}

func parseTime(t *testing.T, timeStr string) time.Time {
	time, err := time.Parse(time.RFC3339, timeStr)
	require.NoError(t, err)
	return time
}
