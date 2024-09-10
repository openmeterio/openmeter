package subscription

import (
	"context"
)

type LifecycleManager interface {
	CanStartNew(ctx context.Context, customerID string, activeSubs []Subscription, newSub SubscriptionCreateInfo) (bool, error)
}

type lifecycleManager struct{}

var _ LifecycleManager = (*lifecycleManager)(nil)

func (lm *lifecycleManager) CanStartNew(ctx context.Context, customerID string, activeSubs []Subscription, newSub SubscriptionCreateInfo) (bool, error) {
	// TODO: list of forbidden cases

	panic("not implemented")
}
