package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDateTime_Parse_Format(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		layout   string
		wantTime time.Time
	}{
		{
			name:     "RFC9557 with America/New_York summer time",
			input:    "2021-07-01T12:34:56-04:00[America/New_York]",
			layout:   RFC9557Layout,
			wantTime: time.Date(2021, 7, 1, 12, 34, 56, 0, MustLoadLocation(t, "America/New_York")),
		},
		{
			name:     "RFC9557 with America/New_York winter time",
			input:    "2021-12-01T12:34:56-05:00[America/New_York]",
			layout:   RFC9557Layout,
			wantTime: time.Date(2021, 12, 1, 12, 34, 56, 0, MustLoadLocation(t, "America/New_York")),
		},
		{
			name:     "RFC9557 with Europe/Berlin",
			input:    "2021-07-01T18:34:56+02:00[Europe/Berlin]",
			layout:   RFC9557Layout,
			wantTime: time.Date(2021, 7, 1, 18, 34, 56, 0, MustLoadLocation(t, "Europe/Berlin")),
		},
		{
			name:     "RFC9557 with Asia/Tokyo",
			input:    "2021-07-01T21:34:56+09:00[Asia/Tokyo]",
			layout:   RFC9557Layout,
			wantTime: time.Date(2021, 7, 1, 21, 34, 56, 0, MustLoadLocation(t, "Asia/Tokyo")),
		},
		{
			name:     "RFC9557 with fractional seconds",
			input:    "2021-07-01T12:34:56.123456789-04:00[America/New_York]",
			layout:   RFC9557NanoLayout,
			wantTime: time.Date(2021, 7, 1, 12, 34, 56, 123456789, MustLoadLocation(t, "America/New_York")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input string
			parsed, err := Parse(tt.input)
			assert.NoError(t, err, "Parse should not return an error")
			assert.True(t, parsed.Equal(tt.wantTime), "Parse should parse the datetime correctly")

			// Format the parsed time back to a string
			formatted := parsed.Format(tt.layout)

			// Verify round-trip: the input time should equal the formatted time
			assert.Equal(t, tt.input, formatted, "round-trip formatting should preserve the original input")
		})
	}
}

