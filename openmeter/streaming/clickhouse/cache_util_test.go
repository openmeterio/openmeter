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

func TestFilterOutNaNValues(t *testing.T) {
	tests := []struct {
		name     string
		input    []meterpkg.MeterQueryRow
		expected []meterpkg.MeterQueryRow
	}{
		{
			name:     "empty slice",
			input:    []meterpkg.MeterQueryRow{},
			expected: []meterpkg.MeterQueryRow{},
		},
		{
			name: "no NaN values",
			input: []meterpkg.MeterQueryRow{
				{Value: 1.0},
				{Value: 2.0},
				{Value: 3.0},
			},
			expected: []meterpkg.MeterQueryRow{
				{Value: 1.0},
				{Value: 2.0},
				{Value: 3.0},
			},
		},
		{
			name: "with NaN values",
			input: []meterpkg.MeterQueryRow{
				{Value: 1.0},
				{Value: math.NaN()},
				{Value: 3.0},
				{Value: math.NaN()},
			},
			expected: []meterpkg.MeterQueryRow{
				{Value: 1.0},
				{Value: 3.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOutNaNValues(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTimeWindowGap(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		from       time.Time
		to         time.Time
		windowSize meterpkg.WindowSize
		rows       []meterpkg.MeterQueryRow
		expected   bool
	}{
		{
			name:       "empty rows",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			rows:       []meterpkg.MeterQueryRow{},
			expected:   false,
		},
		{
			name:       "no gap",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
				},
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
				},
			},
			expected: false,
		},
		{
			name:       "gap in middle counts as gap",
			from:       now,
			to:         now.Add(3 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: now.Add(0 * time.Hour),
					WindowEnd:   now.Add(1 * time.Hour),
				},
				// Gap in the middle:
				// {
				// 	WindowStart: now.Add(1 * time.Hour),
				// 	WindowEnd:   now.Add(2 * time.Hour),
				// },
				{
					WindowStart: now.Add(2 * time.Hour),
					WindowEnd:   now.Add(3 * time.Hour),
				},
			},
			expected: true,
		},
		{
			name:       "gap at start does not count as gap",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				// Gap at start:
				// {
				// 	WindowStart: now.Add(0 * time.Hour),
				// 	WindowEnd:   now.Add(1 * time.Hour),
				// },
				{
					WindowStart: now.Add(1 * time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
				},
			},
			expected: false,
		},
		{
			name:       "gap at end does not count as gap",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			rows: []meterpkg.MeterQueryRow{
				{
					WindowStart: now.Add(0 * time.Hour),
					WindowEnd:   now.Add(1 * time.Hour),
				},
				// Gap at end:
				// {
				// 	WindowStart: now.Add(1 * time.Hour),
				// 	WindowEnd:   now.Add(2 * time.Hour),
				// },
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeWindowGap(tt.from, tt.to, tt.windowSize, tt.rows)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConcatAppend(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]int
		expected []int
	}{
		{
			name:     "empty slices",
			input:    [][]int{},
			expected: []int{},
		},
		{
			name: "single slice",
			input: [][]int{
				{1, 2, 3},
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "multiple slices",
			input: [][]int{
				{1, 2, 3},
				{4, 5, 6},
				{7, 8, 9},
			},
			expected: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name: "empty and non-empty slices",
			input: [][]int{
				{},
				{1, 2, 3},
				{},
				{4, 5, 6},
			},
			expected: []int{1, 2, 3, 4, 5, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := concatAppend(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterRowsOutOfPeriod(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		from       time.Time
		to         time.Time
		windowSize meterpkg.WindowSize
		input      []meterpkg.MeterQueryRow
		expected   []meterpkg.MeterQueryRow
	}{
		{
			name:       "empty input",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			input:      []meterpkg.MeterQueryRow{},
			expected:   []meterpkg.MeterQueryRow{},
		},
		{
			name:       "rows within period",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			input: []meterpkg.MeterQueryRow{
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       1.0,
				},
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
					Value:       2.0,
				},
			},
			expected: []meterpkg.MeterQueryRow{
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       1.0,
				},
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
					Value:       2.0,
				},
			},
		},
		{
			name:       "filter outrows before period",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			input: []meterpkg.MeterQueryRow{
				{
					WindowStart: now.Add(-2 * time.Hour),
					WindowEnd:   now.Add(-1 * time.Hour),
					Value:       1.0,
				},
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       2.0,
				},
			},
			expected: []meterpkg.MeterQueryRow{
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       2.0,
				},
			},
		},
		{
			name:       "filter out rows after period",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			input: []meterpkg.MeterQueryRow{
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       1.0,
				},
				{
					WindowStart: now.Add(2 * time.Hour),
					WindowEnd:   now.Add(3 * time.Hour),
					Value:       2.0,
				},
			},
			expected: []meterpkg.MeterQueryRow{
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       1.0,
				},
			},
		},
		{
			name:       "filter out rows with incomplete windows",
			from:       now,
			to:         now.Add(2 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			input: []meterpkg.MeterQueryRow{
				{
					WindowStart: now.Add(15 * time.Minute), // Incomplete window
					WindowEnd:   now.Add(1 * time.Hour),
					Value:       1.0,
				},
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
					Value:       2.0,
				},
			},
			expected: []meterpkg.MeterQueryRow{
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
					Value:       2.0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterRowsOutOfPeriod(tt.from, tt.to, tt.windowSize, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterAndMaterializeRows(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	subject1 := "subject1"

	tests := []struct {
		name       string
		from       time.Time
		to         time.Time
		windowSize meterpkg.WindowSize
		input      []meterpkg.MeterQueryRow
		expected   []meterpkg.MeterQueryRow
	}{
		{
			name:       "filter out periods and materialize gaps",
			from:       now,
			to:         now.Add(3 * time.Hour),
			windowSize: meterpkg.WindowSizeHour,
			input: []meterpkg.MeterQueryRow{
				// Row before period - should be filtered out
				{
					WindowStart: now.Add(-1 * time.Hour),
					WindowEnd:   now,
					Value:       1.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				// First hour - should be kept
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       2.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				// Incomplete window - should be filtered out
				{
					WindowStart: now.Add(90 * time.Minute),
					WindowEnd:   now.Add(2 * time.Hour),
					Value:       3.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				// Last hour - should be kept
				{
					WindowStart: now.Add(2 * time.Hour),
					WindowEnd:   now.Add(3 * time.Hour),
					Value:       4.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				// Row after period - should be filtered out
				{
					WindowStart: now.Add(3 * time.Hour),
					WindowEnd:   now.Add(4 * time.Hour),
					Value:       5.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
			},
			expected: []meterpkg.MeterQueryRow{
				// Original first hour
				{
					WindowStart: now,
					WindowEnd:   now.Add(time.Hour),
					Value:       2.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				// Materialized gap hour
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(2 * time.Hour),
					Value:       cacheNoValue,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
				// Original last hour
				{
					WindowStart: now.Add(2 * time.Hour),
					WindowEnd:   now.Add(3 * time.Hour),
					Value:       4.0,
					Subject:     &subject1,
					GroupBy:     map[string]*string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First filter out rows that are out of period
			filteredRows := filterRowsOutOfPeriod(tt.from, tt.to, tt.windowSize, tt.input)

			// Then materialize any gaps
			connector := &Connector{}
			materializedRows, err := connector.materializeCacheRows(tt.from, tt.to, tt.windowSize, filteredRows)
			require.NoError(t, err)

			// Combine the filtered and materialized rows
			result := append(filteredRows, materializedRows...)

			// Sort both slices by window start, subject, and group by for consistent comparison
			sort.Slice(result, func(i, j int) bool {
				if result[i].WindowStart.Equal(result[j].WindowStart) {
					if result[i].Subject == nil {
						return true
					}
					if result[j].Subject == nil {
						return false
					}
					if *result[i].Subject != *result[j].Subject {
						return *result[i].Subject < *result[j].Subject
					}
					// Compare group by values if subjects are equal
					iKey := createGroupKeyFromRow(result[i], []string{"group1"})
					jKey := createGroupKeyFromRow(result[j], []string{"group1"})
					return iKey < jKey
				}
				return result[i].WindowStart.Before(result[j].WindowStart)
			})

			// Compare the results
			require.Equal(t, len(tt.expected), len(result), "number of rows should match")
			for i, row := range result {
				rowEqual(t, tt.expected[i], row)
			}
		})
	}
}

// rowEqual compares two MeterQueryRow objects and asserts that they are equal
func rowEqual(t *testing.T, expected, actual meterpkg.MeterQueryRow) {
	assert.Equal(t, expected.WindowStart, actual.WindowStart, "window start")
	assert.Equal(t, expected.WindowEnd, actual.WindowEnd, "window end")

	if math.IsNaN(expected.Value) {
		assert.True(t, math.IsNaN(actual.Value), "value should be NaN")
	} else {
		assert.Equal(t, expected.Value, actual.Value, "value")
	}

	assert.Equal(t, expected.Subject, actual.Subject, "subject")
	assert.Equal(t, expected.GroupBy, actual.GroupBy, "group by")
}
