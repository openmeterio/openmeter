package datetime

import (
	"iter"
	"time"

	"github.com/govalues/decimal"
)

// Period represents a period of time.
type Period struct {
	start    DateTime
	duration Duration
}

// NewPeriod creates a new period.
func NewPeriod(start DateTime, duration Duration) Period {
	return Period{start: start, duration: duration}
}

// Start returns the start of the period.
func (p Period) Start() DateTime {
	return p.start
}

// Duration returns the duration of the period.
func (p Period) Duration() Duration {
	return p.duration
}

// End returns the end of the period.
func (p Period) End() DateTime {
	return p.start.Add(p.duration)
}

// RecurringPeriod represents a recurring period of time.
type RecurringPeriod struct {
	Period
}

// NewRecurringPeriod creates a new recurring period.
func NewRecurringPeriod(start DateTime, duration Duration) RecurringPeriod {
	return RecurringPeriod{Period: NewPeriod(start, duration)}
}

// recurringPeriodIterator maintains state for generating a sequence of RecurringPeriod values at regular intervals.
type recurringPeriodIterator struct {
	Period
	currentIndex decimal.Decimal
}

// Next returns the next recurring date time since the start date time.
// It advances the internal counter and calculates the next occurrence.
func (r *recurringPeriodIterator) next() (DateTime, error) {
	nextIndex, err := r.currentIndex.Add(decimal.One)
	if err != nil {
		return DateTime{}, err
	}

	p, err := r.Period.Duration().Mul(nextIndex)
	if err != nil {
		return DateTime{}, err
	}

	r.currentIndex = nextIndex

	return r.Period.Start().Add(NewDuration(p)), nil
}

// Iter returns an iterator function that yields DateTime values from the start time
// at each duration interval.
func (r *RecurringPeriod) Iter() iter.Seq[DateTime] {
	iter := &recurringPeriodIterator{
		Period:       r.Period,
		currentIndex: decimal.Zero,
	}

	return func(yield func(DateTime) bool) {
		// First yield the starting time
		if !yield(r.Period.Start()) {
			return
		}

		for {
			next, err := iter.next()
			if err != nil {
				// Arithmetic errors in decimal operations indicate programming errors
				return
			}

			if !yield(next) {
				return
			}
		}
	}
}

// IterInRange returns an iterator function that yields DateTime values from the start time
// at each duration interval, but only for periods within the specified time range.
// The range is inclusive of start and exclusive of end.
func (r *RecurringPeriod) IterInRange(start, end time.Time) iter.Seq[DateTime] {
	iter := r.Iter()

	return func(yield func(DateTime) bool) {
		for dt := range iter {
			// Yield the period within range (inclusive of start, exclusive of end)
			if !dt.Time.Before(start) && dt.Time.Before(end) {
				if !yield(dt) {
					return
				}
			}

			// Exit if the time is after the end time
			if dt.Time.After(end) {
				return
			}
		}
	}
}

// ValuesInRange returns a slice of DateTime values from the start time
// at each duration interval, but only for periods within the specified time range.
// The range is inclusive of start and exclusive of end.
func (r *RecurringPeriod) ValuesInRange(start, end time.Time) []DateTime {
	values := make([]DateTime, 0)

	for dt := range r.IterInRange(start, end) {
		values = append(values, dt)
	}

	return values
}
