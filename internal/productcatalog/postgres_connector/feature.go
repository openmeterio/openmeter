package postgres_connector

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/productcatalog/postgres_connector/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/productcatalog/postgres_connector/ent/db/feature"
)

type featureConnector struct {
	logger *slog.Logger
	db     *db.Client
}

func NewFeatureConnector() productcatalog.FeatureDB {
	return &featureConnector{}
}

func (c *featureConnector) CreateFeature(ctx context.Context, feature productcatalog.Feature) (productcatalog.Feature, error) {
	entity, err := c.db.Feature.Create().
		SetID(feature.ID.String()).
		SetName(feature.Name).
		SetNamespace(feature.Namespace).
		SetMeterSlug(feature.MeterSlug).
		SetMeterGroupByFilters(*feature.MeterGroupByFilters).
		Save(ctx)

	if err != nil {
		return productcatalog.Feature{}, err
	}

	return mapFeatureEntity(entity), nil
}

func (c *featureConnector) FindByName(ctx context.Context, namespace string, name string, includeArchived bool) ([]productcatalog.Feature, error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(namespace)).
		Where(db_feature.Name(name))

	if !includeArchived {
		query = query.Where(db_feature.Archived(false))
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	features := make([]productcatalog.Feature, 0, len(entities))
	for _, entity := range entities {
		features = append(features, mapFeatureEntity(entity))
	}

	return features, nil
}

func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID productcatalog.NamespacedFeatureID) error {
	err := c.db.Feature.Update().
		SetArchived(true).
		Where(db_feature.ID(featureID.ID.String())).
		Where(db_feature.Namespace(featureID.Namespace)).
		Exec(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return &productcatalog.FeatureNotFoundError{ID: featureID.ID}
		}

		return fmt.Errorf("failed to delete feature: %w", err)
	}

	return nil
}

func (c *featureConnector) ListFeatures(ctx context.Context, params productcatalog.ListFeaturesParams) ([]productcatalog.Feature, error) {
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
	case productcatalog.FeatureOrderByCreatedAt:
		query = query.Order(db_feature.ByCreatedAt())
	case productcatalog.FeatureOrderByUpdatedAt:
		query = query.Order(db_feature.ByUpdatedAt())
	default:
		query = query.Order(db_feature.ByID())
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	list := make([]productcatalog.Feature, 0, len(entities))
	for _, entity := range entities {
		feature := mapFeatureEntity(entity)
		list = append(list, feature)
	}

	return list, nil
}

func (c *featureConnector) GetByID(ctx context.Context, featureID productcatalog.NamespacedFeatureID) (productcatalog.Feature, error) {
	entity, err := c.db.Feature.Get(ctx, featureID.ID.String())
	if err != nil {
		if db.IsNotFound(err) {
			return productcatalog.Feature{}, &productcatalog.FeatureNotFoundError{ID: featureID.ID}
		}

		return productcatalog.Feature{}, fmt.Errorf("failed to get feature: %w", err)
	}

	if entity.Namespace != featureID.Namespace {
		return productcatalog.Feature{}, &productcatalog.FeatureNotFoundError{ID: featureID.ID}
	}

	featureOut := mapFeatureEntity(entity)
	return featureOut, nil
}

// mapFeatureEntity maps a database feature entity to a feature model.
func mapFeatureEntity(entity *db.Feature) productcatalog.Feature {
	feature := productcatalog.Feature{
		ID:        (productcatalog.FeatureID)(entity.ID),
		Namespace: entity.Namespace,
		Name:      entity.Name,
		MeterSlug: entity.MeterSlug,
		Archived:  entity.Archived,
		CreatedAt: entity.CreatedAt.In(time.UTC),
		UpdatedAt: entity.UpdatedAt.In(time.UTC),
	}

	if len(entity.MeterGroupByFilters) > 0 {
		feature.MeterGroupByFilters = &entity.MeterGroupByFilters
	}

	return feature
}
