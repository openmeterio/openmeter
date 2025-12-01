package adapter

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	addondb "github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	plandb "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	planaddondb "github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListPlanAddons(ctx context.Context, params planaddon.ListPlanAddonsInput) (pagination.Result[planaddon.PlanAddon], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[planaddon.PlanAddon], error) {
		if err := params.Validate(); err != nil {
			return pagination.Result[planaddon.PlanAddon]{}, fmt.Errorf("invalid list add-on assignments parameters: %w", err)
		}

		query := a.db.PlanAddon.Query()

		if len(params.Namespaces) > 0 {
			query = query.Where(planaddondb.NamespaceIn(params.Namespaces...))
		}

		var orFilters []predicate.PlanAddon
		if len(params.IDs) > 0 {
			orFilters = append(orFilters, planaddondb.IDIn(params.IDs...))
		}

		// Plan predicates

		var planOrFilters []predicate.Plan

		if len(params.PlanIDs) > 0 {
			planOrFilters = append(planOrFilters, plandb.IDIn(params.PlanIDs...))
		}

		if len(params.PlanKeys) > 0 {
			planOrFilters = append(planOrFilters, plandb.KeyIn(params.PlanKeys...))
		}

		if len(params.PlanKeyVersions) > 0 {
			var planKeyVersionFilters []predicate.Plan

			for key, version := range params.PlanKeyVersions {
				planOrFilters = append(planOrFilters, plandb.And(plandb.Key(key), plandb.VersionIn(version...)))
			}

			planOrFilters = append(planOrFilters, plandb.Or(planKeyVersionFilters...))
		}

		if len(params.Currencies) > 0 {
			planOrFilters = append(planOrFilters, plandb.CurrencyIn(params.Currencies...))
		}

		if len(planOrFilters) > 0 {
			orFilters = append(orFilters, planaddondb.HasPlanWith(
				plandb.Or(planOrFilters...),
			))
		}

		// Addon predicates

		var addonOrFilters []predicate.Addon

		if len(params.AddonIDs) > 0 {
			addonOrFilters = append(addonOrFilters, addondb.IDIn(params.AddonIDs...))
		}

		if len(params.AddonKeys) > 0 {
			addonOrFilters = append(addonOrFilters, addondb.KeyIn(params.AddonKeys...))
		}

		if len(params.PlanKeyVersions) > 0 {
			var planKeyVersionFilters []predicate.Addon

			for key, version := range params.PlanKeyVersions {
				addonOrFilters = append(addonOrFilters, addondb.And(addondb.Key(key), addondb.VersionIn(version...)))
			}

			addonOrFilters = append(addonOrFilters, addondb.Or(planKeyVersionFilters...))
		}

		if len(params.Currencies) > 0 {
			addonOrFilters = append(addonOrFilters, addondb.CurrencyIn(params.Currencies...))
		}

		if len(addonOrFilters) > 0 {
			orFilters = append(orFilters, planaddondb.HasAddonWith(
				addondb.Or(addonOrFilters...),
			))
		}

		if len(orFilters) > 0 {
			query = query.Where(planaddondb.Or(orFilters...))
		}

		if !params.IncludeDeleted {
			query = query.Where(planaddondb.DeletedAtIsNil())
		}

		// Eager load Plans and Addons
		query = query.
			WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).
			WithAddon(AddonEagerLoadRateCardsWithFeaturesFn)

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case planaddon.OrderByCreatedAt:
			query = query.Order(planaddondb.ByCreatedAt(order...))
		case planaddon.OrderByUpdatedAt:
			query = query.Order(planaddondb.ByUpdatedAt(order...))
		case planaddon.OrderByID:
			fallthrough
		default:
			query = query.Order(planaddondb.ByID(order...))
		}

		response := pagination.Result[planaddon.PlanAddon]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, fmt.Errorf("failed to list plan add-on assignments: %w", err)
		}

		result := make([]planaddon.PlanAddon, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil plan add-on assignments received")
				continue
			}

			planAddon, err := FromPlanAddonRow(*item)
			if err != nil {
				return response, fmt.Errorf("failed to cast add-on: %w", err)
			}

			result = append(result, *planAddon)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo[pagination.Result[planaddon.PlanAddon], *adapter](ctx, a, fn)
}

func (a *adapter) CreatePlanAddon(ctx context.Context, params planaddon.CreatePlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context, a *adapter) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create plan add-on assignment parameters [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		planAddonRow, err := a.db.PlanAddon.Create().
			SetNamespace(params.Namespace).
			SetPlanID(params.PlanID).
			SetAddonID(params.AddonID).
			SetAnnotations(params.Annotations).
			SetFromPlanPhase(params.FromPlanPhase).
			SetNillableMaxQuantity(params.MaxQuantity).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		if planAddonRow == nil {
			return nil, fmt.Errorf("invalid query result: nil plan add-on assignment received [namespace=%s plan.id=%s addon.id=%s]",
				params.Namespace, params.PlanID, params.AddonID)
		}

		// Refetch newly created addon
		planAddonRow, err = a.db.PlanAddon.Query().
			Where(planaddondb.And(
				planaddondb.Namespace(params.Namespace),
				planaddondb.ID(planAddonRow.ID)),
			).
			WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).
			WithAddon(AddonEagerLoadRateCardsWithFeaturesFn).
			First(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		planAddon, err := FromPlanAddonRow(*planAddonRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		return planAddon, nil
	}

	return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)
}

