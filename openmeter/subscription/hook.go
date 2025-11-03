package subscription

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionCommandHook interface {
	// These happen before the fact
	BeforeCreate(context.Context, string, SubscriptionSpec) error
	BeforeContinue(context.Context, SubscriptionView) error
	BeforeUpdate(context.Context, models.NamespacedID, SubscriptionSpec) error
	BeforeDelete(context.Context, SubscriptionView) error

	// These happen after the fact
	AfterCreate(context.Context, SubscriptionView) error
	AfterUpdate(context.Context, SubscriptionView) error
	AfterCancel(context.Context, SubscriptionView) error
	AfterContinue(context.Context, SubscriptionView) error
}

var _ SubscriptionCommandHook = (*NoOpSubscriptionCommandHook)(nil)

type NoOpSubscriptionCommandHook struct{}

func (NoOpSubscriptionCommandHook) BeforeCreate(context.Context, string, SubscriptionSpec) error {
	return nil
}

func (NoOpSubscriptionCommandHook) BeforeContinue(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandHook) BeforeUpdate(context.Context, models.NamespacedID, SubscriptionSpec) error {
	return nil
}

func (NoOpSubscriptionCommandHook) AfterCreate(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandHook) AfterUpdate(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandHook) AfterCancel(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandHook) AfterContinue(context.Context, SubscriptionView) error {
	return nil
}

func (NoOpSubscriptionCommandHook) BeforeDelete(context.Context, SubscriptionView) error {
	return nil
}
