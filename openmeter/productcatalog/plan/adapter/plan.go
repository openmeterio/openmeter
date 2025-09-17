package adapter

import (
	"context"
	"fmt"
	"slices"

	"entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	plandb "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	planaddondb "github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
	phasedb "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	ratecarddb "github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.Result[plan.Plan], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[plan.Plan], error) {
		if err := params.Validate(); err != nil {
			return pagination.Result[plan.Plan]{}, fmt.Errorf("invalid list Plans parameters: %w", err)
		}

		query := a.db.Plan.Query()

		if len(params.Namespaces) > 0 {
			query = query.Where(plandb.NamespaceIn(params.Namespaces...))
		}

		var orFilters []predicate.Plan
		if len(params.IDs) > 0 {
			orFilters = append(orFilters, plandb.IDIn(params.IDs...))
		}

		if len(params.Keys) > 0 {
			orFilters = append(orFilters, plandb.KeyIn(params.Keys...))
		}

		if len(params.KeyVersions) > 0 {
			for key, version := range params.KeyVersions {
				orFilters = append(orFilters, plandb.And(plandb.Key(key), plandb.VersionIn(version...)))
			}
		}

		if len(params.Currencies) > 0 {
			orFilters = append(orFilters, plandb.CurrencyIn(params.Currencies...))
		}

		query = query.Where(plandb.Or(orFilters...))

		if !params.IncludeDeleted {
			query = query.Where(plandb.DeletedAtIsNil())
		}

		if len(params.Status) > 0 {
			var predicates []predicate.Plan

			now := clock.Now().UTC()

			if slices.Contains(params.Status, productcatalog.PlanStatusActive) {
				predicates = append(predicates, plandb.And(
					plandb.EffectiveFromLTE(now),
					plandb.Or(
						plandb.EffectiveToGTE(now),
						plandb.EffectiveToIsNil(),
					),
				))
			}

			if slices.Contains(params.Status, productcatalog.PlanStatusDraft) {
				predicates = append(predicates, plandb.And(
					plandb.EffectiveFromIsNil(),
					plandb.EffectiveToIsNil(),
				))
			}

			if slices.Contains(params.Status, productcatalog.PlanStatusScheduled) {
				predicates = append(predicates, plandb.And(
					plandb.Or(
						plandb.EffectiveFromGT(now),
					),
				))
			}

			if slices.Contains(params.Status, productcatalog.PlanStatusArchived) {
				predicates = append(predicates, plandb.EffectiveToLT(now))
			}

			if slices.Contains(params.Status, productcatalog.PlanStatusInvalid) {
				predicates = append(predicates, func(s *sql.Selector) {
					s.Where(sql.ColumnsLT(plandb.FieldEffectiveTo, plandb.FieldEffectiveFrom))
				})
			}

			query = query.Where(plandb.Or(predicates...))
		}

		// Eager load phases with
		// * ordering by StartAfter
		// * with eager load RateCards
		query = query.WithPhases(
			planPhaseIncludeDeleted(false),
			planPhaseEagerLoadRateCardsFn,
		)

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case plan.OrderByCreatedAt:
			query = query.Order(plandb.ByCreatedAt(order...))
		case plan.OrderByUpdatedAt:
			query = query.Order(plandb.ByUpdatedAt(order...))
		case plan.OrderByVersion:
			query = query.Order(plandb.ByVersion(order...))
		case plan.OrderByKey:
			query = query.Order(plandb.ByVersion(order...))
		case plan.OrderByID:
			fallthrough
		default:
			query = query.Order(plandb.ByID(order...))
		}

		response := pagination.Result[plan.Plan]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, fmt.Errorf("failed to list Plans: %w", err)
		}

		result := make([]plan.Plan, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil Plan received")
				continue
			}

			p, err := FromPlanRow(*item)
			if err != nil {
				return response, fmt.Errorf("failed to cast Plan: %w", err)
			}

			result = append(result, *p)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo[pagination.Result[plan.Plan], *adapter](ctx, a, fn)
}

