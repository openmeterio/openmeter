package datetime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDuration_DivisibleBy(t *testing.T) {
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
			name:     "2 years divisible by 1 year",
			larger:   "P2Y",
			smaller:  "P1Y",
			expected: true,
			hasError: false,
		},
		{
			name:     "2 years divisible by 2 months",
			larger:   "P2Y",
			smaller:  "P2M",
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
			name:     "1 year not divisible by 5 days",
			larger:   "P1Y",
			smaller:  "P5D",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 year not divisible by 365 days",
			larger:   "P1Y",
			smaller:  "P365D",
			expected: false,
			hasError: false,
		},
		{
			name:     "1 year not divisible by 8 hours",
			larger:   "P1Y",
			smaller:  "PT8H",
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
			name:     "zero period is divisible by anything",
			larger:   "PT0S",
			smaller:  "P1D",
			expected: true,
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
			name:     "zero period divisible by zero period",
			larger:   "PT0S",
			smaller:  "PT0S",
			expected: true,
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
			name:     "smaller period larger than larger period 2",
			larger:   "P1W",
			smaller:  "P1M",
			expected: false,
			hasError: false,
		},
		{
			name:     "smaller period larger than larger period 3",
			larger:   "P4W",
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

		// Hour-based tests
		{
			name:     "period with hours only should work",
			larger:   "PT24H",
			smaller:  "PT12H",
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

		// Periods with minutes/seconds (should work with our implementation)
		{
			name:     "10 minutes divisible by 5 minutes",
			larger:   "PT10M",
			smaller:  "PT5M",
			expected: true,
			hasError: false,
		},
		{
			name:     "60 seconds divisible by 30 seconds",
			larger:   "PT60S",
			smaller:  "PT30S",
			expected: true,
			hasError: false,
		},
		{
			name:     "1 hour 30 minutes divisible by 30 minutes",
			larger:   "PT1H30M",
			smaller:  "PT30M",
			expected: true,
			hasError: false,
		},
		{
			name:     "2 hours divisible by 1 hour 30 seconds (not exact)",
			larger:   "PT2H",
			smaller:  "PT1H30S",
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

		// Additional edge cases
		{
			name:     "2 hours divisible by 1 hour",
			larger:   "PT2H",
			smaller:  "PT1H",
			expected: true,
			hasError: false,
		},
		{
			name:     "3 days divisible by 1 day",
			larger:   "P3D",
			smaller:  "P1D",
			expected: true,
			hasError: false,
		},
		{
			name:     "5 hours not divisible by 3 hours",
			larger:   "PT5H",
			smaller:  "PT3H",
			expected: false,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			larger, err := DurationString(tt.larger).Parse()
			assert.NoError(t, err)

			smaller, err := DurationString(tt.smaller).Parse()
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
