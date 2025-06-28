package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseRFC9557(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTime time.Time
		wantErr  bool
	}{
		{
			name:     "RFC9557 with America/New_York summer time",
			input:    "2021-07-01T12:34:56-04:00[America/New_York]",
			wantTime: time.Date(2021, 7, 1, 12, 34, 56, 0, MustLoadLocation(t, "America/New_York")),
			wantErr:  false,
		},
		{
			name:     "RFC9557 with America/New_York winter time",
			input:    "2021-12-01T12:34:56-05:00[America/New_York]",
			wantTime: time.Date(2021, 12, 1, 12, 34, 56, 0, MustLoadLocation(t, "America/New_York")),
			wantErr:  false,
		},
		{
			name:     "RFC9557 with Europe/Berlin",
			input:    "2021-07-01T18:34:56+02:00[Europe/Berlin]",
			wantTime: time.Date(2021, 7, 1, 18, 34, 56, 0, MustLoadLocation(t, "Europe/Berlin")),
			wantErr:  false,
		},
		{
			name:     "RFC9557 with Asia/Tokyo",
			input:    "2021-07-01T21:34:56+09:00[Asia/Tokyo]",
			wantTime: time.Date(2021, 7, 1, 21, 34, 56, 0, MustLoadLocation(t, "Asia/Tokyo")),
			wantErr:  false,
		},
		{
			name:     "RFC9557 with UTC timezone",
			input:    "2021-07-01T16:34:56Z[UTC]",
			wantTime: time.Date(2021, 7, 1, 16, 34, 56, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "RFC9557 with fractional seconds",
			input:    "2021-07-01T12:34:56.123456789-04:00[America/New_York]",
			wantTime: time.Date(2021, 7, 1, 12, 34, 56, 123456789, MustLoadLocation(t, "America/New_York")),
			wantErr:  false,
		},

		{
			name:     "standard RFC3339 without timezone suffix",
			input:    "2021-07-01T16:34:56Z",
			wantTime: time.Date(2021, 7, 1, 16, 34, 56, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "standard RFC3339 with offset, no suffix",
			input:    "2021-07-01T12:34:56-04:00",
			wantTime: time.Date(2021, 7, 1, 12, 34, 56, 0, time.FixedZone("", -4*3600)),
			wantErr:  false,
		},
		{
			name:     "ISO8601 with fractional seconds",
			input:    "2156-10-05T18:36:46.924Z",
			wantTime: time.Date(2156, 10, 5, 18, 36, 46, 924000000, time.UTC),
			wantErr:  false,
		},
		{
			name:     "ISO8601 format without timezone suffix",
			input:    "2021-07-01T16:34:56Z",
			wantTime: time.Date(2021, 7, 1, 16, 34, 56, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "ISO8601 with milliseconds",
			input:    "2021-07-01T16:34:56.123Z",
			wantTime: time.Date(2021, 7, 1, 16, 34, 56, 123000000, time.UTC),
			wantErr:  false,
		},
		{
			name:     "ISO8601 with microseconds",
			input:    "2021-07-01T16:34:56.123456Z",
			wantTime: time.Date(2021, 7, 1, 16, 34, 56, 123456000, time.UTC),
			wantErr:  false,
		},

		// Error cases
		{
			name:     "invalid timezone in suffix",
			input:    "2021-07-01T12:34:56-04:00[Invalid/Timezone]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "malformed timestamp",
			input:    "invalid-timestamp[America/New_York]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "missing closing bracket",
			input:    "2021-07-01T12:34:56-04:00[America/New_York",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "missing opening bracket",
			input:    "2021-07-01T12:34:56-04:00America/New_York]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "empty timezone suffix",
			input:    "2021-07-01T12:34:56-04:00[]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "brackets not at end",
			input:    "2021-07-01T12:34:56-04:00[America/New_York]extra",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "nested brackets - malformed",
			input:    "2021-07-01T12:34:56-04:00[America/[nested]New_York]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "multiple bracket pairs",
			input:    "2021-07-01T12:34:56-04:00[first][America/New_York]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "only opening bracket",
			input:    "2021-07-01T12:34:56-04:00[America/New_York",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "only closing bracket",
			input:    "2021-07-01T12:34:56-04:00America/New_York]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "very short timestamp with brackets",
			input:    "2021[UTC]",
			wantTime: time.Time{},
			wantErr:  true,
		},
		{
			name:     "timezone with special characters",
			input:    "2021-07-01T12:34:56-04:00[America/New_York-Test]",
			wantTime: time.Time{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)

			if tt.wantErr {
				assert.Error(t, err, "expected error for input: %s", tt.input)
				return
			}

			assert.NoError(t, err, "expected no error for input: %s, got: %v", tt.input, err)
			assert.True(t, got.Equal(tt.wantTime), "parsed time mismatch: got %v, want %v", got, tt.wantTime)
		})
	}
}
