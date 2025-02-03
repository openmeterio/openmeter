// Package datex is a wrapper around github.com/rickb777/date/v2 and github.com/rickb777/period
// so we don't depend on it directly.
package datex

import (
	"fmt"
	"testing"
	"time"

	"github.com/rickb777/period"
	"github.com/samber/lo"
)

const MAX_SAFE_ITERATION_COUNT = 1_000_000

type ISOString period.ISOString

func (i ISOString) Parse() (Period, error) {
	res, err := period.Parse(string(i))
	return Period{res}, err
}

func NewPeriod(years, months, weeks, days, hours, minutes, seconds int) Period {
	return Period{
		period.New(years, months, weeks, days, hours, minutes, seconds),
	}
}

// ParsePtrOrNil parses the ISO8601 string representation of the period or if ISOString is nil, returns nil
func (i *ISOString) ParsePtrOrNil() (*Period, error) {
	if i == nil {
		return nil, nil
	}

	d, err := i.Parse()
	if err != nil {
		return nil, err
	}

	return lo.ToPtr(d), nil
}

func (i ISOString) String() string {
	return string(i)
}

type Period struct {
	period.Period
}

// FIXME: clean up add and subtract

func (p Period) Normalise(exact bool) Period {
	return Period{p.Period.Normalise(exact)}
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

// FromDuration creates a Period from a time.Duration
func PeriodsAlign(larger Period, smaller Period) (bool, error) {
	p, err := larger.Subtract(smaller)
	if err != nil {
		return false, err
	}

	if p.Sign() == -1 {
		return false, fmt.Errorf("smaller period is larger than larger period")
	}

	per := smaller
	for i := 1; i < MAX_SAFE_ITERATION_COUNT; i++ {
		per, err = per.Add(smaller)
		if err != nil {
			return false, err
		}

		diff, err := larger.Subtract(per)
		if err != nil {
			return false, err
		}

		// It's an exact match
		if diff.Sign() == 0 {
			return true, nil
		}

		// We've overshot without a match
		if diff.Sign() == -1 {
			return false, nil
		}
	}

	return false, nil
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
func (p Period) ISOString() ISOString {
	return ISOString(p.Period.String())
}

// ISOStringPtrOrNil() returns the ISO8601 string representation of the period or if Period is nil, returns nil
func (d *Period) ISOStringPtrOrNil() *ISOString {
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
