package adapter

import (
	"context"
	"fmt"
	"slices"
	"time"

	"entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	addondb "github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	addonratecarddb "github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	planaddondb "github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
	phasedb "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	ratecarddb "github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListAddons(ctx context.Context, params addon.ListAddonsInput) (pagination.Result[addon.Addon], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[addon.Addon], error) {
		if err := params.Validate(); err != nil {
			return pagination.Result[addon.Addon]{}, fmt.Errorf("invalid list add-on parameters: %w", err)
		}

		query := a.db.Addon.Query()

		if len(params.Namespaces) > 0 {
			query = query.Where(addondb.NamespaceIn(params.Namespaces...))
		}

		var orFilters []predicate.Addon
		if len(params.IDs) > 0 {
			orFilters = append(orFilters, addondb.IDIn(params.IDs...))
		}

		if len(params.Keys) > 0 {
			orFilters = append(orFilters, addondb.KeyIn(params.Keys...))
		}

		if len(params.KeyVersions) > 0 {
			for key, version := range params.KeyVersions {
				orFilters = append(orFilters, addondb.And(addondb.Key(key), addondb.VersionIn(version...)))
			}
		}

		if len(params.Currencies) > 0 {
			orFilters = append(orFilters, addondb.CurrencyIn(params.Currencies...))
		}

		if len(orFilters) > 0 {
			query = query.Where(addondb.Or(orFilters...))
		}

		if !params.IncludeDeleted {
			query = query.Where(addondb.DeletedAtIsNil())
		}

		if len(params.Status) > 0 {
			var predicates []predicate.Addon

			now := clock.Now().UTC()

			if slices.Contains(params.Status, productcatalog.AddonStatusActive) {
				predicates = append(predicates, addondb.And(
					addondb.EffectiveFromLTE(now),
					addondb.Or(
						addondb.EffectiveToGTE(now),
						addondb.EffectiveToIsNil(),
					),
				))
			}

			if slices.Contains(params.Status, productcatalog.AddonStatusDraft) {
				predicates = append(predicates, addondb.And(
					addondb.EffectiveFromIsNil(),
					addondb.EffectiveToIsNil(),
				))
			}

			if slices.Contains(params.Status, productcatalog.AddonStatusArchived) {
				predicates = append(predicates, addondb.EffectiveToLT(now))
			}

			if slices.Contains(params.Status, productcatalog.AddonStatusInvalid) {
				predicates = append(predicates, func(s *sql.Selector) {
					s.Where(sql.ColumnsLT(addondb.FieldEffectiveTo, addondb.FieldEffectiveFrom))
				})
			}

			if len(predicates) > 0 {
				query = query.Where(addondb.Or(predicates...))
			}
		}

		// Eager load ratecards
		query = query.WithRatecards(
			AddonEagerLoadRateCardsFn,
		)

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case addon.OrderByCreatedAt:
			query = query.Order(addondb.ByCreatedAt(order...))
		case addon.OrderByUpdatedAt:
			query = query.Order(addondb.ByUpdatedAt(order...))
		case addon.OrderByVersion:
			query = query.Order(addondb.ByVersion(order...))
		case addon.OrderByKey:
			query = query.Order(addondb.ByVersion(order...))
		case addon.OrderByID:
			fallthrough
		default:
			query = query.Order(addondb.ByID(order...))
		}

		response := pagination.Result[addon.Addon]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, fmt.Errorf("failed to list add-ons: %w", err)
		}

		result := make([]addon.Addon, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil add-on received")
				continue
			}

			p, err := FromAddonRow(*item)
			if err != nil {
				return response, fmt.Errorf("failed to cast add-on: %w", err)
			}

			result = append(result, *p)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo[pagination.Result[addon.Addon], *adapter](ctx, a, fn)
}

func (a *adapter) CreateAddon(ctx context.Context, params addon.CreateAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context, a *adapter) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create add-on parameters: %w", err)
		}

		if params.Version == 0 {
			params.Version = 1
		}

		addonRow, err := a.db.Addon.Create().
			SetKey(params.Key).
			SetNamespace(params.Namespace).
			SetName(params.Name).
			SetNillableDescription(params.Description).
			SetCurrency(params.Currency.String()).
			SetMetadata(params.Metadata).
			SetVersion(params.Version).
			SetAnnotations(params.Annotations).
			SetInstanceType(params.InstanceType).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create add-on [namespace=%s]: %w", params.Namespace, err)
		}

		if addonRow == nil {
			return nil, fmt.Errorf("invalid query result: nil add-on received [namespace=%s]", params.Namespace)
		}

		if len(params.RateCards) > 0 {
			bulk, err := rateCardBulkCreate(a.db.AddonRateCard, params.RateCards, addonRow.ID, params.Namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to bulk create ratecards [namespace=%s id:%s]: %w", params.Namespace, addonRow.ID, err)
			}

			if err = a.db.AddonRateCard.CreateBulk(bulk...).Exec(ctx); err != nil {
				return nil, fmt.Errorf("failed to bulk create ratecards [namespace=%s id:%s]: %w", params.Namespace, addonRow.ID, err)
			}
		}

		// Refetch newly created addon
		addonRow, err = a.db.Addon.Query().
			Where(addondb.And(
				addondb.Namespace(params.Namespace),
				addondb.ID(addonRow.ID)),
			).
			WithRatecards(
				AddonEagerLoadRateCardsFn,
			).
			First(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create add-on [namespace=%s]: %w", params.Namespace, err)
		}

		add, err := FromAddonRow(*addonRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast add-on [namespace=%s id:%s]: %w", params.Namespace, addonRow.ID, err)
		}

		return add, nil
	}

	return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)
}

