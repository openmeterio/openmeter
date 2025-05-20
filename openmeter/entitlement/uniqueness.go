package entitlement

import (
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateUniqueConstraint validates the uniqueness constraints of the entitlements
// The constraint is formally stated as follows:
//
// For entitlement E
// 1. The ActiveFromTime is E.ActiveFrom or E.CreatedAt (if E.ActiveFrom is nil)
// 2. The ActiveToTime is E.ActiveTo or E.DeletedAt (if E.ActiveTo is nil). This can be nil.
//
// Entitlement E is active at time T if and only if:
// 1. E.ActiveFromTime <= T and E.ActiveToTime > T
// 2. E.ActiveFromTime <= T and E.ActiveToTime is nil
//
// For a set of unique entitlements S, where all E in S share the same feature (by key) and subject:
// 1. Let T1 be the first ActiveFromTime for any E in S sorted ascending
// 2. Let T2 be the last ActiveToTime for any E in S sorted ascending
//
// The constraint:
//
// For all E in S at any time T where T1 <= T < T2, there is at most one E that is active.
func ValidateUniqueConstraint(ents []Entitlement) error {
	// Validate all entitlements belong to same feature and subject
	if grouped := lo.GroupBy(ents, func(e Entitlement) string { return e.FeatureKey }); len(grouped) > 1 {
		keys := lo.Keys(grouped)
		slices.Sort(keys)
		return fmt.Errorf("entitlements must belong to the same feature, found %v", keys)
	}
	if grouped := lo.GroupBy(ents, func(e Entitlement) string { return e.SubjectKey }); len(grouped) > 1 {
		keys := lo.Keys(grouped)
		slices.Sort(keys)
		return fmt.Errorf("entitlements must belong to the same subject, found %v", keys)
	}

	type cadencedEnt struct {
		models.CadencedModel
		ent Entitlement
	}

	// We use models.CadenceList to validate the uniqueness constraint.
	timeline := models.NewSortedCadenceList(
		// As entitlements where e.ActiveFromTime() == e.ActiveToTime() can never be active, we should ignore them.
		lo.Map(
			lo.Filter(ents, func(e Entitlement, _ int) bool {
				if e.ActiveToTime() != nil && e.ActiveFromTime().Equal(*e.ActiveToTime()) {
					return false
				}

				return true
			}),
			func(e Entitlement, _ int) cadencedEnt {
				return cadencedEnt{
					CadencedModel: models.CadencedModel{
						ActiveFrom: e.ActiveFromTime(),
						ActiveTo:   e.ActiveToTime(),
					},
					ent: e,
				}
			}))

	if overlaps := timeline.GetOverlaps(); len(overlaps) > 0 {
		// We only return the first overlap
		items := timeline.Cadences()
		return &UniquenessConstraintError{e1: items[overlaps[0].Index1].ent, e2: items[overlaps[0].Index2].ent}
	}

	return nil
}

type UniquenessConstraintError struct {
	e1, e2 Entitlement
}

func (e *UniquenessConstraintError) Error() string {
	return fmt.Sprintf("constraint violated: %v is active at the same time as %v", e.e1.ID, e.e2.ID)
}
