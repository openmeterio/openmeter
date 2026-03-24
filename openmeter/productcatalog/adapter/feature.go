package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbfeature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	dbmeter "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
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
		SetNillableDescription(feat.Description).
		SetKey(feat.Key).
		SetNamespace(feat.Namespace).
		SetMetadata(feat.Metadata).
		SetNillableMeterID(feat.MeterID)

	if len(feat.MeterGroupByFilters) > 0 {
		query = query.
			SetAdvancedMeterGroupByFilters(feat.MeterGroupByFilters).
			SetMeterGroupByFilters(feature.ConvertMeterGroupByFiltersToMapString(feat.MeterGroupByFilters))
	}

	if feat.UnitCost != nil {
		query = query.SetUnitCostType(string(feat.UnitCost.Type))
		switch feat.UnitCost.Type {
		case feature.UnitCostTypeManual:
			if feat.UnitCost.Manual != nil {
				query = query.SetUnitCostManualAmount(feat.UnitCost.Manual.Amount)
			}
		case feature.UnitCostTypeLLM:
			if feat.UnitCost.LLM != nil {
				query = query.
					SetNillableUnitCostLlmProviderProperty(lo.EmptyableToPtr(feat.UnitCost.LLM.ProviderProperty)).
					SetNillableUnitCostLlmProvider(lo.EmptyableToPtr(feat.UnitCost.LLM.Provider)).
					SetNillableUnitCostLlmModelProperty(lo.EmptyableToPtr(feat.UnitCost.LLM.ModelProperty)).
					SetNillableUnitCostLlmModel(lo.EmptyableToPtr(feat.UnitCost.LLM.Model)).
					SetNillableUnitCostLlmTokenTypeProperty(lo.EmptyableToPtr(feat.UnitCost.LLM.TokenTypeProperty)).
					SetNillableUnitCostLlmTokenType(lo.EmptyableToPtr(feat.UnitCost.LLM.TokenType))
			}
		}
	}

	if len(feat.UnitConfig) > 0 {
		query = query.SetUnitConfig(string(feat.UnitConfig))
	}

	entity, err := query.
		Save(ctx)
	if err != nil {
		return feature.Feature{}, err
	}

	// Re-fetch with meter edge for MeterSlug backward compatibility
	if entity.MeterID != nil {
		entity, err = c.db.Feature.Query().
			Where(dbfeature.ID(entity.ID)).
			WithMeter(func(mq *db.MeterQuery) {
				mq.Select(dbmeter.FieldID, dbmeter.FieldKey)
			}).
			Only(ctx)
		if err != nil {
			return feature.Feature{}, err
		}
	}

	return MapFeatureEntity(entity), nil
}

func (c *featureDBAdapter) GetByIdOrKey(ctx context.Context, namespace string, idOrKey string, includeArchived bool) (*feature.Feature, error) {
	query := c.db.Feature.Query().
		// We only need Meter Key for v1 API backward compatibility
		WithMeter(func(mq *db.MeterQuery) {
			mq.Select(dbmeter.FieldID, dbmeter.FieldKey)
		}).
		Where(dbfeature.Namespace(namespace)).
		Where(dbfeature.Or(dbfeature.Key(idOrKey), dbfeature.ID(idOrKey)))

	if !includeArchived {
		query = query.Where(dbfeature.ArchivedAtIsNil())
	}

	// This ensures that the first item is the most recent one
	query = query.Order(dbfeature.ByArchivedAt(sql.OrderDesc(), sql.OrderNullsFirst()))

	entities, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	// FIXME: use models.NewGenericNotFoundError instead of feature.FeatureNotFoundError
	if len(entities) == 0 {
		return nil, &feature.FeatureNotFoundError{ID: idOrKey}
	}

	res := MapFeatureEntity(entities[0])

	return &res, nil
}

