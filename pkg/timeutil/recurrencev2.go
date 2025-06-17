package timeutil

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/isodate"
)

type RecurrenceV2 struct {
	Interval RecurrenceIntervalV2 `json:"period"`
	// Anchor can be an arbitrary anchor time for the recurrence.
	// It does not have to be the last or the next time.
	Anchor time.Time `json:"anchor"`
}

// TODO: Used by subscription

// Returns a period where p.ContainsInclusive(t) is true
func (r RecurrenceV2) GetPeriodAt(t time.Time) (Period, error) {
	var def Period

	nextIt, err := r.NextAfter(t)
	if err != nil {
		return def, err
	}

	// As Period.ContainsInclusive() is inclusive at the start and exclusive at the end, we need to get the next time
	if nextIt.Time().Equal(t) {
		start := nextIt.Time()
		end, err := nextIt.Next()
		if err != nil {
			return def, err
		}

		return Period{start, end}, nil
	}

	// Otherwise the next time will be the end
	prevIt, err := r.PrevBefore(t)
	if err != nil {
		return def, err
	}

	return Period{prevIt.Time(), nextIt.Time()}, nil
}

// Entitlements

// NextAfter returns an iterator pointed to the next time after t that the
// recurrence should occur.
//
// If at t the recurrence should occur, it will return t.
func (r RecurrenceV2) NextAfter(t time.Time) (*recurrenceIterator, error) {
	it := r.Iterator()

	// If the anchor is in the future, we call .Prev() repeatedly. If the new value is Before t, we break
	if it.Time().After(t) {
		ic := 0
		for it.Time().After(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return nil, fmt.Errorf("recurrence.NextAfter: too many iterations")
			}
			ic += 1

			v, err := it.Prev()
			if err != nil {
				return nil, err
			}

			if v.Before(t) {
				_, err := it.Next()
				if err != nil {
					return nil, err
				}

				break
			}
		}
		// If the anchor is in the past, we call .Next() repeatedly. If the new value is !Before T, we break
	} else if it.Time().Before(t) {
		ic := 0
		for it.Time().Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return nil, fmt.Errorf("recurrence.NextAfter: too many iterations")
			}
			ic += 1

			_, err := it.Next()
			if err != nil {
				return nil, err
			}
		}
	}

	return it, nil
}

// PrevBefore returns the previous time before t that the recurrence should occur.
func (r RecurrenceV2) PrevBefore(t time.Time) (*recurrenceIterator, error) {
	it := r.Iterator()

	// If the anchor is in the future, we call .Prev() repeatedly. If the new value is Before t, we break
	if !it.Time().Before(t) {
		ic := 0
		for !it.Time().Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return nil, fmt.Errorf("recurrence.PrevBefore: too many iterations")
			}
			ic += 1

			_, err := it.Prev()
			if err != nil {
				return nil, err
			}
		}
		// If the anchor is T or in the past relative, we call .Next() repeatedly. If the new value is !Before T, we break
	} else {
		ic := 0
		for it.Time().Before(t) {
			if ic >= MAX_SAFE_ITERATIONS {
				return nil, fmt.Errorf("recurrence.PrevBefore: too many iterations")
			}
			ic += 1

			v, err := it.Next()
			if err != nil {
				return nil, err
			}

			if !v.Before(t) {
				_, err := it.Prev()
				if err != nil {
					return nil, err
				}

				break
			}
		}
	}

	return it, nil
}

type recurrenceIterator struct {
	RecurrenceV2
	t   time.Time
	idx int
}

func (r *recurrenceIterator) Time() time.Time {
	return r.t
}

// Next is assumed to be a sample from the recurrence, not an arbitrary time
func (r *recurrenceIterator) Next() (time.Time, error) {
	n, ok := r.addTo(r.Interval.Period, r.idx+1)
	if !ok {
		return time.Time{}, fmt.Errorf("next recurrence calculation wasn't exact, likely a fractional duration: %v", r.Interval)
	}

	r.t = n
	r.idx++

	return n, nil
}

