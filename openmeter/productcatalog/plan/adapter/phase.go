package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	phasedb "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	ratecarddb "github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListPhases(ctx context.Context, params plan.ListPhasesInput) (pagination.PagedResponse[plan.Phase], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.PagedResponse[plan.Phase], error) {
		if err := params.Validate(); err != nil {
			return pagination.PagedResponse[plan.Phase]{}, fmt.Errorf("invalid list PlanPhases parameters: %w", err)
		}

		query := a.db.PlanPhase.Query()

		if len(params.Namespaces) > 0 {
			query = query.Where(phasedb.NamespaceIn(params.Namespaces...))
		}

		var orFilters []predicate.PlanPhase
		if len(params.IDs) > 0 {
			orFilters = append(orFilters, phasedb.IDIn(params.IDs...))
		}

		if len(params.Keys) > 0 {
			orFilters = append(orFilters, phasedb.KeyIn(params.Keys...))
		}

		query = query.Where(phasedb.Or(orFilters...))

		if len(params.PlanIDs) > 0 {
			query = query.Where(phasedb.PlanIDIn(params.PlanIDs...))
		}

		if !params.IncludeDeleted {
			query = query.Where(phasedb.DeletedAtIsNil())
		}

		// Eager load phases with
		// * with eager load RateCards
		query = query.WithRatecards()

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		query = query.Order(phasedb.ByPlanID(order...))

		switch params.OrderBy {
		case plan.OrderByCreatedAt:
			query = query.Order(phasedb.ByCreatedAt(order...))
		case plan.OrderByUpdatedAt:
			query = query.Order(phasedb.ByUpdatedAt(order...))
		case plan.OrderByKey:
			query = query.Order(phasedb.ByKey(order...))
		case plan.OrderByID:
			query = query.Order(phasedb.ByID(order...))
		case plan.OrderByStartAfter:
			fallthrough
		default:
			query = query.Order(phasedb.ByStartAfter(order...))
		}

		response := pagination.PagedResponse[plan.Phase]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, fmt.Errorf("failed to list PlanPhases: %w", err)
		}

		result := make([]plan.Phase, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil PlanPhase received")
				continue
			}

			phase, err := fromPlanPhaseRow(*item)
			if err != nil {
				return response, fmt.Errorf("failed to cast PlanPhase: %w", err)
			}

			result = append(result, *phase)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo[pagination.PagedResponse[plan.Phase], *adapter](ctx, a, fn)
}

func (a *adapter) CreatePhase(ctx context.Context, params plan.CreatePhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Phase, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create PlanPhase parameters: %w", err)
		}

		planPhaseRow, err := a.db.PlanPhase.Create().
			SetNamespace(params.Namespace).
			SetKey(params.Key).
			SetName(params.Name).
			SetNillableDescription(params.Description).
			SetMetadata(params.Metadata).
			SetStartAfter(params.StartAfter.ISOString()).
			SetPlanID(params.PlanID).
			SetDiscounts(params.Discounts).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create PlanPhase: %w", err)
		}

		if planPhaseRow == nil {
			return nil, fmt.Errorf("invalid query result: nil PlanPhase received after create")
		}

		if len(params.RateCards) > 0 {
			bulk, err := rateCardBulkCreate(a.db.PlanRateCard, params.RateCards, planPhaseRow.ID, params.Namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to bulk create RateCards: %w", err)
			}

			if err = a.db.PlanRateCard.CreateBulk(bulk...).Exec(ctx); err != nil {
				return nil, fmt.Errorf("failed to bulk create RateCards for PlanPhase %s: %w", planPhaseRow.ID, err)
			}

			planPhaseRow, err = a.db.PlanPhase.Query().
				Where(phasedb.Namespace(params.Namespace), phasedb.ID(planPhaseRow.ID)).
				WithRatecards(rateCardEagerLoadFeaturesFn).
				First(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get PlanPhase: %w", err)
			}
		}

		planPhase, err := fromPlanPhaseRow(*planPhaseRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast PlanPhase: %w", err)
		}

		return planPhase, nil
	}

	return entutils.TransactingRepo[*plan.Phase, *adapter](ctx, a, fn)
}

