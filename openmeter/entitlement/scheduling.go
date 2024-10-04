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
