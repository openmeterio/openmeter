package entitlement

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var MAX_SAFE_ITERATIONS = 1000000

// Intended for testing mainly
func NewUsagePeriodFromRecurrence(rec timeutil.Recurrence) UsagePeriod {
	return NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
		timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
			return r.Anchor
		})(rec),
	})
}

// When providing an initial value (single element in list), for metered entitlements, timed.GetTime() should return measureUsageFrom!
func NewUsagePeriod(recs []timeutil.Timed[timeutil.Recurrence]) UsagePeriod {
	return UsagePeriod{
		recs: timeutil.NewTimeline(recs),
	}
}

// Intended for testing mainly
func NewUsagePeriodInputFromRecurrence(rec timeutil.Recurrence) UsagePeriodInput {
	return timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
		return r.Anchor
	})(rec)
}

func NewStartingUsagePeriod(rec timeutil.Recurrence, start time.Time) UsagePeriod {
	return NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
		timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
			return start
		})(rec),
	})
}

func NewStartingUsagePeriodInput(rec timeutil.Recurrence, start time.Time) UsagePeriodInput {
	return timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
		return start
	})(rec)
}

type UsagePeriodInput = timeutil.Timed[timeutil.Recurrence]

type UsagePeriod struct {
	recs timeutil.Timeline[timeutil.Recurrence]
}

func (u UsagePeriod) Validate() error {
	var errs []error

	// Let's validate that we do have some recurrences
	if len(u.recs.GetTimes()) == 0 {
		errs = append(errs, errors.New("UsagePeriod must have at least one recurrence"))
	}

	hour := isodate.NewPeriod(0, 0, 0, 0, 1, 0, 0)
	for i := range u.recs.GetTimes() {
		rec := u.recs.GetAt(i).GetValue()

		// Let's validate the recurrence
		if err := rec.Validate(); err != nil {
			errs = append(errs, err)
		}

		// Let's validate that the recurrences are all at least 1 hour long
		if diff, err := rec.Interval.Period.Subtract(hour); err == nil && diff.Sign() == -1 {
			errs = append(errs, errors.New("UsagePeriod must be at least 1 hour"))
		}
	}

	return errors.Join(errs...)
}

func (u UsagePeriod) GetOriginalValueAsUsagePeriodInput() UsagePeriodInput {
	first := u.recs.GetAt(0)

	return timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
		return first.GetTime()
	})(first.GetValue())
}

func (u UsagePeriod) Equal(other UsagePeriod) bool {
	if len(u.recs.GetTimes()) != len(other.recs.GetTimes()) {
		return false
	}

	for i := range u.recs.GetTimes() {
		if !u.recs.GetAt(i).Equal(other.recs.GetAt(i)) {
			return false
		}
	}

	return true
}

func (u UsagePeriod) GetCurrentPeriodAt(at time.Time) (timeutil.ClosedPeriod, error) {
	inpAt, idx, err := u.GetUsagePeriodInputAt(at)
	if err != nil {
		return timeutil.ClosedPeriod{}, err
	}

	if at.Before(inpAt.GetTime()) {
		at = inpAt.GetTime() // If we're querying before the first recurrence, we want to return the first period
	}

	fullPer, err := inpAt.GetValue().GetPeriodAt(at)
	if err != nil {
		return timeutil.ClosedPeriod{}, err
	}

	// We need to truncate the period with any boundaries present
	// Let's truncate the start
	if inpAt.GetTime().After(fullPer.From) {
		fullPer.From = inpAt.GetTime()
	}

	// Let's truncate the end
	if idx < len(u.recs.GetTimes())-1 {
		next := u.recs.GetAt(idx + 1)
		if next.GetTime().Before(fullPer.To) {
			fullPer.To = next.GetTime()
		}
	}

	return fullPer, nil
}

func (u UsagePeriod) GetResetTimelineInclusive(inPeriod timeutil.ClosedPeriod) (timeutil.SimpleTimeline, error) {
	_, firstPerIdx, err := u.GetUsagePeriodInputAt(inPeriod.From)
	if err != nil {
		return timeutil.SimpleTimeline{}, err
	}

	times := []time.Time{}

	// Let's handle the special case when the period starts before the first recurrence
	if start := u.GetOriginalValueAsUsagePeriodInput().GetTime(); !inPeriod.From.After(start) {
		times = append(times, start)
	}

	at := inPeriod.From

	for i := firstPerIdx; i <= len(u.recs.GetTimes())-1; i++ {
		rec := u.recs.GetAt(i)

		// We're surely outside the period
		if rec.GetTime().After(inPeriod.To) {
			break
		}

		// We need to generate all the programmatic reset times for the current recurrence
		limit := inPeriod.To

		if i < len(u.recs.GetTimes())-1 {
			next := u.recs.GetAt(i + 1)
			if next.GetTime().Before(limit) {
				limit = next.GetTime()
			}
		}

		for i := 0; i < MAX_SAFE_ITERATIONS; i++ {
			if i == MAX_SAFE_ITERATIONS {
				return timeutil.SimpleTimeline{}, fmt.Errorf("max safe iterations reached: %d", MAX_SAFE_ITERATIONS)
			}

			per, err := u.GetCurrentPeriodAt(at)
			if err != nil {
				return timeutil.SimpleTimeline{}, err
			}

			// To handle first match if at is aligned with reset times
			if i == 0 && per.From.Equal(at) {
				times = append(times, at)
			}

			// If we're at the limit, we're done
			if per.To.After(limit) {
				break
			}

			at = per.To

			// Otherwise we add the period end no matter what
			times = append(times, at)
		}
	}

	// We are gonna be lazy and simply dedupe the results so its not an issue if we added something twice (due to special case handling)
	times = lo.Uniq(times)

	// TODO: implement
	return timeutil.NewSimpleTimeline(times), nil
}

func (u UsagePeriod) GetUsagePeriodInputAt(at time.Time) (UsagePeriodInput, int, error) {
	// we'll iterate through all recurrences in the timed order (newest to oldest)
	// we want ot return the first which is not after the at time (as that will be effective)
	for i := len(u.recs.GetTimes()) - 1; i >= 0; i-- {
		rec := u.recs.GetAt(i)
		if !rec.GetTime().After(at) {
			return timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
				return rec.GetTime()
			})(rec.GetValue()), i, nil
		}
	}
	// if we don't find any we simply return the last (oldest)
	oldest := u.recs.GetAt(0)
	return timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
		return oldest.GetTime()
	})(oldest.GetValue()), 0, nil
}
