package adapter

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	phasedb "github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type createPhaseInput struct {
	models.NamespacedModel
	productcatalog.Phase

	// PlanID is the ID of the plan the phase belongs to
	PlanID string `json:"planId"`

	// Index is the index of the phase in the plan
	Index int `json:"index"`
}

func (a *adapter) CreatePhase(ctx context.Context, params createPhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Phase, error) {
		planPhaseRow, err := a.db.PlanPhase.Create().
			SetNamespace(params.Namespace).
			SetKey(params.Key).
			SetIndex(params.Index).
			SetName(params.Name).
			SetNillableDescription(params.Description).
			SetMetadata(params.Metadata).
			SetNillableDuration(params.Duration.ISOStringPtrOrNil()).
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
