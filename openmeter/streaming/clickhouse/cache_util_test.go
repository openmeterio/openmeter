package clickhouse

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
				{
					WindowStart: now.Add(time.Hour),
					WindowEnd:   now.Add(1 * time.Hour),
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
					WindowStart: now.Add(2 * time.Hour),
					WindowEnd:   now.Add(3 * time.Hour),
				},
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
