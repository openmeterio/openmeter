package postgres_connector

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/feature"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
)

// CreateFeature creates a feature.
func (c *PostgresConnector) CreateFeature(ctx context.Context, featureIn credit.Feature) (credit.Feature, error) {
	query := c.db.Feature.Create().
		SetNillableID(pgulid.Ptr(featureIn.ID)).
		SetName(featureIn.Name).
		SetNamespace(featureIn.Namespace).
		SetMeterSlug(featureIn.MeterSlug)

	if featureIn.MeterGroupByFilters != nil {
		query.SetMeterGroupByFilters(*featureIn.MeterGroupByFilters)
	}

	entity, err := query.Save(ctx)
	if err != nil {
		return credit.Feature{}, fmt.Errorf("failed to create feature: %w", err)
	}

	featureOut := mapFeatureEntity(entity)
	return featureOut, nil
}

// DeleteFeature deletes a feature.
func (c *PostgresConnector) DeleteFeature(ctx context.Context, id credit.NamespacedID) error {
	err := c.db.Feature.Update().
		SetArchived(true).
		Where(db_feature.ID(pgulid.Wrap(id.ID))).
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

	entities, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	var list []credit.Feature
	for _, entity := range entities {
		feature := mapFeatureEntity(entity)
		list = append(list, feature)
	}

	return list, nil
}

// GetFeature gets a single feature by ID.
func (c *PostgresConnector) GetFeature(ctx context.Context, id credit.NamespacedID) (credit.Feature, error) {
	entity, err := c.db.Feature.Get(ctx, pgulid.Wrap(id.ID))
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
		ID:        &entity.ID.ULID,
		Namespace: entity.Namespace,
		Name:      entity.Name,
		MeterSlug: entity.MeterSlug,
		Archived:  &entity.Archived,
	}

	if len(entity.MeterGroupByFilters) > 0 {
		feature.MeterGroupByFilters = &entity.MeterGroupByFilters
	}

	return feature
}
