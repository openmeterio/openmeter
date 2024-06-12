package credit

import (
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type Engine interface {
	Run(grants []Grant, startingBalances GrantBalanceMap, startingOverage float64, period Period) (endingBalances GrantBalanceMap, endingOverage float64, history []GrantBurnDownHistorySegment, err error)
}

func NewEngine(getFeatureUsage func(from, to time.Time) (float64, error)) Engine {
	return &engine{
		getFeatureUsage: getFeatureUsage,
	}
}

// engine burns down grants based on usage following the rules of Grant BurnDown.
type engine struct {
	// An engine can only be run once
	hasRun bool
	// List of all grants that are active at the relevant period at some point.
	grants []Grant
	// Map of all grants that are active at the relevant period at some point.
	grantsMap map[GrantID]Grant
	// Returns the total feature usage in the queried period
	getFeatureUsage func(from, to time.Time) (float64, error)
	// granularity     models.WindowSize // TODO: implement

	// Whether the engine was able to execute all calculations exactly
	calcsExact bool // TODO: add public API and checking
}

// Ensure engine implements Engine
var _ Engine = (*engine)(nil)

func (e *engine) setup(grants []Grant) {
	e.grants = grants
	e.grantsMap = make(map[GrantID]Grant)
	for _, grant := range grants {
		e.grantsMap[grant.ID] = grant
	}
	e.hasRun = true
}

// Burns down all grants in the defined period by the usage amounts.
func (e *engine) Run(grants []Grant, startingBalances GrantBalanceMap, overage float64, period Period) (GrantBalanceMap, float64, []GrantBurnDownHistorySegment, error) {
	if e.hasRun {
		return nil, 0, nil, fmt.Errorf("engine has already run")
	} else {
		e.setup(grants)
	}
	phases, err := e.GetPhases(period)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get burn phases: %w", err)
	}

	err = prioritizeGrants(e.grants)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to prioritize grants: %w", err)
	}

	balancesAtPhaseStart := startingBalances.Copy()

	rePrioritize := false
	recurredGrants := []GrantID{}

	grantMap := make(map[GrantID]Grant)
	for _, grant := range e.grants {
		grantMap[grant.ID] = grant
	}

	segments := make([]GrantBurnDownHistorySegment, 0, len(phases))

	for _, phase := range phases {
		segment := GrantBurnDownHistorySegment{
			Period:         Period{From: phase.from, To: phase.to},
			BalanceAtStart: balancesAtPhaseStart.Copy(),
			TerminationReasons: SegmentTerminationReason{
				PriorityChange: phase.priorityChange,
				Recurrence:     phase.grantsRecurredAtEnd,
			},
		}

		// reprioritize grants if needed
		if rePrioritize {
			err = prioritizeGrants(e.grants)
			if err != nil {
				return nil, 0, nil, fmt.Errorf("failed to prioritize grants: %w", err)
			}
			rePrioritize = false
		}

		// reset recurring grant balances to full amount
		if len(recurredGrants) > 0 {
			// TODO: its not super neat, maybe have a separate entity with balance...
			for _, grantID := range recurredGrants {
				grant := grantMap[grantID]
				grant, ok := grantMap[grantID]
				if !ok {
					return nil, 0, nil, fmt.Errorf("failed to get grant with id %s", grantID)
				}
				// TODO: handle grant recurrence rollover settings!
				balancesAtPhaseStart.Set(grant.ID, grant.Amount)
			}
		}

		// get active and inactive grants in the phase
		activeGrants := make([]Grant, 0, len(e.grants))
		for _, grant := range e.grants {
			if grant.ActiveAt(phase.from) {
				activeGrants = append(activeGrants, grant)
			} else {
				// grants inactivating have 0 balance
				balancesAtPhaseStart[grant.ID] = 0
			}
		}

		// if a grant becomes active at the start of this period then their balance becomes the full amount
		for _, grant := range activeGrants {
			if grant.EffectiveAt.Equal(phase.from) {
				balancesAtPhaseStart[grant.ID] = grant.Amount
			}
		}

		// query feature usage in the burning phase
		usage, err := e.getFeatureUsage(phase.from, phase.to)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to get feature usage for period %s - %s: %w", period.From, period.To, err)
		}
		balancesAtPhaseStart, segment.GrantUsages, overage, err = e.BurnDownGrants(balancesAtPhaseStart, activeGrants, usage+overage)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to burn down grants in period %s - %s: %w", period.From, period.To, err)
		}

		segment.TotalUsage = usage
		segment.Overage = overage

		segments = append(segments, segment)

		// check if priority changed or grants need to recurr
		if phase.priorityChange {
			rePrioritize = true
		}
		if len(phase.grantsRecurredAtEnd) > 0 {
			recurredGrants = phase.grantsRecurredAtEnd
		}
	}
	return balancesAtPhaseStart, overage, segments, nil
}

