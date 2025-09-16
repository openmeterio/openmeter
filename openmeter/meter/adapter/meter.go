package adapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meterdb "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMeters returns a list of meters.
func (a *Adapter) ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return pagination.Result[meter.Meter]{}, models.NewGenericValidationError(err)
	}

	// Start database query
	query := a.db.Meter.Query()

	// Filtering
	if !params.WithoutNamespace {
		query = query.
			Where(meterdb.NamespaceEQ(params.Namespace))
	}

	if !params.IncludeDeleted {
		query = query.Where(meterdb.DeletedAtIsNil())
	}

	if params.SlugFilter != nil {
		query = query.Where(meterdb.KeyIn(*params.SlugFilter...))
	}

	// Ordering
	if params.Order != "" {
		order := []sql.OrderTermOption{}
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case meter.OrderByKey:
			query = query.Order(meterdb.ByKey(order...))
		case meter.OrderByName:
			query = query.Order(meterdb.ByName(order...))
		case meter.OrderByAggregation:
			query = query.Order(meterdb.ByAggregation(order...))
		case meter.OrderByCreatedAt:
			query = query.Order(meterdb.ByCreatedAt(order...))
		case meter.OrderByUpdatedAt:
			query = query.Order(meterdb.ByUpdatedAt(order...))
		default:
			query = query.Order(meterdb.ByCreatedAt(order...))
		}
	}

	// Pagination
	entities, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return pagination.Result[meter.Meter]{}, err
	}

	// Map to response
	resp, err := pagination.MapResultErr(entities, MapFromEntityFactory)
	if err != nil {
		return pagination.Result[meter.Meter]{}, fmt.Errorf("failed to map meters: %w", err)
	}

	return resp, nil
}

// GetMeterByIDOrSlug returns a meter from the meter store by ID or slug.
func (a *Adapter) GetMeterByIDOrSlug(ctx context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	if err := input.Validate(); err != nil {
		return meter.Meter{}, models.NewGenericValidationError(err)
	}

	entity, err := a.db.Meter.Query().
		Where(meterdb.NamespaceEQ(input.Namespace)).
		Where(meterdb.Or(
			meterdb.ID(input.IDOrSlug),
			meterdb.And(meterdb.Key(input.IDOrSlug), meterdb.DeletedAtIsNil()),
		)).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return meter.Meter{}, meter.NewMeterNotFoundError(input.IDOrSlug)
		}

		return meter.Meter{}, fmt.Errorf("failed to get meter by ID or slug: %w", err)
	}

	meter, err := MapFromEntityFactory(entity)
	if err != nil {
		return meter, fmt.Errorf("failed to map meter: %w", err)
	}

	return meter, nil
}

// MapFromEntityFactory creates a function that maps a meter db entity to a meter model.
func MapFromEntityFactory(entity *db.Meter) (meter.Meter, error) {
	if entity == nil {
		return meter.Meter{}, fmt.Errorf("entity is required")
	}

	return meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: entity.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: entity.CreatedAt,
				UpdatedAt: entity.UpdatedAt,
				DeletedAt: entity.DeletedAt,
			},
			ID:          entity.ID,
			Name:        entity.Name,
			Description: entity.Description,
		},
		Key:           entity.Key,
		Aggregation:   entity.Aggregation,
		EventType:     entity.EventType,
		EventFrom:     entity.EventFrom,
		ValueProperty: entity.ValueProperty,
		GroupBy:       entity.GroupBy,
	}, nil
}
