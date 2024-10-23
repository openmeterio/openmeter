package subscription

import (
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/applieddiscount"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
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
	Plan PlanRef `json:"plan"`
}

type CreateSubscriptionCustomerInput struct {
	CustomerId string         `json:"customerId"`
	Currency   currencyx.Code `json:"currency"`
	ActiveFrom time.Time      `json:"activeFrom,omitempty"`
	ActiveTo   *time.Time     `json:"activeTo,omitempty"`
}

type SubscriptionSpec struct {
	CreateSubscriptionPlanInput
	CreateSubscriptionCustomerInput

	Phases map[string]*SubscriptionPhaseSpec
}

func (s *SubscriptionSpec) GetCreateInput() CreateSubscriptionInput {
	return CreateSubscriptionInput{
		Plan:       s.Plan,
		CustomerId: s.CustomerId,
		Currency:   s.Currency,
		CadencedModel: models.CadencedModel{
			ActiveFrom: s.ActiveFrom,
			ActiveTo:   s.ActiveTo,
		},
	}
}

// GetSortedPhases returns the subscription phase references time sorted order ASC.
func (s *SubscriptionSpec) GetSortedPhases() []*SubscriptionPhaseSpec {
	phases := make([]*SubscriptionPhaseSpec, 0, len(s.Phases))
	for _, phase := range s.Phases {
		phases = append(phases, phase)
	}

	slices.SortStableFunc(phases, func(i, j *SubscriptionPhaseSpec) int {
		iTime, _ := i.StartAfter.AddTo(s.ActiveFrom)
		jTime, _ := j.StartAfter.AddTo(s.ActiveFrom)
		return int(iTime.Sub(jTime))
	})

	return phases
}

func (s *SubscriptionSpec) GetCurrentPhaseAt(t time.Time) (*SubscriptionPhaseSpec, bool) {
	var current *SubscriptionPhaseSpec
	for _, phase := range s.GetSortedPhases() {
		if st, _ := phase.StartAfter.AddTo(s.ActiveFrom); st.Before(t) {
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
	// All consistency checks should happen here
	for _, phase := range s.Phases {
		if err := phase.Validate(); err != nil {
			return fmt.Errorf("phase %s validation failed: %w", phase.PhaseKey, err)
		}
	}
	return nil
}

type CreateSubscriptionPhasePlanInput struct {
	PhaseKey   string       `json:"phaseKey"`
	StartAfter datex.Period `json:"startAfter"`
	// TODO: add back Plan level discounts
	// CreateDiscountInput *applieddiscount.CreateInput
}

type CreateSubscriptionPhaseCustomerInput struct {
	CreateDiscountInput *applieddiscount.Spec `json:"createDiscountInput,omitempty"`
}

type CreateSubscriptionPhaseInput struct {
	CreateSubscriptionPhasePlanInput
	CreateSubscriptionPhaseCustomerInput
}

type SubscriptionPhaseSpec struct {
	CreateSubscriptionPhaseInput
	Items map[string]*SubscriptionItemSpec
}

func (s *SubscriptionPhaseSpec) Validate() error {
	// TODO: implement
	for _, item := range s.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item %s validation failed: %w", item.ItemKey, err)
		}
	}
	return nil
}

type CreateSubscriptionEntitlementSpec struct {
	EntitlementType entitlement.EntitlementType `json:"type"`
	// TODO: Add way to specify MeasureUsageFrom
	// explanation: MeasureUsageFrom cannot have a time.Time anchor when creating from plan. The enum value would also most likely be different.
	IssueAfterReset         *float64 `json:"issueAfterReset,omitempty"`
	IssueAfterResetPriority *uint8   `json:"issueAfterResetPriority,omitempty"`
	IsSoftLimit             *bool    `json:"isSoftLimit,omitempty"`
	Config                  []byte   `json:"config,omitempty"`
	// Explanation: UsagePeriod cannot have a time.Time anchor when creating from plan
	UsagePeriodISODuration *datex.Period `json:"usagePeriodPeriod,omitempty"`
	PreserveOverageAtReset *bool         `json:"preserveOverageAtReset,omitempty"`
}

func (s *CreateSubscriptionEntitlementSpec) ToCreateEntitlementInput(
	namespace string,
	featureKey string,
	subjectKey string,
	cadence models.CadencedModel,
) (*entitlement.CreateEntitlementInputs, error) {
	inputs := &entitlement.CreateEntitlementInputs{
		Namespace:               namespace,
		FeatureKey:              &featureKey,
		SubjectKey:              subjectKey,
		EntitlementType:         s.EntitlementType,
		IssueAfterReset:         s.IssueAfterReset,
		IssueAfterResetPriority: s.IssueAfterResetPriority,
		IsSoftLimit:             s.IsSoftLimit,
		Config:                  s.Config,
		PreserveOverageAtReset:  s.PreserveOverageAtReset,
		ActiveFrom:              &cadence.ActiveFrom,
		ActiveTo:                cadence.ActiveTo,
	}

	if s.UsagePeriodISODuration != nil {
		// FIXME: using cadence.ActiveFrom won't work for upgrade/downgrade scenarios & other cases where partial periods or usage sharing is needed
		usagePeriod, err := recurrence.FromISODuration(s.UsagePeriodISODuration, cadence.ActiveFrom)
		if err != nil {
			return nil, fmt.Errorf("couldnt compute UsagePeriod: %w", err)
		}
		inputs.UsagePeriod = lo.ToPtr(entitlement.UsagePeriod(usagePeriod))
	}

	return inputs, nil
}

