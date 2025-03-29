package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meterdb "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateMeter creates a new meter.
func (a *Adapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	if err := input.Validate(); err != nil {
		return meterpkg.Meter{}, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *Adapter) (meterpkg.Meter, error) {
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

				return meter, nil
			})
	})
}

// UpdateMeter updates a new meter.
func (a *Adapter) UpdateMeter(ctx context.Context, input meterpkg.UpdateMeterInput) (meterpkg.Meter, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
		// Update the meter
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *Adapter) (meterpkg.Meter, error) {
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

				return meter, nil
			})
	})
}

// DeleteMeter deletes a meter.
func (a *Adapter) DeleteMeter(ctx context.Context, meter meterpkg.Meter) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		// Delete the meter
		return entutils.TransactingRepoWithNoValue(
			ctx,
			a,
			func(ctx context.Context, repo *Adapter) error {
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

				return nil
			})
	})
}
