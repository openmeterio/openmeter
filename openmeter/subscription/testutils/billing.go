package subscriptiontestutils

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.CustomerBillingProfileService = (*NoopCustomerOverrideService)(nil)

type NoopCustomerOverrideService struct{}

func (n NoopCustomerOverrideService) SetCustomerBillingProfile(ctx context.Context, input billing.SetCustomerBillingProfileInput) (billing.CustomerBillingProfile, error) {
	return billing.CustomerBillingProfile{}, nil
}

func (n NoopCustomerOverrideService) ClearCustomerBillingProfile(ctx context.Context, input billing.ClearCustomerBillingProfileInput) error {
	return nil
}

func (n NoopCustomerOverrideService) GetCustomerBillingProfile(ctx context.Context, input billing.GetCustomerBillingProfileInput) (billing.CustomerBillingProfile, error) {
	return billing.CustomerBillingProfile{}, nil
}

func (n NoopCustomerOverrideService) GetCustomerApp(ctx context.Context, input billing.GetCustomerAppInput) (app.App, error) {
	return nil, nil
}

func (n NoopCustomerOverrideService) ListCustomerBillingProfiles(ctx context.Context, input billing.ListCustomerBillingProfilesInput) (billing.ListCustomerBillingProfilesResult, error) {
	return billing.ListCustomerBillingProfilesResult{}, nil
}
