package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
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

	windowSize := meter.WindowSizeHour

	tests := []struct {
		name       string
		meter      meter.Meter
		params     streaming.QueryParams
		cachedRows []meterpkg.MeterQueryRow
		freshRows  []meterpkg.MeterQueryRow
		wantCount  int
	}{
		{
			name: "empty cached rows",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params:     streaming.QueryParams{},
			cachedRows: []meterpkg.MeterQueryRow{},
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
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
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
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
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
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
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
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			params: streaming.QueryParams{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMeterQueryRows(tt.meter, tt.params, tt.cachedRows, tt.freshRows)
			assert.Equal(t, tt.wantCount, len(result))

			if tt.meter.Aggregation == meter.MeterAggregationSum && len(tt.params.GroupBy) > 0 && tt.params.WindowSize == nil {
				// If we're aggregating, check that values are summed
				if len(result) == 1 && len(tt.cachedRows) > 0 && len(tt.freshRows) > 0 {
					expectedSum := tt.cachedRows[0].Value + tt.freshRows[0].Value
					assert.Equal(t, expectedSum, result[0].Value)
				}
			}
		})
	}
}

func TestGetMeterQueryRowKey(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"
	group2Value := "group2-value"

	row := meterpkg.MeterQueryRow{
		Subject: &subject,
		GroupBy: map[string]*string{
			"group1": &group1Value,
			"group2": &group2Value,
		},
	}

	tests := []struct {
		name   string
		params streaming.QueryParams
		want   string
	}{
		{
			name: "subject only",
			params: streaming.QueryParams{
				GroupBy: []string{"subject"},
			},
			want: "subject=test-subject;group=subject=nil;",
		},
		{
			name: "with group by fields",
			params: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group2"},
			},
			want: "subject=test-subject;group=group1=group1-value;group=group2=group2-value;group=subject=nil;",
		},
		{
			name: "with missing group by field",
			params: streaming.QueryParams{
				GroupBy: []string{"subject", "group1", "group3"},
			},
			want: "subject=test-subject;group=group1=group1-value;group=group3=nil;group=subject=nil;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMeterQueryRowKey(row, tt.params)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAggregateMeterQueryRows(t *testing.T) {
	subject := "test-subject"
	group1Value := "group1-value"

	windowStart1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	windowEnd1, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowStart2, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	windowEnd2, _ := time.Parse(time.RFC3339, "2023-01-01T02:00:00Z")

	// Rows have the same subject and groupBy values
	rows := []meterpkg.MeterQueryRow{
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
		meter       meter.Meter
		rows        []meterpkg.MeterQueryRow
		wantValue   float64
		wantSubject string
	}{
		{
			name: "sum aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationSum,
			},
			rows:        rows,
			wantValue:   30, // 10 + 20
			wantSubject: subject,
		},
		{
			name: "count aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationCount,
			},
			rows:        rows,
			wantValue:   30, // count should be the same as sum
			wantSubject: subject,
		},
		{
			name: "min aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationMin,
			},
			rows:        rows,
			wantValue:   10, // min of 10 and 20
			wantSubject: subject,
		},
		{
			name: "max aggregation",
			meter: meter.Meter{
				Aggregation: meter.MeterAggregationMax,
			},
			rows:        rows,
			wantValue:   20, // max of 10 and 20
			wantSubject: subject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregateMeterQueryRows(tt.meter, tt.rows)

			assert.Equal(t, tt.wantValue, result.Value)
			require.NotNil(t, result.Subject)
			assert.Equal(t, tt.wantSubject, *result.Subject)

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
