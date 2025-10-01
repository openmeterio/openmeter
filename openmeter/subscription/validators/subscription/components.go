package subscriptionvalidators

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// This file contains components from which the validation pipelines are built
func (v SubscriptionUniqueConstraintValidator) validateUniqueConstraint(ctx context.Context, specs []subscription.SubscriptionSpec) ([]subscription.SubscriptionSpec, error) {
	switch multiSubscriptionEnabled, err := v.Config.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF); {
	case err != nil:
		return nil, fmt.Errorf("failed to check if multi-subscription is enabled: %w", err)
	case multiSubscriptionEnabled:
		return specs, subscription.ValidateUniqueConstraintByFeatures(specs)
	default:
		return specs, subscription.ValidateUniqueConstraintBySubscriptions(specs)
	}
}

func (v SubscriptionUniqueConstraintValidator) collectSubs(ctx context.Context, namespace string, customerID string, starting time.Time) ([]subscription.Subscription, error) {
	return v.collectCustomerSubscriptionsStarting(ctx, namespace, customerID, starting)
}

func (v SubscriptionUniqueConstraintValidator) mapSubsToViews(ctx context.Context) func([]subscription.Subscription) ([]subscription.SubscriptionView, error) {
	return func(subs []subscription.Subscription) ([]subscription.SubscriptionView, error) {
		return v.Config.QueryService.ExpandViews(ctx, subs)
	}
}

func (v SubscriptionUniqueConstraintValidator) mapViewsToSpecs() func([]subscription.SubscriptionView) ([]subscription.SubscriptionSpec, error) {
	return func(views []subscription.SubscriptionView) ([]subscription.SubscriptionSpec, error) {
		return slicesx.Map(views, func(v subscription.SubscriptionView) subscription.SubscriptionSpec {
			return v.AsSpec()
		}), nil
	}
}

func (v SubscriptionUniqueConstraintValidator) includeSubSpec(spec subscription.SubscriptionSpec) func([]subscription.SubscriptionSpec) ([]subscription.SubscriptionSpec, error) {
	return func(specs []subscription.SubscriptionSpec) ([]subscription.SubscriptionSpec, error) {
		return append(specs, spec), nil
	}
}

func (v SubscriptionUniqueConstraintValidator) includeSubViewUnique(view subscription.SubscriptionView) func([]subscription.SubscriptionView) ([]subscription.SubscriptionView, error) {
	return func(views []subscription.SubscriptionView) ([]subscription.SubscriptionView, error) {
		return lo.UniqBy(append(views, view), func(i subscription.SubscriptionView) string {
			return i.Subscription.ID
		}), nil
	}
}

func (v SubscriptionUniqueConstraintValidator) filterSubViews(fn func(subscription.SubscriptionView) bool) func([]subscription.SubscriptionView) ([]subscription.SubscriptionView, error) {
	return func(views []subscription.SubscriptionView) ([]subscription.SubscriptionView, error) {
		return lo.Filter(views, slicesx.AsFilterIteratee(fn)), nil
	}
}
