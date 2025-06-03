package isodate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

func TestISOOperations(t *testing.T) {
	t.Run("Parse", func(t *testing.T) {
		isoDuration := "P1Y2M3DT4H5M6S"

		period, err := isodate.String(isoDuration).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		now := testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z")

		expected := testutils.GetRFC3339Time(t, "2021-03-04T04:05:06Z")
		actual, precise := period.AddTo(now)
		assert.True(t, precise)
		assert.Equal(t, expected, actual)
	})

	t.Run("ParseError", func(t *testing.T) {
		isoDuration := "P1Y2M3DT4H5M6SX"

		_, err := isodate.String(isoDuration).Parse()
		assert.NotNil(t, err)
	})

	t.Run("Works with 0 duration", func(t *testing.T) {
		isoDuration := "PT0S"

		period, err := isodate.String(isoDuration).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		now := testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z")

		expected := testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z")
		actual, precise := period.AddTo(now)
		assert.True(t, precise)
		assert.Equal(t, expected, actual)
	})

	t.Run("Adding periods", func(t *testing.T) {
		isoDuration1 := "PT5M"
		isoDuration2 := "PT1M1S"

		period1, err := isodate.String(isoDuration1).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		period2, err := isodate.String(isoDuration2).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedS := "PT6M1S"
		expected, err := isodate.String(expectedS).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		actual, err := period1.Add(period2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assert.Equal(t, expected, actual)
	})
}

func TestDivisibleBy(t *testing.T) {
	tests := []struct {
		name     string
		larger   string
		smaller  string
		expected bool
		hasError bool
	}{
		// Compatible periods - should be divisible
		{
			name:     "1 year divisible by 1 year",
			larger:   "P1Y",
			smaller:  "P1Y",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 1 month",
			larger:   "P1Y",
			smaller:  "P1M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 1 day",
			larger:   "P1Y",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 1 hour",
			larger:   "P1Y",
			smaller:  "PT1H",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 8 hours",
			larger:   "P1Y",
			smaller:  "PT8H",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 2 months",
			larger:   "P1Y",
			smaller:  "P2M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 3 months",
			larger:   "P1Y",
			smaller:  "P3M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 4 months",
			larger:   "P1Y",
			smaller:  "P4M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 year divisible by 6 months",
			larger:   "P1Y",
			smaller:  "P6M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 month divisible by 1 day",
			larger:   "P1M",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},
		{
			name:     "6 months divisible by 2 months",
			larger:   "P6M",
			smaller:  "P2M",
			expected: true,
			hasError: false,
		},
		{
			name:     "6 months divisible by 3 months",
			larger:   "P6M",
			smaller:  "P3M",
			expected: true,
			hasError: false,
		},
		{
			name:     "4 weeks divisible by 1 week",
			larger:   "P4W",
			smaller:  "P1W",
			expected: true,
			hasError: false,
		},
		{
			name:     "Same periods should be divisible",
			larger:   "P1M",
			smaller:  "P1M",
			expected: true,
			hasError: false,
		},

		// Incompatible periods - should not be divisible
		{
			name:     "1 month not divisible by 3 days",
			larger:   "P1M",
			smaller:  "P3D",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 month not divisible by 1 week",
			larger:   "P1M",
			smaller:  "P1W",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 year not divisible by 5 months",
			larger:   "P1Y",
			smaller:  "P5M",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 year not divisible by 7 months",
			larger:   "P1Y",
			smaller:  "P7M",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 month not divisible by 5 hour",
			larger:   "P1M",
			smaller:  "PT5H",
			expected: false,
			hasError: false,
		},

		// Edge cases - zero periods
		{
			name:     "zero period not divisible by anything",
			larger:   "PT0S",
			smaller:  "P1D",
			expected: false,
			hasError: false,
		},
		{
			name:     "anything not divisible by zero period",
			larger:   "P1D",
			smaller:  "PT0S",
			expected: false,
			hasError: false,
		},
		{
			name:     "zero period not divisible by zero period",
			larger:   "PT0S",
			smaller:  "PT0S",
			expected: false,
			hasError: false,
		},

		// Edge cases - smaller period larger than larger period
		{
			name:     "smaller period larger than larger period",
			larger:   "P1D",
			smaller:  "P1M",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 month smaller than 1 year",
			larger:   "P1M",
			smaller:  "P1Y",
			expected: false,
			hasError: false,
		},

		// Error cases - periods with minutes/seconds
		{
			name:     "both periods with minutes should error",
			larger:   "PT10M",
			smaller:  "PT5M",
			expected: false,
			hasError: true,
		},
		{
			name:     "both periods with seconds should error",
			larger:   "PT60S",
			smaller:  "PT30S",
			expected: false,
			hasError: true,
		},
		{
			name:     "larger period with mixed minutes and hours should error",
			larger:   "PT1H30M",
			smaller:  "PT1H",
			expected: false,
			hasError: true,
		},
		{
			name:     "smaller period with mixed seconds and hours should error",
			larger:   "PT2H",
			smaller:  "PT1H30S",
			expected: false,
			hasError: true,
		},

		// Additional day-based tests
		{
			name:     "30 days divisible by 5 days",
			larger:   "P30D",
			smaller:  "P5D",
			expected: true,
			hasError: false,
		},
		{
			name:     "30 days divisible by 6 days",
			larger:   "P30D",
			smaller:  "P6D",
			expected: true,
			hasError: false,
		},
		{
			name:     "30 days not divisible by 7 days",
			larger:   "P30D",
			smaller:  "P7D",
			expected: false,
			hasError: false,
		},
		{
			name:     "28 days divisible by 7 days",
			larger:   "P28D",
			smaller:  "P7D",
			expected: true,
			hasError: false,
		},
		{
			name:     "365 days divisible by 1 day",
			larger:   "P365D",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},

		// Week-based tests
		{
			name:     "8 weeks divisible by 2 weeks",
			larger:   "P8W",
			smaller:  "P2W",
			expected: true,
			hasError: false,
		},
		{
			name:     "8 weeks not divisible by 3 weeks",
			larger:   "P8W",
			smaller:  "P3W",
			expected: false,
			hasError: false,
		},
		{
			name:     "52 weeks divisible by 4 weeks",
			larger:   "P52W",
			smaller:  "P4W",
			expected: true,
			hasError: false,
		},

		// Complex multi-unit periods
		{
			name:     "2 years divisible by 6 months",
			larger:   "P2Y",
			smaller:  "P6M",
			expected: true,
			hasError: false,
		},
		{
			name:     "18 months divisible by 3 months",
			larger:   "P18M",
			smaller:  "P3M",
			expected: true,
			hasError: false,
		},
		{
			name:     "18 months divisible by 6 months",
			larger:   "P18M",
			smaller:  "P6M",
			expected: true,
			hasError: false,
		},
		{
			name:     "18 months not divisible by 5 months",
			larger:   "P18M",
			smaller:  "P5M",
			expected: false,
			hasError: false,
		},

		// Mixed units that should work
		{
			name:     "1 week divisible by 1 day",
			larger:   "P1W",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},
		{
			name:     "2 weeks divisible by 7 days",
			larger:   "P2W",
			smaller:  "P7D",
			expected: true,
			hasError: false,
		},
		{
			name:     "14 days divisible by 1 week",
			larger:   "P14D",
			smaller:  "P1W",
			expected: true,
			hasError: false,
		},

		// Hour-based tests (should work as they convert to whole days)
		{
			name:     "period with hours only should work",
			larger:   "PT24H",
			smaller:  "PT12H",
			expected: true,
			hasError: false,
		},
		{
			name:     "period with mixed hours should work",
			larger:   "P1DT12H",
			smaller:  "PT6H",
			expected: true,
			hasError: false,
		},
		{
			name:     "48 hours divisible by 24 hours",
			larger:   "PT48H",
			smaller:  "PT24H",
			expected: true,
			hasError: false,
		},
		{
			name:     "72 hours divisible by 24 hours",
			larger:   "PT72H",
			smaller:  "PT24H",
			expected: true,
			hasError: false,
		},
		{
			name:     "25 hours not divisible by 24 hours",
			larger:   "PT25H",
			smaller:  "PT24H",
			expected: false,
			hasError: false,
		},

		// Boundary tests
		{
			name:     "very large periods - 10 years divisible by 1 year",
			larger:   "P10Y",
			smaller:  "P1Y",
			expected: true,
			hasError: false,
		},
		{
			name:     "very large periods - 100 years divisible by 10 years",
			larger:   "P100Y",
			smaller:  "P10Y",
			expected: true,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			larger, err := isodate.String(tt.larger).Parse()
			assert.NoError(t, err)

			smaller, err := isodate.String(tt.smaller).Parse()
			assert.NoError(t, err)

			result, err := larger.DivisibleBy(smaller)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result, "Expected %s to be divisible by %s: %v", tt.larger, tt.smaller, tt.expected)
			}
		})
	}
}
