package subscription

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type SubscriptionView struct {
	Subscription Subscription            `json:"subscription"`
	Customer     customerentity.Customer `json:"customer"`
	Spec         SubscriptionSpec        `json:"spec"`
	Phases       []SubscriptionPhaseView `json:"phases"`
}

func (s SubscriptionView) AsSpec() SubscriptionSpec {
	return s.Spec
}

func (s *SubscriptionView) Validate(includePhases bool) error {
	spec := s.Spec
	if spec.ActiveFrom != s.Subscription.ActiveFrom {
		return fmt.Errorf("subscription active from %v does not match spec active from %v", s.Subscription.ActiveFrom, spec.ActiveFrom)
	}
	if spec.ActiveTo != s.Subscription.ActiveTo {
		return fmt.Errorf("subscription active to %v does not match spec active to %v", s.Subscription.ActiveTo, spec.ActiveTo)
	}
	if spec.CustomerId != s.Subscription.CustomerId {
		return fmt.Errorf("subscription customer id %s does not match spec customer id %s", s.Subscription.CustomerId, spec.CustomerId)
	}
	if spec.Currency != s.Subscription.Currency {
		return fmt.Errorf("subscription currency %s does not match spec currency %s", s.Subscription.Currency, spec.Currency)
	}

	if !spec.Plan.NilEqual(s.Subscription.PlanRef) {
		return fmt.Errorf("subscription plan %v does not match spec plan %v", s.Subscription.PlanRef, spec.Plan)
	}

	if includePhases {
		for _, phase := range s.Phases {
			if err := phase.Validate(true); err != nil {
				return fmt.Errorf("phase %s is invalid: %w", phase.Spec.PhaseKey, err)
			}
		}
	}
	return nil
}

func (v SubscriptionView) WithoutItemHistory() SubscriptionView {
	v2 := v
	for i, phase := range v.Phases {
		phase2 := phase
		phase2.ItemsByKey = make(map[string][]SubscriptionItemView)
		for key, items := range phase.ItemsByKey {
			if len(items) > 0 {
				// only keep the last item
				phase2.ItemsByKey[key] = items[len(items)-1:]
			}
		}
		v2.Phases[i] = phase2
	}
	return v2
}

type SubscriptionPhaseView struct {
	SubscriptionPhase SubscriptionPhase                 `json:"subscriptionPhase"`
	Spec              SubscriptionPhaseSpec             `json:"spec"`
	ItemsByKey        map[string][]SubscriptionItemView `json:"itemsByKey"`
	Discounts         []DiscountView                    `json:"discounts"`
}

func (s *SubscriptionPhaseView) ActiveFrom(subscriptionCadence models.CadencedModel) time.Time {
	t, _ := s.Spec.StartAfter.AddTo(subscriptionCadence.ActiveFrom)
	return t.UTC()
}

func (s *SubscriptionPhaseView) AsSpec() SubscriptionPhaseSpec {
	return s.Spec
}

func (s *SubscriptionPhaseView) Validate(includeItems bool) error {
	if includeItems {
		for _, items := range s.ItemsByKey {
			for _, item := range items {
				if err := item.Validate(); err != nil {
					return fmt.Errorf("item %s in phase %s starting after %s is invalid: %w", item.Spec.ItemKey, item.Spec.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride.ISOStringPtrOrNil(), s.Spec.PhaseKey, err)
				}
			}
		}

		for _, discount := range s.Discounts {
			if err := discount.Validate(); err != nil {
				return fmt.Errorf("discount %s in phase %s is invalid: %w", discount.Discount.ID, s.Spec.PhaseKey, err)
			}
		}
	}

	return nil
}

type DiscountView struct {
	Discount Discount `json:"discount"`
	Spec     DiscountSpec
}

func (d *DiscountView) Validate() error {
	var errs []error

	if err := d.Discount.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discount %s is invalid: %w", d.Discount.ID, err))
	}

	if err := d.Spec.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discount %s spec is invalid: %w", d.Discount.ID, err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type SubscriptionItemView struct {
	SubscriptionItem SubscriptionItem     `json:"subscriptionItem"`
	Spec             SubscriptionItemSpec `json:"spec"`

	Entitlement *SubscriptionEntitlement `json:"entitlement,omitempty"`
}

