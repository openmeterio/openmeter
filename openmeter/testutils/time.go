package testutils

import (
	"testing"
	"time"

	"github.com/openmeterio/openmeter/pkg/datex"
)

func GetRFC3339Time(t *testing.T, timeString string) time.Time {
	t.Helper()
	t1, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		t.Fatalf("Failed to parse time: %v", err)
	}
	return t1
}

func GetISODuration(t *testing.T, durationString string) datex.Period {
	t.Helper()
	d, err := datex.ISOString(durationString).Parse()
	if err != nil {
		t.Fatalf("Failed to parse duration: %v", err)
	}
	return d
}
