package service

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

type buildSyncPlanInput struct {
	Subscription                 subscription.Subscription
	SubscriptionView             *subscription.SubscriptionView
	AsOf                         time.Time
	CustomerDeletedAt            *time.Time
	SubscriptionEndProrationMode billing.SubscriptionEndProrationMode
	Currency                     currencyx.Calculator
	DryRun                       bool
}

// buildSyncPlan builds a sync plan for a subscription. If the subscription is deleted, SubscriptionView should be nil.
func (s *Service) buildSyncPlan(ctx context.Context, input buildSyncPlanInput) (*reconciler.Plan, error) {
	span := tracex.Start[*reconciler.Plan](ctx, s.tracer, "billing.worker.subscription.sync.buildSyncPlan")

	return span.Wrap(func(ctx context.Context) (*reconciler.Plan, error) {
		persistedLoader := persistedstate.NewLoader(s.billingService, s.chargesService)
		persisted, err := persistedLoader.LoadForSubscription(ctx, input.Subscription)
		if err != nil {
			return nil, err
		}

		targetBuilder := targetstate.NewBuilder(s.logger, s.tracer)
		target, err := targetBuilder.Build(ctx, targetstate.BuildInput{
			AsOf:                         input.AsOf,
			CustomerDeletedAt:            input.CustomerDeletedAt,
			SubscriptionEndProrationMode: input.SubscriptionEndProrationMode,
			SubscriptionView:             input.SubscriptionView,
			Persisted:                    persisted,
		})
		if err != nil {
			return nil, err
		}

		persisted, err = s.repairChargeSubscriptionReferences(ctx, persisted, target, input.DryRun)
		if err != nil {
			return nil, err
		}

		return s.reconciler.Plan(ctx, reconciler.PlanInput{
			SubscriptionSettlementMode: input.Subscription.SettlementMode,
			Currency:                   input.Currency,
			Target:                     target,
			Persisted:                  persisted,
		})
	})
}