func (s *SubscriptionItemView) AsSpec() SubscriptionItemSpec {
	return s.Spec
}

func (s *SubscriptionItemView) Validate() error {
	// Let's validate that the RateCard contents match in Spec and SubscriptionItem
	if !s.Spec.RateCard.Equal(s.SubscriptionItem.RateCard) {
		return fmt.Errorf("item %s rate card %+v does not match spec rate card %+v", s.Spec.ItemKey, s.SubscriptionItem.RateCard, s.Spec.RateCard)
	}

	// Let's validate whether it should have an entitlement
	if (s.Entitlement == nil) != (s.SubscriptionItem.RateCard.EntitlementTemplate == nil) {
		return fmt.Errorf("item %s should have an entitlement: %v", s.Spec.ItemKey, s.SubscriptionItem.RateCard.EntitlementTemplate)
	}

	// Let's validate the Entitlement looks as it should
	if s.Entitlement != nil && s.SubscriptionItem.RateCard.EntitlementTemplate != nil {
		// First, lets validate the nested model
		if err := s.Entitlement.Validate(); err != nil {
			return fmt.Errorf("entitlement for item %s is invalid: %w", s.Spec.ItemKey, err)
		}

		// Second, let's validate the linking
		if !reflect.DeepEqual(&s.Entitlement.Entitlement.ID, s.SubscriptionItem.EntitlementID) {
			return fmt.Errorf("entitlement %s does not match item %s entitlement id", s.Entitlement.Entitlement.ID, s.Spec.ItemKey)
		}

		// Third, let's validate it looks according to the Template
		tpl := s.SubscriptionItem.RateCard.EntitlementTemplate
		ent := s.Entitlement.Entitlement

		switch tpl.Type() {
		case entitlement.EntitlementTypeBoolean:
			if ent.EntitlementType != entitlement.EntitlementTypeBoolean {
				return fmt.Errorf("entitlement %s is not boolean", s.Entitlement.Entitlement.ID)
			}
		case entitlement.EntitlementTypeStatic:
			if ent.EntitlementType != entitlement.EntitlementTypeStatic {
				return fmt.Errorf("entitlement %s is not static", s.Entitlement.Entitlement.ID)
			}

			e, err := tpl.AsStatic()
			if err != nil {
				return fmt.Errorf("entitlement template for Item %s is not static: %w", s.SubscriptionItem.Key, err)
			}

			cfgBytes1, err := e.Config.MarshalJSON()
			if err != nil {
				return fmt.Errorf("failed to marshal entitlement template %s config: %w", s.Entitlement.Entitlement.ID, err)
			}

			if string(cfgBytes1) != string(ent.Config) {
				return fmt.Errorf("entitlement %s config does not match template config", s.Entitlement.Entitlement.ID)
			}

		case entitlement.EntitlementTypeMetered:
			mEnt, err := meteredentitlement.ParseFromGenericEntitlement(&ent)
			if err != nil {
				return fmt.Errorf("entitlement %s is not metered: %w", s.Entitlement.Entitlement.ID, err)
			}

			e, err := tpl.AsMetered()
			if err != nil {
				return fmt.Errorf("entitlement template for Item %s is not metered: %w", s.SubscriptionItem.Key, err)
			}

			if e.IsSoftLimit != mEnt.IsSoftLimit {
				return fmt.Errorf("entitlement %s isSoftLimit does not match template isSoftLimit", s.Entitlement.Entitlement.ID)
			}

			if !reflect.DeepEqual(e.IssueAfterReset, convert.SafeDeRef(mEnt.IssueAfterReset, func(m meteredentitlement.IssueAfterReset) *float64 {
				return &m.Amount
			})) {
				return fmt.Errorf("entitlement %s issueAfterReset does not match template issueAfterReset", s.Entitlement.Entitlement.ID)
			}

			if !reflect.DeepEqual(e.IssueAfterResetPriority, convert.SafeDeRef(mEnt.IssueAfterReset, func(m meteredentitlement.IssueAfterReset) *uint8 {
				return m.Priority
			})) {
				return fmt.Errorf("entitlement %s issueAfterResetPriority does not match template issueAfterResetPriority", s.Entitlement.Entitlement.ID)
			}

			// FIXME: instead of this defaulting behavior we should align the types so that MeteredEntitlementTemplate has the same required fields as MeteredEntitlement
			if !reflect.DeepEqual(lo.CoalesceOrEmpty(e.PreserveOverageAtReset, lo.ToPtr(false)), &mEnt.PreserveOverageAtReset) {
				return fmt.Errorf("entitlement %s preserveOverageAtReset does not match template preserveOverageAtReset", s.Entitlement.Entitlement.ID)
			}

			upRec, err := recurrence.FromISODuration(&e.UsagePeriod, mEnt.UsagePeriod.Anchor)
			if err != nil {
				return fmt.Errorf("failed to convert Item %s EntitlementTemplate UsagePeriod ISO duration to Recurrence: %w", s.SubscriptionItem.Key, err)
			}

			up := entitlement.UsagePeriod(upRec)

			if !up.Equal(mEnt.UsagePeriod) {
				return fmt.Errorf("entitlement %s usagePeriod does not match template usagePeriod", s.Entitlement.Entitlement.ID)
			}

		default:
			return fmt.Errorf("entitlement type %s is not supported", s.SubscriptionItem.RateCard.EntitlementTemplate.Type())
		}
	}

	return nil
}