type CreateSubscriptionItemPlanInput struct {
	PhaseKey               string                             `json:"phaseKey"`
	ItemKey                string                             `json:"itemKey"`
	FeatureKey             *string                            `json:"featureKey,omitempty"`
	CreateEntitlementInput *CreateSubscriptionEntitlementSpec `json:"createEntitlementSpec,omitempty"`
	CreatePriceInput       *price.Spec                        `json:"createPriceInput,omitempty"`
}

type SubscriptionItemSpec struct {
	CreateSubscriptionItemPlanInput
}

func (s SubscriptionItemSpec) GetRef(subId string) SubscriptionItemRef {
	return SubscriptionItemRef{
		SubscriptionId: subId,
		PhaseKey:       s.PhaseKey,
		ItemKey:        s.ItemKey,
	}
}

func (s SubscriptionItemSpec) HasPrice() bool {
	return s.CreatePriceInput != nil
}

func (s SubscriptionItemSpec) HasEntitlement() bool {
	return s.CreateEntitlementInput != nil
}

func (s SubscriptionItemSpec) HasFeature() bool {
	return s.FeatureKey != nil
}

func (s *SubscriptionItemSpec) Validate() error {
	if s.CreatePriceInput != nil {
		if s.CreatePriceInput.ItemKey != s.ItemKey {
			return &SpecValidationError{
				AffectedKeys: [][]string{
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
					},
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
						"CreatePriceInput",
						"ItemKey",
					},
				},
				Msg: "ItemKey in CreatePriceInput must match ItemKey",
			}
		}
		if s.CreatePriceInput.PhaseKey != s.PhaseKey {
			return &SpecValidationError{
				AffectedKeys: [][]string{
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
					},
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
						"CreatePriceInput",
						"PhaseKey",
					},
				},
				Msg: "PhaseKey in CreatePriceInput must match PhaseKey",
			}
		}
	}
	if s.CreateEntitlementInput != nil {
		if s.FeatureKey == nil {
			return &SpecValidationError{
				AffectedKeys: [][]string{
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
						"FeatureKey",
					},
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
						"CreateEntitlementInput",
					},
				},
				Msg: "FeatureKey is required for CreateEntitlementInput",
			}
		}
	}
	if s.FeatureKey == nil {
		if s.CreatePriceInput == nil {
			return &SpecValidationError{
				AffectedKeys: [][]string{
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
						"FeatureKey",
					},
				},
				Msg: "FeatureKey is required for Item when Price is not defiend",
			}
		}

		if s.CreatePriceInput.Key != s.ItemKey {
			return &SpecValidationError{
				AffectedKeys: [][]string{
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
					},
					{
						"phaseKey",
						s.PhaseKey,
						"itemKey",
						s.ItemKey,
						"CreatePriceInput",
						"Key",
					},
				},
				Msg: "ItemKey must match Price Key when feature is not present",
			}
		}
	}

	// TODO: implement
	return nil
}

// SpecFromPlan creates a SubscriptionSpec from a Plan and a CreateSubscriptionCustomerInput.
func SpecFromPlan(p Plan, c CreateSubscriptionCustomerInput) (*SubscriptionSpec, error) {
	spec := &SubscriptionSpec{
		CreateSubscriptionPlanInput:     p.ToCreateSubscriptionPlanInput(),
		CreateSubscriptionCustomerInput: c,
		Phases:                          make(map[string]*SubscriptionPhaseSpec),
	}

	if len(p.GetPhases()) == 0 {
		return nil, fmt.Errorf("plan %s version %d has no phases", p.GetKey(), p.GetVersionNumber())
	}

	// Validate that the plan phases are returned in order
	planPhases := p.GetPhases()
	for i := range planPhases {
		if i == 0 {
			continue
		}
		if diff, err := planPhases[i].ToCreateSubscriptionPhasePlanInput().StartAfter.Subtract(planPhases[i-1].ToCreateSubscriptionPhasePlanInput().StartAfter); err != nil || diff.IsNegative() {
			return nil, fmt.Errorf("phases %s and %s of plan %s version %d are in the wrong order", planPhases[i].GetKey(), planPhases[i-1].GetKey(), p.GetKey(), p.GetVersionNumber())
		}
	}

	for _, planPhase := range planPhases {
		phase := SubscriptionPhaseSpec{
			CreateSubscriptionPhaseInput: CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: planPhase.ToCreateSubscriptionPhasePlanInput(),
			},
			// TODO: implement discounts
			// CreateSubscriptionPhaseCustomerInput: CreateSubscriptionPhaseCustomerInput{},
			Items: make(map[string]*SubscriptionItemSpec),
		}

		if len(planPhase.GetRateCards()) == 0 {
			return nil, fmt.Errorf("phase %s of plan %s version %d has no rate cards", phase.PhaseKey, p.GetKey(), p.GetVersionNumber())
		}

		for _, rateCard := range planPhase.GetRateCards() {
			item := SubscriptionItemSpec{
				CreateSubscriptionItemPlanInput: rateCard.ToCreateSubscriptionItemPlanInput(),
			}

			if _, exists := phase.Items[item.ItemKey]; exists {
				return nil, fmt.Errorf("duplicate item key %s in phase %s of plan %s version %d", item.ItemKey, phase.PhaseKey, p.GetKey(), p.GetVersionNumber())
			}

			phase.Items[item.ItemKey] = &item
		}

		if _, exists := spec.Phases[phase.PhaseKey]; exists {
			return nil, fmt.Errorf("duplicate phase key %s in plan %s version %d", phase.PhaseKey, p.GetKey(), p.GetVersionNumber())
		}

		spec.Phases[phase.PhaseKey] = &phase
	}

	// Lets validate the spec
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation failed: %w", err)
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

type SpecValidationError struct {
	// FIXME: This spec is broken and painful, lets improve it
	AffectedKeys [][]string
	Msg          string
}

func (e *SpecValidationError) Error() string {
	return e.Msg
}
