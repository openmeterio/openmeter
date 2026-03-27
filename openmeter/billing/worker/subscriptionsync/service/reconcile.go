package service

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

func (s *Service) buildSyncPlan(ctx context.Context, subsView subscription.SubscriptionView, asOf time.Time, customerDeletedAt *time.Time, currency currencyx.Calculator) (*reconciler.Plan, error) {
	span := tracex.Start[*reconciler.Plan](ctx, s.tracer, "billing.worker.subscription.sync.buildSyncPlan")

	return span.Wrap(func(ctx context.Context) (*reconciler.Plan, error) {
		persisted, err := s.persistedStateLoader.LoadForSubscription(ctx, subsView.Subscription)
		if err != nil {
			return nil, err
		}

		targetBuilder := targetstate.NewBuilder(s.logger, s.tracer)
		target, err := targetBuilder.Build(ctx, targetstate.BuildInput{
			AsOf:              asOf,
			CustomerDeletedAt: customerDeletedAt,
			SubscriptionView:  subsView,
			Persisted:         persisted,
		})
		if err != nil {
			return nil, err
		}

		return s.reconciler.Plan(ctx, reconciler.PlanInput{
			Subscription: subsView.Subscription,
			Currency:     currency,
			Target:       target,
			Persisted:    persisted,
		})
	})
}
