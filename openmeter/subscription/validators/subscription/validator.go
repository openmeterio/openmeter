package subscriptionvalidators

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
	subscription.NoOpSubscriptionCommandHook
	Config SubscriptionUniqueConstraintValidatorConfig
}

func NewSubscriptionUniqueConstraintValidator(config SubscriptionUniqueConstraintValidatorConfig) (subscription.SubscriptionCommandHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subscription unique constraint validator config: %w", err)
	}

	return &SubscriptionUniqueConstraintValidator{
		Config: config,
	}, nil
}

func (v SubscriptionUniqueConstraintValidator) BeforeCreate(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) error {
	subs, err := v.collectCustomerSubscriptionsStarting(ctx, namespace, spec.CustomerId, spec.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.mapSubsToViews(ctx, subs)
	if err != nil {
		return err
	}

	specs, err := v.mapViewsToSpecs(views)
	if err != nil {
		return err
	}

	specs, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return fmt.Errorf("inconsistency error: already scheduled subscriptions are overlapping: %w", err)
	}

	specs, err = v.includeSubSpec(spec, specs)
	if err != nil {
		return err
	}

	_, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}

func (v SubscriptionUniqueConstraintValidator) BeforeUpdate(ctx context.Context, currentId models.NamespacedID, targetSpec subscription.SubscriptionSpec) error {
	// We only do these validations if multi-subscription is enabled
	multiSubscriptionEnabled, err := v.Config.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)
	if err != nil {
		return err
	}

	if !multiSubscriptionEnabled {
		return nil
	}

	subs, err := v.collectCustomerSubscriptionsStarting(ctx, currentId.Namespace, targetSpec.CustomerId, targetSpec.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.mapSubsToViews(ctx, subs)
	if err != nil {
		return err
	}

	views, err = v.filterSubViews(func(v subscription.SubscriptionView) bool {
		// Let's exclude the current subscription as we'll include the new version in the validation instead
		return v.Subscription.ID != currentId.ID
	}, views)
	if err != nil {
		return err
	}

	specs, err := v.mapViewsToSpecs(views)
	if err != nil {
		return err
	}

	specs, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return fmt.Errorf("inconsistency error: already scheduled subscriptions are overlapping: %w", err)
	}

	specs, err = v.includeSubSpec(targetSpec, specs)
	if err != nil {
		return err
	}

	_, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}

func (v SubscriptionUniqueConstraintValidator) BeforeContinue(ctx context.Context, view subscription.SubscriptionView) error {
	// We're only validatint that the subscription can be continued indefinitely
	spec := view.AsSpec()
	spec.ActiveTo = nil

	subs, err := v.collectCustomerSubscriptionsStarting(ctx, view.Customer.Namespace, view.Customer.ID, view.Subscription.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.mapSubsToViews(ctx, subs)
	if err != nil {
		return err
	}

	views, err = v.filterSubViews(func(v subscription.SubscriptionView) bool {
		return v.Subscription.ID != view.Subscription.ID
	}, views)
	if err != nil {
		return err
	}

	specs, err := v.mapViewsToSpecs(views)
	if err != nil {
		return err
	}

	specs, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return fmt.Errorf("inconsistency error: already scheduled subscriptions are overlapping: %w", err)
	}

	specs, err = v.includeSubSpec(spec, specs)
	if err != nil {
		return err
	}

	_, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}

func (v SubscriptionUniqueConstraintValidator) AfterCreate(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) AfterUpdate(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) AfterCancel(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) AfterContinue(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) BeforeDelete(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view)
}

func (v SubscriptionUniqueConstraintValidator) pipelineAfter(ctx context.Context, view subscription.SubscriptionView) error {
	subs, err := v.collectCustomerSubscriptionsStarting(ctx, view.Customer.Namespace, view.Customer.ID, view.Subscription.ActiveFrom)
	if err != nil {
		return err
	}

	views, err := v.mapSubsToViews(ctx, subs)
	if err != nil {
		return err
	}

	views, err = v.includeSubViewUnique(view, views)
	if err != nil {
		return err
	}

	specs, err := v.mapViewsToSpecs(views)
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