func rateCardBulkCreate(c *entdb.AddonRateCardClient, rateCards productcatalog.RateCards, addonID string, ns string) ([]*entdb.AddonRateCardCreate, error) {
	bulk := make([]*entdb.AddonRateCardCreate, 0, len(rateCards))

	for _, rateCard := range rateCards {
		rateCardEntity, err := asAddonRateCardRow(rateCard)
		if err != nil {
			return nil, fmt.Errorf("failed to cast ratecard to db entity: %w", err)
		}

		q := c.Create().
			SetAddonID(addonID).
			SetNamespace(ns).
			SetKey(rateCardEntity.Key).
			SetType(rateCardEntity.Type).
			SetName(rateCardEntity.Name).
			SetNillableDescription(rateCardEntity.Description).
			SetMetadata(rateCardEntity.Metadata).
			SetNillableFeatureKey(rateCardEntity.FeatureKey).
			SetNillableFeaturesID(rateCardEntity.FeatureID).
			SetEntitlementTemplate(rateCardEntity.EntitlementTemplate).
			SetNillableBillingCadence(rateCardEntity.BillingCadence).
			SetDiscounts(rateCardEntity.Discounts)

		if rateCardEntity.TaxConfig != nil {
			q.SetTaxConfig(rateCardEntity.TaxConfig)
		}

		if rateCardEntity.Price != nil {
			q.SetPrice(rateCardEntity.Price)
		}

		bulk = append(bulk, q)
	}

	return bulk, nil
}

func (a *adapter) DeleteAddon(ctx context.Context, params addon.DeleteAddonInput) error {
	fn := func(ctx context.Context, a *adapter) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete add-on parameters: %w", err)
		}

		add, err := a.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{
					Namespace: params.Namespace,
					ID:        params.ID,
				})
			}

			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		deletedAt := time.Now().UTC()
		err = a.db.Addon.UpdateOneID(add.ID).
			Where(addondb.Namespace(add.Namespace)).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{
					Namespace: params.Namespace,
					ID:        params.ID,
				})
			}

			return nil, fmt.Errorf("failed to delete add-on: %w", err)
		}

		return nil, nil
	}

	_, resp := entutils.TransactingRepo[interface{}, *adapter](ctx, a, fn)

	return resp
}

func (a *adapter) GetAddon(ctx context.Context, params addon.GetAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context, a *adapter) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get add-on parameters: %w", err)
		}

		query := a.db.Addon.Query()

		if params.ID != "" { // get Addon by ID
			query = query.Where(addondb.And(
				addondb.Namespace(params.Namespace),
				addondb.ID(params.ID)),
			)
		} else if params.Key != "" {
			if params.Version == 0 {
				if params.IncludeLatest { // get add-ons latest version by Key
					//
					//	SELECT *
					//	FROM Addons WHERE (namespace, key, version) IN
					//	(SELECT namespace, key, MAX(version)
					//		FROM addons
					//		WHERE namespace = '$1' and key = '$2'
					//		GROUP BY (namespace, key)
					//	)
					query = query.Where(func(s *sql.Selector) {
						t := sql.Table(addondb.Table)
						s.Where(
							sql.In(
								t.Wrap(func(b *sql.Builder) {
									b.IdentComma(addondb.FieldNamespace, addondb.FieldKey, addondb.FieldVersion)
								}).String(),
								sql.Select(
									addondb.FieldNamespace, addondb.FieldKey, sql.Max(addondb.FieldVersion),
								).
									From(t).
									Where(sql.And(
										sql.EQ(addondb.FieldNamespace, params.Namespace),
										sql.EQ(addondb.FieldKey, params.Key),
									)).
									GroupBy(addondb.FieldNamespace, addondb.FieldKey),
							),
						)
					})
				} else { // get add-on in active with active status by Key
					now := time.Now().UTC()
					query = query.Where(addondb.And(
						addondb.Namespace(params.Namespace),
						addondb.Key(params.Key),
						addondb.EffectiveFromLTE(now),
						addondb.Or(
							addondb.EffectiveToGT(now),
							addondb.EffectiveToIsNil(),
						),
						addondb.DeletedAtIsNil(),
					))
				}
			} else { // get add-on by Key and Version
				query = query.Where(addondb.And(
					addondb.Namespace(params.Namespace),
					addondb.Key(params.Key),
					addondb.Version(params.Version),
				))
			}
		}

		// Eager load RateCards
		query = query.WithRatecards(
			AddonEagerLoadRateCardsFn,
		)

		if params.Expand.PlanAddons {
			query = query.WithPlans(
				addonEagerLoadActivePlans,
			)
		}

		addonRow, err := query.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{
					Namespace: params.Namespace,
					ID:        params.ID,
					Key:       params.Key,
					Version:   params.Version,
				})
			}

			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		if addonRow == nil {
			return nil, fmt.Errorf("invalid query result: nil add-on received")
		}

		add, err := FromAddonRow(*addonRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast add-on: %w", err)
		}

		return add, nil
	}

	return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)
}

