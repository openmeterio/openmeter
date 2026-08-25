package testutils

import (
	"context"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

var _ charges.SubscriptionService = (*FakeSubscriptionService)(nil)

// FakeSubscriptionService is a charges.SubscriptionService for suites without
// a real subscription stack. It serves subscriptions from the in-memory
// slice, honoring the ID $in filter the customer-charge API facade uses.
type FakeSubscriptionService struct {
	Subscriptions []subscription.Subscription
}

func (s *FakeSubscriptionService) List(_ context.Context, input subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	items := s.Subscriptions

	if input.ID != nil && input.ID.In != nil {
		items = lo.Filter(items, func(sub subscription.Subscription, _ int) bool {
			return slices.Contains(*input.ID.In, sub.ID)
		})
	}

	return subscription.SubscriptionList{
		Page:       input.Page,
		TotalCount: len(items),
		Items:      items,
	}, nil
}
