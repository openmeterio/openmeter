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

func (v SubscriptionUniqueConstraintValidator) validateUniqueConstraint(ctx context.Context, errTmplFn func(error) error) func(result[[]subscription.SubscriptionSpec]) result[[]subscription.SubscriptionSpec] {
	if errTmplFn == nil {
		errTmplFn = func(err error) error {
			return err
		}
	}

	return func(specs result[[]subscription.SubscriptionSpec]) result[[]subscription.SubscriptionSpec] {
		return specs.FlatMap(func(specs []subscription.SubscriptionSpec) result[[]subscription.SubscriptionSpec] {
			switch multiSubscriptionEnabled, err := v.Config.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF); {
			case err != nil:
				return errResult[[]subscription.SubscriptionSpec](fmt.Errorf("failed to check if multi-subscription is enabled: %w", err))
			case multiSubscriptionEnabled:
				return resultFromTouple(specs, subscription.ValidateUniqueConstraintByFeatures(specs)).FlatMapErr(func(err error) result[[]subscription.SubscriptionSpec] {
					return errResult[[]subscription.SubscriptionSpec](errTmplFn(err))
				})
			default:
				return resultFromTouple(specs, subscription.ValidateUniqueConstraintBySubscriptions(specs)).FlatMapErr(func(err error) result[[]subscription.SubscriptionSpec] {
					return errResult[[]subscription.SubscriptionSpec](errTmplFn(err))
				})
			}
		})
	}
}

func (v SubscriptionUniqueConstraintValidator) collectSubs(ctx context.Context, namespace string, customerID string, starting time.Time) result[[]subscription.Subscription] {
	return resultFromTouple(v.collectCustomerSubscriptionsStarting(ctx, namespace, customerID, starting))
}

func (v SubscriptionUniqueConstraintValidator) mapSubsToViews(ctx context.Context) func(result[[]subscription.Subscription]) result[[]subscription.SubscriptionView] {
	return flatMap(
		func(subs []subscription.Subscription) result[[]subscription.SubscriptionView] {
			return resultFromTouple(v.Config.QueryService.ExpandViews(ctx, subs))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) mapViewsToSpecs() func(result[[]subscription.SubscriptionView]) result[[]subscription.SubscriptionSpec] {
	return flatMap(
		func(views []subscription.SubscriptionView) result[[]subscription.SubscriptionSpec] {
			return okResult(slicesx.Map(views, func(v subscription.SubscriptionView) subscription.SubscriptionSpec {
				return v.AsSpec()
			}))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) includeSubSpec(spec subscription.SubscriptionSpec) func(result[[]subscription.SubscriptionSpec]) result[[]subscription.SubscriptionSpec] {
	return flatMap(
		func(specs []subscription.SubscriptionSpec) result[[]subscription.SubscriptionSpec] {
			return okResult(append(specs, spec))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) includeSubViewUnique(view subscription.SubscriptionView) func(result[[]subscription.SubscriptionView]) result[[]subscription.SubscriptionView] {
	return flatMap(
		func(views []subscription.SubscriptionView) result[[]subscription.SubscriptionView] {
			return okResult(lo.UniqBy(append(views, view), func(i subscription.SubscriptionView) string {
				return i.Subscription.ID
			}))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) filterSubViews(fn func(subscription.SubscriptionView) bool) func(result[[]subscription.SubscriptionView]) result[[]subscription.SubscriptionView] {
	return flatMap(
		func(views []subscription.SubscriptionView) result[[]subscription.SubscriptionView] {
			return okResult(lo.Filter(views, slicesx.AsFilterIteratee(fn)))
		},
	)
}
