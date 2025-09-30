package subscriptionvalidators

import (
	"context"
	"errors"

	"github.com/samber/mo"
	"github.com/samber/mo/result"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

func (v SubscriptionUniqueConstraintValidator) pipelineAfter(ctx context.Context, view subscription.SubscriptionView) mo.Result[[]subscription.SubscriptionSpec] {
	return result.Pipe4(
		v.collectSubs(ctx, view.Customer.Namespace, view.Customer.ID, view.Subscription.ActiveFrom),
		v.mapSubsToViews(ctx),
		v.includeSubViewUnique(view),
		v.mapViewsToSpecs(),
		v.validateUniqueConstraint(ctx, nil),
	)
}

func (v SubscriptionUniqueConstraintValidator) pipelineBefore(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) mo.Result[[]subscription.SubscriptionSpec] {
	return result.Pipe5(
		v.collectSubs(ctx, namespace, spec.CustomerId, spec.ActiveFrom),
		v.mapSubsToViews(ctx),
		v.mapViewsToSpecs(),
		v.validateUniqueConstraint(ctx, func(err error) error {
			return errors.New("inconsistency error: already scheduled subscriptions are overlapping")
		}),
		v.includeSubSpec(spec),
		v.validateUniqueConstraint(ctx, nil),
	)
}
