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
	day, err := datex.ISOString("P1D").Parse()
	if err != nil {
		return Recurrence{}, fmt.Errorf("invalid ISO period string used %w", err)
	}
	week, err := datex.ISOString("P1W").Parse()
	if err != nil {
		return Recurrence{}, fmt.Errorf("invalid ISO period string used %w", err)
	}
	month, err := datex.ISOString("P1M").Parse()
	if err != nil {
		return Recurrence{}, fmt.Errorf("invalid ISO period string used %w", err)
	}
	year, err := datex.ISOString("P1Y").Parse()
	if err != nil {
		return Recurrence{}, fmt.Errorf("invalid ISO period string used %w", err)
	}

	if v, err := p.Subtract(day); err == nil && v.IsZero() {
		return Recurrence{
			Anchor:   anchor,
			Interval: RecurrencePeriodDaily,
		}, nil
	} else if v, err := p.Subtract(week); err == nil && v.IsZero() {
		return Recurrence{
			Anchor:   anchor,
			Interval: RecurrencePeriodWeek,
		}, nil
	} else if v, err := p.Subtract(month); err == nil && v.IsZero() {
		return Recurrence{
			Anchor:   anchor,
			Interval: RecurrencePeriodMonth,
		}, nil
	} else if v, err := p.Subtract(year); err == nil && v.IsZero() {
		return Recurrence{
			Anchor:   anchor,
			Interval: RecurrencePeriodYear,
		}, nil
	}

	return Recurrence{}, fmt.Errorf("invalid period, allowed values are 1D, 1W, 1M, 1Y")
}
