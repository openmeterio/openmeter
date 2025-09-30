package subscriptionvalidators

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/ffx"
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
	return v.pipelineBefore(ctx, namespace, spec).Error()
}

func (v SubscriptionUniqueConstraintValidator) ValidateCreated(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view).Error()
}

func (v SubscriptionUniqueConstraintValidator) ValidateUpdated(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view).Error()
}

func (v SubscriptionUniqueConstraintValidator) ValidateCanceled(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view).Error()
}

func (v SubscriptionUniqueConstraintValidator) ValidateContinued(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view).Error()
}

func (v SubscriptionUniqueConstraintValidator) ValidateDeleted(ctx context.Context, view subscription.SubscriptionView) error {
	return v.pipelineAfter(ctx, view).Error()
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
