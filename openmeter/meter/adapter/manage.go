package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateMeter creates a new meter.
func (a manageAdapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	if err := input.Validate(); err != nil {
		return meterpkg.Meter{}, models.NewGenericValidationError(err)
	}

	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *manageAdapter) (meterpkg.Meter, error) {
			entity, err := repo.db.Meter.Create().
				SetNamespace(input.Namespace).
				SetKey(input.Key).
				SetName(input.Name).
				SetNillableDescription(input.Description).
				SetAggregation(input.Aggregation).
				SetEventType(input.EventType).
				SetNillableValueProperty(input.ValueProperty).
				SetGroupBy(input.GroupBy).
				Save(ctx)
			if err != nil {
				if db.IsConstraintError(err) {
					return meterpkg.Meter{}, models.NewGenericConflictError(fmt.Errorf("meter with the same slug already exists"))
				}

				return meterpkg.Meter{}, fmt.Errorf("failed to create meter: %w", err)
			}

			meter, err := MapFromEntityFactory(entity)
			if err != nil {
				return meterpkg.Meter{}, fmt.Errorf("failed to map meter: %w", err)
			}

			// TODO: remove this once we are sure that the namespace is created at signup
			err = a.namespaceManager.CreateNamespace(ctx, input.Namespace)
			if err != nil {
				return meter, fmt.Errorf("failed to create namespace: %w", err)
			}

			// Create the meter in the streaming connector
			err = a.streamingConnector.CreateMeter(ctx, input.Namespace, meter)
			if err != nil {
				return meter, fmt.Errorf("failed to create meter in streaming connector: %w", err)
			}

			return meter, nil
		})
}

// DeleteMeter deletes a meter.
func (a manageAdapter) DeleteMeter(ctx context.Context, input meterpkg.DeleteMeterInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	// Get the meter
	meter, err := a.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput(input))
	if err != nil {
		return err
	}

	// Check if the meter is already deleted
	if meter.DeletedAt != nil {
		return meterpkg.NewMeterNotFoundError(meter.Key)
	}

	// Check if the meter has active features
	hasFeatures, err := a.featureRepository.HasActiveFeatureForMeter(ctx, input.Namespace, meter.Key)
	if err != nil {
		return fmt.Errorf("failed to check if meter has features: %w", err)
	}

	if hasFeatures {
		return models.NewGenericConflictError(
			fmt.Errorf("meter has active features and cannot be deleted"),
		)
	}

	// Check if the meter has active entitlements
	hasEntitlements, err := a.entitlementRepository.HasEntitlementForMeter(ctx, meter.Namespace, meter.Key)
	if err != nil {
		return fmt.Errorf("failed to check if meter has entitlements: %w", err)
	}

	if hasEntitlements {
		return models.NewGenericConflictError(
			fmt.Errorf("meter has active entitlements and cannot be deleted"),
		)
	}

	// Delete the meter
	return entutils.TransactingRepoWithNoValue(
		ctx,
		a,
		func(ctx context.Context, repo *manageAdapter) error {
			_, err := repo.db.Meter.UpdateOneID(meter.ID).
				SetDeletedAt(time.Now()).
				Save(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return meterpkg.NewMeterNotFoundError(meter.Key)
				}

				if db.IsConstraintError(err) {
					return models.NewGenericConflictError(fmt.Errorf("delete first related resources like reports"))
				}

				return fmt.Errorf("failed to delete meter: %w", err)
			}

			// Delete the meter in the streaming connector
			err = a.streamingConnector.DeleteMeter(ctx, input.Namespace, meter)
			if err != nil {
				return fmt.Errorf("failed to delete meter in streaming connector: %w", err)
			}

			return nil
		})
}