func rateCardBulkCreate(c *entdb.PlanRateCardClient, rateCards productcatalog.RateCards, phaseID string, ns string) ([]*entdb.PlanRateCardCreate, error) {
	bulk := make([]*entdb.PlanRateCardCreate, 0, len(rateCards))

	for _, rateCard := range rateCards {
		rateCardEntity, err := asPlanRateCardRow(rateCard)
		if err != nil {
			return nil, fmt.Errorf("failed to cast RateCard to db entity: %w", err)
		}

		q := c.Create().SetPhaseID(phaseID).
			SetNamespace(ns).
			SetKey(rateCardEntity.Key).
			SetType(rateCardEntity.Type).
			SetName(rateCardEntity.Name).
			SetNillableDescription(rateCardEntity.Description).
			SetMetadata(rateCardEntity.Metadata).
			SetNillableFeatureKey(rateCardEntity.FeatureKey).
			SetNillableFeaturesID(rateCardEntity.FeatureID).
			SetEntitlementTemplate(rateCardEntity.EntitlementTemplate).
			SetNillableBillingCadence(rateCardEntity.BillingCadence)

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

func (a *adapter) DeletePhase(ctx context.Context, params plan.DeletePhaseInput) error {
	fn := func(ctx context.Context, a *adapter) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete PlanPhase parameters: %w", err)
		}

		phase, err := a.GetPhase(ctx, plan.GetPhaseInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
			Key:    params.Key,
			PlanID: params.PlanID,
		})
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NotFoundError{
					NamespacedModel: models.NamespacedModel{
						Namespace: params.Namespace,
					},
				}
			}

			return nil, fmt.Errorf("failed to get PlanPhase: %w", err)
		}

		// Soft delete plan phase and its rate cards
		deletedAt := time.Now().UTC()
		err = a.db.PlanPhase.UpdateOneID(phase.ID).
			Where(phasedb.Namespace(phase.Namespace)).
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

			return nil, fmt.Errorf("failed to delete PlanPhase: %w", err)
		}

		err = a.db.PlanRateCard.Update().
			Where(
				ratecarddb.Namespace(phase.Namespace),
				ratecarddb.PhaseID(phase.ID),
			).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to delete RateCards: %w", err)
		}

		return nil, nil
	}

	_, err := entutils.TransactingRepo[interface{}, *adapter](ctx, a, fn)

	return err
}

func (a *adapter) GetPhase(ctx context.Context, params plan.GetPhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Phase, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get PlanPhase parameters: %w", err)
		}

		query := a.db.PlanPhase.Query()

		if params.ID != "" { // get PlanPhase by ID
			query = query.Where(
				phasedb.And(
					phasedb.Namespace(params.Namespace),
					phasedb.ID(params.ID),
				),
			)
		} else if params.Key != "" { // get PlanPhase by Key and Plan.ID
			query = query.Where(
				phasedb.And(
					phasedb.Namespace(params.Namespace),
					phasedb.Key(params.Key),
					phasedb.PlanID(params.PlanID),
				),
			)
		} else {
			return nil, errors.New("invalid get PlanPhase parameters")
		}

		query = query.WithRatecards(rateCardEagerLoadFeaturesFn)

		phaseRow, err := query.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NotFoundError{
					NamespacedModel: models.NamespacedModel{
						Namespace: params.Namespace,
					},
				}
			}

			return nil, fmt.Errorf("failed to get PlanPhase: %w", err)
		}

		if phaseRow == nil {
			return nil, fmt.Errorf("invalid query result: nil PlanPhase received")
		}

		phase, err := fromPlanPhaseRow(*phaseRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast PlanPhase: %w", err)
		}

		return phase, nil
	}

	return entutils.TransactingRepo[*plan.Phase, *adapter](ctx, a, fn)
}

