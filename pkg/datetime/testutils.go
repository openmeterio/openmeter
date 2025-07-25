package datetime

import (
	"testing"
	"time"

	"github.com/rickb777/period"
	"github.com/stretchr/testify/assert"
)

// MustLoadLocation is a helper function that panics if the location cannot be loaded.
func MustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("failed to load timezone location %q: %v", name, err)
	}
	return loc
}

// MustParseDateTime is a helper function to parse time with error checking
func MustParseDateTime(t *testing.T, timeStr string) DateTime {
	t.Helper()
	dt, err := Parse(timeStr)
	assert.NoError(t, err, "failed to parse time string %q", timeStr)
	return dt
}

// MustParseTimeInLocation is a helper function to parse time in specific location
func MustParseTimeInLocation(t *testing.T, timeStr string, loc *time.Location) DateTime {
	t.Helper()
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	assert.NoError(t, err, "failed to parse time string %q", timeStr)
	return DateTime{Time: parsedTime.In(loc)}
}

// MustParseDuration is a helper function to parse duration with error checking
func MustParseDuration(t *testing.T, s string) ISODuration {
	res, err := period.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse period: %v", err)
	}

	return ISODuration{res}
}