func (r *recurrenceIterator) Prev() (time.Time, error) {
	n, ok := r.addTo(isodate.Period{r.Interval.Negate()}, r.idx-1)
	if !ok {
		return time.Time{}, fmt.Errorf("previous recurrence calculation wasn't exact, likely a fractional duration: %v", r.Interval)
	}

	r.t = n
	r.idx--

	return n, nil
}

func (r RecurrenceV2) Iterator() *recurrenceIterator {
	return &recurrenceIterator{
		RecurrenceV2: r,
		t:            r.Anchor,
		idx:          0,
	}
}

type date struct {
	Year  int
	Month int // 1 == January, etc.
	Day   int // 1 == first day of the month
}

func (r RecurrenceV2) addTo(p isodate.Period, count int) (time.Time, bool) {
	// TODO: negative interval support?

	targetPeriod := p.Mul(count)

	// Let's adjust the date parts first

	// TODO: civil also uses Go's time representation
	// date := civil.DateOf(r.Anchor)
	calcDate := date{
		Year:  r.Anchor.Year(),
		Month: int(r.Anchor.Month()),
		Day:   r.Anchor.Day(),
	}

	// Corner case: if anchor is on a leap day, we cannot allow the date to be normalized to March the 1st
	if calcDate.Month == 2 && calcDate.Day == 29 {
		targetYear := calcDate.Year + targetPeriod.Years()

		if daysIn(time.February, targetYear) != 29 {
			calcDate.Day = 28
		}
	}

	calcDate.Year += targetPeriod.Years()

	// Let's apply the month changes first
	if p.Months() != 0 {
		calcDate.Month += targetPeriod.Months()
		calcDate.Year, calcDate.Month = norm(calcDate.Year, calcDate.Month, 12)

		// In case we overshoot the days in the month, we force it to the last day of the month
		daysInMonth := daysIn(time.Month(calcDate.Month), calcDate.Year)
		if daysInMonth < calcDate.Day {
			calcDate.Day = daysInMonth
		}
	}

	// Let's apply the day changes last so that we can support expressions like P1M1D
	calcDate.Day += targetPeriod.DaysIncWeeks()

	// At this point we can use time.Date and live with the normalization results

	out := time.Date(
		calcDate.Year,
		time.Month(calcDate.Month),
		calcDate.Day,
		r.Anchor.Hour()+targetPeriod.Hours(),
		r.Anchor.Minute()+targetPeriod.Minutes(),
		r.Anchor.Second()+targetPeriod.Seconds(),
		r.Anchor.Nanosecond(),
		r.Anchor.Location(),
	)

	return out, true
}

type RecurrenceIntervalV2 struct {
	isodate.Period
}

var (
	RecurrencePeriodDailyV2 = RecurrenceIntervalV2{isodate.NewPeriod(0, 0, 0, 1, 0, 0, 0)}
	RecurrencePeriodWeekV2  = RecurrenceIntervalV2{isodate.NewPeriod(0, 0, 1, 0, 0, 0, 0)}
	RecurrencePeriodMonthV2 = RecurrenceIntervalV2{isodate.NewPeriod(0, 1, 0, 0, 0, 0, 0)}
	RecurrencePeriodYearV2  = RecurrenceIntervalV2{isodate.NewPeriod(1, 0, 0, 0, 0, 0, 0)}
)

func FromISODurationV2(p *isodate.Period, anchor time.Time) (RecurrenceV2, error) {
	if p == nil {
		return RecurrenceV2{}, fmt.Errorf("period cannot be nil")
	}

	return RecurrenceV2{
		Interval: RecurrenceIntervalV2{*p},
		Anchor:   anchor,
	}, nil
}

func daysIn(m time.Month, year int) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// taken from time.go
func norm(hi, lo, base int) (nhi, nlo int) {
	if lo < 0 {
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo
}
