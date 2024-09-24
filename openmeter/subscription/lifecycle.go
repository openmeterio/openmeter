package subscription

import (
	"context"
	"fmt"

	"github.com/samber/lo"
)

// LifecycleManager is responsible for managing the lifecycle rules of subscriptions.
type LifecycleManager interface {
	CanStartNew(ctx context.Context, customerID string, activeSubs []Subscription, newPlan Plan) error
}

type lifecycleManager struct{}

var _ LifecycleManager = (*lifecycleManager)(nil)

func (lm *lifecycleManager) CanStartNew(ctx context.Context, customerID string, prevSubs []Subscription, newPlan Plan) error {
	// Pre-filter subscriptions for later use
	var activeSubs []Subscription

	for _, sub := range prevSubs {
		if sub.IsActive() {
			activeSubs = append(activeSubs, sub)
		}
	}

	// 1. We cannot change to the same plan we're already on
	if lo.SomeBy(activeSubs, func(sub Subscription) bool {
		return sub.TemplatingPlanRef.Key == newPlan.Key
	}) {
		return &ForbiddenError{Message: fmt.Sprintf("Customer %s is already on plan %s", customerID, newPlan.Key)}
	}

	return nil
}
