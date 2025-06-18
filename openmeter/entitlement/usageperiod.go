package entitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type UsagePeriod timeutil.Recurrence

func (u UsagePeriod) Validate() error {
	hour := isodate.NewPeriod(0, 0, 0, 0, 1, 0, 0)
	if diff, err := u.Interval.Period.Subtract(hour); err == nil && diff.Sign() == -1 {
		return errors.New("UsagePeriod must be at least 1 hour")
	}

	return nil
}

func (u UsagePeriod) AsRecurrence() timeutil.Recurrence {
	return timeutil.Recurrence{
		Anchor:   u.Anchor,
		Interval: u.Interval,
	}
}

func (u UsagePeriod) Equal(other UsagePeriod) bool {
	if u.Interval != other.Interval {
		return false
	}

	if !u.Anchor.Equal(other.Anchor) {
		return false
	}

	return true
}

// The returned period is exclusive at the end end inclusive in the start
func (u UsagePeriod) GetCurrentPeriodAt(at time.Time) (timeutil.ClosedPeriod, error) {
	return u.AsRecurrence().GetPeriodAt(at)
}
