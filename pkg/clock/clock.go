package clock

import "time"

var drift time.Duration

func Now() time.Time {
	t := time.Now().Add(-drift)
	return t.Round(0) // Remove monotonic time reading
}

func SetTime(t time.Time) time.Time {
	drift = time.Since(t)
	return Now()
}

func ResetTime() {
	drift = 0
}
