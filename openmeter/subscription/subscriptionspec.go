package subscription

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
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
	Plan *PlanRef `json:"plan"`
}

type CreateSubscriptionCustomerInput struct {
	models.AnnotatedModel `json:",inline"`
	Name                  string         `json:"name"`
	Description           *string        `json:"description,omitempty"`
	CustomerId            string         `json:"customerId"`
	Currency              currencyx.Code `json:"currency"`
	ActiveFrom            time.Time      `json:"activeFrom,omitempty"`
	ActiveTo              *time.Time     `json:"activeTo,omitempty"`
}

type SubscriptionSpec struct {
	CreateSubscriptionPlanInput     `json:",inline"`
	CreateSubscriptionCustomerInput `json:",inline"`

	// We use pointers so Patches can manipulate the spec
	Phases map[string]*SubscriptionPhaseSpec `json:"phases"`
}

func (s *SubscriptionSpec) ToCreateSubscriptionEntityInput(ns string) CreateSubscriptionEntityInput {
	return CreateSubscriptionEntityInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ns,
		},
		Plan:           s.Plan,
		CustomerId:     s.CustomerId,
		Currency:       s.Currency,
		AnnotatedModel: s.AnnotatedModel,
		Name:           s.Name,
		Description:    s.Description,
		CadencedModel: models.CadencedModel{
			ActiveFrom: s.ActiveFrom,
			ActiveTo:   s.ActiveTo,
		},
	}
}

func (s *SubscriptionSpec) GetPhaseCadence(phaseKey string) (models.CadencedModel, error) {
	var def models.CadencedModel
	phase, exists := s.Phases[phaseKey]
	if !exists {
		return def, fmt.Errorf("phase %s not found", phaseKey)
	}

	// Lets calculate the phase Cadence for the new spec
	phaseStartTime, _ := phase.StartAfter.AddTo(s.ActiveFrom)
	var phaseEndTime *time.Time

	// Find the next phase if any
	sortedPhaseSpecs := s.GetSortedPhases()
	for i, p := range sortedPhaseSpecs {
		if p.PhaseKey == phase.PhaseKey && i+1 < len(sortedPhaseSpecs) {
			nextPhase := sortedPhaseSpecs[i+1]
			et, _ := nextPhase.StartAfter.AddTo(s.ActiveFrom)
			phaseEndTime = &et
			break
		}
	}

	// If the subscription is scheduled to end, we have to check whether that end time is before the phase end time
	if s.ActiveTo != nil {
		if phaseEndTime == nil {
			phaseEndTime = s.ActiveTo
		} else if s.ActiveTo.Before(*phaseEndTime) {
			phaseEndTime = s.ActiveTo
		}
	}

	cadence := models.CadencedModel{
		ActiveFrom: phaseStartTime.UTC(),
		ActiveTo: convert.SafeDeRef(phaseEndTime, func(t time.Time) *time.Time {
			// The phase end time cannot be before the phase start time
			if t.Before(phaseStartTime) {
				t = phaseStartTime
			}
			return lo.ToPtr(t.UTC())
		}),
	}

	return cadence, nil
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
		if st, _ := phase.StartAfter.AddTo(s.ActiveFrom); !st.After(t) {
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
	// All consistency checks should happen here
	var errs []error
	for _, phase := range s.Phases {
		if err := phase.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("phase %s validation failed: %w", phase.PhaseKey, err))
		}
	}
	return errors.Join(errs...)
}

