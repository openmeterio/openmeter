package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meterdb "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMeters returns a list of meters.
func (a *adapter) ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.PagedResponse[meter.Meter], error) {
	if err := params.Validate(); err != nil {
		return pagination.PagedResponse[meter.Meter]{}, models.NewGenericValidationError(err)
	}

	query := a.db.Meter.Query()

	if !params.WithoutNamespace {
		query = query.
			Where(meterdb.NamespaceEQ(params.Namespace)).
			Where(meterdb.DeletedAtIsNil())
	}

	if params.SlugFilter != nil {
		query = query.Where(meterdb.KeyIn(*params.SlugFilter...))
	}

	entities, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return pagination.PagedResponse[meter.Meter]{}, err
	}

	resp, err := pagination.MapPagedResponseError(entities, MapFromEntityFactory)
	if err != nil {
		return pagination.PagedResponse[meter.Meter]{}, fmt.Errorf("failed to map meters: %w", err)
	}

	return resp, nil
}

// GetMeterByIDOrSlug returns a meter from the meter store by ID or slug.
func (a *adapter) GetMeterByIDOrSlug(ctx context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	if err := input.Validate(); err != nil {
		return meter.Meter{}, models.NewGenericValidationError(err)
	}

	entity, err := a.db.Meter.Query().
		Where(meterdb.NamespaceEQ(input.Namespace)).
		Where(meterdb.Or(
			meterdb.ID(input.IDOrSlug),
			meterdb.Key(input.IDOrSlug),
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
		Namespace:     entity.Namespace,
		ID:            entity.ID,
		Key:           entity.Key,
		Name:          entity.Name,
		Description:   entity.Description,
		Aggregation:   entity.Aggregation,
		EventType:     entity.EventType,
		ValueProperty: entity.ValueProperty,
		GroupBy:       entity.GroupBy,
		WindowSize:    meter.WindowSizeMinute,
	}, nil
}
