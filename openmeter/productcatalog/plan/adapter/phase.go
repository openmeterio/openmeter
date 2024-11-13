package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	phasedb "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	ratecarddb "github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
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
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create PlanPhase: %w", err)
		}

		if planPhaseRow == nil {
			return nil, fmt.Errorf("invalid query result: nil PlanPhase received after create")
		}

		planPhase, err := fromPlanPhaseRow(*planPhaseRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast PlanPhase %w", err)
		}

		if len(params.RateCards) == 0 {
			return planPhase, nil
		}

		rateCardInputs := make([]entdb.PlanRateCard, 0, len(params.RateCards))
		for _, rateCard := range params.RateCards {
			rateCardInput, err := asPlanRateCardRow(rateCard)
			if err != nil {
				return nil, fmt.Errorf("failed to cast RateCard: %w", err)
			}

			rateCardInputs = append(rateCardInputs, rateCardInput)
		}

		bulk, bulkFn := newRateCardBulkCreate(rateCardInputs, planPhase.ID, params.Namespace)

		if err = a.db.PlanRateCard.MapCreateBulk(bulk, bulkFn).Exec(ctx); err != nil {
			return nil, fmt.Errorf("failed to bulk create RateCards for PlanPhase %s: %w", planPhase.ID, err)
		}

		planPhaseRow, err = a.db.PlanPhase.Query().
			Where(phasedb.Namespace(params.Namespace), phasedb.ID(planPhase.ID)).
			WithRatecards(rateCardEagerLoadFeaturesFn).
			First(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get PlanPhase: %w", err)
		}

		planPhase, err = fromPlanPhaseRow(*planPhaseRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast PlanPhase %w", err)
		}

		return planPhase, nil
	}

	return entutils.TransactingRepo[*plan.Phase, *adapter](ctx, a, fn)
}

func newRateCardBulkCreate(r []entdb.PlanRateCard, phaseID string, ns string) ([]entdb.PlanRateCard, func(*entdb.PlanRateCardCreate, int)) {
	return r, func(q *entdb.PlanRateCardCreate, i int) {
		q.SetPhaseID(phaseID).
			SetNamespace(ns).
			SetKey(r[i].Key).
			SetType(r[i].Type).
			SetName(r[i].Name).
			SetNillableDescription(r[i].Description).
			SetMetadata(r[i].Metadata).
			SetNillableFeatureKey(r[i].FeatureKey).
			SetNillableFeaturesID(r[i].FeatureID).
			SetEntitlementTemplate(r[i].EntitlementTemplate).
			SetNillableBillingCadence(r[i].BillingCadence)

		if r[i].TaxConfig != nil {
			q.SetTaxConfig(r[i].TaxConfig)
		}

		if r[i].Price != nil {
			q.SetPrice(r[i].Price)
		}
	}
}

