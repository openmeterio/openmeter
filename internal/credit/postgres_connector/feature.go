package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateFeature creates a feature.
func (c *PostgresConnector) CreateFeature(ctx context.Context, featureIn credit.Feature) (credit.Feature, error) {
	query := c.db.Feature.Create().
		SetNillableID((*string)(featureIn.ID)).
		SetName(featureIn.Name).
		SetNamespace(featureIn.Namespace).
		SetMeterSlug(featureIn.MeterSlug)

	// validate that the meter exists
	meter, err := c.meterRepository.GetMeterByIDOrSlug(ctx, featureIn.Namespace, featureIn.MeterSlug)
	if err != nil {
		return credit.Feature{}, &models.MeterNotFoundError{MeterSlug: featureIn.MeterSlug}
	}

	if featureIn.MeterGroupByFilters != nil {
		// validate that the MeterGroupByFilters point to actual meter groupbys
		for filterProp := range *featureIn.MeterGroupByFilters {
			if _, ok := meter.GroupBy[filterProp]; !ok {
				meterGroupByColumns := make([]string, 0, len(meter.GroupBy))
				for k := range meter.GroupBy {
					meterGroupByColumns = append(meterGroupByColumns, k)
				}
				return credit.Feature{}, &credit.FeatureInvalidFiltersError{
					RequestedFilters:    *featureIn.MeterGroupByFilters,
					MeterGroupByColumns: meterGroupByColumns,
				}
			}
		}
		query.SetMeterGroupByFilters(*featureIn.MeterGroupByFilters)
	}

	// validate that the feature name is uniq among active features
	r, err := c.db.Feature.Query().
		Where(db_feature.Namespace(featureIn.Namespace)).
		Where(db_feature.Name(featureIn.Name)).
		Where(db_feature.Archived(false)).
		All(ctx)

	if err != nil {
		return credit.Feature{}, fmt.Errorf("failed to query for existing features: %w", err)
	}

	if len(r) > 0 {
		foundFeature := r[0]
		return credit.Feature{},
			&credit.FeatureWithNameAlreadyExistsError{Name: featureIn.Name, ID: credit.FeatureID(foundFeature.ID)}
	}

	entity, err := query.Save(ctx)
	if err != nil {
		return credit.Feature{}, fmt.Errorf("failed to create feature: %w", err)
	}

	featureOut := mapFeatureEntity(entity)
	return featureOut, nil
}

// DeleteFeature deletes a feature.
func (c *PostgresConnector) DeleteFeature(ctx context.Context, id credit.NamespacedFeatureID) error {
	err := c.db.Feature.Update().
		SetArchived(true).
		Where(db_feature.ID(string(id.ID))).
		Where(db_feature.Namespace(id.Namespace)).
		Exec(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return &credit.FeatureNotFoundError{ID: id.ID}
		}

		return fmt.Errorf("failed to delete feature: %w", err)
	}

	return nil
}

// ListFeatures lists features.
func (c *PostgresConnector) ListFeatures(ctx context.Context, params credit.ListFeaturesParams) ([]credit.Feature, error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(params.Namespace))

	if !params.IncludeArchived {
		query = query.Where(db_feature.ArchivedEQ(false))
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	switch params.OrderBy {
	case credit.FeatureOrderByCreatedAt:
		query = query.Order(db_feature.ByCreatedAt())
	case credit.FeatureOrderByUpdatedAt:
		query = query.Order(db_feature.ByUpdatedAt())
	default:
		query = query.Order(db_feature.ByID())
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	list := make([]credit.Feature, 0, len(entities))
	for _, entity := range entities {
		feature := mapFeatureEntity(entity)
		list = append(list, feature)
	}

	return list, nil
}

// GetFeature gets a single feature by ID.
func (c *PostgresConnector) GetFeature(ctx context.Context, id credit.NamespacedFeatureID) (credit.Feature, error) {
	entity, err := c.db.Feature.Get(ctx, string(id.ID))
	if err != nil {
		if db.IsNotFound(err) {
			return credit.Feature{}, &credit.FeatureNotFoundError{ID: id.ID}
		}

		return credit.Feature{}, fmt.Errorf("failed to get feature: %w", err)
	}

	if entity.Namespace != id.Namespace {
		return credit.Feature{}, &credit.FeatureNotFoundError{ID: id.ID}
	}

	featureOut := mapFeatureEntity(entity)
	return featureOut, nil
}

// mapFeatureEntity maps a database feature entity to a feature model.
func mapFeatureEntity(entity *db.Feature) credit.Feature {
	feature := credit.Feature{
		ID:        (*credit.FeatureID)(&entity.ID),
		Namespace: entity.Namespace,
		Name:      entity.Name,
		MeterSlug: entity.MeterSlug,
		Archived:  &entity.Archived,
		CreatedAt: convert.ToPointer(entity.CreatedAt.In(time.UTC)),
		UpdatedAt: convert.ToPointer(entity.UpdatedAt.In(time.UTC)),
	}

	if len(entity.MeterGroupByFilters) > 0 {
		feature.MeterGroupByFilters = &entity.MeterGroupByFilters
	}

	return feature
}