func (a *adapter) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create Plan parameters: %w", err)
		}

		if params.Version == 0 {
			params.Version = 1
		}

		planRow, err := a.db.Plan.Create().
			SetKey(params.Key).
			SetNamespace(params.Namespace).
			SetName(params.Name).
			SetNillableDescription(params.Description).
			SetCurrency(params.Currency.String()).
			SetBillingCadence(params.BillingCadence.ISOString()).
			SetProRatingConfig(params.ProRatingConfig).
			SetMetadata(params.Metadata).
			SetVersion(params.Version).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create Plan: %w", err)
		}

		if planRow == nil {
			return nil, fmt.Errorf("invalid query result: nil Plan received")
		}

		p, err := FromPlanRow(*planRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast Plan: %w", err)
		}

		if len(params.Phases) > 0 {
			p.Phases = make([]plan.Phase, len(params.Phases))
			for idx, phase := range params.Phases {
				planPhase, err := a.createPhase(ctx, createPhaseInput{
					Index: idx,
					NamespacedModel: models.NamespacedModel{
						Namespace: params.Namespace,
					},
					PlanID: p.ID,
					Phase: productcatalog.Phase{
						PhaseMeta: productcatalog.PhaseMeta{
							Key:         phase.Key,
							Name:        phase.Name,
							Description: phase.Description,
							Metadata:    phase.Metadata,
							Duration:    phase.Duration,
						},
						RateCards: phase.RateCards,
					},
				})
				if err != nil {
					return nil, fmt.Errorf("failed to create PlanPhase for Plan: %w", err)
				}

				p.Phases[idx] = *planPhase
			}
		}

		return p, nil
	}

	return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, fn)
}

func (a *adapter) DeletePlan(ctx context.Context, params plan.DeletePlanInput) error {
	fn := func(ctx context.Context, a *adapter) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete Plan parameters: %w", err)
		}

		p, err := a.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{
					Namespace: params.Namespace,
					ID:        params.ID,
				})
			}

			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		deletedAt := clock.Now().UTC()
		err = a.db.Plan.UpdateOneID(p.ID).
			Where(plandb.Namespace(p.Namespace)).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{
					Namespace: params.Namespace,
					ID:        params.ID,
				})
			}

			return nil, fmt.Errorf("failed to delete Plan: %w", err)
		}

		// NOTE(chrisgacsal): do not mark Phases as deleted as fetching a deleted Plan will return 0 Phases
		// making it impossible to get the state of the Plan before deletion.
		//
		//for _, phase := range p.Phases {
		//	err = a.DeletePhase(ctx, plan.DeletePhaseInput{
		//		NamespacedID: models.NamespacedID{
		//			Namespace: params.Namespace,
		//			ID:        phase.ID,
		//		},
		//	})
		//	if err != nil {
		//		return nil, fmt.Errorf("failed to delete PlanPhase for plan: %w", err)
		//	}
		//}

		return nil, nil
	}

	_, resp := entutils.TransactingRepo[interface{}, *adapter](ctx, a, fn)

	return resp
}

func (a *adapter) GetPlan(ctx context.Context, params plan.GetPlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get Plan parameters: %w", err)
		}

		query := a.db.Plan.Query()

		if params.ID != "" { // get Plan by ID
			query = query.Where(plandb.And(
				plandb.Namespace(params.Namespace),
				plandb.ID(params.ID)),
			)
		} else if params.Key != "" {
			if params.Version == 0 {
				if params.IncludeLatest { // get Plan's latest version by Key
					//
					//	SELECT *
					//	FROM plans WHERE (namespace, key, version) IN
					//	(SELECT namespace, key, MAX(version)
					//		FROM plans
					//		WHERE namespace = '$1' and key = '$2'
					//		GROUP BY (namespace, key)
					//	)
					query = query.Where(func(s *sql.Selector) {
						t := sql.Table(plandb.Table)
						s.Where(
							sql.In(
								t.Wrap(func(b *sql.Builder) {
									b.IdentComma(plandb.FieldNamespace, plandb.FieldKey, plandb.FieldVersion)
								}).String(),
								sql.Select(
									plandb.FieldNamespace, plandb.FieldKey, sql.Max(plandb.FieldVersion),
								).
									From(t).
									Where(sql.And(
										sql.EQ(plandb.FieldNamespace, params.Namespace),
										sql.EQ(plandb.FieldKey, params.Key),
									)).
									GroupBy(plandb.FieldNamespace, plandb.FieldKey),
							),
						)
					})
				} else { // get Plan in active with active status by Key
					now := clock.Now().UTC()
					query = query.Where(plandb.And(
						plandb.Namespace(params.Namespace),
						plandb.Key(params.Key),
						plandb.EffectiveFromLTE(now),
						plandb.Or(
							plandb.EffectiveToGT(now),
							plandb.EffectiveToIsNil(),
						),
						plandb.DeletedAtIsNil(),
					))
				}
			} else { // get Plan by Key and Version
				query = query.Where(plandb.And(
					plandb.Namespace(params.Namespace),
					plandb.Key(params.Key),
					plandb.Version(params.Version),
				))
			}
		}

		// Eager load phases with
		// * ordering by StartAfter
		// * with eager load RateCards
		query = query.WithPhases(
			planPhaseIncludeDeleted(false),
			planPhaseEagerLoadRateCardsFn,
		)

		if params.Expand.PlanAddons {
			query = query.WithAddons(
				planEagerLoadActiveAddons,
			)
		}

		planRow, err := query.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{
					Namespace: params.Namespace,
					ID:        params.ID,
					Key:       params.Key,
					Version:   params.Version,
				})
			}

			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		if planRow == nil {
			return nil, fmt.Errorf("invalid query result: nil Plan received")
		}

		p, err := FromPlanRow(*planRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast Plan: %w", err)
		}

		return p, nil
	}

	return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, fn)
}

