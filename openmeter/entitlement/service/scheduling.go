package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ScheduleEntitlement schedules an entitlement for a future date.
func (c *service) ScheduleEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) {
		activeFromTime := defaultx.WithDefault(input.ActiveFrom, clock.Now())

		if err := input.Validate(); err != nil {
			return nil, models.NewGenericValidationError(err)
		}

		customer, err := c.customerService.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.UsageAttribution.ID,
			},
		})
		if err != nil {
			return nil, err
		}

		// ID has priority over key
		featureIdOrKey := input.FeatureID
		if featureIdOrKey == nil {
			featureIdOrKey = input.FeatureKey
		}
		if featureIdOrKey == nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("feature ID or Key is required"))
		}

		feat, err := c.featureConnector.GetFeature(ctx, input.Namespace, *featureIdOrKey, feature.IncludeArchivedFeatureFalse)
		if err != nil || feat == nil {
			return nil, &feature.FeatureNotFoundError{ID: *featureIdOrKey}
		}

		err = c.lockUniqueScope(ctx, input.UsageAttribution.ID, feat.Key)
		if err != nil {
			return nil, err
		}

		// fill featureId and featureKey
		input.FeatureID = &feat.ID
		input.FeatureKey = &feat.Key

		// We set ActiveFrom so it's deterministic from this point on, even if there's a delay until entitlement gets persisted and assigned a CreatedAt
		input.ActiveFrom = &activeFromTime

		// Get scheduled entitlements for customer-feature pair
		scheduledEnts, err := c.entitlementRepo.GetScheduledEntitlements(ctx, input.Namespace, input.UsageAttribution.ID, feat.Key, activeFromTime)
		if err != nil {
			return nil, err
		}

		// Sort scheduled entitlements by activeFromTime ascending
		slices.SortStableFunc(scheduledEnts, func(a, b entitlement.Entitlement) int {
			return int(a.ActiveFromTime().Sub(b.ActiveFromTime()))
		})

		// We need a dummy representation of the entitlement to validate the uniqueness constraint.
		// This is the least sufficient representation on which the constraint check can be performed.
		newEntitlementId := "new-entitlement-id"

		dummy := entitlement.Entitlement{
			GenericProperties: entitlement.GenericProperties{
				ID:         newEntitlementId,
				FeatureKey: *input.FeatureKey,
				CustomerID: input.UsageAttribution.ID,
				ManagedModel: models.ManagedModel{
					CreatedAt: activeFromTime,
				},
				ActiveFrom:  input.ActiveFrom,
				ActiveTo:    input.ActiveTo,
				Annotations: input.Annotations,
			},
		}

		err = entitlement.ValidateUniqueConstraint(append(scheduledEnts, dummy))
		if err != nil {
			if cErr, ok := lo.ErrorsAs[*entitlement.UniquenessConstraintError](err); ok {
				if cErr.E1.ID != newEntitlementId && cErr.E2.ID != newEntitlementId {
					// inconsistency error
					return nil, fmt.Errorf("inconsistency error: scheduled entitlements don't meet uniqueness constraint %w", cErr)
				}
				conflict := cErr.E1
				if conflict.ID == newEntitlementId {
					conflict = cErr.E2
				}

				return nil, &entitlement.AlreadyExistsError{EntitlementID: conflict.ID, FeatureID: conflict.FeatureID, CustomerID: conflict.CustomerID}
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

		err = c.publisher.Publish(ctx, entitlement.NewEntitlementCreatedEventPayloadV2(*ent, customer))
		if err != nil {
			return nil, err
		}

		return ent, err
	})
}

func (c *service) SupersedeEntitlement(ctx context.Context, entitlementId string, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) {
		// Find the entitlement to override
		oldEnt, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: input.Namespace, ID: entitlementId})
		if err != nil {
			return nil, err
		}

		if oldEnt == nil {
			return nil, fmt.Errorf("inconsistency error, entitlement is nil: %s", entitlementId)
		}

		if oldEnt.DeletedAt != nil {
			return nil, &entitlement.AlreadyDeletedError{EntitlementID: oldEnt.ID}
		}

		if err := c.hooks.PreDelete(ctx, oldEnt); err != nil {
			return nil, err
		}

		// ID has priority over key
		featureIdOrKey := input.FeatureID
		if featureIdOrKey == nil {
			featureIdOrKey = input.FeatureKey
		}
		if featureIdOrKey == nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("feature ID or Key is required"))
		}

		feat, err := c.featureConnector.GetFeature(ctx, input.Namespace, *featureIdOrKey, feature.IncludeArchivedFeatureFalse)
		if err != nil {
			return nil, err
		}

		if feat == nil {
			return nil, fmt.Errorf("inconsistency error, feature is nil: %s", *featureIdOrKey)
		}

		err = c.lockUniqueScope(ctx, input.UsageAttribution.ID, feat.Key)
		if err != nil {
			return nil, err
		}

		// Validate that old a new entitlement belong to same feature & customer

		if feat.Key != oldEnt.FeatureKey {
			return nil, models.NewGenericValidationError(fmt.Errorf("old and new entitlements belong to different features"))
		}

		if input.UsageAttribution.ID != oldEnt.CustomerID {
			return nil, models.NewGenericValidationError(fmt.Errorf("old and new entitlements belong to different customers"))
		}

		// To override we close the old entitlement as inactive and create the new one
		activationTime := defaultx.WithDefault(input.ActiveFrom, clock.Now())

		if !activationTime.After(oldEnt.ActiveFromTime()) {
			return nil, models.NewGenericValidationError(fmt.Errorf("new entitlement must be active after the old one"))
		}

		// To avoid unintended consequences, we don't allow overriding an entitlement with another one which wouldn't otherwise be overlapping
		// Otherwise create ScheduleEntitlement would return an InconsistencyError which is hard to make sense of
		if oldEnt.ActiveToTime() != nil && oldEnt.ActiveToTime().Before(activationTime) {
			return nil, models.NewGenericValidationError(fmt.Errorf("new entitlement must be active before the old one ends"))
		}

		// Do the override
		err = c.entitlementRepo.DeactivateEntitlement(ctx, models.NamespacedID{Namespace: input.Namespace, ID: oldEnt.ID}, activationTime)
		if err != nil {
			return nil, err
		}

		// Create new entitlement
		//
		// The Unique Constraint during Scheduling catches the InconsistencyError where the new entitltment would be scheduled active longer then any later entitlement would start.
		return c.ScheduleEntitlement(ctx, input)
	})
}

func (c *service) lockUniqueScope(ctx context.Context, customerID string, featureKey string) error {
	key, err := NewEntitlementUniqueScopeLock(featureKey, customerID)
	if err != nil {
		return err
	}

	return c.locker.LockForTX(ctx, key)
}
