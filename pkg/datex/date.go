// Package datex is a wrapper around github.com/rickb777/date/v2 and github.com/rickb777/period
// so we don't depend on it directly.
package datex

import (
	"testing"
	"time"

	"github.com/rickb777/period"
	"github.com/samber/lo"
)

type ISOString period.ISOString

func (i ISOString) Parse() (Period, error) {
	res, err := period.Parse(string(i))
	return Period{res}, err
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

// FromDuration creates a Period from a time.Duration
func FromDuration(d time.Duration) Period {
	return Period{period.NewOf(d).Normalise(false)}
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

func MustParse(t *testing.T, s string) Period {
	res, err := period.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse period: %v", err)
	}

	return Period{res}
}
