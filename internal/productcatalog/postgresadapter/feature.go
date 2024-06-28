package postgresadapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Adapter implements remote connector interface as driven port.
type featureDBAdapter struct {
	logger *slog.Logger
	db     *db.Client
}

func NewPostgresFeatureRepo(db *db.Client, logger *slog.Logger) productcatalog.FeatureRepo {
	return &featureDBAdapter{
		db:     db,
		logger: logger,
	}
}

func (c *featureDBAdapter) CreateFeature(ctx context.Context, feature productcatalog.CreateFeatureInputs) (productcatalog.Feature, error) {
	query := c.db.Feature.Create().
		SetName(feature.Name).
		SetKey(feature.Key).
		SetNamespace(feature.Namespace).
		SetMetadata(feature.Metadata).
		SetNillableMeterSlug(feature.MeterSlug)

	if feature.MeterGroupByFilters != nil {
		query = query.SetMeterGroupByFilters(feature.MeterGroupByFilters)
	}

	entity, err := query.
		Save(ctx)

	if err != nil {
		return productcatalog.Feature{}, err
	}

	return mapFeatureEntity(entity), nil
}

func (c *featureDBAdapter) GetByIdOrKey(ctx context.Context, namespace string, idOrKey string, includeArchived bool) (*productcatalog.Feature, error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(namespace)).
		Where(db_feature.Or(db_feature.Key(idOrKey), db_feature.ID(idOrKey)))

	if !includeArchived {
		query = query.Where(db_feature.ArchivedAtIsNil())
	}

	entity, err := query.Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, &productcatalog.FeatureNotFoundError{ID: idOrKey}
		}
		return nil, err
	}

	res := mapFeatureEntity(entity)

	return &res, nil
}

func (c *featureDBAdapter) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	_, err := c.GetByIdOrKey(ctx, featureID.Namespace, featureID.ID, true)
	if err != nil {
		return err
	}

	err = c.db.Feature.Update().
		SetArchivedAt(time.Now()).
		Where(db_feature.ID(featureID.ID)).
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

	if params.MeterSlug != "" {
		query.Where(db_feature.MeterSlugEQ(params.MeterSlug))
	}

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

// mapFeatureEntity maps a database feature entity to a feature model.
func mapFeatureEntity(entity *db.Feature) productcatalog.Feature {
	feature := productcatalog.Feature{
		ID:         entity.ID,
		Namespace:  entity.Namespace,
		Name:       entity.Name,
		Key:        entity.Key,
		MeterSlug:  entity.MeterSlug,
		ArchivedAt: entity.ArchivedAt,
		CreatedAt:  entity.CreatedAt.In(time.UTC),
		UpdatedAt:  entity.UpdatedAt.In(time.UTC),
		Metadata:   entity.Metadata,
	}

	if len(entity.MeterGroupByFilters) > 0 {
		feature.MeterGroupByFilters = &entity.MeterGroupByFilters
	}

	return feature
}
