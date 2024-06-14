package postgres_adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/productcatalog/postgres_adapter/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/productcatalog/postgres_adapter/ent/db/feature"
)

// Adapter implements remote connector interface as driven port.
type featureDBAdapter struct {
	logger *slog.Logger
	db     *db.Client
}

func NewPostgresFeatureDBAdapter(db *db.Client, logger *slog.Logger) productcatalog.FeatureDBConnector {
	return &featureDBAdapter{
		db:     db,
		logger: logger,
	}
}

func (c *featureDBAdapter) CreateFeature(ctx context.Context, feature productcatalog.DBCreateFeatureInputs) (productcatalog.Feature, error) {
	entity, err := c.db.Feature.Create().
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

func (c *featureDBAdapter) FindByName(ctx context.Context, namespace string, name string, includeArchived bool) ([]productcatalog.Feature, error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(namespace)).
		Where(db_feature.Name(name))

	if !includeArchived {
		query = query.Where(db_feature.ArchivedAtIsNil())
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

func (c *featureDBAdapter) ArchiveFeature(ctx context.Context, featureID productcatalog.NamespacedFeatureID) error {
	_, err := c.GetByID(ctx, featureID)
	if err != nil {
		return err
	}

	err = c.db.Feature.Update().
		SetArchivedAt(time.Now()).
		Where(db_feature.ID(featureID.ID.String())).
		Where(db_feature.Namespace(featureID.Namespace)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to archive feature: %w", err)
	}

	return nil
}

func (c *featureDBAdapter) ListFeatures(ctx context.Context, params productcatalog.ListFeaturesParams) ([]productcatalog.Feature, error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(params.Namespace))

	if !params.IncludeArchived {
		query = query.Where(db_feature.ArchivedAtIsNil())
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
		query = query.Order(db_feature.ByCreatedAt())
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

func (c *featureDBAdapter) GetByID(ctx context.Context, featureID productcatalog.NamespacedFeatureID) (productcatalog.Feature, error) {
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
		ID:         (productcatalog.FeatureID)(entity.ID),
		Namespace:  entity.Namespace,
		Name:       entity.Name,
		MeterSlug:  entity.MeterSlug,
		ArchivedAt: entity.ArchivedAt,
		CreatedAt:  entity.CreatedAt.In(time.UTC),
		UpdatedAt:  entity.UpdatedAt.In(time.UTC),
	}

	if len(entity.MeterGroupByFilters) > 0 {
		feature.MeterGroupByFilters = &entity.MeterGroupByFilters
	}

	return feature
}
