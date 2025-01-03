package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	plandb "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	phasedb "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.PagedResponse[plan.Plan], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.PagedResponse[plan.Plan], error) {
		if err := params.Validate(); err != nil {
			return pagination.PagedResponse[plan.Plan]{}, fmt.Errorf("invalid list Plans parameters: %w", err)
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

		query = query.Where(plandb.Or(orFilters...))

		if !params.IncludeDeleted {
			query = query.Where(plandb.DeletedAtIsNil())
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

		response := pagination.PagedResponse[plan.Plan]{
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

			p, err := fromPlanRow(*item)
			if err != nil {
				return response, fmt.Errorf("failed to cast Plan: %w", err)
			}

			result = append(result, *p)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo[pagination.PagedResponse[plan.Plan], *adapter](ctx, a, fn)
}

func (a *adapter) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create Plan parameters: %w", err)
		}

		// Create plan

		if params.Version == 0 {
			params.Version = 1
		}

		planRow, err := a.db.Plan.Create().
			SetKey(params.Key).
			SetNamespace(params.Namespace).
			SetName(params.Name).
			SetNillableDescription(params.Description).
			SetCurrency(params.Currency.String()).
			SetMetadata(params.Metadata).
			SetVersion(params.Version).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create Plan: %w", err)
		}

		if planRow == nil {
			return nil, fmt.Errorf("invalid query result: nil Plan received")
		}

		p, err := fromPlanRow(*planRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast Plan: %w", err)
		}

		if len(params.Phases) > 0 {
			p.Phases = make([]plan.Phase, 0, len(params.Phases))
			for _, phase := range params.Phases {
				planPhase, err := a.createPhase(ctx, createPhaseInput{
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
							StartAfter:  phase.StartAfter,
						},
						Discounts: phase.Discounts,
						RateCards: phase.RateCards,
					},
				})
				if err != nil {
					return nil, fmt.Errorf("failed to create PlanPhase for Plan: %w", err)
				}

				p.Phases = append(p.Phases, *planPhase)
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
				return nil, plan.NotFoundError{
					NamespacedModel: models.NamespacedModel{
						Namespace: params.Namespace,
					},
				}
			}

			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		deletedAt := time.Now().UTC()
		err = a.db.Plan.UpdateOneID(p.ID).
			Where(plandb.Namespace(p.Namespace)).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NotFoundError{
					NamespacedModel: models.NamespacedModel{
						Namespace: params.Namespace,
					},
				}
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
					now := time.Now().UTC()
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
			planPhaseAscOrderingByStartAfterFn,
		)

		planRow, err := query.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NotFoundError{
					NamespacedModel: models.NamespacedModel{
						Namespace: params.Namespace,
					},
				}
			}

			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		if planRow == nil {
			return nil, fmt.Errorf("invalid query result: nil Plan received")
		}

		p, err := fromPlanRow(*planRow)
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
				SetNillableEffectiveTo(params.EffectiveTo)

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
				return nil, fmt.Errorf("failed to get updated Plan: %w", err)
			}
		}

		// Return early if there are no updates for PlanPhases
		if params.Phases == nil {
			return p, nil
		}

		// Return early if there are no changes in PlanPhases.
		diffResult := planPhasesDiff(*params.Phases, p.Phases)
		if !diffResult.IsDiff() {
			return p, nil
		}

		phases := make([]plan.Phase, 0, len(p.Phases))

		if len(diffResult.Keep) > 0 {
			phases = append(phases, diffResult.Keep...)
		}

		if len(diffResult.Remove) > 0 {
			for _, deleteInput := range diffResult.Remove {
				deleteInput.Namespace = params.Namespace
				deleteInput.PlanID = p.ID
				err = a.deletePhase(ctx, deleteInput)
				if err != nil {
					return nil, fmt.Errorf("failed to delete PlanPhase: %w", err)
				}
			}
		}

		if len(diffResult.Update) > 0 {
			for _, updateInput := range diffResult.Update {
				updateInput.Namespace = params.Namespace
				updateInput.PlanID = p.ID

				phase, err := a.updatePhase(ctx, updateInput)
				if err != nil {
					return nil, fmt.Errorf("failed to update PlanPhase: %w", err)
				}

				phases = append(phases, *phase)
			}
		}

		if len(diffResult.Add) > 0 {
			for _, createInput := range diffResult.Add {
				createInput.Namespace = params.Namespace
				createInput.PlanID = p.ID

				phase, err := a.createPhase(ctx, createInput)
				if err != nil {
					return nil, fmt.Errorf("failed to create PlanPhase: %w", err)
				}

				phases = append(phases, *phase)
			}
		}

		plan.SortPhases(p.Phases, plan.SortPhasesByStartAfter)

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
			q.Where(phasedb.DeletedAtIsNil())
		}
	}
}

