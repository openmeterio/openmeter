package engine

import (
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
		if grant.EffectiveAt.After(period.From) && (grant.EffectiveAt.Before(period.To)) {
			activityChanges = append(activityChanges, grant.EffectiveAt)
		}
		// grants that expire in the period
		if grant.ExpiresAt != nil {
			if grant.ExpiresAt.After(period.From) && (grant.ExpiresAt.Before(period.To)) {
				activityChanges = append(activityChanges, *grant.ExpiresAt)
			}
		}
		// grants that are deleted in the period
		if grant.DeletedAt != nil {
			if grant.DeletedAt.After(period.From) && (grant.DeletedAt.Before(period.To)) {
				activityChanges = append(activityChanges, *grant.DeletedAt)
			}
		}
		// grants that are voided in the period
		if grant.VoidedAt != nil {
			if grant.VoidedAt.After(period.From) && (grant.VoidedAt.Before(period.To)) {
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
func (e *engine) getGrantRecurrenceTimes(grants []grant.Grant, period timeutil.ClosedPeriod) ([]struct {
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
		// writing all reccurence times until grant is active or period ends
		for it.At.Before(period.To) && grant.ActiveAt(it.At) {
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

// The correct order to burn down grants is:
// 1. Grants with higher priority are burned down first
// 2. Grants with earlier expiration date are burned down first
// 3. For a deterministic order, we sort by our general cursor which is created_at + id
//
// TODO: figure out if this needs to return an error or not
func PrioritizeGrants(grants []grant.Grant) error {
	if len(grants) == 0 {
		// we don't do a thing, return early
		// return fmt.Errorf("no grants to prioritize")
		return nil
	}

	// 3. As a tie breaker, we use our general cursor which is created_at then id
	slices.SortStableFunc(grants, func(i, j grant.Grant) int {
		switch {
		case i.CreatedAt.Before(j.CreatedAt):
			return -1
		case i.CreatedAt.After(j.CreatedAt):
			return 1
		default:
			return strings.Compare(i.ID, j.ID)
		}
	})

	// 2. Grants with earlier expiration date are burned down first
	slices.SortStableFunc(grants, func(i, j grant.Grant) int {
		exp1 := i.GetExpiration()
		exp2 := j.GetExpiration()

		switch {
		// If both have no expiration, we keep the order
		case exp1 == nil && exp2 == nil:
			return 0
		// If the second has an expiration, we put it first
		case exp1 == nil && exp2 != nil:
			return 1
		// If the first has an expiration, we put it first
		case exp2 == nil && exp1 != nil:
			return -1
		}

		// Otherwise we compare the expiration dates
		switch u1, u2 := exp1.Unix(), exp2.Unix(); {
		case u1 < u2:
			return -1
		case u1 > u2:
			return 1
		default:
			return 0
		}
	})

	// 1. Order grant balances by priority
	sort.SliceStable(grants, func(i, j int) bool {
		return grants[i].Priority < grants[j].Priority
	})

	return nil
}
