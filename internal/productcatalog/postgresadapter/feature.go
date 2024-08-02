package postgresadapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/ent/db"
	db_feature "github.com/openmeterio/openmeter/internal/ent/db/feature"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
		SetArchivedAt(clock.Now()).
		Where(db_feature.ID(featureID.ID)).
		Where(db_feature.Namespace(featureID.Namespace)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to archive feature: %w", err)
	}

	return nil
}

func (c *featureDBAdapter) HasActiveFeatureForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error) {
	exists, err := c.db.Feature.Query().
		Where(db_feature.Namespace(namespace)).
		Where(db_feature.MeterSlug(meterSlug)).
		Where(db_feature.Or(db_feature.ArchivedAtIsNil(), db_feature.ArchivedAtGT(clock.Now()))).
		Exist(ctx)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (c *featureDBAdapter) ListFeatures(ctx context.Context, params productcatalog.ListFeaturesParams) (pagination.PagedResponse[productcatalog.Feature], error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(params.Namespace))

	if len(params.MeterSlugs) > 0 {
		query.Where(db_feature.MeterSlugIn(params.MeterSlugs...))
	}

	if len(params.IDs) > 0 {
		query.Where(db_feature.IDIn(params.IDs...))
	}

	if !params.IncludeArchived {
		query = query.Where(db_feature.Or(db_feature.ArchivedAtIsNil(), db_feature.ArchivedAtGT(clock.Now())))
	}

	if params.OrderBy == "" {
		order := []sql.OrderTermOption{}
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case productcatalog.FeatureOrderByCreatedAt:
			query = query.Order(db_feature.ByCreatedAt(order...))
		case productcatalog.FeatureOrderByUpdatedAt:
			query = query.Order(db_feature.ByUpdatedAt(order...))
		default:
			query = query.Order(db_feature.ByCreatedAt(order...))
		}
	}

	response := pagination.PagedResponse[productcatalog.Feature]{
		Page: params.Page,
	}

	// we're using limit and offset
	if params.Page.IsZero() {
		if params.Limit > 0 {
			query = query.Limit(params.Limit)
		}
		if params.Offset > 0 {
			query = query.Offset(params.Offset)
		}

		entities, err := query.All(ctx)
		if err != nil {
			return response, err
		}

		mapped := make([]productcatalog.Feature, 0, len(entities))
		for _, entity := range entities {
			mapped = append(mapped, mapFeatureEntity(entity))
		}

		response.Items = mapped
		return response, nil
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	list := make([]productcatalog.Feature, 0, len(paged.Items))
	for _, entity := range paged.Items {
		feature := mapFeatureEntity(entity)
		list = append(list, feature)
	}

	response.Items = list
	response.TotalCount = paged.TotalCount

	return response, nil
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
		feature.MeterGroupByFilters = entity.MeterGroupByFilters
	}

	return feature
}
