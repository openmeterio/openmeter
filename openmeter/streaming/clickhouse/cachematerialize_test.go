package clickhouse

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

func TestMaterializeCacheRows(t *testing.T) {
	subject1 := "subject1"
	subject2 := "subject2"
	group1Value := "group1-value"

	windowStart1 := parseTime(t, "2025-01-01T00:00:00Z")
	windowEnd1 := parseTime(t, "2025-01-01T01:00:00Z")
	windowStart2 := parseTime(t, "2025-01-01T01:00:00Z")
	windowEnd2 := parseTime(t, "2025-01-01T02:00:00Z")
	windowStart3 := parseTime(t, "2025-01-01T02:00:00Z")
	windowEnd3 := parseTime(t, "2025-01-01T03:00:00Z")

	tests := []struct {
		name       string
		from       time.Time
		to         time.Time
		windowSize meterpkg.WindowSize
		rows       []meterpkg.MeterQueryRow
		want       []meterpkg.MeterQueryRow
	}{
		{
			name:       "no gaps do not materialize",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{}, // No gaps, so no materialized rows
		},
		{
			name:       "no gaps do not materialize with empty values",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       math.NaN(),
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       math.NaN(),
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{}, // No gaps, so no materialized rows
		},
		{
			name:       "no gaps do not materialize with subject",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{}, // No gaps, so no materialized rows
		},
		{
			name:       "no gaps do not materialize with group by",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       20,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			want: []meterpkg.MeterQueryRow{}, // No gaps, so no materialized rows
		},
		{
			name:       "gap at start materializes",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       10,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       math.NaN(),
					GroupBy:     map[string]*string{},
				},
			},
		},
		{
			name:       "gap in middle materializes",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       math.NaN(),
					GroupBy:     map[string]*string{},
				},
			},
		},
		{
			name:       "gap at end materializes",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       30,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       math.NaN(),
					GroupBy:     map[string]*string{},
				},
			},
		},
		{
			name:       "gap in middle with single subject materializes",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       math.NaN(),
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
			},
		},
		{
			name:       "gap in middle, multiple subjects",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					Subject:     &subject2,
					GroupBy:     map[string]*string{},
				},
			},
			want: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       math.NaN(),
					Subject:     &subject2,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       math.NaN(),
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       math.NaN(),
					Subject:     &subject2,
					GroupBy:     map[string]*string{},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       math.NaN(),
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
			},
		},
		{
			name:       "gap in middle, with group by",
			from:       windowStart1,
			to:         windowEnd3,
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart1,
					WindowEnd:   windowEnd1,
					Value:       10,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
				{
					WindowStart: windowStart3,
					WindowEnd:   windowEnd3,
					Value:       30,
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
			want: []meterpkg.MeterQueryRow{
				{
					WindowStart: windowStart2,
					WindowEnd:   windowEnd2,
					Value:       math.NaN(),
					Subject:     &subject1,
					GroupBy: map[string]*string{
						"group1": &group1Value,
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			connector := &Connector{}
			result, err := connector.materializeCacheRows(testCase.from, testCase.to, testCase.windowSize, testCase.rows)
			require.NoError(t, err)

			// Sort both slices by window start and subject for consistent comparison
			sort.Slice(result, func(i, j int) bool {
				if result[i].WindowStart.Equal(result[j].WindowStart) {
					if result[i].Subject == nil {
						return true
					}
					if result[j].Subject == nil {
						return false
					}
					return *result[i].Subject < *result[j].Subject
				}
				return result[i].WindowStart.Before(result[j].WindowStart)
			})

			sort.Slice(testCase.want, func(i, j int) bool {
				if testCase.want[i].WindowStart.Equal(testCase.want[j].WindowStart) {
					if testCase.want[i].Subject == nil {
						return true
					}
					if testCase.want[j].Subject == nil {
						return false
					}
					return *testCase.want[i].Subject < *testCase.want[j].Subject
				}
				return testCase.want[i].WindowStart.Before(testCase.want[j].WindowStart)
			})

			for i, row := range result {
				if i >= len(testCase.want) {
					assert.Fail(t, "result has more rows than want")
					continue
				}

				assert.Equal(t, testCase.want[i].WindowStart, row.WindowStart, "window start")
				assert.Equal(t, testCase.want[i].WindowEnd, row.WindowEnd, "window end")

				if math.IsNaN(testCase.want[i].Value) {
					assert.True(t, math.IsNaN(row.Value), "value")
				} else {
					assert.Equal(t, testCase.want[i].Value, row.Value, "value")
				}

				assert.Equal(t, testCase.want[i].Subject, row.Subject, "subject")
				assert.Equal(t, testCase.want[i].GroupBy, row.GroupBy, "group by")
			}
		})
	}
}
