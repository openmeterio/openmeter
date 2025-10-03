package subscription

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionCommandValidator interface {
	// These happen before the fact
	ValidateCreate(context.Context, string, SubscriptionSpec) error
	ValidateContinue(context.Context, SubscriptionView) error
	ValidateUpdate(context.Context, models.NamespacedID, SubscriptionSpec) error
	// These happen after the fact
	ValidateCreated(context.Context, SubscriptionView) error
	ValidateUpdated(context.Context, SubscriptionView) error
	ValidateCanceled(context.Context, SubscriptionView) error
	ValidateContinued(context.Context, SubscriptionView) error
	ValidateDeleted(context.Context, SubscriptionView) error
}

var _ SubscriptionCommandValidator = (*NoOpSubscriptionCommandValidator)(nil)

type NoOpSubscriptionCommandValidator struct{}

func (NoOpSubscriptionCommandValidator) ValidateCreate(context.Context, string, SubscriptionSpec) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateContinue(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateUpdate(context.Context, models.NamespacedID, SubscriptionSpec) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateCreated(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateUpdated(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateCanceled(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateContinued(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandValidator) ValidateDeleted(context.Context, SubscriptionView) error {
	return nil
}
