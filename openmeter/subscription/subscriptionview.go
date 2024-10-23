package subscription

import (
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
)

type SubscriptionView interface {
	Sub() Subscription
	Phases() []SubscriptionPhaseView
}

type SubscriptionPhaseView interface {
	Key() string
	ActiveFrom() time.Time
	Items() []SubscriptionItemView
}

type SubscriptionItemView interface {
	BillingCadence() time.Duration
	Key() string

	Price() (price.Price, bool)
	// The feature referenced here might since have been archived (both the exact version of the feature under the same key as present when the item was specced, but also the key itself)
	FeatureKey() (string, bool)
	Entitlement() (entitlement.Entitlement, bool)
}

type subscriptionView struct {
	subscription     Subscription
	subscriptionSpec *SubscriptionSpec
	phases           []subscriptionPhaseView
}

var _ SubscriptionView = (*subscriptionView)(nil)

func (s *subscriptionView) Sub() Subscription {
	return s.subscription
}

func (s *subscriptionView) Phases() []SubscriptionPhaseView {
	// Map phases to interface
	phases := make([]SubscriptionPhaseView, 0, len(s.phases))
	for _, p := range s.phases {
		phases = append(phases, &p)
	}

	return phases
}

type subscriptionPhaseView struct {
	subscription Subscription
	spec         *SubscriptionPhaseSpec
	items        []subscriptionItemView
}

var _ SubscriptionPhaseView = (*subscriptionPhaseView)(nil)

func (s *subscriptionPhaseView) Key() string {
	return s.spec.PhaseKey
}

func (s *subscriptionPhaseView) ActiveFrom() time.Time {
	t, _ := s.spec.StartAfter.AddTo(s.subscription.ActiveFrom)
	return t.UTC()
}

func (s *subscriptionPhaseView) Items() []SubscriptionItemView {
	// Map items to interface
	items := make([]SubscriptionItemView, 0, len(s.items))
	for _, i := range s.items {
		items = append(items, &i)
	}

	return items
}

type subscriptionItemView struct {
	subscription Subscription
	spec         *SubscriptionItemSpec

	price       *price.Price
	entitlement *entitlement.Entitlement
}

var _ SubscriptionItemView = (*subscriptionItemView)(nil)

func (s *subscriptionItemView) BillingCadence() time.Duration {
	panic("implement me")
}

func (s *subscriptionItemView) Key() string {
	return s.spec.ItemKey
}

func (s *subscriptionItemView) Price() (price.Price, bool) {
	if s.price == nil {
		return price.Price{}, false
	}
	return *s.price, true
}

func (s *subscriptionItemView) FeatureKey() (string, bool) {
	if s.spec.FeatureKey == nil {
		return "", false
	}
	return *s.spec.FeatureKey, true
}

func (s *subscriptionItemView) Entitlement() (entitlement.Entitlement, bool) {
	if s.entitlement == nil {
		return entitlement.Entitlement{}, false
	}
	return *s.entitlement, true
}

func NewSubscriptionView(
	sub Subscription,
	spec *SubscriptionSpec,
	ents []SubscriptionEntitlement,
	prices []price.Price,
) (SubscriptionView, error) {
	sv := subscriptionView{
		subscription:     sub,
		subscriptionSpec: spec,
	}

	phases := make([]subscriptionPhaseView, 0, len(spec.Phases))
	for _, phaseSpec := range spec.Phases {
		phase := subscriptionPhaseView{
			subscription: sub,
			spec:         phaseSpec,
		}

		items := make([]subscriptionItemView, 0, len(phaseSpec.Items))
		for _, itemSpec := range phaseSpec.Items {
			item := subscriptionItemView{
				subscription: sub,
				spec:         itemSpec,
			}

			// Find matching entitlement
			for _, ent := range ents {
				if item.spec.GetRef(item.subscription.ID).Equals(ent.ItemRef) {
					item.entitlement = &ent.Entitlement
				}
			}

			// Validate whether it should have entitlement
			if item.spec.HasEntitlement() != (item.entitlement != nil) {
				return nil, fmt.Errorf("item %s should have entitlement: %t, but has: %t", item.spec.ItemKey, item.spec.HasEntitlement(), item.entitlement != nil)
			}

			// Find matching price
			for _, p := range prices {
				pRef := SubscriptionItemRef{
					SubscriptionId: p.SubscriptionId,
					PhaseKey:       p.PhaseKey,
					ItemKey:        p.ItemKey,
				}
				if item.spec.GetRef(item.subscription.ID).Equals(pRef) {
					item.price = &p
				}
			}

			// Validate whether it should have prive
			if item.spec.HasPrice() != (item.price != nil) {
				return nil, fmt.Errorf("item %s should have price: %t, but has: %t", item.spec.ItemKey, item.spec.HasPrice(), item.price != nil)
			}

			items = append(items, item)
		}

		phase.items = items
		phases = append(phases, phase)
	}

	// Lets sort phases by start time
	slices.SortStableFunc(phases, func(i, j subscriptionPhaseView) int {
		if i.ActiveFrom().Before(j.ActiveFrom()) {
			return -1
		} else if i.ActiveFrom().After(j.ActiveFrom()) {
			return 1
		} else {
			return 0
		}
	})

	sv.phases = phases
	return &sv, nil
}
