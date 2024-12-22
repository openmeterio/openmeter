package appstripeentityapp

import (
	"context"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ customerapp.App = (*App)(nil)

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	return a.ValidateCustomerByID(ctx, customer.GetID(), capabilities)
}

// ValidateCustomerByID validates if the app can run for the given customer ID
func (a App) ValidateCustomerByID(ctx context.Context, customerID customerentity.CustomerID, capabilities []appentitybase.CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities...); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	// Get Stripe Customer
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      a.GetID(),
		CustomerID: customerID,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	// Stripe Client
	stripeAppData, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Check if the customer exists in Stripe
	stripeCustomer, err := stripeClient.GetCustomer(ctx, stripeCustomerData.StripeCustomerID)
	if err != nil {
		if _, ok := err.(stripeclient.StripeCustomerNotFoundError); ok {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customerID,
				Condition:  fmt.Sprintf("stripe customer %s not found in stripe account %s", stripeCustomerData.StripeCustomerID, stripeAppData.StripeAccountID),
			}
		}

		return err
	}

	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	if slices.Contains(capabilities, appentitybase.CapabilityTypeCalculateTax) || slices.Contains(capabilities, appentitybase.CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, appentitybase.CapabilityTypeCollectPayments) {
		var paymentMethod stripeclient.StripePaymentMethod

		// Check if the customer has a default payment method in OpenMeter
		// If not try to use the Stripe Customer's default payment method
		if stripeCustomerData.StripeDefaultPaymentMethodID != nil {
			// Get the default payment method
			paymentMethod, err = stripeClient.GetPaymentMethod(ctx, *stripeCustomerData.StripeDefaultPaymentMethodID)
			if err != nil {
				if _, ok := err.(stripeclient.StripePaymentMethodNotFoundError); ok {
					return app.CustomerPreConditionError{
						AppID:      a.GetID(),
						AppType:    a.GetType(),
						CustomerID: customerID,
						Condition:  fmt.Sprintf("default payment method %s not found in stripe account %s", *stripeCustomerData.StripeDefaultPaymentMethodID, stripeAppData.StripeAccountID),
					}
				}

				return fmt.Errorf("failed to get default payment method: %w", err)
			}
		} else {
			// Check if the customer has a default payment method
			if stripeCustomer.DefaultPaymentMethod == nil {
				return app.CustomerPreConditionError{
					AppID:      a.GetID(),
					AppType:    a.GetType(),
					CustomerID: customerID,
					Condition:  "stripe customer must have a default payment method",
				}
			}

			paymentMethod = *stripeCustomer.DefaultPaymentMethod
		}

		// Payment method must have a billing address
		// Billing address is required for tax calculation and invoice creation
		if paymentMethod.BillingAddress == nil {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customerID,
				Condition:  "stripe customer default payment method must have a billing address",
			}
		}

		// TODO: should we have currency as an input to validation?
	}

	return nil
}

// GetCustomerData gets the customer data for the app
func (a App) GetCustomerData(ctx context.Context, input appentity.GetAppInstanceCustomerDataInput) (appentity.CustomerData, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("error validating input: %w", err)
	}

	customerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      a.GetID(),
		CustomerID: input.CustomerID,
	})
	if err != nil {
		return customerData, fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	return customerData, nil
}

// UpsertCustomerData upserts the customer data for the app
func (a App) UpsertCustomerData(ctx context.Context, input appentity.UpsertAppInstanceCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("error validating input: %w", err)
	}

	stripeCustomerData, ok := input.Data.(appstripeentity.CustomerData)
	if !ok {
		return fmt.Errorf("error casting stripe customer data")
	}

	// Upsert stripe customer data
	if err := a.StripeAppService.UpsertStripeCustomerData(ctx, appstripeentity.UpsertStripeCustomerDataInput{
		AppID:                        a.GetID(),
		CustomerID:                   input.CustomerID,
		StripeCustomerID:             stripeCustomerData.StripeCustomerID,
		StripeDefaultPaymentMethodID: stripeCustomerData.StripeDefaultPaymentMethodID,
	}); err != nil {
		return fmt.Errorf("failed to upsert stripe customer data: %w", err)
	}

	return nil
}

// DeleteCustomerData deletes the customer data for the app
func (a App) DeleteCustomerData(ctx context.Context, input appentity.DeleteAppInstanceCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("error validating input: %w", err)
	}

	appId := a.GetID()

	// Delete stripe customer data
	if err := a.StripeAppService.DeleteStripeCustomerData(ctx, appstripeentity.DeleteStripeCustomerDataInput{
		AppID:      &appId,
		CustomerID: input.CustomerID,
	}); err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	return nil
}
