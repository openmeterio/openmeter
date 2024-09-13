package subscription

import (
	"context"
	"fmt"

	"github.com/samber/lo"
)

type LifecycleManager interface {
	CanStartNew(ctx context.Context, customerID string, activeSubs []Subscription, newSub SubscriptionCreateInput) error
}

type lifecycleManager struct{}

var _ LifecycleManager = (*lifecycleManager)(nil)

func (lm *lifecycleManager) CanStartNew(ctx context.Context, customerID string, prevSubs []Subscription, newSub SubscriptionCreateInput) error {
	// Pre-filter subscriptions for later use
	var planSubs []Subscription
	var activeSubs []Subscription

	for _, sub := range prevSubs {
		if sub.TemplatingPlanRef.Key == newSub.TemplatingPlanRef.Key {
			planSubs = append(planSubs, sub)
		}

		if sub.IsActive() {
			activeSubs = append(activeSubs, sub)
		}
	}

	if newSub.TrialConfig.IsTrial() {
		// If we're trialing we need to validate that this Plan hasn't been trialed before by the customer
		// TODO: Do we allow that different plans can be trialed independently? Currently yes.
		if lo.SomeBy(planSubs, func(sub Subscription) bool {
			return sub.TrialConfig.IsTrial()
		}) {
			return &ForbiddenError{Message: fmt.Sprintf("Customer %s already has already trialed plan %s", customerID, newSub.TemplatingPlanRef.Key)}
		}

		// If we're trialing we need to validate that we're not currently trialing
		if lo.SomeBy(activeSubs, func(sub Subscription) bool {
			return sub.TrialConfig.IsTrial() && sub.IsActive()
		}) {
			return &ForbiddenError{Message: fmt.Sprintf("Customer %s is already trialing a plan", customerID)}
		}
	} else {
		// We need to validate that the customer doesn't currently have an active subscription
		if len(activeSubs) > 0 {
			return &ForbiddenError{Message: fmt.Sprintf("Customer %s already has an active subscription", customerID)}
		}
	}

	return nil
}
