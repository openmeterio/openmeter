package appstripeentity

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/models"
)

// App represents an installed Stripe app
type App struct {
	appentitybase.AppBase
	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`

	Client              *entdb.Client                    `json:"-"`
	StripeClientFactory stripeclient.StripeClientFactory `json:"-"`
	SecretService       secret.Service                   `json:"-"`
}

func (a App) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if a.Type != appentitybase.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if a.Client == nil {
		return errors.New("client is required")
	}

	if a.StripeAccountId == "" {
		return errors.New("stripe account id is required")
	}

	if a.StripeClientFactory == nil {
		return errors.New("stripe client factory is required")
	}

	if a.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	// Get Stripe Customer
	stripeCustomerDBEntity, err := a.Client.AppStripeCustomer.
		Query().
		Where(appstripecustomerdb.Namespace(a.Namespace)).
		Where(appstripecustomerdb.AppID(a.ID)).
		Where(appstripecustomerdb.CustomerID(customer.ID)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  "customer has no data for stripe app",
			}
		}

		return fmt.Errorf("error getting stripe customer: %w", err)
	}

	// Check if the customer has a Stripe customer ID
	if stripeCustomerDBEntity.StripeCustomerID == nil || *stripeCustomerDBEntity.StripeCustomerID == "" {
		return app.CustomerPreConditionError{
			AppID:      a.GetID(),
			AppType:    a.GetType(),
			CustomerID: customer.GetID(),
			Condition:  "customer must have a stripe customer id",
		}
	}

	// Get Stripe App
	stripeApp, err := a.Client.AppStripe.
		Query().
		Where(appstripedb.Namespace(a.Namespace)).
		Where(appstripedb.ID(a.ID)).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return app.AppNotFoundError{
				AppID: a.GetID(),
			}
		}

		return fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.SecretService.GetAppSecret(ctx, secretentity.GetAppSecretInput{
		NamespacedID: models.NamespacedID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.ID,
		},
		AppID: a.GetID(),
		Key:   APIKeySecretKey,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.StripeClientFactory(stripeclient.StripeClientConfig{
		Namespace: stripeApp.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Check if the customer exists in Stripe
	stripeCustomer, err := stripeClient.GetCustomer(ctx, *stripeCustomerDBEntity.StripeCustomerID)
	if err != nil {
		if _, ok := err.(stripeclient.StripeCustomerNotFoundError); ok {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  fmt.Sprintf("stripe customer %s not found in stripe account %s", *stripeCustomerDBEntity.StripeCustomerID, stripeApp.StripeAccountID),
			}
		}

		return err
	}

	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	if slices.Contains(capabilities, appentitybase.CapabilityTypeCalculateTax) || slices.Contains(capabilities, appentitybase.CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, appentitybase.CapabilityTypeCollectPayments) {
		var paymentMethod stripeclient.StripePaymentMethod

		// Check if the customer has a default payment method in OpenMeter
		// If not try to use the Stripe Customer's default payment method
		if stripeCustomerDBEntity.StripeDefaultPaymentMethodID != nil {
			// Get the default payment method
			paymentMethod, err = stripeClient.GetPaymentMethod(ctx, *stripeCustomerDBEntity.StripeDefaultPaymentMethodID)
			if err != nil {
				if _, ok := err.(stripeclient.StripePaymentMethodNotFoundError); ok {
					return app.CustomerPreConditionError{
						AppID:      a.GetID(),
						AppType:    a.GetType(),
						CustomerID: customer.GetID(),
						Condition:  fmt.Sprintf("default payment method %s not found in stripe account %s", *stripeCustomerDBEntity.StripeDefaultPaymentMethodID, stripeApp.StripeAccountID),
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
