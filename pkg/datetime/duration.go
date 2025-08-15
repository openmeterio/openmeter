package datetime

import (
	"encoding/json"
	"time"

	"github.com/govalues/decimal"
	"github.com/rickb777/period"
	"github.com/samber/lo"
)

// ISODuration represents ISO 8601 duration.
// It is mostly a wrapper around github.com/rickb777/period
type ISODuration struct {
	period.Period
}

func NewISODuration(years, months, weeks, days, hours, minutes, seconds int) ISODuration {
	return ISODuration{
		period.New(years, months, weeks, days, hours, minutes, seconds),
	}
}

// MarshalJSON marshals the Duration to a JSON string.
func (d ISODuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Period)
}

// UnmarshalJSON unmarshals the Duration from a JSON string.
func (d *ISODuration) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &d.Period)
}

// ISOString() returns the ISO8601 string representation of the period
func (p ISODuration) ISOString() ISODurationString {
	return ISODurationString(p.Period.String())
}

// ISOStringPtrOrNil() returns the ISO8601 string representation of the period or if Period is nil, returns nil
func (d *ISODuration) ISOStringPtrOrNil() *ISODurationString {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.ISOString())
}

// Equal returns true if the two periods are equal
func (p *ISODuration) Equal(v *ISODuration) bool {
	if p == nil && v == nil {
		return true
	}

	if p == nil || v == nil {
		return false
	}

	return p.String() == v.String()
}

func (p ISODuration) Normalise(exact bool) ISODuration {
	return ISODuration{p.Period.Normalise(exact)}
}

func (p ISODuration) Simplify(exact bool) ISODuration {
	return ISODuration{p.Period.Simplify(exact)}
}

func (p ISODuration) Negate() ISODuration {
	return ISODuration{p.Period.Negate()}
}

func (p ISODuration) Add(p2 ISODuration) (ISODuration, error) {
	s2 := period.ISOString(p2.String())
	per2, err := period.Parse(string(s2))
	if err != nil {
		return ISODuration{}, err
	}
	p3, err := p.Period.Add(per2)
	if err != nil {
		return ISODuration{}, NewDurationArithmeticError(p.String(), err)
	}
	return ISODuration{p3}, nil
}

func (p ISODuration) Subtract(p2 ISODuration) (ISODuration, error) {
	s2 := period.ISOString(p2.String())
	per2, err := period.Parse(string(s2))
	if err != nil {
		return ISODuration{}, err
	}
	p3, err := p.Period.Subtract(per2)
	if err != nil {
		return ISODuration{}, NewDurationArithmeticError(p.String(), err)
	}
	return ISODuration{p3}, nil
}

// AddTo adds the duration to the time.Time and returns the result and a boolean indicating if the conversion was precise.
// The conversion is always precise but the signature is kept for backwards compatibility.
func (p ISODuration) AddTo(t time.Time) (time.Time, bool) {
	// Use our custom date arithmetic to handle month/year overflow correctly
	result := NewDateTime(t).Add(p).AsTime()

	return result, true
}

func ISODurationBetween(start time.Time, end time.Time) ISODuration {
	per := period.Between(start, end)
	return ISODuration{per}
}

// ISODurationFromDuration creates an IMPRECISE Period from a time.Duration
func ISODurationFromDuration(d time.Duration) ISODuration {
	return ISODuration{period.NewOf(d).Normalise(false).Simplify(false)}
}

