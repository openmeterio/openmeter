package credit

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api/types"
)

type Recurrence struct {
	Period RecurrencePeriod `json:"period"`
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
	switch r.Period {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, 1), nil
	case RecurrencePeriodWeek:
		return t.AddDate(0, 0, 7), nil
	case RecurrencePeriodMonth:
		return t.AddDate(0, 1, 0), nil
	case RecurrencePeriodYear:
		return t.AddDate(1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Period)
}

func (r Recurrence) Prev(t time.Time) (time.Time, error) {
	switch r.Period {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, -1), nil
	case RecurrencePeriodWeek:
		return t.AddDate(0, 0, -7), nil
	case RecurrencePeriodMonth:
		return t.AddDate(0, -1, 0), nil
	case RecurrencePeriodYear:
		return t.AddDate(-1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Period)
}

type RecurrencePeriod types.RecurringPeriodEnum

const (
	RecurrencePeriodDaily RecurrencePeriod = "DAY"
	RecurrencePeriodWeek  RecurrencePeriod = "WEEK"
	RecurrencePeriodMonth RecurrencePeriod = "MONTH"
	RecurrencePeriodYear  RecurrencePeriod = "YEAR"
)

func (RecurrencePeriod) Values() (kinds []string) {
	for _, s := range []RecurrencePeriod{
		RecurrencePeriodDaily,
		RecurrencePeriodWeek,
		RecurrencePeriodMonth,
		RecurrencePeriodYear,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (rp RecurrencePeriod) IsValid() bool {
	switch rp {
	case RecurrencePeriodDaily,
		RecurrencePeriodWeek,
		RecurrencePeriodMonth,
		RecurrencePeriodYear:
		return true
	}
	return false
}
