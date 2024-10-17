// Package datex is a wrapper around github.com/rickb777/date/v2 and github.com/rickb777/period
// so we don't depend on it directly.
package datex

import (
	"time"

	"github.com/rickb777/period"
)

type ISOString period.ISOString

func (i ISOString) Parse() (Period, error) {
	res, err := period.Parse(string(i))
	return Period{res}, err
}

type Period struct {
	period.Period
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

// FromDuration creates an IMPRECISE Period from a time.Duration
func FromDuration(d time.Duration) Period {
	return Period{period.NewOf(d).Normalise(false)}
}
