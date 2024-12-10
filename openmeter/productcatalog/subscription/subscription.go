package plansubscription

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type Plan struct {
	plan.Plan
}

var _ subscription.Plan = &Plan{}

func (p *Plan) GetRef() subscription.PlanRef {
	return subscription.PlanRef{
		Id:      p.ID,
		Key:     p.Key,
		Version: p.Version,
	}
}

func (p *Plan) ToCreateSubscriptionPlanInput() subscription.CreateSubscriptionPlanInput {
	return subscription.CreateSubscriptionPlanInput{
		Plan: &subscription.PlanRef{
			Id:      p.ID,
			Key:     p.Key,
			Version: p.Version,
		},
	}
}

func (p *Plan) GetPhases() []subscription.PlanPhase {
	ps := make([]subscription.PlanPhase, 0, len(p.Phases))
	for _, ph := range p.Phases {
		ps = append(ps, &Phase{
			Phase: ph,
		})
	}

	return ps
}

func (p *Plan) Currency() currencyx.Code {
	return currencyx.Code(p.Plan.Currency)
}

type Phase struct {
	plan.Phase
}

var _ subscription.PlanPhase = &Phase{}

func (p *Phase) ToCreateSubscriptionPhasePlanInput() subscription.CreateSubscriptionPhasePlanInput {
	return subscription.CreateSubscriptionPhasePlanInput{
		PhaseKey:    p.Key,
		StartAfter:  p.StartAfter,
		Name:        p.Name,
		Description: p.Description,
	}
}

func (p *Phase) GetRateCards() []subscription.PlanRateCard {
	rcs := make([]subscription.PlanRateCard, 0, len(p.RateCards))
	for _, rc := range p.RateCards {
		rcs = append(rcs, &RateCard{
			PhaseKey: p.Key,
			RateCard: rc,
		})
	}

	return rcs
}

func (p *Phase) GetKey() string {
	return p.Key
}

type RateCard struct {
	PhaseKey string
	productcatalog.RateCard
}

var _ subscription.PlanRateCard = &RateCard{}

func (r *RateCard) ToCreateSubscriptionItemPlanInput() subscription.CreateSubscriptionItemPlanInput {
	m := r.RateCard.AsMeta()

	var fk *string
	if m.Feature != nil {
		fk = &m.Feature.Key
	}

	var cadence *datex.Period

	// FIXME: BillingCadence could be a method on RateCard
	switch r.RateCard.Type() {
	case productcatalog.FlatFeeRateCardType:
		if rc, ok := r.RateCard.(*productcatalog.FlatFeeRateCard); ok {
			cadence = rc.BillingCadence
		}
	case productcatalog.UsageBasedRateCardType:
		if rc, ok := r.RateCard.(*productcatalog.UsageBasedRateCard); ok {
			cadence = &rc.BillingCadence
		}
	}

	return subscription.CreateSubscriptionItemPlanInput{
		PhaseKey: r.PhaseKey,
		ItemKey:  r.Key(),
		RateCard: subscription.RateCard{
			Name:                m.Name,
			Description:         m.Description,
			FeatureKey:          fk,
			EntitlementTemplate: m.EntitlementTemplate,
			TaxConfig:           m.TaxConfig,
			Price:               m.Price,
			BillingCadence:      cadence,
		},
	}
}

func (r *RateCard) GetKey() string {
	return r.Key()
}
