package subscription

import (
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/applieddiscount"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Spec is the complete generic specification of how a Subscription (sub)Entity should look like.
//
// It is generic as it doesn't include any hard references or exact timestamps or the sort.
//
// Both Subscription, SubscriptionPhase and SubscriptionItem can have 3 interfaces defineing their spec.
// First is which is inferred from the plan content, it is suffixed with PlanInput.
// Second is which is inferred form the customer, it is suffixed with CustomerInput.
// Third is the final spec which is a combination of the above two, it is suffixed with Spec.

type CreateSubscriptionPlanInput struct {
	Plan PlanRef
}

type CreateSubscriptionCustomerInput struct {
	CustomerId string `json:"customerId,omitempty"`
	Currency   models.CurrencyCode
	ActiveFrom time.Time
	ActiveTo   *time.Time
}

type SubscriptionSpec struct {
	CreateSubscriptionPlanInput
	CreateSubscriptionCustomerInput

	Phases map[string]*SubscriptionPhaseSpec
}

// GetSortedPhases returns the subscription phase references time sorted order ASC.
func (s *SubscriptionSpec) GetSortedPhases() []*SubscriptionPhaseSpec {
	phases := make([]*SubscriptionPhaseSpec, 0, len(s.Phases))
	for _, phase := range s.Phases {
		phases = append(phases, phase)
	}

	slices.SortStableFunc(phases, func(i, j *SubscriptionPhaseSpec) int {
		return int((i.StartAfter - j.StartAfter))
	})

	return phases
}

func (s *SubscriptionSpec) GetCurrentPhaseAt(t time.Time) (*SubscriptionPhaseSpec, bool) {
	var current *SubscriptionPhaseSpec
	for _, phase := range s.GetSortedPhases() {
		if s.ActiveFrom.Add(phase.StartAfter).Before(t) {
			current = phase
		} else {
			break
		}
	}

	// The subscription is already expired at that point
	if s.ActiveTo != nil && !s.ActiveTo.After(t) {
		current = nil
	}

	if current == nil {
		return nil, false
	}
	return current, true
}

func (s *SubscriptionSpec) Validate() error {
	// TODO: write validation logic for Subscriptions
	// All validatons should happen here!
	return nil
}

type CreateSubscriptionPhasePlanInput struct {
	PhaseKey   string
	StartAfter time.Duration
	// TODO: add back Plan level discounts
	// CreateDiscountInput *applieddiscount.CreateInput
}

type CreateSubscriptionPhaseCustomerInput struct {
	CreateDiscountInput *applieddiscount.CreateInput
}

type SubscriptionPhaseSpec struct {
	CreateSubscriptionPhasePlanInput
	CreateSubscriptionPhaseCustomerInput
	Items map[string]*SubscriptionItemSpec
}

type CreateSubscriptionItemPlanInput struct {
	PhaseKey               string
	ItemKey                string
	FeatureKey             *string
	CreateEntitlementInput *entitlement.CreateEntitlementInputs
	CreatePriceInput       *price.CreateInput
}

type SubscriptionItemSpec struct {
	CreateSubscriptionItemPlanInput
}

// SpecFromPlan creates a SubscriptionSpec from a Plan and a CreateSubscriptionCustomerInput.
func SpecFromPlan(p Plan, c CreateSubscriptionCustomerInput) (*SubscriptionSpec, error) {
	spec := &SubscriptionSpec{
		CreateSubscriptionPlanInput:     p.ToCreateSubscriptionPlanInput(),
		CreateSubscriptionCustomerInput: c,
		Phases:                          make(map[string]*SubscriptionPhaseSpec),
	}

	if len(p.Phases()) == 0 {
		return nil, fmt.Errorf("plan %s version %d has no phases", p.Key(), p.Version())
	}

	for _, planPhase := range p.Phases() {
		phase := SubscriptionPhaseSpec{
			CreateSubscriptionPhasePlanInput: planPhase.ToCreateSubscriptionPhasePlanInput(),
			// TODO: implement discounts
			// CreateSubscriptionPhaseCustomerInput: CreateSubscriptionPhaseCustomerInput{},
			Items: make(map[string]*SubscriptionItemSpec),
		}

		if len(planPhase.RateCards()) == 0 {
			return nil, fmt.Errorf("phase %s of plan %s version %d has no rate cards", phase.PhaseKey, p.Key(), p.Version())
		}

		for _, rateCard := range planPhase.RateCards() {
			item := SubscriptionItemSpec{
				CreateSubscriptionItemPlanInput: rateCard.ToCreateSubscriptionItemPlanInput(),
			}

			if _, exists := phase.Items[item.ItemKey]; exists {
				return nil, fmt.Errorf("duplicate item key %s in phase %s of plan %s version %d", item.ItemKey, phase.PhaseKey, p.Key(), p.Version())
			}

			phase.Items[item.ItemKey] = &item
		}

		if _, exists := spec.Phases[phase.PhaseKey]; exists {
			return nil, fmt.Errorf("duplicate phase key %s in plan %s version %d", phase.PhaseKey, p.Key(), p.Version())
		}

		spec.Phases[phase.PhaseKey] = &phase
	}

	return spec, nil
}

type SpecOperation int

const (
	SpecOperationCreate = iota
	SpecOperationEdit
)

type ApplyContext struct {
	Operation   SpecOperation
	CurrentTime time.Time
}

// Each Patch applies its changes to the SubscriptionSpec.
type Applies interface {
	ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error
}

func (s *SubscriptionSpec) ApplyPatches(patches []Applies, context ApplyContext) error {
	for i, patch := range patches {
		err := patch.ApplyTo(s, context)
		if err != nil {
			return fmt.Errorf("patch %d failed: %w", i, err)
		}
		if err = s.Validate(); err != nil {
			return fmt.Errorf("patch %d failed during validation: %w", i, err)
		}

	}
	return nil
}
