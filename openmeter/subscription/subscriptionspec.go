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
	"github.com/openmeterio/openmeter/pkg/datetime"
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

	// BillingCadence is the default billing cadence for subscriptions.
	BillingCadence datetime.ISODuration `json:"billing_cadence"`

	// ProRatingConfig is the default pro-rating configuration for subscriptions.
	ProRatingConfig productcatalog.ProRatingConfig `json:"pro_rating_config"`
}

type CreateSubscriptionCustomerInput struct {
	models.MetadataModel `json:",inline"`
	Name                 string             `json:"name"`
	Description          *string            `json:"description,omitempty"`
	CustomerId           string             `json:"customerId"`
	Currency             currencyx.Code     `json:"currency"`
	ActiveFrom           time.Time          `json:"activeFrom,omitempty"`
	ActiveTo             *time.Time         `json:"activeTo,omitempty"`
	BillingAnchor        time.Time          `json:"billingAnchor,omitempty"`
	Annotations          models.Annotations `json:"annotations"`
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
		Plan:            s.Plan,
		CustomerId:      s.CustomerId,
		Currency:        s.Currency,
		BillingCadence:  s.BillingCadence,
		ProRatingConfig: s.ProRatingConfig,
		BillingAnchor:   s.BillingAnchor,
		MetadataModel:   s.MetadataModel,
		Annotations:     s.Annotations,
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
func (s *SubscriptionSpec) GetAlignedBillingPeriodAt(at time.Time) (timeutil.ClosedPeriod, error) {
	var def timeutil.ClosedPeriod

	// Let's be defensive just in case
	if s.BillingCadence.IsZero() {
		return def, fmt.Errorf("subscription has no billing cadence")
	}

	// First, let's try to find the phase at the given time.
	subCad := models.CadencedModel{
		ActiveFrom: s.ActiveFrom,
		ActiveTo:   s.ActiveTo,
	}

	var phase *SubscriptionPhaseSpec

	switch {
	// If the subscription is active at that time we'll have an active phase.
	case subCad.IsActiveAt(at):
		p, ok := s.GetCurrentPhaseAt(at)
		if !ok {
			return def, fmt.Errorf("no active phase found for active subscription at %s", at)
		}
		phase = p
	case at.Before(subCad.ActiveFrom):
		return def, NewErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(at, subCad.ActiveFrom)
	default:
		if subCad.ActiveTo == nil {
			// impossible, but lets be defensive and not panic
			return def, fmt.Errorf("subscription has no activeTo date but is not active at %s", at)
		}

		for _, p := range s.GetSortedPhases() {
			cad, err := s.GetPhaseCadence(p.PhaseKey)
			if err != nil {
				return def, fmt.Errorf("failed to get phase cadence for phase %s: %w", p.PhaseKey, err)
			}

			if cad.ActiveFrom.After(*subCad.ActiveTo) {
				break
			}

			phase = p
		}
	}

	// Let's be defensive once again
	if phase == nil {
		return def, fmt.Errorf("no phase found for subscription billing period calculation at %s", at)
	}

	// TODO(galexi, OM-1418): implement reanchoring

	// We will use the subscription billing anchor as the cadence anchor
	billingRecurrence, err := timeutil.NewRecurrenceFromISODuration(s.BillingCadence, s.BillingAnchor)
	if err != nil {
		return def, fmt.Errorf("failed to get billing recurrence for phase %s: %w", phase.PhaseKey, err)
	}

	period, err := billingRecurrence.GetPeriodAt(at)
	if err != nil {
		return def, fmt.Errorf("failed to get billing period for phase %s at %s: %w", phase.PhaseKey, at, err)
	}

	// The billing period must be contained within the phase
	phaseCadence, err := s.GetPhaseCadence(phase.PhaseKey)
	if err != nil {
		return def, fmt.Errorf("failed to get phase cadence for phase %s: %w", phase.PhaseKey, err)
	}

	if phaseCadence.ActiveTo != nil && phaseCadence.ActiveTo.Before(period.To) {
		period.To = *phaseCadence.ActiveTo
	}

	if phaseCadence.ActiveFrom.After(period.From) {
		period.From = phaseCadence.ActiveFrom
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

	// Let's validate the billing anchor
	// - is present
	if s.BillingAnchor.IsZero() {
		errs = append(errs, ErrSubscriptionBillingAnchorIsRequired)
	}

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

		if err := phase.Validate(cadence); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *SubscriptionSpec) ValidateAlignment() error {
	var errs []error

	for _, phase := range s.GetSortedPhases() {
		for _, itemsByKey := range phase.GetBillableItemsByKey() {
			for idx, item := range itemsByKey {
				fieldSelector := models.NewFieldSelectorGroup(
					phase.FieldDescriptor(),
					models.NewFieldSelector("itemsByKey"),
					models.NewFieldSelector(item.ItemKey).
						WithExpression(models.NewFieldArrIndex(idx)),
				)

				rateCard := item.RateCard
				if rateCard.GetBillingCadence() != nil {
					if err := productcatalog.ValidateBillingCadencesAlign(s.BillingCadence, lo.FromPtr(rateCard.GetBillingCadence())); err != nil {
						errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}

var _ models.CadenceComparable = SubscriptionSpec{}

func (s SubscriptionSpec) GetCadence() models.CadencedModel {
	return models.CadencedModel{
		ActiveFrom: s.ActiveFrom,
		ActiveTo:   s.ActiveTo,
	}
}

type CreateSubscriptionPhasePlanInput struct {
	PhaseKey    string               `json:"key"`
	StartAfter  datetime.ISODuration `json:"startAfter"`
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	SortHint    *uint8               `json:"sortHint,omitempty"`
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
	Duration *datetime.ISODuration `json:"duration"`
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

func (s SubscriptionPhaseSpec) FieldDescriptor() *models.FieldDescriptor {
	return models.NewFieldSelectorGroup(
		models.NewFieldSelector("phases"),
		models.NewFieldSelector(s.PhaseKey),
	).WithAttributes(models.Attributes{
		PhaseDescriptor: true,
	})
}

func (s SubscriptionPhaseSpec) Validate(
	phaseCadence models.CadencedModel,
) error {
	var errs []error

	phaseSelector := s.FieldDescriptor()

	// Phase StartAfter really should not be negative
	if s.StartAfter.IsNegative() {
		errs = append(errs, models.ErrorWithFieldPrefix(
			phaseSelector,
			ErrSubscriptionPhaseStartAfterIsNegative,
		))
	}

	// Let's validate that the phase is not empty
	flat := lo.Flatten(lo.Values(s.ItemsByKey))
	if len(flat) == 0 {
		errs = append(errs, models.ErrorWithFieldPrefix(
			phaseSelector,
			ErrSubscriptionPhaseHasNoItems.With(
				AllowedDuringApplyingToSpecError(),
			),
		))
	}

	for key, items := range s.ItemsByKey {
		for idx, item := range items {
			itemSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("itemsByKey"),
				models.NewFieldSelector(key).
					WithExpression(models.NewFieldArrIndex(idx)),
			)

			// Let's validate key is correct
			if item.ItemKey != key {
				errs = append(errs, models.ErrorWithFieldPrefix(
					itemSelector.WithPrefix(phaseSelector),
					ErrSubscriptionPhaseItemHistoryKeyMismatch,
				))
			}

			// Let's validate the phase linking is correct
			if item.PhaseKey != s.PhaseKey {
				errs = append(errs, models.ErrorWithFieldPrefix(
					itemSelector.WithPrefix(phaseSelector),
					ErrSubscriptionPhaseItemKeyMismatchWithPhaseKey,
				))
			}

			// Let's validate the item contents
			if err := item.Validate(); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(
					itemSelector.WithPrefix(phaseSelector),
					err,
				))
			}
		}

		// Let's validate that the items form a valid non-overlapping timeline
		cadences := make([]models.CadencedModel, 0, len(items))
		for i := range items {
			cadence := items[i].GetCadence(phaseCadence)
			cadences = append(cadences, cadence)
		}

		timeline := models.CadenceList[models.CadencedModel](cadences)

		// We guarantee here that the sorting of items is the same as the sorting of the timeline, which is also a correct sorting
		if !timeline.IsSorted() {
			errs = append(errs, fmt.Errorf("items for key %s are not sorted", key))
		}

		if overlaps := timeline.GetOverlaps(); len(overlaps) > 0 {
			for _, overlap := range overlaps {
				itemSpec1 := items[overlap.Index1]
				itemSpec2 := items[overlap.Index2]

				// error for first item
				errs = append(errs, models.ErrorWithFieldPrefix(
					phaseSelector,
					ErrSubscriptionItemHistoryOverlap.WithField(
						models.NewFieldSelector("itemsByKey"),
						models.NewFieldSelector(key).
							WithExpression(models.NewFieldArrIndex(overlap.Index1)),
					).WithAttrs(models.Attributes{
						"overlaps_with_idx": overlap.Index2,
						"cadence":           overlap.Item1,
						"spec":              itemSpec1,
					}),
				))

				// error for second item
				errs = append(errs, models.ErrorWithFieldPrefix(
					phaseSelector,
					ErrSubscriptionItemHistoryOverlap.WithField(
						models.NewFieldSelector("itemsByKey"),
						models.NewFieldSelector(key).
							WithExpression(models.NewFieldArrIndex(overlap.Index2)),
					).WithAttrs(models.Attributes{
						"overlaps_with_idx": overlap.Index1,
						"cadence":           overlap.Item2,
						"spec":              itemSpec2,
					}),
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
	ActiveFromOverrideRelativeToPhaseStart *datetime.ISODuration `json:"activeFromOverrideRelativeToPhaseStart,omitempty"`
	ActiveToOverrideRelativeToPhaseStart   *datetime.ISODuration `json:"activeToOverrideRelativeToPhaseStart,omitempty"`
	BillingBehaviorOverride
}

func (i *CreateSubscriptionItemCustomerInput) UnmarshalJSON(b []byte) error {
	var serde struct {
		ActiveFromOverrideRelativeToPhaseStart *string `json:"activeFromOverrideRelativeToPhaseStart,omitempty"`
		ActiveToOverrideRelativeToPhaseStart   *string `json:"activeToOverrideRelativeToPhaseStart,omitempty"`
		BillingBehaviorOverride
	}

	if err := json.Unmarshal(b, &serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize CreateSubscriptionItemCustomerInput: %w", err)
	}

	var def CreateSubscriptionItemCustomerInput

	def.BillingBehaviorOverride = serde.BillingBehaviorOverride

	if serde.ActiveFromOverrideRelativeToPhaseStart != nil {
		activeFrom, err := datetime.ISODurationString(*serde.ActiveFromOverrideRelativeToPhaseStart).Parse()
		if err != nil {
			return fmt.Errorf("failed to parse active from override relative to phase start: %w", err)
		}
		def.ActiveFromOverrideRelativeToPhaseStart = &activeFrom
	}

	if serde.ActiveToOverrideRelativeToPhaseStart != nil {
		activeTo, err := datetime.ISODurationString(*serde.ActiveToOverrideRelativeToPhaseStart).Parse()
		if err != nil {
			return fmt.Errorf("failed to parse active to override relative to phase start: %w", err)
		}
		def.ActiveToOverrideRelativeToPhaseStart = &activeTo
	}

	*i = def

	return nil
}

type CreateSubscriptionItemInput struct {
	Annotations                         models.Annotations `json:"annotations"`
	CreateSubscriptionItemPlanInput     `json:",inline"`
	CreateSubscriptionItemCustomerInput `json:",inline"`
}

func (i *CreateSubscriptionItemInput) UnmarshalJSON(b []byte) error {
	var annSerde struct {
		Annotations models.Annotations `json:"annotations"`
	}

	if err := json.Unmarshal(b, &annSerde); err != nil {
		return fmt.Errorf("failed to JSON deserialize CreateSubscriptionItemInput: %w", err)
	}

	var planSerde CreateSubscriptionItemPlanInput

	if err := json.Unmarshal(b, &planSerde); err != nil {
		return fmt.Errorf("failed to JSON deserialize CreateSubscriptionItemInput: %w", err)
	}

	var customerSerde CreateSubscriptionItemCustomerInput

	if err := json.Unmarshal(b, &customerSerde); err != nil {
		return fmt.Errorf("failed to JSON deserialize CreateSubscriptionItemInput: %w", err)
	}

	i.Annotations = annSerde.Annotations
	i.CreateSubscriptionItemPlanInput = planSerde
	i.CreateSubscriptionItemCustomerInput = customerSerde

	return nil
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

type GetFullServicePeriodAtInput struct {
	SubscriptionCadence  models.CadencedModel
	PhaseCadence         models.CadencedModel
	ItemCadence          models.CadencedModel
	At                   time.Time
	AlignedBillingAnchor time.Time
}

func (i GetFullServicePeriodAtInput) isEndOfSubscription() bool {
	return lo.TernaryF(i.SubscriptionCadence.ActiveTo == nil, func() bool { return false }, func() bool { return i.SubscriptionCadence.ActiveTo.Equal(i.At) })
}

func (i GetFullServicePeriodAtInput) Validate() error {
	if i.At.IsZero() {
		return fmt.Errorf("at is zero")
	}

	if i.AlignedBillingAnchor.IsZero() {
		return fmt.Errorf("aligned billing anchor is zero")
	}

	if !i.SubscriptionCadence.AsPeriod().ContainsInclusive(i.At) {
		return fmt.Errorf("subscription is not active at %s: [%s, %s]", i.At, i.SubscriptionCadence.ActiveFrom, i.SubscriptionCadence.ActiveTo)
	}

	// We might attempt to bill these
	isEndOfSubscription := i.isEndOfSubscription()

	if !i.PhaseCadence.IsActiveAt(i.At) && !isEndOfSubscription {
		return fmt.Errorf("phase is not active at %s: [%s, %s]", i.At, i.PhaseCadence.ActiveFrom, i.PhaseCadence.ActiveTo)
	}

	// We might attempt to bill these
	isZeroLengthLastItem := i.At.Equal(i.ItemCadence.ActiveFrom) && i.ItemCadence.ActiveTo != nil && i.ItemCadence.ActiveFrom.Equal(*i.ItemCadence.ActiveTo)

	if !i.ItemCadence.IsActiveAt(i.At) && !isZeroLengthLastItem {
		return fmt.Errorf("item is not active at %s: [%s, %s]", i.At, i.ItemCadence.ActiveFrom, i.ItemCadence.ActiveTo)
	}

	return nil
}

// GetFullServicePeriodAt returns the full service period for an item at a given time
// To get the de-facto service period, use the intersection of the item's activity with the returned period.
func (s SubscriptionItemSpec) GetFullServicePeriodAt(
	inp GetFullServicePeriodAtInput,
) (timeutil.ClosedPeriod, error) {
	if err := inp.Validate(); err != nil {
		return timeutil.ClosedPeriod{}, err
	}

	billingCadence := s.RateCard.GetBillingCadence()
	if billingCadence == nil {
		end := inp.ItemCadence.ActiveFrom

		if inp.ItemCadence.ActiveTo != nil {
			end = *inp.ItemCadence.ActiveTo
		}

		if inp.PhaseCadence.ActiveTo != nil {
			end = *inp.PhaseCadence.ActiveTo
		}

		return timeutil.ClosedPeriod{
			From: inp.ItemCadence.ActiveFrom,
			To:   end,
		}, nil
	}

	rec, err := timeutil.NewRecurrenceFromISODuration(*billingCadence, inp.AlignedBillingAnchor)
	if err != nil {
		return timeutil.ClosedPeriod{}, fmt.Errorf("failed to get recurrence from ISO duration: %w", err)
	}

	return rec.GetPeriodAt(inp.At)
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
	Customer             customer.Customer
	Cadence              models.CadencedModel
	PhaseStart           time.Time
	AlignedBillingAnchor time.Time
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

	scheduleInput := entitlement.CreateEntitlementInputs{
		EntitlementType:  t,
		Namespace:        opts.Customer.Namespace,
		ActiveFrom:       lo.ToPtr(opts.Cadence.ActiveFrom),
		ActiveTo:         opts.Cadence.ActiveTo,
		FeatureKey:       meta.FeatureKey,
		UsageAttribution: opts.Customer.GetUsageAttribution(),
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

		var configJSON string

		err = json.Unmarshal(tpl.Config, &configJSON)
		if err != nil {
			return def, true, fmt.Errorf("failed to unmarshal static entitlement template config: %w", err)
		}

		scheduleInput.Config = &configJSON
	case entitlement.EntitlementTypeMetered:
		tpl, err := meta.EntitlementTemplate.AsMetered()
		if err != nil {
			return def, true, fmt.Errorf("failed to get metered entitlement template: %w", err)
		}

		if opts.AlignedBillingAnchor.IsZero() {
			return def, true, fmt.Errorf("aligned billing anchor shouldn't be zero")
		}

		truncatedAnchorTime := opts.AlignedBillingAnchor.Truncate(time.Minute)
		truncatedMeasureUsageFrom := opts.PhaseStart.Truncate(time.Minute)

		scheduleInput.Metadata = tpl.Metadata
		scheduleInput.IsSoftLimit = &tpl.IsSoftLimit
		scheduleInput.IssueAfterReset = tpl.IssueAfterReset
		scheduleInput.IssueAfterResetPriority = tpl.IssueAfterResetPriority
		scheduleInput.PreserveOverageAtReset = tpl.PreserveOverageAtReset
		rec, err := timeutil.NewRecurrenceFromISODuration(tpl.UsagePeriod, truncatedAnchorTime)
		if err != nil {
			return def, true, fmt.Errorf("failed to get recurrence from ISO duration: %w", err)
		}
		scheduleInput.UsagePeriod = lo.ToPtr(timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
			return r.Anchor
		})(rec))
		mu := &entitlement.MeasureUsageFromInput{}
		err = mu.FromTime(truncatedMeasureUsageFrom)
		if err != nil {
			return def, true, fmt.Errorf("failed to get measure usage from time: %w", err)
		}
		scheduleInput.MeasureUsageFrom = mu
	default:
		return def, true, fmt.Errorf("unsupported entitlement type %s", t)
	}

	return ScheduleSubscriptionEntitlementInput{
		CreateEntitlementInputs: scheduleInput,
		Customer:                opts.Customer,
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
			if s.Annotations == nil {
				s.Annotations = models.Annotations{}
			}

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

	// Let's validate nested models
	if err := s.RateCard.Validate(); err != nil {
		errs = append(errs, models.ErrorWithComponent("rateCard", err))
	}

	// Billing behavior should only be present for billable items
	if s.BillingBehaviorOverride.RestartBillingPeriod != nil && !s.RateCard.IsBillable() {
		errs = append(errs, ErrSubscriptionItemBillingOverrideIsOnlyAllowedForBillableItems)
	}

	// The relative cadence should make sense
	if s.ActiveFromOverrideRelativeToPhaseStart != nil && s.ActiveFromOverrideRelativeToPhaseStart.IsNegative() {
		errs = append(errs, ErrSubscriptionItemActiveFromOverrideRelativeToPhaseStartIsNegative)
	}

	if s.ActiveToOverrideRelativeToPhaseStart != nil && s.ActiveToOverrideRelativeToPhaseStart.IsNegative() {
		errs = append(errs, ErrSubscriptionItemActiveToOverrideRelativeToPhaseStartIsNegative)
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
