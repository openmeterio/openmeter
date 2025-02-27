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

// Calculates the burn phases for the given period.
// The period is expected not to contain any resets.
//
// A new burn phase starts when:
// 1) A grant recurrs
// 2) The burndown order changes
//
// Note that grant balance does not effect the burndown order if we simply ignore grants that don't
// have balance while burning down.
func (e *engine) getPhases(grants []grant.Grant, period timeutil.Period) ([]burnPhase, error) {
	activityChanges := e.getGrantActivityChanges(grants, period)
	recurrenceTimes, err := e.getGrantRecurrenceTimes(grants, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get grant recurrence times: %w", err)
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

	// if both are null then return single phase for entire period
	if len(activityChanges) == 0 && len(recurrenceTimes) == 0 {
		return []burnPhase{{from: period.From, to: period.To}}, nil
	}

	acI, rtI := 0, 0
	phaseFrom := period.From

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
			phases = append(phases, phase)
			phaseFrom = phase.to
		}
	}

	// order here doesn't matter as one or both of them is empty
	// append all activityChanges remaining
	for _, activityChange := range activityChanges[acI:] {
		phases = append(phases, burnPhase{
			from:           phaseFrom,
			to:             activityChange,
			priorityChange: true,
		})
		phaseFrom = activityChange
	}
	// append all recurrenceTimes remaining
	for _, recurrenceTime := range recurrenceTimes[rtI:] {
		phases = append(phases, burnPhase{
			from:                phaseFrom,
			to:                  recurrenceTime.time,
			grantsRecurredAtEnd: recurrenceTime.grantIDs,
		})
		phaseFrom = recurrenceTime.time
	}

	if phaseFrom.Before(period.To) {
		phases = append(phases, burnPhase{
			from: phaseFrom,
			to:   period.To,
		})
	}

	return phases, nil
}
