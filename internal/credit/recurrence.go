package credit

import (
	"fmt"
	"time"
)

type Recurrence struct {
	MaxRolloverAmount float64          `json:"maxRolloverAmount"`
	Period            RecurrencePeriod `json:"period"`
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
	case RecurrencePeriodWeekly:
		return t.AddDate(0, 0, 7), nil
	case RecurrencePeriodMonthly:
		return t.AddDate(0, 1, 0), nil
	case RecurrencePeriodYearly:
		return t.AddDate(1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Period)
}

func (r Recurrence) Prev(t time.Time) (time.Time, error) {
	switch r.Period {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, -1), nil
	case RecurrencePeriodWeekly:
		return t.AddDate(0, 0, -7), nil
	case RecurrencePeriodMonthly:
		return t.AddDate(0, -1, 0), nil
	case RecurrencePeriodYearly:
		return t.AddDate(-1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Period)
}

type RecurrencePeriod string

const (
	RecurrencePeriodDaily   RecurrencePeriod = "DAILY"
	RecurrencePeriodWeekly  RecurrencePeriod = "WEEKLY"
	RecurrencePeriodMonthly RecurrencePeriod = "MONTHLY"
	RecurrencePeriodYearly  RecurrencePeriod = "YEARLY"
)

func (rp RecurrencePeriod) IsValid() bool {
	switch rp {
	case RecurrencePeriodDaily,
		RecurrencePeriodWeekly,
		RecurrencePeriodMonthly,
		RecurrencePeriodYearly:
		return true
	}
	return false
}
