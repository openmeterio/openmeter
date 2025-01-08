package subscription

import (
	"context"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type Validator struct {
	subscription.NoOpSubscriptionValidator
	billingService billing.Service
}

func NewValidator(billingService billing.Service) (*Validator, error) {
	if billingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	return &Validator{
		billingService: billingService,
	}, nil
}

func (v Validator) ValidateCreate(ctx context.Context, view subscription.SubscriptionView) error {
	return v.validateBillingSetup(ctx, view)
}

func (v Validator) ValidateUpdate(ctx context.Context, view subscription.SubscriptionView) error {
	return v.validateBillingSetup(ctx, view)
}

func (v Validator) validateBillingSetup(ctx context.Context, view subscription.SubscriptionView) error {
	// If a subscription is going to be billed (e.g. there are phases with ratecards having prices)
	// let's make sure that the billing setup is valid for the customer

	if !v.hasBillableItems(view) {
		return nil
	}

	// Check if the customer has a billing setup
	customerProfile, err := v.billingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
		Namespace:  view.Subscription.Namespace,
		CustomerID: view.Subscription.CustomerId,
	})
	if err != nil {
		return err
	}

	appBase := customerProfile.Profile.Apps.Invoicing
	customerApp, ok := appBase.(customerapp.App)
	if !ok {
		// This should not happen, as the app should have been already verified by the billing service, but let's make sure
		return fmt.Errorf("app [type=%s, id=%s] does not implement the customer interface",
			appBase.GetType(),
			appBase.GetID().ID)
	}

	return customerApp.ValidateCustomer(ctx, &customerProfile.Customer, []appentitybase.CapabilityType{
		// For now now we only support Stripe with automatic tax calculation and payment collection.
		appentitybase.CapabilityTypeCalculateTax,
		appentitybase.CapabilityTypeInvoiceCustomers,
		appentitybase.CapabilityTypeCollectPayments,
	})
}

func (v Validator) hasBillableItems(view subscription.SubscriptionView) bool {
	for _, phase := range view.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if item.SubscriptionItem.RateCard.Price != nil {
					return true
				}
			}
		}
	}

	return false
}
