package plansubscription

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanInput struct {
	ref *PlanRefInput
}

func (p *PlanInput) Validate() error {
	if p.ref == nil {
		return fmt.Errorf("plan reference must be provided")
	}

	return nil
}

func (p *PlanInput) AsRef() *PlanRefInput {
	return p.ref
}

func (p *PlanInput) FromRef(pr *PlanRefInput) {
	p.ref = pr
}

type PlanRefInput struct {
	Key     string `json:"key"`
	Version *int   `json:"version,omitempty"`
}

type Plan struct {
	productcatalog.Plan
	Ref *models.NamespacedID
}

var _ subscription.Plan = &Plan{}

func (p *Plan) GetName() string {
	return p.Name
}

func (p *Plan) ToCreateSubscriptionPlanInput() subscription.CreateSubscriptionPlanInput {
	// We only store a reference if the Plan exists
	var ref *subscription.PlanRef

	if p.Ref != nil {
		ref = &subscription.PlanRef{
			Id:      p.Ref.ID,
			Key:     p.Key,
			Version: p.Version,
		}
	}

	return subscription.CreateSubscriptionPlanInput{
		Plan:      ref,
		Alignment: p.Alignment,
	}
}

func (p *Plan) GetPhases() []subscription.PlanPhase {
	ps := make([]subscription.PlanPhase, 0, len(p.Phases))
	startAfter := isodate.Period{}
	for _, ph := range p.Phases {
		ps = append(ps, &Phase{
			Phase:      ph,
			StartAfter: startAfter,
		})

		startAfter, _ = startAfter.Add(lo.FromPtr(ph.Duration))
	}

	return ps
}

func (p *Plan) Currency() currencyx.Code {
	return currencyx.Code(p.Plan.Currency)
}

type Phase struct {
	productcatalog.Phase
	StartAfter isodate.Period
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
	return subscription.CreateSubscriptionItemPlanInput{
		PhaseKey: r.PhaseKey,
		ItemKey:  r.Key(),
		RateCard: r.RateCard.Clone(),
	}
}

func (r *RateCard) GetKey() string {
	return r.Key()
}
