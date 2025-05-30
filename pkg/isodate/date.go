// Package datex is a wrapper around github.com/rickb777/date/v2 and github.com/rickb777/period
// so we don't depend on it directly.
package isodate

import (
	"fmt"
	"testing"
	"time"

	"github.com/govalues/decimal"
	"github.com/rickb777/period"
	"github.com/samber/lo"
)

const MAX_SAFE_ITERATION_COUNT = 1_000_000

type String period.ISOString

func (i String) Parse() (Period, error) {
	res, err := period.Parse(string(i))
	return Period{res}, err
}

func NewPeriod(years, months, weeks, days, hours, minutes, seconds int) Period {
	return Period{
		period.New(years, months, weeks, days, hours, minutes, seconds),
	}
}

// ParsePtrOrNil parses the ISO8601 string representation of the period or if ISOString is nil, returns nil
func (i *String) ParsePtrOrNil() (*Period, error) {
	if i == nil {
		return nil, nil
	}

	d, err := i.Parse()
	if err != nil {
		return nil, err
	}

	return lo.ToPtr(d), nil
}

func (i String) String() string {
	return string(i)
}

type Period struct {
	period.Period
}

// FIXME: clean up add and subtract

func (p Period) Normalise(exact bool) Period {
	return Period{p.Period.Normalise(exact)}
}

func (p Period) Simplify(exact bool) Period {
	return Period{p.Period.Simplify(exact)}
}

// InHours returns the value of the period in hours
func (p Period) InHours(daysInMonth int) (decimal.Decimal, error) {
	years, err := p.Period.YearsDecimal().Mul(decimal.MustNew(int64(daysInMonth*12*24), 0))
	if err != nil {
		return decimal.Zero, err
	}
	months, err := p.Period.MonthsDecimal().Mul(decimal.MustNew(int64(daysInMonth*24), 0))
	if err != nil {
		return decimal.Zero, err
	}
	weeks, err := p.Period.WeeksDecimal().Mul(decimal.MustNew(7*24, 0))
	if err != nil {
		return decimal.Zero, err
	}
	days, err := p.Period.DaysDecimal().Mul(decimal.MustNew(24, 0))
	if err != nil {
		return decimal.Zero, err
	}

	v, err := years.Add(months)
	if err != nil {
		return decimal.Zero, err
	}
	v, err = v.Add(weeks)
	if err != nil {
		return decimal.Zero, err
	}
	v, err = v.Add(days)
	if err != nil {
		return decimal.Zero, err
	}
	v, err = v.Add(p.Period.HoursDecimal())
	if err != nil {
		return decimal.Zero, err
	}

	return v, nil
}

func (p Period) Add(p2 Period) (Period, error) {
	s2 := period.ISOString(p2.String())
	per2, err := period.Parse(string(s2))
	if err != nil {
		return Period{}, err
	}
	p3, err := p.Period.Add(per2)
	return Period{p3}, err
}

func (p Period) Subtract(p2 Period) (Period, error) {
	s2 := period.ISOString(p2.String())
	per2, err := period.Parse(string(s2))
	if err != nil {
		return Period{}, err
	}
	p3, err := p.Period.Subtract(per2)
	return Period{p3}, err
}

// Compare returns -1 if p is less than p2, 0 if they are equal, and 1 if p is greater than p2
func (p Period) Compare(p2 Period) int {
	diff, _ := p.Period.Subtract(p2.Period)
	return diff.Sign()
}

// DivisibleBy returns true if the period is divisible by the smaller period (in days).
func (p Period) DivisibleBy(smaller Period) (bool, error) {
	l := p.Simplify(true)
	s := smaller.Simplify(true)

	if l.IsZero() || s.IsZero() || l.Compare(s) < 0 {
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
		if _, r, err := lh.QuoRem(sh); err != nil || !r.IsZero() {
			return false, err
		}
	}

	return true, nil
}

func Between(start time.Time, end time.Time) Period {
	per := period.Between(start, end)
	return Period{per}
}

// FromDuration creates an IMPRECISE Period from a time.Duration
func FromDuration(d time.Duration) Period {
	return Period{period.NewOf(d).Normalise(false).Simplify(false)}
}

// ISOString() returns the ISO8601 string representation of the period
func (p Period) ISOString() String {
	return String(p.Period.String())
}

// ISOStringPtrOrNil() returns the ISO8601 string representation of the period or if Period is nil, returns nil
func (d *Period) ISOStringPtrOrNil() *String {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.ISOString())
}

// Equal returns true if the two periods are equal
func (p *Period) Equal(v *Period) bool {
	if p == nil && v == nil {
		return true
	}

	if p == nil || v == nil {
		return false
	}

	return p.String() == v.String()
}

func MustParse(t *testing.T, s string) Period {
	res, err := period.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse period: %v", err)
	}

	return Period{res}
}