func NewSubscriptionView(
	sub Subscription,
	cust customerentity.Customer,
	phases []SubscriptionPhase,
	items []SubscriptionItem,
	ents []SubscriptionEntitlement,
) (*SubscriptionView, error) {
	spec := SubscriptionSpec{
		CreateSubscriptionPlanInput: CreateSubscriptionPlanInput{Plan: sub.PlanRef},
		CreateSubscriptionCustomerInput: CreateSubscriptionCustomerInput{
			CustomerId:     sub.CustomerId,
			Currency:       sub.Currency,
			ActiveFrom:     sub.ActiveFrom,
			ActiveTo:       sub.ActiveTo,
			AnnotatedModel: sub.AnnotatedModel,
			Name:           sub.Name,
			Description:    sub.Description,
		},
		Phases: make(map[string]*SubscriptionPhaseSpec),
	}

	view := &SubscriptionView{
		Subscription: sub,
		Customer:     cust,
	}

	// Let's validate that all items are used
	unvisitedItems := make(map[string]struct{})
	for _, item := range items {
		// And also that there are no duplicates
		if _, ok := unvisitedItems[item.ID]; ok {
			return nil, fmt.Errorf("item %s is duplicated", item.ID)
		}

		unvisitedItems[item.ID] = struct{}{}
	}

	// Lets validate that all ents are used
	unvisitedEnts := map[string]struct{}{}
	for _, ent := range ents {
		// While here, lets also validate that there are no duplicates
		if _, ok := unvisitedEnts[ent.Entitlement.ID]; ok {
			return nil, fmt.Errorf("entitlement %s is duplicated", ent.Entitlement.ID)
		}

		unvisitedEnts[ent.Entitlement.ID] = struct{}{}
	}

	// Let's sort the phases
	sortedPhases := make([]SubscriptionPhase, len(phases))
	copy(sortedPhases, phases)
	slices.SortStableFunc(sortedPhases, func(i, j SubscriptionPhase) int {
		return i.ActiveFrom.Compare(j.ActiveFrom)
	})

	itemsByPhase := lo.GroupBy(items, func(item SubscriptionItem) string {
		return item.PhaseId
	})

	// Let's start with all the phases
	for _, phase := range sortedPhases {
		// Let's guard against duplicates
		if _, ok := spec.Phases[phase.Key]; ok {
			return nil, fmt.Errorf("phase %s is duplicated", phase.Key)
		}

		phaseStartAfter := datex.Between(sub.ActiveFrom, phase.ActiveFrom)

		phaseSpec := SubscriptionPhaseSpec{
			CreateSubscriptionPhasePlanInput: CreateSubscriptionPhasePlanInput{
				PhaseKey:    phase.Key,
				StartAfter:  phaseStartAfter,
				Name:        phase.Name,
				Description: phase.Description,
			},
			CreateSubscriptionPhaseCustomerInput: CreateSubscriptionPhaseCustomerInput{
				AnnotatedModel: phase.AnnotatedModel,
			},
			ItemsByKey: make(map[string][]*SubscriptionItemSpec),
		}

		phaseView := SubscriptionPhaseView{
			SubscriptionPhase: phase,
			ItemsByKey:        make(map[string][]SubscriptionItemView),
		}

		phaseItems, ok := itemsByPhase[phase.ID]
		if !ok {
			return nil, fmt.Errorf("items for phase %s not found", phase.Key)
		}

		// Let's group the items by key
		phaseItemsByKey := lo.GroupBy(phaseItems, func(item SubscriptionItem) string {
			return item.Key
		})

		// Let's sort the items by start time
		for key := range phaseItemsByKey {
			// Any arbitrary time works as long as its consistent for the comparisons
			slices.SortStableFunc(phaseItemsByKey[key], func(i, j SubscriptionItem) int {
				iT, jT := phase.ActiveFrom, phase.ActiveFrom
				if i.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride != nil {
					iT, _ = i.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride.AddTo(phase.ActiveFrom)
				}
				if j.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride != nil {
					jT, _ = j.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride.AddTo(phase.ActiveFrom)
				}
				return int(iT.Sub(jT))
			})
		}

		for key, items := range phaseItemsByKey {
			for _, item := range items {
				// Sanity check
				if item.PhaseId != phase.ID {
					return nil, fmt.Errorf("item %s of phase %s is not in the correct phase", item.Key, phase.Key)
				}

				// Sanity check 2
				if item.Key != key {
					return nil, fmt.Errorf("item %s of phase %s is not in the correct group", item.Key, phase.Key)
				}

				delete(unvisitedItems, item.ID)

				itemSpec := SubscriptionItemSpec{
					CreateSubscriptionItemInput: CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: CreateSubscriptionItemPlanInput{
							PhaseKey: phase.Key,
							ItemKey:  item.Key,
							RateCard: item.RateCard,
						},
						CreateSubscriptionItemCustomerInput: CreateSubscriptionItemCustomerInput{
							CadenceOverrideRelativeToPhaseStart: CadenceOverrideRelativeToPhaseStart{
								ActiveFromOverride: item.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride,
								ActiveToOverride:   item.CadenceOverrideRelativeToPhaseStart.ActiveToOverride,
							},
						},
					},
				}

				// Let's find the entitlement

				var subEnt *SubscriptionEntitlement
				if ent, ok := lo.Find(ents, func(i SubscriptionEntitlement) bool {
					return reflect.DeepEqual(&i.Entitlement.ID, item.EntitlementID)
				}); ok {
					subEnt = &ent
					delete(unvisitedEnts, ent.Entitlement.ID)
				}

				itemView := SubscriptionItemView{
					SubscriptionItem: item,
					Entitlement:      subEnt,
					Spec:             itemSpec,
				}

				phaseSpec.ItemsByKey[key] = append(phaseSpec.ItemsByKey[item.Key], &itemSpec)
				phaseView.ItemsByKey[key] = append(phaseView.ItemsByKey[key], itemView)
			}
		}

		spec.Phases[phase.Key] = &phaseSpec
		// Let's add spec to view

		phaseView.Spec = phaseSpec

		view.Phases = append(view.Phases, phaseView)
	}

	if len(unvisitedEnts) > 0 {
		return nil, fmt.Errorf("unvisited entitlements: %v", unvisitedEnts)
	}

	if len(unvisitedItems) > 0 {
		return nil, fmt.Errorf("unvisited items: %v", unvisitedItems)
	}

	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec is invalid: %w", err)
	}

	// Let's add spec to view
	view.Spec = spec

	if err := view.Validate(true); err != nil {
		return nil, fmt.Errorf("subscription view is invalid: %w", err)
	}

	return view, nil
}