type CreateSubscriptionPhasePlanInput struct {
	PhaseKey    string       `json:"key"`
	StartAfter  datex.Period `json:"startAfter"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	// TODO: add back Plan level discounts
}

func (i CreateSubscriptionPhasePlanInput) Validate() error {
	if i.PhaseKey == "" {
		return fmt.Errorf("phase key is required")
	}
	if i.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

type CreateSubscriptionPhaseCustomerInput struct {
	models.AnnotatedModel `json:",inline"`
}

type RemoveSubscriptionPhaseShifting int

const (
	RemoveSubscriptionPhaseShiftNext RemoveSubscriptionPhaseShifting = iota
	RemoveSubscriptionPhaseShiftPrev
)

func (s RemoveSubscriptionPhaseShifting) Validate() error {
	if s != RemoveSubscriptionPhaseShiftNext && s != RemoveSubscriptionPhaseShiftPrev {
		return fmt.Errorf("invalid RemoveSubscriptionPhaseShifting value %d", s)
	}
	return nil
}

type RemoveSubscriptionPhaseInput struct {
	Shift RemoveSubscriptionPhaseShifting `json:"shift"`
}

type CreateSubscriptionPhaseInput struct {
	// Duration is required exactly in cases where the phase wouldn't be the last phase.
	Duration *datex.Period `json:"duration"`
	CreateSubscriptionPhasePlanInput
	CreateSubscriptionPhaseCustomerInput
}

func (i CreateSubscriptionPhaseInput) Validate() error {
	if err := i.CreateSubscriptionPhasePlanInput.Validate(); err != nil {
		return err
	}

	return nil
}

type SubscriptionPhaseSpec struct {
	// Duration is not part of the Spec by design
	CreateSubscriptionPhasePlanInput     `json:",inline"`
	CreateSubscriptionPhaseCustomerInput `json:",inline"`

	// In each key, for each phase, we have a list of item specs to account for mid-phase changes
	ItemsByKey map[string][]*SubscriptionItemSpec `json:"itemsByKey"`

	Discounts []DiscountSpec `json:"discounts"`
}

func (s SubscriptionPhaseSpec) ToCreateSubscriptionPhaseEntityInput(
	subscription Subscription,
	activeFrom time.Time,
) CreateSubscriptionPhaseEntityInput {
	return CreateSubscriptionPhaseEntityInput{
		ActiveFrom: activeFrom,
		NamespacedModel: models.NamespacedModel{
			Namespace: subscription.Namespace,
		},
		AnnotatedModel: s.AnnotatedModel,
		SubscriptionID: subscription.ID,
		Key:            s.PhaseKey,
		Name:           s.Name,
		Description:    s.Description,
		StartAfter:     s.StartAfter,
	}
}

func (s *SubscriptionPhaseSpec) Validate() error {
	var errs []error

	// Let's validate that the phase is not empty
	flat := lo.Flatten(lo.Values(s.ItemsByKey))
	if len(flat) == 0 {
		errs = append(errs, &AllowedDuringApplyingPatchesError{
			Inner: &SpecValidationError{
				AffectedKeys: [][]string{
					{
						"phaseKey",
						s.PhaseKey,
					},
				},
				Msg: "Phase must have at least one item",
			},
		})
	}

	for key, items := range s.ItemsByKey {
		for _, item := range items {
			// Let's validate key is correct
			if item.ItemKey != key {
				errs = append(errs, &SpecValidationError{
					AffectedKeys: [][]string{
						{
							"phaseKey",
							s.PhaseKey,
							"itemKey",
							key,
						},
					},
					Msg: "Items must be grouped correctly by key",
				})
			}

			// Let's validate the phase linking is correct
			if item.PhaseKey != s.PhaseKey {
				errs = append(errs, &SpecValidationError{
					AffectedKeys: [][]string{
						{
							"phaseKey",
							s.PhaseKey,
						},
						{
							"phaseKey",
							s.PhaseKey,
							"itemKey",
							item.ItemKey,
							"PhaseKey",
						},
					},
					Msg: "PhaseKey in Item must match Key in Phase",
				})
			}

			// Let's validate the item contents
			if err := item.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("item %s validation failed: %w", item.ItemKey, err))
			}

			// TODO: Let's validate that BillingCadence aligns with phase length
			// TODO: Let's validate that Entitlement UsagePeriod aligns with phase length

			// Example code:

			// 	if upISO := s.CreateEntitlementInput.UsagePeriodISODuration; upISO != nil && s.expectedPhaseDurationISO != nil {
			// 		align, err := datex.PeriodsAlign(*s.expectedPhaseDurationISO, *upISO)
			// 		if err != nil {
			// 			return fmt.Errorf("failed to check if periods align: %w", err)
			// 		}
			// 		if !align {
			// 			return &SpecValidationError{
			// 				AffectedKeys: [][]string{
			// 					{
			// 						"phaseKey",
			// 						s.PhaseKey,
			// 						"itemKey",
			// 						s.ItemKey,
			// 						"CreateEntitlementInput",
			// 						"UsagePeriodISODuration",
			// 					},
			// 				},
			// 				Msg: "Entitlement Usage Period must align with Phase duration",
			// 			}
			// 		}
			// 	}
			// }
		}

		// Let's validate the item ordering
		// We don't know nor need to know the correct phase cadence as long as we use a consistent one
		// Were the items valid for an indefinitely long phase they would be valid for any phase,
		// as that behavior is handled by item.GetCadence.
		// FIXME: though this is correct it is not elegant
		somePhaseCadence := models.CadencedModel{
			ActiveFrom: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		// Let's validate that the items form a valid non-overlapping timeline
		cadences := make([]models.CadencedModel, 0, len(items))
		for i := range items {
			cadence, err := items[i].GetCadence(somePhaseCadence)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to get cadence for item %s: %w", items[i].ItemKey, err))
			}
			cadences = append(cadences, cadence)
		}

		timeline := models.Timeline[models.CadencedModel](cadences)

		if !timeline.IsSorted() {
			errs = append(errs, fmt.Errorf("items for key %s are not sorted", key))
		}

		if overlaps := timeline.GetOverlaps(); len(overlaps) > 0 {
			errs = append(errs, fmt.Errorf("items for key %s are overlapping: %v", key, overlaps))
		}
	}

	for idx, discount := range s.Discounts {
		if err := discount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("discount %d validation failed: %w", idx, err))
		}

		// TODO(discounts): Validate discount-item linking check
		// We need to check that
		// 1. the item with the provided key is present
		for _, key := range discount.Discount.RateCardKeys() {
			if _, ok := s.ItemsByKey[key]; !ok {
				errs = append(errs, &AllowedDuringApplyingPatchesError{Inner: fmt.Errorf("discount %d references non-existing item key %s", idx, key)})
			}
		}
	}

	return errors.Join(errs...)
}

type DiscountSpec struct {
	PhaseKey string `json:"phaseKey"`
	Discount productcatalog.Discount
	CadenceOverrideRelativeToPhaseStart
}

func (d DiscountSpec) Validate() error {
	if err := d.Discount.Validate(); err != nil {
		return fmt.Errorf("discount validation failed: %w", err)
	}

	return nil
}

type CreateSubscriptionItemPlanInput struct {
	PhaseKey string   `json:"phaseKey"`
	ItemKey  string   `json:"itemKey"`
	RateCard RateCard `json:"rateCard"`
}

type CreateSubscriptionItemCustomerInput struct {
	CadenceOverrideRelativeToPhaseStart
}

type CreateSubscriptionItemInput struct {
	CreateSubscriptionItemPlanInput     `json:",inline"`
	CreateSubscriptionItemCustomerInput `json:",inline"`
}

type SubscriptionItemSpec struct {
	CreateSubscriptionItemInput `json:",inline"`
}

func (s SubscriptionItemSpec) GetCadence(phaseCadence models.CadencedModel) (models.CadencedModel, error) {
	return s.CreateSubscriptionItemCustomerInput.CadenceOverrideRelativeToPhaseStart.GetCadence(phaseCadence), nil
}

func (s SubscriptionItemSpec) ToCreateSubscriptionItemEntityInput(
	phaseID models.NamespacedID,
	phaseCadence models.CadencedModel,
	entitlement *entitlement.Entitlement,
) (CreateSubscriptionItemEntityInput, error) {
	itemCadence, err := s.GetCadence(phaseCadence)
	if err != nil {
		return CreateSubscriptionItemEntityInput{}, fmt.Errorf("failed to get cadence for item %s: %w", s.ItemKey, err)
	}

	res := CreateSubscriptionItemEntityInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: phaseID.Namespace,
		},
		CadencedModel:                          itemCadence,
		ActiveFromOverrideRelativeToPhaseStart: s.CreateSubscriptionItemCustomerInput.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride,
		ActiveToOverrideRelativeToPhaseStart:   s.CreateSubscriptionItemCustomerInput.CadenceOverrideRelativeToPhaseStart.ActiveToOverride,
		PhaseID:                                phaseID.ID,
		Key:                                    s.ItemKey,
		RateCard:                               s.CreateSubscriptionItemPlanInput.RateCard,
		Name:                                   s.RateCard.Name,
		Description:                            s.RateCard.Description,
	}

	if entitlement != nil {
		res.EntitlementID = &entitlement.ID
	}

	return res, nil
}

func (s SubscriptionItemSpec) ToScheduleSubscriptionEntitlementInput(
	cust customerentity.Customer,
	cadence models.CadencedModel,
) (ScheduleSubscriptionEntitlementInput, bool, error) {
	var def ScheduleSubscriptionEntitlementInput

	meta := s.RateCard

	if meta.EntitlementTemplate == nil {
		return def, false, nil
	}

	if meta.FeatureKey == nil {
		return def, true, fmt.Errorf("feature is required for rate card where entitlement is present: %s", s.ItemKey)
	}

	t := meta.EntitlementTemplate.Type()
	scheduleInput := entitlement.CreateEntitlementInputs{
		EntitlementType: t,
		Namespace:       cust.Namespace,
		ActiveFrom:      lo.ToPtr(cadence.ActiveFrom),
		ActiveTo:        cadence.ActiveTo,
		FeatureKey:      meta.FeatureKey,
		SubjectKey:      cust.UsageAttribution.SubjectKeys[0], // FIXME: This is error prone
	}

	switch t {
	case entitlement.EntitlementTypeBoolean:
		tpl, err := meta.EntitlementTemplate.AsBoolean()
		if err != nil {
			return def, true, fmt.Errorf("failed to get boolean entitlement template: %w", err)
		}
		scheduleInput.Metadata = tpl.Metadata
	case entitlement.EntitlementTypeStatic:
		tpl, err := meta.EntitlementTemplate.AsStatic()
		if err != nil {
			return def, true, fmt.Errorf("failed to get static entitlement template: %w", err)
		}
		scheduleInput.Metadata = tpl.Metadata
		scheduleInput.Config = tpl.Config
	case entitlement.EntitlementTypeMetered:
		tpl, err := meta.EntitlementTemplate.AsMetered()
		if err != nil {
			return def, true, fmt.Errorf("failed to get metered entitlement template: %w", err)
		}
		truncatedStartTime := cadence.ActiveFrom.Truncate(time.Minute)

		scheduleInput.Metadata = tpl.Metadata
		scheduleInput.IsSoftLimit = &tpl.IsSoftLimit
		scheduleInput.IssueAfterReset = tpl.IssueAfterReset
		scheduleInput.IssueAfterResetPriority = tpl.IssueAfterResetPriority
		scheduleInput.PreserveOverageAtReset = tpl.PreserveOverageAtReset
		rec, err := recurrence.FromISODuration(&tpl.UsagePeriod, truncatedStartTime)
		if err != nil {
			return def, true, fmt.Errorf("failed to get recurrence from ISO duration: %w", err)
		}
		scheduleInput.UsagePeriod = lo.ToPtr(entitlement.UsagePeriod(rec))
		mu := &entitlement.MeasureUsageFromInput{}
		err = mu.FromTime(truncatedStartTime)
		if err != nil {
			return def, true, fmt.Errorf("failed to get measure usage from time: %w", err)
		}
		scheduleInput.MeasureUsageFrom = mu
	default:
		return def, true, fmt.Errorf("unsupported entitlement type %s", t)
	}

	return ScheduleSubscriptionEntitlementInput{
		CreateEntitlementInputs: scheduleInput,
	}, true, nil
}

func (s SubscriptionItemSpec) GetRef(subId string) SubscriptionItemRef {
	return SubscriptionItemRef{
		SubscriptionId: subId,
		PhaseKey:       s.PhaseKey,
		ItemKey:        s.ItemKey,
	}
}

func (s *SubscriptionItemSpec) Validate() error {
	var errs []error
	// TODO: if the price is usage based, we have to validate that that the feature is metered
	// TODO: if the entitlement is metered, we have to validate that the feature is metered

	// Let's validate the key
	if s.RateCard.FeatureKey != nil {
		if s.ItemKey != *s.RateCard.FeatureKey {
			return fmt.Errorf("feature key must match item key when a feature is defined, to avoid duplicate feature assignment")
		}
	}

	// Let's validate nested models
	if err := s.RateCard.Validate(); err != nil {
		errs = append(errs, &SpecValidationError{
			AffectedKeys: [][]string{
				{
					"phaseKey",
					s.PhaseKey,
					"itemKey",
					s.ItemKey,
					"RateCard",
				},
			},
			Msg: fmt.Sprintf("RateCard validation failed: %s", err),
		})
	}

	return errors.Join(errs...)
}

// NewSpecFromPlan creates a SubscriptionSpec from a Plan and a CreateSubscriptionCustomerInput.
func NewSpecFromPlan(p Plan, c CreateSubscriptionCustomerInput) (SubscriptionSpec, error) {
	spec := SubscriptionSpec{
		CreateSubscriptionPlanInput:     p.ToCreateSubscriptionPlanInput(),
		CreateSubscriptionCustomerInput: c,
		Phases:                          make(map[string]*SubscriptionPhaseSpec),
	}

	// Let's find an intelligent name by which we can refer to the plan in contextual errors
	planRefName := "custom plan"

	if ref := p.ToCreateSubscriptionPlanInput().Plan; ref != nil {
		planRefName = fmt.Sprintf("plan %s version %d", ref.Key, ref.Version)
	}

	if len(p.GetPhases()) == 0 {
		return spec, fmt.Errorf("%s has no phases", planRefName)
	}

	// Validate that the plan phases are returned in order
	planPhases := p.GetPhases()
	for i := range planPhases {
		if i == 0 {
			continue
		}
		if diff, err := planPhases[i].ToCreateSubscriptionPhasePlanInput().StartAfter.Subtract(planPhases[i-1].ToCreateSubscriptionPhasePlanInput().StartAfter); err != nil || diff.IsNegative() {
			return spec, fmt.Errorf("phases %s and %s of %s are in the wrong order", planPhases[i].GetKey(), planPhases[i-1].GetKey(), planRefName)
		}
	}

	for _, planPhase := range planPhases {
		if _, ok := spec.Phases[planPhase.GetKey()]; ok {
			return spec, fmt.Errorf("phase %s of %s is duplicated", planPhase.GetKey(), planRefName)
		}

		createSubscriptionPhasePlanInput := planPhase.ToCreateSubscriptionPhasePlanInput()

		phase := &SubscriptionPhaseSpec{
			CreateSubscriptionPhasePlanInput: createSubscriptionPhasePlanInput,
			CreateSubscriptionPhaseCustomerInput: CreateSubscriptionPhaseCustomerInput{
				AnnotatedModel: models.AnnotatedModel{}, // TODO: where should we source this from? inherit from PlanPhase, or Subscription?
			},
			ItemsByKey: make(map[string][]*SubscriptionItemSpec),
		}

		if len(planPhase.GetRateCards()) == 0 {
			return spec, fmt.Errorf("phase %s of %s has no rate cards", phase.PhaseKey, planRefName)
		}

		// We expect that in a plan phase, each rate card is unique by key, so let's validate that
		rcByKey := make(map[string]struct{})

		for _, rateCard := range planPhase.GetRateCards() {
			if _, ok := rcByKey[rateCard.GetKey()]; ok {
				return spec, fmt.Errorf("rate card %s of phase %s of %s is duplicated", rateCard.GetKey(), phase.PhaseKey, planRefName)
			}
			rcByKey[rateCard.GetKey()] = struct{}{}

			createSubscriptionItemPlanInput := rateCard.ToCreateSubscriptionItemPlanInput()
			itemSpec := SubscriptionItemSpec{
				CreateSubscriptionItemInput: CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput:     createSubscriptionItemPlanInput,
					CreateSubscriptionItemCustomerInput: CreateSubscriptionItemCustomerInput{},
				},
			}

			if phase.ItemsByKey[rateCard.GetKey()] == nil {
				phase.ItemsByKey[rateCard.GetKey()] = make([]*SubscriptionItemSpec, 0)
			}
			phase.ItemsByKey[rateCard.GetKey()] = append(phase.ItemsByKey[rateCard.GetKey()], &itemSpec)
		}

		spec.Phases[phase.PhaseKey] = phase
	}

	// Lets validate the spec
	if err := spec.Validate(); err != nil {
		return spec, fmt.Errorf("spec validation failed: %w", err)
	}

	return spec, nil
}

type ApplyContext struct {
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
			if uw, ok := err.(interface{ Unwrap() []error }); ok {
				// If all returned errors are allowed during applying patches, we can continue
				if lo.EveryBy(uw.Unwrap(), func(e error) bool {
					_, ok := lo.ErrorsAs[*AllowedDuringApplyingPatchesError](e)
					return ok
				}) {
					continue
				}
			}
			// Otherwise we return with the error
			return fmt.Errorf("patch %d failed during validation: %w", i, err)
		}
	}

	if err := s.Validate(); err != nil {
		return fmt.Errorf("final validation failed when applying patches: %w", err)
	}

	return nil
}

// Some errors are allowed during applying individual patches, but still mean the Spec as a whole is invalid
type AllowedDuringApplyingPatchesError struct {
	Inner error
}

func (e *AllowedDuringApplyingPatchesError) Error() string {
	return fmt.Sprintf("allowed during incremental validation failed: %s", e.Inner)
}

func (e *AllowedDuringApplyingPatchesError) Unwrap() error {
	return e.Inner
}

type SpecValidationError struct {
	// TODO: This spec is broken and painful, lets improve it
	AffectedKeys [][]string
	Msg          string
}

func (e *SpecValidationError) Error() string {
	return e.Msg
}
