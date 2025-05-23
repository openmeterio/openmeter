package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func TestMergeMeterQueryRows(t *testing.T) {
	subject1 := "subject1"
	subject2 := "subject2"
	group1Value := "group1_value"
	group2Value := "group2_value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	windowSize := meterpkg.WindowSizeHour

	tests := []struct {
		name        string
		meterDef    meterpkg.Meter
		queryParams streaming.QueryParams
		cachedRows  []meterpkg.MeterQueryRow
		freshRows   []meterpkg.MeterQueryRow
		wantCount   int
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
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			wantCount: 1,
		},
		{
			name: "with window size, rows are concatenated",
			meterDef: meterpkg.Meter{
				Aggregation: meterpkg.MeterAggregationSum,
			},
			queryParams: streaming.QueryParams{
				WindowSize: &windowSize,
			},
			cachedRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
				},
			},
			wantCount: 2,
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
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
				},
			},
			wantCount: 1, // Aggregated to a single row
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
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject2,
				},
			},
			wantCount: 2, // One row per subject
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
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
						"group2": &group2Value,
					},
				},
			},
			freshRows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
						"group2": &group2Value,
					},
				},
			},
			wantCount: 1, // Aggregated by groups
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := mergeMeterQueryRows(testCase.meterDef, testCase.queryParams, testCase.cachedRows, testCase.freshRows)
			assert.Equal(t, testCase.wantCount, len(result))

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
	group1Value := "group1-value"
	group2Value := "group2-value"

	testRow := meterpkg.MeterQueryRow{
		Subject: &subject,
		GroupBy: map[string]*string{
			"group1": &group1Value,
			"group2": &group2Value,
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
				"group1": &group1Value,
			},
		},
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       20,
			Subject:     &subject,
			GroupBy: map[string]*string{
				"group1": &group1Value,
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
	subject1 := "test-subject"
	subject2 := "test-subject-2"
	group1Key := "group1"
	group1Value := "group1-value"
	group2Key := "group2"
	group2Value := "group2-value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	rows := []meterpkg.MeterQueryRow{
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
		// Duplicate row
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
		// Row with different group by value
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group2Key: &group2Value,
			},
		},
		// Row with different time
		{
			WindowStart: windowStart2,
			WindowEnd:   windowEnd2,
			Value:       10,
			Subject:     &subject1,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
			},
		},
		// Row with different subject
		{
			WindowStart: windowStart1,
			WindowEnd:   windowEnd1,
			Value:       10,
			Subject:     &subject2,
			GroupBy: map[string]*string{
				group1Key: &group1Value,
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
