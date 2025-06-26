package timeutil

import (
	"testing"
	"time"

	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/stretchr/testify/require"
)

func TestParsingFormating(t *testing.T) {
	require := require.New(t)

	loc, err := time.LoadLocation("Europe/Budapest")
	require.NoError(err)

	parsed, err := ParseRFC9557("2025-03-29T00:00:00.123456789[Europe/Budapest]")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 29, 0, 0, 0, 123456789, loc), parsed.Time())

	require.Equal("2025-03-29T00:00:00.123456789[Europe/Budapest]", parsed.String())

	parsed, err = ParseRFC9557("2025-03-29T00:00:00[Europe/Budapest]")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 29, 0, 0, 0, 0, loc), parsed.Time())

	require.Equal("2025-03-29T00:00:00[Europe/Budapest]", parsed.String())
}

func TestRFC3339Parsing(t *testing.T) {
	require := require.New(t)

	parsed, err := ParseRFC9557("2025-03-29T00:00:00Z")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 29, 0, 0, 0, 0, time.UTC), parsed.Time())
	require.Equal("2025-03-29T00:00:00Z", parsed.String())

	// Given an RFC3339 timestamp is added with a timezone offset, we should normalize it to UTC, location is UTC
	parsed, err = ParseRFC9557("2025-03-29T00:00:00.123456789+01:00")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 28, 23, 0, 0, 123456789, time.UTC), parsed.Time())
	require.Equal("2025-03-28T23:00:00.123456789Z", parsed.String())

	// Given an RFC3339 timestamp is added with a timezone offset, we should normalize it to UTC, location is UTC
	parsed, err = ParseRFC9557("2025-03-29T00:00:00+01:00")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 28, 23, 0, 0, 0, time.UTC), parsed.Time())
	require.Equal("2025-03-28T23:00:00Z", parsed.String())
}

func TestInvalidRFC9557Parsing(t *testing.T) {
	require := require.New(t)

	_, err := ParseRFC9557("2025-03-29T00:00:00.123456789+01:00[Europe/Budapest]")
	require.Error(err)

	_, err = ParseRFC9557("2025-03-29T00:")
	require.Error(err)

	_, err = ParseRFC9557("2025-14-33T33:33:33.123456789[Europe/Budapest]")
	require.Error(err)
}

func TestPeriodCalculations(t *testing.T) {
	require := require.New(t)

	loc, err := time.LoadLocation("Europe/Budapest")
	require.NoError(err)

	parsed, err := ParseRFC9557("2025-03-30T00:00:00[Europe/Budapest]")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 30, 0, 0, 0, 0, loc), parsed.Time())

	recurrence := Recurrence{
		Interval: RecurrenceInterval{
			Period: isodate.NewPeriod(0, 0, 0, 1, 0, 0, 0),
		},
		Anchor: parsed.Time(),
	}

	next, err := recurrence.Next(parsed.Time())
	require.NoError(err)

	require.Equal("2025-03-31T00:00:00[Europe/Budapest]", RFC9557Time{next}.String())
	require.Equal(23*time.Hour, next.Sub(parsed.Time()))
}