func (c *featureDBAdapter) ArchiveFeature(ctx context.Context, params feature.ArchiveFeatureInput) error {
	f, err := c.GetByIdOrKey(ctx, params.Namespace, params.ID, true)
	if err != nil {
		return err
	}

	archivedAt := clock.Now()
	if params.At != nil {
		if params.At.Before(f.UpdatedAt) {
			return &feature.ForbiddenError{Msg: "cannot archive feature at a time before it was last updated", ID: f.ID}
		}

		archivedAt = *params.At
	}

	// FIXME: (OM-1055) we should marry productcatalog/plan with feature so we can do this check outside the db layer
	planReferencesIt, err := c.db.Plan.Query().
		WithPhases(func(qp *db.PlanPhaseQuery) {
			qp.WithRatecards()
		}).
		Where(
			dbplan.Namespace(params.Namespace),
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
			dbsub.Namespace(params.Namespace),
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
		SetArchivedAt(archivedAt).
		Where(dbfeature.ID(params.ID)).
		Where(dbfeature.Namespace(params.Namespace)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to archive feature: %w", err)
	}

	return nil
}

func (c *featureDBAdapter) HasActiveFeatureForMeter(ctx context.Context, namespace string, meterID string) (bool, error) {
	exists, err := c.db.Feature.Query().
		Where(dbfeature.Namespace(namespace)).
		Where(dbfeature.MeterID(meterID)).
		Where(dbfeature.Or(dbfeature.ArchivedAtIsNil(), dbfeature.ArchivedAtGT(clock.Now()))).
		Exist(ctx)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (c *featureDBAdapter) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	query := c.db.Feature.Query().
		// We only need Meter Key for v1 API backward compatibility
		WithMeter(func(mq *db.MeterQuery) {
			mq.Select(dbmeter.FieldID, dbmeter.FieldKey)
		}).
		Where(dbfeature.Namespace(params.Namespace))

	if len(params.MeterIDs) > 0 {
		query = query.Where(dbfeature.MeterIDIn(params.MeterIDs...))
	}

	if len(params.MeterSlugs) > 0 {
		query = query.Where(dbfeature.HasMeterWith(dbmeter.KeyIn(params.MeterSlugs...)))
	}

	if len(params.IDsOrKeys) > 0 {
		query = query.Where(dbfeature.Or(dbfeature.IDIn(params.IDsOrKeys...), dbfeature.KeyIn(params.IDsOrKeys...)))
	}

	if !params.IncludeArchived {
		query = query.Where(dbfeature.Or(dbfeature.ArchivedAtIsNil(), dbfeature.ArchivedAtGT(clock.Now())))
	}

	if params.OrderBy != "" {
		order := []sql.OrderTermOption{}
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case feature.FeatureOrderByKey:
			query = query.Order(dbfeature.ByKey(order...))
		case feature.FeatureOrderByName:
			query = query.Order(dbfeature.ByName(order...))
		case feature.FeatureOrderByCreatedAt:
			query = query.Order(dbfeature.ByCreatedAt(order...))
		case feature.FeatureOrderByUpdatedAt:
			query = query.Order(dbfeature.ByUpdatedAt(order...))
		default:
			query = query.Order(dbfeature.ByCreatedAt(order...))
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
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		Name:        entity.Name,
		Description: entity.Description,
		Key:         entity.Key,
		MeterID:     entity.MeterID,
		ArchivedAt:  entity.ArchivedAt,
		CreatedAt:   entity.CreatedAt.In(time.UTC),
		UpdatedAt:   entity.UpdatedAt.In(time.UTC),
		Metadata:    entity.Metadata,
	}

	// Deprecated: kept for v1 API backward compatibility
	if entity.Edges.Meter != nil {
		f.MeterSlug = &entity.Edges.Meter.Key
	}

	// Use advanced meter group by filters if available
	if len(entity.AdvancedMeterGroupByFilters) > 0 {
		f.MeterGroupByFilters = entity.AdvancedMeterGroupByFilters
	} else if len(entity.MeterGroupByFilters) > 0 {
		f.MeterGroupByFilters = feature.ConvertMapStringToMeterGroupByFilters(entity.MeterGroupByFilters)
	}

	if entity.UnitCostType != nil {
		switch feature.UnitCostType(*entity.UnitCostType) {
		case feature.UnitCostTypeManual:
			var amount alpacadecimal.Decimal
			if entity.UnitCostManualAmount != nil {
				amount = *entity.UnitCostManualAmount
			}
			f.UnitCost = &feature.UnitCost{
				Type: feature.UnitCostTypeManual,
				Manual: &feature.ManualUnitCost{
					Amount: amount,
				},
			}
		case feature.UnitCostTypeLLM:
			f.UnitCost = &feature.UnitCost{
				Type: feature.UnitCostTypeLLM,
				LLM: &feature.LLMUnitCost{
					ProviderProperty:  lo.FromPtrOr(entity.UnitCostLlmProviderProperty, ""),
					Provider:          lo.FromPtrOr(entity.UnitCostLlmProvider, ""),
					ModelProperty:     lo.FromPtrOr(entity.UnitCostLlmModelProperty, ""),
					Model:             lo.FromPtrOr(entity.UnitCostLlmModel, ""),
					TokenTypeProperty: lo.FromPtrOr(entity.UnitCostLlmTokenTypeProperty, ""),
					TokenType:         lo.FromPtrOr(entity.UnitCostLlmTokenType, ""),
				},
			}
		}
	}

	if entity.UnitConfig != nil && *entity.UnitConfig != "" {
		f.UnitConfig = json.RawMessage(*entity.UnitConfig)
	}

	return f
}
