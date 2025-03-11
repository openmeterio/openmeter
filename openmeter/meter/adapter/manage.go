package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meterdb "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// RegisterPreUpdateMeterHook registers a hook to be called before updating a meter.
func (a *manageAdapter) RegisterPreUpdateMeterHook(hook meterpkg.PreUpdateMeterHook) error {
	a.preUpdateHooks = append(a.preUpdateHooks, hook)
	return nil
}

// CreateMeter creates a new meter.
func (a *manageAdapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
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
					SetNillableEventFrom(input.EventFrom).
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
	})
}

// UpdateMeter updates a new meter.
func (a *manageAdapter) UpdateMeter(ctx context.Context, input meterpkg.UpdateMeterInput) (meterpkg.Meter, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
		// Get the meter by ID
		currentMeter, err := a.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
			Namespace: input.ID.Namespace,
			IDOrSlug:  input.ID.ID,
		})
		if err != nil {
			return meterpkg.Meter{}, err
		}

		if err := input.Validate(currentMeter.ValueProperty); err != nil {
			return meterpkg.Meter{}, models.NewGenericValidationError(err)
		}

		// Collect group by changes
		var groupByToDelete []string

		for key := range currentMeter.GroupBy {
			if _, ok := input.GroupBy[key]; !ok {
				groupByToDelete = append(groupByToDelete, key)
			}
		}

		// FIXME: use foreign keys after we migrate Feature reference on meter id
		// Check if features are compatible with the new group by values
		// We only need to check deleted group bys because only those can be incompatible
		if len(groupByToDelete) > 0 {
			// List features depending on the meter
			features, err := a.featureRepository.ListFeatures(ctx, feature.ListFeaturesParams{
				Namespace:  input.ID.Namespace,
				MeterSlugs: []string{currentMeter.Key},
			})
			if err != nil {
				return meterpkg.Meter{}, fmt.Errorf("failed to list features for meter: %w", err)
			}

			// Check if the features are compatible with the new group by values
			for _, feature := range features.Items {
				for _, groupBy := range groupByToDelete {
					if _, ok := feature.MeterGroupByFilters[groupBy]; ok {
						return meterpkg.Meter{}, models.NewGenericConflictError(
							fmt.Errorf("meter group by: %s cannot be dropped because it is used by feature: %s", groupBy, feature.Key),
						)
					}
				}
			}
		}

		// Run pre-update hooks
		for _, hook := range a.preUpdateHooks {
			if err := hook(ctx, input); err != nil {
				return meterpkg.Meter{}, err
			}
		}

		// Update the meter
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *manageAdapter) (meterpkg.Meter, error) {
				entity, err := repo.db.Meter.UpdateOneID(input.ID.ID).
					Where(meterdb.NamespaceEQ(input.ID.Namespace)).
					SetName(input.Name).
					SetNillableDescription(input.Description).
					SetGroupBy(input.GroupBy).
					Save(ctx)
				if err != nil {
					if db.IsConstraintError(err) {
						return meterpkg.Meter{}, models.NewGenericConflictError(fmt.Errorf("meter with the same slug already exists"))
					}

					return meterpkg.Meter{}, fmt.Errorf("failed to update meter: %w", err)
				}

				meter, err := MapFromEntityFactory(entity)
				if err != nil {
					return meterpkg.Meter{}, fmt.Errorf("failed to map meter: %w", err)
				}

				// Update the meter in the streaming connector
				err = a.streamingConnector.UpdateMeter(ctx, input.ID.Namespace, meter)
				if err != nil {
					return meter, fmt.Errorf("failed to update meter in streaming connector: %w", err)
				}

				return meter, nil
			})
	})
}

// DeleteMeter deletes a meter.
func (a *manageAdapter) DeleteMeter(ctx context.Context, input meterpkg.DeleteMeterInput) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
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
	})
}
