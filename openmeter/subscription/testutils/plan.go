package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
)

var ExamplePlanRef subscription.PlanRef = subscription.PlanRef{
	Key:     "test-plan",
	Version: 1,
}

var (
	oneMonthISO, _ = datex.ISOString("P1M").Parse()
	twoMonthISO, _ = datex.ISOString("P2M").Parse()
	sixMonthISO, _ = datex.ISOString("P6M").Parse()
)

var ExamplePlan = &Plan{
	PlanInput: subscription.CreateSubscriptionPlanInput{
		Plan: ExamplePlanRef,
	},
	Phases: []*PlanPhase{
		{
			PhaseInput: subscription.CreateSubscriptionPhasePlanInput{
				PhaseKey:   "test-phase-1",
				StartAfter: oneMonthISO,
			},
			RateCards: []*RateCard{
				{
					RateCardKey: "test-rate-card-1",
					SubscriptionItemCreateInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey:   "test-phase-1",
						ItemKey:    "test-rate-card-1",
						FeatureKey: lo.ToPtr(ExampleFeatureKey),
					},
				},
			},
		},
		{
			PhaseInput: subscription.CreateSubscriptionPhasePlanInput{
				PhaseKey:   "test-phase-2",
				StartAfter: twoMonthISO,
			},
			RateCards: []*RateCard{
				{
					RateCardKey: "test-rate-card-1",
					SubscriptionItemCreateInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey:   "test-phase-2",
						ItemKey:    "test-rate-card-1",
						FeatureKey: lo.ToPtr(ExampleFeatureKey),
						CreateEntitlementInput: &subscription.CreateSubscriptionEntitlementInput{
							EntitlementType:        entitlement.EntitlementTypeMetered,
							IssueAfterReset:        lo.ToPtr(1000.0),
							UsagePeriodISODuration: &ISOMonth,
						},
					},
				},
				{
					RateCardKey: "test-rate-card-2",
					SubscriptionItemCreateInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "test-phase-2",
						ItemKey:  "test-rate-card-2",
						CreatePriceInput: &subscription.CreatePriceInput{
							PhaseKey: "test-phase-2",
							ItemKey:  "test-rate-card-2",
							Value:    "100",
							Key:      "test-rate-card-2",
						},
					},
				},
			},
		},
		{
			PhaseInput: subscription.CreateSubscriptionPhasePlanInput{
				PhaseKey:   "test-phase-3",
				StartAfter: sixMonthISO,
			},
			RateCards: []*RateCard{
				{
					RateCardKey: "test-rate-card-1",
					// We take away he entitlement in this phase
					SubscriptionItemCreateInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey:   "test-phase-3",
						ItemKey:    "test-rate-card-1",
						FeatureKey: lo.ToPtr(ExampleFeatureKey),
					},
				},
			},
		},
	},
}

func NewMockPlanAdapter(t *testing.T) *planAdapter {
	return &planAdapter{}
}

type planAdapter struct {
	store map[string]map[int]*Plan
}

var _ subscription.PlanAdapter = &planAdapter{}

func (a *planAdapter) GetVersion(ctx context.Context, k string, v int) (subscription.Plan, error) {
	versions, ok := a.store[k]
	if !ok {
		return nil, &subscription.PlanNotFoundError{Key: k, Version: v}
	}
	version, ok := versions[v]
	if !ok {
		return nil, &subscription.PlanNotFoundError{Key: k, Version: v}
	}

	return version, nil
}

func (a *planAdapter) AddPlan(plan *Plan) {
	if a.store == nil {
		a.store = make(map[string]map[int]*Plan)
	}

	if _, ok := a.store[plan.PlanInput.Plan.Key]; !ok {
		a.store[plan.PlanInput.Plan.Key] = make(map[int]*Plan)
	}

	a.store[plan.PlanInput.Plan.Key][plan.PlanInput.Plan.Version] = plan
}

func (a *planAdapter) RemovePlan(ref subscription.PlanRef) {
	if _, ok := a.store[ref.Key]; !ok {
		return
	}

	delete(a.store[ref.Key], ref.Version)
}

type Plan struct {
	PlanInput subscription.CreateSubscriptionPlanInput
	Phases    []*PlanPhase
}

var _ subscription.Plan = &Plan{}

func (p *Plan) ToCreateSubscriptionPlanInput() subscription.CreateSubscriptionPlanInput {
	return p.PlanInput
}

func (p *Plan) GetPhases() []subscription.PlanPhase {
	// convert to subscription.PlanPhase
	phases := make([]subscription.PlanPhase, len(p.Phases))
	for i, phase := range p.Phases {
		phases[i] = phase
	}

	return phases
}

func (p *Plan) GetKey() string {
	return p.PlanInput.Plan.Key
}

func (p *Plan) GetVersionNumber() int {
	return p.PlanInput.Plan.Version
}

type PlanPhase struct {
	RateCards  []*RateCard
	PhaseInput subscription.CreateSubscriptionPhasePlanInput
}

var _ subscription.PlanPhase = &PlanPhase{}

func (p *PlanPhase) ToCreateSubscriptionPhasePlanInput() subscription.CreateSubscriptionPhasePlanInput {
	return p.PhaseInput
}

func (p *PlanPhase) GetRateCards() []subscription.RateCard {
	// convert
	rateCards := make([]subscription.RateCard, len(p.RateCards))
	for i, rateCard := range p.RateCards {
		rateCards[i] = rateCard
	}

	return rateCards
}

func (p *PlanPhase) GetKey() string {
	return p.PhaseInput.PhaseKey
}

type RateCard struct {
	RateCardKey                 string
	SubscriptionItemCreateInput subscription.CreateSubscriptionItemPlanInput
}

var _ subscription.RateCard = &RateCard{}

func (r *RateCard) ToCreateSubscriptionItemPlanInput() subscription.CreateSubscriptionItemPlanInput {
	return r.SubscriptionItemCreateInput
}

func (r *RateCard) GetKey() string {
	return r.RateCardKey
}
