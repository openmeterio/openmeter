package timeutil

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/datetime"
)

const MAX_SAFE_ITERATIONS = 10000

type Recurrence struct {
	Interval RecurrenceInterval `json:"interval"`
	// Anchor can be an arbitrary anchor time for the recurrence.
	// It does not have to be the last or the next time.
	Anchor time.Time `json:"anchor"`
}

func (r Recurrence) Validate() error {
	var errs []error

	if r.Interval.ISODuration.Sign() != 1 {
		errs = append(errs, fmt.Errorf("recurrence interval must be positive"))
	}

	if r.Anchor.IsZero() {
		errs = append(errs, fmt.Errorf("recurrence anchor must be set"))
	}

	return errors.Join(errs...)
}

// Returns a period where p.Contains(t) is true
func (r Recurrence) GetPeriodAt(t time.Time) (ClosedPeriod, error) {
	var def ClosedPeriod

	next, err := r.IterateFromNextAfter(t, Exclusive)
	if err != nil {
		return def, err
	}

	// As Period.Contains() is inclusive at the start and exclusive at the end, we need to get the next time
	if next.At.Equal(t) {
		start := next
		end, err := start.Next()
		if err != nil {
			return def, err
		}

		return ClosedPeriod{start.At, end.At}, nil
	}

	// Otherwise the next time will be the end
	prev, err := r.IterateFromPrevBefore(t, Inclusive)
	if err != nil {
		return def, err
	}

	return ClosedPeriod{prev.At, next.At}, nil
}

// IterateFromNextAfter returns the next time after (or equal to) t that the recurrence should occur.
//
// If boundaryBehavior is Inclusive, if t matches a recurrence value, we return t as is.
// If boundaryBehavior is Exclusive, if t matches a recurrence value, we return the next recurrence value.
func (r Recurrence) IterateFromNextAfter(t time.Time, boundaryBehavior Boundary) (RecurrenceIterator, error) {
	if err := boundaryBehavior.Validate(); err != nil {
		return RecurrenceIterator{}, err
	}

	if t.IsZero() {
		return RecurrenceIterator{}, fmt.Errorf("t cannot be zero")
	}

	inclusiveNextAfter, err := r.iterateFromNextAfterInclusive(t)
	if err != nil {
		return RecurrenceIterator{}, err
	}

	if boundaryBehavior == Exclusive && inclusiveNextAfter.At.Equal(t) {
		return inclusiveNextAfter.Next()
	}

	return inclusiveNextAfter, nil
}

// NextAfter is a convenience function that returns the next time after t that the recurrence should occur.
// It is equivalent to calling IterateFromNextAfter and returning the At field of the iterator.
//
// If boundaryBehavior is Inclusive, if t matches a recurrence value, we return t as is.
// If boundaryBehavior is Exclusive, if t matches a recurrence value, we return the next recurrence value.
func (r Recurrence) NextAfter(t time.Time, boundaryBehavior Boundary) (time.Time, error) {
	iter, err := r.IterateFromNextAfter(t, boundaryBehavior)
	if err != nil {
		return time.Time{}, err
	}

	return iter.At, nil
}

func (r Recurrence) iterateFromNextAfterInclusive(t time.Time) (RecurrenceIterator, error) {
	// If the anchor is in the future, we call .Prev() repeatedly. If the new value is Before t, we break
	if r.Anchor.After(t) {
		res := r.Anchor
		ic := 0

		// Calculate the inclusive nextAfter value
		for res.After(t) {
			// TODO: Right now we cannot iterate backwards past 1733, please fix this.
			if ic <= -MAX_SAFE_ITERATIONS {
				return RecurrenceIterator{}, fmt.Errorf("recurrence.NextAfter: too many iterations")
			}
			ic -= 1

			v, err := r.addIntervalNTimes(r.Anchor, ic)
			if err != nil {
				return RecurrenceIterator{}, err
			}

			if v.Before(t) {
				break
			}

			res = v
		}

		return RecurrenceIterator{
			r:         r,
			iteration: ic,
			At:        res,
		}, nil
	}

	// If the anchor is in the past, we call .Next() repeatedly. If the new value is !Before T, we break
	if r.Anchor.Before(t) {
		res := r.Anchor
		ic := 0

		// Calculate the inclusive nextAfter value
		for res.Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return RecurrenceIterator{}, fmt.Errorf("recurrence.NextAfter: too many iterations")
			}
			ic += 1

			v, err := r.addIntervalNTimes(r.Anchor, ic)
			if err != nil {
				return RecurrenceIterator{}, err
			}

			res = v
		}

		return RecurrenceIterator{
			r:         r,
			iteration: ic,
			At:        res,
		}, nil
	}

	return r.Iterator(), nil
}