func (a *adapter) UpdatePhase(ctx context.Context, params plan.UpdatePhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Phase, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update PlanPhase parameters: %w", err)
		}

		p, err := a.GetPhase(ctx, plan.GetPhaseInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
			Key:    params.Key,
			PlanID: params.PlanID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get PlanPhase: %w", err)
		}

		if !params.Equal(*p) {
			query := a.db.PlanPhase.UpdateOneID(p.ID).Where(phasedb.Namespace(params.Namespace))

			if params.Name != nil {
				query = query.SetName(*params.Name)
				p.Name = *params.Name
			}

			if params.Description != nil {
				query = query.SetDescription(*params.Description)
				p.Description = params.Description
			}

			if params.Metadata != nil {
				query = query.SetMetadata(*params.Metadata)
				p.Metadata = *params.Metadata
			}

			if params.StartAfter != nil {
				query = query.SetStartAfter(params.StartAfter.ISOString())
				p.StartAfter = *params.StartAfter
			}

			if params.Discounts != nil {
				query = query.SetDiscounts(*params.Discounts)
				p.Discounts = *params.Discounts
			}

			err = query.Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update PlanPhase: %w", err)
			}
		}

		if params.RateCards != nil {
			diffResult, err := rateCardsDiff(*params.RateCards, p.RateCards)
			if err != nil {
				return nil, fmt.Errorf("failed to generate RateCard diff for PlanPhase update: %w", err)
			}

			if !diffResult.IsDiff() {
				return p, nil
			}

			if len(diffResult.Remove) > 0 {
				for _, rateCard := range diffResult.Remove {
					rateCardEntity, err := asPlanRateCardRow(rateCard)
					if err != nil {
						return nil, fmt.Errorf("failed to cast RateCard to db entity: %w", err)
					}

					err = a.db.PlanRateCard.DeleteOneID(rateCardEntity.ID).
						Where(ratecarddb.Namespace(params.Namespace)).
						Exec(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to delete RateCard: %w", err)
					}
				}
			}

			if len(diffResult.Update) > 0 {
				for _, rateCard := range diffResult.Update {
					rateCardEntity, err := asPlanRateCardRow(rateCard)
					if err != nil {
						return nil, fmt.Errorf("failed to cast RateCard to db entity: %w", err)
					}

					q := a.db.PlanRateCard.UpdateOneID(rateCardEntity.ID).
						Where(ratecarddb.Namespace(params.Namespace)).
						SetOrClearMetadata(&rateCardEntity.Metadata).
						SetName(rateCardEntity.Name).
						SetOrClearDescription(rateCardEntity.Description).
						SetOrClearFeatureKey(rateCardEntity.FeatureKey).
						SetEntitlementTemplate(rateCardEntity.EntitlementTemplate).
						SetOrClearBillingCadence(rateCardEntity.BillingCadence)

					if rateCardEntity.TaxConfig != nil {
						q.SetTaxConfig(rateCardEntity.TaxConfig)
					}

					if rateCardEntity.Price != nil {
						q.SetPrice(rateCardEntity.Price)
					}

					if rateCardEntity.FeatureID == nil {
						q.ClearFeatureID()
					}

					err = q.Exec(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to update RateCard: %w", err)
					}
				}
			}

			if len(diffResult.Add) > 0 {
				bulk, err := rateCardBulkCreate(a.db.PlanRateCard, diffResult.Add, p.ID, params.Namespace)
				if err != nil {
					return nil, fmt.Errorf("failed to bulk create RateCards: %w", err)
				}

				if err = a.db.PlanRateCard.CreateBulk(bulk...).Exec(ctx); err != nil {
					return nil, fmt.Errorf("failed to bulk create RateCards: %w", err)
				}
			}

			p, err = a.GetPhase(ctx, plan.GetPhaseInput{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
				Key:    params.Key,
				PlanID: params.PlanID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get updated PlanPhase: %w", err)
			}
		}

		return p, nil
	}

	return entutils.TransactingRepo[*plan.Phase, *adapter](ctx, a, fn)
}

type rateCardsDiffResult struct {
	// Add defines the list of plan.CreatePhaseInput for plan.Phase objects to add
	Add productcatalog.RateCards

	// Update defines the list of plan.UpdatePhaseInput for plan.Phase objects to update
	Update productcatalog.RateCards

	// Remove defines the list of plan.DeletePhaseInput for plan.Phase identifiers to delete
	Remove productcatalog.RateCards

	// Keep defines the list of plan.Phase to keep unmodified
	Keep productcatalog.RateCards
}

func rateCardsDiff(requested, actual productcatalog.RateCards) (rateCardsDiffResult, error) {
	result := rateCardsDiffResult{}

	actualMap := make(map[string]productcatalog.RateCard, len(actual))
	for _, rc := range actual {
		actualMap[rc.Key()] = rc
	}

	actualVisited := make(map[string]struct{})
	for _, requestedRateCard := range requested {
		requestedRateCardKey := requestedRateCard.Key()

		actualRateCard, ok := actualMap[requestedRateCardKey]

		// Create RateCard
		if !ok {
			result.Add = append(result.Add, requestedRateCard)

			actualVisited[requestedRateCardKey] = struct{}{}

			continue
		}

		// Replace RateCard as type attribute is immutable for RateCards
		if requestedRateCard.Type() != actualRateCard.Type() {
			result.Add = append(result.Add, requestedRateCard)

			result.Remove = append(result.Remove, actualRateCard)

			actualVisited[requestedRateCardKey] = struct{}{}

			continue
		}

		// Update in-place
		if !actualRateCard.Equal(requestedRateCard) {
			if err := actualRateCard.Merge(requestedRateCard); err != nil {
				return result, fmt.Errorf("failed to update RateCard: %w", err)
			}

			result.Update = append(result.Update, actualRateCard)

			actualVisited[requestedRateCardKey] = struct{}{}
		} else { // Keep it as is
			result.Keep = append(result.Keep, actualRateCard)

			actualVisited[requestedRateCardKey] = struct{}{}
		}
	}

	// Collect RateCards to be deleted
	for key, actualRateCard := range actualMap {
		if _, ok := actualVisited[key]; !ok {
			result.Remove = append(result.Remove, actualRateCard)
		}
	}

	return result, nil
}

func (r rateCardsDiffResult) IsDiff() bool {
	return len(r.Add) > 0 || len(r.Update) > 0 || len(r.Remove) > 0
}
