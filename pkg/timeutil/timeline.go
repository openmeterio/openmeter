package timeutil

import (
	"reflect"
	"slices"
	"time"
)

type SimpleTimeline = Timeline[time.Time]

func NewSimpleTimeline(times []time.Time) SimpleTimeline {
	wrapped := make([]Timed[time.Time], len(times))
	for i, t := range times {
		wrapped[i] = AsTimed(func(t time.Time) time.Time { return t })(t)
	}

	return NewTimeline(wrapped)
}

// AsTimed returns a function that converts a value of type T to a Timed value.
func AsTimed[T any](fn func(T) time.Time) func(T) Timed[T] {
	return func(t T) Timed[T] {
		return Timed[T]{
			val: t,
			fn:  fn,
		}
	}
}

type Timed[T any] struct {
	val T
	fn  func(T) time.Time
}

func (t Timed[T]) GetTime() time.Time {
	return t.fn(t.val)
}

func (t Timed[T]) GetValue() T {
	return t.val
}

func (t Timed[T]) Equal(other Timed[T]) bool {
	if !t.GetTime().Equal(other.GetTime()) {
		return false
	}

	return reflect.DeepEqual(t.GetValue(), other.GetValue())
}

type Timeline[T any] struct {
	times []Timed[T]
}

func NewTimeline[T any](times []Timed[T]) Timeline[T] {
	// sort copy of times ASC
	times = slices.Clone(times)

	slices.SortStableFunc(times, func(a, b Timed[T]) int {
		return a.GetTime().Compare(b.GetTime())
	})

	return Timeline[T]{times: times}
}

func (t Timeline[T]) After(at time.Time) Timeline[T] {
	times := make([]Timed[T], 0, len(t.times))
	for _, t := range t.times {
		if t.GetTime().After(at) {
			times = append(times, t)
		}
	}
	return NewTimeline(times)
}

func (t Timeline[T]) Before(at time.Time) Timeline[T] {
	times := make([]Timed[T], 0, len(t.times))
	for _, t := range t.times {
		if t.GetTime().Before(at) {
			times = append(times, t)
		}
	}
	return NewTimeline(times)
}

func (t Timeline[T]) GetTimes() []time.Time {
	times := make([]time.Time, len(t.times))
	for i, t := range t.times {
		times[i] = t.GetTime()
	}
	return times
}

func (t Timeline[T]) GetAt(idx int) Timed[T] {
	return t.times[idx]
}

func (t Timeline[T]) GetBoundingPeriod() ClosedPeriod {
	if len(t.times) == 0 {
		return ClosedPeriod{
			From: time.Time{},
			To:   time.Time{},
		}
	}

	return ClosedPeriod{
		From: t.times[0].GetTime(),
		To:   t.times[len(t.times)-1].GetTime(),
	}
}

// GetClosedPeriods returns the closed periods between the times in the timeline.
func (t Timeline[T]) GetClosedPeriods() []ClosedPeriod {
	if len(t.times) == 0 {
		return []ClosedPeriod{}
	}

	if len(t.times) == 1 {
		return []ClosedPeriod{
			{
				From: t.times[0].GetTime(),
				To:   t.times[0].GetTime(),
			},
		}
	}

	periods := make([]ClosedPeriod, 0, len(t.times)-1)
	for i := 0; i < len(t.times)-1; i++ {
		periods = append(periods, ClosedPeriod{From: t.times[i].GetTime(), To: t.times[i+1].GetTime()})
	}
	return periods
}

// GetOpenPeriods returns the open periods between the times in the timeline.
// For non-empty timelines, the first and last periods will be open at the start and end of the timeline respectively while the rest will effectively be closed.
func (t Timeline[T]) GetOpenPeriods() []OpenPeriod {
	if len(t.times) == 0 {
		return []OpenPeriod{}
	}

	periods := make([]OpenPeriod, 0, len(t.times)+1)

	// First period: open at start, closed at first time
	firstTime := t.times[0].GetTime()
	periods = append(periods, OpenPeriod{
		From: nil,
		To:   &firstTime,
	})

	// Middle periods: closed at both ends
	for i := 0; i < len(t.times)-1; i++ {
		from := t.times[i].GetTime()
		to := t.times[i+1].GetTime()
		periods = append(periods, OpenPeriod{
			From: &from,
			To:   &to,
		})
	}

	// Last period: closed at last time, open at end
	lastTime := t.times[len(t.times)-1].GetTime()
	periods = append(periods, OpenPeriod{
		From: &lastTime,
		To:   nil,
	})

	return periods
}
