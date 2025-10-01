package subscriptionvalidators

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SubscriptionUniqueConstraintValidatorConfig struct {
	FeatureFlags    ffx.Service
	QueryService    subscription.QueryService
	CustomerService customer.Service
}

func (c SubscriptionUniqueConstraintValidatorConfig) Validate() error {
	if c.FeatureFlags == nil {
		return fmt.Errorf("feature flags is required")
	}

	if c.QueryService == nil {
		return fmt.Errorf("query service is required")
	}

	if c.CustomerService == nil {
		return fmt.Errorf("customer service is required")
	}

	return nil
}

type SubscriptionUniqueConstraintValidator struct {
	subscription.NoOpSubscriptionCommandValidator
	Config SubscriptionUniqueConstraintValidatorConfig
}

func NewSubscriptionUniqueConstraintValidator(config SubscriptionUniqueConstraintValidatorConfig) (subscription.SubscriptionCommandValidator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subscription unique constraint validator config: %w", err)
	}

	return &SubscriptionUniqueConstraintValidator{
		Config: config,
	}, nil
}

func (v SubscriptionUniqueConstraintValidator) ValidateCreate(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) error {
	subs, err := v.collectCustomerSubscriptionsStarting(ctx, namespace, spec.CustomerId, spec.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.Config.QueryService.ExpandViews(ctx, subs)
	if err != nil {
		return err
	}

	specs := slicesx.Map(views, func(v subscription.SubscriptionView) subscription.SubscriptionSpec {
		return v.AsSpec()
	})

	specs, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return errors.New("inconsistency error: already scheduled subscriptions are overlapping")
	}

	specs = append(specs, spec)

	_, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}

func (v SubscriptionUniqueConstraintValidator) ValidateContinue(ctx context.Context, view subscription.SubscriptionView) error {
	// We're only validatint that the subscription can be continued indefinitely
	spec := view.AsSpec()
	spec.ActiveTo = nil

	subs, err := v.collectCustomerSubscriptionsStarting(ctx, view.Customer.Namespace, view.Customer.ID, view.Subscription.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.Config.QueryService.ExpandViews(ctx, subs)
	if err != nil {
		return err
	}

	views = lo.Filter(views, func(v subscription.SubscriptionView, _ int) bool {
		return v.Subscription.ID != view.Subscription.ID
	})

	specs := slicesx.Map(views, func(v subscription.SubscriptionView) subscription.SubscriptionSpec {
		return v.AsSpec()
	})

	specs, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return errors.New("inconsistency error: already scheduled subscriptions are overlapping")
	}

	specs = append(specs, spec)

	_, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}

func (v SubscriptionUniqueConstraintValidator) ValidateCreated(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) ValidateUpdated(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) ValidateCanceled(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) ValidateContinued(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) ValidateDeleted(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) pipelineAfter(ctx context.Context, view subscription.SubscriptionView) error {
	subs, err := v.collectCustomerSubscriptionsStarting(ctx, view.Customer.Namespace, view.Customer.ID, view.Subscription.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.mapSubsToViews(ctx)(subs)
	if err != nil {
		return err
	}

	views, err = v.includeSubViewUnique(view)(views)
	if err != nil {
		return err
	}

	specs, err := v.mapViewsToSpecs()(views)
	if err != nil {
		return err
	}

	_, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}

func (v SubscriptionUniqueConstraintValidator) collectCustomerSubscriptionsStarting(ctx context.Context, namespace string, customerID string, starting time.Time) ([]subscription.Subscription, error) {
	return pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[subscription.Subscription], error) {
		return v.Config.QueryService.List(ctx, subscription.ListSubscriptionsInput{
			CustomerIDs:    []string{customerID},
			Namespaces:     []string{namespace},
			ActiveInPeriod: &timeutil.StartBoundedPeriod{From: starting},
		})
	}), 1000)
}
