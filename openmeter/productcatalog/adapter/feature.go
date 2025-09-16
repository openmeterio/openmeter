package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_feature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	dbplan "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	dbplanphase "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	dbratecard "github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	dbsub "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	dbsubitem "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionitem"
	dbsubphase "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
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

func NewPostgresFeatureRepo(db *db.Client, logger *slog.Logger) feature.FeatureRepo {
	return &featureDBAdapter{
		db:     db,
		logger: logger,
	}
}

func (c *featureDBAdapter) CreateFeature(ctx context.Context, feat feature.CreateFeatureInputs) (feature.Feature, error) {
	query := c.db.Feature.Create().
		SetName(feat.Name).
		SetKey(feat.Key).
		SetNamespace(feat.Namespace).
		SetMetadata(feat.Metadata).
		SetNillableMeterSlug(feat.MeterSlug)

	if feat.MeterGroupByFilters != nil {
		query = query.SetMeterGroupByFilters(feat.MeterGroupByFilters)
	}

	entity, err := query.
		Save(ctx)
	if err != nil {
		return feature.Feature{}, err
	}

	return MapFeatureEntity(entity), nil
}

func (c *featureDBAdapter) GetByIdOrKey(ctx context.Context, namespace string, idOrKey string, includeArchived bool) (*feature.Feature, error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(namespace)).
		Where(db_feature.Or(db_feature.Key(idOrKey), db_feature.ID(idOrKey)))

	if !includeArchived {
		query = query.Where(db_feature.ArchivedAtIsNil())
	}

	// This ensures that the first item is the most recent one
	query = query.Order(db_feature.ByArchivedAt(sql.OrderDesc(), sql.OrderNullsFirst()))

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	if len(entities) == 0 {
		return nil, &feature.FeatureNotFoundError{ID: idOrKey}
	}

	res := MapFeatureEntity(entities[0])

	return &res, nil
}

func (c *featureDBAdapter) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	f, err := c.GetByIdOrKey(ctx, featureID.Namespace, featureID.ID, true)
	if err != nil {
		return err
	}

	// FIXME: (OM-1055) we should marry productcatalog/plan with feature so we can do this check outside the db layer
	planReferencesIt, err := c.db.Plan.Query().
		WithPhases(func(qp *db.PlanPhaseQuery) {
			qp.WithRatecards()
		}).
		Where(
			dbplan.Namespace(featureID.Namespace),
			dbplan.EffectiveFromNotNil(),
			dbplan.Or(dbplan.EffectiveToGT(clock.Now()), dbplan.EffectiveToIsNil()),
			dbplan.HasPhasesWith(dbplanphase.HasRatecardsWith(
				dbratecard.Or(dbratecard.FeatureID(f.ID), dbratecard.FeatureKey(f.Key)),
			)),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for plan references: %w", err)
	}

	subsReferencesIt, err := c.db.Subscription.Query().
		WithPhases(func(qp *db.SubscriptionPhaseQuery) {
			qp.WithItems()
		}).
		Where(
			subscriptionrepo.SubscriptionActiveAfter(clock.Now())...,
		).
		Where(
			dbsub.Namespace(featureID.Namespace),
			dbsub.HasPhasesWith(dbsubphase.HasItemsWith(dbsubitem.FeatureKey(f.Key))),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for subscription references: %w", err)
	}

	// As currently features are referenced by IDs instead of Keys, and there's no way to publish a new feature version in a single action,
	// using subscriptions/productcatalog bricks referenced features either way as they can no longer be updated.
	if planReferencesIt {
		return &feature.ForbiddenError{Msg: "feature is referenced by active plan, it cannot be archived", ID: f.ID}
	}

	if subsReferencesIt {
		return &feature.ForbiddenError{Msg: "feature is referenced by active subscription, it cannot be archived", ID: f.ID}
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

func (c *featureDBAdapter) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	query := c.db.Feature.Query().
		Where(db_feature.Namespace(params.Namespace))

	if len(params.MeterSlugs) > 0 {
		query = query.Where(db_feature.MeterSlugIn(params.MeterSlugs...))
	}

	if len(params.IDsOrKeys) > 0 {
		query = query.Where(db_feature.Or(db_feature.IDIn(params.IDsOrKeys...), db_feature.KeyIn(params.IDsOrKeys...)))
	}

	if !params.IncludeArchived {
		query = query.Where(db_feature.Or(db_feature.ArchivedAtIsNil(), db_feature.ArchivedAtGT(clock.Now())))
	}

	if params.OrderBy != "" {
		order := []sql.OrderTermOption{}
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case feature.FeatureOrderByKey:
			query = query.Order(db_feature.ByKey(order...))
		case feature.FeatureOrderByName:
			query = query.Order(db_feature.ByName(order...))
		case feature.FeatureOrderByCreatedAt:
			query = query.Order(db_feature.ByCreatedAt(order...))
		case feature.FeatureOrderByUpdatedAt:
			query = query.Order(db_feature.ByUpdatedAt(order...))
		default:
			query = query.Order(db_feature.ByCreatedAt(order...))
		}
	}

	response := pagination.Result[feature.Feature]{
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

		mapped := make([]feature.Feature, 0, len(entities))
		for _, entity := range entities {
			mapped = append(mapped, MapFeatureEntity(entity))
		}

		response.Items = mapped
		return response, nil
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	list := make([]feature.Feature, 0, len(paged.Items))
	for _, entity := range paged.Items {
		f := MapFeatureEntity(entity)
		list = append(list, f)
	}

	response.Items = list
	response.TotalCount = paged.TotalCount

	return response, nil
}

// mapFeatureEntity maps a database feature entity to a feature model.
func MapFeatureEntity(entity *db.Feature) feature.Feature {
	f := feature.Feature{
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
		f.MeterGroupByFilters = entity.MeterGroupByFilters
	}

	return f
}
