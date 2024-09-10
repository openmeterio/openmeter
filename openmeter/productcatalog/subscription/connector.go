package subscription

import (
	"context"
	"time"
)

type Connector interface {
	// EndAt ends a subscription effective at the provided time.
	EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)

	// StartNew attempts to start a new subscription for a customer effective at the provided time.
	StartNewAt(ctx context.Context, customerID string, at time.Time, info SubscriptionCreateInfo) (Subscription, error)

	// ChangeContents changes the contents of a subscription effective retroactively.
	//
	// Effective retroactively means that the subscription contents are changed and no new version is created to replace the old.
	// For example, if you change some usage based price included in the subscription, the prior usage will also be billed based on the new price.
	ChangeContents(ctx context.Context, subscriptionID string, overrides SubscriptionOverrides) (Subscription, error)
}

type connector struct {
	customerSubscriptionRepo CustomerSubscriptionRepo
	subscriptionRepo         SubscriptionRepo

	lifecycleManager LifecycleManager
}

var _ Connector = (*connector)(nil)

func (c *connector) EndAt(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error) {
	panic("not implemented")
}

func (c *connector) StartNewAt(ctx context.Context, customerID string, at time.Time, info SubscriptionCreateInfo) (Subscription, error) {
	// Check if a customer has active subscription(s)
	activeSubs, err := c.customerSubscriptionRepo.GetActiveSubscriptionsAt(ctx, customerID, at)
	if err != nil {
		return Subscription{}, err
	}

	// If a customer has active subscriptions we need to know if the change is compatible with lifecycle
	if len(activeSubs) > 0 {
		can, err := c.lifecycleManager.CanStartNew(ctx, customerID, activeSubs, info)
		if err != nil {
			return Subscription{}, err
		}

		if !can {
			return Subscription{}, &ForbiddenError{}
		}
	}

	return c.subscriptionRepo.Create(ctx, MapCreateInfoToRepoInput(info))
}

func (c *connector) ChangeContents(ctx context.Context, subscriptionID string, overrides SubscriptionOverrides) (Subscription, error) {
	panic("not implemented")
}

func MapCreateInfoToRepoInput(info SubscriptionCreateInfo) SubscriptionRepoCreateInput {
	panic("not implemented")
}
