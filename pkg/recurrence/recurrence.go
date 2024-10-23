package recurrence

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/datex"
)

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
	if i.After(t) {
		for i.After(t) {
			v, err := r.Prev(i)
			if err != nil {
				return time.Time{}, err
			}
			i = v
		}
	}
	for i.Before(t) {
		v, err := r.Next(i)
		if err != nil {
			return time.Time{}, err
		}
		i = v
	}

	return i, nil
}

// PrevBefore returns the previous time before t that the recurrence should occur.
func (r Recurrence) PrevBefore(t time.Time) (time.Time, error) {
	i := r.Anchor
	if i.Before(t) {
		for i.Before(t) {
			v, err := r.Next(i)
			if err != nil {
				return time.Time{}, err
			}
			i = v
		}
	}
	for i.After(t) || i.Equal(t) {
		v, err := r.Prev(i)
		if err != nil {
			return time.Time{}, err
		}
		i = v
	}

	return i, nil
}

func (r Recurrence) Next(t time.Time) (time.Time, error) {
	switch r.Interval {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, 1), nil
	case RecurrencePeriodWeek:
		return t.AddDate(0, 0, 7), nil
	case RecurrencePeriodMonth:
		return t.AddDate(0, 1, 0), nil
	case RecurrencePeriodYear:
		return t.AddDate(1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Interval)
}

func (r Recurrence) Prev(t time.Time) (time.Time, error) {
	switch r.Interval {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, -1), nil
	case RecurrencePeriodWeek:
		return t.AddDate(0, 0, -7), nil
	case RecurrencePeriodMonth:
		return t.AddDate(0, -1, 0), nil
	case RecurrencePeriodYear:
		return t.AddDate(-1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Interval)
}

type RecurrenceInterval string

const (
	RecurrencePeriodDaily RecurrenceInterval = "DAY"
	RecurrencePeriodWeek  RecurrenceInterval = "WEEK"
	RecurrencePeriodMonth RecurrenceInterval = "MONTH"
	RecurrencePeriodYear  RecurrenceInterval = "YEAR"
)

func (RecurrenceInterval) Values() (kinds []string) {
	for _, s := range []RecurrenceInterval{
		RecurrencePeriodDaily,
		RecurrencePeriodWeek,
		RecurrencePeriodMonth,
		RecurrencePeriodYear,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (rp RecurrenceInterval) IsValid() bool {
	switch rp {
	case RecurrencePeriodDaily,
		RecurrencePeriodWeek,
		RecurrencePeriodMonth,
		RecurrencePeriodYear:
		return true
	}
	return false
}

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