func TestDateTime_Add(t *testing.T) {
	tests := []struct {
		name     string
		start    string
		duration Duration
		expected string
		outputTz *time.Location
	}{
		{
			name:     "add 1 month correctly",
			start:    "2024-12-31T00:00:00Z",
			duration: MustParseDuration(t, "P1M"),
			expected: "2025-01-31T00:00:00Z",
		},
		{
			name:     "add 1 year correctly with leap year handling",
			start:    "2024-02-29T00:00:00Z",
			duration: MustParseDuration(t, "P1Y"),
			expected: "2025-02-28T00:00:00Z",
		},
		{
			name:     "handle leap year transition over multiple years",
			start:    "2024-02-29T00:00:00Z",
			duration: MustParseDuration(t, "P2Y"),
			expected: "2026-02-28T00:00:00Z",
		},
		{
			name:     "add 6 months with month-end handling",
			start:    "2024-12-31T00:00:00Z",
			duration: MustParseDuration(t, "P6M"),
			expected: "2025-06-30T00:00:00Z",
		},
		{
			name:     "add 1 week correctly",
			start:    "2025-02-01T12:00:00Z",
			duration: MustParseDuration(t, "P1W"),
			expected: "2025-02-08T12:00:00Z",
		},
		{
			name:     "add 3 hours correctly",
			start:    "2025-02-01T12:00:00Z",
			duration: MustParseDuration(t, "PT3H"),
			expected: "2025-02-01T15:00:00Z",
		},
		{
			name:     "add 30 minutes correctly",
			start:    "2025-02-01T12:00:00Z",
			duration: MustParseDuration(t, "PT30M"),
			expected: "2025-02-01T12:30:00Z",
		},
		{
			name:     "add 45 seconds correctly",
			start:    "2025-02-01T12:00:00Z",
			duration: MustParseDuration(t, "PT45S"),
			expected: "2025-02-01T12:00:45Z",
		},
		{
			name:     "handle daylight savings transition",
			start:    "2025-03-25T01:00:00+01:00[Europe/Budapest]",
			duration: MustParseDuration(t, "PT2H"),
			expected: "2025-03-25T04:00:00+02:00[Europe/Budapest]",
			outputTz: MustLoadLocation(t, "Europe/Budapest"),
		},
		{
			name:     "handle complex duration with multiple components",
			start:    "2024-01-15T10:30:00Z",
			duration: MustParseDuration(t, "P1Y2M10DT5H30M45S"),
			expected: "2025-03-25T16:00:45Z",
		},
		{
			name:     "handle negative duration",
			start:    "2025-06-15T12:00:00Z",
			duration: MustParseDuration(t, "P-3M"),
			expected: "2025-03-15T12:00:00Z",
		},
		{
			name:     "handle end of month edge case",
			start:    "2025-01-31T00:00:00Z",
			duration: MustParseDuration(t, "P1M"),
			expected: "2025-02-28T00:00:00Z",
		},
		{
			name:     "handle leap second period (December 2016)",
			start:    "2016-11-30T00:00:00Z",
			duration: MustParseDuration(t, "P1M"),
			expected: "2016-12-30T00:00:00Z",
		},
		{
			name:     "handle zero duration",
			start:    "2025-02-01T12:00:00Z",
			duration: MustParseDuration(t, "PT0S"),
			expected: "2025-02-01T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, err := Parse(tt.start)
			assert.NoError(t, err, "failed to parse start time")

			result := start.Add(tt.duration)

			expected, err := Parse(tt.expected)
			assert.NoError(t, err, "failed to parse expected time")

			// Compare in the appropriate timezone
			tz := time.UTC
			if tt.outputTz != nil {
				tz = tt.outputTz
			}

			assert.Equal(t,
				expected.In(tz).Format(RFC9557Layout),
				result.In(tz).Format(RFC9557Layout),
				"duration addition result mismatch")

			// Also test that the time values are equal
			assert.True(t, result.Equal(expected.Time),
				"times should be equal: got %v, want %v",
				result.Format(RFC9557Layout),
				expected.Format(RFC9557Layout))
		})
	}
}

// NOTE: The fuzz test below is commented out because the timezone database might have
// differences between Node.js and Go environments. Uncomment and provide duration_test.json
// file for comprehensive testing.

/*
func TestDuration_Add_Fuzz(t *testing.T) {
	// Read test cases from JSON file
	data, err := os.ReadFile("duration_test.json")
	assert.NoError(t, err)

	var testCases []testCase
	err = json.Unmarshal(data, &testCases)
	assert.NoError(t, err)

	failingCount := 0
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("start_%d", i), func(t *testing.T) {
			ok := assert.Equal(t, tc.Temporal.Start.Format(RFC9557MilliLayout), tc.Internationalized.Start.Format(RFC9557MilliLayout))
			if !ok {
				failingCount++
			}
		})

		t.Run(fmt.Sprintf("end_%d", i), func(t *testing.T) {
			ok := assert.Equal(t, tc.Temporal.End.Format(RFC9557MilliLayout), tc.Internationalized.End.Format(RFC9557MilliLayout))
			if !ok {
				failingCount++
			}
		})

		t.Run(fmt.Sprintf("temporal_case_%d", i), func(t *testing.T) {
			result := tc.Temporal.Start.Add(tc.Duration)

			ok := assert.Equal(t, tc.Temporal.End, result)
			if !ok {
				failingCount++
			}
		})

		t.Run(fmt.Sprintf("internationalized_case_%d", i), func(t *testing.T) {
			result := tc.Internationalized.Start.Add(tc.Duration)

			ok := assert.Equal(t, tc.Internationalized.End.Format(RFC9557MilliLayout), result.Format(RFC9557MilliLayout))
			if !ok {
				failingCount++
			}
		})
	}

	assert.Equal(t, 0, failingCount, "failed %d test cases", failingCount)
}
*/
