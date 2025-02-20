package timeutil

import (
	"slices"
	"time"
)

func NewTimeline(times []time.Time) Timeline {
	// sort copy of times ASC
	times = slices.Clone(times)

	slices.SortStableFunc(times, func(a, b time.Time) int {
		return int(a.Sub(b).Milliseconds())
	})

	return Timeline{
		times: times,
	}
}

type Timeline struct {
	times []time.Time
}

func (t Timeline) GetTimes() []time.Time {
	// Let's always return a non-nil array
	return t.times
}

func (t Timeline) GetBoundingPeriod() Period {
	if len(t.times) == 0 {
		return Period{
			From: time.Time{},
			To:   time.Time{},
		}
	}

	return Period{
		From: t.times[0],
		To:   t.times[len(t.times)-1],
	}
}

func (t Timeline) GetPeriods() []Period {
	if len(t.times) < 2 {
		return []Period{
			{
				From: t.times[0],
				To:   t.times[0],
			},
		}
	}

	periods := make([]Period, 0, len(t.times)-1)
	for i := 0; i < len(t.times)-1; i++ {
		periods = append(periods, Period{From: t.times[i], To: t.times[i+1]})
	}
	return periods
}
