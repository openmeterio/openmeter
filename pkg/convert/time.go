package convert

import "time"

func TimePtrIn(t *time.Time, loc *time.Location) *time.Time {
	if t == nil {
		return nil
	}
	return ToPointer(t.In(loc))
}
