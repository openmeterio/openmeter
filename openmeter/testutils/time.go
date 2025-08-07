package testutils

import (
	"testing"
	"time"
)

func GetRFC3339Time(t *testing.T, timeString string) time.Time {
	t.Helper()
	t1, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		t.Fatalf("Failed to parse time: %v", err)
	}
	return t1
}

func TimeEqualsApproximately(t *testing.T, expected time.Time, actual time.Time, tolerance time.Duration) {
	t.Helper()
	if expected.Before(actual.Add(tolerance)) && expected.After(actual.Add(-tolerance)) {
		return
	}
	t.Fatalf("Expected %v but got %v, outside tolerance of %v", expected, actual, tolerance)
}
