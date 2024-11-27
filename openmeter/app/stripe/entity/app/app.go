package appstripeentityapp

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

const (
	APIKeySecretKey = "stripe_api_key"
)

var _ customerentity.App = (*App)(nil)

// App represents an installed Stripe app
type App struct {
	appentitybase.AppBase
	appstripeentity.AppData

	StripeClientFactory stripeclient.StripeClientFactory `json:"-"`
	AppService          app.AppService
	// TODO: can this be a service? The factory is is in the adapter that the service depends on
	StripeAppService stripeapp.Adapter `json:"-"`
	SecretService    secret.Service    `json:"-"`
}

func (a App) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if err := a.AppData.Validate(); err != nil {
		return fmt.Errorf("error validating stripe app data: %w", err)
	}

	if a.Type != appentitybase.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if err := a.AppData.Validate(); err != nil {
		return fmt.Errorf("error validating stripe app data: %w", err)
	}

	if a.StripeClientFactory == nil {
		return errors.New("stripe client factory is required")
	}

	if a.AppService == nil {
		return errors.New("app service is required")
	}

	if a.StripeAppService == nil {
		return errors.New("stripe app service is required")
	}

	if a.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities...); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	// Get Stripe App
	stripeAppData, err := a.StripeAppService.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: a.GetID(),
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Get Stripe Customer
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      a.GetID(),
		CustomerID: customer.GetID(),
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.SecretService.GetAppSecret(ctx, secretentity.NewSecretID(a.GetID(), stripeAppData.APIKey.ID, appstripeentity.APIKeySecretKey))
	if err != nil {
		return fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.StripeClientFactory(stripeclient.StripeClientConfig{
		Namespace: apiKeySecret.SecretID.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Check if the customer exists in Stripe
	stripeCustomer, err := stripeClient.GetCustomer(ctx, stripeCustomerData.StripeCustomerID)
	if err != nil {
		if _, ok := err.(stripeclient.StripeCustomerNotFoundError); ok {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
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
						CustomerID: customer.GetID(),
						Condition:  fmt.Sprintf("default payment method %s not found in stripe account %s", *stripeCustomerData.StripeDefaultPaymentMethodID, stripeAppData.StripeAccountID),
					}
				}

				return fmt.Errorf("failed to get default payment method: %w", err)
			}
		} else {
			// Check if the customer has a default payment method
			if stripeCustomer.DefaultPaymentMethod != nil {
				paymentMethod = *stripeCustomer.DefaultPaymentMethod
			} else {
				return app.CustomerPreConditionError{
					AppID:      a.GetID(),
					AppType:    a.GetType(),
					CustomerID: customer.GetID(),
					Condition:  "stripe customer must have a default payment method",
				}
			}
		}

		// Payment method must have a billing address
		// Billing address is required for tax calculation and invoice creation
		if paymentMethod.BillingAddress == nil {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  "stripe customer default payment method must have a billing address",
			}
		}

		// TODO: should we have currency as an input to validation?
	}

	return nil
}

// GetCustomerData gets the customer data for the app
func (a App) GetCustomerData(ctx context.Context, input appentity.GetCustomerDataInput) (appentity.CustomerData, error) {
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
func (a App) UpsertCustomerData(ctx context.Context, input appentity.UpsertCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("error validating input: %w", err)
	}

	stripeCustomerData, ok := input.Data.(appstripeentity.CustomerData)
	if !ok {
		return fmt.Errorf("error casting stripe customer data")
	}

	// Make sure the customer has an app relationship
	err := a.AppService.EnsureCustomer(ctx, app.EnsureCustomerInput{
		AppID:      a.GetID(),
		CustomerID: input.CustomerID,
	})
	if err != nil {
		return fmt.Errorf("failed to ensure customer: %w", err)
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
func (a App) DeleteCustomerData(ctx context.Context, input appentity.DeleteCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("error validating input: %w", err)
	}

	appId := a.GetID()

	if err := a.StripeAppService.DeleteStripeCustomerData(ctx, appstripeentity.DeleteStripeCustomerDataInput{
		AppID:      &appId,
		CustomerID: input.CustomerID,
	}); err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	return nil
}
