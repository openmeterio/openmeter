package entitlement

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
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

	// For validating the constraint we sort the entitlements by ActiveFromTime ascending.
	// If any two neighboring entitlements are active at the same time, the constraint is violated.
	// This is equivalent to the above formal definition.
	s := make([]Entitlement, len(ents))
	copy(s, ents)
	slices.SortStableFunc(s, func(a, b Entitlement) int {
		return int(a.ActiveFromTime().Sub(b.ActiveFromTime()).Milliseconds())
	})
	for i := range s {
		if i == 0 {
			continue
		}

		if s[i-1].ActiveToTime() == nil {
			return &UniquenessConstraintError{e1: s[i-1], e2: s[i]}
		}

		if s[i].ActiveFromTime().Before(*s[i-1].ActiveToTime()) {
			return &UniquenessConstraintError{e1: s[i-1], e2: s[i]}
		}
	}

	return nil
}

type UniquenessConstraintError struct {
	e1, e2 Entitlement
}

func (e *UniquenessConstraintError) Error() string {
	return fmt.Sprintf("constraint violated: %v is active at the same time as %v", e.e1.ID, e.e2.ID)
}