var PlanEagerLoadPhasesWithRateCardsWithFeaturesFn = func(pq *entdb.PlanQuery) {
	pq.WithPhases(func(ppq *entdb.PlanPhaseQuery) {
		ppq.WithRatecards(func(prq *entdb.PlanRateCardQuery) {
			prq.WithFeatures()
		})
	})
}

var AddonEagerLoadRateCardsWithFeaturesFn = func(aq *entdb.AddonQuery) {
	aq.WithRatecards(func(arq *entdb.AddonRateCardQuery) {
		arq.WithFeatures()
	})
}

func (a *adapter) DeletePlanAddon(ctx context.Context, params planaddon.DeletePlanAddonInput) error {
	fn := func(ctx context.Context, a *adapter) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete plan add-on assignment parameters: %w", err)
		}

		planAddon, err := a.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			ID:           params.ID,
			PlanIDOrKey:  params.PlanID,
			AddonIDOrKey: params.AddonID,
		})
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{
					Namespace:    params.Namespace,
					ID:           params.ID,
					PlanIDOrKey:  params.PlanID,
					AddonIDOrKey: params.AddonID,
				})
			}

			return nil, fmt.Errorf("failed to get plan add-on assignment: %w", err)
		}

		deletedAt := clock.Now().UTC()
		err = a.db.PlanAddon.UpdateOneID(planAddon.ID).
			Where(planaddondb.Namespace(planAddon.Namespace)).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{
					Namespace:    params.Namespace,
					ID:           planAddon.ID,
					PlanIDOrKey:  params.PlanID,
					AddonIDOrKey: params.AddonID,
				})
			}

			return nil, fmt.Errorf("failed to delete plan add-on assignment [namespace=%s planaddon.id=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, planAddon.ID, params.PlanID, params.AddonID, err)
		}

		return nil, nil
	}

	_, resp := entutils.TransactingRepo[interface{}, *adapter](ctx, a, fn)

	return resp
}

func (a *adapter) GetPlanAddon(ctx context.Context, params planaddon.GetPlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context, a *adapter) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get add-on parameters: %w", err)
		}

		query := a.db.PlanAddon.Query()

		if params.ID != "" { // get plan add-on assignment by ID
			query = query.Where(planaddondb.And(
				planaddondb.Namespace(params.Namespace),
				planaddondb.ID(params.ID)),
			)
		} else {
			query = query.Where(planaddondb.And(
				planaddondb.Namespace(params.Namespace),
				planaddondb.HasPlanWith(plandb.Or(plandb.ID(params.PlanIDOrKey), plandb.Key(params.PlanIDOrKey))),
				planaddondb.HasAddonWith(addondb.Or(addondb.ID(params.AddonIDOrKey), addondb.Key(params.AddonIDOrKey))),
				planaddondb.DeletedAtIsNil(),
			))
		}

		// Eager load Plan and Addon
		query = query.
			WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).
			WithAddon(AddonEagerLoadRateCardsWithFeaturesFn)

		planAddonRow, err := query.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{
					Namespace:    params.Namespace,
					ID:           params.ID,
					PlanIDOrKey:  params.PlanIDOrKey,
					AddonIDOrKey: params.AddonIDOrKey,
				})
			}

			return nil, fmt.Errorf("failed to get plan add-on assignment [namespace=%s planaddon.id=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.ID, params.PlanIDOrKey, params.AddonIDOrKey, err)
		}

		if planAddonRow == nil {
			return nil, fmt.Errorf("invalid query result: nil plan add-on assignments received")
		}

		planAddon, err := FromPlanAddonRow(*planAddonRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast plan add-on assignment [namespace=%s planaddon.id=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.ID, params.PlanIDOrKey, params.AddonIDOrKey, err)
		}

		return planAddon, nil
	}

	return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)
}

func (a *adapter) UpdatePlanAddon(ctx context.Context, params planaddon.UpdatePlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context, a *adapter) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update add-on parameters: %w", err)
		}

		planAddon, err := a.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			ID:           params.ID,
			PlanIDOrKey:  params.PlanID,
			AddonIDOrKey: params.AddonID,
		})
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{
					Namespace:    params.Namespace,
					PlanIDOrKey:  params.PlanID,
					AddonIDOrKey: params.AddonID,
				})
			}

			return nil, fmt.Errorf("failed to get plan add-on assignment for update: %w", err)
		}

		if !params.Equal(*planAddon) {
			query := a.db.PlanAddon.UpdateOneID(planAddon.ID).
				Where(planaddondb.Namespace(params.Namespace)).
				SetOrClearMaxQuantity(params.MaxQuantity)

			if params.FromPlanPhase != nil {
				query = query.SetFromPlanPhase(*params.FromPlanPhase)
			}

			if params.Annotations != nil {
				query = query.SetAnnotations(*params.Annotations)
			}

			if params.Metadata != nil {
				query = query.SetMetadata(*params.Metadata)
			}

			err = query.Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update add-on: %w", err)
			}
		}

		// Plan add-on assignment needs to be re-fetched after updated in order to populate all sub-resources
		planAddon, err = a.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			ID: planAddon.ID,
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

		return planAddon, nil
	}

	return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)
}
