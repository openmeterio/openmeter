package subscriptionvalidators

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/samber/mo/result"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// This file contains components from which the validation pipelines are built

func (v SubscriptionUniqueConstraintValidator) validateUniqueConstraint(ctx context.Context, errTmplFn func(error) error) func(mo.Result[[]subscription.SubscriptionSpec]) mo.Result[[]subscription.SubscriptionSpec] {
	if errTmplFn == nil {
		errTmplFn = func(err error) error {
			return err
		}
	}

	return func(specs mo.Result[[]subscription.SubscriptionSpec]) mo.Result[[]subscription.SubscriptionSpec] {
		return specs.FlatMap(func(specs []subscription.SubscriptionSpec) mo.Result[[]subscription.SubscriptionSpec] {
			switch multiSubscriptionEnabled, err := v.Config.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF); {
			case err != nil:
				return mo.Err[[]subscription.SubscriptionSpec](fmt.Errorf("failed to check if multi-subscription is enabled: %w", err))
			case multiSubscriptionEnabled:
				return mo.TupleToResult(specs, subscription.ValidateUniqueConstraintByFeatures(specs)).MapErr(func(err error) ([]subscription.SubscriptionSpec, error) {
					return nil, errTmplFn(err)
				})
			default:
				return mo.TupleToResult(specs, subscription.ValidateUniqueConstraintBySubscriptions(specs)).MapErr(func(err error) ([]subscription.SubscriptionSpec, error) {
					return nil, errTmplFn(err)
				})
			}
		})
	}
}

func (v SubscriptionUniqueConstraintValidator) collectSubs(ctx context.Context, namespace string, customerID string, starting time.Time) mo.Result[[]subscription.Subscription] {
	return mo.TupleToResult(v.collectCustomerSubscriptionsStarting(ctx, namespace, customerID, starting))
}

func (v SubscriptionUniqueConstraintValidator) mapSubsToViews(ctx context.Context) func(mo.Result[[]subscription.Subscription]) mo.Result[[]subscription.SubscriptionView] {
	return result.FlatMap(
		func(subs []subscription.Subscription) mo.Result[[]subscription.SubscriptionView] {
			return mo.TupleToResult(v.Config.QueryService.ExpandViews(ctx, subs))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) mapViewsToSpecs() func(mo.Result[[]subscription.SubscriptionView]) mo.Result[[]subscription.SubscriptionSpec] {
	return result.FlatMap(
		func(views []subscription.SubscriptionView) mo.Result[[]subscription.SubscriptionSpec] {
			return mo.Ok(slicesx.Map(views, func(v subscription.SubscriptionView) subscription.SubscriptionSpec {
				return v.AsSpec()
			}))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) includeSubSpec(spec subscription.SubscriptionSpec) func(mo.Result[[]subscription.SubscriptionSpec]) mo.Result[[]subscription.SubscriptionSpec] {
	return result.FlatMap(
		func(specs []subscription.SubscriptionSpec) mo.Result[[]subscription.SubscriptionSpec] {
			return mo.Ok(append(specs, spec))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) includeSubViewUnique(view subscription.SubscriptionView) func(mo.Result[[]subscription.SubscriptionView]) mo.Result[[]subscription.SubscriptionView] {
	return result.FlatMap(
		func(views []subscription.SubscriptionView) mo.Result[[]subscription.SubscriptionView] {
			return mo.Ok(lo.UniqBy(append(views, view), func(i subscription.SubscriptionView) string {
				return i.Subscription.ID
			}))
		},
	)
}

func (v SubscriptionUniqueConstraintValidator) filterSubViews(fn func(subscription.SubscriptionView) bool) func(mo.Result[[]subscription.SubscriptionView]) mo.Result[[]subscription.SubscriptionView] {
	return result.FlatMap(
		func(views []subscription.SubscriptionView) mo.Result[[]subscription.SubscriptionView] {
			return mo.Ok(lo.Filter(views, slicesx.AsFilterIteratee(fn)))
		},
	)
}
