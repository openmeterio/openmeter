package engine

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type burnPhase struct {
	from time.Time
	to   time.Time
	// The ID of the grant that recurred marking the end of this phase (if any)
	grantsRecurredAtEnd []string
	// If priority order changes at the end of this phase
	priorityChange bool
}

type phasePlan struct {
	phases                []burnPhase
	grantsRecurredAtStart []string
}

// Calculates the burn phases for the given period.
// The period is expected not to contain any resets.
//
// A new burn phase starts when:
// 1) A grant recurrs
// 2) The burndown order changes
//
// Note that grant balance does not effect the burndown order if we simply ignore grants that don't
// have balance while burning down.
func (e *engine) getPhases(grants []grant.Grant, period timeutil.ClosedPeriod) (phasePlan, error) {
	activityChanges := e.getGrantActivityChanges(grants, period)
	recurrenceTimes, err := e.getGrantRecurrenceTimes(grants, period)
	if err != nil {
		return phasePlan{}, fmt.Errorf("failed to get grant recurrence times: %w", err)
	}
	phases := []burnPhase{}

	// set empty arrays as default values so we don't have nils
	if len(activityChanges) == 0 {
		activityChanges = []time.Time{}
	}
	if len(recurrenceTimes) == 0 {
		recurrenceTimes = []struct {
			time     time.Time
			grantIDs []string
		}{}
	}

	grantsRecurredAtStart := []string{}
	for len(recurrenceTimes) > 0 && recurrenceTimes[0].time.Equal(period.From) {
		grantsRecurredAtStart = append(grantsRecurredAtStart, recurrenceTimes[0].grantIDs...)
		recurrenceTimes = recurrenceTimes[1:]
	}

	// if both are null then return single phase for entire period
	if len(activityChanges) == 0 && len(recurrenceTimes) == 0 {
		return phasePlan{
			phases:                []burnPhase{{from: period.From, to: period.To}},
			grantsRecurredAtStart: grantsRecurredAtStart,
		}, nil
	}

	acI, rtI := 0, 0
	phaseFrom := period.From

	appendPhase := func(phase burnPhase) {
		phases = append(phases, phase)
		phaseFrom = phase.to
	}

	var phase burnPhase
	for len(activityChanges) > acI && len(recurrenceTimes) > rtI {
		// compare the first activity change and the first recurrence time
		// - if they're the same we create a single period and increment both
		// - if not we increment the earlier
		if activityChanges[acI].Before(recurrenceTimes[rtI].time) {
			phase = burnPhase{
				from:           phaseFrom,
				to:             activityChanges[acI],
				priorityChange: true,
			}
			acI++
		} else if activityChanges[acI].After(recurrenceTimes[rtI].time) {
			phase = burnPhase{
				from:                phaseFrom,
				to:                  recurrenceTimes[rtI].time,
				grantsRecurredAtEnd: recurrenceTimes[rtI].grantIDs,
			}
			rtI++
		} else {
			phase = burnPhase{
				from:                phaseFrom,
				to:                  activityChanges[acI],
				priorityChange:      true,
				grantsRecurredAtEnd: recurrenceTimes[rtI].grantIDs,
			}
			acI++
			rtI++
		}

		// If it's a valid phase (non-zero duration), we save it and break
		if phase.to.After(phase.from) {
			appendPhase(phase)
		}
	}

	// order here doesn't matter as one or both of them is empty
	// append all activityChanges remaining
	for _, activityChange := range activityChanges[acI:] {
		appendPhase(burnPhase{
			from:           phaseFrom,
			to:             activityChange,
			priorityChange: true,
		})
	}
	// append all recurrenceTimes remaining
	for _, recurrenceTime := range recurrenceTimes[rtI:] {
		appendPhase(burnPhase{
			from:                phaseFrom,
			to:                  recurrenceTime.time,
			grantsRecurredAtEnd: recurrenceTime.grantIDs,
		})
	}

	if phaseFrom.Before(period.To) {
		appendPhase(burnPhase{
			from: phaseFrom,
			to:   period.To,
		})
	}

	return phasePlan{
		phases:                phases,
		grantsRecurredAtStart: grantsRecurredAtStart,
	}, nil
}
