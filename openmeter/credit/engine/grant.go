package engine

import (
	"cmp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// An activity change is a grant becoming active or a grant expiring.
func (e *engine) getGrantActivityChanges(grants []grant.Grant, period timeutil.ClosedPeriod) []time.Time {
	activityChanges := []time.Time{}

	for _, grant := range grants {
		// grants that take effect in the period
		if period.Contains(grant.EffectiveAt) {
			activityChanges = append(activityChanges, grant.EffectiveAt)
		}
		// grants that expire in the period
		if grant.ExpiresAt != nil {
			if period.Contains(*grant.ExpiresAt) {
				activityChanges = append(activityChanges, *grant.ExpiresAt)
			}
		}
		// grants that are deleted in the period
		if grant.DeletedAt != nil {
			if period.Contains(*grant.DeletedAt) {
				activityChanges = append(activityChanges, *grant.DeletedAt)
			}
		}
		// grants that are voided in the period
		if grant.VoidedAt != nil {
			if period.Contains(*grant.VoidedAt) {
				activityChanges = append(activityChanges, *grant.VoidedAt)
			}
		}
	}

	// FIXME: we should truncate on input but that's hard for voidedAt and deletedAt
	// FIXME: remove truncation
	for i, t := range activityChanges {
		activityChanges[i] = t.Truncate(time.Minute).In(time.UTC)
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
func (e *engine) getGrantRecurrenceTimes(grants []grant.Grant, period timeutil.ClosedPeriod, endBoundaryBehavior timeutil.Boundary) ([]struct {
	time     time.Time
	grantIDs []string
}, error,
) {
	times := []struct {
		time    time.Time
		grantID string
	}{}
	grantsWithRecurrence := lo.Filter(grants, func(grant grant.Grant, _ int) bool {
		return grant.Recurrence != nil
	})
	if len(grantsWithRecurrence) == 0 {
		return nil, nil
	}

	for _, grant := range grantsWithRecurrence {
		it, err := grant.Recurrence.IterateFromNextAfter(
			lo.Latest(grant.EffectiveAt, period.From),
			timeutil.Inclusive,
		)
		if err != nil {
			return nil, err
		}

		// Write all recurrence times in [period.From, period.To]).
		// For a zero-length period [T, T], include T so a recurrence at the
		// start of that period can still be applied.
		// Include period.To for the final run period so a run ending exactly on
		// a recurrence produces a zero-length terminal phase where it is applied.
		inPeriod := func(at time.Time) bool {
			behavior := endBoundaryBehavior

			if period.IsEmpty() && at.Equal(period.From) {
				behavior = timeutil.Inclusive
			}

			switch behavior {
			case timeutil.Inclusive:
				return period.ContainsInclusive(at) // [from, to]
			case timeutil.Exclusive:
				return period.Contains(at) // [from, to)
			default:
				return false
			}
		}

		for inPeriod(it.At) && grant.ActiveAt(it.At) {
			times = append(times, struct {
				time    time.Time
				grantID string
			}{time: it.At, grantID: grant.ID})
			it, err = it.Next()
			if err != nil {
				return nil, err
			}
		}
	}

	// map times to UTC
	for i, t := range times {
		times[i].time = t.time.In(time.UTC)
	}

	// sort times ascending
	sort.Slice(times, func(i, j int) bool {
		return times[i].time.Before(times[j].time)
	})

	// dedupe times by time
	deduped := []struct {
		time     time.Time
		grantIDs []string
	}{}
	for _, t := range times {
		// if the last deduped time is not the same as the current time, add a new deduped time
		if len(deduped) == 0 || !deduped[len(deduped)-1].time.Equal(t.time) {
			deduped = append(deduped, struct {
				time     time.Time
				grantIDs []string
			}{time: t.time, grantIDs: []string{t.grantID}})
			// if the last deduped time is the same as the current time, add the grantID to the last deduped time
		} else {
			deduped[len(deduped)-1].grantIDs = append(deduped[len(deduped)-1].grantIDs, t.grantID)
		}
	}
	return deduped, nil
}

// A grant is relevant if its active at any point during the period, both limits inclusive
// A grant is also relevant if it is mentioned in the balance map
func (e *engine) filterRelevantGrants(grants []grant.Grant, bm balance.Map, period timeutil.ClosedPeriod) []grant.Grant {
	relevant := []grant.Grant{}
	for _, grant := range grants {
		if grant.GetEffectivePeriod().Open().OverlapsInclusive(period.Open()) {
			relevant = append(relevant, grant)
		} else if _, ok := bm[grant.ID]; ok {
			relevant = append(relevant, grant)
		}
	}

	return relevant
}

// PrioritizeGrants orders grants in place by burn-down precedence.
// Lower priority numbers, earlier expirations, and earlier creation cursors burn down first.
func PrioritizeGrants(grants []grant.Grant) {
	slices.SortStableFunc(grants, compareGrantsByBurnDownOrder)
}

func compareGrantsByBurnDownOrder(i, j grant.Grant) int {
	if priorityOrder := cmp.Compare(i.Priority, j.Priority); priorityOrder != 0 {
		return priorityOrder
	}

	iExpiration := i.GetExpiration()
	jExpiration := j.GetExpiration()

	if iExpiration == nil && jExpiration != nil {
		return 1
	}

	if iExpiration != nil && jExpiration == nil {
		return -1
	}

	if iExpiration != nil && jExpiration != nil {
		if expirationOrder := cmp.Compare(iExpiration.Unix(), jExpiration.Unix()); expirationOrder != 0 {
			return expirationOrder
		}
	}

	if creationOrder := i.CreatedAt.Compare(j.CreatedAt); creationOrder != 0 {
		return creationOrder
	}

	return strings.Compare(i.ID, j.ID)
}
