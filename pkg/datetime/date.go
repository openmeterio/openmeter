package datetime

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/govalues/decimal"
	"github.com/rickb777/period"
	"github.com/samber/lo"
)

const MAX_SAFE_ITERATION_COUNT = 1_000_000

type ISODurationString period.ISOString

func (i ISODurationString) Parse() (ISODuration, error) {
	res, err := period.Parse(string(i))
	return ISODuration{res}, err
}

func NewPeriod(years, months, weeks, days, hours, minutes, seconds int) ISODuration {
	return ISODuration{
		period.New(years, months, weeks, days, hours, minutes, seconds),
	}
}

// ParsePtrOrNil parses the ISO8601 string representation of the period or if ISOString is nil, returns nil
func (i *ISODurationString) ParsePtrOrNil() (*ISODuration, error) {
	if i == nil {
		return nil, nil
	}

	d, err := i.Parse()
	if err != nil {
		return nil, err
	}

	return lo.ToPtr(d), nil
}

func (i ISODurationString) String() string {
	return string(i)
}

// ISODuration is a wrapper around github.com/rickb777/period so we don't depend on it directly.
type ISODuration struct {
	period.Period
}

// FIXME: clean up add and subtract

func (p ISODuration) Normalise(exact bool) ISODuration {
	return ISODuration{p.Period.Normalise(exact)}
}

func (p ISODuration) Simplify(exact bool) ISODuration {
	return ISODuration{p.Period.Simplify(exact)}
}

// InHours returns the value of the period in hours
func (p ISODuration) InHours(daysInMonth int) (alpacadecimal.Decimal, error) {
	zero := alpacadecimal.NewFromInt(0)

	// You might be thinking, a year is supposed to be 365 or 366 days, not 372 or 360 or 348 or 336
	// (as this below line calculates it depending on days in the month)
	// Lucky for us, the method as a whole gives correct results
	years, err := p.Period.YearsDecimal().Mul(decimal.MustNew(int64(daysInMonth*12*24), 0))
	if err != nil {
		return zero, err
	}
	months, err := p.Period.MonthsDecimal().Mul(decimal.MustNew(int64(daysInMonth*24), 0))
	if err != nil {
		return zero, err
	}
	weeks, err := p.Period.WeeksDecimal().Mul(decimal.MustNew(7*24, 0))
	if err != nil {
		return zero, err
	}
	days, err := p.Period.DaysDecimal().Mul(decimal.MustNew(24, 0))
	if err != nil {
		return zero, err
	}

	v, err := years.Add(months)
	if err != nil {
		return zero, err
	}
	v, err = v.Add(weeks)
	if err != nil {
		return zero, err
	}
	v, err = v.Add(days)
	if err != nil {
		return zero, err
	}
	v, err = v.Add(p.Period.HoursDecimal())
	if err != nil {
		return zero, err
	}

	scale := v.MinScale()
	whole, frac, ok := v.Int64(scale)
	if !ok {
		return zero, fmt.Errorf("failed to convert to int64")
	}

	if frac != 0 {
		return zero, fmt.Errorf("we shouldn't have any fractional part here")
	}

	return alpacadecimal.NewFromInt(whole), nil
}

func (p ISODuration) Add(p2 ISODuration) (ISODuration, error) {
	s2 := period.ISOString(p2.String())
	per2, err := period.Parse(string(s2))
	if err != nil {
		return ISODuration{}, err
	}
	p3, err := p.Period.Add(per2)
	return ISODuration{p3}, err
}

func (p ISODuration) Subtract(p2 ISODuration) (ISODuration, error) {
	s2 := period.ISOString(p2.String())
	per2, err := period.Parse(string(s2))
	if err != nil {
		return ISODuration{}, err
	}
	p3, err := p.Period.Subtract(per2)
	return ISODuration{p3}, err
}

// DivisibleBy returns true if the period is divisible by the smaller period (in hours).
func (p ISODuration) DivisibleBy(smaller ISODuration) (bool, error) {
	l := p.Simplify(true)
	s := smaller.Simplify(true)

	if l.IsZero() || s.IsZero() {
		return false, nil
	}

	if l.Minutes() != 0 || l.Seconds() != 0 || s.Minutes() != 0 || s.Seconds() != 0 {
		return false, fmt.Errorf("divisible periods must be whole numbers of hours")
	}

	testDaysInMonth := []int{28, 29, 30, 31}
	for _, daysInMonth := range testDaysInMonth {
		// get periods in hours
		lh, err := l.InHours(daysInMonth)
		if err != nil {
			return false, err
		}
		sh, err := s.InHours(daysInMonth)
		if err != nil {
			return false, err
		}

		if _, r := lh.QuoRem(sh, 0); !r.IsZero() {
			return false, err
		}
	}

	return true, nil
}

func Between(start time.Time, end time.Time) ISODuration {
	per := period.Between(start, end)
	return ISODuration{per}
}

// FromDuration creates an IMPRECISE Period from a time.Duration
func FromDuration(d time.Duration) ISODuration {
	return ISODuration{period.NewOf(d).Normalise(false).Simplify(false)}
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

func MustParse(t *testing.T, s string) ISODuration {
	res, err := period.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse period: %v", err)
	}

	return ISODuration{res}
}
