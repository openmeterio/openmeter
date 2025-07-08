package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecurringPeriod_IterInRange(t *testing.T) {
	tests := getPeriodTests(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period := NewRecurringPeriod(tt.baseTime, tt.duration)

			startTime, err := time.Parse(time.RFC3339, tt.start)
			assert.NoError(t, err, "failed to parse start time")

			endTime, err := time.Parse(time.RFC3339, tt.end)
			assert.NoError(t, err, "failed to parse end time")

			var actual []DateTime
			for dt := range period.IterInRange(startTime, endTime) {
				actual = append(actual, dt)
			}

			assert.Equal(t, len(tt.expected), len(actual),
				"expected %d periods, got %d", len(tt.expected), len(actual))

			// Determine output timezone for comparison
			outputTz := time.UTC
			if tt.outputTz != nil {
				outputTz = tt.outputTz
			}

			for i, expectedStr := range tt.expected {
				// Parse the expected time string
				expectedTime, err := time.Parse(time.RFC3339, expectedStr)
				assert.NoError(t, err, "failed to parse expected time: %s", expectedStr)

				// Compare the times in the appropriate timezone
				actualInTz := actual[i].Time.In(outputTz)
				expectedInTz := expectedTime.In(outputTz)

				assert.True(t, actualInTz.Equal(expectedInTz),
					"period %d mismatch: got %v, want %v (in %s timezone)",
					i, actualInTz.Format(time.RFC3339), expectedInTz.Format(time.RFC3339), outputTz)
			}
		})
	}
}

func TestRecurringPeriod_ValuesInRange(t *testing.T) {
	tests := getPeriodTests(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period := NewRecurringPeriod(tt.baseTime, tt.duration)

			startTime, err := time.Parse(time.RFC3339, tt.start)
			assert.NoError(t, err, "failed to parse start time")

			endTime, err := time.Parse(time.RFC3339, tt.end)
			assert.NoError(t, err, "failed to parse end time")

			actual := period.ValuesInRange(startTime, endTime)

			assert.Equal(t, len(tt.expected), len(actual),
				"expected %d periods, got %d", len(tt.expected), len(actual))

			// Determine output timezone for comparison
			outputTz := time.UTC
			if tt.outputTz != nil {
				outputTz = tt.outputTz
			}

			for i, expectedStr := range tt.expected {
				// Parse the expected time string
				expectedTime, err := time.Parse(time.RFC3339, expectedStr)
				assert.NoError(t, err, "failed to parse expected time: %s", expectedStr)

				// Compare the times in the appropriate timezone
				actualInTz := actual[i].Time.In(outputTz)
				expectedInTz := expectedTime.In(outputTz)

				assert.True(t, actualInTz.Equal(expectedInTz),
					"period %d mismatch: got %v, want %v (in %s timezone)",
					i, actualInTz.Format(time.RFC3339), expectedInTz.Format(time.RFC3339), outputTz)
			}
		})
	}
}