var planPhaseAscOrderingByStartAfterFn = func(q *entdb.PlanPhaseQuery) {
	q.Order(phasedb.ByStartAfter(sql.OrderAsc()))
}

var planPhaseEagerLoadRateCardsFn = func(q *entdb.PlanPhaseQuery) {
	q.WithRatecards(rateCardEagerLoadFeaturesFn)
}

var rateCardEagerLoadFeaturesFn = func(q *entdb.PlanRateCardQuery) {
	q.WithFeatures()
}

type planPhasesDiffResult struct {
	// Add defines the list of plan.CreatePhaseInput for plan.Phase objects to add
	Add []createPhaseInput

	// Update defines the list of plan.UpdatePhaseInput for plan.Phase objects to update
	Update []updatePhaseInput

	// Remove defines the list of plan.DeletePhaseInput for plan.Phase identifiers to delete
	Remove []deletePhaseInput

	// Keep defines the list of plan.Phase to keep unmodified
	Keep []plan.Phase
}

func (d planPhasesDiffResult) IsDiff() bool {
	return len(d.Add) > 0 || len(d.Update) > 0 || len(d.Remove) > 0
}

func planPhasesDiff(requested []productcatalog.Phase, actual []plan.Phase) planPhasesDiffResult {
	result := planPhasesDiffResult{}

	requestedMap := make(map[string]productcatalog.Phase, len(requested))
	for _, phase := range requested {
		requestedMap[phase.Key] = phase
	}

	actualMap := make(map[string]plan.Phase, len(actual))
	for _, phase := range actual {
		actualMap[phase.Key] = phase
	}

	phasesVisited := make(map[string]struct{})
	for phaseKey, requestedPhase := range requestedMap {
		actualPhase, ok := actualMap[phaseKey]

		// Collect new phases
		if !ok {
			result.Add = append(result.Add, createPhaseInput{
				Phase: requestedPhase,
			})
			phasesVisited[phaseKey] = struct{}{}

			continue
		}

		// Collect phases to be updated
		if !requestedPhase.Equal(actualPhase.Phase) {
			result.Update = append(result.Update, updatePhaseInput{
				NamespacedID: models.NamespacedID{
					Namespace: actualPhase.Namespace,
					ID:        actualPhase.ID,
				},
				PlanID:      actualPhase.PlanID,
				Key:         actualPhase.Key,
				Name:        &requestedPhase.Name,
				Description: requestedPhase.Description,
				Metadata:    &requestedPhase.Metadata,
				StartAfter:  &requestedPhase.StartAfter,
				RateCards:   &requestedPhase.RateCards,
				Discounts:   &requestedPhase.Discounts,
			})
			phasesVisited[phaseKey] = struct{}{}

			continue
		}

		result.Keep = append(result.Keep, actualPhase)
		phasesVisited[phaseKey] = struct{}{}
	}

	// Collect phases to be deleted
	for phaseKey, actualPhase := range actualMap {
		if _, ok := phasesVisited[phaseKey]; !ok {
			result.Remove = append(result.Remove, deletePhaseInput{
				NamespacedID: models.NamespacedID{
					Namespace: actualPhase.Namespace,
					ID:        actualPhase.ID,
				},
			})
		}
	}

	return result
}
