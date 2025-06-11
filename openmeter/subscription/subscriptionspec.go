package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
	productcatalog.Alignment

	// BillingCadence is the default billing cadence for subscriptions.
	BillingCadence isodate.Period `json:"billing_cadence"`

	// ProRatingConfig is the default pro-rating configuration for subscriptions.
	ProRatingConfig productcatalog.ProRatingConfig `json:"pro_rating_config"`
}

type CreateSubscriptionCustomerInput struct {
	models.MetadataModel `json:",inline"`
	Name                 string         `json:"name"`
	Description          *string        `json:"description,omitempty"`
	CustomerId           string         `json:"customerId"`
	Currency             currencyx.Code `json:"currency"`
	ActiveFrom           time.Time      `json:"activeFrom,omitempty"`
	ActiveTo             *time.Time     `json:"activeTo,omitempty"`
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
		Alignment:       s.Alignment,
		Plan:            s.Plan,
		CustomerId:      s.CustomerId,
		Currency:        s.Currency,
		BillingCadence:  s.BillingCadence,
		ProRatingConfig: s.ProRatingConfig,
		MetadataModel:   s.MetadataModel,
		Name:            s.Name,
		Description:     s.Description,
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
		diff := iTime.Compare(jTime)

		if diff != 0 {
			return diff
		}

		// We do a best effort tie-breaker

		// SortHint "should" be present for all these cases
		if i.SortHint != nil && j.SortHint != nil {
			diff = int(*i.SortHint) - int(*j.SortHint)
		}

		if diff != 0 {
			return diff
		}

		// We still want this to be deterministic so we use phase key as a last resort
		return strings.Compare(i.PhaseKey, j.PhaseKey)
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

func (s *SubscriptionSpec) HasEntitlements() bool {
	return lo.SomeBy(lo.Values(s.Phases), func(p *SubscriptionPhaseSpec) bool {
		return p.HasEntitlements()
	})
}

func (s *SubscriptionSpec) HasBillables() bool {
	return lo.SomeBy(lo.Values(s.Phases), func(p *SubscriptionPhaseSpec) bool {
		return p.HasBillables()
	})
}

func (s *SubscriptionSpec) HasMeteredBillables() bool {
	return lo.SomeBy(lo.Values(s.Phases), func(p *SubscriptionPhaseSpec) bool {
		return p.HasMeteredBillables()
	})
}

// For a phase in an Aligned subscription, there's a single aligned BillingPeriod for all items in that phase.
// The period starts with the phase and iterates every subscription.BillingCadence duration, but can be reanchored to the time of an edit.
func (s *SubscriptionSpec) GetAlignedBillingPeriodAt(phaseKey string, at time.Time) (timeutil.ClosedPeriod, error) {
	var def timeutil.ClosedPeriod

	phase, exists := s.Phases[phaseKey]
	if !exists {
		return def, fmt.Errorf("phase %s not found", phaseKey)
	}

	if !s.Alignment.BillablesMustAlign {
		return def, AlignmentError{Inner: fmt.Errorf("non-aligned subscription doesn't have recurring billing cadence")}
	}

	phaseCadence, err := s.GetPhaseCadence(phaseKey)
	if err != nil {
		return def, fmt.Errorf("failed to get phase cadence for phase %s: %w", phaseKey, err)
	}

	subCad := models.CadencedModel{
		ActiveFrom: s.ActiveFrom,
		ActiveTo:   s.ActiveTo,
	}

	switch {
	case subCad.IsActiveAt(at):
		if !phaseCadence.IsActiveAt(at) {
			return def, fmt.Errorf("phase %s is not active at %s, ", phaseKey, at)
		}
	case at.Before(subCad.ActiveFrom):
		return def, fmt.Errorf("at %s is before the subscription active from %s, cannot calculate billing period", at, subCad.ActiveFrom)
	default:
		// We allow querying billing period after the subscription is inactive
	}

	if err := phase.Validate(phaseCadence, s.Alignment); err != nil {
		return def, fmt.Errorf("phase %s validation failed: %w", phaseKey, err)
	}

	if s.BillingCadence.IsZero() {
		return def, NoBillingPeriodError{Inner: fmt.Errorf("subscription has no billing cadence")}
	}

	dur := s.BillingCadence

	// Reanchoring is only possible by billables
	billables := phase.GetBillableItemsByKey()

	faltBillables := lo.Flatten(lo.Values(billables))
	recurringFlatBillables := lo.Filter(faltBillables, func(i *SubscriptionItemSpec, _ int) bool {
		return i.RateCard.GetBillingCadence() != nil
	})

	// To find the period anchor, we need to know if any item serves as a reanchor point (RestartBillingPeriod)
	reanchoringItems := lo.Filter(recurringFlatBillables, func(i *SubscriptionItemSpec, _ int) bool {
		return i.BillingBehaviorOverride.RestartBillingPeriod != nil && *i.BillingBehaviorOverride.RestartBillingPeriod
	})

	reanchoringItems = lo.UniqBy(reanchoringItems, func(i *SubscriptionItemSpec) *isodate.Period { return i.ActiveFromOverrideRelativeToPhaseStart })

	anchorTimes := []time.Time{phaseCadence.ActiveFrom}
	anchorTimes = append(anchorTimes, lo.Map(reanchoringItems, func(i *SubscriptionItemSpec, _ int) time.Time { return i.GetCadence(phaseCadence).ActiveFrom })...)

	// Let's sort in descending
	slices.SortFunc(anchorTimes, func(i, j time.Time) int { return -i.Compare(j) })

	// Anchor is the anchor time to be used at the queried time
	anchor := phaseCadence.ActiveFrom

	for _, anc := range anchorTimes {
		// Lets find the first thats not after the time
		if !anc.After(at) {
			anchor = anc
			break
		}
	}

	// Now let's sort in ascending and find if there's a reanchor point after the queried time
	slices.SortFunc(anchorTimes, func(i, j time.Time) int { return i.Compare(j) })

	var reanchor *time.Time
	for _, anc := range anchorTimes {
		if anc.After(at) {
			reanchor = &anc
			break
		}
	}

	recurrenceOfAnchor, err := timeutil.RecurrenceFromISODuration(&dur, anchor)
	if err != nil {
		return def, fmt.Errorf("failed to get recurrence from ISO duration: %w", err)
	}

	period, err := recurrenceOfAnchor.GetPeriodAt(at)
	if err != nil {
		return def, fmt.Errorf("failed to get period at %s: %w", at, err)
	}

	// If the phase ends we have to truncate the period (this also includes the subscription end)
	if phaseCadence.ActiveTo != nil && phaseCadence.ActiveTo.Before(period.To) {
		period.To = *phaseCadence.ActiveTo
	}

	// If there's a reanchor we have to truncate the period
	if reanchor != nil && reanchor.Before(period.To) {
		period.To = *reanchor
	}

	return period, nil
}

// SyncAnnotations serves as a central place where we can calculate annotation default for the Subscription contents
func (s *SubscriptionSpec) SyncAnnotations() error {
	for _, phase := range s.GetSortedPhases() {
		if err := phase.SyncAnnotations(); err != nil {
			return fmt.Errorf("failed to sync annotations for phase %s: %w", phase.PhaseKey, err)
		}
	}

	return nil
}

func (s *SubscriptionSpec) Validate() error {
	// All consistency checks should happen here
	var errs []error

	sortedPhases := s.GetSortedPhases()
	for idx, phase := range sortedPhases {
		// Let's validate that if there are phases with the same start time, they have sort hint present
		if idx > 0 {
			prevPhase := sortedPhases[idx-1]
			if prevPhase.StartAfter.Equal(&phase.StartAfter) {
				if phase.SortHint == nil || prevPhase.SortHint == nil {
					errs = append(errs, fmt.Errorf("phase %s has the same start time as phase %s but no sort hint", phase.PhaseKey, prevPhase.PhaseKey))
				}
			}
		}

		cadence, err := s.GetPhaseCadence(phase.PhaseKey)
		if err != nil {
			errs = append(errs, fmt.Errorf("during validating spec failed to get phase cadence for phase %s: %w", phase.PhaseKey, err))
			continue
		}

		if err := phase.Validate(cadence, s.Alignment); err != nil {
			errs = append(errs, fmt.Errorf("phase %s validation failed: %w", phase.PhaseKey, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *SubscriptionSpec) ValidateAlignment() error {
	if !s.Alignment.BillablesMustAlign {
		return nil
	}

	var errs []error

	for _, phase := range s.GetSortedPhases() {
		for _, itemsByKey := range phase.GetBillableItemsByKey() {
			for _, item := range itemsByKey {
				rateCard := item.RateCard
				if rateCard.GetBillingCadence() != nil {
					if err := productcatalog.ValidateBillingCadencesAlign(s.BillingCadence, lo.FromPtr(rateCard.GetBillingCadence())); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateSubscriptionPhasePlanInput struct {
	PhaseKey    string         `json:"key"`
	StartAfter  isodate.Period `json:"startAfter"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	SortHint    *uint8         `json:"sortHint,omitempty"`
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
	models.MetadataModel `json:",inline"`
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
	Duration *isodate.Period `json:"duration"`
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
		MetadataModel:  s.MetadataModel,
		SubscriptionID: subscription.ID,
		Key:            s.PhaseKey,
		Name:           s.Name,
		Description:    s.Description,
		StartAfter:     s.StartAfter,
		SortHint:       s.SortHint,
	}
}

// GetBillableItemsByKey returns a map of billable items by key
func (s SubscriptionPhaseSpec) GetBillableItemsByKey() map[string][]*SubscriptionItemSpec {
	res := make(map[string][]*SubscriptionItemSpec)
	for key, items := range s.ItemsByKey {
		for _, item := range items {
			if item.RateCard.AsMeta().Price != nil {
				if res[key] == nil {
					res[key] = make([]*SubscriptionItemSpec, 0)
				}
				res[key] = append(res[key], item)
			}
		}
	}
	return res
}

func (s SubscriptionPhaseSpec) HasEntitlements() bool {
	return lo.SomeBy(lo.Flatten(lo.Values(s.ItemsByKey)), func(item *SubscriptionItemSpec) bool {
		return item.RateCard.AsMeta().EntitlementTemplate != nil
	})
}

func (s SubscriptionPhaseSpec) HasMeteredBillables() bool {
	return lo.SomeBy(lo.Flatten(lo.Values(s.ItemsByKey)), func(item *SubscriptionItemSpec) bool {
		return item.RateCard.AsMeta().Price != nil && item.RateCard.AsMeta().Price.Type() != productcatalog.FlatPriceType
	})
}

func (s SubscriptionPhaseSpec) HasBillables() bool {
	return len(s.GetBillableItemsByKey()) > 0
}

func (s SubscriptionPhaseSpec) SyncAnnotations() error {
	for _, items := range s.ItemsByKey {
		for idx, item := range items {
			if err := item.SyncAnnotations(); err != nil {
				return fmt.Errorf("failed to sync annotations for item %s at index %d: %w", item.ItemKey, idx, err)
			}
		}
	}

	return nil
}

func (s SubscriptionPhaseSpec) Validate(
	phaseCadence models.CadencedModel,
	alignment productcatalog.Alignment,
) error {
	var errs []error

	// Phase StartAfter really should not be negative
	if s.StartAfter.IsNegative() {
		errs = append(errs, fmt.Errorf("phase start after cannot be negative"))
	}

	// Let's validate that the phase is not empty
	flat := lo.Flatten(lo.Values(s.ItemsByKey))
	if len(flat) == 0 {
		errs = append(errs, &AllowedDuringApplyingToSpecError{
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
		}

		// Let's validate that the items form a valid non-overlapping timeline
		cadences := make([]models.CadencedModel, 0, len(items))
		for i := range items {
			cadence := items[i].GetCadence(phaseCadence)
			cadences = append(cadences, cadence)
		}

		timeline := models.CadenceList[models.CadencedModel](cadences)

		if !timeline.IsSorted() {
			errs = append(errs, fmt.Errorf("items for key %s are not sorted", key))
		}

		if overlaps := timeline.GetOverlaps(); len(overlaps) > 0 {
			for _, overlap := range overlaps {
				itemSpec1 := items[overlap.Index1]
				itemSpec2 := items[overlap.Index2]
				errs = append(errs, fmt.Errorf(
					"items for key %s are overlapping (indexes %d and %d): reason: %s. Item 1 Spec: %+v, Cadence: %+v. Item 2 Spec: %+v, Cadence: %+v",
					key,
					overlap.Index1,
					overlap.Index2,
					overlap.Reason,
					itemSpec1,
					overlap.Item1,
					itemSpec2,
					overlap.Item2,
				))
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

type CreateSubscriptionItemPlanInput struct {
	PhaseKey string                  `json:"phaseKey"`
	ItemKey  string                  `json:"itemKey"`
	RateCard productcatalog.RateCard `json:"rateCard"`
}

func (i *CreateSubscriptionItemPlanInput) UnmarshalJSON(b []byte) error {
	var serdeTyp struct {
		RateCard productcatalog.RateCardSerde `json:"rateCard"`
	}

	if err := json.Unmarshal(b, &serdeTyp); err != nil {
		return fmt.Errorf("failed to JSON deserialize SubscriptionItemSpec: %w", err)
	}

	serde := struct {
		RateCard productcatalog.RateCard
		PhaseKey string `json:"phaseKey"`
		ItemKey  string `json:"itemKey"`
	}{
		RateCard: i.RateCard,
		PhaseKey: i.PhaseKey,
		ItemKey:  i.ItemKey,
	}

	switch serdeTyp.RateCard.Type {
	case productcatalog.FlatFeeRateCardType:
		serde.RateCard = &productcatalog.FlatFeeRateCard{}
	case productcatalog.UsageBasedRateCardType:
		serde.RateCard = &productcatalog.UsageBasedRateCard{}
	default:
		return fmt.Errorf("invalid RateCard type: %s", serdeTyp.RateCard.Type)
	}

	if err := json.Unmarshal(b, &serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize SubscriptionItemPlanInput: %w", err)
	}

	i.RateCard = serde.RateCard
	i.PhaseKey = serde.PhaseKey
	i.ItemKey = serde.ItemKey

	return nil
}

type CreateSubscriptionItemCustomerInput struct {
	ActiveFromOverrideRelativeToPhaseStart *isodate.Period `json:"activeFromOverrideRelativeToPhaseStart"`
	ActiveToOverrideRelativeToPhaseStart   *isodate.Period `json:"activeToOverrideRelativeToPhaseStart,omitempty"`
	BillingBehaviorOverride
}

type CreateSubscriptionItemInput struct {
	Annotations                         models.Annotations `json:"annotations"`
	CreateSubscriptionItemPlanInput     `json:",inline"`
	CreateSubscriptionItemCustomerInput `json:",inline"`
}

type SubscriptionItemSpec struct {
	CreateSubscriptionItemInput `json:",inline"`
}

func (s SubscriptionItemSpec) GetCadence(phaseCadence models.CadencedModel) models.CadencedModel {
	start := phaseCadence.ActiveFrom
	if s.ActiveFromOverrideRelativeToPhaseStart != nil {
		start, _ = s.ActiveFromOverrideRelativeToPhaseStart.AddTo(phaseCadence.ActiveFrom)
	}

	if phaseCadence.ActiveTo != nil {
		if phaseCadence.ActiveTo.Before(start) {
			// If the intended start time is after the intended end time of the phase, the item will have 0 lifetime at the end of the phase
			// This scenario is possible when Subscriptions are canceled (before the phase ends)
			return models.CadencedModel{
				ActiveFrom: *phaseCadence.ActiveTo,
				ActiveTo:   phaseCadence.ActiveTo,
			}
		}
	}

	end := phaseCadence.ActiveTo

	if s.ActiveToOverrideRelativeToPhaseStart != nil {
		endTime, _ := s.ActiveToOverrideRelativeToPhaseStart.AddTo(phaseCadence.ActiveFrom)

		if phaseCadence.ActiveTo != nil && phaseCadence.ActiveTo.Before(endTime) {
			// Phase Cadence overrides item cadence in all cases
			endTime = *phaseCadence.ActiveTo
		}

		end = &endTime
	}

	return models.CadencedModel{
		ActiveFrom: start,
		ActiveTo:   end,
	}
}

// GetFullServicePeriodAt returns the full service period for an item at a given time
// To get the de-facto service period, use the intersection of the item's activity with the returned period.
func (s SubscriptionItemSpec) GetFullServicePeriodAt(
	phaseCadence models.CadencedModel,
	itemCadence models.CadencedModel,
	at time.Time,
	alignedBillingAnchor *time.Time,
) (timeutil.ClosedPeriod, error) {
	if !s.RateCard.AsMeta().IsBillable() {
		return timeutil.ClosedPeriod{}, fmt.Errorf("item is not billable")
	}

	if !itemCadence.IsActiveAt(at) {
		return timeutil.ClosedPeriod{}, fmt.Errorf("item is not active at %s", at)
	}

	if !phaseCadence.IsActiveAt(at) {
		return timeutil.ClosedPeriod{}, fmt.Errorf("phase is not active at %s", at)
	}

	billingCadence := s.RateCard.GetBillingCadence()
	if billingCadence == nil {
		end := itemCadence.ActiveFrom

		if itemCadence.ActiveTo != nil {
			end = *itemCadence.ActiveTo
		}

		if phaseCadence.ActiveTo != nil {
			end = *phaseCadence.ActiveTo
		}

		return timeutil.ClosedPeriod{
			From: itemCadence.ActiveFrom,
			To:   end,
		}, nil
	}

	billingAnchor := lo.FromPtrOr(alignedBillingAnchor, itemCadence.ActiveFrom)

	rec, err := timeutil.RecurrenceFromISODuration(billingCadence, billingAnchor)
	if err != nil {
		return timeutil.ClosedPeriod{}, fmt.Errorf("failed to get recurrence from ISO duration: %w", err)
	}

	return rec.GetPeriodAt(at)
}

func (s SubscriptionItemSpec) ToCreateSubscriptionItemEntityInput(
	phaseID models.NamespacedID,
	phaseCadence models.CadencedModel,
	entitlement *entitlement.Entitlement,
) (CreateSubscriptionItemEntityInput, error) {
	itemCadence := s.GetCadence(phaseCadence)

	res := CreateSubscriptionItemEntityInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: phaseID.Namespace,
		},
		Annotations:                            s.Annotations,
		CadencedModel:                          itemCadence,
		ActiveFromOverrideRelativeToPhaseStart: s.CreateSubscriptionItemCustomerInput.ActiveFromOverrideRelativeToPhaseStart,
		ActiveToOverrideRelativeToPhaseStart:   s.CreateSubscriptionItemCustomerInput.ActiveToOverrideRelativeToPhaseStart,
		PhaseID:                                phaseID.ID,
		Key:                                    s.ItemKey,
		RateCard:                               s.CreateSubscriptionItemPlanInput.RateCard,
		Name:                                   s.RateCard.AsMeta().Name,
		Description:                            s.RateCard.AsMeta().Description,
		BillingBehaviorOverride:                s.BillingBehaviorOverride,
	}

	if entitlement != nil {
		res.EntitlementID = &entitlement.ID
	}

	return res, nil
}

type ToScheduleSubscriptionEntitlementInputOptions struct {
	Customer     customer.Customer
	Cadence      models.CadencedModel
	PhaseCadence models.CadencedModel
	IsAligned    bool
}

func (s SubscriptionItemSpec) ToScheduleSubscriptionEntitlementInput(
	opts ToScheduleSubscriptionEntitlementInputOptions,
) (ScheduleSubscriptionEntitlementInput, bool, error) {
	var def ScheduleSubscriptionEntitlementInput

	meta := s.RateCard.AsMeta()

	if meta.EntitlementTemplate == nil {
		return def, false, nil
	}

	if meta.FeatureKey == nil {
		return def, true, fmt.Errorf("feature is required for rate card where entitlement is present: %s", s.ItemKey)
	}

	t := meta.EntitlementTemplate.Type()
	subjectKey, err := opts.Customer.UsageAttribution.GetSubjectKey()
	if err != nil {
		return def, true, fmt.Errorf("failed to get subject key for customer %s: %w", opts.Customer.ID, err)
	}

	scheduleInput := entitlement.CreateEntitlementInputs{
		EntitlementType: t,
		Namespace:       opts.Customer.Namespace,
		ActiveFrom:      lo.ToPtr(opts.Cadence.ActiveFrom),
		ActiveTo:        opts.Cadence.ActiveTo,
		FeatureKey:      meta.FeatureKey,
		SubjectKey:      subjectKey,
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
		truncatedStartTime := opts.Cadence.ActiveFrom.Truncate(time.Minute)

		if opts.IsAligned {
			truncatedStartTime = opts.PhaseCadence.ActiveFrom.Truncate(time.Minute)
		}

		scheduleInput.Metadata = tpl.Metadata
		scheduleInput.IsSoftLimit = &tpl.IsSoftLimit
		scheduleInput.IssueAfterReset = tpl.IssueAfterReset
		scheduleInput.IssueAfterResetPriority = tpl.IssueAfterResetPriority
		scheduleInput.PreserveOverageAtReset = tpl.PreserveOverageAtReset
		rec, err := timeutil.RecurrenceFromISODuration(&tpl.UsagePeriod, truncatedStartTime)
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

func (s *SubscriptionItemSpec) SyncAnnotations() error {
	met := s.RateCard.AsMeta()

	if met.EntitlementTemplate != nil && met.EntitlementTemplate.Type() == entitlement.EntitlementTypeBoolean {
		count := AnnotationParser.GetBooleanEntitlementCount(s.Annotations)
		if count == 0 {
			if _, err := AnnotationParser.SetBooleanEntitlementCount(s.Annotations, 1); err != nil {
				return fmt.Errorf("failed to set boolean entitlement count: %w", err)
			}
		}
	}

	return nil
}

func (s *SubscriptionItemSpec) Validate() error {
	var errs []error
	// TODO: if the price is usage based, we have to validate that that the feature is metered
	// TODO: if the entitlement is metered, we have to validate that the feature is metered

	if s.RateCard == nil {
		return fmt.Errorf("rate card is required")
	}

	// Let's validate the key
	if s.RateCard.AsMeta().FeatureKey != nil {
		if s.ItemKey != *s.RateCard.AsMeta().FeatureKey {
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

	// Billing behavior should only be present for billable items
	if s.BillingBehaviorOverride.RestartBillingPeriod != nil && s.RateCard.AsMeta().Price == nil {
		errs = append(errs, fmt.Errorf("billing behavior override is only allowed for billable items"))
	}

	// The relative cadence should make sense
	if s.ActiveFromOverrideRelativeToPhaseStart != nil && s.ActiveFromOverrideRelativeToPhaseStart.IsNegative() {
		errs = append(errs, fmt.Errorf("active from override relative to phase start cannot be negative"))
	}

	if s.ActiveToOverrideRelativeToPhaseStart != nil && s.ActiveToOverrideRelativeToPhaseStart.IsNegative() {
		errs = append(errs, fmt.Errorf("active to override relative to phase start cannot be negative"))
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
				MetadataModel: models.MetadataModel{}, // TODO: where should we source this from? inherit from PlanPhase, or Subscription?
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

			annotations := models.Annotations{}
			if _, err := AnnotationParser.AddOwnerSubSystem(annotations, OwnerSubscriptionSubSystem); err != nil {
				return spec, fmt.Errorf("failed to add owner system to rate card %s of phase %s of %s: %w", rateCard.GetKey(), phase.PhaseKey, planRefName, err)
			}

			itemSpec := SubscriptionItemSpec{
				CreateSubscriptionItemInput: CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput:     createSubscriptionItemPlanInput,
					CreateSubscriptionItemCustomerInput: CreateSubscriptionItemCustomerInput{},
					Annotations:                         annotations,
				},
			}

			if phase.ItemsByKey[rateCard.GetKey()] == nil {
				phase.ItemsByKey[rateCard.GetKey()] = make([]*SubscriptionItemSpec, 0)
			}
			phase.ItemsByKey[rateCard.GetKey()] = append(phase.ItemsByKey[rateCard.GetKey()], &itemSpec)
		}

		spec.Phases[phase.PhaseKey] = phase
	}

	// Lets sync annotations for the spec
	if err := spec.SyncAnnotations(); err != nil {
		return spec, fmt.Errorf("failed to sync annotations: %w", err)
	}

	// Lets validate the spec
	if err := spec.Validate(); err != nil {
		return spec, fmt.Errorf("spec validation failed: %w", err)
	}

	return spec, nil
}

func (s *SubscriptionSpec) Apply(applies AppliesToSpec, context ApplyContext) error {
	err := applies.ApplyTo(s, context)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	if err := s.SyncAnnotations(); err != nil {
		return fmt.Errorf("failed to sync annotations: %w", err)
	}

	return s.Validate()
}

func (s *SubscriptionSpec) ApplyMany(applieses []AppliesToSpec, aCtx ApplyContext) error {
	if err := NewAggregateAppliesToSpec(applieses).ApplyTo(s, aCtx); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	if err := s.Validate(); err != nil {
		return fmt.Errorf("final validation failed when applying patches: %w", err)
	}

	return nil
}

type SpecValidationError struct {
	AffectedKeys [][]string
	Msg          string
}

func (e *SpecValidationError) Error() string {
	return e.Msg
}

// AlignmentError is an error that occurs when the spec is not aligned but we expect it to be.
type AlignmentError struct {
	Inner error
}

func (e AlignmentError) Error() string {
	return fmt.Sprintf("alignment error: %s", e.Inner)
}

func (e AlignmentError) Unwrap() error {
	return e.Inner
}

// NoBillingPeriodError is an error that occurs when a phase has no billing period.
type NoBillingPeriodError struct {
	Inner error
}

func (e NoBillingPeriodError) Error() string {
	return fmt.Sprintf("no billing period: %s", e.Inner)
}

func (e NoBillingPeriodError) Unwrap() error {
	return e.Inner
}
