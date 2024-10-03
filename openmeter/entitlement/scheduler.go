package entitlement

import (
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"

	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ScheduleEntitlement schedules an entitlement for a future date.
func (c *entitlementConnector) ScheduleEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error) {
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*Entitlement, error) {
		activeFromTime := defaultx.WithDefault(input.ActiveFrom, clock.Now())

		if input.ActiveTo != nil && input.ActiveFrom == nil {
			return nil, &models.GenericUserError{Message: "ActiveFrom must be set if ActiveTo is set"}
		}
		if input.ActiveTo != nil && !input.ActiveTo.After(activeFromTime) {
			return nil, &models.GenericUserError{Message: "ActiveTo must be after ActiveFrom"}
		}

		// ID has priority over key
		featureIdOrKey := input.FeatureID
		if featureIdOrKey == nil {
			featureIdOrKey = input.FeatureKey
		}
		if featureIdOrKey == nil {
			return nil, &models.GenericUserError{Message: "Feature ID or Key is required"}
		}

		feat, err := c.featureConnector.GetFeature(ctx, input.Namespace, *featureIdOrKey, feature.IncludeArchivedFeatureFalse)
		if err != nil || feat == nil {
			return nil, &feature.FeatureNotFoundError{ID: *featureIdOrKey}
		}

		// fill featureId and featureKey
		input.FeatureID = &feat.ID
		input.FeatureKey = &feat.Key

		// We set ActiveFrom so it's deterministic from this point on, even if there's a delay until entitlement gets persisted and assigned a CreatedAt
		input.ActiveFrom = &activeFromTime

		// Get scheduled entitlements for subject-feature pair
		scheduledEnts, err := c.entitlementRepo.GetScheduledEntitlements(ctx, input.Namespace, models.SubjectKey(input.SubjectKey), feat.Key, activeFromTime)
		if err != nil {
			return nil, err
		}

		// Sort scheduled entitlements by activeFromTime ascending
		slices.SortStableFunc(scheduledEnts, func(a, b Entitlement) int {
			return int(a.ActiveFromTime().Sub(b.ActiveFromTime()))
		})

		// We need a dummy representation of the entitlement to validate the uniqueness constraint.
		// This is the least sufficient representation on which the constraint check can be performed.
		newEntitlementId := "new-entitlement-id"

		dummy := Entitlement{
			GenericProperties: GenericProperties{
				ID:         newEntitlementId,
				FeatureKey: *input.FeatureKey,
				SubjectKey: input.SubjectKey,
				ManagedModel: models.ManagedModel{
					CreatedAt: activeFromTime,
				},
				ActiveFrom: input.ActiveFrom,
				ActiveTo:   input.ActiveTo,
			},
		}

		err = ValidateUniqueConstraint(append(scheduledEnts, dummy))
		if err != nil {
			if cErr, ok := lo.ErrorsAs[*UniquenessConstraintError](err); ok {
				if cErr.e1.ID != newEntitlementId && cErr.e2.ID != newEntitlementId {
					// inconsistency error
					return nil, fmt.Errorf("inconsistency error: scheduled entitlements don't meet uniqueness constraint %w", cErr)
				}
				conflict := cErr.e1
				if conflict.ID == newEntitlementId {
					conflict = cErr.e2
				}
				return nil, &AlreadyExistsError{EntitlementID: conflict.ID, FeatureID: conflict.FeatureID, SubjectKey: conflict.SubjectKey}
			} else {
				return nil, err
			}
		}

		connector, err := c.getTypeConnector(input)
		if err != nil {
			return nil, err
		}
		repoInputs, err := connector.BeforeCreate(input, *feat)
		if err != nil {
			return nil, err
		}

		ent, err := c.entitlementRepo.CreateEntitlement(ctx, *repoInputs)
		if err != nil {
			return nil, err
		}

		err = connector.AfterCreate(ctx, ent)
		if err != nil {
			return nil, err
		}

		err = c.publisher.Publish(ctx, EntitlementCreatedEvent{
			Entitlement: *ent,
			Namespace: eventmodels.NamespaceID{
				ID: input.Namespace,
			},
		})
		if err != nil {
			return nil, err
		}

		return ent, err
	})
}

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