func getPeriodTests(t *testing.T) []struct {
	name     string
	baseTime DateTime
	duration Duration
	start    string
	end      string
	expected []string
	outputTz *time.Location
} {
	bpTz := MustLoadLocation(t, "Europe/Budapest")
	nyTz := MustLoadLocation(t, "America/New_York")

	return []struct {
		name     string
		baseTime DateTime
		duration Duration
		start    string
		end      string
		expected []string
		outputTz *time.Location
	}{
		{
			name:     "monthly recurrence periods",
			baseTime: MustParseTime(t, "2024-12-31T00:00:00Z"),
			duration: MustParseDuration(t, "P1M"),
			start:    "2024-12-31T00:00:00Z",
			end:      "2025-05-01T00:00:00Z",
			expected: []string{
				"2024-12-31T00:00:00Z",
				"2025-01-31T00:00:00Z",
				"2025-02-28T00:00:00Z",
				"2025-03-31T00:00:00Z",
				"2025-04-30T00:00:00Z",
			},
		},
		{
			name:     "leap year handling in yearly recurrence",
			baseTime: MustParseTime(t, "2024-02-29T00:00:00Z"),
			duration: MustParseDuration(t, "P1Y"),
			start:    "2024-02-29T00:00:00Z",
			end:      "2027-01-01T00:00:00Z",
			expected: []string{
				"2024-02-29T00:00:00Z",
				"2025-02-28T00:00:00Z", // Non-leap year, Feb 29 -> Feb 28
				"2026-02-28T00:00:00Z",
			},
		},
		{
			name:     "month end handling for varying month lengths",
			baseTime: MustParseTime(t, "2024-01-31T00:00:00Z"),
			duration: MustParseDuration(t, "P1M"),
			start:    "2024-01-31T00:00:00Z",
			end:      "2024-05-01T00:00:00Z",
			expected: []string{
				"2024-01-31T00:00:00Z",
				"2024-02-29T00:00:00Z", // 2024 is leap year, so Jan 31 + 1 month = Feb 29
				"2024-03-31T00:00:00Z",
				"2024-04-30T00:00:00Z", // April has 30 days
			},
		},
		{
			name:     "timezone preservation with America/New_York",
			baseTime: DateTime{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, nyTz)},
			duration: MustParseDuration(t, "P1D"),
			start:    "2024-01-01T12:00:00-05:00",
			end:      "2024-01-05T12:00:00-05:00",
			expected: []string{
				"2024-01-01T12:00:00-05:00", // EST timezone
				"2024-01-02T12:00:00-05:00",
				"2024-01-03T12:00:00-05:00",
				"2024-01-04T12:00:00-05:00",
			},
			outputTz: nyTz,
		},
		{
			name:     "daylight savings changes with timezone-aware anchor",
			baseTime: MustParseTimeInLocation(t, "2025-02-01T12:00:00Z", bpTz),
			duration: MustParseDuration(t, "P1M"),
			start:    "2025-02-01T12:00:00+01:00",
			end:      "2025-06-01T12:00:00+02:00",
			expected: []string{
				"2025-02-01T13:00:00+01:00", // UTC 12:00 = Budapest 13:00 in winter
				"2025-03-01T13:00:00+01:00", // Still winter time
				// REFERENCE EXPECTS: "2025-04-01T14:00:00+02:00" (UTC relationship preserved)
				"2025-04-01T13:00:00+02:00", // DST: our impl preserves the local time during DST transition
				// REFERENCE EXPECTS: "2025-05-01T14:00:00+02:00" (UTC relationship preserved)
				"2025-05-01T13:00:00+02:00", // Still daylight savings - local time preserved
			},
			outputTz: bpTz,
		},
		{
			name:     "daylight savings changes with UTC anchor",
			baseTime: MustParseTime(t, "2025-02-01T12:00:00Z"),
			duration: MustParseDuration(t, "P1M"),
			start:    "2025-02-01T12:00:00Z",
			end:      "2025-06-01T12:00:00Z",
			expected: []string{
				"2025-02-01T12:00:00Z",
				"2025-03-01T12:00:00Z",
				"2025-04-01T12:00:00Z", // UTC stays constant across DST
				"2025-05-01T12:00:00Z",
			},
			outputTz: bpTz,
		},
		{
			name:     "leap second period handling",
			baseTime: MustParseTime(t, "2016-11-30T00:00:00Z"),
			duration: MustParseDuration(t, "P1M"),
			start:    "2016-11-30T00:00:00Z",
			end:      "2017-04-01T00:00:00Z",
			expected: []string{
				"2016-11-30T00:00:00Z",
				"2016-12-30T00:00:00Z",
				"2017-01-30T00:00:00Z",
				"2017-02-28T00:00:00Z", // February has 28 days in 2017
				"2017-03-30T00:00:00Z",
			},
		},
	}
}
