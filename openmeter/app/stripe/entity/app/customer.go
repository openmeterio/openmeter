package appstripeentityapp

import (
	"context"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ customerapp.App = (*App)(nil)

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer *customer.Customer, capabilities []app.CapabilityType) error {
	return a.ValidateCustomerByID(ctx, customer.GetID(), capabilities)
}

// ValidateCustomerByID validates if the app can run for the given customer ID
func (a App) ValidateCustomerByID(ctx context.Context, customerID customer.CustomerID, capabilities []app.CapabilityType) error {
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
	stripeAppData, stripeClient, err := a.getStripeClient(ctx, "validateCustomer", "customer_id", customerID.ID)
	if err != nil {
		return fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Check if the customer exists in Stripe
	stripeCustomer, err := stripeClient.GetCustomer(ctx, stripeCustomerData.StripeCustomerID)
	if err != nil {
		if _, ok := err.(stripeclient.StripeCustomerNotFoundError); ok {
			return app.NewAppCustomerPreConditionError(
				a.GetID(),
				a.GetType(),
				&customerID,
				fmt.Sprintf("stripe customer %s not found in stripe account %s", stripeCustomerData.StripeCustomerID, stripeAppData.StripeAccountID),
			)
		}

		return err
	}

	// Get customer billing profile
	customerBillingProfile, err := a.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customerID,
	})
	if err != nil {
		return fmt.Errorf("failed to get customer override: %w", err)
	}

	collectionMethod := customerBillingProfile.MergedProfile.WorkflowConfig.Payment.CollectionMethod

	// Validate customer for payment capabilitie
	if slices.Contains(capabilities, app.CapabilityTypeCollectPayments) {
		switch collectionMethod {
		// With auto charge collection method requires the customer to have a payment method and a billing address
		case billing.CollectionMethodChargeAutomatically:
			var paymentMethod stripeclient.StripePaymentMethod

			// Check if the customer has a default payment method in OpenMeter
			// If not try to use the Stripe Customer's default payment method
			if stripeCustomerData.StripeDefaultPaymentMethodID != nil {
				// Get the default payment method
				paymentMethod, err = stripeClient.GetPaymentMethod(ctx, *stripeCustomerData.StripeDefaultPaymentMethodID)
				if err != nil {
					if _, ok := err.(stripeclient.StripePaymentMethodNotFoundError); ok {
						return app.NewAppCustomerPreConditionError(
							a.GetID(),
							a.GetType(),
							&customerID,
							fmt.Sprintf("default payment method %s not found in stripe account %s", *stripeCustomerData.StripeDefaultPaymentMethodID, stripeAppData.StripeAccountID),
						)
					}

					return fmt.Errorf("failed to get default payment method: %w", err)
				}
			} else {
				// Check if the customer has a default payment method
				if stripeCustomer.DefaultPaymentMethod == nil {
					return app.NewAppCustomerPreConditionError(
						a.GetID(),
						a.GetType(),
						&customerID,
						"stripe customer must have a default payment method",
					)
				}

				paymentMethod = *stripeCustomer.DefaultPaymentMethod
			}

			// Payment method must have a billing address
			// Billing address is required for tax calculation and invoice creation
			if paymentMethod.BillingAddress == nil {
				return app.NewAppCustomerPreConditionError(
					a.GetID(),
					a.GetType(),
					&customerID,
					"stripe customer default payment method must have a billing address",
				)
			}
		case billing.CollectionMethodSendInvoice:
			// With send invoice collection method, the customer must have an email address
			if stripeCustomer.Email == nil {
				return app.NewAppCustomerPreConditionError(
					a.GetID(),
					a.GetType(),
					&customerID,
					fmt.Sprintf("stripe customer missing email: in order to create invoices that are sent to the stripe customer, the stripe customer %s must have a valid email", stripeCustomerData.StripeCustomerID),
				)
			}

		default:
			return fmt.Errorf("unsupported collection method: %s", collectionMethod)
		}
	}

	return nil
}

// GetCustomerData gets the customer data for the app
func (a App) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			err,
		)
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
func (a App) UpsertCustomerData(ctx context.Context, input app.UpsertAppInstanceCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			err,
		)
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
func (a App) DeleteCustomerData(ctx context.Context, input app.DeleteAppInstanceCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			err,
		)
	}

	appId := a.GetID()

	// Delete stripe customer data
	if err := a.StripeAppService.DeleteStripeCustomerData(ctx, appstripeentity.DeleteStripeCustomerDataInput{
		AppID:      &appId,
		CustomerID: &input.CustomerID,
	}); err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	return nil
}
