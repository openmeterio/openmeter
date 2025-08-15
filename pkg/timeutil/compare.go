package timeutil

import "time"

func Compare(a, b time.Time) int {
	return int(a.Sub(b))
}

func Later(t1 time.Time, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