// Burns down the grants of the priority sorted list. Manages overage.
// All calculations are done during this function.
//
// FIXME: calcualtions happen on inexact representations as float64, this can lead to rounding errors.
func (e *engine) BurnDownGrants(startingBalances GrantBalanceMap, prioritized []Grant, usage float64) (GrantBalanceMap, []GrantUsage, float64, error) {
	balances := startingBalances.Copy()
	uses := make([]GrantUsage, 0, len(prioritized))
	exactUsage := alpacadecimal.NewFromFloat(usage)

	getFloat := func(d alpacadecimal.Decimal) float64 {
		f, exact := d.Float64()
		if !exact {
			e.calcsExact = false
		}
		return f
	}

	for _, grant := range prioritized {
		grantBalance := balances[grant.ID]
		// if grant has no balance, skip
		if grantBalance == 0 {
			continue
		}
		exactBalance := alpacadecimal.NewFromFloat(grantBalance)
		// if grant balance is less than usage, burn the grant and subtract the balance from usage
		if exactBalance.LessThanOrEqual(exactUsage) {
			balances.Set(grant.ID, 0) // 0 usage to avoid arithmetic errors
			exactUsage = exactUsage.Sub(exactBalance)
			uses = append(uses, GrantUsage{
				GrantID:           grant.ID,
				Usage:             grantBalance,
				TerminationReason: GrantUsageTerminationReasonExhausted,
			})
			// if grant balance is more than usage, burn the grant with the usage
		} else {
			balances.Burn(grant.ID, getFloat(exactUsage))
			uses = append(uses, GrantUsage{
				GrantID:           grant.ID,
				Usage:             getFloat(exactUsage),
				TerminationReason: GrantUsageTerminationReasonSegmentTermination,
			})
			exactUsage = alpacadecimal.NewFromFloat(0)
		}
	}

	return balances, uses, getFloat(exactUsage), nil
}