// DivisibleBy returns true if the duration is divisible by the smaller duration.
func (d ISODuration) DivisibleBy(smaller ISODuration) (bool, error) {
	l := d.Simplify(true)
	s := smaller.Simplify(true)

	if l.IsZero() {
		return false, nil
	}

	if s.IsZero() {
		return false, nil
	}

	// Test with different days-per-month and hours-per-day scenarios
	testDaysInMonth := []int{28, 29, 30, 31}
	testHoursInDays := []int{25, 24, 23}

	for _, daysInMonth := range testDaysInMonth {
		for _, hoursInDays := range testHoursInDays {
			largerSeconds, err := convertPeriodToSeconds(l.Period, daysInMonth, hoursInDays)
			if err != nil {
				return false, err
			}

			smallerSeconds, err := convertPeriodToSeconds(s.Period, daysInMonth, hoursInDays)
			if err != nil {
				return false, err
			}

			if smallerSeconds.IsZero() {
				return false, nil
			}

			quotient, remainder, err := largerSeconds.QuoRem(smallerSeconds)
			if err != nil {
				return false, err
			}
			if !remainder.IsZero() {
				return false, nil
			}

			if quotient.Sign() <= 0 || !quotient.IsInt() {
				return false, nil
			}
		}
	}

	return true, nil
}

func (p ISODuration) Mul(n int) (ISODuration, error) {
	nDecimal, err := decimal.NewFromInt64(int64(n), 0, 0)
	if err != nil {
		return ISODuration{}, err
	}

	per, err := p.Period.Mul(nDecimal)
	if err != nil {
		return ISODuration{}, err
	}
	return ISODuration{per}, nil
}

// convertPeriodToSeconds converts a period to total seconds using decimal precision
func convertPeriodToSeconds(p period.Period, daysInMonth int, hoursInDays int) (decimal.Decimal, error) {
	zero := decimal.Zero

	yearsMul, err := decimal.New(int64(daysInMonth*12*hoursInDays*3600), 0)
	if err != nil {
		return zero, err
	}

	// Convert years to seconds: years * (daysInMonth * 12) * hoursInDays * 3600
	years, err := p.YearsDecimal().Mul(yearsMul)
	if err != nil {
		return zero, err
	}

	monthsMul, err := decimal.New(int64(daysInMonth*hoursInDays*3600), 0)
	if err != nil {
		return zero, err
	}

	// Convert months to seconds: months * daysInMonth * hoursInDays * 3600
	months, err := p.MonthsDecimal().Mul(monthsMul)
	if err != nil {
		return zero, err
	}

	weeksMul, err := decimal.New(int64(7*hoursInDays*3600), 0)
	if err != nil {
		return zero, err
	}

	// Convert weeks to seconds: weeks * 7 * hoursInDays * 3600
	weeks, err := p.WeeksDecimal().Mul(weeksMul)
	if err != nil {
		return zero, err
	}

	// Convert days to seconds: days * hoursInDays * 3600
	daysMul, err := decimal.New(int64(hoursInDays*3600), 0)
	if err != nil {
		return zero, err
	}

	days, err := p.DaysDecimal().Mul(daysMul)
	if err != nil {
		return zero, err
	}

	// Convert hours to seconds: hours * 3600
	hoursMul, err := decimal.New(int64(3600), 0)
	if err != nil {
		return zero, err
	}

	hours, err := p.HoursDecimal().Mul(hoursMul)
	if err != nil {
		return zero, err
	}

	// Convert minutes to seconds: minutes * 60
	minutesMul, err := decimal.New(int64(60), 0)
	if err != nil {
		return zero, err
	}

	minutes, err := p.MinutesDecimal().Mul(minutesMul)
	if err != nil {
		return zero, err
	}

	// Seconds are already in the right unit
	seconds := p.SecondsDecimal()

	// Sum all components
	result, err := years.Add(months)
	if err != nil {
		return zero, err
	}
	result, err = result.Add(weeks)
	if err != nil {
		return zero, err
	}
	result, err = result.Add(days)
	if err != nil {
		return zero, err
	}
	result, err = result.Add(hours)
	if err != nil {
		return zero, err
	}
	result, err = result.Add(minutes)
	if err != nil {
		return zero, err
	}
	result, err = result.Add(seconds)
	if err != nil {
		return zero, err
	}

	return result, nil
}