// IterateFromPrevBefore returns the previous time before (or equal to) t that the recurrence should occur.
//
// If boundaryBehavior is Inclusive, if t matches a recurrence value, we return t as is.
// If boundaryBehavior is Exclusive, if t matches a recurrence value, we return the previous recurrence value.
func (r Recurrence) IterateFromPrevBefore(t time.Time, boundaryBehavior Boundary) (RecurrenceIterator, error) {
	if err := boundaryBehavior.Validate(); err != nil {
		return RecurrenceIterator{}, err
	}

	if t.IsZero() {
		return RecurrenceIterator{}, fmt.Errorf("t cannot be zero")
	}

	inclusivePrevBefore, err := r.iterateFromPrevBeforeInclusive(t)
	if err != nil {
		return RecurrenceIterator{}, err
	}

	if boundaryBehavior == Exclusive && inclusivePrevBefore.At.Equal(t) {
		return inclusivePrevBefore.Prev()
	}

	return inclusivePrevBefore, nil
}

// PrevBefore is a convenience function that returns the previous time before t that the recurrence should occur.
// It is equivalent to calling IterateFromPrevBefore and returning the At field of the iterator.
//
// If boundaryBehavior is Inclusive, if t matches a recurrence value, we return t as is.
// If boundaryBehavior is Exclusive, if t matches a recurrence value, we return the previous recurrence value.
func (r Recurrence) PrevBefore(t time.Time, boundaryBehavior Boundary) (time.Time, error) {
	iter, err := r.IterateFromPrevBefore(t, boundaryBehavior)
	if err != nil {
		return time.Time{}, err
	}

	return iter.At, nil
}

// IterateFromPrevBefore returns the previous time before t that the recurrence should occur.
//
// If t is an iteration boundary, it will return the previous iteration (as the start of a period is inclusive).
func (r Recurrence) iterateFromPrevBeforeInclusive(t time.Time) (RecurrenceIterator, error) {
	// If the anchor is in the future, we call .Prev() repeatedly. If the new value is Before t, we break
	if r.Anchor.After(t) {
		res := r.Anchor
		ic := 0

		for res.After(t) {
			if ic <= -MAX_SAFE_ITERATIONS {
				return RecurrenceIterator{}, fmt.Errorf("recurrence.PrevBefore: too many iterations")
			}
			ic -= 1

			v, err := r.addIntervalNTimes(r.Anchor, ic)
			if err != nil {
				return RecurrenceIterator{}, err
			}
			res = v
		}

		return RecurrenceIterator{
			r:         r,
			iteration: ic,
			At:        res,
		}, nil
		// If the anchor is T or in the past relative, we call .Next() repeatedly. If the new value is !Before T, we break
	}

	if r.Anchor.Before(t) {
		res := r.Anchor
		ic := 0

		for res.Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return RecurrenceIterator{}, fmt.Errorf("recurrence.PrevBefore: too many iterations")
			}
			ic += 1

			v, err := r.addIntervalNTimes(r.Anchor, ic)
			if err != nil {
				return RecurrenceIterator{}, err
			}

			if v.After(t) {
				break
			}

			res = v
		}

		return RecurrenceIterator{
			r:         r,
			iteration: ic,
			At:        res,
		}, nil
	}

	return r.Iterator(), nil
}

// Iterator returns an iterator that starts at the anchor.
func (r Recurrence) Iterator() RecurrenceIterator {
	return RecurrenceIterator{
		r:         r,
		iteration: 0,
		At:        r.Anchor,
	}
}

func (r Recurrence) addIntervalNTimes(t time.Time, nrIntervals int) (time.Time, error) {
	interval, err := r.Interval.Mul(nrIntervals)
	if err != nil {
		return time.Time{}, err
	}

	n, ok := interval.AddTo(t)
	if !ok {
		return time.Time{}, fmt.Errorf("next recurrence calculation wasn't exact, likely a fractional duration: %v", r.Interval)
	}
	return n, nil
}

type RecurrenceIterator struct {
	r         Recurrence
	iteration int
	At        time.Time
}

func (i RecurrenceIterator) Next() (RecurrenceIterator, error) {
	return i.iteratorWithDelta(1)
}

func (i RecurrenceIterator) Prev() (RecurrenceIterator, error) {
	return i.iteratorWithDelta(-1)
}

func (i RecurrenceIterator) iteratorWithDelta(delta int) (RecurrenceIterator, error) {
	i.iteration += delta

	res, err := i.r.addIntervalNTimes(i.r.Anchor, i.iteration)
	if err != nil {
		return RecurrenceIterator{}, err
	}

	i.At = res

	return i, nil
}

type RecurrenceInterval struct {
	datetime.ISODuration
}

var (
	RecurrencePeriodDaily RecurrenceInterval = RecurrenceInterval{datetime.DurationDay}
	RecurrencePeriodWeek  RecurrenceInterval = RecurrenceInterval{datetime.DurationWeek}
	RecurrencePeriodMonth RecurrenceInterval = RecurrenceInterval{datetime.DurationMonth}
	RecurrencePeriodYear  RecurrenceInterval = RecurrenceInterval{datetime.DurationYear}
)

func NewRecurrenceFromISODuration(p datetime.ISODuration, anchor time.Time) (Recurrence, error) {
	return NewRecurrence(RecurrenceInterval{p}, anchor)
}

func NewRecurrence(p RecurrenceInterval, anchor time.Time) (Recurrence, error) {
	rec := Recurrence{
		Interval: p,
		Anchor:   anchor,
	}

	if err := rec.Validate(); err != nil {
		return Recurrence{}, err
	}

	return rec, nil
}
