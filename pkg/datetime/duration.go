package datetime

import (
	"encoding/json"

	"github.com/govalues/decimal"
	"github.com/rickb777/period"
	"github.com/samber/lo"
)

// Duration represents a duration of time.
type Duration struct {
	period.Period
}

// NewDuration creates a new Duration from a period.Period.
func NewDuration(p period.Period) Duration {
	return Duration{p}
}

// DurationString is a string that represents a duration of time.
type DurationString period.ISOString

// Parse parses an ISO8601 duration string into a Duration.
func (d DurationString) Parse() (Duration, error) {
	res, err := period.Parse(string(d))
	if err != nil {
		return Duration{}, NewDurationParseError(string(d), err)
	}

	return Duration{res}, nil
}

// ParsePtrOrNil parses the ISO8601 string representation of the duration or if it is nil, returns nil
func (i *DurationString) ParsePtrOrNil() (*Duration, error) {
	if i == nil {
		return nil, nil
	}

	d, err := i.Parse()
	if err != nil {
		return nil, err
	}

	return lo.ToPtr(d), nil
}

// MarshalJSON marshals the Duration to a JSON string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Period)
}

// UnmarshalJSON unmarshals the Duration from a JSON string.
func (d *Duration) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &d.Period)
}

// Add adds two durations together.
func (d Duration) Add(d2 Duration) (Duration, error) {
	p, err := d.Period.Add(d2.Period)
	if err != nil {
		return Duration{}, err
	}

	return Duration{p}, nil
}

// Subtract subtracts one duration from another.
func (d Duration) Subtract(d2 Duration) (Duration, error) {
	p, err := d.Period.Subtract(d2.Period)
	if err != nil {
		return Duration{}, err
	}

	return Duration{p}, nil
}

// DivisibleBy returns true if the duration is divisible by the smaller duration.
func (d Duration) DivisibleBy(smaller Duration) (bool, error) {
	l := d.Simplify(true)
	s := smaller.Simplify(true)

	if l.IsZero() {
		return true, nil
	}

	if s.IsZero() {
		return false, nil
	}

	// Test with different days-per-month and hours-per-day scenarios
	testDaysInMonth := []int{28, 29, 30, 31}
	testHoursInDays := []int{24, 23}

	for _, daysInMonth := range testDaysInMonth {
		for _, hoursInDays := range testHoursInDays {
			largerSeconds, err := convertPeriodToSeconds(l, daysInMonth, hoursInDays)
			if err != nil {
				return false, err
			}

			smallerSeconds, err := convertPeriodToSeconds(s, daysInMonth, hoursInDays)
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

// convertPeriodToSeconds converts a period to total seconds using decimal precision
func convertPeriodToSeconds(p period.Period, daysInMonth int, hoursInDays int) (decimal.Decimal, error) {
	zero := decimal.MustNew(0, 0)

	// Convert years to seconds: years * (daysInMonth * 12) * hoursInDays * 3600
	years, err := p.YearsDecimal().Mul(decimal.MustNew(int64(daysInMonth*12*hoursInDays*3600), 0))
	if err != nil {
		return zero, err
	}

	// Convert months to seconds: months * daysInMonth * hoursInDays * 3600
	months, err := p.MonthsDecimal().Mul(decimal.MustNew(int64(daysInMonth*hoursInDays*3600), 0))
	if err != nil {
		return zero, err
	}

	// Convert weeks to seconds: weeks * 7 * hoursInDays * 3600
	weeks, err := p.WeeksDecimal().Mul(decimal.MustNew(int64(7*hoursInDays*3600), 0))
	if err != nil {
		return zero, err
	}

	// Convert days to seconds: days * hoursInDays * 3600
	days, err := p.DaysDecimal().Mul(decimal.MustNew(int64(hoursInDays*3600), 0))
	if err != nil {
		return zero, err
	}

	// Convert hours to seconds: hours * 3600
	hours, err := p.HoursDecimal().Mul(decimal.MustNew(3600, 0))
	if err != nil {
		return zero, err
	}

	// Convert minutes to seconds: minutes * 60
	minutes, err := p.MinutesDecimal().Mul(decimal.MustNew(60, 0))
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
