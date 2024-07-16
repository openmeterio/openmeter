package clock

import "time"

var drift time.Duration

func Now() time.Time {
	return time.Now().Add(-drift)
}

func SetTime(t time.Time) time.Time {
	drift = time.Since(t)
	return Now()
}

func ResetTime() {
	drift = 0
}
