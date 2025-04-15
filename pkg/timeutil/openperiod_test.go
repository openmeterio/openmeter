package timeutil

import (
	"testing"
	"time"
)

func TestOpenPeriod(t *testing.T) {
	now := time.Now()
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)
	thirtyMinLater := now.Add(30 * time.Minute)

	t.Run("ContainsInclusive", func(t *testing.T) {
		tests := []struct {
			name     string
			period   OpenPeriod
			testTime time.Time
			want     bool
		}{
			{
				name:     "both bounds nil",
				period:   OpenPeriod{},
				testTime: now,
				want:     true,
			},
			{
				name:     "from bound only, time after",
				period:   OpenPeriod{From: &before},
				testTime: now,
				want:     true,
			},
			{
				name:     "from bound only, time before",
				period:   OpenPeriod{From: &now},
				testTime: before,
				want:     false,
			},
			{
				name:     "from bound only, time equal",
				period:   OpenPeriod{From: &now},
				testTime: now,
				want:     true,
			},
			{
				name:     "to bound only, time before",
				period:   OpenPeriod{To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "to bound only, time after",
				period:   OpenPeriod{To: &now},
				testTime: after,
				want:     false,
			},
			{
				name:     "to bound only, time equal",
				period:   OpenPeriod{To: &now},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time inside",
				period:   OpenPeriod{From: &before, To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time equal to from",
				period:   OpenPeriod{From: &now, To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time equal to to",
				period:   OpenPeriod{From: &before, To: &now},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time outside before",
				period:   OpenPeriod{From: &now, To: &after},
				testTime: before,
				want:     false,
			},
			{
				name:     "both bounds, time outside after",
				period:   OpenPeriod{From: &before, To: &now},
				testTime: after,
				want:     false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.period.ContainsInclusive(tt.testTime); got != tt.want {
					t.Errorf("OpenPeriod.ContainsInclusive() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("ContainsExclusive", func(t *testing.T) {
		tests := []struct {
			name     string
			period   OpenPeriod
			testTime time.Time
			want     bool
		}{
			{
				name:     "both bounds nil",
				period:   OpenPeriod{},
				testTime: now,
				want:     true,
			},
			{
				name:     "from bound only, time after",
				period:   OpenPeriod{From: &before},
				testTime: now,
				want:     true,
			},
			{
				name:     "from bound only, time before",
				period:   OpenPeriod{From: &now},
				testTime: before,
				want:     false,
			},
			{
				name:     "from bound only, time equal",
				period:   OpenPeriod{From: &now},
				testTime: now,
				want:     false,
			},
			{
				name:     "to bound only, time before",
				period:   OpenPeriod{To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "to bound only, time after",
				period:   OpenPeriod{To: &now},
				testTime: after,
				want:     false,
			},
			{
				name:     "to bound only, time equal",
				period:   OpenPeriod{To: &now},
				testTime: now,
				want:     false,
			},
			{
				name:     "both bounds, time inside",
				period:   OpenPeriod{From: &before, To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time equal to from",
				period:   OpenPeriod{From: &now, To: &after},
				testTime: now,
				want:     false,
			},
			{
				name:     "both bounds, time equal to to",
				period:   OpenPeriod{From: &before, To: &now},
				testTime: now,
				want:     false,
			},
			{
				name:     "both bounds, time outside before",
				period:   OpenPeriod{From: &now, To: &after},
				testTime: before,
				want:     false,
			},
			{
				name:     "both bounds, time outside after",
				period:   OpenPeriod{From: &before, To: &now},
				testTime: after,
				want:     false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.period.ContainsExclusive(tt.testTime); got != tt.want {
					t.Errorf("OpenPeriod.ContainsExclusive() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Contains", func(t *testing.T) {
		tests := []struct {
			name     string
			period   OpenPeriod
			testTime time.Time
			want     bool
		}{
			{
				name:     "both bounds nil",
				period:   OpenPeriod{},
				testTime: now,
				want:     true,
			},
			{
				name:     "from bound only, time after",
				period:   OpenPeriod{From: &before},
				testTime: now,
				want:     true,
			},
			{
				name:     "from bound only, time before",
				period:   OpenPeriod{From: &now},
				testTime: before,
				want:     false,
			},
			{
				name:     "from bound only, time equal",
				period:   OpenPeriod{From: &now},
				testTime: now,
				want:     true,
			},
			{
				name:     "to bound only, time before",
				period:   OpenPeriod{To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "to bound only, time after",
				period:   OpenPeriod{To: &now},
				testTime: after,
				want:     false,
			},
			{
				name:     "to bound only, time equal",
				period:   OpenPeriod{To: &now},
				testTime: now,
				want:     false,
			},
			{
				name:     "both bounds, time inside",
				period:   OpenPeriod{From: &before, To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time equal to from",
				period:   OpenPeriod{From: &now, To: &after},
				testTime: now,
				want:     true,
			},
			{
				name:     "both bounds, time equal to to",
				period:   OpenPeriod{From: &before, To: &now},
				testTime: now,
				want:     false,
			},
			{
				name:     "both bounds, time outside before",
				period:   OpenPeriod{From: &now, To: &after},
				testTime: before,
				want:     false,
			},
			{
				name:     "both bounds, time outside after",
				period:   OpenPeriod{From: &before, To: &now},
				testTime: after,
				want:     false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.period.Contains(tt.testTime); got != tt.want {
					t.Errorf("OpenPeriod.Contains() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Intersection", func(t *testing.T) {
		tests := []struct {
			name     string
			period1  OpenPeriod
			period2  OpenPeriod
			expected *OpenPeriod
		}{
			{
				name:     "both periods empty",
				period1:  OpenPeriod{},
				period2:  OpenPeriod{},
				expected: &OpenPeriod{},
			},
			{
				name:     "first period empty",
				period1:  OpenPeriod{},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: &OpenPeriod{From: &before, To: &after},
			},
			{
				name:     "second period empty",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{},
				expected: &OpenPeriod{From: &before, To: &after},
			},
			{
				name:     "overlapping periods",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: &OpenPeriod{From: &now, To: &after},
			},
			{
				name:     "non-overlapping periods",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &after, To: nil},
				expected: nil,
			},
			{
				name:     "period1 contains period2",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &now, To: &thirtyMinLater},
				expected: &OpenPeriod{From: &now, To: &thirtyMinLater},
			},
			{
				name:     "period2 contains period1",
				period1:  OpenPeriod{From: &now, To: &thirtyMinLater},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: &OpenPeriod{From: &now, To: &thirtyMinLater},
			},
			{
				name:     "touching periods (no overlap)",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &now, To: &after},
				expected: nil,
			},
			{
				name:     "both periods open-ended in same direction",
				period1:  OpenPeriod{From: &before, To: nil},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: &OpenPeriod{From: &now, To: nil},
			},
			{
				name:     "both periods open-ended in opposite directions",
				period1:  OpenPeriod{From: nil, To: &now},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.period1.Intersection(tt.period2)

				if tt.expected == nil {
					if result != nil {
						t.Errorf("Expected nil result, got %+v", *result)
					}
					return
				}

				if result == nil {
					t.Errorf("Expected non-nil result %+v, got nil", *tt.expected)
					return
				}

				// Check From value
				if (tt.expected.From == nil) != (result.From == nil) {
					t.Errorf("Incorrect From nil status, expected %v, got %v", tt.expected.From == nil, result.From == nil)
				} else if tt.expected.From != nil && result.From != nil && !tt.expected.From.Equal(*result.From) {
					t.Errorf("Incorrect From value, expected %v, got %v", *tt.expected.From, *result.From)
				}

				// Check To value
				if (tt.expected.To == nil) != (result.To == nil) {
					t.Errorf("Incorrect To nil status, expected %v, got %v", tt.expected.To == nil, result.To == nil)
				} else if tt.expected.To != nil && result.To != nil && !tt.expected.To.Equal(*result.To) {
					t.Errorf("Incorrect To value, expected %v, got %v", *tt.expected.To, *result.To)
				}
			})
		}
	})

	t.Run("Union", func(t *testing.T) {
		tests := []struct {
			name     string
			period1  OpenPeriod
			period2  OpenPeriod
			expected OpenPeriod
		}{
			{
				name:     "both periods empty",
				period1:  OpenPeriod{},
				period2:  OpenPeriod{},
				expected: OpenPeriod{},
			},
			{
				name:     "first period empty",
				period1:  OpenPeriod{},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: OpenPeriod{},
			},
			{
				name:     "second period empty",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{},
				expected: OpenPeriod{},
			},
			{
				name:     "overlapping periods",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: OpenPeriod{From: &before, To: nil},
			},
			{
				name:     "non-overlapping periods",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &after, To: nil},
				expected: OpenPeriod{From: &before, To: nil},
			},
			{
				name:     "period1 contains period2",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &now, To: &thirtyMinLater},
				expected: OpenPeriod{From: &before, To: &after},
			},
			{
				name:     "period2 contains period1",
				period1:  OpenPeriod{From: &now, To: &thirtyMinLater},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: OpenPeriod{From: &before, To: &after},
			},
			{
				name:     "touching periods",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &now, To: &after},
				expected: OpenPeriod{From: &before, To: &after},
			},
			{
				name:     "both periods open-ended in same direction",
				period1:  OpenPeriod{From: &before, To: nil},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: OpenPeriod{From: &before, To: nil},
			},
			{
				name:     "both periods open-ended in opposite directions",
				period1:  OpenPeriod{From: nil, To: &now},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: OpenPeriod{From: nil, To: nil},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.period1.Union(tt.period2)

				// Check From value
				if (tt.expected.From == nil) != (result.From == nil) {
					t.Errorf("Incorrect From nil status, expected %v, got %v", tt.expected.From == nil, result.From == nil)
				} else if tt.expected.From != nil && result.From != nil && !tt.expected.From.Equal(*result.From) {
					t.Errorf("Incorrect From value, expected %v, got %v", *tt.expected.From, *result.From)
				}

				// Check To value
				if (tt.expected.To == nil) != (result.To == nil) {
					t.Errorf("Incorrect To nil status, expected %v, got %v", tt.expected.To == nil, result.To == nil)
				} else if tt.expected.To != nil && result.To != nil && !tt.expected.To.Equal(*result.To) {
					t.Errorf("Incorrect To value, expected %v, got %v", *tt.expected.To, *result.To)
				}
			})
		}
	})

	t.Run("IsSupersetOf", func(t *testing.T) {
		tests := []struct {
			name     string
			period1  OpenPeriod
			period2  OpenPeriod
			expected bool
		}{
			{
				name:     "empty period is superset of empty period",
				period1:  OpenPeriod{},
				period2:  OpenPeriod{},
				expected: true,
			},
			{
				name:     "empty period is superset of non-empty period",
				period1:  OpenPeriod{},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: true,
			},
			{
				name:     "non-empty period is not superset of empty period",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{},
				expected: false,
			},
			{
				name:     "period contains other period",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &now, To: &thirtyMinLater},
				expected: true,
			},
			{
				name:     "period does not contain other period (starts after)",
				period1:  OpenPeriod{From: &now, To: &after},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: false,
			},
			{
				name:     "period does not contain other period (ends before)",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: false,
			},
			{
				name:     "period with open end contains period with closed end",
				period1:  OpenPeriod{From: &before, To: nil},
				period2:  OpenPeriod{From: &now, To: &after},
				expected: true,
			},
			{
				name:     "period with open end contains period with closed end (same time)",
				period1:  OpenPeriod{From: &now, To: nil},
				period2:  OpenPeriod{From: &now, To: &after},
				expected: true,
			},
			{
				name:     "period with open start contains period with closed start",
				period1:  OpenPeriod{From: nil, To: &after},
				period2:  OpenPeriod{From: &before, To: &now},
				expected: true,
			},
			{
				name:     "period with open start contains period with closed start (same time)",
				period1:  OpenPeriod{From: nil, To: &after},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: true,
			},
			{
				name:     "period does not contains touching period",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &now, To: &after},
				expected: false,
			},
			{
				name:     "identical period contains itself",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &before, To: &now},
				expected: true,
			},
			{
				name:     "period with closed end does not contain period with open end",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &now, To: nil},
				expected: false,
			},
			{
				name:     "period with closed start does not contain period with open start",
				period1:  OpenPeriod{From: &now, To: &after},
				period2:  OpenPeriod{From: nil, To: &now},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.period1.IsSupersetOf(tt.period2)
				if result != tt.expected {
					t.Errorf("IsSupersetOf() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("Difference", func(t *testing.T) {
		tests := []struct {
			name     string
			period1  OpenPeriod
			period2  OpenPeriod
			expected []OpenPeriod
		}{
			{
				name:     "no intersection",
				period1:  OpenPeriod{From: &before, To: &now},
				period2:  OpenPeriod{From: &after, To: nil},
				expected: []OpenPeriod{{From: &before, To: &now}},
			},
			{
				name:     "identical periods",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: []OpenPeriod{},
			},
			{
				name:     "period2 entirely contains period1",
				period1:  OpenPeriod{From: &now, To: &thirtyMinLater},
				period2:  OpenPeriod{From: &before, To: &after},
				expected: []OpenPeriod{},
			},
			{
				name:    "period1 entirely contains period2",
				period1: OpenPeriod{From: &before, To: &after},
				period2: OpenPeriod{From: &now, To: &thirtyMinLater},
				expected: []OpenPeriod{
					{From: &before, To: &now},
					{From: &thirtyMinLater, To: &after},
				},
			},
			{
				name:    "partial overlap - period2 starts before period1",
				period1: OpenPeriod{From: &now, To: &after},
				period2: OpenPeriod{From: &before, To: &thirtyMinLater},
				expected: []OpenPeriod{
					{From: &thirtyMinLater, To: &after},
				},
			},
			{
				name:    "partial overlap - period2 ends after period1",
				period1: OpenPeriod{From: &before, To: &thirtyMinLater},
				period2: OpenPeriod{From: &now, To: &after},
				expected: []OpenPeriod{
					{From: &before, To: &now},
				},
			},
			{
				name:    "period1 open from start",
				period1: OpenPeriod{From: nil, To: &after},
				period2: OpenPeriod{From: &now, To: &thirtyMinLater},
				expected: []OpenPeriod{
					{From: nil, To: &now},
					{From: &thirtyMinLater, To: &after},
				},
			},
			{
				name:    "period1 open to end",
				period1: OpenPeriod{From: &before, To: nil},
				period2: OpenPeriod{From: &now, To: &after},
				expected: []OpenPeriod{
					{From: &before, To: &now},
					{From: &after, To: nil},
				},
			},
			{
				name:    "period1 completely open (nil bounds)",
				period1: OpenPeriod{From: nil, To: nil},
				period2: OpenPeriod{From: &now, To: &after},
				expected: []OpenPeriod{
					{From: nil, To: &now},
					{From: &after, To: nil},
				},
			},
			{
				name:     "period2 completely open (nil bounds)",
				period1:  OpenPeriod{From: &before, To: &after},
				period2:  OpenPeriod{From: nil, To: nil},
				expected: []OpenPeriod{},
			},
			{
				name:    "both periods open on opposite ends, with overlap",
				period1: OpenPeriod{From: nil, To: &thirtyMinLater},
				period2: OpenPeriod{From: &now, To: nil},
				expected: []OpenPeriod{
					{From: nil, To: &now},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.period1.Difference(tt.period2)

				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d period(s), got %d", len(tt.expected), len(result))
					return
				}

				for i, expectedPeriod := range tt.expected {
					if i >= len(result) {
						t.Errorf("Missing expected period at index %d", i)
						continue
					}

					// Check From value
					if (expectedPeriod.From == nil) != (result[i].From == nil) {
						t.Errorf("Period %d: Incorrect From nil status, expected %v, got %v",
							i, expectedPeriod.From == nil, result[i].From == nil)
					} else if expectedPeriod.From != nil && result[i].From != nil &&
						!expectedPeriod.From.Equal(*result[i].From) {
						t.Errorf("Period %d: Incorrect From value, expected %v, got %v",
							i, *expectedPeriod.From, *result[i].From)
					}

					// Check To value
					if (expectedPeriod.To == nil) != (result[i].To == nil) {
						t.Errorf("Period %d: Incorrect To nil status, expected %v, got %v",
							i, expectedPeriod.To == nil, result[i].To == nil)
					} else if expectedPeriod.To != nil && result[i].To != nil &&
						!expectedPeriod.To.Equal(*result[i].To) {
						t.Errorf("Period %d: Incorrect To value, expected %v, got %v",
							i, *expectedPeriod.To, *result[i].To)
					}
				}
			})
		}
	})
}
