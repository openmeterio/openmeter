package subscriptionvalidators

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// This file contains components from which the validation pipelines are built
func (v SubscriptionUniqueConstraintValidator) validateUniqueConstraint(ctx context.Context, specs []subscription.SubscriptionSpec) ([]subscription.SubscriptionSpec, error) {
	multiSubscriptionEnabled, err := v.Config.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)
	if err != nil {
		return nil, fmt.Errorf("failed to check if multi-subscription is enabled: %w", err)
	}

	if multiSubscriptionEnabled {
		return specs, subscription.ValidateUniqueConstraintByFeatures(specs)
	}

	return specs, subscription.ValidateUniqueConstraintBySubscriptions(specs)
}

func (v SubscriptionUniqueConstraintValidator) mapSubsToViews(ctx context.Context, subs []subscription.Subscription) ([]subscription.SubscriptionView, error) {
	return v.Config.QueryService.ExpandViews(ctx, subs)
}

func (v SubscriptionUniqueConstraintValidator) mapViewsToSpecs(views []subscription.SubscriptionView) ([]subscription.SubscriptionSpec, error) {
	return slicesx.Map(views, func(v subscription.SubscriptionView) subscription.SubscriptionSpec {
		return v.AsSpec()
	}), nil
}

func (v SubscriptionUniqueConstraintValidator) includeSubSpec(spec subscription.SubscriptionSpec, subs []subscription.SubscriptionSpec) ([]subscription.SubscriptionSpec, error) {
	return append(subs, spec), nil
}

func (v SubscriptionUniqueConstraintValidator) includeSubViewUnique(view subscription.SubscriptionView, views []subscription.SubscriptionView) ([]subscription.SubscriptionView, error) {
	return lo.UniqBy(append(views, view), func(i subscription.SubscriptionView) string {
		return i.Subscription.ID
	}), nil
}

func (v SubscriptionUniqueConstraintValidator) filterSubViews(fn func(subscription.SubscriptionView) bool, views []subscription.SubscriptionView) ([]subscription.SubscriptionView, error) {
	return lo.Filter(views, slicesx.AsFilterIteratee(fn)), nil
}