var addonEagerLoadActivePlans = func(paq *entdb.PlanAddonQuery) {
	paq.Where(
		planaddondb.Or(
			planaddondb.DeletedAtIsNil(),
			planaddondb.DeletedAtGT(clock.Now().UTC()),
		),
	).WithPlan(func(pq *entdb.PlanQuery) {
		pq.WithPhases(func(ppq *entdb.PlanPhaseQuery) {
			ppq.Where(
				phasedb.Or(
					phasedb.DeletedAtIsNil(),
					phasedb.DeletedAtGT(clock.Now().UTC()),
				),
			).WithRatecards(func(prcq *entdb.PlanRateCardQuery) {
				prcq.Where(
					ratecarddb.Or(
						ratecarddb.DeletedAtIsNil(),
						ratecarddb.DeletedAtGT(clock.Now().UTC()),
					),
				).WithFeatures()
			})
		})
	})
}

func (a *adapter) UpdateAddon(ctx context.Context, params addon.UpdateAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context, a *adapter) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update add-on parameters: %w", err)
		}

		add, err := a.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		if !params.Equal(*add) {
			query := a.db.Addon.UpdateOneID(add.ID).
				Where(addondb.Namespace(params.Namespace)).
				SetNillableName(params.Name).
				SetNillableDescription(params.Description).
				SetNillableEffectiveFrom(params.EffectiveFrom).
				SetNillableEffectiveTo(params.EffectiveTo)

			if params.Metadata != nil {
				query = query.SetMetadata(*params.Metadata)
			}

			if params.Annotations != nil {
				query = query.SetAnnotations(*params.Annotations)
			}

			if params.InstanceType != nil {
				query = query.SetInstanceType(*params.InstanceType)
			}

			err = query.Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update add-on: %w", err)
			}

			// Addon needs to be refetched after updated in order to populate all subresources
			add, err = a.GetAddon(ctx, addon.GetAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			})
			if err != nil {
				if entdb.IsNotFound(err) {
					return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{
						Namespace: params.Namespace,
						ID:        params.ID,
					})
				}

				return nil, fmt.Errorf("failed to get updated add-on: %w", err)
			}
		}

		// Return early if there are no updates for AddonPhases
		if params.RateCards == nil {
			return add, nil
		}

		// Delete all existing ratecards
		_, err = a.db.AddonRateCard.Delete().
			Where(addonratecarddb.AddonID(add.ID)).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to delete add-on ratecards: %w", err)
		}

		if len(*params.RateCards) > 0 {
			bulk, err := rateCardBulkCreate(a.db.AddonRateCard, *params.RateCards, add.ID, params.Namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to bulk create ratecards [namespace=%s id:%s]: %w", params.Namespace, add.ID, err)
			}

			if err = a.db.AddonRateCard.CreateBulk(bulk...).Exec(ctx); err != nil {
				return nil, fmt.Errorf("failed to bulk create ratecards [namespace=%s id:%s]: %w", params.Namespace, add.ID, err)
			}
		}

		// Refetch updated add-on
		addonRow, err := a.db.Addon.Query().
			Where(addondb.And(
				addondb.Namespace(params.Namespace),
				addondb.ID(add.ID)),
			).
			WithRatecards(
				AddonEagerLoadRateCardsFn,
			).
			First(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update add-on [namespace=%s]: %w", params.Namespace, err)
		}

		add, err = FromAddonRow(*addonRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast updated add-on [namespace=%s id:%s]: %w", params.Namespace, addonRow.ID, err)
		}

		return add, nil
	}

	return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)
}

var AddonEagerLoadRateCardsFn = func(q *entdb.AddonRateCardQuery) {
	q.Where(addonratecarddb.Or(
		addonratecarddb.DeletedAtIsNil(),
		addonratecarddb.DeletedAtGT(clock.Now().UTC()),
	)).WithFeatures()
}
