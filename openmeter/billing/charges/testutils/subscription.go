package testutils

import (
	"context"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ charges.SubscriptionService = (*FakeSubscriptionService)(nil)

// FakeSubscriptionService is a charges.SubscriptionService for suites without
// a real subscription stack. It serves views from the in-memory slice,
// honoring the ID $in filter the customer-charge API facade uses.
type FakeSubscriptionService struct {
	Subscriptions []subscription.SubscriptionView
}

func (s *FakeSubscriptionService) ListViews(_ context.Context, input subscription.ListSubscriptionsInput) (pagination.Result[subscription.SubscriptionView], error) {
	items := s.Subscriptions

	if input.ID != nil && input.ID.In != nil {
		items = lo.Filter(items, func(view subscription.SubscriptionView, _ int) bool {
			return slices.Contains(*input.ID.In, view.Subscription.ID)
		})
	}

	return pagination.Result[subscription.SubscriptionView]{
		Page:       input.Page,
		TotalCount: len(items),
		Items:      items,
	}, nil
}
