package clock

import (
	"sync/atomic"
	"time"
)

var (
	drift      int64 // store drift as nanoseconds
	frozen     int32 // use atomic int32 for boolean
	frozenTime atomic.Value
)

func Now() time.Time {
	if atomic.LoadInt32(&frozen) == 1 {
		return frozenTime.Load().(time.Time)
	}
	driftDuration := time.Duration(atomic.LoadInt64(&drift))
	t := time.Now().Add(-driftDuration)
	return t.Round(0) // Remove monotonic time reading
}

func SetTime(t time.Time) time.Time {
	driftDuration := time.Since(t).Nanoseconds()
	atomic.StoreInt64(&drift, driftDuration)
	return Now()
}

func ResetTime() {
	atomic.StoreInt64(&drift, 0)
}

func FreezeTime(t time.Time) {
	atomic.StoreInt32(&frozen, 1)
	frozenTime.Store(t)
}

func UnFreeze() {
	atomic.StoreInt32(&frozen, 0)
}
