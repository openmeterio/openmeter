package subscription

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Validator struct {
	subscription.NoOpSubscriptionCommandHook
	billingService billing.Service
}

func NewValidator(billingService billing.Service) (subscription.SubscriptionCommandHook, error) {
	if billingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	return &Validator{
		billingService: billingService,
	}, nil
}

func (v Validator) AfterCreate(ctx context.Context, view subscription.SubscriptionView) error {
	err := v.validateBillingSetup(ctx, view)
	if err != nil {
		return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))
	}

	return nil
}

func (v Validator) AfterUpdate(ctx context.Context, view subscription.SubscriptionView) error {
	err := v.validateBillingSetup(ctx, view)
	if err != nil {
		return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))
	}

	return nil
}

func (v Validator) validateBillingSetup(ctx context.Context, view subscription.SubscriptionView) error {
	// If a subscription is going to be billed (e.g. there are phases with ratecards having prices)
	// let's make sure that the billing setup is valid for the customer

	if !v.hasBillableItems(view) {
		return nil
	}

	// Check if the customer has a billing setup
	customerProfile, err := v.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: view.Subscription.Namespace,
			ID:        view.Subscription.CustomerId,
		},
		Expand: billing.CustomerOverrideExpand{
			Apps:     true,
			Customer: true,
		},
	})
	if err != nil {
		return err
	}

	appBase := customerProfile.MergedProfile.Apps.Invoicing
	customerApp, err := customerapp.AsCustomerApp(appBase)
	if err != nil {
		// This should not happen, as the app should have been already verified by the billing service, but let's make sure
		return err
	}

	return customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{
		// For now now we only support Stripe with automatic tax calculation and payment collection.
		app.CapabilityTypeCalculateTax,
		app.CapabilityTypeInvoiceCustomers,
		app.CapabilityTypeCollectPayments,
	})
}

func (v Validator) hasBillableItems(view subscription.SubscriptionView) bool {
	for _, phase := range view.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if item.SubscriptionItem.RateCard.AsMeta().Price != nil {
					return true
				}
			}
		}
	}

	return false
}
