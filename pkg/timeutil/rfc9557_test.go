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

func TestUTCParsing(t *testing.T) {
	require := require.New(t)

	parsed, err := ParseRFC9557("2025-03-29T00:00:00Z")
	require.NoError(err)
	require.Equal(time.Date(2025, 3, 29, 0, 0, 0, 0, time.UTC), parsed.Time())
	require.Equal("2025-03-29T00:00:00Z", parsed.String())
}

func TestFixedZoneParsing(t *testing.T) {
	require := require.New(t)

	_, err := ParseRFC9557("2025-03-29T00:00:00+01:00")
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
