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
			name:     "1 month divisible by 1 day",
			larger:   "P1M",
			smaller:  "P1D",
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
			name:     "1 year divisible by 12 months",
			larger:   "P1Y",
			smaller:  "P12M",
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
			name:     "1 year and 1 month is divisible by 1 day",
			larger:   "P1Y1M",
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
			name:     "1 month divisible by 1 day",
			larger:   "P1M",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 day not divisible by 1 hour (different units)",
			larger:   "P1D",
			smaller:  "PT1H",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 day not divisible by 2 hours (different units)",
			larger:   "P1D",
			smaller:  "PT2H",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 day not divisible by 4 hours (different units)",
			larger:   "P1D",
			smaller:  "PT4H",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 day not divisible by 6 hours (different units)",
			larger:   "P1D",
			smaller:  "PT6H",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 day not divisible by 8 hours (different units)",
			larger:   "P1D",
			smaller:  "PT8H",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 day not divisible by 12 hours (different units)",
			larger:   "P1D",
			smaller:  "PT12H",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 hour divisible by 1 minute",
			larger:   "PT1H",
			smaller:  "PT1M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 hour divisible by 15 minutes",
			larger:   "PT1H",
			smaller:  "PT15M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 hour divisible by 30 minutes",
			larger:   "PT1H",
			smaller:  "PT30M",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 minute divisible by 1 second",
			larger:   "PT1M",
			smaller:  "PT1S",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 minute divisible by 15 seconds",
			larger:   "PT1M",
			smaller:  "PT15S",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 minute divisible by 30 seconds",
			larger:   "PT1M",
			smaller:  "PT30S",
			expected: true,
			hasError: false,
		},
		{
			name:     "24 hours divisible by 1 hour",
			larger:   "PT24H",
			smaller:  "PT1H",
			expected: true,
			hasError: false,
		},
		{
			name:     "24 hours divisible by 2 hours",
			larger:   "PT24H",
			smaller:  "PT2H",
			expected: true,
			hasError: false,
		},
		{
			name:     "24 hours divisible by 4 hours",
			larger:   "PT24H",
			smaller:  "PT4H",
			expected: true,
			hasError: false,
		},
		{
			name:     "24 hours divisible by 6 hours",
			larger:   "PT24H",
			smaller:  "PT6H",
			expected: true,
			hasError: false,
		},
		{
			name:     "24 hours divisible by 8 hours",
			larger:   "PT24H",
			smaller:  "PT8H",
			expected: true,
			hasError: false,
		},
		{
			name:     "24 hours divisible by 12 hours",
			larger:   "PT24H",
			smaller:  "PT12H",
			expected: true,
			hasError: false,
		},
		{
			name:     "7 days divisible by 1 day",
			larger:   "P7D",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 week divisible by 1 day",
			larger:   "P1W",
			smaller:  "P1D",
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
