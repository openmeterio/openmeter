package recurrence

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/datex"
)

const MAX_SAFE_ITERATIONS = 10000

type Recurrence struct {
	Interval RecurrenceInterval `json:"period"`
	// Anchor can be an arbitrary anchor time for the recurrence.
	// It does not have to be the last or the next time.
	Anchor time.Time `json:"anchor"`
}

// Returns a period where p.Contains(t) is true
func (r Recurrence) GetPeriodAt(t time.Time) (Period, error) {
	var def Period

	next, err := r.NextAfter(t)
	if err != nil {
		return def, err
	}

	// As Period.Contains() is inclusive at the start and exclusive at the end, we need to get the next time
	if next.Equal(t) {
		start := next
		end, err := r.Next(start)
		if err != nil {
			return def, err
		}

		return Period{start, end}, nil
	}

	// Otherwise the next time will be the end
	prev, err := r.PrevBefore(t)
	if err != nil {
		return def, err
	}

	return Period{prev, next}, nil
}

// NextAfter returns the next time after t that the recurrence should occur.
// If at t the recurrence should occur, it will return t.
func (r Recurrence) NextAfter(t time.Time) (time.Time, error) {
	i := r.Anchor

	// If the anchor is in the future, we call .Prev() repeatedly. If the new value is Before t, we break
	if i.After(t) {
		ic := 0
		for i.After(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return time.Time{}, fmt.Errorf("recurrence.NextAfter: too many iterations")
			}
			ic += 1

			v, err := r.Prev(i)
			if err != nil {
				return time.Time{}, err
			}

			if v.Before(t) {
				break
			}

			i = v
		}
		// If the anchor is in the past, we call .Next() repeatedly. If the new value is !Before T, we break
	} else if i.Before(t) {
		ic := 0
		for i.Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return time.Time{}, fmt.Errorf("recurrence.NextAfter: too many iterations")
			}
			ic += 1

			v, err := r.Next(i)
			if err != nil {
				return time.Time{}, err
			}

			i = v
		}
	}

	return i, nil
}

// PrevBefore returns the previous time before t that the recurrence should occur.
func (r Recurrence) PrevBefore(t time.Time) (time.Time, error) {
	i := r.Anchor

	// If the anchor is in the future, we call .Prev() repeatedly. If the new value is Before t, we break
	if !i.Before(t) {
		ic := 0
		for !i.Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return time.Time{}, fmt.Errorf("recurrence.PrevBefore: too many iterations")
			}
			ic += 1

			v, err := r.Prev(i)
			if err != nil {
				return time.Time{}, err
			}
			i = v
		}
		// If the anchor is T or in the past relative, we call .Next() repeatedly. If the new value is !Before T, we break
	} else {
		ic := 0
		for i.Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return time.Time{}, fmt.Errorf("recurrence.PrevBefore: too many iterations")
			}
			ic += 1

			v, err := r.Next(i)
			if err != nil {
				return time.Time{}, err
			}

			if !v.Before(t) {
				break
			}

			i = v
		}
	}

	return i, nil
}

// Beware that calling Next then Prev on the result may not return the same time!

func (r Recurrence) Next(t time.Time) (time.Time, error) {
	n, ok := r.Interval.AddTo(t)
	if !ok {
		return time.Time{}, fmt.Errorf("next recurrence calculation wasn't exact, likely a fractional duration: %v", r.Interval)
	}
	return n, nil
}

func (r Recurrence) Prev(t time.Time) (time.Time, error) {
	n, ok := r.Interval.Negate().AddTo(t)
	if !ok {
		return time.Time{}, fmt.Errorf("previous recurrence calculation wasn't exact, likely a fractional duration: %v", r.Interval)
	}
	return n, nil
}

type RecurrenceInterval struct {
	datex.Period
}

var (
	RecurrencePeriodDaily RecurrenceInterval = RecurrenceInterval{datex.NewPeriod(0, 0, 0, 1, 0, 0, 0)}
	RecurrencePeriodWeek  RecurrenceInterval = RecurrenceInterval{datex.NewPeriod(0, 0, 1, 0, 0, 0, 0)}
	RecurrencePeriodMonth RecurrenceInterval = RecurrenceInterval{datex.NewPeriod(0, 1, 0, 0, 0, 0, 0)}
	RecurrencePeriodYear  RecurrenceInterval = RecurrenceInterval{datex.NewPeriod(1, 0, 0, 0, 0, 0, 0)}
)

func FromISODuration(p *datex.Period, anchor time.Time) (Recurrence, error) {
	if p == nil {
		return Recurrence{}, fmt.Errorf("period cannot be nil")
	}

	return Recurrence{
		Interval: RecurrenceInterval{*p},
		Anchor:   anchor,
	}, nil
}
