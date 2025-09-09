package subscriptiontestutils

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.CustomerOverrideService = (*NoopCustomerOverrideService)(nil)

type NoopCustomerOverrideService struct{}

func (n NoopCustomerOverrideService) UpsertCustomerOverride(ctx context.Context, input billing.UpsertCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	return billing.CustomerOverrideWithDetails{}, nil
}

func (n NoopCustomerOverrideService) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	return nil
}

func (n NoopCustomerOverrideService) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	return billing.CustomerOverrideWithDetails{}, nil
}

func (n NoopCustomerOverrideService) GetCustomerApp(ctx context.Context, input billing.GetCustomerAppInput) (app.App, error) {
	return nil, nil
}

func (n NoopCustomerOverrideService) ListCustomerOverrides(ctx context.Context, input billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesResult, error) {
	return billing.ListCustomerOverridesResult{}, nil
}
