package subscription

import (
	"fmt"
	"slices"
	"time"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
)

type SubscriptionView interface {
	Sub() Subscription
	Customer() customerentity.Customer
	Phases() []SubscriptionPhaseView
	// Returns the Spec for the Subscription.
	// .Validate() guarantees that the spec matches the View.
	AsSpec() *SubscriptionSpec
	Validate(includePhases bool) error
}

type SubscriptionPhaseView interface {
	Key() string
	ActiveFrom() time.Time
	Items() []SubscriptionItemView

	// Returns the Spec for the Phase.
	// .Validate() guarantees that the spec matches the View.
	AsSpec() *SubscriptionPhaseSpec
	Validate(includeItems bool) error
}

type SubscriptionItemView interface {
	BillingCadence() time.Duration
	Key() string

	Price() (price.Price, bool)
	// The feature referenced here might since have been archived (both the exact version of the feature under the same key as present when the item was specced, but also the key itself)
	FeatureKey() (string, bool)
	Entitlement() (SubscriptionEntitlement, bool)

	// Returns the Spec for the Item.
	// .Validate() guarantees that the spec matches the View.
	AsSpec() *SubscriptionItemSpec
	Validate() error
}

type subscriptionView struct {
	subscription     Subscription
	customer         customerentity.Customer
	subscriptionSpec *SubscriptionSpec
	phases           []subscriptionPhaseView
}

var _ SubscriptionView = (*subscriptionView)(nil)

func (s *subscriptionView) Sub() Subscription {
	return s.subscription
}

func (s *subscriptionView) Customer() customerentity.Customer {
	return s.customer
}

func (s *subscriptionView) Phases() []SubscriptionPhaseView {
	// Map phases to interface
	phases := make([]SubscriptionPhaseView, 0, len(s.phases))
	for _, p := range s.phases {
		phases = append(phases, &p)
	}

	return phases
}

func (s *subscriptionView) AsSpec() *SubscriptionSpec {
	return s.subscriptionSpec
}

func (s *subscriptionView) Validate(includePhases bool) error {
	spec := s.AsSpec()
	if spec == nil {
		return fmt.Errorf("subscription has no spec")
	}
	if spec.ActiveFrom != s.subscription.ActiveFrom {
		return fmt.Errorf("subscription active from %v does not match spec active from %v", s.subscription.ActiveFrom, spec.ActiveFrom)
	}
	if spec.ActiveTo != s.subscription.ActiveTo {
		return fmt.Errorf("subscription active to %v does not match spec active to %v", s.subscription.ActiveTo, spec.ActiveTo)
	}
	if spec.CustomerId != s.subscription.CustomerId {
		return fmt.Errorf("subscription customer id %s does not match spec customer id %s", s.subscription.CustomerId, spec.CustomerId)
	}
	if spec.Currency != s.subscription.Currency {
		return fmt.Errorf("subscription currency %s does not match spec currency %s", s.subscription.Currency, spec.Currency)
	}
	if !spec.Plan.Equals(s.subscription.Plan) {
		return fmt.Errorf("subscription plan %v does not match spec plan %v", s.subscription.Plan, spec.Plan)
	}

	if includePhases {
		for _, phase := range s.phases {
			if err := phase.Validate(true); err != nil {
				return fmt.Errorf("phase %s is invalid: %w", phase.spec.PhaseKey, err)
			}
		}
	}
	return nil
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

func (s *subscriptionPhaseView) AsSpec() *SubscriptionPhaseSpec {
	return s.spec
}

func (s *subscriptionPhaseView) Validate(includeItems bool) error {
	spec := s.AsSpec()
	if spec == nil {
		return fmt.Errorf("phase %s has no spec", s.spec.PhaseKey)
	}
	if includeItems {
		for _, item := range s.items {
			if err := item.Validate(); err != nil {
				return fmt.Errorf("item %s in phase %s is invalid: %w", item.spec.ItemKey, s.spec.PhaseKey, err)
			}
		}
	}
	return nil
}

type subscriptionItemView struct {
	subscription Subscription
	spec         *SubscriptionItemSpec

	price       *price.Price
	entitlement *SubscriptionEntitlement
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

func (s *subscriptionItemView) Entitlement() (SubscriptionEntitlement, bool) {
	if s.entitlement == nil {
		return SubscriptionEntitlement{}, false
	}
	return *s.entitlement, true
}

func (s *subscriptionItemView) AsSpec() *SubscriptionItemSpec {
	return s.spec
}

func (s *subscriptionItemView) Validate() error {
	spec := s.AsSpec()
	if spec == nil {
		return fmt.Errorf("item %s in phase %s has no spec", s.spec.ItemKey, s.spec.PhaseKey)
	}

	if spec.HasEntitlement() != (s.entitlement != nil) {
		return fmt.Errorf("item %s should have entitlement: %t, but has: %t", s.spec.ItemKey, s.spec.HasEntitlement(), s.entitlement != nil)
	}
	if s.entitlement != nil {
		if spec.ItemKey != s.entitlement.ItemRef.ItemKey {
			return fmt.Errorf("item %s should match entitlement item key %s", s.spec.ItemKey, s.entitlement.ItemRef.ItemKey)
		}
		if spec.PhaseKey != s.entitlement.ItemRef.PhaseKey {
			return fmt.Errorf("item %s should match entitlement phase key %s", s.spec.ItemKey, s.entitlement.ItemRef.PhaseKey)
		}
		if err := s.entitlement.Validate(); err != nil {
			return fmt.Errorf("entitlement for item %s is invalid: %w", s.spec.ItemKey, err)
		}
	}
	if spec.HasPrice() != (s.price != nil) {
		return fmt.Errorf("item %s should have price: %t, but has: %t", s.spec.ItemKey, s.spec.HasPrice(), s.price != nil)
	}
	if s.price != nil {
		if s.price.SubscriptionId != s.subscription.ID {
			return fmt.Errorf("item %s should match price subscription id %s", s.spec.ItemKey, s.subscription.ID)
		}
		if s.price.PhaseKey != s.spec.PhaseKey {
			return fmt.Errorf("item %s should match price phase key %s", s.spec.ItemKey, s.price.PhaseKey)
		}
		if s.price.ItemKey != s.spec.ItemKey {
			return fmt.Errorf("item %s should match price item key %s", s.spec.ItemKey, s.price.ItemKey)
		}
		if spec.CreatePriceInput.ItemKey != s.price.ItemKey {
			return fmt.Errorf("item %s should match price item key %s", s.spec.ItemKey, s.price.ItemKey)
		}
		if spec.CreatePriceInput.PhaseKey != s.price.PhaseKey {
			return fmt.Errorf("item %s should match price phase key %s", s.spec.ItemKey, s.price.PhaseKey)
		}
		if spec.CreatePriceInput.Key != s.price.Key {
			return fmt.Errorf("item %s should match price key %s", s.spec.ItemKey, s.price.Key)
		}
		if spec.CreatePriceInput.Value != s.price.Value {
			return fmt.Errorf("item %s should match price value %s", s.spec.ItemKey, s.price.Value)
		}
	}

	return nil
}

func NewSubscriptionView(
	sub Subscription,
	cust customerentity.Customer,
	spec *SubscriptionSpec,
	ents []SubscriptionEntitlement,
	prices []price.Price,
) (SubscriptionView, error) {
	sv := subscriptionView{
		subscription:     sub,
		customer:         cust,
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
					item.entitlement = &ent
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