func (a *adapter) UpdatePlan(ctx context.Context, params plan.UpdatePlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update Plan parameters: %w", err)
		}

		p, err := a.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		if !params.Equal(*p) {
			query := a.db.Plan.UpdateOneID(p.ID).
				Where(plandb.Namespace(params.Namespace)).
				SetNillableName(params.Name).
				SetNillableDescription(params.Description).
				SetNillableEffectiveFrom(params.EffectiveFrom).
				SetNillableEffectiveTo(params.EffectiveTo).
				SetNillableBillingCadence(params.BillingCadence.ISOStringPtrOrNil()).
				SetNillableProRatingConfig(params.ProRatingConfig)

			if params.Metadata != nil {
				query = query.SetMetadata(*params.Metadata)
			}

			err = query.Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update Plan: %w", err)
			}

			// Plan needs to be refetched after updated in order to populate all subresources
			p, err = a.GetPlan(ctx, plan.GetPlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			})
			if err != nil {
				if entdb.IsNotFound(err) {
					return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{
						Namespace: params.Namespace,
						ID:        params.ID,
					})
				}

				return nil, fmt.Errorf("failed to get updated Plan: %w", err)
			}
		}

		// Return early if there are no updates for PlanPhases
		if params.Phases == nil {
			return p, nil
		}

		// Delete all existing PlanPhases
		_, err = a.db.PlanPhase.Delete().Where(phasedb.PlanID(p.ID)).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to delete PlanPhases: %w", err)
		}

		// Create new PlanPhases
		phases := make([]plan.Phase, len(*params.Phases))
		for idx, phase := range *params.Phases {
			planPhase, err := a.createPhase(ctx, createPhaseInput{
				Index: idx,
				NamespacedModel: models.NamespacedModel{
					Namespace: params.Namespace,
				},
				PlanID: p.ID,
				Phase:  phase,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create PlanPhase: %w", err)
			}

			phases[idx] = *planPhase
		}
		p.Phases = phases

		return p, nil
	}

	return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, fn)
}

func planPhaseIncludeDeleted(include bool) func(*entdb.PlanPhaseQuery) {
	if include {
		return func(q *entdb.PlanPhaseQuery) {}
	} else {
		return func(q *entdb.PlanPhaseQuery) {
			q.Where(phasedb.Or(phasedb.DeletedAtIsNil(), phasedb.DeletedAtGT(clock.Now().UTC())))
		}
	}
}

var planEagerLoadActiveAddons = func(paq *entdb.PlanAddonQuery) {
	paq.Where(
		planaddondb.Or(
			planaddondb.DeletedAtIsNil(),
			planaddondb.DeletedAtGT(clock.Now().UTC()),
		),
	).WithAddon(func(aq *entdb.AddonQuery) {
		aq.WithRatecards(func(arq *entdb.AddonRateCardQuery) {
			arq.WithFeatures()
		})
	})
}

var planPhaseEagerLoadRateCardsFn = func(q *entdb.PlanPhaseQuery) {
	q.WithRatecards(func(prcq *entdb.PlanRateCardQuery) {
		prcq.Where(ratecarddb.Or(ratecarddb.DeletedAtIsNil(), ratecarddb.DeletedAtGT(clock.Now().UTC())))
		rateCardEagerLoadFeaturesFn(prcq)
	})
}

var rateCardEagerLoadFeaturesFn = func(q *entdb.PlanRateCardQuery) {
	q.WithFeatures()
}
