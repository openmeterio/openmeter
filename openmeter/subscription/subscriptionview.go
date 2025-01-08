package subscription

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/convert"
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

type SubscriptionPhaseView struct {
	SubscriptionPhase SubscriptionPhase                 `json:"subscriptionPhase"`
	Spec              SubscriptionPhaseSpec             `json:"spec"`
	ItemsByKey        map[string][]SubscriptionItemView `json:"itemsByKey"`
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
					return fmt.Errorf("item %s in phase %s starting after %s is invalid: %w", item.Spec.ItemKey, item.Spec.ActiveFromOverrideRelativeToPhaseStart.ISOStringPtrOrNil(), s.Spec.PhaseKey, err)
				}
			}
		}
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
	spec, err := NewSpecFromEntities(sub, phases, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create spec: %w", err)
	}

	if spec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	// Spec already has to validate that sub, phases and items are linked together so we don't need to do that again here
	// Lets validate that all ents are linked correctly
	unvisitedEnts := map[string]struct{}{}
	for _, ent := range ents {
		// While here, lets also validate that there are no duplicates
		if _, ok := unvisitedEnts[ent.Entitlement.ID]; ok {
			return nil, fmt.Errorf("entitlement %s is duplicated", ent.Entitlement.ID)
		}

		unvisitedEnts[ent.Entitlement.ID] = struct{}{}
	}

	// Needed for item - spec matching, see below
	visitedItemIDs := map[string]struct{}{}

	sv := SubscriptionView{
		Subscription: sub,
		Customer:     cust,
		Spec:         *spec,
	}

	phaseViews := make([]SubscriptionPhaseView, 0, len(spec.Phases))
	for _, phaseSpec := range spec.GetSortedPhases() {
		if phaseSpec == nil {
			return nil, fmt.Errorf("phase spec is nil")
		}

		phase, ok := lo.Find(phases, func(i SubscriptionPhase) bool {
			return i.Key == phaseSpec.PhaseKey
		})
		if !ok {
			return nil, fmt.Errorf("phase %s not found", phaseSpec.PhaseKey)
		}
		phaseView := SubscriptionPhaseView{
			Spec:              *phaseSpec,
			SubscriptionPhase: phase,
		}

		phaseCadenceBySpec, err := spec.GetPhaseCadence(phaseSpec.PhaseKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get phase cadence for phase %s: %w", phaseSpec.PhaseKey, err)
		}

		itemViewsByKey := make(map[string][]SubscriptionItemView)
		for key, itemsByKey := range phaseSpec.ItemsByKey {
			itemViews := make([]SubscriptionItemView, 0, len(itemsByKey))
			for _, itemSpec := range itemsByKey {
				specEntityInput, err := itemSpec.ToCreateSubscriptionItemEntityInput(phase.NamespacedID, phaseCadenceBySpec, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to convert item spec %+v to entity input: %w", itemSpec, err)
				}
				// To find the exact matching item requires for ItemSpecs of a given key to be unique.
				// This is not enforced or required otherwise.
				// As a result, the best we can do is find an item that matches the spec fully, for all specs.
				// We also have to take care not to reuse the same item for multiple specs.
				matchingItem, ok := lo.Find(items, func(i SubscriptionItem) bool {
					itemEntityInput := i.AsEntityInput()

					// Let's ignore the linking fields as they cannot be calculated from the spec
					// FIXME: We can no longer compare based on entity inputs properly, figure out a new method
					itemEntityInput.EntitlementID = nil

					if specEntityInput.Equal(itemEntityInput) {
						// If it's already been used, even if it matches, we cannot reuse it
						if _, ok := visitedItemIDs[i.ID]; ok {
							return false
						}

						visitedItemIDs[i.ID] = struct{}{}

						return true
					}

					return false
				})
				if !ok {
					return nil, fmt.Errorf("item %s in phase %s not found for spec %+v", itemSpec.ItemKey, phaseSpec.PhaseKey, itemSpec)
				}

				var subEnt *SubscriptionEntitlement
				if ent, ok := lo.Find(ents, func(i SubscriptionEntitlement) bool {
					return reflect.DeepEqual(&i.Entitlement.ID, matchingItem.EntitlementID)
				}); ok {
					subEnt = &ent
					delete(unvisitedEnts, ent.Entitlement.ID)
				}

				itemView := SubscriptionItemView{
					SubscriptionItem: matchingItem,
					Spec:             *itemSpec,
					Entitlement:      subEnt,
				}

				itemViews = append(itemViews, itemView)
			}
			itemViewsByKey[key] = itemViews
		}

		phaseView.ItemsByKey = itemViewsByKey
		phaseViews = append(phaseViews, phaseView)
	}

	if len(unvisitedEnts) > 0 {
		return nil, fmt.Errorf("unvisited entitlements: %v", unvisitedEnts)
	}

	// Lets sort phases by start time
	slices.SortStableFunc(phaseViews, func(i, j SubscriptionPhaseView) int {
		if i.ActiveFrom(sub.CadencedModel).Before(j.ActiveFrom(sub.CadencedModel)) {
			return -1
		} else if i.ActiveFrom(sub.CadencedModel).After(j.ActiveFrom(sub.CadencedModel)) {
			return 1
		} else {
			return 0
		}
	})

	sv.Phases = phaseViews

	if err := sv.Validate(true); err != nil {
		return nil, fmt.Errorf("subscription view is invalid: %w", err)
	}

	return &sv, nil
}
