package timeutil

import "time"

func Compare(a, b time.Time) int {
	return int(a.Sub(b))
}