func (a *adapter) DeletePhase(ctx context.Context, params plan.DeletePhaseInput) error {
	fn := func(ctx context.Context, a *adapter) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete PlanPhase parameters: %w", err)
		}

		if params.SkipSoftDelete {
			// Delete plan phase permanently
			err := a.db.PlanPhase.DeleteOneID(params.ID).
				Where(phasedb.Namespace(params.Namespace)).
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
		} else {
			phaseRow, err := a.db.PlanPhase.Query().
				Where(
					phasedb.And(
						phasedb.Namespace(params.Namespace),
						phasedb.ID(params.ID),
					),
				).
				WithRatecards().
				First(ctx)
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
			err = a.db.PlanPhase.UpdateOneID(params.ID).
				Where(phasedb.Namespace(params.Namespace)).
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

			for _, ratecard := range phaseRow.Edges.Ratecards {
				err = a.db.PlanRateCard.UpdateOneID(ratecard.ID).
					Where(ratecarddb.Namespace(params.Namespace)).
					SetDeletedAt(deletedAt).
					Exec(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to delete RateCard: %w", err)
				}
			}
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

			err = query.Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update PlanPhase: %w", err)
			}
		}

		if params.RateCards != nil && len(*params.RateCards) > 0 {
			diffResult, err := rateCardsDiff(*params.RateCards, p.RateCards)
			if err != nil {
				return nil, fmt.Errorf("failed to generate RateCard diff for PlanPhase update: %w", err)
			}

			if !diffResult.IsDiff() {
				return p, nil
			}

			if len(diffResult.Add) > 0 {
				bulk, bulkFn := newRateCardBulkCreate(diffResult.Add, p.ID, params.Namespace)

				if err = a.db.PlanRateCard.MapCreateBulk(bulk, bulkFn).Exec(ctx); err != nil {
					return nil, fmt.Errorf("failed to bulk create RateCards: %w", err)
				}
			}

			if len(diffResult.Remove) > 0 {
				for _, rateCard := range diffResult.Remove {
					err = a.db.PlanRateCard.DeleteOneID(rateCard.ID).Where(ratecarddb.Namespace(params.Namespace)).Exec(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to delete RateCard: %w", err)
					}
				}
			}

			if len(diffResult.Update) > 0 {
				for _, rateCardInput := range diffResult.Update {
					q := a.db.PlanRateCard.UpdateOneID(rateCardInput.ID).
						Where(ratecarddb.Namespace(params.Namespace)).
						SetOrClearMetadata(&rateCardInput.Metadata).
						SetName(rateCardInput.Name).
						SetOrClearDescription(rateCardInput.Description).
						SetOrClearFeatureKey(rateCardInput.FeatureKey).
						SetEntitlementTemplate(rateCardInput.EntitlementTemplate).
						SetOrClearBillingCadence(rateCardInput.BillingCadence)

					if rateCardInput.TaxConfig != nil {
						q.SetTaxConfig(rateCardInput.TaxConfig)
					}

					if rateCardInput.Price != nil {
						q.SetPrice(rateCardInput.Price)
					}

					if rateCardInput.FeatureID == nil {
						q.ClearFeatureID()
					}

					err = q.Exec(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to update RateCard: %w", err)
					}
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
	Add []entdb.PlanRateCard

	// Update defines the list of plan.UpdatePhaseInput for plan.Phase objects to update
	Update []entdb.PlanRateCard

	// Remove defines the list of plan.DeletePhaseInput for plan.Phase identifiers to delete
	Remove []entdb.PlanRateCard

	// Keep defines the list of plan.Phase to keep unmodified
	Keep []entdb.PlanRateCard
}

func rateCardsDiff(inputs, rateCards []plan.RateCard) (rateCardsDiffResult, error) {
	result := rateCardsDiffResult{}

	inputsMap := make(map[string]entdb.PlanRateCard, len(inputs))
	for _, input := range inputs {
		rc, err := asPlanRateCardRow(input)
		if err != nil {
			return result, fmt.Errorf("failed to cast RateCard: %w", err)
		}

		inputsMap[rc.Key] = rc
	}

	rateCardsMap := make(map[string]entdb.PlanRateCard, len(rateCards))
	for _, rateCard := range rateCards {
		rc, err := asPlanRateCardRow(rateCard)
		if err != nil {
			return result, fmt.Errorf("failed to cast RateCard: %w", err)
		}

		rateCardsMap[rc.Key] = rc
	}

	rateCardsVisited := make(map[string]struct{})
	for rateCardKey, input := range inputsMap {
		rateCard, ok := rateCardsMap[rateCardKey]

		// Collect new phases
		if !ok {
			result.Add = append(result.Add, input)

			rateCardsVisited[rateCardKey] = struct{}{}

			continue
		}

		// Collect rate cards to be updated
		match, err := rateCardCmp(input, rateCard)
		if err != nil {
			return result, fmt.Errorf("failed to compare RateCard: %w", err)
		}

		if !match {
			input.Namespace = rateCard.Namespace
			input.ID = rateCard.ID
			input.PhaseID = rateCard.PhaseID
			result.Update = append(result.Update, input)

			rateCardsVisited[rateCardKey] = struct{}{}
		} else {
			result.Keep = append(result.Keep, rateCard)

			rateCardsVisited[rateCardKey] = struct{}{}
		}
	}

	// Collect RateCards to be deleted
	if len(rateCardsVisited) != len(rateCardsMap) {
		for rateCardKey, rateCard := range rateCardsMap {
			if _, ok := rateCardsVisited[rateCardKey]; !ok {
				result.Remove = append(result.Remove, rateCard)
			}
		}
	}

	return result, nil
}

func (r rateCardsDiffResult) IsDiff() bool {
	return len(r.Add) > 0 || len(r.Update) > 0 || len(r.Remove) > 0
}

func rateCardCmp(r1, r2 entdb.PlanRateCard) (bool, error) {
	if r1.Namespace != r2.Namespace {
		return false, nil
	}

	if !plan.MetadataEqual(r1.Metadata, r2.Metadata) {
		return false, nil
	}

	if r1.Type != r2.Type {
		return false, nil
	}

	if r1.Name != r2.Name {
		return false, nil
	}

	if lo.FromPtrOr(r1.Description, "") != lo.FromPtrOr(r2.Description, "") {
		return false, nil
	}

	if lo.FromPtrOr(r1.FeatureKey, "") != lo.FromPtrOr(r2.FeatureKey, "") {
		return false, nil
	}

	if lo.FromPtrOr(r1.FeatureID, "") != lo.FromPtrOr(r2.FeatureID, "") {
		return false, nil
	}

	tmpl1, err := json.Marshal(r1.EntitlementTemplate)
	if err != nil {
		return false, fmt.Errorf("failed to marshal EntitlementTemplate: %w", err)
	}

	tmpl2, err := json.Marshal(r2.EntitlementTemplate)
	if err != nil {
		return false, fmt.Errorf("failed to marshal EntitlementTemplate: %w", err)
	}

	if string(tmpl1) != string(tmpl2) {
		return false, nil
	}

	tax1, err := json.Marshal(r1.TaxConfig)
	if err != nil {
		return false, fmt.Errorf("failed to marshal TaxConfig: %w", err)
	}

	tax2, err := json.Marshal(r2.TaxConfig)
	if err != nil {
		return false, fmt.Errorf("failed to marshal TaxConfig: %w", err)
	}

	if string(tax1) != string(tax2) {
		return false, nil
	}

	if lo.FromPtrOr(r1.BillingCadence, "") != lo.FromPtrOr(r2.BillingCadence, "") {
		return false, nil
	}

	price1, err := json.Marshal(r1.Price)
	if err != nil {
		return false, fmt.Errorf("failed to marshal Price: %w", err)
	}

	price2, err := json.Marshal(r2.Price)
	if err != nil {
		return false, fmt.Errorf("failed to marshal Price: %w", err)
	}

	if string(price1) != string(price2) {
		return false, nil
	}

	return true, nil
}
