package subscription

import "context"

type SubscriptionValidator interface {
	ValidateCreate(context.Context, SubscriptionView) error
	ValidateUpdate(context.Context, SubscriptionView) error
	ValidateCancel(context.Context, SubscriptionView) error
	ValidateContinue(context.Context, SubscriptionView) error
	ValidateDelete(context.Context, SubscriptionView) error
}

var _ SubscriptionValidator = (*NoOpSubscriptionValidator)(nil)

type NoOpSubscriptionValidator struct{}

func (NoOpSubscriptionValidator) ValidateCreate(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionValidator) ValidateUpdate(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionValidator) ValidateCancel(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionValidator) ValidateContinue(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionValidator) ValidateDelete(context.Context, SubscriptionView) error {
	return nil
}
