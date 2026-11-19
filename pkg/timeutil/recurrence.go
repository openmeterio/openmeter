package timeutil

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/openmeterio/openmeter/pkg/datetime"
)

const MAX_SAFE_ITERATIONS = 1_000_000

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
	if t.IsZero() {
		return ClosedPeriod{}, fmt.Errorf("t cannot be zero")
	}

	// Period.Contains() is inclusive at the start and exclusive at the end, so the period
	// starts at the last boundary not after t and ends at the boundary that follows it.
	start, err := r.lastIterationNotAfter(t)
	if err != nil {
		return ClosedPeriod{}, err
	}

	end, err := start.Next()
	if err != nil {
		return ClosedPeriod{}, err
	}

	return ClosedPeriod{start.At, end.At}, nil
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

	prev, err := r.lastIterationNotAfter(t)
	if err != nil {
		return RecurrenceIterator{}, err
	}

	if boundaryBehavior == Inclusive && prev.At.Equal(t) {
		return prev, nil
	}

	return prev.Next()
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

	prev, err := r.lastIterationNotAfter(t)
	if err != nil {
		return RecurrenceIterator{}, err
	}

	if boundaryBehavior == Exclusive && prev.At.Equal(t) {
		return prev.Prev()
	}

	return prev, nil
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

// lastIterationNotAfter returns the last recurrence boundary at or before t.
//
// Boundary i is always computed as Anchor + i*Interval (never by accumulating steps), so
// month-end clamping does not drift. Boundaries grow monotonically with i, which lets us
// jump to an estimated index and correct it locally instead of walking from the anchor.
// The cost is independent of how many intervals separate t from the anchor.
func (r Recurrence) lastIterationNotAfter(t time.Time) (RecurrenceIterator, error) {
	iteration := r.estimateIteration(t)

	at, err := r.addIntervalNTimes(r.Anchor, iteration)
	if err != nil {
		return RecurrenceIterator{}, err
	}

	// The estimate assumes a nominal interval length (see estimateIteration), so the
	// boundary it lands on can be a few steps off in either direction.
	for steps := 0; at.After(t); steps++ {
		if steps >= MAX_SAFE_ITERATIONS {
			return RecurrenceIterator{}, fmt.Errorf("recurrence: too many iterations")
		}

		iteration--

		at, err = r.addIntervalNTimes(r.Anchor, iteration)
		if err != nil {
			return RecurrenceIterator{}, err
		}
	}

	for steps := 0; ; steps++ {
		if steps >= MAX_SAFE_ITERATIONS {
			return RecurrenceIterator{}, fmt.Errorf("recurrence: too many iterations")
		}

		next, err := r.addIntervalNTimes(r.Anchor, iteration+1)
		if err != nil {
			return RecurrenceIterator{}, err
		}

		if next.After(t) {
			break
		}

		iteration++
		at = next
	}

	return RecurrenceIterator{
		r:         r,
		iteration: iteration,
		At:        at,
	}, nil
}

// maxEstimatedIteration keeps the estimate far away from int overflow in Interval.Mul.
const maxEstimatedIteration = 1 << 40

// estimateIteration returns the index of a boundary close to t, using the interval's
// nominal length (a year is 365.2425 days, a month 1/12 of that, a day 24 hours). The
// real boundaries deviate from that by month-length and DST differences only, which
// lastIterationNotAfter corrects for.
func (r Recurrence) estimateIteration(t time.Time) int {
	step := r.Interval.DurationApprox().Seconds()
	if step <= 0 {
		return 0
	}

	// time.Time.Sub saturates beyond ~292 years, so compute the elapsed seconds directly.
	elapsed := float64(t.Unix()-r.Anchor.Unix()) + float64(t.Nanosecond()-r.Anchor.Nanosecond())/float64(time.Second)

	estimate := math.Floor(elapsed / step)

	switch {
	case math.IsNaN(estimate):
		return 0
	case estimate > maxEstimatedIteration:
		return maxEstimatedIteration
	case estimate < -maxEstimatedIteration:
		return -maxEstimatedIteration
	}

	return int(estimate)
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
	// Adding a zero calendar duration rebuilds the wall clock time, which resolves an
	// ambiguous DST time (the repeated hour) to its first occurrence. Boundary 0 must
	// stay the anchor itself.
	if nrIntervals == 0 {
		return t, nil
	}

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