// Calculates the burn phases for the given period.
//
// A new burn phase starts when:
// 1) A grant recurrs
// 2) The burndown order changes
//
// Note that grant balance does not effect the burndown order if we simply ignore grants that don't
// have balance while burning down.
//
// TODO: rounding?
func (e *engine) GetPhases(period Period) ([]burnPhase, error) {
	activityChanges := e.getGrantActivityChanges(period)
	recurrenceTimes, err := e.getGrantRecurrenceTimes(period)
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
			grantIDs []GrantID
		}{}
	}

	// if both are null then return single phase for entire period
	if len(activityChanges) == 0 && len(recurrenceTimes) == 0 {
		return []burnPhase{{from: period.From, to: period.To}}, nil
	}

	acI, rtI := 0, 0
	phaseFrom := period.From

	for len(activityChanges) > acI && len(recurrenceTimes) > rtI {
		// compare the first activity change and the first recurrence time
		// - if they're the same we create a single period and increment both
		// - if not we increment the earlier
		if activityChanges[acI].Before(recurrenceTimes[rtI].time) {
			phases = append(phases, burnPhase{
				from:           phaseFrom,
				to:             activityChanges[acI],
				priorityChange: true,
			})
			phaseFrom = activityChanges[acI]
			acI++
		} else if activityChanges[acI].After(recurrenceTimes[rtI].time) {
			phases = append(phases, burnPhase{
				from:                phaseFrom,
				to:                  recurrenceTimes[rtI].time,
				grantsRecurredAtEnd: recurrenceTimes[rtI].grantIDs,
			})
			phaseFrom = recurrenceTimes[rtI].time
			rtI++
		} else {
			phases = append(phases, burnPhase{
				from:                phaseFrom,
				to:                  activityChanges[acI],
				priorityChange:      true,
				grantsRecurredAtEnd: recurrenceTimes[rtI].grantIDs,
			})
			phaseFrom = activityChanges[acI]
			acI++
			rtI++

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

// An activity change is a grant becoming active or a grant expiring.
func (e *engine) getGrantActivityChanges(period Period) []time.Time {
	activityChanges := []time.Time{}
	for _, grant := range e.grants {
		// grants that take effect in the period
		if grant.EffectiveAt.After(period.From) && (grant.EffectiveAt.Before(period.To)) {
			activityChanges = append(activityChanges, grant.EffectiveAt)
		}
		// grants that expire in the period
		if grant.ExpiresAt.After(period.From) && (grant.ExpiresAt.Before(period.To)) {
			activityChanges = append(activityChanges, grant.ExpiresAt)
		}
	}

	sort.Slice(activityChanges, func(i, j int) bool {
		return activityChanges[i].Before(activityChanges[j])
	})

	deduped := []time.Time{}
	for _, t := range activityChanges {
		if len(deduped) == 0 || !deduped[len(deduped)-1].Equal(t) {
			deduped = append(deduped, t)
		}
	}

	return deduped
}

// Get all times grants recurr in the period.
func (e *engine) getGrantRecurrenceTimes(period Period) ([]struct {
	time     time.Time
	grantIDs []GrantID
}, error) {
	times := []struct {
		time    time.Time
		grantID GrantID
	}{}
	grantsWithRecurrence := slicesx.Filter(e.grants, func(grant Grant) bool {
		return grant.Recurrence != nil
	})
	if len(grantsWithRecurrence) == 0 {
		return nil, nil
	}

	for _, grant := range grantsWithRecurrence {
		i, err := grant.Recurrence.NextAfter(later(grant.EffectiveAt, period.From))
		if err != nil {
			return nil, err
		}
		// writing all reccurence times until grant is active or period ends
		for i.Before(period.To) && grant.ActiveAt(i) {
			times = append(times, struct {
				time    time.Time
				grantID GrantID
			}{time: i, grantID: grant.ID})
			i, err = grant.Recurrence.Next(i)
			if err != nil {
				return nil, err
			}
		}
	}

	// sort times ascending
	sort.Slice(times, func(i, j int) bool {
		return times[i].time.Before(times[j].time)
	})

	// dedupe times by time
	deduped := []struct {
		time     time.Time
		grantIDs []GrantID
	}{}
	for _, t := range times {
		// if the last deduped time is not the same as the current time, add a new deduped time
		if len(deduped) == 0 || !deduped[len(deduped)-1].time.Equal(t.time) {
			deduped = append(deduped, struct {
				time     time.Time
				grantIDs []GrantID
			}{time: t.time, grantIDs: []GrantID{t.grantID}})
			// if the last deduped time is the same as the current time, add the grantID to the last deduped time
		} else {
			deduped[len(deduped)-1].grantIDs = append(deduped[len(deduped)-1].grantIDs, t.grantID)
		}
	}
	return deduped, nil
}

type burnPhase struct {
	from time.Time
	to   time.Time
	// The ID of the grant that recurred marking the end of this phase (if any)
	grantsRecurredAtEnd []GrantID
	// If priority order changes at the end of this phase
	priorityChange bool
}

// The correct order to burn down grants is:
// 1. Grants with higher priority are burned down first
// 2. Grants with earlier expiration date are burned down first
func prioritizeGrants(grants []Grant) error {
	if len(grants) == 0 {
		return fmt.Errorf("no grants to prioritize")
	}

	// 2. Grants with earlier expiration date are burned down first
	sort.Slice(grants, func(i, j int) bool {
		return grants[i].GetExpiration().Unix() < grants[j].GetExpiration().Unix()
	})

	// 1. Order grant balances by priority
	sort.Slice(grants, func(i, j int) bool {
		return grants[i].Priority < grants[j].Priority
	})

	return nil
}

func later(t1 time.Time, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
