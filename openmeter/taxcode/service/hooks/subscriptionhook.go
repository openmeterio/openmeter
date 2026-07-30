package hooks

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	SubscriptionHook     = models.ServiceHook[taxcode.TaxCode]
	NoopSubscriptionHook = models.NoopServiceHook[taxcode.TaxCode]
)

type SubscriptionHookConfig struct {
	SubscriptionService subscription.Service
}

func (e SubscriptionHookConfig) Validate() error {
	if e.SubscriptionService == nil {
		return fmt.Errorf("subscription service is required")
	}

	return nil
}

var _ models.ServiceHook[taxcode.TaxCode] = (*subscriptionHook)(nil)

type subscriptionHook struct {
	NoopSubscriptionHook

	subscriptionService subscription.Service
}

func NewSubscriptionHook(config SubscriptionHookConfig) (SubscriptionHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subscription hook config: %w", err)
	}

	return &subscriptionHook{
		subscriptionService: config.SubscriptionService,
	}, nil
}

func (e *subscriptionHook) PreDelete(ctx context.Context, tc *taxcode.TaxCode) error {
	affected, err := e.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{tc.Namespace},
		Status: []subscription.SubscriptionStatus{
			subscription.SubscriptionStatusActive,
			subscription.SubscriptionStatusCanceled,
			subscription.SubscriptionStatusInactive,
			subscription.SubscriptionStatusScheduled,
		},
		TaxCodes: &filter.FilterString{
			In: &[]string{
				tc.ID,
			},
		},
		Page: pagination.Page{
			PageSize:   5,
			PageNumber: 1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	if len(affected.Items) == 0 {
		return nil
	}

	var errs []error

	// The subscription List result carries no rate cards, so expand each matched
	// subscription to a view to reach its rate cards. GetView is used per subscription
	// instead of ExpandViews because ExpandViews only supports a single customer id,
	// while the matched subscriptions may belong to different customers.
	for _, sub := range affected.Items {
		view, err := e.subscriptionService.GetView(ctx, sub.NamespacedID)
		if err != nil {
			return fmt.Errorf("failed to get subscription view: %w", err)
		}

		for _, phase := range view.Phases {
			for _, items := range phase.ItemsByKey {
				for _, item := range items {
					taxCodeID := item.SubscriptionItem.RateCard.AsMeta().TaxCodeReference()
					if taxCodeID == nil || *taxCodeID != tc.ID {
						continue
					}

					errs = append(errs, taxcode.NewTaxCodeReferencedByRateCardError(tc.ID, item.SubscriptionItem.RateCard.Key()))
				}
			}
		}
	}

	if len(errs) == 0 {
		return fmt.Errorf("subscription %s matched tax code filter but no rate card references tax code %s", affected.Items[0].ID, tc.ID)
	}

	return errors.Join(errs...)
}
