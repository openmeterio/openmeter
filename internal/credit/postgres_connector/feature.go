package postgres_connector

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	connector "github.com/openmeterio/openmeter/internal/credit"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/feature"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
)

// CreateFeature creates a feature.
func (c *PostgresConnector) CreateFeature(ctx context.Context, namespace string, featureIn credit_model.Feature) (credit_model.Feature, error) {
	query := c.db.Feature.Create().
		SetNillableID(pgulid.Ptr(featureIn.ID)).
		SetName(featureIn.Name).
		SetNamespace(namespace).
		SetMeterSlug(featureIn.MeterSlug)

	if featureIn.MeterGroupByFilters != nil {
		query.SetMeterGroupByFilters(*featureIn.MeterGroupByFilters)
	}

	entity, err := query.Save(ctx)
	if err != nil {
		return credit_model.Feature{}, fmt.Errorf("failed to create feature: %w", err)
	}

	featureOut := mapFeatureEntity(entity)
	return featureOut, nil
}

// DeleteFeature deletes a feature.
func (c *PostgresConnector) DeleteFeature(ctx context.Context, namespace string, id ulid.ULID) error {
	err := c.db.Feature.UpdateOneID(pgulid.Wrap(id)).SetArchived(true).Exec(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return &credit_model.FeatureNotFoundError{ID: id}
		}

		return fmt.Errorf("failed to delete feature: %w", err)
	}

	return nil
}

// ListFeatures lists features.
func (c *PostgresConnector) ListFeatures(ctx context.Context, namespace string, params connector.ListFeaturesParams) ([]credit_model.Feature, error) {
	query := c.db.Feature.Query()
	if !params.IncludeArchived {
		query = query.Where(db_feature.ArchivedEQ(false))
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	var list []credit_model.Feature
	for _, entity := range entities {
		feature := mapFeatureEntity(entity)
		list = append(list, feature)
	}

	return list, nil
}

// GetFeature gets a single feature by ID.
func (c *PostgresConnector) GetFeature(ctx context.Context, namespace string, id ulid.ULID) (credit_model.Feature, error) {
	entity, err := c.db.Feature.Get(ctx, pgulid.Wrap(id))
	if err != nil {
		if db.IsNotFound(err) {
			return credit_model.Feature{}, &credit_model.FeatureNotFoundError{ID: id}
		}

		return credit_model.Feature{}, fmt.Errorf("failed to get feature: %w", err)
	}

	featureOut := mapFeatureEntity(entity)
	return featureOut, nil
}

// mapFeatureEntity maps a database feature entity to a feature model.
func mapFeatureEntity(entity *db.Feature) credit_model.Feature {
	feature := credit_model.Feature{
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
